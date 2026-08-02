// Comandos sob demanda (Fase 5, início): usuário dispara um teste (ex.:
// "ping agora"), o backend enfileira (Postgres, não Redis — ver comentário
// na migração 0004_agent_commands) e o agente do site consulta/executa na
// mesma conexão outbound de telemetria (docs/architecture/03-fluxo-de-dados.md
// §3.2, ADR-001).
package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"egger/api/internal/store"
)

var supportedCommandTypes = map[string]bool{
	store.AgentCommandTypePing:        true,
	store.AgentCommandTypeDNSLookup:   true,
	store.AgentCommandTypeTraceroute:  true,
	store.AgentCommandTypeBatchPing:   true,
	store.AgentCommandTypeSSLCheck:    true,
	store.AgentCommandTypeHTTPRequest: true,
	store.AgentCommandTypeLANScan:     true,
}

// minLANScanPrefixLen limita o tamanho do bloco CIDR aceito pro LAN
// scanner a no máximo 1024 endereços (/22) — bastante pra descobrir
// dispositivos numa LAN residencial/pequena empresa, sem virar uma
// varredura de rede arbitrariamente grande.
const minLANScanPrefixLen = 22

var supportedHTTPMethods = map[string]bool{
	"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true, "HEAD": true, "OPTIONS": true,
}

// maxBatchPingTargets é um limite de sanidade, não uma primitiva de
// varredura em massa (threat-model.md §5): o caso de uso é testar uma lista
// pequena de alvos já conhecidos do usuário (ex.: alguns APs/servidores),
// não enumerar uma faixa de rede.
const maxBatchPingTargets = 20

type createCommandRequest struct {
	Type   string          `json:"type"`
	Params json.RawMessage `json:"params"`
}

type pingCommandParams struct {
	Target   string `json:"target"`
	Protocol string `json:"protocol"`
}

var supportedPingProtocols = map[string]bool{
	"icmp": true, "tcp": true, "http": true, "dns": true,
}

type dnsLookupCommandParams struct {
	Hostname string `json:"hostname"`
}

type tracerouteCommandParams struct {
	Target string `json:"target"`
}

type batchPingCommandParams struct {
	Targets  []string `json:"targets"`
	Protocol string   `json:"protocol"`
}

type sslCheckCommandParams struct {
	Target string `json:"target"`
	Port   int    `json:"port"`
}

type httpRequestCommandParams struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

type lanScanCommandParams struct {
	CIDR string `json:"cidr"`
}

// validatePrivateIPv4CIDR implementa a mitigação exigida pelo threat model
// (§5, "Riscos específicos... port scanner/LAN scanner") antes de expor uma
// ferramenta que varre múltiplos hosts: o alvo nunca pode ser uma rede
// pública (só existe risco de "usar o backend como ferramenta de ataque a
// terceiros" se o alvo puder ser de terceiros) e o bloco não pode ser maior
// que o necessário pra descobrir dispositivos numa LAN residencial/pequena.
func validatePrivateIPv4CIDR(cidr string) error {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return errors.New("params.cidr inválido")
	}
	// ParseCIDR já devolve ipnet.IP como o endereço de rede (mascarado) —
	// verificamos a rede e o broadcast, não o endereço original informado
	// (que pode não estar alinhado à máscara).
	network := ipnet.IP.To4()
	if network == nil || len(ipnet.Mask) != net.IPv4len {
		return errors.New("params.cidr precisa ser um bloco IPv4")
	}
	ones, _ := ipnet.Mask.Size()
	if ones < minLANScanPrefixLen {
		return fmt.Errorf("params.cidr aceita no máximo /%d (1024 endereços)", minLANScanPrefixLen)
	}
	if !network.IsPrivate() {
		return errors.New("params.cidr precisa estar dentro de um intervalo privado (RFC 1918)")
	}
	broadcast := make(net.IP, len(network))
	for i := range broadcast {
		broadcast[i] = network[i] | ^ipnet.Mask[i]
	}
	if !broadcast.IsPrivate() {
		return errors.New("params.cidr precisa estar inteiramente dentro de um intervalo privado (RFC 1918)")
	}
	return nil
}

// handleCreateCommand valida o tipo/params antes de persistir — nunca
// aceita um comando que o produto não sabe executar, mesmo que o schema
// jsonb da coluna `params` aceitasse qualquer coisa.
func (s *Server) handleCreateCommand(w http.ResponseWriter, r *http.Request) {
	correlationID := correlationIDFromContext(r.Context())

	user, _ := userFromContext(r.Context())
	if !s.commandLimiter.Allow(user.ID.String()) {
		writeError(w, correlationID, http.StatusTooManyRequests, "rate_limited", "muitos comandos em pouco tempo, tente novamente em instantes")
		return
	}

	siteID, err := uuid.Parse(r.PathValue("siteId"))
	if err != nil {
		writeError(w, correlationID, http.StatusBadRequest, "invalid_site_id", "siteId inválido")
		return
	}

	var req createCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "corpo da requisição inválido")
		return
	}
	if !supportedCommandTypes[req.Type] {
		writeError(w, correlationID, http.StatusBadRequest, "unsupported_command_type", "tipo de comando não suportado: "+req.Type)
		return
	}

	switch req.Type {
	case store.AgentCommandTypePing:
		var p pingCommandParams
		if len(req.Params) == 0 {
			writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "params.target é obrigatório para type=ping")
			return
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "params inválido")
			return
		}
		if p.Target == "" {
			writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "params.target é obrigatório")
			return
		}
		if p.Protocol == "" {
			p.Protocol = "icmp"
		}
		if !supportedPingProtocols[p.Protocol] {
			writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "params.protocol inválido (icmp, tcp, http ou dns)")
			return
		}
		req.Params, _ = json.Marshal(p)

	case store.AgentCommandTypeDNSLookup:
		var p dnsLookupCommandParams
		if len(req.Params) == 0 {
			writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "params.hostname é obrigatório para type=dns_lookup")
			return
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "params inválido")
			return
		}
		if p.Hostname == "" {
			writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "params.hostname é obrigatório")
			return
		}
		req.Params, _ = json.Marshal(p)

	case store.AgentCommandTypeTraceroute:
		var p tracerouteCommandParams
		if len(req.Params) == 0 {
			writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "params.target é obrigatório para type=traceroute")
			return
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "params inválido")
			return
		}
		if p.Target == "" {
			writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "params.target é obrigatório")
			return
		}
		req.Params, _ = json.Marshal(p)

	case store.AgentCommandTypeBatchPing:
		var p batchPingCommandParams
		if len(req.Params) == 0 {
			writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "params.targets é obrigatório para type=batch_ping")
			return
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "params inválido")
			return
		}
		if len(p.Targets) == 0 {
			writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "params.targets precisa ter ao menos 1 alvo")
			return
		}
		if len(p.Targets) > maxBatchPingTargets {
			writeError(w, correlationID, http.StatusBadRequest, "invalid_body", fmt.Sprintf("params.targets aceita no máximo %d alvos", maxBatchPingTargets))
			return
		}
		for _, t := range p.Targets {
			if t == "" {
				writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "params.targets não pode conter alvo vazio")
				return
			}
		}
		if p.Protocol == "" {
			p.Protocol = "icmp"
		}
		if !supportedPingProtocols[p.Protocol] {
			writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "params.protocol inválido (icmp, tcp, http ou dns)")
			return
		}
		req.Params, _ = json.Marshal(p)

	case store.AgentCommandTypeSSLCheck:
		var p sslCheckCommandParams
		if len(req.Params) == 0 {
			writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "params.target é obrigatório para type=ssl_check")
			return
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "params inválido")
			return
		}
		if p.Target == "" {
			writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "params.target é obrigatório")
			return
		}
		if p.Port == 0 {
			p.Port = 443
		}
		if p.Port < 1 || p.Port > 65535 {
			writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "params.port precisa estar entre 1 e 65535")
			return
		}
		req.Params, _ = json.Marshal(p)

	case store.AgentCommandTypeHTTPRequest:
		var p httpRequestCommandParams
		if len(req.Params) == 0 {
			writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "params.url é obrigatório para type=http_request")
			return
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "params inválido")
			return
		}
		parsed, err := url.Parse(p.URL)
		if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "params.url precisa ser uma URL http(s) válida")
			return
		}
		if p.Method == "" {
			p.Method = "GET"
		}
		if !supportedHTTPMethods[strings.ToUpper(p.Method)] {
			writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "params.method inválido (GET, POST, PUT, PATCH, DELETE, HEAD ou OPTIONS)")
			return
		}
		req.Params, _ = json.Marshal(p)

	case store.AgentCommandTypeLANScan:
		var p lanScanCommandParams
		if len(req.Params) == 0 {
			writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "params.cidr é obrigatório para type=lan_scan")
			return
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "params inválido")
			return
		}
		if p.CIDR == "" {
			writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "params.cidr é obrigatório")
			return
		}
		if err := validatePrivateIPv4CIDR(p.CIDR); err != nil {
			writeError(w, correlationID, http.StatusBadRequest, "invalid_body", err.Error())
			return
		}
		req.Params, _ = json.Marshal(p)
	}

	now := time.Now().UTC()

	cmd, err := s.agentCommands.Create(r.Context(), siteID, user.ID, req.Type, req.Params, now)
	if err != nil {
		if errors.Is(err, store.ErrNoActiveAgent) {
			writeError(w, correlationID, http.StatusConflict, "no_active_agent", "nenhum agente ativo neste site para executar o comando")
			return
		}
		writeError(w, correlationID, http.StatusInternalServerError, "internal_error", "erro ao criar comando")
		return
	}

	writeJSON(w, http.StatusAccepted, commandToJSON(cmd))
}

func (s *Server) handleGetCommand(w http.ResponseWriter, r *http.Request) {
	correlationID := correlationIDFromContext(r.Context())

	id, err := uuid.Parse(r.PathValue("commandId"))
	if err != nil {
		writeError(w, correlationID, http.StatusBadRequest, "invalid_command_id", "commandId inválido")
		return
	}

	cmd, err := s.agentCommands.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, correlationID, http.StatusNotFound, "not_found", "comando não encontrado")
			return
		}
		writeError(w, correlationID, http.StatusInternalServerError, "internal_error", "erro ao buscar comando")
		return
	}

	writeJSON(w, http.StatusOK, commandToJSON(cmd))
}

// handleClaimAgentCommands é chamado periodicamente pelo agente (mesmo
// padrão do heartbeat) para saber se há comandos pendentes — o agente nunca
// expõe porta de entrada, sempre consulta (ADR-001).
func (s *Server) handleClaimAgentCommands(w http.ResponseWriter, r *http.Request) {
	correlationID := correlationIDFromContext(r.Context())
	agent, _ := agentFromContext(r.Context())

	const claimLimit = 5
	now := time.Now().UTC()
	cmds, err := s.agentCommands.ClaimPending(r.Context(), agent.ID, claimLimit, now)
	if err != nil {
		writeError(w, correlationID, http.StatusInternalServerError, "internal_error", "erro ao buscar comandos pendentes")
		return
	}

	items := make([]map[string]any, 0, len(cmds))
	for _, c := range cmds {
		items = append(items, commandToJSON(c))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

type commandResultRequest struct {
	Status string          `json:"status"` // completed | failed
	Result json.RawMessage `json:"result"`
	Error  string          `json:"error"`
}

// handleReportCommandResult recebe o resultado de um comando previamente
// reivindicado (claimed) pelo próprio agente autenticado — um agente nunca
// pode reportar resultado de um comando de outro agente/site.
func (s *Server) handleReportCommandResult(w http.ResponseWriter, r *http.Request) {
	correlationID := correlationIDFromContext(r.Context())
	agent, _ := agentFromContext(r.Context())

	commandID, err := uuid.Parse(r.PathValue("commandId"))
	if err != nil {
		writeError(w, correlationID, http.StatusBadRequest, "invalid_command_id", "commandId inválido")
		return
	}

	var req commandResultRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "corpo da requisição inválido")
		return
	}
	if req.Status != store.AgentCommandStatusCompleted && req.Status != store.AgentCommandStatusFailed {
		writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "status deve ser completed ou failed")
		return
	}

	cmd, err := s.agentCommands.Get(r.Context(), commandID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, correlationID, http.StatusNotFound, "not_found", "comando não encontrado")
			return
		}
		writeError(w, correlationID, http.StatusInternalServerError, "internal_error", "erro ao buscar comando")
		return
	}
	if cmd.AgentID != agent.ID {
		writeError(w, correlationID, http.StatusForbidden, "forbidden", "comando não pertence a este agente")
		return
	}

	now := time.Now().UTC()
	if req.Status == store.AgentCommandStatusFailed {
		if req.Error == "" {
			req.Error = "falha não especificada pelo agente"
		}
		if err := s.agentCommands.Fail(r.Context(), commandID, req.Error, now); err != nil {
			writeError(w, correlationID, http.StatusInternalServerError, "internal_error", "erro ao registrar falha do comando")
			return
		}
	} else {
		if err := s.agentCommands.Complete(r.Context(), commandID, req.Result, now); err != nil {
			writeError(w, correlationID, http.StatusInternalServerError, "internal_error", "erro ao registrar resultado do comando")
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func commandToJSON(c store.AgentCommand) map[string]any {
	out := map[string]any{
		"id":         c.ID.String(),
		"site_id":    c.SiteID.String(),
		"agent_id":   c.AgentID.String(),
		"type":       c.Type,
		"params":     json.RawMessage(c.Params),
		"status":     c.Status,
		"created_at": c.CreatedAt.Format(time.RFC3339),
	}
	if c.Result != nil {
		out["result"] = json.RawMessage(c.Result)
	}
	if c.Error != nil {
		out["error"] = *c.Error
	}
	if c.ClaimedAt != nil {
		out["claimed_at"] = c.ClaimedAt.Format(time.RFC3339)
	}
	if c.CompletedAt != nil {
		out["completed_at"] = c.CompletedAt.Format(time.RFC3339)
	}
	return out
}
