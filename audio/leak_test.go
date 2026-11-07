package audio_test

import (
	"context"
	"testing"

	"github.com/rm4n0s/glutter/audio"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/internal/leaktest"
	"github.com/rm4n0s/glutter/widgets"
)

// TestLeaksStreamsAndDevices runs devices and streams for a long time (SPEC §15, "Memory over time"): the live heap and the
// goroutines must not grow, and the devices of the backend must all be closed.
func TestLeaksStreamsAndDevices(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, tester *gluttertest.Tester, sys *audio.System) func()
	}{
		{"a device opened and closed, with streams", func(t *testing.T, tester *gluttertest.Tester, sys *audio.System) func() {
			piece := make([]float32, 2*480)
			return func() {
				dev, err := sys.OpenPlayback(context.Background())
				if err != nil {
					t.Fatal(err)
				}
				a := dev.NewStream(audio.Format{Rate: 44100, Channels: 2})
				b := dev.NewCallbackStream(audio.Format{Rate: 32000, Channels: 1}, func(out []float32) { out[0] = 0.1 })
				_ = a.Put(piece)
				buf := make([]float32, 2*480)
				tester.Audio().Playback().PullInto(buf)
				b.Close()
				if err := dev.Close(); err != nil {
					t.Fatal(err)
				}
				// The change of state was posted to the UI thread: the test runs what was posted, as the app's loop does.
				tester.Pump(0)
				if n := len(tester.Audio().Devices()); n != 0 {
					t.Fatalf("%d devices are left open", n)
				}
			}
		}},
		{"a stream that plays for a long time", func(t *testing.T, tester *gluttertest.Tester, sys *audio.System) func() {
			dev, err := sys.OpenPlayback(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			st := dev.NewStream(audio.Format{Rate: 44100, Channels: 2})
			td := tester.Audio().Playback()
			piece := make([]float32, 2*441)
			buf := make([]float32, 2*480)
			return func() {
				for i := 0; i < 50; i++ {
					_ = st.Put(piece)
					td.PullInto(buf)
				}
			}
		}},
		{"streams that are made and closed", func(t *testing.T, tester *gluttertest.Tester, sys *audio.System) func() {
			dev, err := sys.OpenPlayback(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			td := tester.Audio().Playback()
			piece := make([]float32, 2*100)
			buf := make([]float32, 2*480)
			return func() {
				for i := 0; i < 20; i++ {
					st := dev.NewStream(audio.Format{Rate: 22050, Channels: 2})
					_ = st.Put(piece)
					td.PullInto(buf)
					st.Close()
				}
			}
		}},
		{"a capture that is read", func(t *testing.T, tester *gluttertest.Tester, sys *audio.System) func() {
			dev, err := sys.OpenCapture(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			st := dev.NewStream(audio.Format{Rate: 16000, Channels: 1})
			td := tester.Audio().Capture()
			in := make([]float32, 480)
			out := make([]float32, 480)
			return func() {
				for i := 0; i < 50; i++ {
					td.Push(in)
					st.Get(out)
				}
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tester := gluttertest.New(t)
			sys := audio.ForBinding(tester.Binding)
			leaktest.Run(t, "audio/"+tt.name, func() func() { return tt.setup(t, tester, sys) })
		})
	}
}

// TestLeaksAThousandStreamsOpenedAndClosed opens a stream of a device, plays it and closes it 1,000 times (4,000 in the second run;
// step 31.11, SPEC §28.6): the heap and the goroutines stay flat and the device is left with no stream.
func TestLeaksAThousandStreamsOpenedAndClosed(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{DiscardScenes: true})
	ts.PumpWidget(widgets.NewSizedBox(nil))
	sys := audio.ForBinding(ts.Binding)
	dev, err := sys.OpenPlayback(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	td := ts.Audio().Playback()
	piece := make([]float32, 220)
	buf := make([]float32, 2*480)
	leaktest.Run(t, "audio/a thousand streams", func() func() {
		return func() {
			st := dev.NewStream(audio.Format{Rate: 22050, Channels: 1})
			_ = st.Put(piece)
			td.PullInto(buf)
			st.Close()
			cb := dev.NewCallbackStream(audio.Format{Rate: 22050, Channels: 1}, func(out []float32) {})
			td.PullInto(buf)
			cb.Close()
			td.PullInto(buf)
		}
	}, leaktest.Opts{N: 1000, Warmup: 50})
	if err := dev.Close(); err != nil {
		t.Fatal(err)
	}
}
