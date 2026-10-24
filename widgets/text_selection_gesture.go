package widgets

import (
	"reflect"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// TextSelectionGestureDetectorBuilderDelegate is the interface of a text field
// that a TextSelectionGestureDetectorBuilder works for.
//
// Flutter: widgets/text_selection.dart TextSelectionGestureDetectorBuilderDelegate
type TextSelectionGestureDetectorBuilderDelegate interface {
	// EditableTextKey is the key of the EditableText of the text field.
	//
	// Flutter: widgets/text_selection.dart TextSelectionGestureDetectorBuilderDelegate.editableTextKey
	EditableTextKey() *GlobalKey[*EditableTextState]
	// ForcePressEnabled is whether a force press selects a word. It is for iOS and
	// not ported: the detector has no force press recognizer.
	//
	// Flutter: widgets/text_selection.dart TextSelectionGestureDetectorBuilderDelegate.forcePressEnabled
	ForcePressEnabled() bool
	// SelectionEnabled is whether the user can select text.
	//
	// Flutter: widgets/text_selection.dart TextSelectionGestureDetectorBuilderDelegate.selectionEnabled
	SelectionEnabled() bool
}

// TextSelectionGestureHandler is the set of the callbacks that a text field's
// selection gesture detector builder implements. The base builder implements all
// of them; a builder for a text field embeds it and overrides the ones it needs
// (its self value is what the detector calls).
//
// Flutter: widgets/text_selection.dart TextSelectionGestureDetectorBuilder
type TextSelectionGestureHandler interface {
	OnTapTrackStart()
	OnTapTrackReset()
	OnTapDown(details gestures.TapDragDownDetails)
	OnUserTapAlwaysCalled() bool
	OnUserTap()
	OnSingleTapUp(details gestures.TapDragUpDetails)
	OnSingleTapCancel()
	OnSingleLongTapStart(details gestures.LongPressStartDetails)
	OnSingleLongTapMoveUpdate(details gestures.LongPressMoveUpdateDetails)
	OnSingleLongTapEnd(details gestures.LongPressEndDetails)
	OnSingleLongTapCancel()
	OnSecondaryTap()
	OnSecondaryTapDown(details gestures.TapDownDetails)
	OnDoubleTapDown(details gestures.TapDragDownDetails)
	OnTripleTapDown(details gestures.TapDragDownDetails)
	OnDragSelectionStart(details gestures.TapDragStartDetails)
	OnDragSelectionUpdate(details gestures.TapDragUpdateDetails)
	OnDragSelectionEnd(details gestures.TapDragEndDetails)
}

// TextSelectionGestureDetectorBuilder builds the gesture detector of a text
// field: a tap puts the caret, a double tap selects a word, a triple tap selects
// a line (a paragraph on some platforms), a drag selects a range, and a long
// press selects a word. Embed it in the builder of a text field and call
// InitTextSelectionGestureDetectorBuilder.
//
// Deviation: the floating cursor, the stylus handwriting, the force press and
// the spell check toolbar of the mobile platforms are not ported (SPEC §12.10).
//
// Flutter: widgets/text_selection.dart TextSelectionGestureDetectorBuilder
type TextSelectionGestureDetectorBuilder struct {
	delegate TextSelectionGestureDetectorBuilderDelegate
	self     TextSelectionGestureHandler

	shouldShowSelectionToolbar bool
	shouldShowSelectionHandles bool

	isShiftPressed            bool
	dragStartScrollOffset     float64
	dragStartViewportOffset   float64
	dragStartSelection        foundation.Opt[services.TextSelection]
	longPressStartedWithoutFx bool
}

// InitTextSelectionGestureDetectorBuilder sets up the builder; self is the
// outer value that implements the handler (the builder itself when it isn't
// embedded).
//
// Flutter: widgets/text_selection.dart TextSelectionGestureDetectorBuilder.new
func InitTextSelectionGestureDetectorBuilder(b *TextSelectionGestureDetectorBuilder, self TextSelectionGestureHandler, delegate TextSelectionGestureDetectorBuilderDelegate) {
	b.self, b.delegate = self, delegate
	b.shouldShowSelectionToolbar, b.shouldShowSelectionHandles = true, true
}

// NewTextSelectionGestureDetectorBuilder creates a builder with the default
// behavior.
//
// Flutter: widgets/text_selection.dart TextSelectionGestureDetectorBuilder.new
func NewTextSelectionGestureDetectorBuilder(delegate TextSelectionGestureDetectorBuilderDelegate) *TextSelectionGestureDetectorBuilder {
	b := &TextSelectionGestureDetectorBuilder{}
	InitTextSelectionGestureDetectorBuilder(b, b, delegate)
	return b
}

// ShouldShowSelectionToolbar is whether the last gesture asks for the toolbar.
//
// Flutter: widgets/text_selection.dart TextSelectionGestureDetectorBuilder.shouldShowSelectionToolbar
func (b *TextSelectionGestureDetectorBuilder) ShouldShowSelectionToolbar() bool {
	return b.shouldShowSelectionToolbar
}

// ShouldShowSelectionHandles is whether the last gesture asks for the handles.
//
// Flutter: widgets/text_selection.dart TextSelectionGestureDetectorBuilder.shouldShowSelectionHandles
func (b *TextSelectionGestureDetectorBuilder) ShouldShowSelectionHandles() bool {
	return b.shouldShowSelectionHandles
}

// EditableText is the state of the EditableText.
//
// Flutter: widgets/text_selection.dart TextSelectionGestureDetectorBuilder.editableText
func (b *TextSelectionGestureDetectorBuilder) EditableText() *EditableTextState {
	s, _ := b.delegate.EditableTextKey().CurrentState()
	return s
}

// RenderEditable is the render object of the EditableText.
//
// Flutter: widgets/text_selection.dart TextSelectionGestureDetectorBuilder.renderEditable
func (b *TextSelectionGestureDetectorBuilder) RenderEditable() *rendering.RenderEditable {
	return b.EditableText().renderEditable()
}

func (b *TextSelectionGestureDetectorBuilder) lastSecondaryTapWasOnSelection() bool {
	r := b.RenderEditable()
	pos, ok := r.LastSecondaryTapDownPosition().Get()
	sel, hasSel := r.Selection().Get()
	if !ok || !hasSel {
		return false
	}
	textPosition := r.GetPositionForPoint(pos)
	return sel.Start() <= textPosition.Offset && sel.End() >= textPosition.Offset
}

// expandSelection moves the base or the extent, whichever is closer to the
// offset, to it.
func (b *TextSelectionGestureDetectorBuilder) expandSelection(offset ui.Offset, cause services.SelectionChangedCause, from foundation.Opt[services.TextSelection]) {
	r := b.RenderEditable()
	tapped := r.GetPositionForPoint(offset)
	cur, _ := r.Selection().Get()
	selection := from.OrElse(cur)
	baseIsCloser := abs(tapped.Offset-selection.BaseOffset) < abs(tapped.Offset-selection.ExtentOffset)
	next := selection
	if baseIsCloser {
		next.BaseOffset = selection.ExtentOffset
	}
	next.ExtentOffset = tapped.Offset
	e := b.EditableText()
	e.UserUpdateTextEditingValue(e.TextEditingValue().CopyWith(services.TextEditingValueCopy{Selection: foundation.Some(next)}), cause)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// extendSelection moves the extent to the offset.
func (b *TextSelectionGestureDetectorBuilder) extendSelection(offset ui.Offset, cause services.SelectionChangedCause) {
	r := b.RenderEditable()
	tapped := r.GetPositionForPoint(offset)
	selection, _ := r.Selection().Get()
	selection.ExtentOffset = tapped.Offset
	e := b.EditableText()
	e.UserUpdateTextEditingValue(e.TextEditingValue().CopyWith(services.TextEditingValueCopy{Selection: foundation.Some(selection)}), cause)
}

func (b *TextSelectionGestureDetectorBuilder) scrollPosition() float64 {
	if c := b.delegate.EditableTextKey().CurrentContext(); c != nil {
		if s := ScrollableMaybeOf(c); s != nil {
			return s.Position().Pixels()
		}
	}
	return 0
}

func (b *TextSelectionGestureDetectorBuilder) scrollDirection() painting.AxisDirection {
	if c := b.delegate.EditableTextKey().CurrentContext(); c != nil {
		if s := ScrollableMaybeOf(c); s != nil {
			return s.AxisDirection()
		}
	}
	return painting.AxisDirectionLeft
}

// dragOffsets are how far the viewport of the editable and the scrollable
// outside it scrolled since the start of the drag.
func (b *TextSelectionGestureDetectorBuilder) dragOffsets() (editableOffset, scrollableOffset ui.Offset) {
	r := b.RenderEditable()
	if m, ok := r.MaxLines().Get(); ok && m == 1 {
		editableOffset = ui.Offset{Dx: r.Offset().Pixels() - b.dragStartViewportOffset}
	} else {
		editableOffset = ui.Offset{Dy: r.Offset().Pixels() - b.dragStartViewportOffset}
	}
	if painting.AxisDirectionToAxis(b.scrollDirection()) == painting.AxisHorizontal {
		scrollableOffset = ui.Offset{Dx: b.scrollPosition() - b.dragStartScrollOffset}
	} else {
		scrollableOffset = ui.Offset{Dy: b.scrollPosition() - b.dragStartScrollOffset}
	}
	return editableOffset, scrollableOffset
}

func hasShift(binding *WidgetsBinding) bool {
	keys := binding.Keyboard().LogicalKeysPressed()
	_, l := keys[services.LogicalKeyboardKeyShiftLeft]
	_, r := keys[services.LogicalKeyboardKeyShiftRight]
	return l || r
}

// OnTapTrackStart remembers whether Shift is pressed.
//
// Flutter: widgets/text_selection.dart TextSelectionGestureDetectorBuilder.onTapTrackStart
func (b *TextSelectionGestureDetectorBuilder) OnTapTrackStart() {
	b.isShiftPressed = hasShift(b.EditableText().binding())
}

// OnTapTrackReset forgets that Shift was pressed.
//
// Flutter: widgets/text_selection.dart TextSelectionGestureDetectorBuilder.onTapTrackReset
func (b *TextSelectionGestureDetectorBuilder) OnTapTrackReset() { b.isShiftPressed = false }

// OnTapDown handles a pointer that goes down, which might be a tap.
//
// Flutter: widgets/text_selection.dart TextSelectionGestureDetectorBuilder.onTapDown
func (b *TextSelectionGestureDetectorBuilder) OnTapDown(details gestures.TapDragDownDetails) {
	if !b.delegate.SelectionEnabled() {
		return
	}
	r := b.RenderEditable()
	r.HandleTapDown(gestures.NewTapDownDetails(gestures.TapDownOpts{GlobalPosition: details.GlobalPosition}))
	kind, hasKind := details.Kind.Get()
	b.shouldShowSelectionToolbar = !hasKind || kind == ui.PointerDeviceKindTouch || kind == ui.PointerDeviceKindStylus
	b.shouldShowSelectionHandles = b.shouldShowSelectionToolbar
	sel, hasSel := r.Selection().Get()
	_ = sel
	isShiftPressedValid := b.isShiftPressed && hasSel
	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformMacOS:
		b.EditableText().HideToolbar(true)
		if isShiftPressedValid {
			from := foundation.None[services.TextSelection]()
			if !r.HasFocus() {
				from = foundation.Some(services.NewTextSelectionCollapsed(0))
			}
			b.expandSelection(details.GlobalPosition, services.SelectionChangedCauseTap, from)
			return
		}
		r.SelectPosition(services.SelectionChangedCauseTap)
	case foundation.TargetPlatformLinux, foundation.TargetPlatformWindows:
		b.EditableText().HideToolbar(true)
		if isShiftPressedValid {
			b.extendSelection(details.GlobalPosition, services.SelectionChangedCauseTap)
			return
		}
		r.SelectPosition(services.SelectionChangedCauseTap)
	}
}

// OnUserTapAlwaysCalled is whether OnUserTap is called for every tap rather than
// the single ones.
//
// Flutter: widgets/text_selection.dart TextSelectionGestureDetectorBuilder.onUserTapAlwaysCalled
func (b *TextSelectionGestureDetectorBuilder) OnUserTapAlwaysCalled() bool { return false }

// OnUserTap is called for a tap; a text field overrides it to call its onTap.
//
// Flutter: widgets/text_selection.dart TextSelectionGestureDetectorBuilder.onUserTap
func (b *TextSelectionGestureDetectorBuilder) OnUserTap() {}

// OnSingleTapUp handles a tap that is released and is not part of a double tap.
//
// Flutter: widgets/text_selection.dart TextSelectionGestureDetectorBuilder.onSingleTapUp
func (b *TextSelectionGestureDetectorBuilder) OnSingleTapUp(details gestures.TapDragUpDetails) {
	e := b.EditableText()
	if !b.delegate.SelectionEnabled() {
		e.RequestKeyboard()
		return
	}
	r := b.RenderEditable()
	_, hasSel := r.Selection().Get()
	isShiftPressedValid := b.isShiftPressed && hasSel
	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformAndroid, foundation.TargetPlatformFuchsia:
		e.HideToolbar(false)
		if isShiftPressedValid {
			b.extendSelection(details.GlobalPosition, services.SelectionChangedCauseTap)
			return
		}
		r.SelectPosition(services.SelectionChangedCauseTap)
	case foundation.TargetPlatformIOS:
		if isShiftPressedValid {
			from := foundation.None[services.TextSelection]()
			if !r.HasFocus() {
				from = foundation.Some(services.NewTextSelectionCollapsed(0))
			}
			b.expandSelection(details.GlobalPosition, services.SelectionChangedCauseTap, from)
			return
		}
		if k := details.Kind; k == ui.PointerDeviceKindTouch || k == ui.PointerDeviceKindUnknown {
			// Toggle the toolbar when the tap is exclusively within the bounds of a
			// non-collapsed previous selection, and the editable is focused.
			//
			// Toggle the toolbar if the previous selection is collapsed, the tap is on
			// the selection, the TextAffinity remains the same, the editable field is
			// not read only, and the editable is focused. The TextAffinity is important
			// when the cursor is on the boundary of a line wrap: if the affinity is
			// different (i.e. it is downstream), the selection should move to the
			// following line and not toggle the toolbar.
			//
			// Selects the word edge closest to the tap when the editable is not
			// focused, or if the tap was neither exclusively or inclusively on the
			// previous selection. If the selection remains the same after selecting the
			// word edge, then we toggle the toolbar, if the editable field is not read
			// only. If the selection changes then we hide the toolbar.
			previous := e.TextEditingValue().Selection
			if sel, ok := r.Selection().Get(); ok {
				previous = sel
			}
			textPosition := r.GetPositionForPoint(details.GlobalPosition)
			isAffinityTheSame := textPosition.Affinity == previous.Affinity
			switch {
			case ((b.positionWasOnSelectionExclusive(textPosition) && !previous.IsCollapsed()) ||
				(b.positionWasOnSelectionInclusive(textPosition) && previous.IsCollapsed() && isAffinityTheSame && !r.ReadOnly())) && r.HasFocus():
				e.ToggleToolbar(false)
			default:
				r.SelectWordEdge(services.SelectionChangedCauseTap)
				if previous == e.TextEditingValue().Selection && r.HasFocus() && !r.ReadOnly() {
					e.ToggleToolbar(false)
				} else {
					e.HideToolbar(false)
				}
			}
		} else {
			r.SelectPosition(services.SelectionChangedCauseTap)
			e.HideToolbar(true)
		}
	}
	e.RequestKeyboard()
}

func (b *TextSelectionGestureDetectorBuilder) positionWasOnSelectionExclusive(p ui.TextPosition) bool {
	sel, ok := b.RenderEditable().Selection().Get()
	return ok && sel.Start() < p.Offset && sel.End() > p.Offset
}

func (b *TextSelectionGestureDetectorBuilder) positionWasOnSelectionInclusive(p ui.TextPosition) bool {
	sel, ok := b.RenderEditable().Selection().Get()
	return ok && sel.Start() <= p.Offset && sel.End() >= p.Offset
}

// OnSingleTapCancel is called when a tap is cancelled.
//
// Flutter: widgets/text_selection.dart TextSelectionGestureDetectorBuilder.onSingleTapCancel
func (b *TextSelectionGestureDetectorBuilder) OnSingleTapCancel() {}

// showMagnifierIfSupportedByPlatform shows the magnifier on supported
// platforms at the given offset, currently only Android and iOS.
//
// Flutter: widgets/text_selection.dart TextSelectionGestureDetectorBuilder._showMagnifierIfSupportedByPlatform
func (b *TextSelectionGestureDetectorBuilder) showMagnifierIfSupportedByPlatform(positionToShow ui.Offset) {
	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformAndroid, foundation.TargetPlatformIOS:
		if e := b.EditableText(); e != nil {
			e.ShowMagnifier(positionToShow)
		}
	}
}

// hideMagnifierIfSupportedByPlatform hides the magnifier on supported
// platforms, currently only Android and iOS.
//
// Flutter: widgets/text_selection.dart TextSelectionGestureDetectorBuilder._hideMagnifierIfSupportedByPlatform
func (b *TextSelectionGestureDetectorBuilder) hideMagnifierIfSupportedByPlatform() {
	if !b.isEditableTextMounted() {
		return
	}

	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformAndroid, foundation.TargetPlatformIOS:
		if e := b.EditableText(); e != nil {
			e.HideMagnifier()
		}
	}
}

// isEditableTextMounted is whether the EditableText is mounted.
//
// Flutter: widgets/text_selection.dart TextSelectionGestureDetectorBuilder._isEditableTextMounted
func (b *TextSelectionGestureDetectorBuilder) isEditableTextMounted() bool {
	c := b.delegate.EditableTextKey().CurrentContext()
	return c != nil && c.Mounted()
}

// OnSingleLongTapStart handles the start of a long press, which selects a word.
//
// Flutter: widgets/text_selection.dart TextSelectionGestureDetectorBuilder.onSingleLongTapStart
func (b *TextSelectionGestureDetectorBuilder) OnSingleLongTapStart(details gestures.LongPressStartDetails) {
	if !b.delegate.SelectionEnabled() {
		return
	}
	r := b.RenderEditable()
	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformIOS, foundation.TargetPlatformMacOS:
		if !r.HasFocus() {
			b.longPressStartedWithoutFx = true
			r.SelectWord(services.SelectionChangedCauseLongPress)
		} else if r.ReadOnly() {
			r.SelectWord(services.SelectionChangedCauseLongPress)
		} else {
			r.SelectPositionAt(details.GlobalPosition, foundation.None[ui.Offset](), services.SelectionChangedCauseLongPress)
		}
	default:
		r.SelectWord(services.SelectionChangedCauseLongPress)
	}

	b.showMagnifierIfSupportedByPlatform(details.GlobalPosition)

	b.dragStartViewportOffset = r.Offset().Pixels()
	b.dragStartScrollOffset = b.scrollPosition()
}

// OnSingleLongTapMoveUpdate handles the move of a long press, which selects the
// words from the start to the pointer.
//
// Flutter: widgets/text_selection.dart TextSelectionGestureDetectorBuilder.onSingleLongTapMoveUpdate
func (b *TextSelectionGestureDetectorBuilder) OnSingleLongTapMoveUpdate(details gestures.LongPressMoveUpdateDetails) {
	if !b.delegate.SelectionEnabled() {
		return
	}
	r := b.RenderEditable()
	editableOffset, scrollableOffset := b.dragOffsets()
	selectWords := true
	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformIOS, foundation.TargetPlatformMacOS:
		if !b.longPressStartedWithoutFx && !r.ReadOnly() {
			r.SelectPositionAt(details.GlobalPosition, foundation.None[ui.Offset](), services.SelectionChangedCauseLongPress)
			selectWords = false
		}
	}
	if selectWords {
		r.SelectWordsInRange(details.GlobalPosition.Sub(details.OffsetFromOrigin).Sub(editableOffset).Sub(scrollableOffset),
			foundation.Some(details.GlobalPosition), services.SelectionChangedCauseLongPress)
	}

	b.showMagnifierIfSupportedByPlatform(details.GlobalPosition)
}

// OnSingleLongTapEnd handles the end of a long press.
//
// Flutter: widgets/text_selection.dart TextSelectionGestureDetectorBuilder.onSingleLongTapEnd
func (b *TextSelectionGestureDetectorBuilder) OnSingleLongTapEnd(gestures.LongPressEndDetails) {
	b.onSingleLongTapEndOrCancel()
	if b.shouldShowSelectionToolbar {
		b.EditableText().TryShowToolbar()
	}
}

// OnSingleLongTapCancel handles a cancelled long press.
//
// Flutter: widgets/text_selection.dart TextSelectionGestureDetectorBuilder.onSingleLongTapCancel
func (b *TextSelectionGestureDetectorBuilder) OnSingleLongTapCancel() { b.onSingleLongTapEndOrCancel() }

func (b *TextSelectionGestureDetectorBuilder) onSingleLongTapEndOrCancel() {
	b.hideMagnifierIfSupportedByPlatform()
	b.longPressStartedWithoutFx = false
	b.dragStartViewportOffset = 0
	b.dragStartScrollOffset = 0
}

// OnSecondaryTap handles a tap with the secondary button, which shows the
// context menu.
//
// Flutter: widgets/text_selection.dart TextSelectionGestureDetectorBuilder.onSecondaryTap
func (b *TextSelectionGestureDetectorBuilder) OnSecondaryTap() {
	if !b.delegate.SelectionEnabled() {
		return
	}
	r := b.RenderEditable()
	e := b.EditableText()
	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformIOS, foundation.TargetPlatformMacOS:
		if !b.lastSecondaryTapWasOnSelection() || !r.HasFocus() {
			r.SelectWord(services.SelectionChangedCauseTap)
		}
		if b.shouldShowSelectionToolbar {
			e.HideToolbar(true)
			e.TryShowToolbar()
		}
	default:
		if !r.HasFocus() {
			r.SelectPosition(services.SelectionChangedCauseTap)
		}
		e.ToggleToolbar(true)
	}
}

// OnTertiaryTapUp handles a tap with the tertiary (middle) button: on Linux
// desktops it puts the caret where the pointer is and pastes the primary
// selection there.
//
// Flutter: no counterpart
func (b *TextSelectionGestureDetectorBuilder) OnTertiaryTapUp(details gestures.TapUpDetails) {
	if !b.delegate.SelectionEnabled() || foundation.DefaultTargetPlatform() != foundation.TargetPlatformLinux {
		return
	}
	e := b.EditableText()
	if e.Widget().o.ReadOnly {
		return
	}
	b.RenderEditable().SelectPositionAt(details.GlobalPosition, foundation.None[ui.Offset](), services.SelectionChangedCauseTap)
	e.PastePrimarySelection(services.SelectionChangedCauseTap)
}

// OnSecondaryTapDown handles a pointer that goes down with the secondary
// button.
//
// Flutter: widgets/text_selection.dart TextSelectionGestureDetectorBuilder.onSecondaryTapDown
func (b *TextSelectionGestureDetectorBuilder) OnSecondaryTapDown(details gestures.TapDownDetails) {
	b.RenderEditable().HandleSecondaryTapDown(gestures.NewTapDownDetails(gestures.TapDownOpts{GlobalPosition: details.GlobalPosition}))
	b.shouldShowSelectionToolbar = true
	k, ok := details.Kind.Get()
	b.shouldShowSelectionHandles = !ok || k == ui.PointerDeviceKindTouch || k == ui.PointerDeviceKindStylus
}

// OnDoubleTapDown handles the second tap of a double tap, which selects a word.
//
// Flutter: widgets/text_selection.dart TextSelectionGestureDetectorBuilder.onDoubleTapDown
func (b *TextSelectionGestureDetectorBuilder) OnDoubleTapDown(gestures.TapDragDownDetails) {
	if b.delegate.SelectionEnabled() {
		b.RenderEditable().SelectWord(services.SelectionChangedCauseDoubleTap)
		if b.shouldShowSelectionToolbar {
			b.EditableText().TryShowToolbar()
		}
	}
}

func (b *TextSelectionGestureDetectorBuilder) selectParagraphsInRange(from ui.Offset, to foundation.Opt[ui.Offset], cause services.SelectionChangedCause) {
	b.selectTextBoundariesInRange(services.NewParagraphBoundary(b.EditableText().TextEditingValue().Text), from, to, cause)
}

func (b *TextSelectionGestureDetectorBuilder) selectLinesInRange(from ui.Offset, to foundation.Opt[ui.Offset], cause services.SelectionChangedCause) {
	b.selectTextBoundariesInRange(services.NewLineBoundary(b.RenderEditable()), from, to, cause)
}

func (b *TextSelectionGestureDetectorBuilder) moveToTextBoundary(extent ui.TextPosition, boundary services.TextBoundary) ui.TextRange {
	text := b.EditableText().TextEditingValue().Text
	at := extent.Offset
	if at == len(text) {
		at--
	}
	start, _ := boundary.GetLeadingTextBoundaryAt(at)
	end, ok := boundary.GetTrailingTextBoundaryAt(extent.Offset)
	if !ok {
		end = len(text)
	}
	return ui.TextRange{Start: start, End: end}
}

func (b *TextSelectionGestureDetectorBuilder) selectTextBoundariesInRange(boundary services.TextBoundary, from ui.Offset, to foundation.Opt[ui.Offset], cause services.SelectionChangedCause) {
	r := b.RenderEditable()
	fromPosition := r.GetPositionForPoint(from)
	fromRange := b.moveToTextBoundary(fromPosition, boundary)
	toPosition := fromPosition
	if t, ok := to.Get(); ok {
		toPosition = r.GetPositionForPoint(t)
	}
	toRange := fromRange
	if toPosition != fromPosition {
		toRange = b.moveToTextBoundary(toPosition, boundary)
	}
	var newSelection services.TextSelection
	if fromRange.Start < toRange.End {
		newSelection = services.NewTextSelection(fromRange.Start, toRange.End)
	} else {
		newSelection = services.NewTextSelection(fromRange.End, toRange.Start)
	}
	e := b.EditableText()
	e.UserUpdateTextEditingValue(e.TextEditingValue().CopyWith(services.TextEditingValueCopy{Selection: foundation.Some(newSelection)}), cause)
}

// OnTripleTapDown handles the third tap of a triple tap, which selects a line (a
// paragraph on some platforms, all the text in a single line field).
//
// Flutter: widgets/text_selection.dart TextSelectionGestureDetectorBuilder.onTripleTapDown
func (b *TextSelectionGestureDetectorBuilder) OnTripleTapDown(details gestures.TapDragDownDetails) {
	if !b.delegate.SelectionEnabled() {
		return
	}
	r := b.RenderEditable()
	if m, ok := r.MaxLines().Get(); ok && m == 1 {
		b.EditableText().SelectAll(services.SelectionChangedCauseTap)
	} else if foundation.DefaultTargetPlatform() == foundation.TargetPlatformLinux {
		b.selectLinesInRange(details.GlobalPosition, foundation.None[ui.Offset](), services.SelectionChangedCauseTap)
	} else {
		b.selectParagraphsInRange(details.GlobalPosition, foundation.None[ui.Offset](), services.SelectionChangedCauseTap)
	}
	if b.shouldShowSelectionToolbar {
		b.EditableText().TryShowToolbar()
	}
}

// OnDragSelectionStart handles the start of a drag that selects.
//
// Flutter: widgets/text_selection.dart TextSelectionGestureDetectorBuilder.onDragSelectionStart
func (b *TextSelectionGestureDetectorBuilder) OnDragSelectionStart(details gestures.TapDragStartDetails) {
	if !b.delegate.SelectionEnabled() {
		return
	}
	r := b.RenderEditable()
	kind, hasKind := details.Kind.Get()
	b.shouldShowSelectionToolbar = !hasKind || kind == ui.PointerDeviceKindTouch || kind == ui.PointerDeviceKindStylus
	b.shouldShowSelectionHandles = b.shouldShowSelectionToolbar
	b.dragStartSelection = r.Selection()
	b.dragStartScrollOffset = b.scrollPosition()
	b.dragStartViewportOffset = r.Offset().Pixels()
	if effectiveConsecutiveTapCount(details.ConsecutiveTapCount) > 1 {
		return
	}
	sel, hasSel := r.Selection().Get()
	if b.isShiftPressed && hasSel && sel.IsValid() {
		switch foundation.DefaultTargetPlatform() {
		case foundation.TargetPlatformIOS, foundation.TargetPlatformMacOS:
			b.expandSelection(details.GlobalPosition, services.SelectionChangedCauseDrag, foundation.None[services.TextSelection]())
		default:
			b.extendSelection(details.GlobalPosition, services.SelectionChangedCauseDrag)
		}
		return
	}
	isPrecise := hasKind && (kind == ui.PointerDeviceKindMouse || kind == ui.PointerDeviceKindTrackpad)
	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformIOS:
		if isPrecise {
			r.SelectPositionAt(details.GlobalPosition, foundation.None[ui.Offset](), services.SelectionChangedCauseDrag)
		}
	case foundation.TargetPlatformAndroid, foundation.TargetPlatformFuchsia:
		switch {
		case isPrecise:
			r.SelectPositionAt(details.GlobalPosition, foundation.None[ui.Offset](), services.SelectionChangedCauseDrag)
		case hasKind:
			// For Android, Fuchsia, and iOS platforms, a touch drag does not
			// initiate unless the editable has focus.
			if r.HasFocus() {
				r.SelectPositionAt(details.GlobalPosition, foundation.None[ui.Offset](), services.SelectionChangedCauseDrag)
				b.showMagnifierIfSupportedByPlatform(details.GlobalPosition)
			}
		}
	default:
		r.SelectPositionAt(details.GlobalPosition, foundation.None[ui.Offset](), services.SelectionChangedCauseDrag)
	}
}

// OnDragSelectionUpdate handles the move of a drag that selects.
//
// Flutter: widgets/text_selection.dart TextSelectionGestureDetectorBuilder.onDragSelectionUpdate
func (b *TextSelectionGestureDetectorBuilder) OnDragSelectionUpdate(details gestures.TapDragUpdateDetails) {
	if !b.delegate.SelectionEnabled() {
		return
	}
	r := b.RenderEditable()
	e := b.EditableText()
	if !b.isShiftPressed {
		editableOffset, scrollableOffset := b.dragOffsets()
		dragStart := details.GlobalPosition.Sub(details.OffsetFromOrigin)
		from := dragStart.Sub(editableOffset).Sub(scrollableOffset)
		to := foundation.Some(details.GlobalPosition)
		kind, hasKind := details.Kind.Get()
		isPrecise := hasKind && (kind == ui.PointerDeviceKindMouse || kind == ui.PointerDeviceKindTrackpad)
		switch effectiveConsecutiveTapCount(details.ConsecutiveTapCount) {
		case 2:
			// Select word by word.
			r.SelectWordsInRange(from, to, services.SelectionChangedCauseDrag)
			if hasKind && !isPrecise {
				b.showMagnifierIfSupportedByPlatform(details.GlobalPosition)
			}
			return
		case 3:
			// Select paragraph-by-paragraph.
			switch foundation.DefaultTargetPlatform() {
			case foundation.TargetPlatformLinux:
				b.selectLinesInRange(from, to, services.SelectionChangedCauseDrag)
			case foundation.TargetPlatformWindows, foundation.TargetPlatformMacOS:
				b.selectParagraphsInRange(from, to, services.SelectionChangedCauseDrag)
			default:
				if isPrecise {
					b.selectParagraphsInRange(from, to, services.SelectionChangedCauseDrag)
				}
				// Triple tap to drag is not present on these platforms when using
				// non-precise pointer devices at the moment.
			}
			return
		}
		switch foundation.DefaultTargetPlatform() {
		case foundation.TargetPlatformIOS:
			// With a mouse device, a drag should select the range from the origin
			// of the drag to the current position of the drag. With a touch
			// device, nothing should happen.
			if isPrecise {
				r.SelectPositionAt(from, to, services.SelectionChangedCauseDrag)
			}
		case foundation.TargetPlatformAndroid, foundation.TargetPlatformFuchsia:
			// With a precise pointer device, such as a mouse, trackpad, or stylus,
			// the drag will select the text spanning the origin of the drag to the
			// end of the drag. With a touch device, the cursor should move with the
			// drag.
			switch {
			case isPrecise || (hasKind && (kind == ui.PointerDeviceKindStylus || kind == ui.PointerDeviceKindInvertedStylus)):
				r.SelectPositionAt(from, to, services.SelectionChangedCauseDrag)
			case hasKind:
				if r.HasFocus() {
					r.SelectPositionAt(details.GlobalPosition, foundation.None[ui.Offset](), services.SelectionChangedCauseDrag)
					b.showMagnifierIfSupportedByPlatform(details.GlobalPosition)
				}
			}
		default:
			r.SelectPositionAt(from, to, services.SelectionChangedCauseDrag)
		}
		return
	}
	start, _ := b.dragStartSelection.Get()
	if start.IsCollapsed() || (foundation.DefaultTargetPlatform() != foundation.TargetPlatformIOS && foundation.DefaultTargetPlatform() != foundation.TargetPlatformMacOS) {
		b.extendSelection(details.GlobalPosition, services.SelectionChangedCauseDrag)
		return
	}
	selection := e.TextEditingValue().Selection
	nextExtent := r.GetPositionForPoint(details.GlobalPosition)
	forward := start.BaseOffset < start.ExtentOffset
	var inverted bool
	if forward {
		inverted = nextExtent.Offset < start.BaseOffset
	} else {
		inverted = nextExtent.Offset > start.BaseOffset
	}
	update := func(sel services.TextSelection) {
		e.UserUpdateTextEditingValue(e.TextEditingValue().CopyWith(services.TextEditingValueCopy{Selection: foundation.Some(sel)}), services.SelectionChangedCauseDrag)
	}
	switch {
	case inverted && selection.BaseOffset == start.BaseOffset:
		update(services.NewTextSelection(start.ExtentOffset, nextExtent.Offset))
	case !inverted && nextExtent.Offset != start.BaseOffset && selection.BaseOffset != start.BaseOffset:
		update(services.NewTextSelection(start.BaseOffset, nextExtent.Offset))
	default:
		b.extendSelection(details.GlobalPosition, services.SelectionChangedCauseDrag)
	}
}

// OnDragSelectionEnd handles the end of a drag that selects.
//
// Flutter: widgets/text_selection.dart TextSelectionGestureDetectorBuilder.onDragSelectionEnd
func (b *TextSelectionGestureDetectorBuilder) OnDragSelectionEnd(details gestures.TapDragEndDetails) {
	if b.shouldShowSelectionToolbar && effectiveConsecutiveTapCount(details.ConsecutiveTapCount) == 2 {
		b.EditableText().TryShowToolbar()
	}
	if b.isShiftPressed {
		b.dragStartSelection = foundation.None[services.TextSelection]()
	}

	b.hideMagnifierIfSupportedByPlatform()
}

// BuildGestureDetector builds the detector for the text field's child.
//
// Flutter: widgets/text_selection.dart TextSelectionGestureDetectorBuilder.buildGestureDetector
func (b *TextSelectionGestureDetectorBuilder) BuildGestureDetector(child Widget, opts ...TextSelectionGestureDetectorBuildOpts) Widget {
	o := firstOpts(opts)
	h := b.self
	var tertiary gestures.GestureTapUpCallback
	if t, ok := h.(interface{ OnTertiaryTapUp(gestures.TapUpDetails) }); ok {
		tertiary = t.OnTertiaryTapUp
	}
	return NewTextSelectionGestureDetector(child, TextSelectionGestureDetectorOpts{
		OnTertiaryTapUp:      tertiary,
		Key:                  o.Key,
		OnTapTrackStart:      h.OnTapTrackStart,
		OnTapTrackReset:      h.OnTapTrackReset,
		OnTapDown:            h.OnTapDown,
		OnSecondaryTap:       h.OnSecondaryTap,
		OnSecondaryTapDown:   h.OnSecondaryTapDown,
		OnSingleTapUp:        h.OnSingleTapUp,
		OnSingleTapCancel:    h.OnSingleTapCancel,
		OnUserTap:            h.OnUserTap,
		OnSingleLongTapStart: h.OnSingleLongTapStart, OnSingleLongTapMoveUpdate: h.OnSingleLongTapMoveUpdate,
		OnSingleLongTapEnd: h.OnSingleLongTapEnd, OnSingleLongTapCancel: h.OnSingleLongTapCancel,
		OnDoubleTapDown: h.OnDoubleTapDown, OnTripleTapDown: h.OnTripleTapDown,
		OnDragSelectionStart: h.OnDragSelectionStart, OnDragSelectionUpdate: h.OnDragSelectionUpdate, OnDragSelectionEnd: h.OnDragSelectionEnd,
		OnUserTapAlwaysCalled: h.OnUserTapAlwaysCalled(), Behavior: o.Behavior,
	})
}

// TextSelectionGestureDetectorBuildOpts are the optional parameters of
// BuildGestureDetector.
//
// Flutter: widgets/text_selection.dart TextSelectionGestureDetectorBuilder.buildGestureDetector
type TextSelectionGestureDetectorBuildOpts struct {
	// Key is the key of the detector.
	Key foundation.Key
	// Behavior is how the detector behaves during hit testing.
	Behavior rendering.HitTestBehavior
}

// effectiveConsecutiveTapCount is the number of consecutive taps the platform
// counts: some platforms start over after three, others alternate between one
// and two.
//
// Flutter: widgets/text_selection.dart _TextSelectionGestureDetectorState._getEffectiveConsecutiveTapCount
func effectiveConsecutiveTapCount(raw int) int {
	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformIOS, foundation.TargetPlatformMacOS:
		return min(raw, 3)
	case foundation.TargetPlatformWindows:
		if raw < 2 {
			return raw
		}
		return 2 + raw%2
	}
	if raw <= 3 {
		return raw
	}
	if raw%3 == 0 {
		return 3
	}
	return raw % 3
}

// TextSelectionGestureDetectorOpts are the optional parameters of
// NewTextSelectionGestureDetector.
//
// Flutter: widgets/text_selection.dart TextSelectionGestureDetector.new
type TextSelectionGestureDetectorOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// OnTapTrackStart is called when a pointer goes down, before a tap.
	OnTapTrackStart func()
	// OnTapTrackReset is called when the tracking of a tap ends.
	OnTapTrackReset func()
	// OnTapDown is called when a pointer goes down, with the count of the
	// consecutive taps.
	OnTapDown gestures.GestureTapDragDownCallback
	// OnSecondaryTap is called for a tap with the secondary button.
	OnSecondaryTap gestures.GestureTapCallback
	// OnSecondaryTapDown is called when the secondary button goes down.
	OnSecondaryTapDown gestures.GestureTapDownCallback
	// OnTertiaryTapUp is called for a tap with the tertiary (middle) button.
	//
	// Flutter: no counterpart
	OnTertiaryTapUp gestures.GestureTapUpCallback
	// OnSingleTapUp is called when a tap that is not part of a double tap ends.
	OnSingleTapUp gestures.GestureTapDragUpCallback
	// OnSingleTapCancel is called when a tap is cancelled.
	OnSingleTapCancel gestures.GestureCancelCallback
	// OnUserTap is called for a single tap, after OnSingleTapUp.
	OnUserTap gestures.GestureTapCallback
	// OnSingleLongTapStart is called when a long press starts.
	OnSingleLongTapStart gestures.GestureLongPressStartCallback
	// OnSingleLongTapMoveUpdate is called when a long press moves.
	OnSingleLongTapMoveUpdate gestures.GestureLongPressMoveUpdateCallback
	// OnSingleLongTapEnd is called when a long press ends.
	OnSingleLongTapEnd gestures.GestureLongPressEndCallback
	// OnSingleLongTapCancel is called when a long press is cancelled.
	OnSingleLongTapCancel gestures.GestureLongPressCancelCallback
	// OnDoubleTapDown is called when the second tap of a double tap goes down.
	OnDoubleTapDown gestures.GestureTapDragDownCallback
	// OnTripleTapDown is called when the third tap of a triple tap goes down.
	OnTripleTapDown gestures.GestureTapDragDownCallback
	// OnDragSelectionStart is called when a drag starts.
	OnDragSelectionStart gestures.GestureTapDragStartCallback
	// OnDragSelectionUpdate is called when a drag moves.
	OnDragSelectionUpdate gestures.GestureTapDragUpdateCallback
	// OnDragSelectionEnd is called when a drag ends.
	OnDragSelectionEnd gestures.GestureTapDragEndCallback
	// OnUserTapAlwaysCalled makes OnUserTap run for every tap.
	OnUserTapAlwaysCalled bool
	// Behavior is how the detector behaves during hit testing.
	Behavior rendering.HitTestBehavior
}

// TextSelectionGestureDetector is a gesture detector for the selection of a text
// field: taps, double and triple taps, drags and long presses. The consecutive
// taps are counted by the platform's rules.
//
// Flutter: widgets/text_selection.dart TextSelectionGestureDetector
type TextSelectionGestureDetector struct {
	StatefulWidget
	child Widget
	o     TextSelectionGestureDetectorOpts
}

// NewTextSelectionGestureDetector creates the widget.
//
// Flutter: widgets/text_selection.dart TextSelectionGestureDetector.new
func NewTextSelectionGestureDetector(child Widget, opts ...TextSelectionGestureDetectorOpts) *TextSelectionGestureDetector {
	o := firstOpts(opts)
	return &TextSelectionGestureDetector{StatefulWidget: NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), child: child, o: o}
}

// CreateState creates the state.
//
// Flutter: widgets/text_selection.dart TextSelectionGestureDetector.createState
func (w *TextSelectionGestureDetector) CreateState() State {
	return &textSelectionGestureDetectorState{}
}

type textSelectionGestureDetectorState struct {
	StateOf[*TextSelectionGestureDetector]
}

func (s *textSelectionGestureDetectorState) handleTapDown(details gestures.TapDragDownDetails) {
	o := s.Widget().o
	if o.OnTapDown != nil {
		o.OnTapDown(details)
	}
	switch effectiveConsecutiveTapCount(details.ConsecutiveTapCount) {
	case 2:
		if o.OnDoubleTapDown != nil {
			o.OnDoubleTapDown(details)
		}
	case 3:
		if o.OnTripleTapDown != nil {
			o.OnTripleTapDown(details)
		}
	}
}

func (s *textSelectionGestureDetectorState) handleTapUp(details gestures.TapDragUpDetails) {
	o := s.Widget().o
	if effectiveConsecutiveTapCount(details.ConsecutiveTapCount) == 1 {
		if o.OnSingleTapUp != nil {
			o.OnSingleTapUp(details)
		}
		if o.OnUserTap != nil {
			o.OnUserTap()
		}
	} else if o.OnUserTapAlwaysCalled && o.OnUserTap != nil {
		o.OnUserTap()
	}
}

// Build makes the gesture detector with the recognizers for the callbacks.
//
// Flutter: widgets/text_selection.dart _TextSelectionGestureDetectorState.build
func (s *textSelectionGestureDetectorState) Build(bc BuildContext) Widget {
	w := s.Widget()
	o := w.o
	binding := WidgetsBindingOf(bc).Gestures()
	g := map[reflect.Type]GestureRecognizerFactory{}
	g[reflect.TypeFor[*gestures.TapGestureRecognizer]()] = NewGestureRecognizerFactoryWithHandlers(
		func() *gestures.TapGestureRecognizer {
			return gestures.NewTapGestureRecognizer(binding, gestures.TapGestureRecognizerOpts{DebugOwner: s})
		},
		func(r *gestures.TapGestureRecognizer) {
			r.OnSecondaryTap, r.OnSecondaryTapDown = o.OnSecondaryTap, o.OnSecondaryTapDown
			r.OnTertiaryTapUp = o.OnTertiaryTapUp
		})
	if o.OnSingleLongTapStart != nil || o.OnSingleLongTapMoveUpdate != nil || o.OnSingleLongTapEnd != nil || o.OnSingleLongTapCancel != nil {
		g[reflect.TypeFor[*gestures.LongPressGestureRecognizer]()] = NewGestureRecognizerFactoryWithHandlers(
			func() *gestures.LongPressGestureRecognizer {
				return gestures.NewLongPressGestureRecognizer(binding, gestures.LongPressGestureRecognizerOpts{
					DebugOwner: s, SupportedDevices: []ui.PointerDeviceKind{ui.PointerDeviceKindTouch}})
			},
			func(r *gestures.LongPressGestureRecognizer) {
				r.OnLongPressStart, r.OnLongPressMoveUpdate, r.OnLongPressEnd, r.OnLongPressCancel =
					o.OnSingleLongTapStart, o.OnSingleLongTapMoveUpdate, o.OnSingleLongTapEnd, o.OnSingleLongTapCancel
			})
	}
	if o.OnDragSelectionStart != nil || o.OnDragSelectionUpdate != nil || o.OnDragSelectionEnd != nil {
		configure := func(r *gestures.BaseTapAndDragGestureRecognizer) {
			r.DragStartBehavior = gestures.DragStartBehaviorDown
			r.OnTapTrackStart, r.OnTapTrackReset = o.OnTapTrackStart, o.OnTapTrackReset
			r.OnTapDown = s.handleTapDown
			r.OnDragStart, r.OnDragUpdate, r.OnDragEnd = o.OnDragSelectionStart, o.OnDragSelectionUpdate, o.OnDragSelectionEnd
			r.OnTapUp = s.handleTapUp
			r.OnCancel = o.OnSingleTapCancel
		}
		switch foundation.DefaultTargetPlatform() {
		case foundation.TargetPlatformLinux, foundation.TargetPlatformMacOS, foundation.TargetPlatformWindows:
			g[reflect.TypeFor[*gestures.TapAndPanGestureRecognizer]()] = NewGestureRecognizerFactoryWithHandlers(
				func() *gestures.TapAndPanGestureRecognizer {
					return gestures.NewTapAndPanGestureRecognizer(binding, gestures.TapAndDragGestureRecognizerOpts{DebugOwner: s})
				},
				func(r *gestures.TapAndPanGestureRecognizer) { configure(&r.BaseTapAndDragGestureRecognizer) })
		default:
			g[reflect.TypeFor[*gestures.TapAndHorizontalDragGestureRecognizer]()] = NewGestureRecognizerFactoryWithHandlers(
				func() *gestures.TapAndHorizontalDragGestureRecognizer {
					return gestures.NewTapAndHorizontalDragGestureRecognizer(binding, gestures.TapAndDragGestureRecognizerOpts{DebugOwner: s})
				},
				func(r *gestures.TapAndHorizontalDragGestureRecognizer) {
					configure(&r.BaseTapAndDragGestureRecognizer)
					r.EagerVictoryOnDrag = foundation.DefaultTargetPlatform() != foundation.TargetPlatformIOS
				})
		}
	}
	return NewRawGestureDetector(w.child, RawGestureDetectorOpts{Gestures: g, ExcludeFromSemantics: true, Behavior: o.Behavior})
}
