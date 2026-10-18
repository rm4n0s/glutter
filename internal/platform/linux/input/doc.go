// Package input turns what the Linux windowing systems report about pointers and
// touch screens into ui.PointerData, the same way for Wayland and X11. It has no
// dependency on either: the backends feed it plain events (positions, buttons,
// scroll axes, gestures, fingers) and send on what it returns.
//
// Flutter: no counterpart (the embedders do this on each platform)
package input
