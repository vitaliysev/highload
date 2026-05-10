package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"marketplace/listing/internal/domain"
	"marketplace/listing/internal/service"
)

type ListingHandler struct {
	svc *service.ListingService
}

func New(svc *service.ListingService) *ListingHandler {
	return &ListingHandler{svc: svc}
}

func (h *ListingHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/listings", h.create)
	mux.HandleFunc("GET /api/v1/listings/{id}", h.getCard)
	mux.HandleFunc("POST /api/v1/listings/{id}/photos/upload-url", h.uploadURL)
	mux.HandleFunc("POST /api/v1/listings/{id}/promote", h.promote)
	mux.HandleFunc("GET /api/v1/users/{id}/listings", h.getUserListings)
	mux.HandleFunc("GET /health", h.health)
}

type createRequest struct {
	UserID      string  `json:"user_id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Category    string  `json:"category"`
	Location    string  `json:"location"`
}

func (h *ListingHandler) create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	listing := &domain.Listing{
		UserID:      userID,
		Title:       req.Title,
		Description: req.Description,
		Price:       req.Price,
		Category:    req.Category,
		Location:    req.Location,
	}

	created, err := h.svc.Create(r.Context(), listing)
	if err != nil {
		if errors.Is(err, domain.ErrBadRequest) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, created)
}

func (h *ListingHandler) getCard(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	card, err := h.svc.GetCard(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, card)
}

type uploadURLRequest struct {
	UserID      string `json:"user_id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
}

func (h *ListingHandler) uploadURL(w http.ResponseWriter, r *http.Request) {
	listingID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req uploadURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	allowed := map[string]bool{"image/jpeg": true, "image/png": true, "image/webp": true}
	if !allowed[req.ContentType] {
		writeError(w, http.StatusBadRequest, "unsupported content_type")
		return
	}
	if req.SizeBytes > 5*1024*1024 {
		writeError(w, http.StatusBadRequest, "file too large, max 5 MB")
		return
	}

	photo, uploadURL, err := h.svc.GetUploadURL(r.Context(), userID, listingID, req.Filename, req.ContentType, req.SizeBytes)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			writeError(w, http.StatusNotFound, "listing not found")
		case errors.Is(err, domain.ErrForbidden):
			writeError(w, http.StatusForbidden, "listing does not belong to user")
		case errors.Is(err, domain.ErrPhotoLimitReached):
			writeError(w, http.StatusConflict, "photo limit reached (max 10)")
		default:
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"upload_url": uploadURL,
		"photo_id":   photo.ID,
		"expires_at": photo.CreatedAt.Add(15 * 60 * 1e9),
	})
}

type promoteRequest struct {
	UserID        string `json:"user_id"`
	Plan          string `json:"plan"`
	PaymentMethod string `json:"payment_method"`
}

func (h *ListingHandler) promote(w http.ResponseWriter, r *http.Request) {
	listingID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req promoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	payment, promo, err := h.svc.Promote(r.Context(), userID, listingID, req.Plan, req.PaymentMethod)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			writeError(w, http.StatusNotFound, "listing not found")
		case errors.Is(err, domain.ErrForbidden):
			writeError(w, http.StatusForbidden, "listing does not belong to user")
		case errors.Is(err, domain.ErrConflict):
			writeError(w, http.StatusConflict, "listing already has an active promotion")
		case errors.Is(err, domain.ErrBadRequest):
			writeError(w, http.StatusBadRequest, "unknown plan or listing not published")
		default:
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"payment_id": payment.ID,
		"amount":     payment.Amount,
		"currency":   payment.Currency,
		"plan":       promo.Plan,
		"expires_at": promo.ExpiresAt,
	})
}

func (h *ListingHandler) getUserListings(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	f := domain.UserListingsFilter{
		Page:    1,
		PerPage: 20,
	}

	if s := r.URL.Query().Get("status"); s != "" {
		st := domain.ListingStatus(s)
		f.Status = &st
	}
	if p := r.URL.Query().Get("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			f.Page = n
		}
	}
	if pp := r.URL.Query().Get("per_page"); pp != "" {
		if n, err := strconv.Atoi(pp); err == nil && n > 0 && n <= 50 {
			f.PerPage = n
		}
	}

	listings, total, err := h.svc.GetUserListings(r.Context(), userID, f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"total":    total,
		"page":     f.Page,
		"per_page": f.PerPage,
		"items":    listings,
	})
}

func (h *ListingHandler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
