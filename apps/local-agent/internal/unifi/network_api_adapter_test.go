package unifi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNetworkAPIAdapter_ListSites(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-KEY") != "test-key" {
			t.Errorf("esperava X-API-KEY=test-key, recebeu %q", r.Header.Get("X-API-KEY"))
		}
		if r.URL.Path != "/proxy/network/integration/v1/sites" {
			t.Errorf("path inesperado: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"offset": 0, "limit": 25, "count": 1, "totalCount": 1,
			"data": []map[string]any{
				{"id": "site-1", "internalReference": "default", "name": "Default"},
			},
		})
	}))
	defer server.Close()

	adapter := NewNetworkAPIAdapter(server.URL, "test-key")
	sites, err := adapter.ListSites(context.Background())
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if len(sites) != 1 || sites[0].ID != "site-1" || sites[0].Name != "Default" {
		t.Fatalf("resultado inesperado: %+v", sites)
	}
}

func TestNetworkAPIAdapter_ListDevices(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/proxy/network/integration/v1/sites/site-1/devices" {
			t.Errorf("path inesperado: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"offset": 0, "limit": 25, "count": 1, "totalCount": 1,
			"data": []map[string]any{
				{
					"id": "dev-1", "macAddress": "aa:bb:cc:dd:ee:ff", "ipAddress": "192.168.110.79",
					"name": "AP Exemplo", "model": "U7 Pro", "state": "ONLINE",
					"firmwareVersion": "8.7.11", "firmwareUpdatable": false,
					"features": []string{"accessPoint"}, "interfaces": []string{"radios"},
				},
			},
		})
	}))
	defer server.Close()

	adapter := NewNetworkAPIAdapter(server.URL, "test-key")
	devices, err := adapter.ListDevices(context.Background(), "site-1")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if len(devices) != 1 {
		t.Fatalf("esperava 1 dispositivo, obtive %d", len(devices))
	}
	d := devices[0]
	if d.Model != "U7 Pro" || d.FirmwareVersion != "8.7.11" || d.State != "ONLINE" {
		t.Fatalf("dispositivo inesperado: %+v", d)
	}
	if len(d.Features) != 1 || d.Features[0] != "accessPoint" {
		t.Fatalf("features inesperadas: %+v", d.Features)
	}
}

func TestNetworkAPIAdapter_ListClients_Paginates(t *testing.T) {
	callCount := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		offset := r.URL.Query().Get("offset")
		var data []map[string]any
		var count int
		switch offset {
		case "0":
			for i := 0; i < 2; i++ {
				data = append(data, map[string]any{
					"type": "WIRED", "id": "client-" + string(rune('a'+i)), "name": "Cliente",
					"connectedAt": "2026-08-01T10:00:00Z", "ipAddress": "192.168.110.10",
					"macAddress": "aa:bb:cc:dd:ee:0" + string(rune('0'+i)), "uplinkDeviceId": "dev-1",
				})
			}
			count = 2
		default:
			data = nil
			count = 0
		}
		json.NewEncoder(w).Encode(map[string]any{
			"offset": 0, "limit": 200, "count": count, "totalCount": 2, "data": data,
		})
	}))
	defer server.Close()

	adapter := NewNetworkAPIAdapter(server.URL, "test-key")
	clients, err := adapter.ListClients(context.Background(), "site-1")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if len(clients) != 2 {
		t.Fatalf("esperava 2 clientes (paginação completa), obtive %d", len(clients))
	}
	if callCount != 1 {
		t.Fatalf("esperava 1 chamada (totalCount atingido na primeira página), fez %d", callCount)
	}
}

func TestNetworkAPIAdapter_ErrorStatus(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{"error": "unauthorized"})
	}))
	defer server.Close()

	adapter := NewNetworkAPIAdapter(server.URL, "chave-errada")
	_, err := adapter.ListSites(context.Background())
	if err == nil {
		t.Fatal("esperava erro para status 401")
	}
}
