package types

// ErrorResponse is the generic error JSON shape returned by the API
type ErrorResponse struct {
	Message string `json:"message"`
}
