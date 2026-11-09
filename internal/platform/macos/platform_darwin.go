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
	// QuitOnLastWindowClosed says that Run returns when the last window is closed; None means true. When it
	// is false the app stays running in the Dock with no window (SPEC §29.4).
	QuitOnLastWindowClosed foundation.Opt[bool]
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
	// kb is the keyboard: the responder of the key events and the layout (keyboard_darwin.go).
	kb *keyboard
	// menus is the menu bar (menu_darwin.go).
	menus *menuBar
	// accessory says the app is an accessory app (Options.Accessory).
	accessory bool
	// quitOnLastClose is Options.QuitOnLastWindowClosed with its default.
	quitOnLastClose bool
	// deskState has the desktop services (desktop_darwin.go).
	deskState
	// performDrag moves a window by the press that is down (dragWindow); the tests replace it, since a scripted press
	// is not a button that is really down.
	performDrag func(w *window, ev objc.ID)
	// testEvent, when set by a test, is the current event: the events that the tests send to a view are not
	// dispatched by NSApp, which has none then.
	testEvent func() objc.ID
	// drag is the drag that the app started and that has not ended; dragMask is what its source allows (dnd_darwin.go).
	drag     *dragSession
	dragMask uint64
	// beginDrag begins the dragging session; the tests replace it, since a session of the system follows the real
	// mouse.
	beginDrag func(w *window, items, ev, source objc.ID)
	// testDragLocation, when set by a test, is the place of a drag: a fake NSDraggingInfo cannot return the NSPoint.
	testDragLocation func(info objc.ID) nsPoint
	// testDoubleClick, when set by a test, is the setting of the double click on a title bar.
	testDoubleClick func() string
	// modals are the modal windows that are open and the windows they block (modal.go).
	modals modalStack
	// watch reads the system settings and watches them (settings_darwin.go).
	watch *settingsWatch
	// lc remembers the lifecycle state that was reported; appHidden says the app is hidden (Command+H);
	// ending says the loop has ended (lifecycle_darwin.go).
	lc        lifecycleTracker
	appHidden bool
	ending    bool

	// ptr is the state machine of the mouse, the trackpad and the pen of every window (mouse.go).
	ptr *pointerMachine
	// ctrlClick says that the primary button that is down was a Control-click, made the secondary button.
	ctrlClick bool
	// The pen of a tablet that is in range: its device id, whether it is the eraser end, and its buttons.
	penDevice     int
	penEraser     bool
	stylusButtons int
	// cursors are the NSCursors that were asked for, kept for the life of the platform; blankCursor is the
	// transparent one of MouseCursorNone.
	cursors     map[string]objc.ID
	blankCursor objc.ID

	// wakes counts what woke the loop (a posted function, a timer, a tick), for the tests that look for
	// an idle app.
	wakes atomic.Uint64

	devEvents  []DevEvent
	devStarted bool
	// devDone says the script ran its last step.
	devDone bool
	// framesDrawn counts the frames that the handler was asked for since New (the scripts wait for one).
	framesDrawn uint64
}

var (
	_ platform.Platform     = (*Platform)(nil)
	_ platform.ScriptWaiter = (*Platform)(nil)
)

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
	p := &Platform{log: opts.Logger, getenv: opts.Getenv, s: s, start: time.Now(), windows: map[ui.ViewID]*window{}, ptr: newPointerMachine(), penDevice: stylusDeviceBase}
	if p.log == nil {
		p.log = slog.Default()
	}
	if p.getenv == nil {
		p.getenv = os.Getenv
	}
	p.accessory = opts.Accessory
	p.quitOnLastClose = opts.QuitOnLastWindowClosed.OrElse(true)
	p.performDrag = dragWindow
	p.beginDrag = beginDragSession
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
		class("NSWindow").Send(sel("setAllowsAutomaticWindowTabbing:"), false)
	})
	p.menus = newMenuBar(p)
	if !opts.Accessory {
		p.menus.installDefault()
	}
	p.kb = newKeyboard(p)
	if foundation.DebugMode {
		p.applyDevAppearance()
	}
	p.watch = newSettingsWatch(p)
	p.clip.p = p
	if foundation.DebugMode {
		if name := p.getenv(DevPasteboardEnv); name != "" {
			ns := objcx.NewString(name)
			p.clip.pb = objcx.Retain(class("NSPasteboard").Send(sel("pasteboardWithName:"), ns))
			p.clip.named = true
			objcx.Release(ns)
		}
	}
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
	p.quit, p.ending = false, false
	defer context.AfterFunc(ctx, func() { p.Post(p.stopLoop) })()

	// The windows made before Run report their size, draw their first frame and show themselves.
	p.Post(p.showNewWindows)
	if ctx.Err() == nil {
		objcx.WithPool(func() { p.app.Send(sel("run")) })
	}
	p.ending = true
	p.updateLifecycle()
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
	if p.desk != nil {
		p.desk.closeServices()
	}
	if p.clip.pb != 0 {
		if p.clip.named {
			p.clip.pb.Send(sel("releaseGlobally"))
		}
		objcx.Release(p.clip.pb)
		p.clip.pb = 0
	}
	if p.watch != nil {
		p.watch.close()
		p.watch = nil
	}
	if p.menus != nil {
		p.menus.close()
		p.menus = nil
	}
	if p.kb != nil {
		p.kb.close()
		p.kb = nil
	}
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

// requestQuit is Cmd+Q, the Dock's Quit and a logout (SPEC §29.8): the windows are asked to close in the
// order they were made, as the user's close button asks, and each one that agrees is closed at once, as a
// document app closes its documents when it quits. A window that refuses (the framework's exit request is
// cancelled) stops the quit and stays, with the windows that did not come yet; the app ends when the last
// window is closed.
func (p *Platform) requestQuit() {
	for _, w := range p.sortedWindows() {
		if p.handler != nil && !p.handler.OnCloseRequest(w.id) {
			return
		}
		w.Close()
	}
	if len(p.windows) == 0 {
		p.stopLoop()
	}
}
