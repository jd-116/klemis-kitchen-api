package types

// Location is the internal representation of a location object,
// taken directly from MongoDB.
// Note: this struct should not be returned from the API directly;
// instead, use the LocationMetadata struct
type Location struct {
	ID                 string         `json:"id" bson:"id"`
	Name               string         `json:"name" bson:"name"`
	Location           GeoCoordinates `json:"location" bson:"location"`
	TransactIdentifier string         `json:"transact_identifier" bson:"transact_identifier"`
}

// Inner gets the inner representation for this location
func (l *Location) Inner() LocationMetadata {
	return LocationMetadata{
		ID:       l.ID,
		Name:     l.Name,
		Location: l.Location,
	}
}

// LocationMetadata is the external representation of a location
type LocationMetadata struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Location GeoCoordinates `json:"location"`
}

// GeoCoordinates is the representation of a pair of GPS coordinates
// that contains latitude and longitude
type GeoCoordinates struct {
	Latitude  float64 `json:"latitude" bson:"latitude"`
	Longitude float64 `json:"longitude" bson:"longitude"`
}

// LocationCreate is the partial Location struct that is sent in POST requests
type LocationCreate struct {
	Name               string         `json:"name" bson:"name"`
	Location           GeoCoordinates `json:"location" bson:"location"`
	TransactIdentifier string         `json:"transact_identifier" bson:"transact_identifier"`
}
