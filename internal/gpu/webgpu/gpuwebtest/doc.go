// Command gpuwebtest is the program of the browser tests of the WebGPU renderer (step 21.5,
// SPEC §9.5, §17). The query parameter "mode" says what it does, and it posts what it saw to the
// test server as a result of kind "gpuweb":
//
//   - parity: draws every proof scene with the explicit renderer over WebGPU, reads the frame
//     back (a read that finishes in a later task) and compares it with the software renderer's,
//     in the same program; the snapshot (soft) is compared too, and the browser's validation
//     errors are collected
//   - unavailable: makes WebGPU fail in each of the ways it can (no navigator.gpu, no adapter, no
//     device, a rejected shader, no answer in time) and asks for swapchains that cannot be made,
//     and reports the tags and routes
//   - loss: destroys the GPUDevice under a running renderer, twice, and reports what the renderer
//     did: the first loss is recovered on a device that was asked for without waiting, the second
//     within ten seconds ends the frame with GPUDeviceLost
//   - choose: runs a small Material app through the engine and reports the kind of context its
//     canvas ended up with and what RunApp returned (the query parameter "poison" hides
//     navigator.gpu, and "poisongl" makes getContext("webgl2") answer null)
//
// It only does something on the web; elsewhere it says so and exits. It is built by the tests with
// the web build tool (webtool.Build), not run by hand.
//
// Flutter: no counterpart
package main
