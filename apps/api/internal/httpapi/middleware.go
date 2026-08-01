package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/trace"
)

// withObservability envolve todo handler com correlation ID, log estruturado,
// span de trace e recuperação de panic — nunca deixamos um panic derrubar o
// processo silenciosamente nem sem log (Seção 20).
func (s *Server) withObservability(name string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		correlationID := r.Header.Get("X-Correlation-ID")
		if correlationID == "" {
			correlationID = newCorrelationID()
		}
		ctx := withCorrelationID(r.Context(), correlationID)

		ctx, span := s.tracer.Start(ctx, name, trace.WithAttributes())
		defer span.End()

		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		rec.Header().Set("X-Correlation-ID", correlationID)

		defer func() {
			if rec2 := recover(); rec2 != nil {
				s.logger.Error("panic recuperado",
					slog.Any("panic", rec2),
					slog.String("correlation_id", correlationID),
					slog.String("path", r.URL.Path),
				)
				writeError(rec, correlationID, http.StatusInternalServerError, "internal_error", "erro interno")
			}
			s.logger.Info("requisição atendida",
				slog.String("correlation_id", correlationID),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rec.status),
				slog.Duration("duration", time.Since(start)),
			)
		}()

		next(rec, r.WithContext(ctx))
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
