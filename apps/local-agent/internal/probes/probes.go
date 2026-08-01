// Package probes implementa os testes ativos do agente (Seção 5 do
// documento-fonte): TCP, HTTP e DNS aqui; ICMP em icmp.go (melhor esforço,
// depende de privilégio do SO — ver comentário lá).
//
// Todo probe devolve amostras de latência em milissegundos e uma taxa de
// perda — nunca inventa um valor quando a medição falha; falhas contam para
// packet_loss_pct, não são silenciadas.
package probes

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"sort"
	"time"
)

type Result struct {
	Target        string
	Protocol      string // icmp | tcp | http | dns
	LatencyMsP50  *float64
	LatencyMsP95  *float64
	LatencyMsP99  *float64
	JitterMs      *float64
	PacketLossPct float64
	ExecutedAt    time.Time
}

// Options controla uma rodada de amostragem (Seção 5: "Percentis p50, p95 e
// p99; Outliers; Variação temporal").
type Options struct {
	Attempts int           // quantas amostras coletar
	Timeout  time.Duration // timeout por tentativa
	Interval time.Duration // espera entre tentativas
}

func DefaultOptions() Options {
	return Options{Attempts: 5, Timeout: 2 * time.Second, Interval: 200 * time.Millisecond}
}

// summarize calcula p50/p95/p99/jitter a partir de amostras de latência em
// milissegundos e a taxa de perda a partir de quantas tentativas falharam.
func summarize(target, protocol string, samples []float64, attempts int, executedAt time.Time) Result {
	loss := float64(attempts-len(samples)) / float64(attempts) * 100

	res := Result{
		Target:        target,
		Protocol:      protocol,
		PacketLossPct: loss,
		ExecutedAt:    executedAt,
	}
	if len(samples) == 0 {
		return res
	}

	sorted := append([]float64(nil), samples...)
	sort.Float64s(sorted)

	p50 := percentile(sorted, 50)
	p95 := percentile(sorted, 95)
	p99 := percentile(sorted, 99)
	res.LatencyMsP50 = &p50
	res.LatencyMsP95 = &p95
	res.LatencyMsP99 = &p99

	if len(sorted) > 1 {
		jitter := meanAbsoluteDeviation(samples)
		res.JitterMs = &jitter
	}

	return res
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 1 {
		return sorted[0]
	}
	rank := p / 100 * float64(len(sorted)-1)
	lower := int(rank)
	upper := lower + 1
	if upper >= len(sorted) {
		return sorted[len(sorted)-1]
	}
	frac := rank - float64(lower)
	return sorted[lower] + (sorted[upper]-sorted[lower])*frac
}

// meanAbsoluteDeviation entre amostras consecutivas — uma medida simples de
// jitter (variação de latência entre uma amostra e a seguinte).
func meanAbsoluteDeviation(samples []float64) float64 {
	if len(samples) < 2 {
		return 0
	}
	var sum float64
	for i := 1; i < len(samples); i++ {
		diff := samples[i] - samples[i-1]
		if diff < 0 {
			diff = -diff
		}
		sum += diff
	}
	return sum / float64(len(samples)-1)
}

// ProbeTCP mede o tempo de estabelecimento de conexão TCP (handshake).
func ProbeTCP(ctx context.Context, target string, opts Options) Result {
	executedAt := time.Now().UTC()
	var samples []float64

	for i := 0; i < opts.Attempts; i++ {
		attemptCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
		start := time.Now()
		conn, err := (&net.Dialer{}).DialContext(attemptCtx, "tcp", target)
		elapsed := time.Since(start)
		cancel()
		if err == nil {
			samples = append(samples, float64(elapsed.Microseconds())/1000)
			conn.Close()
		}
		if i < opts.Attempts-1 {
			time.Sleep(opts.Interval)
		}
	}

	return summarize(target, "tcp", samples, opts.Attempts, executedAt)
}

// ProbeHTTP mede o tempo total até o primeiro byte da resposta (TTFB
// aproximado, incluindo DNS+connect+TLS — Seção 5 trata cada etapa em
// separado num "HTTP Client" completo; aqui é o teste de disponibilidade
// simples).
func ProbeHTTP(ctx context.Context, url string, opts Options) Result {
	executedAt := time.Now().UTC()
	var samples []float64

	client := &http.Client{
		Timeout: opts.Timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
		},
	}

	for i := 0; i < opts.Attempts; i++ {
		attemptCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
		req, err := http.NewRequestWithContext(attemptCtx, http.MethodGet, url, nil)
		if err == nil {
			start := time.Now()
			resp, reqErr := client.Do(req)
			elapsed := time.Since(start)
			if reqErr == nil {
				resp.Body.Close()
				if resp.StatusCode < 500 {
					samples = append(samples, float64(elapsed.Microseconds())/1000)
				}
			}
		}
		cancel()
		if i < opts.Attempts-1 {
			time.Sleep(opts.Interval)
		}
	}

	return summarize(url, "http", samples, opts.Attempts, executedAt)
}

// ProbeDNS mede o tempo de resolução de nome usando o resolvedor informado
// (Seção 5: "DNS ping"; comparação entre resolvedores é responsabilidade de
// quem chama, rodando este probe várias vezes com resolver diferente).
func ProbeDNS(ctx context.Context, hostname string, resolverAddr string, opts Options) Result {
	executedAt := time.Now().UTC()
	var samples []float64

	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: opts.Timeout}
			addr := address
			if resolverAddr != "" {
				addr = resolverAddr
			}
			return d.DialContext(ctx, network, addr)
		},
	}

	for i := 0; i < opts.Attempts; i++ {
		attemptCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
		start := time.Now()
		_, err := resolver.LookupHost(attemptCtx, hostname)
		elapsed := time.Since(start)
		cancel()
		if err == nil {
			samples = append(samples, float64(elapsed.Microseconds())/1000)
		}
		if i < opts.Attempts-1 {
			time.Sleep(opts.Interval)
		}
	}

	target := hostname
	if resolverAddr != "" {
		target = hostname + "@" + resolverAddr
	}
	return summarize(target, "dns", samples, opts.Attempts, executedAt)
}
