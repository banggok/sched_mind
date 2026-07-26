package httpjson

import (
	"encoding/json"
	"net/http"
)

// Write serializes payload as JSON using the supplied HTTP status code.
func Write(response http.ResponseWriter, status int, payload any) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(payload)
}
