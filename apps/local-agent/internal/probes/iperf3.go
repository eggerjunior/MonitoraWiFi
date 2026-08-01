// Speed test modo LAN (Seção 5, "iPerf3"): mede throughput real contra um
// servidor iperf3 já existente em outro nó da LAN (capability-matrix.md
// §4: "Requer binário/servidor iperf3 em outro nó da LAN" — o agente nunca
// sobe seu próprio servidor iperf3 nem escolhe um alvo por conta própria).
// Reimplementar o protocolo iperf3 em Go não vale o esforço/risco frente ao
// binário oficial: chamamos `iperf3` via subprocesso e parseamos a saída
// JSON (`-J`), exatamente como uma pessoa rodaria manualmente.
package probes

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Iperf3Available reporta se o binário `iperf3` existe no PATH — checado uma
// vez na inicialização do agente. Se ausente, o modo LAN fica desativado e
// isso é logado claramente (nunca finge um resultado).
func Iperf3Available() bool {
	_, err := exec.LookPath("iperf3")
	return err == nil
}

type iperf3EndSum struct {
	BitsPerSecond float64 `json:"bits_per_second"`
}

type iperf3Output struct {
	End struct {
		SumSent     iperf3EndSum `json:"sum_sent"`
		SumReceived iperf3EndSum `json:"sum_received"`
	} `json:"end"`
	Error string `json:"error"`
}

// RunIperf3SpeedTest roda um teste de download (-R, servidor → agente) e um
// de upload (agente → servidor) contra um servidor iperf3 já em execução em
// `target` (host:porta, ex.: "192.168.1.50:5201"). Usa sempre a vazão
// medida do lado receptor (goodput real entregue), não a do lado emissor.
func RunIperf3SpeedTest(ctx context.Context, target string, duration time.Duration) SpeedTestResult {
	executedAt := time.Now().UTC()
	result := SpeedTestResult{Mode: "lan", ExecutedAt: executedAt}

	if !Iperf3Available() {
		result.Errors = append(result.Errors, "binário iperf3 não encontrado no PATH — modo LAN indisponível neste host")
		return result
	}

	host, port, err := splitHostPort(target)
	if err != nil {
		result.Errors = append(result.Errors, "alvo iperf3 inválido: "+err.Error())
		return result
	}

	if mbps, err := runIperf3(ctx, host, port, duration, true); err != nil {
		result.Errors = append(result.Errors, "download (iperf3 -R): "+err.Error())
	} else {
		result.DownloadMbps = &mbps
	}

	if mbps, err := runIperf3(ctx, host, port, duration, false); err != nil {
		result.Errors = append(result.Errors, "upload (iperf3): "+err.Error())
	} else {
		result.UploadMbps = &mbps
	}

	return result
}

func splitHostPort(target string) (host, port string, err error) {
	idx := strings.LastIndex(target, ":")
	if idx == -1 {
		return "", "", fmt.Errorf("esperado host:porta, recebido %q", target)
	}
	host = target[:idx]
	port = target[idx+1:]
	if _, err := strconv.Atoi(port); err != nil {
		return "", "", fmt.Errorf("porta inválida em %q", target)
	}
	return host, port, nil
}

func runIperf3(ctx context.Context, host, port string, duration time.Duration, reverse bool) (float64, error) {
	seconds := int(duration.Seconds())
	if seconds < 1 {
		seconds = 1
	}
	args := []string{"-c", host, "-p", port, "-J", "-t", strconv.Itoa(seconds)}
	if reverse {
		args = append(args, "-R")
	}

	cmdCtx, cancel := context.WithTimeout(ctx, duration+10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "iperf3", args...)
	out, runErr := cmd.Output()
	if len(out) == 0 {
		if runErr != nil {
			return 0, runErr
		}
		return 0, fmt.Errorf("iperf3 não retornou saída")
	}

	var parsed iperf3Output
	if err := json.Unmarshal(out, &parsed); err != nil {
		return 0, fmt.Errorf("saída do iperf3 não é JSON válido: %w", err)
	}
	if parsed.Error != "" {
		return 0, fmt.Errorf("iperf3: %s", parsed.Error)
	}

	bps := parsed.End.SumReceived.BitsPerSecond
	if bps == 0 {
		return 0, fmt.Errorf("iperf3 não reportou throughput (bits_per_second == 0)")
	}
	return bps / 1_000_000, nil
}
