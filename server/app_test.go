package main

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDiskRoundTrip(t *testing.T) {
	dir := t.TempDir()
	app := newApp(dir)
	app.al.current = []Item{{Source: "anilibria", ID: 1, Alias: "a", Shiki: 5, Name: "Имя\nс переводом", Desc: "строка1\nстрока2"}}
	app.ym.current = []Item{{Source: "yummy", ID: 2, Shiki: 5, Name: "Имя 2"}, {Source: "yummy", ID: 3, Name: "Другое"}}
	app.saveDisk()

	loaded := newApp(dir)
	if !loaded.loadDisk() {
		t.Fatal("load failed")
	}
	if len(loaded.al.current) != 1 || len(loaded.ym.current) != 2 {
		t.Fatalf("counts: %d %d", len(loaded.al.current), len(loaded.ym.current))
	}
	if loaded.al.current[0].Desc != "строка1\nстрока2" {
		t.Fatalf("newline inside string broke: %q", loaded.al.current[0].Desc)
	}
	loaded.rebuildLocked()
	snap := loaded.snap.Load()
	if snap == nil || len(snap.Entries) != 2 {
		t.Fatal("snapshot not built from disk")
	}
}

func TestLoadDiskMissingOrBroken(t *testing.T) {
	app := newApp(t.TempDir())
	if app.loadDisk() {
		t.Fatal("must not load a missing file")
	}
}

func TestSourceFinishRules(t *testing.T) {
	var s SourceState
	s.current = make([]Item, 100)

	s.finish(nil, errors.New("down"))
	if len(s.current) != 100 || s.getState() != "error" {
		t.Fatalf("source down must keep old items: %d %s", len(s.current), s.getState())
	}
	s.finish(make([]Item, 10), nil)
	if len(s.current) != 100 || s.getState() != "error" {
		t.Fatalf("tiny result must not replace: %d %s", len(s.current), s.getState())
	}
	s.finish(make([]Item, 90), errors.New("one page failed"))
	if len(s.current) != 90 || s.getState() != "error" {
		t.Fatalf("mostly good result is adopted: %d %s", len(s.current), s.getState())
	}
	s.finish(make([]Item, 120), nil)
	if len(s.current) != 120 || s.getState() != "done" {
		t.Fatalf("full result: %d %s", len(s.current), s.getState())
	}
}

func TestHandlersBasics(t *testing.T) {
	app := newApp(t.TempDir())
	handler := app.routes()

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/v1/home", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("home before load: %d", rec.Code)
	}

	app.snap.Store(buildSnapshot(sampleEntries()))

	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/v1/status", nil))
	var status map[string]any
	json.Unmarshal(rec.Body.Bytes(), &status)
	if status["ready"] != true || status["titles"].(float64) != 6 {
		t.Fatalf("status: %v", status)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatal("cors missing")
	}

	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/v1/title/nope", nil))
	if rec.Code != 404 {
		t.Fatalf("title 404: %d", rec.Code)
	}
	var failure map[string]string
	json.Unmarshal(rec.Body.Bytes(), &failure)
	if failure["error"] == "" {
		t.Fatal("error body missing")
	}

	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("OPTIONS", "/v1/catalog", nil))
	if rec.Code != 204 {
		t.Fatalf("options: %d", rec.Code)
	}
}

func TestETagAndGzip(t *testing.T) {
	app := newApp(t.TempDir())
	app.snap.Store(buildSnapshot(syntheticEntries(300)))
	handler := app.routes()

	req := httptest.NewRequest("GET", "/v1/catalog?limit=100", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatal("not gzipped")
	}
	reader, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(reader)
	var result catalogResult
	if err := json.Unmarshal(body, &result); err != nil || len(result.Items) != 100 {
		t.Fatalf("bad gzip body: %v %d", err, len(result.Items))
	}

	etag := rec.Header().Get("ETag")
	req = httptest.NewRequest("GET", "/v1/catalog?limit=100", nil)
	req.Header.Set("If-None-Match", etag)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotModified {
		t.Fatalf("want 304, got %d", rec.Code)
	}

	for _, path := range []string{"/v1/home", "/v1/filters"} {
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		etag = rec.Header().Get("ETag")
		req = httptest.NewRequest("GET", path, nil)
		req.Header.Set("If-None-Match", etag)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotModified {
			t.Fatalf("%s want 304, got %d", path, rec.Code)
		}
	}
}

func TestEmbedPage(t *testing.T) {
	app := newApp(t.TempDir())
	handler := app.routes()
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/v1/embed?u=https%3A%2F%2Fkodikplayer.com%2Fa%22%3E%3Cscript%3E", nil))
	if rec.Code != 200 {
		t.Fatalf("embed: %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/v1/embed?u=https%3A%2F%2Fevil.com%2Fa", nil))
	if rec.Code != 400 {
		t.Fatalf("evil embed: %d", rec.Code)
	}
}

func TestScheduleBuild(t *testing.T) {
	snap := buildSnapshot(sampleEntries())
	var list []alScheduleItem
	var item alScheduleItem
	item.Release.Alias = "naruto"
	item.Release.PublishDay.Value = 3
	item.NextEpisode = 7
	list = append(list, item)
	var unknown alScheduleItem
	unknown.Release.Alias = "not-in-catalog"
	unknown.Release.PublishDay.Value = 3
	list = append(list, unknown)

	days := buildSchedule(snap, list)
	if len(days) != 7 || days[2].Name != "Среда" || len(days[2].Items) != 1 || days[2].Items[0].Episode != "7" {
		t.Fatalf("schedule: %+v", days[2])
	}
}
