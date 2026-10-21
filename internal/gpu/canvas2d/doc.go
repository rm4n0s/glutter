// Package canvas2d is the last renderer of the web (SPEC §9.6): it draws each frame with
// the software renderer (internal/gpu/soft) and puts the image into the page's 2D canvas
// with putImageData, so that an app shows even where neither WebGPU nor WebGL 2 works. A
// frame crosses into JavaScript in one js.CopyBytesToJS.
//
// The drawing and the composition over the background are plain Go (renderer.go) and run
// under scripts/test-wasm.sh and the normal tests; the canvas is touched only in the files
// that carry the js tag.
//
// Flutter: no counterpart (Flutter web's last resort is CanvasKit or Skwasm).
package canvas2d
