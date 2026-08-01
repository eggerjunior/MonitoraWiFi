package probes

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRunHTTPSpeedTest_DownloadAndUpload(t *testing.T) {
	payloadSize := 2 * 1024 * 1024 // 2 MiB
	payload := make([]byte, payloadSize)

	downloadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(payload)
	}))
	defer downloadServer.Close()

	uploadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer uploadServer.Close()

	// Listener TCP local para servir de alvo de latência ociosa/sob carga.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("erro ao abrir listener de teste: %v", err)
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

	opts := DefaultSpeedTestOptions(downloadServer.URL, uploadServer.URL, ln.Addr().String())
	opts.UploadSizeBytes = 512 * 1024
	opts.Timeout = 10 * time.Second

	result := RunHTTPSpeedTest(context.Background(), opts)

	if len(result.Errors) > 0 {
		t.Fatalf("esperava speed test sem erros, recebeu: %v", result.Errors)
	}
	if result.DownloadMbps == nil || *result.DownloadMbps <= 0 {
		t.Fatalf("esperava download_mbps > 0, recebeu %v", result.DownloadMbps)
	}
	if result.UploadMbps == nil || *result.UploadMbps <= 0 {
		t.Fatalf("esperava upload_mbps > 0, recebeu %v", result.UploadMbps)
	}
	if result.IdleLatencyMs == nil {
		t.Fatal("esperava latência ociosa medida")
	}
	if result.Mode != "http" {
		t.Fatalf("esperava mode=http, recebeu %q", result.Mode)
	}
}

func TestRunHTTPSpeedTest_ReportsErrorsWithoutInventingValues(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("erro ao reservar porta: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close()

	opts := DefaultSpeedTestOptions("http://127.0.0.1:1/download-invalido", "http://127.0.0.1:1/upload-invalido", addr)
	opts.Timeout = 2 * time.Second

	result := RunHTTPSpeedTest(context.Background(), opts)

	if result.DownloadMbps != nil {
		t.Fatal("não deveria haver download_mbps quando a URL é inválida — nunca inventar dado")
	}
	if result.UploadMbps != nil {
		t.Fatal("não deveria haver upload_mbps quando a URL é inválida — nunca inventar dado")
	}
	if len(result.Errors) == 0 {
		t.Fatal("esperava erros registrados explicitamente, não uma falha silenciosa")
	}
}
