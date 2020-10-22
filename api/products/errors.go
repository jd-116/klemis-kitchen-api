package products

import "fmt"

// CacheNotInitializedError is an error used to encode when the cache has not been initialized
type CacheNotInitializedError struct {
	Action string
}

// NewCacheNotInitializedError constructs a new CacheNotInitializedError
func NewCacheNotInitializedError(action string) *CacheNotInitializedError {
	return &CacheNotInitializedError{
		Action: action,
	}
}

func (e *CacheNotInitializedError) Error() string {
	return fmt.Sprintf("cannot %s: cache has not been initialized", e.Action)
}

// LocationNotFoundError is an error used to encode when a location isn't found
type LocationNotFoundError struct {
	Identifier string
}

// NewLocationNotFoundError constructs a new LocationNotFoundError
func NewLocationNotFoundError(identifier string) *LocationNotFoundError {
	return &LocationNotFoundError{
		Identifier: identifier,
	}
}

func (e *LocationNotFoundError) Error() string {
	return fmt.Sprintf("location with identifier '%s' not found in the Transact API cache",
		e.Identifier)
}

// PartialProductNotFoundError is an error used to encode when a partial product isn't found
type PartialProductNotFoundError struct {
	Location string
	ID       string
}

// NewPartialProductNotFoundError constructs a new PartialProductNotFoundError
func NewPartialProductNotFoundError(location string, id string) *PartialProductNotFoundError {
	return &PartialProductNotFoundError{
		ID:       id,
		Location: location,
	}
}

func (e *PartialProductNotFoundError) Error() string {
	return fmt.Sprintf("product with identifier '%s' at location '%s' not found in the Transact API cache",
		e.ID, e.Location)
}
