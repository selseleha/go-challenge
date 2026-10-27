package segment

import "time"

// MembershipTTL is how long a user remains counted in a segment after being tagged.
const MembershipTTL = 14 * 24 * time.Hour

// Membership is a user tagged into a segment at a point in time.
type Membership struct {
	UserID   string
	Segment  string
	TaggedAt time.Time
}

// Active reports whether the membership still counts at now.
// At exactly MembershipTTL after TaggedAt the membership is expired.
func (m Membership) Active(now time.Time) bool {
	return now.Sub(m.TaggedAt) < MembershipTTL
}
