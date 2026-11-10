package cupertino

import (
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// textFieldHorizontalCursorOffsetPixels is how far the cursor is moved to the side, in physical pixels, to look
// like the one of iOS.
//
// Flutter: cupertino/text_field.dart _iOSHorizontalCursorOffsetPixels
const textFieldHorizontalCursorOffsetPixels = -2.0

var (
	textFieldDefaultRoundedBorderColor = NewCupertinoDynamicColorWithBrightness(ui.NewColor(0x33000000), ui.NewColor(0x33FFFFFF))
	textFieldDefaultRoundedFillColor   = NewCupertinoDynamicColorWithBrightness(CupertinoColors.White, CupertinoColors.Black)
	textFieldDisabledBackground        = NewCupertinoDynamicColorWithBrightness(ui.NewColor(0xFFFAFAFA), ui.NewColor(0xFF050505))
	textFieldClearButtonColor          = NewCupertinoDynamicColorWithBrightness(ui.NewColor(0x33000000), ui.NewColor(0x33FFFFFF))
)

// OverlayVisibilityMode says when a prefix, a suffix or the clear button of a text field is shown.
//
// Flutter: cupertino/text_field.dart OverlayVisibilityMode
type OverlayVisibilityMode int

const (
	// OverlayVisibilityModeUnspecified is the zero value: the default of the option.
	OverlayVisibilityModeUnspecified OverlayVisibilityMode = iota
	// OverlayVisibilityModeNever never shows the overlay.
	//
	// Flutter: cupertino/text_field.dart OverlayVisibilityMode.never
	OverlayVisibilityModeNever
	// OverlayVisibilityModeEditing shows the overlay only while the field has text.
	//
	// Flutter: cupertino/text_field.dart OverlayVisibilityMode.editing
	OverlayVisibilityModeEditing
	// OverlayVisibilityModeNotEditing shows the overlay only while the field has no text.
	//
	// Flutter: cupertino/text_field.dart OverlayVisibilityMode.notEditing
	OverlayVisibilityModeNotEditing
	// OverlayVisibilityModeAlways always shows the overlay.
	//
	// Flutter: cupertino/text_field.dart OverlayVisibilityMode.always
	OverlayVisibilityModeAlways
)

// CupertinoTextFieldMisspelledTextStyle is the style of a misspelled word on iOS: a dotted red underline. It is
// read-only.
//
// Flutter: cupertino/text_field.dart CupertinoTextField.cupertinoMisspelledTextStyle
var CupertinoTextFieldMisspelledTextStyle = painting.NewTextStyle(painting.TextStyleOpts{
	Decoration:      foundation.Some(ui.TextDecorationUnderline),
	DecorationColor: foundation.Some(CupertinoColors.SystemRed.Color),
	DecorationStyle: ui.TextDecorationStyleDotted,
})

// CupertinoTextFieldMisspelledSelectionColor is the color of the selection of a misspelled word on iOS.
//
// Flutter: cupertino/text_field.dart CupertinoTextField.kMisspelledSelectionColor
var CupertinoTextFieldMisspelledSelectionColor = ui.NewColor(0x62ff9699)

// CupertinoTextFieldDefaultSpellCheckSuggestionsToolbarBuilder builds the spell check toolbar of iOS.
//
// Flutter: cupertino/text_field.dart CupertinoTextField.defaultSpellCheckSuggestionsToolbarBuilder
func CupertinoTextFieldDefaultSpellCheckSuggestionsToolbarBuilder(_ widgets.BuildContext, editableTextState *widgets.EditableTextState) widgets.Widget {
	return NewCupertinoSpellCheckSuggestionsToolbarEditableText(editableTextState)
}

// CupertinoTextFieldInferIOSSpellCheckConfiguration completes the configuration of the spell check with the style,
// the selection color and the toolbar of iOS.
//
// Flutter: cupertino/text_field.dart CupertinoTextField.inferIOSSpellCheckConfiguration
func CupertinoTextFieldInferIOSSpellCheckConfiguration(configuration *widgets.SpellCheckConfiguration) *widgets.SpellCheckConfiguration {
	if configuration == nil || configuration.Equal(widgets.NewSpellCheckConfigurationDisabled()) {
		return widgets.NewSpellCheckConfigurationDisabled()
	}
	copy := widgets.SpellCheckConfigurationCopy{
		MisspelledTextStyle:                 configuration.MisspelledTextStyle(),
		MisspelledSelectionColor:            configuration.MisspelledSelectionColor(),
		SpellCheckSuggestionsToolbarBuilder: configuration.SpellCheckSuggestionsToolbarBuilder(),
	}
	if copy.MisspelledTextStyle == nil {
		copy.MisspelledTextStyle = CupertinoTextFieldMisspelledTextStyle
	}
	if copy.MisspelledSelectionColor.IsNone() {
		copy.MisspelledSelectionColor = foundation.Some(CupertinoTextFieldMisspelledSelectionColor)
	}
	if copy.SpellCheckSuggestionsToolbarBuilder == nil {
		copy.SpellCheckSuggestionsToolbarBuilder = CupertinoTextFieldDefaultSpellCheckSuggestionsToolbarBuilder
	}
	return configuration.CopyWith(copy)
}

// cupertinoTextFieldSelectionBuilder is the gesture detector builder of a CupertinoTextField.
//
// Flutter: cupertino/text_field.dart _CupertinoTextFieldSelectionGestureDetectorBuilder
type cupertinoTextFieldSelectionBuilder struct {
	widgets.TextSelectionGestureDetectorBuilder
	state *cupertinoTextFieldState
}

// OnSingleTapUp ignores a tap on the clear button, and calls the field's onTap after the others.
//
// Flutter: cupertino/text_field.dart _CupertinoTextFieldSelectionGestureDetectorBuilder.onSingleTapUp
func (b *cupertinoTextFieldSelectionBuilder) OnSingleTapUp(details gestures.TapDragUpDetails) {
	// Because TextSelectionGestureDetector listens to taps that happen on children of the field, a tap on the clear
	// button must not move the cursor.
	if e := b.state.clearKey.CurrentElement(); e != nil {
		renderBox := e.(widgets.BuildContext).FindRenderObject().(rendering.Box)
		localOffset := renderBox.GlobalToLocal(details.GlobalPosition, nil)
		if renderBox.HitTest(rendering.NewBoxHitTestResult(), localOffset) {
			return
		}
	}
	b.TextSelectionGestureDetectorBuilder.OnSingleTapUp(details)
	if f := b.state.Widget().o.OnTap; f != nil {
		f()
	}
}

// OnDragSelectionEnd asks for the keyboard when a drag that selects ends.
//
// Flutter: cupertino/text_field.dart _CupertinoTextFieldSelectionGestureDetectorBuilder.onDragSelectionEnd
func (b *cupertinoTextFieldSelectionBuilder) OnDragSelectionEnd(details gestures.TapDragEndDetails) {
	b.state.requestKeyboard()
	b.TextSelectionGestureDetectorBuilder.OnDragSelectionEnd(details)
}

// CupertinoTextFieldOpts are the optional parameters of NewCupertinoTextField and
// NewCupertinoTextFieldBorderless.
//
// Flutter: cupertino/text_field.dart CupertinoTextField.new
type CupertinoTextFieldOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// GroupID is the group of the field in a TapRegion; the default is widgets.EditableTextGroupID.
	GroupID any
	// Controller is the controller of the text; the default is one of the field's own.
	Controller *widgets.TextEditingController
	// FocusNode is the focus node of the field; the default is one of the field's own.
	FocusNode *widgets.FocusNode
	// UndoController is the controller of the undo history.
	UndoController *widgets.UndoHistoryController
	// Decoration is painted around the field; nil is the rounded border of iOS for NewCupertinoTextField and no
	// decoration for NewCupertinoTextFieldBorderless.
	//
	// Deviation: the colors of a decoration are plain colors, so they do not change with the brightness (the
	// default one does); a Flutter decoration may have a CupertinoDynamicColor (SPEC §12.10).
	Decoration *painting.BoxDecoration
	// Padding is the padding around the text; the default is 7 on all sides.
	Padding painting.EdgeInsetsGeometry
	// Placeholder is the text shown while the field is empty.
	Placeholder foundation.Opt[string]
	// PlaceholderStyle is the style of the placeholder; the default is the weight 400 in the placeholder color.
	PlaceholderStyle *painting.TextStyle
	// Prefix is shown before the text.
	Prefix widgets.Widget
	// PrefixMode is when the prefix is shown; the default is always.
	PrefixMode OverlayVisibilityMode
	// Suffix is shown after the text.
	Suffix widgets.Widget
	// SuffixMode is when the suffix is shown; the default is always.
	SuffixMode OverlayVisibilityMode
	// CrossAxisAlignment is how the prefix, the text and the suffix are aligned vertically; the default is center.
	CrossAxisAlignment rendering.CrossAxisAlignment
	// ClearButtonMode is when the clear button is shown; the default is never.
	ClearButtonMode OverlayVisibilityMode
	// ClearButtonSemanticLabel is the semantics label of the clear button; the default is the localized one.
	ClearButtonSemanticLabel foundation.Opt[string]
	// KeyboardType is the type of keyboard; the default is text, or multiline for a field of several lines.
	KeyboardType foundation.Opt[services.TextInputType]
	// TextInputAction is the action of the keyboard's action button.
	TextInputAction services.TextInputAction
	// TextCapitalization is how the keyboard capitalizes.
	TextCapitalization services.TextCapitalization
	// Style is the style of the text; it is merged into the text style of the theme.
	Style *painting.TextStyle
	// StrutStyle is the strut style of the text.
	StrutStyle *painting.StrutStyle
	// TextAlign is how the text is aligned horizontally.
	TextAlign ui.TextAlign
	// TextAlignVertical is how the text is aligned vertically; the default is the center for a field that has a
	// decoration and the top for one that has none.
	TextAlignVertical foundation.Opt[painting.TextAlignVertical]
	// TextDirection is the direction of the text.
	TextDirection ui.TextDirection
	// ReadOnly says that the text cannot be edited.
	ReadOnly bool
	// ShowCursor is whether the cursor is shown; the default is when the field is focused.
	ShowCursor foundation.Opt[bool]
	// Autofocus asks for the focus when the field is built.
	Autofocus bool
	// ObscuringCharacter replaces each character when the text is obscured; the default is a bullet.
	ObscuringCharacter string
	// ObscureText hides the text.
	ObscureText bool
	// Autocorrect is whether the keyboard corrects the text; the default is true.
	Autocorrect foundation.Opt[bool]
	// SmartDashesType is how dashes are made smart; the default is enabled, and disabled for an obscured field.
	SmartDashesType services.SmartDashesType
	// SmartQuotesType is how quotes are made smart; the default is enabled, and disabled for an obscured field.
	SmartQuotesType services.SmartQuotesType
	// DisableSuggestions turns off the suggestions of the keyboard (Flutter's enableSuggestions is true by
	// default).
	DisableSuggestions bool
	// MaxLines is the maximum number of lines; the default is 1, and rendering.MaxLinesUnlimited is no limit.
	MaxLines foundation.Opt[int]
	// MinLines is the minimum number of lines.
	MinLines foundation.Opt[int]
	// Expands makes the field fill its parent.
	Expands bool
	// MaxLength is the maximum number of characters.
	MaxLength foundation.Opt[int]
	// MaxLengthEnforcement is how the maximum length is enforced; the default is the platform's.
	MaxLengthEnforcement services.MaxLengthEnforcement
	// OnChanged is called when the text changes.
	OnChanged func(text string)
	// OnEditingComplete is called when the user finishes editing.
	OnEditingComplete func()
	// OnSubmitted is called when the user submits the text.
	OnSubmitted func(text string)
	// OnTapOutside is called when a press is outside the field while it has the focus.
	OnTapOutside widgets.TapRegionCallback
	// OnTapUpOutside is called when a press that began outside the field is released.
	OnTapUpOutside widgets.TapRegionUpCallback
	// InputFormatters change the text as the user types.
	InputFormatters []services.TextInputFormatter
	// Enabled is whether the field answers input; the default is true.
	Enabled foundation.Opt[bool]
	// CursorWidth is the width of the cursor; the default is 2.
	CursorWidth foundation.Opt[float64]
	// CursorHeight is the height of the cursor; the default is the height of the font.
	CursorHeight foundation.Opt[float64]
	// CursorRadius is the radius of the corners of the cursor; the default is 2.
	CursorRadius foundation.Opt[ui.Radius]
	// CursorOpacityAnimates is whether the cursor fades in and out; the default is true.
	CursorOpacityAnimates foundation.Opt[bool]
	// CursorColor is the color of the cursor; the default is the one of the selection style, or the primary color.
	CursorColor ui.ColorValue
	// SelectionHeightStyle is the height style of the selection boxes.
	SelectionHeightStyle ui.BoxHeightStyle
	// SelectionWidthStyle is the width style of the selection boxes.
	SelectionWidthStyle ui.BoxWidthStyle
	// KeyboardAppearance is the brightness of the keyboard; the default is the one of the theme.
	KeyboardAppearance ui.Brightness
	// ScrollPadding is how far from the edges the field is scrolled when it is brought into view; the default is
	// 20.
	ScrollPadding foundation.Opt[painting.EdgeInsets]
	// DragStartBehavior is when a drag starts.
	DragStartBehavior gestures.DragStartBehavior
	// EnableInteractiveSelection is whether the user can select the text; the default is true, except for a field
	// that is obscured and read-only.
	EnableInteractiveSelection foundation.Opt[bool]
	// SelectAllOnFocus selects all the text when the field takes the focus.
	SelectAllOnFocus foundation.Opt[bool]
	// SelectionControls are the handles of the selection; the default is the ones of the platform.
	SelectionControls widgets.TextSelectionControls
	// OnTap is called when the field is tapped.
	OnTap func()
	// ScrollController is the controller of the scroll of the text.
	ScrollController widgets.ScrollController
	// ScrollPhysics is the physics of the scroll of the text.
	ScrollPhysics widgets.ScrollPhysics
	// AutofillHints are the hints for the autofill service.
	AutofillHints []string
	// ClipBehavior is how the text is clipped; the default is ClipHardEdge.
	ClipBehavior ui.Clip
	// DisableIMEPersonalizedLearning turns off the personalized learning of the keyboard (Flutter's
	// enableIMEPersonalizedLearning is true by default).
	DisableIMEPersonalizedLearning bool
	// EnableInlinePrediction is whether the keyboard predicts inline.
	EnableInlinePrediction foundation.Opt[bool]
	// ContextMenuBuilder builds the context menu; the default is the adaptive toolbar of the platform.
	ContextMenuBuilder widgets.EditableTextContextMenuBuilder
	// NoContextMenu shows no context menu (Flutter's contextMenuBuilder is null).
	NoContextMenu bool
	// SpellCheckConfiguration is the configuration of the spell check; the default is none.
	SpellCheckConfiguration *widgets.SpellCheckConfiguration
	// MagnifierConfiguration is the configuration of the magnifier; the default is the one of iOS.
	MagnifierConfiguration *widgets.TextMagnifierConfiguration
}

// CupertinoTextField is an iOS styled text field: the text, a placeholder, a prefix, a suffix, a clear button, and
// the behavior of the caret, the selection, the handles, the toolbar and the magnifier of iOS and macOS.
//
// Flutter: cupertino/text_field.dart CupertinoTextField
type CupertinoTextField struct {
	widgets.StatefulWidget
	o                 CupertinoTextFieldOpts
	maxLines          foundation.Opt[int]
	defaultDecoration bool
}

// NewCupertinoTextField creates a text field with the rounded border of iOS. In debug builds it panics with the
// tags CupertinoTextFieldObscuringCharacterInvalid, CupertinoTextFieldMaxLinesInvalid,
// CupertinoTextFieldMinLinesInvalid, CupertinoTextFieldMinLinesGreaterThanMaxLines,
// CupertinoTextFieldExpandsWithLines, CupertinoTextFieldObscuredMultiline, CupertinoTextFieldMaxLengthInvalid and
// CupertinoTextFieldNewlineActionWithTextKeyboard when the arguments are not valid.
//
// Flutter: cupertino/text_field.dart CupertinoTextField.new
func NewCupertinoTextField(opts ...CupertinoTextFieldOpts) *CupertinoTextField {
	return newCupertinoTextField(firstOf(opts), true)
}

// NewCupertinoTextFieldBorderless creates a text field with no decoration unless one is given.
//
// Flutter: cupertino/text_field.dart CupertinoTextField.borderless
func NewCupertinoTextFieldBorderless(opts ...CupertinoTextFieldOpts) *CupertinoTextField {
	return newCupertinoTextField(firstOf(opts), false)
}

func newCupertinoTextField(o CupertinoTextFieldOpts, defaultDecoration bool) *CupertinoTextField {
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
	if foundation.DebugMode {
		if foundation.GraphemeCount(o.ObscuringCharacter) != 1 {
			panic(errors.New(tagCupertinoTextFieldObscuringCharacterInvalid, "the obscuring character must be one character", "obscuringCharacter", o.ObscuringCharacter))
		}
		if m, ok := maxLines.Get(); ok && m <= 0 {
			panic(errors.New(tagCupertinoTextFieldMaxLinesInvalid, "maxLines must be positive", "maxLines", m))
		}
		if m, ok := o.MinLines.Get(); ok && m <= 0 {
			panic(errors.New(tagCupertinoTextFieldMinLinesInvalid, "minLines must be positive", "minLines", m))
		}
		if mx, ok := maxLines.Get(); ok && o.MinLines.IsSome() && mx < o.MinLines.Value() {
			panic(errors.New(tagCupertinoTextFieldMinLinesGreaterThanMaxLines, "minLines can't be greater than maxLines", "minLines", o.MinLines.Value(), "maxLines", mx))
		}
		if o.Expands && (maxLines.IsSome() || o.MinLines.IsSome()) {
			panic(errors.New(tagCupertinoTextFieldExpandsWithLines, "minLines and maxLines must be unset when expands is true"))
		}
		if o.ObscureText && (maxLines.IsNone() || maxLines.Value() != 1) {
			panic(errors.New(tagCupertinoTextFieldObscuredMultiline, "obscured fields cannot be multiline"))
		}
		if n, ok := o.MaxLength.Get(); ok && n <= 0 {
			panic(errors.New(tagCupertinoTextFieldMaxLengthInvalid, "maxLength must be positive", "maxLength", n))
		}
		single := maxLines.IsSome() && maxLines.Value() == 1
		if o.TextInputAction == services.TextInputActionNewline && !single && o.KeyboardType.IsSome() && o.KeyboardType.Value() == services.TextInputTypeText {
			panic(errors.New(tagCupertinoTextFieldNewlineActionWithTextKeyboard, "use the multiline keyboardType with the newline textInputAction on a multiline CupertinoTextField"))
		}
	}
	if o.KeyboardType.IsNone() {
		if maxLines.IsSome() && maxLines.Value() == 1 {
			o.KeyboardType = foundation.Some(services.TextInputTypeText)
		} else {
			o.KeyboardType = foundation.Some(services.TextInputTypeMultiline)
		}
	}
	if o.SmartDashesType == services.SmartDashesTypeUnspecified {
		o.SmartDashesType = services.SmartDashesTypeEnabled
		if o.ObscureText {
			o.SmartDashesType = services.SmartDashesTypeDisabled
		}
	}
	if o.SmartQuotesType == services.SmartQuotesTypeUnspecified {
		o.SmartQuotesType = services.SmartQuotesTypeEnabled
		if o.ObscureText {
			o.SmartQuotesType = services.SmartQuotesTypeDisabled
		}
	}
	if o.GroupID == nil {
		o.GroupID = widgets.EditableTextGroupID
	}
	if o.Padding == nil {
		o.Padding = painting.NewEdgeInsetsAll(7.0)
	}
	if o.PrefixMode == OverlayVisibilityModeUnspecified {
		o.PrefixMode = OverlayVisibilityModeAlways
	}
	if o.SuffixMode == OverlayVisibilityModeUnspecified {
		o.SuffixMode = OverlayVisibilityModeAlways
	}
	if o.ClearButtonMode == OverlayVisibilityModeUnspecified {
		o.ClearButtonMode = OverlayVisibilityModeNever
	}
	if o.CrossAxisAlignment == rendering.CrossAxisAlignmentUnspecified {
		o.CrossAxisAlignment = rendering.CrossAxisAlignmentCenter
	}
	o.CursorWidth = foundation.Some(o.CursorWidth.OrElse(2.0))
	o.CursorRadius = foundation.Some(o.CursorRadius.OrElse(ui.NewRadiusCircular(2.0)))
	o.CursorOpacityAnimates = foundation.Some(o.CursorOpacityAnimates.OrElse(true))
	o.ScrollPadding = foundation.Some(o.ScrollPadding.OrElse(painting.NewEdgeInsetsAll(20.0)))
	o.EnableInteractiveSelection = foundation.Some(o.EnableInteractiveSelection.OrElse(!o.ReadOnly || !o.ObscureText))
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipHardEdge
	}
	if o.DragStartBehavior == gestures.DragStartBehaviorUnspecified {
		o.DragStartBehavior = gestures.DragStartBehaviorStart
	}
	return &CupertinoTextField{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), o: o, maxLines: maxLines,
		defaultDecoration: defaultDecoration && o.Decoration == nil}
}

// Controller is the controller of the text, if one was given.
//
// Flutter: cupertino/text_field.dart CupertinoTextField.controller
func (w *CupertinoTextField) Controller() *widgets.TextEditingController { return w.o.Controller }

// FocusNode is the focus node of the field, if one was given.
//
// Flutter: cupertino/text_field.dart CupertinoTextField.focusNode
func (w *CupertinoTextField) FocusNode() *widgets.FocusNode { return w.o.FocusNode }

// MaxLines is the maximum number of lines; none is no limit.
//
// Flutter: cupertino/text_field.dart CupertinoTextField.maxLines
func (w *CupertinoTextField) MaxLines() foundation.Opt[int] { return w.maxLines }

// MinLines is the minimum number of lines.
//
// Flutter: cupertino/text_field.dart CupertinoTextField.minLines
func (w *CupertinoTextField) MinLines() foundation.Opt[int] { return w.o.MinLines }

// MaxLength is the maximum number of characters.
//
// Flutter: cupertino/text_field.dart CupertinoTextField.maxLength
func (w *CupertinoTextField) MaxLength() foundation.Opt[int] { return w.o.MaxLength }

// ObscureText is whether the text is hidden.
//
// Flutter: cupertino/text_field.dart CupertinoTextField.obscureText
func (w *CupertinoTextField) ObscureText() bool { return w.o.ObscureText }

// ReadOnly is whether the text cannot be edited.
//
// Flutter: cupertino/text_field.dart CupertinoTextField.readOnly
func (w *CupertinoTextField) ReadOnly() bool { return w.o.ReadOnly }

// Enabled is whether the field answers input.
//
// Flutter: cupertino/text_field.dart CupertinoTextField.enabled
func (w *CupertinoTextField) Enabled() bool { return w.o.Enabled.OrElse(true) }

// SelectionEnabled is whether the user can select the text.
//
// Flutter: cupertino/text_field.dart CupertinoTextField.selectionEnabled
func (w *CupertinoTextField) SelectionEnabled() bool { return w.o.EnableInteractiveSelection.Value() }

// Placeholder is the text shown while the field is empty.
//
// Flutter: cupertino/text_field.dart CupertinoTextField.placeholder
func (w *CupertinoTextField) Placeholder() foundation.Opt[string] { return w.o.Placeholder }

// CreateState creates the state.
//
// Flutter: cupertino/text_field.dart CupertinoTextField.createState
func (w *CupertinoTextField) CreateState() widgets.State { return &cupertinoTextFieldState{} }

type cupertinoTextFieldState struct {
	widgets.StateOf[*CupertinoTextField]
	clearKey             *widgets.GlobalKey[widgets.State]
	controller           *widgets.TextEditingController
	focusNode            *widgets.FocusNode
	focusListener        foundation.ListenerHandle
	focusListenerOn      *widgets.FocusNode
	showSelectionHandles bool
	selectionBuilder     *cupertinoTextFieldSelectionBuilder
	editableTextKey      *widgets.GlobalKey[*widgets.EditableTextState]
}

// EditableTextKey is the key of the EditableText.
func (s *cupertinoTextFieldState) EditableTextKey() *widgets.GlobalKey[*widgets.EditableTextState] {
	return s.editableTextKey
}

// ForcePressEnabled is true: Flutter has the force press for iOS (it is not ported).
func (s *cupertinoTextFieldState) ForcePressEnabled() bool { return true }

// SelectionEnabled is whether the user can select the text.
func (s *cupertinoTextFieldState) SelectionEnabled() bool { return s.Widget().SelectionEnabled() }

func (s *cupertinoTextFieldState) effectiveController() *widgets.TextEditingController {
	if c := s.Widget().o.Controller; c != nil {
		return c
	}
	return s.controller
}

func (s *cupertinoTextFieldState) effectiveFocusNode() *widgets.FocusNode {
	if f := s.Widget().o.FocusNode; f != nil {
		return f
	}
	if s.focusNode == nil {
		s.focusNode = widgets.NewFocusNode(widgets.FocusNodeOpts{})
	}
	return s.focusNode
}

func (s *cupertinoTextFieldState) effectiveMaxLengthEnforcement() services.MaxLengthEnforcement {
	if e := s.Widget().o.MaxLengthEnforcement; e != services.MaxLengthEnforcementUnspecified {
		return e
	}
	return services.LengthLimitingTextInputFormatterGetDefaultMaxLengthEnforcement(foundation.DefaultTargetPlatform())
}

func (s *cupertinoTextFieldState) InitState() {
	s.clearKey = widgets.NewGlobalKey[widgets.State]()
	s.editableTextKey = widgets.NewGlobalKey[*widgets.EditableTextState]()
	s.selectionBuilder = &cupertinoTextFieldSelectionBuilder{state: s}
	widgets.InitTextSelectionGestureDetectorBuilder(&s.selectionBuilder.TextSelectionGestureDetectorBuilder, s.selectionBuilder, s)
	if s.Widget().o.Controller == nil {
		s.controller = widgets.NewTextEditingController()
	}
	s.effectiveFocusNode().SetCanRequestFocus(s.Widget().Enabled())
	s.listenToFocus()
}

func (s *cupertinoTextFieldState) listenToFocus() {
	s.focusListenerOn = s.effectiveFocusNode()
	s.focusListener = s.focusListenerOn.AddListener(s.handleFocusChanged)
}

func (s *cupertinoTextFieldState) DidUpdateWidget(old *CupertinoTextField) {
	w := s.Widget()
	if w.o.Controller == nil && old.o.Controller != nil {
		s.controller = widgets.NewTextEditingControllerFromValue(old.o.Controller.Value())
	} else if w.o.Controller != nil && old.o.Controller == nil {
		s.controller.Dispose()
		s.controller = nil
	}
	if s.focusListenerOn != s.effectiveFocusNode() {
		s.focusListenerOn.RemoveListener(s.focusListener)
		s.listenToFocus()
	}
	s.effectiveFocusNode().SetCanRequestFocus(w.Enabled())
}

func (s *cupertinoTextFieldState) Dispose() {
	s.focusListenerOn.RemoveListener(s.focusListener)
	if s.focusNode != nil {
		s.focusNode.Dispose()
	}
	if s.controller != nil {
		s.controller.Dispose()
	}
}

func (s *cupertinoTextFieldState) editableText() *widgets.EditableTextState {
	e, _ := s.editableTextKey.CurrentState()
	return e
}

func (s *cupertinoTextFieldState) requestKeyboard() {
	if e := s.editableText(); e != nil {
		e.RequestKeyboard()
	}
}

func (s *cupertinoTextFieldState) handleFocusChanged() { s.SetState(func() {}) }

func (s *cupertinoTextFieldState) shouldShowSelectionHandles(cause services.SelectionChangedCause) bool {
	b := &s.selectionBuilder.TextSelectionGestureDetectorBuilder
	// When the text field is activated by something that doesn't trigger the selection toolbar, we shouldn't show
	// the handles either.
	if !b.ShouldShowSelectionToolbar() || !b.ShouldShowSelectionHandles() {
		return false
	}
	if s.effectiveController().Selection().IsCollapsed() {
		return false
	}
	if cause == services.SelectionChangedCauseKeyboard {
		return false
	}
	if cause == services.SelectionChangedCauseStylusHandwriting {
		return true
	}
	return s.effectiveController().Text() != ""
}

func (s *cupertinoTextFieldState) handleSelectionChanged(selection services.TextSelection, cause services.SelectionChangedCause) {
	willShowSelectionHandles := s.shouldShowSelectionHandles(cause)
	if willShowSelectionHandles != s.showSelectionHandles {
		s.SetState(func() { s.showSelectionHandles = willShowSelectionHandles })
	}
	if cause == services.SelectionChangedCauseLongPress {
		if e := s.editableText(); e != nil {
			e.BringIntoView(ui.NewTextPosition(selection.ExtentOffset))
		}
	}
	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformMacOS, foundation.TargetPlatformLinux, foundation.TargetPlatformWindows:
		if cause == services.SelectionChangedCauseDrag {
			if e := s.editableText(); e != nil {
				e.HideToolbar(true)
			}
		}
	}
}

func shouldShowOverlayAttachment(attachment OverlayVisibilityMode, hasText bool) bool {
	switch attachment {
	case OverlayVisibilityModeNever:
		return false
	case OverlayVisibilityModeAlways:
		return true
	case OverlayVisibilityModeEditing:
		return hasText
	case OverlayVisibilityModeNotEditing:
		return !hasText
	}
	return false
}

func (s *cupertinoTextFieldState) hasDecoration() bool {
	o := s.Widget().o
	return o.Placeholder.IsSome() || o.ClearButtonMode != OverlayVisibilityModeNever || o.Prefix != nil || o.Suffix != nil
}

func (s *cupertinoTextFieldState) textAlignVertical() painting.TextAlignVertical {
	if v, ok := s.Widget().o.TextAlignVertical.Get(); ok {
		return v
	}
	if s.hasDecoration() {
		return painting.TextAlignVerticalCenter
	}
	return painting.TextAlignVerticalTop
}

func (s *cupertinoTextFieldState) onClearButtonTapped() {
	hadText := s.effectiveController().Text() != ""
	s.effectiveController().Clear()
	if hadText {
		if f := s.Widget().o.OnChanged; f != nil {
			f(s.effectiveController().Text())
		}
	}
}

func (s *cupertinoTextFieldState) buildClearButton(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	clearLabel, ok := w.o.ClearButtonSemanticLabel.Get()
	if !ok {
		clearLabel = CupertinoLocalizationsOf(bc).ClearButtonLabel()
	}
	var onTap func()
	if w.Enabled() {
		onTap = s.onClearButtonTapped
	}
	return widgets.NewSemantics(
		widgets.NewGestureDetector(
			widgets.NewPadding(painting.NewEdgeInsetsSymmetric(0, 6.0),
				widgets.NewIcon(CupertinoIcons.ClearThickCircled(), widgets.IconOpts{Size: foundation.Some(18.0),
					Color: foundation.Some(CupertinoDynamicColorResolve(textFieldClearButtonColor, bc))})),
			widgets.GestureDetectorOpts{Key: s.clearKey, OnTap: onTap}),
		widgets.SemanticsOpts{Button: foundation.Some(true), Label: foundation.Some(clearLabel)})
}

func (s *cupertinoTextFieldState) addTextDependentAttachments(bc widgets.BuildContext, editableText widgets.Widget, textStyle, placeholderStyle *painting.TextStyle) widgets.Widget {
	if !s.hasDecoration() {
		return editableText
	}
	w := s.Widget()
	controller := s.effectiveController()
	return widgets.NewAnimatedBuilder(controller, func(bc widgets.BuildContext, child widgets.Widget) widgets.Widget {
		hasText := controller.Text() != ""
		var placeholder widgets.Widget
		if text, ok := w.o.Placeholder.Get(); ok {
			maxLines := w.maxLines
			if hasText {
				maxLines = foundation.Some(1)
			}
			placeholder = widgets.NewVisibility(widgets.NewSizedBox(
				widgets.NewPadding(w.o.Padding, widgets.NewText(text, widgets.TextOpts{
					MaxLines: maxLines, Overflow: placeholderStyle.Overflow, Style: placeholderStyle, TextAlign: w.o.TextAlign})),
				widgets.SizedBoxOpts{Width: foundation.Some(math.Inf(1))}),
				widgets.VisibilityOpts{MaintainAnimation: true, MaintainSize: true, MaintainState: true, Visible: foundation.Some(!hasText)})
		}
		var prefixWidget widgets.Widget
		if shouldShowOverlayAttachment(w.o.PrefixMode, hasText) {
			prefixWidget = w.o.Prefix
		}
		showUserSuffix := shouldShowOverlayAttachment(w.o.SuffixMode, hasText)
		showClearButton := shouldShowOverlayAttachment(w.o.ClearButtonMode, hasText)
		var suffixWidget widgets.Widget
		switch {
		case showUserSuffix && showClearButton:
			suffixWidget = w.o.Suffix
			if suffixWidget == nil {
				suffixWidget = s.buildClearButton(bc)
			}
		case showUserSuffix:
			suffixWidget = w.o.Suffix
		case showClearButton:
			suffixWidget = s.buildClearButton(bc)
		}
		textDirection := w.o.TextDirection
		if textDirection == ui.TextDirectionUnspecified {
			textDirection = widgets.DirectionalityOf(bc)
		}
		baseline := func(b ui.TextBaseline) ui.TextBaseline {
			if b == ui.TextBaselineUnspecified {
				return ui.TextBaselineAlphabetic
			}
			return b
		}
		var row []widgets.Widget
		if prefixWidget != nil {
			row = append(row, prefixWidget)
		}
		row = append(row, widgets.NewExpanded(widgets.NewDirectionality(textDirection, &baselineAlignedStack{
			SlottedMultiChildRenderObjectWidget: widgets.NewSlottedMultiChildRenderObjectWidgetBase(),
			placeholder:                         placeholder, editableText: child, textAlignVertical: s.textAlignVertical(),
			editableTextBaseline: baseline(textStyle.TextBaseline), placeholderBaseline: baseline(placeholderStyle.TextBaseline)})))
		if suffixWidget != nil {
			row = append(row, suffixWidget)
		}
		return widgets.NewRow(row, widgets.RowOpts{CrossAxisAlignment: w.o.CrossAxisAlignment})
	}, editableText)
}

func (s *cupertinoTextFieldState) Build(bc widgets.BuildContext) widgets.Widget {
	widgets.DebugCheckHasDirectionality(bc)
	w := s.Widget()
	controller := s.effectiveController()
	focusNode := s.effectiveFocusNode()
	textSelectionControls := w.o.SelectionControls
	var handleDidGainAccessibilityFocus, handleDidLoseAccessibilityFocus func()
	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformIOS, foundation.TargetPlatformAndroid, foundation.TargetPlatformFuchsia:
		if textSelectionControls == nil {
			textSelectionControls = CupertinoTextSelectionHandleControls
		}
	default: // linux, macOS, windows
		if textSelectionControls == nil {
			textSelectionControls = CupertinoDesktopTextSelectionHandleControls
		}
		handleDidGainAccessibilityFocus = func() {
			if !focusNode.HasFocus() && focusNode.CanRequestFocus() {
				focusNode.RequestFocus()
			}
		}
		handleDidLoseAccessibilityFocus = func() { focusNode.Unfocus() }
	}
	enabled := w.Enabled()
	cursorOffset := foundation.Some(ui.Offset{Dx: textFieldHorizontalCursorOffsetPixels / widgets.MediaQueryDevicePixelRatioOf(bc)})
	var formatters []services.TextInputFormatter
	formatters = append(formatters, w.o.InputFormatters...)
	if w.o.MaxLength.IsSome() {
		formatters = append(formatters, services.NewLengthLimitingTextInputFormatter(w.o.MaxLength,
			services.LengthLimitingTextInputFormatterOpts{MaxLengthEnforcement: s.effectiveMaxLengthEnforcement()}))
	}
	themeData := CupertinoThemeOf(bc)
	textStyle := themeData.TextTheme().TextStyle().Merge(w.o.Style)
	placeholderStyle := w.o.PlaceholderStyle
	if placeholderStyle == nil {
		placeholderStyle = painting.NewTextStyle(painting.TextStyleOpts{FontWeight: ui.FontWeightW400,
			Color: foundation.Some(CupertinoDynamicColorResolve(CupertinoColors.PlaceholderText, bc))})
	}
	placeholderStyle = textStyle.Merge(placeholderStyle)
	keyboardAppearance := w.o.KeyboardAppearance
	if keyboardAppearance == ui.BrightnessUnspecified {
		keyboardAppearance = CupertinoThemeBrightnessOf(bc)
	}
	selectionStyle := widgets.DefaultSelectionStyleOf(bc)
	var cursorColor ui.Color
	switch {
	case w.o.CursorColor != nil:
		cursorColor = CupertinoDynamicColorResolve(w.o.CursorColor, bc)
	case selectionStyle.CursorColor().IsSome():
		cursorColor = selectionStyle.CursorColor().Value()
	default:
		cursorColor = CupertinoDynamicColorResolve(themeData.PrimaryColor(), bc)
	}
	disabledColor := CupertinoDynamicColorResolve(textFieldDisabledBackground, bc)

	effectiveDecoration := w.o.Decoration
	if w.defaultDecoration {
		side := painting.NewBorderSide(painting.BorderSideOpts{Color: foundation.Some(CupertinoDynamicColorResolve(textFieldDefaultRoundedBorderColor, bc)), Width: foundation.Some(0.0)})
		effectiveDecoration = painting.NewBoxDecoration(painting.BoxDecorationOpts{
			Color:        foundation.Some(CupertinoDynamicColorResolve(textFieldDefaultRoundedFillColor, bc)),
			Border:       painting.Border{Top: side, Right: side, Bottom: side, Left: side},
			BorderRadius: painting.NewBorderRadiusAll(ui.NewRadiusCircular(5.0)),
		})
		if !enabled {
			effectiveDecoration = effectiveDecoration.CopyWith(painting.BoxDecorationCopy{Color: foundation.Some(disabledColor)})
		}
	}
	var selectionColor ui.Color
	if c, ok := selectionStyle.SelectionColor().Get(); ok {
		selectionColor = c
	} else {
		selectionColor = CupertinoDynamicColorResolve(themeData.PrimaryColor(), bc).WithOpacity(0.2)
	}
	spellCheckConfiguration := CupertinoTextFieldInferIOSSpellCheckConfiguration(w.o.SpellCheckConfiguration)
	magnifierConfiguration := w.o.MagnifierConfiguration
	if magnifierConfiguration == nil {
		magnifierConfiguration = iosMagnifierConfiguration
	}
	contextMenuBuilder := w.o.ContextMenuBuilder
	switch {
	case w.o.NoContextMenu:
		contextMenuBuilder = nil
	case contextMenuBuilder == nil:
		contextMenuBuilder = defaultCupertinoTextFieldContextMenuBuilder
	}
	var controls widgets.TextSelectionControls
	if w.SelectionEnabled() {
		controls = textSelectionControls
	}
	var selectionColorOpt foundation.Opt[ui.Color]
	if focusNode.HasFocus() {
		selectionColorOpt = foundation.Some(selectionColor)
	}
	maxLines := w.maxLines
	if maxLines.IsNone() {
		maxLines = foundation.Some(rendering.MaxLinesUnlimited)
	}
	paddedEditable := widgets.NewPadding(w.o.Padding, widgets.NewRepaintBoundary(widgets.NewEditableText(controller, focusNode, textStyle, cursorColor,
		CupertinoDynamicColorResolve(CupertinoColors.InactiveGray, bc),
		widgets.EditableTextOpts{
			Key:                            s.editableTextKey,
			UndoController:                 w.o.UndoController,
			ReadOnly:                       w.o.ReadOnly || !enabled,
			ShowCursor:                     w.o.ShowCursor,
			ShowSelectionHandles:           s.showSelectionHandles,
			KeyboardType:                   w.o.KeyboardType,
			TextInputAction:                w.o.TextInputAction,
			TextCapitalization:             w.o.TextCapitalization,
			StrutStyle:                     w.o.StrutStyle,
			TextAlign:                      w.o.TextAlign,
			TextDirection:                  w.o.TextDirection,
			Autofocus:                      w.o.Autofocus,
			ObscuringCharacter:             w.o.ObscuringCharacter,
			ObscureText:                    w.o.ObscureText,
			Autocorrect:                    w.o.Autocorrect,
			SmartDashesType:                w.o.SmartDashesType,
			SmartQuotesType:                w.o.SmartQuotesType,
			DisableSuggestions:             w.o.DisableSuggestions,
			MaxLines:                       maxLines,
			MinLines:                       w.o.MinLines,
			Expands:                        w.o.Expands,
			MagnifierConfiguration:         magnifierConfiguration,
			SelectionColor:                 selectionColorOpt,
			SelectionControls:              controls,
			GroupID:                        w.o.GroupID,
			OnChanged:                      w.o.OnChanged,
			OnSelectionChanged:             s.handleSelectionChanged,
			OnEditingComplete:              w.o.OnEditingComplete,
			OnSubmitted:                    w.o.OnSubmitted,
			OnTapOutside:                   w.o.OnTapOutside,
			OnTapUpOutside:                 w.o.OnTapUpOutside,
			InputFormatters:                formatters,
			RendererIgnoresPointer:         true,
			CursorWidth:                    w.o.CursorWidth,
			CursorHeight:                   w.o.CursorHeight,
			CursorRadius:                   w.o.CursorRadius,
			CursorOpacityAnimates:          w.o.CursorOpacityAnimates.Value(),
			CursorOffset:                   cursorOffset,
			PaintCursorAboveText:           true,
			AutocorrectionTextRectColor:    foundation.Some(selectionColor),
			SelectionHeightStyle:           w.o.SelectionHeightStyle,
			SelectionWidthStyle:            w.o.SelectionWidthStyle,
			ScrollPadding:                  w.o.ScrollPadding,
			KeyboardAppearance:             keyboardAppearance,
			DragStartBehavior:              w.o.DragStartBehavior,
			ScrollController:               w.o.ScrollController,
			ScrollPhysics:                  w.o.ScrollPhysics,
			EnableInteractiveSelection:     w.o.EnableInteractiveSelection,
			SelectAllOnFocus:               w.o.SelectAllOnFocus,
			AutofillHints:                  w.o.AutofillHints,
			ClipBehavior:                   w.o.ClipBehavior,
			DisableIMEPersonalizedLearning: w.o.DisableIMEPersonalizedLearning,
			EnableInlinePrediction:         w.o.EnableInlinePrediction,
			ContextMenuBuilder:             contextMenuBuilder,
			SpellCheckConfiguration:        spellCheckConfiguration,
		})))

	var onTap func()
	if enabled && !w.o.ReadOnly {
		onTap = func() {
			if !controller.Selection().IsValid() {
				controller.SetSelection(services.NewTextSelectionCollapsed(len(controller.Text())))
			}
			s.requestKeyboard()
		}
	}
	var onFocus func()
	if enabled {
		onFocus = func() {
			if focusNode.CanRequestFocus() && !focusNode.HasFocus() {
				focusNode.RequestFocus()
			} else if !w.o.ReadOnly {
				s.requestKeyboard()
			}
		}
	}
	containerOpts := widgets.ContainerOpts{}
	if effectiveDecoration != nil {
		containerOpts.Decoration = effectiveDecoration
	} else if !enabled {
		containerOpts.Color = foundation.Some(disabledColor)
	}
	return widgets.NewSemantics(widgets.NewTextFieldTapRegion(widgets.NewIgnorePointer(
		widgets.NewContainer(s.selectionBuilder.BuildGestureDetector(
			widgets.NewAlign(s.addTextDependentAttachments(bc, paddedEditable, textStyle, placeholderStyle),
				widgets.AlignOpts{Alignment: painting.Alignment{X: -1.0, Y: s.textAlignVertical().Y}, WidthFactor: foundation.Some(1.0), HeightFactor: foundation.Some(1.0)}),
			widgets.TextSelectionGestureDetectorBuildOpts{Behavior: rendering.HitTestBehaviorTranslucent}), containerOpts),
		widgets.IgnorePointerOpts{Ignoring: foundation.Some(!enabled)}), widgets.TextFieldTapRegionOpts{}),
		widgets.SemanticsOpts{Enabled: foundation.Some(enabled), OnTap: onTap,
			OnDidGainAccessibilityFocus: handleDidGainAccessibilityFocus, OnDidLoseAccessibilityFocus: handleDidLoseAccessibilityFocus, OnFocus: onFocus})
}

// iosMagnifierConfiguration is the configuration of the magnifier of iOS: the Cupertino magnifier on iOS and Android,
// and none elsewhere.
//
// Flutter: cupertino/text_field.dart CupertinoTextField._iosMagnifierConfiguration
var iosMagnifierConfiguration = widgets.NewTextMagnifierConfiguration(widgets.TextMagnifierConfigurationOpts{
	MagnifierBuilder: func(_ widgets.BuildContext, controller *widgets.MagnifierController, magnifierInfo *foundation.ValueNotifier[widgets.MagnifierInfo]) widgets.Widget {
		switch foundation.DefaultTargetPlatform() {
		case foundation.TargetPlatformAndroid, foundation.TargetPlatformIOS:
			return NewCupertinoTextMagnifier(controller, magnifierInfo)
		}
		return nil
	},
})

func defaultCupertinoTextFieldContextMenuBuilder(_ widgets.BuildContext, state *widgets.EditableTextState) widgets.Widget {
	return NewCupertinoAdaptiveTextSelectionToolbarEditableText(state)
}

// baselineAlignedStackSlot is a slot of the baseline aligned stack.
//
// Flutter: cupertino/text_field.dart _BaselineAlignedStackSlot
type baselineAlignedStackSlot int

const (
	baselineAlignedStackSlotPlaceholder baselineAlignedStackSlot = iota
	baselineAlignedStackSlotEditableText
)

// baselineAlignedStack stacks the placeholder and the editable text with their baselines aligned.
//
// Flutter: cupertino/text_field.dart _BaselineAlignedStack
type baselineAlignedStack struct {
	widgets.SlottedMultiChildRenderObjectWidget
	editableTextBaseline, placeholderBaseline ui.TextBaseline
	textAlignVertical                         painting.TextAlignVertical
	editableText, placeholder                 widgets.Widget
}

func (w *baselineAlignedStack) Slots() []any {
	return []any{baselineAlignedStackSlotPlaceholder, baselineAlignedStackSlotEditableText}
}

func (w *baselineAlignedStack) ChildForSlot(slot any) widgets.Widget {
	if slot.(baselineAlignedStackSlot) == baselineAlignedStackSlotPlaceholder {
		return w.placeholder
	}
	return w.editableText
}

func (w *baselineAlignedStack) CreateRenderObject(widgets.BuildContext) rendering.RenderObject {
	r := &renderBaselineAlignedStack{textAlignVertical: w.textAlignVertical, editableTextBaseline: w.editableTextBaseline, placeholderBaseline: w.placeholderBaseline}
	rendering.InitRenderBox(&r.RenderBox, r)
	rendering.InitSlottedContainer(&r.SlottedContainer, r)
	return r
}

func (w *baselineAlignedStack) UpdateRenderObject(_ widgets.BuildContext, ro rendering.RenderObject) {
	r := ro.(*renderBaselineAlignedStack)
	if r.textAlignVertical != w.textAlignVertical {
		r.textAlignVertical = w.textAlignVertical
		r.MarkNeedsLayout()
	}
	if r.editableTextBaseline != w.editableTextBaseline {
		r.editableTextBaseline = w.editableTextBaseline
		r.MarkNeedsLayout()
	}
	if r.placeholderBaseline != w.placeholderBaseline {
		r.placeholderBaseline = w.placeholderBaseline
		r.MarkNeedsLayout()
	}
}

// renderBaselineAlignedStack lays the placeholder and the editable text out so that their baselines align.
//
// Flutter: cupertino/text_field.dart _RenderBaselineAlignedStack
type renderBaselineAlignedStack struct {
	rendering.RenderBox
	rendering.SlottedContainer
	textAlignVertical                         painting.TextAlignVertical
	editableTextBaseline, placeholderBaseline ui.TextBaseline
}

func (r *renderBaselineAlignedStack) SetupParentData(child rendering.RenderObject) {
	if _, ok := child.ParentData().(*rendering.BoxParentData); !ok {
		child.AsRenderObject().SetParentData(&rendering.BoxParentData{})
	}
}

func (r *renderBaselineAlignedStack) placeholderChild() rendering.Box {
	c := r.SlotChild(baselineAlignedStackSlotPlaceholder)
	if c == nil {
		return nil
	}
	return c.(rendering.Box)
}

func (r *renderBaselineAlignedStack) editableTextChild() rendering.Box {
	return r.SlotChild(baselineAlignedStackSlotEditableText).(rendering.Box)
}

func (r *renderBaselineAlignedStack) ComputeMinIntrinsicHeight(width float64) float64 {
	h := 0.0
	if p := r.placeholderChild(); p != nil {
		h = p.GetMinIntrinsicHeight(width)
	}
	return math.Max(h, r.editableTextChild().GetMinIntrinsicHeight(width))
}

func (r *renderBaselineAlignedStack) ComputeMaxIntrinsicHeight(width float64) float64 {
	h := 0.0
	if p := r.placeholderChild(); p != nil {
		h = p.GetMaxIntrinsicHeight(width)
	}
	return math.Max(h, r.editableTextChild().GetMaxIntrinsicHeight(width))
}

func (r *renderBaselineAlignedStack) ComputeMinIntrinsicWidth(height float64) float64 {
	w := 0.0
	if p := r.placeholderChild(); p != nil {
		w = p.GetMinIntrinsicWidth(height)
	}
	return math.Max(w, r.editableTextChild().GetMinIntrinsicWidth(height))
}

func (r *renderBaselineAlignedStack) ComputeMaxIntrinsicWidth(height float64) float64 {
	w := 0.0
	if p := r.placeholderChild(); p != nil {
		w = p.GetMaxIntrinsicWidth(height)
	}
	return math.Max(w, r.editableTextChild().GetMaxIntrinsicWidth(height))
}

func (r *renderBaselineAlignedStack) PerformLayout() {
	constraints := r.BoxConstraints()
	placeholder := r.placeholderChild()
	editableText := r.editableTextChild()
	editableData := editableText.ParentData().(*rendering.BoxParentData)
	r.SetSize(r.computeSize(constraints, rendering.ChildLayoutHelper.LayoutChild, rendering.ChildLayoutHelper.GetBaseline))
	editableTextBaselineValue := editableText.GetDistanceToBaseline(r.editableTextBaseline).Value()
	placeholderBaselineValue := foundation.None[float64]()
	if placeholder != nil {
		placeholderBaselineValue = placeholder.GetDistanceToBaseline(r.placeholderBaseline)
	}
	baselineDiff := ui.Offset{}
	if v, ok := placeholderBaselineValue.Get(); ok {
		baselineDiff = ui.Offset{Dy: editableTextBaselineValue - v}
	}
	verticalAlignment := painting.Alignment{Y: r.textAlignVertical.Y}
	size := r.Size()
	editableData.Offset = verticalAlignment.AlongOffset(ui.Offset{Dx: size.Width - editableText.Size().Width, Dy: size.Height - editableText.Size().Height})
	if placeholder != nil {
		placeholder.ParentData().(*rendering.BoxParentData).Offset = editableData.Offset.Add(baselineDiff)
	}
}

func (r *renderBaselineAlignedStack) Paint(context *rendering.PaintingContext, offset ui.Offset) {
	if p := r.placeholderChild(); p != nil {
		context.PaintChild(p, offset.Add(p.ParentData().(*rendering.BoxParentData).Offset))
	}
	e := r.editableTextChild()
	context.PaintChild(e, offset.Add(e.ParentData().(*rendering.BoxParentData).Offset))
}

func (r *renderBaselineAlignedStack) ComputeDryLayout(constraints rendering.BoxConstraints) ui.Size {
	return r.computeSize(constraints, rendering.ChildLayoutHelper.DryLayoutChild, rendering.ChildLayoutHelper.GetDryBaseline)
}

func (r *renderBaselineAlignedStack) computeSize(constraints rendering.BoxConstraints, layoutChild rendering.ChildLayouter, getBaseline rendering.ChildBaselineGetter) ui.Size {
	width, height := constraints.MinWidth, constraints.MinHeight
	editableText := r.editableTextChild()
	editableTextSize := layoutChild(editableText, constraints)
	editableTextBaselineValue := getBaseline(editableText, constraints, r.editableTextBaseline).Value()
	editableTextDescent := editableTextSize.Height - editableTextBaselineValue
	if placeholder := r.placeholderChild(); placeholder != nil {
		placeholderSize := layoutChild(placeholder, constraints)
		width = math.Max(width, placeholderSize.Width)
		placeholderBaselineValue := getBaseline(placeholder, constraints, r.placeholderBaseline).Value()
		placeholderDescent := placeholderSize.Height - placeholderBaselineValue
		maxExtentBaseline := math.Max(editableTextBaselineValue, placeholderBaselineValue) + math.Max(editableTextDescent, placeholderDescent)
		height = math.Max(height, maxExtentBaseline)
	}
	height = math.Max(height, editableTextSize.Height)
	width = math.Max(width, editableTextSize.Width)
	return constraints.Constrain(ui.Size{Width: width, Height: height})
}

func (r *renderBaselineAlignedStack) HitTestChildren(result *rendering.BoxHitTestResult, position ui.Offset) bool {
	editableText := r.editableTextChild()
	offset := editableText.ParentData().(*rendering.BoxParentData).Offset
	return result.AddWithPaintOffset(&offset, position, func(result *rendering.BoxHitTestResult, transformed ui.Offset) bool {
		return editableText.HitTest(result, transformed)
	})
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagCupertinoTextFieldObscuringCharacterInvalid     = "CupertinoTextFieldObscuringCharacterInvalid"
	tagCupertinoTextFieldMaxLinesInvalid               = "CupertinoTextFieldMaxLinesInvalid"
	tagCupertinoTextFieldMinLinesInvalid               = "CupertinoTextFieldMinLinesInvalid"
	tagCupertinoTextFieldMinLinesGreaterThanMaxLines   = "CupertinoTextFieldMinLinesGreaterThanMaxLines"
	tagCupertinoTextFieldExpandsWithLines              = "CupertinoTextFieldExpandsWithLines"
	tagCupertinoTextFieldObscuredMultiline             = "CupertinoTextFieldObscuredMultiline"
	tagCupertinoTextFieldMaxLengthInvalid              = "CupertinoTextFieldMaxLengthInvalid"
	tagCupertinoTextFieldNewlineActionWithTextKeyboard = "CupertinoTextFieldNewlineActionWithTextKeyboard"
)
