package main

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

const alBase = "https://aniliberty.top"

type alRelease struct {
	ID    int    `json:"id"`
	Alias string `json:"alias"`
	Year  int    `json:"year"`
	Type  struct {
		Description string `json:"description"`
	} `json:"type"`
	Name struct {
		Main        string  `json:"main"`
		English     string  `json:"english"`
		Alternative *string `json:"alternative"`
	} `json:"name"`
	Season struct {
		Description string `json:"description"`
	} `json:"season"`
	Shikimori struct {
		ID     flexInt   `json:"id"`
		Rating flexFloat `json:"rating"`
	} `json:"shikimori"`
	Mal struct {
		ID     flexInt   `json:"id"`
		Rating flexFloat `json:"rating"`
	} `json:"mal"`
	Rating struct {
		Average flexFloat `json:"average"`
	} `json:"rating"`
	Poster struct {
		Src       string `json:"src"`
		Optimized struct {
			Src string `json:"src"`
		} `json:"optimized"`
	} `json:"poster"`
	FreshAt   string `json:"fresh_at"`
	IsOngoing bool   `json:"is_ongoing"`
	AgeRating struct {
		Label string `json:"label"`
	} `json:"age_rating"`
	Description   string  `json:"description"`
	EpisodesTotal flexInt `json:"episodes_total"`
	Genres        []struct {
		Name string `json:"name"`
	} `json:"genres"`
	Favorites flexFloat `json:"added_in_users_favorites"`
}

type alPage struct {
	Data []alRelease `json:"data"`
	Meta struct {
		Pagination struct {
			TotalPages int `json:"total_pages"`
		} `json:"pagination"`
	} `json:"meta"`
}

func alURL(path string) string {
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "http") {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return alBase + path
}

func convertAL(r alRelease) Item {
	item := Item{
		Source: "anilibria",
		ID:     r.ID,
		Alias:  r.Alias,
		Shiki:  int(r.Shikimori.ID),
		MAL:    int(r.Mal.ID),
		Name:   strings.TrimSpace(r.Name.Main),
		NameEn: strings.TrimSpace(r.Name.English),
		Year:   r.Year,
		Type:   canonType(r.Type.Description),
		Season: r.Season.Description,
		Age:    r.AgeRating.Label,
		Desc:   strings.TrimSpace(r.Description),
		PopRaw: float64(r.Favorites),
	}
	if r.Name.Alternative != nil {
		item.Alt = splitAlt(*r.Name.Alternative)
	}
	if r.IsOngoing {
		item.Status = "ongoing"
	} else {
		item.Status = "released"
	}
	item.Episodes = int(r.EpisodesTotal)
	item.Rating = float64(r.Shikimori.Rating)
	if item.Rating == 0 {
		item.Rating = float64(r.Mal.Rating)
	}
	if item.Rating == 0 {
		item.Rating = float64(r.Rating.Average)
	}
	if r.Poster.Optimized.Src != "" {
		item.Poster = alURL(r.Poster.Optimized.Src)
	} else {
		item.Poster = alURL(r.Poster.Src)
	}
	item.Backdrop = alURL(r.Poster.Src)
	if item.Backdrop == "" {
		item.Backdrop = item.Poster
	}
	if t, err := time.Parse(time.RFC3339, r.FreshAt); err == nil {
		item.Fresh = t.Unix()
	}
	for _, g := range r.Genres {
		name := canonGenre(g.Name)
		if name != "" && !hasString(item.Genres, name) {
			item.Genres = append(item.Genres, name)
		}
	}
	return item
}

func crawlAnilibria(report func([]Item)) ([]Item, error) {
	endpoint := alBase + "/api/v1/anime/catalog/releases"
	var first alPage
	body := `{"page":1,"limit":50,"f":{"sorting":"FRESH_AT_DESC"}}`
	if err := fetchJSON("POST", endpoint, []byte(body), &first); err != nil {
		return nil, err
	}
	var mu sync.Mutex
	var items []Item
	seen := map[int]bool{}
	add := func(page alPage) {
		mu.Lock()
		defer mu.Unlock()
		for _, r := range page.Data {
			if seen[r.ID] || r.Name.Main == "" {
				continue
			}
			seen[r.ID] = true
			items = append(items, convertAL(r))
		}
		report(items)
	}
	add(first)
	totalPages := first.Meta.Pagination.TotalPages
	failed := 0
	runJobs(totalPages-1, 4, func(i int) {
		page := i + 2
		payload := `{"page":` + strconv.Itoa(page) + `,"limit":50,"f":{"sorting":"FRESH_AT_DESC"}}`
		var result alPage
		if err := fetchJSON("POST", endpoint, []byte(payload), &result); err != nil {
			mu.Lock()
			failed++
			mu.Unlock()
			return
		}
		add(result)
	})
	mu.Lock()
	defer mu.Unlock()
	if failed > 0 {
		return items, fmt.Errorf("anilibria: %d pages failed", failed)
	}
	return items, nil
}

type alEpisode struct {
	Name    string  `json:"name"`
	Ordinal float64 `json:"ordinal"`
	Opening struct {
		Start flexInt `json:"start"`
		Stop  flexInt `json:"stop"`
	} `json:"opening"`
	Ending struct {
		Start flexInt `json:"start"`
		Stop  flexInt `json:"stop"`
	} `json:"ending"`
	HLS480   string  `json:"hls_480"`
	HLS720   string  `json:"hls_720"`
	HLS1080  string  `json:"hls_1080"`
	Duration flexInt `json:"duration"`
}

type alReleaseDetail struct {
	Episodes []alEpisode `json:"episodes"`
}

func fetchAnilibriaVoice(releaseID int) (*Voice, error) {
	var detail alReleaseDetail
	err := fetchJSON("GET", alBase+"/api/v1/anime/releases/"+strconv.Itoa(releaseID), nil, &detail)
	if err != nil {
		return nil, err
	}
	voice := Voice{ID: "al", Label: "AniLibria", Source: "anilibria", Kind: "hls", Player: "", Episodes: []VoiceEpisode{}}
	for _, e := range detail.Episodes {
		hls := map[string]string{}
		if e.HLS480 != "" {
			hls["480"] = e.HLS480
		}
		if e.HLS720 != "" {
			hls["720"] = e.HLS720
		}
		if e.HLS1080 != "" {
			hls["1080"] = e.HLS1080
		}
		if len(hls) == 0 {
			continue
		}
		ep := VoiceEpisode{
			N:        strconv.FormatFloat(e.Ordinal, 'f', -1, 64),
			Title:    e.Name,
			HLS:      hls,
			Duration: int(e.Duration),
		}
		if e.Opening.Stop > 0 {
			ep.Opening = &Skip{Start: int(e.Opening.Start), Stop: int(e.Opening.Stop)}
		}
		if e.Ending.Stop > 0 {
			ep.Ending = &Skip{Start: int(e.Ending.Start), Stop: int(e.Ending.Stop)}
		}
		voice.Episodes = append(voice.Episodes, ep)
	}
	if len(voice.Episodes) == 0 {
		return nil, nil
	}
	return &voice, nil
}

type alScheduleItem struct {
	Release struct {
		Alias      string `json:"alias"`
		PublishDay struct {
			Value int `json:"value"`
		} `json:"publish_day"`
	} `json:"release"`
	NextEpisode flexInt `json:"next_release_episode_number"`
	Published   *struct {
		Ordinal float64 `json:"ordinal"`
	} `json:"published_release_episode"`
}

func fetchAnilibriaSchedule() ([]alScheduleItem, error) {
	var list []alScheduleItem
	err := fetchJSON("GET", alBase+"/api/v1/anime/schedule/week", nil, &list)
	return list, err
}
