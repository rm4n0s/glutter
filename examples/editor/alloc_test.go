package main

import (
	"runtime"
	"testing"
	"time"
)

// bytesPerFrame runs frame the given number of times and returns the bytes the process allocated
// for each; nothing else may run while it measures.
func bytesPerFrame(frames int, frame func()) uint64 {
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	for i := 0; i < frames; i++ {
		frame()
	}
	runtime.ReadMemStats(&after)
	return (after.TotalAlloc - before.TotalAlloc) / uint64(frames)
}

// caretFrameBound is what the framework may allocate for a frame in which only the caret of the
// editor blinks, with a text of any length (step 29.15); it allocates about 14 KB.
const caretFrameBound = 128 << 10

func TestCaretFrameOfTheFrameworkAllocatesLittleAndNothingForTheLengthOfTheText(t *testing.T) {
	tests := []struct {
		name  string
		lines int
	}{{"a short text", 60}, {"a long text", 1500}}
	var perFrame []uint64
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := caretTester(t, tt.lines, true)
			frame := func() { ts.PumpFrame(500 * time.Millisecond) }
			for range 8 {
				frame()
			}
			bytes := bytesPerFrame(30, frame)
			t.Logf("%d bytes for each frame", bytes)
			if bytes > caretFrameBound {
				t.Errorf("a caret frame allocates %d bytes, want at most %d", bytes, caretFrameBound)
			}
			perFrame = append(perFrame, bytes)
		})
	}
	if len(perFrame) == 2 && perFrame[1] > perFrame[0]+perFrame[0]/2+8<<10 {
		t.Errorf("%d bytes for a frame of the long text and %d for the short one: the bytes grow with the text", perFrame[1], perFrame[0])
	}
}
