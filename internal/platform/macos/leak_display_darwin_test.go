//go:build darwin && !ios && glutter_display

package macos

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/leaktest"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/darwin/dl"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
	"github.com/rm4n0s/glutter/ui"
)

// residentKB is the resident memory of this process in kilobytes, from ps. It is where an Objective-C object that the autorelease
// pools or a missing Release keep alive shows, which the Go heap does not see.
func residentKB(t *testing.T) int {
	t.Helper()
	out, err := exec.Command("ps", "-o", "rss=", "-p", strconv.Itoa(os.Getpid())).Output()
	if err != nil {
		t.Fatal(err)
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// residentSlack is the growth of the resident memory (in KB) between N and 4N steps that the allocator's own caches explain: 24 MB.
const residentSlack = 24 << 10

// residentGrowth is a counter of the resident memory that stays 0 unless it grew by more than residentSlack from the reading
// after N steps to the one after 4N (leaktest reads each counter at the base, after N and after 4N steps, in that order).
func residentGrowth(t *testing.T) func() int {
	reads, atN := 0, 0
	return func() int {
		reads++
		rss := residentKB(t)
		switch reads {
		case 2:
			atN = rss
		case 3:
			if rss-atN > residentSlack {
				return 1
			}
		}
		return 0
	}
}

// leakCounters are the numbers besides the Go heap that a scenario of the platform must not grow: the handles of the Objective-C
// registry, the blocks of C memory, the C callbacks of the process (never freed) and the resident memory (with the slack of what
// the allocator keeps).
func leakCounters(t *testing.T) map[string]func() int {
	return map[string]func() int{
		"Objective-C handles":                     objcx.HandleCount,
		"blocks of C memory":                      func() int { return int(dl.LiveBlocks()) },
		"C callbacks":                             dl.CallbackCount,
		"resident memory grew by more than 24 MB": residentGrowth(t),
	}
}

// TestLeaksWindowsOpenAndClose opens and closes a window over and over, with a frame between (SPEC §15, "Memory over time", step
// 33.15). A window is a window, a view, a layer, a surface pool and the classes' objects: every one of them must be given back.
func TestLeaksWindowsOpenAndClose(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		// One window stays open: the loop ends with the last window.
		anchor := inputWindow(s, h)
		defer s.onUI(anchor.Close)
		leaktest.Run(t, "windows opened and closed", func() func() {
			return func() {
				w := s.newWindow(platform.WindowConfig{Size: ui.Size{Width: 200, Height: 120}})
				frames := h.frameCount()
				s.onUI(w.RequestFrame)
				s.waitFor("a frame", 5*time.Second, func() bool { return h.frameCount() > frames })
				s.onUI(w.Close)
			}
		}, leaktest.Opts{N: 30, Warmup: 5, Counters: leakCounters(t), CounterSlack: 0})
	})
}

// TestLeaksResizes changes the size of a window over and over: each size is a new drawable and, with the software renderer, a new
// buffer.
func TestLeaksResizes(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		w := inputWindow(s, h)
		sizes := []ui.Size{{Width: 300, Height: 200}, {Width: 420, Height: 260}, {Width: 360, Height: 300}}
		i := 0
		leaktest.Run(t, "window resizes", func() func() {
			return func() {
				size := sizes[i%len(sizes)]
				i++
				s.onUI(func() { w.SetSize(size) })
				s.waitFor("the new size", 5*time.Second, func() bool {
					m, ok := h.lastMetrics()
					return ok && int(m.PhysicalSize.Width) > 0
				})
			}
		}, leaktest.Opts{N: 60, Warmup: 6, Counters: leakCounters(t), CounterSlack: 0})
		s.onUI(w.Close)
	})
}

// TestLeaksInputMethodSessions composes text with the input method's calls over and over, and commits it.
func TestLeaksInputMethodSessions(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		w := textWindow(s, h, "", 0, 0)
		leaktest.Run(t, "input method sessions", func() func() {
			return func() {
				s.onUI(func() { w.p.runDevText(DevEvent{Kind: "compose", Steps: []string{"に", "にほ", "日本"}}) })
				h.takeTexts()
			}
		}, leaktest.Opts{N: 100, Warmup: 10, Counters: leakCounters(t), CounterSlack: 0})
		s.onUI(w.Close)
	})
}
