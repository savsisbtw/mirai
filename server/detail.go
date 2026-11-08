package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const voiceTTL = 10 * time.Minute

type voiceEntry struct {
	voices  []Voice
	expires time.Time
}

type voiceCache struct {
	mu    sync.Mutex
	items map[string]voiceEntry
}

func (c *voiceCache) get(id string) ([]Voice, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.items[id]
	if !ok || time.Now().After(entry.expires) {
		return nil, false
	}
	return entry.voices, true
}

func (c *voiceCache) put(id string, voices []Voice, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.items) > 500 {
		now := time.Now()
		for key, entry := range c.items {
			if now.After(entry.expires) {
				delete(c.items, key)
			}
		}
	}
	c.items[id] = voiceEntry{voices: voices, expires: time.Now().Add(ttl)}
}

func (a *App) loadVoices(e *Entry) []Voice {
	if cached, ok := a.voices.get(e.ID); ok {
		return cached
	}
	var alVoice *Voice
	var ymVoices []Voice
	var alErr, ymErr error
	var wg sync.WaitGroup
	if e.ALID > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			alVoice, alErr = fetchAnilibriaVoice(e.ALID)
		}()
	}
	if e.YID > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ymVoices, ymErr = fetchYummyVoices(e.YID)
		}()
	}
	wg.Wait()

	voices := []Voice{}
	if alVoice != nil {
		voices = append(voices, *alVoice)
	}
	voices = append(voices, ymVoices...)

	ttl := voiceTTL
	if alErr != nil || ymErr != nil {
		ttl = 30 * time.Second
	}
	a.voices.put(e.ID, voices, ttl)
	return voices
}

func (a *App) handleTitle(w http.ResponseWriter, r *http.Request) {
	snap := a.snap.Load()
	if snap == nil {
		writeError(w, r, http.StatusServiceUnavailable, "catalog is loading")
		return
	}
	index, ok := snap.ByID[r.PathValue("id")]
	if !ok {
		writeError(w, r, http.StatusNotFound, "title not found")
		return
	}
	e := &snap.Entries[index]
	alt := e.Alt
	if alt == nil {
		alt = []string{}
	}
	detail := TitleDetail{
		TitleFull: makeFull(e),
		NameAlt:   alt,
		Voices:    a.loadVoices(e),
	}
	writeJSON(w, r, http.StatusOK, detail)
}

type scheduleCache struct {
	mu      sync.Mutex
	snapID  string
	fetched time.Time
	body    *Blob
}

var dayNames = []string{"Понедельник", "Вторник", "Среда", "Четверг", "Пятница", "Суббота", "Воскресенье"}

type scheduleItem struct {
	Title   json.RawMessage `json:"title"`
	Episode string          `json:"episode"`
}

type scheduleDay struct {
	Day   int            `json:"day"`
	Name  string         `json:"name"`
	Items []scheduleItem `json:"items"`
}

func buildSchedule(snap *Snapshot, list []alScheduleItem) []scheduleDay {
	days := make([]scheduleDay, 7)
	for i := range days {
		days[i] = scheduleDay{Day: i + 1, Name: dayNames[i], Items: []scheduleItem{}}
	}
	for _, item := range list {
		day := item.Release.PublishDay.Value
		if day < 1 || day > 7 {
			continue
		}
		index, ok := snap.ByAlias[item.Release.Alias]
		if !ok {
			continue
		}
		episode := ""
		if item.NextEpisode > 0 {
			episode = strconv.Itoa(int(item.NextEpisode))
		} else if item.Published != nil {
			episode = strconv.FormatFloat(item.Published.Ordinal, 'f', -1, 64)
		}
		days[day-1].Items = append(days[day-1].Items, scheduleItem{Title: snap.Titles[index], Episode: episode})
	}
	return days
}

func (a *App) handleSchedule(w http.ResponseWriter, r *http.Request) {
	snap := a.snap.Load()
	if snap == nil {
		writeError(w, r, http.StatusServiceUnavailable, "catalog is loading")
		return
	}
	a.schedule.mu.Lock()
	cached := a.schedule.body
	fresh := cached != nil && a.schedule.snapID == snap.ID && time.Since(a.schedule.fetched) < 5*time.Minute
	a.schedule.mu.Unlock()
	if fresh {
		serveBlob(w, r, cached)
		return
	}
	list, err := fetchAnilibriaSchedule()
	if err != nil {
		if cached != nil {
			serveBlob(w, r, cached)
			return
		}
		writeJSON(w, r, http.StatusOK, buildSchedule(snap, nil))
		return
	}
	body, _ := json.Marshal(buildSchedule(snap, list))
	blob := newBlob(body, `"`+snap.ID+`-sch`+hashString(string(body))+`"`)
	a.schedule.mu.Lock()
	a.schedule.body = blob
	a.schedule.snapID = snap.ID
	a.schedule.fetched = time.Now()
	a.schedule.mu.Unlock()
	serveBlob(w, r, blob)
}
