// Package cf is the part of Core Foundation that the macOS and iOS backends use (SPEC §29.2, §29.3):
// strings, dictionaries, the run loop, and the version-0 run loop sources and the timers that carry
// Post and the timer heap of the UI thread.
//
// # Reference counting
//
// Core Foundation follows the Create rule, and so does this package: a function whose name starts with
// New returns a reference that the caller owns (+1) and must give back with Release; every other
// function that returns a Ref returns one that the caller does not own and must Retain to keep (the
// run loops, the modes, the value that a dictionary holds). A Source or a Timer owns its Core
// Foundation object and gives it back in Close, which every owner calls exactly once.
//
// # Threads
//
// A run loop belongs to a thread. Only the UI thread, which is the main thread, adds sources and timers
// to the main run loop and runs it; Source.Signal and WakeUp are the calls that other goroutines may
// make (the one thing Core Foundation documents as thread safe for a source).
//
// The package has files for darwin only; this file keeps it listed on every OS.
//
// Flutter: no counterpart.
package cf
