// Command canvaswebtest is the program of the browser tests of the 2D canvas renderer (step 49.1,
// SPEC §9.6, §17). The query parameter "mode" says what it does, and it posts what it saw to the
// test server as a result of kind "canvasweb":
//
//   - issue: runs the reproduction of a reported issue (internal/issues) that the query parameter
//     "issue" names with the canvas2d renderer into a canvas of the page, and reports the lines of
//     its report: soft-mipmaps, and guard-allocs (step 52.1), which draws nothing and runs in the
//     page's Go with no canvas
//
// It only does something on the web; elsewhere it says so and exits. It is built by the tests with
// the web build tool (webtool.Build), not run by hand.
//
// Flutter: no counterpart
package main
