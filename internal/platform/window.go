package platform

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// WindowKind is the kind of a window. The zero value, WindowKindUnspecified,
// means WindowKindRegular.
//
// Flutter: widgets/_window.dart WindowArchetype
type WindowKind int

// The values of WindowKind.
const (
	// WindowKindUnspecified is the zero value and means WindowKindRegular.
	//
	// Flutter: no counterpart
	WindowKindUnspecified WindowKind = iota
	// WindowKindRegular is an ordinary top-level window.
	//
	// Flutter: widgets/_window.dart WindowArchetype.regular
	WindowKindRegular
	// WindowKindDialog is a dialog, optionally modal, of a parent window.
	//
	// Flutter: widgets/_window.dart WindowArchetype.dialog
	WindowKindDialog
	// WindowKindBorderless is a window without decorations, such as a
	// tooltip-style window.
	//
	// Flutter: no counterpart
	WindowKindBorderless
)

// WindowState is the state of a window. The zero value, WindowStateUnspecified,
// means WindowStateNormal.
//
// Flutter: no counterpart
type WindowState int

// The values of WindowState.
const (
	// WindowStateUnspecified is the zero value and means WindowStateNormal.
	//
	// Flutter: no counterpart
	WindowStateUnspecified WindowState = iota
	// WindowStateNormal is a window that is neither maximized, minimized nor
	// fullscreen.
	//
	// Flutter: no counterpart
	WindowStateNormal
	// WindowStateMaximized is a maximized window.
	//
	// Flutter: no counterpart
	WindowStateMaximized
	// WindowStateMinimized is a minimized (iconified) window.
	//
	// Flutter: no counterpart
	WindowStateMinimized
	// WindowStateFullscreen is a fullscreen window.
	//
	// Flutter: no counterpart
	WindowStateFullscreen
)

// Decorations is how the title bar and the borders of a window are made. The
// zero value, DecorationsUnspecified, means DecorationsAuto.
//
// Flutter: no counterpart
type Decorations int

// The values of Decorations.
const (
	// DecorationsUnspecified is the zero value and means DecorationsAuto.
	//
	// Flutter: no counterpart
	DecorationsUnspecified Decorations = iota
	// DecorationsAuto uses the first of these that works: the compositor's own
	// (server-side), libdecor, and the title bar of the framework (SPEC §8.2).
	//
	// Flutter: no counterpart
	DecorationsAuto
	// DecorationsApp skips the first two: the app places a WindowDecoration or its
	// own title bar.
	//
	// Flutter: no counterpart
	DecorationsApp
	// DecorationsLibdecor uses libdecor even where the compositor has its own title
	// bars; without libdecor it is DecorationsAuto. It is a test aid
	// (GLUTTER_DECORATIONS=libdecor): the apps ask for DecorationsAuto or
	// DecorationsApp.
	//
	// Flutter: no counterpart
	DecorationsLibdecor
)

// WindowConfig describes a window to create.
//
// Flutter: widgets/_window.dart WindowController
type WindowConfig struct {
	// Kind is the kind of window.
	Kind WindowKind
	// Parent is the view of the parent window of a dialog.
	Parent ui.ViewID
	// HasParent is whether Parent is set.
	HasParent bool
	// Modal is whether a dialog blocks its parent.
	Modal bool
	// Title is the window title.
	Title string
	// AppID is the application ID: the Wayland app_id or the X11 WM_CLASS. It
	// must match the app's .desktop file.
	AppID string
	// Size is the initial size in logical pixels. A zero size lets the backend
	// choose.
	Size ui.Size
	// MinSize is the smallest size in logical pixels; zero means no minimum.
	MinSize ui.Size
	// MaxSize is the largest size in logical pixels; zero means no maximum.
	MaxSize ui.Size
	// Resizable is whether the user can resize the window. None means the
	// default, true; use IsResizable to read it.
	Resizable foundation.Opt[bool]
	// Decorated is whether the window has a title bar and borders. None means
	// the default, true; use IsDecorated to read it.
	Decorated foundation.Opt[bool]
	// State is the initial window state.
	State WindowState
	// Decorations is how the title bar is made; it does not matter when Decorated
	// is false.
	Decorations Decorations
	// Icon is the icon of the window; the zero value is none.
	Icon ui.WindowIcon
	// ContentType is the hint about what the window shows.
	ContentType ui.ContentType
}

// IsResizable is whether the user can resize the window; the default is true.
//
// Flutter: no counterpart
func (c WindowConfig) IsResizable() bool { return c.Resizable.OrElse(true) }

// IsDecorated is whether the window has a title bar and borders; the default
// is true.
//
// Flutter: no counterpart
func (c WindowConfig) IsDecorated() bool { return c.Decorated.OrElse(true) }

// DefaultWindowConfig returns a WindowConfig with the defaults: a regular,
// resizable, decorated window of a size the backend chooses. It is the zero
// value.
//
// Flutter: no counterpart
func DefaultWindowConfig() WindowConfig { return WindowConfig{} }

// ViewMetrics describes the size, scale and system areas of a view. Only
// the size and the scale are required; the engine fills in the rest from the
// display and the defaults.
//
// Flutter: ui/window.dart ViewConfiguration
type ViewMetrics struct {
	// PhysicalSize is the size of the view in physical pixels.
	PhysicalSize ui.Size
	// DevicePixelRatio is the number of physical pixels per logical pixel.
	DevicePixelRatio float64
	// ViewPadding is the area the system covers, in physical pixels.
	ViewPadding ui.ViewPadding
	// ViewInsets is the area covered by system UI the app can draw below.
	ViewInsets ui.ViewPadding
	// SystemGestureInsets is the area where the system handles gestures.
	SystemGestureInsets ui.ViewPadding
	// Display is the display the view is on. None means the engine chooses
	// (the primary display).
	Display foundation.Opt[int]
	// Window says who draws the title bar and what state the window is in. A
	// backend that does not know leaves the zero value.
	Window ui.WindowInfo
}

// Display describes one connected display.
//
// Flutter: ui/window.dart Display
type Display struct {
	// ID identifies the display.
	ID int
	// Name is the name the compositor or X server gives the display.
	Name string
	// Bounds is the display's position and size in logical pixels.
	Bounds ui.Rect
	// DevicePixelRatio is the number of physical pixels per logical pixel.
	DevicePixelRatio float64
	// RefreshRate is the refresh rate in frames per second.
	RefreshRate float64
	// Primary is whether this is the primary display.
	Primary bool
}
