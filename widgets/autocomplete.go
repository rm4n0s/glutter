package widgets

import (
	"fmt"
	"math"
	"reflect"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// AutocompleteOptionsBuilder is the type of the function that builds the
// options for the current text editing value. It calls deliver with the options,
// right away or later (on the UI thread, or through the binding's Post from
// another goroutine).
//
// Deviation: Flutter's builder returns a FutureOr<Iterable<T>>; Go has no
// futures, so the options are delivered through a callback (SPEC §12.10).
//
// Flutter: widgets/autocomplete.dart AutocompleteOptionsBuilder
type AutocompleteOptionsBuilder[T comparable] func(textEditingValue services.TextEditingValue, deliver func(options []T))

// AutocompleteOnSelected is the type of the callback that is called when an
// option is selected.
//
// Flutter: widgets/autocomplete.dart AutocompleteOnSelected
type AutocompleteOnSelected[T comparable] func(option T)

// AutocompleteOptionsViewBuilder is the type of the function that builds the
// view of the options.
//
// Flutter: widgets/autocomplete.dart AutocompleteOptionsViewBuilder
type AutocompleteOptionsViewBuilder[T comparable] func(bc BuildContext, onSelected AutocompleteOnSelected[T], options []T) Widget

// AutocompleteFieldViewBuilder is the type of the function that builds the
// field view. The textEditingController and the focusNode are those the
// RawAutocomplete listens to; onFieldSubmitted must be called when the field is
// submitted.
//
// Flutter: widgets/autocomplete.dart AutocompleteFieldViewBuilder
type AutocompleteFieldViewBuilder func(bc BuildContext, textEditingController *TextEditingController, focusNode *FocusNode, onFieldSubmitted foundation.VoidCallback) Widget

// AutocompleteOptionToString is the type of the function that turns an option
// into the string shown in the field.
//
// Flutter: widgets/autocomplete.dart AutocompleteOptionToString
type AutocompleteOptionToString[T comparable] func(option T) string

// OptionsViewOpenDirection is the direction in which to open the options view.
//
// Flutter: widgets/autocomplete.dart OptionsViewOpenDirection
type OptionsViewOpenDirection int

const (
	// OptionsViewOpenDirectionUnspecified is the zero value: the default, down.
	OptionsViewOpenDirectionUnspecified OptionsViewOpenDirection = iota
	// OptionsViewOpenDirectionUp opens the options view above the field.
	//
	// Flutter: widgets/autocomplete.dart OptionsViewOpenDirection.up
	OptionsViewOpenDirectionUp
	// OptionsViewOpenDirectionDown opens the options view below the field.
	//
	// Flutter: widgets/autocomplete.dart OptionsViewOpenDirection.down
	OptionsViewOpenDirectionDown
	// OptionsViewOpenDirectionMostSpace opens the options view in the direction
	// with the most space.
	//
	// Flutter: widgets/autocomplete.dart OptionsViewOpenDirection.mostSpace
	OptionsViewOpenDirectionMostSpace
)

// Name is the name of the direction.
//
// Flutter: widgets/autocomplete.dart OptionsViewOpenDirection.name
func (d OptionsViewOpenDirection) Name() string {
	switch d {
	case OptionsViewOpenDirectionUp:
		return "up"
	case OptionsViewOpenDirectionDown:
		return "down"
	case OptionsViewOpenDirectionMostSpace:
		return "mostSpace"
	}
	return "unspecified"
}

// String is "OptionsViewOpenDirection." and the name.
//
// Flutter: widgets/autocomplete.dart OptionsViewOpenDirection.toString
func (d OptionsViewOpenDirection) String() string { return "OptionsViewOpenDirection." + d.Name() }

// RawAutocompleteOpts are the optional parameters of NewRawAutocomplete.
//
// Flutter: widgets/autocomplete.dart RawAutocomplete.new
type RawAutocompleteOpts[T comparable] struct {
	// Key is the key of the widget.
	Key foundation.Key
	// OptionsViewOpenDirection is the direction in which to open the options
	// view; the default is down.
	OptionsViewOpenDirection OptionsViewOpenDirection
	// DisplayStringForOption returns the string to show in the field for an
	// option; the default is the default string of the option.
	DisplayStringForOption AutocompleteOptionToString[T]
	// FieldViewBuilder builds the field. If it is nil, the field is built outside
	// of the RawAutocomplete, with the FocusNode, the TextEditingController and
	// a key given, and its submit is passed to RawAutocompleteOnFieldSubmitted.
	FieldViewBuilder AutocompleteFieldViewBuilder
	// FocusNode is the node of the field; it must be given together with the
	// TextEditingController.
	FocusNode *FocusNode
	// OnSelected is called when an option is selected by the user.
	OnSelected AutocompleteOnSelected[T]
	// TextEditingController is the controller of the field; it must be given
	// together with the FocusNode.
	TextEditingController *TextEditingController
	// InitialValue is the initial value of the field; it can't be given together
	// with a TextEditingController.
	InitialValue foundation.Opt[services.TextEditingValue]
}

// RawAutocomplete is a widget for helping the user make a selection by entering
// some text and choosing from among a list of options.
//
// The user's text is passed to the optionsBuilder, which builds the options of
// the current text. When options are available, the optionsViewBuilder builds
// a view of them, shown over the other widgets of the app in the overlay.
//
// Deviation: the announcement of the search results to the semantics service is
// not ported (SPEC §12.10).
//
// Flutter: widgets/autocomplete.dart RawAutocomplete
type RawAutocomplete[T comparable] struct {
	StatefulWidget
	optionsViewBuilder AutocompleteOptionsViewBuilder[T]
	optionsBuilder     AutocompleteOptionsBuilder[T]
	o                  RawAutocompleteOpts[T]
}

// NewRawAutocomplete creates the widget. In debug builds it panics with the tag
// RawAutocompleteNoField when there is neither a field view builder nor a key,
// a focus node and a text editing controller, with RawAutocompleteFocusNodeMismatch
// when only one of the focus node and the controller is given, and with
// RawAutocompleteInitialValueAndController when both an initial value and a
// controller are given.
//
// Flutter: widgets/autocomplete.dart RawAutocomplete.new
func NewRawAutocomplete[T comparable](optionsViewBuilder AutocompleteOptionsViewBuilder[T], optionsBuilder AutocompleteOptionsBuilder[T],
	opts ...RawAutocompleteOpts[T]) *RawAutocomplete[T] {
	o := firstOpts(opts)
	if foundation.DebugMode {
		if o.FieldViewBuilder == nil && (o.Key == nil || o.FocusNode == nil || o.TextEditingController == nil) {
			panic(errors.New("RawAutocompleteNoField", "pass in a FieldViewBuilder, or otherwise create a separate field and pass in the FocusNode, TextEditingController, and a key; use the key with RawAutocompleteOnFieldSubmitted"))
		}
		if (o.FocusNode == nil) != (o.TextEditingController == nil) {
			panic(errors.New("RawAutocompleteFocusNodeMismatch", "the FocusNode and the TextEditingController must be given together"))
		}
		if o.TextEditingController != nil && o.InitialValue.IsSome() {
			panic(errors.New("RawAutocompleteInitialValueAndController", "TextEditingController and InitialValue cannot be simultaneously defined"))
		}
	}
	if o.DisplayStringForOption == nil {
		o.DisplayStringForOption = func(option T) string { return fmt.Sprint(option) }
	}
	if o.OptionsViewOpenDirection == OptionsViewOpenDirectionUnspecified {
		o.OptionsViewOpenDirection = OptionsViewOpenDirectionDown
	}
	return &RawAutocomplete[T]{StatefulWidget: NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), optionsViewBuilder: optionsViewBuilder, optionsBuilder: optionsBuilder, o: o}
}

// rawAutocompleteState is what RawAutocompleteOnFieldSubmitted needs of the
// state of any type of options.
type rawAutocompleteState interface{ onFieldSubmitted() }

// RawAutocompleteOnFieldSubmitted notifies the RawAutocomplete with the key that
// the field was submitted. Use it when the field is built outside of the
// RawAutocomplete.
//
// Flutter: widgets/autocomplete.dart RawAutocomplete.onFieldSubmitted
func RawAutocompleteOnFieldSubmitted(key *GlobalKey[State]) {
	st, ok := key.CurrentState()
	if !ok {
		panic(errors.New("RawAutocompleteKeyNotMounted", "the key of the RawAutocomplete is not mounted"))
	}
	st.(rawAutocompleteState).onFieldSubmitted()
}

// CreateState creates the state.
//
// Flutter: widgets/autocomplete.dart RawAutocomplete.createState
func (w *RawAutocomplete[T]) CreateState() State {
	return &rawAutocompleteStateOf[T]{
		optionsViewController:  NewOverlayPortalController(OverlayPortalControllerOpts{DebugLabel: "_RawAutocompleteState"}),
		highlightedOptionIndex: foundation.NewValueNotifier(0),
	}
}

// autocompletePageSize is the number of options to scroll by "page", such as
// when using the page up/down keys.
const autocompletePageSize = 4

// autocompleteMinUsableHeight is a big enough height for about one item in the
// default options view. The assumption is that the user likely wants the list of
// options to move to stay on the screen rather than get any smaller than this.
//
// Flutter: widgets/autocomplete.dart _RawAutocompleteState._kMinUsableHeight
const autocompleteMinUsableHeight = MinInteractiveDimension

type rawAutocompleteStateOf[T comparable] struct {
	StateOf[*RawAutocomplete[T]]
	optionsViewController *OverlayPortalController

	hasFocus  bool
	selecting bool

	internalTextEditingController *TextEditingController
	internalFocusNode             *FocusNode
	controllerHandle              foundation.ListenerHandle
	focusHandle                   foundation.ListenerHandle

	actionMap ActionMap
	options   []T
	selection foundation.Opt[T]
	// lastFieldText is nil initially so when this widget gets focused for the
	// first time it will try to run the options builder.
	lastFieldText          foundation.Opt[string]
	highlightedOptionIndex *foundation.ValueNotifier[int]
	// onChangedCallID avoids a situation where options is updated by an older
	// call when multiple onChangedField calls are running simultaneously.
	onChangedCallID int
}

func (s *rawAutocompleteStateOf[T]) textEditingController() *TextEditingController {
	if c := s.Widget().o.TextEditingController; c != nil {
		return c
	}
	if s.internalTextEditingController == nil {
		s.internalTextEditingController = NewTextEditingController()
		s.internalTextEditingController.AddListener(s.onChangedField)
	}
	return s.internalTextEditingController
}

func (s *rawAutocompleteStateOf[T]) focusNode() *FocusNode {
	if f := s.Widget().o.FocusNode; f != nil {
		return f
	}
	if s.internalFocusNode == nil {
		s.internalFocusNode = NewFocusNode()
		s.internalFocusNode.AddListener(s.onFocusChange)
	}
	return s.internalFocusNode
}

func autocompleteShortcuts() ShortcutMap {
	m := ShortcutMap{
		{Activator: NewSingleActivator(services.LogicalKeyboardKeyArrowUp), Intent: AutocompletePreviousOptionIntent{}},
		{Activator: NewSingleActivator(services.LogicalKeyboardKeyArrowDown), Intent: AutocompleteNextOptionIntent{}},
		{Activator: NewSingleActivator(services.LogicalKeyboardKeyPageUp), Intent: AutocompletePreviousPageOptionIntent{}},
		{Activator: NewSingleActivator(services.LogicalKeyboardKeyPageDown), Intent: AutocompleteNextPageOptionIntent{}},
	}
	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformIOS, foundation.TargetPlatformMacOS:
		m = append(m,
			ShortcutEntry{Activator: NewSingleActivator(services.LogicalKeyboardKeyArrowUp, SingleActivatorOpts{Meta: true}), Intent: AutocompleteFirstOptionIntent{}},
			ShortcutEntry{Activator: NewSingleActivator(services.LogicalKeyboardKeyArrowDown, SingleActivatorOpts{Meta: true}), Intent: AutocompleteLastOptionIntent{}})
	default:
		m = append(m,
			ShortcutEntry{Activator: NewSingleActivator(services.LogicalKeyboardKeyArrowUp, SingleActivatorOpts{Control: true}), Intent: AutocompleteFirstOptionIntent{}},
			ShortcutEntry{Activator: NewSingleActivator(services.LogicalKeyboardKeyArrowDown, SingleActivatorOpts{Control: true}), Intent: AutocompleteLastOptionIntent{}})
	}
	return m
}

func (s *rawAutocompleteStateOf[T]) canShowOptionsView() bool {
	return s.focusNode().HasFocus() && len(s.options) > 0
}

func (s *rawAutocompleteStateOf[T]) onFocusChange() {
	if s.focusNode().HasFocus() != s.hasFocus {
		s.hasFocus = s.focusNode().HasFocus()
		// Gaining focus can open the options view (if there are options). Losing
		// focus always closes it.
		s.updateOptionsViewVisibility()
	}
}

func (s *rawAutocompleteStateOf[T]) updateOptionsViewVisibility() {
	if s.canShowOptionsView() {
		s.optionsViewController.Show()
	} else if s.optionsViewController.IsShowing() {
		s.optionsViewController.Hide()
	}
}

// onChangedField is called when the text editing controller changes.
func (s *rawAutocompleteStateOf[T]) onChangedField() {
	// During a selection, changes to the field text should not trigger options
	// update.
	if s.selecting {
		return
	}
	value := s.textEditingController().Value()
	// Makes sure that options change only when content of the field changes.
	shouldUpdateOptions := false
	if s.lastFieldText.IsNone() || value.Text != s.lastFieldText.Value() {
		shouldUpdateOptions = true
		s.onChangedCallID++
	}
	s.lastFieldText = foundation.Some(value.Text)
	callID := s.onChangedCallID
	s.optionsBuilder()(value, func(options []T) {
		if !s.Mounted() {
			return
		}
		// Makes sure that previous call results do not replace new ones.
		if callID != s.onChangedCallID || !shouldUpdateOptions {
			return
		}
		s.options = options
		s.updateHighlight(s.highlightedOptionIndex.Value())
		if sel, ok := s.selection.Get(); ok && value.Text != s.Widget().o.DisplayStringForOption(sel) {
			s.selection = foundation.None[T]()
		}
		s.updateOptionsViewVisibility()
	})
}

func (s *rawAutocompleteStateOf[T]) optionsBuilder() AutocompleteOptionsBuilder[T] {
	return s.Widget().optionsBuilder
}

// onFieldSubmitted is called from the field view when the user submits the field.
func (s *rawAutocompleteStateOf[T]) onFieldSubmitted() {
	if s.optionsViewController.IsShowing() {
		s.selectOption(s.options[s.highlightedOptionIndex.Value()])
	}
}

// selectOption selects the given option and updates the widget.
func (s *rawAutocompleteStateOf[T]) selectOption(next T) {
	if sel, ok := s.selection.Get(); ok && sel == next {
		return
	}
	s.selecting = true
	s.selection = foundation.Some(next)
	selectionString := s.Widget().o.DisplayStringForOption(next)
	s.textEditingController().SetValue(services.TextEditingValue{
		Selection: services.NewTextSelectionCollapsed(len(selectionString)), Text: selectionString,
	})
	s.lastFieldText = foundation.Some(selectionString)
	if f := s.Widget().o.OnSelected; f != nil {
		f(next)
	}
	if s.optionsViewController.IsShowing() {
		// Close the options view after a selection is made.
		s.optionsViewController.Hide()
	}
	s.selecting = false
}

func (s *rawAutocompleteStateOf[T]) updateHighlight(nextIndex int) {
	if len(s.options) == 0 {
		s.highlightedOptionIndex.SetValue(0)
		return
	}
	s.highlightedOptionIndex.SetValue(min(max(nextIndex, 0), len(s.options)-1))
}

func (s *rawAutocompleteStateOf[T]) highlightOption(index int) {
	if foundation.DebugMode && !s.canShowOptionsView() {
		panic(errors.New("RawAutocompleteOptionsNotShowable", "the options view can not be shown, so no option can be highlighted"))
	}
	s.updateOptionsViewVisibility()
	s.updateHighlight(index)
}

func (s *rawAutocompleteStateOf[T]) hideOptions(i DismissIntent) any {
	if s.optionsViewController.IsShowing() {
		s.optionsViewController.Hide()
		return nil
	}
	return ActionsInvoke(s.Context(), i)
}

func (s *rawAutocompleteStateOf[T]) buildOptionsView(bc BuildContext, layoutInfo OverlayChildLayoutInfo) Widget {
	if layoutInfo.ChildPaintTransform.Determinant() == 0.0 {
		// The child is not visible.
		return NewSizedBoxShrink(nil)
	}
	fieldSize := layoutInfo.ChildSize
	invertTransform := layoutInfo.ChildPaintTransform.Clone()
	invertTransform.Invert()

	mediaQueryPadding := MediaQueryPaddingOf(bc)
	viewInsets := MediaQueryViewInsetsOf(bc)
	overlayRect := mediaQueryPadding.DeflateRect(viewInsets.DeflateRect(ui.NewRectFromLTWH(0, 0, layoutInfo.OverlaySize.Width, layoutInfo.OverlaySize.Height)))

	// This may not work well if the paint transform has rotation in it.
	// TransformRect returns the bounding rect of the rotated overlay rect.
	overlayRectInField := invertTransform.TransformRect(overlayRect)

	spaceAbove := -overlayRectInField.Top
	spaceBelow := overlayRectInField.Bottom - fieldSize.Height
	var opensUp bool
	switch s.Widget().o.OptionsViewOpenDirection {
	case OptionsViewOpenDirectionUp:
		opensUp = true
	case OptionsViewOpenDirectionMostSpace:
		opensUp = spaceAbove > spaceBelow
	}
	optionsViewMaxHeight := overlayRectInField.Bottom - fieldSize.Height
	if opensUp {
		optionsViewMaxHeight = -overlayRectInField.Top
	}
	boundingBox := ui.Size{Width: fieldSize.Width, Height: math.Max(optionsViewMaxHeight, autocompleteMinUsableHeight)}
	originY := overlayRectInField.Bottom - boundingBox.Height
	if opensUp {
		originY = overlayRectInField.Top
	}
	transform := layoutInfo.ChildPaintTransform.Clone()
	transform.Translate(0, float32(originY), 0)
	child := NewBuilder(func(bc BuildContext) Widget {
		return s.Widget().optionsViewBuilder(bc, s.selectOption, s.options)
	})
	alignment := painting.AlignmentGeometry(painting.AlignmentDirectionalTopStart)
	if opensUp {
		alignment = painting.AlignmentDirectionalBottomStart
	}
	return NewTransform(transform,
		NewAlign(NewConstrainedBox(rendering.NewBoxConstraintsTight(boundingBox),
			NewAlign(NewTextFieldTapRegion(
				NewAutocompleteHighlightedOption(s.highlightedOptionIndex,
					// Exclude the options overlay from the ambient focus traversal
					// tree. Autocomplete options are navigated by arrow keys (via the
					// widget's own shortcuts) and selected via Enter or tap, so they
					// don't participate in TAB traversal. Without this, TAB from the
					// field would detour into focusable items in the options overlay
					// instead of advancing to the next form field.
					NewExcludeFocus(child))),
				AlignOpts{Alignment: alignment})),
			AlignOpts{Alignment: painting.AlignmentTopLeft}))
}

// InitState starts to listen to the controller and the focus node.
//
// Flutter: widgets/autocomplete.dart _RawAutocompleteState.initState
func (s *rawAutocompleteStateOf[T]) InitState() {
	w := s.Widget()
	initial := w.o.TextEditingController
	if initial == nil {
		if v, ok := w.o.InitialValue.Get(); ok {
			s.internalTextEditingController = NewTextEditingControllerFromValue(v)
		} else {
			s.internalTextEditingController = NewTextEditingController()
		}
		initial = s.internalTextEditingController
	}
	s.controllerHandle = initial.AddListener(s.onChangedField)
	s.hasFocus = s.focusNode().HasFocus()
	if w.o.FocusNode != nil {
		s.focusHandle = w.o.FocusNode.AddListener(s.onFocusChange)
	}
	s.actionMap = s.buildActions()
}

func (s *rawAutocompleteStateOf[T]) buildActions() ActionMap {
	enabled := s.canShowOptionsView
	m := ActionMap{}
	add := func(a Action) { m[a.IntentType()] = a }
	add(newAutocompleteCallbackAction(func(AutocompletePreviousOptionIntent) { s.highlightOption(s.highlightedOptionIndex.Value() - 1) }, enabled))
	add(newAutocompleteCallbackAction(func(AutocompleteNextOptionIntent) { s.highlightOption(s.highlightedOptionIndex.Value() + 1) }, enabled))
	add(newAutocompleteCallbackAction(func(AutocompleteFirstOptionIntent) { s.highlightOption(0) }, enabled))
	add(newAutocompleteCallbackAction(func(AutocompleteLastOptionIntent) { s.highlightOption(len(s.options) - 1) }, enabled))
	add(newAutocompleteCallbackAction(func(AutocompleteNextPageOptionIntent) {
		s.highlightOption(s.highlightedOptionIndex.Value() + autocompletePageSize)
	}, enabled))
	add(newAutocompleteCallbackAction(func(AutocompletePreviousPageOptionIntent) {
		s.highlightOption(s.highlightedOptionIndex.Value() - autocompletePageSize)
	}, enabled))
	d := NewCallbackAction(s.hideOptions)
	m[d.IntentType()] = d
	return m
}

// DidUpdateWidget moves the listeners to the new controller and focus node.
//
// Flutter: widgets/autocomplete.dart _RawAutocompleteState.didUpdateWidget
func (s *rawAutocompleteStateOf[T]) DidUpdateWidget(old *RawAutocomplete[T]) {
	w := s.Widget()
	if old.o.TextEditingController != w.o.TextEditingController {
		if old.o.TextEditingController != nil {
			old.o.TextEditingController.RemoveListener(s.controllerHandle)
		}
		if old.o.TextEditingController == nil && s.internalTextEditingController != nil {
			s.internalTextEditingController.RemoveListener(s.controllerHandle)
			s.internalTextEditingController.Dispose()
			s.internalTextEditingController = nil
		}
		if w.o.TextEditingController != nil {
			s.controllerHandle = w.o.TextEditingController.AddListener(s.onChangedField)
		}
	}
	if old.o.FocusNode != w.o.FocusNode {
		if old.o.FocusNode != nil {
			old.o.FocusNode.RemoveListener(s.focusHandle)
		}
		if old.o.FocusNode == nil && s.internalFocusNode != nil {
			s.internalFocusNode.Dispose()
			s.internalFocusNode = nil
		}
		if w.o.FocusNode != nil {
			s.focusHandle = w.o.FocusNode.AddListener(s.updateOptionsViewVisibility)
		}
	}
}

// Dispose releases the listeners and the controllers it made.
//
// Flutter: widgets/autocomplete.dart _RawAutocompleteState.dispose
func (s *rawAutocompleteStateOf[T]) Dispose() {
	w := s.Widget()
	if w.o.TextEditingController != nil {
		w.o.TextEditingController.RemoveListener(s.controllerHandle)
	}
	if s.internalTextEditingController != nil {
		s.internalTextEditingController.Dispose()
	}
	if w.o.FocusNode != nil {
		w.o.FocusNode.RemoveListener(s.focusHandle)
	}
	if s.internalFocusNode != nil {
		s.internalFocusNode.Dispose()
	}
	s.highlightedOptionIndex.Dispose()
}

// Build builds the field and the portal of the options view.
//
// Flutter: widgets/autocomplete.dart _RawAutocompleteState.build
func (s *rawAutocompleteStateOf[T]) Build(bc BuildContext) Widget {
	w := s.Widget()
	var fieldView Widget
	if w.o.FieldViewBuilder != nil {
		fieldView = w.o.FieldViewBuilder(bc, s.textEditingController(), s.focusNode(), s.onFieldSubmitted)
	}
	if fieldView == nil {
		// Horizontally expand to make sure the options view's width won't be zero.
		fieldView = NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(math.Inf(1)), Height: foundation.Some(0.0)})
	}
	return NewOverlayPortalOverlayChildLayoutBuilder(s.optionsViewController, s.buildOptionsView,
		NewTextFieldTapRegion(NewShortcuts(autocompleteShortcuts(), NewActions(s.actionMap, fieldView))))
}

// autocompleteCallbackAction is Flutter's _AutocompleteCallbackAction: its
// enabled state determines whether the action will consume the key shortcut or
// let it continue on to the underlying text field. They should only be enabled
// when the options are showing so shortcuts can be used to navigate them.
//
// Flutter: widgets/autocomplete.dart _AutocompleteCallbackAction
type autocompleteCallbackAction[T Intent] struct {
	ActionBase
	onInvoke  func(intent T)
	isEnabled func() bool
}

func newAutocompleteCallbackAction[T Intent](onInvoke func(intent T), isEnabled func() bool) *autocompleteCallbackAction[T] {
	a := &autocompleteCallbackAction[T]{onInvoke: onInvoke, isEnabled: isEnabled}
	InitActionBase(&a.ActionBase, a)
	return a
}

func (a *autocompleteCallbackAction[T]) IntentType() reflect.Type { return reflect.TypeFor[T]() }

func (a *autocompleteCallbackAction[T]) Invoke(intent Intent) any {
	a.onInvoke(intent.(T))
	return nil
}

func (a *autocompleteCallbackAction[T]) IsEnabled(Intent) bool { return a.isEnabled() }

func (a *autocompleteCallbackAction[T]) ConsumesKey(intent Intent) bool { return a.IsEnabled(intent) }

// AutocompletePreviousOptionIntent is an intent to highlight the previous option
// in the list of options.
//
// Flutter: widgets/autocomplete.dart AutocompletePreviousOptionIntent
type AutocompletePreviousOptionIntent struct{ IntentBase }

// AutocompleteNextOptionIntent is an intent to highlight the next option in the
// list of options.
//
// Flutter: widgets/autocomplete.dart AutocompleteNextOptionIntent
type AutocompleteNextOptionIntent struct{ IntentBase }

// AutocompleteFirstOptionIntent is an intent to highlight the first option in
// the list of options.
//
// Flutter: widgets/autocomplete.dart AutocompleteFirstOptionIntent
type AutocompleteFirstOptionIntent struct{ IntentBase }

// AutocompleteLastOptionIntent is an intent to highlight the last option in the
// list of options.
//
// Flutter: widgets/autocomplete.dart AutocompleteLastOptionIntent
type AutocompleteLastOptionIntent struct{ IntentBase }

// AutocompleteNextPageOptionIntent is an intent to highlight the option one page
// after the current highlighted option.
//
// Flutter: widgets/autocomplete.dart AutocompleteNextPageOptionIntent
type AutocompleteNextPageOptionIntent struct{ IntentBase }

// AutocompletePreviousPageOptionIntent is an intent to highlight the option one
// page before the current highlighted option.
//
// Flutter: widgets/autocomplete.dart AutocompletePreviousPageOptionIntent
type AutocompletePreviousPageOptionIntent struct{ IntentBase }

// AutocompleteHighlightedOption is an inherited widget that tells which option
// of the options view is highlighted.
//
// Flutter: widgets/autocomplete.dart AutocompleteHighlightedOption
type AutocompleteHighlightedOption struct {
	InheritedNotifier
	highlightIndex *foundation.ValueNotifier[int]
}

// NewAutocompleteHighlightedOption creates the widget.
//
// Flutter: widgets/autocomplete.dart AutocompleteHighlightedOption.new
func NewAutocompleteHighlightedOption(highlightIndexNotifier *foundation.ValueNotifier[int], child Widget, opts ...WidgetOpts) *AutocompleteHighlightedOption {
	return &AutocompleteHighlightedOption{NewInheritedNotifierBase(highlightIndexNotifier, child, opts...), highlightIndexNotifier}
}

// AutocompleteHighlightedOptionOf returns the index of the highlighted option
// of the closest AutocompleteHighlightedOption above the context, or 0.
//
// Flutter: widgets/autocomplete.dart AutocompleteHighlightedOption.of
func AutocompleteHighlightedOptionOf(bc BuildContext) int {
	if w, ok := DependOn[*AutocompleteHighlightedOption](bc); ok && w.highlightIndex != nil {
		return w.highlightIndex.Value()
	}
	return 0
}
