// Wake-on-LAN (ADR-008): executado exclusivamente pelo agente local, nunca
// pelo app iOS — iOS exige o entitlement de Multicast Networking (aprovação
// discricionária da Apple) e tem bugs de plataforma documentados e não
// resolvidos para UDP broadcast.
package probes

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"strconv"
	"syscall"
)

// BuildMagicPacket monta o payload padrão de Wake-on-LAN: 6 bytes 0xFF
// seguidos do endereço MAC repetido 16 vezes (102 bytes no total).
func BuildMagicPacket(mac net.HardwareAddr) []byte {
	packet := make([]byte, 0, 102)
	packet = append(packet, bytes.Repeat([]byte{0xFF}, 6)...)
	for i := 0; i < 16; i++ {
		packet = append(packet, mac...)
	}
	return packet
}

// SendWakeOnLAN envia o magic packet real via UDP — habilita SO_BROADCAST
// no socket pra permitir endereços de broadcast (ex.: 255.255.255.255),
// mas funciona igual contra um destino unicast (usado nos testes).
func SendWakeOnLAN(ctx context.Context, macAddress, destIP string, port int) error {
	mac, err := net.ParseMAC(macAddress)
	if err != nil {
		return fmt.Errorf("endereço MAC inválido: %w", err)
	}
	if len(mac) != 6 {
		return fmt.Errorf("endereço MAC precisa ter 6 octetos (EUI-48), recebeu %d", len(mac))
	}

	conn, err := net.ListenPacket("udp4", ":0")
	if err != nil {
		return fmt.Errorf("erro ao abrir socket UDP: %w", err)
	}
	defer conn.Close()

	if udpConn, ok := conn.(*net.UDPConn); ok {
		if rawConn, err := udpConn.SyscallConn(); err == nil {
			_ = rawConn.Control(func(fd uintptr) {
				_ = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1)
			})
		}
	}

	dst, err := net.ResolveUDPAddr("udp4", net.JoinHostPort(destIP, strconv.Itoa(port)))
	if err != nil {
		return fmt.Errorf("endereço de destino inválido: %w", err)
	}

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetWriteDeadline(deadline)
	}

	packet := BuildMagicPacket(mac)
	if _, err := conn.WriteTo(packet, dst); err != nil {
		return fmt.Errorf("erro ao enviar magic packet: %w", err)
	}
	return nil
}
