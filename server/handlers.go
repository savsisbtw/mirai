package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"hash/fnv"
	"net/http"
	"strconv"
	"strings"
)

const version = "0.1.0"

type Blob struct {
	Body []byte
	Gz   []byte
	ETag string
}

func newBlob(body []byte, etag string) *Blob {
	b := &Blob{Body: body, ETag: etag}
	if len(body) > 1024 {
		var buf bytes.Buffer
		w, _ := gzip.NewWriterLevel(&buf, gzip.DefaultCompression)
		w.Write(body)
		w.Close()
		b.Gz = buf.Bytes()
	}
	return b
}

func hashString(s string) string {
	h := fnv.New64a()
	h.Write([]byte(s))
	return strconv.FormatUint(h.Sum64(), 36)
}

func acceptsGzip(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")
}

func setCommon(w http.ResponseWriter) {
	h := w.Header()
	h.Set("Access-Control-Allow-Origin", "*")
	h.Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	h.Set("Access-Control-Allow-Headers", "*")
}

func serveBlob(w http.ResponseWriter, r *http.Request, b *Blob) {
	h := w.Header()
	h.Set("Content-Type", "application/json; charset=utf-8")
	h.Set("ETag", b.ETag)
	h.Set("Cache-Control", "no-cache")
	h.Set("Vary", "Accept-Encoding")
	if strings.Contains(r.Header.Get("If-None-Match"), b.ETag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	if b.Gz != nil && acceptsGzip(r) {
		h.Set("Content-Encoding", "gzip")
		h.Set("Content-Length", strconv.Itoa(len(b.Gz)))
		w.Write(b.Gz)
		return
	}
	h.Set("Content-Length", strconv.Itoa(len(b.Body)))
	w.Write(b.Body)
}

func writeJSON(w http.ResponseWriter, r *http.Request, status int, value any) {
	body, err := json.Marshal(value)
	if err != nil {
		status = http.StatusInternalServerError
		body = []byte(`{"error":"internal error"}`)
	}
	h := w.Header()
	h.Set("Content-Type", "application/json; charset=utf-8")
	h.Set("Vary", "Accept-Encoding")
	if len(body) > 1024 && acceptsGzip(r) {
		h.Set("Content-Encoding", "gzip")
		w.WriteHeader(status)
		gz, _ := gzip.NewWriterLevel(w, gzip.BestSpeed)
		gz.Write(body)
		gz.Close()
		return
	}
	w.WriteHeader(status)
	w.Write(body)
}

func writeError(w http.ResponseWriter, r *http.Request, status int, text string) {
	writeJSON(w, r, status, map[string]string{"error": text})
}

func (a *App) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/status", a.handleStatus)
	mux.HandleFunc("GET /v1/home", a.handleHome)
	mux.HandleFunc("GET /v1/catalog", a.handleCatalog)
	mux.HandleFunc("GET /v1/filters", a.handleFilters)
	mux.HandleFunc("GET /v1/title/{id}", a.handleTitle)
	mux.HandleFunc("GET /v1/schedule", a.handleSchedule)
	mux.HandleFunc("GET /v1/img", a.handleImage)
	mux.HandleFunc("GET /v1/embed", a.handleEmbed)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setCommon(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if !strings.HasPrefix(r.URL.Path, "/v1/") {
			writeError(w, r, http.StatusNotFound, "not found")
			return
		}
		mux.ServeHTTP(w, r)
	})
}

func (a *App) handleStatus(w http.ResponseWriter, r *http.Request) {
	snap := a.snap.Load()
	count := 0
	if snap != nil {
		count = len(snap.Entries)
	}
	writeJSON(w, r, http.StatusOK, map[string]any{
		"ready":     count > 0,
		"titles":    count,
		"anilibria": a.al.getState(),
		"yummy":     a.ym.getState(),
		"version":   version,
	})
}

func (a *App) handleHome(w http.ResponseWriter, r *http.Request) {
	snap := a.snap.Load()
	if snap == nil {
		writeError(w, r, http.StatusServiceUnavailable, "catalog is loading")
		return
	}
	serveBlob(w, r, snap.Home)
}

func (a *App) handleFilters(w http.ResponseWriter, r *http.Request) {
	snap := a.snap.Load()
	if snap == nil {
		writeError(w, r, http.StatusServiceUnavailable, "catalog is loading")
		return
	}
	serveBlob(w, r, snap.Filters)
}

func (a *App) handleCatalog(w http.ResponseWriter, r *http.Request) {
	snap := a.snap.Load()
	if snap == nil {
		writeError(w, r, http.StatusServiceUnavailable, "catalog is loading")
		return
	}
	serveBlob(w, r, snap.catalogBlob(parseQuery(r.URL.Query())))
}
