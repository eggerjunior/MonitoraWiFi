// Package store define os modelos de domínio e as interfaces de repositório da
// Fase 1. Interfaces (não só implementação Postgres) existem para permitir fakes
// em teste de handler, sem precisar de um banco real (Seção 21: "não exigir uma
// controladora real nos testes automatizados" — aplicado aqui ao banco).
package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("recurso não encontrado")
var ErrConflict = errors.New("conflito de dados (ex.: e-mail já cadastrado)")

type Role string

const (
	RoleOwner         Role = "owner"
	RoleAdministrator Role = "administrator"
	RoleOperator      Role = "operator"
	RoleViewer        Role = "viewer"
	RoleAuditor       Role = "auditor"
)

type Organization struct {
	ID        uuid.UUID
	Name      string
	PlanTier  string
	CreatedAt time.Time
}

type Site struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Name           string
	Timezone       string
	CreatedAt      time.Time
}

type User struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Email          string
	PasswordHash   string
	Role           Role
	MFAEnrolledAt  *time.Time
	CreatedAt      time.Time
}

type Session struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	CreatedAt time.Time
	ExpiresAt time.Time
	RevokedAt *time.Time
}

type AuditEntry struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	ActorUserID    *uuid.UUID
	Action         string
	ResourceType   string
	ResourceID     *uuid.UUID
	Diff           map[string]any
	ActorIP        string
	CreatedAt      time.Time
}

type Page struct {
	Page     int
	PageSize int
}

type OrganizationStore interface {
	List(ctx context.Context, page Page) ([]Organization, int, error)
}

type SiteStore interface {
	ListByOrganization(ctx context.Context, orgID uuid.UUID, page Page) ([]Site, int, error)
	Get(ctx context.Context, id uuid.UUID) (Site, error)
}

type UserStore interface {
	GetByEmail(ctx context.Context, email string) (User, error)
	Get(ctx context.Context, id uuid.UUID) (User, error)
}

type SessionStore interface {
	Create(ctx context.Context, s Session) error
	GetByTokenHash(ctx context.Context, tokenHash string) (Session, error)
	Revoke(ctx context.Context, id uuid.UUID) error
}

type AuditStore interface {
	Record(ctx context.Context, e AuditEntry) error
}
