// Package memory provides an in-memory segment.Repository for tests and local runs.
//
// Storage layout: map[segment]map[userID]taggedAt
//   - ingest (Upsert): O(1) per event
//   - estimate (CountActive): O(users in that segment), not O(all users)
//
// This matches the challenge scale profile (millions of users, hundreds of segments):
// work is proportional to the queried segment, not the whole population.
package memory
