package main

import "testing"

func findEntry(entries []Entry, id string) *Entry {
	for i := range entries {
		if entries[i].ID == id {
			return &entries[i]
		}
	}
	return nil
}

func TestMergeByShikimori(t *testing.T) {
	entries := sampleEntries()
	e := findEntry(entries, "s21")
	if e == nil {
		t.Fatal("s21 not found")
	}
	if len(e.Sources) != 2 || e.YID != 100 || e.ALID != 1 {
		t.Fatalf("not merged: %+v", e.Sources)
	}
	if e.Name != "Ван-Пис" || e.Poster != "https://aniliberty.top/a.webp" {
		t.Fatalf("anilibria must win: %q %q", e.Name, e.Poster)
	}
	if e.Desc != "yummy desc" {
		t.Fatalf("empty description must be filled from yummy: %q", e.Desc)
	}
	if e.Rating != 8.8 {
		t.Fatalf("rating: %v", e.Rating)
	}
}

func TestMergeByMALAndAlias(t *testing.T) {
	entries := sampleEntries()
	naruto := findEntry(entries, "s20")
	if naruto == nil || naruto.YID != 101 {
		t.Fatalf("mal merge failed: %+v", naruto)
	}
	clover := findEntry(entries, "achernyj-klever")
	if clover == nil || clover.YID != 102 || len(clover.Sources) != 2 {
		t.Fatalf("alias merge failed: %+v", clover)
	}
}

func TestMergeRatingFromAnyNonZero(t *testing.T) {
	al := []Item{{Source: "anilibria", ID: 1, Alias: "a", Shiki: 5, Name: "A", Rating: 0}}
	ym := []Item{{Source: "yummy", ID: 2, Shiki: 5, Name: "A2", Rating: 7.3}}
	entries := mergeItems(al, ym)
	if len(entries) != 1 || entries[0].Rating != 7.3 {
		t.Fatalf("rating: %+v", entries)
	}
}

func TestMergeKeepsUnmatchedAndSecondDuplicate(t *testing.T) {
	entries := sampleEntries()
	if len(entries) != 6 {
		t.Fatalf("want 6 entries, got %d", len(entries))
	}
	only := findEntry(entries, "s777")
	if only == nil || len(only.Sources) != 1 || only.Sources[0] != "yummy" {
		t.Fatalf("yummy only: %+v", only)
	}
	second := findEntry(entries, "y104")
	if second == nil {
		t.Fatal("second yummy duplicate must stay a separate title with unique id")
	}
}

func TestMergeIDsAreUnique(t *testing.T) {
	al := []Item{
		{Source: "anilibria", ID: 1, Alias: "x1", Shiki: 9, Name: "X1"},
		{Source: "anilibria", ID: 2, Alias: "x2", Shiki: 9, Name: "X2"},
		{Source: "anilibria", ID: 3, Alias: "x3", Name: "X3"},
	}
	entries := mergeItems(al, nil)
	seen := map[string]bool{}
	for _, e := range entries {
		if seen[e.ID] {
			t.Fatalf("duplicate id %s", e.ID)
		}
		seen[e.ID] = true
	}
	if entries[0].ID != "s9" || entries[1].ID != "ax2" || entries[2].ID != "ax3" {
		t.Fatalf("ids: %v %v %v", entries[0].ID, entries[1].ID, entries[2].ID)
	}
}

func TestPercentilesIgnoreZero(t *testing.T) {
	items := []Item{{PopRaw: 10}, {PopRaw: 0}, {PopRaw: 20}}
	pop := percentiles(items)
	if pop[2] != 1 || pop[0] <= pop[1] || pop[1] != 0 {
		t.Fatalf("pop: %v", pop)
	}
}
