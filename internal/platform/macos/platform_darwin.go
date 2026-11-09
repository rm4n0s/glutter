//go:build darwin && !ios

package macos

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/darwin/cf"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
	"github.com/rm4n0s/glutter/ui"
)

// Options are the optional parameters of New.
//
// Flutter: no counterpart
type Options struct {
	// Logger gets the platform's log; slog.Default() when nil.
	Logger *slog.Logger
	// Getenv reads the environment (GLUTTER_DEV_EVENTS, GLUTTER_DISPLAYLINK); os.Getenv when nil.
	Getenv func(string) string
	// Accessory makes the app an accessory app: no Dock icon, no menu bar, never activated. The display
	// tests use it so that they never take the focus from the user's apps (SPEC §29.13).
	Accessory bool
}

// Platform is the AppKit backend of platform.Platform: the app and its run loop on the main thread,
// the windows, the displays and the frame pacing (SPEC §29.3, §29.4). Everything but Post belongs to the
// UI thread, which is the main thread of the process.
//
// Flutter: shell/platform/darwin/macos/framework/Source/FlutterEngine.mm FlutterEngine
type Platform struct {
	log    *slog.Logger
	getenv func(string) string
	s      *system
	start  time.Time
	// base is the media time (CACurrentMediaTime) at New: frame times are counted from it.
	base float64

	app      objc.ID
	delegate objc.ID
	handle   objcx.Handle

	// The posted functions and the source that wakes the loop for them.
	mu      sync.Mutex
	posted  []func()
	running bool
	source  *cf.Source
	timer   *cf.Timer

	// Everything below belongs to the UI thread.
	handler  platform.Handler
	windows  map[ui.ViewID]*window
	nextView ui.ViewID
	timers   timerHeap
	// quit says the loop ends: the last window closed, or a panic is on its way out.
	quit     bool
	panicked *panicRecord
	closed   bool
	clip     clipboard

	// wakes counts what woke the loop (a posted function, a timer, a tick), for the tests that look for
	// an idle app.
	wakes atomic.Uint64

	devEvents  []DevEvent
	devStarted bool
}

var _ platform.Platform = (*Platform)(nil)

// panicRecord keeps a panic of the handler that was recovered at the boundary of a callback from AppKit,
// so that Run can raise it again outside of AppKit's frames.
type panicRecord struct {
	value any
	stack []byte
}

// New starts the platform on the calling thread, which must be the main thread of the process (SPEC
// §29.3): it loads the frameworks, registers the classes that Go implements, makes the application with
// its delegate and its menu, and adds the source and the timer of the loop to the main run loop. Close
// undoes it. It fails with MacOSNotMainThread when the thread is another, and with the loader's errors
// when a framework or a class is missing.
//
// Flutter: shell/platform/darwin/macos/framework/Source/FlutterEngine.mm FlutterEngine
func New(opts Options) (*Platform, error) {
	s, err := sys()
	if err != nil {
		return nil, err
	}
	if s.pthreadMainNP() == 0 {
		return nil, errors.New(TagMacOSNotMainThread, "AppKit works only on the main thread of the process: call Run from main")
	}
	p := &Platform{log: opts.Logger, getenv: opts.Getenv, s: s, start: time.Now(), windows: map[ui.ViewID]*window{}}
	if p.log == nil {
		p.log = slog.Default()
	}
	if p.getenv == nil {
		p.getenv = os.Getenv
	}
	p.base = s.mediaTime()
	p.handle = objcx.NewHandle(p)
	objcx.WithPool(func() {
		p.app = class("NSApplication").Send(sel("sharedApplication"))
		policy := activationPolicyRegular
		if opts.Accessory {
			policy = activationPolicyAccessory
		}
		p.app.Send(sel("setActivationPolicy:"), policy)
		p.delegate = s.appDelegate.New(p.handle)
		p.app.Send(sel("setDelegate:"), p.delegate)
		if !opts.Accessory {
			p.app.Send(sel("setMainMenu:"), buildMainMenu(processName()))
		}
		class("NSWindow").Send(sel("setAllowsAutomaticWindowTabbing:"), false)
	})
	// The run loop's own pieces: one source for Post, one timer for the heap.
	p.source = cf.NewSource(func() { p.guard(p.runPosted) })
	p.source.AddTo(cf.MainRunLoop(), cf.CommonModes())
	p.timer = cf.NewTimer(func() { p.guard(p.runTimers) })
	p.timer.AddTo(cf.MainRunLoop(), cf.CommonModes())
	if foundation.DebugMode {
		if script := p.getenv(DevEventsEnv); script != "" {
			events, err := ParseDevEvents(script)
			if err != nil {
				p.teardown()
				return nil, err
			}
			p.devEvents = events
		}
	}
	return p, nil
}

// processName is the name of the executable, which the menu bar shows for the app (SPEC §29.1).
func processName() string {
	if name := filepath.Base(os.Args[0]); name != "" && name != "." {
		return name
	}
	return "Glutter"
}

// Run owns the UI thread's loop (SPEC §29.3): it runs [NSApp run] on the main thread, which dispatches
// the events, the posted functions, the timers and the frames. It returns nil when ctx is cancelled or
// the last window was closed. A second call while Run is active fails with PlatformAlreadyRunning. A
// panic of the handler ends the loop and is raised again here.
//
// Flutter: shell/platform/darwin/macos/framework/Source/FlutterEngine.mm FlutterEngine run
func (p *Platform) Run(ctx context.Context, h platform.Handler) error {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return errors.New(TagPlatformAlreadyRunning, "Run was called while the platform is already running")
	}
	p.running = true
	p.mu.Unlock()
	defer func() {
		p.mu.Lock()
		p.running = false
		p.mu.Unlock()
	}()
	p.handler = h
	defer func() { p.handler = nil }()
	p.quit = false
	defer context.AfterFunc(ctx, func() { p.Post(p.stopLoop) })()

	// The windows made before Run report their size, draw their first frame and show themselves.
	p.Post(p.showNewWindows)
	if ctx.Err() == nil {
		objcx.WithPool(func() { p.app.Send(sel("run")) })
	}
	if rec := p.panicked; rec != nil {
		p.panicked = nil
		p.log.Error("the handler panicked in a callback from AppKit", "panic", rec.value, "stack", string(rec.stack))
		panic(rec.value)
	}
	return nil
}

// stopLoop ends [NSApp run]: stop: takes effect after the next event, so an event is posted to wake the
// loop.
func (p *Platform) stopLoop() {
	p.quit = true
	p.app.Send(sel("stop:"), objc.ID(0))
	ev := class("NSEvent").Send(sel("otherEventWithType:location:modifierFlags:timestamp:windowNumber:context:subtype:data1:data2:"),
		eventTypeApplicationDefined, nsPoint{}, uint64(0), float64(0), int64(0), objc.ID(0), int16(0), int64(0), int64(0))
	p.app.Send(sel("postEvent:atStart:"), ev, true)
}

// Post runs fn on the UI thread inside Run. It is safe from any goroutine; the function goes in a queue
// under a mutex and a version-0 run loop source in the common modes is signalled, so that it runs while a
// menu is tracked or a window is resized too (SPEC §29.3). The functions run in the order each goroutine
// posted them.
//
// Flutter: shell/platform/darwin/macos/framework/Source/FlutterRunLoop.swift FlutterRunLoop performBlock
func (p *Platform) Post(fn func()) {
	p.mu.Lock()
	p.posted = append(p.posted, fn)
	p.mu.Unlock()
	p.source.Signal()
	cf.WakeUp(cf.MainRunLoop())
}

// runPosted runs the functions that Post queued, each in an autorelease pool. A Post made by one of them
// signals the source again and so is not lost.
func (p *Platform) runPosted() {
	p.wakes.Add(1)
	p.mu.Lock()
	fns := p.posted
	p.posted = nil
	p.mu.Unlock()
	for _, fn := range fns {
		objcx.WithPool(fn)
	}
}

// runTimers runs the timers of the heap that are due and sets the run loop timer to the next deadline.
func (p *Platform) runTimers() {
	p.wakes.Add(1)
	objcx.WithPool(func() { p.timers.runDue(time.Now()) })
	p.rearm()
}

// rearm sets the run loop timer to the earliest deadline of the heap, or stops it when none is armed.
func (p *Platform) rearm() {
	if at, ok := p.timers.deadline(); ok {
		p.timer.SetNextFire(at)
	} else {
		p.timer.Disarm()
	}
}

// addTimer arms fn to run on the UI thread at the time and returns the id to cancel it with.
func (p *Platform) addTimer(at time.Time, fn func()) timerID {
	id := p.timers.add(at, fn)
	p.rearm()
	return id
}

// guard runs a callback from AppKit and keeps a panic from unwinding through AppKit's frames: it is
// kept, the loop ends, and Run raises it again (SPEC §29.3).
func (p *Platform) guard(fn func()) {
	defer func() {
		if v := recover(); v != nil {
			if p.panicked == nil {
				p.panicked = &panicRecord{value: v, stack: debug.Stack()}
			}
			p.stopLoop()
		}
	}()
	fn()
}

// Wakes is how many times the loop woke because of the platform's own source, timer or frame ticks since
// New: an idle app does not make it grow (the tests of SPEC §29.13 read it).
//
// Flutter: no counterpart
func (p *Platform) Wakes() uint64 { return p.wakes.Load() }

// Clipboard returns the pasteboard, which fails with ClipboardUnavailable until step 33.5.
func (p *Platform) Clipboard() platform.Clipboard { return &p.clip }

// Settings returns the system settings: light, no accent, the locale en-US, until step 33.6 reads them.
func (p *Platform) Settings() platform.Settings {
	return platform.Settings{
		PlatformBrightness: ui.BrightnessLight,
		TextScaleFactor:    1,
		Locales:            []ui.Locale{ui.NewLocale("en", "US")},
	}
}

// Services returns nil: macOS has no desktop services until step 33.12.
func (p *Platform) Services() platform.Services { return nil }

// Close releases the loop's source and timer and closes the windows that are left. It is for the thread
// that made the platform.
//
// Flutter: no counterpart
func (p *Platform) Close() error {
	if p.closed {
		return nil
	}
	p.closed = true
	objcx.WithPool(func() {
		for _, w := range p.sortedWindows() {
			w.Close()
		}
	})
	p.teardown()
	return nil
}

func (p *Platform) teardown() {
	if p.timer != nil {
		p.timer.Close()
		p.timer = nil
	}
	if p.source != nil {
		p.source.Close()
		p.source = nil
	}
	if p.delegate != 0 {
		p.app.Send(sel("setDelegate:"), objc.ID(0))
		objcx.Release(p.delegate)
		p.delegate = 0
	}
	objcx.DropHandle(p.handle)
}

// newTestPlatformOnly is for the tests: the number of windows.
func (p *Platform) windowCount() int { return len(p.windows) }

// requestQuit is Cmd+Q, the Dock's Quit and a logout (SPEC §29.8): the windows are asked to close like
// the user's close button does, and the app ends when they all did. The framework's exit request
// replaces this in step 33.6.
func (p *Platform) requestQuit() {
	for _, w := range p.sortedWindows() {
		if p.handler != nil && !p.handler.OnCloseRequest(w.id) {
			return
		}
	}
	for _, w := range p.sortedWindows() {
		w.Close()
	}
	if len(p.windows) == 0 {
		p.stopLoop()
	}
}
