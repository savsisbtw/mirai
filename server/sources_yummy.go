package main

import (
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const ymBase = "https://api.yani.tv"

type ymAnime struct {
	AnimeID     flexInt `json:"anime_id"`
	Title       string  `json:"title"`
	Year        flexInt `json:"year"`
	Description string  `json:"description"`
	Poster      struct {
		Big      string `json:"big"`
		Medium   string `json:"medium"`
		Fullsize string `json:"fullsize"`
		Huge     string `json:"huge"`
	} `json:"poster"`
	Type struct {
		Short string `json:"shortname"`
		Name  string `json:"name"`
	} `json:"type"`
	Season      flexInt `json:"season"`
	AnimeStatus struct {
		Alias string `json:"alias"`
	} `json:"anime_status"`
	MinAge struct {
		Title string `json:"title"`
	} `json:"min_age"`
	RemoteIDs struct {
		Shikimori flexInt `json:"shikimori_id"`
		MAL       flexInt `json:"myanimelist_id"`
		Alias     string  `json:"anilibria_alias"`
	} `json:"remote_ids"`
	Genres []struct {
		Title string `json:"title"`
	} `json:"genres"`
	Rating struct {
		Average   flexFloat `json:"average"`
		Shikimori flexFloat `json:"shikimori_rating"`
		MAL       flexFloat `json:"myanimelist_rating"`
	} `json:"rating"`
	Views flexFloat `json:"views"`
}

type ymList struct {
	Response []ymAnime `json:"response"`
}

var ymSeasons = []string{"", "Зима", "Весна", "Лето", "Осень"}

var ymAges = map[string]string{
	"g":     "0+",
	"pg":    "6+",
	"pg-13": "12+",
	"r-17+": "16+",
	"r+":    "18+",
	"rx":    "18+",
}

func ymImage(path string) string {
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "//") {
		return "https:" + path
	}
	return path
}

func convertYummy(a ymAnime) Item {
	item := Item{
		Source: "yummy",
		ID:     int(a.AnimeID),
		Alias:  a.RemoteIDs.Alias,
		Shiki:  int(a.RemoteIDs.Shikimori),
		MAL:    int(a.RemoteIDs.MAL),
		Name:   strings.TrimSpace(a.Title),
		Year:   int(a.Year),
		Status: a.AnimeStatus.Alias,
		Desc:   strings.TrimSpace(a.Description),
		PopRaw: float64(a.Views),
	}
	item.Type = canonType(a.Type.Short)
	if item.Type == "" {
		item.Type = canonType(a.Type.Name)
	}
	if item.Status != "ongoing" && item.Status != "announce" {
		item.Status = "released"
	}
	season := int(a.Season)
	if season >= 1 && season <= 4 {
		item.Season = ymSeasons[season]
	}
	age := strings.ToLower(strings.TrimSpace(a.MinAge.Title))
	if label, ok := ymAges[age]; ok {
		item.Age = label
	} else if age != "unknown" {
		item.Age = a.MinAge.Title
	}
	item.Rating = float64(a.Rating.Shikimori)
	if item.Rating == 0 {
		item.Rating = float64(a.Rating.MAL)
	}
	if item.Rating == 0 {
		item.Rating = float64(a.Rating.Average)
	}
	item.Poster = ymImage(a.Poster.Big)
	if item.Poster == "" {
		item.Poster = ymImage(a.Poster.Medium)
	}
	if item.Poster == "" {
		item.Poster = ymImage(a.Poster.Fullsize)
	}
	item.Backdrop = ymImage(a.Poster.Fullsize)
	if item.Backdrop == "" {
		item.Backdrop = item.Poster
	}
	for _, g := range a.Genres {
		name := canonGenre(g.Title)
		if name != "" && !hasString(item.Genres, name) {
			item.Genres = append(item.Genres, name)
		}
	}
	return item
}

func ymYearRange(job int) (int, int) {
	if job == 0 {
		return 1900, 1959
	}
	year := 1959 + job
	return year, year
}

func ymJobCount() int {
	lastYear := time.Now().Year() + 2
	return lastYear - 1959 + 1
}

func fetchYummyYear(from, to int, ascending bool) ([]ymAnime, error) {
	var result []ymAnime
	direction := "false"
	if ascending {
		direction = "true"
	}
	for offset := 0; offset < 1000; offset += 100 {
		target := ymBase + "/anime?limit=100&offset=" + strconv.Itoa(offset) +
			"&from_year=" + strconv.Itoa(from) + "&to_year=" + strconv.Itoa(to) +
			"&sort=views&sort_forward=" + direction
		var page ymList
		if err := fetchJSON("GET", target, nil, &page); err != nil {
			return result, err
		}
		result = append(result, page.Response...)
		if len(page.Response) < 100 {
			break
		}
	}
	return result, nil
}

type crawlError struct {
	source string
	failed int
}

func (e *crawlError) Error() string {
	return e.source + ": " + strconv.Itoa(e.failed) + " jobs failed"
}

func crawlYummy(report func([]Item)) ([]Item, error) {
	var mu sync.Mutex
	var items []Item
	seen := map[int]bool{}
	failed := 0
	add := func(list []ymAnime) {
		mu.Lock()
		defer mu.Unlock()
		for _, a := range list {
			id := int(a.AnimeID)
			if id == 0 || seen[id] || strings.TrimSpace(a.Title) == "" {
				continue
			}
			seen[id] = true
			items = append(items, convertYummy(a))
		}
		report(items)
	}
	runJobs(ymJobCount(), 6, func(job int) {
		from, to := ymYearRange(job)
		list, err := fetchYummyYear(from, to, false)
		add(list)
		if err != nil {
			mu.Lock()
			failed++
			mu.Unlock()
			return
		}
		if len(list) >= 1000 {
			more, err := fetchYummyYear(from, to, true)
			add(more)
			if err != nil {
				mu.Lock()
				failed++
				mu.Unlock()
			}
		}
	})
	mu.Lock()
	defer mu.Unlock()
	if failed > 0 {
		return items, &crawlError{source: "yummy", failed: failed}
	}
	return items, nil
}

type ymVideo struct {
	Number    flexString `json:"number"`
	IframeURL string     `json:"iframe_url"`
	Data      struct {
		Player  string `json:"player"`
		Dubbing string `json:"dubbing"`
	} `json:"data"`
}

type ymVideos struct {
	Response []ymVideo `json:"response"`
}

var playerHosts = []string{"kodikplayer.com", "yani.tv", "yummyani.me", "aksor.tv"}

func allowedPlayerHost(host string) bool {
	host = strings.ToLower(host)
	for _, allowed := range playerHosts {
		if host == allowed || strings.HasSuffix(host, "."+allowed) {
			return true
		}
	}
	return false
}

func cleanPlayerURL(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "//") {
		raw = "https:" + raw
	}
	if !strings.HasPrefix(raw, "https://") {
		return "", false
	}
	parsed, err := url.Parse(raw)
	if err != nil || !allowedPlayerHost(parsed.Hostname()) {
		return "", false
	}
	return raw, true
}

func episodeLess(a, b string) bool {
	x, errX := strconv.ParseFloat(a, 64)
	y, errY := strconv.ParseFloat(b, 64)
	if errX == nil && errY == nil {
		return x < y
	}
	return a < b
}

func groupYummyVideos(videos []ymVideo) []Voice {
	var voices []*Voice
	index := map[string]*Voice{}
	seenEpisode := map[string]bool{}
	for _, v := range videos {
		link, ok := cleanPlayerURL(v.IframeURL)
		if !ok {
			continue
		}
		dubbing := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(v.Data.Dubbing), "Озвучка "))
		player := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(v.Data.Player), "Плеер "))
		if dubbing == "" {
			dubbing = "Без названия"
		}
		number := strings.TrimSpace(string(v.Number))
		key := dubbing + "|" + player
		voice := index[key]
		if voice == nil {
			voice = &Voice{
				Label:    dubbing,
				Source:   "yummy",
				Kind:     "embed",
				Player:   player,
				Episodes: []VoiceEpisode{},
			}
			index[key] = voice
			voices = append(voices, voice)
		}
		if seenEpisode[key+"|"+number] {
			continue
		}
		seenEpisode[key+"|"+number] = true
		voice.Episodes = append(voice.Episodes, VoiceEpisode{N: number, URL: link})
	}
	for _, voice := range voices {
		episodes := voice.Episodes
		sort.SliceStable(episodes, func(i, j int) bool {
			return episodeLess(episodes[i].N, episodes[j].N)
		})
	}
	sort.SliceStable(voices, func(i, j int) bool {
		if len(voices[i].Episodes) != len(voices[j].Episodes) {
			return len(voices[i].Episodes) > len(voices[j].Episodes)
		}
		return voices[i].Label < voices[j].Label
	})
	usedIDs := map[string]int{}
	result := make([]Voice, 0, len(voices))
	for _, voice := range voices {
		id := "ym-" + slug(voice.Label) + "-" + slug(voice.Player)
		usedIDs[id]++
		if usedIDs[id] > 1 {
			id = id + "-" + strconv.Itoa(usedIDs[id])
		}
		voice.ID = id
		result = append(result, *voice)
	}
	return result
}

func fetchYummyVoices(yummyID int) ([]Voice, error) {
	var list ymVideos
	err := fetchJSON("GET", ymBase+"/anime/"+strconv.Itoa(yummyID)+"/videos", nil, &list)
	if err != nil {
		return nil, err
	}
	return groupYummyVideos(list.Response), nil
}
