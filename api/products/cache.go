package products

import (
	"sync"
)

// Cache represents a cache of Partial Products
// that implements the PartialProductProvider interface
type Cache struct {
	sync.Mutex
	loaded          bool
	locations       []string
	partialProducts map[string]map[string]PartialProduct
}

// Load loads a cache from the source products map,
// marking it as ready.
//
// Note: uses passed in map as the inner map;
// the passed in map cannot be reused by the caller afterwards
func (c *Cache) Load(partialProducts map[string]map[string]PartialProduct) {
	c.Lock()
	defer c.Unlock()

	// Mark as loaded and load the map
	c.loaded = true
	c.partialProducts = partialProducts

	// Build the location identifiers slice
	locations := make([]string, 0)
	for location := range partialProducts {
		locations = append(locations, location)
	}
	c.locations = locations
}

// GetAllLocations gets all location identifiers
func (c *Cache) GetAllLocations() ([]string, error) {
	c.Lock()
	defer c.Unlock()

	if !c.loaded {
		return nil, NewCacheNotInitializedError("list locations from the Transact API")
	}

	return c.locations, nil
}

// GetAllProducts gets all products for the given location identifier
func (c *Cache) GetAllProducts(location string) ([]PartialProduct, error) {
	c.Lock()
	defer c.Unlock()

	if !c.loaded {
		return nil, NewCacheNotInitializedError("list products at location from the Transact API")
	}

	if locationProducts, ok := c.partialProducts[location]; ok {
		// Construct a slice of all products at that location
		partialProducts := []PartialProduct{}
		for _, value := range locationProducts {
			partialProducts = append(partialProducts, value)
		}

		return partialProducts, nil
	}

	return nil, NewLocationNotFoundError(location)
}

// GetProduct gets a single partial product from the given location with the given ID
func (c *Cache) GetProduct(location string, id string) (*PartialProduct, error) {
	c.Lock()
	defer c.Unlock()

	if !c.loaded {
		return nil, NewCacheNotInitializedError("get product at location from the Transact API")
	}

	if locationProducts, ok := c.partialProducts[location]; ok {
		// Attempt to find the given partial product in this location
		if partialProduct, ok := locationProducts[id]; ok {
			return &partialProduct, nil
		}

		return nil, NewPartialProductNotFoundError(location, id)
	}

	return nil, NewLocationNotFoundError(location)
}
