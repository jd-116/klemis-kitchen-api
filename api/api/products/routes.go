package products

import (
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"

	"github.com/go-chi/chi"
	"github.com/lithammer/fuzzysearch/fuzzy"

	"github.com/jd-116/klemis-kitchen-api/db"
	"github.com/jd-116/klemis-kitchen-api/products"
	"github.com/jd-116/klemis-kitchen-api/types"
	"github.com/jd-116/klemis-kitchen-api/util"
)

func Routes(database db.Provider, products products.Provider) *chi.Mux {
	router := chi.NewRouter()
	router.Get("/", GetAll(database, products))
	router.Get("/{id}", GetSingle(database, products))
	return router
}

// Gets all products from the database,
// with an optional search querystring param
func GetAll(database db.Provider, cacheProducts products.Provider) http.HandlerFunc {
	// Use a closure to inject the database provider
	return func(w http.ResponseWriter, r *http.Request) {
		// See if we have search parameter,
		// which can be empty
		search := strings.ToLower(r.URL.Query().Get("search"))

		dbLocations, err := database.GetAllLocations(r.Context())
		if err != nil {
			util.Error(w, err)
			return
		}

		dbProducts, err := database.GetAllProducts(r.Context())
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
		for id, _ := range productMap {
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

// Gets a single product from the database by its ID
func GetSingle(database db.Provider, cacheProducts products.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" {
			util.ErrorWithCode(w, errors.New("the URL parameter is empty"),
				http.StatusBadRequest)
			return
		}

		product, err := database.GetProduct(r.Context(), id)
		if err != nil {
			util.Error(w, err)
			return
		}

		locations, err := cacheProducts.GetAllLocations()
		if err != nil {
			util.Error(w, err)
			return
		}

		dbLocations, err := database.GetAllLocations(r.Context())
		if err != nil {
			util.Error(w, err)
			return
		}

		var finalProduct productsData
		finalProduct.partialProduct.ID = id

		for _, dblocation := range dbLocations {
			for _, val := range locations {
				if dblocation.TransactIdentifier == val {
					singleProduct, err := cacheProducts.GetProduct(val, id)
					if err != nil {
						util.Error(w, err)
						return
					}
					finalProduct.amounts[val] = singleProduct.Amount
				}
			}
		}

		var resultProduct types.ProductData
		resultProduct.ID = finalProduct.partialProduct.ID
		resultProduct.Name = finalProduct.partialProduct.Name
		resultProduct.Nutrition = product.Nutrition
		resultProduct.Thumbnail = product.Thumbnail
		resultProduct.Amounts = finalProduct.amounts

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
