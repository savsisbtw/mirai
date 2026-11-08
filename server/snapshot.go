package main

import (
	"bytes"
	"encoding/json"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Mask [3]uint64

type Row struct {
	Year   int16
	Type   uint8
	Status uint8
	Mask   Mask
}

type Snapshot struct {
	ID      string
	Entries []Entry
	Titles  [][]byte
	Hay     []string
	Rows    []Row

	OrderPopular []int32
	OrderNew     []int32
	OrderRating  []int32
	OrderName    []int32

	ByID       map[string]int32
	ByAlias    map[string]int32
	GenreBit   map[string]int
	TypeID     map[string]int
	Home       *Blob
	Filters    *Blob
	cache      sync.Map
	cacheCount atomic.Int32
}

const maxCacheEntries = 1500

func statusCode(status string) uint8 {
	switch status {
	case "ongoing":
		return 1
	case "released":
		return 2
	case "announce":
		return 3
	}
	return 0
}

func posterPath(source string) string {
	if source == "" {
		return ""
	}
	return "/v1/img?u=" + url.QueryEscape(source)
}

func makeTitle(e *Entry) TitleOut {
	t := TitleOut{
		ID:      e.ID,
		Name:    e.Name,
		NameEn:  e.NameEn,
		Year:    e.Year,
		Type:    e.Type,
		Status:  e.Status,
		Season:  e.Season,
		Age:     e.Age,
		Rating:  float64(int(e.Rating*100+0.5)) / 100,
		Poster:  posterPath(e.Poster),
		Genres:  e.Genres,
		Sources: e.Sources,
	}
	if t.Genres == nil {
		t.Genres = []string{}
	}
	if e.Episodes > 0 {
		count := e.Episodes
		t.Episodes = &count
	}
	return t
}

func makeFull(e *Entry) TitleFull {
	backdrop := e.Backdrop
	if backdrop == "" {
		backdrop = e.Poster
	}
	return TitleFull{
		TitleOut:    makeTitle(e),
		Description: e.Desc,
		Backdrop:    posterPath(backdrop),
	}
}

func buildSnapshot(entries []Entry) *Snapshot {
	s := &Snapshot{
		ID:       strconv.FormatInt(time.Now().UnixNano(), 36),
		Entries:  entries,
		Titles:   make([][]byte, len(entries)),
		Hay:      make([]string, len(entries)),
		Rows:     make([]Row, len(entries)),
		ByID:     make(map[string]int32, len(entries)),
		ByAlias:  map[string]int32{},
		GenreBit: map[string]int{},
		TypeID:   map[string]int{},
	}

	genreCount := map[string]int{}
	typeCount := map[string]int{}
	yearSet := map[int]bool{}
	for i := range entries {
		for _, g := range entries[i].Genres {
			genreCount[g]++
		}
		if entries[i].Type != "" {
			typeCount[entries[i].Type]++
		}
		if entries[i].Year > 0 {
			yearSet[entries[i].Year] = true
		}
	}

	genreNames := sortedByCount(genreCount)
	for i, name := range genreNames {
		if i < 192 {
			s.GenreBit[norm(name)] = i
		}
	}
	typeNames := sortedByCount(typeCount)
	for i, name := range typeNames {
		s.TypeID[norm(name)] = i + 1
	}

	for i := range entries {
		e := &entries[i]
		row := Row{Year: int16(e.Year), Status: statusCode(e.Status)}
		if id, ok := s.TypeID[norm(e.Type)]; ok {
			row.Type = uint8(id)
		}
		for _, g := range e.Genres {
			if bit, ok := s.GenreBit[norm(g)]; ok {
				row.Mask[bit/64] |= 1 << (bit % 64)
			}
		}
		s.Rows[i] = row
		s.ByID[e.ID] = int32(i)
		if e.ALAlias != "" {
			s.ByAlias[e.ALAlias] = int32(i)
		}
	}

	parallelRange(len(entries), 8, func(lo, hi int) {
		for i := lo; i < hi; i++ {
			e := &entries[i]
			title, _ := json.Marshal(makeTitle(e))
			s.Titles[i] = title

			var hay strings.Builder
			hay.WriteString(searchNorm(e.Name))
			if e.NameEn != "" {
				hay.WriteString("\n")
				hay.WriteString(searchNorm(e.NameEn))
			}
			for _, alt := range e.Alt {
				hay.WriteString("\n")
				hay.WriteString(searchNorm(alt))
			}
			s.Hay[i] = hay.String()
		}
	})

	var wg sync.WaitGroup
	wg.Add(4)
	go func() {
		defer wg.Done()
		s.OrderPopular = s.makeOrder(func(a, b *Entry) bool {
			if a.Pop != b.Pop {
				return a.Pop > b.Pop
			}
			return a.Rating > b.Rating
		})
	}()
	go func() {
		defer wg.Done()
		s.OrderNew = s.makeOrder(func(a, b *Entry) bool {
			if a.Year != b.Year {
				return a.Year > b.Year
			}
			if a.Fresh != b.Fresh {
				return a.Fresh > b.Fresh
			}
			return a.Pop > b.Pop
		})
	}()
	go func() {
		defer wg.Done()
		s.OrderRating = s.makeOrder(func(a, b *Entry) bool {
			if a.Rating != b.Rating {
				return a.Rating > b.Rating
			}
			return a.Pop > b.Pop
		})
	}()
	go func() {
		defer wg.Done()
		names := make([]string, len(entries))
		for i := range entries {
			names[i] = norm(entries[i].Name)
		}
		order := identityOrder(len(entries))
		sort.SliceStable(order, func(a, b int) bool {
			return names[order[a]] < names[order[b]]
		})
		s.OrderName = order
	}()
	wg.Wait()

	s.buildFilters(genreNames, genreCount, typeNames, yearSet)
	s.buildHome()
	return s
}

func identityOrder(n int) []int32 {
	order := make([]int32, n)
	for i := range order {
		order[i] = int32(i)
	}
	return order
}

func (s *Snapshot) makeOrder(less func(a, b *Entry) bool) []int32 {
	order := identityOrder(len(s.Entries))
	sort.SliceStable(order, func(a, b int) bool {
		return less(&s.Entries[order[a]], &s.Entries[order[b]])
	})
	return order
}

func sortedByCount(counts map[string]int) []string {
	names := make([]string, 0, len(counts))
	for name := range counts {
		names = append(names, name)
	}
	sort.Slice(names, func(a, b int) bool {
		if counts[names[a]] != counts[names[b]] {
			return counts[names[a]] > counts[names[b]]
		}
		return names[a] < names[b]
	})
	return names
}

func (s *Snapshot) buildFilters(genreNames []string, genreCount map[string]int, typeNames []string, yearSet map[int]bool) {
	type genreOut struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	genres := make([]genreOut, 0, len(genreNames))
	for _, name := range genreNames {
		genres = append(genres, genreOut{name, genreCount[name]})
	}
	years := make([]int, 0, len(yearSet))
	for year := range yearSet {
		years = append(years, year)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(years)))
	if typeNames == nil {
		typeNames = []string{}
	}
	body, _ := json.Marshal(map[string]any{"genres": genres, "years": years, "types": typeNames})
	s.Filters = newBlob(body, `"`+s.ID+`-filters"`)
}

func (s *Snapshot) buildHome() {
	var featuredIdx []int32
	candidates := make([]int32, 0)
	for _, i := range s.OrderPopular {
		e := &s.Entries[i]
		if e.Status == "ongoing" && e.Poster != "" && e.Desc != "" {
			candidates = append(candidates, i)
		}
		if len(candidates) >= 60 {
			break
		}
	}
	sort.SliceStable(candidates, func(a, b int) bool {
		x := &s.Entries[candidates[a]]
		y := &s.Entries[candidates[b]]
		return x.Pop*0.6+x.Rating/10*0.4 > y.Pop*0.6+y.Rating/10*0.4
	})
	for _, i := range candidates {
		if len(featuredIdx) == 5 {
			break
		}
		featuredIdx = append(featuredIdx, i)
	}

	var featured [][]byte
	for _, i := range featuredIdx {
		full := makeFull(&s.Entries[i])
		data, _ := json.Marshal(full)
		featured = append(featured, data)
	}

	var latest []int32
	freshOrder := identityOrder(len(s.Entries))
	sort.SliceStable(freshOrder, func(a, b int) bool {
		return s.Entries[freshOrder[a]].Fresh > s.Entries[freshOrder[b]].Fresh
	})
	for _, i := range freshOrder {
		if s.Entries[i].Fresh == 0 || len(latest) == 20 {
			break
		}
		latest = append(latest, i)
	}

	var ongoing []int32
	for _, i := range s.OrderPopular {
		if s.Entries[i].Status == "ongoing" {
			ongoing = append(ongoing, i)
		}
		if len(ongoing) == 20 {
			break
		}
	}

	var popular []int32
	for _, i := range s.OrderPopular {
		popular = append(popular, i)
		if len(popular) == 20 {
			break
		}
	}

	var buf bytes.Buffer
	buf.WriteString(`{"featured":`)
	writeRaw(&buf, featured)
	buf.WriteString(`,"latest":`)
	writeRaw(&buf, s.pick(latest))
	buf.WriteString(`,"ongoing":`)
	writeRaw(&buf, s.pick(ongoing))
	buf.WriteString(`,"popular":`)
	writeRaw(&buf, s.pick(popular))
	buf.WriteString(`}`)
	s.Home = newBlob(buf.Bytes(), `"`+s.ID+`-home"`)
}

func (s *Snapshot) pick(indexes []int32) [][]byte {
	result := make([][]byte, 0, len(indexes))
	for _, i := range indexes {
		result = append(result, s.Titles[i])
	}
	return result
}

func writeRaw(buf *bytes.Buffer, items [][]byte) {
	buf.WriteByte('[')
	for i, item := range items {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.Write(item)
	}
	buf.WriteByte(']')
}

type Query struct {
	Q        string
	Genres   []string
	Year     string
	Type     string
	Status   string
	Sort     string
	Page     int
	Limit    int
	yearFrom int
	yearTo   int
}

func parseQuery(values url.Values) Query {
	q := Query{
		Q:      searchNorm(values.Get("q")),
		Year:   strings.TrimSpace(values.Get("year")),
		Type:   strings.TrimSpace(values.Get("type")),
		Status: strings.TrimSpace(values.Get("status")),
		Sort:   strings.TrimSpace(values.Get("sort")),
	}
	for _, g := range strings.Split(values.Get("genre"), ",") {
		g = strings.TrimSpace(g)
		if g != "" {
			q.Genres = append(q.Genres, g)
		}
	}
	switch q.Sort {
	case "popular", "new", "rating", "name":
	default:
		q.Sort = "popular"
	}
	q.Page, _ = strconv.Atoi(values.Get("page"))
	if q.Page < 1 {
		q.Page = 1
	}
	q.Limit, _ = strconv.Atoi(values.Get("limit"))
	if q.Limit < 1 {
		q.Limit = 48
	}
	if q.Limit > 100 {
		q.Limit = 100
	}
	if q.Year != "" {
		parts := strings.SplitN(q.Year, "-", 2)
		q.yearFrom, _ = strconv.Atoi(strings.TrimSpace(parts[0]))
		q.yearTo = q.yearFrom
		if len(parts) == 2 {
			q.yearTo, _ = strconv.Atoi(strings.TrimSpace(parts[1]))
		}
		if q.yearTo < q.yearFrom {
			q.yearFrom, q.yearTo = q.yearTo, q.yearFrom
		}
	}
	return q
}

func (q Query) key() string {
	return q.Q + "|" + strings.Join(q.Genres, ",") + "|" + q.Year + "|" + q.Type + "|" + q.Status + "|" +
		q.Sort + "|" + strconv.Itoa(q.Page) + "|" + strconv.Itoa(q.Limit)
}

type filter struct {
	mask       Mask
	hasYear    bool
	yearFrom   int16
	yearTo     int16
	typeID     uint8
	status     uint8
	impossible bool
}

func (s *Snapshot) makeFilter(q Query) filter {
	var f filter
	for _, g := range q.Genres {
		bit, ok := s.GenreBit[norm(g)]
		if !ok {
			f.impossible = true
			return f
		}
		f.mask[bit/64] |= 1 << (bit % 64)
	}
	if q.Year != "" {
		f.hasYear = true
		f.yearFrom = int16(q.yearFrom)
		f.yearTo = int16(q.yearTo)
	}
	if q.Type != "" {
		id, ok := s.TypeID[norm(q.Type)]
		if !ok {
			f.impossible = true
			return f
		}
		f.typeID = uint8(id)
	}
	if q.Status != "" {
		f.status = statusCode(strings.ToLower(q.Status))
		if f.status == 0 {
			f.impossible = true
		}
	}
	return f
}

func (f *filter) isEmpty() bool {
	return f.mask == Mask{} && !f.hasYear && f.typeID == 0 && f.status == 0
}

func (f *filter) match(r *Row) bool {
	for k := 0; k < len(f.mask); k++ {
		if (r.Mask[k] & f.mask[k]) != f.mask[k] {
			return false
		}
	}
	if f.hasYear && (r.Year < f.yearFrom || r.Year > f.yearTo) {
		return false
	}
	if f.typeID != 0 && r.Type != f.typeID {
		return false
	}
	if f.status != 0 && r.Status != f.status {
		return false
	}
	return true
}

func isWordBoundary(hay string, pos int) bool {
	prev := hay[pos-1]
	if prev >= 0x80 {
		return false
	}
	isLetter := prev >= 'a' && prev <= 'z'
	isDigit := prev >= '0' && prev <= '9'
	return !isLetter && !isDigit
}

func searchScore(hay, q string) int {
	best := 0
	start := 0
	for start <= len(hay)-len(q) {
		found := strings.Index(hay[start:], q)
		if found < 0 {
			break
		}
		pos := start + found
		if pos == 0 || hay[pos-1] == '\n' {
			return 3
		}
		if isWordBoundary(hay, pos) {
			best = 2
		} else if best == 0 {
			best = 1
		}
		start = pos + 1
	}
	return best
}

func (s *Snapshot) order(sortName string) []int32 {
	switch sortName {
	case "new":
		return s.OrderNew
	case "rating":
		return s.OrderRating
	case "name":
		return s.OrderName
	}
	return s.OrderPopular
}

func (s *Snapshot) catalog(q Query) []byte {
	f := s.makeFilter(q)
	order := s.order(q.Sort)
	offset := (q.Page - 1) * q.Limit
	var window []int32
	total := 0

	if f.impossible {
		order = nil
	}

	if q.Q != "" {
		var top, word, sub []int32
		for _, i := range order {
			if !f.match(&s.Rows[i]) {
				continue
			}
			switch searchScore(s.Hay[i], q.Q) {
			case 3:
				top = append(top, i)
			case 2:
				word = append(word, i)
			case 1:
				sub = append(sub, i)
			}
		}
		all := make([]int32, 0, len(top)+len(word)+len(sub))
		all = append(all, top...)
		all = append(all, word...)
		all = append(all, sub...)
		total = len(all)
		if offset < total {
			window = all[offset:min(offset+q.Limit, total)]
		}
	} else if f.isEmpty() {
		total = len(order)
		if offset < total {
			window = order[offset:min(offset+q.Limit, total)]
		}
	} else {
		window = make([]int32, 0, q.Limit)
		for _, i := range order {
			if !f.match(&s.Rows[i]) {
				continue
			}
			total++
			if total > offset && len(window) < q.Limit {
				window = append(window, i)
			}
		}
	}

	pages := (total + q.Limit - 1) / q.Limit
	if pages < 1 {
		pages = 1
	}
	var buf bytes.Buffer
	buf.Grow(len(window)*400 + 64)
	buf.WriteString(`{"items":`)
	writeRaw(&buf, s.pick(window))
	buf.WriteString(`,"total":`)
	buf.WriteString(strconv.Itoa(total))
	buf.WriteString(`,"page":`)
	buf.WriteString(strconv.Itoa(q.Page))
	buf.WriteString(`,"pages":`)
	buf.WriteString(strconv.Itoa(pages))
	buf.WriteString(`}`)
	return buf.Bytes()
}

func (s *Snapshot) catalogBlob(q Query) *Blob {
	key := q.key()
	if cached, ok := s.cache.Load(key); ok {
		return cached.(*Blob)
	}
	body := s.catalog(q)
	blob := newBlob(body, `"`+s.ID+`-c`+hashString(key)+`"`)
	if s.cacheCount.Load() < maxCacheEntries {
		s.cacheCount.Add(1)
		s.cache.Store(key, blob)
	}
	return blob
}
