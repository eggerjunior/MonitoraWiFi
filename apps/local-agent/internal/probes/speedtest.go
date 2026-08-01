// Speed test — Seção 5: "Não depender exclusivamente de um servidor público
// arbitrário" e separar claramente PHY/throughput/velocidade de Internet.
// Este arquivo implementa o modo "HTTP" (arquivo controlado, tamanho
// configurável — Seção 5.3): download e upload contra URLs configuráveis,
// mais bufferbloat medido como o aumento de latência sob carga em relação à
// latência ociosa. O modo LAN (iPerf3) e a comparação entre resolvedores
// ficam para uma iteração futura (registrado como pendência, não fingido).
package probes

import (
	"bytes"
	"context"
	"crypto/rand"
	"io"
	"net/http"
	"sort"
	"sync"
	"time"
)

type SpeedTestOptions struct {
	// DownloadURL serve um arquivo de tamanho conhecido/controlado (Seção
	// 5.3) — nunca um servidor de terceiros escolhido arbitrariamente sem
	// configuração explícita.
	DownloadURL string
	// UploadURL recebe um corpo POST e descarta — usado só para medir
	// throughput de envio.
	UploadURL       string
	UploadSizeBytes int
	// LatencyTarget (host:porta) é usado para medir latência ociosa e,
	// durante a transferência, latência sob carga — a diferença é o
	// indicador de bufferbloat.
	LatencyTarget string
	Timeout       time.Duration
}

func DefaultSpeedTestOptions(downloadURL, uploadURL, latencyTarget string) SpeedTestOptions {
	return SpeedTestOptions{
		DownloadURL:     downloadURL,
		UploadURL:       uploadURL,
		UploadSizeBytes: 4 * 1024 * 1024, // 4 MiB — grande o bastante para medir throughput, pequeno o bastante para não saturar um uplink residencial por muito tempo
		LatencyTarget:   latencyTarget,
		Timeout:         20 * time.Second,
	}
}

type SpeedTestResult struct {
	Mode            string
	DownloadMbps    *float64
	UploadMbps      *float64
	IdleLatencyMs   *float64
	LoadedLatencyMs *float64
	BufferbloatMs   *float64
	JitterMs        *float64
	ExecutedAt      time.Time
	// Errors registra falhas parciais (ex.: upload falhou mas download
	// funcionou) — nunca omitidas silenciosamente (Seção 2.1).
	Errors []string
}

// RunHTTPSpeedTest mede download e upload contra as URLs configuradas,
// medindo latência sob carga concorrentemente para estimar bufferbloat.
func RunHTTPSpeedTest(ctx context.Context, opts SpeedTestOptions) SpeedTestResult {
	executedAt := time.Now().UTC()
	result := SpeedTestResult{Mode: "http", ExecutedAt: executedAt}

	idleLatency := ProbeTCP(ctx, opts.LatencyTarget, Options{Attempts: 5, Timeout: 2 * time.Second, Interval: 100 * time.Millisecond})
	if idleLatency.LatencyMsP50 != nil {
		result.IdleLatencyMs = idleLatency.LatencyMsP50
	} else {
		result.Errors = append(result.Errors, "latência ociosa: todas as tentativas falharam")
	}

	// Sonda de latência sob carga roda em paralelo com as transferências,
	// desde o início do download até o fim do upload — cobre a janela toda
	// em que o link está sob uso, não apenas um instante isolado.
	loadedCtx, cancelLoaded := context.WithCancel(ctx)
	var loadedSamples []float64
	var loadedMu sync.Mutex
	var loadedWg sync.WaitGroup
	loadedWg.Add(1)
	go func() {
		defer loadedWg.Done()
		for {
			select {
			case <-loadedCtx.Done():
				return
			default:
			}
			r := ProbeTCP(loadedCtx, opts.LatencyTarget, Options{Attempts: 1, Timeout: 2 * time.Second})
			if r.LatencyMsP50 != nil {
				loadedMu.Lock()
				loadedSamples = append(loadedSamples, *r.LatencyMsP50)
				loadedMu.Unlock()
			}
			time.Sleep(300 * time.Millisecond)
		}
	}()

	if opts.DownloadURL != "" {
		mbps, err := measureDownload(ctx, opts.DownloadURL, opts.Timeout)
		if err != nil {
			result.Errors = append(result.Errors, "download: "+err.Error())
		} else {
			result.DownloadMbps = &mbps
		}
	}

	if opts.UploadURL != "" {
		mbps, err := measureUpload(ctx, opts.UploadURL, opts.UploadSizeBytes, opts.Timeout)
		if err != nil {
			result.Errors = append(result.Errors, "upload: "+err.Error())
		} else {
			result.UploadMbps = &mbps
		}
	}

	cancelLoaded()
	loadedWg.Wait()

	if len(loadedSamples) > 0 {
		p50 := percentile(sortedCopy(loadedSamples), 50)
		result.LoadedLatencyMs = &p50
		if result.IdleLatencyMs != nil {
			bufferbloat := p50 - *result.IdleLatencyMs
			if bufferbloat < 0 {
				bufferbloat = 0
			}
			result.BufferbloatMs = &bufferbloat
		}
		if len(loadedSamples) > 1 {
			jitter := meanAbsoluteDeviation(loadedSamples)
			result.JitterMs = &jitter
		}
	}

	return result
}

func sortedCopy(samples []float64) []float64 {
	out := append([]float64(nil), samples...)
	sort.Float64s(out)
	return out
}

func measureDownload(ctx context.Context, url string, timeout time.Duration) (float64, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}

	client := &http.Client{}
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	n, err := io.Copy(io.Discard, resp.Body)
	elapsed := time.Since(start)
	if err != nil && err != io.EOF {
		return 0, err
	}

	return bytesToMbps(n, elapsed), nil
}

func measureUpload(ctx context.Context, url string, sizeBytes int, timeout time.Duration) (float64, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	payload := make([]byte, sizeBytes)
	if _, err := rand.Read(payload); err != nil {
		return 0, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return 0, err
	}
	req.ContentLength = int64(sizeBytes)
	req.Header.Set("Content-Type", "application/octet-stream")

	client := &http.Client{}
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	elapsed := time.Since(start)

	return bytesToMbps(int64(sizeBytes), elapsed), nil
}

func bytesToMbps(bytes int64, elapsed time.Duration) float64 {
	if elapsed <= 0 {
		return 0
	}
	bits := float64(bytes) * 8
	return bits / elapsed.Seconds() / 1_000_000
}
