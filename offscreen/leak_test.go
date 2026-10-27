package offscreen_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/internal/gpu/gputest/editorscene"
	"github.com/rm4n0s/glutter/internal/leaktest"
	"github.com/rm4n0s/glutter/offscreen"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// TestLeaksOffscreen checks the memory of the off-screen drawing (step 29.21, SPEC §15 "Memory
// over time"): frames of a surface and of a view, and views that are made and closed.
func TestLeaksOffscreen(t *testing.T) {
	size := ui.Size{Width: counterW, Height: counterH}
	tests := []struct {
		name string
		n    int
		step func(t *testing.T) func()
	}{
		{"surface, caret frames of the editor", 300, func(t *testing.T) func() {
			on, off := editorscene.CaretScenes(t, 200)
			d := offscreen.NewDetacher()
			scenes := [2]*ui.Scene{d.Detach(on), d.Detach(off)}
			s := offscreen.NewSurface(editorscene.Width, editorscene.Height)
			i := 0
			return func() {
				if _, err := s.Draw(scenes[i%2]); err != nil {
					t.Fatal(err)
				}
				i++
			}
		}},
		{"detacher, a new scene for each frame", 300, func(t *testing.T) func() {
			on, off := editorscene.CaretScenes(t, 200)
			d := offscreen.NewDetacher()
			scenes := [2]*ui.Scene{on, off}
			i := 0
			return func() { _ = d.Detach(scenes[i%2]); i++ }
		}},
		{"a view that gets taps", 100, func(t *testing.T) func() {
			probe := &counterProbe{}
			h := newHarness(t, newCounter(probe), size)
			h.step(0)
			taps := 0
			return func() {
				tap(h.view, fabX, fabY)
				h.step(pace)
				if taps++; taps%10 == 0 {
					// The text of the counter goes back to 0: the shaped runs and analyses of the
					// text are kept in caches of a bound, which a text that never repeats would fill.
					probe.s.SetState(func() { probe.s.count = 0 })
					h.step(pace)
				}
				h.mu.Lock()
				h.frames = h.frames[:0] // the harness keeps a copy of every frame
				h.mu.Unlock()
			}
		}},
		{"views made and closed", 40, func(t *testing.T) func() {
			parent := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 320, Height: 200}, DiscardScenes: true})
			parent.PumpWidget(widgets.NewSizedBoxShrink(nil))
			return func() {
				v, err := offscreen.NewViewWithClock(parent.Binding, size, newLightCounter(nil), func(offscreen.Frame) {}, parent.Clock)
				if err != nil {
					t.Fatal(err)
				}
				parent.Pump(0)
				v.WaitIdle()
				parent.Pump(40 * time.Millisecond)
				v.WaitIdle()
				v.Close()
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			leaktest.Run(t, "offscreen/"+tt.name, func() func() { return tt.step(t) }, leaktest.Opts{N: tt.n})
		})
	}
}
