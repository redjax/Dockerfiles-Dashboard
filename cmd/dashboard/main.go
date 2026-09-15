package main

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	"local/dockerfiles/internal/dashboard"
)

//go:embed web
var webFiles embed.FS

const (
	defaultDataFile = "dockerfiles.metadata.json"
	defaultAddr     = ":8080"
)

func main() {
	dataFile := os.Getenv("DATA_FILE")
	if dataFile == "" {
		dataFile = defaultDataFile
	}

	addr := os.Getenv("DASHBOARD_ADDR")
	if addr == "" {
		addr = defaultAddr
	}

	tmpl, err := template.ParseFS(
		webFiles,
		"web/dashboard/index.html",
	)
	if err != nil {
		log.Fatalf("loading dashboard template: %v", err)
	}

	staticFiles, err := fs.Sub(
		webFiles,
		"web/dashboard",
	)
	if err != nil {
		log.Fatalf("creating embedded static filesystem: %v", err)
	}

	handler := dashboard.NewHandler(
		dataFile,
		tmpl,
	)

	mux := http.NewServeMux()

	mux.HandleFunc("/", handler.Index)
	mux.HandleFunc("/api/metadata", handler.Metadata)
	mux.HandleFunc("/health", health)

	staticHandler := http.FileServer(
		http.FS(staticFiles),
	)

	mux.Handle(
		"/style.css",
		staticHandler,
	)

	mux.Handle(
		"/dashboard.js",
		staticHandler,
	)

	server := &http.Server{
		Addr:              addr,
		Handler:           securityHeaders(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf(
		"dashboard listening on http://localhost%s",
		addr,
	)

	log.Printf(
		"reading metadata from %s",
		dataFile,
	)

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func health(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		w.Header().Set(
			"Allow",
			http.MethodGet,
		)

		http.Error(
			w,
			"method not allowed",
			http.StatusMethodNotAllowed,
		)

		return
	}

	w.Header().Set(
		"Content-Type",
		"text/plain; charset=utf-8",
	)

	_, _ = fmt.Fprintln(
		w,
		"ok",
	)
}

func securityHeaders(
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(
		func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			w.Header().Set(
				"X-Content-Type-Options",
				"nosniff",
			)

			w.Header().Set(
				"Referrer-Policy",
				"no-referrer",
			)

			w.Header().Set(
				"X-Frame-Options",
				"DENY",
			)

			next.ServeHTTP(
				w,
				r,
			)
		},
	)
}
