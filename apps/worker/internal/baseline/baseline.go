// Package baseline implementa detecção de anomalias estatísticas
// explicáveis (Fase 7, roadmap: "Anomalias estatísticas explicáveis
// (baseline por hora/dia da semana)"). Não depende de nenhum hardware ou
// integração externa — é um algoritmo puro sobre séries de amostras
// (latência de ping, throughput de speed test etc.) já coletadas nas
// Fases 2-4.
//
// Nunca reporta anomalia a partir de um bucket com poucas amostras
// históricas (MinBucketSamples) — "sem dado suficiente" é sempre preferível
// a um falso positivo (Seção 2.1, mesmo princípio de nunca inventar certeza
// que os dados não sustentam).
package baseline

import (
	"math"
	"time"
)

// Sample é uma observação de uma métrica em um instante — genérico o
// bastante para ping (latência) e speed test (download/upload/etc).
type Sample struct {
	Time  time.Time
	Value float64
}

// BucketKey agrupa amostras por hora do dia e dia da semana — um baseline
// separado para "terça-feira às 20h" e "domingo às 6h", por exemplo,
// porque o padrão normal de uso de rede varia por esse ciclo.
type BucketKey struct {
	Weekday time.Weekday
	Hour    int
}

type Stats struct {
	Mean   float64
	StdDev float64
	Count  int
}

type Baseline struct {
	Buckets map[BucketKey]Stats
}

// Compute agrupa as amostras por bucket e calcula média/desvio-padrão
// (populacional) de cada um.
func Compute(samples []Sample) Baseline {
	grouped := map[BucketKey][]float64{}
	for _, s := range samples {
		key := BucketKey{Weekday: s.Time.Weekday(), Hour: s.Time.Hour()}
		grouped[key] = append(grouped[key], s.Value)
	}

	buckets := make(map[BucketKey]Stats, len(grouped))
	for key, values := range grouped {
		buckets[key] = computeStats(values)
	}
	return Baseline{Buckets: buckets}
}

func computeStats(values []float64) Stats {
	n := len(values)
	if n == 0 {
		return Stats{}
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(n)

	var sumSq float64
	for _, v := range values {
		d := v - mean
		sumSq += d * d
	}
	return Stats{Mean: mean, StdDev: math.Sqrt(sumSq / float64(n)), Count: n}
}

// MinBucketSamples é o mínimo de amostras históricas num bucket para seu
// baseline ser considerado confiável o bastante para gerar anomalias.
const MinBucketSamples = 5

// Anomaly é uma amostra cujo desvio em relação ao baseline do seu bucket
// excedeu o threshold (em desvios-padrão — "z-score").
type Anomaly struct {
	Sample     Sample
	BucketMean float64
	BucketSize int
	ZScore     float64
}

// Detect compara `samples` (tipicamente o período recente) contra `b`
// (baseline computado de um período histórico anterior e maior). threshold
// típico: 3.0 (99.7% de confiança sob normalidade aproximada).
func Detect(samples []Sample, b Baseline, threshold float64) []Anomaly {
	var anomalies []Anomaly
	for _, s := range samples {
		key := BucketKey{Weekday: s.Time.Weekday(), Hour: s.Time.Hour()}
		stats, ok := b.Buckets[key]
		if !ok || stats.Count < MinBucketSamples || stats.StdDev == 0 {
			continue
		}
		z := (s.Value - stats.Mean) / stats.StdDev
		if math.Abs(z) >= threshold {
			anomalies = append(anomalies, Anomaly{
				Sample:     s,
				BucketMean: stats.Mean,
				BucketSize: stats.Count,
				ZScore:     z,
			})
		}
	}
	return anomalies
}
