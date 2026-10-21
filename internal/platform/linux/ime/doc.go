// Package ime connects a window system backend to an input method framework
// over D-Bus (IBus or Fcitx5): the same way for X11, and for Wayland on a
// compositor without text-input-v3 (SPEC §8.3, §8.6). The calls to the
// framework block, so they run on a goroutine of their own; what comes back
// is handed to the UI thread through a Poster.
//
// Flutter: no counterpart
package ime
