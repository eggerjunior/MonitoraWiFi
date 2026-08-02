package probes

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sort"
	"strconv"
	"sync"
	"syscall"
	"time"
)

// maxLANScanHosts é um limite de sanidade (defesa em profundidade — a API
// já restringe o tamanho do CIDR aceito, ver handlers_commands.go): mesmo
// que essa validação falhe, o agente nunca varre mais que isso numa única
// execução.
const maxLANScanHosts = 1024

// LANScanOptions controla como cada host é sondado — portas comuns de
// serviço (não privilégio de rede especial, ao contrário de ICMP), timeout
// curto por porta e um limite de goroutines concorrentes.
type LANScanOptions struct {
	Ports       []int
	Timeout     time.Duration
	Concurrency int
}

func DefaultLANScanOptions() LANScanOptions {
	return LANScanOptions{
		Ports:       []int{22, 80, 443, 445, 3389, 8080, 139},
		Timeout:     300 * time.Millisecond,
		Concurrency: 32,
	}
}

type LANScanResult struct {
	CIDR       string
	Hosts      []string // IPs que responderam em pelo menos uma porta — ordenados
	ExecutedAt time.Time
}

// ExpandCIDRHosts lista todo endereço dentro do bloco CIDR informado — sem
// excluir endereço de rede/broadcast (não é uma implementação de protocolo
// de rede, é uma varredura de descoberta simples; incluir esses endereços é
// inofensivo, eles só aparecem como "sem resposta").
func ExpandCIDRHosts(cidr string) ([]string, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("CIDR inválido: %w", err)
	}

	var hosts []string
	current := make(net.IP, len(ip.Mask(ipnet.Mask)))
	copy(current, ip.Mask(ipnet.Mask))
	for ipnet.Contains(current) {
		hosts = append(hosts, current.String())
		if len(hosts) > maxLANScanHosts {
			return nil, fmt.Errorf("bloco CIDR grande demais (máximo %d endereços)", maxLANScanHosts)
		}
		incIP(current)
	}
	return hosts, nil
}

func incIP(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			return
		}
	}
}

// isHostAlive tenta conectar em cada porta comum — uma conexão
// bem-sucedida ou uma recusa explícita (ECONNREFUSED, que só um host
// realmente ligado devolve) contam como "vivo". Timeout/sem rota conta
// como "sem resposta", nunca como falso positivo.
func isHostAlive(ctx context.Context, ip string, opts LANScanOptions) bool {
	dialer := net.Dialer{Timeout: opts.Timeout}
	for _, port := range opts.Ports {
		addr := net.JoinHostPort(ip, strconv.Itoa(port))
		conn, err := dialer.DialContext(ctx, "tcp", addr)
		if err == nil {
			conn.Close()
			return true
		}
		if errors.Is(err, syscall.ECONNREFUSED) {
			return true
		}
	}
	return false
}

// ScanLAN varre todo host do bloco CIDR concorrentemente (bounded worker
// pool) e devolve os que responderam de verdade — nunca inventa hosts.
func ScanLAN(ctx context.Context, cidr string, opts LANScanOptions) (LANScanResult, error) {
	executedAt := time.Now().UTC()

	hosts, err := ExpandCIDRHosts(cidr)
	if err != nil {
		return LANScanResult{}, err
	}

	sem := make(chan struct{}, opts.Concurrency)
	var mu sync.Mutex
	var alive []string
	var wg sync.WaitGroup

	for _, ip := range hosts {
		wg.Add(1)
		sem <- struct{}{}
		go func(ip string) {
			defer wg.Done()
			defer func() { <-sem }()
			if isHostAlive(ctx, ip, opts) {
				mu.Lock()
				alive = append(alive, ip)
				mu.Unlock()
			}
		}(ip)
	}
	wg.Wait()

	sort.Strings(alive)
	return LANScanResult{CIDR: cidr, Hosts: alive, ExecutedAt: executedAt}, nil
}
