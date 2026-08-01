package agent

import (
	"fmt"
	"time"

	"egger/local-agent/internal/apiclient"
	"egger/local-agent/internal/probes"
)

// toPingTestPayload converte o resultado de uma sonda no formato aceito pelo
// backend. A idempotency_key é determinística a partir do próprio resultado
// (alvo + protocolo + instante de execução em nanossegundos) — gerada uma
// vez, na hora em que o item entra na fila; reenvios da mesma entrada da
// fila reusam a mesma chave (Seção 3: "reenvio idempotente").
func toPingTestPayload(r probes.Result) apiclient.PingTestPayload {
	return apiclient.PingTestPayload{
		Target:         r.Target,
		Protocol:       r.Protocol,
		LatencyMsP50:   r.LatencyMsP50,
		LatencyMsP95:   r.LatencyMsP95,
		LatencyMsP99:   r.LatencyMsP99,
		JitterMs:       r.JitterMs,
		PacketLossPct:  &r.PacketLossPct,
		ExecutedAt:     r.ExecutedAt.Format(time.RFC3339Nano),
		IdempotencyKey: fmt.Sprintf("%s|%s|%d", r.Protocol, r.Target, r.ExecutedAt.UnixNano()),
	}
}

func toSpeedTestPayload(r probes.SpeedTestResult) apiclient.SpeedTestPayload {
	return apiclient.SpeedTestPayload{
		Mode:            r.Mode,
		DownloadMbps:    r.DownloadMbps,
		UploadMbps:      r.UploadMbps,
		IdleLatencyMs:   r.IdleLatencyMs,
		LoadedLatencyMs: r.LoadedLatencyMs,
		BufferbloatMs:   r.BufferbloatMs,
		JitterMs:        r.JitterMs,
		ExecutedAt:      r.ExecutedAt.Format(time.RFC3339Nano),
		IdempotencyKey:  fmt.Sprintf("speedtest|%s|%d", r.Mode, r.ExecutedAt.UnixNano()),
	}
}
