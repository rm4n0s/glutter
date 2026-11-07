//go:build race

package main

// raceEnabled is true when the tests run under the race detector, which changes what a call allocates.
const raceEnabled = true
