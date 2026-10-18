// Package dl is the one safe way to load C libraries and call into them
// without cgo: it wraps purego's Dlopen, Dlsym and callbacks, tags every
// failure, and offers the libc helpers (malloc, free, C strings) and the
// struct-layout test helper that the backends share.
//
// The rules for calling C through purego are in docs/SPEC.md §8.1. Only the UI
// thread calls into the libraries opened here.
//
// Flutter: no counterpart.
package dl
