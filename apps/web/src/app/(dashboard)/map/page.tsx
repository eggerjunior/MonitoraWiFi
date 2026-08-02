import Link from "next/link";

import { apiFetch } from "@/lib/api-server";
import { getCurrentSite } from "@/lib/current-site";
import type { Page as ApiPage, SpatialSurvey } from "@/lib/api-types";
import { computeIdwGrid } from "@/lib/spatial-heatmap";

// Levantamento espacial (Fase 6): lista os levantamentos reais enviados
// pelo app iOS (posição ARKit + RTT/SSID medidos no próprio ponto) e, para
// um levantamento selecionado, um heatmap 2D top-down (eixo X × eixo Z da
// sessão ARKit) — grade interpolada por IDW (spatial-heatmap.ts) entre as
// amostras reais, com opacidade proporcional à distância até a amostra mais
// próxima (nunca 100% opaca fora de medição real), mais os pontos sólidos
// das amostras de verdade por cima. Sem biblioteca de gráfico (nenhuma
// existe no projeto), SVG puro. Não é malha 3D (fora de escopo desta
// primeira fatia — ver docs/architecture/06-roadmap.md Fase 6).
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
            Visão de cima (eixo X × eixo Z da sessão ARKit) — a área de fundo é uma estimativa
            interpolada (IDW) entre as amostras reais, cada vez mais transparente quanto mais
            longe de uma medição real; os pontos sólidos são as amostras de verdade. Posições
            são relativas à origem da sessão de captura, não georreferenciadas.
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

  const heatmap = computeIdwGrid(
    samples.map((s) => ({ positionX: s.position_x, positionZ: s.position_z, rttMs: s.rtt_ms })),
    24
  );

  // Células a essa distância (ou mais) de qualquer amostra real ficam na
  // opacidade mínima — heurística ligada à própria escala do levantamento,
  // não um valor fixo arbitrário. Nunca chega a 100% opaco fora de uma
  // medição real (máximo 0.7), pra sempre ser visualmente distinguível dos
  // pontos sólidos (amostra de verdade).
  const maxRelevantDistance = Math.max(rangeX, rangeZ) * 0.4;
  function opacityForDistance(distance: number): number {
    const confidence = Math.max(0, 1 - distance / maxRelevantDistance);
    return 0.15 + confidence * 0.55;
  }

  return (
    <div className="mt-3 overflow-x-auto rounded-lg border border-egg-border bg-egg-surface p-4">
      <svg width={size} height={size} role="img" aria-label="Heatmap interpolado do levantamento, visão de cima">
        <rect x={0} y={0} width={size} height={size} fill="none" />
        {heatmap &&
          heatmap.cells.map((cell, i) => {
            if (cell.valueMs === null) return null;
            const [cx, cy] = toSvg(cell.x, cell.z);
            const w = heatmap.cellWidth * scale;
            const h = heatmap.cellHeight * scale;
            return (
              <rect
                key={`cell-${i}`}
                x={cx - w / 2}
                y={cy - h / 2}
                width={w}
                height={h}
                fill={rttColor(cell.valueMs)}
                opacity={opacityForDistance(cell.distanceToNearestSample)}
              >
                <title>
                  {`~${cell.valueMs.toFixed(0)} ms estimado (interpolado, ${cell.distanceToNearestSample.toFixed(
                    1
                  )} m da amostra real mais próxima)`}
                </title>
              </rect>
            );
          })}
        {samples.map((s, i) => {
          const [cx, cy] = toSvg(s.position_x, s.position_z);
          return (
            <circle
              key={`point-${i}`}
              cx={cx}
              cy={cy}
              r={5}
              fill={rttColor(s.rtt_ms)}
              stroke="white"
              strokeWidth={1}
            >
              <title>{s.rtt_ms === null ? "Falhou (amostra real)" : `${s.rtt_ms.toFixed(1)} ms (amostra real)`}</title>
            </circle>
          );
        })}
      </svg>
      <div className="mt-2 flex flex-wrap gap-4 text-xs text-egg-text-secondary">
        <LegendDot color="#0F893E" label="Boa (< 50ms)" />
        <LegendDot color="#A3690A" label="Média (50–150ms)" />
        <LegendDot color="#DC2626" label="Ruim (> 150ms)" />
        <LegendDot color="#9CA3AF" label="Falhou" />
      </div>
      <p className="mt-2 text-xs text-egg-text-disabled">
        Pontos com borda branca = amostra real medida. Área sombreada sem borda = estimativa por
        interpolação, nunca uma medição — passe o mouse sobre qualquer célula pra ver a distância
        até a amostra real mais próxima.
      </p>
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
