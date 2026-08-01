package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_Enroll(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/agents/enroll" || r.Method != http.MethodPost {
			t.Errorf("requisição inesperada: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["enrollment_token"] != "tok-123" {
			t.Errorf("esperava enrollment_token=tok-123, recebeu %q", body["enrollment_token"])
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(EnrollResult{AgentID: "agent-1", AgentSecret: "secret-1", SiteID: "site-1"})
	}))
	defer server.Close()

	client := New(server.URL)
	result, err := client.Enroll(context.Background(), "tok-123", "host-1", "linux_amd64", "0.1.0")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if result.AgentID != "agent-1" || result.AgentSecret != "secret-1" {
		t.Fatalf("resultado inesperado: %+v", result)
	}
}

func TestClient_Enroll_InvalidToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid_token", "message": "token inválido"})
	}))
	defer server.Close()

	client := New(server.URL)
	_, err := client.Enroll(context.Background(), "tok-invalido", "host-1", "linux_amd64", "0.1.0")
	if err == nil {
		t.Fatal("esperava erro para token inválido")
	}
	apiErr, ok := err.(*ApiError)
	if !ok {
		t.Fatalf("esperava *ApiError, recebeu %T", err)
	}
	if apiErr.Status != http.StatusUnauthorized {
		t.Fatalf("esperava status 401, recebeu %d", apiErr.Status)
	}
}

func TestClient_Heartbeat_SendsBearerToken(t *testing.T) {
	var receivedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := New(server.URL)
	err := client.Heartbeat(context.Background(), "agent-1", "meu-secret", HeartbeatPayload{Status: "ok"})
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if receivedAuth != "Bearer meu-secret" {
		t.Fatalf("esperava header Authorization Bearer, recebeu %q", receivedAuth)
	}
}

func TestClient_SendTelemetry(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		tests, _ := body["ping_tests"].([]any)
		if len(tests) != 1 {
			t.Errorf("esperava 1 ping_test no corpo, recebeu %d", len(tests))
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	client := New(server.URL)
	err := client.SendTelemetry(context.Background(), "agent-1", "secret", []PingTestPayload{
		{Target: "1.1.1.1", Protocol: "icmp", ExecutedAt: "2026-01-01T00:00:00Z", IdempotencyKey: "k1"},
	})
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
}

func TestClient_ClaimCommands(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/agents/agent-1/commands" || r.Method != http.MethodGet {
			t.Errorf("requisição inesperada: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer meu-secret" {
			t.Errorf("esperava Authorization Bearer, recebeu %q", r.Header.Get("Authorization"))
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{"id": "cmd-1", "site_id": "site-1", "type": "ping", "params": json.RawMessage(`{"target":"1.1.1.1","protocol":"icmp"}`), "status": "claimed"},
			},
		})
	}))
	defer server.Close()

	client := New(server.URL)
	cmds, err := client.ClaimCommands(context.Background(), "agent-1", "meu-secret")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if len(cmds) != 1 || cmds[0].ID != "cmd-1" || cmds[0].Type != "ping" {
		t.Fatalf("resultado inesperado: %+v", cmds)
	}
}

func TestClient_ClaimCommands_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"items": []map[string]any{}})
	}))
	defer server.Close()

	client := New(server.URL)
	cmds, err := client.ClaimCommands(context.Background(), "agent-1", "meu-secret")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if len(cmds) != 0 {
		t.Fatalf("esperava 0 comandos, obtive %d", len(cmds))
	}
}

func TestClient_ReportCommandResult(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/agents/agent-1/commands/cmd-1/result" || r.Method != http.MethodPost {
			t.Errorf("requisição inesperada: %s %s", r.Method, r.URL.Path)
		}
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := New(server.URL)
	err := client.ReportCommandResult(context.Background(), "agent-1", "meu-secret", "cmd-1", "completed",
		map[string]any{"latency_ms_p50": 12.3}, "")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if received["status"] != "completed" {
		t.Fatalf("status enviado = %v, esperado completed", received["status"])
	}
	result, ok := received["result"].(map[string]any)
	if !ok || result["latency_ms_p50"] != 12.3 {
		t.Fatalf("result enviado inesperado: %+v", received)
	}
	if _, hasError := received["error"]; hasError {
		t.Fatalf("não esperava campo error quando errMsg é vazio, recebi: %+v", received)
	}
}

func TestClient_ReportCommandResult_Failure(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := New(server.URL)
	err := client.ReportCommandResult(context.Background(), "agent-1", "meu-secret", "cmd-1", "failed", nil, "alvo inválido")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if received["status"] != "failed" || received["error"] != "alvo inválido" {
		t.Fatalf("corpo enviado inesperado: %+v", received)
	}
	if _, hasResult := received["result"]; hasResult {
		t.Fatalf("não esperava campo result quando result é nil, recebi: %+v", received)
	}
}
