package gpu

// SurfaceKind says which windowing system a SurfaceHandle belongs to, so that a
// GPU backend knows what Display and Window mean.
//
// Flutter: no counterpart
type SurfaceKind string

// The kinds of surface.
const (
	// SurfaceKindUnspecified is the zero value: no surface.
	//
	// Flutter: no counterpart
	SurfaceKindUnspecified SurfaceKind = ""
	// SurfaceKindWayland is a window of a Wayland compositor. Display is the
	// connection and Window the native window that the backend renders into.
	//
	// Flutter: no counterpart
	SurfaceKindWayland SurfaceKind = "wayland"
	// SurfaceKindX11 is a window of an X server. Display is the connection and
	// Window the window ID.
	//
	// Flutter: no counterpart
	SurfaceKindX11 SurfaceKind = "x11"
	// SurfaceKindWeb is the canvas of a web page. Window is the ID of the view, which
	// names the canvas (the attribute data-glutter-view); Display and Surface are zero.
	//
	// Flutter: no counterpart
	SurfaceKindWeb SurfaceKind = "web"
	// SurfaceKindFake is the surface of the scripted test platform. No GPU
	// backend renders into it.
	//
	// Flutter: no counterpart
	SurfaceKindFake SurfaceKind = "fake"
)

// SurfaceHandle identifies a native drawing surface. A platform backend creates
// it and only a GPU backend looks inside. Display, Window and Surface hold native
// addresses or IDs whose meaning depends on Kind; they are opaque to everyone
// else.
//
// Flutter: no counterpart
type SurfaceHandle struct {
	// Kind says how to read Display and Window.
	Kind SurfaceKind
	// Display is the native connection to the windowing system.
	Display uintptr
	// Window is the native window that EGL draws into: the wl_egl_window on Wayland.
	Window uintptr
	// Surface is the native surface that the explicit APIs (Vulkan) make their own
	// surface from: the wl_surface on Wayland, the window ID again on X11.
	Surface uintptr
}
