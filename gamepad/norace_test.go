//go:build !race

package gamepad_test

// raceEnabled is true when the tests run under the race detector, which changes what a call allocates.
const raceEnabled = false
