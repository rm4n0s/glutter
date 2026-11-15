// Package issues holds the reproductions of reported issues that the tests of every platform
// share: what to draw or do, a report of what came of it that can be written as text on a device
// and read on the host, and the check of the report. A fix is tried on one machine, and its
// renderer or platform code runs on the others too (Vulkan, OpenGL ES, Metal, Direct3D 12, WebGPU;
// Linux, Android, Windows, macOS, iOS, the web), so each platform has a test that runs the same
// reproduction and the same check: in the test process where a device can be opened there, in
// an app on the emulator, the Simulator or a browser page where it cannot.
//
// The package is plain Go with no OS tags, and it does not import testing: the apps of the device
// tests link it (examples/dev/android-parity, the page apps gpuwebtest and glwebtest).
package issues
