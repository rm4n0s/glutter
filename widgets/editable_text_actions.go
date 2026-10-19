package widgets

import (
	"reflect"
	"unicode/utf8"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// defaultEditableScrollPadding is the default padding that is kept around the
// caret when the field scrolls to show it.
//
// Flutter: widgets/editable_text.dart EditableText.scrollPadding
var defaultEditableScrollPadding = painting.NewEdgeInsetsAll(20)

// applyTextBoundary is a function that moves a position by a text boundary.
//
// Flutter: widgets/editable_text.dart _ApplyTextBoundary
type applyTextBoundary func(extent ui.TextPosition, forward bool, boundary services.TextBoundary) ui.TextPosition

// moveBeyondTextBoundary is the closest boundary location to the extent but not
// including the extent itself (unless already at the start/end of the text), in
// the direction specified by forward.
//
// Flutter: widgets/editable_text.dart EditableTextState._moveBeyondTextBoundary
func (s *EditableTextState) moveBeyondTextBoundary(extent ui.TextPosition, forward bool, boundary services.TextBoundary) ui.TextPosition {
	var offset int
	if forward {
		o, ok := boundary.GetTrailingTextBoundaryAt(extent.Offset)
		if !ok {
			o = len(s.value().Text)
		}
		offset = o
	} else {
		// If x is a boundary defined by the boundary, most boundaries (except the
		// line breaker) guarantee x == getLeadingTextBoundaryAt(x). Use x - 1 here
		// to make sure we don't get stuck at the fixed point x.
		o, _ := boundary.GetLeadingTextBoundaryAt(extent.Offset - 1)
		offset = o
	}
	return ui.NewTextPosition(offset)
}

// moveToTextBoundary is the closest boundary location to the extent, including
// the extent itself, in the direction specified by forward. It is a fixed point
// of itself, and used for the line boundaries, since performing "move to line
// start" more than once usually doesn't move you to the previous line.
//
// Flutter: widgets/editable_text.dart EditableTextState._moveToTextBoundary
func (s *EditableTextState) moveToTextBoundary(extent ui.TextPosition, forward bool, boundary services.TextBoundary) ui.TextPosition {
	var caretOffset int
	if extent.Affinity == ui.TextAffinityUpstream {
		if extent.Offset < 1 && !forward {
			return ui.NewTextPosition(0)
		}
		// When the text affinity is upstream, the caret is associated with the
		// grapheme before the offset.
		caretOffset = max(0, extent.Offset-1)
	} else {
		caretOffset = extent.Offset
	}
	if forward {
		o, ok := boundary.GetTrailingTextBoundaryAt(caretOffset)
		if !ok {
			o = len(s.value().Text)
		}
		return ui.NewTextPosition(o, ui.TextPositionOpts{Affinity: ui.TextAffinityUpstream})
	}
	o, _ := boundary.GetLeadingTextBoundaryAt(caretOffset)
	return ui.NewTextPosition(o)
}

func (s *EditableTextState) characterBoundary() services.TextBoundary {
	if s.Widget().o.ObscureText {
		return newCodePointBoundary(s.value().Text)
	}
	return services.NewCharacterBoundary(s.value().Text)
}

func (s *EditableTextState) nextWordBoundary() services.TextBoundary {
	if s.Widget().o.ObscureText {
		return s.documentBoundary()
	}
	return s.renderEditable().WordBoundaries().MoveByWordBoundary
}

func (s *EditableTextState) linebreak() services.TextBoundary {
	if s.Widget().o.ObscureText {
		return s.documentBoundary()
	}
	return services.NewLineBoundary(s.renderEditable())
}

func (s *EditableTextState) paragraphBoundary() services.TextBoundary {
	return services.NewParagraphBoundary(s.value().Text)
}

func (s *EditableTextState) documentBoundary() services.TextBoundary {
	return services.NewDocumentBoundary(s.value().Text)
}

func (s *EditableTextState) makeOverridable(a Action) Action {
	return NewOverridableAction(a, s.Context())
}

// transposeCharacters swaps the two characters around the caret, or before it at
// the end of the text.
//
// Flutter: widgets/editable_text.dart EditableTextState._transposeCharacters
func (s *EditableTextState) transposeCharacters(TransposeCharactersIntent) any {
	v := s.value()
	if foundation.GraphemeCount(v.Text) <= 1 || !v.Selection.IsCollapsed() || v.Selection.BaseOffset == 0 {
		return nil
	}
	bounds := foundation.GraphemeBoundaries(v.Text)
	at := v.Selection.BaseOffset
	atEnd := at == len(v.Text)
	// The index of the boundary at (or after) the caret.
	idx := 0
	for idx < len(bounds) && bounds[idx] < at {
		idx++
	}
	var first, second int // the byte ranges of the two characters are [first, mid) and [mid, second)
	var mid int
	if atEnd {
		first, mid, second = bounds[len(bounds)-3], bounds[len(bounds)-2], bounds[len(bounds)-1]
	} else {
		if idx+1 >= len(bounds) || idx == 0 {
			return nil
		}
		first, mid, second = bounds[idx-1], bounds[idx], bounds[idx+1]
	}
	text := v.Text[:first] + v.Text[mid:second] + v.Text[first:mid] + v.Text[second:]
	s.UserUpdateTextEditingValue(services.NewTextEditingValue(services.TextEditingValueOpts{
		Text:      text,
		Selection: foundation.Some(services.NewTextSelectionCollapsed(second)),
	}), services.SelectionChangedCauseKeyboard)
	return nil
}

func (s *EditableTextState) updateSelection(intent UpdateSelectionIntent) any {
	n := len(intent.CurrentTextEditingValue.Text)
	if foundation.DebugMode && (intent.NewSelection.Start() > n || intent.NewSelection.End() > n) {
		panic(errors.New("UpdateSelectionOutOfRange", "the selection must not exceed the current text length",
			"selection", intent.NewSelection.String(), "textLength", n))
	}
	s.BringIntoView(intent.NewSelection.Extent())
	s.UserUpdateTextEditingValue(intent.CurrentTextEditingValue.CopyWith(services.TextEditingValueCopy{Selection: foundation.Some(intent.NewSelection)}), intent.Cause)
	return nil
}

func (s *EditableTextState) scrollToDocumentBoundary(intent ScrollToDocumentBoundaryIntent) any {
	if intent.forward {
		s.BringIntoView(ui.NewTextPosition(len(s.value().Text)))
	} else {
		s.BringIntoView(ui.NewTextPosition(0))
	}
	return nil
}

// scroll scrolls a page up or down; the field scrolls to the end when it is a
// single line.
//
// Flutter: widgets/editable_text.dart EditableTextState._scroll
func (s *EditableTextState) scroll(intent ScrollIntent) any {
	if intent.Type != ScrollIncrementTypePage {
		return nil
	}
	c := s.scrollController()
	position := c.Position()
	if m, ok := s.Widget().o.MaxLines.Get(); ok && m == 1 {
		c.JumpTo(position.MaxScrollExtent())
		return nil
	}
	// If the field isn't scrollable, do nothing. For example, when the lines of
	// text is less than maxLines, the field has nothing to scroll.
	if position.MaxScrollExtent() == 0 && position.MinScrollExtent() == 0 {
		return nil
	}
	state, ok := s.scrollableKey.CurrentState()
	if !ok {
		return nil
	}
	increment := ScrollActionGetDirectionalIncrement(state.(*ScrollableState), intent)
	destination := clampFloat(position.Pixels()+increment, position.MinScrollExtent(), position.MaxScrollExtent())
	if destination == position.Pixels() {
		return nil
	}
	c.JumpTo(destination)
	return nil
}

// buildActions makes the map of the actions of the field.
//
// Flutter: widgets/editable_text.dart EditableTextState._actions
func (s *EditableTextState) buildActions() ActionMap {
	update := func(boundary func() services.TextBoundary, apply applyTextBoundary, ignoreNonCollapsed, isExpand, extentAtIndex bool) *updateTextSelectionAction {
		return newUpdateTextSelectionAction(s, boundary, apply, ignoreNonCollapsed, isExpand, extentAtIndex)
	}
	beyond, to := applyTextBoundary(s.moveBeyondTextBoundary), applyTextBoundary(s.moveToTextBoundary)
	m := ActionMap{}
	add := func(a Action) { m[a.IntentType()] = a }
	add(NewDoNothingActionFor[DoNothingAndStopPropagationTextIntent](DoNothingActionOpts{DoesNotConsumeKey: true}))
	add(NewCallbackAction(func(i ReplaceTextIntent) any { s.replaceText(i); return nil }))
	add(NewCallbackAction(s.updateSelection))
	add(NewDirectionalFocusActionForTextField())
	add(NewCallbackAction(func(i DismissIntent) any {
		// TODO(step 11.5): hide the toolbar first when it is visible.
		return ActionsInvoke(s.Context(), i)
	}))
	// Delete
	add(s.makeOverridable(newDeleteTextAction[DeleteCharacterIntent](s, s.characterBoundary, beyond)))
	add(s.makeOverridable(newDeleteTextAction[DeleteToNextWordBoundaryIntent](s, s.nextWordBoundary, beyond)))
	add(s.makeOverridable(newDeleteTextAction[DeleteToLineBreakIntent](s, s.linebreak, to)))
	// Extend/Move Selection
	add(s.makeOverridable(newTypedUpdate[ExtendSelectionByCharacterIntent](update(s.characterBoundary, beyond, false, false, false))))
	add(s.makeOverridable(newTypedUpdate[ExtendSelectionToNextWordBoundaryIntent](update(s.nextWordBoundary, beyond, true, false, false))))
	add(s.makeOverridable(newTypedUpdate[ExtendSelectionToNextParagraphBoundaryIntent](update(s.paragraphBoundary, beyond, true, false, false))))
	add(s.makeOverridable(newTypedUpdate[ExtendSelectionToLineBreakIntent](update(s.linebreak, to, true, false, false))))
	m[IntentTypeOf[ExtendSelectionVerticallyToAdjacentLineIntent]()] = s.makeOverridable(newTypedContextAction[ExtendSelectionVerticallyToAdjacentLineIntent](s.verticalSelectionMover))
	m[IntentTypeOf[ExtendSelectionVerticallyToAdjacentPageIntent]()] = s.makeOverridable(newTypedContextAction[ExtendSelectionVerticallyToAdjacentPageIntent](s.verticalSelectionMover))
	add(s.makeOverridable(newTypedUpdate[ExtendSelectionToNextParagraphBoundaryOrCaretLocationIntent](update(s.paragraphBoundary, beyond, true, false, false))))
	add(s.makeOverridable(newTypedUpdate[ExtendSelectionToDocumentBoundaryIntent](update(s.documentBoundary, beyond, true, false, false))))
	add(s.makeOverridable(newTypedUpdate[ExtendSelectionToNextWordBoundaryOrCaretLocationIntent](update(s.nextWordBoundary, beyond, true, false, false))))
	add(s.makeOverridable(NewCallbackAction(s.scrollToDocumentBoundary)))
	add(NewCallbackAction(s.scroll))
	// Expand Selection
	add(s.makeOverridable(newTypedUpdate[ExpandSelectionToLineBreakIntent](update(s.linebreak, to, true, true, false))))
	add(s.makeOverridable(newTypedUpdate[ExpandSelectionToDocumentBoundaryIntent](update(s.documentBoundary, to, true, true, true))))
	// Copy Paste
	add(s.makeOverridable(newSelectAllAction(s)))
	add(s.makeOverridable(newCopySelectionAction(s)))
	add(s.makeOverridable(newPasteSelectionAction(s)))
	add(s.makeOverridable(NewCallbackAction(s.transposeCharacters)))
	add(s.makeOverridable(newTapOutsideAction()))
	add(s.makeOverridable(newTapUpOutsideAction()))
	return m
}

// NewDoNothingActionFor creates a DoNothingAction for an intent type.
//
// Flutter: widgets/actions.dart DoNothingAction
func NewDoNothingActionFor[T Intent](opts ...DoNothingActionOpts) Action {
	return &typedAction{Action: NewDoNothingAction(opts...), intentType: reflect.TypeFor[T]()}
}

// typedAction gives an action the type of intents it handles, when the action
// can handle several (or the base one).
type typedAction struct {
	Action
	intentType reflect.Type
}

func (a *typedAction) IntentType() reflect.Type { return a.intentType }

// typedContextAction gives a context action the type of intents it handles, when
// one action handles several types.
type typedContextAction struct {
	ContextAction
	intentType reflect.Type
}

func (a *typedContextAction) IntentType() reflect.Type { return a.intentType }

func newTypedContextAction[T Intent](a ContextAction) Action {
	return &typedContextAction{ContextAction: a, intentType: reflect.TypeFor[T]()}
}

func newTypedUpdate[T DirectionalCaretMovementIntent](a *updateTextSelectionAction) Action {
	return newTypedContextAction[T](a)
}

// codePointBoundary is a TextBoundary that finds the boundaries of the code
// points of a text, for an obscured field.
//
// Flutter: widgets/editable_text.dart _CodePointBoundary
type codePointBoundary struct {
	services.TextBoundaryDefaults
	text string
}

func newCodePointBoundary(text string) *codePointBoundary {
	b := &codePointBoundary{text: text}
	b.Self = b
	return b
}

// start is the byte offset where the code point that contains the byte starts.
func (b *codePointBoundary) start(position int) int {
	for position > 0 && position < len(b.text) && !utf8.RuneStart(b.text[position]) {
		position--
	}
	return position
}

func (b *codePointBoundary) GetLeadingTextBoundaryAt(position int) (int, bool) {
	if b.text == "" || position < 0 {
		return 0, false
	}
	if position == 0 {
		return 0, true
	}
	if position >= len(b.text) {
		return len(b.text), true
	}
	return b.start(position), true
}

func (b *codePointBoundary) GetTrailingTextBoundaryAt(position int) (int, bool) {
	if b.text == "" || position >= len(b.text) {
		return 0, false
	}
	if position < 0 {
		return 0, true
	}
	_, size := utf8.DecodeRuneInString(b.text[b.start(position):])
	return b.start(position) + size, true
}

func (b *codePointBoundary) GetTextBoundaryAt(position int) ui.TextRange {
	return b.TextBoundaryDefaults.GetTextBoundaryAt(position)
}

// deleteTextAction deletes text in a direction.
//
// Flutter: widgets/editable_text.dart _DeleteTextAction
type deleteTextAction[T DirectionalTextEditingIntent] struct {
	ContextActionBase
	state       *EditableTextState
	getBoundary func() services.TextBoundary
	apply       applyTextBoundary
}

func newDeleteTextAction[T DirectionalTextEditingIntent](state *EditableTextState, boundary func() services.TextBoundary, apply applyTextBoundary) *deleteTextAction[T] {
	a := &deleteTextAction[T]{state: state, getBoundary: boundary, apply: apply}
	InitContextActionBase(&a.ContextActionBase, a)
	return a
}

func (a *deleteTextAction[T]) IntentType() reflect.Type { return reflect.TypeFor[T]() }

func (a *deleteTextAction[T]) IsActionEnabled() bool {
	return !a.state.Widget().o.ReadOnly && a.state.value().Selection.IsValid()
}

func (a *deleteTextAction[T]) InvokeWithContext(intent Intent, bc BuildContext) any {
	state := a.state
	v := state.value()
	selection := v.Selection
	if !selection.IsValid() {
		return nil
	}
	atomic := state.characterBoundary()
	if !selection.IsCollapsed() {
		start, ok := atomic.GetLeadingTextBoundaryAt(selection.Start())
		if !ok {
			start = len(v.Text)
		}
		end, _ := atomic.GetTrailingTextBoundaryAt(selection.End() - 1)
		return ActionsInvoke(bc, NewReplaceTextIntent(v, "", ui.TextRange{Start: start, End: end}, services.SelectionChangedCauseKeyboard))
	}
	forward := intent.(DirectionalTextEditingIntent).IsForward()
	target := a.apply(selection.Base(), forward, a.getBoundary()).Offset
	var base int
	if forward {
		b, ok := atomic.GetLeadingTextBoundaryAt(selection.BaseOffset)
		if !ok {
			b = len(v.Text)
		}
		base = b
	} else {
		base, _ = atomic.GetTrailingTextBoundaryAt(selection.BaseOffset - 1)
	}
	rangeToDelete := services.NewTextSelection(base, target).Range()
	return ActionsInvoke(bc, NewReplaceTextIntent(v, "", rangeToDelete, services.SelectionChangedCauseKeyboard))
}

// updateTextSelectionAction moves or extends the selection by a boundary.
//
// Flutter: widgets/editable_text.dart _UpdateTextSelectionAction
type updateTextSelectionAction struct {
	ContextActionBase
	state                 *EditableTextState
	getBoundary           func() services.TextBoundary
	apply                 applyTextBoundary
	ignoreNonCollapsedSel bool
	isExpand, extentAtIdx bool
}

func newUpdateTextSelectionAction(state *EditableTextState, boundary func() services.TextBoundary, apply applyTextBoundary, ignoreNonCollapsed, isExpand, extentAtIndex bool) *updateTextSelectionAction {
	a := &updateTextSelectionAction{state: state, getBoundary: boundary, apply: apply, ignoreNonCollapsedSel: ignoreNonCollapsed, isExpand: isExpand, extentAtIdx: extentAtIndex}
	InitContextActionBase(&a.ContextActionBase, a)
	return a
}

func (a *updateTextSelectionAction) IntentType() reflect.Type {
	return reflect.TypeFor[DirectionalCaretMovementIntent]()
}

func (a *updateTextSelectionAction) IsActionEnabled() bool {
	return a.state.value().Selection.IsValid()
}

func (a *updateTextSelectionAction) isAtWordwrapUpstream(position ui.TextPosition) bool {
	end := ui.NewTextPosition(a.state.renderEditable().GetLineAtOffset(position).End(), ui.TextPositionOpts{Affinity: ui.TextAffinityUpstream})
	text := a.state.TextEditingValue().Text
	return end == position && end.Offset != len(text) && position.Offset < len(text) && text[position.Offset] != '\n'
}

func (a *updateTextSelectionAction) isAtWordwrapDownstream(position ui.TextPosition) bool {
	start := ui.NewTextPosition(a.state.renderEditable().GetLineAtOffset(position).Start())
	text := a.state.TextEditingValue().Text
	return start == position && start.Offset != 0 && text[position.Offset-1] != '\n'
}

func (a *updateTextSelectionAction) InvokeWithContext(i Intent, bc BuildContext) any {
	intent := i.(DirectionalCaretMovementIntent)
	state := a.state
	v := state.value()
	selection := v.Selection
	collapse := intent.CollapseSelection() || !state.Widget().SelectionEnabled()
	if !selection.IsCollapsed() && !a.ignoreNonCollapsedSel && collapse {
		offset := selection.Start()
		if intent.IsForward() {
			offset = selection.End()
		}
		return ActionsInvoke(bc, NewUpdateSelectionIntent(v, services.NewTextSelectionCollapsed(offset), services.SelectionChangedCauseKeyboard))
	}
	extent := selection.Extent()
	if intent.ContinuesAtWrap() {
		if intent.IsForward() && a.isAtWordwrapUpstream(extent) {
			extent = ui.NewTextPosition(extent.Offset)
		} else if !intent.IsForward() && a.isAtWordwrapDownstream(extent) {
			extent = ui.NewTextPosition(extent.Offset, ui.TextPositionOpts{Affinity: ui.TextAffinityUpstream})
		}
	}
	shouldTargetBase := a.isExpand
	if shouldTargetBase {
		if intent.IsForward() {
			shouldTargetBase = selection.BaseOffset > selection.ExtentOffset
		} else {
			shouldTargetBase = selection.BaseOffset < selection.ExtentOffset
		}
	}
	from := extent
	if shouldTargetBase {
		from = selection.Base()
	}
	newExtent := a.apply(from, intent.IsForward(), a.getBoundary())
	var newSelection services.TextSelection
	switch {
	case collapse || (!a.isExpand && newExtent.Offset == selection.BaseOffset):
		newSelection = services.NewTextSelectionFromPosition(newExtent)
	case a.isExpand:
		newSelection = selection.ExpandTo(newExtent, services.TextSelectionExpandOpts{ExtentAtIndex: a.extentAtIdx || selection.IsCollapsed()})
	default:
		newSelection = selection.ExtendTo(newExtent)
	}
	shouldCollapseToBase := intent.CollapseAtReversal() &&
		(selection.BaseOffset-selection.ExtentOffset)*(selection.BaseOffset-newSelection.ExtentOffset) < 0
	if shouldCollapseToBase {
		newSelection = services.NewTextSelectionFromPosition(selection.Base())
	}
	return ActionsInvoke(bc, NewUpdateSelectionIntent(v, newSelection, services.SelectionChangedCauseKeyboard))
}

// updateTextSelectionVerticallyAction moves the selection a line or a page up or
// down, keeping the horizontal position of the caret through consecutive moves.
//
// Flutter: widgets/editable_text.dart _UpdateTextSelectionVerticallyAction
type updateTextSelectionVerticallyAction struct {
	ContextActionBase
	state        *EditableTextState
	verticalRun  *rendering.VerticalCaretMovementRun
	runSelection foundation.Opt[services.TextSelection]
}

func newUpdateTextSelectionVerticallyAction(state *EditableTextState) *updateTextSelectionVerticallyAction {
	a := &updateTextSelectionVerticallyAction{state: state}
	InitContextActionBase(&a.ContextActionBase, a)
	return a
}

func (a *updateTextSelectionVerticallyAction) IntentType() reflect.Type {
	return reflect.TypeFor[DirectionalCaretMovementIntent]()
}

func (a *updateTextSelectionVerticallyAction) IsActionEnabled() bool {
	return a.state.value().Selection.IsValid()
}

// stopCurrentVerticalRunIfSelectionChanges ends the run when the selection is
// changed by something else.
//
// Flutter: widgets/editable_text.dart _UpdateTextSelectionVerticallyAction.stopCurrentVerticalRunIfSelectionChanges
func (a *updateTextSelectionVerticallyAction) stopCurrentVerticalRunIfSelectionChanges() {
	runSelection, ok := a.runSelection.Get()
	if !ok {
		return
	}
	a.runSelection = foundation.Some(a.state.value().Selection)
	current := a.state.Widget().controller.Selection()
	continueRun := current.IsValid() && current.IsCollapsed() && current.BaseOffset == runSelection.BaseOffset && current.ExtentOffset == runSelection.ExtentOffset
	if !continueRun {
		a.verticalRun = nil
		a.runSelection = foundation.None[services.TextSelection]()
	}
}

func (a *updateTextSelectionVerticallyAction) InvokeWithContext(i Intent, bc BuildContext) any {
	intent := i.(DirectionalCaretMovementIntent)
	state := a.state
	collapse := intent.CollapseSelection() || !state.Widget().SelectionEnabled()
	value := state.textEditingValueForTextLayoutMetrics()
	if !value.Selection.IsValid() {
		return nil
	}
	if a.verticalRun != nil && !a.verticalRun.IsValid() {
		a.verticalRun = nil
		a.runSelection = foundation.None[services.TextSelection]()
	}
	r := state.renderEditable()
	current := a.verticalRun
	if current == nil {
		sel, _ := r.Selection().Get()
		current = r.StartVerticalCaretMovement(sel.Extent())
	}
	var shouldMove bool
	if _, page := i.(ExtendSelectionVerticallyToAdjacentPageIntent); page {
		sign := -1.0
		if intent.IsForward() {
			sign = 1.0
		}
		shouldMove = current.MoveByOffset(sign * r.Size().Height)
	} else if intent.IsForward() {
		shouldMove = current.MoveNext()
	} else {
		shouldMove = current.MovePrevious()
	}
	var newExtent ui.TextPosition
	switch {
	case shouldMove:
		newExtent = current.Current()
	case intent.IsForward():
		newExtent = ui.NewTextPosition(len(value.Text))
	default:
		newExtent = ui.NewTextPosition(0)
	}
	var newSelection services.TextSelection
	if collapse {
		newSelection = services.NewTextSelectionFromPosition(newExtent)
	} else {
		newSelection = value.Selection.ExtendTo(newExtent)
	}
	ActionsInvoke(bc, NewUpdateSelectionIntent(value, newSelection, services.SelectionChangedCauseKeyboard))
	if state.value().Selection.Equal(newSelection) {
		a.verticalRun = current
		a.runSelection = foundation.Some(newSelection)
	}
	return nil
}

// selectAllAction selects all the text.
//
// Flutter: widgets/editable_text.dart _SelectAllAction
type selectAllAction struct {
	ContextActionBase
	state *EditableTextState
}

func newSelectAllAction(state *EditableTextState) *selectAllAction {
	a := &selectAllAction{state: state}
	InitContextActionBase(&a.ContextActionBase, a)
	return a
}

func (a *selectAllAction) IntentType() reflect.Type { return reflect.TypeFor[SelectAllTextIntent]() }

func (a *selectAllAction) InvokeWithContext(i Intent, bc BuildContext) any {
	if !a.state.Widget().SelectionEnabled() {
		return nil
	}
	v := a.state.value()
	return ActionsInvoke(bc, NewUpdateSelectionIntent(v, services.NewTextSelection(0, len(v.Text)), i.(SelectAllTextIntent).Cause))
}

// copySelectionAction copies or cuts the selection.
//
// Flutter: widgets/editable_text.dart _CopySelectionAction
type copySelectionAction struct {
	ContextActionBase
	state *EditableTextState
}

func newCopySelectionAction(state *EditableTextState) *copySelectionAction {
	a := &copySelectionAction{state: state}
	InitContextActionBase(&a.ContextActionBase, a)
	return a
}

func (a *copySelectionAction) IntentType() reflect.Type {
	return reflect.TypeFor[CopySelectionTextIntent]()
}

func (a *copySelectionAction) InvokeWithContext(i Intent, _ BuildContext) any {
	intent := i.(CopySelectionTextIntent)
	sel := a.state.value().Selection
	if !sel.IsValid() || sel.IsCollapsed() || !a.state.Widget().SelectionEnabled() {
		return nil
	}
	if intent.CollapseSelection {
		a.state.CutSelection(intent.Cause)
	} else {
		a.state.CopySelection(intent.Cause)
	}
	return nil
}

// pasteSelectionAction pastes the clipboard text.
//
// Flutter: widgets/editable_text.dart _PasteSelectionAction
type pasteSelectionAction struct {
	ContextActionBase
	state *EditableTextState
}

func newPasteSelectionAction(state *EditableTextState) *pasteSelectionAction {
	a := &pasteSelectionAction{state: state}
	InitContextActionBase(&a.ContextActionBase, a)
	return a
}

func (a *pasteSelectionAction) IntentType() reflect.Type { return reflect.TypeFor[PasteTextIntent]() }

func (a *pasteSelectionAction) InvokeWithContext(i Intent, _ BuildContext) any {
	if !a.state.Widget().SelectionEnabled() {
		return nil
	}
	a.state.PasteText(i.(PasteTextIntent).Cause)
	return nil
}

// tapOutsideAction unfocuses the field when a pointer goes down outside it.
//
// Flutter: widgets/editable_text.dart _EditableTextTapOutsideAction
type tapOutsideAction struct{ ContextActionBase }

func newTapOutsideAction() *tapOutsideAction {
	a := &tapOutsideAction{}
	InitContextActionBase(&a.ContextActionBase, a)
	return a
}

func (a *tapOutsideAction) IntentType() reflect.Type {
	return reflect.TypeFor[EditableTextTapOutsideIntent]()
}

func (a *tapOutsideAction) InvokeWithContext(i Intent, _ BuildContext) any {
	intent := i.(EditableTextTapOutsideIntent)
	if tapOutsideUnfocuses(foundation.DefaultTargetPlatform(), intent.PointerDownEvent.Kind()) {
		intent.FocusNode.Unfocus()
	}
	return nil
}

// tapOutsideUnfocuses says whether a pointer of the kind going down outside the
// field unfocuses it on the platform. It panics with the tag
// EditableTextTapOutsideTrackpad for a trackpad on a mobile platform.
func tapOutsideUnfocuses(platform foundation.TargetPlatform, kind ui.PointerDeviceKind) bool {
	switch platform {
	case foundation.TargetPlatformAndroid, foundation.TargetPlatformIOS, foundation.TargetPlatformFuchsia:
		switch kind {
		case ui.PointerDeviceKindTouch:
			// Taps on touch screens don't unfocus: scrolling must not either.
			return false
		case ui.PointerDeviceKindTrackpad:
			panic(errors.New("EditableTextTapOutsideTrackpad", "unexpected pointer down event for trackpad"))
		}
	}
	return true
}

// tapUpOutsideAction does nothing by default: a tap up outside the field.
//
// Flutter: widgets/editable_text.dart _EditableTextTapUpOutsideAction
type tapUpOutsideAction struct{ ContextActionBase }

func newTapUpOutsideAction() *tapUpOutsideAction {
	a := &tapUpOutsideAction{}
	InitContextActionBase(&a.ContextActionBase, a)
	return a
}

func (a *tapUpOutsideAction) IntentType() reflect.Type {
	return reflect.TypeFor[EditableTextTapUpOutsideIntent]()
}

func (a *tapUpOutsideAction) InvokeWithContext(Intent, BuildContext) any { return nil }
