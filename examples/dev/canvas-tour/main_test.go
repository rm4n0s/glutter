package main

import (
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/fake"
	"github.com/rm4n0s/glutter/internal/proof"
	"github.com/rm4n0s/glutter/ui"
)

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func TestPagesAreTheProofScenesInOrder(t *testing.T) {
	tour, err := newTour(env(nil))
	if err != nil {
		t.Fatal(err)
	}
	// The pages start with the planned order, every registered scene is a page, and the
	// grid for the size of the window is the last.
	for i, n := range order {
		if tour.pages[i] != n {
			t.Errorf("page %d = %q, want %q", i+1, tour.pages[i], n)
		}
	}
	if len(tour.pages) != len(proof.Names())+1 || tour.pages[len(tour.pages)-1] != gridPage {
		t.Errorf("%d pages for %d scenes: %v", len(tour.pages), len(proof.Names()), tour.pages)
	}
	if tour.page != 0 {
		t.Errorf("the first page is %d", tour.page)
	}
}

func TestStartPage(t *testing.T) {
	tests := []struct {
		value string
		want  string
	}{
		{"", "shapes-basic"},
		{"1", "shapes-basic"},
		{"2", "shapes-clips"},
		{"5", "effects"},
		{"6", "text-tour"},
		{"7", gridPage},
		{"blend-modes", "blend-modes"},
		{gridPage, gridPage},
	}
	for _, tt := range tests {
		tour, err := newTour(env(map[string]string{"GLUTTER_DEV_PAGE": tt.value}))
		if err != nil {
			t.Fatalf("%q: %v", tt.value, err)
		}
		if got := tour.pages[tour.page]; got != tt.want {
			t.Errorf("%q starts at %q, want %q", tt.value, got, tt.want)
		}
	}
}

func TestStartPageInvalid(t *testing.T) {
	for _, v := range []string{"0", "8", "-1", "nope", "shapes"} {
		_, err := newTour(env(map[string]string{"GLUTTER_DEV_PAGE": v}))
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != "TourPageInvalid" || !appErr.HasRoute("newTour->tour.find.TourPageInvalid") {
			t.Errorf("%q: error = %v, want TourPageInvalid", v, err)
		}
	}
}

func windowOf(t *testing.T) (*fake.Window, *fake.Platform) {
	t.Helper()
	p := fake.New()
	w, err := p.NewWindow(platform.DefaultWindowConfig())
	if err != nil {
		t.Fatal(err)
	}
	return w.(*fake.Window), p
}

func key(logical uint64, typ ui.KeyEventType) ui.KeyData {
	return ui.KeyData{Type: typ, Logical: logical}
}

func TestKeysChangeThePage(t *testing.T) {
	tour, _ := newTour(env(nil))
	win, p := windowOf(t)
	stopped := 0
	tour.win, tour.stop = win, func() { stopped++ }
	n := len(tour.pages)
	frames := func() int {
		c := 0
		for _, e := range p.Log() {
			if e.Kind == fake.EntryRequestFrame {
				c++
			}
		}
		return c
	}
	tests := []struct {
		name  string
		key   ui.KeyData
		page  int
		frame bool
	}{
		{"right", key(keyArrowRite, ui.KeyEventTypeDown), 1, true},
		{"page down", key(keyPageDown, ui.KeyEventTypeDown), 2, true},
		{"space", key(keySpace, ui.KeyEventTypeDown), 3, true},
		{"a repeat counts", key(keyArrowRite, ui.KeyEventTypeRepeat), 4, true},
		{"a release does not", key(keyArrowRite, ui.KeyEventTypeUp), 4, false},
		{"left", key(keyArrowLeft, ui.KeyEventTypeDown), 3, true},
		{"page up", key(keyPageUp, ui.KeyEventTypeDown), 2, true},
		{"another key", key('x', ui.KeyEventTypeDown), 2, false},
	}
	for _, tt := range tests {
		before := frames()
		tour.onKey(tt.key)
		if tour.page != tt.page {
			t.Errorf("%s: page = %d, want %d", tt.name, tour.page, tt.page)
		}
		if got := frames() - before; (got == 1) != tt.frame || got > 1 {
			t.Errorf("%s: %d frames requested, want frame=%v", tt.name, got, tt.frame)
		}
	}
	// The pages wrap around in both directions.
	tour.page = n - 1
	tour.onKey(key(keyArrowRite, ui.KeyEventTypeDown))
	if tour.page != 0 {
		t.Errorf("after the last page: %d", tour.page)
	}
	tour.onKey(key(keyArrowLeft, ui.KeyEventTypeDown))
	if tour.page != n-1 {
		t.Errorf("before the first page: %d", tour.page)
	}
	if stopped != 0 {
		t.Error("stopped by a navigation key")
	}
	tour.onKey(key(keyEscape, ui.KeyEventTypeDown))
	if stopped != 1 {
		t.Errorf("Escape stopped %d times", stopped)
	}
}

func TestKeysWithoutAWindow(t *testing.T) {
	tour, _ := newTour(env(nil))
	tour.onKey(key(keyArrowRite, ui.KeyEventTypeDown)) // no window and no stop function yet
	tour.onKey(key(keyEscape, ui.KeyEventTypeDown))
	if tour.page != 1 {
		t.Errorf("page = %d", tour.page)
	}
}

func TestSceneIsBuiltOnce(t *testing.T) {
	tour, _ := newTour(env(map[string]string{"GLUTTER_DEV_PAGE": "shapes-clips"}))
	a := tour.scene(ui.Size{}, 1)
	if a == nil || tour.scene(ui.Size{Width: 5}, 2) != a {
		t.Error("a page's scene is built once and reused")
	}
	tour.show(1)
	if tour.scene(ui.Size{}, 1) == a {
		t.Error("another page has another scene")
	}
}

func TestGridPageDependsOnTheWindowSize(t *testing.T) {
	tour, _ := newTour(env(map[string]string{"GLUTTER_DEV_PAGE": gridPage}))
	a := tour.scene(ui.Size{Width: 960, Height: 720}, 1.5)
	if tour.scene(ui.Size{Width: 960, Height: 720}, 1.5) != a {
		t.Error("the grid is built once for a size")
	}
	b := tour.scene(ui.Size{Width: 800, Height: 600}, 1.5)
	if b == a {
		t.Error("a window of another size has another grid")
	}
	// The grid's lines are on whole pixels one wide: the picture is 1-pixel rectangles,
	// the border included.
	img, err := soft.Render(a, 960, 720)
	if err != nil {
		t.Fatal(err)
	}
	px := func(x, y int) [4]uint8 {
		i := img.PixOffset(x, y)
		return [4]uint8{img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3]}
	}
	accent := [4]uint8{0x67, 0x50, 0xA4, 255}
	dark := [4]uint8{0x4A, 0x44, 0x58, 255}
	faint := [4]uint8{0xC9, 0xC5, 0xD4, 255}
	white := [4]uint8{255, 255, 255, 255}
	for _, tt := range []struct {
		name string
		x, y int
		want [4]uint8
	}{
		{"top left corner", 0, 0, accent},
		{"bottom right corner", 959, 719, accent},
		{"left edge", 0, 300, accent},
		{"bottom edge", 300, 719, accent},
		{"a faint vertical line", 16, 40, faint},
		{"the pixel next to it", 17, 40, white},
		{"a dark vertical line", 80, 40, dark},
		{"a dark horizontal line", 40, 160, dark},
		{"a faint horizontal line", 40, 176, faint},
		{"between lines", 40, 41, white},
		{"the center's mark", 480, 366, accent},
	} {
		if got := px(tt.x, tt.y); got != tt.want {
			t.Errorf("%s: pixel %d,%d = %v, want %v", tt.name, tt.x, tt.y, got, tt.want)
		}
	}
}
