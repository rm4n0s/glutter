package x11

import (
	"log/slog"
	"math"
	"os"
	"strconv"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// defaultCursorSize is the size of the XCursor theme when neither the settings nor
// XCURSOR_SIZE say.
const defaultCursorSize = 24

// cursorKey identifies a loaded cursor: the same cursor is loaded again for each size
// (the size of the cursor times the scale of the monitor of the window).
type cursorKey struct {
	cursor platform.MouseCursor
	size   int32
}

// cursors loads cursors from the XCursor theme (libXcursor) and sets them on
// windows. The theme and the size come from the system settings (configure), else
// from XCURSOR_THEME and XCURSOR_SIZE; a window on a monitor with a scale gets the
// size times the scale. A cursor is loaded once for each size. It belongs to the UI
// thread.
type cursors struct {
	p *Platform
	// loaded maps a cursor and size to the X cursor ID; 0 is cached too, for a cursor
	// the theme lacks, so that it is reported once.
	loaded    map[cursorKey]uint64
	invisible uint64
	// themeName and baseSize come from the settings; empty and 0 mean the
	// environment. themeApplied is the theme that XcursorSetTheme was last given.
	themeName    string
	baseSize     int32
	themeApplied string
	// getenv reads the environment; tests replace it.
	getenv func(string) string
	// load opens libXcursor; tests replace it.
	load func() (*xcursor, error)
	// failed is whether libXcursor is unusable (reported once).
	failed bool
}

func newCursors(p *Platform) *cursors {
	return &cursors{p: p, loaded: map[cursorKey]uint64{}, load: loadXcursorOnce, getenv: os.Getenv}
}

// apply sets the cursor over the window. A cursor that cannot be loaded leaves the
// window with the cursor of its parent (the root's), and is reported in the log.
func (c *cursors) apply(w *window, cursor platform.MouseCursor) {
	id, err := c.find(cursor, c.sizeFor(w))
	if err != nil {
		slog.Warn("the cursor could not be loaded", "cursor", cursor.String(), "error", err)
		id = 0
	}
	c.p.x.defineCursor(c.p.display, w.xid, id) // 0 is None: the parent's cursor
	c.p.x.flush(c.p.display)
}

// find returns the X cursor for the cursor, loading it on first use. It fails with
// CursorLibraryFailed when libXcursor cannot be loaded (reported once), and
// CursorNotFound when the theme has none of the names of the cursor.
func (c *cursors) find(cursor platform.MouseCursor, size int32) (uint64, error) {
	if c.failed {
		return 0, nil
	}
	lib, err := c.load()
	if err != nil {
		c.failed = true
		md := map[string]any{}
		if appErr, ok := errors.FromError(err); ok {
			md = appErr.Metadata
		}
		return 0, errors.New("CursorLibraryFailed", "could not load libXcursor", "dlerror", md["dlerror"], "names", md["names"])
	}
	if cursor == platform.MouseCursorNone {
		return c.hidden(), nil
	}
	key := cursorKey{cursor, size}
	if id, ok := c.loaded[key]; ok {
		return id, nil
	}
	c.useTheme(lib, size)
	names := linux.CursorNames(cursor)
	for _, name := range names {
		cname := dl.CString(name)
		id := lib.libraryLoadCursor(c.p.display, cname)
		dl.Free(cname)
		if id != 0 {
			c.loaded[key] = id
			return id, nil
		}
	}
	c.loaded[key] = 0
	return 0, errors.New("CursorNotFound", "the XCursor theme has no such cursor", "cursor", cursor.String(), "names", names)
}

// useTheme tells libXcursor the theme and the size of the cursors to load. Without a
// size it would take one from Xft.dpi, which a window's scale already counts.
func (c *cursors) useTheme(lib *xcursor, size int32) {
	name := c.name()
	if name == "" && c.themeApplied != "" {
		name = "default" // the settings no longer name a theme
	}
	if name != "" && name != c.themeApplied {
		cname := dl.CString(name)
		lib.setTheme(c.p.display, cname)
		dl.Free(cname)
		c.themeApplied = name
	}
	lib.setDefaultSize(c.p.display, size)
}

// name is the theme name: the settings', else XCURSOR_THEME, else "" (libXcursor
// chooses).
func (c *cursors) name() string {
	if c.themeName != "" {
		return c.themeName
	}
	return c.getenv("XCURSOR_THEME")
}

// sizeFor is the size in pixels of the cursors of a window: the settings' size, else
// XCURSOR_SIZE, else 24, times the scale of the window.
func (c *cursors) sizeFor(w *window) int32 {
	size := int32(defaultCursorSize)
	switch {
	case c.baseSize > 0:
		size = c.baseSize
	default:
		if n, err := strconv.Atoi(c.getenv("XCURSOR_SIZE")); err == nil && n > 0 {
			size = int32(n)
		}
	}
	if w != nil && w.scale > 1 {
		return int32(math.Round(float64(size) * w.scale))
	}
	return size
}

// configure sets the theme and the size from the system settings (SPEC §8.8); an
// empty name and a size of 0 mean the environment. A change loads the cursors again
// and sets them on the windows.
//
// TODO(step 14.5): the platform calls it with the settings it reads and follows.
func (c *cursors) configure(theme string, size int) {
	if theme == c.themeName && int32(size) == c.baseSize {
		return
	}
	c.themeName, c.baseSize = theme, int32(max(size, 0))
	c.freeLoaded()
	for _, w := range c.p.windows {
		c.apply(w, w.cursor)
	}
}

// hidden returns a cursor that shows nothing: the empty bitmap used as both the
// shape and the mask of a core cursor, which every X server understands.
func (c *cursors) hidden() uint64 {
	if c.invisible != 0 {
		return c.invisible
	}
	p := c.p
	data := dl.Calloc(1, 8) // eight clear bits
	color := dl.Calloc(1, 16)
	defer dl.Free(data)
	defer dl.Free(color)
	bitmap := p.x.createBitmap(p.display, p.root, data, 1, 1)
	if bitmap == 0 {
		return 0
	}
	c.invisible = p.x.createPixmapCursor(p.display, bitmap, bitmap, color, color, 0, 0)
	p.x.freePixmap(p.display, bitmap)
	return c.invisible
}

// close frees the cursors.
func (c *cursors) close() {
	c.freeLoaded()
	if c.invisible != 0 {
		c.p.x.freeCursor(c.p.display, c.invisible)
	}
	c.invisible = 0
}

// freeLoaded frees the cursors that were loaded from the theme. A window that still
// has one keeps drawing it until it gets another.
func (c *cursors) freeLoaded() {
	for _, id := range c.loaded {
		if id != 0 {
			c.p.x.freeCursor(c.p.display, id)
		}
	}
	c.loaded = map[cursorKey]uint64{}
}
