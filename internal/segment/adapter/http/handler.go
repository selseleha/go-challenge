package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/selseleha/go-challenge/internal/segment"
)

// Handler is the inbound HTTP adapter for estimate, health, and admin reset.
type Handler struct {
	estimator segment.Estimator
	health    segment.HealthChecker
	resetter  segment.Resetter
	adminTok  string
	mux       *http.ServeMux
}

// Deps groups inbound ports for the HTTP adapter.
type Deps struct {
	Estimator  segment.Estimator
	Health     segment.HealthChecker
	Resetter   segment.Resetter
	AdminToken string
}

// NewHandler wires REST routes.
func NewHandler(d Deps) *Handler {
	h := &Handler{
		estimator: d.Estimator,
		health:    d.Health,
		resetter:  d.Resetter,
		adminTok:  d.AdminToken,
		mux:       http.NewServeMux(),
	}
	h.mux.HandleFunc("GET /v1/segments/{segment}/estimate", h.estimate)
	h.mux.HandleFunc("GET /v1/health", h.healthCheck)
	h.mux.HandleFunc("POST /v1/admin/reset", h.reset)
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
	Error  string `json:"error"`
	Status string `json:"status,omitempty"`
}

type healthResponse struct {
	Status     string  `json:"status"`
	Store      string  `json:"store"`
	UptimeSec  float64 `json:"uptime_sec"`
	GoRoutines int     `json:"goroutines"`
	AllocMiB   float64 `json:"alloc_mib"`
	SysMiB     float64 `json:"sys_mib"`
	NumGC      uint32  `json:"num_gc"`
	ProbeMS    float64 `json:"probe_ms"`
	Product    struct {
		Store              string `json:"store"`
		MembershipTTL      string `json:"membership_ttl"`
		OutboxBatchSize    int    `json:"outbox_batch_size"`
		OutboxPollInterval string `json:"outbox_poll_interval"`
		WarmPageSize       int    `json:"warm_page_size"`
		CleanupRetention   string `json:"cleanup_retention"`
	} `json:"product"`
	Hot struct {
		Kind          string   `json:"kind"`
		OK            bool     `json:"ok"`
		LatencyMS     float64  `json:"latency_ms"`
		Segments      int64    `json:"segments"`
		Members       int64    `json:"members"`
		SourceMembers int64    `json:"source_members"`
		HasSource     bool     `json:"has_source"`
		SegmentNames  []string `json:"segment_names"`
		Detail        string   `json:"detail"`
	} `json:"hot_store"`
}

type resetResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func (h *Handler) estimate(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.PathValue("segment"))
	if name == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "segment is required"})
		return
	}

	count, err := h.estimator.Estimate(r.Context(), name)
	if err != nil {
		if errors.Is(err, segment.ErrNotReady) {
			writeJSON(w, http.StatusServiceUnavailable, errorResponse{
				Error:  err.Error(),
				Status: segment.StatusResetting,
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, estimateResponse{Segment: name, Count: count})
}

func (h *Handler) healthCheck(w http.ResponseWriter, r *http.Request) {
	info := h.health.Health(r.Context())
	code := http.StatusOK
	if info.Status != segment.StatusReady {
		code = http.StatusServiceUnavailable
	}
	if wantsHTML(r) {
		writeHealthHTML(w, code, info)
		return
	}
	writeJSON(w, code, toHealthResponse(info))
}

func toHealthResponse(info segment.HealthInfo) healthResponse {
	resp := healthResponse{
		Status:     info.Status,
		Store:      info.Store,
		UptimeSec:  info.Uptime.Seconds(),
		GoRoutines: info.GoRoutines,
		AllocMiB:   info.AllocMiB,
		SysMiB:     info.SysMiB,
		NumGC:      info.NumGC,
		ProbeMS:    info.ProbeMS,
	}
	resp.Product.Store = info.Product.Store
	resp.Product.MembershipTTL = info.Product.MembershipTTL
	resp.Product.OutboxBatchSize = info.Product.OutboxBatchSize
	resp.Product.OutboxPollInterval = info.Product.OutboxPollInterval
	resp.Product.WarmPageSize = info.Product.WarmPageSize
	resp.Product.CleanupRetention = info.Product.CleanupRetention
	resp.Hot.Kind = info.Hot.Kind
	resp.Hot.OK = info.Hot.OK
	resp.Hot.LatencyMS = info.Hot.LatencyMS
	resp.Hot.Segments = info.Hot.Segments
	resp.Hot.Members = info.Hot.Members
	resp.Hot.SourceMembers = info.Hot.SourceMembers
	resp.Hot.HasSource = info.Hot.HasSource
	resp.Hot.SegmentNames = info.Hot.SegmentNames
	if resp.Hot.SegmentNames == nil {
		resp.Hot.SegmentNames = []string{}
	}
	resp.Hot.Detail = info.Hot.Detail
	return resp
}

func (h *Handler) reset(w http.ResponseWriter, r *http.Request) {
	if !h.authorized(r) {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}

	err := h.resetter.Reset(r.Context())
	switch {
	case errors.Is(err, segment.ErrResetInProgress):
		writeJSON(w, http.StatusConflict, errorResponse{
			Error:  err.Error(),
			Status: segment.StatusResetting,
		})
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "reset failed"})
	default:
		info := h.health.Health(r.Context())
		msg := "hot store cleared"
		if info.Status == segment.StatusResetting {
			msg = "hot store cleared; reloading from postgres asynchronously"
		}
		code := http.StatusOK
		if info.Status == segment.StatusResetting {
			code = http.StatusAccepted
		}
		writeJSON(w, code, resetResponse{Status: info.Status, Message: msg})
	}
}

func (h *Handler) authorized(r *http.Request) bool {
	if h.adminTok == "" {
		return false
	}
	if t := r.Header.Get("X-Admin-Token"); t != "" && t == h.adminTok {
		return true
	}
	const prefix = "Bearer "
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, prefix) && strings.TrimPrefix(auth, prefix) == h.adminTok {
		return true
	}
	return false
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
