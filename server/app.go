package main

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type SourceState struct {
	mu      sync.Mutex
	state   string
	current []Item
	pending []Item
}

func (s *SourceState) getState() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

func (s *SourceState) setState(state string) {
	s.mu.Lock()
	s.state = state
	s.mu.Unlock()
}

func (s *SourceState) items() []Item {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.current) > 0 {
		return s.current
	}
	return s.pending
}

func (s *SourceState) setPending(items []Item) {
	s.mu.Lock()
	s.pending = items
	s.mu.Unlock()
}

func (s *SourceState) needsPartial() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.current) == 0 && len(s.pending) > 0
}

func (s *SourceState) finish(items []Item, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	adopt := len(items) > 0 && (len(s.current) == 0 || len(items)*5 >= len(s.current)*4)
	if adopt {
		s.current = items
	}
	s.pending = nil
	if err == nil && adopt {
		s.state = "done"
	} else if err == nil && len(s.current) > 0 {
		s.state = "error"
		log.Printf("source returned too few items (%d), keeping old %d", len(items), len(s.current))
	} else {
		s.state = "error"
	}
}

type App struct {
	dataDir     string
	snap        atomic.Pointer[Snapshot]
	al          SourceState
	ym          SourceState
	buildMu     sync.Mutex
	lastPartial time.Time
	voices      voiceCache
	schedule    scheduleCache
	images      imageFlights
}

func newApp(dataDir string) *App {
	a := &App{dataDir: dataDir}
	a.al.state = "running"
	a.ym.state = "running"
	a.images.waiting = map[string]*imageFlight{}
	a.voices.items = map[string]voiceEntry{}
	return a
}

func (a *App) snapshotPath() string {
	return filepath.Join(a.dataDir, "catalog.json")
}

func (a *App) loadDisk() bool {
	data, err := os.ReadFile(a.snapshotPath())
	if err != nil {
		return false
	}
	lines := bytes.Split(data, []byte("\n"))
	if len(lines) < 3 {
		return false
	}
	lines = lines[1 : len(lines)-1]
	items := make([]Item, len(lines))
	valid := make([]bool, len(lines))
	parallelRange(len(lines), 8, func(lo, hi int) {
		for i := lo; i < hi; i++ {
			line := bytes.TrimSuffix(bytes.TrimSpace(lines[i]), []byte(","))
			if len(line) == 0 || line[0] != '{' {
				continue
			}
			if json.Unmarshal(line, &items[i]) == nil {
				valid[i] = true
			}
		}
	})
	var al, ym []Item
	for i, item := range items {
		if !valid[i] {
			continue
		}
		if item.Source == "anilibria" {
			al = append(al, item)
		} else {
			ym = append(ym, item)
		}
	}
	if len(al)+len(ym) == 0 {
		return false
	}
	a.al.current = al
	a.ym.current = ym
	return true
}

func (a *App) saveDisk() {
	var buf bytes.Buffer
	buf.WriteString(`{"saved":` + strconv.FormatInt(time.Now().Unix(), 10) + `,"items":[` + "\n")
	all := append(append([]Item{}, a.al.items()...), a.ym.items()...)
	for i, item := range all {
		data, err := json.Marshal(item)
		if err != nil {
			continue
		}
		buf.Write(data)
		if i < len(all)-1 {
			buf.WriteByte(',')
		}
		buf.WriteByte('\n')
	}
	buf.WriteString("]}\n")
	tmp := a.snapshotPath() + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o644); err != nil {
		log.Printf("save: %v", err)
		return
	}
	if err := os.Rename(tmp, a.snapshotPath()); err != nil {
		log.Printf("save: %v", err)
	}
}

func (a *App) rebuild() {
	entries := mergeItems(a.al.items(), a.ym.items())
	if len(entries) == 0 {
		return
	}
	a.snap.Store(buildSnapshot(entries))
}

func (a *App) rebuildLocked() {
	a.buildMu.Lock()
	defer a.buildMu.Unlock()
	a.rebuild()
}

func (a *App) partialRebuild() {
	if !a.al.needsPartial() && !a.ym.needsPartial() {
		return
	}
	if !a.buildMu.TryLock() {
		return
	}
	defer a.buildMu.Unlock()
	snap := a.snap.Load()
	total := len(a.al.items()) + len(a.ym.items())
	recent := time.Since(a.lastPartial) < 1500*time.Millisecond
	if snap != nil && recent && len(snap.Entries)*2 > total {
		return
	}
	a.lastPartial = time.Now()
	a.rebuild()
}

func (a *App) runSource(name string, s *SourceState, crawl func(func([]Item)) ([]Item, error)) {
	started := time.Now()
	s.setState("running")
	items, err := crawl(func(part []Item) {
		s.setPending(part)
		a.partialRebuild()
	})
	s.finish(items, err)
	log.Printf("%s: %d items in %s, state %s, err %v", name, len(items), time.Since(started).Round(time.Millisecond), s.getState(), err)
	a.rebuildLocked()
}

func (a *App) crawlOnce() {
	started := time.Now()
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		a.runSource("anilibria", &a.al, crawlAnilibria)
	}()
	go func() {
		defer wg.Done()
		a.runSource("yummy", &a.ym, crawlYummy)
	}()
	wg.Wait()
	a.saveDisk()
	snap := a.snap.Load()
	if snap != nil {
		log.Printf("crawl done in %s, titles %d", time.Since(started).Round(time.Millisecond), len(snap.Entries))
	}
}

func (a *App) crawlLoop() {
	for {
		a.crawlOnce()
		time.Sleep(6 * time.Hour)
	}
}
