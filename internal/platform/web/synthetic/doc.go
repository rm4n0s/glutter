// Package synthetic dispatches synthetic DOM events at the canvas of a Glutter app, the way the
// browser would deliver real ones: the debug-only script of the web platform (GLUTTER_DEV_EVENTS,
// for proof screenshots) and the browser tests use it. It only builds for js/wasm.
//
// A listener that the browser calls while Go is calling dispatchEvent runs on the goroutine that
// dispatched the event, and the listeners of the web platform that wait for the UI goroutine's
// answer (keydown, keyup, wheel) wait for it: so the events must be dispatched from a goroutine
// other than the UI goroutine, which is the main one (SPEC §23.3).
//
// Flutter: no counterpart
package synthetic
