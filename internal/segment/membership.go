package segment

import "time"

// DefaultMembershipTTL is the challenge default (two weeks) when config is unset.
const DefaultMembershipTTL = 14 * 24 * time.Hour

// Membership is a user tagged into a segment at a point in time.
type Membership struct {
	UserID   string
	Segment  string
	TaggedAt time.Time
}

// Active reports whether the membership still counts at now for the given ttl.
// At exactly ttl after TaggedAt the membership is expired.
func (m Membership) Active(now time.Time, ttl time.Duration) bool {
	return now.Sub(m.TaggedAt) < ttl
}
