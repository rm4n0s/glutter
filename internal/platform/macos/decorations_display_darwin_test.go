//go:build darwin && !ios && glutter_display

package macos

import (
	"math"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

func TestSystemButtonsOfAWindowWhoseAppDrawsTheTitleBar(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		w := s.newWindow(platform.WindowConfig{Size: ui.Size{Width: 400, Height: 300}, Decorations: platform.DecorationsApp})
		s.waitFor("the first frame", 5*time.Second, func() bool { return h.frameCount() > 0 })
		m, ok := h.lastMetrics()
		if !ok {
			t.Fatal("no metrics")
		}
		before := m.Window.SystemButtons
		t.Logf("the traffic lights where AppKit puts them: %v", before)
		if m.Window.Decoration != ui.DecorationApp {
			t.Errorf("decoration = %v", m.Window.Decoration)
		}
		// Three buttons at the top left (this machine writes from left to right), inside the view.
		if before.IsEmpty() || before.Left < 0 || before.Top < 0 || before.Right > 120 || before.Bottom > 60 || before.Width() < 40 {
			t.Fatalf("the area of the traffic lights is %v", before)
		}

		// The app says how high its bar is: the buttons go to the middle of it, at the same place along it,
		// and the framework hears where they are.
		for _, height := range []float64{40, 36, 60} {
			count := len(h.metricsSnapshot())
			s.onUI(func() { w.PerformWindowAction(ui.WindowAction{Kind: ui.WindowActionTitleBarHeight, Height: height}) })
			if got := len(h.metricsSnapshot()); got <= count { // the window becoming the key window reports too
				t.Errorf("height %v: no metrics were reported", height)
			}
			m, _ = h.lastMetrics()
			after := m.Window.SystemButtons
			if mid := (after.Top + after.Bottom) / 2; math.Abs(mid-height/2) > 0.5 {
				t.Errorf("height %v: the middle of the traffic lights is at %v (%v)", height, mid, after)
			}
			if after.Left != before.Left || after.Right != before.Right || after.Height() != before.Height() {
				t.Errorf("height %v: the traffic lights are %v, they were %v", height, after, before)
			}
		}
		// The same height again changes nothing and reports nothing.
		var same int
		s.onUI(func() {
			count := len(h.metricsSnapshot())
			w.PerformWindowAction(ui.WindowAction{Kind: ui.WindowActionTitleBarHeight, Height: 60})
			same = len(h.metricsSnapshot()) - count
		})
		if same != 0 {
			t.Errorf("the same height reported %d metrics", same)
		}

		// AppKit lays its title bar out again at a resize: the buttons stay in the middle of the app's bar.
		s.onUI(func() { w.SetSize(ui.Size{Width: 500, Height: 320}) })
		m, _ = h.lastMetrics()
		if after := m.Window.SystemButtons; math.Abs((after.Top+after.Bottom)/2-30) > 0.5 || after.Left != before.Left {
			t.Errorf("after a resize the traffic lights are %v", after)
		}
		// A new title too.
		s.onUI(func() { w.SetTitle("Another title") })
		var now ui.Rect
		s.onUI(func() { now = w.systemButtons() })
		if math.Abs((now.Top+now.Bottom)/2-30) > 0.5 {
			t.Errorf("after a new title the traffic lights are %v", now)
		}
		s.onUI(w.Close)
	})
}

func TestSystemButtonsAreEmptyWhereTheSystemDrawsTheTitleBar(t *testing.T) {
	tests := []struct {
		name string
		cfg  platform.WindowConfig
	}{
		{"the system's title bar", platform.WindowConfig{Size: ui.Size{Width: 300, Height: 200}}},
		{"a borderless window", platform.WindowConfig{Size: ui.Size{Width: 300, Height: 200}, Kind: platform.WindowKindBorderless}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := &recorder{}
			runApp(t, Options{}, h, func(s *session) {
				w := s.newWindow(tc.cfg)
				s.waitFor("the first frame", 5*time.Second, func() bool { return h.frameCount() > 0 })
				s.onUI(func() { w.PerformWindowAction(ui.WindowAction{Kind: ui.WindowActionTitleBarHeight, Height: 40}) })
				m, ok := h.lastMetrics()
				if !ok {
					t.Fatal("no metrics")
				}
				if !m.Window.SystemButtons.IsEmpty() {
					t.Errorf("SystemButtons = %v", m.Window.SystemButtons)
				}
				s.onUI(w.Close)
			})
		})
	}
}
