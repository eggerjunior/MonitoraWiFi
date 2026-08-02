import { apiFetch } from "@/lib/api-server";
import { getCurrentSite } from "@/lib/current-site";
import { ReportsPanel } from "@/components/ReportsPanel";
import type {
  Diagnosis,
  ImpactLevel,
  Page as ApiPage,
  Recommendation,
  Report,
  RiskLevel,
} from "@/lib/api-types";

// Relatórios (Fase 7, motor de correlação): diagnósticos e recomendações são
// calculados pelo worker (apps/worker/internal/diagnostics) a partir de
// anomalias reais — nunca gerados sem evidência. Relatórios são agregados
// sob demanda pelo backend (POST /sites/{id}/reports), não pré-gerados.
export default async function ReportsPage() {
  const current = await getCurrentSite();
  if ("error" in current) {
    return (
      <div className="max-w-3xl">
        <h1 className="text-xl font-semibold text-egg-text-primary">Relatórios</h1>
        <div className="mt-4 rounded-lg border border-dashed border-egg-border p-6 text-sm text-egg-text-secondary">
          {current.error}
        </div>
      </div>
    );
  }
  const { site } = current;

  const [diagnoses, recommendations, reports] = await Promise.all([
    apiFetch<ApiPage<Diagnosis>>(`/sites/${site.id}/diagnoses?page=1&page_size=50`),
    apiFetch<ApiPage<Recommendation>>(`/sites/${site.id}/recommendations?page=1&page_size=50`),
    apiFetch<ApiPage<Report>>(`/sites/${site.id}/reports?page=1&page_size=20`),
  ]);

  return (
    <div className="max-w-3xl space-y-8">
      <div>
        <h1 className="text-xl font-semibold text-egg-text-primary">Relatórios</h1>
        <p className="mt-1 text-sm text-egg-text-secondary">
          Site: {site.name} · motor de correlação (Fase 7) — nunca diagnostica ou
          recomenda sem evidência real (anomalias estatísticas já detectadas).
        </p>
      </div>

      <section>
        <h2 className="text-sm font-semibold text-egg-text-primary">Diagnósticos</h2>
        {diagnoses.items.length === 0 ? (
          <p className="mt-2 rounded-lg border border-dashed border-egg-border p-4 text-sm text-egg-text-secondary">
            Nenhum diagnóstico ainda — pode ser que ainda não haja evidência
            suficiente (anomalias reais) ou que a rede esteja dentro do padrão.
          </p>
        ) : (
          <ul className="mt-2 space-y-2">
            {diagnoses.items.map((d) => (
              <DiagnosisRow key={d.id} diagnosis={d} />
            ))}
          </ul>
        )}
      </section>

      <section>
        <h2 className="text-sm font-semibold text-egg-text-primary">Recomendações</h2>
        {recommendations.items.length === 0 ? (
          <p className="mt-2 rounded-lg border border-dashed border-egg-border p-4 text-sm text-egg-text-secondary">
            Nenhuma recomendação ainda — cada recomendação exige um diagnóstico
            real por trás, nunca é gerada sozinha.
          </p>
        ) : (
          <ul className="mt-2 space-y-2">
            {recommendations.items.map((r) => (
              <RecommendationRow key={r.id} recommendation={r} />
            ))}
          </ul>
        )}
      </section>

      <section>
        <ReportsPanel siteId={site.id} initialReports={reports.items} />
      </section>
    </div>
  );
}

function DiagnosisRow({ diagnosis }: { diagnosis: Diagnosis }) {
  return (
    <li className="rounded-lg border border-egg-border bg-egg-surface p-4 text-sm">
      <div className="flex items-center justify-between">
        <span className="font-medium text-egg-text-primary">{categoryLabel(diagnosis.category)}</span>
        <div className="flex gap-1.5">
          <Badge label="impacto" level={diagnosis.impact} />
          <Badge label="risco" level={diagnosis.risk} />
        </div>
      </div>
      <p className="mt-2 text-egg-text-secondary">{diagnosis.summary}</p>
      <p className="mt-2 text-xs text-egg-text-disabled">
        Confiança: {(diagnosis.confidence * 100).toFixed(0)}% · {diagnosis.evidence.length}{" "}
        {diagnosis.evidence.length === 1 ? "anomalia como evidência" : "anomalias como evidência"}
      </p>
    </li>
  );
}

function RecommendationRow({ recommendation }: { recommendation: Recommendation }) {
  return (
    <li className="rounded-lg border border-egg-border bg-egg-surface p-4 text-sm">
      <div className="flex items-center justify-between">
        <span className="font-medium text-egg-text-primary">Ação recomendada</span>
        <div className="flex gap-1.5">
          <Badge label="impacto" level={recommendation.impact} />
          <Badge label="risco" level={recommendation.risk} />
        </div>
      </div>
      <p className="mt-2 text-egg-text-secondary">{recommendation.action}</p>
      <p className="mt-2 text-xs text-egg-text-disabled">
        Confiança: {(recommendation.confidence * 100).toFixed(0)}%
      </p>
    </li>
  );
}

function Badge({ label, level }: { label: string; level: ImpactLevel | RiskLevel }) {
  const color =
    level === "high" ? "text-egg-critical" : level === "medium" ? "text-egg-warning" : "text-egg-success";
  return (
    <span className={`text-xs font-medium ${color}`}>
      {label}: {levelLabel(level)}
    </span>
  );
}

function levelLabel(level: ImpactLevel | RiskLevel): string {
  if (level === "high") return "alto";
  if (level === "medium") return "médio";
  return "baixo";
}

function categoryLabel(category: Diagnosis["category"]): string {
  if (category === "internet_slow") return "Internet lenta";
  if (category === "wifi_slow") return "Wi-Fi lento";
  return category;
}
