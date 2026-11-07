//go:build linux && !android && glutter_audio

package pulse_test

// These tests talk to a real PulseAudio protocol server, the private one that scripts/headless-audio.sh starts (PipeWire with a null
// sink, glutter_sink, whose monitor is the recording of what is played, and a null source, glutter_source). They are run with
//
//	scripts/headless-audio.sh -- go test -tags glutter_audio ./internal/platform/linux/pulse
//
// and skip when there is no such server. GLUTTER_AUDIO_SOAK is the number of seconds of the long test (60 by default; 600 is the
// run of step 31.5).

import (
	"context"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/gputest"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/pulse"
	"github.com/rm4n0s/glutter/ui"
)

const (
	toneHz    = 440.0
	toneLevel = 0.5
	rate      = 48000
)

var stereo = platform.AudioFormat{Rate: rate, Channels: 2}

func requireServer(t *testing.T) {
	t.Helper()
	if os.Getenv("GLUTTER_AUDIO_SERVER_RUNTIME") == "" {
		t.Skip("no private audio server: run this with scripts/headless-audio.sh -- go test -tags glutter_audio ...")
	}
}

// tone is a playback handler that plays a sine tone, any number of channels with the same sound, and keeps how the calls came.
type tone struct {
	mu       sync.Mutex
	channels int
	phase    float64
	step     float64
	level    float32
	fills    int
	frames   int
	last     time.Time
	maxGap   time.Duration
	states   []platform.AudioState
}

func newTone(rate, channels int) *tone {
	return &tone{channels: channels, step: 2 * math.Pi * toneHz / float64(rate), level: toneLevel}
}

func (s *tone) Fill(out []float32) {
	now := time.Now()
	for i := 0; i+s.channels <= len(out); i += s.channels {
		v := s.level * float32(math.Sin(s.phase))
		s.phase += s.step
		if s.phase > 2*math.Pi {
			s.phase -= 2 * math.Pi
		}
		for c := 0; c < s.channels; c++ {
			out[i+c] = v
		}
	}
	s.mu.Lock()
	if !s.last.IsZero() {
		s.maxGap = max(s.maxGap, now.Sub(s.last))
	}
	s.last = now
	s.fills++
	s.frames += len(out) / s.channels
	s.mu.Unlock()
}

func (s *tone) Capture([]float32) {}

func (s *tone) StateChanged(st platform.AudioState) {
	s.mu.Lock()
	s.states = append(s.states, st)
	s.mu.Unlock()
}

func (s *tone) fillCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.fills
}

func (s *tone) stateList() []platform.AudioState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]platform.AudioState(nil), s.states...)
}

// recording is a capture handler of a stereo device: it keeps the left channel (when keep is set) and checks the tone as it goes,
// so that a recording of ten minutes does not have to be kept.
type recording struct {
	mu      sync.Mutex
	keep    bool
	samples []float32
	frames  int
	// The checks that need no copy of the recording: the largest step between two samples, and how many steps are larger than a
	// sine of the level and the frequency can make (a jump: samples that were dropped, repeated or replaced by silence).
	prev    float32
	have    bool
	jumps   int
	peak    float32
	skip    int // frames at the start that are not checked
	maxStep float32
}

// stepLimit is the largest step of a sine of toneLevel and toneHz at rate, with half as much again for the rounding.
var stepLimit = float32(1.5 * toneLevel * 2 * math.Pi * toneHz / rate)

func (r *recording) Fill([]float32) {}

func (r *recording) Capture(in []float32) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := 0; i+2 <= len(in); i += 2 {
		v := in[i]
		r.frames++
		if r.keep {
			r.samples = append(r.samples, v)
		}
		if r.frames <= r.skip {
			r.prev, r.have = v, true
			continue
		}
		r.peak = max(r.peak, float32(math.Abs(float64(v))))
		if r.have {
			d := float32(math.Abs(float64(v - r.prev)))
			r.maxStep = max(r.maxStep, d)
			if d > stepLimit {
				r.jumps++
			}
		}
		r.prev, r.have = v, true
	}
}

func (r *recording) StateChanged(platform.AudioState) {}

func (r *recording) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.frames
}

func (r *recording) peakSince() float32 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.peak
}

func (r *recording) resetPeak() {
	r.mu.Lock()
	r.peak = 0
	r.mu.Unlock()
}

// amplitudeAndResidual fits a sine of toneHz to the samples and returns its amplitude and the part of the energy that it does not
// explain (0 for a pure tone).
func amplitudeAndResidual(samples []float32) (amp, residual float64) {
	var i, q, energy float64
	for n, v := range samples {
		ph := 2 * math.Pi * toneHz * float64(n) / rate
		i += float64(v) * math.Cos(ph)
		q += float64(v) * math.Sin(ph)
		energy += float64(v) * float64(v)
	}
	n := float64(len(samples))
	amp = 2 * math.Hypot(i, q) / n
	explained := amp * amp / 2 * n
	return amp, (energy - explained) / energy
}

func eventually(t *testing.T, what string, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func openMonitor(t *testing.T, svc *pulse.Service, name string, r *recording) platform.AudioDevice {
	t.Helper()
	dev, err := svc.Open(context.Background(), platform.AudioCapture, platform.AudioOpenOptions{ID: name, Format: stereo}, r)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = dev.Close() })
	return dev
}

func play(t *testing.T, svc *pulse.Service, h platform.AudioHandler, format platform.AudioFormat) platform.AudioDevice {
	t.Helper()
	dev, err := svc.Open(context.Background(), platform.AudioPlayback, platform.AudioOpenOptions{Format: format}, h)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = dev.Close() })
	return dev
}

func pactl(t *testing.T, args ...string) string {
	t.Helper()
	out, err := exec.Command("pactl", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("pactl %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// sinkIndex is the index that the server gives a sink, as a string.
func sinkIndex(t *testing.T, name string) string {
	t.Helper()
	for _, line := range strings.Split(pactl(t, "list", "short", "sinks"), "\n") {
		if f := strings.Fields(line); len(f) > 1 && f[1] == name {
			return f[0]
		}
	}
	t.Fatalf("no sink %s", name)
	return ""
}

// sinkOfTheInput is the index of the sink that the one playing stream is on.
func sinkOfTheInput(t *testing.T) string {
	t.Helper()
	lines := strings.Split(pactl(t, "list", "short", "sink-inputs"), "\n")
	if len(lines) != 1 {
		t.Fatalf("%d streams play, want one: %v", len(lines), lines)
	}
	return strings.Fields(lines[0])[1]
}

func control(t *testing.T, action string) {
	t.Helper()
	script := os.Getenv("GLUTTER_AUDIO_CONTROL")
	if script == "" {
		t.Skip("no control script of the server")
	}
	if out, err := exec.Command(script, action).CombinedOutput(); err != nil {
		t.Fatalf("%s %s: %v\n%s", script, action, err, out)
	}
}

func TestLiveDevices(t *testing.T) {
	requireServer(t)
	svc := pulse.New(pulse.Options{})
	defer svc.Close()
	sinks, err := svc.Devices(platform.AudioPlayback)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, d := range sinks {
		if d.ID == "glutter_sink" {
			found = true
			if !d.Default || d.Kind != platform.AudioPlayback || d.Name != "Glutter test sink" || d.Format.Channels != 2 || d.Format.Rate != rate {
				t.Errorf("the sink: %+v", d)
			}
		}
	}
	if !found {
		t.Errorf("sinks %+v have no glutter_sink", sinks)
	}
	sources, err := svc.Devices(platform.AudioCapture)
	if err != nil {
		t.Fatal(err)
	}
	found = false
	for _, d := range sources {
		if strings.HasSuffix(d.ID, ".monitor") {
			t.Errorf("the monitor %s is listed as a microphone", d.ID)
		}
		if d.ID == "glutter_source" {
			found = true
			if !d.Default || d.Kind != platform.AudioCapture {
				t.Errorf("the source: %+v", d)
			}
		}
	}
	if !found {
		t.Errorf("sources %+v have no glutter_source", sources)
	}
}

func TestLivePlaybackMatchesTheMonitor(t *testing.T) {
	requireServer(t)
	svc := pulse.New(pulse.Options{})
	defer svc.Close()
	rec := &recording{keep: true, skip: rate / 5}
	openMonitor(t, svc, "glutter_sink.monitor", rec)
	h := newTone(rate, 2)
	play(t, svc, h, stereo)
	time.Sleep(1500 * time.Millisecond)

	rec.mu.Lock()
	defer rec.mu.Unlock()
	samples := rec.samples[rate/5:]
	if len(samples) < rate {
		t.Fatalf("recorded %d frames in 1.5 s", len(rec.samples))
	}
	amp, residual := amplitudeAndResidual(samples)
	if math.Abs(amp-toneLevel) > 0.01 || residual > 0.001 {
		t.Errorf("recorded a tone of amplitude %.4f with %.4f of the energy outside it, want %.2f and 0", amp, residual, toneLevel)
	}
	if rec.jumps != 0 {
		t.Errorf("%d jumps in the recording (largest step %.4f, a sine's is %.4f)", rec.jumps, rec.maxStep, stepLimit/1.5)
	}
}

func TestLiveFormats(t *testing.T) {
	requireServer(t)
	for _, f := range []platform.AudioFormat{{Rate: 44100, Channels: 1}, {Rate: 96000, Channels: 2}, {Rate: 22050, Channels: 6}, {Rate: 8000, Channels: 2}, {Rate: 48000, Channels: 8}} {
		t.Run(strconv.Itoa(f.Rate)+"Hz"+strconv.Itoa(f.Channels), func(t *testing.T) {
			svc := pulse.New(pulse.Options{})
			defer svc.Close()
			rec := &recording{keep: true}
			openMonitor(t, svc, "glutter_sink.monitor", rec)
			h := newTone(f.Rate, f.Channels)
			dev := play(t, svc, h, f)
			if dev.Format() != f {
				t.Errorf("format %v", dev.Format())
			}
			eventually(t, "the tone on the monitor", 5*time.Second, func() bool { return rec.peakSince() > 0.3 })
			time.Sleep(300 * time.Millisecond)
			// A tone through a rate or a channel conversion keeps its pitch (the monitor is stereo at 48 kHz): the pitch is what is
			// compared, with the amplitude left to the server's mixing.
			rec.mu.Lock()
			s := rec.samples
			rec.mu.Unlock()
			s = s[max(0, len(s)-rate/4):]
			amp, residual := amplitudeAndResidual(s)
			if amp < 0.05 || residual > 0.05 {
				t.Errorf("the monitor has a tone of %.3f with %.3f of the energy outside it", amp, residual)
			}
		})
	}
}

func TestLiveCaptureFromTheNullSource(t *testing.T) {
	requireServer(t)
	for _, f := range []platform.AudioFormat{stereo, {Rate: 16000, Channels: 1}} {
		t.Run(strconv.Itoa(f.Rate), func(t *testing.T) {
			svc := pulse.New(pulse.Options{})
			defer svc.Close()
			h := &monoCapture{}
			dev, err := svc.Open(context.Background(), platform.AudioCapture, platform.AudioOpenOptions{ID: "glutter_source", Format: f}, h)
			if err != nil {
				t.Fatal(err)
			}
			if dev.Format() != f {
				t.Errorf("format %v", dev.Format())
			}
			start := time.Now()
			time.Sleep(time.Second)
			if err := dev.Close(); err != nil {
				t.Fatal(err)
			}
			elapsed := time.Since(start)
			h.mu.Lock()
			defer h.mu.Unlock()
			want := float64(f.Rate) * elapsed.Seconds()
			if got := float64(h.samples / f.Channels); math.Abs(got-want) > want*0.15 {
				t.Errorf("%v frames in %v, want about %v", got, elapsed, want)
			}
			if h.peak != 0 {
				t.Errorf("the null source made sound (peak %v)", h.peak)
			}
			n := h.samples
			time.Sleep(50 * time.Millisecond)
			if h.samples != n {
				t.Error("frames came after Close returned")
			}
		})
	}
}

type monoCapture struct {
	mu      sync.Mutex
	samples int
	peak    float32
}

func (*monoCapture) Fill([]float32) {}
func (m *monoCapture) Capture(in []float32) {
	m.mu.Lock()
	m.samples += len(in)
	for _, v := range in {
		m.peak = max(m.peak, float32(math.Abs(float64(v))))
	}
	m.mu.Unlock()
}
func (*monoCapture) StateChanged(platform.AudioState) {}

func TestLivePauseResume(t *testing.T) {
	requireServer(t)
	svc := pulse.New(pulse.Options{})
	defer svc.Close()
	rec := &recording{}
	openMonitor(t, svc, "glutter_sink.monitor", rec)
	h := newTone(rate, 2)
	dev := play(t, svc, h, stereo)
	eventually(t, "the tone", 5*time.Second, func() bool { return rec.peakSince() > 0.4 })
	if err := dev.Pause(); err != nil {
		t.Fatal(err)
	}
	n := h.fillCount()
	time.Sleep(500 * time.Millisecond) // what was buffered plays out
	rec.resetPeak()
	time.Sleep(300 * time.Millisecond)
	if p := rec.peakSince(); p > 0.001 {
		t.Errorf("the monitor has a peak of %v while paused", p)
	}
	if h.fillCount() != n {
		t.Errorf("the handler was called %d times while paused", h.fillCount()-n)
	}
	if err := dev.Resume(); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the tone after the resume", 5*time.Second, func() bool { return rec.peakSince() > 0.4 })
	if err := dev.Pause(); err != nil {
		t.Fatal(err)
	}
	if err := dev.Close(); err != nil { // closed while paused
		t.Fatal(err)
	}
}

func TestLiveOpenErrors(t *testing.T) {
	requireServer(t)
	svc := pulse.New(pulse.Options{})
	defer svc.Close()
	for _, kind := range []platform.AudioKind{platform.AudioPlayback, platform.AudioCapture} {
		dev, err := svc.Open(context.Background(), kind, platform.AudioOpenOptions{ID: "no_such_device"}, &recording{})
		if err == nil {
			// PipeWire's PulseAudio server falls back to the default device for a name it does not know; PulseAudio refuses it.
			t.Logf("kind %d: the server opened a device for a name it does not know", kind)
			_ = dev.Close()
			continue
		}
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != platform.TagAudioDeviceOpenFailed || !appErr.HasRoute("Service.Open.AudioDeviceOpenFailed") {
			t.Errorf("kind %d: %v", kind, err)
		}
	}
	// The service still works after that.
	dev := play(t, svc, newTone(rate, 2), stereo)
	if err := dev.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestLiveDefaultDeviceChange(t *testing.T) {
	requireServer(t)
	svc := pulse.New(pulse.Options{})
	defer svc.Close()
	changes := make(chan struct{}, 64)
	svc.SetChangeHandler(func() {
		select {
		case changes <- struct{}{}:
		default:
		}
	})
	if _, err := svc.Devices(platform.AudioPlayback); err != nil { // connects, and subscribes
		t.Fatal(err)
	}
	first, second := &recording{}, &recording{}
	openMonitor(t, svc, "glutter_sink.monitor", first)
	module := pactl(t, "load-module", "module-null-sink", "sink_name=glutter_sink2", "sink_properties=device.description=Second")
	t.Cleanup(func() { _ = exec.Command("pactl", "unload-module", module).Run() })
	eventually(t, "the change handler to hear of the new sink", 5*time.Second, func() bool { return len(changes) > 0 })
	openMonitor(t, svc, "glutter_sink2.monitor", second)

	h := newTone(rate, 2)
	play(t, svc, h, stereo) // the default device: no name
	eventually(t, "the tone on the first sink", 5*time.Second, func() bool { return first.peakSince() > 0.4 })
	if second.peakSince() > 0.001 {
		t.Fatal("the tone is on the second sink before the default changed")
	}

	for len(changes) > 0 {
		<-changes
	}
	pactl(t, "set-default-sink", "glutter_sink2")
	eventually(t, "the tone to move to the second sink", 10*time.Second, func() bool { return second.peakSince() > 0.4 })
	eventually(t, "the change handler to hear of the default", 5*time.Second, func() bool { return len(changes) > 0 })
	// The first sink is idle now. Its monitor is not the proof of that (an idle null sink leaves its last buffer on the monitor);
	// the server's own list is: the one stream that plays is on the second sink.
	if got, want := sinkOfTheInput(t), sinkIndex(t, "glutter_sink2"); got != want {
		t.Errorf("the stream is on sink %s, want glutter_sink2 (%s)", got, want)
	}
	sinks, err := svc.Devices(platform.AudioPlayback)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range sinks {
		if d.Default != (d.ID == "glutter_sink2") {
			t.Errorf("%s has Default %v after the default changed", d.ID, d.Default)
		}
	}
	if st := h.stateList(); len(st) != 0 {
		t.Errorf("states %v: a device that moved with the default is not lost", st)
	}

	// Back to the first, so that what follows finds the server as it was.
	pactl(t, "set-default-sink", "glutter_sink")
	eventually(t, "the stream to move back", 10*time.Second, func() bool { return sinkOfTheInput(t) == sinkIndex(t, "glutter_sink") })
}

func TestLiveServerLostAndBack(t *testing.T) {
	requireServer(t)
	svc := pulse.New(pulse.Options{})
	defer svc.Close()
	h := newTone(rate, 2)
	dev := play(t, svc, h, stereo)
	eventually(t, "fills", 5*time.Second, func() bool { return h.fillCount() > 5 })

	control(t, "stop")
	t.Cleanup(func() { _ = exec.Command(os.Getenv("GLUTTER_AUDIO_CONTROL"), "start").Run() })
	eventually(t, "the device to be lost", 10*time.Second, func() bool {
		st := h.stateList()
		return len(st) > 0 && st[len(st)-1] == platform.AudioLost
	})
	if st := h.stateList(); len(st) != 1 {
		t.Errorf("states %v, want a single AudioLost", st)
	}
	err := dev.Pause()
	if appErr, ok := errors.FromError(err); !ok || appErr.Tag != platform.TagAudioDeviceLost {
		t.Errorf("Pause of the lost device: %v", err)
	}
	_, err = svc.Devices(platform.AudioPlayback)
	if appErr, ok := errors.FromError(err); !ok || appErr.Tag != platform.TagAudioUnavailable || !appErr.HasRoute("Service.Devices->Service.connect.AudioUnavailable") {
		t.Errorf("Devices without a server: %v", err)
	}
	_, err = svc.Open(context.Background(), platform.AudioPlayback, platform.AudioOpenOptions{}, newTone(rate, 2))
	if appErr, ok := errors.FromError(err); !ok || appErr.Tag != platform.TagAudioUnavailable || !appErr.HasRoute("Service.Open->Service.connect.AudioUnavailable") {
		t.Errorf("Open without a server: %v", err)
	}
	if err := dev.Close(); err != nil {
		t.Errorf("Close of the lost device: %v", err)
	}

	control(t, "start")
	var devices []platform.AudioDeviceInfo
	eventually(t, "the server to answer again", 15*time.Second, func() bool {
		var err error
		devices, err = svc.Devices(platform.AudioPlayback)
		return err == nil
	})
	if len(devices) == 0 {
		t.Error("no devices after the server came back")
	}
	rec := &recording{}
	openMonitor(t, svc, "glutter_sink.monitor", rec)
	h2 := newTone(rate, 2)
	play(t, svc, h2, stereo)
	eventually(t, "the tone after the server came back", 10*time.Second, func() bool { return rec.peakSince() > 0.4 })
}

// TestLiveNoGapWhileDrawing plays a tone for GLUTTER_AUDIO_SOAK seconds (60 by default) while a goroutine draws a scene 60 times a
// second with the software renderer, and checks the recording of the monitor for gaps: the tone has no jump, the monitor got all
// the frames, and the handler was never kept waiting.
func TestLiveNoGapWhileDrawing(t *testing.T) {
	requireServer(t)
	seconds := 60
	if v := os.Getenv("GLUTTER_AUDIO_SOAK"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			t.Fatalf("GLUTTER_AUDIO_SOAK=%q is not a number of seconds", v)
		}
		seconds = n
	}
	if testing.Short() {
		seconds = 5
	}
	svc := pulse.New(pulse.Options{})
	defer svc.Close()
	rec := &recording{skip: rate}
	mon := openMonitor(t, svc, "glutter_sink.monitor", rec)
	h := newTone(rate, 2)
	dev := play(t, svc, h, stereo)
	eventually(t, "the tone", 5*time.Second, func() bool { return rec.count() > rate })

	var wg sync.WaitGroup
	stop := make(chan struct{})
	var frames, slow int
	wg.Add(1)
	go func() {
		defer wg.Done()
		tick := time.NewTicker(time.Second / 60)
		defer tick.Stop()
		for n := 0; ; n++ {
			select {
			case <-stop:
				return
			case <-tick.C:
			}
			begin := time.Now()
			scene := gputest.CanvasScene(func(c *ui.Canvas) {
				for i := 0; i < 20; i++ {
					c.DrawCircle(ui.Offset{Dx: 40 + float64((n*5+i*37)%400), Dy: 40 + float64((n*3+i*53)%280)}, 24, gputest.FillPaint(0xFF000000|uint32(i*0x050607)))
				}
			})
			if _, err := soft.Render(scene, 480, 360); err != nil {
				t.Error(err)
				return
			}
			frames++
			if time.Since(begin) > time.Second/60 {
				slow++
			}
		}
	}()

	startFrames, startFills := rec.count(), h.fillCount()
	start := time.Now()
	time.Sleep(time.Duration(seconds) * time.Second)
	elapsed := time.Since(start)
	close(stop)
	wg.Wait()
	if err := mon.Close(); err != nil {
		t.Fatal(err)
	}
	if err := dev.Close(); err != nil {
		t.Fatal(err)
	}

	underruns := dev.(interface{ Underruns() int }).Underruns()
	rec.mu.Lock()
	defer rec.mu.Unlock()
	h.mu.Lock()
	defer h.mu.Unlock()
	got := rec.frames - startFrames
	want := rate * elapsed.Seconds()
	t.Logf("%v: drew %d frames (%d took longer than 1/60 s), the handler was called %d times (the longest gap %v), the monitor got %d frames (want about %.0f), %d jumps (largest step %.4f), the server ran out of frames %d times",
		elapsed.Round(time.Second), frames, slow, h.fills-startFills, h.maxGap, got, want, rec.jumps, rec.maxStep, underruns)
	if rec.jumps != 0 {
		t.Errorf("%d jumps in the recording of the tone: samples were lost", rec.jumps)
	}
	if math.Abs(float64(got)-want) > want*0.01 {
		t.Errorf("the monitor got %d frames in %v, want %.0f (a gap makes fewer)", got, elapsed, want)
	}
	if h.maxGap > 250*time.Millisecond {
		t.Errorf("the handler was kept waiting %v", h.maxGap)
	}
	if st := h.states; len(st) != 0 {
		t.Errorf("states %v", st)
	}
}
