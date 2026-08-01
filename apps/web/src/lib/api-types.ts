// Tipos espelhando packages/contracts/openapi.yaml. Nesta fase são escritos à
// mão; a Fase 2 deve introduzir geração automática a partir do OpenAPI
// (ADR-005) para eliminar o risco de divergência manual entre backend e web.

export type Role = "owner" | "administrator" | "operator" | "viewer" | "auditor";

export interface User {
  id: string;
  organization_id: string;
  email: string;
  role: Role;
  mfa_enrolled: boolean;
  created_at: string;
}

export interface AuthSession {
  user: User;
  expires_at: string;
}

export interface ApiError {
  error: string;
  message: string;
  correlation_id: string;
}

export interface Organization {
  id: string;
  name: string;
  plan_tier: string;
  created_at: string;
}

export interface Site {
  id: string;
  organization_id: string;
  name: string;
  timezone: string;
  created_at: string;
}

export interface Page<T> {
  items: T[];
  page: number;
  page_size: number;
  total: number;
}
