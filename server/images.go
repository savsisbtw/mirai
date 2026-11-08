package main

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var imageClient = &http.Client{
	Timeout:   10 * time.Second,
	Transport: client.Transport,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > 3 || !allowedImageHost(req.URL.Hostname()) {
			return errors.New("redirect not allowed")
		}
		return nil
	},
}

type imageFlight struct {
	done chan struct{}
	err  error
}

type imageFlights struct {
	mu      sync.Mutex
	waiting map[string]*imageFlight
}

func allowedImageHost(host string) bool {
	host = strings.ToLower(host)
	return host == "aniliberty.top" || host == "yani.tv" || strings.HasSuffix(host, ".yani.tv")
}

func cleanImageURL(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "//") {
		raw = "https:" + raw
	} else if strings.HasPrefix(raw, "/") {
		raw = "https://aniliberty.top" + raw
	}
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return "", false
	}
	if !allowedImageHost(parsed.Hostname()) {
		return "", false
	}
	return parsed.String(), true
}

func imageContentType(target string, data []byte) string {
	parsed, err := url.Parse(target)
	if err == nil {
		switch strings.ToLower(path.Ext(parsed.Path)) {
		case ".jpg", ".jpeg":
			return "image/jpeg"
		case ".png":
			return "image/png"
		case ".webp":
			return "image/webp"
		case ".avif":
			return "image/avif"
		case ".gif":
			return "image/gif"
		}
	}
	return http.DetectContentType(data)
}

func downloadImage(target, file string) error {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(300*attempt) * time.Millisecond)
		}
		req, err := http.NewRequest("GET", target, nil)
		if err != nil {
			return err
		}
		req.Header.Set("User-Agent", userAgent)
		resp, err := imageClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		data, err := io.ReadAll(io.LimitReader(resp.Body, 12<<20))
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode != 200 {
			lastErr = fmt.Errorf("image status %d", resp.StatusCode)
			if resp.StatusCode < 500 && resp.StatusCode != 429 {
				return lastErr
			}
			continue
		}
		if len(data) == 0 {
			lastErr = errors.New("empty image")
			continue
		}
		tmp := file + ".tmp" + hex.EncodeToString(data[:min(4, len(data))])
		if err := os.WriteFile(tmp, data, 0o644); err != nil {
			return err
		}
		return os.Rename(tmp, file)
	}
	return lastErr
}

func (a *App) ensureImage(target, file string) error {
	a.images.mu.Lock()
	flight, running := a.images.waiting[target]
	if !running {
		flight = &imageFlight{done: make(chan struct{})}
		a.images.waiting[target] = flight
	}
	a.images.mu.Unlock()
	if running {
		<-flight.done
		return flight.err
	}
	flight.err = downloadImage(target, file)
	a.images.mu.Lock()
	delete(a.images.waiting, target)
	a.images.mu.Unlock()
	close(flight.done)
	return flight.err
}

func (a *App) handleImage(w http.ResponseWriter, r *http.Request) {
	target, ok := cleanImageURL(r.URL.Query().Get("u"))
	if !ok {
		writeError(w, r, http.StatusBadRequest, "bad image url")
		return
	}
	sum := sha1.Sum([]byte(target))
	dir := filepath.Join(a.dataDir, "img")
	file := filepath.Join(dir, hex.EncodeToString(sum[:]))
	if _, err := os.Stat(file); err != nil {
		os.MkdirAll(dir, 0o755)
		if err := a.ensureImage(target, file); err != nil {
			writeError(w, r, http.StatusBadGateway, "image source unavailable")
			return
		}
	}
	f, err := os.Open(file)
	if err != nil {
		writeError(w, r, http.StatusBadGateway, "image source unavailable")
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		writeError(w, r, http.StatusBadGateway, "image source unavailable")
		return
	}
	head := make([]byte, 512)
	n, _ := f.Read(head)
	f.Seek(0, io.SeekStart)
	h := w.Header()
	h.Set("Content-Type", imageContentType(target, head[:n]))
	h.Set("Cache-Control", "public, max-age=31536000, immutable")
	http.ServeContent(w, r, "", info.ModTime(), f)
}

func (a *App) handleEmbed(w http.ResponseWriter, r *http.Request) {
	link, ok := cleanPlayerURL(r.URL.Query().Get("u"))
	if !ok {
		writeError(w, r, http.StatusBadRequest, "bad player url")
		return
	}
	page := `<!doctype html>
<html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<meta name="referrer" content="origin">
<style>html,body{margin:0;height:100%;background:#000}iframe{position:fixed;inset:0;width:100%;height:100%;border:0}</style>
</head><body>
<iframe src="` + html.EscapeString(link) + `" allow="autoplay; fullscreen" allowfullscreen referrerpolicy="origin"></iframe>
</body></html>`
	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Cache-Control", "no-store")
	w.Write([]byte(page))
}
