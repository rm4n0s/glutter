//go:build amd64 || arm64

package goid

// Token is a value that tells the calling goroutine from every other goroutine
// that is alive. On amd64 and arm64 it is the address of the runtime's record
// of the goroutine (the "g" that the runtime keeps in a register or in thread
// local storage), read by a function in Go assembly (getg_amd64.s,
// getg_arm64.s) in about a nanosecond, at any stack depth. It is never 0.
//
// The record belongs to the goroutine for its whole life and the runtime never
// frees it, but it hands it to a new goroutine after the first one ended: do
// not keep a token longer than its goroutine lives.
//
// Flutter: no counterpart
func Token() uint64
