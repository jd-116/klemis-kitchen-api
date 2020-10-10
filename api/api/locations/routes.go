package locations

import (
	"encoding/json"
	"errors"
	"net/http"
	"sort"

	"github.com/go-chi/chi"
	"github.com/lithammer/fuzzysearch/fuzzy"

	"github.com/jd-116/klemis-kitchen-api/db"
	"github.com/jd-116/klemis-kitchen-api/products"
	"github.com/jd-116/klemis-kitchen-api/types"
	"github.com/jd-116/klemis-kitchen-api/util"
)

func Routes(database db.Provider, products products.Provider) *chi.Mux {
	router := chi.NewRouter()
	router.Get("/", GetAll(database))
	router.Get("/{id}", GetSingle(database))
	router.Get("/{id}/products", GetProducts(database, products))
	router.Get("/{id}/products/{product_id}", GetProduct(database, products))
	router.Post("/", Create(database))
	router.Delete("/{id}", Delete(database))
	router.Patch("/", Update(database))
	return router
}

// Gets all locations from the database
func GetAll(database db.Provider) http.HandlerFunc {
	// Use a closure to inject the database provider
	return func(w http.ResponseWriter, r *http.Request) {
		locations, err := database.GetAllLocations(r.Context())
		if err != nil {
			util.Error(w, err)
			return
		}

		// Extract the location metadata before returning it
		locationMetadata := make([]types.LocationMetadata, len(locations))
		for _, location := range locations {
			locationMetadata = append(locationMetadata, location.LocationMetadata)
		}

		// Return the list in a JSON object
		jsonResponse, err := json.Marshal(map[string]interface{}{
			"locations": locationMetadata,
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

// Gets a single location from the database by its ID
func GetSingle(database db.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" {
			util.ErrorWithCode(w, errors.New("the URL parameter is empty"),
				http.StatusBadRequest)
			return
		}

		location, err := database.GetLocation(r.Context(), id)
		if err != nil {
			util.Error(w, err)
			return
		}

		// Return the single location as the top-level JSON
		// (make sure to return the inner metadata instead of the full struct)
		jsonResponse, err := json.Marshal(location.LocationMetadata)
		if err != nil {
			util.ErrorWithCode(w, err, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResponse)
	}
}

// Gets all products that exist at this location,
// with an optional search querystring param
func GetProducts(database db.Provider, products products.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" {
			util.ErrorWithCode(w, errors.New("the URL parameter is empty"),
				http.StatusBadRequest)
			return
		}

		// See if we have search parameter,
		// which can be empty
		search := r.URL.Query().Get("search")

		dbLocation, err := database.GetLocation(r.Context(), id)
		if err != nil {
			util.Error(w, err)
			return
		}

		partialProducts, err := products.GetAllProducts(dbLocation.TransactIdentifier)
		if err != nil {
			util.Error(w, err)
		}

		dbProducts, err := database.GetAllProducts(r.Context())
		if err != nil {
			util.Error(w, err)
		}

		// Create id -> dbProduct map so we can index it quickly
		dbProductMap := make(map[string]types.ProductMetadata)
		for _, dbProduct := range dbProducts {
			dbProductMap[dbProduct.ID] = dbProduct
		}

		// Merge db products with partial products
		// to make `LocationProductDataSearch` structs
		locationProducts := make([]types.LocationProductDataSearch, len(partialProducts))
		for _, partialProduct := range partialProducts {
			// Make sure the name passes a search if it was given
			if search != "" && !fuzzy.MatchNormalized(search, partialProduct.Name) {
				continue
			}

			locationProduct := types.LocationProductDataSearch{
				Name:      partialProduct.Name,
				ID:        partialProduct.ID,
				Amount:    partialProduct.Amount,
				Thumbnail: nil,
			}

			// See if this has additional metadata, and attach if so
			if dbProduct, ok := dbProductMap[locationProduct.ID]; ok {
				locationProduct.Thumbnail = dbProduct.Thumbnail
			}

			locationProducts = append(locationProducts, locationProduct)
		}

		// Sort the location products map in the order of descending ID
		sort.Slice(locationProducts, func(i, j int) bool {
			return locationProducts[i].ID < locationProducts[j].ID
		})

		// Return the list in a JSON object
		jsonResponse, err := json.Marshal(map[string]interface{}{
			"products": locationProducts,
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

// Gets a single product at this location
func GetProduct(database db.Provider, products products.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		locationID := chi.URLParam(r, "id")
		if locationID == "" {
			util.ErrorWithCode(w, errors.New("the location URL parameter is empty"),
				http.StatusBadRequest)
			return
		}

		productID := chi.URLParam(r, "product_id")
		if productID == "" {
			util.ErrorWithCode(w, errors.New("the product URL parameter is empty"),
				http.StatusBadRequest)
			return
		}

		dbLocation, err := database.GetLocation(r.Context(), locationID)
		if err != nil {
			util.Error(w, err)
			return
		}

		partialProduct, err := products.GetProduct(dbLocation.TransactIdentifier, productID)
		if err != nil {
			util.Error(w, err)
		}

		// Construct a `LocationProductData` struct
		resultProduct := types.LocationProductData{
			ID:        partialProduct.ID,
			Name:      partialProduct.Name,
			Amount:    partialProduct.Amount,
			Nutrition: nil,
			Thumbnail: nil,
		}

		// See if this has a corresponding DB product object
		if dbProduct, err := database.GetProduct(r.Context(), productID); err == nil {
			resultProduct.Nutrition = dbProduct.Nutrition
			resultProduct.Thumbnail = dbProduct.Thumbnail
		}

		// Return the product as JSON
		jsonResponse, err := json.Marshal(resultProduct)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResponse)
	}
}

// Creates a new location in the database
func Create(database db.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var location types.Location
		err := json.NewDecoder(r.Body).Decode(&location)
		if err != nil {
			util.Error(w, err)
			return
		}

		err = database.CreateLocation(r.Context(), location)
		if err != nil {
			util.Error(w, err)
			return
		}

		// Return the single location as the top-level JSON
		jsonResponse, err := json.Marshal(location)
		if err != nil {
			util.ErrorWithCode(w, err, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write(jsonResponse)
	}
}

// Deletes a location in the database
func Delete(database db.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" {
			util.ErrorWithCode(w, errors.New("the URL parameter is empty"),
				http.StatusBadRequest)
			return
		}

		err := database.DeleteLocation(r.Context(), id)
		if err != nil {
			util.Error(w, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// Updates a location in the database
func Update(database db.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" {
			util.ErrorWithCode(w, errors.New("the URL parameter is empty"),
				http.StatusBadRequest)
			return
		}

		partial := make(map[string]interface{})
		err := json.NewDecoder(r.Body).Decode(&partial)
		if err != nil {
			util.Error(w, err)
			return
		}

		updated, err := database.UpdateLocation(r.Context(), id, partial)
		if err != nil {
			util.Error(w, err)
			return
		}

		// Return the updated location as the top-level JSON
		jsonResponse, err := json.Marshal(updated)
		if err != nil {
			util.ErrorWithCode(w, err, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResponse)
	}
}
