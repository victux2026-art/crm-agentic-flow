package main

import (
	"embed"
	"io/fs"
	"net/http"

	chi "github.com/go-chi/chi/v5"
)

//go:embed static/*
var uiFS embed.FS

func mountUIRoutes(r chi.Router) {
	staticFS, err := fs.Sub(uiFS, "static")
	if err != nil {
		panic(err)
	}

	fileServer := http.FileServer(http.FS(staticFS))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/app", http.StatusFound)
	})

	r.Get("/app", func(w http.ResponseWriter, r *http.Request) {
		serveEmbeddedFile(w, staticFS, "index.html", "text/html; charset=utf-8")
	})

	r.Get("/styles.css", func(w http.ResponseWriter, r *http.Request) {
		serveEmbeddedFile(w, staticFS, "styles.css", "text/css; charset=utf-8")
	})

	r.Get("/app.js", func(w http.ResponseWriter, r *http.Request) {
		serveEmbeddedFile(w, staticFS, "app.js", "application/javascript; charset=utf-8")
	})

	r.Handle("/assets/*", http.StripPrefix("/assets/", fileServer))
}

func serveEmbeddedFile(w http.ResponseWriter, filesystem fs.FS, name, contentType string) {
	content, err := fs.ReadFile(filesystem, name)
	if err != nil {
		http.NotFound(w, nil)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}
