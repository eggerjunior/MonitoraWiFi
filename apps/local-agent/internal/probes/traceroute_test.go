package probes

import (
	"context"
	"testing"
	"time"
)

// TestTraceroute_Loopback roda um traceroute real contra 127.0.0.1 — deve
// alcançar o destino no primeiro salto (echo reply imediato, sem
// roteadores intermediários). Se o ambiente não tiver permissão ICMP
// (net.ipv4.ping_group_range/CAP_NET_RAW), pula explicitamente — mesmo
// padrão de icmp.go (nunca finge sucesso sem a sonda real funcionar).
func TestTraceroute_Loopback(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := Traceroute(ctx, "127.0.0.1", 5, 2*time.Second)

	if result.Error != "" {
		t.Skipf("ICMP indisponível neste ambiente de teste: %s", result.Error)
	}
	if !result.Reached {
		t.Fatalf("esperava alcançar 127.0.0.1, hops: %+v", result.Hops)
	}
	if len(result.Hops) != 1 {
		t.Fatalf("esperava 1 salto até o loopback, obtive %d: %+v", len(result.Hops), result.Hops)
	}
	hop := result.Hops[0]
	if hop.Address != "127.0.0.1" {
		t.Errorf("endereço do salto = %q, esperado 127.0.0.1", hop.Address)
	}
	if hop.RTTMs == nil {
		t.Error("esperava RTT medido para o salto que respondeu")
	}
}

func TestTraceroute_InvalidTarget(t *testing.T) {
	ctx := context.Background()
	result := Traceroute(ctx, "este-hostname-nao-existe.invalid", 5, 1*time.Second)
	if result.Error == "" && result.Reached {
		t.Fatal("esperava erro ou falha ao resolver um hostname inválido")
	}
}
