//go:build !race

package viewport3d_test

// raceEnabled is whether the test binary has the race detector, whose instrumentation changes what a function allocates.
const raceEnabled = false
