package agent

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"egger/local-agent/internal/apiclient"
	"egger/local-agent/internal/config"
	"egger/local-agent/internal/state"
)

type fakeEnroller struct {
	result apiclient.EnrollResult
	err    error
	calls  int
}

func (f *fakeEnroller) Enroll(ctx context.Context, enrollmentToken, hostname, platform, version string) (apiclient.EnrollResult, error) {
	f.calls++
	return f.result, f.err
}

func TestEnsureIdentity_EnrollsWhenNoStateExists(t *testing.T) {
	cfg := config.Config{
		StateFilePath:   filepath.Join(t.TempDir(), "agent.json"),
		EnrollmentToken: "tok-123",
		Hostname:        "host-1",
		Platform:        "linux_amd64",
		Version:         "0.1.0",
	}
	enroller := &fakeEnroller{result: apiclient.EnrollResult{AgentID: "agent-1", AgentSecret: "secret-1", SiteID: "site-1"}}

	id, err := EnsureIdentity(context.Background(), cfg, enroller)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if id.AgentID != "agent-1" || id.AgentSecret != "secret-1" {
		t.Fatalf("identidade inesperada: %+v", id)
	}
	if enroller.calls != 1 {
		t.Fatalf("esperava 1 chamada a Enroll, houve %d", enroller.calls)
	}

	// Persistiu — uma segunda chamada não deve enrolar de novo.
	saved, err := state.Load(cfg.StateFilePath)
	if err != nil || saved == nil {
		t.Fatalf("esperava identidade persistida em disco, err=%v saved=%+v", err, saved)
	}
}

func TestEnsureIdentity_ReusesExistingState(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "agent.json")
	if err := state.Save(statePath, state.Identity{AgentID: "existing", AgentSecret: "shh", SiteID: "site-1"}); err != nil {
		t.Fatalf("erro ao preparar estado: %v", err)
	}

	cfg := config.Config{StateFilePath: statePath}
	enroller := &fakeEnroller{}

	id, err := EnsureIdentity(context.Background(), cfg, enroller)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if id.AgentID != "existing" {
		t.Fatalf("esperava reusar identidade existente, recebeu %+v", id)
	}
	if enroller.calls != 0 {
		t.Fatalf("não deveria chamar Enroll quando já há identidade salva, houve %d chamadas", enroller.calls)
	}
}

func TestEnsureIdentity_FailsWithoutTokenOrState(t *testing.T) {
	cfg := config.Config{StateFilePath: filepath.Join(t.TempDir(), "agent.json")}
	enroller := &fakeEnroller{}

	_, err := EnsureIdentity(context.Background(), cfg, enroller)
	if err == nil {
		t.Fatal("esperava erro quando não há estado nem ENROLLMENT_TOKEN")
	}
}

func TestEnsureIdentity_PropagatesEnrollError(t *testing.T) {
	cfg := config.Config{
		StateFilePath:   filepath.Join(t.TempDir(), "agent.json"),
		EnrollmentToken: "tok-invalido",
	}
	enroller := &fakeEnroller{err: errors.New("token inválido")}

	_, err := EnsureIdentity(context.Background(), cfg, enroller)
	if err == nil {
		t.Fatal("esperava erro propagado do enroller")
	}
}
