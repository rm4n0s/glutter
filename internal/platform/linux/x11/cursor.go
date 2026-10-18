package x11

import (
	"log/slog"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// cursors loads cursors from the XCursor theme (libXcursor, which reads
// XCURSOR_THEME and XCURSOR_SIZE) and sets them on windows. A cursor is loaded
// once. It belongs to the UI thread.
type cursors struct {
	p *Platform
	// loaded maps a cursor to the X cursor ID; 0 is cached too, for a cursor the
	// theme lacks, so that it is reported once.
	loaded    map[platform.MouseCursor]uint64
	invisible uint64
	// load opens libXcursor; tests replace it.
	load func() (*xcursor, error)
	// failed is whether libXcursor is unusable (reported once).
	failed bool
}

func newCursors(p *Platform) *cursors {
	return &cursors{p: p, loaded: map[platform.MouseCursor]uint64{}, load: loadXcursorOnce}
}

// apply sets the cursor over the window. A cursor that cannot be loaded leaves the
// window with the cursor of its parent (the root's), and is reported in the log.
func (c *cursors) apply(w *window, cursor platform.MouseCursor) {
	id, err := c.find(cursor)
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
func (c *cursors) find(cursor platform.MouseCursor) (uint64, error) {
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
	if id, ok := c.loaded[cursor]; ok {
		return id, nil
	}
	names := linux.CursorNames(cursor)
	for _, name := range names {
		cname := dl.CString(name)
		id := lib.libraryLoadCursor(c.p.display, cname)
		dl.Free(cname)
		if id != 0 {
			c.loaded[cursor] = id
			return id, nil
		}
	}
	c.loaded[cursor] = 0
	return 0, errors.New("CursorNotFound", "the XCursor theme has no such cursor", "cursor", cursor.String(), "names", names)
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
	for _, id := range c.loaded {
		if id != 0 {
			c.p.x.freeCursor(c.p.display, id)
		}
	}
	if c.invisible != 0 {
		c.p.x.freeCursor(c.p.display, c.invisible)
	}
	c.loaded, c.invisible = map[platform.MouseCursor]uint64{}, 0
}
