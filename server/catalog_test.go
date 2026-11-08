package main

import (
	"reflect"
	"strconv"
	"testing"
)

func TestSearchYoE(t *testing.T) {
	s := buildSnapshot(sampleEntries())
	for _, q := range []string{"черный клевер", "чёрный клевер", "ЧЁРНЫЙ", "чЕрНыЙ"} {
		result := runCatalog(t, s, "q="+q)
		if result.Total != 1 || result.Items[0].ID != "achernyj-klever" {
			t.Fatalf("%q: %+v", q, ids(result.Items))
		}
	}
	result := runCatalog(t, s, "q=ежика")
	if result.Total != 1 {
		t.Fatalf("ёжика: %d", result.Total)
	}
}

func TestSearchPunctuationAndEnglish(t *testing.T) {
	s := buildSnapshot(sampleEntries())
	if result := runCatalog(t, s, "q=ван пис"); result.Total == 0 || result.Items[0].ID != "s21" {
		t.Fatalf("ван пис: %+v", ids(result.Items))
	}
	if result := runCatalog(t, s, "q=black clover"); result.Total != 1 {
		t.Fatalf("english: %d", result.Total)
	}
}

func TestSearchRanking(t *testing.T) {
	entries := []Entry{
		{ID: "a", Name: "Магическая битва", Pop: 0.9},
		{ID: "b", Name: "Тёмная магия", Pop: 0.8},
		{ID: "c", Name: "Магия", Pop: 0.1},
		{ID: "d", Name: "Немагический", Pop: 1.0},
	}
	s := buildSnapshot(entries)
	result := runCatalog(t, s, "q=маг")
	got := ids(result.Items)
	want := []string{"a", "c", "b", "d"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestSearchNoMatch(t *testing.T) {
	s := buildSnapshot(sampleEntries())
	result := runCatalog(t, s, "q=zzzzzz")
	if result.Total != 0 || len(result.Items) != 0 || result.Pages != 1 {
		t.Fatalf("%+v", result)
	}
}

func TestFilterGenreAnd(t *testing.T) {
	s := buildSnapshot(sampleEntries())
	result := runCatalog(t, s, "genre=Экшен,Фэнтези")
	if result.Total != 1 || result.Items[0].ID != "achernyj-klever" {
		t.Fatalf("%v", ids(result.Items))
	}
	if result := runCatalog(t, s, "genre=Экшен"); result.Total != 4 {
		t.Fatalf("экшен: %d", result.Total)
	}
	if result := runCatalog(t, s, "genre=Несуществующий"); result.Total != 0 {
		t.Fatalf("unknown genre: %d", result.Total)
	}
}

func TestFilterYearTypeStatus(t *testing.T) {
	s := buildSnapshot(sampleEntries())
	if result := runCatalog(t, s, "year=2002"); result.Total != 1 || result.Items[0].ID != "s20" {
		t.Fatalf("year: %v", ids(result.Items))
	}
	if result := runCatalog(t, s, "year=2017-2024"); result.Total != 3 {
		t.Fatalf("year range: %v", ids(result.Items))
	}
	if result := runCatalog(t, s, "type=Фильм"); result.Total != 1 || result.Items[0].Name != "Фильм про ёжика" {
		t.Fatalf("type: %v", ids(result.Items))
	}
	if result := runCatalog(t, s, "status=ongoing"); result.Total != 1 || result.Items[0].ID != "s21" {
		t.Fatalf("status: %v", ids(result.Items))
	}
	if result := runCatalog(t, s, "status=announce&type=ONA"); result.Total != 1 {
		t.Fatalf("combo: %v", ids(result.Items))
	}
	if result := runCatalog(t, s, "status=wrong"); result.Total != 0 {
		t.Fatalf("bad status: %d", result.Total)
	}
}

func TestSorts(t *testing.T) {
	s := buildSnapshot(sampleEntries())
	byName := runCatalog(t, s, "sort=name")
	for i := 1; i < len(byName.Items); i++ {
		if norm(byName.Items[i-1].Name) > norm(byName.Items[i].Name) {
			t.Fatalf("name order broken: %q > %q", byName.Items[i-1].Name, byName.Items[i].Name)
		}
	}
	byRating := runCatalog(t, s, "sort=rating")
	for i := 1; i < len(byRating.Items); i++ {
		if byRating.Items[i-1].Rating < byRating.Items[i].Rating {
			t.Fatal("rating order broken")
		}
	}
	byNew := runCatalog(t, s, "sort=new")
	if byNew.Items[0].Year != 2024 {
		t.Fatalf("new first: %+v", byNew.Items[0])
	}
	popular := runCatalog(t, s, "")
	if popular.Items[0].ID != "s21" {
		t.Fatalf("popular first: %s", popular.Items[0].ID)
	}
}

func TestPagination(t *testing.T) {
	s := buildSnapshot(syntheticEntries(250))
	first := runCatalog(t, s, "limit=100")
	if first.Total != 250 || first.Pages != 3 || len(first.Items) != 100 {
		t.Fatalf("%d %d %d", first.Total, first.Pages, len(first.Items))
	}
	last := runCatalog(t, s, "limit=100&page=3")
	if len(last.Items) != 50 || last.Page != 3 {
		t.Fatalf("last: %d", len(last.Items))
	}
	beyond := runCatalog(t, s, "limit=100&page=9")
	if len(beyond.Items) != 0 || beyond.Total != 250 {
		t.Fatalf("beyond: %d", len(beyond.Items))
	}
	capped := runCatalog(t, s, "limit=5000")
	if len(capped.Items) != 100 {
		t.Fatalf("limit cap: %d", len(capped.Items))
	}
	seen := map[string]bool{}
	for page := 1; page <= 3; page++ {
		result := runCatalog(t, s, "limit=100&page="+strconv.Itoa(page))
		for _, item := range result.Items {
			if seen[item.ID] {
				t.Fatalf("duplicate across pages: %s", item.ID)
			}
			seen[item.ID] = true
		}
	}
}

func TestFiltersAndHomeBlobs(t *testing.T) {
	s := buildSnapshot(sampleEntries())
	if s.Filters == nil || s.Home == nil || s.Filters.ETag == "" {
		t.Fatal("blobs missing")
	}
	if len(s.GenreBit) == 0 || len(s.TypeID) == 0 {
		t.Fatal("indexes missing")
	}
}

func TestCatalogBlobCacheAndETag(t *testing.T) {
	s := buildSnapshot(sampleEntries())
	values := parseQuery(map[string][]string{"q": {"наруто"}})
	first := s.catalogBlob(values)
	second := s.catalogBlob(values)
	if first != second {
		t.Fatal("second call must come from cache")
	}
	other := s.catalogBlob(parseQuery(map[string][]string{"q": {"клевер"}}))
	if other.ETag == first.ETag {
		t.Fatal("etag must differ per query")
	}
}
