package baseline

import (
	"math"
	"testing"
	"time"
)

// fixedTime retorna um horário real (não um valor mágico) numa terça-feira
// às 20h — usado para controlar o bucket exercitado no teste.
func fixedTime(dayOffset, hour int) time.Time {
	// 2026-08-04 é uma terça-feira.
	base := time.Date(2026, 8, 4, hour, 0, 0, 0, time.UTC)
	return base.AddDate(0, 0, dayOffset*7) // mesma terça-feira, semanas diferentes
}

func TestCompute_MeanAndStdDev(t *testing.T) {
	// Valores com média e desvio-padrão conhecidos de antemão (dados de
	// teste sintéticos, não uma medição real — nunca apresentados ao
	// usuário como tal): 10, 12, 14, 16, 18 → média 14, stddev populacional = sqrt(8) ≈ 2.828
	samples := []Sample{
		{Time: fixedTime(0, 20), Value: 10},
		{Time: fixedTime(1, 20), Value: 12},
		{Time: fixedTime(2, 20), Value: 14},
		{Time: fixedTime(3, 20), Value: 16},
		{Time: fixedTime(4, 20), Value: 18},
	}

	b := Compute(samples)
	key := BucketKey{Weekday: time.Tuesday, Hour: 20}
	stats, ok := b.Buckets[key]
	if !ok {
		t.Fatalf("esperava bucket %+v presente, buckets: %+v", key, b.Buckets)
	}
	if stats.Count != 5 {
		t.Errorf("Count = %d, esperado 5", stats.Count)
	}
	if math.Abs(stats.Mean-14) > 0.001 {
		t.Errorf("Mean = %f, esperado 14", stats.Mean)
	}
	wantStdDev := math.Sqrt(8)
	if math.Abs(stats.StdDev-wantStdDev) > 0.001 {
		t.Errorf("StdDev = %f, esperado %f", stats.StdDev, wantStdDev)
	}
}

func TestCompute_SeparaBucketsPorHoraEDiaDaSemana(t *testing.T) {
	tuesday20h := time.Date(2026, 8, 4, 20, 0, 0, 0, time.UTC)
	wednesday6h := time.Date(2026, 8, 5, 6, 0, 0, 0, time.UTC)

	samples := []Sample{
		{Time: tuesday20h, Value: 100},
		{Time: wednesday6h, Value: 5},
	}
	b := Compute(samples)

	if len(b.Buckets) != 2 {
		t.Fatalf("esperava 2 buckets distintos, obtive %d: %+v", len(b.Buckets), b.Buckets)
	}
	tueStats := b.Buckets[BucketKey{Weekday: time.Tuesday, Hour: 20}]
	wedStats := b.Buckets[BucketKey{Weekday: time.Wednesday, Hour: 6}]
	if tueStats.Mean != 100 || wedStats.Mean != 5 {
		t.Fatalf("valores não separados corretamente por bucket: terça=%+v, quarta=%+v", tueStats, wedStats)
	}
}

func TestDetect_FlagsOutlier(t *testing.T) {
	// Baseline histórico: 10 amostras em torno de 20ms (desvio pequeno).
	var historical []Sample
	values := []float64{18, 19, 20, 21, 22, 19, 20, 21, 20, 20}
	for i, v := range values {
		historical = append(historical, Sample{Time: fixedTime(i, 20), Value: v})
	}
	b := Compute(historical)

	// Amostra recente MUITO fora do padrão (200ms, mesma hora/dia da semana).
	recent := []Sample{
		{Time: fixedTime(20, 20), Value: 200},
	}

	anomalies := Detect(recent, b, 3.0)
	if len(anomalies) != 1 {
		t.Fatalf("esperava 1 anomalia detectada, obtive %d", len(anomalies))
	}
	if anomalies[0].Sample.Value != 200 {
		t.Errorf("anomalia inesperada: %+v", anomalies[0])
	}
	if anomalies[0].ZScore < 3.0 {
		t.Errorf("ZScore = %f, esperado >= 3.0 para ser reportado", anomalies[0].ZScore)
	}
}

func TestDetect_NaoFlagaValorNormal(t *testing.T) {
	var historical []Sample
	values := []float64{18, 19, 20, 21, 22, 19, 20, 21, 20, 20}
	for i, v := range values {
		historical = append(historical, Sample{Time: fixedTime(i, 20), Value: v})
	}
	b := Compute(historical)

	// 21ms está bem dentro do padrão histórico (média ~20, desvio pequeno).
	recent := []Sample{{Time: fixedTime(20, 20), Value: 21}}

	anomalies := Detect(recent, b, 3.0)
	if len(anomalies) != 0 {
		t.Fatalf("esperava 0 anomalias para valor dentro do padrão, obtive %d: %+v", len(anomalies), anomalies)
	}
}

func TestDetect_SemBaselineSuficiente_NuncaFlaga(t *testing.T) {
	// Só 2 amostras históricas — abaixo de MinBucketSamples (5). Mesmo um
	// valor bem diferente não deve ser reportado: não há baseline
	// confiável o bastante para afirmar que é anômalo (nunca inventar
	// certeza que os dados não sustentam).
	historical := []Sample{
		{Time: fixedTime(0, 20), Value: 20},
		{Time: fixedTime(1, 20), Value: 20},
	}
	b := Compute(historical)

	recent := []Sample{{Time: fixedTime(2, 20), Value: 500}}
	anomalies := Detect(recent, b, 3.0)
	if len(anomalies) != 0 {
		t.Fatalf("esperava 0 anomalias sem baseline suficiente (MinBucketSamples=%d), obtive %d", MinBucketSamples, len(anomalies))
	}
}

func TestDetect_BucketDesconhecido_NuncaFlaga(t *testing.T) {
	b := Compute([]Sample{{Time: fixedTime(0, 20), Value: 20}, {Time: fixedTime(1, 20), Value: 20}, {Time: fixedTime(2, 20), Value: 20}, {Time: fixedTime(3, 20), Value: 20}, {Time: fixedTime(4, 20), Value: 20}})

	// Bucket completamente diferente (outro dia/hora) sem nenhum histórico.
	recent := []Sample{{Time: time.Date(2026, 8, 5, 3, 0, 0, 0, time.UTC), Value: 999}}
	anomalies := Detect(recent, b, 3.0)
	if len(anomalies) != 0 {
		t.Fatalf("esperava 0 anomalias para bucket sem nenhum histórico, obtive %d", len(anomalies))
	}
}
