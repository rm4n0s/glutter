package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
)

// editorText is a text of n lines of about 60 characters each.
func editorText(n int) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "Line %d: the quick brown fox jumps over the lazy dog, 0123456789\n", i)
	}
	return b.String()
}

// caretTester is the editor with n lines of text, focused, with the caret blinking.
func caretTester(tb testing.TB, n int, discard bool) *gluttertest.Tester {
	tb.Helper()
	useLinux(tb)
	ts := gluttertest.New(tb, gluttertest.Opts{Size: ui.Size{Width: 820, Height: 460}, DiscardScenes: discard})
	ts.PumpWidget(newApp(options{dir: tb.TempDir(), text: foundation.Some(editorText(n))}))
	for range 200 {
		ts.Pump(10 * time.Millisecond)
		if editorOf(ts).loaded {
			break
		}
		time.Sleep(time.Millisecond)
	}
	ts.Pump(50 * time.Millisecond)
	// Two blinks, so that every cache the caret frame fills is full.
	ts.Pump(500 * time.Millisecond)
	ts.Pump(500 * time.Millisecond)
	return ts
}

// BenchmarkCaretFrame is the framework's share of a frame in which only the caret of the editor
// blinks (build, layout, paint and compositing, SPEC §15): the bytes and allocations of a frame
// must not grow with the length of the text. Run it with
//
//	go test ./examples/editor -run xxx -bench CaretFrame -benchmem
func BenchmarkCaretFrame(b *testing.B) {
	for _, lines := range []int{50, 500, 5000} {
		b.Run(fmt.Sprintf("lines=%d", lines), func(b *testing.B) {
			ts := caretTester(b, lines, true)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ts.PumpFrame(500 * time.Millisecond)
			}
		})
	}
}
