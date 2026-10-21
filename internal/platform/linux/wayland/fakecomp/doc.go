// Package fakecomp is a tiny fake Wayland compositor for tests: it speaks the
// wire protocol over a socket pair, answers the requests that the tests send,
// and logs them. It is written by hand from the protocol description and is
// independent of the generated bindings, so it can check them. It needs no
// display.
//
// Flutter: no counterpart.
package fakecomp
