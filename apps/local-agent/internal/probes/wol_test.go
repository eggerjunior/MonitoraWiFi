package probes

import (
	"bytes"
	"net"
	"strconv"
	"testing"
	"time"
)

func TestBuildMagicPacket(t *testing.T) {
	mac, err := net.ParseMAC("AA:BB:CC:DD:EE:FF")
	if err != nil {
		t.Fatalf("erro ao parsear MAC: %v", err)
	}
	packet := BuildMagicPacket(mac)

	if len(packet) != 102 {
		t.Fatalf("tamanho do pacote = %d, esperado 102", len(packet))
	}
	for i := 0; i < 6; i++ {
		if packet[i] != 0xFF {
			t.Fatalf("byte de sincronização[%d] = %x, esperado 0xFF", i, packet[i])
		}
	}
	for i := 0; i < 16; i++ {
		got := packet[6+i*6 : 6+(i+1)*6]
		if !bytes.Equal(got, mac) {
			t.Fatalf("repetição %d do MAC = %x, esperado %x", i, got, []byte(mac))
		}
	}
}

// TestSendWakeOnLAN_EnviaPacoteReal confirma que o pacote é de fato enviado
// via UDP real — um listener local recebe os bytes crus e confirma que
// batem exatamente com o magic packet esperado, nunca uma simulação.
func TestSendWakeOnLAN_EnviaPacoteReal(t *testing.T) {
	conn, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("erro ao abrir listener UDP: %v", err)
	}
	defer conn.Close()

	_, portStr, err := net.SplitHostPort(conn.LocalAddr().String())
	if err != nil {
		t.Fatalf("erro ao separar porta: %v", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("porta inválida: %v", err)
	}

	if err := SendWakeOnLAN(t.Context(), "AA:BB:CC:DD:EE:FF", "127.0.0.1", port); err != nil {
		t.Fatalf("erro inesperado ao enviar: %v", err)
	}

	buf := make([]byte, 256)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, _, err := conn.ReadFrom(buf)
	if err != nil {
		t.Fatalf("erro ao receber pacote real: %v", err)
	}

	mac, _ := net.ParseMAC("AA:BB:CC:DD:EE:FF")
	want := BuildMagicPacket(mac)
	if !bytes.Equal(buf[:n], want) {
		t.Fatalf("pacote recebido = %x, esperado %x", buf[:n], want)
	}
}

func TestSendWakeOnLAN_MACInvalido(t *testing.T) {
	err := SendWakeOnLAN(t.Context(), "não-é-um-mac", "127.0.0.1", 9)
	if err == nil {
		t.Fatal("esperava erro para MAC inválido")
	}
}
