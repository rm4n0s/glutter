// Command glwebtest is the program of the browser tests of the WebGL 2 renderer (step 21.4,
// SPEC §9.1, §17). The query parameter "mode" says what it does, and it posts what it saw to the
// test server as a result of kind "glweb":
//
//   - parity: draws every proof scene with the gles renderer on WebGL 2, reads the frame back
//     and compares it with the software renderer's, in the same program
//   - unavailable: asks for a context on canvases that cannot give one and reports the tags
//   - loss: loses the context with WEBGL_lose_context, has it restored, and reports what the
//     renderer did in each state
//   - choose: runs a small Material app through the engine and reports the kind of context its
//     canvas ended up with, and what RunApp returned (the query parameter "poison" makes
//     getContext("webgl2") answer null, as a browser without WebGL 2 does)
//   - restore: the same app, with the context lost and restored while it is idle, counting the
//     animation frames that were asked for
//
// It only does something on the web; elsewhere it says so and exits. It is built by the tests with
// the web build tool (webtool.Build), not run by hand.
//
// Flutter: no counterpart
package main
