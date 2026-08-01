// Package config carrega a configuração do backend a partir de variáveis de
// ambiente. Nenhum segredo tem valor padrão embutido no código (Seção 2.2).
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr        string
	DatabaseURL     string
	MigrationsDir   string
	SessionTTL      time.Duration
	OTelServiceName string
	Environment     string
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:        getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		MigrationsDir:   getEnv("MIGRATIONS_DIR", "../../infrastructure/database/migrations"),
		OTelServiceName: getEnv("OTEL_SERVICE_NAME", "egger-api"),
		Environment:     getEnv("APP_ENV", "development"),
	}

	if cfg.DatabaseURL == "" {
		return cfg, fmt.Errorf("DATABASE_URL não definido")
	}

	ttlHours := getEnv("SESSION_TTL_HOURS", "168") // 7 dias
	hours, err := strconv.Atoi(ttlHours)
	if err != nil {
		return cfg, fmt.Errorf("SESSION_TTL_HOURS inválido: %w", err)
	}
	cfg.SessionTTL = time.Duration(hours) * time.Hour

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
