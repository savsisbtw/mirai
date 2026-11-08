package main

import (
	"encoding/json"
	"math/rand"
	"net/url"
	"strconv"
	"testing"
)

type catalogResult struct {
	Items []TitleOut `json:"items"`
	Total int        `json:"total"`
	Page  int        `json:"page"`
	Pages int        `json:"pages"`
}

func runCatalog(t testing.TB, s *Snapshot, query string) catalogResult {
	values, err := url.ParseQuery(query)
	if err != nil {
		t.Fatal(err)
	}
	var result catalogResult
	if err := json.Unmarshal(s.catalog(parseQuery(values)), &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func ids(items []TitleOut) []string {
	var result []string
	for _, item := range items {
		result = append(result, item.ID)
	}
	return result
}

func sampleEntries() []Entry {
	items := []Item{
		{Source: "anilibria", ID: 1, Alias: "one-piece", Shiki: 21, MAL: 21, Name: "Ван-Пис", NameEn: "One Piece", Year: 1999, Type: "ТВ", Status: "ongoing", Rating: 8.8, Poster: "https://aniliberty.top/a.webp", Genres: []string{"Экшен", "Приключения"}, PopRaw: 5000, Fresh: 300},
		{Source: "anilibria", ID: 2, Alias: "naruto", Shiki: 20, MAL: 20, Name: "Наруто", NameEn: "Naruto", Year: 2002, Type: "ТВ", Status: "released", Rating: 8.0, Genres: []string{"Экшен"}, PopRaw: 4000, Fresh: 200},
		{Source: "anilibria", ID: 3, Alias: "chernyj-klever", Name: "Чёрный клевер", NameEn: "Black Clover", Year: 2017, Type: "ТВ", Status: "released", Rating: 8.1, Genres: []string{"Экшен", "Фэнтези"}, PopRaw: 3000, Fresh: 100},
		{Source: "anilibria", ID: 4, Alias: "film-x", Name: "Фильм про ёжика", Year: 2020, Type: "Фильм", Status: "released", Rating: 0, Genres: []string{"Комедия"}, PopRaw: 10},
		{Source: "yummy", ID: 100, Shiki: 21, Name: "Ван-Пис (ТВ)", Year: 1999, Rating: 8.9, Poster: "https://imgproxy.yani.tv/y.webp", Desc: "yummy desc", PopRaw: 1000000, Genres: []string{"Драма"}},
		{Source: "yummy", ID: 101, MAL: 20, Name: "Наруто ТВ-1", Rating: 7.9, PopRaw: 900000},
		{Source: "yummy", ID: 102, Alias: "chernyj-klever", Name: "Чёрный клевер ТВ", Rating: 8.2, PopRaw: 800000},
		{Source: "yummy", ID: 103, Shiki: 777, Name: "Только на юмми", NameEn: "Only Yummy", Year: 2024, Type: "ONA", Status: "announce", Rating: 6.5, PopRaw: 700000, Genres: []string{"Экшен", "Драма"}},
		{Source: "yummy", ID: 104, Shiki: 21, Name: "Ван-Пис дубль", Year: 1999, PopRaw: 100},
	}
	var al, ym []Item
	for _, item := range items {
		if item.Source == "anilibria" {
			al = append(al, item)
		} else {
			ym = append(ym, item)
		}
	}
	return mergeItems(al, ym)
}

var syllables = []string{"ка", "на", "ру", "то", "ми", "со", "ра", "ю", "ки", "до", "ва", "ли", "ше", "ко", "ни", "ту", "ме", "ха", "сэ", "ри"}
var genreNames = []string{"Экшен", "Комедия", "Драма", "Фэнтези", "Романтика", "Школа", "Меха", "Спорт", "Ужасы", "Детектив", "Магия", "Космос", "Музыка", "Игры", "Исэкай", "Сёнэн", "Сэйнэн", "Триллер", "Фантастика", "Приключения"}
var typeNames = []string{"ТВ", "Фильм", "OVA", "ONA", "Спешл"}
var statusNames = []string{"released", "released", "released", "ongoing", "announce"}

func syntheticName(r *rand.Rand) string {
	count := 2 + r.Intn(3)
	name := ""
	for i := 0; i < count; i++ {
		name += syllables[r.Intn(len(syllables))]
	}
	return name
}

func syntheticEntries(n int) []Entry {
	r := rand.New(rand.NewSource(42))
	entries := make([]Entry, n)
	for i := range entries {
		genres := []string{}
		for g := 0; g < 1+r.Intn(4); g++ {
			name := genreNames[r.Intn(len(genreNames))]
			if !hasString(genres, name) {
				genres = append(genres, name)
			}
		}
		entries[i] = Entry{
			ID:      "s" + strconv.Itoa(i+1),
			Name:    syntheticName(r) + " " + syntheticName(r),
			NameEn:  syntheticName(r),
			Alt:     []string{syntheticName(r)},
			Year:    1990 + r.Intn(37),
			Type:    typeNames[r.Intn(len(typeNames))],
			Status:  statusNames[r.Intn(len(statusNames))],
			Rating:  float64(r.Intn(1000)) / 100,
			Poster:  "https://imgproxy.yani.tv/p/" + strconv.Itoa(i) + ".webp",
			Desc:    "описание",
			Genres:  genres,
			Sources: []string{"yummy"},
			Fresh:   int64(r.Intn(100000)),
			Pop:     r.Float64(),
		}
	}
	return entries
}
