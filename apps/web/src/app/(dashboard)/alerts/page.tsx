import { apiFetch } from "@/lib/api-server";
import { getCurrentSite } from "@/lib/current-site";
import type { Anomaly, Page as ApiPage } from "@/lib/api-types";
import { provenance } from "@egger/tokens";

// Alertas (Fase 4): única fonte real hoje é o worker de anomalias
// estatísticas (Fase 7, `GET /sites/{id}/anomalies`) — nunca reportado sem
// histórico suficiente (baseline por hora/dia da semana). Não existe ainda
// um schema de alerta com severidade/status/ciclo de vida próprio (ver
// docs/architecture/05-modelo-dados.md, entidade ALERT — não implementada);
// severidade aqui é derivada do z-score na própria UI, não inventada.
const CRITICAL_Z_SCORE = 5;

export default async function AlertsPage() {
  const current = await getCurrentSite();
  if ("error" in current) {
    return <EmptyState message={current.error} />;
  }
  const { site } = current;

  const anomalies = await apiFetch<ApiPage<Anomaly>>(
    `/sites/${site.id}/anomalies?page=1&page_size=50`,
  );

  return (
    <div className="max-w-3xl space-y-6">
      <div>
        <h1 className="text-xl font-semibold text-egg-text-primary">Alertas</h1>
        <p className="mt-1 text-sm text-egg-text-secondary">
          Site: {site.name} · {anomalies.total}{" "}
          {anomalies.total === 1 ? "anomalia detectada" : "anomalias detectadas"}
        </p>
      </div>

      {anomalies.items.length === 0 ? (
        <EmptyCard message="Nenhuma anomalia detectada ainda — pode ser que ainda não haja histórico suficiente (o worker nunca reporta sem uma amostra mínima do bucket) ou que a rede esteja dentro do padrão." />
      ) : (
        <ul className="space-y-2">
          {anomalies.items.map((a) => (
            <AnomalyRow key={a.id} anomaly={a} />
          ))}
        </ul>
      )}

      <p className="text-xs text-egg-text-disabled">
        Hoje só a latência de ping é monitorada (
        <code className="rounded bg-egg-background px-1 py-0.5">ping_latency_ms_p50</code>
        ) — cobertura de speed test ainda não implementada (ver
        docs/architecture/06-roadmap.md, Fase 7).
      </p>
    </div>
  );
}

function AnomalyRow({ anomaly }: { anomaly: Anomaly }) {
  const isCritical = Math.abs(anomaly.z_score) >= CRITICAL_Z_SCORE;
  return (
    <li className="rounded-lg border border-egg-border bg-egg-surface p-4">
      <div className="flex items-center justify-between">
        <span className="font-medium text-egg-text-primary">{metricLabel(anomaly.metric)}</span>
        <span
          className={`text-xs font-medium ${isCritical ? "text-egg-critical" : "text-egg-warning"}`}
        >
          {isCritical ? "Crítico" : "Atenção"}
        </span>
      </div>
      <div className="mt-2 grid grid-cols-2 gap-3 text-sm sm:grid-cols-4">
        <Field label="Valor observado" value={anomaly.value.toFixed(1)} />
        <Field label="Média esperada" value={anomaly.bucket_mean.toFixed(1)} />
        <Field label="Z-score" value={anomaly.z_score.toFixed(2)} />
        <Field label="Amostras no bucket" value={String(anomaly.bucket_size)} />
      </div>
      <p className="mt-2 text-xs text-egg-text-secondary">
        Observado em {new Date(anomaly.observed_at).toLocaleString()}
      </p>
      <p className="mt-1 text-xs text-egg-text-disabled">Fonte: {provenance.estimated.label}</p>
    </li>
  );
}

function metricLabel(metric: string): string {
  if (metric === "ping_latency_ms_p50") return "Latência de ping (p50)";
  return metric;
}

function Field({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="text-xs text-egg-text-secondary">{label}</div>
      <div className="font-medium text-egg-text-primary">{value}</div>
    </div>
  );
}

function EmptyState({ message }: { message: string }) {
  return (
    <div className="max-w-3xl">
      <h1 className="text-xl font-semibold text-egg-text-primary">Alertas</h1>
      <div className="mt-4 rounded-lg border border-dashed border-egg-border p-6 text-sm text-egg-text-secondary">
        {message}
      </div>
    </div>
  );
}

function EmptyCard({ message }: { message: string }) {
  return (
    <div className="rounded-lg border border-dashed border-egg-border p-4 text-sm text-egg-text-secondary">
      {message}
    </div>
  );
}
