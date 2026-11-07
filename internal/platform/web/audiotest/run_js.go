//go:build js && wasm

package main

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"os"
	"sync"
	"syscall/js"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/web"
	m "github.com/rm4n0s/glutter/material"
)

// observations is what the program posts as the result "audio". A field that a mode does not fill stays empty.
type observations struct {
	Mode string `json:"mode"`
	// Isolated is crossOriginIsolated of the page.
	Isolated bool `json:"isolated"`
	// OpenTag is the tag of the error of Open, "" when it opened.
	OpenTag string `json:"openTag"`
	// Rate and Channels are the format of the device; Shared says that its ring is a SharedArrayBuffer.
	Rate     int  `json:"rate"`
	Channels int  `json:"channels"`
	Shared   bool `json:"shared"`
	// States are the states that the handler was told, in order.
	States []string `json:"states"`

	// play: what the tap on the node read while the device ran (the peak and the sign changes of a window of WindowFrames
	// frames), the frames the worklet took in a second, the underruns of the ring, the peak while the device was paused and the
	// frames it took then, and the frames it took after the resume.
	Peak            float64 `json:"peak"`
	Crossings       int     `json:"crossings"`
	WindowFrames    int     `json:"windowFrames"`
	FramesPerSecond float64 `json:"framesPerSecond"`
	Underruns       float64 `json:"underruns"`
	Overruns        float64 `json:"overruns"`
	PausedPeak      float64 `json:"pausedPeak"`
	PausedFrames    float64 `json:"pausedFrames"`
	ResumedPeak     float64 `json:"resumedPeak"`
	ResumedFrames   float64 `json:"resumedFrames"`
	// StressUnderruns are the underruns that came while the program kept the thread busy for 50 ms at a time (a heavy frame).
	StressUnderruns float64 `json:"stressUnderruns"`
	StressPeak      float64 `json:"stressPeak"`
	CloseError      string  `json:"closeError"`
	ClosedOrGone    bool    `json:"closedOrGone"`

	// gesture: the state of the context after the open and after half a second, the calls of resume before and after a gesture,
	// and the frames that the handler was asked for while the context waited.
	ContextState      string `json:"contextState"`
	ResumeBefore      int    `json:"resumeBefore"`
	ResumeAfter       int    `json:"resumeAfter"`
	FramesWhileWaited int64  `json:"framesWhileWaited"`

	// capture: the frames the handler got in two seconds and whether any of them was not silence.
	CaptureFrames  int64 `json:"captureFrames"`
	CaptureNonZero bool  `json:"captureNonZero"`
}

// tone plays a sine of 440 Hz on every channel and records what it is asked.
type tone struct {
	mu       sync.Mutex
	channels int
	step     float64
	phase    float64
	frames   int64
	states   []string
	captured int64
	nonZero  bool
}

func (t *tone) configure(format platform.AudioFormat) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.channels = format.Channels
	t.step = 2 * math.Pi * 440 / float64(format.Rate)
}

func (t *tone) Fill(out []float32) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.channels == 0 {
		clear(out)
		return
	}
	for i := 0; i+t.channels <= len(out); i += t.channels {
		v := float32(0.5 * math.Sin(t.phase))
		t.phase += t.step
		for c := 0; c < t.channels; c++ {
			out[i+c] = v
		}
	}
	t.frames += int64(len(out) / t.channels)
}

func (t *tone) Capture(in []float32) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.channels > 0 {
		t.captured += int64(len(in) / t.channels)
	}
	for _, v := range in {
		if v != 0 {
			t.nonZero = true
			break
		}
	}
}

func (t *tone) StateChanged(s platform.AudioState) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.states = append(t.states, stateName(s))
}

func (t *tone) seen(state string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, s := range t.states {
		if s == state {
			return true
		}
	}
	return false
}

func (t *tone) framesNow() int64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.frames
}

func stateName(s platform.AudioState) string {
	switch s {
	case platform.AudioRunning:
		return "running"
	case platform.AudioPaused:
		return "paused"
	case platform.AudioSuspended:
		return "suspended"
	case platform.AudioLost:
		return "lost"
	}
	return "closed"
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	p := newProbe()
	go func() {
		postResult("audio", observe(p))
	}()
	return m.RunApp(newAudioApp(p), m.RunAppOpts{Window: m.WindowConfig{Title: "Audio test"}})
}

func tagOf(err error) string {
	if appErr, ok := errors.FromError(err); ok {
		return appErr.Tag
	}
	return "none"
}

// installScripts defines the functions the driver calls in the page: a tap on a worklet node that measures what it plays, and a spy
// on the resume of the context.
func installScripts() {
	js.Global().Get("Function").New(`
		globalThis.glutterTap = function (index) {
			var reg = globalThis.__glutterAudio;
			var an = reg.context.createAnalyser();
			an.fftSize = 2048;
			reg.nodes[index].connect(an);
			globalThis.glutterAn = an;
			globalThis.glutterBuf = new Float32Array(an.fftSize);
			return an.fftSize;
		};
		globalThis.glutterMeasure = function () {
			var an = globalThis.glutterAn, b = globalThis.glutterBuf;
			an.getFloatTimeDomainData(b);
			var peak = 0, cross = 0;
			for (var i = 0; i < b.length; i++) {
				var v = Math.abs(b[i]);
				if (v > peak) peak = v;
				if (i > 0 && (b[i - 1] < 0) !== (b[i] < 0)) cross++;
			}
			return [peak, cross];
		};
		globalThis.glutterSpy = function () {
			var ctx = globalThis.__glutterAudio.context;
			var orig = ctx.resume.bind(ctx);
			globalThis.glutterResumes = 0;
			ctx.resume = function () { globalThis.glutterResumes++; return orig(); };
		};
	`).Invoke()
}

// stat reads a number of the first device in the debug registry's statistics.
func stat(key string) float64 {
	list := js.Global().Get("__glutterAudio").Call("stats")
	if list.Length() == 0 {
		return -1
	}
	v := list.Index(0).Get(key)
	if v.Type() != js.TypeNumber {
		return -1
	}
	return v.Float()
}

func statBool(key string) bool {
	list := js.Global().Get("__glutterAudio").Call("stats")
	return list.Length() > 0 && list.Index(0).Get(key).Truthy()
}

// observe waits for the app and runs the mode. It runs on a goroutine of its own: Open of a recording waits for the user.
func observe(p *probe) observations {
	o := observations{Mode: os.Getenv("GLUTTER_AUDIO_MODE")}
	<-p.ready
	g := js.Global()
	o.Isolated = g.Get("crossOriginIsolated").Truthy()
	installScripts()
	var host platform.AudioHost
	if !p.binding.PlatformDispatcher().HostAs(&host) || host.Audio() == nil {
		o.OpenTag = "NoAudioService"
		return o
	}
	svc := host.Audio()
	h := &tone{}
	ctx := context.Background()
	kind := platform.AudioPlayback
	if o.Mode == "capture" {
		kind = platform.AudioCapture
	}
	dev, err := svc.Open(ctx, kind, platform.AudioOpenOptions{}, h)
	if err != nil {
		o.OpenTag = tagOf(err)
		return o
	}
	format := dev.Format()
	h.configure(format)
	o.Rate, o.Channels = format.Rate, format.Channels
	switch o.Mode {
	case "play":
		play(&o, dev, h)
	case "gesture":
		gesture(&o, dev, h)
	case "capture":
		capture(&o, dev, h)
	}
	if o.Mode != "play" {
		if err := dev.Close(); err != nil {
			o.CloseError = tagOf(err)
		}
	}
	h.mu.Lock()
	o.States = append([]string(nil), h.states...)
	h.mu.Unlock()
	return o
}

func waitFor(timeout time.Duration, cond func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return cond()
}

func measure() (peak float64, crossings int) {
	r := js.Global().Call("glutterMeasure")
	return r.Index(0).Float(), r.Index(1).Int()
}

func play(o *observations, dev platform.AudioDevice, h *tone) {
	g := js.Global()
	waitFor(10*time.Second, func() bool { return h.seen("running") })
	o.ContextState = g.Get("__glutterAudio").Get("context").Get("state").String()
	if !h.seen("running") {
		return
	}
	o.Shared = statBool("shared")
	o.WindowFrames = g.Call("glutterTap", 0).Int()
	time.Sleep(time.Second)

	r0 := stat("read")
	t0 := time.Now()
	time.Sleep(time.Second)
	r1 := stat("read")
	o.FramesPerSecond = (r1 - r0) / time.Since(t0).Seconds()
	o.Peak, o.Crossings = measure()
	o.Underruns = stat("underruns")
	o.Overruns = stat("overruns")

	if err := dev.Pause(); err != nil {
		log.Println("pause:", err)
	}
	time.Sleep(300 * time.Millisecond)
	p0 := stat("read")
	time.Sleep(300 * time.Millisecond)
	o.PausedFrames = stat("read") - p0
	o.PausedPeak, _ = measure()

	if err := dev.Resume(); err != nil {
		log.Println("resume:", err)
	}
	s0 := stat("read")
	time.Sleep(500 * time.Millisecond)
	o.ResumedFrames = stat("read") - s0
	o.ResumedPeak, _ = measure()

	// Frames of 50 ms: the one thread of the page is busy, so neither the pump nor the worklet's messages are served, and the buffer
	// of 100 ms must carry the sound over each of them.
	u0 := stat("underruns")
	end := time.Now().Add(1500 * time.Millisecond)
	for time.Now().Before(end) {
		busy := time.Now().Add(50 * time.Millisecond)
		for time.Now().Before(busy) {
		}
		time.Sleep(20 * time.Millisecond)
	}
	o.StressUnderruns = stat("underruns") - u0
	o.StressPeak, _ = measure()

	if err := dev.Close(); err != nil {
		o.CloseError = tagOf(err)
	}
	time.Sleep(200 * time.Millisecond)
	o.ClosedOrGone = js.Global().Get("__glutterAudio").Call("stats").Length() == 0 || statBool("closed")
}

func gesture(o *observations, dev platform.AudioDevice, h *tone) {
	g := js.Global()
	g.Call("glutterSpy")
	time.Sleep(500 * time.Millisecond)
	o.ContextState = g.Get("__glutterAudio").Get("context").Get("state").String()
	o.ResumeBefore = g.Get("glutterResumes").Int()
	o.FramesWhileWaited = h.framesNow()
	// A gesture that the browser does not trust still reaches the page's listeners, which ask the context to resume.
	g.Get("document").Call("dispatchEvent", g.Get("Event").New("pointerdown"))
	time.Sleep(200 * time.Millisecond)
	o.ResumeAfter = g.Get("glutterResumes").Int()
}

func capture(o *observations, dev platform.AudioDevice, h *tone) {
	time.Sleep(2 * time.Second)
	h.mu.Lock()
	o.CaptureFrames, o.CaptureNonZero = h.captured, h.nonZero
	h.mu.Unlock()
}

// postResult posts a result to the test server; the answer is not waited for.
func postResult(kind string, data any) {
	body, err := json.Marshal(map[string]any{"kind": kind, "data": data})
	if err != nil {
		panic(err)
	}
	init := map[string]any{"method": "POST", "headers": map[string]any{"Content-Type": "application/json"}, "body": string(body)}
	js.Global().Call("fetch", web.ResultsPath, init)
}
