-- Fase 1: schema mínimo para autenticação, RBAC, organizações e sites.
-- Ver docs/architecture/05-modelo-dados.md para o modelo completo (fases futuras).

CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE organizations (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name         text NOT NULL,
    plan_tier    text NOT NULL DEFAULT 'standard',
    created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE sites (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id  uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name             text NOT NULL,
    timezone         text NOT NULL DEFAULT 'UTC',
    address          jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at       timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_sites_organization_id ON sites(organization_id);

CREATE TABLE users (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id   uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email             citext NOT NULL,
    password_hash     text NOT NULL,
    role              text NOT NULL CHECK (role IN ('owner','administrator','operator','viewer','auditor')),
    mfa_enrolled_at   timestamptz,
    created_at        timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_users_email ON users (email);
CREATE INDEX idx_users_organization_id ON users(organization_id);

CREATE TABLE sessions (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash    text NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    expires_at    timestamptz NOT NULL,
    revoked_at    timestamptz
);
CREATE UNIQUE INDEX idx_sessions_token_hash ON sessions(token_hash);
CREATE INDEX idx_sessions_user_id ON sessions(user_id);

CREATE TABLE audit_log (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id  uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    actor_user_id    uuid REFERENCES users(id) ON DELETE SET NULL,
    action           text NOT NULL,
    resource_type    text NOT NULL,
    resource_id      uuid,
    diff             jsonb,
    actor_ip         inet,
    created_at       timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_audit_log_org_created ON audit_log(organization_id, created_at DESC);
