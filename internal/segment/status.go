package segment

import (
	"errors"
	"sync"
)

// System status values reported by health and used to gate estimates.
const (
	StatusReady     = "ready"
	StatusResetting = "resetting"
)

// ErrNotReady is returned when estimate/ingest is blocked during reset.
var ErrNotReady = errors.New("system is resetting; hot store is being rebuilt")

// ErrResetInProgress is returned when a reset is already running.
var ErrResetInProgress = errors.New("reset already in progress")

// Readiness tracks whether the hot store is safe to query.
type Readiness struct {
	mu     sync.RWMutex
	status string
}

// NewReadiness starts in ready state (normal boot has no blocking warm).
func NewReadiness() *Readiness {
	return &Readiness{status: StatusReady}
}

func (r *Readiness) Status() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.status
}

func (r *Readiness) IsReady() bool {
	return r.Status() == StatusReady
}

// BeginReset moves ready → resetting. Fails if already resetting.
func (r *Readiness) BeginReset() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.status == StatusResetting {
		return ErrResetInProgress
	}
	r.status = StatusResetting
	return nil
}

func (r *Readiness) MarkReady() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.status = StatusReady
}
