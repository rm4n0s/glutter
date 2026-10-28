//go:build amd64 || arm64

package goid

import (
	"sync"
	"sync/atomic"
)

// The goroutine's number is a field of the runtime's record of the goroutine
// (its "g"). Go does not export the layout of the g and it changes between
// versions, so the offset of the field is not written down here: the first call
// of [Token] finds it (calibrate) by comparing the words at the start of the g
// with the numbers that runtime.Stack gives, and keeps it for the life of the
// process. If the comparison does not find exactly one offset, Token uses [ID].

const (
	// windowBytes is how far into the g the calibration reads, and the most that
	// gword can read: the assembly masks its argument to 0xf8, so no offset can
	// reach past the first 256 bytes of a g, whatever the code calls it with.
	windowBytes = 256
	// windowWords is the size of that window in words.
	windowWords = windowBytes / 8
	// freshGoroutines is how many new goroutines the calibration samples besides
	// the calling one (at least 3).
	freshGoroutines = 4
)

// gword is the 8-byte word at byte offset off&0xf8 of the running goroutine's g,
// read by a function in Go assembly (getg_amd64.s, getg_arm64.s) that is
// NOSPLIT and has no frame.
func gword(off uintptr) uint64

// offsetState is what the calibration found: 0 before it ran, fallback when it
// did not find exactly one offset, and otherwise the offset of the goroutine's
// number plus one.
var offsetState atomic.Uint32

// fallback is the value of offsetState that means Token uses ID.
const fallback = ^uint32(0)

var calibrated sync.Once

// Token is a value that tells the calling goroutine from every other goroutine,
// alive or ended. It is the goroutine's number, the one that [ID] and the header
// of runtime.Stack give, and the runtime never gives a number to a second
// goroutine, so a token can be kept and compared after its goroutine ended.
//
// On amd64 and arm64 the number is read from the runtime's record of the
// goroutine by a function in Go assembly (getg_amd64.s, getg_arm64.s) in about
// two nanoseconds, at any stack depth. The offset of the number in that record
// is found once, by the first call (which also runs in a few microseconds, on
// the calling goroutine and on new ones, and is safe from any goroutine at any
// time); if that finds no single offset, every call is [ID], which is exact and
// slow. It is never 0 except when runtime.Stack gives no number.
//
// Flutter: no counterpart
func Token() uint64 {
	if s := offsetState.Load(); s != 0 && s != fallback {
		return gword(uintptr(s - 1))
	}
	return tokenSlow()
}

// tokenSlow is Token before the calibration ran and when it fell back.
func tokenSlow() uint64 {
	calibrated.Do(calibrate)
	if s := offsetState.Load(); s != fallback {
		return gword(uintptr(s - 1))
	}
	return ID()
}

// fastPath is whether Token reads the number from the g (after the calibration
// ran, and found its offset).
func fastPath() bool {
	calibrated.Do(calibrate)
	return offsetState.Load() != fallback
}

// sample is what is read on one goroutine: the first words of its g, and its
// number from runtime.Stack.
type sample struct {
	id    uint64
	words [windowWords]uint64
}

// readSample reads the window of the calling goroutine's g and its number.
func readSample() sample {
	var s sample
	for i := range s.words {
		s.words[i] = gword(uintptr(i * 8))
	}
	s.id = ID()
	return s
}

// calibrate finds the offset of the goroutine's number in the g and stores it
// in offsetState. It samples the calling goroutine and freshGoroutines new ones
// that are alive at the same time (so their numbers differ).
func calibrate() {
	samples := make([]sample, freshGoroutines+1)
	var started, wg sync.WaitGroup
	release := make(chan struct{})
	started.Add(freshGoroutines)
	wg.Add(freshGoroutines)
	for i := 1; i <= freshGoroutines; i++ {
		go func() {
			defer wg.Done()
			samples[i] = readSample()
			started.Done()
			<-release
		}()
	}
	started.Wait()
	samples[0] = readSample()
	close(release)
	wg.Wait()
	if off, ok := findOffset(samples); ok {
		offsetState.Store(uint32(off) + 1)
		return
	}
	offsetState.Store(fallback)
}

// findOffset is the offset, in bytes, of the one word that equals the number of
// its goroutine in every sample. It reports false when there are fewer than 4
// samples, when two samples have the same number or a number 0 (the samples
// would not tell offsets apart), and when no offset or more than one offset
// matches on all of them.
func findOffset(samples []sample) (int, bool) {
	if len(samples) < 4 {
		return 0, false
	}
	seen := make(map[uint64]bool, len(samples))
	for _, s := range samples {
		if s.id == 0 || seen[s.id] {
			return 0, false
		}
		seen[s.id] = true
	}
	found := -1
	for w := range windowWords {
		match := true
		for _, s := range samples {
			if s.words[w] != s.id {
				match = false
				break
			}
		}
		if !match {
			continue
		}
		if found >= 0 {
			return 0, false
		}
		found = w * 8
	}
	return found, found >= 0
}
