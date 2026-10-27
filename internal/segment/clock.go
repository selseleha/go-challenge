package segment

import "time"

// Clock abstracts time so expiry logic is testable without sleeping.
type Clock interface {
	Now() time.Time
}

// SystemClock uses the real wall clock in UTC.
type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now().UTC() }

// FixedClock returns a constant time; useful in tests.
type FixedClock struct{ T time.Time }

func (c FixedClock) Now() time.Time { return c.T }
