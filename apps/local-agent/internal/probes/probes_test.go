package probes

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func fastOptions() Options {
	return Options{Attempts: 3, Timeout: time.Second, Interval: 10 * time.Millisecond}
}

func TestProbeTCP_Success(t *testing.T) {
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

	res := ProbeTCP(context.Background(), ln.Addr().String(), fastOptions())

	if res.Protocol != "tcp" {
		t.Fatalf("esperava protocol=tcp, recebeu %q", res.Protocol)
	}
	if res.PacketLossPct != 0 {
		t.Fatalf("esperava 0%% de perda contra um listener local saudável, recebeu %.1f%%", res.PacketLossPct)
	}
	if res.LatencyMsP50 == nil {
		t.Fatal("esperava latência p50 preenchida")
	}
}

func TestProbeTCP_ConnectionRefused(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("erro ao reservar porta: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close() // porta fechada de propósito — ninguém escutando

	res := ProbeTCP(context.Background(), addr, fastOptions())

	if res.PacketLossPct != 100 {
		t.Fatalf("esperava 100%% de perda contra porta fechada, recebeu %.1f%%", res.PacketLossPct)
	}
	if res.LatencyMsP50 != nil {
		t.Fatal("não deveria haver latência quando todas as tentativas falham — nunca inventar dado")
	}
}

func TestProbeHTTP_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	res := ProbeHTTP(context.Background(), server.URL, fastOptions())

	if res.PacketLossPct != 0 {
		t.Fatalf("esperava 0%% de perda contra servidor de teste saudável, recebeu %.1f%%", res.PacketLossPct)
	}
	if res.LatencyMsP50 == nil {
		t.Fatal("esperava latência p50 preenchida")
	}
}

func TestProbeHTTP_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	res := ProbeHTTP(context.Background(), server.URL, fastOptions())

	if res.PacketLossPct != 100 {
		t.Fatalf("esperava 100%% de perda quando o servidor retorna 5xx, recebeu %.1f%%", res.PacketLossPct)
	}
}

func TestProbeDNS_Success(t *testing.T) {
	res := ProbeDNS(context.Background(), "localhost", "", fastOptions())

	if res.PacketLossPct != 0 {
		t.Fatalf("esperava 0%% de perda ao resolver 'localhost', recebeu %.1f%%", res.PacketLossPct)
	}
	if res.LatencyMsP50 == nil {
		t.Fatal("esperava latência p50 preenchida")
	}
}

func TestProbeDNS_NonExistentDomain(t *testing.T) {
	res := ProbeDNS(context.Background(), "este-dominio-nao-deveria-existir-jamais.invalid", "", fastOptions())

	if res.PacketLossPct != 100 {
		t.Fatalf("esperava 100%% de perda para domínio inexistente, recebeu %.1f%%", res.PacketLossPct)
	}
}

func TestCompareDNSResolvers_MisturaSucessoEFalha(t *testing.T) {
	// "localhost" não serve de alvo aqui: o resolvedor Go trata nomes
	// presentes em /etc/hosts como caso especial e nunca chega a discar o
	// endereço de resolvedor customizado, o que mascararia o próprio
	// comportamento sendo testado. Usa um hostname real da internet — este
	// ambiente de desenvolvimento já confirmou ter acesso real à internet
	// (mesma premissa da validação de RDAP/WHOIS, Fase 5).
	resolvers := []ResolverEntry{
		{Name: "sistema (padrão da rede)", Addr: ""},
		// 203.0.113.1 é TEST-NET-3 (RFC 5737, documentação) — nunca roteável,
		// garante timeout/falha determinística sem depender de nenhum
		// resolvedor real estar de fato inalcançável por acaso.
		{Name: "inalcançável (RFC 5737)", Addr: "203.0.113.1:53"},
	}

	results := CompareDNSResolvers(context.Background(), "cloudflare.com", resolvers, 3*time.Second)

	if len(results) != 2 {
		t.Fatalf("esperava 2 resultados, recebeu %d", len(results))
	}

	sistema := results[0]
	if sistema.Resolver != "sistema (padrão da rede)" {
		t.Fatalf("resolver inesperado no índice 0: %q", sistema.Resolver)
	}
	if sistema.Error != "" {
		t.Fatalf("esperava sucesso resolvendo 'cloudflare.com' pelo sistema, recebeu erro: %s", sistema.Error)
	}
	if len(sistema.Addresses) == 0 {
		t.Fatal("esperava ao menos um endereço resolvido pelo sistema — nunca inventar dado, mas também não esconder sucesso real")
	}

	inalcancavel := results[1]
	if inalcancavel.Error == "" {
		t.Fatal("esperava erro explícito contra um resolvedor inalcançável — nunca inventar endereço quando a resolução falha")
	}
	if len(inalcancavel.Addresses) != 0 {
		t.Fatalf("não deveria haver endereços quando a resolução falhou, recebeu %v", inalcancavel.Addresses)
	}
}

func TestSummarize_NeverInventsLatencyWithZeroSamples(t *testing.T) {
	res := summarize("alvo", "tcp", nil, 5, time.Now())
	if res.LatencyMsP50 != nil || res.LatencyMsP95 != nil || res.LatencyMsP99 != nil || res.JitterMs != nil {
		t.Fatal("sem amostras, nenhum campo de latência/jitter deve ser preenchido — princípio de nunca inventar dado (Seção 2.1)")
	}
	if res.PacketLossPct != 100 {
		t.Fatalf("esperava 100%% de perda com zero amostras de 5 tentativas, recebeu %.1f%%", res.PacketLossPct)
	}
}

func TestSummarize_PercentilesAreOrdered(t *testing.T) {
	samples := []float64{10, 12, 11, 50, 13, 9, 14}
	res := summarize("alvo", "tcp", samples, len(samples), time.Now())

	if *res.LatencyMsP50 > *res.LatencyMsP95 || *res.LatencyMsP95 > *res.LatencyMsP99 {
		t.Fatalf("percentis fora de ordem: p50=%.2f p95=%.2f p99=%.2f", *res.LatencyMsP50, *res.LatencyMsP95, *res.LatencyMsP99)
	}
}
