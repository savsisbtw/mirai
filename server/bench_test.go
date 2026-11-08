package main

import (
	"net/http/httptest"
	"sort"
	"testing"
	"time"
)

func benchApp(b testing.TB) *App {
	app := newApp(b.TempDir())
	app.snap.Store(buildSnapshot(syntheticEntries(20000)))
	return app
}

func BenchmarkBuildSnapshot20k(b *testing.B) {
	entries := syntheticEntries(20000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buildSnapshot(entries)
	}
}

func BenchmarkCatalogHandlerCached(b *testing.B) {
	app := benchApp(b)
	handler := app.routes()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/v1/catalog?limit=48", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		handler.ServeHTTP(httptest.NewRecorder(), req)
	}
}

func BenchmarkCatalogComputeDefault(b *testing.B) {
	snap := buildSnapshot(syntheticEntries(20000))
	q := parseQuery(map[string][]string{})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		snap.catalog(q)
	}
}

func BenchmarkCatalogComputeGenreYear(b *testing.B) {
	snap := buildSnapshot(syntheticEntries(20000))
	q := parseQuery(map[string][]string{"genre": {"Экшен,Драма"}, "year": {"2005-2020"}, "sort": {"rating"}})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		snap.catalog(q)
	}
}

func BenchmarkCatalogComputeSearch(b *testing.B) {
	snap := buildSnapshot(syntheticEntries(20000))
	q := parseQuery(map[string][]string{"q": {"кару"}})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		snap.catalog(q)
	}
}

func BenchmarkHomeHandler(b *testing.B) {
	app := benchApp(b)
	handler := app.routes()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/v1/home", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		handler.ServeHTTP(httptest.NewRecorder(), req)
	}
}

func TestCatalogLatencyP99(t *testing.T) {
	snap := buildSnapshot(syntheticEntries(20000))
	queries := []map[string][]string{
		{},
		{"sort": {"new"}},
		{"sort": {"name"}, "page": {"50"}},
		{"genre": {"Экшен,Драма"}, "year": {"2005-2020"}},
		{"q": {"кару"}},
		{"q": {"на"}, "type": {"ТВ"}},
		{"status": {"ongoing"}, "sort": {"rating"}},
	}
	var times []time.Duration
	for round := 0; round < 300; round++ {
		for _, values := range queries {
			start := time.Now()
			snap.catalog(parseQuery(values))
			times = append(times, time.Since(start))
		}
	}
	sort.Slice(times, func(a, b int) bool { return times[a] < times[b] })
	p50 := times[len(times)/2]
	p99 := times[len(times)*99/100]
	t.Logf("uncached compute, 20000 titles: p50 %s p99 %s max %s", p50, p99, times[len(times)-1])
	if p99 > 5*time.Millisecond {
		t.Fatalf("p99 too slow: %s", p99)
	}
}
