package types

import "time"

// Location is the internal representation of a location object,
// taken directly from MongoDB.
// Note: this struct should not be returned from the API directly;
// instead, use the LocationMetadata struct
type Location struct {
	TransactIdentifier string `json:"transact_identifier"`
	LocationMetadata
}

// LocationMetadata is the external representation of a location
type LocationMetadata struct {
	ID                 string         `json:"id"`
	Name               string         `json:"name"`
	LastDelivery       time.Time      `json:"last_delivery"`
	Location           GeoCoordinates `json:"location"`
}

// GeoCoordinates is the representation of a pair of GPS coordinates
// that contains latitude and longitude
type GeoCoordinates struct {
	Latitude  int `json:"latitutde"`
	Longitude int `json:"longitude"`
}
