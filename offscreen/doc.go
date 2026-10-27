// Package offscreen draws scenes into pixels in the app's own process, with no
// window and no GPU: a software ui.Rasterizer, a Detacher that makes a scene safe
// to draw on any goroutine, and a Surface that keeps the pixels of one view across
// frames and redraws only what changed (SPEC §9.9, D28). It is plain Go over the
// software renderer and builds for every platform.
//
// Flutter: no counterpart
package offscreen
