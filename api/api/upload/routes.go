package products

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi"

	"github.com/jd-116/klemis-kitchen-api/auth"
	"github.com/jd-116/klemis-kitchen-api/upload"
	"github.com/jd-116/klemis-kitchen-api/util"
)

const multipartPartKey string = "file"

// Routes creates a new Chi router with all of the routes for the upload,
// at the root level
func Routes(uploadProvider upload.Provider) *chi.Mux {
	router := chi.NewRouter()

	// Load the valid list of mime types from the environment
	validMimeTypes := make(map[string]struct{})
	if value, ok := os.LookupEnv("UPLOAD_MIME_TYPES"); ok {
		value = strings.TrimSpace(value)
		if value != "" {
			validMimeTypeSlice := strings.Split(value, "|")
			for _, m := range validMimeTypeSlice {
				validMimeTypes[m] = struct{}{}
			}
		}
	}

	// Create the mime type validator
	validMime := func(m string) bool {
		if len(validMimeTypes) == 0 {
			return true
		}

		if _, ok := validMimeTypes[m]; ok {
			return true
		}

		return false
	}

	// Admin-only routes
	router.Group(func(r chi.Router) {
		// Ensure the user has access
		r.Use(auth.AdminAuthenticated)
		r.Post("/", Upload(uploadProvider, validMime))
	})
	return router
}

// Upload provides a pass-through route that takes in a multi-part
// HTTP request and uploads it to S3,
// returning a URL that can be used to reference the image
func Upload(uploadProvider upload.Provider, validMime func(string) bool) http.HandlerFunc {
	// Use a closure to inject the database provider
	return func(w http.ResponseWriter, r *http.Request) {
		// Limit the read size to the configured size
		r.Body = http.MaxBytesReader(w, r.Body, uploadProvider.MaxBytes())

		// Get the mutlipart part for the file
		mr, err := r.MultipartReader()
		if err != nil {
			util.Error(w, err)
			return
		}
		var uploadFile *multipart.Part = nil
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				util.Error(w, err)
				return
			}

			if p.FormName() == multipartPartKey {
				uploadFile = p
				break
			}
		}

		// Ensure a file was passed in
		if uploadFile == nil {
			util.ErrorWithCode(w,
				fmt.Errorf("Expected multipart form submission with '%s' entry", multipartPartKey),
				http.StatusBadRequest)
			return
		}

		// Make sure the MIME type is valid
		// Only the first 512 bytes are used to sniff the content type,
		// so create a multi-reader to pass the file on
		headerBuffer := make([]byte, 512)
		_, err = uploadFile.Read(headerBuffer)
		if err != nil {
			util.Error(w, err)
			return
		}
		headerReader := bytes.NewReader(headerBuffer)
		fileReader := io.MultiReader(headerReader, uploadFile)

		// Use the net/http package's handy DetectContentType function. Always returns a valid
		// content-type by returning "application/octet-stream" if no others seemed to match.
		contentType := http.DetectContentType(headerBuffer)
		if contentType == "application/octet-stream" || !validMime(contentType) {
			util.ErrorWithCode(w,
				fmt.Errorf("Unsupported file upload MIME type '%s'", contentType),
				http.StatusBadRequest)
			return
		}

		// Derive the file extension based on the Mime type
		fileExtensions, err := mime.ExtensionsByType(contentType)
		if err != nil {
			util.ErrorWithCode(w, err, http.StatusBadRequest)
			return
		}
		if len(fileExtensions) == 0 {
			util.ErrorWithCode(w,
				fmt.Errorf("Unsupported file upload MIME type '%s'", contentType),
				http.StatusBadRequest)
			return
		}
		fileExt := fileExtensions[0]

		// Stream the file into the upload provider
		fileURL, err := uploadProvider.Upload(r.Context(), fileReader, fileExt, contentType)
		if err != nil {
			util.Error(w, err)
			return
		}

		// Return the resultant URL in a JSON object
		jsonResponse, err := json.Marshal(map[string]interface{}{"url": fileURL})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResponse)
	}
}
