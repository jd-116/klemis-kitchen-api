package types

import "time"

// ErrorResponse is the generic error JSON shape returned by the API
type ErrorResponse struct {
	Message string `json:"message"`
}

// Session is the JSON shape that is used to track authenticated sessions
type Session struct {
	Username     string    `json:"username"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	IssuedAt     time.Time `json:"issued_at"`
	ExpiresAfter *int64    `json:"expires_after"`
}
