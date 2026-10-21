package gluttertest

import (
	"context"
	"fmt"
	"image"
	"runtime"
	"strings"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/internal/fakeclock"
	"github.com/rm4n0s/glutter/internal/goldens"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/internal/renderhost"
	"github.com/rm4n0s/glutter/internal/renderingtest"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// TB is the part of testing.TB that gluttertest needs. *testing.T and
// *testing.B implement it; gluttertest's own tests give a fake one to see how a
// test fails.
//
// Flutter: no counterpart
type TB interface {
	Helper()
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
	Fatal(args ...any)
	Logf(format string, args ...any)
	Cleanup(func())
}

// Opts are the optional parameters of New.
//
// Flutter: flutter_test/lib/src/binding.dart TestWidgetsFlutterBinding
type Opts struct {
	// Size is the size of the view in logical pixels; the default is 800x600,
	// the size of flutter_test's view.
	Size ui.Size
	// DevicePixelRatio is the number of physical pixels per logical pixel; the
	// default is 1.
	DevicePixelRatio float64
	// TextEngine lays out the text. The default has only the fonts bundled with
	// Glutter, so that the text is the same on every machine.
	TextEngine ui.ParagraphFactory
	// Debug are the debug flags; the default is a fresh set, all off.
	Debug *widgets.DebugOptions
	// DiscardScenes drops the scenes that the view renders instead of keeping them
	// for Scenes and Screenshot: for benchmarks that run many frames.
	//
	// Flutter: no counterpart
	DiscardScenes bool
	// ErrorWidgetBuilder replaces the widget that is built for a widget that
	// panicked in Build.
	ErrorWidgetBuilder func(err error) widgets.Widget
}

// Tester is a widgets binding on a fake platform with one view, a fake clock
// and the software renderer. It stands for flutter_test's WidgetTester and the
// binding under it.
//
// Flutter: flutter_test/lib/src/widget_tester.dart WidgetTester
type Tester struct {
	tb TB
	// Binding is the widgets binding of the app under test.
	Binding *widgets.WidgetsBinding
	// Clock is the fake clock. Pump moves it; the timers of the recognizers and
	// of the image cache run when it passes them.
	Clock *fakeclock.Fake
	// View is the implicit view of the platform.
	View *ui.View

	host      *renderhost.Host
	d         *ui.PlatformDispatcher
	start     time.Time
	cfg       ui.ViewConfiguration
	platform  ui.PlatformConfiguration
	errs      []error
	handle    *semantics.SemanticsHandle
	pointerID int

	mouseID    int
	mouseAdded bool
	// scrollPointers are the pointers of the non-mouse devices that scrolled.
	scrollPointers map[ui.PointerDeviceKind]int
	mousePosition  ui.Offset
}

// New creates a tester. It fails the test, when the test ends, if the framework
// reported an error that the test did not take (TakeError).
//
// Flutter: flutter_test/lib/src/binding.dart TestWidgetsFlutterBinding.ensureInitialized
func New(tb TB, opts ...Opts) *Tester {
	tb.Helper()
	var o Opts
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.Size == (ui.Size{}) {
		o.Size = ui.Size{Width: 800, Height: 600}
	}
	if o.DevicePixelRatio == 0 {
		o.DevicePixelRatio = 1
	}
	if o.TextEngine == nil {
		o.TextEngine = renderingtest.BundledTextEngine()
	}
	t := &Tester{tb: tb, host: &renderhost.Host{DiscardScenes: o.DiscardScenes}, Clock: fakeclock.New(time.Time{})}
	t.start = t.Clock.Now()
	t.d = ui.NewPlatformDispatcher(t.host)
	t.d.SetTextEngine(o.TextEngine)
	t.cfg = ui.ViewConfiguration{
		PhysicalSize:     ui.Size{Width: o.Size.Width * o.DevicePixelRatio, Height: o.Size.Height * o.DevicePixelRatio},
		DevicePixelRatio: o.DevicePixelRatio,
	}
	t.View = t.d.AddView(ui.ImplicitViewID, t.cfg)
	ctx, cancel := context.WithCancel(context.Background())
	tb.Cleanup(cancel)
	po := widgetsPaintingOpts(t.Clock)
	t.Binding = widgets.NewWidgetsBinding(ctx, t.d, widgets.WidgetsBindingOpts{
		Debug:              o.Debug,
		ErrorReporter:      func(err error) { t.errs = append(t.errs, err) },
		ErrorWidgetBuilder: o.ErrorWidgetBuilder,
		Clock:              clockAdapter{t.Clock},
		Painting:           po,
	})
	tb.Cleanup(t.checkNoErrorsLeft)
	return t
}

// adopt makes the goroutine of the caller the UI thread of the app: the test is
// the UI thread, whichever goroutine the test framework runs it (or its
// subtests) in. The goroutines that the test starts are not, and hand work over
// with SetState.
func (t *Tester) adopt() { t.Binding.UIThread().Adopt() }

// clockAdapter makes a fake clock the clock of the gesture binding.
type clockAdapter struct{ c *fakeclock.Fake }

func (a clockAdapter) Now() time.Time { return a.c.Now() }
func (a clockAdapter) AfterFunc(d time.Duration, f func()) gestures.SamplingTimer {
	return a.c.AfterFunc(d, f)
}

// checkNoErrorsLeft fails the test for the errors that it did not take.
func (t *Tester) checkNoErrorsLeft() {
	if len(t.errs) == 0 {
		return
	}
	var b strings.Builder
	for i, err := range t.errs {
		fmt.Fprintf(&b, "\n  %d. %s", i+1, describeError(err))
	}
	t.tb.Errorf("the framework reported %d error(s) that the test did not take with TakeError:%s", len(t.errs), b.String())
}

func describeError(err error) string {
	if appErr, ok := errors.FromError(err); ok {
		if ctx, ok := appErr.Metadata["context"].(string); ok {
			return appErr.Tag + ": " + appErr.Message + " (" + ctx + ")"
		}
		return appErr.Tag + ": " + appErr.Message
	}
	return err.Error()
}

// fail stops the test with the error.
func (t *Tester) fail(err error) {
	t.tb.Helper()
	t.tb.Fatal(err)
}

// TakeError returns the oldest error that the framework reported and that the
// test did not take yet, and forgets it; nil when there is none. A test that
// expects an error takes it and checks its tag.
//
// Flutter: flutter_test/lib/src/binding.dart TestWidgetsFlutterBinding.takeException
func (t *Tester) TakeError() error {
	if len(t.errs) == 0 {
		return nil
	}
	err := t.errs[0]
	t.errs = t.errs[1:]
	return err
}

// TakeErrors returns all the errors that the framework reported and the test did
// not take yet, oldest first, and forgets them.
//
// Flutter: flutter_test/lib/src/binding.dart TestWidgetsFlutterBinding.takeException
func (t *Tester) TakeErrors() []error {
	errs := t.errs
	t.errs = nil
	return errs
}

// Errors are the errors that were reported and not taken, oldest first.
//
// Flutter: no counterpart
func (t *Tester) Errors() []error { return append([]error(nil), t.errs...) }

// Dispatcher is the platform dispatcher of the app: the test calls it as the
// platform would (for the lifecycle state, the close request, the keys).
//
// Flutter: flutter_test/lib/src/binding.dart TestWidgetsFlutterBinding.platformDispatcher
func (t *Tester) Dispatcher() *ui.PlatformDispatcher { return t.d }

// ExitRequests is how many times the app asked the platform to exit the
// application (WidgetsBinding.ExitApplication with the fake platform).
//
// Flutter: no counterpart
func (t *Tester) ExitRequests() int { return t.host.ExitRequests }

// CursorActivation is a system mouse cursor that the framework asked the platform
// to show.
//
// Flutter: no counterpart (a call of the flutter/mousecursor channel)
type CursorActivation struct {
	// Device is the pointer device the cursor is for.
	Device int
	// Kind is the kind of the system cursor, such as "click".
	Kind string
}

// ActivatedCursors are the system mouse cursors that the framework asked the
// platform to show, in order.
//
// Flutter: no counterpart (the log of a mock handler of SystemChannels.mouseCursor)
func (t *Tester) ActivatedCursors() []CursorActivation {
	out := make([]CursorActivation, len(t.host.ActivatedCursors))
	for i, a := range t.host.ActivatedCursors {
		out[i] = CursorActivation{a.Device, a.Kind}
	}
	return out
}

// WindowActions are the actions that a title bar of the app asked of the window
// (move, resize, minimize, ...), in order.
//
// Flutter: no counterpart
func (t *Tester) WindowActions() []ui.WindowAction {
	t.host.Lock()
	defer t.host.Unlock()
	return append([]ui.WindowAction(nil), t.host.WindowActions...)
}

// ClipboardText is the text on the fake system clipboard, and whether there is
// any. The framework's Clipboard calls use it.
//
// Flutter: no counterpart (the state of flutter_test's MockClipboard)
func (t *Tester) ClipboardText() (string, bool) {
	t.host.Lock()
	defer t.host.Unlock()
	return t.host.Clipboard, t.host.HasClipboard
}

// SetClipboardText puts text on the fake system clipboard, as another
// application would.
//
// Flutter: no counterpart (MockClipboard.clipboardData)
func (t *Tester) SetClipboardText(text string) {
	t.host.Lock()
	defer t.host.Unlock()
	t.host.Clipboard, t.host.HasClipboard, t.host.ClipboardPNG, t.host.ClipboardURIs = text, true, nil, nil
}

// ClearClipboard empties the fake system clipboard.
//
// Flutter: no counterpart (MockClipboard.clipboardData = null)
func (t *Tester) ClearClipboard() {
	t.host.Lock()
	defer t.host.Unlock()
	t.host.Clipboard, t.host.HasClipboard, t.host.ClipboardPNG, t.host.ClipboardURIs = "", false, nil, nil
}

// PrimaryText is the text on the fake primary selection, and whether there is
// any. Selecting text in a text field puts it there.
//
// Flutter: no counterpart
func (t *Tester) PrimaryText() (string, bool) {
	t.host.Lock()
	defer t.host.Unlock()
	return t.host.Primary.Text, t.host.Primary.HasText
}

// SetPrimaryText puts text on the fake primary selection, as another
// application would.
//
// Flutter: no counterpart
func (t *Tester) SetPrimaryText(text string) {
	t.host.Lock()
	defer t.host.Unlock()
	t.host.Primary = ui.ClipboardContent{Text: text, HasText: true}
}

// ClipboardPNG is the image on the fake system clipboard, or nil.
//
// Flutter: no counterpart
func (t *Tester) ClipboardPNG() []byte {
	t.host.Lock()
	defer t.host.Unlock()
	return t.host.ClipboardPNG
}

// SetClipboardPNG puts an image on the fake system clipboard, as another
// application would.
//
// Flutter: no counterpart
func (t *Tester) SetClipboardPNG(png []byte) {
	t.host.Lock()
	defer t.host.Unlock()
	t.host.ClipboardPNG = png
}

// ClipboardURIs are the URIs on the fake system clipboard, or nil.
//
// Flutter: no counterpart
func (t *Tester) ClipboardURIs() []string {
	t.host.Lock()
	defer t.host.Unlock()
	return t.host.ClipboardURIs
}

// SetClipboardURIs puts URIs (such as copied files) on the fake system
// clipboard, as another application would.
//
// Flutter: no counterpart
func (t *Tester) SetClipboardURIs(uris []string) {
	t.host.Lock()
	defer t.host.Unlock()
	t.host.ClipboardURIs = uris
}

// TextInputStates are the states of the text field that the framework gave to the
// input method of the fake platform, in order: the last one is what the
// input method knows now.
//
// Flutter: flutter_test/lib/src/test_text_input.dart TestTextInput
func (t *Tester) TextInputStates() []ui.TextInputState {
	t.host.Lock()
	defer t.host.Unlock()
	return append([]ui.TextInputState(nil), t.host.TextInputStates...)
}

// SendTextInputEvent plays the input method: it asks the focused text field for
// a change (a commit, a preedit, a deletion), and runs what that posted.
//
// Flutter: flutter_test/lib/src/test_text_input.dart TestTextInput.updateEditingValue
func (t *Tester) SendTextInputEvent(event ui.TextInputEvent) {
	t.tb.Helper()
	t.d.DispatchTextInput(event)
	t.FlushPosted()
}

// Elapsed is how much fake time has passed since the tester was made.
//
// Flutter: flutter_test/lib/src/binding.dart AutomatedTestWidgetsFlutterBinding.clock
func (t *Tester) Elapsed() time.Duration { return t.Clock.Since(t.start) }

// PumpWidgetOpts are the optional parameters of PumpWidget.
//
// Flutter: flutter_test/lib/src/widget_tester.dart WidgetTester.pumpWidget
type PumpWidgetOpts struct {
	// Duration is how far to advance the clock before the frame; see Pump.
	Duration time.Duration
	// NoView says not to wrap the widget in a View for the view of the tester
	// (Flutter's wrapWithView: false); the widget must be or contain one.
	NoView bool
}

// PumpWidget makes the widget the root of the app, in a View for the view of the
// tester, and pumps a frame. Calling it again replaces the widget.
//
// Flutter: flutter_test/lib/src/widget_tester.dart WidgetTester.pumpWidget
func (t *Tester) PumpWidget(w widgets.Widget, opts ...PumpWidgetOpts) {
	t.adopt()
	t.tb.Helper()
	var o PumpWidgetOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	root := w
	if !o.NoView {
		root = t.Binding.WrapWithDefaultView(w)
	}
	t.Binding.AttachRootWidget(root)
	t.Binding.Scheduler().ScheduleFrame()
	t.Pump(o.Duration)
}

// Pump advances the fake clock by the duration (if there is one), then runs one
// frame at the new time if a frame is scheduled, and the work that was posted to
// the UI thread (also between the begin and the draw phase of the frame, like the
// microtasks of flutter_test). The timers that come due while the clock advances
// run first.
//
// Flutter: flutter_test/lib/src/widget_tester.dart WidgetTester.pump
func (t *Tester) Pump(duration ...time.Duration) {
	t.adopt()
	t.tb.Helper()
	t.PumpFrame(firstDuration(duration))
}

func firstDuration(d []time.Duration) time.Duration {
	if len(d) > 0 {
		return d[0]
	}
	return 0
}

// PumpFrame is Pump for a caller that is the UI thread already (the goroutine that
// made the tester, or the last one that called Pump): it does not look for the
// calling goroutine, which costs an allocation, so that a benchmark measures the
// frame and not the harness.
//
// Flutter: no counterpart
func (t *Tester) PumpFrame(duration time.Duration) {
	if duration > 0 {
		t.Clock.Advance(duration)
	}
	// Lets the goroutines of the app (an image that decodes, a State.Go) run and post
	// their results. Where the program has one thread (js/wasm) they run only when
	// this goroutine yields; elsewhere they run alongside, and this costs nothing.
	runtime.Gosched()
	t.host.RunPosted()
	sched := t.Binding.Scheduler()
	if sched.HasScheduledFrame() {
		sched.HandleBeginFrame(foundation.Some(t.Elapsed()))
		// The microtasks that the tickers of the frame scheduled run before the
		// draw phase, as flutter_test flushes them between the two.
		t.host.RunPosted()
		sched.HandleDrawFrame()
	}
	t.host.RunPosted()
}

// PumpAndSettleOpts are the optional parameters of PumpAndSettle.
//
// Flutter: flutter_test/lib/src/widget_tester.dart WidgetTester.pumpAndSettle
type PumpAndSettleOpts struct {
	// Duration is the time between the frames; the default is 100 ms.
	Duration time.Duration
	// Timeout is how much fake time to wait for the frames to stop; the default
	// is 10 minutes.
	Timeout time.Duration
}

// PumpAndSettle pumps frames, Duration apart, until no frame is scheduled
// anymore, and returns how many times it pumped. If frames are still scheduled
// after Timeout of fake time the test fails with the tag PumpAndSettleTimeout.
//
// Flutter: flutter_test/lib/src/widget_tester.dart WidgetTester.pumpAndSettle
func (t *Tester) PumpAndSettle(opts ...PumpAndSettleOpts) int {
	t.tb.Helper()
	var o PumpAndSettleOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.Duration == 0 {
		o.Duration = 100 * time.Millisecond
	}
	if o.Timeout == 0 {
		o.Timeout = 10 * time.Minute
	}
	end := t.Clock.Now().Add(o.Timeout)
	count := 0
	for {
		if t.Clock.Now().After(end) {
			t.fail(errors.New("PumpAndSettleTimeout", "PumpAndSettle timed out: frames were still scheduled after the timeout",
				"timeout", o.Timeout.String(), "pumps", count))
			return count
		}
		t.Pump(o.Duration)
		count++
		if !t.Binding.Scheduler().HasScheduledFrame() {
			return count
		}
	}
}

// PumpFramesOpts are the optional parameters of PumpFrames.
//
// Flutter: flutter_test/lib/src/widget_tester.dart WidgetTester.pumpFrames
type PumpFramesOpts struct {
	// Interval is the time between the frames; the default is 16.683 ms, a frame
	// at 60 Hz.
	Interval time.Duration
}

// PumpFrames pumps the widget, then frames Interval apart until maxDuration of
// fake time has passed. It is for animations that are watched frame by frame.
//
// Flutter: flutter_test/lib/src/widget_tester.dart WidgetTester.pumpFrames
func (t *Tester) PumpFrames(w widgets.Widget, maxDuration time.Duration, opts ...PumpFramesOpts) {
	t.tb.Helper()
	interval := 16*time.Millisecond + 683*time.Microsecond
	if len(opts) > 0 && opts[0].Interval > 0 {
		interval = opts[0].Interval
	}
	t.PumpWidget(w)
	for elapsed := time.Duration(0); elapsed < maxDuration; elapsed += interval {
		t.Pump(interval)
	}
}

// FlushPosted runs the functions that were posted to the UI thread (by
// State.Go, for one) without running a frame.
//
// Flutter: flutter_test/lib/src/widget_tester.dart WidgetTester.idle
func (t *Tester) FlushPosted() {
	t.adopt()
	t.host.RunPosted()
}

// SetViewSize changes the size of the view, in logical pixels, and tells the
// framework as the platform would; the next Pump lays out again.
//
// Flutter: flutter_test/lib/src/window.dart TestFlutterView.physicalSize
func (t *Tester) SetViewSize(size ui.Size) {
	t.adopt()
	t.cfg.PhysicalSize = ui.Size{Width: size.Width * t.cfg.DevicePixelRatio, Height: size.Height * t.cfg.DevicePixelRatio}
	t.d.UpdateViewConfiguration(ui.ImplicitViewID, t.cfg)
}

// SetDevicePixelRatio changes the pixel ratio of the view, keeping its logical
// size.
//
// Flutter: flutter_test/lib/src/window.dart TestFlutterView.devicePixelRatio
func (t *Tester) SetDevicePixelRatio(ratio float64) {
	t.adopt()
	logical := ui.Size{Width: t.cfg.PhysicalSize.Width / t.cfg.DevicePixelRatio, Height: t.cfg.PhysicalSize.Height / t.cfg.DevicePixelRatio}
	t.cfg.DevicePixelRatio = ratio
	t.cfg.PhysicalSize = ui.Size{Width: logical.Width * ratio, Height: logical.Height * ratio}
	t.d.UpdateViewConfiguration(ui.ImplicitViewID, t.cfg)
}

// SetViewConfiguration changes the configuration of the view with the function,
// from the current one (sizes in physical pixels), and tells the framework.
//
// Flutter: flutter_test/lib/src/window.dart TestFlutterView
func (t *Tester) SetViewConfiguration(change func(c *ui.ViewConfiguration)) {
	t.adopt()
	change(&t.cfg)
	t.d.UpdateViewConfiguration(ui.ImplicitViewID, t.cfg)
}

// SetPlatformConfiguration changes the system settings (locales, text scale,
// brightness, accessibility features) with the function and tells the
// framework.
//
// Flutter: flutter_test/lib/src/window.dart TestPlatformDispatcher
func (t *Tester) SetPlatformConfiguration(change func(c *ui.PlatformConfiguration)) {
	t.adopt()
	change(&t.platform)
	t.d.UpdatePlatformConfiguration(t.platform)
}

// SetLifecycleState tells the framework that the app lifecycle state changed, as
// the platform would.
//
// Flutter: flutter_test/lib/src/binding.dart TestWidgetsFlutterBinding.handleAppLifecycleStateChanged
func (t *Tester) SetLifecycleState(state ui.AppLifecycleState) {
	t.adopt()
	t.d.SetAppLifecycleState(state)
}

// EnsureSemantics asks for semantics until the handle is disposed; pump a frame
// before looking for a semantics label. Test cleanup disposes the handle if the
// test did not.
//
// Flutter: flutter_test/lib/src/widget_tester.dart WidgetTester.ensureSemantics
func (t *Tester) EnsureSemantics() *semantics.SemanticsHandle {
	t.adopt()
	if t.handle == nil {
		t.handle = t.Binding.Renderer().SemanticsBinding().EnsureSemantics()
		t.tb.Cleanup(func() {
			if t.Binding.Renderer().SemanticsBinding().DebugOutstandingSemanticsHandles() > 0 {
				t.handle.Dispose()
			}
		})
	}
	return t.handle
}

// Scenes are the scenes that the view rendered, one for each frame that was
// composited.
//
// Flutter: no counterpart
func (t *Tester) Scenes() []*ui.Scene { return t.host.Scenes() }

// Screenshot renders the last scene with the software renderer, in physical
// pixels. It fails the test when no frame was composited yet.
//
// Flutter: flutter_test/lib/src/matchers.dart matchesGoldenFile
func (t *Tester) Screenshot() *image.RGBA {
	t.tb.Helper()
	scenes := t.Scenes()
	if len(scenes) == 0 {
		t.fail(errors.New("ScreenshotNoFrame", "no frame was composited yet; call PumpWidget or Pump first"))
		return nil
	}
	img, err := soft.Render(scenes[len(scenes)-1], int(t.cfg.PhysicalSize.Width), int(t.cfg.PhysicalSize.Height))
	if err != nil {
		t.fail(err)
		return nil
	}
	return img
}

// MatchesGolden compares the view, as the last frame drew it, with the golden
// image testdata/goldens/<name>.png of the package under test. Run the tests
// with -update-goldens to write the image, and look at every golden that
// changed (CLAUDE.md).
//
// Flutter: flutter_test/lib/src/matchers.dart matchesGoldenFile
func (t *Tester) MatchesGolden(name string) {
	t.tb.Helper()
	img := t.Screenshot()
	if img == nil {
		return
	}
	goldens.Check(t.tb, name, img)
}
