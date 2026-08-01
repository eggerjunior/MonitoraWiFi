package agent

import (
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

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
					{"id": "cmd-2", "site_id": "site-1", "type": "port_scan", "params": json.RawMessage(`{}`), "status": "claimed"},
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
