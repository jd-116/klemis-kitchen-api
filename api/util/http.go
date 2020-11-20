package util

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/jd-116/klemis-kitchen-api/cas"
	"github.com/jd-116/klemis-kitchen-api/db"
	"github.com/jd-116/klemis-kitchen-api/products"
	"github.com/jd-116/klemis-kitchen-api/types"
)

// ResponseCodeFromError resolves a status code from an error
func ResponseCodeFromError(err error) int {
	switch err.(type) {
	case *db.DuplicateIDError:
		return http.StatusBadRequest
	case *db.NotFoundError:
		return http.StatusNotFound
	case *cas.CASValidationFailedError:
		return http.StatusUnauthorized
	case *products.CacheNotInitializedError:
		return http.StatusTooEarly
	case *products.LocationNotFoundError:
		return http.StatusNotFound
	case *products.PartialProductNotFoundError:
		return http.StatusNotFound
	case *json.InvalidUTF8Error:
		return http.StatusBadRequest
	case *json.InvalidUnmarshalError:
		return http.StatusBadRequest
	case *json.MarshalerError:
		return http.StatusBadRequest
	case *json.SyntaxError:
		return http.StatusBadRequest
	case *json.UnmarshalFieldError:
		return http.StatusBadRequest
	case *json.UnmarshalTypeError:
		return http.StatusBadRequest
	case *json.UnsupportedValueError:
		return http.StatusBadRequest
	case *json.UnsupportedTypeError:
		return http.StatusBadRequest
	case *url.Error:
		return http.StatusBadRequest
	case *url.EscapeError:
		return http.StatusBadRequest
	case *url.InvalidHostError:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// Error creates a standardized error response
func Error(w http.ResponseWriter, originalError error) {
	ErrorWithCode(w, originalError, ResponseCodeFromError(originalError))
}

// ErrorWithCode creates a standardized error response with a status code
func ErrorWithCode(w http.ResponseWriter, originalError error, statusCode int) {
	response := types.ErrorResponse{
		Message: fmt.Sprint(originalError),
	}

	log.Printf("error while handling HTTP request: %s\n", originalError)
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(statusCode)
	w.Write(jsonResponse)
}
