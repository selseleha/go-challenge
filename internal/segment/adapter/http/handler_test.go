package http_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

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

func TestHandler_Estimate_ReturnsCountJSON(t *testing.T) {
	est := &stubEstimator{count: 42}
	h := segmenthttp.NewHandler(est)

	req := httptest.NewRequest(http.MethodGet, "/v1/segments/sports/estimate", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}

	var body struct {
		Segment string `json:"segment"`
		Count   int64  `json:"count"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Segment != "sports" || body.Count != 42 {
		t.Fatalf("body = %+v, want segment=sports count=42", body)
	}
	if est.segment != "sports" {
		t.Fatalf("estimator got segment %q, want sports", est.segment)
	}
}

func TestHandler_Estimate_RejectsEmptySegment(t *testing.T) {
	est := &stubEstimator{count: 1}
	h := segmenthttp.NewHandler(est)

	req := httptest.NewRequest(http.MethodGet, "/v1/segments/%20%20/estimate", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	assertJSONError(t, rec, "segment is required")
	if est.segment != "" {
		t.Fatalf("estimator should not be called, got segment %q", est.segment)
	}
}

func TestHandler_Estimate_MapsServiceErrorTo500(t *testing.T) {
	est := &stubEstimator{err: errors.New("boom")}
	h := segmenthttp.NewHandler(est)

	req := httptest.NewRequest(http.MethodGet, "/v1/segments/sports/estimate", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
	assertJSONError(t, rec, "internal server error")
}

func TestHandler_Estimate_MethodNotAllowed(t *testing.T) {
	h := segmenthttp.NewHandler(&stubEstimator{})

	req := httptest.NewRequest(http.MethodPost, "/v1/segments/sports/estimate", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed && rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 405 or 404; body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandler_Estimate_UnknownRoute(t *testing.T) {
	h := segmenthttp.NewHandler(&stubEstimator{})

	req := httptest.NewRequest(http.MethodGet, "/v1/unknown", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func assertJSONError(t *testing.T, rec *httptest.ResponseRecorder, wantMsg string) {
	t.Helper()
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}
	var body struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	if body.Error != wantMsg {
		t.Fatalf("error = %q, want %q", body.Error, wantMsg)
	}
}
