package widgets

import (
	"slices"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// AppPrivateCommandCallback is the type of the function that EditableText calls
// for a private command of the input method.
//
// Flutter: widgets/editable_text.dart AppPrivateCommandCallback
type AppPrivateCommandCallback func(action string, data map[string]any)

// SelectionChangedCallback is the type of the function that EditableText calls
// when the selection changes.
//
// Flutter: widgets/editable_text.dart SelectionChangedCallback
type SelectionChangedCallback func(selection services.TextSelection, cause services.SelectionChangedCause)

// EditableTextContextMenuBuilder builds the context menu of an EditableText.
//
// Flutter: widgets/editable_text.dart EditableTextContextMenuBuilder
type EditableTextContextMenuBuilder func(bc BuildContext, state *EditableTextState) Widget

// EditableTextGroupID is the default group ID of the tap region of an
// EditableText: Flutter's EditableText type. It is the group of
// NewTextFieldTapRegion.
//
// Flutter: widgets/editable_text.dart EditableText.groupId
var EditableTextGroupID = TextFieldTapRegionGroupID{}

// EditableTextOpts are the optional parameters of NewEditableText.
//
// Flutter: widgets/editable_text.dart EditableText.new
type EditableTextOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// ReadOnly makes the text not editable.
	ReadOnly bool
	// ObscuringCharacter replaces the text when it is obscured; it must be one
	// character. The default is '•'.
	ObscuringCharacter string
	// ObscureText hides the text, for passwords. An obscured field can't be
	// multiline.
	ObscureText bool
	// Autocorrect says whether the input method corrects the text; the default is
	// true, or inferred from AutofillHints.
	Autocorrect foundation.Opt[bool]
	// SmartDashesType says whether dashes are replaced; the default is enabled,
	// or disabled for an obscured field.
	SmartDashesType services.SmartDashesType
	// SmartQuotesType says whether quotes are replaced; the default is enabled,
	// or disabled for an obscured field.
	SmartQuotesType services.SmartQuotesType
	// DisableSuggestions turns the suggestions of the input method off (Flutter's
	// enableSuggestions is true by default).
	DisableSuggestions bool
	// StrutStyle is the strut of the text; the default is made from the style.
	StrutStyle *painting.StrutStyle
	// TextAlign is how the text is aligned; the default is start.
	TextAlign ui.TextAlign
	// TextDirection is the direction of the text; the default is the one of the
	// Directionality.
	TextDirection ui.TextDirection
	// TextCapitalization says how the input method capitalizes.
	TextCapitalization services.TextCapitalization
	// Locale selects the glyphs and the language of the input.
	Locale foundation.Opt[ui.Locale]
	// TextScaler scales the text; the default is the one of the MediaQuery.
	TextScaler painting.TextScaler
	// MaxLines is the maximum number of lines; the default is 1 and
	// MaxLinesUnlimited (Flutter's null) is no limit.
	MaxLines foundation.Opt[int]
	// MinLines is the minimum number of lines.
	MinLines foundation.Opt[int]
	// Expands makes the field as tall as its parent allows; MaxLines and
	// MinLines must then be unset.
	Expands bool
	// DisableForceLine lets a single-line field be as narrow as its text
	// (Flutter's forceLine is true by default).
	DisableForceLine bool
	// TextHeightBehavior says how the line height applies to the first and last
	// lines; the default is the one of DefaultTextHeightBehavior.
	TextHeightBehavior *ui.TextHeightBehavior
	// TextWidthBasis says how the width of the text is measured; the default is
	// parent.
	TextWidthBasis painting.TextWidthBasis
	// Autofocus asks for the focus when the field is built.
	Autofocus bool
	// ShowCursor says whether the caret is shown; the default is !ReadOnly.
	ShowCursor foundation.Opt[bool]
	// ShowSelectionHandles says whether the selection handles are shown.
	ShowSelectionHandles bool
	// SelectionColor is the color of the selected text's background.
	SelectionColor foundation.Opt[ui.Color]
	// KeyboardType is the kind of text the input method offers; the default is
	// inferred from AutofillHints and MaxLines.
	KeyboardType foundation.Opt[services.TextInputType]
	// TextInputAction is the action of the Enter key; the default is newline for a
	// multiline field and done otherwise.
	TextInputAction services.TextInputAction
	// OnChanged is called when the text changes.
	OnChanged func(text string)
	// OnEditingComplete is called when the user finishes editing, such as with
	// Enter. Without it, the default is to clear the composing region and unfocus
	// (or move the focus for next and previous).
	OnEditingComplete func()
	// OnSubmitted is called with the text when the user finishes editing.
	OnSubmitted func(text string)
	// OnAppPrivateCommand is called for a private command of the input method.
	OnAppPrivateCommand AppPrivateCommandCallback
	// OnSelectionChanged is called when the selection changes.
	OnSelectionChanged SelectionChangedCallback
	// SelectionControls build the selection handles; nil for none.
	SelectionControls TextSelectionControls
	// OnSelectionHandleTapped is called when a selection handle is tapped.
	OnSelectionHandleTapped func()
	// GroupID is the group of the tap region of the field; the default is
	// EditableTextGroupID.
	GroupID any
	// OnTapOutside is called when a pointer goes down outside the field's tap
	// region; the default unfocuses the field.
	OnTapOutside TapRegionCallback
	// OnTapUpOutside is called when a pointer goes up outside the field's tap
	// region.
	OnTapUpOutside TapRegionUpCallback
	// InputFormatters change the text the user enters.
	InputFormatters []services.TextInputFormatter
	// MouseCursor is the cursor over the field; the default is the text cursor.
	MouseCursor services.MouseCursor
	// RendererIgnoresPointer makes the render object ignore taps and long
	// presses, for a gesture detector above it.
	RendererIgnoresPointer bool
	// CursorWidth is the width of the caret; the default is 2.
	CursorWidth foundation.Opt[float64]
	// CursorHeight is the height of the caret; the default is the line height.
	CursorHeight foundation.Opt[float64]
	// CursorRadius rounds the corners of the caret.
	CursorRadius foundation.Opt[ui.Radius]
	// CursorOpacityAnimates makes the caret fade instead of blink.
	//
	// Deviation: the fade is not ported; the caret blinks (SPEC §12.10).
	CursorOpacityAnimates bool
	// CursorOffset moves the caret from where the text puts it.
	CursorOffset foundation.Opt[ui.Offset]
	// PaintCursorAboveText paints the caret over the text.
	PaintCursorAboveText bool
	// SelectionHeightStyle is the height of the boxes of the selection; the
	// default is includeLineSpacingMiddle.
	SelectionHeightStyle ui.BoxHeightStyle
	// SelectionWidthStyle is the width of the boxes of the selection; the default
	// is max.
	SelectionWidthStyle ui.BoxWidthStyle
	// ScrollPadding is the space kept around the caret when the field scrolls to
	// show it; the default is 20 on all sides.
	ScrollPadding foundation.Opt[painting.EdgeInsets]
	// KeyboardAppearance is the brightness of the keyboard of the input method.
	KeyboardAppearance ui.Brightness
	// DragStartBehavior is where a drag starts; the default is start.
	DragStartBehavior gestures.DragStartBehavior
	// EnableInteractiveSelection says whether the user can select text; the
	// default is true unless the text is read-only and obscured.
	EnableInteractiveSelection foundation.Opt[bool]
	// SelectAllOnFocus selects the text when the field gets the focus; the
	// default is true on desktop, for a single-line field.
	SelectAllOnFocus foundation.Opt[bool]
	// ScrollController controls the scrolling of the field.
	ScrollController ScrollController
	// ScrollPhysics are the physics of the scrolling of the field.
	ScrollPhysics ScrollPhysics
	// AutocorrectionTextRectColor is the color behind the text the input method
	// corrects.
	AutocorrectionTextRectColor foundation.Opt[ui.Color]
	// AutofillHints say what the field is for.
	AutofillHints []string
	// AutofillClient is the client to autofill instead of the field.
	AutofillClient services.AutofillClient
	// ClipBehavior is how the text is clipped; the default is hard edge.
	ClipBehavior ui.Clip
	// ScrollBehavior is the behavior of the scrolling; the default hides the
	// overscroll and shows scrollbars for multiline fields.
	ScrollBehavior ScrollBehavior
	// DisableIMEPersonalizedLearning turns the learning of the input method off
	// (Flutter's enableIMEPersonalizedLearning is true by default).
	DisableIMEPersonalizedLearning bool
	// ContextMenuBuilder builds the context menu.
	ContextMenuBuilder EditableTextContextMenuBuilder
	// UndoController controls the undo history from outside.
	UndoController *UndoHistoryController
	// HintLocales are the languages the user is expected to type in.
	HintLocales []ui.Locale
	// EnableInlinePrediction turns the inline prediction of the input method on
	// or off.
	EnableInlinePrediction foundation.Opt[bool]
	// MagnifierConfiguration is reserved for the magnifier of the selection
	// handles.
	//
	// Deviation: the magnifier is not ported (SPEC §12.10).
	MagnifierConfiguration any
}

// EditableText is a basic text input field: it shows the text of a
// TextEditingController, the caret and the selection, handles the keys of the
// text editing shortcuts, and talks to the input method. Material's TextField is
// built on it.
//
// Deviation: spell check, the stylus handwriting, Live Text, the processing of
// the text by other apps, the floating cursor, the magnifier and the autofill
// groups are not ported (SPEC §12.10).
//
// Flutter: widgets/editable_text.dart EditableText
type EditableText struct {
	StatefulWidget
	controller            *TextEditingController
	focusNode             *FocusNode
	style                 *painting.TextStyle
	cursorColor           ui.Color
	backgroundCursorColor ui.Color
	o                     EditableTextOpts
	// The parameters that default from others.
	autocorrect                bool
	smartDashesType            services.SmartDashesType
	smartQuotesType            services.SmartQuotesType
	keyboardType               services.TextInputType
	inputFormatters            []services.TextInputFormatter
	showCursor                 bool
	enableInteractiveSelection bool
	selectAllOnFocus           bool
	strutStyle                 *painting.StrutStyle
	selectionHeightStyle       ui.BoxHeightStyle
	selectionWidthStyle        ui.BoxWidthStyle
}

// NewEditableText creates the widget. In debug builds it panics with the tag
// EditableTextObscuringCharacterInvalid when the obscuring character is not one
// character, EditableTextMinLinesInvalid when the minimum number of lines is not
// positive, EditableTextMinLinesGreaterThanMaxLines when it is more than the
// maximum, EditableTextExpandsWithLines when the field expands and has a number
// of lines, and EditableTextObscuredMultiline for an obscured multiline field.
//
// Flutter: widgets/editable_text.dart EditableText.new
func NewEditableText(controller *TextEditingController, focusNode *FocusNode, style *painting.TextStyle, cursorColor, backgroundCursorColor ui.Color, opts ...EditableTextOpts) *EditableText {
	o := firstOpts(opts)
	if o.ObscuringCharacter == "" {
		o.ObscuringCharacter = "•"
	}
	maxLines := o.MaxLines
	switch {
	case maxLines.IsNone():
		maxLines = foundation.Some(1)
	case maxLines.Value() == rendering.MaxLinesUnlimited:
		maxLines = foundation.None[int]()
	}
	o.MaxLines = maxLines
	if foundation.DebugMode {
		if foundation.GraphemeCount(o.ObscuringCharacter) != 1 {
			panic(errors.New("EditableTextObscuringCharacterInvalid", "the obscuring character must be one character", "obscuringCharacter", o.ObscuringCharacter))
		}
		if m, ok := o.MinLines.Get(); ok && m <= 0 {
			panic(errors.New("EditableTextMinLinesInvalid", "minLines must be positive", "minLines", m))
		}
		if mx, ok := maxLines.Get(); ok && o.MinLines.IsSome() && mx < o.MinLines.Value() {
			panic(errors.New("EditableTextMinLinesGreaterThanMaxLines", "minLines can't be greater than maxLines", "minLines", o.MinLines.Value(), "maxLines", mx))
		}
		if o.Expands && (maxLines.IsSome() || o.MinLines.IsSome()) {
			panic(errors.New("EditableTextExpandsWithLines", "minLines and maxLines must be unset when expands is true"))
		}
		if o.ObscureText && (maxLines.IsNone() || maxLines.Value() != 1) {
			panic(errors.New("EditableTextObscuredMultiline", "obscured fields cannot be multiline"))
		}
	}
	w := &EditableText{
		StatefulWidget: NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), controller: controller, focusNode: focusNode, style: style,
		cursorColor: cursorColor, backgroundCursorColor: backgroundCursorColor, o: o,
	}
	w.autocorrect = o.Autocorrect.OrElse(true)
	w.smartDashesType = o.SmartDashesType
	if w.smartDashesType == services.SmartDashesTypeUnspecified {
		w.smartDashesType = services.SmartDashesTypeEnabled
		if o.ObscureText {
			w.smartDashesType = services.SmartDashesTypeDisabled
		}
	}
	w.smartQuotesType = o.SmartQuotesType
	if w.smartQuotesType == services.SmartQuotesTypeUnspecified {
		w.smartQuotesType = services.SmartQuotesTypeEnabled
		if o.ObscureText {
			w.smartQuotesType = services.SmartQuotesTypeDisabled
		}
	}
	w.keyboardType = o.KeyboardType.OrElse(inferKeyboardType(o.AutofillHints, maxLines))
	w.inputFormatters = o.InputFormatters
	if m, ok := maxLines.Get(); ok && m == 1 {
		w.inputFormatters = append([]services.TextInputFormatter{services.FilteringTextInputFormatterSingleLine}, o.InputFormatters...)
	}
	w.showCursor = o.ShowCursor.OrElse(!o.ReadOnly)
	w.enableInteractiveSelection = o.EnableInteractiveSelection.OrElse(!o.ReadOnly || !o.ObscureText)
	w.selectAllOnFocus = o.SelectAllOnFocus.OrElse(defaultSelectAllOnFocus())
	if o.StrutStyle == nil {
		w.strutStyle = painting.NewStrutStyleFromTextStyle(style, painting.StrutStyleOpts{ForceStrutHeight: foundation.Some(true)})
	} else {
		w.strutStyle = o.StrutStyle.InheritFromTextStyle(style)
	}
	w.selectionHeightStyle = o.SelectionHeightStyle
	if w.selectionHeightStyle == ui.BoxHeightStyleUnspecified {
		w.selectionHeightStyle = EditableTextDefaultSelectionHeightStyle
	}
	w.selectionWidthStyle = o.SelectionWidthStyle
	if w.selectionWidthStyle == ui.BoxWidthStyleUnspecified {
		w.selectionWidthStyle = EditableTextDefaultSelectionWidthStyle
	}
	return w
}

// EditableTextDefaultSelectionHeightStyle is the default height of the boxes of
// the selection.
//
// Flutter: widgets/editable_text.dart EditableText.defaultSelectionHeightStyle
const EditableTextDefaultSelectionHeightStyle = ui.BoxHeightStyleIncludeLineSpacingMiddle

// EditableTextDefaultSelectionWidthStyle is the default width of the boxes of
// the selection.
//
// Flutter: widgets/editable_text.dart EditableText.defaultSelectionWidthStyle
const EditableTextDefaultSelectionWidthStyle = ui.BoxWidthStyleMax

func defaultSelectAllOnFocus() bool {
	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformLinux, foundation.TargetPlatformMacOS, foundation.TargetPlatformWindows:
		return true
	}
	return false
}

// inferKeyboardType is the keyboard type of a field without one: from the first
// autofill hint, or multiline or text.
//
// Flutter: widgets/editable_text.dart EditableText._inferKeyboardType
func inferKeyboardType(autofillHints []string, maxLines foundation.Opt[int]) services.TextInputType {
	single := maxLines.IsSome() && maxLines.Value() == 1
	if len(autofillHints) == 0 {
		if single {
			return services.TextInputTypeText
		}
		return services.TextInputTypeMultiline
	}
	if !single {
		return services.TextInputTypeMultiline
	}
	if t, ok := autofillKeyboardTypes()[autofillHints[0]]; ok {
		return t
	}
	return services.TextInputTypeText
}

// Controller is the controller of the text.
//
// Flutter: widgets/editable_text.dart EditableText.controller
func (w *EditableText) Controller() *TextEditingController { return w.controller }

// FocusNode is the focus node of the field.
//
// Flutter: widgets/editable_text.dart EditableText.focusNode
func (w *EditableText) FocusNode() *FocusNode { return w.focusNode }

// Style is the style of the text.
//
// Flutter: widgets/editable_text.dart EditableText.style
func (w *EditableText) Style() *painting.TextStyle { return w.style }

// ReadOnly is whether the text can't be edited.
//
// Flutter: widgets/editable_text.dart EditableText.readOnly
func (w *EditableText) ReadOnly() bool { return w.o.ReadOnly }

// ObscureText is whether the text is hidden.
//
// Flutter: widgets/editable_text.dart EditableText.obscureText
func (w *EditableText) ObscureText() bool { return w.o.ObscureText }

// ObscuringCharacter is the character that hides the text.
//
// Flutter: widgets/editable_text.dart EditableText.obscuringCharacter
func (w *EditableText) ObscuringCharacter() string { return w.o.ObscuringCharacter }

// MaxLines is the maximum number of lines; none for no limit.
//
// Flutter: widgets/editable_text.dart EditableText.maxLines
func (w *EditableText) MaxLines() foundation.Opt[int] { return w.o.MaxLines }

// MinLines is the minimum number of lines.
//
// Flutter: widgets/editable_text.dart EditableText.minLines
func (w *EditableText) MinLines() foundation.Opt[int] { return w.o.MinLines }

// KeyboardType is the kind of text of the field.
//
// Flutter: widgets/editable_text.dart EditableText.keyboardType
func (w *EditableText) KeyboardType() services.TextInputType { return w.keyboardType }

// ShowCursor is whether the caret is shown.
//
// Flutter: widgets/editable_text.dart EditableText.showCursor
func (w *EditableText) ShowCursor() bool { return w.showCursor }

// InputFormatters are the formatters of the text, with the single line one for a
// field with one line.
//
// Flutter: widgets/editable_text.dart EditableText.inputFormatters
func (w *EditableText) InputFormatters() []services.TextInputFormatter {
	return slices.Clone(w.inputFormatters)
}

// SelectionEnabled is whether the user can select text.
//
// Flutter: widgets/editable_text.dart EditableText.selectionEnabled
func (w *EditableText) SelectionEnabled() bool { return w.enableInteractiveSelection }

// userSelectionEnabled is whether the user can select text, which is not the
// case for a read-only obscured field.
//
// Flutter: widgets/editable_text.dart EditableText._userSelectionEnabled
func (w *EditableText) userSelectionEnabled() bool {
	return w.enableInteractiveSelection && (!w.o.ReadOnly || !w.o.ObscureText)
}

// StrutStyle is the strut of the text, inherited from the style.
//
// Flutter: widgets/editable_text.dart EditableText.strutStyle
func (w *EditableText) StrutStyle() *painting.StrutStyle { return w.strutStyle }

// CreateState creates the state.
//
// Flutter: widgets/editable_text.dart EditableText.createState
func (w *EditableText) CreateState() State { return newEditableTextState() }

// DebugFillProperties adds the properties of the field.
//
// Flutter: widgets/editable_text.dart EditableText.debugFillProperties
func (w *EditableText) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewDiagnosticsProperty("controller", w.controller))
	p.Add(foundation.NewDiagnosticsProperty("focusNode", w.focusNode))
	p.Add(foundation.NewFlagProperty("obscureText", foundation.Some(w.o.ObscureText), foundation.FlagPropertyOpts{IfTrue: "obscureText", DefaultValue: foundation.Some(false)}))
	p.Add(foundation.NewFlagProperty("readOnly", foundation.Some(w.o.ReadOnly), foundation.FlagPropertyOpts{IfTrue: "readOnly", DefaultValue: foundation.Some(false)}))
	p.Add(foundation.NewFlagProperty("expands", foundation.Some(w.o.Expands), foundation.FlagPropertyOpts{IfTrue: "expands", DefaultValue: foundation.Some(false)}))
	p.Add(foundation.NewFlagProperty("autofocus", foundation.Some(w.o.Autofocus), foundation.FlagPropertyOpts{IfTrue: "autofocus", DefaultValue: foundation.Some(false)}))
	if m, ok := w.o.MaxLines.Get(); ok {
		p.Add(foundation.NewIntProperty("maxLines", foundation.Some(m), foundation.NumberPropertyOpts{DefaultValue: foundation.Some(1)}))
	}
	if m, ok := w.o.MinLines.Get(); ok {
		p.Add(foundation.NewIntProperty("minLines", foundation.Some(m)))
	}
}
