package main

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/leaktest"
)

// TestLeaksIdleCaretFrames checks that the frames in which only the caret of the editor blinks
// keep the heap and the goroutines flat (step 29.21, SPEC §15 "Memory over time"): the live heap
// and the goroutines after N and after 4N frames, for a short and a long text.
func TestLeaksIdleCaretFrames(t *testing.T) {
	for _, lines := range []int{50, 1500} {
		name := "framework/" + map[int]string{50: "50 lines", 1500: "1500 lines"}[lines]
		t.Run(name, func(t *testing.T) {
			leaktest.Run(t, "editor/caret frames/"+name, func() func() {
				ts := caretTester(t, lines, true)
				return func() { ts.PumpFrame(500 * time.Millisecond); ts.ClearLogs() }
			}, leaktest.Opts{N: 300})
		})
	}
}
