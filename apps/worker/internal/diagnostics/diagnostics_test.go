package diagnostics

import (
	"strings"
	"testing"
	"time"
)

func TestDiagnose_SemEvidencia(t *testing.T) {
	if got := Diagnose(nil); got != nil {
		t.Fatalf("esperava nil sem evidência, obtive %+v", got)
	}
}

func TestDiagnose_IgnoraAnomaliaNaDirecaoBoa(t *testing.T) {
	// Download acima da média é uma anomalia estatística, mas não é
	// evidência de lentidão — o motor não deve diagnosticar nada aqui.
	evidence := []AnomalyEvidence{
		{Metric: "speedtest_download_mbps", ObservedAt: fixedTime(0), Value: 500, BucketMean: 100, ZScore: 6},
	}
	if got := Diagnose(evidence); len(got) != 0 {
		t.Fatalf("esperava nenhum diagnóstico (direção boa), obtive %+v", got)
	}
}

func TestDiagnose_IgnoraMetricaDesconhecida(t *testing.T) {
	evidence := []AnomalyEvidence{
		{Metric: "metrica_nova_nao_mapeada", ObservedAt: fixedTime(0), Value: 1, BucketMean: 100, ZScore: 6},
	}
	if got := Diagnose(evidence); len(got) != 0 {
		t.Fatalf("esperava nenhum diagnóstico (métrica não mapeada), obtive %+v", got)
	}
}

func TestDiagnose_InternetSlow_UmaMetrica(t *testing.T) {
	evidence := []AnomalyEvidence{
		{Metric: "ping_latency_ms_p50", ObservedAt: fixedTime(0), Value: 400, BucketMean: 20, ZScore: 6},
	}
	diagnoses := Diagnose(evidence)
	if len(diagnoses) != 1 {
		t.Fatalf("esperava 1 diagnóstico, obtive %d", len(diagnoses))
	}
	d := diagnoses[0]
	if d.Category != CategoryInternetSlow {
		t.Fatalf("categoria esperada internet_slow, obtive %s", d.Category)
	}
	if d.Confidence != 0.5 {
		t.Fatalf("confiança esperada 0.5 (1 métrica), obtive %v", d.Confidence)
	}
	if d.Impact != "medium" {
		t.Fatalf("impacto esperado medium (z=6), obtive %s", d.Impact)
	}
	if d.Risk != "low" {
		t.Fatalf("risco esperado low (1 anomalia), obtive %s", d.Risk)
	}
	if !strings.Contains(d.Summary, "Internet lenta") {
		t.Fatalf("summary deveria mencionar 'Internet lenta': %s", d.Summary)
	}
}

func TestDiagnose_InternetSlow_TresMetricas_ConfiancaAlta(t *testing.T) {
	evidence := []AnomalyEvidence{
		{Metric: "ping_latency_ms_p50", ObservedAt: fixedTime(0), Value: 400, BucketMean: 20, ZScore: 9},
		{Metric: "speedtest_download_mbps", ObservedAt: fixedTime(1), Value: 5, BucketMean: 100, ZScore: 9},
		{Metric: "speedtest_upload_mbps", ObservedAt: fixedTime(2), Value: 1, BucketMean: 20, ZScore: 9},
	}
	diagnoses := Diagnose(evidence)
	if len(diagnoses) != 1 {
		t.Fatalf("esperava 1 diagnóstico, obtive %d", len(diagnoses))
	}
	d := diagnoses[0]
	if d.Confidence != 0.9 {
		t.Fatalf("confiança esperada 0.9 (3 métricas), obtive %v", d.Confidence)
	}
	if d.Impact != "high" {
		t.Fatalf("impacto esperado high (z=9), obtive %s", d.Impact)
	}
	if d.Risk != "medium" {
		t.Fatalf("risco esperado medium (3 anomalias), obtive %s", d.Risk)
	}
}

func TestDiagnose_WifiSlow_EvidenciaLAN(t *testing.T) {
	evidence := []AnomalyEvidence{
		{Metric: "speedtest_lan_download_mbps", ObservedAt: fixedTime(0), Value: 10, BucketMean: 500, ZScore: 7},
	}
	diagnoses := Diagnose(evidence)
	if len(diagnoses) != 1 || diagnoses[0].Category != CategoryWifiSlow {
		t.Fatalf("esperava 1 diagnóstico wifi_slow, obtive %+v", diagnoses)
	}
}

func TestDiagnose_AmbasCategorias_OrdenadoAlfabeticamente(t *testing.T) {
	evidence := []AnomalyEvidence{
		{Metric: "speedtest_lan_download_mbps", ObservedAt: fixedTime(0), Value: 10, BucketMean: 500, ZScore: 7},
		{Metric: "ping_latency_ms_p50", ObservedAt: fixedTime(1), Value: 400, BucketMean: 20, ZScore: 7},
	}
	diagnoses := Diagnose(evidence)
	if len(diagnoses) != 2 {
		t.Fatalf("esperava 2 diagnósticos, obtive %d", len(diagnoses))
	}
	if diagnoses[0].Category != CategoryInternetSlow || diagnoses[1].Category != CategoryWifiSlow {
		t.Fatalf("ordem inesperada: %s, %s", diagnoses[0].Category, diagnoses[1].Category)
	}
}

func TestRecommend_InternetSlow_Isolado(t *testing.T) {
	diagnoses := []Diagnosis{
		{Category: CategoryInternetSlow, Confidence: 0.5, Impact: "medium", Risk: "low",
			Evidence: []AnomalyEvidence{{Metric: "ping_latency_ms_p50"}}},
	}
	recs := Recommend(diagnoses)
	if len(recs) != 1 {
		t.Fatalf("esperava 1 recomendação, obtive %d", len(recs))
	}
	if !strings.Contains(recs[0].Action, "provedor") {
		t.Fatalf("recomendação de internet_slow deveria mencionar o provedor: %s", recs[0].Action)
	}
}

func TestRecommend_WifiSlow_SemInternetSlow_MencionaInternetNormal(t *testing.T) {
	diagnoses := []Diagnosis{
		{Category: CategoryWifiSlow, Confidence: 0.5, Impact: "medium", Risk: "low",
			Evidence: []AnomalyEvidence{{Metric: "speedtest_lan_download_mbps"}}},
	}
	recs := Recommend(diagnoses)
	if len(recs) != 1 {
		t.Fatalf("esperava 1 recomendação, obtive %d", len(recs))
	}
	if !strings.Contains(recs[0].Action, "internet permaneceu dentro do padrão") {
		t.Fatalf("recomendação deveria afirmar que a internet estava normal: %s", recs[0].Action)
	}
}

func TestRecommend_WifiSlow_ComInternetSlow_NaoAfirmaInternetNormal(t *testing.T) {
	diagnoses := []Diagnosis{
		{Category: CategoryInternetSlow, Confidence: 0.5, Impact: "medium", Risk: "low",
			Evidence: []AnomalyEvidence{{Metric: "ping_latency_ms_p50"}}},
		{Category: CategoryWifiSlow, Confidence: 0.5, Impact: "medium", Risk: "low",
			Evidence: []AnomalyEvidence{{Metric: "speedtest_lan_download_mbps"}}},
	}
	recs := Recommend(diagnoses)
	var wifiAction string
	for _, r := range recs {
		if r.Category == CategoryWifiSlow {
			wifiAction = r.Action
		}
	}
	if wifiAction == "" {
		t.Fatal("esperava recomendação wifi_slow")
	}
	if strings.Contains(wifiAction, "internet permaneceu dentro do padrão") {
		t.Fatalf("recomendação não deveria afirmar internet normal quando internet_slow também foi diagnosticada: %s", wifiAction)
	}
	if !strings.Contains(wifiAction, "também mostrou desvio") {
		t.Fatalf("recomendação deveria mencionar a correlação com internet_slow: %s", wifiAction)
	}
}

func fixedTime(offsetHours int) time.Time {
	return time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC).Add(time.Duration(offsetHours) * time.Hour)
}
