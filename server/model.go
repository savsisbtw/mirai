package main

type Item struct {
	Source   string   `json:"source"`
	ID       int      `json:"id"`
	Alias    string   `json:"alias,omitempty"`
	Shiki    int      `json:"shiki,omitempty"`
	MAL      int      `json:"mal,omitempty"`
	Name     string   `json:"name"`
	NameEn   string   `json:"name_en,omitempty"`
	Alt      []string `json:"alt,omitempty"`
	Year     int      `json:"year,omitempty"`
	Type     string   `json:"type,omitempty"`
	Status   string   `json:"status,omitempty"`
	Season   string   `json:"season,omitempty"`
	Age      string   `json:"age,omitempty"`
	Rating   float64  `json:"rating,omitempty"`
	Poster   string   `json:"poster,omitempty"`
	Backdrop string   `json:"backdrop,omitempty"`
	Desc     string   `json:"desc,omitempty"`
	Episodes int      `json:"episodes,omitempty"`
	Genres   []string `json:"genres,omitempty"`
	Fresh    int64    `json:"fresh,omitempty"`
	PopRaw   float64  `json:"pop,omitempty"`
}

type Entry struct {
	ID       string
	ALID     int
	ALAlias  string
	YID      int
	Shiki    int
	MAL      int
	Name     string
	NameEn   string
	Alt      []string
	Year     int
	Type     string
	Status   string
	Season   string
	Age      string
	Rating   float64
	Poster   string
	Backdrop string
	Desc     string
	Episodes int
	Genres   []string
	Sources  []string
	Fresh    int64
	Pop      float64
}

type TitleOut struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	NameEn   string   `json:"name_en"`
	Year     int      `json:"year"`
	Type     string   `json:"type"`
	Status   string   `json:"status"`
	Season   string   `json:"season"`
	Age      string   `json:"age"`
	Rating   float64  `json:"rating"`
	Poster   string   `json:"poster"`
	Episodes *int     `json:"episodes"`
	Genres   []string `json:"genres"`
	Sources  []string `json:"sources"`
}

type TitleFull struct {
	TitleOut
	Description string `json:"description"`
	Backdrop    string `json:"backdrop"`
}

type Skip struct {
	Start int `json:"start"`
	Stop  int `json:"stop"`
}

type VoiceEpisode struct {
	N        string            `json:"n"`
	Title    string            `json:"title,omitempty"`
	HLS      map[string]string `json:"hls,omitempty"`
	Opening  *Skip             `json:"opening,omitempty"`
	Ending   *Skip             `json:"ending,omitempty"`
	Duration int               `json:"duration,omitempty"`
	URL      string            `json:"url,omitempty"`
}

type Voice struct {
	ID       string         `json:"id"`
	Label    string         `json:"label"`
	Source   string         `json:"source"`
	Kind     string         `json:"kind"`
	Player   string         `json:"player"`
	Episodes []VoiceEpisode `json:"episodes"`
}

type TitleDetail struct {
	TitleFull
	NameAlt []string `json:"name_alt"`
	Voices  []Voice  `json:"voices"`
}
