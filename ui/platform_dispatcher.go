package ui

import (
	"context"
	"slices"
	"time"

	"github.com/rm4n0s/errors"
)

// PlatformHost is what the dispatcher asks of the engine. The engine
// implements it; apps never do.
//
// Flutter: no counterpart (the engine's native side of dart:ui)
type PlatformHost interface {
	// ScheduleFrame asks for OnBeginFrame and OnDrawFrame to be called at the
	// next moment the display wants a frame.
	ScheduleFrame()
	// RenderView draws the scene into the view.
	RenderView(view ViewID, scene *Scene) error
	// RequestAppExit asks the engine to end the app.
	RequestAppExit()
	// Post runs fn on the UI thread, after the work that is running now. It is
	// safe to call from any goroutine.
	Post(fn func())
}

// SemanticsHost is what the engine implements to take part in semantics. The
// dispatcher uses it when the host has it; a host without it ignores semantics.
//
// Flutter: no counterpart (the engine's native side of dart:ui)
type SemanticsHost interface {
	// UpdateSemantics gives the view the changes to its semantics tree.
	UpdateSemantics(view ViewID, update *SemanticsUpdate)
	// SetSemanticsTreeEnabled says whether the framework produces a semantics
	// tree; the engine then keeps its accessibility bridge in step.
	SetSemanticsTreeEnabled(enabled bool)
}

// TextInputHost is what the engine implements to give the input method (IME) of
// the platform the state of the focused text field. The dispatcher uses it when
// the host has it; a host without it has no input method.
//
// Flutter: no counterpart (the engine's side of the flutter/textinput channel)
type TextInputHost interface {
	// SetTextInput gives the input method the state of the focused text field.
	// A state with Enabled false turns the input method off.
	SetTextInput(state TextInputState)
}

// CursorHost is what the engine implements to show the system mouse cursors. The
// dispatcher uses it when the host has it; a host without it shows none.
//
// Flutter: no counterpart (the engine's side of the flutter/mousecursor channel)
type CursorHost interface {
	// ActivateSystemCursor shows the system cursor of the kind (the kind of a
	// SystemMouseCursor, such as "click") over the window that the device's
	// pointer is in.
	ActivateSystemCursor(device int, kind string)
}

// ContextMenuHost is what the engine implements to show or hide the browser's context menu.
// The dispatcher uses it when the host has it; a host without it has no such menu.
//
// Flutter: no counterpart (the engine's side of the flutter/contextmenu channel)
type ContextMenuHost interface {
	// SetBrowserContextMenuEnabled shows (true) or hides the browser's context menu. It reports
	// whether a window has a menu to change.
	SetBrowserContextMenuEnabled(enabled bool) bool
}

// ClipboardHost is what the engine implements to give access to the system
// clipboard. The dispatcher uses it when the host has it; a host without it
// fails the clipboard calls with ClipboardUnavailable. The functions that the
// methods take are called later on the UI thread, never before the method has
// returned.
//
// Flutter: no counterpart (the engine's side of the Clipboard calls of the
// flutter/platform channel)
type ClipboardHost interface {
	// SetClipboardText makes text the content of the clipboard; done, when it is
	// not nil, gets the error if any.
	SetClipboardText(ctx context.Context, text string, done func(err error))
	// GetClipboardText reads the text of the clipboard; ok is false when the
	// clipboard has no text.
	GetClipboardText(ctx context.Context, done func(text string, ok bool, err error))
}

// PlatformConfiguration is the system-wide configuration the app follows. The
// engine hands one to the dispatcher whenever it changes.
//
// Flutter: ui/platform_dispatcher.dart PlatformConfiguration
type PlatformConfiguration struct {
	// Locales are the user's locales, most preferred first.
	Locales []Locale
	// TextScaleFactor is the system text scale; 0 means 1.
	TextScaleFactor float64
	// PlatformBrightness is the system color scheme; BrightnessUnspecified
	// means light.
	PlatformBrightness Brightness
	// AccessibilityFeatures are the system's accessibility settings.
	AccessibilityFeatures AccessibilityFeatures
	// AccentColor is the system accent color, or nil if there is none.
	AccentColor *Color
	// UIFontFamily and UIFontSize are the family and the size, in points, of
	// the desktop's interface font (GNOME's font-name); empty and 0 when the
	// platform does not say.
	//
	// Flutter: no counterpart
	UIFontFamily string
	UIFontSize   float64
	// MonospaceFontFamily and MonospaceFontSize are the family and the size, in
	// points, of the desktop's monospace font (GNOME's monospace-font-name); empty
	// and 0 when the platform does not say. A code editor uses them as its default
	// font.
	//
	// Flutter: no counterpart
	MonospaceFontFamily string
	MonospaceFontSize   float64
}

// PlatformDispatcher is the central point where the framework meets the
// platform: the views, the frame callbacks, the input, and the system
// settings.
//
// The engine creates one with NewPlatformDispatcher and gives it to the
// framework (SPEC §7, §13). Everything here runs on the UI thread. The
// callbacks are plain fields; set the ones you need. The methods under
// "Called by the engine" are for the engine to report what happens; an app
// does not call them.
//
// Flutter: ui/platform_dispatcher.dart PlatformDispatcher
type PlatformDispatcher struct {
	// OnMetricsChanged is called when the metrics of a view change.
	//
	// Flutter: ui/platform_dispatcher.dart PlatformDispatcher.onMetricsChanged
	OnMetricsChanged func()
	// OnLocaleChanged is called when the locales change.
	//
	// Flutter: ui/platform_dispatcher.dart PlatformDispatcher.onLocaleChanged
	OnLocaleChanged func()
	// OnTextScaleFactorChanged is called when the text scale factor changes.
	//
	// Flutter: ui/platform_dispatcher.dart PlatformDispatcher.onTextScaleFactorChanged
	OnTextScaleFactorChanged func()
	// OnPlatformBrightnessChanged is called when the system brightness
	// changes.
	//
	// Flutter: ui/platform_dispatcher.dart PlatformDispatcher.onPlatformBrightnessChanged
	OnPlatformBrightnessChanged func()
	// OnPlatformConfigurationChanged is called when any system setting
	// changes, after the callback for that setting.
	//
	// Flutter: ui/platform_dispatcher.dart PlatformDispatcher.onPlatformConfigurationChanged
	OnPlatformConfigurationChanged func()
	// OnAccessibilityFeaturesChanged is called when the accessibility
	// features change.
	//
	// Flutter: ui/platform_dispatcher.dart PlatformDispatcher.onAccessibilityFeaturesChanged
	OnAccessibilityFeaturesChanged func()
	// OnFontsChanged is called after fonts were added to the text engine
	// (LoadFontFromList), so that the framework lays the text out again.
	//
	// Flutter: services/system_channels.dart SystemChannels.system (the fontsChange message)
	OnFontsChanged func()
	// OnSemanticsEnabledChanged is called when an assistive technology starts or
	// stops asking for semantics.
	//
	// Flutter: ui/platform_dispatcher.dart PlatformDispatcher.onSemanticsEnabledChanged
	OnSemanticsEnabledChanged func()
	// OnSemanticsActionEvent is called when an assistive technology asks a node
	// to perform an action.
	//
	// Flutter: ui/platform_dispatcher.dart PlatformDispatcher.onSemanticsActionEvent
	OnSemanticsActionEvent func(event SemanticsActionEvent)
	// OnBeginFrame is called to start a frame, with the time of the frame
	// since an arbitrary fixed point. The times never go back. Animation
	// tickers run here.
	//
	// Flutter: ui/platform_dispatcher.dart PlatformDispatcher.onBeginFrame
	OnBeginFrame func(frameTime time.Duration)
	// OnDrawFrame is called to build the frame's scene and render it, after
	// OnBeginFrame.
	//
	// Flutter: ui/platform_dispatcher.dart PlatformDispatcher.onDrawFrame
	OnDrawFrame func()
	// OnReportTimings is called with the timings of the frames that were drawn since
	// the last call: the engine reports a batch after each frame. The engine measures
	// the frames only while this is set (NeedsReportTimings).
	//
	// Flutter: ui/platform_dispatcher.dart PlatformDispatcher.onReportTimings
	OnReportTimings func(timings []FrameTiming)
	// OnPointerDataPacket is called with the pointer reports that arrived
	// since the last frame, at the start of a frame.
	//
	// Flutter: ui/platform_dispatcher.dart PlatformDispatcher.onPointerDataPacket
	OnPointerDataPacket func(packet PointerDataPacket)
	// OnKeyData is called for each key event. It returns whether the event
	// was handled.
	//
	// Flutter: ui/platform_dispatcher.dart PlatformDispatcher.onKeyData
	OnKeyData func(data KeyData) bool
	// OnTextInput is called with a change that the input method asks of the
	// focused text field (preedit, commit or a deletion).
	//
	// Flutter: services/text_input.dart TextInput._handleTextInputInvocation
	OnTextInput func(event TextInputEvent)
	// OnAppLifecycleStateChanged is called when the app lifecycle state
	// changes.
	//
	// Flutter: services/binding.dart (the flutter/lifecycle channel)
	OnAppLifecycleStateChanged func(state AppLifecycleState)
	// OnExitRequested is called when the user asks to close a view (the close
	// button of a window). It returns the response of the app: with
	// AppExitResponseCancel the window stays open. Nil lets the window close.
	// The framework installs it to ask the observers of the widgets binding
	// (SPEC §8.7 "Closing").
	//
	// Flutter: services/binding.dart ServicesBinding.handleRequestAppExit
	OnExitRequested func(view ViewID) AppExitResponse
	// OnViewFocusChange is called when a view gets or loses the focus.
	//
	// Flutter: ui/platform_dispatcher.dart PlatformDispatcher.onViewFocusChange
	OnViewFocusChange func(event ViewFocusEvent)
	// OnDrop is called for a drag and drop operation over a view, and returns the
	// action the framework accepts at that place (DropActionNone for none). A drop
	// carries the data.
	//
	// Flutter: no counterpart
	OnDrop func(event DropEvent) DropActions
	// OnDragEnd is called when a drag that the app started with StartDrag is over.
	//
	// Flutter: no counterpart
	OnDragEnd func(result DragResult)
	// OnWindowCloseRequested is called when the user asks to close the window of a
	// view. It returns true when the framework takes over, as the owner of a window
	// controller does: it decides whether to destroy the window, and the engine does
	// not close it. OnExitRequested is not asked then.
	//
	// Flutter: widgets/_window.dart RegularWindowControllerDelegate.onWindowCloseRequested
	OnWindowCloseRequested func(view ViewID) bool
	// IsIdle reports whether the framework has nothing more to show: no frame is
	// scheduled and no work that changes the picture is pending (an image that is
	// loading, an animation). The engine asks it to decide when the
	// self-capture (GLUTTER_SCREENSHOT) may save the window. Nil counts as idle.
	//
	// Flutter: no counterpart
	IsIdle func() bool

	host          PlatformHost
	views         map[ViewID]*View
	displays      []Display
	config        PlatformConfiguration
	lifecycle     AppLifecycleState
	lastFrameTime time.Duration
	textEngine    ParagraphFactory
	semanticsOn   bool
}

// NewPlatformDispatcher creates a dispatcher with no views. The host is the
// engine; it may be nil in a test, where scheduling does nothing and
// rendering returns PlatformHostMissing.
//
// Flutter: ui/platform_dispatcher.dart PlatformDispatcher.new
func NewPlatformDispatcher(host PlatformHost) *PlatformDispatcher {
	return &PlatformDispatcher{host: host, views: map[ViewID]*View{}}
}

// Views are the views that exist, in order of their IDs.
//
// Flutter: ui/platform_dispatcher.dart PlatformDispatcher.views
func (d *PlatformDispatcher) Views() []*View {
	views := make([]*View, 0, len(d.views))
	for _, v := range d.views {
		views = append(views, v)
	}
	slices.SortFunc(views, func(a, b *View) int {
		switch {
		case a.id < b.id:
			return -1
		case a.id > b.id:
			return 1
		}
		return 0
	})
	return views
}

// View returns the view with the ID, or nil if there is none.
//
// Flutter: ui/platform_dispatcher.dart PlatformDispatcher.view
func (d *PlatformDispatcher) View(id ViewID) *View { return d.views[id] }

// ImplicitView returns the first view, the one with ImplicitViewID, or nil if
// it was closed.
//
// Flutter: ui/platform_dispatcher.dart PlatformDispatcher.implicitView
func (d *PlatformDispatcher) ImplicitView() *View { return d.views[ImplicitViewID] }

// Displays are the connected displays.
//
// Flutter: ui/platform_dispatcher.dart PlatformDispatcher.displays
func (d *PlatformDispatcher) Displays() []Display { return slices.Clone(d.displays) }

// Locales are the user's locales, most preferred first.
//
// Flutter: ui/platform_dispatcher.dart PlatformDispatcher.locales
func (d *PlatformDispatcher) Locales() []Locale { return slices.Clone(d.config.Locales) }

// Locale is the user's most preferred locale; the empty locale if the
// platform reports none.
//
// Flutter: ui/platform_dispatcher.dart PlatformDispatcher.locale
func (d *PlatformDispatcher) Locale() Locale {
	if len(d.config.Locales) == 0 {
		return Locale{}
	}
	return d.config.Locales[0]
}

// TextScaleFactor is the system text scale; 1 unless the user changed it.
//
// Flutter: ui/platform_dispatcher.dart PlatformDispatcher.textScaleFactor
func (d *PlatformDispatcher) TextScaleFactor() float64 {
	if d.config.TextScaleFactor == 0 {
		return 1
	}
	return d.config.TextScaleFactor
}

// PlatformBrightness is the system color scheme; BrightnessLight unless the
// platform says dark.
//
// Flutter: ui/platform_dispatcher.dart PlatformDispatcher.platformBrightness
func (d *PlatformDispatcher) PlatformBrightness() Brightness {
	if d.config.PlatformBrightness == BrightnessUnspecified {
		return BrightnessLight
	}
	return d.config.PlatformBrightness
}

// AccessibilityFeatures are the system's accessibility settings.
//
// Flutter: ui/platform_dispatcher.dart PlatformDispatcher.accessibilityFeatures
func (d *PlatformDispatcher) AccessibilityFeatures() AccessibilityFeatures {
	return d.config.AccessibilityFeatures
}

// AccentColor is the system accent color, or nil if there is none.
//
// Flutter: ui/platform_dispatcher.dart PlatformDispatcher.systemAccentColor (in newer versions)
func (d *PlatformDispatcher) AccentColor() *Color { return d.config.AccentColor }

// AppLifecycleState is the lifecycle state of the app;
// AppLifecycleStateUnspecified until the platform reports one.
//
// Flutter: ui/platform_dispatcher.dart PlatformDispatcher.initialLifecycleState
func (d *PlatformDispatcher) AppLifecycleState() AppLifecycleState { return d.lifecycle }

// ScheduleFrame asks for OnBeginFrame and OnDrawFrame to be called at the next
// moment the display wants a frame. Asking again before that is the same as
// asking once.
//
// Flutter: ui/platform_dispatcher.dart PlatformDispatcher.scheduleFrame
func (d *PlatformDispatcher) ScheduleFrame() {
	if d.host != nil {
		d.host.ScheduleFrame()
	}
}

// Post runs fn on the UI thread, after the work that is running now, and
// wakes the event loop. It is safe to call from any goroutine; functions
// posted by one goroutine run in the order it posted them. Without an engine
// behind the dispatcher, fn runs at once.
//
// Flutter: dart:async Timer.run, scheduleMicrotask (the event loop of the UI isolate)
func (d *PlatformDispatcher) Post(fn func()) {
	if d.host == nil {
		fn()
		return
	}
	d.host.Post(fn)
}

// ActivateSystemCursor asks the engine to show the system cursor of the kind for
// the device. Nothing happens when the host cannot show cursors.
//
// Deviation: Flutter sends it as the activateSystemCursor call of the
// flutter/mousecursor platform channel (SPEC §12.10).
//
// Flutter: services/mouse_cursor.dart _SystemMouseCursorSession.activate
func (d *PlatformDispatcher) ActivateSystemCursor(device int, kind string) {
	if h, ok := d.host.(CursorHost); ok {
		h.ActivateSystemCursor(device, kind)
	}
}

// SetBrowserContextMenuEnabled asks the engine to show or to hide the browser's context
// menu, and reports whether it did: a host that is not a browser has no such menu and
// nothing happens.
//
// Deviation: Flutter sends it as the enableContextMenu and disableContextMenu calls of the
// flutter/contextmenu platform channel (SPEC §12.10).
//
// Flutter: services/browser_context_menu.dart BrowserContextMenu.enableContextMenu
func (d *PlatformDispatcher) SetBrowserContextMenuEnabled(enabled bool) bool {
	if h, ok := d.host.(ContextMenuHost); ok {
		return h.SetBrowserContextMenuEnabled(enabled)
	}
	return false
}

func (d *PlatformDispatcher) performWindowAction(view ViewID, action WindowAction) {
	if h, ok := d.host.(WindowHost); ok {
		h.PerformWindowAction(view, action)
	}
}

// SetTextInput gives the input method of the platform the state of the focused
// text field. Nothing happens when the host has no input method.
//
// Deviation: Flutter sends it as the TextInput.* calls of the flutter/textinput
// platform channel (SPEC §12.10).
//
// Flutter: services/text_input.dart _PlatformTextInputControl
func (d *PlatformDispatcher) SetTextInput(state TextInputState) {
	if h, ok := d.host.(TextInputHost); ok {
		h.SetTextInput(state)
	}
}

// SetClipboardText asks the engine to put text on the system clipboard. done,
// when it is not nil, is called later on the UI thread with the error if any.
// A host without a clipboard fails with ClipboardUnavailable.
//
// Deviation: Flutter sends it as the Clipboard.setData call of the
// flutter/platform channel (SPEC §12.10).
//
// Flutter: services/clipboard.dart Clipboard.setData
func (d *PlatformDispatcher) SetClipboardText(ctx context.Context, text string, done func(err error)) {
	if h, ok := d.host.(ClipboardHost); ok {
		h.SetClipboardText(ctx, text, done)
		return
	}
	if done != nil {
		err := errors.New(tagClipboardUnavailable, "the engine has no system clipboard")
		d.Post(func() { done(err) })
	}
}

// GetClipboardText asks the engine to read the text of the system clipboard.
// done is called later on the UI thread; ok is false when the clipboard has no
// text. A host without a clipboard fails with ClipboardUnavailable.
//
// Deviation: Flutter sends it as the Clipboard.getData call of the
// flutter/platform channel (SPEC §12.10).
//
// Flutter: services/clipboard.dart Clipboard.getData
func (d *PlatformDispatcher) GetClipboardText(ctx context.Context, done func(text string, ok bool, err error)) {
	if h, ok := d.host.(ClipboardHost); ok {
		h.GetClipboardText(ctx, done)
		return
	}
	err := errors.New(tagClipboardUnavailable, "the engine has no system clipboard")
	d.Post(func() { done("", false, err) })
}

// SetClipboard asks the engine to put the content on the selection of the kind
// (the system clipboard or the primary selection). done, when it is not nil, is
// called later on the UI thread with the error if any. A host without a rich
// clipboard fails with ClipboardUnavailable, except for the text of the system
// clipboard.
//
// Flutter: no counterpart
func (d *PlatformDispatcher) SetClipboard(ctx context.Context, kind ClipboardKind, content ClipboardContent, done func(err error)) {
	if h, ok := d.host.(RichClipboardHost); ok {
		h.SetClipboard(ctx, kind, content, done)
		return
	}
	if kind != ClipboardKindPrimary && content.HasText && len(content.PNG) == 0 && len(content.URIs) == 0 {
		d.SetClipboardText(ctx, content.Text, done)
		return
	}
	if done != nil {
		err := errors.New(tagClipboardUnavailable, "the engine has no system clipboard")
		d.Post(func() { done(err) })
	}
}

// GetClipboard asks the engine to read the selection of the kind in the format.
// done is called later on the UI thread; the content is empty when the selection
// has no such format. A host without a rich clipboard fails with
// ClipboardUnavailable, except for the text of the system clipboard.
//
// Flutter: no counterpart
func (d *PlatformDispatcher) GetClipboard(ctx context.Context, kind ClipboardKind, format string, done func(content ClipboardContent, err error)) {
	if h, ok := d.host.(RichClipboardHost); ok {
		h.GetClipboard(ctx, kind, format, done)
		return
	}
	if kind != ClipboardKindPrimary && format == ClipboardFormatText {
		d.GetClipboardText(ctx, func(text string, ok bool, err error) {
			done(ClipboardContent{Text: text, HasText: ok}, err)
		})
		return
	}
	err := errors.New(tagClipboardUnavailable, "the engine has no system clipboard")
	d.Post(func() { done(ClipboardContent{}, err) })
}

// RequestAppExit asks the engine to end the app: the windows close and the
// run loop returns.
//
// Flutter: services/system_channels.dart (System.exitApplication)
func (d *PlatformDispatcher) RequestAppExit() {
	if d.host != nil {
		d.host.RequestAppExit()
	}
}

// RequestClose is called by the engine when the user asks to close the view. It
// returns whether the view may close: false when OnExitRequested cancels.
//
// Flutter: services/binding.dart ServicesBinding.handleRequestAppExit
func (d *PlatformDispatcher) RequestClose(id ViewID) bool {
	if d.OnWindowCloseRequested != nil && d.OnWindowCloseRequested(id) {
		return false
	}
	if d.OnExitRequested == nil {
		return true
	}
	return d.OnExitRequested(id) != AppExitResponseCancel
}

// AddView makes a view with the ID and configuration and returns it. A second
// view with the same ID replaces the first. Called by the engine.
//
// Flutter: ui/platform_dispatcher.dart PlatformDispatcher.addView (the engine's _addView)
func (d *PlatformDispatcher) AddView(id ViewID, cfg ViewConfiguration) *View {
	v := &View{d: d, id: id, cfg: cfg}
	d.views[id] = v
	return v
}

// RemoveView removes the view with the ID. Called by the engine.
//
// Flutter: ui/platform_dispatcher.dart PlatformDispatcher.removeView (the engine's _removeView)
func (d *PlatformDispatcher) RemoveView(id ViewID) {
	if v, ok := d.views[id]; ok {
		v.destroyed = true // the framework may still render into it until its widget goes
		delete(d.views, id)
	}
}

// UpdateViewConfiguration sets the configuration of the view and calls
// OnMetricsChanged. A view that does not exist is ignored. Called by the
// engine.
//
// Flutter: ui/platform_dispatcher.dart PlatformDispatcher._updateWindowMetrics
func (d *PlatformDispatcher) UpdateViewConfiguration(id ViewID, cfg ViewConfiguration) {
	v, ok := d.views[id]
	if !ok {
		return
	}
	v.cfg = cfg
	if d.OnMetricsChanged != nil {
		d.OnMetricsChanged()
	}
}

// SetDisplays sets the connected displays. Called by the engine.
//
// Flutter: ui/platform_dispatcher.dart PlatformDispatcher._updateDisplays
func (d *PlatformDispatcher) SetDisplays(displays []Display) { d.displays = slices.Clone(displays) }

// UpdatePlatformConfiguration sets the system settings and calls the callbacks
// of the ones that changed (locales, text scale factor, brightness,
// accessibility features), then OnPlatformConfigurationChanged. Called by the
// engine.
//
// Flutter: ui/platform_dispatcher.dart PlatformDispatcher._updateLocales and _updateUserSettingsData
func (d *PlatformDispatcher) UpdatePlatformConfiguration(cfg PlatformConfiguration) {
	old := d.config
	d.config = cfg
	d.config.Locales = slices.Clone(cfg.Locales)
	changed := false
	call := func(f func()) {
		changed = true
		if f != nil {
			f()
		}
	}
	if !slices.Equal(old.Locales, d.config.Locales) {
		call(d.OnLocaleChanged)
	}
	scale := func(c PlatformConfiguration) float64 {
		if c.TextScaleFactor == 0 {
			return 1
		}
		return c.TextScaleFactor
	}
	if scale(old) != scale(d.config) {
		call(d.OnTextScaleFactorChanged)
	}
	bright := func(c PlatformConfiguration) Brightness {
		if c.PlatformBrightness == BrightnessUnspecified {
			return BrightnessLight
		}
		return c.PlatformBrightness
	}
	if bright(old) != bright(d.config) {
		call(d.OnPlatformBrightnessChanged)
	}
	if old.AccessibilityFeatures != d.config.AccessibilityFeatures {
		call(d.OnAccessibilityFeaturesChanged)
	}
	if (old.AccentColor == nil) != (d.config.AccentColor == nil) ||
		(old.AccentColor != nil && *old.AccentColor != *d.config.AccentColor) {
		changed = true
	}
	if old.UIFontFamily != d.config.UIFontFamily || old.UIFontSize != d.config.UIFontSize ||
		old.MonospaceFontFamily != d.config.MonospaceFontFamily || old.MonospaceFontSize != d.config.MonospaceFontSize {
		changed = true
	}
	if changed && d.OnPlatformConfigurationChanged != nil {
		d.OnPlatformConfigurationChanged()
	}
}

// SetAppLifecycleState sets the lifecycle state and calls
// OnAppLifecycleStateChanged when it changed. Called by the engine.
//
// Flutter: services/binding.dart (the flutter/lifecycle channel)
func (d *PlatformDispatcher) SetAppLifecycleState(state AppLifecycleState) {
	if state == d.lifecycle {
		return
	}
	d.lifecycle = state
	if d.OnAppLifecycleStateChanged != nil {
		d.OnAppLifecycleStateChanged(state)
	}
}

// BeginFrame starts a frame: it calls OnBeginFrame with the frame time. A
// time earlier than the last one is raised to it, so the times never go back.
// Called by the engine.
//
// Flutter: ui/platform_dispatcher.dart PlatformDispatcher._beginFrame
func (d *PlatformDispatcher) BeginFrame(frameTime time.Duration) {
	if frameTime < d.lastFrameTime {
		frameTime = d.lastFrameTime
	}
	d.lastFrameTime = frameTime
	if d.OnBeginFrame != nil {
		d.OnBeginFrame(frameTime)
	}
}

// DrawFrame ends a frame: it calls OnDrawFrame. Called by the engine.
//
// Flutter: ui/platform_dispatcher.dart PlatformDispatcher._drawFrame
func (d *PlatformDispatcher) DrawFrame() {
	if d.OnDrawFrame != nil {
		d.OnDrawFrame()
	}
}

// NeedsReportTimings says whether anything listens for frame timings, so that the
// engine measures them only when they are used. Called by the engine.
//
// Flutter: ui/platform_dispatcher.dart PlatformDispatcher._setNeedsReportTimings
func (d *PlatformDispatcher) NeedsReportTimings() bool { return d.OnReportTimings != nil }

// ReportTimings gives the timings of the frames that were drawn to OnReportTimings.
// Called by the engine.
//
// Flutter: ui/platform_dispatcher.dart PlatformDispatcher._reportTimings
func (d *PlatformDispatcher) ReportTimings(timings []FrameTiming) {
	if d.OnReportTimings != nil && len(timings) > 0 {
		d.OnReportTimings(timings)
	}
}

// DispatchPointerDataPacket gives the pointer reports to OnPointerDataPacket.
// Called by the engine.
//
// Flutter: ui/platform_dispatcher.dart PlatformDispatcher._dispatchPointerDataPacket
func (d *PlatformDispatcher) DispatchPointerDataPacket(packet PointerDataPacket) {
	if d.OnPointerDataPacket != nil {
		d.OnPointerDataPacket(packet)
	}
}

// DispatchKeyData gives a key event to OnKeyData and returns whether it was
// handled (false when nothing listens). Called by the engine.
//
// Flutter: ui/platform_dispatcher.dart PlatformDispatcher._dispatchKeyData
func (d *PlatformDispatcher) DispatchKeyData(data KeyData) bool {
	if d.OnKeyData == nil {
		return false
	}
	return d.OnKeyData(data)
}

// DispatchTextInput gives a change asked by the input method to OnTextInput.
// Called by the engine.
//
// Flutter: services/text_input.dart TextInput._handleTextInputInvocation
func (d *PlatformDispatcher) DispatchTextInput(event TextInputEvent) {
	if d.OnTextInput != nil {
		d.OnTextInput(event)
	}
}

// SetTextEngine sets the text engine that lays out paragraphs. The engine calls
// it at startup.
//
// Flutter: no counterpart (Dart reaches the engine's font collection globally)
func (d *PlatformDispatcher) SetTextEngine(engine ParagraphFactory) { d.textEngine = engine }

// LoadFontFromList loads a font while the app runs, and makes it usable in
// TextStyle.FontFamily and TextStyle.FontFamilyFallback under the family name;
// list is a TrueType or OpenType file. A family can have several fonts (regular,
// bold, italic...); the weight and slant of each come from the font. An empty
// family takes the name inside the font. The text laid out before is laid out
// again (OnFontsChanged). It must be called on the UI thread, and fails with
// FontLoadUnsupported when the text engine cannot load fonts, FontDataInvalid
// when list is not a font, and FontFamilyMissing when there is no family name.
//
// Flutter: ui/text.dart loadFontFromList
func (d *PlatformDispatcher) LoadFontFromList(list []byte, fontFamily string) error {
	loader, ok := d.textEngine.(FontLoadingEngine)
	if !ok {
		return errors.New(tagFontLoadUnsupported, "the text engine cannot load fonts", "family", fontFamily)
	}
	if err := loader.LoadFontFromList(list, fontFamily); err != nil {
		return err
	}
	if d.OnFontsChanged != nil {
		d.OnFontsChanged()
	}
	return nil
}

// ResolveFont says which face a font family resolves to for a weight and a slant: the
// one that draws a letter of a TextStyle that names only that family. A generic family
// (monospace) gives the system's font for it, and a name that the user's configuration
// stands for another (Arial) the font of that. It is false when the family has no font, or when
// the text engine cannot say. A zero weight is FontWeightW400.
//
// Flutter: no counterpart
func (d *PlatformDispatcher) ResolveFont(family string, weight FontWeight, style FontStyle) (ResolvedFont, bool) {
	q, ok := d.textEngine.(FontQueryEngine)
	if !ok {
		return ResolvedFont{}, false
	}
	return q.ResolveFont(family, weight, style)
}

// InstalledFontFamilies lists the font families that a TextStyle can name: the system's,
// the bundled and the loaded ones, sorted without regard to case. It is nil when the text engine
// cannot say. The first call on a machine that has no fontconfig scans the font directories.
//
// Flutter: no counterpart
func (d *PlatformDispatcher) InstalledFontFamilies() []string {
	q, ok := d.textEngine.(FontQueryEngine)
	if !ok {
		return nil
	}
	return q.InstalledFontFamilies()
}

// UIFont is the family and the size, in points, of the desktop's interface font; empty and
// 0 when the platform does not say.
//
// Flutter: no counterpart
func (d *PlatformDispatcher) UIFont() (family string, size float64) {
	return d.config.UIFontFamily, d.config.UIFontSize
}

// MonospaceFont is the family and the size, in points, of the desktop's monospace font; empty and
// 0 when the platform does not say.
//
// Flutter: no counterpart
func (d *PlatformDispatcher) MonospaceFont() (family string, size float64) {
	return d.config.MonospaceFontFamily, d.config.MonospaceFontSize
}

// TextEngine is the text engine that lays out paragraphs; it is nil until the
// engine sets one. The framework gives it to ParagraphBuilder.
//
// Flutter: no counterpart (Dart reaches the engine's font collection globally)
func (d *PlatformDispatcher) TextEngine() ParagraphFactory { return d.textEngine }

// SemanticsEnabled reports whether an assistive technology asks for semantics.
//
// Flutter: ui/platform_dispatcher.dart PlatformDispatcher.semanticsEnabled
func (d *PlatformDispatcher) SemanticsEnabled() bool { return d.semanticsOn }

// SetSemanticsEnabled says whether an assistive technology asks for semantics,
// and calls OnSemanticsEnabledChanged if that changed. Called by the engine.
//
// Flutter: ui/platform_dispatcher.dart PlatformDispatcher._updateSemanticsEnabled
func (d *PlatformDispatcher) SetSemanticsEnabled(enabled bool) {
	if d.semanticsOn == enabled {
		return
	}
	d.semanticsOn = enabled
	if d.OnSemanticsEnabledChanged != nil {
		d.OnSemanticsEnabledChanged()
	}
}

// DispatchSemanticsActionEvent reports an action that an assistive technology
// asked for. Called by the engine.
//
// Flutter: ui/platform_dispatcher.dart PlatformDispatcher._dispatchSemanticsAction
func (d *PlatformDispatcher) DispatchSemanticsActionEvent(event SemanticsActionEvent) {
	if d.OnSemanticsActionEvent != nil {
		d.OnSemanticsActionEvent(event)
	}
}

// SetSemanticsTreeEnabled tells the engine whether the framework produces a
// semantics tree.
//
// Flutter: ui/platform_dispatcher.dart PlatformDispatcher.setSemanticsTreeEnabled
func (d *PlatformDispatcher) SetSemanticsTreeEnabled(enabled bool) {
	if h, ok := d.host.(SemanticsHost); ok {
		h.SetSemanticsTreeEnabled(enabled)
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagClipboardUnavailable = "ClipboardUnavailable"
	tagFontLoadUnsupported  = "FontLoadUnsupported"
)
