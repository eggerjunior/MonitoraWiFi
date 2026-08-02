// Interpolação espacial (IDW — inverse distance weighting) entre amostras
// reais do levantamento espacial (Fase 6) — gera uma grade contínua de
// qualidade de RTT estimada entre os pontos medidos. Nunca finge que uma
// célula distante de qualquer amostra real tem a mesma confiança de uma
// célula perto de uma medição de verdade: cada célula carrega a distância
// até a amostra mais próxima, usada pela UI pra reduzir a opacidade em
// áreas puramente estimadas (docs/architecture/04-estrategia-lidar.md:
// "todo pixel do heatmap fora de uma amostra real deve poder mostrar... a
// distância à amostra real mais próxima").
export interface HeatmapSample {
  positionX: number;
  positionZ: number;
  rttMs: number | null;
}

export interface HeatmapCell {
  x: number;
  z: number;
  valueMs: number | null;
  distanceToNearestSample: number;
}

export interface HeatmapGrid {
  cells: HeatmapCell[];
  minX: number;
  maxX: number;
  minZ: number;
  maxZ: number;
  cellWidth: number;
  cellHeight: number;
}

/**
 * Gera uma grade `gridSize` × `gridSize` cobrindo a caixa delimitadora das
 * amostras, com o valor de cada célula interpolado por IDW a partir das
 * amostras com RTT medido (nunca inventa RTT pra amostras que falharam —
 * elas só contam para a distância/confiança, não para o valor interpolado).
 */
export function computeIdwGrid(
  samples: HeatmapSample[],
  gridSize: number,
  power = 2
): HeatmapGrid | null {
  if (samples.length === 0) return null;

  const xs = samples.map((s) => s.positionX);
  const zs = samples.map((s) => s.positionZ);
  const minX = Math.min(...xs);
  const maxX = Math.max(...xs);
  const minZ = Math.min(...zs);
  const maxZ = Math.max(...zs);
  const rangeX = Math.max(maxX - minX, 0.5);
  const rangeZ = Math.max(maxZ - minZ, 0.5);
  const cellWidth = rangeX / gridSize;
  const cellHeight = rangeZ / gridSize;

  const measured = samples.filter((s) => s.rttMs !== null) as (HeatmapSample & { rttMs: number })[];

  const cells: HeatmapCell[] = [];
  for (let row = 0; row < gridSize; row++) {
    for (let col = 0; col < gridSize; col++) {
      const cx = minX + (col + 0.5) * cellWidth;
      const cz = minZ + (row + 0.5) * cellHeight;

      let distanceToNearestSample = Infinity;
      for (const s of samples) {
        const d = Math.hypot(s.positionX - cx, s.positionZ - cz);
        if (d < distanceToNearestSample) distanceToNearestSample = d;
      }

      let valueMs: number | null = null;
      if (measured.length > 0) {
        const exact = measured.find((s) => Math.hypot(s.positionX - cx, s.positionZ - cz) < 1e-6);
        if (exact) {
          valueMs = exact.rttMs;
        } else {
          let weightSum = 0;
          let valueSum = 0;
          for (const s of measured) {
            const d = Math.hypot(s.positionX - cx, s.positionZ - cz);
            const weight = 1 / Math.pow(d, power);
            weightSum += weight;
            valueSum += weight * s.rttMs;
          }
          valueMs = valueSum / weightSum;
        }
      }

      cells.push({ x: cx, z: cz, valueMs, distanceToNearestSample });
    }
  }

  return { cells, minX, maxX, minZ, maxZ, cellWidth, cellHeight };
}
