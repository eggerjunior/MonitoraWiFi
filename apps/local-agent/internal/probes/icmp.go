package probes

import (
	"context"
	"net"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

// ProbeICMP usa um socket ICMP não privilegiado (SOCK_DGRAM), que no Linux
// exige que o processo tenha permissão de acordo com
// net.ipv4.ping_group_range (glibc/kernel) ou a capability CAP_NET_RAW —
// nem sempre disponível (ex.: dentro de um container sem essa capability).
// Quando indisponível, o erro é reportado como perda de 100% — nunca
// inventamos uma latência.
func ProbeICMP(ctx context.Context, target string, opts Options) Result {
	executedAt := time.Now().UTC()
	var samples []float64

	conn, err := icmp.ListenPacket("udp4", "0.0.0.0")
	if err != nil {
		// Sem permissão para ICMP neste ambiente — reporta perda total em
		// vez de tentar um fallback silencioso que mascare a causa real.
		return summarize(target, "icmp", nil, opts.Attempts, executedAt)
	}
	defer conn.Close()

	dst, err := net.ResolveIPAddr("ip4", target)
	if err != nil {
		return summarize(target, "icmp", nil, opts.Attempts, executedAt)
	}

	for i := 0; i < opts.Attempts; i++ {
		if ctx.Err() != nil {
			break
		}

		msg := icmp.Message{
			Type: ipv4.ICMPTypeEcho,
			Code: 0,
			Body: &icmp.Echo{
				ID:   os.Getpid() & 0xffff,
				Seq:  i + 1,
				Data: []byte("egger-network-intelligence"),
			},
		}
		wb, err := msg.Marshal(nil)
		if err != nil {
			continue
		}

		start := time.Now()
		if _, err := conn.WriteTo(wb, &net.UDPAddr{IP: dst.IP}); err == nil {
			_ = conn.SetReadDeadline(start.Add(opts.Timeout))
			rb := make([]byte, 1500)
			if n, _, err := conn.ReadFrom(rb); err == nil {
				if _, parseErr := icmp.ParseMessage(1, rb[:n]); parseErr == nil {
					samples = append(samples, float64(time.Since(start).Microseconds())/1000)
				}
			}
		}

		if i < opts.Attempts-1 {
			time.Sleep(opts.Interval)
		}
	}

	return summarize(target, "icmp", samples, opts.Attempts, executedAt)
}
