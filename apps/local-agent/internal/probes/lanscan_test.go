package probes

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"
)

func TestExpandCIDRHosts(t *testing.T) {
	hosts, err := ExpandCIDRHosts("127.0.0.0/30")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	want := []string{"127.0.0.0", "127.0.0.1", "127.0.0.2", "127.0.0.3"}
	if len(hosts) != len(want) {
		t.Fatalf("hosts = %v, esperado %v", hosts, want)
	}
	for i, h := range hosts {
		if h != want[i] {
			t.Fatalf("hosts[%d] = %s, esperado %s", i, h, want[i])
		}
	}
}

func TestExpandCIDRHosts_Unico(t *testing.T) {
	hosts, err := ExpandCIDRHosts("127.0.0.1/32")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if len(hosts) != 1 || hosts[0] != "127.0.0.1" {
		t.Fatalf("hosts = %v, esperado [127.0.0.1]", hosts)
	}
}

func TestExpandCIDRHosts_BlocoGrandeDemais(t *testing.T) {
	_, err := ExpandCIDRHosts("10.0.0.0/8")
	if err == nil {
		t.Fatal("esperava erro para bloco CIDR maior que o limite de sanidade")
	}
}

// TestIsHostAlive_ConexaoAceita usa um listener TCP real — a conexão real
// bem-sucedida é o sinal mais forte de "vivo".
func TestIsHostAlive_ConexaoAceita(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("erro ao abrir listener: %v", err)
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

	_, portStr, _ := net.SplitHostPort(ln.Addr().String())
	port, _ := strconv.Atoi(portStr)
	opts := LANScanOptions{Ports: []int{port}, Timeout: time.Second}

	if !isHostAlive(t.Context(), "127.0.0.1", opts) {
		t.Fatal("esperava alive=true contra um listener real")
	}
}

// TestIsHostAlive_ConexaoRecusada fecha o listener imediatamente antes de
// conectar — o SO responde com um RST real (ECONNREFUSED), que também
// prova que o host está de pé (só um host ligado devolve RST).
func TestIsHostAlive_ConexaoRecusada(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("erro ao abrir listener: %v", err)
	}
	_, portStr, _ := net.SplitHostPort(ln.Addr().String())
	port, _ := strconv.Atoi(portStr)
	ln.Close()

	opts := LANScanOptions{Ports: []int{port}, Timeout: time.Second}
	if !isHostAlive(t.Context(), "127.0.0.1", opts) {
		t.Fatal("esperava alive=true com ECONNREFUSED real (host de pé, porta fechada)")
	}
}

// TestIsHostAlive_ContextoExpirado prova que qualquer erro de dial que não
// seja ECONNREFUSED (aqui, um contexto já expirado) é tratado como "sem
// resposta", nunca como falso positivo.
func TestIsHostAlive_ContextoExpirado(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), time.Nanosecond)
	defer cancel()
	time.Sleep(time.Millisecond)

	opts := LANScanOptions{Ports: []int{80}, Timeout: time.Second}
	if isHostAlive(ctx, "127.0.0.1", opts) {
		t.Fatal("esperava alive=false com contexto já expirado")
	}
}

func TestScanLAN_AgregaResultadosReais(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("erro ao abrir listener: %v", err)
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
	_, portStr, _ := net.SplitHostPort(ln.Addr().String())
	port, _ := strconv.Atoi(portStr)

	opts := LANScanOptions{Ports: []int{port}, Timeout: 500 * time.Millisecond, Concurrency: 4}
	result, err := ScanLAN(t.Context(), "127.0.0.0/30", opts)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	// Todo endereço de 127.0.0.0/8 é loopback local — o SO responde
	// ECONNREFUSED de verdade pra qualquer um deles nessa porta específica
	// sem listener, então os 4 endereços devem aparecer como "vivos" (cada
	// um é, de fato, o próprio host local respondendo).
	if len(result.Hosts) != 4 {
		t.Fatalf("esperava 4 hosts vivos (todo 127.0.0.0/8 é loopback local), obtive: %+v", result.Hosts)
	}
}
