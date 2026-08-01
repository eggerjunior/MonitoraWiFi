import { apiFetch } from "@/lib/api-server";
import type {
  Organization,
  Page as ApiPage,
  PingTest,
  Site,
  SpeedTest,
} from "@/lib/api-types";
import { provenance, type MetricSource } from "@egger/tokens";

// Primeiro dashboard real da Fase 2 (roadmap): dados de ping/speed test do
// agente local, direto do backend — nunca simulados. Antes de haver um
// agente enrolado em algum site, os estados abaixo são honestamente vazios,
// não "0%" inventado (Seção 2.1).
export default async function InternetPage() {
  const orgs = await apiFetch<ApiPage<Organization>>("/organizations?page=1&page_size=1");
  const org = orgs.items[0];

  if (!org) {
    return (
      <EmptyState message="Nenhuma organização cadastrada ainda — nada para exibir." />
    );
  }

  const sites = await apiFetch<ApiPage<Site>>(
    `/sites?organization_id=${org.id}&page=1&page_size=1`,
  );
  const site = sites.items[0];

  if (!site) {
    return <EmptyState message="Nenhum site cadastrado nesta organização ainda." />;
  }

  const [pingTests, speedTests] = await Promise.all([
    apiFetch<ApiPage<PingTest>>(`/sites/${site.id}/ping-tests?page=1&page_size=20`),
    apiFetch<ApiPage<SpeedTest>>(`/sites/${site.id}/speed-tests?page=1&page_size=5`),
  ]);

  return (
    <div className="max-w-3xl space-y-8">
      <div>
        <h1 className="text-xl font-semibold text-egg-text-primary">Internet</h1>
        <p className="mt-1 text-sm text-egg-text-secondary">
          Site: {site.name} · dados do agente local ({pingTests.total} testes de ping,{" "}
          {speedTests.total} speed tests registrados)
        </p>
      </div>

      <section>
        <h2 className="text-sm font-semibold text-egg-text-primary">
          Speed test mais recente
        </h2>
        {speedTests.items.length === 0 ? (
          <EmptyCard message="Nenhum speed test recebido ainda. Requer um agente local enrolado e SPEEDTEST_DOWNLOAD_URL/SPEEDTEST_UPLOAD_URL configurados." />
        ) : (
          <SpeedTestCard test={speedTests.items[0]} />
        )}
      </section>

      <section>
        <h2 className="text-sm font-semibold text-egg-text-primary">
          Testes de ping recentes
        </h2>
        {pingTests.items.length === 0 ? (
          <EmptyCard message="Nenhum resultado de ping recebido ainda. Requer um agente local enrolado neste site." />
        ) : (
          <ul className="mt-2 space-y-2">
            {pingTests.items.map((t) => (
              <PingTestRow key={t.id} test={t} />
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}

function SpeedTestCard({ test }: { test: SpeedTest }) {
  return (
    <div className="mt-2 grid grid-cols-2 gap-4 rounded-lg border border-egg-border bg-egg-surface p-4 sm:grid-cols-4">
      <Metric label="Download" value={formatMbps(test.download_mbps)} />
      <Metric label="Upload" value={formatMbps(test.upload_mbps)} />
      <Metric label="Latência ociosa" value={formatMs(test.idle_latency_ms)} />
      <Metric label="Bufferbloat" value={formatMs(test.bufferbloat_ms)} />
      <ProvenanceTag source="agent_http" className="col-span-2 sm:col-span-4" />
    </div>
  );
}

function PingTestRow({ test }: { test: PingTest }) {
  const source = protocolToSource(test.protocol);
  return (
    <li className="rounded-lg border border-egg-border bg-egg-surface p-3">
      <div className="flex items-center justify-between">
        <span className="font-medium text-egg-text-primary">{test.target}</span>
        <span className="text-xs uppercase text-egg-text-secondary">{test.protocol}</span>
      </div>
      <div className="mt-1 grid grid-cols-3 gap-3 text-sm">
        <Metric label="p50" value={formatMs(test.latency_ms_p50)} compact />
        <Metric label="perda" value={formatPct(test.packet_loss_pct)} compact />
        <Metric label="jitter" value={formatMs(test.jitter_ms)} compact />
      </div>
      <ProvenanceTag source={source} className="mt-2" />
    </li>
  );
}

function Metric({
  label,
  value,
  compact,
}: {
  label: string;
  value: string;
  compact?: boolean;
}) {
  return (
    <div>
      <div className={compact ? "text-xs text-egg-text-secondary" : "text-xs text-egg-text-secondary"}>
        {label}
      </div>
      <div className={compact ? "text-sm font-medium text-egg-text-primary" : "text-lg font-semibold text-egg-text-primary"}>
        {value}
      </div>
    </div>
  );
}

function ProvenanceTag({ source, className = "" }: { source: MetricSource; className?: string }) {
  const info = provenance[source];
  return (
    <div className={`text-xs text-egg-text-disabled ${className}`}>
      Fonte: {info.label}
    </div>
  );
}

function EmptyState({ message }: { message: string }) {
  return (
    <div className="max-w-3xl">
      <h1 className="text-xl font-semibold text-egg-text-primary">Internet</h1>
      <div className="mt-4 rounded-lg border border-dashed border-egg-border p-6 text-sm text-egg-text-secondary">
        {message}
      </div>
    </div>
  );
}

function EmptyCard({ message }: { message: string }) {
  return (
    <div className="mt-2 rounded-lg border border-dashed border-egg-border p-4 text-sm text-egg-text-secondary">
      {message}
    </div>
  );
}

function protocolToSource(protocol: PingTest["protocol"]): MetricSource {
  switch (protocol) {
    case "icmp":
      return "agent_icmp";
    case "tcp":
      return "agent_tcp";
    case "http":
      return "agent_http";
    case "dns":
      return "agent_dns";
  }
}

function formatMbps(value: number | null): string {
  return value === null ? "Indisponível" : `${value.toFixed(1)} Mbps`;
}

function formatMs(value: number | null): string {
  return value === null ? "Indisponível" : `${value.toFixed(1)} ms`;
}

function formatPct(value: number | null): string {
  return value === null ? "Indisponível" : `${value.toFixed(1)}%`;
}
