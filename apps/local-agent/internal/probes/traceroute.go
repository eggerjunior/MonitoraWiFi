// Traceroute (Fase 5): reusa o mesmo socket ICMP não privilegiado de
// icmp.go (SOCK_DGRAM "udp4" — depende de net.ipv4.ping_group_range/
// CAP_NET_RAW, melhor esforço). Incrementa o TTL a cada salto e escuta por
// "Time Exceeded" (roteador intermediário) ou "Echo Reply" (destino) —
// nunca inventa um salto sem resposta real: reporta como timeout.
package probes

import (
	"context"
	"net"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type TracerouteHop struct {
	Hop     int
	Address string // vazio se o salto não respondeu (timeout)
	RTTMs   *float64
}

type TracerouteResult struct {
	Target     string
	Hops       []TracerouteHop
	Reached    bool
	ExecutedAt time.Time
	Error      string // preenchido só se o traceroute nem pôde começar (ex.: sem permissão ICMP)
}

func Traceroute(ctx context.Context, target string, maxHops int, perHopTimeout time.Duration) TracerouteResult {
	executedAt := time.Now().UTC()
	result := TracerouteResult{Target: target, ExecutedAt: executedAt}

	conn, err := icmp.ListenPacket("udp4", "0.0.0.0")
	if err != nil {
		result.Error = "sem permissão para ICMP neste host (net.ipv4.ping_group_range/CAP_NET_RAW)"
		return result
	}
	defer conn.Close()

	pconn := conn.IPv4PacketConn()
	if pconn == nil {
		result.Error = "socket ICMP não suporta IPv4 packet control neste host"
		return result
	}

	dst, err := net.ResolveIPAddr("ip4", target)
	if err != nil {
		result.Error = "não foi possível resolver o alvo: " + err.Error()
		return result
	}

	for ttl := 1; ttl <= maxHops; ttl++ {
		if ctx.Err() != nil {
			break
		}
		if err := pconn.SetTTL(ttl); err != nil {
			break
		}

		hop := TracerouteHop{Hop: ttl}
		msg := icmp.Message{
			Type: ipv4.ICMPTypeEcho,
			Code: 0,
			Body: &icmp.Echo{
				ID:   os.Getpid() & 0xffff,
				Seq:  ttl,
				Data: []byte("egger-traceroute"),
			},
		}
		wb, err := msg.Marshal(nil)
		if err != nil {
			result.Hops = append(result.Hops, hop)
			continue
		}

		start := time.Now()
		if _, err := conn.WriteTo(wb, &net.UDPAddr{IP: dst.IP}); err == nil {
			_ = conn.SetReadDeadline(start.Add(perHopTimeout))
			rb := make([]byte, 1500)
			n, peer, err := conn.ReadFrom(rb)
			if err == nil {
				rtt := float64(time.Since(start).Microseconds()) / 1000
				hop.RTTMs = &rtt
				if udpAddr, ok := peer.(*net.UDPAddr); ok {
					hop.Address = udpAddr.IP.String()
				}
				if parsed, parseErr := icmp.ParseMessage(1, rb[:n]); parseErr == nil {
					if parsed.Type == ipv4.ICMPTypeEchoReply {
						result.Hops = append(result.Hops, hop)
						result.Reached = true
						return result
					}
				}
			}
		}
		result.Hops = append(result.Hops, hop)
	}

	return result
}
