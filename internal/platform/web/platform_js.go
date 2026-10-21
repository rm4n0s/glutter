//go:build js && wasm

package web

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"syscall/js"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// Options are the optional parameters of New.
//
// Flutter: no counterpart
type Options struct {
	// Logger receives the platform's log; slog.Default() when nil.
	Logger *slog.Logger
}

// Platform is the platform.Platform of the web: it draws into one canvas in a host
// element of the page. The UI goroutine is the main goroutine; the browser's callbacks
// post to its queue (SPEC §23.3).
//
// Flutter: lib/web_ui/lib/src/engine/view_embedder/flutter_view_manager.dart FlutterViewManager
type Platform struct {
	log      *slog.Logger
	q        *queue
	cfg      Config
	host     js.Value
	fullPage bool
	// os and firefox are the operating system and the browser, found once at startup.
	os      OperatingSystem
	firefox bool
	// in is the state of the input (input_js.go).
	in inputState
	// devEvents is the script of GLUTTER_DEV_EVENTS, in debug builds; devStarted says it was
	// started.
	devEvents  []DevEvent
	devStarted bool

	// Everything below belongs to the UI goroutine.
	handler platform.Handler
	running bool
	win     *window
	// lifecycle is the state last reported, so that only changes are.
	lifecycle ui.AppLifecycleState

	// The callbacks are made once, here, and kept for the life of the app (SPEC §23.3).
	funcs     []js.Func
	rafFunc   js.Func
	resizeFn  js.Func
	dprFn     js.Func
	lifeFn    js.Func
	settingFn js.Func
	noopFn    js.Func
	shotOK    js.Func
	shotFail  js.Func
	// shotDone is waiting for the answer of the screenshot that is being posted.
	shotDone func(error)
	observer js.Value
	dprQuery js.Value
	queries  struct{ dark, motion, contrast js.Value }
}

var (
	_ platform.Platform         = (*Platform)(nil)
	_ platform.ScreenshotPoster = (*Platform)(nil)
	_ platform.ExitController   = (*Platform)(nil)
)

// New finds the host element of the page and prepares the platform: it tells the
// framework which operating system the browser runs on and makes the callbacks. It
// fails with BackendUnavailable when the program is not in a web page (it runs under
// Node, say), with WebConfigInvalid when glutterConfig has a field of the wrong type,
// and with HostElementNotFound when glutterConfig.hostElement matches nothing.
//
// Flutter: lib/web_ui/lib/src/engine/view_embedder/flutter_view_manager.dart FlutterViewManager.FlutterViewManager
func New(opts ...Options) (*Platform, error) {
	var o Options
	if len(opts) > 0 {
		o = opts[0]
	}
	log := o.Logger
	if log == nil {
		log = slog.Default()
	}
	doc := global().Get("document")
	if !isObject(doc) {
		return nil, errNoPage()
	}
	cfg, err := ParseConfig(readConfigFields())
	if err != nil {
		return nil, err
	}
	p := &Platform{log: log, q: newQueue(), cfg: cfg, fullPage: cfg.HostElement == ""}
	if p.fullPage {
		p.host = doc.Get("body")
	} else {
		p.host = queryHost(doc, cfg.HostElement)
		if !isObject(p.host) {
			return nil, errHostElementNotFound(cfg.HostElement)
		}
	}
	if foundation.DebugMode {
		if script := os.Getenv(DevEventsEnv); script != "" {
			events, err := ParseDevEvents(script)
			if err != nil {
				return nil, err
			}
			p.devEvents = events
		}
	}
	nav := global().Get("navigator")
	ua := nav.Get("userAgent").String()
	p.os = DetectOS(nav.Get("platform").String(), ua, int(number(nav.Get("maxTouchPoints"), 0)))
	p.firefox = IsFirefox(ua)
	foundation.SetBrowserTargetPlatform(p.os.TargetPlatform())

	p.rafFunc = p.keep(js.FuncOf(p.onAnimationFrame))
	p.resizeFn = p.keep(js.FuncOf(p.onResize))
	p.dprFn = p.keep(js.FuncOf(p.onDPRChange))
	p.lifeFn = p.keep(js.FuncOf(p.onLifecycleEvent))
	p.settingFn = p.keep(js.FuncOf(p.onSettingsEvent))
	p.noopFn = p.keep(js.FuncOf(func(js.Value, []js.Value) any { return nil }))
	p.shotOK = p.keep(js.FuncOf(p.onScreenshotResponse))
	p.shotFail = p.keep(js.FuncOf(p.onScreenshotFailure))
	p.makeInputFuncs()
	p.queries.dark = matchMedia("(prefers-color-scheme: dark)")
	p.queries.motion = matchMedia("(prefers-reduced-motion: reduce)")
	p.queries.contrast = matchMedia("(prefers-contrast: more)")
	return p, nil
}

// keep remembers a callback so that Close releases it, and returns it.
func (p *Platform) keep(f js.Func) js.Func {
	p.funcs = append(p.funcs, f)
	return f
}

// readConfigFields reads globalThis.glutterConfig into the plain values that
// ParseConfig takes. A field that is not a string becomes its js.Type, which ParseConfig
// does not accept.
func readConfigFields() map[string]any {
	c := global().Get("glutterConfig")
	if !isObject(c) {
		return nil
	}
	fields := map[string]any{}
	if v := c.Get("hostElement"); v.Type() != js.TypeUndefined && v.Type() != js.TypeNull {
		if v.Type() == js.TypeString {
			fields["hostElement"] = v.String()
		} else {
			fields["hostElement"] = v.Type()
		}
	}
	return fields
}

// queryHost returns the first element that matches the selector, or the undefined
// value when none does or the selector is not valid.
func queryHost(doc js.Value, selector string) js.Value {
	host := js.Undefined()
	try(func() { host = doc.Call("querySelector", selector) })
	return host
}

// Run installs the listeners of the browser, reports the lifecycle and the size of the
// view, and runs the queue of the UI goroutine until ctx ends (SPEC §23.3). The caller
// is the main goroutine. A second call while it runs fails with PlatformAlreadyRunning.
//
// Flutter: lib/web_ui/lib/src/engine/platform_dispatcher.dart EnginePlatformDispatcher
func (p *Platform) Run(ctx context.Context, h platform.Handler) error {
	if p.running {
		return errAlreadyRunning()
	}
	p.running, p.handler = true, h
	p.installListeners()
	defer func() {
		p.removeListeners()
		p.running, p.handler = false, nil
	}()
	p.lifecycle = p.currentLifecycle()
	h.OnLifecycle(p.lifecycle)
	if w := p.win; w != nil {
		h.OnMetrics(w.id, w.geo.ViewMetrics())
		if p.lifecycle == ui.AppLifecycleStateResumed {
			h.OnViewFocus(w.id, true)
		}
		if w.frameWanted {
			w.scheduleFrame()
		}
	}
	p.q.run(ctx)
	return nil
}

// Post runs fn on the UI goroutine, in order with the browser's events. It is safe to
// call from any goroutine.
//
// Flutter: no counterpart
func (p *Platform) Post(fn func()) { p.q.post(fn) }

// NewWindow makes the one window of the page: a canvas in the host element. A second
// window, a dialog, or a window with a parent fails with WindowUnsupported.
//
// Flutter: lib/web_ui/lib/src/engine/view_embedder/flutter_view_manager.dart FlutterViewManager.registerView
func (p *Platform) NewWindow(cfg platform.WindowConfig) (platform.Window, error) {
	if p.win != nil {
		return nil, errWindowUnsupported("a second window was asked for")
	}
	if (cfg.Kind != platform.WindowKindUnspecified && cfg.Kind != platform.WindowKindRegular) || cfg.HasParent {
		return nil, errWindowUnsupported("dialogs and borderless windows are not supported")
	}
	p.win = newWindow(p, ui.ImplicitViewID, cfg)
	return p.win, nil
}

// Displays lists the one display of the page, from screen (SPEC §23.2).
//
// Flutter: lib/web_ui/lib/src/engine/display.dart EngineFlutterDisplay
func (p *Platform) Displays() []platform.Display {
	screen := global().Get("screen")
	return []platform.Display{DisplayOf(number(screen.Get("width"), 0), number(screen.Get("height"), 0), p.dpr())}
}

// Clipboard returns the clipboard, which fails until step 21.3.
//
// Flutter: lib/web_ui/lib/src/engine/clipboard.dart ClipboardMessageHandler
func (p *Platform) Clipboard() platform.Clipboard {
	// TODO(step 21.3): navigator.clipboard and the paste event.
	return clipboard{post: p.Post}
}

// Settings reads the preferences of the user from the browser now.
//
// Flutter: lib/web_ui/lib/src/engine/platform_dispatcher.dart EnginePlatformDispatcher.configuration
func (p *Platform) Settings() platform.Settings {
	prefs := Prefs{
		Dark:          p.queries.dark.Get("matches").Bool(),
		ReducedMotion: p.queries.motion.Get("matches").Bool(),
		HighContrast:  p.queries.contrast.Get("matches").Bool(),
	}
	langs := global().Get("navigator").Get("languages")
	if isObject(langs) {
		for i := 0; i < langs.Length(); i++ {
			prefs.Languages = append(prefs.Languages, langs.Index(i).String())
		}
	}
	return SettingsOf(prefs)
}

// Services returns nil: the web has no desktop services of the engine yet (SPEC §23.7).
//
// Flutter: no counterpart
func (p *Platform) Services() platform.Services { return nil }

// Close takes the canvas out of the page and releases the callbacks. The platform is not
// used after it.
//
// Flutter: lib/web_ui/lib/src/engine/view_embedder/flutter_view_manager.dart FlutterViewManager.unregisterView
func (p *Platform) Close() error {
	if p.win != nil {
		p.win.Close()
	}
	p.removeListeners()
	for _, f := range p.funcs {
		f.Release()
	}
	p.funcs = nil
	return nil
}

// dpr is the device pixel ratio of the page now.
func (p *Platform) dpr() float64 { return SanitizeDPR(number(global().Get("devicePixelRatio"), 1)) }

// currentLifecycle is the lifecycle state of the page now.
func (p *Platform) currentLifecycle() ui.AppLifecycleState {
	doc := global().Get("document")
	visible := doc.Get("visibilityState").String() == "visible"
	focused := doc.Call("hasFocus").Bool()
	return LifecycleOf(visible, focused)
}

// installListeners starts the observers and the event listeners. They stay until Close or
// the end of Run.
func (p *Platform) installListeners() {
	doc := global().Get("document")
	win := global()
	doc.Call("addEventListener", "visibilitychange", p.lifeFn)
	win.Call("addEventListener", "focus", p.lifeFn)
	win.Call("addEventListener", "blur", p.lifeFn)
	for _, q := range []js.Value{p.queries.dark, p.queries.motion, p.queries.contrast} {
		q.Call("addEventListener", "change", p.settingFn)
	}
	win.Call("addEventListener", "languagechange", p.settingFn)
	p.watchDPR()
	if p.win != nil {
		p.observe(p.win.canvas)
		p.installInput()
	}
}

// removeListeners stops what installListeners started.
func (p *Platform) removeListeners() {
	doc := global().Get("document")
	win := global()
	doc.Call("removeEventListener", "visibilitychange", p.lifeFn)
	win.Call("removeEventListener", "focus", p.lifeFn)
	win.Call("removeEventListener", "blur", p.lifeFn)
	for _, q := range []js.Value{p.queries.dark, p.queries.motion, p.queries.contrast} {
		q.Call("removeEventListener", "change", p.settingFn)
	}
	win.Call("removeEventListener", "languagechange", p.settingFn)
	if isObject(p.dprQuery) {
		p.dprQuery.Call("removeEventListener", "change", p.dprFn)
		p.dprQuery = js.Undefined()
	}
	if isObject(p.observer) {
		p.observer.Call("disconnect")
		p.observer = js.Undefined()
	}
	p.removeInput()
}

// observe starts the ResizeObserver on the canvas. It asks for the exact device pixels
// (devicePixelContentBoxSize) and falls back to the CSS size where the browser does not
// know that box (SPEC §23.2).
func (p *Platform) observe(canvas js.Value) {
	p.observer = global().Get("ResizeObserver").New(p.resizeFn)
	opts := map[string]any{"box": "device-pixel-content-box"}
	if !try(func() { p.observer.Call("observe", canvas, opts) }) {
		p.observer.Call("observe", canvas)
	}
}

// watchDPR listens for the next change of the device pixel ratio: the media query for
// the current ratio stops matching when it changes (SPEC §23.2).
func (p *Platform) watchDPR() {
	if isObject(p.dprQuery) {
		p.dprQuery.Call("removeEventListener", "change", p.dprFn)
	}
	p.dprQuery = matchMedia(ResolutionQuery(p.dpr()))
	p.dprQuery.Call("addEventListener", "change", p.dprFn, map[string]any{"once": true})
}

// onAnimationFrame is the requestAnimationFrame callback: it queues the frame with the
// timestamp the browser gives.
func (p *Platform) onAnimationFrame(_ js.Value, args []js.Value) any {
	ms := number(args[0], 0)
	p.q.post(func() { p.runFrame(ms) })
	return nil
}

// runFrame runs the frame of the window on the UI goroutine.
func (p *Platform) runFrame(ms float64) {
	w := p.win
	if w == nil || !p.running {
		return
	}
	w.rafPending, w.frameWanted = false, false
	p.handler.OnFrame(w.id, FrameTime(ms))
	p.startDevEvents()
}

// onResize is the ResizeObserver callback: it copies the sizes of the last entry into Go
// values and queues the change.
func (p *Platform) onResize(_ js.Value, args []js.Value) any {
	entries := args[0]
	if n := entries.Length(); n > 0 {
		e := entries.Index(n - 1)
		css := Geometry{}
		if cb := e.Get("contentBoxSize"); isObject(cb) {
			box := cb
			if cb.Get("length").Type() == js.TypeNumber {
				box = cb.Index(0)
			}
			css.CSSWidth, css.CSSHeight = number(box.Get("inlineSize"), 0), number(box.Get("blockSize"), 0)
		} else if rect := e.Get("contentRect"); isObject(rect) {
			css.CSSWidth, css.CSSHeight = number(rect.Get("width"), 0), number(rect.Get("height"), 0)
		}
		if dp := e.Get("devicePixelContentBoxSize"); isObject(dp) && dp.Get("length").Type() == js.TypeNumber && dp.Length() > 0 {
			css.DeviceWidth, css.DeviceHeight = int(number(dp.Index(0).Get("inlineSize"), 0)), int(number(dp.Index(0).Get("blockSize"), 0))
		}
		css.DPR = p.dpr()
		p.q.post(func() { p.setGeometry(css) })
	}
	return nil
}

// onDPRChange is the callback of the resolution media query: the device pixel ratio
// changed, so the displays and the size of the view in device pixels did too.
func (p *Platform) onDPRChange(js.Value, []js.Value) any {
	dpr := p.dpr()
	p.q.post(func() {
		if p.handler != nil {
			p.handler.OnDisplaysChanged()
		}
		if w := p.win; w != nil {
			g := w.geo
			g.DPR, g.DeviceWidth, g.DeviceHeight = dpr, 0, 0 // the observer says the exact size again
			p.setGeometry(g)
		}
		p.watchDPR()
	})
	return nil
}

// setGeometry gives the view its new size, when it changed.
func (p *Platform) setGeometry(g Geometry) {
	w := p.win
	if w == nil || g == w.geo {
		return
	}
	before := w.geo.ViewMetrics()
	w.geo = g
	// The observer's first report repeats the size that was measured when the canvas was made,
	// now in exact device pixels: the engine hears about a change of the view only.
	if after := g.ViewMetrics(); p.handler != nil && after != before {
		p.handler.OnMetrics(w.id, after)
	}
}

// onLifecycleEvent is the callback of visibilitychange, focus and blur.
func (p *Platform) onLifecycleEvent(js.Value, []js.Value) any {
	state := p.currentLifecycle()
	ts := number(global().Get("performance").Call("now"), 0)
	p.q.post(func() {
		if state != p.lifecycle && p.handler != nil {
			was := p.lifecycle
			p.lifecycle = state
			p.handler.OnLifecycle(state)
			p.reportFocus(was, state, ts)
		}
	})
	return nil
}

// reportFocus tells the engine that the view got or lost the keyboard focus when the page did,
// and releases the keys that are down when it lost it: their keyups go elsewhere.
func (p *Platform) reportFocus(was, now ui.AppLifecycleState, ts float64) {
	w := p.win
	if w == nil {
		return
	}
	hadFocus, hasFocus := was == ui.AppLifecycleStateResumed, now == ui.AppLifecycleStateResumed
	switch {
	case hasFocus && !hadFocus:
		p.handler.OnViewFocus(w.id, true)
	case hadFocus && !hasFocus:
		if p.in.keys != nil {
			p.in.keys.ReleaseAll(ts)
		}
		p.handler.OnViewFocus(w.id, false)
	}
}

// onSettingsEvent is the callback of the preference media queries and languagechange.
func (p *Platform) onSettingsEvent(js.Value, []js.Value) any {
	s := p.Settings()
	p.q.post(func() {
		if p.handler != nil {
			p.handler.OnSettings(s)
		}
	})
	return nil
}

// PostScreenshot sends the PNG of the self-capture to the server that serves the page, at
// ScreenshotPath, with fetch. The answer arrives in a then callback, which posts done to
// the UI goroutine: the UI goroutine never waits for a promise (SPEC §23.3).
//
// Flutter: no counterpart
func (p *Platform) PostScreenshot(png []byte, done func(err error)) {
	p.shotDone = done
	body := global().Get("Uint8Array").New(len(png))
	js.CopyBytesToJS(body, png)
	init := map[string]any{"method": "POST", "headers": map[string]any{"Content-Type": "image/png"}, "body": body}
	global().Call("fetch", ScreenshotPath, init).Call("then", p.shotOK).Call("catch", p.shotFail)
}

// onScreenshotResponse is the then callback of the screenshot's fetch.
func (p *Platform) onScreenshotResponse(_ js.Value, args []js.Value) any {
	var err error
	if !args[0].Get("ok").Bool() {
		err = errScreenshotPostFailed("the server answered with status " + strconv.Itoa(args[0].Get("status").Int()))
	}
	p.finishScreenshot(err)
	return nil
}

// onScreenshotFailure is the catch callback of the screenshot's fetch.
func (p *Platform) onScreenshotFailure(_ js.Value, args []js.Value) any {
	p.finishScreenshot(errScreenshotPostFailed(args[0].Get("message").String()))
	return nil
}

// finishScreenshot tells the engine, on the UI goroutine, how the post ended.
func (p *Platform) finishScreenshot(err error) {
	done := p.shotDone
	p.shotDone = nil
	if done != nil {
		p.q.post(func() { done(err) })
	}
}

// CanExit is false: a page cannot close itself, so the app's request to exit does nothing
// (SPEC §23.3).
//
// Flutter: no counterpart
func (p *Platform) CanExit() bool { return false }
