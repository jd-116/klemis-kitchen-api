package products

import (
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"

	"github.com/go-chi/chi"
	"github.com/lithammer/fuzzysearch/fuzzy"

	"github.com/jd-116/klemis-kitchen-api/auth"
	"github.com/jd-116/klemis-kitchen-api/db"
	"github.com/jd-116/klemis-kitchen-api/products"
	"github.com/jd-116/klemis-kitchen-api/types"
	"github.com/jd-116/klemis-kitchen-api/util"
)

// Routes creates a new Chi router with all of the routes for the product resource,
// at the root level
func Routes(database db.Provider, products products.Provider) *chi.Mux {
	router := chi.NewRouter()
	router.Get("/", GetAll(database, database, products))
	router.Get("/{id}", GetSingle(database, database, products))

	// Admin-only routes
	router.Group(func(r chi.Router) {
		// Ensure the user has access
		r.Use(auth.AdminAuthenticated)
		r.Patch("/{id}", Update(database))
	})
	return router
}

// GetAll gets all products from the database,
// with an optional search querystring param
func GetAll(productMetadataProvider db.ProductMetadataProvider, locationProvider db.LocationProvider,
	cacheProducts products.Provider) http.HandlerFunc {

	// Use a closure to inject the database provider
	return func(w http.ResponseWriter, r *http.Request) {
		// See if we have search parameter,
		// which can be empty
		search := strings.ToLower(r.URL.Query().Get("search"))

		dbLocations, err := locationProvider.GetAllLocations(r.Context())
		if err != nil {
			util.Error(w, err)
			return
		}

		dbProducts, err := productMetadataProvider.GetAllProducts(r.Context())
		if err != nil {
			util.Error(w, err)
			return
		}

		cacheLocations, err := cacheProducts.GetAllLocations()
		if err != nil {
			util.Error(w, err)
			return
		}

		// Create database location identifier set so that all
		// locations from Transact have corresponding concrete locations
		locationIdentifierSet := make(map[string]struct{})
		for _, dbLocation := range dbLocations {
			locationIdentifierSet[dbLocation.TransactIdentifier] = struct{}{}
		}

		// Create id -> ProductDataSearch map
		// Since some locations might have PartialProducts with duplicate IDs
		productMap := make(map[string]types.ProductDataSearch)
		for _, cacheLocation := range cacheLocations {
			// Make sure this is a concrete location
			if _, ok := locationIdentifierSet[cacheLocation]; !ok {
				continue
			}

			partialProducts, err := cacheProducts.GetAllProducts(cacheLocation)
			if err != nil {
				util.Error(w, err)
				return
			}

			for _, partialProduct := range partialProducts {
				// Only create a new ProductDataSearch if this ID isn't already in the map
				if _, ok := productMap[partialProduct.ID]; ok {
					continue
				}

				// Make sure the name passes a search if it was given
				if search != "" && !fuzzy.MatchNormalized(search, strings.ToLower(partialProduct.Name)) {
					continue
				}

				productMap[partialProduct.ID] = types.ProductDataSearch{
					Name:      partialProduct.Name,
					ID:        partialProduct.ID,
					Thumbnail: nil,
					Nutrition: nil,
				}
			}
		}

		// Fold in any thumbnail/nutrition metadata for each DB Product
		for _, dbProduct := range dbProducts {
			if product, ok := productMap[dbProduct.ID]; ok {
				// Update the ProductDataSearch struct with the metadata
				product.Thumbnail = dbProduct.Thumbnail
				product.Nutrition = dbProduct.Nutrition
				productMap[dbProduct.ID] = product
			}
		}

		// Collect the product map into a slice,
		// in the order of ascending IDs by first extracting all IDs
		ids := []string{}
		for id := range productMap {
			ids = append(ids, id)
		}
		sort.Strings(ids)

		// Finally, actually collect
		resultProducts := []types.ProductDataSearch{}
		for _, id := range ids {
			if product, ok := productMap[id]; ok {
				resultProducts = append(resultProducts, product)
			}
		}

		// Return the list in a JSON object
		jsonResponse, err := json.Marshal(map[string]interface{}{
			"products": resultProducts,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResponse)
	}
}

type productsData struct {
	partialProduct products.PartialProduct
	amounts        map[string]int
}

// GetSingle gets a single product from the database by its ID
func GetSingle(productMetadataProvider db.ProductMetadataProvider, locationProvider db.LocationProvider,
	cacheProducts products.Provider) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" {
			util.ErrorWithCode(w, errors.New("the URL parameter is empty"),
				http.StatusBadRequest)
			return
		}

		productMetadata, err := productMetadataProvider.GetProduct(r.Context(), id)
		if err != nil {
			// Continue with partial product
			productMetadata = nil
		}

		cacheLocations, err := cacheProducts.GetAllLocations()
		if err != nil {
			util.Error(w, err)
			return
		}

		dbLocations, err := locationProvider.GetAllLocations(r.Context())
		if err != nil {
			util.Error(w, err)
			return
		}

		// Create identifier -> DB Location map
		dbLocationMap := make(map[string]types.Location)
		for _, dbLocation := range dbLocations {
			dbLocationMap[dbLocation.TransactIdentifier] = dbLocation
		}

		var finalProduct productsData
		finalProduct.partialProduct.ID = id
		finalProduct.amounts = make(map[string]int)

		for _, cacheLocation := range cacheLocations {
			// Make sure this is a concrete location
			if dbLocation, ok := dbLocationMap[cacheLocation]; ok {
				singleProduct, err := cacheProducts.GetProduct(cacheLocation, id)
				if err != nil {
					util.Error(w, err)
					return
				}

				finalProduct.amounts[dbLocation.ID] = singleProduct.Amount

				// Store the name if not set
				if finalProduct.partialProduct.Name == "" {
					finalProduct.partialProduct.Name = singleProduct.Name
				}
			}
		}

		var resultProduct types.ProductData
		resultProduct.ID = finalProduct.partialProduct.ID
		resultProduct.Name = finalProduct.partialProduct.Name
		resultProduct.Amounts = finalProduct.amounts

		// Attach product metadata if found
		if productMetadata != nil {
			resultProduct.Nutrition = productMetadata.Nutrition
			resultProduct.Thumbnail = productMetadata.Thumbnail
		}

		// Return the single product as the top-level JSON
		jsonResponse, err := json.Marshal(resultProduct)
		if err != nil {
			util.ErrorWithCode(w, err, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResponse)
	}
}

// Update updates a products metadata in the database
func Update(productMetadataProvider db.ProductMetadataProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" {
			util.ErrorWithCode(w, errors.New("the URL parameter is empty"),
				http.StatusBadRequest)
			return
		}

		productMetadata, err := productMetadataProvider.GetProduct(r.Context(), id)
		if err != nil {
			// Continue with partial product
			productMetadata = nil
		}

		if productMetadata != nil {
			partial := make(map[string]interface{})
			err := json.NewDecoder(r.Body).Decode(&partial)
			if err != nil {
				util.Error(w, err)
				return
			}

			updated, err := productMetadataProvider.UpdateProduct(r.Context(), id, partial)
			if err != nil {
				util.Error(w, err)
				return
			}

			// Return the updated product metadata as the top-level JSON
			jsonResponse, err := json.Marshal(updated)
			if err != nil {
				util.ErrorWithCode(w, err, http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(jsonResponse)

		} else {

			var productMetadata types.ProductMetadata
			err := json.NewDecoder(r.Body).Decode(&productMetadata)
			if err != nil {
				util.Error(w, err)
				return
			}

			productMetadata.ID = strings.TrimSpace(id)
			if productMetadata.ID == "" {
				util.ErrorWithCode(w, errors.New("productMetadata ID cannot be empty"),
					http.StatusBadRequest)
				return
			}

			err = productMetadataProvider.CreateProduct(r.Context(), productMetadata)
			if err != nil {
				util.Error(w, err)
				return
			}

			// Return the updated product metadata as the top-level JSON
			jsonResponse, err := json.Marshal(productMetadata)
			if err != nil {
				util.ErrorWithCode(w, err, http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(jsonResponse)
		}
	}
}
