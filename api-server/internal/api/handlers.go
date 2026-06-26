package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"api-server/internal/model"
	"api-server/internal/repository"
)

const (
	defaultLimit = 100
	maxLimit     = 500
)

type Handler struct {
	products *repository.ProductRepository
}

func NewHandler(products *repository.ProductRepository) *Handler {
	return &Handler{products: products}
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, model.HealthResponse{Status: "ok"})
}

func (h *Handler) ListStock(w http.ResponseWriter, r *http.Request) {
	products, err := h.products.ListStock(r.Context())
	if err != nil {
		log.Printf("list stock: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to load product stock")
		return
	}
	if products == nil {
		products = []model.ProductStock{}
	}
	writeJSON(w, http.StatusOK, products)
}

func (h *Handler) ListMovements(w http.ResponseWriter, r *http.Request) {
	sku := chi.URLParam(r, "sku")
	if sku == "" {
		writeError(w, http.StatusBadRequest, "sku is required")
		return
	}

	limit, offset, ok := parsePagination(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid pagination values")
		return
	}

	exists, err := h.products.ProductExists(r.Context(), sku)
	if err != nil {
		log.Printf("check product %q exists: %v", sku, err)
		writeError(w, http.StatusInternalServerError, "failed to load product")
		return
	}
	if !exists {
		writeError(w, http.StatusNotFound, "product not found")
		return
	}

	movements, err := h.products.ListMovements(r.Context(), sku, limit, offset)
	if err != nil {
		log.Printf("list movements for %q: %v", sku, err)
		writeError(w, http.StatusInternalServerError, "failed to load movements")
		return
	}
	if movements == nil {
		movements = []model.Movement{}
	}
	writeJSON(w, http.StatusOK, movements)
}

func parsePagination(r *http.Request) (limit int, offset int, ok bool) {
	limit = defaultLimit
	offset = 0

	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil || parsedLimit <= 0 || parsedLimit > maxLimit {
			return 0, 0, false
		}
		limit = parsedLimit
	}

	if rawOffset := r.URL.Query().Get("offset"); rawOffset != "" {
		parsedOffset, err := strconv.Atoi(rawOffset)
		if err != nil || parsedOffset < 0 {
			return 0, 0, false
		}
		offset = parsedOffset
	}

	return limit, offset, true
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("encode json response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, model.ErrorResponse{Error: message})
}
