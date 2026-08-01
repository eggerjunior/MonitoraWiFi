package probes

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

// TestCheckTLS_CertificadoAutoassinado usa o servidor TLS real do
// httptest (handshake de verdade, não mockado) — o certificado é
// autoassinado, então a verificação de cadeia deve falhar de forma
// honesta (ValidNow=false), mas os metadados do certificado (emissor,
// validade) precisam vir da conexão real.
func TestCheckTLS_CertificadoAutoassinado(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	host, portStr, err := net.SplitHostPort(server.Listener.Addr().String())
	if err != nil {
		t.Fatalf("erro ao separar host/porta: %v", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("porta inválida: %v", err)
	}

	result := CheckTLS(t.Context(), host, port, 2*time.Second)

	if !result.Reached {
		t.Fatalf("esperava handshake TLS bem-sucedido, obtive erro: %s", result.Error)
	}
	if result.ValidNow {
		t.Fatal("esperava ValidNow=false para certificado autoassinado não confiável")
	}
	if result.VerifyError == "" {
		t.Fatal("esperava VerifyError explicando por que a cadeia não é confiável")
	}
	if result.NotAfter.IsZero() || result.NotBefore.IsZero() {
		t.Fatal("esperava NotBefore/NotAfter reais extraídos do certificado apresentado")
	}
	if result.Issuer == "" {
		t.Fatal("esperava Issuer extraído do certificado real")
	}
}

func TestCheckTLS_ConexaoRecusada(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("erro ao abrir listener: %v", err)
	}
	host, portStr, _ := net.SplitHostPort(ln.Addr().String())
	port, _ := strconv.Atoi(portStr)
	ln.Close() // fecha imediatamente para garantir "connection refused" real

	result := CheckTLS(t.Context(), host, port, 2*time.Second)

	if result.Reached {
		t.Fatal("esperava Reached=false contra uma porta fechada")
	}
	if result.Error == "" {
		t.Fatal("esperava mensagem de erro de conexão recusada")
	}
}
