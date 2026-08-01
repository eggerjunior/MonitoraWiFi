package httpapi

import (
	"io"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"

	"egger/api/internal/store"
)

func newTestServer(pinger Pinger, users *fakeUsers, sessions *fakeSessions, orgs *fakeOrgs, sites *fakeSites) *Server {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	agents := newFakeAgents()
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

		Agents:            agents,
		AgentEnrollTokens: newFakeAgentEnrollmentTokens(),
		AgentHeartbeats:   &fakeAgentHeartbeats{},
		PingTests:         newFakePingTests(),
		SpeedTests:        newFakeSpeedTests(),
		AgentCommands:     newFakeAgentCommands(agents),
	})
}

// agentTestDeps agrupa os fakes de agente para testes que precisam inspecionar
// o estado após a chamada (ex.: verificar que um heartbeat foi de fato gravado).
type agentTestDeps struct {
	server            *Server
	users             *fakeUsers
	sessions          *fakeSessions
	agents            *fakeAgents
	agentEnrollTokens *fakeAgentEnrollmentTokens
	agentHeartbeats   *fakeAgentHeartbeats
	pingTests         *fakePingTests
	speedTests        *fakeSpeedTests
	agentCommands     *fakeAgentCommands
}

func newAgentTestServer(users ...store.User) agentTestDeps {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	fu := newFakeUsers(users...)
	fs := newFakeSessions()
	fa := newFakeAgents()
	fet := newFakeAgentEnrollmentTokens()
	fhb := &fakeAgentHeartbeats{}
	fpt := newFakePingTests()
	fst := newFakeSpeedTests()
	fac := newFakeAgentCommands(fa)

	server := NewServer(Deps{
		Logger:            logger,
		Tracer:            otel.Tracer("test"),
		Pool:              &fakePinger{},
		Orgs:              &fakeOrgs{},
		Sites:             &fakeSites{},
		Users:             fu,
		Sessions:          fs,
		Audit:             &fakeAudit{},
		SessionTTL:        time.Hour,
		Agents:            fa,
		AgentEnrollTokens: fet,
		AgentHeartbeats:   fhb,
		PingTests:         fpt,
		SpeedTests:        fst,
		AgentCommands:     fac,
	})

	return agentTestDeps{
		server:            server,
		users:             fu,
		sessions:          fs,
		agents:            fa,
		agentEnrollTokens: fet,
		agentHeartbeats:   fhb,
		pingTests:         fpt,
		speedTests:        fst,
		agentCommands:     fac,
	}
}
