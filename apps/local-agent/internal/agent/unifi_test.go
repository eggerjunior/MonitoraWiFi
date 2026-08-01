package agent

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"egger/local-agent/internal/config"
	"egger/local-agent/internal/state"
)

// TestSyncUniFiOnce exercita o fluxo real: adaptador de verdade
// (NetworkAPIAdapter) contra um servidor TLS de teste simulando o console,
// e o agente enviando o resultado a um backend de teste real — só as duas
// pontas externas (console UniFi, backend) são substituídas por servidores
// de teste; a lógica de conversão e envio é a de produção.
func TestSyncUniFiOnce(t *testing.T) {
	uniFiServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/proxy/network/integration/v1/sites/site-1/devices":
			json.NewEncoder(w).Encode(map[string]any{
				"offset": 0, "limit": 25, "count": 1, "totalCount": 1,
				"data": []map[string]any{
					{
						"id": "dev-1", "macAddress": "aa:bb:cc:dd:ee:ff", "ipAddress": "192.168.110.79",
						"name": "AP Teste", "model": "U7 Pro", "state": "ONLINE",
						"firmwareVersion": "8.7.11", "firmwareUpdatable": false,
						"features": []string{"accessPoint"}, "interfaces": []string{"radios"},
					},
				},
			})
		case "/proxy/network/integration/v1/sites/site-1/clients":
			json.NewEncoder(w).Encode(map[string]any{
				"offset": 0, "limit": 200, "count": 1, "totalCount": 1,
				"data": []map[string]any{
					{
						"type": "WIRED", "id": "client-1", "name": "Cliente Teste",
						"connectedAt": "2026-08-01T10:00:00Z", "ipAddress": "192.168.110.10",
						"macAddress": "11:22:33:44:55:66", "uplinkDeviceId": "dev-1",
					},
				},
			})
		default:
			t.Errorf("path inesperado no servidor UniFi de teste: %s", r.URL.Path)
		}
	}))
	defer uniFiServer.Close()

	var receivedBody map[string]any
	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/agents/agent-1/unifi-inventory" {
			json.NewDecoder(r.Body).Decode(&receivedBody)
			w.WriteHeader(http.StatusOK)
			return
		}
		t.Errorf("requisição inesperada ao backend de teste: %s %s", r.Method, r.URL.Path)
	}))
	defer backendServer.Close()

	statePath := filepath.Join(t.TempDir(), "agent.json")
	if err := state.Save(statePath, state.Identity{AgentID: "agent-1", AgentSecret: "secret-1", SiteID: "site-1"}); err != nil {
		t.Fatalf("erro ao preparar estado: %v", err)
	}

	cfg := config.Config{
		BackendURL:    backendServer.URL,
		StateFilePath: statePath,
		UniFiEnabled:  true,
		UniFiBaseURL:  uniFiServer.URL,
		UniFiAPIKey:   "test-key",
		UniFiSiteID:   "site-1",
	}

	a, err := New(t.Context(), cfg, testLogger())
	if err != nil {
		t.Fatalf("erro ao construir agente: %v", err)
	}
	if a.unifiProvider == nil {
		t.Fatal("esperava unifiProvider configurado quando UniFiEnabled=true")
	}

	a.syncUniFiOnce(t.Context())

	if receivedBody == nil {
		t.Fatal("esperava que o agente enviasse o inventário ao backend")
	}
	devices, ok := receivedBody["devices"].([]any)
	if !ok || len(devices) != 1 {
		t.Fatalf("devices inesperado: %+v", receivedBody["devices"])
	}
	device := devices[0].(map[string]any)
	if device["model"] != "U7 Pro" || device["firmware_version"] != "8.7.11" {
		t.Fatalf("payload de dispositivo inesperado: %+v", device)
	}

	clients, ok := receivedBody["clients"].([]any)
	if !ok || len(clients) != 1 {
		t.Fatalf("clients inesperado: %+v", receivedBody["clients"])
	}
	client := clients[0].(map[string]any)
	if client["uplink_device_id"] != "dev-1" {
		t.Fatalf("payload de cliente inesperado: %+v", client)
	}
}

func TestUniFiDisabled_NoProvider(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "agent.json")
	if err := state.Save(statePath, state.Identity{AgentID: "agent-1", AgentSecret: "secret-1", SiteID: "site-1"}); err != nil {
		t.Fatalf("erro ao preparar estado: %v", err)
	}

	cfg := config.Config{
		BackendURL:    "http://localhost:0",
		StateFilePath: statePath,
	}

	a, err := New(t.Context(), cfg, testLogger())
	if err != nil {
		t.Fatalf("erro ao construir agente: %v", err)
	}
	if a.unifiProvider != nil {
		t.Fatal("esperava unifiProvider nil quando UNIFI_* não está configurado")
	}
}
