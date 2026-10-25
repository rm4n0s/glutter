// Package layouttest checks the Go mirrors of Win32 and COM structures, and the positions of COM
// methods in their vtables, against the mingw-w64 headers (SPEC §25.2, §6.3 rule 11).
//
// A test describes what Go believes (the size and field offsets of a mirror read by reflection, the
// index of each method, the value of each constant). The package writes a C file of _Static_assert
// lines from the description and compiles it with `x86_64-w64-mingw32-gcc -fsyntax-only`
// (-DCOBJMACROS, for the C declarations of the vtables) against the real headers. It starts no
// process but the compiler, so it needs no Wine and runs on the Linux host; a wrong mirror is a
// compile error that names the structure and the field. It is skipped, with the install command,
// when the compiler or a header is missing.
//
// The package has no OS tag, so it is listed on every OS; its compiler tests run on the host only.
//
// Flutter: no counterpart
package layouttest
