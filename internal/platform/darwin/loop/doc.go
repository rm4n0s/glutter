// Package loop has the plain-Go parts of the run loop that macOS and iOS share (SPEC §29.3, §30.2): the heap of timers that one
// CFRunLoopTimer is set to, the merging of the frame requests of a window, the pacer that starts and stops the display link, and the
// gate that offers one tick of a display link's thread at a time to the loop. Nothing here calls the operating system, so the tests
// run on every host.
//
// Flutter: no counterpart
package loop
