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
