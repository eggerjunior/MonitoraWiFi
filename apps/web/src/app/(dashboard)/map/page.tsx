import Link from "next/link";

import { apiFetch } from "@/lib/api-server";
import { getCurrentSite } from "@/lib/current-site";
import type { Page as ApiPage, SpatialSurvey } from "@/lib/api-types";

// Levantamento espacial (Fase 6): lista os levantamentos reais enviados
// pelo app iOS (posição ARKit + RTT/SSID medidos no próprio ponto) e, para
// um levantamento selecionado, um scatter 2D top-down (eixo X × eixo Z da
// sessão ARKit) colorido pela qualidade do RTT — sem biblioteca de gráfico
// (nenhuma existe no projeto), SVG puro, mesmo padrão de "sem chart lib" já
// usado no resto do produto. Não é um heatmap 3D/malha (fora de escopo
// desta primeira fatia — ver docs/architecture/06-roadmap.md Fase 6).
export default async function MapPage({
  searchParams,
}: {
  searchParams: Promise<{ survey?: string }>;
}) {
  const current = await getCurrentSite();
  if ("error" in current) {
    return (
      <div className="max-w-3xl">
        <h1 className="text-xl font-semibold text-egg-text-primary">Levantamento espacial</h1>
        <div className="mt-4 rounded-lg border border-dashed border-egg-border p-6 text-sm text-egg-text-secondary">
          {current.error}
        </div>
      </div>
    );
  }
  const { site } = current;
  const { survey: selectedSurveyId } = await searchParams;

  const surveyPage = await apiFetch<ApiPage<SpatialSurvey>>(
    `/sites/${site.id}/spatial-surveys?page=1&page_size=50`
  );

  const selectedSurvey = selectedSurveyId
    ? await apiFetch<SpatialSurvey>(`/spatial-surveys/${selectedSurveyId}`)
    : null;

  return (
    <div className="max-w-3xl space-y-8">
      <div>
        <h1 className="text-xl font-semibold text-egg-text-primary">Levantamento espacial</h1>
        <p className="mt-1 text-sm text-egg-text-secondary">
          Site: {site.name} · capturado pelo app iOS (ARKit/LiDAR quando disponível)
        </p>
      </div>

      <section>
        <h2 className="text-sm font-semibold text-egg-text-primary">
          Levantamentos ({surveyPage.total})
        </h2>
        {surveyPage.items.length === 0 ? (
          <div className="mt-2 rounded-lg border border-dashed border-egg-border p-4 text-sm text-egg-text-secondary">
            Nenhum levantamento ainda — use o app iOS (&ldquo;Mapa&rdquo; → &ldquo;Novo
            levantamento&rdquo;) para capturar um.
          </div>
        ) : (
          <ul className="mt-2 space-y-2">
            {surveyPage.items.map((s) => (
              <li key={s.id}>
                <Link
                  href={`/map?survey=${s.id}`}
                  className={`block rounded-lg border p-3 text-sm ${
                    s.id === selectedSurveyId
                      ? "border-egg-accent bg-egg-surface"
                      : "border-egg-border bg-egg-surface"
                  }`}
                >
                  <div className="flex items-center justify-between">
                    <span className="font-medium text-egg-text-primary">{s.name}</span>
                    <span className="text-xs text-egg-text-secondary">
                      {s.sample_count} amostra(s)
                    </span>
                  </div>
                  <p className="mt-1 text-xs text-egg-text-secondary">
                    {s.device_model} · {s.lidar_used ? "LiDAR" : "sem LiDAR"} ·{" "}
                    {new Date(s.created_at).toLocaleString()}
                  </p>
                </Link>
              </li>
            ))}
          </ul>
        )}
      </section>

      {selectedSurvey && (
        <section>
          <h2 className="text-sm font-semibold text-egg-text-primary">{selectedSurvey.name}</h2>
          <p className="mt-1 text-xs text-egg-text-secondary">
            Visão de cima (eixo X × eixo Z da sessão ARKit) — cada ponto é uma amostra real,
            colorida pela latência (RTT) medida no local. Posições são relativas à origem da
            sessão de captura, não georreferenciadas.
          </p>
          <SurveyScatterPlot samples={selectedSurvey.samples ?? []} />
        </section>
      )}
    </div>
  );
}

function rttColor(rttMs: number | null): string {
  if (rttMs === null) return "#9CA3AF"; // falhou — cinza
  if (rttMs < 50) return "#0F893E"; // boa — verde (mesmo tom do design token success)
  if (rttMs < 150) return "#A3690A"; // média — âmbar (mesmo tom do warning)
  return "#DC2626"; // ruim — vermelho
}

function SurveyScatterPlot({ samples }: { samples: { position_x: number; position_z: number; rtt_ms: number | null }[] }) {
  if (samples.length === 0) {
    return (
      <div className="mt-3 rounded-lg border border-dashed border-egg-border p-4 text-sm text-egg-text-secondary">
        Este levantamento não tem amostras.
      </div>
    );
  }

  const xs = samples.map((s) => s.position_x);
  const zs = samples.map((s) => s.position_z);
  const minX = Math.min(...xs);
  const maxX = Math.max(...xs);
  const minZ = Math.min(...zs);
  const maxZ = Math.max(...zs);
  const rangeX = Math.max(maxX - minX, 0.5);
  const rangeZ = Math.max(maxZ - minZ, 0.5);

  const size = 320;
  const padding = 20;
  const scale = (size - padding * 2) / Math.max(rangeX, rangeZ);

  function toSvg(x: number, z: number): [number, number] {
    const svgX = padding + (x - minX) * scale;
    const svgY = padding + (z - minZ) * scale;
    return [svgX, svgY];
  }

  return (
    <div className="mt-3 overflow-x-auto rounded-lg border border-egg-border bg-egg-surface p-4">
      <svg width={size} height={size} role="img" aria-label="Mapa de amostras do levantamento, visão de cima">
        <rect x={0} y={0} width={size} height={size} fill="none" />
        {samples.map((s, i) => {
          const [cx, cy] = toSvg(s.position_x, s.position_z);
          return <circle key={i} cx={cx} cy={cy} r={5} fill={rttColor(s.rtt_ms)} />;
        })}
      </svg>
      <div className="mt-2 flex gap-4 text-xs text-egg-text-secondary">
        <LegendDot color="#0F893E" label="Boa (< 50ms)" />
        <LegendDot color="#A3690A" label="Média (50–150ms)" />
        <LegendDot color="#DC2626" label="Ruim (> 150ms)" />
        <LegendDot color="#9CA3AF" label="Falhou" />
      </div>
    </div>
  );
}

function LegendDot({ color, label }: { color: string; label: string }) {
  return (
    <span className="flex items-center gap-1">
      <span className="inline-block h-2 w-2 rounded-full" style={{ backgroundColor: color }} />
      {label}
    </span>
  );
}
