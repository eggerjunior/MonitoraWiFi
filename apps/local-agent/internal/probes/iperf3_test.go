package probes

import (
	"context"
	"net"
	"os/exec"
	"strconv"
	"testing"
	"time"
)

// TestRunIperf3SpeedTest roda um servidor iperf3 real (subprocesso `iperf3
// -s`) e o cliente real contra ele — nunca contra um mock/fake do
// protocolo. Se o binário `iperf3` não estiver instalado no ambiente de
// teste, o teste é pulado explicitamente (não é uma falha silenciosa nem um
// resultado fabricado).
func TestRunIperf3SpeedTest_DownloadAndUpload(t *testing.T) {
	if !Iperf3Available() {
		t.Skip("binário iperf3 não encontrado no PATH — pulando teste real")
	}

	port := freeTCPPort(t)

	// Sem "-1": o teste faz duas conexões de cliente sequenciais (download
	// e upload) — um servidor "one-off" encerraria após a primeira. O
	// servidor é morto explicitamente no fim do teste (não há como pedir
	// para ele parar sozinho de forma limpa nesse modo).
	server := exec.Command("iperf3", "-s", "-p", strconv.Itoa(port))
	if err := server.Start(); err != nil {
		t.Fatalf("erro ao iniciar servidor iperf3 de teste: %v", err)
	}
	t.Cleanup(func() {
		_ = server.Process.Kill()
		_ = server.Wait()
	})

	waitForIperf3Ready(t, "127.0.0.1:"+strconv.Itoa(port), 8*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result := RunIperf3SpeedTest(ctx, "127.0.0.1:"+strconv.Itoa(port), 1*time.Second)

	if len(result.Errors) > 0 {
		t.Fatalf("esperava teste sem erros, obtive: %v", result.Errors)
	}
	if result.Mode != "lan" {
		t.Errorf("Mode = %q, esperado \"lan\"", result.Mode)
	}
	if result.DownloadMbps == nil || *result.DownloadMbps <= 0 {
		t.Errorf("DownloadMbps deveria ser > 0, obtive %v", result.DownloadMbps)
	}
	if result.UploadMbps == nil || *result.UploadMbps <= 0 {
		t.Errorf("UploadMbps deveria ser > 0, obtive %v", result.UploadMbps)
	}
}

func TestRunIperf3SpeedTest_ServerIndisponivel(t *testing.T) {
	if !Iperf3Available() {
		t.Skip("binário iperf3 não encontrado no PATH — pulando teste real")
	}

	// Porta livre sem nenhum servidor escutando — deve reportar erro
	// honesto, nunca um throughput inventado.
	port := freeTCPPort(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := RunIperf3SpeedTest(ctx, "127.0.0.1:"+strconv.Itoa(port), 1*time.Second)

	if len(result.Errors) == 0 {
		t.Fatal("esperava erro reportado quando não há servidor iperf3 escutando")
	}
	if result.DownloadMbps != nil || result.UploadMbps != nil {
		t.Errorf("não deveria haver throughput sem servidor: download=%v upload=%v", result.DownloadMbps, result.UploadMbps)
	}
}

func TestRunIperf3SpeedTest_AlvoInvalido(t *testing.T) {
	ctx := context.Background()
	result := RunIperf3SpeedTest(ctx, "sem-porta", 1*time.Second)
	if len(result.Errors) == 0 {
		t.Fatal("esperava erro para alvo sem porta")
	}
}

func freeTCPPort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("erro ao obter porta livre: %v", err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

// waitForIperf3Ready espera o servidor aceitar uma sessão real do cliente
// iperf3 (não um dial TCP cru — o protocolo do iperf3 trata a conexão de
// controle de forma própria, e um connect/close genérico pode confundir um
// servidor "-1"/one-off; aqui o servidor é persistente, mas ainda assim o
// mais realista é esperar com o próprio protocolo).
func waitForIperf3Ready(t *testing.T, addr string, timeout time.Duration) {
	t.Helper()
	host, port, err := splitHostPort(addr)
	if err != nil {
		t.Fatalf("endereço de teste inválido: %v", err)
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		// O contexto aqui precisa ser folgado o bastante para caber a
		// duração do teste (1s) + a margem interna de runIperf3
		// (duration+10s) — um timeout mais curto matava o processo
		// `iperf3` no meio do teste de prontidão.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, err := runIperf3(ctx, host, port, 1*time.Second, false)
		cancel()
		if err == nil {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("servidor iperf3 de teste não ficou pronto a tempo em %s", addr)
}
