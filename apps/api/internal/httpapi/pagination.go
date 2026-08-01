package httpapi

import (
	"net/http"
	"strconv"

	"egger/api/internal/store"
)

func parsePage(r *http.Request) store.Page {
	page := 1
	if v, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && v > 0 {
		page = v
	}
	pageSize := 20
	if v, err := strconv.Atoi(r.URL.Query().Get("page_size")); err == nil && v > 0 && v <= 100 {
		pageSize = v
	}
	return store.Page{Page: page, PageSize: pageSize}
}
