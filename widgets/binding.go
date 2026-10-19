package widgets

import (
	"context"
	"log/slog"
	"slices"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// WidgetsBindingObserver is an object that wants to know about changes of the
// system that the binding hears of: the metrics of the views, the text scale
// factor, the brightness, the locales, the accessibility features, the app
// lifecycle, the memory pressure, the pop of a route and the request to exit.
//
// Embed WidgetsBindingObserverBase to get a no-op for each method and override
// the ones you need, then add the observer with WidgetsBinding.AddObserver.
//
// Deviation: DidRequestAppExit returns its answer where Flutter's returns a
// Future (SPEC §12.10).
//
// Flutter: widgets/binding.dart WidgetsBindingObserver
type WidgetsBindingObserver interface {
	// DidPopRoute is called when the system asks to pop the current route
	// (the back button). Return true to handle it.
	//
	// Flutter: widgets/binding.dart WidgetsBindingObserver.didPopRoute
	DidPopRoute() bool
	// DidPushRouteInformation is called when the system pushes a new route (a
	// link that opens the app). Return true to handle it.
	//
	// Deviation: Flutter's default calls the deprecated didPushRoute with the URI
	// string; that method is not ported (SPEC §12.10).
	//
	// Flutter: widgets/binding.dart WidgetsBindingObserver.didPushRouteInformation
	DidPushRouteInformation(routeInformation *RouteInformation) bool
	// DidChangeMetrics is called when the metrics of a view change.
	//
	// Flutter: widgets/binding.dart WidgetsBindingObserver.didChangeMetrics
	DidChangeMetrics()
	// DidChangeTextScaleFactor is called when the system text scale factor
	// changes.
	//
	// Flutter: widgets/binding.dart WidgetsBindingObserver.didChangeTextScaleFactor
	DidChangeTextScaleFactor()
	// DidChangePlatformBrightness is called when the system brightness changes.
	//
	// Flutter: widgets/binding.dart WidgetsBindingObserver.didChangePlatformBrightness
	DidChangePlatformBrightness()
	// DidChangeLocales is called when the system locales change.
	//
	// Flutter: widgets/binding.dart WidgetsBindingObserver.didChangeLocales
	DidChangeLocales(locales []ui.Locale)
	// DidChangeAppLifecycleState is called when the app lifecycle state changes.
	//
	// Flutter: widgets/binding.dart WidgetsBindingObserver.didChangeAppLifecycleState
	DidChangeAppLifecycleState(state ui.AppLifecycleState)
	// DidRequestAppExit is called when the app is asked to exit; the answer
	// Cancel keeps the app running (when the exit is cancelable).
	//
	// Flutter: widgets/binding.dart WidgetsBindingObserver.didRequestAppExit
	DidRequestAppExit() ui.AppExitResponse
	// DidHaveMemoryPressure is called when the system is low on memory.
	//
	// Flutter: widgets/binding.dart WidgetsBindingObserver.didHaveMemoryPressure
	DidHaveMemoryPressure()
	// DidChangeAccessibilityFeatures is called when the accessibility features
	// change.
	//
	// Flutter: widgets/binding.dart WidgetsBindingObserver.didChangeAccessibilityFeatures
	DidChangeAccessibilityFeatures()
}

// WidgetsBindingObserverBase implements WidgetsBindingObserver with methods that
// do nothing (DidRequestAppExit lets the app exit). Embed it in an observer.
//
// Flutter: widgets/binding.dart WidgetsBindingObserver
type WidgetsBindingObserverBase struct{}

// DidPopRoute does not handle the pop.
//
// Flutter: widgets/binding.dart WidgetsBindingObserver.didPopRoute
func (WidgetsBindingObserverBase) DidPopRoute() bool { return false }

// DidPushRouteInformation does not handle the route.
//
// Flutter: widgets/binding.dart WidgetsBindingObserver.didPushRouteInformation
func (WidgetsBindingObserverBase) DidPushRouteInformation(*RouteInformation) bool { return false }

// DidChangeMetrics does nothing.
//
// Flutter: widgets/binding.dart WidgetsBindingObserver.didChangeMetrics
func (WidgetsBindingObserverBase) DidChangeMetrics() {}

// DidChangeTextScaleFactor does nothing.
//
// Flutter: widgets/binding.dart WidgetsBindingObserver.didChangeTextScaleFactor
func (WidgetsBindingObserverBase) DidChangeTextScaleFactor() {}

// DidChangePlatformBrightness does nothing.
//
// Flutter: widgets/binding.dart WidgetsBindingObserver.didChangePlatformBrightness
func (WidgetsBindingObserverBase) DidChangePlatformBrightness() {}

// DidChangeLocales does nothing.
//
// Flutter: widgets/binding.dart WidgetsBindingObserver.didChangeLocales
func (WidgetsBindingObserverBase) DidChangeLocales([]ui.Locale) {}

// DidChangeAppLifecycleState does nothing.
//
// Flutter: widgets/binding.dart WidgetsBindingObserver.didChangeAppLifecycleState
func (WidgetsBindingObserverBase) DidChangeAppLifecycleState(ui.AppLifecycleState) {}

// DidRequestAppExit lets the app exit.
//
// Flutter: widgets/binding.dart WidgetsBindingObserver.didRequestAppExit
func (WidgetsBindingObserverBase) DidRequestAppExit() ui.AppExitResponse {
	return ui.AppExitResponseExit
}

// DidHaveMemoryPressure does nothing.
//
// Flutter: widgets/binding.dart WidgetsBindingObserver.didHaveMemoryPressure
func (WidgetsBindingObserverBase) DidHaveMemoryPressure() {}

// DidChangeAccessibilityFeatures does nothing.
//
// Flutter: widgets/binding.dart WidgetsBindingObserver.didChangeAccessibilityFeatures
func (WidgetsBindingObserverBase) DidChangeAccessibilityFeatures() {}

// WidgetsBinding is the glue between the widgets layer and the engine. It owns
// the bindings of the layers below (scheduler, gestures, painting, rendering),
// the build owner and the root element, builds the widgets before each frame
// lays out and paints, and tells the WidgetsBindingObservers what the platform
// changed. An app has one; RunApp makes it, and widgets reach it with
// WidgetsBindingOf.
//
// Deviation: Flutter's is a mixin on BindingBase with a static instance; this
// one is a struct made from the other bindings and reached through the
// BuildContext (SPEC §12.10, §13).
//
// Flutter: widgets/binding.dart WidgetsBinding
type WidgetsBinding struct {
	d            *ui.PlatformDispatcher
	scheduler    *scheduler.SchedulerBinding
	gesture      *gestures.GestureBinding
	painting     *painting.PaintingBinding
	renderer     *rendering.RendererBinding
	keyboard     *services.HardwareKeyboard
	navigator    *services.SystemNavigator
	sound        *services.SystemSound
	clipboard    *services.Clipboard
	textInput    *services.TextInput
	focusManager *FocusManager
	owner        *BuildOwner
	debug        *DebugOptions
	report       foundation.ErrorReporter
	thread       *foundation.UIThread

	observers   []WidgetsBindingObserver
	switcher    ApplicationSwitcherDescription
	rootElement *RootElement

	readyToProduceFrames       bool
	needToReportFirstFrame     bool
	firstFrameDone             chan struct{}
	debugBuildingDirtyElements bool
}

// ApplicationSwitcherDescription is how the app is described to the system's
// task switcher and window title: a label and the primary color.
//
// Flutter: services/system_chrome.dart ApplicationSwitcherDescription
type ApplicationSwitcherDescription struct {
	// Label is the title of the app.
	Label string
	// PrimaryColor is the color of the app's task switcher entry.
	PrimaryColor ui.Color
}

// ApplicationSwitcherDescription is the description that the last Title widget
// set.
//
// Deviation: Flutter's SystemChrome sends it to the embedder through a platform
// channel; here the binding keeps it, and the window title is wired to it in
// step 10 (SPEC §12.10).
//
// Flutter: services/system_chrome.dart SystemChrome.setApplicationSwitcherDescription
func (b *WidgetsBinding) ApplicationSwitcherDescription() ApplicationSwitcherDescription {
	return b.switcher
}

// SetApplicationSwitcherDescription sets the description of the app.
//
// Flutter: services/system_chrome.dart SystemChrome.setApplicationSwitcherDescription
func (b *WidgetsBinding) SetApplicationSwitcherDescription(d ApplicationSwitcherDescription) {
	b.switcher = d
}

// WidgetsBindingOpts are the optional parameters of NewWidgetsBinding.
//
// Flutter: widgets/binding.dart WidgetsBinding.initInstances
type WidgetsBindingOpts struct {
	// Debug are the debug flags; the default is a fresh set, all off.
	Debug *DebugOptions
	// ErrorReporter receives the errors that the framework recovers from or
	// reports instead of throwing (SPEC §14.2). The default logs the error, as
	// JSON, through Logger.
	ErrorReporter foundation.ErrorReporter
	// ErrorWidgetBuilder builds the widget that replaces a widget that panicked
	// in Build; the default is described in BuildOwner.ErrorWidgetBuilder.
	ErrorWidgetBuilder func(err error) Widget
	// Logger gets the log of the default error reporter; slog.Default() when nil.
	Logger *slog.Logger
	// Clock is the clock of the gesture binding (the recognizers' timers, the
	// velocity trackers and the resampler); the default is the system clock.
	// Tests give a fake one.
	Clock gestures.SamplingClock
	// Painting are the options of the painting binding (image cache, asset
	// bundle, HTTP client). Its Post, AfterFunc and Reporter are set by default
	// from the dispatcher and the error reporter.
	Painting painting.PaintingBindingOpts
	// UIThread is the UI thread; the default is the calling goroutine.
	UIThread *foundation.UIThread
}

// NewWidgetsBinding creates the binding for a dispatcher, with the scheduler,
// gesture, painting and rendering bindings under it. Call it on the UI thread.
// ctx bounds the work of the painting binding (image loading).
//
// Frames stay off until the root widget is attached (AttachRootWidget).
//
// Flutter: widgets/binding.dart WidgetsBinding.initInstances
func NewWidgetsBinding(ctx context.Context, d *ui.PlatformDispatcher, opts ...WidgetsBindingOpts) *WidgetsBinding {
	o := firstOpts(opts)
	if o.Debug == nil {
		o.Debug = NewDebugOptions()
	}
	if o.Debug.Rendering == nil {
		o.Debug.Rendering = rendering.NewDebugOptions()
	}
	if o.UIThread == nil {
		o.UIThread = foundation.NewUIThread()
	}
	if o.ErrorReporter == nil {
		o.ErrorReporter = DefaultErrorReporter(o.Logger)
	}
	o.Debug.Rendering.UIThread = o.UIThread
	b := &WidgetsBinding{d: d, debug: o.Debug, report: o.ErrorReporter, thread: o.UIThread,
		needToReportFirstFrame: true, firstFrameDone: make(chan struct{})}

	b.scheduler = scheduler.NewSchedulerBinding(d)
	b.scheduler.ErrorReporter = o.ErrorReporter
	b.scheduler.FramesGate = func() bool { return b.readyToProduceFrames }
	b.gesture = gestures.NewGestureBinding(d, b.scheduler, gestures.GestureBindingOpts{ErrorReporter: o.ErrorReporter, Clock: o.Clock})
	po := o.Painting
	if po.Reporter == nil {
		po.Reporter = o.ErrorReporter
	}
	b.painting = painting.NewPaintingBinding(ctx, b.scheduler, po)
	b.renderer = rendering.NewRendererBinding(d, b.scheduler, b.gesture, rendering.RendererBindingOpts{
		Debug: o.Debug.Rendering, ReportError: o.ErrorReporter,
	})
	b.keyboard = services.NewHardwareKeyboard(services.HardwareKeyboardOpts{
		ErrorReporter:            o.ErrorReporter,
		DebugPrintKeyboardEvents: func() bool { return b.debug.PrintKeyboardEvents },
		DebugPrint:               b.debug.DebugPrint,
	})
	d.OnKeyData = b.keyboard.HandleKeyData
	b.navigator = &services.SystemNavigator{OnPop: func(foundation.Opt[bool]) { d.RequestAppExit() }}
	b.sound = &services.SystemSound{}
	b.clipboard = services.NewClipboard(d)
	b.textInput = services.NewTextInput(services.TextInputOpts{Backend: d, Post: d.Post})
	d.OnTextInput = b.textInput.HandleTextInput
	b.owner = NewBuildOwner(BuildOwnerOpts{
		OnBuildScheduled: b.handleBuildScheduled, OnError: o.ErrorReporter, Debug: o.Debug,
		Post: d.Post, UIThread: o.UIThread, ErrorWidgetBuilder: o.ErrorWidgetBuilder,
	})
	b.owner.binding = b
	b.focusManager = NewFocusManager(b)
	b.focusManager.RegisterGlobalHandlers()
	b.renderer.WrapDrawFrame(b.drawFrame)

	// The callbacks that the layers below set are called first (the renderer
	// binding hears of the metrics, the semantics binding of the accessibility
	// features), then the observers.
	previousMetrics := d.OnMetricsChanged
	d.OnMetricsChanged = func() {
		if previousMetrics != nil {
			previousMetrics()
		}
		b.notifyObservers("didChangeMetrics", WidgetsBindingObserver.DidChangeMetrics)
	}
	previousFeatures := d.OnAccessibilityFeaturesChanged
	d.OnAccessibilityFeaturesChanged = func() {
		if previousFeatures != nil {
			previousFeatures()
		}
		b.notifyObservers("didChangeAccessibilityFeatures", WidgetsBindingObserver.DidChangeAccessibilityFeatures)
	}
	d.OnTextScaleFactorChanged = func() {
		b.notifyObservers("didChangeTextScaleFactor", WidgetsBindingObserver.DidChangeTextScaleFactor)
	}
	d.OnPlatformBrightnessChanged = func() {
		b.notifyObservers("didChangePlatformBrightness", WidgetsBindingObserver.DidChangePlatformBrightness)
	}
	d.OnLocaleChanged = b.handleLocaleChanged
	d.OnAppLifecycleStateChanged = b.handleAppLifecycleStateChanged
	d.OnExitRequested = func(ui.ViewID) ui.AppExitResponse { return b.HandleRequestAppExit() }
	return b
}

// DefaultErrorReporter makes the error reporter that logs each error at the
// error level, with its ToJson as an attribute so that it can be replayed in a
// test with errors.FromJson. A nil logger means slog.Default().
//
// Flutter: foundation/assertions.dart FlutterError.presentError
func DefaultErrorReporter(logger *slog.Logger) foundation.ErrorReporter {
	return func(err error) {
		l := logger
		if l == nil {
			l = slog.Default()
		}
		if appErr, ok := errors.FromError(err); ok {
			if j, jerr := appErr.ToJson(); jerr == nil {
				l.Error("glutter: "+appErr.Tag+": "+appErr.Message, "error", j)
			} else {
				l.Error("glutter: "+appErr.Tag+": "+appErr.Message, "error", appErr)
			}
			return
		}
		l.Error("glutter: "+err.Error(), "error", err)
	}
}

// WidgetsBindingOf is the widgets binding of the app that the context belongs
// to. It panics with the tag WidgetsBindingMissing when the context's build
// owner was not made by a binding.
//
// Flutter: widgets/binding.dart WidgetsBinding.instance
func WidgetsBindingOf(bc BuildContext) *WidgetsBinding {
	if b := MaybeWidgetsBindingOf(bc); b != nil {
		return b
	}
	panic(errors.New("WidgetsBindingMissing",
		"WidgetsBindingOf was called with a context whose build owner does not belong to a WidgetsBinding; widgets run by RunApp and by gluttertest always have one"))
}

// MaybeWidgetsBindingOf is WidgetsBindingOf that returns nil instead of
// panicking.
//
// Flutter: widgets/binding.dart WidgetsBinding.instance
func MaybeWidgetsBindingOf(bc BuildContext) *WidgetsBinding {
	if bc == nil {
		return nil
	}
	if o := bc.Owner(); o != nil {
		return o.binding
	}
	return nil
}

// PlatformDispatcher is the dispatcher the binding listens to.
//
// Flutter: widgets/binding.dart WidgetsBinding.platformDispatcher
func (b *WidgetsBinding) PlatformDispatcher() *ui.PlatformDispatcher { return b.d }

// Scheduler is the scheduler binding.
//
// Flutter: scheduler/binding.dart SchedulerBinding.instance
func (b *WidgetsBinding) Scheduler() *scheduler.SchedulerBinding { return b.scheduler }

// Gestures is the gesture binding.
//
// Flutter: gestures/binding.dart GestureBinding.instance
func (b *WidgetsBinding) Gestures() *gestures.GestureBinding { return b.gesture }

// Painting is the painting binding.
//
// Flutter: painting/binding.dart PaintingBinding.instance
func (b *WidgetsBinding) Painting() *painting.PaintingBinding { return b.painting }

// Renderer is the renderer binding.
//
// Flutter: rendering/binding.dart RendererBinding.instance
func (b *WidgetsBinding) Renderer() *rendering.RendererBinding { return b.renderer }

// BuildOwner is the build owner of the app's widget tree.
//
// Flutter: widgets/binding.dart WidgetsBinding.buildOwner
func (b *WidgetsBinding) BuildOwner() *BuildOwner { return b.owner }

// DebugOptions are the debug flags of the app.
//
// Flutter: widgets/debug.dart
func (b *WidgetsBinding) DebugOptions() *DebugOptions { return b.debug }

// ErrorReporter is the reporter that every part of the framework reports to.
//
// Flutter: foundation/assertions.dart FlutterError.onError
func (b *WidgetsBinding) ErrorReporter() foundation.ErrorReporter { return b.report }

// UIThread is the UI thread of the app.
//
// Flutter: no counterpart
func (b *WidgetsBinding) UIThread() *foundation.UIThread { return b.thread }

// Post runs fn on the UI thread; it is safe to call from any goroutine
// (SPEC §13).
//
// Flutter: no counterpart
func (b *WidgetsBinding) Post(fn func()) { b.d.Post(fn) }

// RootElement is the root of the widget tree, or nil before AttachRootWidget.
//
// Flutter: widgets/binding.dart WidgetsBinding.rootElement
func (b *WidgetsBinding) RootElement() *RootElement { return b.rootElement }

// Keyboard is the hardware keyboard: the keys that are pressed, the lock modes,
// and the handlers that see the key events first. The binding feeds it the
// engine's key data.
//
// Flutter: services/binding.dart ServicesBinding.keyboard
func (b *WidgetsBinding) Keyboard() *services.HardwareKeyboard { return b.keyboard }

// SystemNavigator is the navigation of the system around the app. Its Pop ends
// the app.
//
// Flutter: services/system_navigator.dart SystemNavigator
func (b *WidgetsBinding) SystemNavigator() *services.SystemNavigator { return b.navigator }

// SystemSound plays the sounds of the system; see services.SystemSound.
//
// Flutter: services/system_sound.dart SystemSound
func (b *WidgetsBinding) SystemSound() *services.SystemSound { return b.sound }

// Clipboard is the system clipboard; see services.Clipboard.
//
// Flutter: services/clipboard.dart Clipboard
func (b *WidgetsBinding) Clipboard() *services.Clipboard { return b.clipboard }

// TextInput is the text input service: the text fields attach to it to use the
// input method (IME) of the platform; see services.TextInput.
//
// Flutter: services/text_input.dart TextInput
func (b *WidgetsBinding) TextInput() *services.TextInput { return b.textInput }

// IsRootWidgetAttached is whether the root widget was attached.
//
// Flutter: widgets/binding.dart WidgetsBinding.isRootWidgetAttached
func (b *WidgetsBinding) IsRootWidgetAttached() bool { return b.rootElement != nil }

// AddObserver registers an observer. An observer registered twice is called
// twice.
//
// Flutter: widgets/binding.dart WidgetsBinding.addObserver
func (b *WidgetsBinding) AddObserver(observer WidgetsBindingObserver) {
	b.observers = append(b.observers, observer)
}

// RemoveObserver removes an observer and reports whether it was registered.
//
// Flutter: widgets/binding.dart WidgetsBinding.removeObserver
func (b *WidgetsBinding) RemoveObserver(observer WidgetsBindingObserver) bool {
	i := slices.Index(b.observers, observer)
	if i < 0 {
		return false
	}
	b.observers = slices.Delete(b.observers, i, i+1)
	return true
}

// notifyObservers calls the method on a copy of the list of observers. A panic
// in an observer is reported and the others are still called.
func (b *WidgetsBinding) notifyObservers(name string, call func(WidgetsBindingObserver)) {
	for _, observer := range slices.Clone(b.observers) {
		foundation.Guard(b.report, foundation.CallbackPanicked,
			"while dispatching notifications for WidgetsBindingObserver."+name, func() []any {
				return []any{"library", "widgets library", "observer", foundation.RuntimeTypeName(observer)}
			}, func() { call(observer) })
	}
}

func (b *WidgetsBinding) handleLocaleChanged() {
	locales := b.d.Locales()
	b.notifyObservers("didChangeLocales", func(o WidgetsBindingObserver) { o.DidChangeLocales(locales) })
}

// handleAppLifecycleStateChanged applies a change of the lifecycle state that
// the platform reports. The platform may skip states (a minimized window goes
// from resumed to hidden), so the intermediate states are generated, and each
// state change reaches the scheduler and the observers in turn.
//
// Flutter: services/binding.dart ServicesBinding._handleLifecycleMessage
func (b *WidgetsBinding) handleAppLifecycleStateChanged(state ui.AppLifecycleState) {
	for _, s := range generateStateTransitions(b.scheduler.LifecycleState(), state) {
		b.scheduler.HandleAppLifecycleStateChanged(s)
		b.notifyObservers("didChangeAppLifecycleState", func(o WidgetsBindingObserver) { o.DidChangeAppLifecycleState(s) })
	}
}

// generateStateTransitions is the list of states to go through from the previous
// state to the new one: the states between them, in the order of the lifecycle
// (detached, resumed, inactive, hidden, paused) when the new state is later in
// the order, and in the reverse order when it is earlier, so that a listener
// sees only the transitions that its callbacks know. Detaching goes through all
// the states up to paused. With no previous state it jumps to the new state.
//
// Flutter: services/binding.dart ServicesBinding._generateStateTransitions
func generateStateTransitions(previous, state ui.AppLifecycleState) []ui.AppLifecycleState {
	if previous == state {
		return nil
	}
	if previous == ui.AppLifecycleStateUnspecified {
		return []ui.AppLifecycleState{state}
	}
	var changes []ui.AppLifecycleState
	switch {
	case state == ui.AppLifecycleStateDetached:
		for s := previous + 1; s <= ui.AppLifecycleStatePaused; s++ {
			changes = append(changes, s)
		}
		changes = append(changes, ui.AppLifecycleStateDetached)
	case previous > state:
		for s := state; s < previous; s++ {
			changes = slices.Insert(changes, 0, s)
		}
	default:
		for s := previous + 1; s <= state; s++ {
			changes = append(changes, s)
		}
	}
	return changes
}

// HandleMemoryPressure tells the painting binding to drop its image cache and
// the observers about it. Nothing calls it on Linux desktop yet.
//
// Flutter: widgets/binding.dart WidgetsBinding.handleMemoryPressure
func (b *WidgetsBinding) HandleMemoryPressure() {
	b.painting.HandleMemoryPressure()
	b.notifyObservers("didHaveMemoryPressure", WidgetsBindingObserver.DidHaveMemoryPressure)
}

// HandlePopRoute tells the observers, in order of registration, that the system
// asks to pop the current route; the first that returns true handles it. It
// reports whether one did. If none does, the app asks to exit, as Flutter's
// SystemNavigator.pop does.
//
// Flutter: widgets/binding.dart WidgetsBinding.handlePopRoute
func (b *WidgetsBinding) HandlePopRoute() bool {
	for _, observer := range slices.Clone(b.observers) {
		handled := false
		foundation.Guard(b.report, foundation.CallbackPanicked,
			"while dispatching notifications for WidgetsBindingObserver.didPopRoute", nil,
			func() { handled = observer.DidPopRoute() })
		if handled {
			return true
		}
	}
	b.navigator.Pop()
	return false
}

// HandlePushRouteInformation tells the observers, in order of registration, that
// the system pushes a route; the first that returns true handles it. It reports
// whether one did.
//
// Flutter: widgets/binding.dart WidgetsBinding.handlePushRoute
func (b *WidgetsBinding) HandlePushRouteInformation(routeInformation *RouteInformation) bool {
	for _, observer := range slices.Clone(b.observers) {
		handled := false
		foundation.Guard(b.report, foundation.CallbackPanicked,
			"while dispatching notifications for WidgetsBindingObserver.didPushRouteInformation", nil,
			func() { handled = observer.DidPushRouteInformation(routeInformation) })
		if handled {
			return true
		}
	}
	return false
}

// HandleRequestAppExit asks the observers whether the app may exit: the answer
// is Cancel if any observer says Cancel, else Exit. It does not exit.
//
// Flutter: widgets/binding.dart WidgetsBinding.handleRequestAppExit
func (b *WidgetsBinding) HandleRequestAppExit() ui.AppExitResponse {
	response := ui.AppExitResponseExit
	for _, observer := range slices.Clone(b.observers) {
		foundation.Guard(b.report, foundation.CallbackPanicked,
			"while dispatching notifications for WidgetsBindingObserver.didRequestAppExit", nil,
			func() {
				if observer.DidRequestAppExit() == ui.AppExitResponseCancel {
					response = ui.AppExitResponseCancel
				}
			})
	}
	return response
}

// ExitApplication ends the app. A cancelable exit asks the observers first and
// does nothing if one cancels; a required exit does not ask. It returns the
// response.
//
// Flutter: services/binding.dart ServicesBinding.exitApplication
func (b *WidgetsBinding) ExitApplication(exitType ui.AppExitType) ui.AppExitResponse {
	if exitType == ui.AppExitTypeCancelable && b.HandleRequestAppExit() == ui.AppExitResponseCancel {
		return ui.AppExitResponseCancel
	}
	b.d.RequestAppExit()
	return ui.AppExitResponseExit
}

func (b *WidgetsBinding) handleBuildScheduled() {
	if foundation.DebugMode && b.debugBuildingDirtyElements {
		panic(errors.New("BuildScheduledDuringFrame",
			"while the widget tree was being built, laid out, and painted, a new frame was scheduled to rebuild the widget tree; this might be because SetState was called from a layout or paint callback; if a change is needed to the widget tree, it should be applied as the tree is being built; scheduling a change for the subsequent frame instead results in an interface that lags behind by one frame; if this was done to make your build dependent on a size measured at layout time, consider using a LayoutBuilder, CustomSingleChildLayout, or CustomMultiChildLayout; if, on the other hand, the one frame delay is the desired effect, for example because this is an animation, consider scheduling the frame in a post-frame callback using SchedulerBinding.AddPostFrameCallback or using an AnimationController to trigger the animation"))
	}
	b.scheduler.EnsureVisualUpdate()
}

// DebugBuildingDirtyElements is whether the binding is building, laying out and
// painting a frame (debug builds only).
//
// Flutter: widgets/binding.dart WidgetsBinding.debugBuildingDirtyElements
func (b *WidgetsBinding) DebugBuildingDirtyElements() bool { return b.debugBuildingDirtyElements }

// drawFrame builds the dirty widgets, lets the rendering binding lay out, paint
// and composite, and finalizes the tree.
//
// Flutter: widgets/binding.dart WidgetsBinding.drawFrame
func (b *WidgetsBinding) drawFrame(super func() error) error {
	if foundation.DebugMode {
		b.debugBuildingDirtyElements = true
		defer func() { b.debugBuildingDirtyElements = false }()
	}
	if b.rootElement != nil {
		b.owner.BuildScope(b.rootElement, nil)
	}
	err := super()
	b.owner.FinalizeTree()
	if b.needToReportFirstFrame && b.renderer.SendFramesToEngine() {
		b.needToReportFirstFrame = false
		close(b.firstFrameDone)
	}
	return err
}

// FirstFrameRasterized is whether the first frame was sent to the engine.
//
// Deviation: Flutter waits for the engine to report the frame rasterized; the
// engine has no such report, so a frame counts when View.Render returned
// (SPEC §12.10).
//
// Flutter: widgets/binding.dart WidgetsBinding.firstFrameRasterized
func (b *WidgetsBinding) FirstFrameRasterized() bool { return !b.needToReportFirstFrame }

// WaitUntilFirstFrameRasterized returns a channel that is closed when the first
// frame was sent to the engine.
//
// Flutter: widgets/binding.dart WidgetsBinding.waitUntilFirstFrameRasterized
func (b *WidgetsBinding) WaitUntilFirstFrameRasterized() <-chan struct{} { return b.firstFrameDone }

// WrapWithDefaultView wraps the widget in a View for the implicit view of the
// dispatcher, as RunApp does for the app. It panics with the tag
// ImplicitViewMissing when the platform gave no implicit view.
//
// Flutter: widgets/binding.dart WidgetsBinding.wrapWithDefaultView
func (b *WidgetsBinding) WrapWithDefaultView(root Widget) Widget {
	view := b.d.ImplicitView()
	if view == nil {
		panic(errors.New("ImplicitViewMissing",
			"the app requested a view, but the platform did not provide one; this is likely because the app called RunApp to render its root widget, which expects the platform to provide a default view to render into (the implicit view); try using RunWidget instead of RunApp, which allows you to provide a View widget without requiring a default view"))
	}
	return NewView(view, root)
}

// AttachRootWidget makes the widget the root of the tree: the first call
// creates the root element and mounts the widget below it; later calls replace
// the widget. It makes the binding produce frames and schedules the first one.
//
// Flutter: widgets/binding.dart WidgetsBinding.attachRootWidget
func (b *WidgetsBinding) AttachRootWidget(root Widget) {
	b.AttachToBuildOwner(NewRootWidget(root, RootWidgetOpts{DebugShortDescription: "[root]"}))
}

// AttachToBuildOwner attaches the root widget to the build owner.
//
// Flutter: widgets/binding.dart WidgetsBinding.attachToBuildOwner
func (b *WidgetsBinding) AttachToBuildOwner(root *RootWidget) {
	bootstrap := b.rootElement == nil
	b.readyToProduceFrames = true
	b.rootElement = root.Attach(b.owner, b.rootElement)
	if bootstrap {
		b.scheduler.EnsureVisualUpdate()
	}
}

// Reassemble marks every element for a rebuild and every render object for
// layout and paint, as after a hot reload.
//
// Flutter: widgets/binding.dart WidgetsBinding.performReassemble
func (b *WidgetsBinding) Reassemble() {
	if b.rootElement != nil {
		b.owner.Reassemble(b.rootElement)
	}
	b.renderer.Reassemble()
}

// RunPostFrame calls fn after the current frame, or after the next one if none
// is running. It is a short form of Scheduler().AddPostFrameCallback.
//
// Flutter: scheduler/binding.dart SchedulerBinding.addPostFrameCallback
func (b *WidgetsBinding) RunPostFrame(fn func(timeStamp time.Duration)) {
	b.scheduler.AddPostFrameCallback(fn)
	b.scheduler.EnsureVisualUpdate()
}

// RootWidgetOpts are the optional parameters of NewRootWidget.
//
// Flutter: widgets/binding.dart RootWidget.new
type RootWidgetOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// DebugShortDescription is what ToStringShort returns.
	DebugShortDescription string
}

// RootWidget is the widget at the root of the tree: it stands for the app and
// has the root element, which has no parent.
//
// Flutter: widgets/binding.dart RootWidget
type RootWidget struct {
	WidgetBase
	child Widget
	desc  string
}

// NewRootWidget creates the root widget with its child (which may be nil).
//
// Flutter: widgets/binding.dart RootWidget.new
func NewRootWidget(child Widget, opts ...RootWidgetOpts) *RootWidget {
	o := firstOpts(opts)
	return &RootWidget{WidgetBase: NewWidgetBase(WidgetOpts{Key: o.Key}), child: child, desc: o.DebugShortDescription}
}

// Child is the widget below the root.
//
// Flutter: widgets/binding.dart RootWidget.child
func (w *RootWidget) Child() Widget { return w.child }

// ToStringShort is the debug short description, or the type name.
//
// Flutter: widgets/binding.dart RootWidget.toStringShort
func (w *RootWidget) ToStringShort() string {
	if w.desc != "" {
		return w.desc
	}
	return "RootWidget"
}

// CreateElement creates the root element.
//
// Flutter: widgets/binding.dart RootWidget.createElement
func (w *RootWidget) CreateElement() Element {
	e := &RootElement{}
	InitElement(&e.ElementBase, e, w)
	return e
}

// Attach creates the root element, gives it the owner and mounts it; or, given
// the element of an earlier call, marks it for an update to this widget.
//
// Flutter: widgets/binding.dart RootWidget.attach
func (w *RootWidget) Attach(owner *BuildOwner, element *RootElement) *RootElement {
	if element != nil {
		element.newWidget = w
		element.MarkNeedsBuild()
		return element
	}
	owner.LockState(func() {
		element = w.CreateElement().(*RootElement)
		element.AssignOwner(owner)
	})
	owner.BuildScope(element, func() { element.Mount(nil, nil) })
	return element
}

// RootElement is the element of a RootWidget: the root of the element tree.
//
// Flutter: widgets/binding.dart RootElement
type RootElement struct {
	ElementBase
	child     Element
	newWidget *RootWidget
}

// VisitChildren visits the child.
//
// Flutter: widgets/binding.dart RootElement.visitChildren
func (e *RootElement) VisitChildren(visitor ElementVisitor) {
	if e.child != nil {
		visitor(e.child)
	}
}

// ForgetChild drops the child.
//
// Flutter: widgets/binding.dart RootElement.forgetChild
func (e *RootElement) ForgetChild(child Element) {
	e.child = nil
	e.ElementBase.ForgetChild(child)
}

// Mount mounts the element as the root and builds its child.
//
// Flutter: widgets/binding.dart RootElement.mount
func (e *RootElement) Mount(parent Element, slot any) {
	if foundation.DebugMode && parent != nil {
		panic(errors.New("RootElementHasParent", "the root element was mounted with a parent"))
	}
	e.ElementBase.Mount(parent, slot)
	e.rebuild()
	e.ElementBase.PerformRebuild() // clears the dirty flag
}

// Update replaces the widget and rebuilds the child.
//
// Flutter: widgets/binding.dart RootElement.update
func (e *RootElement) Update(newWidget Widget) {
	e.ElementBase.Update(newWidget)
	e.rebuild()
}

// PerformRebuild applies the widget that Attach set aside.
//
// Flutter: widgets/binding.dart RootElement.performRebuild
func (e *RootElement) PerformRebuild() {
	if e.newWidget != nil {
		w := e.newWidget
		e.newWidget = nil
		e.Update(w)
	}
	e.ElementBase.PerformRebuild()
}

// rebuild updates the child. A panic is reported and leaves the tree empty.
func (e *RootElement) rebuild() {
	var report foundation.ErrorReporter
	if e.owner != nil {
		report = e.owner.OnError
	}
	panicked := foundation.Guard(report, foundation.BuildPanicked, "attaching to the render tree", func() []any {
		return []any{"library", "widgets library", "element", e.String()}
	}, func() {
		e.child = e.UpdateChild(e.child, e.widget.(*RootWidget).child, nil)
	})
	if panicked {
		e.child = nil
	}
}

// DebugDoingBuild is false: this element has no build phase.
//
// Flutter: widgets/binding.dart RootElement.debugDoingBuild
func (e *RootElement) DebugDoingBuild() bool { return false }

// DebugExpectsRenderObjectForSlot is false.
//
// Flutter: widgets/binding.dart RootElement.debugExpectsRenderObjectForSlot
func (e *RootElement) DebugExpectsRenderObjectForSlot(any) bool { return false }
