package segment_test

import (
	"testing"

	"github.com/selseleha/go-challenge/internal/segment"
)

func TestReadiness_BeginReset_AndReady(t *testing.T) {
	r := segment.NewReadiness()
	if !r.IsReady() {
		t.Fatal("expected ready at start")
	}
	if err := r.BeginReset(); err != nil {
		t.Fatal(err)
	}
	if r.IsReady() || r.Status() != segment.StatusResetting {
		t.Fatalf("status = %s", r.Status())
	}
	if err := r.BeginReset(); err != segment.ErrResetInProgress {
		t.Fatalf("second reset err = %v", err)
	}
	r.MarkReady()
	if !r.IsReady() {
		t.Fatal("expected ready after MarkReady")
	}
}
