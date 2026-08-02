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

export type PingProtocol = "icmp" | "tcp" | "http" | "dns";

export interface PingTest {
  id: string;
  agent_id: string;
  target: string;
  protocol: PingProtocol;
  latency_ms_p50: number | null;
  latency_ms_p95: number | null;
  latency_ms_p99: number | null;
  jitter_ms: number | null;
  packet_loss_pct: number | null;
  executed_at: string;
}

export type CommandStatus = "pending" | "claimed" | "completed" | "failed";

export interface PingCommandResult {
  target: string;
  protocol: PingProtocol;
  latency_ms_p50: number | null;
  latency_ms_p95: number | null;
  latency_ms_p99: number | null;
  jitter_ms: number | null;
  packet_loss_pct: number | null;
  executed_at: string;
}

export interface BatchPingCommandResult {
  protocol: PingProtocol;
  results: PingCommandResult[];
  executed_at: string;
}

export interface DnsLookupCommandResult {
  hostname: string;
  addresses: string[];
  duration_ms: number;
  executed_at: string;
}

export interface TracerouteHop {
  hop: number;
  address: string;
  rtt_ms: number | null;
}

export interface TracerouteCommandResult {
  target: string;
  reached: boolean;
  hops: TracerouteHop[];
  executed_at: string;
}

export interface SslCheckCommandResult {
  target: string;
  port: number;
  valid_now: boolean;
  verify_error: string;
  not_before: string;
  not_after: string;
  days_until_expiry: number;
  issuer: string;
  subject: string;
  dns_names: string[];
  executed_at: string;
}

export interface RdapEvent {
  action: string;
  date: string;
}

export interface RdapResult {
  query: string;
  server: string;
  object_class_name: string;
  handle: string;
  name: string;
  status: string[];
  events: RdapEvent[];
  nameservers?: string[];
  raw: Record<string, unknown>;
}

export interface HttpRequestCommandResult {
  url: string;
  method: string;
  status_code: number;
  status_text: string;
  headers: Record<string, string>;
  body_snippet: string;
  body_truncated: boolean;
  content_length: number;
  duration_ms: number;
  executed_at: string;
}

export interface LanScanCommandResult {
  cidr: string;
  hosts: string[];
  executed_at: string;
}

export interface WakeOnLanCommandResult {
  mac_address: string;
  broadcast_ip: string;
  port: number;
  executed_at: string;
}

export interface PortScanCommandResult {
  target: string;
  open_ports: number[];
  executed_at: string;
}

export interface DnsResolverResult {
  resolver: string;
  addresses: string[];
  duration_ms: number;
  error: string;
}

export interface DnsResolverCompareCommandResult {
  hostname: string;
  resolvers: DnsResolverResult[];
  executed_at: string;
}

export interface Anomaly {
  id: string;
  metric: string;
  observed_at: string;
  value: number;
  bucket_mean: number;
  bucket_size: number;
  z_score: number;
  detected_at: string;
}

export interface Command {
  id: string;
  site_id: string;
  agent_id: string;
  type: string;
  params: Record<string, unknown>;
  status: CommandStatus;
  // Formato depende de `type` — ver PingCommandResult, DnsLookupCommandResult,
  // TracerouteCommandResult para os shapes conhecidos.
  result: Record<string, unknown> | null;
  error?: string | null;
  created_at: string;
  claimed_at?: string | null;
  completed_at?: string | null;
}

export interface UniFiDevice {
  id: string;
  external_id: string;
  mac_address: string;
  ip_address: string;
  name: string;
  model: string;
  state: string;
  firmware_version: string;
  features: string[];
  interfaces: string[];
  // external_id do dispositivo upstream (ex.: o switch a que um AP está
  // conectado) — vazio pro dispositivo raiz (gateway). Confirmado em
  // 2026-08-02 contra a instalação real (GET .../devices/{id}).
  uplink_device_id: string;
}

export interface UniFiDeviceList {
  items: UniFiDevice[];
}

export type UniFiClientType = "WIRED" | "WIRELESS";

export interface UniFiClient {
  id: string;
  external_id: string;
  type: UniFiClientType;
  name: string;
  ip_address: string;
  mac_address: string;
  connected_at: string | null;
  uplink_device_id: string;
}

export interface UniFiClientList {
  items: UniFiClient[];
}

export type SpeedTestMode = "internet" | "lan" | "http";

export interface SpeedTest {
  id: string;
  agent_id: string;
  mode: SpeedTestMode;
  download_mbps: number | null;
  upload_mbps: number | null;
  idle_latency_ms: number | null;
  loaded_latency_ms: number | null;
  bufferbloat_ms: number | null;
  jitter_ms: number | null;
  executed_at: string;
}
