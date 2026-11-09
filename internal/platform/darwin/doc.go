// Package darwin holds what macOS and iOS share (SPEC §29.2): the loader of the system frameworks
// (dl), the Objective-C helpers (objcx), Core Foundation (cf), and the layout test helper for the Go
// mirrors of C structures (layouttest). Its sub-packages are called through purego, with no cgo.
// Only the engine and the platform backends import them, never a public package (SPEC §6.3 rule 1).
//
// Flutter: no counterpart.
package darwin
