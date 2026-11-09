// Package iokit is the gamepad backend of macOS (SPEC §29.11): the pads that IOKit's HID manager reports, read as raw axes,
// buttons and hats, and the pads that have a HID driver (package gamepadhid), read through their reports.
//
// Everything that decides something is plain Go with no OS tag, so tables of elements and of values test it on any host: which
// HID elements make axes, buttons and hats (elements.go), how a value is scaled and a hat decoded, which devices are pads, the
// GUID and the name of a pad, and the Service that turns what a Manager says into what platform.PadHandler gets (service.go).
// The calls into IOKit are in the files with the darwin tag (iokit_darwin.go, transport_darwin.go), which make a Manager.
//
// The rules of which elements count, their order, the calibration of an axis and the decoding of a hat are those of SDL 3's
// src/joystick/darwin/SDL_iokitjoystick.c; the code here is a new Go version of them and is changed.
//
// Flutter: no counterpart
package iokit
