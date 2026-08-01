// Package config carrega a configuração do agente a partir de variáveis de
// ambiente — sem valor padrão embutido para segredos (Seção 2.2).
package config

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Target struct {
	Name     string
	Protocol string // icmp | tcp | http | dns
}

type Config struct {
	BackendURL        string
	EnrollmentToken   string // só usado se ainda não houver identidade salva
	StateFilePath     string
	QueueFilePath     string
	Hostname          string
	Platform          string
	Version           string
	HeartbeatInterval time.Duration
	ProbeInterval     time.Duration
	QueueMaxItems     int
	Targets           []Target
}

func Load() (Config, error) {
	cfg := Config{
		BackendURL:      getEnv("BACKEND_URL", "http://localhost:8080/api/v1"),
		EnrollmentToken: os.Getenv("ENROLLMENT_TOKEN"),
		StateFilePath:   getEnv("STATE_FILE", "/etc/egger-agent/agent.json"),
		QueueFilePath:   getEnv("QUEUE_FILE", "/var/lib/egger-agent/queue.jsonl"),
		Version:         getEnv("AGENT_VERSION", "dev"),
		Platform:        detectPlatform(),
	}

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown-host"
	}
	cfg.Hostname = getEnv("AGENT_HOSTNAME", hostname)

	heartbeatSeconds, err := parseIntEnv("HEARTBEAT_INTERVAL_SECONDS", 30)
	if err != nil {
		return cfg, err
	}
	cfg.HeartbeatInterval = time.Duration(heartbeatSeconds) * time.Second

	probeSeconds, err := parseIntEnv("PROBE_INTERVAL_SECONDS", 60)
	if err != nil {
		return cfg, err
	}
	cfg.ProbeInterval = time.Duration(probeSeconds) * time.Second

	queueMax, err := parseIntEnv("QUEUE_MAX_ITEMS", 10000)
	if err != nil {
		return cfg, err
	}
	cfg.QueueMaxItems = queueMax

	cfg.Targets = parseTargets(getEnv("TARGETS", "1.1.1.1:icmp,8.8.8.8:icmp,https://www.cloudflare.com:http,cloudflare.com:dns"))

	return cfg, nil
}

func detectPlatform() string {
	switch runtime.GOOS {
	case "linux":
		if runtime.GOARCH == "arm64" {
			return "linux_arm64"
		}
		return "linux_amd64"
	case "darwin":
		return "macos"
	default:
		return runtime.GOOS + "_" + runtime.GOARCH
	}
}

func parseTargets(raw string) []Target {
	var targets []Target
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		idx := strings.LastIndex(entry, ":")
		if idx == -1 {
			continue
		}
		name := entry[:idx]
		protocol := entry[idx+1:]
		targets = append(targets, Target{Name: name, Protocol: protocol})
	}
	return targets
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func parseIntEnv(key string, fallback int) (int, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s inválido: %w", key, err)
	}
	return v, nil
}
