package httpapi

import (
	"io"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
)

func newTestServer(pinger Pinger, users *fakeUsers, sessions *fakeSessions, orgs *fakeOrgs, sites *fakeSites) *Server {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewServer(Deps{
		Logger:     logger,
		Tracer:     otel.Tracer("test"),
		Pool:       pinger,
		Orgs:       orgs,
		Sites:      sites,
		Users:      users,
		Sessions:   sessions,
		Audit:      &fakeAudit{},
		SessionTTL: time.Hour,
	})
}
