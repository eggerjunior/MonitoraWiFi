import { apiFetch } from "@/lib/api-server";
import { getCurrentSite } from "@/lib/current-site";
import type {
  Anomaly,
  Page as ApiPage,
  PingTest,
  SpeedTest,
} from "@/lib/api-types";
import { provenance } from "@egger/tokens";

// Histórico (Fase 4): séries reais já coletadas — ping/speed test do
// agente (Fase 2) e anomalias estatísticas (Fase 7). Sem biblioteca de
// gráfico (nenhuma existe no projeto ainda) — segue o padrão de lista
// paginada já usado no resto do produto (ver internet/page.tsx). Distinto
// de "Relatórios" (Fase 7, geração de arquivo) — aqui é só visualização do
// que já existe.
export default async function HistoryPage() {
  const current = await getCurrentSite();
  if ("error" in current) {
    return <EmptyState message={current.error} />;
  }
  const { site } = current;

  const [pingTests, speedTests, anomalies] = await Promise.all([
    apiFetch<ApiPage<PingTest>>(`/sites/${site.id}/ping-tests?page=1&page_size=20`),
    apiFetch<ApiPage<SpeedTest>>(`/sites/${site.id}/speed-tests?page=1&page_size=20`),
    apiFetch<ApiPage<Anomaly>>(`/sites/${site.id}/anomalies?page=1&page_size=20`),
  ]);

  return (
    <div className="max-w-3xl space-y-8">
      <div>
        <h1 className="text-xl font-semibold text-egg-text-primary">Histórico</h1>
        <p className="mt-1 text-sm text-egg-text-secondary">Site: {site.name}</p>
      </div>

      <section>
        <h2 className="text-sm font-semibold text-egg-text-primary">
          Speed tests ({speedTests.total})
        </h2>
        {speedTests.items.length === 0 ? (
          <EmptyCard message="Nenhum speed test registrado ainda." />
        ) : (
          <ul className="mt-2 space-y-2">
            {speedTests.items.map((t) => (
              <li key={t.id} className="rounded-lg border border-egg-border bg-egg-surface p-3">
                <div className="flex items-center justify-between">
                  <span className="font-medium text-egg-text-primary">
                    {formatMbps(t.download_mbps)} ↓ / {formatMbps(t.upload_mbps)} ↑
                  </span>
                  <span className="text-xs uppercase text-egg-text-secondary">{t.mode}</span>
                </div>
                <p className="mt-1 text-xs text-egg-text-secondary">
                  {new Date(t.executed_at).toLocaleString()}
                </p>
              </li>
            ))}
          </ul>
        )}
      </section>

      <section>
        <h2 className="text-sm font-semibold text-egg-text-primary">
          Testes de ping ({pingTests.total})
        </h2>
        {pingTests.items.length === 0 ? (
          <EmptyCard message="Nenhum teste de ping registrado ainda." />
        ) : (
          <ul className="mt-2 space-y-2">
            {pingTests.items.map((t) => (
              <li key={t.id} className="rounded-lg border border-egg-border bg-egg-surface p-3">
                <div className="flex items-center justify-between">
                  <span className="font-medium text-egg-text-primary">{t.target}</span>
                  <span className="text-xs text-egg-text-secondary">{formatMs(t.latency_ms_p50)}</span>
                </div>
                <p className="mt-1 text-xs text-egg-text-secondary">
                  {new Date(t.executed_at).toLocaleString()}
                </p>
              </li>
            ))}
          </ul>
        )}
      </section>

      <section>
        <h2 className="text-sm font-semibold text-egg-text-primary">
          Anomalias ({anomalies.total})
        </h2>
        {anomalies.items.length === 0 ? (
          <EmptyCard message="Nenhuma anomalia detectada ainda." />
        ) : (
          <ul className="mt-2 space-y-2">
            {anomalies.items.map((a) => (
              <li key={a.id} className="rounded-lg border border-egg-border bg-egg-surface p-3">
                <div className="flex items-center justify-between">
                  <span className="font-medium text-egg-text-primary">{a.metric}</span>
                  <span className="text-xs text-egg-text-secondary">
                    {a.value.toFixed(1)} (esperado {a.bucket_mean.toFixed(1)})
                  </span>
                </div>
                <p className="mt-1 text-xs text-egg-text-secondary">
                  {new Date(a.observed_at).toLocaleString()}
                </p>
                <p className="mt-1 text-xs text-egg-text-disabled">
                  Fonte: {provenance.estimated.label}
                </p>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}

function formatMbps(value: number | null): string {
  return value === null ? "Indisponível" : `${value.toFixed(1)} Mbps`;
}

function formatMs(value: number | null): string {
  return value === null ? "Indisponível" : `${value.toFixed(1)} ms`;
}

function EmptyState({ message }: { message: string }) {
  return (
    <div className="max-w-3xl">
      <h1 className="text-xl font-semibold text-egg-text-primary">Histórico</h1>
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
