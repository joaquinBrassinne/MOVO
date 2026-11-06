package httpx

import (
	"encoding/json"
	"io"
	"net/http"
)

// Codes shared by every module's endpoints.
const (
	CodeInvalidRequest = "invalid_request"
	CodeInternal       = "internal_error"
)

// maxBodyBytes caps a JSON request body. The API only receives small forms, so a
// larger body is hostile or broken.
const maxBodyBytes = 64 << 10

// WriteJSON writes body as JSON with the given status.
func WriteJSON(w http.ResponseWriter, status int, body any) {
	writeJSON(w, status, body)
}

// DecodeJSON reads the request body into dst and rejects anything but exactly one
// well-formed JSON value with known fields. On failure it has already written
// the 400 error envelope and returns false, so the caller just returns.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		WriteError(w, http.StatusBadRequest, CodeInvalidRequest, "request body is not valid JSON for this endpoint", nil)
		return false
	}

	// A second value, or garbage after the first, means the body is not what the
	// client claims to have sent.
	if _, err := decoder.Token(); err != io.EOF {
		WriteError(w, http.StatusBadRequest, CodeInvalidRequest, "request body must contain a single JSON value", nil)
		return false
	}

	return true
}
