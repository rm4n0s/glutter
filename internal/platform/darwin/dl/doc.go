// Package dl is the one safe way to load the system frameworks of macOS and iOS and to call into them
// without cgo: it wraps purego's Dlopen, Dlsym and callbacks, tags every failure, and offers the libc
// helpers (malloc, free, C strings) that the backends share. It is the counterpart of
// internal/platform/linux/dl (SPEC §29.2).
//
// A framework is opened by its absolute path under /System/Library/Frameworks, never from a
// DYLD_* path or from the program's folder, so a library planted next to the program is not loaded.
// Only the UI thread calls into the libraries opened here (SPEC §29.3).
//
// The package has files for darwin only; this file keeps the package listed on every OS.
//
// Flutter: no counterpart.
package dl
