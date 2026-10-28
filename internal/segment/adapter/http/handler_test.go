package http_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/selseleha/go-challenge/internal/segment"
	segmenthttp "github.com/selseleha/go-challenge/internal/segment/adapter/http"
)

type stubEstimator struct {
	count   int64
	err     error
	segment string
}

func (s *stubEstimator) Estimate(_ context.Context, segment string) (int64, error) {
	s.segment = segment
	return s.count, s.err
}

type stubHealth struct {
	info segment.HealthInfo
}

func (s *stubHealth) Health(context.Context) segment.HealthInfo { return s.info }

type stubReset struct {
	err error
}

func (s *stubReset) Reset(context.Context) error { return s.err }

func newHandler(est segment.Estimator, health segment.HealthChecker, reset segment.Resetter) *segmenthttp.Handler {
	if health == nil {
		health = &stubHealth{info: segment.HealthInfo{Status: segment.StatusReady, Store: "memory"}}
	}
	if reset == nil {
		reset = &stubReset{}
	}
	return segmenthttp.NewHandler(segmenthttp.Deps{
		Estimator:  est,
		Health:     health,
		Resetter:   reset,
		AdminToken: "secret",
	})
}

func TestHandler_Estimate_ReturnsCountJSON(t *testing.T) {
	est := &stubEstimator{count: 42}
	h := newHandler(est, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/segments/sports/estimate", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Segment string `json:"segment"`
		Count   int64  `json:"count"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&body)
	if body.Segment != "sports" || body.Count != 42 {
		t.Fatalf("body = %+v", body)
	}
}

func TestHandler_Estimate_RejectsEmptySegment(t *testing.T) {
	h := newHandler(&stubEstimator{count: 1}, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/v1/segments/%20%20/estimate", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestHandler_Estimate_NotReadyReturns503(t *testing.T) {
	h := newHandler(&stubEstimator{err: segment.ErrNotReady}, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/v1/segments/sports/estimate", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d; body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Error  string `json:"error"`
		Status string `json:"status"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&body)
	if body.Status != segment.StatusResetting || body.Error == "" {
		t.Fatalf("body = %+v", body)
	}
}

func TestHandler_Estimate_MapsServiceErrorTo500(t *testing.T) {
	h := newHandler(&stubEstimator{err: errors.New("boom")}, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/v1/segments/sports/estimate", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestHandler_Health_Ready(t *testing.T) {
	h := newHandler(&stubEstimator{}, &stubHealth{info: segment.HealthInfo{
		Status: segment.StatusReady,
		Store:  "postgres_redis",
		Hot: segment.StoreStats{
			Kind:          "postgres_redis",
			OK:            true,
			Members:       10,
			SourceMembers: 12,
			HasSource:     true,
			SegmentNames:  []string{"sports", "music"},
		},
	}}, nil)
	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"source_members":12`) || !strings.Contains(body, `"sports"`) {
		t.Fatalf("health json = %s", body)
	}
	if !strings.Contains(body, `"product"`) {
		t.Fatalf("expected product in health json: %s", body)
	}
}

func TestHandler_Health_HTMLForBrowsers(t *testing.T) {
	h := newHandler(&stubEstimator{}, &stubHealth{info: segment.HealthInfo{
		Status: segment.StatusReady,
		Store:  "redis",
		Hot:    segment.StoreStats{Kind: "redis", OK: true, Members: 3, Segments: 1},
	}}, nil)
	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if ct != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Estimation Service") || !strings.Contains(body, "Segment estimates") {
		t.Fatalf("unexpected html: %s", body)
	}
	if !strings.Contains(body, "Postgres") || !strings.Contains(body, "/v1/segments/") {
		t.Fatalf("monitor markup missing charts/poll: %s", body[:min(300, len(body))])
	}
}

func TestHandler_Health_ResettingReturns503(t *testing.T) {
	h := newHandler(&stubEstimator{}, &stubHealth{info: segment.HealthInfo{Status: segment.StatusResetting, Store: "postgres_redis"}}, nil)
	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d; body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandler_Reset_RequiresToken(t *testing.T) {
	h := newHandler(&stubEstimator{}, nil, &stubReset{})
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/reset", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestHandler_Reset_AcceptedWhenStillResetting(t *testing.T) {
	h := newHandler(
		&stubEstimator{},
		&stubHealth{info: segment.HealthInfo{Status: segment.StatusResetting, Store: "postgres_redis"}},
		&stubReset{},
	)
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/reset", nil)
	req.Header.Set("X-Admin-Token", "secret")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d; body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandler_Reset_ConflictWhenInProgress(t *testing.T) {
	h := newHandler(&stubEstimator{}, nil, &stubReset{err: segment.ErrResetInProgress})
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/reset", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d", rec.Code)
	}
}
