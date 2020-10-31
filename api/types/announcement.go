package types

import "time"

// Announcement is the document stored in MongoDB for a single announcement
type Announcement struct {
	ID        string    `json:"id" bson:"id"`
	Title     string    `json:"title" bson:"title"`
	Body      string    `json:"body" bson:"body"`
	Timestamp time.Time `json:"timestamp" bson:"timestamp"`
}

// AnnouncementCreate is supplied through the dashboard and converted into
// an Announcement
type AnnouncementCreate struct {
	Title     string    `json:"title" bson:"title"`
	Body      string    `json:"body" bson:"body"`
	Timestamp time.Time `json:"timestamp" bson:"timestamp"`
}
