//go:build race

package svg

// raceEnabled says whether the tests run under the race detector, which makes text
// layout about 25 times slower: the time limits of the tests do not hold then.
const raceEnabled = true
