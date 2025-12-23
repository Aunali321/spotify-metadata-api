package api

import (
	"embed"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"metadata-api/internal/db"
)

//go:embed openapi.yaml
var openapiSpec embed.FS

type Handler struct {
	db *db.DB
}

func New(database *db.DB) *Handler {
	return &Handler{db: database}
}

func (h *Handler) Routes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /lookup/isrc/{isrc}", h.lookupISRC)
	mux.HandleFunc("GET /lookup/track/{id}", h.lookupTrack)
	mux.HandleFunc("GET /lookup/artist/{id}", h.lookupArtist)
	mux.HandleFunc("GET /lookup/album/{id}", h.lookupAlbum)
	mux.HandleFunc("GET /lookup/album/{id}/tracks", h.albumTracks)
	mux.HandleFunc("GET /search/artist", h.searchArtist)
	mux.HandleFunc("GET /search/track", h.searchTrack)
	mux.HandleFunc("GET /health", h.health)

	mux.HandleFunc("GET /openapi.yaml", h.openapiSpec)
	mux.HandleFunc("GET /docs", h.swaggerUI)
	mux.HandleFunc("GET /", h.swaggerUI)

	return mux
}

func (h *Handler) openapiSpec(w http.ResponseWriter, r *http.Request) {
	data, err := openapiSpec.ReadFile("openapi.yaml")
	if err != nil {
		http.Error(w, "spec not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/yaml")
	w.Write(data)
}

func (h *Handler) swaggerUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
  <title>Spotify Metadata API</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
  <style>
    body { margin: 0; }
    .swagger-ui .topbar { display: none; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    SwaggerUIBundle({
      url: '/openapi.yaml',
      dom_id: '#swagger-ui',
      presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
      layout: 'BaseLayout'
    });
  </script>
</body>
</html>`))
}

func (h *Handler) lookupISRC(w http.ResponseWriter, r *http.Request) {
	isrc := r.PathValue("isrc")
	if isrc == "" {
		http.Error(w, "isrc required", http.StatusBadRequest)
		return
	}

	tracks, err := h.db.LookupISRC(r.Context(), isrc)
	if err != nil {
		slog.Error("lookup isrc", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, tracks)
}

func (h *Handler) lookupTrack(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}

	track, err := h.db.LookupTrack(r.Context(), id)
	if err != nil {
		slog.Error("lookup track", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if track == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	writeJSON(w, track)
}

func (h *Handler) lookupArtist(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}

	artist, err := h.db.LookupArtist(r.Context(), id)
	if err != nil {
		slog.Error("lookup artist", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if artist == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	writeJSON(w, artist)
}

func (h *Handler) lookupAlbum(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}

	album, err := h.db.LookupAlbum(r.Context(), id)
	if err != nil {
		slog.Error("lookup album", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if album == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	writeJSON(w, album)
}

func (h *Handler) albumTracks(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}

	tracks, err := h.db.GetAlbumTracks(r.Context(), id)
	if err != nil {
		slog.Error("album tracks", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, tracks)
}

func (h *Handler) searchArtist(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		http.Error(w, "q parameter required", http.StatusBadRequest)
		return
	}

	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	artists, err := h.db.SearchArtist(r.Context(), q, limit)
	if err != nil {
		slog.Error("search artist", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, artists)
}

func (h *Handler) searchTrack(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		http.Error(w, "q parameter required", http.StatusBadRequest)
		return
	}

	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	tracks, err := h.db.SearchTrack(r.Context(), q, limit)
	if err != nil {
		slog.Error("search track", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, tracks)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("encode json", "err", err)
	}
}
