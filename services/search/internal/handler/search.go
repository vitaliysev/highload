package handler

import (
	"net/http"
	"strconv"

	"marketplace/search/internal/domain"
	"marketplace/search/internal/service"
)

type SearchHandler struct {
	svc *service.SearchService
}

func New(svc *service.SearchService) *SearchHandler {
	return &SearchHandler{svc: svc}
}

func (h *SearchHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/listings/search", h.search)
	mux.HandleFunc("GET /health", h.health)
}

// параметры: q, category, price_min, price_max, location, limit, offset
func (h *SearchHandler) search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	query := domain.SearchQuery{
		Query:    q.Get("q"),
		Category: q.Get("category"),
		Location: q.Get("location"),
		Limit:    20,
	}

	if v := q.Get("price_min"); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid price_min")
			return
		}
		query.PriceMin = &f
	}
	if v := q.Get("price_max"); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid price_max")
			return
		}
		query.PriceMax = &f
	}
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 || n > 100 {
			writeError(w, http.StatusBadRequest, "invalid limit (1-100)")
			return
		}
		query.Limit = n
	}
	if v := q.Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			writeError(w, http.StatusBadRequest, "invalid offset")
			return
		}
		query.Offset = n
	}

	result, err := h.svc.Search(r.Context(), query)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "search failed")
		return
	}

	// Circuit Breaker
	if result.Degraded {
		w.Header().Set("X-Search-Degraded", "true")
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *SearchHandler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
