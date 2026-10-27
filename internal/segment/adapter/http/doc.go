// Package http is the inbound REST adapter for estimating segment membership.
//
// Route: GET /v1/segments/{segment}/estimate
// Response: {"segment":"<name>","count":<n>}
// Errors: JSON {"error":"<message>"} with 400 for validation and 500 for service failures.
//
// Handlers stay thin: validate input, call segment.Estimator, map results to HTTP.
package http
