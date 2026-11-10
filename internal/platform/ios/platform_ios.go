//go:build ios

package ios

import (
	"context"
	"log/slog"
	"os"
	"runtime/debug"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/darwin"
	"github.com/rm4n0s/glutter/internal/platform/darwin/cf"
	"github.com/rm4n0s/glutter/internal/platform/darwin/dl"
	"github.com/rm4n0s/glutter/internal/platform/darwin/loop"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
	"github.com/rm4n0s/glutter/internal/platform/pointerconv"
	"github.com/rm4n0s/glutter/ui"
)

// Options are the optional parameters of New.
//
// Flutter: no counterpart
type Options struct {
	// Logger gets the platform's log; slog.Default() when nil.
	Logger *slog.Logger
	// Getenv reads the environment; os.Getenv when nil.
	Getenv func(string) string
}

// system is what the platform loads once: the clock of the display link and the classes that Go implements.
type system struct {
	// mediaTime is CACurrentMediaTime: seconds on the clock that CADisplayLink timestamps use.
	mediaTime func() float64
	// playSound is AudioServicesPlaySystemSound: the vibration and the sounds of the system (haptics_ios.go).
	playSound func(uint32)
	// view is the class of the view of a window, scene the class of the delegate of the scene, textView, textPos and textRange the
	// classes of the text input (textinput_ios.go), and keyboard the class of the observer of the keyboard's frame.
	view, scene, textView, textPos, textRange, keyboard, viewController *objcx.RegisteredClass
}

// Platform is the UIKit backend of platform.Platform: the app and its run loop on the main thread, the scene with its windows, the
// displays and the frame pacing (SPEC §30.3, §30.4). Everything but Post belongs to the UI thread, which is the main thread of
// the process. A process has one platform: UIApplicationMain runs once, and the classes that Go implements are named by the
// process.
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterEngine.mm FlutterEngine
type Platform struct {
	log    *slog.Logger
	getenv func(string) string
	s      *system
	start  time.Time
	// base is the media time (CACurrentMediaTime) at New: frame times are counted from it.
	base float64

	// The posted functions and the source that wakes the loop for them.
	mu     sync.Mutex
	posted []func()
	source *cf.Source
	timer  *cf.Timer

	// Everything below belongs to the UI thread.
	handler  platform.Handler
	windows  map[ui.ViewID]*window
	nextView ui.ViewID
	timers   loop.Timers
	// scenes are the window scenes that are connected (retained) and what each one is doing.
	scenes map[objc.ID]activation
	// scene is the first scene that connected: the windows of the app are made in it.
	scene objc.ID
	lc    lifecycleTracker
	// ended says the app is ending: the lifecycle is Detached.
	ended  bool
	closed bool

	deskState

	// checker is the UITextChecker of the spell check, 0 until the first check (spellcheck_ios.go).
	checker objc.ID

	// handle is the platform's handle for the objects of UIKit that find it (the keyboard observer), and observer that object.
	handle   objcx.Handle
	observer objc.ID

	// touch, conv and keys turn the touches, the pointer and the keys into the events of the framework (input_ios.go).
	touch *touchTracker
	conv  *pointerconv.Converter
	keys  *keyResponder

	// devEvents is the script of GLUTTER_DEV_EVENTS; devStarted says it began and devDone that its last step ran.
	devEvents  []DevEvent
	devStarted bool
	devDone    bool
	// devBegan is when the script began.
	devBegan time.Time

	// wakes counts what woke the loop (a posted function, a timer, a tick), for the tests that look for an idle app.
	wakes atomic.Uint64
	// framesDrawn counts the frames that the handler was asked for since New.
	framesDrawn uint64
}

var (
	_ platform.Platform     = (*Platform)(nil)
	_ platform.ScriptWaiter = (*Platform)(nil)
)

// New starts the platform: it loads the frameworks, registers the classes that Go implements (the view and the scene delegate) and adds
// the source and the timer of the loop to the main run loop. It does not need the main thread yet (Run does) and nothing is on the
// screen until Run has started UIApplicationMain and the scene has connected. Close undoes it.
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterEngine.mm FlutterEngine
func New(opts Options) (*Platform, error) {
	p := &Platform{log: opts.Logger, getenv: opts.Getenv, start: time.Now(), windows: map[ui.ViewID]*window{}, scenes: map[objc.ID]activation{}}
	if p.log == nil {
		p.log = slog.Default()
	}
	if p.getenv == nil {
		p.getenv = os.Getenv
	}
	s, err := p.load()
	if err != nil {
		return nil, err
	}
	p.s = s
	p.base = s.mediaTime()
	p.handle = objcx.NewHandle(p)
	p.observeKeyboard()
	p.touch, p.conv = newTouchTracker(), pointerconv.New()
	p.keys = newKeyResponder(p.sendKey)
	if foundation.DebugMode {
		if script := p.getenv(DevEventsEnv); script != "" {
			if p.devEvents, err = ParseDevEvents(script); err != nil {
				return nil, err
			}
		}
	}
	// The run loop's own pieces: one source for Post, one timer for the heap.
	p.source = cf.NewSource(func() { p.guard(p.runPosted) })
	p.source.AddTo(cf.MainRunLoop(), cf.CommonModes())
	p.timer = cf.NewTimer(func() { p.guard(p.runTimers) })
	p.timer.AddTo(cf.MainRunLoop(), cf.CommonModes())
	return p, nil
}

// load binds the clock and registers the classes.
func (p *Platform) load() (*system, error) {
	qc, err := dl.OpenFramework("QuartzCore")
	if err != nil {
		return nil, err
	}
	s := &system{}
	if err := qc.Bind(&s.mediaTime, "CACurrentMediaTime"); err != nil {
		return nil, err
	}
	if s.view, err = registerView(); err != nil {
		return nil, err
	}
	if s.scene, err = registerPlatformSceneDelegate(p); err != nil {
		return nil, err
	}
	if s.textView, s.textPos, s.textRange, err = registerText(); err != nil {
		return nil, err
	}
	if s.keyboard, err = registerKeyboardObserver(); err != nil {
		return nil, err
	}
	if s.viewController, err = registerViewController(); err != nil {
		return nil, err
	}
	if err := loadAudioToolbox(s); err != nil {
		return nil, err
	}
	return s, nil
}

// Run owns the UI thread's loop: it starts UIApplicationMain on the main thread, whose run loop dispatches the events, the posted
// functions, the timers and the frames. UIApplicationMain does not return, so Run does not either while the app runs; when ctx is
// cancelled the lifecycle is Detached and the process exits with status 0 (SPEC §30.3). Run fails with IOSNotMainThread when the
// thread is not the main thread of the process. It is called once.
//
// Deviation: Run does not return when ctx is cancelled: UIKit has no way out of UIApplicationMain, so the process ends.
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterEngine.mm FlutterEngine run
func (p *Platform) Run(ctx context.Context, h platform.Handler) error {
	if ctx.Err() != nil {
		return nil
	}
	p.handler = h
	defer context.AfterFunc(ctx, func() { p.Post(p.finish) })()
	return launch(nil, func() (*objcx.RegisteredClass, error) { return p.s.scene, nil })
}

// finish ends the app: the lifecycle says Detached, and the process exits.
func (p *Platform) finish() {
	p.ended = true
	p.updateLifecycle()
	os.Exit(0)
}

// Post runs fn on the UI thread inside Run. It is safe from any goroutine; the function goes in a queue under a mutex and a
// version-0 run loop source in the common modes is signalled, so that it runs while a scroll tracks too. The functions run in the
// order each goroutine posted them.
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterEngine.mm FlutterEngine
func (p *Platform) Post(fn func()) {
	p.mu.Lock()
	p.posted = append(p.posted, fn)
	p.mu.Unlock()
	p.source.Signal()
	cf.WakeUp(cf.MainRunLoop())
}

// runPosted runs the functions that Post queued, each in an autorelease pool. A Post made by one of them signals the source again
// and so is not lost.
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
	objcx.WithPool(func() { p.timers.RunDue(time.Now()) })
	p.rearm()
}

// rearm sets the run loop timer to the earliest deadline of the heap, or stops it when none is armed.
func (p *Platform) rearm() {
	if at, ok := p.timers.Deadline(); ok {
		p.timer.SetNextFire(at)
	} else {
		p.timer.Disarm()
	}
}

// addTimer arms fn to run on the UI thread at the time and returns the id to cancel it with.
func (p *Platform) addTimer(at time.Time, fn func()) loop.TimerID {
	id := p.timers.Add(at, fn)
	p.rearm()
	return id
}

// guard runs a callback from UIKit and keeps a panic from unwinding through UIKit's frames: it is logged with its stack, and raised
// again on a goroutine of its own, which ends the process the way an uncaught panic does.
func (p *Platform) guard(fn func()) {
	defer func() {
		if v := recover(); v != nil {
			p.log.Error("the handler panicked in a callback from UIKit", "panic", v, "stack", string(debug.Stack()))
			go func() { panic(v) }()
		}
	}()
	fn()
}

// Wakes is how many times the loop woke because of the platform's own source, timer or frame ticks since New: an idle app does not
// make it grow.
//
// Flutter: no counterpart
func (p *Platform) Wakes() uint64 { return p.wakes.Load() }

// FramesDrawn is how many frames the handler was asked for since New.
//
// Flutter: no counterpart
func (p *Platform) FramesDrawn() uint64 { return p.framesDrawn }

// Clipboard returns the pasteboard of the app: UIPasteboard's general one, with the reads on goroutines of their own (the paste prompt)
// and the answers on the UI thread (SPEC §30.6).
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterPlatformPlugin.mm
func (p *Platform) Clipboard() platform.Clipboard {
	return pasteboard{post: p.Post, back: uiPasteboard{}}
}

// Settings is what the platform knows of the user's settings until step 35.7: a light theme, the default text scale and English.
func (p *Platform) Settings() platform.Settings {
	return platform.Settings{PlatformBrightness: ui.BrightnessLight, TextScaleFactor: 1, Locales: []ui.Locale{{LanguageCode: "en", CountryCode: "US"}}}
}

// Services returns nil: iOS has no services until step 35.6.
func (p *Platform) Services() platform.Services { return nil }

// Displays are the screens of the device: the main screen, in points.
func (p *Platform) Displays() []platform.Display {
	var out []platform.Display
	objcx.WithPool(func() {
		screen := class("UIScreen").Send(sel("mainScreen"))
		bounds := send[darwin.CGRect](screen, "bounds")
		rate := float64(send[int64](screen, "maximumFramesPerSecond"))
		out = []platform.Display{{
			ID: 0, Name: "main", Primary: true,
			Bounds:           ui.NewRectFromLTWH(bounds.Origin.X, bounds.Origin.Y, bounds.Size.Width, bounds.Size.Height),
			DevicePixelRatio: send[float64](screen, "scale"), RefreshRate: rate,
		}}
	})
	return out
}

// Close closes the windows that are left and releases the loop's source and timer. It is for the thread that made the platform.
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
		for scene := range p.scenes {
			objcx.Release(scene)
		}
		p.scenes = map[objc.ID]activation{}
	})
	p.stopObservingKeyboard()
	if p.desk != nil {
		p.desk.closeServices()
	}
	objcx.Release(p.checker)
	p.checker = 0
	objcx.DropHandle(p.handle)
	if p.timer != nil {
		p.timer.Close()
		p.timer = nil
	}
	if p.source != nil {
		p.source.Close()
		p.source = nil
	}
	return nil
}

// sortedWindows are the windows in the order they were made.
func (p *Platform) sortedWindows() []*window {
	out := make([]*window, 0, len(p.windows))
	for _, w := range p.windows {
		out = append(out, w)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].id < out[j].id })
	return out
}

// frameTime turns a time of the media clock into the duration since the platform started.
func (p *Platform) frameTime(media float64) time.Duration {
	return time.Duration((media - p.base) * float64(time.Second))
}

// updateLifecycle reports the lifecycle state when it changed.
func (p *Platform) updateLifecycle() {
	scenes := make([]activation, 0, len(p.scenes))
	for _, a := range p.scenes {
		scenes = append(scenes, a)
	}
	state, changed := p.lc.update(scenes, p.ended)
	if changed && p.handler != nil {
		p.handler.OnLifecycle(state)
	}
}

// foreground says a scene is on the screen, active or not: the windows draw then.
func (p *Platform) foreground() bool {
	for _, a := range p.scenes {
		if a == activationForegroundActive || a == activationForegroundInactive {
			return true
		}
	}
	return false
}

func class(name string) objc.ID { return objc.ID(objcx.MustClass(name)) }

func sel(name string) objc.SEL { return objcx.Sel(name) }

func send[T any](id objc.ID, name string, args ...any) T { return objc.Send[T](id, sel(name), args...) }
