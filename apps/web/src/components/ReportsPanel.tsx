"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";

import type { Report, ReportContent } from "@/lib/api-types";

// Painel interativo de relatórios (Fase 7): gera um relatório novo (POST via
// BFF) e permite expandir um relatório já gerado pra ver o conteúdo completo
// sob demanda (a listagem não traz `content` — ver handlers_reports.go). O
// servidor de verdade (apps/api) é a única fonte dos dados; este componente
// só orquestra as chamadas, nunca inventa um número.
export function ReportsPanel({
  siteId,
  initialReports,
}: {
  siteId: string;
  initialReports: Report[];
}) {
  const router = useRouter();
  const [isGenerating, setIsGenerating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [expanded, setExpanded] = useState<Record<string, ReportContent | "loading" | "error">>({});

  async function handleGenerate() {
    setIsGenerating(true);
    setError(null);
    try {
      const res = await fetch(`/api/sites/${siteId}/reports`, { method: "POST" });
      const body = await res.json();
      if (!res.ok) {
        setError(body.message ?? "Erro ao gerar relatório.");
        return;
      }
      router.refresh();
    } catch {
      setError("Erro de rede ao gerar relatório.");
    } finally {
      setIsGenerating(false);
    }
  }

  async function toggleExpand(reportId: string) {
    if (expanded[reportId] && expanded[reportId] !== "error") {
      setExpanded((prev) => {
        const next = { ...prev };
        delete next[reportId];
        return next;
      });
      return;
    }
    setExpanded((prev) => ({ ...prev, [reportId]: "loading" }));
    try {
      const res = await fetch(`/api/reports/${reportId}`);
      if (!res.ok) {
        setExpanded((prev) => ({ ...prev, [reportId]: "error" }));
        return;
      }
      const report: Report = await res.json();
      setExpanded((prev) => ({ ...prev, [reportId]: report.content ?? "error" }));
    } catch {
      setExpanded((prev) => ({ ...prev, [reportId]: "error" }));
    }
  }

  return (
    <div>
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold text-egg-text-primary">Relatórios gerados</h2>
        <button
          type="button"
          onClick={handleGenerate}
          disabled={isGenerating}
          className="rounded-md bg-egg-accent px-4 py-1.5 text-sm font-medium text-white disabled:opacity-50"
        >
          {isGenerating ? "Gerando…" : "Gerar relatório (últimos 7 dias)"}
        </button>
      </div>

      {error && <p className="mt-2 text-sm text-egg-critical">{error}</p>}

      {initialReports.length === 0 ? (
        <p className="mt-3 rounded-lg border border-dashed border-egg-border p-4 text-sm text-egg-text-secondary">
          Nenhum relatório gerado ainda.
        </p>
      ) : (
        <ul className="mt-3 space-y-2">
          {initialReports.map((report) => {
            const state = expanded[report.id];
            return (
              <li key={report.id} className="rounded-lg border border-egg-border bg-egg-surface p-4 text-sm">
                <div className="flex items-center justify-between">
                  <span className="text-egg-text-primary">
                    {new Date(report.period_start).toLocaleDateString()} –{" "}
                    {new Date(report.period_end).toLocaleDateString()}
                  </span>
                  <button
                    type="button"
                    onClick={() => toggleExpand(report.id)}
                    className="text-xs font-medium text-egg-accent"
                  >
                    {state && state !== "error" ? "Recolher" : "Ver conteúdo"}
                  </button>
                </div>
                <p className="mt-1 text-xs text-egg-text-disabled">
                  Gerado em {new Date(report.generated_at).toLocaleString()}
                </p>

                {state === "loading" && (
                  <p className="mt-2 text-xs text-egg-text-secondary">Carregando…</p>
                )}
                {state === "error" && (
                  <p className="mt-2 text-xs text-egg-critical">Erro ao carregar o conteúdo.</p>
                )}
                {state && state !== "loading" && state !== "error" && (
                  <div className="mt-3 space-y-2 border-t border-egg-border pt-3">
                    <p className="text-xs text-egg-text-secondary">
                      {state.anomaly_count}{" "}
                      {state.anomaly_count === 1 ? "anomalia" : "anomalias"} no período ·{" "}
                      {state.diagnoses.length}{" "}
                      {state.diagnoses.length === 1 ? "diagnóstico" : "diagnósticos"} ·{" "}
                      {state.recommendations.length}{" "}
                      {state.recommendations.length === 1 ? "recomendação" : "recomendações"}
                    </p>
                    {state.diagnoses.map((d, i) => (
                      <p key={i} className="text-xs text-egg-text-primary">
                        {d.summary}
                      </p>
                    ))}
                  </div>
                )}
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
