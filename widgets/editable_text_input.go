package widgets

import (
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// onCursorColorTick sets the color of the caret from the blink, and whether it is
// shown.
//
// Flutter: widgets/editable_text.dart EditableTextState._onCursorColorTick
func (s *EditableTextState) onCursorColorTick() {
	w := s.Widget()
	if r := s.RenderEditable(); r != nil {
		r.SetCursorColor(foundation.Some(s.cursorColor()))
	}
	s.cursorVisibility.SetValue(w.showCursor && (s.deterministicCursor() || s.blinkController().Value() > 0))
}

// deterministicCursor says whether DebugOptions.DeterministicCursor is on.
func (s *EditableTextState) deterministicCursor() bool {
	return foundation.DebugMode && s.binding().DebugOptions().DeterministicCursor
}

func (s *EditableTextState) showBlinkingCursor() bool {
	return s.hasFocus() && s.value().Selection.IsCollapsed() && s.Widget().showCursor && s.tickersEnabled && true
}

// CursorCurrentlyVisible is whether the caret is on at the moment of the blink.
//
// Flutter: widgets/editable_text.dart EditableTextState.cursorCurrentlyVisible
func (s *EditableTextState) CursorCurrentlyVisible() bool { return s.blinkController().Value() > 0 }

// CursorBlinkInterval is the time the caret is on, and then off.
//
// Flutter: widgets/editable_text.dart EditableTextState.cursorBlinkInterval
func (s *EditableTextState) CursorBlinkInterval() time.Duration { return cursorBlinkHalfPeriod }

func (s *EditableTextState) startCursorBlink() {
	if !s.Widget().showCursor || !s.tickersEnabled {
		return
	}
	if s.cursorTimer != nil {
		s.cursorTimer.cancel()
	}
	s.blinkController().SetValue(1.0)
	if s.deterministicCursor() {
		return
	}
	s.cursorTimer = newPeriodicTimer(s.binding().Gestures(), cursorBlinkHalfPeriod, s.onCursorTick)
}

func (s *EditableTextState) onCursorTick() {
	if !s.cursorTimer.isActive() && s.tickersEnabled {
		s.cursorTimer = newPeriodicTimer(s.binding().Gestures(), cursorBlinkHalfPeriod, s.onCursorTick)
	}
	if s.blinkController().Value() == 0 {
		s.blinkController().SetValue(1)
	} else {
		s.blinkController().SetValue(0)
	}
}

func (s *EditableTextState) stopCursorBlink(resetCharTicks bool) {
	// If the caret is animating, stop the animation, and we always want the
	// caret to be visible when the floating cursor is enabled.
	s.blinkController().SetValue(0)
	if s.cursorTimer != nil {
		s.cursorTimer.cancel()
		s.cursorTimer = nil
	}
}

func (s *EditableTextState) startOrStopCursorTimerIfNeeded() {
	if !s.showBlinkingCursor() {
		s.stopCursorBlink(true)
	} else if s.cursorTimer == nil {
		s.startCursorBlink()
	}
}

func (s *EditableTextState) didChangeTextEditingValue() {
	w := s.Widget()
	if s.hasFocus() && !s.value().Selection.IsValid() {
		// If this field is focused and the selection is invalid, place the caret at
		// the end. Does not rely on handleFocusChanged because it makes selection
		// handles visible on Android. Unregister as a listener to the text
		// controller while making the change.
		w.controller.RemoveListener(s.controllerHandle)
		sel, _ := s.adjustedSelectionWhenFocused()
		w.controller.SetSelection(sel)
		s.controllerHandle = w.controller.AddListener(s.didChangeTextEditingValue)
	}
	s.updateRemoteEditingValueIfNeeded()
	s.startOrStopCursorTimerIfNeeded()
	s.updateOrDisposeSelectionOverlayIfNeeded()
	s.SetState(func() {})
	s.verticalSelectionMover.stopCurrentVerticalRunIfSelectionChanges()
}

func (s *EditableTextState) handleFocusChanged() {
	s.openOrCloseInputConnectionIfNeeded()
	s.startOrStopCursorTimerIfNeeded()
	if s.hasFocus() {
		// Listen for changing view insets, which indicates the keyboard showing up.
		insets, _ := MediaQueryMaybeViewInsetsOf(s.Context())
		s.lastBottomViewInset = insets.Bottom
		if !s.Widget().o.ReadOnly {
			s.scheduleShowCaretOnScreen(true)
		}
		if sel, ok := s.adjustedSelectionWhenFocused(); ok {
			s.handleSelectionChanged(sel, services.SelectionChangedCauseUnspecified)
		}
	} else {
		s.SetState(func() { s.currentPromptRectRange = foundation.None[ui.TextRange]() })
	}
	s.UpdateKeepAlive()
}

// adjustedSelectionWhenFocused is the selection to set when the field gets the
// focus: all the text for a single line field (as the desktop ones do), or the
// end of the text when there is no selection.
//
// Flutter: widgets/editable_text.dart EditableTextState._adjustedSelectionWhenFocused
func (s *EditableTextState) adjustedSelectionWhenFocused() (services.TextSelection, bool) {
	w := s.Widget()
	v := s.value()
	shouldSelectAll := w.selectAllOnFocus && w.SelectionEnabled() && !s.isMultiline() && !s.nextFocusChangeIntern && !s.justResumed
	s.justResumed = false
	switch {
	case shouldSelectAll:
		return services.NewTextSelection(0, len(v.Text)), true
	case !v.Selection.IsValid():
		return services.NewTextSelectionCollapsed(len(v.Text)), true
	}
	return services.TextSelection{}, false
}

// flagInternalFocus marks the next focus change as one that originates within
// the field, so that it is not treated as one from outside.
//
// Flutter: widgets/editable_text.dart EditableTextState._flagInternalFocus
func (s *EditableTextState) flagInternalFocus() {
	s.nextFocusChangeIntern = true
	m := s.binding().FocusManager()
	if s.focusManagerHandles.hasU {
		m.RemoveListener(s.focusManagerHandles.unflag)
	}
	s.focusManagerHandles.unflag = m.AddListener(s.unflagInternalFocus)
	s.focusManagerHandles.hasU = true
}

func (s *EditableTextState) unflagInternalFocus() {
	s.nextFocusChangeIntern = false
	if s.focusManagerHandles.hasU {
		s.binding().FocusManager().RemoveListener(s.focusManagerHandles.unflag)
		s.focusManagerHandles.hasU = false
	}
}

// compositeCallback tells the input method the size and the transform of the
// field when its layer is composited.
//
// Flutter: widgets/editable_text.dart EditableTextState._compositeCallback
func (s *EditableTextState) compositeCallback(rendering.Layer) {
	r := s.RenderEditable()
	// The callback can be invoked when the layer is detached. The input connection
	// can be closed by the platform in which case this widget doesn't rebuild.
	if r == nil || !r.Attached() || !s.hasInputConnection() {
		return
	}
	s.updateSizeAndTransform()
}

// updateSizeAndTransform must be called after layout.
//
// Flutter: widgets/editable_text.dart EditableTextState._updateSizeAndTransform
func (s *EditableTextState) updateSizeAndTransform() {
	r := s.renderEditable()
	s.textInputConnection.SetEditableSizeAndTransform(r.Size(), r.GetTransformTo(nil))
}

func (s *EditableTextState) schedulePeriodicPostFrameCallbacks() {
	// The field can be out of the tree by the time the callback runs.
	if !s.hasInputConnection() || s.RenderEditable() == nil {
		return
	}
	s.updateComposingRectIfNeeded()
	s.updateCaretRectIfNeeded()
	// The callback does not request a frame, or the callbacks would keep the frames coming.
	s.binding().Scheduler().AddPostFrameCallback(func(time.Duration) { s.schedulePeriodicPostFrameCallbacks() })
}

// updateComposingRectIfNeeded sends the composing rect to the input method; the
// rect of the caret when there is no composing text yet.
//
// Flutter: widgets/editable_text.dart EditableTextState._updateComposingRectIfNeeded
func (s *EditableTextState) updateComposingRectIfNeeded() {
	composing := s.value().Composing
	r := s.renderEditable()
	rect, ok := r.GetRectForComposingRange(composing)
	if !ok {
		offset := 0
		if composing.IsValid() {
			offset = composing.Start
		}
		rect = r.GetLocalRectForCaret(ui.NewTextPosition(offset))
	}
	s.textInputConnection.SetComposingRect(rect)
}

// updateCaretRectIfNeeded sends the rect of the caret, at the start of the
// selection, to the input method.
//
// Flutter: widgets/editable_text.dart EditableTextState._updateCaretRectIfNeeded
func (s *EditableTextState) updateCaretRectIfNeeded() {
	r := s.renderEditable()
	sel, ok := r.Selection().Get()
	if !ok || !sel.IsValid() {
		return
	}
	s.textInputConnection.SetCaretRect(r.GetLocalRectForCaret(ui.NewTextPosition(sel.Start())))
}

// CurrentTextEditingValue is the value for the input method.
//
// Flutter: widgets/editable_text.dart EditableTextState.currentTextEditingValue
func (s *EditableTextState) CurrentTextEditingValue() (services.TextEditingValue, bool) {
	return s.value(), true
}

// CurrentAutofillScope is the autofill group of the field, which is always nil.
//
// Flutter: widgets/editable_text.dart EditableTextState.currentAutofillScope
func (s *EditableTextState) CurrentAutofillScope() services.AutofillScope { return nil }

// UpdateEditingValue takes the value that the input method sends.
//
// Flutter: widgets/editable_text.dart EditableTextState.updateEditingValue
func (s *EditableTextState) UpdateEditingValue(value services.TextEditingValue) {
	// This method handles text editing state updates from the platform text input
	// plugin. The field may not have the focus or an open input connection, as
	// autofill can update a disconnected field. Since we still have to support
	// keyboard select, this is the best place to disable text updating.
	if !s.shouldCreateInputConnection() {
		return
	}
	cur := s.value()
	if s.checkNeedsAdjustAffinity(value) {
		sel := value.Selection
		sel.Affinity = cur.Selection.Affinity
		value = value.CopyWith(services.TextEditingValueCopy{Selection: foundation.Some(sel)})
	}
	if s.Widget().o.ReadOnly {
		// In the read-only case, we only care about selection changes, and reject
		// everything else.
		value = cur.CopyWith(services.TextEditingValueCopy{Selection: foundation.Some(value.Selection)})
	}
	s.lastKnownRemoteValue = foundation.Some(value)
	if value == cur {
		// This is possible, for example, when the numeric keyboard is input, the
		// engine will notify twice for the same value.
		return
	}
	if value.Text == cur.Text && value.Composing == cur.Composing {
		// `selection` is the only change.
		s.handleSelectionChanged(value.Selection, services.SelectionChangedCauseKeyboard)
	} else {
		if value.Text != cur.Text {
			// Hide the toolbar if the text was changed, but only hide the toolbar
			// overlay; the selection handle's visibility will be handled by
			// handleSelectionChanged.
			s.HideToolbar(false)
		}
		s.currentPromptRectRange = foundation.None[ui.TextRange]()
		s.formatAndSetValue(value, services.SelectionChangedCauseKeyboard, false)
	}
	if s.showBlinkingCursor() && s.cursorTimer != nil {
		// To keep the caret from blinking while typing, restart the timer here.
		s.stopCursorBlink(false)
		s.startCursorBlink()
	}
	// Wherever the value is changed by the user, schedule a show caret on screen to
	// make sure the user can see the changes they just made. Programmatic changes
	// to the value do not trigger the behavior even if the text field is focused.
	s.scheduleShowCaretOnScreen(true)
}

func (s *EditableTextState) checkNeedsAdjustAffinity(value services.TextEditingValue) bool {
	cur := s.value()
	// Trust the engine affinity if the text changes or selection changes.
	return value.Text == cur.Text && value.Selection.IsCollapsed() == cur.Selection.IsCollapsed() &&
		value.Selection.Start() == cur.Selection.Start() && value.Selection.Affinity != cur.Selection.Affinity
}

// PerformAction does what the action of the input method asks, such as Enter.
//
// Flutter: widgets/editable_text.dart EditableTextState.performAction
func (s *EditableTextState) PerformAction(action services.TextInputAction) {
	switch action {
	case services.TextInputActionNewline:
		// If this is a multiline field, do nothing for a "newline" action; the
		// newline is already inserted. Otherwise, finalize editing.
		if !s.isMultiline() {
			s.finalizeEditing(action, true)
		}
	case services.TextInputActionDone, services.TextInputActionGo, services.TextInputActionNext,
		services.TextInputActionPrevious, services.TextInputActionSearch, services.TextInputActionSend:
		s.finalizeEditing(action, true)
	default:
		// Finalize editing, but don't give up focus because this keyboard action
		// does not imply the user is done inputting information.
		s.finalizeEditing(action, false)
	}
}

// PerformPrivateCommand passes a private command of the input method on.
//
// Flutter: widgets/editable_text.dart EditableTextState.performPrivateCommand
func (s *EditableTextState) PerformPrivateCommand(action string, data map[string]any) {
	if f := s.Widget().o.OnAppPrivateCommand; f != nil {
		f(action, data)
	}
}

// InsertContent is not supported: the field has no content insertion
// configuration.
//
// Flutter: widgets/editable_text.dart EditableTextState.insertContent
func (s *EditableTextState) InsertContent(services.KeyboardInsertedContent) {}

// UpdateFloatingCursor does nothing: the floating cursor is for iOS.
//
// Flutter: widgets/editable_text.dart EditableTextState.updateFloatingCursor
func (s *EditableTextState) UpdateFloatingCursor(services.RawFloatingCursorPoint) {}

// finalizeEditing takes the actions that are necessary now that the user has
// completed editing.
//
// Flutter: widgets/editable_text.dart EditableTextState._finalizeEditing
func (s *EditableTextState) finalizeEditing(action services.TextInputAction, shouldUnfocus bool) {
	w := s.Widget()
	if w.o.OnEditingComplete != nil {
		s.report("EditableTextCallbackPanicked", "while calling onEditingComplete for "+action.String(), w.o.OnEditingComplete)
	} else {
		// Default behavior if the developer did not provide an onEditingComplete
		// callback: finalize editing and remove the focus, or move it to the
		// next/previous field, depending on the action.
		w.controller.ClearComposing()
		if shouldUnfocus {
			switch action {
			case services.TextInputActionNext:
				w.focusNode.NextFocus()
			case services.TextInputActionPrevious:
				w.focusNode.PreviousFocus()
			default:
				w.focusNode.Unfocus()
			}
		}
	}
	if w.o.OnSubmitted == nil {
		return
	}
	// Invoke the optional callback with the user's submitted content.
	text := s.value().Text
	s.report("EditableTextCallbackPanicked", "while calling onSubmitted for "+action.String(), func() { w.o.OnSubmitted(text) })
	// If shouldUnfocus is true, the text field should no longer be focused after
	// the microtask queue is drained. But in case the developer cancelled the
	// focus change in the onSubmitted callback by focusing this input field again,
	// reset the input method.
	if shouldUnfocus {
		s.scheduleRestartConnection()
	}
}

// BeginBatchEdit starts a batch of edits: the input method is told about the
// value after the last one ends.
//
// Flutter: widgets/editable_text.dart EditableTextState.beginBatchEdit
func (s *EditableTextState) BeginBatchEdit() { s.batchEditDepth++ }

// EndBatchEdit ends a batch of edits.
//
// Flutter: widgets/editable_text.dart EditableTextState.endBatchEdit
func (s *EditableTextState) EndBatchEdit() {
	s.batchEditDepth--
	s.updateRemoteEditingValueIfNeeded()
}

func (s *EditableTextState) updateRemoteEditingValueIfNeeded() {
	if s.batchEditDepth > 0 || !s.hasInputConnection() {
		return
	}
	local := s.value()
	if last, ok := s.lastKnownRemoteValue.Get(); ok && last == local {
		return
	}
	s.textInputConnection.SetEditingState(local)
	s.lastKnownRemoteValue = foundation.Some(local)
}

// getOffsetToRevealCaret is the scroll offset closest to the current one that
// fully reveals the caret rect. If the rect is too large to be revealed in the
// render editable, it is centered along the main axis. In a multiline field, the
// rect's height is first extended to the preferred line height.
//
// Flutter: widgets/editable_text.dart EditableTextState._getOffsetToRevealCaret
func (s *EditableTextState) getOffsetToRevealCaret(rect ui.Rect) rendering.RevealedOffset {
	c := s.scrollController()
	position := c.Position()
	if !position.AllowImplicitScrolling() {
		return rendering.RevealedOffset{Offset: c.Offset(), Rect: rect}
	}
	r := s.renderEditable()
	editableSize := r.Size()
	var additional float64
	var unit ui.Offset
	if !s.isMultiline() {
		if rect.Width() >= editableSize.Width {
			// Center rect if it's oversized.
			additional = editableSize.Width/2 - rect.Center().Dx
		} else {
			// Valid additional offsets range from (rect.right - size.width) to
			// (rect.left). Pick the closest one if out of range.
			additional = clampFloat(0, rect.Right-editableSize.Width, rect.Left)
		}
		unit = ui.Offset{Dx: 1}
	} else {
		// The caret is vertically centered within the line. Expand the caret's
		// height so that it spans the line because we're going to ensure that the
		// entire expanded caret is scrolled into view.
		expanded := ui.NewRectFromCenter(rect.Center(), rect.Width(), max(rect.Height(), r.PreferredLineHeight()))
		if expanded.Height() >= editableSize.Height {
			additional = editableSize.Height/2 - expanded.Center().Dy
		} else {
			additional = clampFloat(0, expanded.Bottom-editableSize.Height, expanded.Top)
		}
		unit = ui.Offset{Dy: 1}
	}
	// No overscrolling when encountering tall fonts/scripts that extend past the
	// ascent.
	target := clampFloat(additional+c.Offset(), position.MinScrollExtent(), position.MaxScrollExtent())
	delta := c.Offset() - target
	return rendering.RevealedOffset{Rect: rect.Shift(unit.Mul(delta)), Offset: target}
}

func clampFloat(x, lo, hi float64) float64 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

// openInputConnection must be called after layout.
//
// Flutter: widgets/editable_text.dart EditableTextState._openInputConnection
func (s *EditableTextState) openInputConnection() {
	if !s.shouldCreateInputConnection() {
		return
	}
	if !s.hasInputConnection() {
		local := s.value()
		s.textInputConnection = s.binding().TextInput().Attach(s, s.TextInputConfiguration())
		s.updateSizeAndTransform()
		s.schedulePeriodicPostFrameCallbacks()
		s.textInputConnection.UpdateStyle(s.textInputStyle())
		s.textInputConnection.SetEditingState(local)
		s.textInputConnection.Show()
		if len(s.Widget().o.AutofillHints) > 0 {
			// Request autofill after the size and the transform have been sent to
			// the platform text input plugin.
			s.textInputConnection.RequestAutofill()
		}
		s.lastKnownRemoteValue = foundation.Some(local)
	} else {
		s.textInputConnection.Show()
	}
}

func (s *EditableTextState) closeInputConnectionIfNeeded() {
	if s.hasInputConnection() {
		s.textInputConnection.Close()
		s.textInputConnection = nil
		s.lastKnownRemoteValue = foundation.None[services.TextEditingValue]()
	}
}

func (s *EditableTextState) openOrCloseInputConnectionIfNeeded() {
	if s.hasFocus() && s.Widget().focusNode.ConsumeKeyboardToken() {
		s.openInputConnection()
	} else if !s.hasFocus() {
		s.closeInputConnectionIfNeeded()
		s.Widget().controller.ClearComposing()
	}
}

func (s *EditableTextState) scheduleRestartConnection() {
	if s.restartConnScheduled {
		return
	}
	s.restartConnScheduled = true
	s.binding().Post(s.restartConnectionIfNeeded)
}

// restartConnectionIfNeeded discards the current connection and establishes a
// new one, to reset the input method after a submit that put the focus back.
//
// Flutter: widgets/editable_text.dart EditableTextState._restartConnectionIfNeeded
func (s *EditableTextState) restartConnectionIfNeeded() {
	s.restartConnScheduled = false
	if !s.Mounted() || !s.hasInputConnection() || !s.shouldCreateInputConnection() {
		return
	}
	s.textInputConnection.Close()
	s.textInputConnection = nil
	s.lastKnownRemoteValue = foundation.None[services.TextEditingValue]()
	conn := s.binding().TextInput().Attach(s, s.TextInputConfiguration())
	s.textInputConnection = conn
	conn.Show()
	conn.UpdateStyle(s.textInputStyle())
	conn.SetEditingState(s.value())
	s.lastKnownRemoteValue = foundation.Some(s.value())
}

// DidChangeInputControl shows or hides the new input control.
//
// Flutter: widgets/editable_text.dart EditableTextState.didChangeInputControl
func (s *EditableTextState) DidChangeInputControl(oldControl, newControl services.TextInputControl) {
	if s.hasFocus() && s.hasInputConnection() {
		if oldControl != nil {
			oldControl.Hide()
		}
		if newControl != nil {
			newControl.Show()
		}
	}
}

// OnFocusReceived asks the field for the focus, for an input method that sends
// it.
//
// Flutter: widgets/editable_text.dart EditableTextState.onFocusReceived
func (s *EditableTextState) OnFocusReceived() bool {
	w := s.Widget()
	if s.Mounted() && !s.hasFocus() && w.focusNode.CanRequestFocus() {
		w.focusNode.RequestFocus()
		return true
	}
	return false
}

// ConnectionClosed unfocuses the field when the input method closes the
// connection.
//
// Flutter: widgets/editable_text.dart EditableTextState.connectionClosed
func (s *EditableTextState) ConnectionClosed() {
	if s.hasInputConnection() {
		s.textInputConnection.ConnectionClosedReceived()
		s.textInputConnection = nil
		s.lastKnownRemoteValue = foundation.None[services.TextEditingValue]()
		s.Widget().focusNode.Unfocus()
	}
}

// ShowAutocorrectionPromptRect highlights the range that the input method
// corrects.
//
// Flutter: widgets/editable_text.dart EditableTextState.showAutocorrectionPromptRect
func (s *EditableTextState) ShowAutocorrectionPromptRect(start, end int) {
	s.SetState(func() { s.currentPromptRectRange = foundation.Some(ui.NewTextRange(start, end)) })
}

// InsertTextPlaceholder does nothing: the stylus handwriting is not ported.
//
// Flutter: widgets/editable_text.dart EditableTextState.insertTextPlaceholder
func (s *EditableTextState) InsertTextPlaceholder(ui.Size) {}

// RemoveTextPlaceholder does nothing: the stylus handwriting is not ported.
//
// Flutter: widgets/editable_text.dart EditableTextState.removeTextPlaceholder
func (s *EditableTextState) RemoveTextPlaceholder() {}

// PerformSelector does nothing: the selectors are those of the macOS input
// method.
//
// Flutter: widgets/editable_text.dart EditableTextState.performSelector
func (s *EditableTextState) PerformSelector(string) {}

// ShowToolbar is called by the input method to show the toolbar.
//
// Flutter: widgets/editable_text.dart EditableTextState.showToolbar
func (s *EditableTextState) ShowToolbar() { s.TryShowToolbar() }

// TryShowToolbar shows the toolbar if the field can, and says whether it did.
// Without a selection overlay it never does.
//
// Flutter: widgets/editable_text.dart EditableTextState.showToolbar
func (s *EditableTextState) TryShowToolbar() bool {
	if s.selectionOverlay == nil || s.selectionOverlay.ToolbarIsVisible() {
		return false
	}
	s.clipboardStatus.Update()
	s.selectionOverlay.ShowToolbar()
	// Listen to parent scroll events when the toolbar is visible so it can be
	// hidden during a scroll on supported platforms.
	if platformSupportsFadeOnScroll() {
		s.listeningToScrollObserver = true
		if s.scrollObserver != nil {
			s.scrollObserver.RemoveListener(s.scrollObserverHandle)
		}
		s.scrollObserver = ScrollNotificationObserverMaybeOf(s.Context())
		if s.scrollObserver != nil {
			s.scrollObserverHandle = s.scrollObserver.AddListener(s.handleContextMenuOnParentScroll)
		}
	}
	return true
}

// HideToolbar hides the toolbar, and the handles when hideHandles is set.
//
// Flutter: widgets/editable_text.dart EditableTextState.hideToolbar
func (s *EditableTextState) HideToolbar(hideHandles bool) {
	// Stop listening to parent scroll events when the toolbar is hidden.
	s.disposeScrollNotificationObserver()
	switch {
	case hideHandles:
		// Hide the handles and the toolbar.
		if s.selectionOverlay != nil {
			s.selectionOverlay.Hide()
		}
	case s.selectionOverlay != nil && s.selectionOverlay.ToolbarIsVisible():
		// Hide only the toolbar but not the handles.
		s.selectionOverlay.HideToolbar()
	}
}

// ToggleToolbar shows the toolbar if it is hidden and hides it if it is shown.
//
// Flutter: widgets/editable_text.dart EditableTextState.toggleToolbar
func (s *EditableTextState) ToggleToolbar(hideHandles bool) {
	if s.ensureSelectionOverlay().ToolbarIsVisible() {
		s.HideToolbar(hideHandles)
	} else {
		s.TryShowToolbar()
	}
}

// RequestKeyboard opens the input method, and gives the focus to the field.
//
// Flutter: widgets/editable_text.dart EditableTextState.requestKeyboard
func (s *EditableTextState) RequestKeyboard() {
	if s.hasFocus() {
		s.openInputConnection()
	} else {
		s.flagInternalFocus()
		// This eventually calls openInputConnection also, see handleFocusChanged.
		s.Widget().focusNode.RequestFocus()
	}
}

// handleSelectionChanged sets the selection and tells the widget about it.
//
// Flutter: widgets/editable_text.dart EditableTextState._handleSelectionChanged
func (s *EditableTextState) handleSelectionChanged(selection services.TextSelection, cause services.SelectionChangedCause) {
	w := s.Widget()
	// We return early if the selection is not valid. This can happen when the text
	// of the field is updated at the same time as the selection is changed by a
	// gesture event.
	if n := len(w.controller.Text()); n < selection.End() || n < selection.Start() {
		return
	}
	w.controller.SetSelection(selection)
	// This will show the keyboard for all selection changes on the field except
	// for those triggered by a keyboard input. Typically the field shouldn't take
	// user keyboard input if it's not focused already.
	if cause != services.SelectionChangedCauseKeyboard {
		s.RequestKeyboard()
	}
	s.updateSelectionOverlayAfterSelectionChange()
	s.schedulePrimarySelection()
	if w.o.OnSelectionChanged != nil {
		s.report("EditableTextCallbackPanicked", "while calling onSelectionChanged for "+cause.String(), func() { w.o.OnSelectionChanged(selection, cause) })
	}
	// To keep the caret from blinking while it moves, restart the timer here.
	if s.showBlinkingCursor() && s.cursorTimer != nil {
		s.stopCursorBlink(false)
		s.startCursorBlink()
	}
}

// scheduleShowCaretOnScreen scrolls to the caret after the frame.
//
// Flutter: widgets/editable_text.dart EditableTextState._scheduleShowCaretOnScreen
func (s *EditableTextState) scheduleShowCaretOnScreen(withAnimation bool) {
	if s.showCaretScheduled {
		return
	}
	s.showCaretScheduled = true
	s.binding().RunPostFrame(func(time.Duration) {
		s.showCaretScheduled = false
		// Since we are in a post frame callback, check the context in case the
		// render editable has been disposed.
		r := s.RenderEditable()
		if r == nil || !s.Mounted() {
			return
		}
		sel, ok := r.Selection().Get()
		c := s.scrollController()
		if !ok || !sel.IsValid() || !c.HasClients() {
			return
		}
		// Enlarge the target rect by scrollPadding to ensure that the caret is not
		// positioned directly at the edge after scrolling.
		padding := s.caretPadding(r.PreferredLineHeight())
		caretRect := r.GetLocalRectForCaret(sel.Extent())
		target := s.getOffsetToRevealCaret(caretRect)
		var rectToReveal ui.Rect
		selection := s.value().Selection
		if selection.IsCollapsed() {
			rectToReveal = target.Rect
		} else {
			boxes := r.GetBoxesForSelection(selection)
			// boxes may be empty if, for example, the selection does not encompass
			// a full character, like if it only contained part of an extended
			// grapheme cluster.
			switch {
			case len(boxes) == 0:
				rectToReveal = target.Rect
			case selection.BaseOffset < selection.ExtentOffset:
				rectToReveal = boxes[len(boxes)-1].ToRect()
			default:
				rectToReveal = boxes[0].ToRect()
			}
		}
		if withAnimation {
			c.AnimateTo(target.Offset, caretAnimationDuration, animation.Curves.FastOutSlowIn)
			r.ShowOnScreen(rendering.ShowOnScreenOpts{Rect: foundation.Some(padding.InflateRect(rectToReveal)), Duration: caretAnimationDuration, Curve: animation.Curves.FastOutSlowIn})
		} else {
			c.JumpTo(target.Offset)
			r.ShowOnScreen(rendering.ShowOnScreenOpts{Rect: foundation.Some(padding.InflateRect(rectToReveal))})
		}
	})
}

// userUpdateTextEditingValue
//
// Flutter: widgets/editable_text.dart EditableTextState.userUpdateTextEditingValue

// UserUpdateTextEditingValue sets the value that the user made, with the cause
// of the change. The formatters apply, and the field scrolls to the caret.
//
// Flutter: widgets/editable_text.dart EditableTextState.userUpdateTextEditingValue
func (s *EditableTextState) UserUpdateTextEditingValue(value services.TextEditingValue, cause services.SelectionChangedCause) {
	// Compare the current value with the pre-format new value, in case the
	// formatter would reject the change.
	var shouldShowCaret bool
	if s.Widget().o.ReadOnly {
		shouldShowCaret = !s.value().Selection.Equal(value.Selection)
	} else {
		shouldShowCaret = s.value() != value
	}
	if shouldShowCaret {
		s.scheduleShowCaretOnScreen(true)
	}
	// Even if the value doesn't change, it may be necessary to focus and build the
	// selection overlay. For example, this happens when right clicking an unfocused
	// field that previously had a selection in the same spot.
	if value == s.value() {
		if !s.hasFocus() {
			s.flagInternalFocus()
			s.Widget().focusNode.RequestFocus()
			s.ensureSelectionOverlay()
		}
		return
	}
	s.formatAndSetValue(value, cause, true)
}

// BringIntoView scrolls to show the position.
//
// Flutter: widgets/editable_text.dart EditableTextState.bringIntoView
func (s *EditableTextState) BringIntoView(position ui.TextPosition) {
	r := s.renderEditable()
	local := r.GetLocalRectForCaret(position)
	target := s.getOffsetToRevealCaret(local)
	s.scrollController().JumpTo(target.Offset)
	r.ShowOnScreen(rendering.ShowOnScreenOpts{Rect: foundation.Some(target.Rect)})
}

// formatAndSetValue runs the formatters on the value and sets it, then reports
// the changes of the selection and of the text.
//
// Flutter: widgets/editable_text.dart EditableTextState._formatAndSetValue
func (s *EditableTextState) formatAndSetValue(value services.TextEditingValue, cause services.SelectionChangedCause, userInteraction bool) {
	w := s.Widget()
	old := s.value()
	textChanged := old.Text != value.Text
	textCommitted := !old.Composing.IsCollapsed() && value.Composing.IsCollapsed()
	selectionChanged := !old.Selection.Equal(value.Selection)
	if textChanged || textCommitted {
		// Only apply input formatters if the text has changed (including uncommitted
		// text in the composing region), or when the user committed the composing
		// text.
		s.report("EditableTextFormatterPanicked", "while applying input formatters", func() {
			for _, f := range w.inputFormatters {
				value = f.FormatEditUpdate(old, value)
			}
		})
	}
	oldSelection := s.value().Selection
	// Put all optional user callback invocations in a batch edit to prevent sending
	// multiple updates of the editing value to the input method.
	s.BeginBatchEdit()
	w.controller.SetValue(value)
	// Changes made by the keyboard can sometimes be "out of band" for listening
	// components, so always send those events, even if we didn't think it changed.
	// Also, the user long pressing should always send a selection change as well.
	if selectionChanged || (userInteraction && (cause == services.SelectionChangedCauseLongPress || cause == services.SelectionChangedCauseKeyboard)) {
		s.handleSelectionChanged(s.value().Selection, cause)
		s.bringIntoViewBySelectionState(oldSelection, value.Selection, cause)
	}
	current := s.value().Text
	if old.Text != current && w.o.OnChanged != nil {
		s.report("EditableTextCallbackPanicked", "while calling onChanged", func() { w.o.OnChanged(current) })
	}
	s.EndBatchEdit()
}

func (s *EditableTextState) bringIntoViewBySelectionState(oldSelection, newSelection services.TextSelection, cause services.SelectionChangedCause) {
	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformIOS, foundation.TargetPlatformMacOS:
		if cause == services.SelectionChangedCauseLongPress || cause == services.SelectionChangedCauseDrag {
			s.BringIntoView(newSelection.Extent())
		}
	default:
		if cause == services.SelectionChangedCauseDrag {
			if oldSelection.BaseOffset != newSelection.BaseOffset {
				s.BringIntoView(newSelection.Base())
			} else if oldSelection.ExtentOffset != newSelection.ExtentOffset {
				s.BringIntoView(newSelection.Extent())
			}
		}
	}
}

func (s *EditableTextState) onTapOutside(bc BuildContext, event *gestures.PointerDownEvent) {
	s.hadFocusOnTapDown = true
	if f := s.Widget().o.OnTapOutside; f != nil {
		f(event)
	} else {
		s.defaultOnTapOutside(bc, event)
	}
}

func (s *EditableTextState) onTapUpOutside(bc BuildContext, event *gestures.PointerUpEvent) {
	if !s.hadFocusOnTapDown {
		return
	}
	// Reset so that subsequent events don't trigger the callback based on old
	// information.
	s.hadFocusOnTapDown = false
	if f := s.Widget().o.OnTapUpOutside; f != nil {
		f(event)
	} else {
		ActionsInvoke(bc, EditableTextTapUpOutsideIntent{FocusNode: s.Widget().focusNode, PointerUpEvent: event})
	}
}

// defaultOnTapOutside is what happens when a pointer goes down outside the field
// and no callback is given: the field loses the focus.
//
// Flutter: widgets/editable_text.dart EditableTextState._defaultOnTapOutside
func (s *EditableTextState) defaultOnTapOutside(bc BuildContext, event *gestures.PointerDownEvent) {
	ActionsInvoke(bc, EditableTextTapOutsideIntent{FocusNode: s.Widget().focusNode, PointerDownEvent: event})
}

// replaceText replaces the text of a range of a value.
//
// Flutter: widgets/editable_text.dart EditableTextState._replaceText
func (s *EditableTextState) replaceText(intent ReplaceTextIntent) {
	old := s.value()
	newValue := intent.CurrentTextEditingValue.Replaced(intent.ReplacementRange, intent.ReplacementText)
	s.UserUpdateTextEditingValue(newValue, intent.Cause)
	// If there's no change in text and selection (e.g. when selecting and pasting
	// identical text), the widget won't be rebuilt on value update. Handle this by
	// calling didChangeTextEditingValue so caret and scroll updates can happen.
	if newValue == old {
		s.didChangeTextEditingValue()
	}
}
