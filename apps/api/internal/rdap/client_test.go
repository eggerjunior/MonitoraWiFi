package rdap

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_LookupDomain(t *testing.T) {
	rdapServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/domain/example.test" {
			t.Errorf("path inesperado: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/rdap+json")
		json.NewEncoder(w).Encode(map[string]any{
			"objectClassName": "domain",
			"handle":          "EXAMPLE-TEST",
			"ldhName":         "EXAMPLE.TEST",
			"status":          []string{"active"},
			"events": []map[string]string{
				{"eventAction": "registration", "eventDate": "2020-01-01T00:00:00Z"},
			},
			"nameservers": []map[string]string{
				{"ldhName": "ns1.example.test"},
				{"ldhName": "ns2.example.test"},
			},
		})
	}))
	defer rdapServer.Close()

	bootstrapServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"services": []any{
				[]any{[]string{"test"}, []string{rdapServer.URL}},
			},
		})
	}))
	defer bootstrapServer.Close()

	client := NewClientWithBootstrapURLs(bootstrapServer.URL, "http://unused.invalid/ipv4.json", "http://unused.invalid/ipv6.json")
	result, err := client.Lookup(t.Context(), "example.test")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if result.Name != "EXAMPLE.TEST" {
		t.Errorf("Name = %q, esperado EXAMPLE.TEST", result.Name)
	}
	if result.Handle != "EXAMPLE-TEST" {
		t.Errorf("Handle = %q, esperado EXAMPLE-TEST", result.Handle)
	}
	if len(result.Events) != 1 || result.Events[0].Action != "registration" {
		t.Errorf("Events inesperado: %+v", result.Events)
	}
	if len(result.Nameservers) != 2 {
		t.Errorf("Nameservers inesperado: %+v", result.Nameservers)
	}
	if len(result.Status) != 1 || result.Status[0] != "active" {
		t.Errorf("Status inesperado: %+v", result.Status)
	}
}

func TestClient_LookupIP(t *testing.T) {
	rdapServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ip/192.0.2.1" {
			t.Errorf("path inesperado: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"objectClassName": "ip network",
			"handle":          "TEST-NET-1",
			"name":            "TEST-NET-1-BLOCK",
		})
	}))
	defer rdapServer.Close()

	bootstrapServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"services": []any{
				[]any{[]string{"192.0.2.0/24"}, []string{rdapServer.URL}},
			},
		})
	}))
	defer bootstrapServer.Close()

	client := NewClientWithBootstrapURLs("http://unused.invalid/dns.json", bootstrapServer.URL, "http://unused.invalid/ipv6.json")
	result, err := client.Lookup(t.Context(), "192.0.2.1")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if result.Name != "TEST-NET-1-BLOCK" {
		t.Errorf("Name = %q, esperado TEST-NET-1-BLOCK", result.Name)
	}
	if result.Handle != "TEST-NET-1" {
		t.Errorf("Handle = %q, esperado TEST-NET-1", result.Handle)
	}
}

func TestClient_NoServerFound(t *testing.T) {
	bootstrapServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"services": []any{}})
	}))
	defer bootstrapServer.Close()

	client := NewClientWithBootstrapURLs(bootstrapServer.URL, bootstrapServer.URL, bootstrapServer.URL)
	_, err := client.Lookup(t.Context(), "example.semservidor")
	if err == nil {
		t.Fatal("esperava erro quando nenhum servidor RDAP é encontrado")
	}
}

func TestClient_BootstrapCache(t *testing.T) {
	rdapServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"objectClassName": "domain", "ldhName": "CACHE.TEST"})
	}))
	defer rdapServer.Close()

	fetchCount := 0
	bootstrapServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fetchCount++
		json.NewEncoder(w).Encode(map[string]any{
			"services": []any{
				[]any{[]string{"cache"}, []string{rdapServer.URL}},
			},
		})
	}))
	defer bootstrapServer.Close()

	client := NewClientWithBootstrapURLs(bootstrapServer.URL, "http://unused.invalid/ipv4.json", "http://unused.invalid/ipv6.json")
	if _, err := client.Lookup(t.Context(), "a.cache"); err != nil {
		t.Fatalf("erro na primeira consulta: %v", err)
	}
	if _, err := client.Lookup(t.Context(), "b.cache"); err != nil {
		t.Fatalf("erro na segunda consulta: %v", err)
	}
	if fetchCount != 1 {
		t.Fatalf("esperava 1 busca ao bootstrap (cache reaproveitado), obtive %d", fetchCount)
	}
}
