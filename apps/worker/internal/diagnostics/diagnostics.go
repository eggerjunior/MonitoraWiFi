// Package diagnostics implementa o motor de correlação/diagnóstico da
// Fase 7 (roadmap: "motor de correlação/diagnóstico (Internet lenta,
// Wi-Fi lento, cliente desconectando), recomendações com evidência/
// confiança/impacto/risco"). Algoritmo puro baseado em regras sobre
// anomalias já detectadas pelo package baseline — não é machine learning,
// é correlação explicável: cada diagnóstico carrega as anomalias reais que
// o sustentam, nunca uma conclusão sem evidência rastreável.
//
// Categoria "cliente desconectando" foi deliberadamente deixada de fora
// desta fatia: `unifi_clients` é um snapshot de estado atual (substituído a
// cada sincronização, não uma série histórica — ver migração
// 0005_unifi_inventory), então não existe hoje nenhuma fonte real que prove
// reconexões repetidas de um cliente ao longo do tempo. Adicionar essa
// categoria sem essa fonte real violaria o mesmo princípio que já guia todo
// o resto deste pacote (nunca diagnosticar sem evidência real).
package diagnostics

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

const (
	CategoryInternetSlow = "internet_slow"
	CategoryWifiSlow     = "wifi_slow"
)

type direction int

const (
	higherIsBad direction = iota
	lowerIsBad
)

type metricSpec struct {
	category  string
	direction direction
	label     string
}

// metricSpecs é a única fonte de verdade sobre quais métricas alimentam
// qual categoria de diagnóstico, e em que direção um desvio é "ruim" (ex.:
// download abaixo da média é ruim; download acima da média não é evidência
// de lentidão nenhuma, mesmo que estatisticamente anômalo). Métrica não
// listada aqui é ignorada pelo motor — nunca usada como evidência por
// engano.
var metricSpecs = map[string]metricSpec{
	"ping_latency_ms_p50":         {category: CategoryInternetSlow, direction: higherIsBad, label: "latência de ping (internet)"},
	"speedtest_download_mbps":     {category: CategoryInternetSlow, direction: lowerIsBad, label: "velocidade de download (internet)"},
	"speedtest_upload_mbps":       {category: CategoryInternetSlow, direction: lowerIsBad, label: "velocidade de upload (internet)"},
	"speedtest_bufferbloat_ms":    {category: CategoryInternetSlow, direction: higherIsBad, label: "bufferbloat (internet)"},
	"speedtest_lan_download_mbps": {category: CategoryWifiSlow, direction: lowerIsBad, label: "velocidade de download (LAN)"},
	"speedtest_lan_upload_mbps":   {category: CategoryWifiSlow, direction: lowerIsBad, label: "velocidade de upload (LAN)"},
}

// AnomalyEvidence é a anomalia real (já persistida pelo package baseline)
// que este motor consome como entrada — nenhum campo aqui é inventado, tudo
// vem de uma linha real da tabela `anomalies`.
type AnomalyEvidence struct {
	ID         string
	Metric     string
	ObservedAt time.Time
	Value      float64
	BucketMean float64
	ZScore     float64
}

// Diagnosis é uma conclusão do motor de correlação para uma categoria,
// sustentada por uma ou mais AnomalyEvidence reais.
type Diagnosis struct {
	Category    string
	Summary     string
	Confidence  float64 // 0–1
	Impact      string  // low | medium | high
	Risk        string  // low | medium | high
	Evidence    []AnomalyEvidence
	WindowStart time.Time
	WindowEnd   time.Time
}

// Recommendation é a ação sugerida a partir de um Diagnosis — nunca gerada
// sem um diagnóstico (e portanto sem evidência) por trás.
type Recommendation struct {
	Category   string
	Action     string
	Confidence float64
	Impact     string
	Risk       string
	Evidence   []AnomalyEvidence
}

// Diagnose agrupa evidência (anomalias reais) por categoria e produz um
// Diagnosis por categoria com evidência suficiente. Nunca produz um
// Diagnosis para uma categoria sem nenhuma anomalia real na direção "ruim"
// daquela categoria — mesmo princípio de honestidade do package baseline
// ("sem dado suficiente" é sempre preferível a inventar um diagnóstico).
func Diagnose(evidence []AnomalyEvidence) []Diagnosis {
	byCategory := map[string][]AnomalyEvidence{}
	for _, e := range evidence {
		spec, known := metricSpecs[e.Metric]
		if !known {
			continue
		}
		if !isBadDirection(e, spec) {
			continue
		}
		byCategory[spec.category] = append(byCategory[spec.category], e)
	}

	var out []Diagnosis
	for category, items := range byCategory {
		out = append(out, buildDiagnosis(category, items))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Category < out[j].Category })
	return out
}

func isBadDirection(e AnomalyEvidence, spec metricSpec) bool {
	switch spec.direction {
	case higherIsBad:
		return e.Value > e.BucketMean
	case lowerIsBad:
		return e.Value < e.BucketMean
	default:
		return false
	}
}

func buildDiagnosis(category string, items []AnomalyEvidence) Diagnosis {
	distinctMetrics := map[string]bool{}
	var sumAbsZ float64
	windowStart, windowEnd := items[0].ObservedAt, items[0].ObservedAt
	for _, e := range items {
		distinctMetrics[e.Metric] = true
		sumAbsZ += math.Abs(e.ZScore)
		if e.ObservedAt.Before(windowStart) {
			windowStart = e.ObservedAt
		}
		if e.ObservedAt.After(windowEnd) {
			windowEnd = e.ObservedAt
		}
	}
	avgAbsZ := sumAbsZ / float64(len(items))

	confidence := confidenceFor(len(distinctMetrics))
	impact := impactFor(avgAbsZ)
	risk := riskFor(len(items))

	return Diagnosis{
		Category:    category,
		Summary:     summaryFor(category, items, distinctMetrics, avgAbsZ, windowStart, windowEnd),
		Confidence:  confidence,
		Impact:      impact,
		Risk:        risk,
		Evidence:    items,
		WindowStart: windowStart,
		WindowEnd:   windowEnd,
	}
}

// confidenceFor cresce com o número de métricas distintas que corroboram o
// mesmo diagnóstico — uma única métrica anômala é bem menos convincente
// que três métricas independentes apontando na mesma direção.
func confidenceFor(distinctMetrics int) float64 {
	switch {
	case distinctMetrics >= 3:
		return 0.9
	case distinctMetrics == 2:
		return 0.75
	default:
		return 0.5
	}
}

// impactFor usa os mesmos limiares de z-score já adotados como "crítico"
// em apps/web (Alertas, CRITICAL_Z_SCORE=5) e apps/ios (AlertsView) — não
// inventa uma escala nova.
func impactFor(avgAbsZ float64) string {
	switch {
	case avgAbsZ >= 8:
		return "high"
	case avgAbsZ >= 5:
		return "medium"
	default:
		return "low"
	}
}

// riskFor: um problema recorrente (várias anomalias na janela) é maior
// risco de ser um estado degradado persistente do que um evento isolado.
func riskFor(evidenceCount int) string {
	switch {
	case evidenceCount >= 5:
		return "high"
	case evidenceCount >= 2:
		return "medium"
	default:
		return "low"
	}
}

func summaryFor(category string, items []AnomalyEvidence, distinctMetrics map[string]bool, avgAbsZ float64, windowStart, windowEnd time.Time) string {
	title := categoryTitle(category)
	metrics := metricLabels(distinctMetrics)
	return fmt.Sprintf(
		"%s: %d %s em %s entre %s e %s, desvio médio de %.1fσ da média histórica do site.",
		title, len(items), anomaliaReaisPhrase(len(items)), strings.Join(metrics, ", "),
		windowStart.Format(time.RFC3339), windowEnd.Format(time.RFC3339), avgAbsZ,
	)
}

// anomaliaReaisPhrase evita concatenar "s" cegamente num plural irregular
// (real → reais, não "reals") — bug real que a validação desta mesma fase
// contra Postgres pegou.
func anomaliaReaisPhrase(count int) string {
	if count == 1 {
		return "anomalia real"
	}
	return "anomalias reais"
}

func categoryTitle(category string) string {
	switch category {
	case CategoryInternetSlow:
		return "Internet lenta"
	case CategoryWifiSlow:
		return "Wi-Fi lento"
	default:
		return category
	}
}

func metricLabels(distinct map[string]bool) []string {
	labels := make([]string, 0, len(distinct))
	for metric := range distinct {
		if spec, ok := metricSpecs[metric]; ok {
			labels = append(labels, spec.label)
		} else {
			labels = append(labels, metric)
		}
	}
	sort.Strings(labels)
	return labels
}

// Recommend gera uma recomendação por diagnóstico — recebe o lote inteiro
// (não um Diagnosis isolado) porque o texto de "Wi-Fi lento" precisa saber
// se "Internet lenta" também foi diagnosticada na mesma janela, pra nunca
// afirmar "a internet está normal" quando isso não é verdade.
func Recommend(diagnoses []Diagnosis) []Recommendation {
	hasInternetSlow := false
	for _, d := range diagnoses {
		if d.Category == CategoryInternetSlow {
			hasInternetSlow = true
		}
	}

	out := make([]Recommendation, 0, len(diagnoses))
	for _, d := range diagnoses {
		out = append(out, Recommendation{
			Category:   d.Category,
			Action:     actionFor(d, hasInternetSlow),
			Confidence: d.Confidence,
			Impact:     d.Impact,
			Risk:       d.Risk,
			Evidence:   d.Evidence,
		})
	}
	return out
}

func actionFor(d Diagnosis, hasInternetSlow bool) string {
	metrics := strings.Join(metricLabels(distinctMetricsOf(d.Evidence)), ", ")
	switch d.Category {
	case CategoryInternetSlow:
		return fmt.Sprintf(
			"Verificar o link com o provedor de internet (modem/ONU, cabo/fibra de entrada, ou a WAN em uso caso haja failover configurado) — %s mostrou desvio real do padrão histórico deste site na janela analisada.",
			metrics,
		)
	case CategoryWifiSlow:
		if hasInternetSlow {
			return fmt.Sprintf(
				"Verificar a rede local (Wi-Fi, switch, cabeamento) entre o agente e o roteador — %s mostrou desvio real dentro da própria LAN. A internet também mostrou desvio no mesmo período: os dois problemas podem estar relacionados (ex.: um link de WAN sobrecarregando o roteador) ou ser independentes — investigar a rede local não dispensa verificar o link do provedor também.",
				metrics,
			)
		}
		return fmt.Sprintf(
			"Verificar a rede local (Wi-Fi, switch, cabeamento) entre o agente e o roteador — %s mostrou desvio real dentro da própria LAN, enquanto a internet permaneceu dentro do padrão no mesmo período, o que aponta para um problema local, não do provedor.",
			metrics,
		)
	default:
		return fmt.Sprintf("Investigar %s — desvio real detectado sem regra de recomendação específica ainda.", metrics)
	}
}

func distinctMetricsOf(evidence []AnomalyEvidence) map[string]bool {
	out := map[string]bool{}
	for _, e := range evidence {
		out[e.Metric] = true
	}
	return out
}
