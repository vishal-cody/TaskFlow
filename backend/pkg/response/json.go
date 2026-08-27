// package response provides standardized json response helpers.
//
// success responses use an envelope: { "data": ... }
// error responses use:               { "error": { "message": "...", "code": "..." } }
//
// this lives in pkg/ because it could be shared across api and worker http surfaces.
package response

import (
	"encoding/json"
	"net/http"
)

// envelope wraps successful responses.
type envelope struct {
	Data interface{} `json:"data"`
}

// errorbody wraps error responses.
type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// json writes a success json response with the given status code.
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(envelope{Data: data}) //nolint:errcheck
}

// error writes a json error response.
func Error(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errorBody{ //nolint:errcheck
		Error: errorDetail{Message: message},
	})
}

// errorwithcode writes a json error response with a machine-readable code.
func ErrorWithCode(w http.ResponseWriter, status int, message, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errorBody{ //nolint:errcheck
		Error: errorDetail{Message: message, Code: code},
	})
}
