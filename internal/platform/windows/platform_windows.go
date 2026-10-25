//go:build windows

package windows

import (
	"context"
	"log/slog"
	"os"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// The messages that the platform sends to its service window (SPEC §25.3).
const (
	// msgWake is posted by Post when no wake-up is pending: the posted functions run.
	msgWake = win32.WMApp + 1
	// msgVsync is posted by the vsync thread: a frame may be drawn.
	msgVsync = win32.WMApp + 2
)

// Options are the optional parameters of New.
//
// Flutter: no counterpart
type Options struct {
	// Logger gets the platform's log; slog.Default() when nil.
	Logger *slog.Logger
	// Getenv reads the environment (GLUTTER_DEV_EVENTS); os.Getenv when nil.
	Getenv func(string) string
}

// Platform is the Win32 backend of platform.Platform: the windows, the message loop, the service
// window and the frame pacing of Windows (SPEC §25.3, §25.4). Everything but Post belongs to the UI
// thread, which is the thread that called New: the windows, the loop and the OLE apartment live
// there.
//
// Flutter: shell/platform/windows/flutter_windows_engine.cc FlutterWindowsEngine
type Platform struct {
	log    *slog.Logger
	getenv func(string) string

	instance uintptr
	classes  classes
	// service is the HWND of the service window, set once; Post reads it from other goroutines.
	service      atomic.Uintptr
	serviceToken uintptr
	cursor, icon uintptr
	start        time.Time

	// The posted functions, and whether a wake-up message is on its way.
	mu          sync.Mutex
	posted      []func()
	wakePending atomic.Bool
	running     bool

	// Everything below belongs to the UI thread.
	handler  platform.Handler
	windows  map[uintptr]*window // by HWND
	nextView ui.ViewID
	timers   timerHeap
	gate     vsyncGate
	vsync    *vsyncThread
	// quit says the loop ends: the last window was destroyed, or a panic is on its way out.
	quit     bool
	panicked *panicRecord
	// dark is whether the apps of the system are dark (read once; step 28.6 follows the setting).
	dark bool
	// oleReady says OleInitialize worked, so that Close undoes it.
	oleReady bool
	// appIDSet says the explicit AppUserModelID was set (it is the process's, set once).
	appIDSet bool
	closed   bool
	// refreshRate is the rate of the primary monitor.
	refreshRate float64
	// modalTimers counts the modal loops that run a timer (the move and size loop and the menus).
	modalTimers int

	// pointers keeps the state of the mouse, touch and pen pointers of every window (pointerstate.go).
	pointers *pointerTracker
	// keys handles the key messages of every window: they share one keyboard (keymanager.go).
	keys *keyboardManager

	// turns counts the turns of the message loop, for the tests that look for an idle app.
	turns atomic.Uint64

	// devEvents is the script of GLUTTER_DEV_EVENTS, in debug builds; devStarted says it began and
	// devDone is closed when it ended.
	devEvents  []DevEvent
	devStarted bool
	devDone    chan struct{}
}

var _ platform.Platform = (*Platform)(nil)

// panicRecord keeps a panic of the handler that was recovered at the window procedure, so that Run
// can raise it again outside of Windows' frames.
type panicRecord struct {
	value any
	stack []byte
}

// New starts the platform on the calling thread, which becomes the UI thread (SPEC §25.3): it
// makes the process per-monitor DPI aware when the manifest did not, attaches the console or the
// debugger to the output, initializes OLE, registers the window classes and makes the service
// window. Close undoes it. It fails with OleInitFailed, Win32WindowClassFailed or
// Win32WindowCreateFailed, and with LibraryLoadFailed when a system DLL is missing.
//
// Flutter: shell/platform/windows/flutter_windows_engine.cc FlutterWindowsEngine
func New(opts Options) (*Platform, error) {
	p := &Platform{
		log: opts.Logger, getenv: opts.Getenv, start: time.Now(),
		windows: map[uintptr]*window{}, devDone: make(chan struct{}),
		refreshRate: defaultRefreshRate,
	}
	if p.log == nil {
		p.log = slog.Default()
	}
	if p.getenv == nil {
		p.getenv = os.Getenv
	}
	p.pointers = newPointerTracker(p.pointerClock)
	p.keys = newKeyboardManager(winKeyHost{}, p.pointerClock, p.sendKey)
	p.setupConsole()
	p.setupDPIAwareness()
	if err := win32.OleInitialize(); err != nil {
		return nil, err
	}
	p.oleReady = true
	fail := func(err error) (*Platform, error) {
		p.teardown()
		return nil, err
	}

	instance, err := win32.GetModuleHandle("")
	if err != nil {
		return fail(err)
	}
	p.instance, p.classes.instance = instance, instance
	p.cursor = win32.LoadCursor(0, win32.IDCArrow)
	if p.icon = win32.LoadIcon(instance, 1); p.icon == 0 {
		p.icon = win32.LoadIcon(0, 32512 /* IDI_APPLICATION */)
	}
	if err := p.classes.register(windowClassName, p.icon, p.cursor); err != nil {
		return fail(err)
	}
	if err := p.classes.register(serviceClassName, 0, 0); err != nil {
		return fail(err)
	}
	// The service window is a top-level window of its own class that is never shown: a
	// message-only window would receive no broadcast (SPEC §25.3).
	p.serviceToken = registerTarget(serviceTarget{p})
	service, err := win32.CreateWindowEx(0, serviceClassName, "", win32.WSOverlapped, 0, 0, 0, 0, 0, 0, instance, p.serviceToken)
	if err != nil {
		return fail(err)
	}
	p.service.Store(service)

	v, found, err := win32.RegGetDWORD(win32.HKeyCurrentUser, personalizeKey, appsUseLightThemeValue)
	if err != nil {
		p.log.Warn("the system's app mode could not be read; the title bars are light", "err", err)
	}
	p.dark = darkTitleBar(v, found)

	p.vsync = newVsyncThread(p.log, func() {
		if h := p.service.Load(); h != 0 {
			_ = win32.PostMessage(h, msgVsync, 0, 0) // the window is gone when Close has run; there is nothing to wake
		}
	})
	p.refreshRate = p.primaryRefreshRate()
	p.vsync.setRefreshRate(p.refreshRate)

	if foundation.DebugMode {
		if script := p.getenv(DevEventsEnv); script != "" {
			events, err := ParseDevEvents(script)
			if err != nil {
				return fail(err)
			}
			p.devEvents = events
		}
	}
	return p, nil
}

// The registry key and value that say whether apps use the light theme.
const (
	personalizeKey         = `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`
	appsUseLightThemeValue = "AppsUseLightTheme"
)

// setupDPIAwareness makes the process per-monitor DPI aware (v2) when its manifest did not
// (SPEC §25.1): the manifest applies before the first window exists, which is the better way, and
// then Windows refuses the call with "access denied", which is fine. Another failure is logged;
// the app still runs, blurry on a monitor with another scale.
//
// Flutter: no counterpart
func (p *Platform) setupDPIAwareness() {
	err := win32.SetProcessDpiAwarenessContext(win32.DPIAwarenessContextPerMonitorAwareV2)
	if err == nil {
		return
	}
	if appErr, ok := errors.FromError(err); ok && appErr.Tag == win32.TagWindowsDPIAwarenessFailed {
		if code, _ := appErr.Metadata["code"].(int); code == win32.ErrorAccessDenied {
			return
		}
	}
	p.log.Warn("the process could not be made per-monitor DPI aware", "err", err)
}

// Run owns the UI thread's message loop (SPEC §25.3): each turn dispatches every message of the
// thread's queue, runs the posted functions and the timers that are due, and then waits for the
// next message or timer with MsgWaitForMultipleObjectsEx, for ever when none is armed. It returns
// nil when ctx is cancelled or the last window was destroyed. A second call while Run is active
// fails with PlatformAlreadyRunning. A panic of the handler ends the loop and is raised again here.
//
// Flutter: shell/platform/windows/task_runner_window.cc TaskRunnerWindow::PollOnce
func (p *Platform) Run(ctx context.Context, h platform.Handler) error {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return errors.New(tagPlatformAlreadyRunning, "Run was called while the platform is already running")
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
	defer context.AfterFunc(ctx, p.wake)()

	// The windows made before Run draw their first frame and show themselves.
	p.showNewWindows()
	for ctx.Err() == nil && !p.quit {
		p.turns.Add(1)
		p.turn()
		if p.quit || ctx.Err() != nil {
			break
		}
		if err := p.wait(); err != nil {
			return err
		}
	}
	if rec := p.panicked; rec != nil {
		p.panicked = nil
		p.log.Error("the handler panicked in a window procedure", "panic", rec.value, "stack", string(rec.stack))
		panic(rec.value)
	}
	return nil
}

// turn is one pass of the loop: the messages, the posted functions, the timers, and a request for
// the frames that are wanted.
func (p *Platform) turn() {
	p.pump()
	p.runPosted()
	p.timers.runDue(time.Now())
	p.scheduleVsync()
}

// pump dispatches every message in the thread's queue.
func (p *Platform) pump() {
	var m win32.Msg
	for !p.quit && win32.PeekMessage(&m, 0, 0, 0, win32.PMRemove) {
		win32.TranslateMessage(&m)
		win32.DispatchMessage(&m)
	}
}

// wait sleeps until a message arrives or the next timer is due.
func (p *Platform) wait() error {
	timeout := win32.WaitInfinite
	if d, ok := p.timers.waitFor(time.Now()); ok {
		timeout = (d + time.Millisecond - 1) / time.Millisecond * time.Millisecond // round up, or it wakes early and spins
	}
	_, err := win32.MsgWaitForMultipleObjectsEx(nil, timeout, win32.QSAllInput, win32.MWMOInputAvailable)
	return err
}

// Post runs fn on the UI thread inside Run. It is safe from any goroutine; the function goes in a
// queue under a mutex and a wake-up message is posted only when none is pending, so 10,000 posts
// never fill the message queue. The functions run in the order each goroutine posted them.
//
// Flutter: shell/platform/windows/task_runner_window.cc TaskRunnerWindow::WakeUp
func (p *Platform) Post(fn func()) {
	p.mu.Lock()
	p.posted = append(p.posted, fn)
	p.mu.Unlock()
	p.wake()
}

// wake posts the wake-up message, unless one is on its way.
func (p *Platform) wake() {
	if !p.wakePending.CompareAndSwap(false, true) {
		return
	}
	h := p.service.Load()
	if h == 0 {
		p.wakePending.Store(false)
		return
	}
	if err := win32.PostMessage(h, msgWake, 0, 0); err != nil {
		p.wakePending.Store(false)
		p.log.Error("the wake-up message could not be posted", "err", err)
	}
}

// runPosted runs the functions that Post queued. The flag is cleared first, so a Post made by one
// of them, or while they run, posts a new message and is not lost.
func (p *Platform) runPosted() {
	p.wakePending.Store(false)
	p.mu.Lock()
	fns := p.posted
	p.posted = nil
	p.mu.Unlock()
	for _, fn := range fns {
		fn()
	}
}

// guard runs the handling of a message and keeps a panic from unwinding through Windows' frames:
// it is kept, the loop ends, and Run raises it again (SPEC §25.3).
func (p *Platform) guard(fn func() uintptr) (result uintptr) {
	defer func() {
		if v := recover(); v != nil {
			if p.panicked == nil {
				p.panicked = &panicRecord{value: v, stack: debug.Stack()}
			}
			p.quit = true
			result = 0
		}
	}()
	return fn()
}

// serviceTarget is the message target of the service window.
type serviceTarget struct{ p *Platform }

func (t serviceTarget) handleMessage(hwnd uintptr, msg uint32, wparam, lparam uintptr) uintptr {
	p := t.p
	return p.guard(func() uintptr {
		switch msg {
		case msgWake:
			p.runPosted()
			return 0
		case msgVsync:
			p.onVsync()
			return 0
		case win32.WMDisplayChange:
			p.displaysChanged()
			return 0
		}
		return win32.DefWindowProc(hwnd, msg, wparam, lparam)
	})
}

// Clipboard returns a clipboard that fails every call with ClipboardUnavailable until step 28.6.
//
// Flutter: no counterpart
func (p *Platform) Clipboard() platform.Clipboard { return unavailableClipboard{} }

// Settings returns the defaults until step 28.6 reads the settings of Windows: a light theme, a
// text scale of 1 and the locale en-US.
//
// Flutter: no counterpart
func (p *Platform) Settings() platform.Settings {
	return platform.Settings{
		PlatformBrightness: ui.BrightnessLight, TextScaleFactor: 1,
		Locales: []ui.Locale{ui.NewLocale("en", "US")},
	}
}

// Services returns nil: Windows has no services until step 28.11 (the tray, the notifications)
// and 28.13 (accessibility).
//
// Flutter: no counterpart
func (p *Platform) Services() platform.Services { return nil }

// Close destroys the windows that are left and the service window, stops the vsync thread and
// undoes OleInitialize. Do not use the platform afterwards, and do not call it while Run is
// active. It must run on the UI thread.
//
// Flutter: shell/platform/windows/flutter_windows_engine.cc FlutterWindowsEngine::~FlutterWindowsEngine
func (p *Platform) Close() error {
	p.teardown()
	return nil
}

// teardown releases what New made, in the reverse order.
func (p *Platform) teardown() {
	if p.closed {
		return
	}
	p.closed = true
	for _, w := range p.sortedWindows() {
		w.Close()
	}
	if p.vsync != nil {
		p.vsync.stop()
		p.vsync = nil
	}
	if h := p.service.Swap(0); h != 0 {
		_ = win32.DestroyWindow(h) // the window of this thread; nothing reads the error
		unregisterTarget(p.serviceToken)
	}
	p.classes.unregister()
	if p.keepOnCount() > 0 {
		win32.SetThreadExecutionState(win32.ESContinuous)
	}
	if p.oleReady {
		win32.OleUninitialize()
		p.oleReady = false
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagPlatformAlreadyRunning = "PlatformAlreadyRunning"
)

// startDevEvents starts the replay of the script of GLUTTER_DEV_EVENTS, once, after the first
// frame (SPEC §25.6). Each step runs on the UI thread, from the timer heap.
//
// Flutter: no counterpart
func (p *Platform) startDevEvents() {
	if p.devStarted || len(p.devEvents) == 0 {
		return
	}
	p.devStarted = true
	p.runDevEvents(0)
}

// runDevEvents runs the steps from index i; a wait arms a timer for the rest.
func (p *Platform) runDevEvents(i int) {
	for ; i < len(p.devEvents); i++ {
		ev := p.devEvents[i]
		if ev.Kind == "wait" {
			next := i + 1
			p.timers.add(time.Now().Add(ev.Wait), func() { p.runDevEvents(next) })
			return
		}
		switch ev.Kind {
		case "key", "keydown", "keyup":
			p.runDevKey(ev)
		default:
			p.runDevMouse(ev)
		}
	}
	select {
	case <-p.devDone:
	default:
		close(p.devDone)
	}
}
