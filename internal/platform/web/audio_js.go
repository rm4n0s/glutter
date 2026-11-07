//go:build js && wasm

package web

import (
	"context"
	"io"
	"log/slog"
	"runtime/debug"
	"sync"
	"syscall/js"
	"time"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/platform"
)

// The audio of the web (SPEC §28.3, step 31.6), designed after the Emscripten driver of SDL 3 (src/audio/emscripten/
// SDL_emscriptenaudio.c; zlib license, third_party/sdl/LICENSE), translated to Go and changed: SDL feeds a ScriptProcessorNode from
// the callback of the page, and here an AudioWorklet (glutter-audio-worklet.js, which glutter-web writes into the build) plays what
// the Go program keeps in a ring of frames. The ring is a SharedArrayBuffer when the page is cross-origin isolated (glutter-web
// serve sends the headers that make it so) and a pair of rings joined by postMessage when it is not.
//
// The Go program is the single thread of the page, so the handler of a device runs on a goroutine of its own (the pump, every 10 ms)
// that is served between the events of the page; it fills the ring up to the buffer (100 ms by default), which is what carries the
// sound over the frames that take long. A page may not make sound before the user has pressed a key or clicked: the AudioContext
// starts suspended, the devices say AudioSuspended, and the first gesture resumes it (the listeners are on the document, in the
// capture phase). Open never waits for the worklet or the gesture (SPEC §23.3); a capture waits for the answer of the user to the
// microphone, as it must to say AudioCapturePermissionDenied, so an app opens it from a goroutine of its own and not from the UI
// goroutine.

// webAudio is the audio service of the page.
type webAudio struct {
	log *slog.Logger

	mu       sync.Mutex
	ctx      js.Value
	hasCtx   bool
	module   moduleState
	moduleCh chan struct{} // closed when the module is loaded or failed
	devices  map[*audioDevice]struct{}
	handler  func()
	listed   []MediaDevice
	listing  bool
	closed   bool
	pumping  bool
	stop     chan struct{}

	callbacksMade                                               bool
	stateFn, gestureFn, moduleOK, moduleFail, listFn, devChange js.Func
	gestureOn                                                   bool
	debug                                                       js.Value
}

type moduleState uint8

const (
	moduleNone moduleState = iota
	moduleLoading
	moduleReady
	moduleFailed
)

// gestures are the events that give the page the right to make sound.
var gestures = []string{"pointerdown", "mousedown", "touchend", "keydown", "click"}

func newWebAudio(log *slog.Logger) *webAudio {
	if log == nil {
		log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return &webAudio{log: log, devices: map[*audioDevice]struct{}{}}
}

var _ platform.Audio = (*webAudio)(nil)

// audioService makes the audio service on the first call.
func (p *Platform) audioService() platform.Audio {
	if p.audio == nil {
		p.audio = newWebAudio(p.log)
	}
	return p.audio
}

// makeCallbacks makes the callbacks that the browser calls, once. mu is held.
func (a *webAudio) makeCallbacks() {
	if a.callbacksMade {
		return
	}
	a.callbacksMade = true
	a.stateFn = js.FuncOf(func(js.Value, []js.Value) any {
		a.contextStateChanged()
		return nil
	})
	a.gestureFn = js.FuncOf(func(js.Value, []js.Value) any {
		a.resume()
		return nil
	})
	a.moduleOK = js.FuncOf(func(js.Value, []js.Value) any {
		a.moduleSettled(moduleReady, "")
		return nil
	})
	a.moduleFail = js.FuncOf(func(_ js.Value, args []js.Value) any {
		reason := ""
		if len(args) > 0 {
			reason = describe(args[0])
		}
		a.moduleSettled(moduleFailed, reason)
		return nil
	})
	a.listFn = js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) > 0 {
			a.listDone(args[0])
		}
		return nil
	})
	a.devChange = js.FuncOf(func(js.Value, []js.Value) any {
		a.startListing()
		return nil
	})
}

// describe is the text of a JavaScript error or rejection.
func describe(v js.Value) string {
	if !isObject(v) {
		return v.String()
	}
	name, msg := v.Get("name"), v.Get("message")
	if name.Type() == js.TypeString && msg.Type() == js.TypeString {
		return name.String() + ": " + msg.String()
	}
	return js.Global().Get("String").Invoke(v).String()
}

// errorName is the name of a DOMException, "" for anything else.
func errorName(v js.Value) string {
	if isObject(v) && v.Get("name").Type() == js.TypeString {
		return v.Get("name").String()
	}
	return ""
}

// ensureContext makes the AudioContext, once: it takes the rate that the first device asks for when the browser lets it. It fails
// with AudioUnavailable when the page has no Web Audio, or no audio worklet (which needs a secure context).
func (a *webAudio) ensureContext(rate int) (js.Value, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed {
		return js.Undefined(), errors.New(platform.TagAudioUnavailable, "the audio of the page was closed")
	}
	if a.hasCtx {
		return a.ctx, nil
	}
	ctor := global().Get("AudioContext")
	if !isObject(ctor) {
		ctor = global().Get("webkitAudioContext")
	}
	if !isObject(ctor) {
		return js.Undefined(), errors.New(platform.TagAudioUnavailable, "the browser has no Web Audio")
	}
	var ctx js.Value
	made := false
	if rate > 0 {
		made = try(func() { ctx = ctor.New(map[string]any{"latencyHint": "interactive", "sampleRate": rate}) })
	}
	if !made {
		made = try(func() { ctx = ctor.New(map[string]any{"latencyHint": "interactive"}) })
	}
	if !made {
		return js.Undefined(), errors.New(platform.TagAudioUnavailable, "the browser could not make an audio context")
	}
	if !isObject(ctx.Get("audioWorklet")) {
		try(func() { ctx.Call("close") })
		return js.Undefined(), errors.New(platform.TagAudioUnavailable, "the browser has no audio worklet: the page must be served over https or from localhost")
	}
	a.makeCallbacks()
	a.ctx, a.hasCtx = ctx, true
	ctx.Call("addEventListener", "statechange", a.stateFn)
	if ctx.Get("state").String() != "running" {
		a.listenForGestures(true)
	}
	if foundation.DebugMode {
		a.registerDebug()
	}
	return ctx, nil
}

// listenForGestures starts or stops waiting for the first gesture. mu is held.
func (a *webAudio) listenForGestures(on bool) {
	if on == a.gestureOn {
		return
	}
	a.gestureOn = on
	doc := global().Get("document")
	for _, ev := range gestures {
		if on {
			doc.Call("addEventListener", ev, a.gestureFn, true)
		} else {
			doc.Call("removeEventListener", ev, a.gestureFn, true)
		}
	}
}

// resume asks the context to run: it is called by the gesture listener, which is where the browser allows it.
func (a *webAudio) resume() {
	a.mu.Lock()
	ctx, ok := a.ctx, a.hasCtx
	a.mu.Unlock()
	if !ok {
		return
	}
	try(func() {
		p := ctx.Call("resume")
		// A rejection is the browser saying that this was no gesture it counts; the next one is tried.
		p.Call("catch", a.moduleFail)
	})
}

// contextStateChanged follows the context: a running one starts the devices that wait, and one that stops (an interruption, the
// browser's decision) suspends them.
func (a *webAudio) contextStateChanged() {
	a.mu.Lock()
	if !a.hasCtx {
		a.mu.Unlock()
		return
	}
	state := a.ctx.Get("state").String()
	if state == "running" {
		a.listenForGestures(false)
	} else if state != "closed" {
		a.listenForGestures(true)
	}
	list := a.deviceList()
	a.mu.Unlock()
	for _, d := range list {
		d.contextState(state)
	}
}

// deviceList is a copy of the set of devices. mu is held.
func (a *webAudio) deviceList() []*audioDevice {
	list := make([]*audioDevice, 0, len(a.devices))
	for d := range a.devices {
		list = append(list, d)
	}
	return list
}

// loadModule loads the worklet, once. mu is held.
func (a *webAudio) loadModule() {
	if a.module != moduleNone {
		return
	}
	a.module = moduleLoading
	a.moduleCh = make(chan struct{})
	url := global().Get("URL").New(AudioWorkletFile, global().Get("document").Get("baseURI")).Get("href")
	var p js.Value
	if !try(func() { p = a.ctx.Get("audioWorklet").Call("addModule", url) }) {
		a.module = moduleFailed
		close(a.moduleCh)
		return
	}
	p.Call("then", a.moduleOK, a.moduleFail)
}

// moduleSettled is the answer of addModule (or of a resume that was refused).
func (a *webAudio) moduleSettled(state moduleState, reason string) {
	a.mu.Lock()
	if a.module != moduleLoading {
		// The refusal of a resume, which comes through the same function: it only means to try again.
		a.mu.Unlock()
		return
	}
	a.module = state
	close(a.moduleCh)
	list := a.deviceList()
	a.mu.Unlock()
	if state == moduleFailed {
		a.log.Error("audio: the audio worklet could not be loaded; is glutter-audio-worklet.js next to index.html?", "reason", reason)
	}
	for _, d := range list {
		d.tryStart()
	}
}

// Devices lists the devices of a kind: the browser's list (which it gives only asynchronously, and without names before the user
// allowed a microphone), or the default device.
//
// Flutter: no counterpart
func (a *webAudio) Devices(kind platform.AudioKind) ([]platform.AudioDeviceInfo, error) {
	a.mu.Lock()
	a.makeCallbacks()
	listed := a.listed
	a.mu.Unlock()
	a.startListing()
	if list := AudioDevicesOf(listed, kind); len(list) > 0 {
		return list, nil
	}
	return []platform.AudioDeviceInfo{DefaultAudioDevice(kind)}, nil
}

// startListing asks the browser for its list of devices, once at a time.
func (a *webAudio) startListing() {
	a.mu.Lock()
	md := global().Get("navigator").Get("mediaDevices")
	if a.listing || a.closed || !isObject(md) || md.Get("enumerateDevices").Type() != js.TypeFunction {
		a.mu.Unlock()
		return
	}
	a.listing = true
	a.makeCallbacks()
	first := a.listed == nil
	fn := a.listFn
	a.mu.Unlock()
	if first {
		try(func() { md.Call("addEventListener", "devicechange", a.devChange) })
	}
	if !try(func() { md.Call("enumerateDevices").Call("then", fn) }) {
		a.mu.Lock()
		a.listing = false
		a.mu.Unlock()
	}
}

// listDone takes the list that the browser gave and tells the change handler when it differs.
func (a *webAudio) listDone(arr js.Value) {
	var list []MediaDevice
	for i := 0; i < arr.Length(); i++ {
		d := arr.Index(i)
		list = append(list, MediaDevice{Kind: d.Get("kind").String(), ID: d.Get("deviceId").String(), Label: d.Get("label").String()})
	}
	a.mu.Lock()
	a.listing = false
	changed := len(list) != len(a.listed)
	for i := 0; !changed && i < len(list); i++ {
		changed = list[i] != a.listed[i]
	}
	if a.listed == nil {
		changed = changed || len(list) > 0
	}
	a.listed = append([]MediaDevice{}, list...)
	fn := a.handler
	a.mu.Unlock()
	if changed && fn != nil {
		fn()
	}
}

// SetChangeHandler sets what is called when the browser's list of devices changes.
//
// Flutter: no counterpart
func (a *webAudio) SetChangeHandler(fn func()) {
	a.mu.Lock()
	a.handler = fn
	a.mu.Unlock()
}

// Open opens a device and returns at once: the device says AudioSuspended until the page may make sound (the first gesture) and the
// worklet is loaded, and then starts by itself. It fails with AudioUnavailable when the browser has no Web Audio or no worklet,
// AudioFormatUnsupported for a format out of the limits, and AudioDeviceOpenFailed for an output that is not the default. A capture
// waits for the answer of the user to the microphone, and fails with AudioCapturePermissionDenied when the answer is no.
//
// Flutter: no counterpart
func (a *webAudio) Open(ctx context.Context, kind platform.AudioKind, opts platform.AudioOpenOptions, h platform.AudioHandler) (platform.AudioDevice, error) {
	if err := ctx.Err(); err != nil {
		return nil, errors.NewErr(platform.TagAudioDeviceOpenFailed, err).SetMetadata("id", opts.ID)
	}
	if r, c := opts.Format.Rate, opts.Format.Channels; r < 0 || c < 0 || c > 8 || (r != 0 && (r < 1000 || r > 768000)) {
		return nil, errors.New(platform.TagAudioFormatUnsupported, "web devices take 1 to 8 channels and a rate of 1000 to 768000", "rate", r, "channels", c)
	}
	if kind == platform.AudioPlayback && opts.ID != "" && opts.ID != "default" {
		return nil, errors.New(platform.TagAudioDeviceOpenFailed, "a page plays on the default output of the browser", "id", opts.ID)
	}
	actx, err := a.ensureContext(opts.Format.Rate)
	if err != nil {
		return nil, err
	}
	d := &audioDevice{a: a, kind: kind, h: h, opts: opts}
	d.format.Rate = int(actx.Get("sampleRate").Float())
	if kind == platform.AudioPlayback {
		d.format.Channels = PlaybackChannels(opts.Format.Channels, int(number(actx.Get("destination").Get("maxChannelCount"), 2)))
	} else {
		d.format.Channels = opts.Format.Channels
		if d.format.Channels == 0 {
			d.format.Channels = 1
		}
		if err := d.openMicrophone(ctx); err != nil {
			return nil, err
		}
	}
	d.target = AudioTarget(opts.BufferFrames, d.format.Rate)
	d.capacity = RingFrames(d.format.Rate, time.Duration(d.target)*time.Second/time.Duration(d.format.Rate))
	a.mu.Lock()
	if a.closed {
		a.mu.Unlock()
		d.dropMicrophone()
		return nil, errors.New(platform.TagAudioUnavailable, "the audio of the page was closed")
	}
	a.devices[d] = struct{}{}
	a.loadModule()
	a.mu.Unlock()
	d.tryStart()
	if !d.isStarted() {
		d.tellWaiting(true, true)
		// The state may have changed while the device was told: a context that started in between has already tried.
		d.tryStart()
	}
	return d, nil
}

// startPump starts the goroutine that serves the devices, if it does not run. mu is held.
func (a *webAudio) startPump() {
	if a.pumping {
		return
	}
	a.pumping = true
	a.stop = make(chan struct{})
	go func(stop <-chan struct{}) {
		tick := time.NewTicker(AudioPumpInterval)
		defer tick.Stop()
		for {
			select {
			case <-stop:
				return
			case <-tick.C:
			}
			a.mu.Lock()
			list := a.deviceList()
			a.mu.Unlock()
			for _, d := range list {
				d.pump()
			}
		}
	}(a.stop)
}

// stopPumpIfIdle ends the pump when no device is left.
func (a *webAudio) stopPumpIfIdle() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.devices) == 0 && a.pumping {
		a.pumping = false
		close(a.stop)
	}
}

// Close closes the devices and the context.
func (a *webAudio) Close() {
	a.mu.Lock()
	a.closed = true
	list := a.deviceList()
	a.mu.Unlock()
	for _, d := range list {
		if err := d.Close(); err != nil {
			a.log.Debug("audio: a device could not be closed", "err", err)
		}
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.hasCtx {
		a.listenForGestures(false)
		try(func() { a.ctx.Call("removeEventListener", "statechange", a.stateFn) })
		try(func() { a.ctx.Call("close") })
		a.hasCtx = false
	}
	if a.callbacksMade {
		md := global().Get("navigator").Get("mediaDevices")
		if isObject(md) {
			try(func() { md.Call("removeEventListener", "devicechange", a.devChange) })
		}
		a.stateFn.Release()
		a.gestureFn.Release()
		a.moduleOK.Release()
		a.moduleFail.Release()
		a.listFn.Release()
		a.devChange.Release()
		a.callbacksMade = false
	}
}

// registerDebug exposes the context, the nodes and the numbers of the rings as globalThis.__glutterAudio, in debug builds only: the
// browser tests put a tap on the nodes and read the underruns. mu is held.
func (a *webAudio) registerDebug() {
	reg := global().Get("__glutterAudio")
	if !isObject(reg) {
		reg = global().Get("Object").New()
		reg.Set("nodes", global().Get("Array").New())
		global().Set("__glutterAudio", reg)
		reg.Set("stats", js.FuncOf(func(js.Value, []js.Value) any { return a.statsValue() }))
	}
	reg.Set("context", a.ctx)
	a.debug = reg
}

// statsValue is the numbers of every device, as a JavaScript array.
func (a *webAudio) statsValue() any {
	a.mu.Lock()
	list := a.deviceList()
	a.mu.Unlock()
	out := make([]any, 0, len(list))
	for _, d := range list {
		out = append(out, d.stats())
	}
	return out
}

// audioDevice is an open device of the web: a worklet node and its ring.
type audioDevice struct {
	a    *webAudio
	kind platform.AudioKind
	h    platform.AudioHandler
	opts platform.AudioOpenOptions
	// format, target and capacity are set before the device is shared.
	format   platform.AudioFormat
	target   int
	capacity int

	// The microphone: the stream and its source, which Open makes before the device exists.
	stream js.Value

	// nmu orders the notices of Suspended and Running (tellWaiting); it is taken before mu.
	nmu sync.Mutex

	// mu guards what follows, and is held while the handler runs, so that Close finds it idle.
	mu            sync.Mutex
	started       bool
	paused        bool
	closed        bool
	lost          bool
	toldSuspended bool
	shared        bool
	node          js.Value
	source        js.Value
	silence       js.Value
	i32, u8       js.Value
	msgFn         js.Func
	hasMsgFn      bool
	written       uint32 // message mode playback: the frames that were sent
	reported      uint32 // message mode playback: the frames the worklet says it played
	underruns     int
	scratch       []float32
}

var _ platform.AudioDevice = (*audioDevice)(nil)

// Format returns the format of the frames of the handler.
func (d *audioDevice) Format() platform.AudioFormat { return d.format }

func (d *audioDevice) isStarted() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.started
}

// openMicrophone asks the browser for the microphone and waits for the answer of the user, until ctx ends.
func (d *audioDevice) openMicrophone(ctx context.Context) error {
	md := global().Get("navigator").Get("mediaDevices")
	if !isObject(md) || md.Get("getUserMedia").Type() != js.TypeFunction {
		return errors.New(platform.TagAudioUnavailable, "the browser cannot record: the page must be served over https or from localhost")
	}
	audio := map[string]any{"echoCancellation": false, "noiseSuppression": false, "autoGainControl": false, "channelCount": map[string]any{"ideal": d.format.Channels}}
	if id := d.opts.ID; id != "" && id != "default" {
		audio["deviceId"] = map[string]any{"exact": id}
	}
	type answer struct {
		stream js.Value
		name   string
		reason string
	}
	got := make(chan answer, 1)
	okFn := js.FuncOf(func(_ js.Value, args []js.Value) any {
		got <- answer{stream: args[0]}
		return nil
	})
	failFn := js.FuncOf(func(_ js.Value, args []js.Value) any {
		got <- answer{name: errorName(args[0]), reason: describe(args[0])}
		return nil
	})
	defer okFn.Release()
	defer failFn.Release()
	if !try(func() { md.Call("getUserMedia", map[string]any{"audio": audio}).Call("then", okFn, failFn) }) {
		return errors.New(platform.TagAudioDeviceOpenFailed, "the browser did not take the request for the microphone")
	}
	select {
	case r := <-got:
		if r.name != "" || !isObject(r.stream) {
			tag := MediaErrorTag(r.name)
			return errors.New(tag, "the microphone could not be opened", "error", r.name, "reason", r.reason)
		}
		d.stream = r.stream
		// The track says how many channels the microphone has.
		if tracks := r.stream.Call("getAudioTracks"); tracks.Length() > 0 {
			if n := tracks.Index(0).Call("getSettings").Get("channelCount"); n.Type() == js.TypeNumber && n.Int() >= 1 && n.Int() <= 8 && d.opts.Format.Channels == 0 {
				d.format.Channels = n.Int()
			}
		}
		// The browser lists the microphones with their names now.
		go d.a.startListing()
		return nil
	case <-ctx.Done():
		// The answer that comes later is dropped, and the microphone it opens is let go.
		go func() {
			r := <-got
			if isObject(r.stream) {
				stopTracks(r.stream)
			}
		}()
		return errors.NewErr(platform.TagAudioDeviceOpenFailed, ctx.Err()).SetMetadata("id", d.opts.ID)
	}
}

// stopTracks ends the tracks of a stream: the browser's sign that the microphone is in use goes out.
func stopTracks(stream js.Value) {
	try(func() {
		tracks := stream.Call("getTracks")
		for i := 0; i < tracks.Length(); i++ {
			tracks.Index(i).Call("stop")
		}
	})
}

// dropMicrophone lets go of the microphone of a device that was not opened.
func (d *audioDevice) dropMicrophone() {
	if isObject(d.stream) {
		stopTracks(d.stream)
	}
}

// contextState is told by the service that the context is in a state.
func (d *audioDevice) contextState(state string) {
	switch state {
	case "running":
		d.tryStart()
		d.tellWaiting(false, false)
	case "closed":
		d.lose("the audio context was closed")
	default:
		d.mu.Lock()
		started := d.started
		d.mu.Unlock()
		if started {
			d.tellWaiting(true, false)
		}
	}
}

// tellWaiting tells the handler that the device waits for the browser (Suspended) or goes on (Running), once for each change. It
// tells nothing to a device that is closed or lost, and Running to none that has not started. A device that waits because it has not
// started (ifStopped) is not told when it has started in the meantime. nmu keeps the order of the two states: a device that starts
// while it is being told that it waits is told that it runs after that.
func (d *audioDevice) tellWaiting(waiting, ifStopped bool) {
	d.nmu.Lock()
	defer d.nmu.Unlock()
	d.mu.Lock()
	if d.closed || d.lost || d.toldSuspended == waiting || (!waiting && !d.started) || (waiting && ifStopped && d.started) {
		d.mu.Unlock()
		return
	}
	d.toldSuspended = waiting
	d.mu.Unlock()
	if waiting {
		d.h.StateChanged(platform.AudioSuspended)
	} else {
		d.h.StateChanged(platform.AudioRunning)
	}
}

// lose ends the device because the browser did, and tells the handler once.
func (d *audioDevice) lose(reason string) {
	d.mu.Lock()
	if d.lost || d.closed {
		d.mu.Unlock()
		return
	}
	d.lost = true
	d.a.log.Warn("audio: a device is lost", "reason", reason)
	d.releaseNodes()
	d.mu.Unlock()
	d.h.StateChanged(platform.AudioLost)
}

// tryStart makes the node and starts the device when the page may make sound and the worklet is loaded.
func (d *audioDevice) tryStart() {
	a := d.a
	a.mu.Lock()
	has, ctx, module := a.hasCtx, a.ctx, a.module
	a.mu.Unlock()
	if !has || module == moduleNone || module == moduleLoading {
		return
	}
	if module == moduleFailed {
		d.lose("the audio worklet could not be loaded")
		return
	}
	if ctx.Get("state").String() != "running" {
		return
	}
	d.mu.Lock()
	if d.started || d.closed || d.lost {
		d.mu.Unlock()
		return
	}
	err := d.build(ctx)
	if err != nil {
		d.lost = true
		d.mu.Unlock()
		a.log.Error("audio: the node of a device could not be made", "reason", err.Error())
		d.h.StateChanged(platform.AudioLost)
		return
	}
	d.started = true
	d.mu.Unlock()
	a.mu.Lock()
	a.startPump()
	a.mu.Unlock()
	d.tellWaiting(false, false)
}

// build makes the ring and the worklet node and connects it. mu is held.
func (d *audioDevice) build(ctx js.Value) (err error) {
	ch := d.format.Channels
	isolated := global().Get("crossOriginIsolated").Truthy() && isObject(global().Get("SharedArrayBuffer"))
	var sab js.Value
	if isolated {
		bytes := RingHeaderWords*4 + d.capacity*ch*4
		if !try(func() { sab = global().Get("SharedArrayBuffer").New(bytes) }) {
			isolated = false
		}
	}
	if isolated {
		d.shared = true
		d.i32 = global().Get("Int32Array").New(sab, 0, RingHeaderWords)
		d.u8 = global().Get("Uint8Array").New(sab, RingHeaderWords*4, d.capacity*ch*4)
		d.store(RingChannels, int32(ch))
		d.store(RingCapacity, int32(d.capacity))
	}
	processor := AudioPlaybackProcessor
	options := map[string]any{
		"processorOptions":   map[string]any{"channels": ch, "capacity": d.capacity, "shared": nil, "block": int(int64(d.format.Rate) * int64(AudioCaptureBlock) / int64(time.Second))},
		"outputChannelCount": []any{ch},
	}
	if isolated {
		options["processorOptions"].(map[string]any)["shared"] = sab
	}
	if d.kind == platform.AudioCapture {
		processor = AudioCaptureProcessor
		options["numberOfInputs"] = 1
		options["numberOfOutputs"] = 1
		options["channelCount"] = ch
		options["channelCountMode"] = "explicit"
	} else {
		options["numberOfInputs"] = 0
		options["numberOfOutputs"] = 1
	}
	ok := try(func() {
		d.node = global().Get("AudioWorkletNode").New(ctx, processor, options)
		if !d.shared {
			d.msgFn = js.FuncOf(d.onMessage)
			d.hasMsgFn = true
			d.node.Get("port").Set("onmessage", d.msgFn)
		}
		dest := ctx.Get("destination")
		if d.kind == platform.AudioCapture {
			d.source = ctx.Call("createMediaStreamSource", d.stream)
			d.source.Call("connect", d.node)
			// A node that nothing listens to is not run by every browser: it feeds a silent gain, which feeds the output.
			d.silence = ctx.Call("createGain")
			d.silence.Get("gain").Set("value", 0)
			d.node.Call("connect", d.silence)
			d.silence.Call("connect", dest)
		} else {
			if ch > 2 && int(number(dest.Get("maxChannelCount"), 2)) >= ch {
				dest.Set("channelCount", ch)
				dest.Set("channelInterpretation", "discrete")
			}
			d.node.Call("connect", dest)
		}
	})
	if !ok {
		d.releaseNodes()
		return errors.New(platform.TagAudioDeviceOpenFailed, "the browser did not take the worklet node", "channels", ch, "processor", processor)
	}
	if foundation.DebugMode && isObject(global().Get("__glutterAudio")) {
		global().Get("__glutterAudio").Get("nodes").Call("push", d.node)
	}
	if d.paused {
		d.setFlags(RingFlagPaused)
	}
	return nil
}

// store, load and add are Atomics on the header of a shared ring. mu is held.
func (d *audioDevice) store(word int, v int32) {
	global().Get("Atomics").Call("store", d.i32, word, v)
}

func (d *audioDevice) load(word int) uint32 {
	return uint32(int32(global().Get("Atomics").Call("load", d.i32, word).Int()))
}

// setFlags sets the flags word of the ring: in the memory when it is shared, else in a message to the worklet. mu is held.
func (d *audioDevice) setFlags(flags int) {
	if d.shared {
		d.store(RingFlags, int32(flags))
		return
	}
	if isObject(d.node) {
		d.node.Get("port").Call("postMessage", map[string]any{"type": "flags", "flags": flags})
	}
}

// onMessage is what the worklet posts: the level of a playback, or a block of a capture.
func (d *audioDevice) onMessage(_ js.Value, args []js.Value) any {
	if len(args) == 0 {
		return nil
	}
	msg := args[0].Get("data")
	switch msg.Get("type").String() {
	case "level":
		d.mu.Lock()
		d.reported = uint32(int64(msg.Get("read").Float()))
		d.underruns = int(number(msg.Get("underruns"), 0))
		d.mu.Unlock()
	case "data":
		d.onBlock(msg.Get("samples"))
	}
	return nil
}

// onBlock hands a block of a capture to the handler.
func (d *audioDevice) onBlock(samples js.Value) {
	n := samples.Length()
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.started || d.paused || d.closed || d.lost || n == 0 {
		return
	}
	buf := d.grow(n)
	js.CopyBytesToGo(floatBytes(buf), global().Get("Uint8Array").New(samples.Get("buffer"), samples.Get("byteOffset"), n*4))
	d.callHandler(func() { d.h.Capture(buf) })
}

// grow is a scratch buffer of n samples. mu is held.
func (d *audioDevice) grow(n int) []float32 {
	if len(d.scratch) < n {
		d.scratch = make([]float32, n)
	}
	return d.scratch[:n]
}

// floatBytes views float32 samples as bytes (the memory of a Go program on the web is little-endian, as the typed arrays are).
func floatBytes(f []float32) []byte {
	if len(f) == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(&f[0])), len(f)*4)
}

// callHandler calls fn, which calls the handler; a panic of the handler loses the device, and does not end the pump. mu is held, and
// is let go of for the call of StateChanged.
func (d *audioDevice) callHandler(fn func()) {
	defer func() {
		if r := recover(); r != nil {
			d.a.log.Error("audio: the handler of a device panicked", "panic", r, "stack", string(debug.Stack()))
			d.lost = true
			d.releaseNodes()
			go d.h.StateChanged(platform.AudioLost)
		}
	}()
	fn()
}

// pump serves the device: it tops up a playback's ring, or takes what a capture's ring has.
func (d *audioDevice) pump() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.started || d.paused || d.closed || d.lost {
		return
	}
	if d.kind == platform.AudioPlayback {
		d.produce()
	} else if d.shared {
		d.drain()
	}
}

// produce fills the ring with what the handler gives, up to the target. mu is held.
func (d *audioDevice) produce() {
	var w, r uint32
	if d.shared {
		w, r = d.load(RingWrite), d.load(RingRead)
		d.underruns = int(d.load(RingUnderruns))
	} else {
		w, r = d.written, d.reported
	}
	need := FramesToProduce(w, r, d.capacity, d.target)
	ch := d.format.Channels
	for need > 0 && !d.lost {
		n := min(need, AudioMaxChunk)
		buf := d.grow(n * ch)
		d.callHandler(func() { d.h.Fill(buf) })
		if d.lost {
			return
		}
		d.send(buf, n, w)
		w += uint32(n)
		need -= n
	}
}

// send puts n frames in the ring at the counter w. mu is held.
func (d *audioDevice) send(buf []float32, n int, w uint32) {
	ch := d.format.Channels
	if !d.shared {
		samples := global().Get("Float32Array").New(n * ch)
		js.CopyBytesToJS(global().Get("Uint8Array").New(samples.Get("buffer")), floatBytes(buf[:n*ch]))
		d.node.Get("port").Call("postMessage", map[string]any{"type": "data", "samples": samples}, []any{samples.Get("buffer")})
		d.written = w + uint32(n)
		return
	}
	start := int(w & uint32(d.capacity-1))
	first := min(n, d.capacity-start)
	js.CopyBytesToJS(d.u8.Call("subarray", start*ch*4, (start+first)*ch*4), floatBytes(buf[:first*ch]))
	if n > first {
		js.CopyBytesToJS(d.u8.Call("subarray", 0, (n-first)*ch*4), floatBytes(buf[first*ch:n*ch]))
	}
	d.store(RingWrite, int32(w+uint32(n)))
}

// drain hands the frames of a capture's shared ring to the handler. mu is held.
func (d *audioDevice) drain() {
	ch := d.format.Channels
	w, r := d.load(RingWrite), d.load(RingRead)
	d.underruns = int(d.load(RingOverruns))
	for w != r && !d.lost {
		n := min(int(w-r), AudioMaxChunk)
		buf := d.grow(n * ch)
		start := int(r & uint32(d.capacity-1))
		first := min(n, d.capacity-start)
		js.CopyBytesToGo(floatBytes(buf[:first*ch]), d.u8.Call("subarray", start*ch*4, (start+first)*ch*4))
		if n > first {
			js.CopyBytesToGo(floatBytes(buf[first*ch:n*ch]), d.u8.Call("subarray", 0, (n-first)*ch*4))
		}
		d.callHandler(func() { d.h.Capture(buf) })
		r += uint32(n)
		d.store(RingRead, int32(r))
	}
}

// stats is the numbers of the ring, for the browser tests (debug builds).
func (d *audioDevice) stats() any {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := map[string]any{
		"kind": int(d.kind), "started": d.started, "paused": d.paused, "lost": d.lost, "closed": d.closed, "shared": d.shared,
		"capacity": d.capacity, "target": d.target, "rate": d.format.Rate, "channels": d.format.Channels, "underruns": d.underruns,
	}
	if d.shared && d.started && !d.closed {
		out["write"], out["read"] = float64(d.load(RingWrite)), float64(d.load(RingRead))
		out["underruns"] = float64(d.load(RingUnderruns))
		out["overruns"] = float64(d.load(RingOverruns))
	} else {
		out["write"], out["read"] = float64(d.written), float64(d.reported)
	}
	return out
}

// Pause stops calling the handler; the node plays silence and keeps its frames. It fails with AudioDeviceLost when the device is gone.
func (d *audioDevice) Pause() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.lost || d.closed {
		return errors.New(platform.TagAudioDeviceLost, "the audio device is gone")
	}
	d.paused = true
	if d.started {
		d.setFlags(RingFlagPaused)
	}
	return nil
}

// Resume starts calling the handler again. It fails with AudioDeviceLost when the device is gone.
func (d *audioDevice) Resume() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.lost || d.closed {
		return errors.New(platform.TagAudioDeviceLost, "the audio device is gone")
	}
	d.paused = false
	if d.started {
		d.setFlags(0)
	}
	return nil
}

// Close ends the device; when it returns the handler is not running and is never called again.
func (d *audioDevice) Close() error {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return nil
	}
	d.closed = true
	if d.started && !d.lost {
		d.setFlags(RingFlagClosed)
	}
	d.releaseNodes()
	d.mu.Unlock()
	a := d.a
	a.mu.Lock()
	delete(a.devices, d)
	a.mu.Unlock()
	a.stopPumpIfIdle()
	return nil
}

// releaseNodes disconnects the node and lets go of what the device holds in the page. mu is held.
func (d *audioDevice) releaseNodes() {
	try(func() {
		if isObject(d.source) {
			d.source.Call("disconnect")
		}
		if isObject(d.node) {
			d.node.Call("disconnect")
			d.node.Get("port").Set("onmessage", js.Null())
			d.node.Get("port").Call("close")
		}
		if isObject(d.silence) {
			d.silence.Call("disconnect")
		}
	})
	if isObject(d.stream) {
		stopTracks(d.stream)
		d.stream = js.Undefined()
	}
	if d.hasMsgFn {
		d.msgFn.Release()
		d.hasMsgFn = false
	}
	d.node, d.source, d.silence = js.Undefined(), js.Undefined(), js.Undefined()
	d.started = false
}
