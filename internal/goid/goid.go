// Package goid tells goroutines apart. Go gives no way to ask for the number of
// the running goroutine, so there are two ways to tell them apart here:
//
//   - [Token] is what the framework compares on its hot paths (is this call
//     from the UI goroutine? SPEC §13). On amd64 and arm64 it is the address of
//     the runtime's record of the running goroutine (its "g"), read by a
//     function in Go assembly in a nanosecond; on every other architecture it
//     is [ID].
//   - [ID] is the number of the goroutine, parsed from the header of
//     runtime.Stack. It walks the stack: about 1.4 µs at a shallow depth and
//     30 µs and more at 200 frames. It is for messages (the goroutine numbers
//     that a WrongThread error names), not for a path that runs for every
//     widget or render object.
//
// A token is unique among the goroutines that are alive. The runtime reuses the
// record of a goroutine that has ended, so a token must not be kept past the
// life of its goroutine: a new goroutine can have the token of a dead one.
//
// Flutter: no counterpart (Dart has isolates, not goroutines)
package goid

import "runtime"

// ID is the number of the calling goroutine, read from runtime.Stack. It costs
// microseconds and grows with the depth of the stack; use [Token] to compare
// goroutines.
func ID() uint64 {
	var buf [32]byte
	return parseHeader(buf[:runtime.Stack(buf[:], false)])
}

// parseHeader reads the number from the header of a goroutine's stack, which
// is "goroutine 123 [running]:". It is 0 when there is no number.
func parseHeader(header []byte) uint64 {
	const prefix = "goroutine "
	if len(header) <= len(prefix) || string(header[:len(prefix)]) != prefix {
		return 0
	}
	var id uint64
	for _, c := range header[len(prefix):] {
		if c < '0' || c > '9' {
			break
		}
		id = id*10 + uint64(c-'0')
	}
	return id
}
