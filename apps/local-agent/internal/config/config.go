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

// BuildVersion é sobrescrita em tempo de build via ldflags
// (-X .../config.BuildVersion=1.2.3, ver .github/workflows/local-agent-release.yml
// e scripts/build.sh) — "dev" é o fallback de builds locais/manuais, igual ao
// padrão usado no app iOS (skill ildemar_app-versioning).
var BuildVersion = "dev"

type Target struct {
	Name     string
	Protocol string // icmp | tcp | http | dns
}

type Config struct {
	BackendURL         string
	EnrollmentToken    string // só usado se ainda não houver identidade salva
	StateFilePath      string
	QueueFilePath      string
	SpeedQueueFilePath string
	Hostname           string
	Platform           string
	Version            string
	HeartbeatInterval  time.Duration
	ProbeInterval      time.Duration
	QueueMaxItems      int
	Targets            []Target

	// Comandos sob demanda (Fase 5, início — docs/architecture/03-fluxo-de-dados.md
	// §3.2): o agente consulta o backend periodicamente por comandos
	// pendentes, na mesma conexão outbound do heartbeat.
	CommandPollInterval time.Duration

	// Speed test (Seção 5.3, modo HTTP) — desativado se DownloadURL e
	// UploadURL estiverem ambos vazios (nunca escolhemos um servidor de
	// terceiros por conta própria).
	SpeedTestEnabled         bool
	SpeedTestInterval        time.Duration
	SpeedTestDownloadURL     string
	SpeedTestUploadURL       string
	SpeedTestUploadSizeBytes int
	SpeedTestLatencyTarget   string

	// Speed test modo LAN (Seção 5, iPerf3) — desativado se
	// SPEEDTEST_LAN_TARGET estiver vazio. Requer um servidor iperf3 já em
	// execução em outro nó da LAN (capability-matrix.md §4); o agente nunca
	// sobe seu próprio servidor nem escolhe um alvo por conta própria.
	SpeedTestLANEnabled  bool
	SpeedTestLANTarget   string
	SpeedTestLANDuration time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		BackendURL:      getEnv("BACKEND_URL", "http://localhost:8080/api/v1"),
		EnrollmentToken: os.Getenv("ENROLLMENT_TOKEN"),
		StateFilePath:   getEnv("STATE_FILE", "/etc/egger-agent/agent.json"),
		QueueFilePath:   getEnv("QUEUE_FILE", "/var/lib/egger-agent/queue.jsonl"),
		Version:         getEnv("AGENT_VERSION", BuildVersion),
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

	cfg.SpeedQueueFilePath = getEnv("SPEED_QUEUE_FILE", "/var/lib/egger-agent/speed-queue.jsonl")
	cfg.SpeedTestDownloadURL = os.Getenv("SPEEDTEST_DOWNLOAD_URL")
	cfg.SpeedTestUploadURL = os.Getenv("SPEEDTEST_UPLOAD_URL")
	cfg.SpeedTestLatencyTarget = getEnv("SPEEDTEST_LATENCY_TARGET", "1.1.1.1:443")
	cfg.SpeedTestEnabled = cfg.SpeedTestDownloadURL != "" || cfg.SpeedTestUploadURL != ""

	uploadSize, err := parseIntEnv("SPEEDTEST_UPLOAD_SIZE_BYTES", 4*1024*1024)
	if err != nil {
		return cfg, err
	}
	cfg.SpeedTestUploadSizeBytes = uploadSize

	speedIntervalMinutes, err := parseIntEnv("SPEEDTEST_INTERVAL_MINUTES", 30)
	if err != nil {
		return cfg, err
	}
	cfg.SpeedTestInterval = time.Duration(speedIntervalMinutes) * time.Minute

	commandPollSeconds, err := parseIntEnv("COMMAND_POLL_INTERVAL_SECONDS", 5)
	if err != nil {
		return cfg, err
	}
	cfg.CommandPollInterval = time.Duration(commandPollSeconds) * time.Second

	cfg.SpeedTestLANTarget = os.Getenv("SPEEDTEST_LAN_TARGET")
	cfg.SpeedTestLANEnabled = cfg.SpeedTestLANTarget != ""

	lanDurationSeconds, err := parseIntEnv("SPEEDTEST_LAN_DURATION_SECONDS", 5)
	if err != nil {
		return cfg, err
	}
	cfg.SpeedTestLANDuration = time.Duration(lanDurationSeconds) * time.Second

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
