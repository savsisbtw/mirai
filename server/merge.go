package main

import (
	"sort"
	"strconv"
)

func percentiles(items []Item) []float64 {
	result := make([]float64, len(items))
	order := make([]int, len(items))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		return items[order[a]].PopRaw > items[order[b]].PopRaw
	})
	total := float64(len(items))
	for rank, i := range order {
		if items[i].PopRaw <= 0 {
			result[i] = 0
			continue
		}
		result[i] = 1 - float64(rank)/total
	}
	return result
}

func entryFromItem(item Item, pop float64) Entry {
	e := Entry{
		Shiki:    item.Shiki,
		MAL:      item.MAL,
		Name:     item.Name,
		NameEn:   item.NameEn,
		Alt:      append([]string(nil), item.Alt...),
		Year:     item.Year,
		Type:     item.Type,
		Status:   item.Status,
		Season:   item.Season,
		Age:      item.Age,
		Rating:   item.Rating,
		Poster:   item.Poster,
		Backdrop: item.Backdrop,
		Desc:     item.Desc,
		Episodes: item.Episodes,
		Genres:   append([]string(nil), item.Genres...),
		Fresh:    item.Fresh,
		Pop:      pop,
	}
	if item.Source == "anilibria" {
		e.ALID = item.ID
		e.ALAlias = item.Alias
		e.Sources = []string{"anilibria"}
	} else {
		e.YID = item.ID
		e.Sources = []string{"yummy"}
	}
	return e
}

func addAlt(e *Entry, name string) {
	if name == "" {
		return
	}
	key := norm(name)
	if key == norm(e.Name) || key == norm(e.NameEn) {
		return
	}
	for _, alt := range e.Alt {
		if norm(alt) == key {
			return
		}
	}
	e.Alt = append(e.Alt, name)
}

func mergeYummyInto(e *Entry, item Item, pop float64) {
	e.YID = item.ID
	if !hasString(e.Sources, "yummy") {
		e.Sources = append(e.Sources, "yummy")
	}
	if e.Rating == 0 {
		e.Rating = item.Rating
	}
	if e.Shiki == 0 {
		e.Shiki = item.Shiki
	}
	if e.MAL == 0 {
		e.MAL = item.MAL
	}
	if e.Poster == "" {
		e.Poster = item.Poster
	}
	if e.Backdrop == "" {
		e.Backdrop = item.Backdrop
	}
	if e.Desc == "" {
		e.Desc = item.Desc
	}
	if e.Year == 0 {
		e.Year = item.Year
	}
	if e.Type == "" {
		e.Type = item.Type
	}
	if e.Season == "" {
		e.Season = item.Season
	}
	if e.Age == "" {
		e.Age = item.Age
	}
	if len(e.Genres) == 0 {
		e.Genres = append([]string(nil), item.Genres...)
	}
	if e.Status == "" {
		e.Status = item.Status
	}
	addAlt(e, item.Name)
	for _, alt := range item.Alt {
		addAlt(e, alt)
	}
	if pop > e.Pop {
		e.Pop = pop
	}
}

func mergeItems(anilibria, yummy []Item) []Entry {
	alPop := percentiles(anilibria)
	ymPop := percentiles(yummy)

	entries := make([]Entry, 0, len(anilibria)+len(yummy))
	byShiki := map[int]int{}
	byMAL := map[int]int{}
	byAlias := map[string]int{}

	register := func(index int) {
		e := entries[index]
		if e.Shiki > 0 {
			if _, exists := byShiki[e.Shiki]; !exists {
				byShiki[e.Shiki] = index
			}
		}
		if e.MAL > 0 {
			if _, exists := byMAL[e.MAL]; !exists {
				byMAL[e.MAL] = index
			}
		}
		if e.ALAlias != "" {
			if _, exists := byAlias[e.ALAlias]; !exists {
				byAlias[e.ALAlias] = index
			}
		}
	}

	for i, item := range anilibria {
		entries = append(entries, entryFromItem(item, alPop[i]))
		register(len(entries) - 1)
	}

	for i, item := range yummy {
		found := -1
		if item.Shiki > 0 {
			if index, ok := byShiki[item.Shiki]; ok {
				found = index
			}
		}
		if found < 0 && item.MAL > 0 {
			if index, ok := byMAL[item.MAL]; ok {
				found = index
			}
		}
		if found < 0 && item.Alias != "" {
			if index, ok := byAlias[item.Alias]; ok {
				found = index
			}
		}
		if found >= 0 && entries[found].YID == 0 {
			mergeYummyInto(&entries[found], item, ymPop[i])
			register(found)
			continue
		}
		entries = append(entries, entryFromItem(item, ymPop[i]))
		if found < 0 {
			register(len(entries) - 1)
		}
	}

	used := map[string]bool{}
	for i := range entries {
		entries[i].ID = pickID(&entries[i], used)
	}
	return entries
}

func pickID(e *Entry, used map[string]bool) string {
	var candidates []string
	if e.Shiki > 0 {
		candidates = append(candidates, "s"+strconv.Itoa(e.Shiki))
	}
	if e.ALAlias != "" {
		candidates = append(candidates, "a"+e.ALAlias)
	}
	if e.YID > 0 {
		candidates = append(candidates, "y"+strconv.Itoa(e.YID))
	}
	if e.ALID > 0 {
		candidates = append(candidates, "l"+strconv.Itoa(e.ALID))
	}
	for _, id := range candidates {
		if !used[id] {
			used[id] = true
			return id
		}
	}
	id := "x" + strconv.Itoa(len(used))
	used[id] = true
	return id
}
