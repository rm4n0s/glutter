// Package layouttest checks the Go mirrors of C and Objective-C structures that the macOS and iOS
// backends pass to and get from the system (NSRect, CGPoint, CVTimeStamp, AudioStreamBasicDescription,
// ...) against the headers of the SDK (SPEC §29.2, §8.1 rule 3).
//
// A test describes what Go believes (the size and field offsets of a mirror read by reflection, the
// value of each constant). The package writes a C file of _Static_assert lines from the description
// and compiles it with `xcrun clang -fsyntax-only` against the SDK of the Command Line Tools or
// Xcode. It starts no process but the compiler; a wrong mirror is a compile error that names the
// structure and the field. It is skipped, with the install command, when xcrun or a header is
// missing, which includes every host that is not a Mac.
//
// The package has no OS tag, so it is listed on every OS; its compiler tests run on a Mac only.
// Windows has its own (internal/platform/windows/win32/layouttest, which uses mingw-w64).
//
// Flutter: no counterpart
package layouttest
