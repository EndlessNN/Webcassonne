package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/EndlessNN/Webcassonne/server/internal/store"
)

func TestHealth(t *testing.T) {
	rec := httptest.NewRecorder()
	NewRouter(store.NewMemory(), nil).ServeHTTP(rec, httptest.NewRequest("GET", "/api/health", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	want := HealthResponse{Status: "ok", Storage: StorageHealth{Type: "memory", Rooms: 0}}
	if got != want {
		t.Fatalf("body = %+v, want %+v", got, want)
	}
}

func TestUnknownAPIPathReturnsErrorFormat(t *testing.T) {
	rec := httptest.NewRecorder()
	NewRouter(store.NewMemory(), nil).ServeHTTP(rec, httptest.NewRequest("GET", "/api/nope", nil))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	var got ErrorBody
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Error.Code != "NOT_FOUND" || got.Error.Message == "" {
		t.Fatalf("error = %+v", got.Error)
	}
}
