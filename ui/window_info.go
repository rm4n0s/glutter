package ui

// DecorationMode says who draws the title bar and the borders of a window.
//
// Flutter: no counterpart
type DecorationMode int

// The values of DecorationMode.
const (
	// DecorationNone is a window without a title bar: a borderless window.
	//
	// Flutter: no counterpart
	DecorationNone DecorationMode = iota
	// DecorationServerSide is a window whose title bar the compositor or the window
	// manager draws.
	//
	// Flutter: no counterpart
	DecorationServerSide
	// DecorationLibdecor is a window whose title bar libdecor draws, around the
	// window's surface (Wayland without server-side decorations, such as GNOME).
	//
	// Flutter: no counterpart
	DecorationLibdecor
	// DecorationApp is a window whose title bar the app draws: with
	// WindowConfig.Decorations set to DecorationsApp, or when nothing else is
	// available. The window has to show a WindowDecoration, or its own title bar.
	//
	// Flutter: no counterpart
	DecorationApp
)

// Name is the name of the mode.
//
// Flutter: no counterpart
func (m DecorationMode) Name() string {
	switch m {
	case DecorationNone:
		return "none"
	case DecorationServerSide:
		return "serverSide"
	case DecorationLibdecor:
		return "libdecor"
	case DecorationApp:
		return "app"
	}
	return "unknown"
}

// String is "DecorationMode." and the name.
//
// Flutter: no counterpart
func (m DecorationMode) String() string { return "DecorationMode." + m.Name() }

// WindowCapabilities are the things that the compositor lets the user do with a
// window, as xdg_toplevel.wm_capabilities reports them. The zero value is
// "nothing"; WindowInfo says whether the compositor reported them at all.
//
// Flutter: no counterpart
type WindowCapabilities struct {
	// WindowMenu is whether the window can show the window menu.
	WindowMenu bool
	// Maximize is whether the window can be maximized.
	Maximize bool
	// Fullscreen is whether the window can be made fullscreen.
	Fullscreen bool
	// Minimize is whether the window can be minimized.
	Minimize bool
}

// AllWindowCapabilities is every capability: what a window has when the
// compositor does not say (xdg_wm_base below version 5).
//
// Flutter: no counterpart
var AllWindowCapabilities = WindowCapabilities{WindowMenu: true, Maximize: true, Fullscreen: true, Minimize: true}

// WindowInfo is what the platform tells about the decorations and the state of a
// window. A title bar that the app draws needs all of it.
//
// Flutter: no counterpart
type WindowInfo struct {
	// Decoration is who draws the title bar.
	Decoration DecorationMode
	// Capabilities are what the user may do with the window.
	Capabilities WindowCapabilities
	// Title is the title of the window.
	Title string
	// Maximized is whether the window fills the work area.
	Maximized bool
	// Fullscreen is whether the window fills the screen.
	Fullscreen bool
	// Activated is whether the window has the keyboard focus.
	Activated bool
	// Tiled is whether the compositor tiles the window: its edges touch other
	// windows or the screen, so it has no borders to resize from.
	Tiled bool
}

// ResizeEdge is an edge or a corner of a window that the user can drag to
// resize it.
//
// Flutter: no counterpart
type ResizeEdge int

// The values of ResizeEdge.
const (
	// ResizeEdgeTop is the top edge.
	//
	// Flutter: no counterpart
	ResizeEdgeTop ResizeEdge = iota
	// ResizeEdgeBottom is the bottom edge.
	//
	// Flutter: no counterpart
	ResizeEdgeBottom
	// ResizeEdgeLeft is the left edge.
	//
	// Flutter: no counterpart
	ResizeEdgeLeft
	// ResizeEdgeRight is the right edge.
	//
	// Flutter: no counterpart
	ResizeEdgeRight
	// ResizeEdgeTopLeft is the top left corner.
	//
	// Flutter: no counterpart
	ResizeEdgeTopLeft
	// ResizeEdgeTopRight is the top right corner.
	//
	// Flutter: no counterpart
	ResizeEdgeTopRight
	// ResizeEdgeBottomLeft is the bottom left corner.
	//
	// Flutter: no counterpart
	ResizeEdgeBottomLeft
	// ResizeEdgeBottomRight is the bottom right corner.
	//
	// Flutter: no counterpart
	ResizeEdgeBottomRight
)

// WindowActionKind is what a title bar asks the window to do.
//
// Flutter: no counterpart
type WindowActionKind int

// The values of WindowActionKind.
const (
	// WindowActionMove starts moving the window with the pointer that is down.
	//
	// Flutter: no counterpart
	WindowActionMove WindowActionKind = iota
	// WindowActionResize starts resizing the window from an edge.
	//
	// Flutter: no counterpart
	WindowActionResize
	// WindowActionShowMenu shows the window menu at a position.
	//
	// Flutter: no counterpart
	WindowActionShowMenu
	// WindowActionMinimize minimizes the window.
	//
	// Flutter: no counterpart
	WindowActionMinimize
	// WindowActionMaximize maximizes the window.
	//
	// Flutter: no counterpart
	WindowActionMaximize
	// WindowActionRestore restores a maximized window.
	//
	// Flutter: no counterpart
	WindowActionRestore
	// WindowActionClose asks to close the window, as the user's close button does.
	//
	// Flutter: no counterpart
	WindowActionClose
	// WindowActionFullscreen makes the window fullscreen.
	//
	// Flutter: widgets/_window.dart RegularWindowController.setFullscreen
	WindowActionFullscreen
	// WindowActionUnfullscreen takes the window out of fullscreen.
	//
	// Flutter: widgets/_window.dart RegularWindowController.setFullscreen
	WindowActionUnfullscreen
	// WindowActionActivate asks for the window to be shown at its size and position,
	// out of the minimized state, and to get the focus. The platform may refuse the
	// focus and the stacking order.
	//
	// Flutter: widgets/_window.dart RegularWindowController.activate
	WindowActionActivate
)

// WindowAction is a request of a title bar that the app draws.
//
// Flutter: no counterpart
type WindowAction struct {
	// Kind is what to do.
	Kind WindowActionKind
	// Edge is the edge that WindowActionResize drags.
	Edge ResizeEdge
	// Position is where WindowActionShowMenu opens the menu, in logical pixels of
	// the window.
	Position Offset
}

// WindowHost is what the engine implements to carry out the actions of a title
// bar that the app draws. A host without it ignores them.
//
// Flutter: no counterpart (the window controller of widgets/_window.dart)
type WindowHost interface {
	// PerformWindowAction does the action for the window of the view. Moves and
	// resizes need the input event that is going on (the last pointer press), so
	// call it from the handler of that event.
	PerformWindowAction(view ViewID, action WindowAction)
}
