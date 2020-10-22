package util

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jd-116/klemis-kitchen-api/db"
	"github.com/jd-116/klemis-kitchen-api/types"
)

// ResponseCodeFromError resolves a status code from an error
func ResponseCodeFromError(err error) int {
	switch err.(type) {
	case *db.DuplicateIDError:
		return http.StatusBadRequest
	case *db.NotFoundError:
		return http.StatusNotFound
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

	jsonResponse, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(statusCode)
	w.Write(jsonResponse)
}
