//go:build linux

package wayland

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/fakecomp"
	"github.com/rm4n0s/glutter/ui"
)

// The outputs of the compositor: Displays, the logical size, fractional scales,
// rotation, hotplug and the display of a window (SPEC §8.2, §8.11).

func TestDisplaysDescribeTheOutputs(t *testing.T) {
	tests := []struct {
		name string
		cfg  fakecomp.Config
		want platform.Display
	}{
		{
			name: "wl_output only",
			cfg:  fakecomp.Config{OutputScale: 2},
			want: platform.Display{
				ID: 2, Name: "TEST-1", Primary: true, DevicePixelRatio: 2, RefreshRate: 60,
				Bounds: ui.NewRectFromLTWH(-10, 20, 960, 540),
			},
		},
		{
			name: "logical size gives a fractional scale",
			cfg: fakecomp.Config{
				Output:      fakecomp.Output{Width: 3840, Height: 2160, Refresh: 143981},
				XdgOutput:   &fakecomp.XdgOutput{X: 0, Y: 0, Width: 1536, Height: 864},
				OutputScale: 2,
			},
			want: platform.Display{
				ID: 2, Name: "TEST-1", Primary: true, DevicePixelRatio: 2.5, RefreshRate: 143.981,
				Bounds: ui.NewRectFromLTWH(0, 0, 1536, 864),
			},
		},
		{
			name: "rotated by 90 degrees swaps the mode",
			cfg: fakecomp.Config{
				Output:      fakecomp.Output{Width: 1080, Height: 1920, Refresh: 60000, Transform: 1},
				OutputScale: 1,
			},
			want: platform.Display{
				ID: 2, Name: "TEST-1", Primary: true, DevicePixelRatio: 1, RefreshRate: 60,
				Bounds: ui.NewRectFromLTWH(-10, 20, 1920, 1080),
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.cfg.Globals = scaleGlobals()
			p, _, err := connectWith(t, tc.cfg, Options{NoLibdecor: true})
			if err != nil {
				t.Fatal(err)
			}
			got := p.Displays()
			if len(got) != 1 || got[0] != tc.want {
				t.Errorf("Displays() = %+v, want [%+v]", got, tc.want)
			}
		})
	}
}

func TestOutputsComeAndGo(t *testing.T) {
	e := startEGLWindowWith(t, fakecomp.Config{Globals: scaleGlobals(), XdgOutput: &fakecomp.XdgOutput{Width: 1920, Height: 1080}})
	waitUntil(t, func() bool { return e.h.frameCount() == 1 }, "the first frame was never requested")
	if n := len(e.h.lastDisplays()); n > 1 {
		t.Fatalf("displays at the start = %d", n)
	}

	e.srv.AddOutput(30)
	waitUntil(t, func() bool { return len(e.h.lastDisplays()) == 2 }, "a plugged-in output never showed up in Displays")
	d := e.h.lastDisplays()
	if d[0].ID != 2 || !d[0].Primary || d[1].ID != 30 || d[1].Primary {
		t.Errorf("displays after the hotplug = %+v", d)
	}

	e.srv.RemoveOutput(30)
	waitUntil(t, func() bool { return len(e.h.lastDisplays()) == 1 }, "an unplugged output stayed in Displays")
	if got := e.h.lastDisplays()[0]; got.ID != 2 || !got.Primary {
		t.Errorf("displays after the unplug = %+v", got)
	}
	if len(e.srv.RequestsWith("wl_output.release")) != 1 {
		t.Errorf("the removed output was not released: %v", e.srv.RequestsWith("wl_output"))
	}
}

func TestTheViewMetricsNameTheDisplayOfTheWindow(t *testing.T) {
	// A compositor that has no preferred scale for a surface: the scale is the largest
	// of the outputs the window is on.
	e := startEGLWindowWith(t, fakecomp.Config{PreferredBufferScale: -1})
	waitUntil(t, func() bool { return e.h.frameCount() == 1 }, "the first frame was never requested")
	waitUntil(t, func() bool {
		m := e.h.metricsSeen()
		return len(m) > 0 && m[len(m)-1].Display == foundation.Some(2)
	}, "the metrics never named the display the surface entered")

	// A second output at scale 2: the window moves onto it, and leaves the first.
	e.srv.AddOutput(30)
	waitUntil(t, func() bool { return len(e.h.lastDisplays()) == 2 }, "no second output")
	e.srv.SetScaleOfOutput(30, 2)
	frames := e.h.frameCount()
	e.srv.EnterOutputNamed(30)
	e.srv.LeaveOutputNamed(2)
	waitUntil(t, func() bool {
		m := e.h.metricsSeen()
		last := m[len(m)-1]
		return last.Display == foundation.Some(30) && last.DevicePixelRatio == 2
	}, "the metrics did not follow the window to the second output")
	// The window is drawn at the scale of the output it is on, in the frame that follows.
	waitUntil(t, func() bool { return e.h.frameCount() > frames }, "no frame was rendered at the new scale")
	if last := e.lastMetrics(); last.PhysicalSize != (ui.Size{Width: 1280, Height: 800}) {
		t.Errorf("physical size on the scale-2 output = %+v", last.PhysicalSize)
	}

	// Back on the first output.
	e.srv.EnterOutputNamed(2)
	e.srv.LeaveOutputNamed(30)
	waitUntil(t, func() bool {
		m := e.h.metricsSeen()
		last := m[len(m)-1]
		return last.Display == foundation.Some(2) && last.DevicePixelRatio == 1
	}, "the metrics did not follow the window back to the first output")
}
