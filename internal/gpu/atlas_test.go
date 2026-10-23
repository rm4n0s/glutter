package gpu

import (
	"testing"

	"github.com/rm4n0s/errors"
)

func route(t *testing.T, err error, tag string) *errors.Error {
	t.Helper()
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != tag {
		t.Fatalf("error = %v, want %s", err, tag)
	}
	return appErr
}

func mustPut(t *testing.T, a *Atlas[int], key, w, h int) *AtlasEntry[int] {
	t.Helper()
	e, err := a.Put(key, w, h)
	if err != nil {
		t.Fatalf("Put(%d, %d, %d): %v", key, w, h, err)
	}
	return e
}

func TestAtlasPlacesTilesBySizeClass(t *testing.T) {
	a := NewAtlas[int](512, 4)
	tests := []struct {
		w, h  int
		class int
	}{{10, 10, 32}, {32, 5, 32}, {33, 1, 64}, {64, 64, 64}, {65, 3, 128}, {200, 100, 256}, {256, 256, 256}}
	for i, tt := range tests {
		e := mustPut(t, a, i, tt.w, tt.h)
		if e.class != tt.class || e.W != tt.w || e.H != tt.h || e.Empty {
			t.Errorf("%dx%d: class %d, size %dx%d", tt.w, tt.h, e.class, e.W, e.H)
		}
		if e.X%tt.class != 0 || e.Y%tt.class != 0 || e.X+tt.class > 512 || e.Y+tt.class > 512 {
			t.Errorf("%dx%d: slot at %d,%d is not a slot of %d on a 512 page", tt.w, tt.h, e.X, e.Y, tt.class)
		}
		if got := a.Get(i); got != e {
			t.Errorf("Get(%d) = %v", i, got)
		}
	}
	// No two tiles of the same page overlap.
	type rect struct{ page, x0, y0, x1, y1 int }
	var rects []rect
	for i := range tests {
		e := a.Get(i)
		rects = append(rects, rect{e.Page, e.X, e.Y, e.X + e.class, e.Y + e.class})
	}
	for i := range rects {
		for j := i + 1; j < len(rects); j++ {
			p, q := rects[i], rects[j]
			if p.page == q.page && p.x0 < q.x1 && q.x0 < p.x1 && p.y0 < q.y1 && q.y0 < p.y1 {
				t.Errorf("tiles %d and %d overlap: %+v %+v", i, j, p, q)
			}
		}
	}
	if st := a.Stats(); st.Entries != len(tests) || st.Pages < 4 || st.Evicted != 0 {
		t.Errorf("stats = %+v", st)
	}
}

func TestAtlasFillsAPageBeforeMakingAnother(t *testing.T) {
	a := NewAtlas[int](64, 3) // a page holds four 32 pixel slots
	for i := 0; i < 4; i++ {
		if e := mustPut(t, a, i, 20, 20); e.Page != 0 {
			t.Errorf("tile %d is on page %d", i, e.Page)
		}
	}
	if e := mustPut(t, a, 4, 20, 20); e.Page != 1 || a.Pages() != 2 {
		t.Errorf("the fifth tile is on page %d of %d", e.Page, a.Pages())
	}
	if a.PageSize() != 64 {
		t.Errorf("PageSize = %d", a.PageSize())
	}
}

func TestAtlasEvictsTheLeastRecentlyUsed(t *testing.T) {
	a := NewAtlas[int](64, 1) // four slots, no more pages
	a.BeginFrame()
	for i := 0; i < 4; i++ {
		mustPut(t, a, i, 10, 10)
	}
	slot := a.entries[0] // not Get: that would make the tile young
	x, y := slot.X, slot.Y
	// Tiles of an earlier frame can go. Using 2 and 3 makes 0 and 1 the oldest, 0 first.
	a.BeginFrame()
	a.Get(2)
	a.Get(3)
	e := mustPut(t, a, 10, 10, 10)
	if e.X != x || e.Y != y {
		t.Errorf("the new tile took %d,%d, want the slot of tile 0 at %d,%d", e.X, e.Y, x, y)
	}
	if a.Get(0) != nil || a.Get(1) == nil {
		t.Error("tile 0 should be evicted and tile 1 kept")
	}
	// A touch makes a tile young: 1 is used again, so 2 goes next.
	a.BeginFrame()
	a.Get(1)
	mustPut(t, a, 11, 10, 10)
	if a.Get(2) != nil || a.Get(1) == nil || a.Get(3) == nil {
		t.Error("tile 2 should have been evicted")
	}
	if st := a.Stats(); st.Evicted != 2 || st.Entries != 4 {
		t.Errorf("stats = %+v", st)
	}
}

func TestAtlasNeverEvictsATileOfTheFrame(t *testing.T) {
	a := NewAtlas[int](64, 1)
	a.BeginFrame()
	for i := 0; i < 4; i++ {
		mustPut(t, a, i, 10, 10)
	}
	_, err := a.Put(9, 10, 10)
	appErr := route(t, err, "AtlasFull")
	if !appErr.HasRoute("Atlas[...].Put->Atlas[...].allocate.AtlasFull") {
		t.Errorf("route = %s", appErr.Route())
	}
	// Next frame the tiles are old, and one of them gives way.
	a.BeginFrame()
	mustPut(t, a, 9, 10, 10)
}

func TestAtlasReassignsAnIdlePageToAnotherClass(t *testing.T) {
	a := NewAtlas[int](64, 1)
	a.BeginFrame()
	for i := 0; i < 4; i++ {
		mustPut(t, a, i, 10, 10) // the page is of 32 pixel slots
	}
	// A 64 pixel tile needs the whole page; the 32 pixel tiles are in use now.
	_, err := a.Put(9, 60, 60)
	_ = route(t, err, "AtlasFull")
	a.BeginFrame()
	e := mustPut(t, a, 9, 60, 60)
	if e.class != 64 || e.X != 0 || e.Y != 0 || e.Page != 0 {
		t.Errorf("tile = %+v", e)
	}
	for i := 0; i < 4; i++ {
		if a.Get(i) != nil {
			t.Errorf("tile %d survived the page being reassigned", i)
		}
	}
	if st := a.Stats(); st.Entries != 1 || st.Evicted != 4 {
		t.Errorf("stats = %+v", st)
	}
}

func TestAtlasEmptyTiles(t *testing.T) {
	a := NewAtlas[string](64, 1)
	e, err := a.Put("nothing", 0, 7)
	if err != nil {
		t.Fatal(err)
	}
	if !e.Empty || e.W != 0 || e.H != 0 {
		t.Errorf("entry = %+v", e)
	}
	if a.Pages() != 0 {
		t.Error("an empty tile made a page")
	}
	if a.Get("nothing") != e {
		t.Error("an empty tile is cached too")
	}
	// Empty tiles are evicted from the list like any other, without freeing a slot.
	a.BeginFrame()
	for i := 0; i < 4; i++ {
		if _, err := a.Put(string(rune('a'+i)), 5, 5); err != nil {
			t.Fatal(err)
		}
	}
	a.BeginFrame()
	if _, err := a.Put("more", 5, 5); err != nil {
		t.Fatal(err)
	}
}

func TestAtlasErrors(t *testing.T) {
	a := NewAtlas[int](512, 2)
	for _, sz := range [][2]int{{257, 1}, {1, 257}, {-1, 5}, {5, -1}} {
		_, err := a.Put(1, sz[0], sz[1])
		appErr := route(t, err, "AtlasTileInvalid")
		if !appErr.HasRoute("Atlas[...].Put.AtlasTileInvalid") {
			t.Errorf("route = %s", appErr.Route())
		}
	}
	mustPut(t, a, 1, 5, 5)
	defer func() {
		e, _ := recover().(error)
		appErr, ok := errors.FromError(e)
		if !ok || appErr.Tag != tagAtlasKeyStored || !appErr.HasRoute("Atlas[...].Put.AtlasKeyStored") {
			t.Errorf("expected AtlasKeyStored, got %v", e)
		}
	}()
	_, _ = a.Put(1, 5, 5)
}

func TestAtlasClear(t *testing.T) {
	a := NewAtlas[int](64, 2)
	for i := 0; i < 6; i++ {
		mustPut(t, a, i, 20, 20)
	}
	a.Clear()
	if st := a.Stats(); st.Pages != 0 || st.Entries != 0 {
		t.Errorf("stats after Clear = %+v", st)
	}
	if e := mustPut(t, a, 0, 20, 20); e.Page != 0 || e.X != 0 || e.Y != 0 {
		t.Errorf("first tile after Clear = %+v", e)
	}
}
