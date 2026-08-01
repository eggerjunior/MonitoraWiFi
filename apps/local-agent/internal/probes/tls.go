package probes

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"strconv"
	"time"
)

// TLSCheckResult é o resultado de uma verificação de certificado TLS sob
// demanda (Fase 5: "SSL/TLS checker"). O handshake sempre é feito com
// InsecureSkipVerify — o objetivo é inspecionar o certificado apresentado
// mesmo quando ele é inválido/expirado/autoassinado, e então validar a
// cadeia manualmente contra os certificados raiz do sistema, nunca
// reportando "válido" sem essa checagem explícita.
type TLSCheckResult struct {
	Target          string
	Port            int
	Reached         bool
	ValidNow        bool
	VerifyError     string
	NotBefore       time.Time
	NotAfter        time.Time
	DaysUntilExpiry int
	Issuer          string
	Subject         string
	DNSNames        []string
	Error           string
	ExecutedAt      time.Time
}

// CheckTLS conecta em host:port via TLS e inspeciona o certificado
// apresentado pelo servidor — nunca simula validade/expiração a partir de
// um valor inventado (Seção 2.1).
func CheckTLS(ctx context.Context, host string, port int, timeout time.Duration) TLSCheckResult {
	executedAt := time.Now().UTC()
	result := TLSCheckResult{Target: host, Port: port, ExecutedAt: executedAt}

	dialer := &net.Dialer{Timeout: timeout}
	addr := net.JoinHostPort(host, strconv.Itoa(port))

	rawConn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	conn := tls.Client(rawConn, &tls.Config{
		InsecureSkipVerify: true, //nolint:gosec // handshake propositalmente sem verificação automática — a cadeia é verificada manualmente abaixo, sempre.
		ServerName:         host,
	})
	defer conn.Close()

	if err := conn.HandshakeContext(ctx); err != nil {
		result.Error = "falha no handshake TLS: " + err.Error()
		return result
	}
	result.Reached = true

	state := conn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		result.Error = "servidor não apresentou certificado"
		return result
	}

	cert := state.PeerCertificates[0]
	result.NotBefore = cert.NotBefore
	result.NotAfter = cert.NotAfter
	result.DaysUntilExpiry = int(time.Until(cert.NotAfter).Hours() / 24)
	result.Issuer = cert.Issuer.String()
	result.Subject = cert.Subject.String()
	result.DNSNames = cert.DNSNames

	opts := x509.VerifyOptions{
		DNSName:       host,
		Intermediates: x509.NewCertPool(),
		CurrentTime:   time.Now(),
	}
	for _, intermediate := range state.PeerCertificates[1:] {
		opts.Intermediates.AddCert(intermediate)
	}
	if _, err := cert.Verify(opts); err != nil {
		result.VerifyError = err.Error()
	} else {
		result.ValidNow = true
	}

	return result
}
