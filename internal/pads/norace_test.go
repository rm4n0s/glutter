//go:build !race

package pads

// raceEnabled is true when the tests run under the race detector, which changes what a call allocates.
const raceEnabled = false
