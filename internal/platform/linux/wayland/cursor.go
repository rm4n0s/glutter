//go:build linux

package wayland

import (
	"log/slog"
	"os"
	"strconv"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/wl"
)

// defaultCursorSize is the size of the XCursor theme when XCURSOR_SIZE is not set.
const defaultCursorSize = 24

// cursorShapes maps a cursor to the shape that wp_cursor_shape_v1 names. The
// compositor draws it, with the user's theme, at the right scale.
var cursorShapes = map[platform.MouseCursor]wl.WpCursorShapeDeviceV1Shape{
	platform.MouseCursorUnspecified:           wl.WpCursorShapeDeviceV1ShapeDefault,
	platform.MouseCursorBasic:                 wl.WpCursorShapeDeviceV1ShapeDefault,
	platform.MouseCursorClick:                 wl.WpCursorShapeDeviceV1ShapePointer,
	platform.MouseCursorForbidden:             wl.WpCursorShapeDeviceV1ShapeNotAllowed,
	platform.MouseCursorWait:                  wl.WpCursorShapeDeviceV1ShapeWait,
	platform.MouseCursorProgress:              wl.WpCursorShapeDeviceV1ShapeProgress,
	platform.MouseCursorContextMenu:           wl.WpCursorShapeDeviceV1ShapeContextMenu,
	platform.MouseCursorHelp:                  wl.WpCursorShapeDeviceV1ShapeHelp,
	platform.MouseCursorText:                  wl.WpCursorShapeDeviceV1ShapeText,
	platform.MouseCursorVerticalText:          wl.WpCursorShapeDeviceV1ShapeVerticalText,
	platform.MouseCursorCell:                  wl.WpCursorShapeDeviceV1ShapeCell,
	platform.MouseCursorPrecise:               wl.WpCursorShapeDeviceV1ShapeCrosshair,
	platform.MouseCursorMove:                  wl.WpCursorShapeDeviceV1ShapeMove,
	platform.MouseCursorGrab:                  wl.WpCursorShapeDeviceV1ShapeGrab,
	platform.MouseCursorGrabbing:              wl.WpCursorShapeDeviceV1ShapeGrabbing,
	platform.MouseCursorNoDrop:                wl.WpCursorShapeDeviceV1ShapeNoDrop,
	platform.MouseCursorAlias:                 wl.WpCursorShapeDeviceV1ShapeAlias,
	platform.MouseCursorCopy:                  wl.WpCursorShapeDeviceV1ShapeCopy,
	platform.MouseCursorDisappearing:          wl.WpCursorShapeDeviceV1ShapeDefault,
	platform.MouseCursorAllScroll:             wl.WpCursorShapeDeviceV1ShapeAllScroll,
	platform.MouseCursorResizeLeftRight:       wl.WpCursorShapeDeviceV1ShapeEwResize,
	platform.MouseCursorResizeUpDown:          wl.WpCursorShapeDeviceV1ShapeNsResize,
	platform.MouseCursorResizeUpLeftDownRight: wl.WpCursorShapeDeviceV1ShapeNwseResize,
	platform.MouseCursorResizeUpRightDownLeft: wl.WpCursorShapeDeviceV1ShapeNeswResize,
	platform.MouseCursorResizeUp:              wl.WpCursorShapeDeviceV1ShapeNResize,
	platform.MouseCursorResizeDown:            wl.WpCursorShapeDeviceV1ShapeSResize,
	platform.MouseCursorResizeLeft:            wl.WpCursorShapeDeviceV1ShapeWResize,
	platform.MouseCursorResizeRight:           wl.WpCursorShapeDeviceV1ShapeEResize,
	platform.MouseCursorResizeUpLeft:          wl.WpCursorShapeDeviceV1ShapeNwResize,
	platform.MouseCursorResizeUpRight:         wl.WpCursorShapeDeviceV1ShapeNeResize,
	platform.MouseCursorResizeDownLeft:        wl.WpCursorShapeDeviceV1ShapeSwResize,
	platform.MouseCursorResizeDownRight:       wl.WpCursorShapeDeviceV1ShapeSeResize,
	platform.MouseCursorResizeColumn:          wl.WpCursorShapeDeviceV1ShapeColResize,
	platform.MouseCursorResizeRow:             wl.WpCursorShapeDeviceV1ShapeRowResize,
	platform.MouseCursorZoomIn:                wl.WpCursorShapeDeviceV1ShapeZoomIn,
	platform.MouseCursorZoomOut:               wl.WpCursorShapeDeviceV1ShapeZoomOut,
}

// cursors sets the cursor of the pointer: with wp_cursor_shape_v1 when the
// compositor has it, otherwise with an image from the XCursor theme (the theme
// and size of XCURSOR_THEME and XCURSOR_SIZE, as libwayland-cursor reads them).
// It belongs to the UI thread.
type cursors struct {
	p *Platform

	// The XCursor path. api is loaded on first use; a theme that cannot be used
	// is not tried again. A theme is loaded for each integer scale in use, at the
	// cursor size times the scale, so the cursor is sharp on every output.
	api      *waylandCursor
	themes   map[int32]uintptr // by scale
	failed   bool
	surface  *wl.WlSurface
	scale    int32                    // the buffer scale of the cursor surface
	wrappers map[uintptr]*wl.WlBuffer // the buffers of the themes, by address
	// themeName and baseSize come from the system settings (configure); empty and 0
	// mean XCURSOR_THEME, XCURSOR_SIZE and then 24.
	themeName string
	baseSize  int32
	// getenv reads the environment, and load opens libwayland-cursor; tests
	// replace them.
	getenv func(string) string
	load   func() (*waylandCursor, error)
}

func newCursors(p *Platform) *cursors {
	return &cursors{p: p, themes: map[int32]uintptr{}, wrappers: map[uintptr]*wl.WlBuffer{}, getenv: os.Getenv, load: loadWaylandCursor}
}

// apply sets the cursor for the pointer, which must be in a window: the serial of
// its enter event is what the compositor checks.
func (c *cursors) apply(pt *pointer, cursor platform.MouseCursor) {
	switch {
	case cursor == platform.MouseCursorNone:
		pt.obj.SetCursor(pt.serial, nil, 0, 0) // no surface hides the cursor
	case pt.shape != nil:
		shape, ok := cursorShapes[cursor]
		if !ok {
			shape = wl.WpCursorShapeDeviceV1ShapeDefault
		}
		pt.shape.SetShape(pt.serial, shape)
	default:
		if err := c.applyFromTheme(pt, cursor); err != nil {
			slog.Warn("the cursor could not be set", "cursor", cursor.String(), "error", err)
		}
	}
}

// applyFromTheme draws the cursor from the XCursor theme into the cursor surface.
func (c *cursors) applyFromTheme(pt *pointer, cursor platform.MouseCursor) error {
	if c.failed {
		return nil // already reported
	}
	scale := int32(1)
	if pt.focus != nil {
		scale = max(1, (pt.focus.scale+119)/120) // the integer scale above a fractional one
	}
	image, buffer, err := c.find(cursor, scale)
	if err != nil {
		return err
	}
	if c.surface == nil {
		c.surface = c.p.sess.compositor.CreateSurface()
	}
	wrapper, ok := c.wrappers[buffer]
	if !ok {
		wrapper = c.p.sess.conn.WrapBuffer(buffer)
		c.wrappers[buffer] = wrapper
	}
	if c.scale != scale {
		c.surface.SetBufferScale(scale)
		c.scale = scale
	}
	c.surface.Attach(wrapper, 0, 0)
	c.surface.Damage(0, 0, (int32(image.width)+scale-1)/scale, (int32(image.height)+scale-1)/scale)
	c.surface.Commit()
	// The hotspot is in the surface's coordinates, the image in pixels.
	pt.obj.SetCursor(pt.serial, c.surface, int32(image.hotspotX)/scale, int32(image.hotspotY)/scale)
	return nil
}

// find loads the theme if needed and returns the image of the cursor, with its
// buffer. It fails with CursorLibraryFailed when libwayland-cursor is unusable
// (reported once), CursorThemeFailed when the theme cannot be loaded (also once),
// and CursorNotFound when the theme has none of the names of the cursor.
func (c *cursors) find(cursor platform.MouseCursor, scale int32) (wlCursorImage, uintptr, error) {
	theme := c.themes[scale]
	if theme == 0 {
		var err error
		if theme, err = c.loadTheme(scale); err != nil {
			c.failed = true
			return wlCursorImage{}, 0, err
		}
	}
	names := linux.CursorNames(cursor)
	for _, name := range names {
		if image, buffer, ok := c.api.cursorImage(theme, name); ok {
			return image, buffer, nil
		}
	}
	return wlCursorImage{}, 0, errors.New("CursorNotFound", "the XCursor theme has no such cursor",
		"cursor", cursor.String(), "names", names)
}

// size is the cursor size in pixels at scale 1: the settings', else XCURSOR_SIZE,
// else 24.
func (c *cursors) size() int32 {
	if c.baseSize > 0 {
		return c.baseSize
	}
	if v := c.getenv("XCURSOR_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return int32(n)
		}
	}
	return defaultCursorSize
}

// name is the theme name: the settings', else XCURSOR_THEME, else "" for the default
// theme.
func (c *cursors) name() string {
	if c.themeName != "" {
		return c.themeName
	}
	return c.getenv("XCURSOR_THEME")
}

// loadTheme loads the theme for an integer scale: at the cursor size times the
// scale.
func (c *cursors) loadTheme(scale int32) (uintptr, error) {
	if c.api == nil {
		api, err := c.load()
		if err != nil {
			md := map[string]any{}
			if appErr, ok := errors.FromError(err); ok {
				md = appErr.Metadata
			}
			return 0, errors.New("CursorLibraryFailed", "could not load libwayland-cursor", "dlerror", md["dlerror"], "names", md["names"])
		}
		c.api = api
	}
	size := c.size() * scale
	name := uintptr(0) // NULL: the default theme
	if v := c.name(); v != "" {
		name = dl.CString(v)
		defer dl.Free(name)
	}
	theme := c.api.themeLoad(name, size, c.p.sess.shm.Ptr())
	if theme == 0 {
		return 0, errors.New("CursorThemeFailed", "wl_cursor_theme_load failed", "theme", c.name(), "size", size)
	}
	c.themes[scale] = theme
	return theme, nil
}

// configure sets the theme and the size from the system settings (SPEC §8.8); an
// empty name and a size of 0 mean the environment and then the defaults. A change
// loads the themes again, and the cursor over the window with the pointer is set
// again.
func (c *cursors) configure(theme string, size int) {
	if theme == c.themeName && int32(size) == c.baseSize {
		return
	}
	c.themeName, c.baseSize = theme, int32(max(size, 0))
	c.destroyThemes()
	c.failed = false
	if pt := c.p.pointer; pt != nil && pt.focus != nil {
		c.apply(pt, pt.focus.cursor)
	}
}

// destroyThemes destroys the themes with their buffers.
func (c *cursors) destroyThemes() {
	for scale, theme := range c.themes {
		c.api.themeDestroy(theme)
		delete(c.themes, scale)
	}
	clear(c.wrappers)
}

// scaleChanged sets the cursor again when the window with the pointer got another
// scale: the theme for that scale draws it.
func (c *cursors) scaleChanged(w *window) {
	if pt := c.p.pointer; pt != nil && pt.focus == w && pt.shape == nil && w.cursor != platform.MouseCursorNone {
		c.apply(pt, w.cursor)
	}
}

// close destroys the cursor surface and the theme with its buffers.
func (c *cursors) close() {
	if c.surface != nil {
		c.surface.Destroy()
		c.surface = nil
	}
	c.destroyThemes()
}
