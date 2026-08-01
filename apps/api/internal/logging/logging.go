// Package logging fornece o logger estruturado (JSON) usado por todo o backend
// (Seção 3: "logs JSON").
package logging

import (
	"log/slog"
	"os"
)

func New(environment string) *slog.Logger {
	level := slog.LevelInfo
	if environment == "development" {
		level = slog.LevelDebug
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})
	return slog.New(handler)
}
