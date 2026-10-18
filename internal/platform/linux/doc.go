// Package linux holds what the Linux backends share: the event loop, the key
// tables, system settings and the choice between Wayland and X11.
//
// Flutter: the Linux embedder (engine/src/flutter/shell/platform/linux).
package linux

//go:generate go run ../../../cmd/glutter-gen keys -reference ../../../reference/flutter -out keymap_gen.go
