package main

import (
	"log"
	"net/http"
	"os"

	"github.com/EndlessNN/Webcassonne/server/internal/httpapi"
	"github.com/EndlessNN/Webcassonne/server/internal/store"
)

func main() {
	addr := ":" + env("PORT", "8090")
	webDir := env("WEB_DIR", "../web/dist")

	var static http.Handler
	if _, err := os.Stat(webDir); err == nil {
		static = http.FileServer(http.Dir(webDir))
	} else {
		log.Printf("%s не знайдено: UI не роздається (зберіть web/ або запустіть npm run dev)", webDir)
	}

	log.Printf("Webcassonne: http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, httpapi.NewRouter(store.NewMemory(), static)))
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
