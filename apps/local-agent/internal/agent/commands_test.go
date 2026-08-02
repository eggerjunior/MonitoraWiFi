package agent

import (
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"egger/local-agent/internal/config"
	"egger/local-agent/internal/state"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// newTestAgentForCommands monta um Agent real (não um fake opaco) contra um
// backend de teste real — só a fronteira HTTP é substituída; o dispatch de
// comando e a execução da sonda usam o código de produção de verdade.
func newTestAgentForCommands(t *testing.T, backendURL string) *Agent {
	t.Helper()

	statePath := filepath.Join(t.TempDir(), "agent.json")
	if err := state.Save(statePath, state.Identity{AgentID: "agent-1", AgentSecret: "secret-1", SiteID: "site-1"}); err != nil {
		t.Fatalf("erro ao preparar estado: %v", err)
	}

	cfg := config.Config{
		BackendURL:    backendURL,
		StateFilePath: statePath,
	}

	a, err := New(t.Context(), cfg, testLogger())
	if err != nil {
		t.Fatalf("erro ao construir agente: %v", err)
	}
	return a
}

func TestPollAndRunCommands_Ping(t *testing.T) {
	// Listener TCP real — a sonda "ping" tipo tcp mede latência contra ele
	// de verdade, não um valor inventado.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("erro ao abrir listener de teste: %v", err)
	}
	defer ln.Close()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()
	target := ln.Addr().String()

	var reportedBody map[string]any
	claimed := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/agents/agent-1/commands":
			if claimed {
				json.NewEncoder(w).Encode(map[string]any{"items": []map[string]any{}})
				return
			}
			claimed = true
			params, _ := json.Marshal(map[string]string{"target": target, "protocol": "tcp"})
			json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{
					{"id": "cmd-1", "site_id": "site-1", "type": "ping", "params": json.RawMessage(params), "status": "claimed"},
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/agents/agent-1/commands/cmd-1/result":
			json.NewDecoder(r.Body).Decode(&reportedBody)
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("requisição inesperada: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	a := newTestAgentForCommands(t, server.URL)
	a.pollAndRunCommands(t.Context())

	if reportedBody == nil {
		t.Fatal("esperava que o agente reportasse um resultado, nenhum recebido")
	}
	if reportedBody["status"] != "completed" {
		t.Fatalf("status reportado = %v, esperado completed", reportedBody["status"])
	}
	result, ok := reportedBody["result"].(map[string]any)
	if !ok {
		t.Fatalf("esperava campo result no reporte, recebi: %+v", reportedBody)
	}
	if result["protocol"] != "tcp" || result["target"] != target {
		t.Fatalf("result inesperado: %+v", result)
	}
	if result["packet_loss_pct"] != 0.0 {
		t.Fatalf("esperava 0%% de perda contra um listener real, obtive %v", result["packet_loss_pct"])
	}
}

func TestPollAndRunCommands_SSLCheck(t *testing.T) {
	// Servidor TLS real (httptest) — o comando ssl_check precisa fazer um
	// handshake TLS de verdade e extrair os metadados reais do certificado
	// apresentado, nunca inventar validade/emissor.
	tlsServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer tlsServer.Close()
	host, portStr, err := net.SplitHostPort(tlsServer.Listener.Addr().String())
	if err != nil {
		t.Fatalf("erro ao separar host/porta: %v", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("porta inválida: %v", err)
	}

	var reportedBody map[string]any
	claimed := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/agents/agent-1/commands":
			if claimed {
				json.NewEncoder(w).Encode(map[string]any{"items": []map[string]any{}})
				return
			}
			claimed = true
			params, _ := json.Marshal(map[string]any{"target": host, "port": port})
			json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{
					{"id": "cmd-6", "site_id": "site-1", "type": "ssl_check", "params": json.RawMessage(params), "status": "claimed"},
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/agents/agent-1/commands/cmd-6/result":
			json.NewDecoder(r.Body).Decode(&reportedBody)
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("requisição inesperada: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	a := newTestAgentForCommands(t, server.URL)
	a.pollAndRunCommands(t.Context())

	if reportedBody == nil {
		t.Fatal("esperava que o agente reportasse um resultado, nenhum recebido")
	}
	if reportedBody["status"] != "completed" {
		t.Fatalf("status reportado = %v, esperado completed: %+v", reportedBody["status"], reportedBody)
	}
	result, ok := reportedBody["result"].(map[string]any)
	if !ok {
		t.Fatalf("esperava campo result no reporte, recebi: %+v", reportedBody)
	}
	if result["valid_now"] != false {
		t.Fatalf("esperava valid_now=false para certificado autoassinado, obtive: %+v", result)
	}
	if result["issuer"] == "" || result["issuer"] == nil {
		t.Fatalf("esperava issuer extraído do certificado real, obtive: %+v", result)
	}
	if result["not_after"] == "" || result["not_after"] == nil {
		t.Fatalf("esperava not_after extraído do certificado real, obtive: %+v", result)
	}
}

func TestPollAndRunCommands_HTTPRequest(t *testing.T) {
	// Servidor HTTP real — o comando http_request precisa fazer uma
	// requisição real e devolver status/headers/corpo reais.
	var receivedMethod, receivedHeader, receivedBody string
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedHeader = r.Header.Get("X-Teste")
		body, _ := io.ReadAll(r.Body)
		receivedBody = string(body)
		w.Header().Set("X-Resposta", "ok")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"echo":"real"}`))
	}))
	defer targetServer.Close()

	var reportedBody map[string]any
	claimed := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/agents/agent-1/commands":
			if claimed {
				json.NewEncoder(w).Encode(map[string]any{"items": []map[string]any{}})
				return
			}
			claimed = true
			params, _ := json.Marshal(map[string]any{
				"url":     targetServer.URL,
				"method":  "post",
				"headers": map[string]string{"X-Teste": "valor-real"},
				"body":    "corpo-real",
			})
			json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{
					{"id": "cmd-7", "site_id": "site-1", "type": "http_request", "params": json.RawMessage(params), "status": "claimed"},
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/agents/agent-1/commands/cmd-7/result":
			json.NewDecoder(r.Body).Decode(&reportedBody)
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("requisição inesperada: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	a := newTestAgentForCommands(t, server.URL)
	a.pollAndRunCommands(t.Context())

	if receivedMethod != http.MethodPost {
		t.Fatalf("servidor alvo recebeu método %q, esperado POST", receivedMethod)
	}
	if receivedHeader != "valor-real" {
		t.Fatalf("servidor alvo recebeu header X-Teste=%q, esperado valor-real", receivedHeader)
	}
	if receivedBody != "corpo-real" {
		t.Fatalf("servidor alvo recebeu corpo %q, esperado corpo-real", receivedBody)
	}

	if reportedBody == nil {
		t.Fatal("esperava que o agente reportasse um resultado, nenhum recebido")
	}
	if reportedBody["status"] != "completed" {
		t.Fatalf("status reportado = %v, esperado completed: %+v", reportedBody["status"], reportedBody)
	}
	result, ok := reportedBody["result"].(map[string]any)
	if !ok {
		t.Fatalf("esperava campo result no reporte, recebi: %+v", reportedBody)
	}
	if result["status_code"] != float64(http.StatusCreated) {
		t.Fatalf("status_code = %v, esperado 201", result["status_code"])
	}
	if result["body_snippet"] != `{"echo":"real"}` {
		t.Fatalf("body_snippet = %v, esperado corpo real do servidor", result["body_snippet"])
	}
	headers, ok := result["headers"].(map[string]any)
	if !ok || headers["X-Resposta"] != "ok" {
		t.Fatalf("esperava header X-Resposta=ok na resposta real, obtive: %+v", result["headers"])
	}
}

func TestPollAndRunCommands_LANScan(t *testing.T) {
	// CIDR real pequeno (127.0.0.0/30, todo ele loopback local de verdade)
	// — o SO responde ECONNREFUSED de verdade nas portas comuns pra cada um
	// dos 4 endereços (nenhum serviço ouvindo nelas neste ambiente), o que
	// já basta pra provar que o host está de pé.
	var reportedBody map[string]any
	claimed := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/agents/agent-1/commands":
			if claimed {
				json.NewEncoder(w).Encode(map[string]any{"items": []map[string]any{}})
				return
			}
			claimed = true
			params, _ := json.Marshal(map[string]any{"cidr": "127.0.0.0/30"})
			json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{
					{"id": "cmd-8", "site_id": "site-1", "type": "lan_scan", "params": json.RawMessage(params), "status": "claimed"},
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/agents/agent-1/commands/cmd-8/result":
			json.NewDecoder(r.Body).Decode(&reportedBody)
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("requisição inesperada: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	a := newTestAgentForCommands(t, server.URL)
	a.pollAndRunCommands(t.Context())

	if reportedBody == nil {
		t.Fatal("esperava que o agente reportasse um resultado, nenhum recebido")
	}
	if reportedBody["status"] != "completed" {
		t.Fatalf("status reportado = %v, esperado completed: %+v", reportedBody["status"], reportedBody)
	}
	result, ok := reportedBody["result"].(map[string]any)
	if !ok {
		t.Fatalf("esperava campo result no reporte, recebi: %+v", reportedBody)
	}
	hosts, ok := result["hosts"].([]any)
	// Todo 127.0.0.0/8 é loopback local — os 4 endereços do /30 aparecem
	// como vivos (usa portas comuns; o SO responde ECONNREFUSED de
	// verdade nas que não têm o listener real).
	if !ok || len(hosts) != 4 {
		t.Fatalf("esperava 4 hosts no resultado (127.0.0.0/8 é loopback), recebi: %+v", result["hosts"])
	}
}

func TestPollAndRunCommands_WakeOnLAN(t *testing.T) {
	// Listener UDP real na loopback — o comando wake_on_lan precisa enviar
	// o magic packet de verdade, não simular.
	udpConn, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("erro ao abrir listener UDP: %v", err)
	}
	defer udpConn.Close()
	_, udpPortStr, _ := net.SplitHostPort(udpConn.LocalAddr().String())
	udpPort, _ := strconv.Atoi(udpPortStr)

	received := make(chan []byte, 1)
	go func() {
		buf := make([]byte, 256)
		n, _, err := udpConn.ReadFrom(buf)
		if err == nil {
			received <- buf[:n]
		}
	}()

	var reportedBody map[string]any
	claimed := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/agents/agent-1/commands":
			if claimed {
				json.NewEncoder(w).Encode(map[string]any{"items": []map[string]any{}})
				return
			}
			claimed = true
			params, _ := json.Marshal(map[string]any{
				"mac_address":  "AA:BB:CC:DD:EE:FF",
				"broadcast_ip": "127.0.0.1",
				"port":         udpPort,
			})
			json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{
					{"id": "cmd-9", "site_id": "site-1", "type": "wake_on_lan", "params": json.RawMessage(params), "status": "claimed"},
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/agents/agent-1/commands/cmd-9/result":
			json.NewDecoder(r.Body).Decode(&reportedBody)
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("requisição inesperada: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	a := newTestAgentForCommands(t, server.URL)
	a.pollAndRunCommands(t.Context())

	if reportedBody == nil {
		t.Fatal("esperava que o agente reportasse um resultado, nenhum recebido")
	}
	if reportedBody["status"] != "completed" {
		t.Fatalf("status reportado = %v, esperado completed: %+v", reportedBody["status"], reportedBody)
	}

	select {
	case packet := <-received:
		if len(packet) != 102 {
			t.Fatalf("pacote real recebido tem %d bytes, esperado 102", len(packet))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("nenhum magic packet real recebido no listener UDP")
	}
}

func TestPollAndRunCommands_PortScan(t *testing.T) {
	// Listener TCP real — o comando port_scan precisa achar essa porta de
	// verdade dentro do intervalo pedido, e nenhuma outra.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("erro ao abrir listener: %v", err)
	}
	defer ln.Close()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()
	_, portStr, _ := net.SplitHostPort(ln.Addr().String())
	port, _ := strconv.Atoi(portStr)

	var reportedBody map[string]any
	claimed := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/agents/agent-1/commands":
			if claimed {
				json.NewEncoder(w).Encode(map[string]any{"items": []map[string]any{}})
				return
			}
			claimed = true
			params, _ := json.Marshal(map[string]any{"target": "127.0.0.1", "start_port": port - 2, "end_port": port + 2})
			json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{
					{"id": "cmd-10", "site_id": "site-1", "type": "port_scan", "params": json.RawMessage(params), "status": "claimed"},
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/agents/agent-1/commands/cmd-10/result":
			json.NewDecoder(r.Body).Decode(&reportedBody)
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("requisição inesperada: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	a := newTestAgentForCommands(t, server.URL)
	a.pollAndRunCommands(t.Context())

	if reportedBody == nil {
		t.Fatal("esperava que o agente reportasse um resultado, nenhum recebido")
	}
	if reportedBody["status"] != "completed" {
		t.Fatalf("status reportado = %v, esperado completed: %+v", reportedBody["status"], reportedBody)
	}
	result, ok := reportedBody["result"].(map[string]any)
	if !ok {
		t.Fatalf("esperava campo result no reporte, recebi: %+v", reportedBody)
	}
	openPorts, ok := result["open_ports"].([]any)
	if !ok || len(openPorts) != 1 || int(openPorts[0].(float64)) != port {
		t.Fatalf("open_ports = %v, esperado apenas [%d]", result["open_ports"], port)
	}
}

func TestPollAndRunCommands_TipoNaoSuportado(t *testing.T) {
	var reportedBody map[string]any
	claimed := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/agents/agent-1/commands":
			if claimed {
				json.NewEncoder(w).Encode(map[string]any{"items": []map[string]any{}})
				return
			}
			claimed = true
			json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{
					{"id": "cmd-2", "site_id": "site-1", "type": "carrier_pigeon", "params": json.RawMessage(`{}`), "status": "claimed"},
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/agents/agent-1/commands/cmd-2/result":
			json.NewDecoder(r.Body).Decode(&reportedBody)
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("requisição inesperada: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	a := newTestAgentForCommands(t, server.URL)
	a.pollAndRunCommands(t.Context())

	if reportedBody == nil {
		t.Fatal("esperava reporte de falha para tipo não suportado")
	}
	if reportedBody["status"] != "failed" {
		t.Fatalf("status reportado = %v, esperado failed", reportedBody["status"])
	}
	if reportedBody["error"] == nil || reportedBody["error"] == "" {
		t.Fatal("esperava mensagem de erro explicando o tipo não suportado")
	}
}

func TestPollAndRunCommands_DNSLookup(t *testing.T) {
	var reportedBody map[string]any
	claimed := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/agents/agent-1/commands":
			if claimed {
				json.NewEncoder(w).Encode(map[string]any{"items": []map[string]any{}})
				return
			}
			claimed = true
			params, _ := json.Marshal(map[string]string{"hostname": "localhost"})
			json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{
					{"id": "cmd-3", "site_id": "site-1", "type": "dns_lookup", "params": json.RawMessage(params), "status": "claimed"},
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/agents/agent-1/commands/cmd-3/result":
			json.NewDecoder(r.Body).Decode(&reportedBody)
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("requisição inesperada: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	a := newTestAgentForCommands(t, server.URL)
	a.pollAndRunCommands(t.Context())

	if reportedBody == nil {
		t.Fatal("esperava que o agente reportasse um resultado, nenhum recebido")
	}
	if reportedBody["status"] != "completed" {
		t.Fatalf("status reportado = %v, esperado completed", reportedBody["status"])
	}
	result, ok := reportedBody["result"].(map[string]any)
	if !ok {
		t.Fatalf("esperava campo result no reporte, recebi: %+v", reportedBody)
	}
	addrs, ok := result["addresses"].([]any)
	if !ok || len(addrs) == 0 {
		t.Fatalf("esperava pelo menos um endereço resolvido para localhost, obtive: %+v", result["addresses"])
	}
}

func TestPollAndRunCommands_Traceroute(t *testing.T) {
	var reportedBody map[string]any
	claimed := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/agents/agent-1/commands":
			if claimed {
				json.NewEncoder(w).Encode(map[string]any{"items": []map[string]any{}})
				return
			}
			claimed = true
			params, _ := json.Marshal(map[string]string{"target": "127.0.0.1"})
			json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{
					{"id": "cmd-4", "site_id": "site-1", "type": "traceroute", "params": json.RawMessage(params), "status": "claimed"},
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/agents/agent-1/commands/cmd-4/result":
			json.NewDecoder(r.Body).Decode(&reportedBody)
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("requisição inesperada: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	a := newTestAgentForCommands(t, server.URL)
	a.pollAndRunCommands(t.Context())

	if reportedBody == nil {
		t.Fatal("esperava que o agente reportasse um resultado, nenhum recebido")
	}
	if reportedBody["status"] == "failed" {
		t.Skipf("ICMP indisponível neste ambiente de teste: %v", reportedBody["error"])
	}
	if reportedBody["status"] != "completed" {
		t.Fatalf("status reportado = %v, esperado completed", reportedBody["status"])
	}
	result, ok := reportedBody["result"].(map[string]any)
	if !ok {
		t.Fatalf("esperava campo result no reporte, recebi: %+v", reportedBody)
	}
	if result["reached"] != true {
		t.Fatalf("esperava reached=true para traceroute até 127.0.0.1, obtive: %+v", result)
	}
}

func TestPollAndRunCommands_BatchPing(t *testing.T) {
	// Dois listeners TCP reais — o batch_ping precisa medir cada alvo de
	// verdade, não inventar N resultados a partir de uma única sonda.
	ln1, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("erro ao abrir listener 1: %v", err)
	}
	defer ln1.Close()
	ln2, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("erro ao abrir listener 2: %v", err)
	}
	defer ln2.Close()
	for _, ln := range []net.Listener{ln1, ln2} {
		go func(l net.Listener) {
			for {
				conn, err := l.Accept()
				if err != nil {
					return
				}
				conn.Close()
			}
		}(ln)
	}
	targets := []string{ln1.Addr().String(), ln2.Addr().String()}

	var reportedBody map[string]any
	claimed := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/agents/agent-1/commands":
			if claimed {
				json.NewEncoder(w).Encode(map[string]any{"items": []map[string]any{}})
				return
			}
			claimed = true
			params, _ := json.Marshal(map[string]any{"targets": targets, "protocol": "tcp"})
			json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{
					{"id": "cmd-5", "site_id": "site-1", "type": "batch_ping", "params": json.RawMessage(params), "status": "claimed"},
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/agents/agent-1/commands/cmd-5/result":
			json.NewDecoder(r.Body).Decode(&reportedBody)
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("requisição inesperada: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	a := newTestAgentForCommands(t, server.URL)
	a.pollAndRunCommands(t.Context())

	if reportedBody == nil {
		t.Fatal("esperava que o agente reportasse um resultado, nenhum recebido")
	}
	if reportedBody["status"] != "completed" {
		t.Fatalf("status reportado = %v, esperado completed", reportedBody["status"])
	}
	result, ok := reportedBody["result"].(map[string]any)
	if !ok {
		t.Fatalf("esperava campo result no reporte, recebi: %+v", reportedBody)
	}
	results, ok := result["results"].([]any)
	if !ok || len(results) != 2 {
		t.Fatalf("esperava 2 resultados no batch, recebi: %+v", result)
	}
	seen := map[string]bool{}
	for _, r := range results {
		item := r.(map[string]any)
		seen[item["target"].(string)] = true
		if item["packet_loss_pct"] != 0.0 {
			t.Fatalf("esperava 0%% de perda contra listener real, obtive %v", item["packet_loss_pct"])
		}
	}
	for _, target := range targets {
		if !seen[target] {
			t.Fatalf("alvo %s não apareceu nos resultados: %+v", target, results)
		}
	}
}
