package agent

import (
	"testing"
	"time"

	"egger/local-agent/internal/probes"
)

func TestToPingTestPayload_IdempotencyKeyIsStableAndUnique(t *testing.T) {
	latency := 12.5
	loss := 0.0
	executedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	r1 := probes.Result{Target: "1.1.1.1", Protocol: "icmp", LatencyMsP50: &latency, PacketLossPct: loss, ExecutedAt: executedAt}
	r2 := probes.Result{Target: "1.1.1.1", Protocol: "icmp", LatencyMsP50: &latency, PacketLossPct: loss, ExecutedAt: executedAt}
	r3 := probes.Result{Target: "8.8.8.8", Protocol: "icmp", LatencyMsP50: &latency, PacketLossPct: loss, ExecutedAt: executedAt}

	p1 := toPingTestPayload(r1)
	p2 := toPingTestPayload(r2)
	p3 := toPingTestPayload(r3)

	if p1.IdempotencyKey != p2.IdempotencyKey {
		t.Fatalf("mesmo resultado deveria gerar a mesma chave: %q vs %q", p1.IdempotencyKey, p2.IdempotencyKey)
	}
	if p1.IdempotencyKey == p3.IdempotencyKey {
		t.Fatalf("alvos diferentes não deveriam colidir na mesma chave: %q", p1.IdempotencyKey)
	}
	if p1.Target != "1.1.1.1" || p1.Protocol != "icmp" {
		t.Fatalf("payload não preservou target/protocol: %+v", p1)
	}
}
