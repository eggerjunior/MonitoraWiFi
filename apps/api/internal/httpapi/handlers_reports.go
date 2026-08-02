// Relatórios (Fase 7, "relatórios completos"): gerados sob demanda,
// agregando anomalias/diagnósticos/recomendações reais de um período —
// nunca pré-gerados nem armazenados em blob storage externo (não há essa
// infraestrutura neste projeto ainda), o conteúdo inteiro fica em
// `reports.content` (jsonb).
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"

	"egger/api/internal/store"
)

// reportFetchLimit é o teto de itens (mais recentes primeiro, por
// site — não por período) considerados na agregação de cada fonte. Não há
// filtro por período nas queries existentes de anomalies/diagnoses/
// recommendations (ListBySite pagina por inserção, não por período), então
// o handler busca até este teto e filtra pelo período em memória — honesto
// o bastante pro volume real de dados deste produto hoje (Fase 7 só foi
// destravada com ~1 dia de histórico real); revisar se o volume crescer
// muito além disso.
const reportFetchLimit = 500

const defaultReportWindow = 7 * 24 * time.Hour

type createReportRequest struct {
	PeriodStart *string `json:"period_start"`
	PeriodEnd   *string `json:"period_end"`
}

type reportContent struct {
	PeriodStart       string                 `json:"period_start"`
	PeriodEnd         string                 `json:"period_end"`
	AnomalyCount      int                    `json:"anomaly_count"`
	AnomaliesByMetric map[string]int         `json:"anomalies_by_metric"`
	Diagnoses         []reportDiagnosis      `json:"diagnoses"`
	Recommendations   []reportRecommendation `json:"recommendations"`
}

type reportDiagnosis struct {
	Category    string  `json:"category"`
	Summary     string  `json:"summary"`
	Confidence  float64 `json:"confidence"`
	Impact      string  `json:"impact"`
	Risk        string  `json:"risk"`
	WindowStart string  `json:"window_start"`
	WindowEnd   string  `json:"window_end"`
}

type reportRecommendation struct {
	Category   string  `json:"category"`
	Action     string  `json:"action"`
	Confidence float64 `json:"confidence"`
	Impact     string  `json:"impact"`
	Risk       string  `json:"risk"`
}

func (s *Server) handleCreateReport(w http.ResponseWriter, r *http.Request) {
	correlationID := correlationIDFromContext(r.Context())
	user, _ := userFromContext(r.Context())

	siteID, err := uuid.Parse(r.PathValue("siteId"))
	if err != nil {
		writeError(w, correlationID, http.StatusBadRequest, "invalid_site_id", "siteId inválido")
		return
	}

	periodEnd := time.Now().UTC()
	periodStart := periodEnd.Add(-defaultReportWindow)

	if r.ContentLength != 0 {
		var req createReportRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "corpo da requisição inválido")
			return
		}
		if req.PeriodStart != nil {
			t, err := time.Parse(time.RFC3339, *req.PeriodStart)
			if err != nil {
				writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "period_start precisa ser RFC3339")
				return
			}
			periodStart = t
		}
		if req.PeriodEnd != nil {
			t, err := time.Parse(time.RFC3339, *req.PeriodEnd)
			if err != nil {
				writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "period_end precisa ser RFC3339")
				return
			}
			periodEnd = t
		}
	}
	if !periodStart.Before(periodEnd) {
		writeError(w, correlationID, http.StatusBadRequest, "invalid_body", "period_start precisa ser anterior a period_end")
		return
	}

	content, err := s.buildReportContent(r.Context(), siteID, periodStart, periodEnd)
	if err != nil {
		writeError(w, correlationID, http.StatusInternalServerError, "internal_error", "erro ao agregar dados do relatório")
		return
	}
	contentJSON, err := json.Marshal(content)
	if err != nil {
		writeError(w, correlationID, http.StatusInternalServerError, "internal_error", "erro ao serializar relatório")
		return
	}

	report, err := s.reports.Create(r.Context(), store.Report{
		SiteID:      siteID,
		Kind:        "diagnostics_summary",
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		Content:     contentJSON,
		GeneratedBy: &user.ID,
	})
	if err != nil {
		writeError(w, correlationID, http.StatusInternalServerError, "internal_error", "erro ao gravar relatório")
		return
	}

	writeJSON(w, http.StatusCreated, reportToJSON(report, true))
}

// buildReportContent agrega anomalias/diagnósticos/recomendações reais do
// site na janela pedida — nunca inventa uma seção vazia como se fosse
// "sem problema": ausência de diagnósticos/recomendações significa
// exatamente isso, sem problema real detectado na janela (ou histórico
// insuficiente pro worker ter avaliado, mesma honestidade de sempre).
func (s *Server) buildReportContent(ctx context.Context, siteID uuid.UUID, periodStart, periodEnd time.Time) (reportContent, error) {
	anomalies, _, err := s.anomalies.ListBySite(ctx, siteID, store.Page{Page: 1, PageSize: reportFetchLimit})
	if err != nil {
		return reportContent{}, err
	}
	diagnoses, _, err := s.diagnoses.ListBySite(ctx, siteID, store.Page{Page: 1, PageSize: reportFetchLimit})
	if err != nil {
		return reportContent{}, err
	}
	recommendations, _, err := s.recommendations.ListBySite(ctx, siteID, store.Page{Page: 1, PageSize: reportFetchLimit})
	if err != nil {
		return reportContent{}, err
	}

	byMetric := map[string]int{}
	anomalyCount := 0
	for _, a := range anomalies {
		if a.ObservedAt.Before(periodStart) || a.ObservedAt.After(periodEnd) {
			continue
		}
		anomalyCount++
		byMetric[a.Metric]++
	}

	var reportDiagnoses []reportDiagnosis
	for _, d := range diagnoses {
		if d.WindowEnd.Before(periodStart) || d.WindowEnd.After(periodEnd) {
			continue
		}
		reportDiagnoses = append(reportDiagnoses, reportDiagnosis{
			Category:    d.Category,
			Summary:     d.Summary,
			Confidence:  d.Confidence,
			Impact:      d.Impact,
			Risk:        d.Risk,
			WindowStart: d.WindowStart.Format(time.RFC3339),
			WindowEnd:   d.WindowEnd.Format(time.RFC3339),
		})
	}

	diagnosisCategory := map[uuid.UUID]string{}
	for _, d := range diagnoses {
		diagnosisCategory[d.ID] = d.Category
	}
	var reportRecommendations []reportRecommendation
	for _, rec := range recommendations {
		if rec.CreatedAt.Before(periodStart) || rec.CreatedAt.After(periodEnd) {
			continue
		}
		reportRecommendations = append(reportRecommendations, reportRecommendation{
			Category:   diagnosisCategory[rec.DiagnosisID],
			Action:     rec.Action,
			Confidence: rec.Confidence,
			Impact:     rec.Impact,
			Risk:       rec.Risk,
		})
	}

	return reportContent{
		PeriodStart:       periodStart.Format(time.RFC3339),
		PeriodEnd:         periodEnd.Format(time.RFC3339),
		AnomalyCount:      anomalyCount,
		AnomaliesByMetric: byMetric,
		Diagnoses:         reportDiagnoses,
		Recommendations:   reportRecommendations,
	}, nil
}

func (s *Server) handleListReports(w http.ResponseWriter, r *http.Request) {
	correlationID := correlationIDFromContext(r.Context())

	siteID, err := uuid.Parse(r.PathValue("siteId"))
	if err != nil {
		writeError(w, correlationID, http.StatusBadRequest, "invalid_site_id", "siteId inválido")
		return
	}

	page := parsePage(r)
	reports, total, err := s.reports.ListBySite(r.Context(), siteID, page)
	if err != nil {
		writeError(w, correlationID, http.StatusInternalServerError, "internal_error", "erro ao listar relatórios")
		return
	}

	items := make([]map[string]any, 0, len(reports))
	for _, rep := range reports {
		items = append(items, reportToJSON(rep, false))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items":     items,
		"page":      page.Page,
		"page_size": page.PageSize,
		"total":     total,
	})
}

func (s *Server) handleGetReport(w http.ResponseWriter, r *http.Request) {
	correlationID := correlationIDFromContext(r.Context())

	id, err := uuid.Parse(r.PathValue("reportId"))
	if err != nil {
		writeError(w, correlationID, http.StatusBadRequest, "invalid_report_id", "reportId inválido")
		return
	}

	report, err := s.reports.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, correlationID, http.StatusNotFound, "not_found", "relatório não encontrado")
			return
		}
		writeError(w, correlationID, http.StatusInternalServerError, "internal_error", "erro ao buscar relatório")
		return
	}

	writeJSON(w, http.StatusOK, reportToJSON(report, true))
}

func reportToJSON(rep store.Report, includeContent bool) map[string]any {
	out := map[string]any{
		"id":           rep.ID.String(),
		"site_id":      rep.SiteID.String(),
		"kind":         rep.Kind,
		"period_start": rep.PeriodStart.Format(time.RFC3339),
		"period_end":   rep.PeriodEnd.Format(time.RFC3339),
		"generated_at": rep.GeneratedAt.Format(time.RFC3339),
	}
	if rep.GeneratedBy != nil {
		out["generated_by"] = rep.GeneratedBy.String()
	}
	if includeContent {
		out["content"] = rep.Content
	}
	return out
}
