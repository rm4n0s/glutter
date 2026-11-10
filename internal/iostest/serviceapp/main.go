//go:build ios

// Command serviceapp is the app of the Simulator tests of step 35.6: it runs the iOS platform with a handler of its own and, after the
// first frame, the steps that the environment gives it (serviceslines.StepsEnv): the system UI, the clipboard, the spell check, the
// haptics and the sounds, opening a URI and a notification. Each step prints a line that the test reads.
package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/iostest/serviceslines"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/ios"
	"github.com/rm4n0s/glutter/internal/platform/mobile"
	"github.com/rm4n0s/glutter/ui"
)

// stream is the audio handler of the app: a sine wave that counts the frames it was asked for, and says the states it is told.
type stream struct {
	mu     sync.Mutex
	frames int
	phase  float64
	rate   float64
	chans  int
}

func (s *stream) Fill(out []float32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := len(out) / s.chans
	for i := 0; i < n; i++ {
		v := float32(0.1 * math.Sin(s.phase))
		s.phase += 2 * math.Pi * 440 / s.rate
		for c := 0; c < s.chans; c++ {
			out[i*s.chans+c] = v
		}
	}
	s.frames += n
}

func (s *stream) Capture([]float32) {}

func (s *stream) StateChanged(st platform.AudioState) {
	name := "other"
	switch st {
	case platform.AudioPaused:
		name = "paused"
	case platform.AudioRunning:
		name = "running"
	case platform.AudioLost:
		name = "lost"
	}
	fmt.Printf("%s %s\n", serviceslines.State, name)
}

func (s *stream) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.frames
}

// padLog says the pads that connect.
type padLog struct{}

func (padLog) PadAdded(i platform.PadInfo) {
	fmt.Printf("%s pad added name=%q standard=%v\n", serviceslines.Step, i.Name, i.Standard)
}
func (padLog) PadRemoved(id platform.PadID) {
	fmt.Printf("%s pad removed %d\n", serviceslines.Step, id)
}
func (padLog) PadEvents(e []platform.PadEvent) {
	fmt.Printf("%s pad events %d\n", serviceslines.Step, len(e))
}

type handler struct {
	platform.NopHandler
	p       *ios.Platform
	w       platform.Window
	started bool
	// pending counts the steps that finish on a goroutine of their own (open, notify, paste): the last line waits for them.
	pending sync.WaitGroup
	// audio is the device that the step audio left open, with its stream.
	audio       platform.AudioDevice
	audioStream *stream
	// audioReady is closed when audio is set: the interruption steps wait for it.
	audioReady chan struct{}
}

// OnSettings prints the settings that the system changed to.
func (h *handler) OnSettings(s platform.Settings) { h.printSettings(s) }

func (h *handler) printSettings(s platform.Settings) {
	brightness := "light"
	if s.PlatformBrightness == ui.BrightnessDark {
		brightness = "dark"
	}
	font17 := 17.0
	if s.ScaleFontSize != nil {
		font17 = s.ScaleFontSize(17)
	}
	locale := ""
	if len(s.Locales) > 0 {
		locale = s.Locales[0].String()
	}
	fmt.Printf("%s brightness=%s scale=%.4f font17=%.2f bold=%v motion=%v contrast=%v voiceover=%v h24=%v locale=%s\n", serviceslines.Settings,
		brightness, s.TextScaleFactor, font17, s.BoldText, s.DisableAnimations, s.HighContrast, s.AccessibleNavigation, s.AlwaysUse24HourFormat, locale)
}

func (h *handler) OnFrame(_ ui.ViewID, _ time.Duration) {
	if h.started {
		return
	}
	h.started = true
	steps := strings.Split(os.Getenv(serviceslines.StepsEnv), ";")
	h.printSettings(h.p.Settings())
	fmt.Println(serviceslines.Ready)
	go func() {
		for _, s := range steps {
			time.Sleep(400 * time.Millisecond)
			if s = strings.TrimSpace(s); s != "" {
				h.p.Post(func() { h.run(s) })
			}
		}
		time.Sleep(400 * time.Millisecond)
		h.pending.Wait()
		fmt.Println(serviceslines.Done)
	}()
}

// tagOf is the tag of the error, "nil" for none.
func tagOf(err error) string {
	if err == nil {
		return "nil"
	}
	if appErr, ok := errors.FromError(err); ok {
		return appErr.Tag
	}
	return "untagged:" + err.Error()
}

func say(step, result string) { fmt.Printf("%s %s %s\n", serviceslines.Step, step, result) }

// run does one step on the UI thread.
func (h *handler) run(step string) {
	name, arg, _ := strings.Cut(step, ":")
	chrome, _ := h.w.(ui.SystemChromeHost)
	switch name {
	case "style":
		b := ui.BrightnessLight
		if arg == "dark" {
			b = ui.BrightnessDark
		}
		chrome.SetSystemUIOverlayStyle(ui.SystemUIOverlayStyle{StatusBarBrightness: b})
		say(step, "done")
	case "mode":
		switch arg {
		case "edge":
			chrome.SetEnabledSystemUIMode(ui.SystemUIModeEdgeToEdge, nil)
		case "hidden":
			chrome.SetEnabledSystemUIMode(ui.SystemUIModeImmersiveSticky, nil)
		case "top":
			chrome.SetEnabledSystemUIMode(ui.SystemUIModeManual, []ui.SystemUIOverlay{ui.SystemUIOverlayTop})
		case "bottom":
			chrome.SetEnabledSystemUIMode(ui.SystemUIModeManual, []ui.SystemUIOverlay{ui.SystemUIOverlayBottom})
		}
		say(step, "done")
	case "orient":
		switch arg {
		case "portrait":
			chrome.SetPreferredOrientations([]ui.DeviceOrientation{ui.DeviceOrientationPortraitUp})
		case "landscape":
			chrome.SetPreferredOrientations([]ui.DeviceOrientation{ui.DeviceOrientationLandscapeLeft, ui.DeviceOrientationLandscapeRight})
		default:
			chrome.SetPreferredOrientations(nil)
		}
		say(step, "done")
	case "settings":
		h.printSettings(h.p.Settings())
		say(step, "done")
	case "report":
		h.p.ReportChrome()
	case "copy":
		h.pending.Add(1)
		h.p.Clipboard().SetText(context.Background(), arg, func(err error) { defer h.pending.Done(); say(step, "err="+tagOf(err)) })
	case "paste":
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		h.pending.Add(1)
		h.p.Clipboard().GetText(ctx, func(text string, ok bool, err error) {
			defer h.pending.Done()
			cancel()
			say(step, fmt.Sprintf("ok=%v text=%q err=%s", ok, text, tagOf(err)))
		})
	case "spell":
		locale, text, _ := strings.Cut(arg, ":")
		sc, ok := h.w.(ui.SpellCheckHost)
		if !ok {
			say(step, "err=nohost")
			return
		}
		h.pending.Add(1)
		sc.SpellCheck(context.Background(), locale, text, func(spans []ui.SpellCheckSpan, err error) {
			defer h.pending.Done()
			var parts []string
			for _, s := range spans {
				first := ""
				if len(s.Suggestions) > 0 {
					first = s.Suggestions[0]
				}
				parts = append(parts, fmt.Sprintf("%d,%d,%s", s.Start, s.End, first))
			}
			say(step, fmt.Sprintf("spans=%s err=%s", strings.Join(parts, ";"), tagOf(err)))
		})
	case "haptic":
		for _, k := range []ui.HapticFeedbackKind{ui.HapticFeedbackLightImpact, ui.HapticFeedbackMediumImpact, ui.HapticFeedbackHeavyImpact,
			ui.HapticFeedbackSelectionClick, ui.HapticFeedbackVibrate} {
			chrome.PerformHapticFeedback(k)
		}
		say(step, "done")
	case "sound":
		for _, k := range []ui.SystemSoundKind{ui.SystemSoundClick, ui.SystemSoundTick, ui.SystemSoundAlert} {
			chrome.PlaySystemSound(k)
		}
		say(step, "done")
	case "audio":
		h.pending.Add(1)
		go h.playAudio(step)
	case "interrupt":
		h.pending.Add(1)
		go func() {
			defer h.pending.Done()
			select {
			case <-h.audioReady:
			case <-time.After(15 * time.Second):
				say(step, "err=nodevice")
				return
			}
			h.p.Post(func() {
				h.p.SendAudioInterruption(arg == "began")
				say(step, "done")
			})
			// The device's state changes arrive after the notification: the last line waits for them.
			time.Sleep(500 * time.Millisecond)
		}()
	case "sensor":
		kind := platform.SensorAccelerometer
		if arg == "gyro" {
			kind = platform.SensorGyroscope
		}
		sens := h.p.Services().Sensors()
		_, err := sens.Open(context.Background(), kind, platform.SensorOpenOptions{}, sensorSink{})
		say(step, fmt.Sprintf("available=%v err=%s", sens.Available(kind), tagOf(err)))
	case "pads":
		pads := h.p.Services().Gamepads()
		say(step, "mapping="+pads.MappingPlatform()+" err="+tagOf(pads.Start(padLog{})))
	case "open":
		h.pending.Add(1)
		go func() {
			defer h.pending.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			say(step, "err="+tagOf(h.p.Desktop().OpenURI(ctx, arg)))
		}()
	case "notify":
		h.pending.Add(1)
		go func() {
			defer h.pending.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_, err := h.p.Desktop().ShowNotification(ctx, mobile.NotificationSpec{Title: "serviceapp", Body: "a notification"})
			say(step, "err="+tagOf(err))
		}()
	default:
		say(step, "unknown")
	}
}

// sensorSink is a sensor handler that wants nothing.
type sensorSink struct{}

func (sensorSink) SensorReadings([]platform.SensorReading) {}

// playAudio plays a stream for a while and says how many frames were asked for, and leaves the device open for the interruption steps.
func (h *handler) playAudio(step string) {
	defer h.pending.Done()
	audio := h.p.Services().Audio()
	devices, err := audio.Devices(platform.AudioPlayback)
	if err != nil {
		say(step, "err="+tagOf(err))
		return
	}
	st := &stream{rate: 48000, chans: 2}
	start := time.Now()
	dev, err := audio.Open(context.Background(), platform.AudioPlayback, platform.AudioOpenOptions{Format: platform.AudioFormat{Rate: 48000, Channels: 2}}, st)
	if err != nil {
		say(step, "err="+tagOf(err))
		return
	}
	time.Sleep(1500 * time.Millisecond)
	elapsed := time.Since(start)
	fmt.Printf("%s frames=%d rate=%d elapsed=%d devices=%d\n", serviceslines.Audio, st.count(), dev.Format().Rate, elapsed.Milliseconds(), len(devices))
	h.audio, h.audioStream = dev, st
	close(h.audioReady)
}

func main() {
	p, err := ios.New(ios.Options{})
	if err != nil {
		fmt.Fprintln(os.Stderr, "serviceapp:", err)
		os.Exit(1)
	}
	w, err := p.NewWindow(platform.WindowConfig{})
	if err != nil {
		fmt.Fprintln(os.Stderr, "serviceapp:", err)
		os.Exit(1)
	}
	go func() {
		for range time.Tick(100 * time.Millisecond) {
			p.Post(w.RequestFrame)
		}
	}()
	h := &handler{p: p, w: w, audioReady: make(chan struct{})}
	if err := p.Run(context.Background(), h); err != nil {
		fmt.Fprintln(os.Stderr, "serviceapp:", err)
		os.Exit(1)
	}
}
