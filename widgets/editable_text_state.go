package widgets

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// cursorBlinkHalfPeriod is how long the caret is on, and then off.
//
// Flutter: widgets/editable_text.dart _kCursorBlinkHalfPeriod
const cursorBlinkHalfPeriod = 500 * time.Millisecond

// caretAnimationDuration is how long the field scrolls to show the caret.
//
// Flutter: widgets/editable_text.dart EditableTextState._caretAnimationDuration
const caretAnimationDuration = 100 * time.Millisecond

// periodicTimer is Dart's Timer.periodic on the gesture binding's clock.
type periodicTimer struct {
	binding *gestures.GestureBinding
	period  time.Duration
	fn      func()
	timer   *gestures.Timer
	active  bool
}

func newPeriodicTimer(binding *gestures.GestureBinding, period time.Duration, fn func()) *periodicTimer {
	p := &periodicTimer{binding: binding, period: period, fn: fn, active: true}
	p.schedule()
	return p
}

func (p *periodicTimer) schedule() {
	p.timer = p.binding.NewTimer(p.period, func() {
		if !p.active {
			return
		}
		p.schedule()
		p.fn()
	})
}

func (p *periodicTimer) cancel() {
	p.active = false
	if p.timer != nil {
		p.timer.Cancel()
	}
}

func (p *periodicTimer) isActive() bool { return p != nil && p.active }

// EditableTextState is the state of an EditableText. It is the text input
// client the input method talks to, and the delegate of the render object for
// the clipboard and the selection.
//
// Flutter: widgets/editable_text.dart EditableTextState
type EditableTextState struct {
	StateOf[*EditableText]
	TickerProvider
	AutomaticKeepAliveClient
	WidgetsBindingObserverBase

	cursorTimer        *periodicTimer
	cursorBlinkOpacity *animation.AnimationController
	cursorVisibility   *foundation.ValueNotifier[bool]
	editableKey        *GlobalKey[State]
	scrollableKey      *GlobalKey[State]
	clipboardStatus    *ClipboardStatusNotifier
	selectionOverlay   *TextSelectionOverlay

	scrollObserver               *ScrollNotificationObserverState
	scrollObserverHandle         ScrollNotificationListenerHandle
	listeningToScrollObserver    bool
	dataWhenToolbarShow          *toolbarShowData
	showToolbarOnScreenScheduled bool

	textInputConnection *services.TextInputConnection
	internalScrollCtl   ScrollController
	toolbarLayerLink    *rendering.LayerLink
	startHandleLink     *rendering.LayerLink
	endHandleLink       *rendering.LayerLink
	didAutoFocus        bool
	style               *painting.TextStyle
	tickersEnabled      bool
	justResumed         bool

	lastKnownRemoteValue   foundation.Opt[services.TextEditingValue]
	batchEditDepth         int
	hadFocusOnTapDown      bool
	nextFocusChangeIntern  bool
	showCaretScheduled     bool
	primaryScheduled       bool
	restartConnScheduled   bool
	currentPromptRectRange foundation.Opt[ui.TextRange]

	actions                ActionMap
	verticalSelectionMover *updateTextSelectionVerticallyAction
	focusManagerHandles    focusManagerListeners
	controllerHandle       foundation.ListenerHandle
	focusHandle            foundation.ListenerHandle
	clipboardHandle        foundation.ListenerHandle
	cursorTickHandle       foundation.ListenerHandle
	lastBottomViewInset    float64

	// spellCheckConfiguration is the configuration that spell check runs with: the
	// widget's, with the service inferred.
	spellCheckConfiguration *SpellCheckConfiguration
	spellCheckResults       *services.SpellCheckResults
}

type focusManagerListeners struct {
	unflag foundation.ListenerHandle
	reset  foundation.ListenerHandle
	hasU   bool
	hasR   bool
}

func newEditableTextState() *EditableTextState {
	return &EditableTextState{
		editableKey:      NewGlobalKey[State](),
		scrollableKey:    NewGlobalKey[State](),
		toolbarLayerLink: rendering.NewLayerLink(),
		startHandleLink:  rendering.NewLayerLink(),
		endHandleLink:    rendering.NewLayerLink(),
		tickersEnabled:   true,
		cursorVisibility: foundation.NewValueNotifier(true),
	}
}

func (s *EditableTextState) binding() *WidgetsBinding { return WidgetsBindingOf(s.Context()) }

// report runs fn and reports a panic in it as an error, which is what Flutter
// does for the callbacks of the field.
func (s *EditableTextState) report(tag, what string, fn func()) {
	foundation.Guard(s.binding().ErrorReporter(), tag, what, nil, fn)
}

func (s *EditableTextState) blinkController() *animation.AnimationController {
	if s.cursorBlinkOpacity == nil {
		s.cursorBlinkOpacity = animation.NewAnimationController(s)
		s.cursorTickHandle = s.cursorBlinkOpacity.AddListener(s.onCursorColorTick)
	}
	return s.cursorBlinkOpacity
}

func (s *EditableTextState) scrollController() ScrollController {
	if c := s.Widget().o.ScrollController; !isNil(c) {
		return c
	}
	if s.internalScrollCtl == nil {
		s.internalScrollCtl = NewScrollController()
	}
	return s.internalScrollCtl
}

func (s *EditableTextState) hasInputConnection() bool {
	return s.textInputConnection != nil && s.textInputConnection.Attached()
}

func (s *EditableTextState) hasFocus() bool { return s.Widget().focusNode.HasFocus() }

func (s *EditableTextState) isMultiline() bool {
	m, ok := s.Widget().o.MaxLines.Get()
	return !ok || m != 1
}

func (s *EditableTextState) value() services.TextEditingValue { return s.Widget().controller.Value() }

func (s *EditableTextState) textDirection() ui.TextDirection {
	if d := s.Widget().o.TextDirection; d != ui.TextDirectionUnspecified {
		return d
	}
	return DirectionalityOf(s.Context())
}

func (s *EditableTextState) shouldCreateInputConnection() bool { return !s.Widget().o.ReadOnly }

// WantKeepAlive keeps the field alive in a lazy list while it has the focus.
//
// Flutter: widgets/editable_text.dart EditableTextState.wantKeepAlive
func (s *EditableTextState) WantKeepAlive() bool { return s.Widget().focusNode.HasFocus() }

// RenderEditable is the render object that shows the text. It is only available
// once the field is built.
//
// Flutter: widgets/editable_text.dart EditableTextState.renderEditable
func (s *EditableTextState) RenderEditable() *rendering.RenderEditable {
	bc := s.editableKey.CurrentContext()
	if bc == nil {
		return nil
	}
	// The callbacks of the composition and the post-frame callbacks can come after
	// the field was taken out of the tree: its element is inactive then.
	if el, ok := bc.(Element); ok && el.AsElement().lifecycle != lifecycleActive {
		return nil
	}
	r, _ := bc.FindRenderObject().(*rendering.RenderEditable)
	return r
}

func (s *EditableTextState) renderEditable() *rendering.RenderEditable {
	r := s.RenderEditable()
	if r == nil {
		panic(errors.New(tagEditableTextNotBuilt, "the render object of the EditableText is not available before it is built"))
	}
	return r
}

// cursorColor is the color of the caret, with the opacity of the blink.
//
// Flutter: widgets/editable_text.dart EditableTextState._cursorColor
func (s *EditableTextState) cursorColor() ui.Color {
	w := s.Widget()
	opacity := min(float64(w.cursorColor.Alpha())/255.0, s.blinkController().Value())
	return w.cursorColor.WithOpacity(opacity)
}

// CutEnabled is whether the cut command is available.
//
// Flutter: widgets/editable_text.dart EditableTextState.cutEnabled
func (s *EditableTextState) CutEnabled() bool {
	w := s.Widget()
	return !w.o.ReadOnly && !w.o.ObscureText && !s.value().Selection.IsCollapsed()
}

// CopyEnabled is whether the copy command is available.
//
// Flutter: widgets/editable_text.dart EditableTextState.copyEnabled
func (s *EditableTextState) CopyEnabled() bool {
	return !s.Widget().o.ObscureText && !s.value().Selection.IsCollapsed()
}

// PasteEnabled is whether the paste command is available.
//
// Flutter: widgets/editable_text.dart EditableTextState.pasteEnabled
func (s *EditableTextState) PasteEnabled() bool {
	return !s.Widget().o.ReadOnly && s.clipboardStatus.Value() == ClipboardStatusPasteable
}

// SelectAllEnabled is whether the select all command is available.
//
// Flutter: widgets/editable_text.dart EditableTextState.selectAllEnabled
func (s *EditableTextState) SelectAllEnabled() bool {
	w := s.Widget()
	if !w.enableInteractiveSelection || (w.o.ReadOnly && w.o.ObscureText) {
		return false
	}
	v := s.value()
	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformMacOS:
		return false
	case foundation.TargetPlatformIOS:
		return v.Text != "" && v.Selection.IsCollapsed()
	}
	return v.Text != "" && (v.Selection.Start() != 0 || v.Selection.End() != len(v.Text))
}

func (s *EditableTextState) selectedTextIsNotBlank() bool {
	v := s.value()
	return !s.Widget().o.ObscureText && !v.Selection.IsCollapsed() && strings.TrimSpace(v.Selection.Range().TextInside(v.Text)) != ""
}

// LookUpEnabled is whether the look up command is available, which is only on
// iOS.
//
// Flutter: widgets/editable_text.dart EditableTextState.lookUpEnabled
func (s *EditableTextState) LookUpEnabled() bool {
	return foundation.DefaultTargetPlatform() == foundation.TargetPlatformIOS && s.selectedTextIsNotBlank()
}

// SearchWebEnabled is whether the search web command is available, which is only
// on iOS.
//
// Flutter: widgets/editable_text.dart EditableTextState.searchWebEnabled
func (s *EditableTextState) SearchWebEnabled() bool { return s.LookUpEnabled() }

// ShareEnabled is whether the share command is available, which is only on
// Android and iOS.
//
// Flutter: widgets/editable_text.dart EditableTextState.shareEnabled
func (s *EditableTextState) ShareEnabled() bool {
	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformAndroid, foundation.TargetPlatformIOS:
		return s.selectedTextIsNotBlank()
	}
	return false
}

// LiveTextInputEnabled is whether Live Text input is available; it is not
// ported.
//
// Flutter: widgets/editable_text.dart EditableTextState.liveTextInputEnabled
func (s *EditableTextState) LiveTextInputEnabled() bool { return false }

func (s *EditableTextState) onChangedClipboardStatus() { s.SetState(func() {}) }

// CopySelection copies the selected text to the clipboard.
//
// Flutter: widgets/editable_text.dart EditableTextState.copySelection
func (s *EditableTextState) CopySelection(cause services.SelectionChangedCause) {
	v := s.value()
	selection := v.Selection
	if selection.IsCollapsed() || s.Widget().o.ObscureText {
		return
	}
	s.binding().Clipboard().SetData(s.Lifetime(), services.NewClipboardData(selection.Range().TextInside(v.Text)), func(err error) {
		s.reportClipboardError("while copying selection to clipboard", err)
	})
	if cause == services.SelectionChangedCauseToolbar {
		s.BringIntoView(v.Selection.Extent())
		s.HideToolbar(false)
		switch foundation.DefaultTargetPlatform() {
		case foundation.TargetPlatformAndroid, foundation.TargetPlatformFuchsia:
			// Collapse the selection and hide the toolbar and handles.
			s.UserUpdateTextEditingValue(services.NewTextEditingValue(services.TextEditingValueOpts{
				Text: v.Text, Selection: foundation.Some(services.NewTextSelectionCollapsed(v.Selection.End())),
			}), services.SelectionChangedCauseToolbar)
		}
	}
	s.clipboardStatus.Update()
}

// schedulePrimarySelection puts the selected text on the primary selection
// after the frame, once for all the changes of the selection of the frame (a
// drag changes it at every move). Linux desktops paste the primary selection
// with a middle click; the other platforms have none.
//
// Flutter: no counterpart
func (s *EditableTextState) schedulePrimarySelection() {
	if s.primaryScheduled || foundation.DefaultTargetPlatform() != foundation.TargetPlatformLinux {
		return
	}
	s.primaryScheduled = true
	s.binding().RunPostFrame(func(time.Duration) {
		s.primaryScheduled = false
		if !s.Mounted() {
			return
		}
		v := s.value()
		selection := v.Selection
		if !selection.IsValid() || selection.IsCollapsed() || s.Widget().o.ObscureText || selection.End() > len(v.Text) {
			return
		}
		// A platform without a primary selection fails the call; there is
		// nothing to do about it.
		s.binding().Clipboard().SetPrimaryData(s.Lifetime(), services.NewClipboardData(selection.Range().TextInside(v.Text)), nil)
	})
}

// PastePrimarySelection pastes the primary selection over the selection, as a
// middle click does on Linux desktops.
//
// Flutter: no counterpart
func (s *EditableTextState) PastePrimarySelection(cause services.SelectionChangedCause) {
	if !s.allowPaste() {
		return
	}
	ctx, cancel := context.WithTimeout(s.Lifetime(), 5*time.Second)
	s.binding().Clipboard().GetPrimaryData(ctx, services.ClipboardKTextPlain, func(data *services.ClipboardData, err error) {
		cancel()
		if err != nil {
			// A platform without a primary selection fails the call.
			return
		}
		if data == nil || !s.Mounted() {
			return
		}
		s.pasteText(cause, data.Text)
	})
}

// CutSelection copies the selected text to the clipboard and removes it.
//
// Flutter: widgets/editable_text.dart EditableTextState.cutSelection
func (s *EditableTextState) CutSelection(cause services.SelectionChangedCause) {
	w := s.Widget()
	if w.o.ReadOnly || w.o.ObscureText {
		return
	}
	v := s.value()
	selection := v.Selection
	if selection.IsCollapsed() {
		return
	}
	s.binding().Clipboard().SetData(s.Lifetime(), services.NewClipboardData(selection.Range().TextInside(v.Text)), func(err error) {
		s.reportClipboardError("while cutting selection to clipboard", err)
	})
	s.replaceText(NewReplaceTextIntent(v, "", selection.Range(), cause))
	if cause == services.SelectionChangedCauseToolbar {
		// Schedule a call to BringIntoView after the render editable updates.
		s.binding().RunPostFrame(func(time.Duration) {
			if s.Mounted() {
				s.BringIntoView(s.value().Selection.Extent())
			}
		})
		s.HideToolbar(true)
	}
	s.clipboardStatus.Update()
}

func (s *EditableTextState) reportClipboardError(what string, err error) {
	if err != nil {
		s.binding().ErrorReporter()(errors.NewErr(tagEditableTextClipboardFailed, err).SetMetadata("context", what))
	}
}

func (s *EditableTextState) allowPaste() bool {
	return !s.Widget().o.ReadOnly && s.value().Selection.IsValid()
}

// PasteText pastes the text of the clipboard over the selection. The clipboard
// is read asynchronously.
//
// Flutter: widgets/editable_text.dart EditableTextState.pasteText
func (s *EditableTextState) PasteText(cause services.SelectionChangedCause) {
	if !s.allowPaste() {
		return
	}
	ctx, cancel := context.WithTimeout(s.Lifetime(), 5*time.Second)
	s.binding().Clipboard().GetData(ctx, services.ClipboardKTextPlain, func(data *services.ClipboardData, err error) {
		cancel()
		if err != nil {
			s.binding().ErrorReporter()(errors.NewErr(tagEditableTextPasteFailed, err))
			return
		}
		if data == nil || !s.Mounted() {
			return
		}
		s.pasteText(cause, data.Text)
	})
}

func (s *EditableTextState) pasteText(cause services.SelectionChangedCause, text string) {
	if !s.allowPaste() {
		return
	}
	// After the paste, the caret is collapsed and located after the pasted
	// content.
	v := s.value()
	selection := v.Selection
	last := max(selection.BaseOffset, selection.ExtentOffset)
	collapsed := v.CopyWith(services.TextEditingValueCopy{Selection: foundation.Some(services.NewTextSelectionCollapsed(last))})
	s.UserUpdateTextEditingValue(collapsed.Replaced(selection.Range(), text), cause)
	if cause == services.SelectionChangedCauseToolbar {
		s.binding().RunPostFrame(func(time.Duration) {
			if s.Mounted() {
				s.BringIntoView(s.value().Selection.Extent())
			}
		})
		s.HideToolbar(true)
	}
}

// SelectAll selects all the text.
//
// Flutter: widgets/editable_text.dart EditableTextState.selectAll
func (s *EditableTextState) SelectAll(cause services.SelectionChangedCause) {
	w := s.Widget()
	if w.o.ReadOnly && w.o.ObscureText {
		// If we can't modify it, and we can't copy it, there's no point in
		// selecting it.
		return
	}
	v := s.value()
	s.UserUpdateTextEditingValue(v.CopyWith(services.TextEditingValueCopy{
		Selection: foundation.Some(services.NewTextSelection(0, len(v.Text))),
	}), cause)
	if cause == services.SelectionChangedCauseToolbar {
		switch foundation.DefaultTargetPlatform() {
		case foundation.TargetPlatformMacOS, foundation.TargetPlatformLinux, foundation.TargetPlatformWindows:
			s.HideToolbar(true)
		}
		switch foundation.DefaultTargetPlatform() {
		case foundation.TargetPlatformAndroid, foundation.TargetPlatformFuchsia, foundation.TargetPlatformLinux, foundation.TargetPlatformWindows:
			s.BringIntoView(s.value().Selection.Extent())
		}
	}
}

// TextEditingValue is the value of the text, selection and composing range.
//
// Flutter: widgets/editable_text.dart EditableTextState.textEditingValue
func (s *EditableTextState) TextEditingValue() services.TextEditingValue { return s.value() }

// textEditingValueForTextLayoutMetrics is the value of the render object, which
// is the one of the last build.
//
// Flutter: widgets/editable_text.dart EditableTextState._textEditingValueforTextLayoutMetrics
func (s *EditableTextState) textEditingValueForTextLayoutMetrics() services.TextEditingValue {
	bc := s.editableKey.CurrentContext()
	if bc != nil {
		if w, ok := bc.Widget().(*editableRender); ok {
			return w.value
		}
	}
	panic(errors.New(tagEditableTextNotMounted, "the editable must be mounted"))
}

// InitState starts to listen to the controller and the focus node.
//
// Flutter: widgets/editable_text.dart EditableTextState.initState
func (s *EditableTextState) InitState() {
	w := s.Widget()
	b := s.binding()
	if foundation.IsWeb {
		s.clipboardStatus = NewWebClipboardStatusNotifier(b)
	} else {
		s.clipboardStatus = NewClipboardStatusNotifier(b)
	}
	s.clipboardHandle = s.clipboardStatus.AddListener(s.onChangedClipboardStatus)
	s.controllerHandle = w.controller.AddListener(s.didChangeTextEditingValue)
	s.focusHandle = w.focusNode.AddListener(s.handleFocusChanged)
	s.cursorVisibility.SetValue(w.showCursor)
	s.spellCheckConfiguration = s.inferSpellCheckConfiguration(w.o.SpellCheckConfiguration, w.o.ObscureText, w.keyboardType, w.o.AutofillHints)
	s.verticalSelectionMover = newUpdateTextSelectionVerticallyAction(s)
	s.actions = s.buildActions()
	b.AddObserver(s)
}

// DidChangeAppLifecycleState remembers that the app resumed, so that the field
// doesn't select all its text when it gets the focus back.
//
// Flutter: widgets/editable_text.dart EditableTextState._onResume
func (s *EditableTextState) DidChangeAppLifecycleState(state ui.AppLifecycleState) {
	if state != ui.AppLifecycleStateResumed {
		return
	}
	s.justResumed = true
	// To prevent adding multiple listeners, remove any existing one first.
	s.removeFocusManagerReset()
	m := s.binding().FocusManager()
	s.focusManagerHandles.reset = m.AddListener(s.resetJustResumed)
	s.focusManagerHandles.hasR = true
}

func (s *EditableTextState) removeFocusManagerReset() {
	if s.focusManagerHandles.hasR {
		s.binding().FocusManager().RemoveListener(s.focusManagerHandles.reset)
		s.focusManagerHandles.hasR = false
	}
}

func (s *EditableTextState) resetJustResumed() {
	s.justResumed = false
	s.removeFocusManagerReset()
}

// DidChangeDependencies updates the style and the focus on build, and restarts
// the blinking when the tickers are turned on or off.
//
// Flutter: widgets/editable_text.dart EditableTextState.didChangeDependencies
func (s *EditableTextState) DidChangeDependencies() {
	w := s.Widget()
	s.style = w.style
	if MediaQueryBoldTextOf(s.Context()) {
		s.style = w.style.Merge(painting.NewTextStyle(painting.TextStyleOpts{FontWeight: ui.FontWeightBold}))
	}
	if !s.didAutoFocus && w.o.Autofocus {
		s.didAutoFocus = true
		s.binding().RunPostFrame(func(time.Duration) {
			if s.Mounted() && s.renderEditableHasSize() {
				s.flagInternalFocus()
				FocusScopeOf(s.Context()).Autofocus(w.focusNode)
			}
		})
	}
	// Restart or stop the blinking caret when the TickerMode changes.
	newTickerEnabled := TickerModeValuesOf(s.Context()).Enabled
	if s.tickersEnabled != newTickerEnabled {
		s.tickersEnabled = newTickerEnabled
		if s.showBlinkingCursor() {
			s.startCursorBlink()
		} else if !s.tickersEnabled && s.cursorTimer != nil {
			s.stopCursorBlink(true)
		}
	}
	if s.hasInputConnection() {
		s.textInputConnection.UpdateConfig(s.TextInputConfiguration())
		// The style may have changed due to dependency changes.
		s.binding().RunPostFrame(func(time.Duration) {
			if !s.Mounted() || !s.hasInputConnection() {
				return
			}
			s.textInputConnection.UpdateStyle(s.textInputStyle())
		})
	}
}

func (s *EditableTextState) renderEditableHasSize() bool {
	r := s.RenderEditable()
	return r != nil && r.HasSize()
}

// DidUpdateWidget moves the listeners and updates the input method for the new
// widget.
//
// Flutter: widgets/editable_text.dart EditableTextState.didUpdateWidget
func (s *EditableTextState) DidUpdateWidget(old *EditableText) {
	w := s.Widget()
	if w.controller != old.controller {
		old.controller.RemoveListener(s.controllerHandle)
		s.controllerHandle = w.controller.AddListener(s.didChangeTextEditingValue)
		s.updateRemoteEditingValueIfNeeded()
	}
	s.updateSelectionOverlayAfterWidgetChange(old)
	if w.focusNode != old.focusNode {
		old.focusNode.RemoveListener(s.focusHandle)
		s.focusHandle = w.focusNode.AddListener(s.handleFocusChanged)
		s.UpdateKeepAlive()
	}
	if !s.shouldCreateInputConnection() {
		s.closeInputConnectionIfNeeded()
	} else if old.o.ReadOnly && s.hasFocus() {
		// openInputConnection must be called after layout information is available.
		s.binding().RunPostFrame(func(time.Duration) { s.openInputConnection() })
	}
	if s.hasInputConnection() && (old.o.ObscureText != w.o.ObscureText || old.keyboardType != w.keyboardType) {
		s.textInputConnection.UpdateConfig(s.TextInputConfiguration())
	}
	if !old.o.SpellCheckConfiguration.Equal(w.o.SpellCheckConfiguration) || old.o.ObscureText != w.o.ObscureText ||
		old.keyboardType != w.keyboardType || !slices.Equal(old.o.AutofillHints, w.o.AutofillHints) {
		s.spellCheckConfiguration = s.inferSpellCheckConfiguration(w.o.SpellCheckConfiguration, w.o.ObscureText, w.keyboardType, w.o.AutofillHints)
		if s.SpellCheckEnabled() {
			if text := s.value().Text; text != "" {
				s.performSpellCheck(text)
			}
		} else {
			s.spellCheckResults = nil
		}
	}
	if w.style != old.style {
		// The connection picks up the new style when it attaches in
		// openInputConnection.
		s.style = w.style
		if MediaQueryBoldTextOf(s.Context()) {
			s.style = w.style.Merge(painting.NewTextStyle(painting.TextStyleOpts{FontWeight: ui.FontWeightBold}))
		}
		if s.hasInputConnection() {
			// Schedule the style update after layout to ensure the preferred
			// line height is computed with the new style.
			s.binding().RunPostFrame(func(time.Duration) {
				if !s.Mounted() || !s.hasInputConnection() {
					return
				}
				s.textInputConnection.UpdateStyle(s.textInputStyle())
			})
		}
	}
	if w.showCursor != old.showCursor {
		s.startOrStopCursorTimerIfNeeded()
	}
	if w.SelectionEnabled() && s.PasteEnabled() {
		s.clipboardStatus.Update()
	}
}

func (s *EditableTextState) textInputStyle() services.TextInputStyle {
	st := s.style
	var family foundation.Opt[string]
	if st.FontFamily != "" {
		family = foundation.Some(st.FontFamily)
	}
	return services.NewTextInputStyle(s.textDirection(), s.Widget().o.TextAlign, services.TextInputStyleOpts{
		FontFamily: family, FontSize: st.FontSize, FontWeight: st.FontWeight, LetterSpacing: st.LetterSpacing, WordSpacing: st.WordSpacing,
		LineHeight: foundation.Some(s.renderEditable().PreferredLineHeight()),
	})
}

// Dispose closes the input connection and stops the blinking.
//
// Flutter: widgets/editable_text.dart EditableTextState.dispose
func (s *EditableTextState) Dispose() {
	w := s.Widget()
	if c := s.internalScrollCtl; c != nil {
		c.Dispose()
	}
	w.controller.RemoveListener(s.controllerHandle)
	s.closeInputConnectionIfNeeded()
	if s.cursorTimer != nil {
		s.cursorTimer.cancel()
		s.cursorTimer = nil
	}
	if s.cursorBlinkOpacity != nil {
		s.cursorBlinkOpacity.RemoveListener(s.cursorTickHandle)
		s.cursorBlinkOpacity.Dispose()
		s.cursorBlinkOpacity = nil
	}
	if s.selectionOverlay != nil {
		s.selectionOverlay.Dispose()
		s.selectionOverlay = nil
	}
	w.focusNode.RemoveListener(s.focusHandle)
	b := s.binding()
	b.RemoveObserver(s)
	s.disposeScrollNotificationObserver()
	s.clipboardStatus.RemoveListener(s.clipboardHandle)
	s.clipboardStatus.Dispose()
	s.cursorVisibility.Dispose()
	if s.focusManagerHandles.hasU {
		b.FocusManager().RemoveListener(s.focusManagerHandles.unflag)
	}
	s.removeFocusManagerReset()
}

// TextInputConfiguration is the configuration of the input method for the field.
//
// Flutter: widgets/editable_text.dart EditableTextState.textInputConfiguration
func (s *EditableTextState) TextInputConfiguration() services.TextInputConfiguration {
	w := s.Widget()
	autofill := foundation.None[services.AutofillConfiguration]()
	if len(w.o.AutofillHints) > 0 {
		autofill = foundation.Some(services.NewAutofillConfiguration(s.AutofillID(), slicesClone(w.o.AutofillHints), s.value()))
	}
	action := w.o.TextInputAction
	if action == services.TextInputActionUnspecified {
		action = services.TextInputActionDone
		if w.keyboardType == services.TextInputTypeMultiline {
			action = services.TextInputActionNewline
		}
	}
	var mimeTypes []string
	return services.NewTextInputConfiguration(services.TextInputConfigurationOpts{
		ViewID:                      foundation.Some(int(ViewOf(s.Context()).ViewID())),
		InputType:                   w.keyboardType,
		ReadOnly:                    w.o.ReadOnly,
		ObscureText:                 w.o.ObscureText,
		DisableAutocorrect:          !w.autocorrect,
		SmartDashesType:             w.smartDashesType,
		SmartQuotesType:             w.smartQuotesType,
		DisableSuggestions:          w.o.DisableSuggestions,
		DisableInteractiveSelection: !w.userSelectionEnabled(),
		InputAction:                 action,
		TextCapitalization:          w.o.TextCapitalization,
		KeyboardAppearance:          w.o.KeyboardAppearance,
		AutofillConfiguration:       autofill, DisableIMEPersonalizedLearning: w.o.DisableIMEPersonalizedLearning,
		AllowedMimeTypes: mimeTypes, HintLocales: w.o.HintLocales, EnableInlinePrediction: w.o.EnableInlinePrediction,
	})
}

// AutofillID is the identifier of the field for autofill.
//
// Flutter: widgets/editable_text.dart EditableTextState.autofillId
func (s *EditableTextState) AutofillID() string { return "EditableText-" + foundation.ShortHash(s) }

// Autofill sets the value from an autofill.
//
// Flutter: widgets/editable_text.dart EditableTextState.autofill
func (s *EditableTextState) Autofill(value services.TextEditingValue) { s.UpdateEditingValue(value) }

// BuildTextSpan builds the span that the field shows: the text, with the
// composing range underlined.
//
// Flutter: widgets/editable_text.dart EditableTextState.buildTextSpan
func (s *EditableTextState) BuildTextSpan() *painting.TextSpan {
	w := s.Widget()
	v := s.value()
	if w.o.ObscureText {
		text := strings.Repeat(w.o.ObscuringCharacter, foundation.GraphemeCount(v.Text))
		return painting.NewTextSpan(painting.TextSpanOpts{Style: s.style, Text: foundation.Some(text)})
	}
	withComposing := !w.o.ReadOnly && s.hasFocus()
	if s.spellCheckResultsReceived() {
		// If the composing range is out of range for the current text, ignore it to
		// preserve the tree integrity, otherwise in release mode a range panic
		// would be thrown and this EditableText would be built with a broken subtree.
		composingRegionOutOfRange := !v.IsComposingRangeValid() || !withComposing
		return BuildTextSpanWithSpellCheckSuggestions(v, composingRegionOutOfRange, s.style,
			s.spellCheckConfiguration.MisspelledTextStyle(), *s.spellCheckResults)
	}
	// Read only mode should not paint text composing.
	return w.controller.BuildTextSpan(s.Context(), s.style, withComposing)
}

func slicesClone(in []string) []string { return append([]string(nil), in...) }

// Build builds the field.
//
// Flutter: widgets/editable_text.dart EditableTextState.build
func (s *EditableTextState) Build(bc BuildContext) Widget {
	s.KeepAliveBuild()
	w := s.Widget()
	textScaler := w.o.TextScaler
	if textScaler == nil {
		textScaler = MediaQueryTextScalerOf(bc)
	}
	inputType := ui.SemanticsInputTypeText
	switch w.keyboardType {
	case services.TextInputTypePhone:
		inputType = ui.SemanticsInputTypePhone
	case services.TextInputTypeURL:
		inputType = ui.SemanticsInputTypeURL
	case services.TextInputTypeEmailAddress:
		inputType = ui.SemanticsInputTypeEmail
	}
	groupID := w.o.GroupID
	if groupID == nil {
		groupID = EditableTextGroupID
	}
	cursor := w.o.MouseCursor
	if cursor == nil {
		cursor = services.SystemMouseCursors.Text
	}
	return newCompositionCallback(s.compositeCallback, s.hasInputConnection(),
		NewActions(s.actions,
			NewBuilder(func(bc BuildContext) Widget {
				var tapOutside TapRegionCallback
				if s.hasFocus() {
					tapOutside = func(event *gestures.PointerDownEvent) { s.onTapOutside(bc, event) }
				}
				return NewTextFieldTapRegion(
					NewMouseRegion(
						NewUndoHistory[services.TextEditingValue](w.controller, func(v services.TextEditingValue) {
							s.UserUpdateTextEditingValue(v, services.SelectionChangedCauseKeyboard)
						}, w.focusNode,
							NewFocus(NewNotificationListener(func(n ScrollNotification) bool {
								s.handleContextMenuOnScroll(n)
								return false
							}, s.buildScrollable(bc, textScaler, inputType)),
								FocusOpts{FocusNode: w.focusNode, ExcludeSemantics: true, DebugLabel: "EditableText"}),
							UndoHistoryOpts[services.TextEditingValue]{
								ShouldChangeUndoStack: s.shouldChangeUndoStack, UndoStackModifier: undoStackModifier, Controller: w.o.UndoController,
							}),
						MouseRegionOpts{Cursor: cursor}),
					TextFieldTapRegionOpts{TapRegionOpts: TapRegionOpts{
						GroupID: groupID, OnTapOutside: tapOutside,
						OnTapUpOutside: func(event *gestures.PointerUpEvent) { s.onTapUpOutside(bc, event) }, DebugLabel: "EditableText",
					}})
			})))
}

// shouldChangeUndoStack says whether a change of the value goes on the undo
// stack: composing text doesn't, and neither does a change of the selection
// alone.
//
// Flutter: widgets/editable_text.dart EditableTextState.build
func (s *EditableTextState) shouldChangeUndoStack(oldValue services.TextEditingValue, hasOld bool, newValue services.TextEditingValue) bool {
	if !newValue.Selection.IsValid() {
		return false
	}
	if !hasOld {
		return true
	}
	if foundation.DefaultTargetPlatform() != foundation.TargetPlatformAndroid {
		// Composing text is not counted in history coalescing.
		if !s.Widget().controller.Value().Composing.IsCollapsed() {
			return false
		}
	}
	return oldValue.Text != newValue.Text || oldValue.Composing != newValue.Composing
}

// undoStackModifier discards the composing region when pushing a new entry to
// the undo stack, on Android.
//
// Flutter: widgets/editable_text.dart EditableTextState.build
func undoStackModifier(value services.TextEditingValue) services.TextEditingValue {
	if foundation.DefaultTargetPlatform() == foundation.TargetPlatformAndroid {
		return value.CopyWith(services.TextEditingValueCopy{Composing: foundation.Some(ui.TextRangeEmpty)})
	}
	return value
}

func (s *EditableTextState) buildScrollable(bc BuildContext, textScaler painting.TextScaler, inputType ui.SemanticsInputType) Widget {
	w := s.Widget()
	multiline := s.isMultiline()
	axis := painting.AxisDirectionRight
	if multiline {
		axis = painting.AxisDirectionDown
	}
	behavior := w.o.ScrollBehavior
	if behavior == nil {
		behavior = ScrollConfigurationOf(bc).CopyWith(ScrollBehaviorCopy{Scrollbars: foundation.Some(multiline), Overscroll: foundation.Some(false)})
	}
	return NewScrollable(func(bc BuildContext, offset rendering.ViewportOffset) Widget {
		return NewCompositedTransformTarget(s.toolbarLayerLink,
			NewSemantics(s.buildEditable(bc, offset, textScaler), SemanticsOpts{
				InputType: foundation.Some(inputType), OnCopy: s.semanticsOnCopy(), OnCut: s.semanticsOnCut(), OnPaste: s.semanticsOnPaste(),
			}))
	}, ScrollableOpts{
		Key: s.scrollableKeyForWidget(), ExcludeFromSemantics: true, AxisDirection: axis, Controller: s.scrollController(),
		Physics: w.o.ScrollPhysics, DragStartBehavior: w.o.DragStartBehavior, ScrollBehavior: behavior,
	})
}

func (s *EditableTextState) scrollableKeyForWidget() foundation.Key { return s.scrollableKey }

func (s *EditableTextState) buildEditable(bc BuildContext, offset rendering.ViewportOffset, textScaler painting.TextScaler) Widget {
	w := s.Widget()
	selectionColor := w.o.SelectionColor
	if s.selectionOverlay != nil && s.selectionOverlay.SpellCheckToolbarIsVisible() {
		if c := s.spellCheckConfiguration.MisspelledSelectionColor(); c.IsSome() {
			selectionColor = c
		}
	}
	cursorOffset := w.o.CursorOffset.OrElse(ui.OffsetZero)
	var heightBehavior = w.o.TextHeightBehavior
	if heightBehavior == nil {
		heightBehavior = DefaultTextHeightBehaviorMaybeOf(bc)
	}
	basis := w.o.TextWidthBasis
	if basis == painting.TextWidthBasisUnspecified {
		basis = painting.TextWidthBasisParent
	}
	dpr := 1.0
	if r, ok := MediaQueryMaybeDevicePixelRatioOf(bc); ok {
		dpr = r
	}
	clip := w.o.ClipBehavior
	if clip == ui.ClipUnspecified {
		clip = ui.ClipHardEdge
	}
	e := &editableRender{
		inlineSpan: s.BuildTextSpan(), value: s.value(), cursorColor: foundation.Some(s.cursorColor()),
		startHandleLayerLink: s.startHandleLink, endHandleLayerLink: s.endHandleLink,
		backgroundCursorColor: foundation.Some(w.backgroundCursorColor), showCursor: s.cursorVisibility,
		forceLine: !w.o.DisableForceLine, readOnly: w.o.ReadOnly, hasFocus: s.hasFocus(), maxLines: w.o.MaxLines, minLines: w.o.MinLines,
		expands: w.o.Expands, strutStyle: w.strutStyle, selectionColor: selectionColor, textScaler: textScaler, textAlign: w.o.TextAlign,
		textDirection: s.textDirection(), locale: w.o.Locale, obscuringCharacter: w.o.ObscuringCharacter, obscureText: w.o.ObscureText,
		textHeightBehavior: heightBehavior, textWidthBasis: basis, offset: offset, rendererIgnoresPointer: w.o.RendererIgnoresPointer,
		cursorWidth: w.o.CursorWidth.OrElse(2.0), cursorHeight: w.o.CursorHeight, cursorRadius: w.o.CursorRadius, cursorOffset: cursorOffset,
		paintCursorAboveText: w.o.PaintCursorAboveText, selectionHeightStyle: w.selectionHeightStyle, selectionWidthStyle: w.selectionWidthStyle,
		enableInteractiveSel: w.userSelectionEnabled(), textSelectionDelegate: s, devicePixelRatio: dpr,
		promptRectRange: s.currentPromptRectRange, promptRectColor: w.o.AutocorrectionTextRectColor, clipBehavior: clip,
	}
	return newEditableRender(s.editableKey, e)
}

func (s *EditableTextState) semanticsOnCopy() func() {
	if s.Widget().SelectionEnabled() && s.hasFocus() && s.CopyEnabled() {
		return func() { s.CopySelection(services.SelectionChangedCauseToolbar) }
	}
	return nil
}

func (s *EditableTextState) semanticsOnCut() func() {
	if s.Widget().SelectionEnabled() && s.hasFocus() && s.CutEnabled() {
		return func() { s.CutSelection(services.SelectionChangedCauseToolbar) }
	}
	return nil
}

func (s *EditableTextState) semanticsOnPaste() func() {
	if s.Widget().SelectionEnabled() && s.hasFocus() && s.PasteEnabled() && s.clipboardStatus.Value() == ClipboardStatusPasteable {
		return func() { s.PasteText(services.SelectionChangedCauseToolbar) }
	}
	return nil
}

// DidChangeMetrics updates the selection overlay and shows the caret when the
// keyboard shows up.
//
// Flutter: widgets/editable_text.dart EditableTextState.didChangeMetrics
func (s *EditableTextState) DidChangeMetrics() {
	if !s.Mounted() {
		return
	}
	// The view is read, not the MediaQuery: when the metrics change, the MediaQuery
	// is not built with the new insets yet.
	insets := ViewOf(s.Context()).ViewInsets()
	if s.lastBottomViewInset != insets.Bottom {
		s.binding().RunPostFrame(func(time.Duration) {
			if s.selectionOverlay != nil {
				s.selectionOverlay.UpdateForScroll()
			}
		})
		if s.lastBottomViewInset < insets.Bottom {
			// Because the metrics change signal from the engine comes here every
			// frame, there is no need to show the caret with animation.
			s.scheduleShowCaretOnScreen(false)
		}
	}
	s.lastBottomViewInset = insets.Bottom
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagEditableTextClipboardFailed = "EditableTextClipboardFailed"
	tagEditableTextNotBuilt        = "EditableTextNotBuilt"
	tagEditableTextNotMounted      = "EditableTextNotMounted"
	tagEditableTextPasteFailed     = "EditableTextPasteFailed"
)
