// Package goid tells goroutines apart. Go gives no way to ask for the number of
// the running goroutine, so it is read from the header of runtime.Stack. The
// framework uses it to find out whether a call comes from the UI goroutine
// (SPEC §13); it is not on any path that runs for every widget or render
// object.
//
// Flutter: no counterpart (Dart has isolates, not goroutines)
package goid

import "runtime"

// ID is the number of the calling goroutine.
func ID() uint64 {
	var buf [32]byte
	n := runtime.Stack(buf[:], false)
	// The header is "goroutine 123 [running]:".
	const prefix = "goroutine "
	if n <= len(prefix) {
		return 0
	}
	var id uint64
	for _, c := range buf[len(prefix):n] {
		if c < '0' || c > '9' {
			break
		}
		id = id*10 + uint64(c-'0')
	}
	return id
}
