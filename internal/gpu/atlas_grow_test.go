package gpu

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// TestAtlasGrowsForAFrame checks the pages over the number the atlas keeps: a frame whose tiles
// are all in use makes it grow up to the frame limit, old tiles give way first, and EndFrame
// frees the pages over the kept number that the frame did not use. The atlas has pages of 64
// pixels: four slots of 32 pixels, or one of 64. Each row is a script:
//
//	begin            BeginFrame
//	put A-B SIZE     Put the keys A to B, tiles of SIZE x SIZE
//	full KEY SIZE    Put fails with AtlasFull
//	get A-B          Get the keys A to B, which are there
//	gone A-B         the keys A to B are not in the atlas (asked without marking them used)
//	end I,J          EndFrame returns the pages I and J, in that order ("end" alone: none)
//	pages N L O      Pages() is N, and Stats says L pages of which O are over the kept number
//	on KEY PAGE      the tile of KEY is on the page
//	clear            Clear
func TestAtlasGrowsForAFrame(t *testing.T) {
	tests := []struct {
		name        string
		kept, limit int
		script      []string
	}{
		{"a frame that needs more pages than the atlas keeps gets them, and they go when a frame does not use them", 1, 3, []string{
			"begin", "put 0-11 10", "pages 3 3 2", "on 0 0", "on 4 1", "on 8 2", "end", "pages 3 3 2",
			"begin", "get 0-3", "end 2,1", "pages 1 1 0", "gone 4-11", "get 0-3",
		}},
		{"the pages that a frame uses stay, however many they are", 1, 3, []string{
			"begin", "put 0-11 10", "end",
			"begin", "get 0-11", "end", "pages 3 3 2",
			"begin", "get 0-11", "end", "pages 3 3 2",
		}},
		{"a page in the middle is freed, the others keep their index, and a new page takes the empty one", 1, 3, []string{
			"begin", "put 0-11 10", "end",
			"begin", "get 0-0", "get 8-8", "end 1", "pages 3 2 1", "gone 4-7", "on 8 2",
			"begin", "get 0-3", "get 8-11", "put 20-20 10", "on 20 1", "pages 3 3 2", "on 8 2", "end",
		}},
		{"only as many pages are freed as there are over the kept number", 2, 4, []string{
			"begin", "put 0-15 10", "pages 4 4 2", "end",
			// Nothing is used: the two pages over the kept number go, from the last index.
			"begin", "end 3,2", "pages 2 2 0", "get 0-7", "gone 8-15",
		}},
		{"the tiles of an earlier frame give way before the atlas grows", 1, 3, []string{
			"begin", "put 0-3 10", "end",
			"begin", "put 4-7 10", "pages 1 1 0", "gone 0-3", "end",
		}},
		{"an idle page of another class is taken before the atlas grows", 1, 3, []string{
			"begin", "put 0-3 10", "end",
			"begin", "put 9-9 60", "pages 1 1 0", "on 9 0", "gone 0-3", "end",
		}},
		{"a tile of another class that the frame needs gets a page over the kept number", 1, 2, []string{
			"begin", "put 0-3 10", "put 9-9 60", "pages 2 2 1", "on 9 1", "end",
			// The page of the small tiles goes; the large tile stays where it is.
			"begin", "get 9-9", "end 0", "pages 2 1 0", "on 9 1", "gone 0-3",
			"begin", "get 9-9", "put 0-3 10", "on 0 0", "pages 2 2 1", "end",
		}},
		{"at the frame limit a tile has no room, and the next frame has", 1, 2, []string{
			"begin", "put 0-7 10", "full 8 10", "full 9 60", "pages 2 2 1", "get 0-7", "end",
			// The new tile takes the slot of the tile used longest ago, on the first page.
			"begin", "put 8-8 10", "on 8 0", "gone 0-0", "pages 2 2 1", "end 1", "pages 1 1 0",
		}},
		{"without a frame limit the atlas does not grow", 2, 0, []string{
			"begin", "put 0-7 10", "full 8 10", "pages 2 2 0", "end",
		}},
		{"a frame limit below the kept pages is the kept pages", 2, 1, []string{
			"begin", "put 0-7 10", "full 8 10", "pages 2 2 0", "end",
		}},
		{"an empty tile needs no page and keeps none", 1, 2, []string{
			"begin", "put 0-7 10", "put 20-20 0", "end",
			"begin", "get 0-3", "get 20-20", "end 1", "get 20-20", "pages 1 1 0",
		}},
		{"clearing forgets the pages over the kept number too", 1, 3, []string{
			"begin", "put 0-11 10", "clear", "pages 0 0 0", "gone 0-11",
			"begin", "put 0-3 10", "pages 1 1 0", "end",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewAtlas[int](64, tt.kept)
			if tt.limit != 0 {
				a.SetFrameLimit(tt.limit)
			}
			if a.KeptPages() != tt.kept || a.FrameLimit() != max(tt.limit, tt.kept) {
				t.Fatalf("kept %d, limit %d", a.KeptPages(), a.FrameLimit())
			}
			for i, line := range tt.script {
				f := strings.Fields(line)
				at := fmt.Sprintf("step %d (%s)", i, line)
				num := func(s string) int {
					n, err := strconv.Atoi(s)
					if err != nil {
						t.Fatalf("%s: %v", at, err)
					}
					return n
				}
				keys := func(s string) (int, int) {
					lo, hi, _ := strings.Cut(s, "-")
					return num(lo), num(hi)
				}
				switch f[0] {
				case "begin":
					a.BeginFrame()
				case "put":
					lo, hi := keys(f[1])
					for k := lo; k <= hi; k++ {
						if _, err := a.Put(k, num(f[2]), num(f[2])); err != nil {
							t.Fatalf("%s: key %d: %v", at, k, err)
						}
					}
				case "full":
					_, err := a.Put(num(f[1]), num(f[2]), num(f[2]))
					appErr := route(t, err, TagAtlasFull)
					if !appErr.HasRoute("Atlas[...].Put->Atlas[...].allocate.AtlasFull") {
						t.Errorf("%s: route = %s", at, appErr.Route())
					}
					if got := appErr.Metadata["pages"]; got != a.Stats().Pages {
						t.Errorf("%s: the error says %v pages, the atlas has %d", at, got, a.Stats().Pages)
					}
				case "get":
					lo, hi := keys(f[1])
					for k := lo; k <= hi; k++ {
						if a.Get(k) == nil {
							t.Fatalf("%s: key %d is not in the atlas", at, k)
						}
					}
				case "gone":
					lo, hi := keys(f[1])
					for k := lo; k <= hi; k++ {
						if _, ok := a.entries[k]; ok {
							t.Fatalf("%s: key %d is still in the atlas", at, k)
						}
					}
				case "end":
					var want []int
					if len(f) > 1 {
						for _, s := range strings.Split(f[1], ",") {
							want = append(want, num(s))
						}
					}
					if got := a.EndFrame(); !slices.Equal(got, want) {
						t.Fatalf("%s: freed the pages %v, want %v", at, got, want)
					}
				case "pages":
					st := a.Stats()
					if a.Pages() != num(f[1]) || st.Pages != num(f[2]) || st.PagesOverBudget != num(f[3]) {
						t.Fatalf("%s: Pages() %d, %d pages, %d over the kept number", at, a.Pages(), st.Pages, st.PagesOverBudget)
					}
				case "on":
					e, ok := a.entries[num(f[1])]
					if !ok || e.Page != num(f[2]) {
						t.Fatalf("%s: the tile is %+v", at, e)
					}
				case "clear":
					a.Clear()
				default:
					t.Fatalf("%s: unknown step", at)
				}
				checkAtlas(t, a, at)
			}
		})
	}
}

// checkAtlas checks what must hold of an atlas after any call: every tile with a slot is on a
// page that exists, of its class, in a slot no other tile has, and the counts of the pages and
// of their slots are those of the tiles.
func checkAtlas[K comparable](t *testing.T, a *Atlas[K], at string) {
	t.Helper()
	live := 0
	used := map[int]int{}
	for i, p := range a.pages {
		if p != nil {
			live++
			used[i] = 0
		}
	}
	if live != a.live || live > a.frameLimit {
		t.Fatalf("%s: %d pages, the atlas counts %d, the frame limit is %d", at, live, a.live, a.frameLimit)
	}
	if n := len(a.pages); n > 0 && a.pages[n-1] == nil {
		t.Fatalf("%s: the last index has no page", at)
	}
	slots := map[[2]int]bool{}
	for k, e := range a.entries {
		if e.Empty {
			continue
		}
		if e.Page < 0 || e.Page >= len(a.pages) || a.pages[e.Page] == nil {
			t.Fatalf("%s: the tile %v is on the page %d, which the atlas does not have", at, k, e.Page)
		}
		p := a.pages[e.Page]
		if p.class != e.class || e.slot < 0 || e.slot >= p.side*p.side {
			t.Fatalf("%s: the tile %v of class %d has the slot %d of a page of class %d", at, k, e.class, e.slot, p.class)
		}
		if slots[[2]int{e.Page, e.slot}] {
			t.Fatalf("%s: two tiles have the slot %d of the page %d", at, e.slot, e.Page)
		}
		slots[[2]int{e.Page, e.slot}] = true
		if slices.Contains(p.free, e.slot) {
			t.Fatalf("%s: the slot %d of the page %d is in use and free", at, e.slot, e.Page)
		}
		used[e.Page]++
	}
	for i, n := range used {
		if p := a.pages[i]; p.used != n || len(p.free) != p.side*p.side-n {
			t.Fatalf("%s: the page %d has %d tiles, counts %d and has %d free slots of %d", at, i, n, p.used, len(p.free), p.side*p.side)
		}
	}
	if a.lru.Len() != len(a.entries) {
		t.Fatalf("%s: %d tiles, %d in the order of use", at, len(a.entries), a.lru.Len())
	}
}
