package agent

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"egger/local-agent/internal/probes"
)

// TestProbeByProtocol_TodosOsProtocolos exercita o dispatcher de protocolo
// diretamente (sem passar pela fila de comando completa) contra sondas
// reais — cada um dos 4 protocolos suportados, mais o caso de protocolo
// desconhecido.
func TestProbeByProtocol_TodosOsProtocolos(t *testing.T) {
	opts := probes.Options{Attempts: 1, Timeout: 2 * time.Second}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	result, ok := probeByProtocol(t.Context(), server.URL, "http", opts)
	if !ok {
		t.Fatal("esperava ok=true para protocolo http")
	}
	if result.Protocol != "http" {
		t.Fatalf("result.Protocol = %q, esperado http", result.Protocol)
	}

	result, ok = probeByProtocol(t.Context(), "localhost", "dns", opts)
	if !ok {
		t.Fatal("esperava ok=true para protocolo dns")
	}
	if result.Protocol != "dns" {
		t.Fatalf("result.Protocol = %q, esperado dns", result.Protocol)
	}

	_, ok = probeByProtocol(t.Context(), "1.1.1.1", "icmp", opts)
	if !ok {
		t.Fatal("esperava ok=true para protocolo icmp (independente de o ICMP ter privilégio no ambiente)")
	}

	_, ok = probeByProtocol(t.Context(), "1.1.1.1:80", "tcp", opts)
	if !ok {
		t.Fatal("esperava ok=true para protocolo tcp")
	}

	_, ok = probeByProtocol(t.Context(), "1.1.1.1", "carrier-pigeon", opts)
	if ok {
		t.Fatal("esperava ok=false para protocolo desconhecido")
	}
}
