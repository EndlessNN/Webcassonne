package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/EndlessNN/Webcassonne/server/internal/httpapi"
	"github.com/EndlessNN/Webcassonne/server/internal/room"
	"github.com/EndlessNN/Webcassonne/server/internal/store"
)

func main() {
	addr := ":" + env("PORT", "8090")
	webDir := env("WEB_DIR", "../web/dist")

	var static http.Handler
	if _, err := os.Stat(webDir); err == nil {
		files := http.FileServer(http.Dir(webDir))
		static = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/r/") { // посилання-запрошення відкриває SPA
				http.ServeFile(w, r, filepath.Join(webDir, "index.html"))
				return
			}
			files.ServeHTTP(w, r)
		})
	} else {
		log.Printf("%s не знайдено: UI не роздається (зберіть web/ або запустіть npm run dev)", webDir)
	}

	s := store.NewMemory()
	log.Printf("Webcassonne: http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, httpapi.NewRouter(s, room.NewService(s), static)))
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
