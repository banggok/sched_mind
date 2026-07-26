package httpjson

import (
	"net/http/httptest"
	"testing"
)

func TestWrite(t *testing.T) {
	t.Parallel()

	response := httptest.NewRecorder()
	Write(response, 201, map[string]string{"status": "created"})

	if response.Code != 201 {
		t.Fatalf("expected status 201, got %d", response.Code)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("expected application/json content type, got %q", contentType)
	}
	if body := response.Body.String(); body != "{\"status\":\"created\"}\n" {
		t.Fatalf("unexpected response body %q", body)
	}
}
