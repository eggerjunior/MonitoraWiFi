package probes

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strconv"
	"sync"
	"time"
)

// maxPortScanRange é um limite de sanidade (defesa em profundidade — a API
// já valida o mesmo limite, ver handlers_commands.go): mesmo que essa
// validação falhe, o agente nunca varre mais portas que isso numa única
// execução.
const maxPortScanRange = 1024

type PortScanOptions struct {
	Timeout     time.Duration
	Concurrency int
}

func DefaultPortScanOptions() PortScanOptions {
	return PortScanOptions{Timeout: 300 * time.Millisecond, Concurrency: 64}
}

type PortScanResult struct {
	Target     string
	OpenPorts  []int
	ExecutedAt time.Time
}

// ScanPorts tenta conectar em cada porta do intervalo — só reporta como
// aberta a porta que aceitou a conexão de verdade (nunca uma lista
// inventada); qualquer outra (recusada, sem resposta) fica de fora.
func ScanPorts(ctx context.Context, target string, startPort, endPort int, opts PortScanOptions) (PortScanResult, error) {
	if startPort < 1 || endPort > 65535 || startPort > endPort {
		return PortScanResult{}, fmt.Errorf("intervalo de portas inválido")
	}
	if endPort-startPort+1 > maxPortScanRange {
		return PortScanResult{}, fmt.Errorf("intervalo de portas grande demais (máximo %d portas)", maxPortScanRange)
	}

	executedAt := time.Now().UTC()
	dialer := net.Dialer{Timeout: opts.Timeout}

	sem := make(chan struct{}, opts.Concurrency)
	var mu sync.Mutex
	var open []int
	var wg sync.WaitGroup

	for port := startPort; port <= endPort; port++ {
		wg.Add(1)
		sem <- struct{}{}
		go func(port int) {
			defer wg.Done()
			defer func() { <-sem }()
			addr := net.JoinHostPort(target, strconv.Itoa(port))
			conn, err := dialer.DialContext(ctx, "tcp", addr)
			if err == nil {
				conn.Close()
				mu.Lock()
				open = append(open, port)
				mu.Unlock()
			}
		}(port)
	}
	wg.Wait()

	sort.Ints(open)
	return PortScanResult{Target: target, OpenPorts: open, ExecutedAt: executedAt}, nil
}
