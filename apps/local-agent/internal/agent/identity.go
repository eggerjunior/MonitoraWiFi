package agent

import (
	"context"
	"fmt"

	"egger/local-agent/internal/apiclient"
	"egger/local-agent/internal/config"
	"egger/local-agent/internal/state"
)

// Enroller é satisfeito por *apiclient.Client; a interface existe para
// permitir um fake em teste (Seção 21: agente testável sem backend real).
type Enroller interface {
	Enroll(ctx context.Context, enrollmentToken, hostname, platform, version string) (apiclient.EnrollResult, error)
}

// EnsureIdentity carrega a identidade persistida (state.Load) ou, se ainda
// não existir, troca o ENROLLMENT_TOKEN por uma credencial de longa duração
// (ADR-006) e a persiste — o token de enrolamento nunca é reusado depois
// disso (a API o invalida no primeiro uso).
func EnsureIdentity(ctx context.Context, cfg config.Config, enroller Enroller) (state.Identity, error) {
	existing, err := state.Load(cfg.StateFilePath)
	if err != nil {
		return state.Identity{}, fmt.Errorf("carregar identidade local: %w", err)
	}
	if existing != nil {
		return *existing, nil
	}

	if cfg.EnrollmentToken == "" {
		return state.Identity{}, fmt.Errorf("nenhuma identidade salva em %s e ENROLLMENT_TOKEN não foi definido", cfg.StateFilePath)
	}

	result, err := enroller.Enroll(ctx, cfg.EnrollmentToken, cfg.Hostname, cfg.Platform, cfg.Version)
	if err != nil {
		return state.Identity{}, fmt.Errorf("enrolar agente: %w", err)
	}

	id := state.Identity{AgentID: result.AgentID, AgentSecret: result.AgentSecret, SiteID: result.SiteID}
	if err := state.Save(cfg.StateFilePath, id); err != nil {
		return state.Identity{}, fmt.Errorf("persistir identidade: %w", err)
	}
	return id, nil
}
