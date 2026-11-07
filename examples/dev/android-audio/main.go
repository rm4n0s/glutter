// Command android-audio is a development example (step 31.6): a Material page that plays a 440 Hz tone through the public audio
// package and, besides, through a device of the platform's own (to read the AAudio stream's timestamp, which the public package does
// not show), and logs what the audio does twice a second. The device tests start it on the emulator and check that the callbacks run,
// that the timestamps advance, that the stream pauses when the app is hidden and goes on when it is shown, and that a recording asks
// for the permission. It may use internal packages, and runs where there is an audio service (the stats are for Android only).
//
// The log lines (the default logger's, so "INFO " comes first):
//
//	audio state state=<running|paused|lost|closed>
//	audio stats public=<frames played by the public device> platform=<fills of the platform's device> frame=<n> nanos=<n> ts=<bool> ...
//	audio capture failed err=<tag>
//	audio capture frames=<n>
//
// GLUTTER_AUDIO_CAPTURE=1 also opens a recording device. Run it on a device or the emulator with:
//
//	go run ./cmd/glutter-android run -permission android.permission.RECORD_AUDIO ./examples/dev/android-audio
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"math"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/audio"
	"github.com/rm4n0s/glutter/internal/platform"
	m "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/widgets"
)

func main() {
	err := m.RunApp(newApp(), m.RunAppOpts{
		AppID:  "dev.glutter.androidaudio",
		Window: m.WindowConfig{Title: "Android audio", Size: m.Size{Width: 480, Height: 640}},
	})
	if err != nil {
		log.Fatal(err)
	}
}

func newApp() m.Widget {
	return m.NewMaterialApp(m.MaterialAppOpts{
		Title:                      "Android audio",
		DebugShowCheckedModeBanner: m.Some(false),
		Home:                       &page{StatefulWidget: m.NewStatefulWidgetBase()},
	})
}

type page struct{ m.StatefulWidget }

func (p *page) CreateState() m.State { return &pageState{} }

// tone plays a sine of 440 Hz on every channel, and counts the frames and the calls.
type tone struct {
	channels int
	phase    float64
	step     float64
	frames   atomic.Int64
	fills    atomic.Int64
}

func (t *tone) fill(out []float32) {
	for i := 0; i+t.channels <= len(out); i += t.channels {
		v := float32(0.1 * math.Sin(t.phase))
		t.phase += t.step
		for c := 0; c < t.channels; c++ {
			out[i+c] = v
		}
	}
	t.frames.Add(int64(len(out) / t.channels))
	t.fills.Add(1)
}

// platformHandler is the handler of the platform's own device: it plays the same tone and counts the calls.
type platformHandler struct{ tone *tone }

func (h platformHandler) Fill(out []float32) { h.tone.fill(out) }
func (platformHandler) Capture([]float32)    {}
func (platformHandler) StateChanged(s platform.AudioState) {
	slog.Info("audio platform state", "state", int(s))
}

type pageState struct {
	m.StateOf[*page]
	once sync.Once
	stop chan struct{}
}

func (s *pageState) DidChangeDependencies() { s.once.Do(func() { s.start(s.Context()) }) }

func (s *pageState) Dispose() {
	if s.stop != nil {
		close(s.stop)
	}
}

func stateName(st audio.DeviceState) string {
	switch st {
	case audio.DeviceStateRunning:
		return "running"
	case audio.DeviceStatePaused:
		return "paused"
	case audio.DeviceStateSuspended:
		return "suspended"
	case audio.DeviceStateLost:
		return "lost"
	}
	return "closed"
}

func tagOf(err error) string {
	if appErr, ok := errors.FromError(err); ok {
		return appErr.Tag
	}
	return "none"
}

func (s *pageState) start(bc m.BuildContext) {
	s.stop = make(chan struct{})
	ctx := context.Background()
	sys := audio.Of(bc)
	public := &tone{channels: 2}
	// Right after the emulator has booted, its audio service may not have an output yet and AAudio answers "illegal argument": the
	// app tries again for a few seconds.
	var dev *audio.Device
	var err error
	for attempt := 0; ; attempt++ {
		dev, err = sys.OpenPlayback(ctx)
		if err == nil {
			break
		}
		if attempt == 20 {
			slog.Error("audio playback failed", "err", tagOf(err), "detail", err.Error(), "meta", fmt.Sprint(metaOf(err)))
			return
		}
		slog.Info("audio playback retry", "attempt", attempt, "err", tagOf(err))
		time.Sleep(500 * time.Millisecond)
	}
	rate := dev.Format().Rate
	public.step = 2 * math.Pi * 440 / float64(rate)
	dev.NewCallbackStream(audio.Format{Rate: rate, Channels: 2}, public.fill)
	dev.AddListener(func() { slog.Info("audio state", "state", stateName(dev.State())) })
	slog.Info("audio device", "rate", rate, "channels", dev.Format().Channels)

	// The platform's own device, for the timestamp.
	var pdev platform.AudioDevice
	own := &tone{channels: 2}
	var host platform.AudioHost
	if widgets.WidgetsBindingOf(bc).PlatformDispatcher().HostAs(&host) && host.Audio() != nil {
		pdev, err = host.Audio().Open(ctx, platform.AudioPlayback, platform.AudioOpenOptions{PauseInBackground: true}, platformHandler{own})
		if err != nil {
			slog.Error("audio platform device failed", "err", tagOf(err))
			pdev = nil
		} else {
			own.channels = pdev.Format().Channels
			own.step = 2 * math.Pi * 440 / float64(pdev.Format().Rate)
		}
	}

	// A recording asks the user, and OpenCapture waits for the answer: it runs on a goroutine of its own.
	var capturing atomic.Bool
	var captured atomic.Int64
	if os.Getenv("GLUTTER_AUDIO_CAPTURE") == "1" {
		go func() {
			capture, err := sys.OpenCapture(ctx)
			if err != nil {
				slog.Error("audio capture failed", "err", tagOf(err))
				return
			}
			defer func() { _ = capture.Close() }() // the app is ending: nothing to report
			capturing.Store(true)
			cs := capture.NewStream(capture.Format())
			buf := make([]float32, 4800*capture.Format().Channels)
			for {
				select {
				case <-s.stop:
					return
				case <-time.After(50 * time.Millisecond):
				}
				captured.Add(int64(cs.Get(buf)))
			}
		}()
	}

	go func() {
		tick := time.NewTicker(500 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-s.stop:
				return
			case <-tick.C:
			}
			var frame, nanos int64
			var ok bool
			var xruns, buffer, burst int
			if pdev != nil {
				frame, nanos, ok, xruns, buffer, burst = snapshot(pdev)
			}
			slog.Info("audio stats", "public", public.frames.Load(), "platform", own.fills.Load(), "frame", frame, "nanos", nanos, "ts", ok,
				"xruns", xruns, "buffer", buffer, "burst", burst)
			if capturing.Load() {
				slog.Info("audio capture", "frames", captured.Load())
			}
		}
	}()
}

func (s *pageState) Build(bc m.BuildContext) m.Widget {
	return m.NewScaffold(m.ScaffoldOpts{
		AppBar: m.NewAppBar(m.AppBarOpts{Title: m.NewText("Android audio")}),
		Body:   m.NewCenter(m.NewText("440 Hz")),
	})
}

func metaOf(err error) any {
	if appErr, ok := errors.FromError(err); ok {
		return appErr.Metadata
	}
	return nil
}
