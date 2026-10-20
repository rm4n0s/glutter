// Package decor binds libdecor, the library that draws native title bars for
// Wayland compositors without server-side decorations (GNOME). libdecor is
// loaded at run time through purego and is optional (SPEC §8.1, §8.2).
//
// libdecor creates the xdg_surface and the xdg_toplevel of a window itself, so a
// window that uses it does not make its own. It draws the title bar and the
// borders in subsurfaces of the window's wl_surface, and reports the configure,
// close and commit events through callbacks. The callbacks are made once for the
// process (SPEC §8.1 rule 2) and find the Go object by the user data.
package decor
