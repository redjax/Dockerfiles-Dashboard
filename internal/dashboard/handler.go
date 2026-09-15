package dashboard

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"time"

	"local/dockerfiles/internal/services/fsService"
)

type Handler struct {
	dataFile string
	template *template.Template
}

func NewHandler(
	dataFile string,
	tmpl *template.Template,
) *Handler {
	return &Handler{
		dataFile: dataFile,
		template: tmpl,
	}
}

func (h *Handler) loadMetadata() (
	fsService.Metadata,
	error,
) {
	svc := fsService.New()

	return svc.LoadJsonMetadata(
		h.dataFile,
	)
}

func (h *Handler) Index(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

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

	metadata, err := h.loadMetadata()
	if err != nil {
		http.Error(
			w,
			"failed to load metadata: "+err.Error(),
			http.StatusInternalServerError,
		)

		return
	}

	data := BuildPageData(
		metadata,
	)

	w.Header().Set(
		"Content-Type",
		"text/html; charset=utf-8",
	)

	if err := h.template.Execute(
		w,
		data,
	); err != nil {
		log.Printf(
			"rendering dashboard: %v",
			err,
		)
	}
}

func (h *Handler) Metadata(
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

	metadata, err := h.loadMetadata()
	if err != nil {
		http.Error(
			w,
			"failed to load metadata: "+err.Error(),
			http.StatusInternalServerError,
		)

		return
	}

	w.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)

	w.Header().Set(
		"Cache-Control",
		"no-store",
	)

	response := struct {
		Metadata  fsService.Metadata `json:"metadata"`
		Timestamp time.Time          `json:"timestamp"`
	}{
		Metadata:  metadata,
		Timestamp: time.Now().UTC(),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf(
			"encoding metadata response: %v",
			err,
		)
	}
}
