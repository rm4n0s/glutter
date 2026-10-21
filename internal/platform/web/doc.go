// Package web is the platform backend of the web (GOOS=js GOARCH=wasm, SPEC §23): one
// window that is a canvas in a host element of the page, frames from
// requestAnimationFrame, and the lifecycle and the settings of the browser.
//
// The files that talk to the browser carry the js tag; the rest (the size and device
// pixel ratio math, the config, the query parameters of the page, the detection of the
// operating system, the settings, the event queue) is plain Go that the normal test run
// and scripts/test-wasm.sh both test.
//
// Flutter: engine/src/flutter/lib/web_ui/lib/src/engine/view_embedder.
package web

//go:generate go run ../../../cmd/glutter-gen keys -reference ../../../reference/flutter -out "" -web-out keymap_gen.go
