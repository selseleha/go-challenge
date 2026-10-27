package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/selseleha/go-challenge/internal/segment"
)

// Handler is the inbound HTTP adapter for segment estimation.
type Handler struct {
	estimator segment.Estimator
	mux       *http.ServeMux
}

// NewHandler wires an Estimator into the REST estimate routes.
func NewHandler(estimator segment.Estimator) *Handler {
	h := &Handler{estimator: estimator, mux: http.NewServeMux()}
	h.mux.HandleFunc("GET /v1/segments/{segment}/estimate", h.estimate)
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

type estimateResponse struct {
	Segment string `json:"segment"`
	Count   int64  `json:"count"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (h *Handler) estimate(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.PathValue("segment"))
	if name == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "segment is required"})
		return
	}

	count, err := h.estimator.Estimate(r.Context(), name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, estimateResponse{Segment: name, Count: count})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
