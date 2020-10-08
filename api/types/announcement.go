package types

// Announcement is ...
type Announcement struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Timestamp string `json:"timestamp"`
	Important bool   `json:"important"`
}

// ErrorResponse is ...
type ErrorResponse struct {
	Message string `json:"message"`
}
