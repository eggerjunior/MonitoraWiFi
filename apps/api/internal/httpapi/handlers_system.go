package httpapi

import (
	"net/http"

	"egger/api/internal/buildinfo"
)

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"version": buildinfo.Version,
		"commit":  buildinfo.GitCommit,
	})
}

func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	deps := s.checkReadiness(r.Context())

	status := "ready"
	code := http.StatusOK
	for _, v := range deps {
		if v != "ok" {
			status = "not_ready"
			code = http.StatusServiceUnavailable
			break
		}
	}

	writeJSON(w, code, map[string]any{
		"status":       status,
		"dependencies": deps,
	})
}
