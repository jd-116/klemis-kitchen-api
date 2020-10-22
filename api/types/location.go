package types

import "time"

// Location is the internal representation of a location object,
// taken directly from MongoDB.
// Note: this struct should not be returned from the API directly;
// instead, use the LocationMetadata struct
type Location struct {
	TransactIdentifier string         `json:"transact_identifier" bson:"transact_identifier"`
	ID                 string         `json:"id" bson:"id"`
	Name               string         `json:"name" bson:"name"`
	LastDelivery       *time.Time     `json:"last_delivery" bson:"last_delivery"`
	Location           GeoCoordinates `json:"location" bson:"location"`
}

// Inner gets the inner representation for this location
func (l *Location) Inner() LocationMetadata {
	return LocationMetadata{
		ID:           l.ID,
		Name:         l.Name,
		LastDelivery: l.LastDelivery,
		Location:     l.Location,
	}
}

// LocationMetadata is the external representation of a location
type LocationMetadata struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	LastDelivery *time.Time     `json:"last_delivery"`
	Location     GeoCoordinates `json:"location"`
}

// GeoCoordinates is the representation of a pair of GPS coordinates
// that contains latitude and longitude
type GeoCoordinates struct {
	Latitude  float64 `json:"latitude" bson:"latitude"`
	Longitude float64 `json:"longitude" bson:"longitude"`
}
