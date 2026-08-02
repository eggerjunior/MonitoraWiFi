package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"egger/api/internal/store"
)

func TestUniFiInventory_FullFlow(t *testing.T) {
	siteID := uuid.New()
	admin := store.User{ID: uuid.New(), Email: "admin@example.com", PasswordHash: mustHash(t, "senha12345"), Role: store.RoleAdministrator}
	deps := newAgentTestServer(admin)
	cookie := loginAndGetCookie(t, deps.server, "admin@example.com", "senha12345")

	agentID, agentSecret := enrollTestAgent(t, deps, siteID)

	// 1. Antes de qualquer sincronização, a lista deve vir vazia (honesto,
	// nunca inventado).
	emptyReq := httptest.NewRequest(http.MethodGet, "/api/v1/sites/"+siteID.String()+"/unifi/devices", nil)
	emptyReq.AddCookie(cookie)
	emptyRec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(emptyRec, emptyReq)
	var emptyResp struct {
		Items []any `json:"items"`
	}
	json.Unmarshal(emptyRec.Body.Bytes(), &emptyResp)
	if len(emptyResp.Items) != 0 {
		t.Fatalf("esperava 0 dispositivos antes da sincronização, obtive %d", len(emptyResp.Items))
	}

	// 2. Agente envia o inventário.
	inventoryBody, _ := json.Marshal(map[string]any{
		"devices": []map[string]any{
			{
				"external_id": "dev-1", "mac_address": "aa:bb:cc:dd:ee:ff", "ip_address": "192.168.110.79",
				"name": "AP Teste", "model": "U7 Pro", "state": "ONLINE", "firmware_version": "8.7.11",
				"features": []string{"accessPoint"}, "interfaces": []string{"radios"},
				"uplink_device_id": "switch-1",
			},
		},
		"clients": []map[string]any{
			{
				"external_id": "client-1", "type": "WIRED", "name": "Cliente Teste",
				"ip_address": "192.168.110.10", "mac_address": "11:22:33:44:55:66",
				"connected_at": "2026-08-01T10:00:00Z", "uplink_device_id": "dev-1",
			},
		},
	})
	invReq := httptest.NewRequest(http.MethodPost, "/api/v1/agents/"+agentID+"/unifi-inventory", bytes.NewReader(inventoryBody))
	invReq.Header.Set("Authorization", "Bearer "+agentSecret)
	invRec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(invRec, invReq)
	if invRec.Code != http.StatusOK {
		t.Fatalf("esperava 200 ao enviar inventário, recebeu %d: %s", invRec.Code, invRec.Body.String())
	}

	// 3. Usuário lista dispositivos e clientes — devem refletir o que foi enviado.
	devReq := httptest.NewRequest(http.MethodGet, "/api/v1/sites/"+siteID.String()+"/unifi/devices", nil)
	devReq.AddCookie(cookie)
	devRec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(devRec, devReq)
	var devResp struct {
		Items []map[string]any `json:"items"`
	}
	json.Unmarshal(devRec.Body.Bytes(), &devResp)
	if len(devResp.Items) != 1 || devResp.Items[0]["model"] != "U7 Pro" {
		t.Fatalf("dispositivos inesperados: %+v", devResp.Items)
	}
	if devResp.Items[0]["uplink_device_id"] != "switch-1" {
		t.Fatalf("esperava uplink_device_id=switch-1 (topologia dispositivo->dispositivo), obtive: %+v", devResp.Items[0])
	}

	clientReq := httptest.NewRequest(http.MethodGet, "/api/v1/sites/"+siteID.String()+"/unifi/clients", nil)
	clientReq.AddCookie(cookie)
	clientRec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(clientRec, clientReq)
	var clientResp struct {
		Items []map[string]any `json:"items"`
	}
	json.Unmarshal(clientRec.Body.Bytes(), &clientResp)
	if len(clientResp.Items) != 1 || clientResp.Items[0]["uplink_device_id"] != "dev-1" {
		t.Fatalf("clientes inesperados: %+v", clientResp.Items)
	}

	// 4. Uma nova sincronização substitui o inventário anterior (estado
	// atual, não acumula histórico) — enviar uma lista vazia deve zerar.
	emptyInventory, _ := json.Marshal(map[string]any{"devices": []map[string]any{}, "clients": []map[string]any{}})
	replaceReq := httptest.NewRequest(http.MethodPost, "/api/v1/agents/"+agentID+"/unifi-inventory", bytes.NewReader(emptyInventory))
	replaceReq.Header.Set("Authorization", "Bearer "+agentSecret)
	replaceRec := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(replaceRec, replaceReq)
	if replaceRec.Code != http.StatusOK {
		t.Fatalf("esperava 200 na segunda sincronização, recebeu %d", replaceRec.Code)
	}

	devReq2 := httptest.NewRequest(http.MethodGet, "/api/v1/sites/"+siteID.String()+"/unifi/devices", nil)
	devReq2.AddCookie(cookie)
	devRec2 := httptest.NewRecorder()
	deps.server.Routes().ServeHTTP(devRec2, devReq2)
	var devResp2 struct {
		Items []any `json:"items"`
	}
	json.Unmarshal(devRec2.Body.Bytes(), &devResp2)
	if len(devResp2.Items) != 0 {
		t.Fatalf("esperava 0 dispositivos após sincronização vazia (substituição, não acumulação), obtive %d", len(devResp2.Items))
	}
}
