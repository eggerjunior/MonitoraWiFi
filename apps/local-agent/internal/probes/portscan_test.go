package probes

import (
	"net"
	"strconv"
	"testing"
	"time"
)

// TestScanPorts_EncontraPortaAbertaReal usa dois listeners TCP reais e um
// intervalo de portas que os contém — só as portas com listener real devem
// aparecer como abertas.
func TestScanPorts_EncontraPortaAbertaReal(t *testing.T) {
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

	opts := PortScanOptions{Timeout: 500 * time.Millisecond, Concurrency: 16}
	result, err := ScanPorts(t.Context(), "127.0.0.1", port-2, port+2, opts)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if len(result.OpenPorts) != 1 || result.OpenPorts[0] != port {
		t.Fatalf("portas abertas = %v, esperado apenas [%d]", result.OpenPorts, port)
	}
}

func TestScanPorts_IntervaloInvalido(t *testing.T) {
	opts := DefaultPortScanOptions()
	if _, err := ScanPorts(t.Context(), "127.0.0.1", 100, 50, opts); err == nil {
		t.Fatal("esperava erro com start > end")
	}
	if _, err := ScanPorts(t.Context(), "127.0.0.1", 1, 2000, opts); err == nil {
		t.Fatal("esperava erro com intervalo maior que o limite de sanidade")
	}
}
