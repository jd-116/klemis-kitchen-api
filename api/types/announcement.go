package types

// Announcement is ...
type Announcement struct {
	ID        string `json:"id" bson:"id"`
	Name      string `json:"name" bson:"name"`
	Timestamp string `json:"timestamp" bson:"timestamp"`
	Important bool   `json:"important" bson:"important"`
}

// ErrorResponse is ...
type ErrorResponse struct {
	Message string `json:"message"`
}
