// Package httpapi — REST-ендпоінти з docs/api.md.
package httpapi

import (
	"net/http"

	"github.com/EndlessNN/Webcassonne/server/internal/store"
)

type HealthResponse struct {
	Status  string        `json:"status"`
	Storage StorageHealth `json:"storage"`
}

type StorageHealth struct {
	Type  string `json:"type"`
	Rooms int    `json:"rooms"`
}

// NewRouter реєструє /api/*; решту шляхів обробляє static (зібраний web/), якщо він не nil.
func NewRouter(s *store.Memory, static http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, HealthResponse{
			Status:  "ok",
			Storage: StorageHealth{Type: "memory", Rooms: s.RoomCount()},
		})
	})
	mux.HandleFunc("/api/", func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Ендпоінт не існує")
	})
	if static != nil {
		mux.Handle("/", static)
	}
	return mux
}
