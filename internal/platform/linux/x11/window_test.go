package x11

import (
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
	"github.com/rm4n0s/glutter/ui"
)

func TestSizeLimits(t *testing.T) {
	resizable := platform.DefaultWindowConfig()
	fixed := platform.DefaultWindowConfig()
	fixed.Resizable = foundation.Some(false)
	limited := platform.DefaultWindowConfig()
	limited.MinSize = ui.Size{Width: 200, Height: 100}
	limited.MaxSize = ui.Size{Width: 1000, Height: 800}
	onlyMinWidth := platform.DefaultWindowConfig()
	onlyMinWidth.MinSize = ui.Size{Width: 300}

	tests := []struct {
		name     string
		cfg      platform.WindowConfig
		min, max *[2]int32
	}{
		{"no limits", resizable, nil, nil},
		{"a window that cannot be resized is its own limit", fixed, &[2]int32{640, 400}, &[2]int32{640, 400}},
		{"both limits", limited, &[2]int32{200, 100}, &[2]int32{1000, 800}},
		{"a limit in one dimension", onlyMinWidth, &[2]int32{300, 0}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			min, max := sizeLimits(tt.cfg, 640, 400, 1)
			if (min == nil) != (tt.min == nil) || (min != nil && *min != *tt.min) {
				t.Errorf("min = %v, want %v", min, tt.min)
			}
			if (max == nil) != (tt.max == nil) || (max != nil && *max != *tt.max) {
				t.Errorf("max = %v, want %v", max, tt.max)
			}
		})
	}
}

func TestLatin1(t *testing.T) {
	tests := []struct{ in, want string }{
		{"Glutter", "Glutter"},
		{"", ""},
		{"café", "caf\xe9"},
		{"日本語 ok", "??? ok"},
		{"ÿĀ", "\xff?"},
	}
	for _, tt := range tests {
		if got := latin1(tt.in); got != tt.want {
			t.Errorf("latin1(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestRefreshIntervalOf(t *testing.T) {
	tests := []struct {
		rate int
		want time.Duration
	}{
		{60, time.Second / 60},
		{144, time.Second / 144},
		{30, time.Second / 30},
		{0, time.Second / 60},    // XRandR says nothing
		{-1, time.Second / 60},   // nonsense
		{5000, time.Second / 60}, // nonsense
	}
	for _, tt := range tests {
		if got := refreshIntervalOf(tt.rate); got != tt.want {
			t.Errorf("refreshIntervalOf(%d) = %v, want %v", tt.rate, got, tt.want)
		}
	}
}

// fakeCursors is a cursors with a fake libXcursor and a Platform whose X calls
// are recorded instead of sent.
type fakeCursors struct {
	c       *cursors
	loads   []string
	defined [][2]uint64 // window, cursor
	ids     map[string]uint64
	freed   []uint64
	images  int
	themes  []string // the names given to XcursorSetTheme
	sizes   []int32  // the sizes given to XcursorSetDefaultSize
}

func newFakeCursors(ids map[string]uint64) *fakeCursors {
	f := &fakeCursors{ids: ids}
	p := &Platform{x: &xlib{
		defineCursor: func(_ uintptr, window, cursor uint64) int32 {
			f.defined = append(f.defined, [2]uint64{window, cursor})
			return 0
		},
		createBitmap: func(_ uintptr, _ uint64, data uintptr, w, h uint32) uint64 {
			f.images++
			if *view[byte](data) != 0 || w != 1 || h != 1 {
				panic("the hidden cursor's bitmap is not empty")
			}
			return 55
		},
		createPixmapCursor: func(_ uintptr, source, mask uint64, _, _ uintptr, _, _ uint32) uint64 {
			if source != 55 || mask != 55 {
				panic("the hidden cursor does not use the empty bitmap as shape and mask")
			}
			return 777
		},
		freePixmap: func(uintptr, uint64) int32 { return 0 },
		flush:      func(uintptr) int32 { return 0 },
		freeCursor: func(_ uintptr, id uint64) int32 { f.freed = append(f.freed, id); return 0 },
	}}
	f.c = newCursors(p)
	f.c.getenv = func(string) string { return "" }
	f.c.load = func() (*xcursor, error) {
		return &xcursor{
			setTheme: func(_ uintptr, name uintptr) int32 {
				f.themes = append(f.themes, dl.GoString(name))
				return 1
			},
			setDefaultSize: func(_ uintptr, size int32) int32 {
				f.sizes = append(f.sizes, size)
				return 1
			},
			libraryLoadCursor: func(_ uintptr, name uintptr) uint64 {
				n := dl.GoString(name)
				f.loads = append(f.loads, n)
				return f.ids[n]
			},
		}, nil
	}
	return f
}

func TestCursorsLoadOnceAndSetOnTheWindow(t *testing.T) {
	f := newFakeCursors(map[string]uint64{"default": 11, "text": 12})
	w := &window{xid: 0x500}
	f.c.apply(w, platform.MouseCursorBasic)
	f.c.apply(w, platform.MouseCursorText)
	f.c.apply(w, platform.MouseCursorBasic) // cached
	if len(f.loads) != 2 {
		t.Errorf("loads %v, want one per cursor", f.loads)
	}
	want := [][2]uint64{{0x500, 11}, {0x500, 12}, {0x500, 11}}
	for i, d := range f.defined {
		if d != want[i] {
			t.Errorf("defined[%d] = %v, want %v", i, d, want[i])
		}
	}
}

func TestCursorsTryTheNamesInOrder(t *testing.T) {
	// The theme has only the old X name for the pointing hand.
	f := newFakeCursors(map[string]uint64{"hand2": 21})
	id, err := f.c.find(platform.MouseCursorClick, 24)
	if err != nil || id != 21 {
		t.Fatalf("find = %d, %v", id, err)
	}
	if len(f.loads) != 2 || f.loads[0] != "pointer" || f.loads[1] != "hand2" {
		t.Errorf("tried %v, want pointer then hand2", f.loads)
	}
}

func TestCursorNotFoundIsReportedOnceAndFallsBackToTheDefault(t *testing.T) {
	f := newFakeCursors(map[string]uint64{})
	_, err := f.c.find(platform.MouseCursorZoomIn, 24)
	assertRoute(t, err, "cursors.find.CursorNotFound")
	if appErr, _ := errors.FromError(err); appErr.Metadata["cursor"] != "MouseCursorZoomIn" {
		t.Errorf("metadata %v", appErr.Metadata)
	}
	// The window gets None, which is the cursor of its parent.
	w := &window{xid: 9}
	f.c.apply(w, platform.MouseCursorZoomIn)
	if len(f.defined) != 1 || f.defined[0] != [2]uint64{9, 0} {
		t.Errorf("defined %v", f.defined)
	}
	// The miss is cached: the theme is not asked again.
	n := len(f.loads)
	f.c.apply(w, platform.MouseCursorZoomIn)
	if len(f.loads) != n {
		t.Error("a cursor that the theme lacks was looked up again")
	}
}

func TestCursorLibraryFailureIsReportedOnce(t *testing.T) {
	f := newFakeCursors(nil)
	calls := 0
	f.c.load = func() (*xcursor, error) {
		calls++
		return nil, errors.New("LibraryLoadFailed", "x", "names", []string{"libXcursor.so.1"}, "dlerror", "missing")
	}
	_, err := f.c.find(platform.MouseCursorBasic, 24)
	assertRoute(t, err, "cursors.find.CursorLibraryFailed")
	f.c.apply(&window{xid: 1}, platform.MouseCursorBasic)
	f.c.apply(&window{xid: 1}, platform.MouseCursorText)
	if calls != 1 {
		t.Errorf("the library was loaded %d times after it failed", calls)
	}
	if !f.c.failed {
		t.Error("failed is not set")
	}
}

func TestHiddenCursorIsAnEmptyBitmap(t *testing.T) {
	f := newFakeCursors(nil)
	w := &window{xid: 3}
	f.c.apply(w, platform.MouseCursorNone)
	f.c.apply(w, platform.MouseCursorNone)
	if f.images != 1 {
		t.Errorf("%d bitmaps made, want 1", f.images)
	}
	if len(f.defined) != 2 || f.defined[0][1] != 777 || f.defined[1][1] != 777 {
		t.Errorf("defined %v", f.defined)
	}
	f.c.close()
	if len(f.freed) != 1 || f.freed[0] != 777 {
		t.Errorf("freed %v", f.freed)
	}
}

func TestCursorsCloseFreesTheLoadedOnes(t *testing.T) {
	f := newFakeCursors(map[string]uint64{"default": 11, "text": 12})
	for _, c := range []platform.MouseCursor{platform.MouseCursorBasic, platform.MouseCursorText} {
		if _, err := f.c.find(c, 24); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := f.c.find(platform.MouseCursorZoomIn, 24); err == nil { // a miss: nothing to free
		t.Fatal("a cursor that the theme lacks was found")
	}
	f.c.close()
	if len(f.freed) != 2 {
		t.Errorf("freed %v, want the two loaded cursors", f.freed)
	}
}

// assertRoute checks that err is an *errors.Error whose route contains route.
func assertRoute(t *testing.T, err error, route string) {
	t.Helper()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if !appErr.HasRoute(route) {
		t.Errorf("unexpected failure path: %s, want %s", appErr.Route(), route)
	}
}

func TestNewFailsWithoutAServer(t *testing.T) {
	_, err := New(":9999")
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != "X11OpenDisplayFailed" || !appErr.HasRoute("New.X11OpenDisplayFailed") {
		t.Fatalf("New(:9999) = %v, want X11OpenDisplayFailed via New", err)
	}
	if appErr.Metadata["display"] != ":9999" {
		t.Errorf("metadata %v", appErr.Metadata)
	}
}

func TestLoadFailedKeepsWhatTheLoaderReported(t *testing.T) {
	loader := errors.New("LibraryLoadFailed", "could not load the library", "names", []string{"libX11.so.6"}, "dlerror", "no such file")
	err := loadFailed("libX11.so.6", loader)
	assertRoute(t, err, "loadFailed.X11LoadFailed")
	appErr, _ := errors.FromError(err)
	if appErr.Metadata["library"] != "libX11.so.6" || appErr.Metadata["dlerror"] != "no such file" {
		t.Errorf("metadata %v", appErr.Metadata)
	}
}

func TestNewWindowFailsWhenTheServerCreatesNoWindow(t *testing.T) {
	created := 0
	p := &Platform{
		x: &xlib{
			createWindow: func(uintptr, uint64, int32, int32, uint32, uint32, uint32, int32, uint32, uintptr, uint64, uintptr) uint64 {
				created++
				return 0
			},
		},
		windows:   map[uint64]*window{},
		visualFor: func(uintptr) (uint32, error) { return 0, errors.New("EGLNoConfig", "no config") },
	}
	_, err := p.NewWindow(platform.DefaultWindowConfig())
	assertRoute(t, err, "Platform.NewWindow.X11WindowFailed")
	if created != 1 || len(p.windows) != 0 {
		t.Errorf("created %d windows, %d registered; want 1 and 0", created, len(p.windows))
	}
}

func TestCursorSizeAndThemeFollowTheSettingsAndTheScale(t *testing.T) {
	tests := []struct {
		name      string
		env       map[string]string
		theme     string
		size      int
		scale     float64
		wantSize  int32
		wantTheme []string
	}{
		{"defaults", nil, "", 0, 1, 24, nil},
		{"environment", map[string]string{"XCURSOR_SIZE": "32", "XCURSOR_THEME": "Adwaita"}, "", 0, 1, 32, []string{"Adwaita"}},
		{"bad size in the environment", map[string]string{"XCURSOR_SIZE": "big"}, "", 0, 1, 24, nil},
		{"settings win", map[string]string{"XCURSOR_SIZE": "32", "XCURSOR_THEME": "Adwaita"}, "Bibata", 48, 1, 48, []string{"Bibata"}},
		{"scale 2", nil, "", 0, 2, 48, nil},
		{"fractional scale", nil, "", 0, 1.5, 36, nil},
		{"settings and scale", nil, "Bibata", 36, 1.75, 63, []string{"Bibata"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakeCursors(map[string]uint64{"default": 11})
			f.c.getenv = func(k string) string { return tt.env[k] }
			f.c.themeName, f.c.baseSize = tt.theme, int32(tt.size)
			f.c.apply(&window{xid: 1, scale: tt.scale}, platform.MouseCursorBasic)
			if len(f.sizes) != 1 || f.sizes[0] != tt.wantSize {
				t.Errorf("sizes = %v, want [%d]", f.sizes, tt.wantSize)
			}
			if !reflect.DeepEqual(f.themes, tt.wantTheme) {
				t.Errorf("themes = %v, want %v", f.themes, tt.wantTheme)
			}
		})
	}
}

func TestCursorsAreLoadedAgainForAnotherScaleOrSettings(t *testing.T) {
	f := newFakeCursors(map[string]uint64{"default": 11})
	one, two := &window{xid: 1, scale: 1}, &window{xid: 2, scale: 2}
	f.c.p.windows = map[uint64]*window{1: one, 2: two}
	f.c.apply(one, platform.MouseCursorBasic)
	f.c.apply(two, platform.MouseCursorBasic)
	f.c.apply(one, platform.MouseCursorBasic) // cached
	if want := []int32{24, 48}; !reflect.DeepEqual(f.sizes, want) {
		t.Errorf("sizes = %v, want one load per size %v", f.sizes, want)
	}

	// New settings free the cursors and set them again on every window.
	f.defined, f.freed = nil, nil
	f.c.configure("Bibata", 32)
	if len(f.freed) != 2 {
		t.Errorf("freed %v, want the two cursors of the old settings", f.freed)
	}
	if len(f.defined) != 2 {
		t.Errorf("defined %v, want both windows to get a cursor", f.defined)
	}
	if got := f.themes; len(got) != 1 || got[0] != "Bibata" {
		t.Errorf("themes = %v", got)
	}
	got := slices.Sorted(slices.Values(f.sizes[len(f.sizes)-2:])) // the windows are visited in no order
	if !reflect.DeepEqual(got, []int32{32, 64}) {
		t.Errorf("sizes after the settings = %v, want 32 and 64", got)
	}
	// The same settings change nothing; settings without a theme go back to the default.
	f.defined = nil
	f.c.configure("Bibata", 32)
	if len(f.defined) != 0 {
		t.Error("the same settings set the cursors again")
	}
	f.c.configure("", 0)
	if got := f.themes; len(got) != 2 || got[1] != "default" {
		t.Errorf("themes = %v, want the default theme after the settings lost theirs", got)
	}
}

func TestKeepScreenOnIsUnavailableOnX11(t *testing.T) {
	var w window
	err := w.KeepScreenOn(true)
	appErr, ok := errors.FromError(err)
	if !ok || !appErr.HasRoute("IdleInhibitUnavailable") {
		t.Fatalf("err = %v, want the tag IdleInhibitUnavailable", err)
	}
	if err := w.SetIcon(ui.WindowIcon{Name: "x"}); err != nil {
		t.Errorf("SetIcon: %v", err)
	}
	w.SetContentType(ui.ContentTypeGame)
}
