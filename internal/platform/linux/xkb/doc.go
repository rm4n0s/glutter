// Package xkb binds libxkbcommon (through purego, SPEC §8.1) and turns key
// presses into Flutter key data: the physical key, the logical key and the text,
// for any keyboard layout, with dead keys and compose sequences. Wayland and X11
// both use it.
//
// Flutter: no counterpart (the embedders use GTK or the platform's own key code
// for this)
package xkb
