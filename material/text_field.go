// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"strconv"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// TextFieldNoMaxLength is the value of MaxLength for a field that counts the
// characters but does not limit them.
//
// Flutter: material/text_field.dart TextField.noMaxLength
const TextFieldNoMaxLength = -1

// iOSHorizontalOffset is the horizontal offset of the caret on the Apple
// platforms, in logical pixels per device pixel.
//
// Flutter: material/selectable_text.dart iOSHorizontalOffset
const iOSHorizontalOffset = -2.0

// InputCounterWidgetBuilder is the type of the function that builds the counter
// of a TextField. It returns nil for no counter.
//
// Flutter: material/text_field.dart InputCounterWidgetBuilder
type InputCounterWidgetBuilder func(bc widgets.BuildContext, currentLength int, maxLength foundation.Opt[int], isFocused bool) widgets.Widget

// textFieldSelectionBuilder is the gesture detector builder of a TextField: it
// tells the field about the taps.
//
// Flutter: material/text_field.dart _TextFieldSelectionGestureDetectorBuilder
type textFieldSelectionBuilder struct {
	widgets.TextSelectionGestureDetectorBuilder
	state *textFieldState
}

// OnUserTapAlwaysCalled is whether the field's onTap is called for every tap.
func (b *textFieldSelectionBuilder) OnUserTapAlwaysCalled() bool {
	return b.state.Widget().o.OnTapAlwaysCalled
}

// OnUserTap calls the field's onTap.
func (b *textFieldSelectionBuilder) OnUserTap() {
	if f := b.state.Widget().o.OnTap; f != nil {
		f()
	}
}

// TextFieldOpts are the optional parameters of NewTextField.
//
// Flutter: material/text_field.dart TextField.new
type TextFieldOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// GroupID is the group of the tap region of the field; the default is
	// widgets.EditableTextGroupID.
	GroupID any
	// Controller controls the text; the field makes its own when it is nil.
	Controller *widgets.TextEditingController
	// FocusNode controls the focus; the field makes its own when it is nil.
	FocusNode *widgets.FocusNode
	// UndoController controls the undo history from outside.
	UndoController *widgets.UndoHistoryController
	// Decoration is the decoration around the field; the default is an empty
	// InputDecoration (which still gets the defaults of the theme).
	Decoration *InputDecoration
	// NoDecoration shows the editable text alone, with no decoration (Flutter's
	// decoration: null).
	NoDecoration bool
	// KeyboardType is the kind of text the input method offers; the default is
	// text for a field of one line and multiline for the others.
	KeyboardType foundation.Opt[services.TextInputType]
	// TextInputAction is the action of the Enter key.
	TextInputAction services.TextInputAction
	// TextCapitalization says how the input method capitalizes.
	TextCapitalization services.TextCapitalization
	// Style is the style of the text; the default is the body large style of the
	// theme (the title medium one with Material 2).
	Style *painting.TextStyle
	// StateStyle is the style of the text when it depends on the state of the
	// field; it is used when Style is nil.
	StateStyle widgets.WidgetStateProperty[*painting.TextStyle]
	// StrutStyle is the strut of the text.
	StrutStyle *painting.StrutStyle
	// TextAlign is how the text is aligned horizontally; the default is start.
	TextAlign ui.TextAlign
	// TextAlignVertical is how the text is aligned vertically in the decoration.
	TextAlignVertical foundation.Opt[painting.TextAlignVertical]
	// TextDirection is the direction of the text; the default is the one of the
	// Directionality.
	TextDirection ui.TextDirection
	// ReadOnly makes the text not editable.
	ReadOnly bool
	// ShowCursor says whether the caret is shown; the default is !ReadOnly.
	ShowCursor foundation.Opt[bool]
	// Autofocus asks for the focus when the field is built.
	Autofocus bool
	// StatesController controls the states of the field (hovered, focused,
	// disabled, error); the field makes its own when it is nil.
	StatesController *widgets.WidgetStatesController
	// ObscuringCharacter replaces the text when it is obscured; it must be one
	// character. The default is '•'.
	ObscuringCharacter string
	// ObscureText hides the text, for passwords.
	ObscureText bool
	// Autocorrect says whether the input method corrects the text.
	Autocorrect foundation.Opt[bool]
	// SmartDashesType says whether dashes are replaced.
	SmartDashesType services.SmartDashesType
	// SmartQuotesType says whether quotes are replaced.
	SmartQuotesType services.SmartQuotesType
	// DisableSuggestions turns the suggestions of the input method off (Flutter's
	// enableSuggestions is true by default).
	DisableSuggestions bool
	// MaxLines is the maximum number of lines; the default is 1 and
	// rendering.MaxLinesUnlimited (Flutter's null) is no limit.
	MaxLines foundation.Opt[int]
	// MinLines is the minimum number of lines.
	MinLines foundation.Opt[int]
	// Expands makes the field as tall as its parent allows; MaxLines and
	// MinLines must then be unset.
	Expands bool
	// MaxLength is the maximum number of characters; TextFieldNoMaxLength counts
	// them without a limit.
	MaxLength foundation.Opt[int]
	// MaxLengthEnforcement says how the limit is enforced; the default is the one
	// of the platform.
	MaxLengthEnforcement services.MaxLengthEnforcement
	// OnChanged is called when the text changes.
	OnChanged func(text string)
	// OnEditingComplete is called when the user finishes editing.
	OnEditingComplete func()
	// OnSubmitted is called with the text when the user finishes editing.
	OnSubmitted func(text string)
	// OnAppPrivateCommand is called for a private command of the input method.
	OnAppPrivateCommand widgets.AppPrivateCommandCallback
	// InputFormatters change the text the user enters.
	InputFormatters []services.TextInputFormatter
	// Enabled says whether the field is enabled; the default is the one of the
	// decoration, which is true.
	Enabled foundation.Opt[bool]
	// IgnorePointers makes the field ignore the pointers; the default is the
	// opposite of Enabled.
	IgnorePointers foundation.Opt[bool]
	// CursorWidth is the width of the caret; the default is 2.
	CursorWidth foundation.Opt[float64]
	// CursorHeight is the height of the caret; the default is the line height.
	CursorHeight foundation.Opt[float64]
	// CursorRadius rounds the corners of the caret.
	CursorRadius foundation.Opt[ui.Radius]
	// CursorOpacityAnimates makes the caret fade instead of blink.
	CursorOpacityAnimates foundation.Opt[bool]
	// CursorColor is the color of the caret; the default is the one of the
	// selection style, or the primary color.
	CursorColor foundation.Opt[ui.Color]
	// CursorErrorColor is the color of the caret when the field has an error.
	CursorErrorColor foundation.Opt[ui.Color]
	// SelectionHeightStyle is the height of the boxes of the selection.
	SelectionHeightStyle ui.BoxHeightStyle
	// SelectionWidthStyle is the width of the boxes of the selection.
	SelectionWidthStyle ui.BoxWidthStyle
	// KeyboardAppearance is the brightness of the keyboard of the input method;
	// the default is the one of the theme.
	KeyboardAppearance ui.Brightness
	// ScrollPadding is the space kept around the caret when the field scrolls to
	// show it; the default is 20 on all sides.
	ScrollPadding foundation.Opt[painting.EdgeInsets]
	// DragStartBehavior is where a drag starts; the default is start.
	DragStartBehavior gestures.DragStartBehavior
	// EnableInteractiveSelection says whether the user can select text; the
	// default is true unless the text is read-only and obscured.
	EnableInteractiveSelection foundation.Opt[bool]
	// SelectAllOnFocus selects the text when the field gets the focus; the
	// default is true on desktop, for a single-line field.
	SelectAllOnFocus foundation.Opt[bool]
	// SelectionControls build the selection handles; the default is the controls
	// of the platform.
	SelectionControls widgets.TextSelectionControls
	// OnTap is called when the user taps the field.
	OnTap func()
	// OnTapAlwaysCalled calls OnTap for every tap, even a tap that other
	// gestures use.
	OnTapAlwaysCalled bool
	// OnTapOutside is called when a pointer goes down outside the field; the
	// default unfocuses the field.
	OnTapOutside widgets.TapRegionCallback
	// OnTapUpOutside is called when a pointer goes up outside the field.
	OnTapUpOutside widgets.TapRegionUpCallback
	// MouseCursor is the cursor over the field; the default is the text cursor
	// (the basic one when the field is disabled).
	MouseCursor services.MouseCursor
	// BuildCounter builds the counter below the field.
	BuildCounter InputCounterWidgetBuilder
	// ScrollController controls the scrolling of the field.
	ScrollController widgets.ScrollController
	// ScrollPhysics are the physics of the scrolling of the field.
	ScrollPhysics widgets.ScrollPhysics
	// AutofillHints say what the field is for.
	AutofillHints []string
	// ClipBehavior is how the text is clipped; the default is hard edge.
	ClipBehavior ui.Clip
	// DisableIMEPersonalizedLearning turns the learning of the input method off
	// (Flutter's enableIMEPersonalizedLearning is true by default).
	DisableIMEPersonalizedLearning bool
	// EnableInlinePrediction turns the inline prediction of the input method on
	// or off.
	EnableInlinePrediction foundation.Opt[bool]
	// ContextMenuBuilder builds the context menu; the default is an
	// AdaptiveTextSelectionToolbar.
	ContextMenuBuilder widgets.EditableTextContextMenuBuilder
	// NoContextMenu shows no context menu (Flutter's contextMenuBuilder: null).
	NoContextMenu bool
	// CannotRequestFocus keeps the field from taking the focus (Flutter's
	// canRequestFocus is true by default).
	CannotRequestFocus bool
	// HintLocales are the languages the user is expected to type in.
	HintLocales []ui.Locale
	// MagnifierConfiguration is the configuration for the magnifier of this text
	// field.
	//
	// By default (nil), it builds a TextMagnifier on Android and nothing on the
	// other platforms (TextMagnifierAdaptiveMagnifierConfiguration). To disable
	// the magnifier, consider passing TextMagnifierConfigurationDisabled.
	MagnifierConfiguration *widgets.TextMagnifierConfiguration
	// SpellCheckConfiguration details how spell check is performed: the
	// SpellCheckService, the style of the misspelled words and the toolbar with the
	// suggestions. Spell check is disabled for password input, and when the
	// configuration is nil. An enabled configuration needs no more than
	// widgets.NewSpellCheckConfiguration(): the style of the misspelled words
	// defaults to TextFieldMaterialMisspelledTextStyle, the toolbar to
	// TextFieldDefaultSpellCheckSuggestionsToolbarBuilder, and the service to the
	// platform's spell checker (Android only; elsewhere spell check stays off unless
	// a service is given).
	SpellCheckConfiguration *widgets.SpellCheckConfiguration
}

// TextField is a Material text field: the user types text into it with the
// keyboard. It is an EditableText in an InputDecorator.
//
// Deviation: the restoration of the state, the
// stylus handwriting, the content insertion, the deprecated toolbarOptions and
// the system context menu are not ported, and the Cupertino looks of the Apple
// platforms are replaced with the Material one (SPEC §12.10).
//
// Flutter: material/text_field.dart TextField
type TextField struct {
	widgets.StatefulWidget
	o        TextFieldOpts
	maxLines foundation.Opt[int]
}

// NewTextField creates a text field. In debug builds it panics with the tag
// TextFieldObscuringCharacterInvalid when the obscuring character is not one
// character, TextFieldMaxLinesInvalid when the maximum number of lines is not
// positive, TextFieldMinLinesInvalid when the minimum is not positive,
// TextFieldMinLinesGreaterThanMaxLines when it is more than the maximum,
// TextFieldExpandsWithLines when the field expands and has a number of lines,
// TextFieldObscuredMultiline for an obscured field of several lines,
// TextFieldMaxLengthInvalid when the limit is neither TextFieldNoMaxLength nor
// positive, and TextFieldNewlineActionWithTextKeyboard for the newline action
// with the text keyboard on several lines.
//
// Flutter: material/text_field.dart TextField.new
func NewTextField(opts ...TextFieldOpts) *TextField {
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
	if foundation.DebugMode {
		if foundation.GraphemeCount(o.ObscuringCharacter) != 1 {
			panic(errors.New(tagTextFieldObscuringCharacterInvalid, "the obscuring character must be one character", "obscuringCharacter", o.ObscuringCharacter))
		}
		if m, ok := maxLines.Get(); ok && m <= 0 {
			panic(errors.New(tagTextFieldMaxLinesInvalid, "maxLines must be positive", "maxLines", m))
		}
		if m, ok := o.MinLines.Get(); ok && m <= 0 {
			panic(errors.New(tagTextFieldMinLinesInvalid, "minLines must be positive", "minLines", m))
		}
		if mx, ok := maxLines.Get(); ok && o.MinLines.IsSome() && mx < o.MinLines.Value() {
			panic(errors.New(tagTextFieldMinLinesGreaterThanMaxLines, "minLines can't be greater than maxLines", "minLines", o.MinLines.Value(), "maxLines", mx))
		}
		if o.Expands && (maxLines.IsSome() || o.MinLines.IsSome()) {
			panic(errors.New(tagTextFieldExpandsWithLines, "minLines and maxLines must be unset when expands is true"))
		}
		if o.ObscureText && (maxLines.IsNone() || maxLines.Value() != 1) {
			panic(errors.New(tagTextFieldObscuredMultiline, "obscured fields cannot be multiline"))
		}
		if n, ok := o.MaxLength.Get(); ok && n != TextFieldNoMaxLength && n <= 0 {
			panic(errors.New(tagTextFieldMaxLengthInvalid, "maxLength must be TextFieldNoMaxLength or positive", "maxLength", n))
		}
		single := maxLines.IsSome() && maxLines.Value() == 1
		if o.TextInputAction == services.TextInputActionNewline && !single && o.KeyboardType.IsSome() && o.KeyboardType.Value() == services.TextInputTypeText {
			panic(errors.New(tagTextFieldNewlineActionWithTextKeyboard, "use the multiline keyboardType with the newline textInputAction on a multiline TextField"))
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
	o.CursorWidth = foundation.Some(o.CursorWidth.OrElse(2.0))
	o.ScrollPadding = foundation.Some(o.ScrollPadding.OrElse(painting.NewEdgeInsetsAll(20.0)))
	o.EnableInteractiveSelection = foundation.Some(o.EnableInteractiveSelection.OrElse(!o.ReadOnly || !o.ObscureText))
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipHardEdge
	}
	if o.DragStartBehavior == gestures.DragStartBehaviorUnspecified {
		o.DragStartBehavior = gestures.DragStartBehaviorStart
	}
	return &TextField{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), o: o, maxLines: maxLines}
}

// Controller controls the text; nil when the field makes its own.
//
// Flutter: material/text_field.dart TextField.controller
func (w *TextField) Controller() *widgets.TextEditingController { return w.o.Controller }

// FocusNode controls the focus; nil when the field makes its own.
//
// Flutter: material/text_field.dart TextField.focusNode
func (w *TextField) FocusNode() *widgets.FocusNode { return w.o.FocusNode }

// Decoration is the decoration of the field.
//
// Flutter: material/text_field.dart TextField.decoration
func (w *TextField) Decoration() *InputDecoration {
	if w.o.NoDecoration {
		return nil
	}
	if w.o.Decoration == nil {
		return &InputDecoration{}
	}
	return w.o.Decoration
}

// MaxLines is the maximum number of lines; none for no limit.
//
// Flutter: material/text_field.dart TextField.maxLines
func (w *TextField) MaxLines() foundation.Opt[int] { return w.maxLines }

// MinLines is the minimum number of lines.
//
// Flutter: material/text_field.dart TextField.minLines
func (w *TextField) MinLines() foundation.Opt[int] { return w.o.MinLines }

// MaxLength is the maximum number of characters.
//
// Flutter: material/text_field.dart TextField.maxLength
func (w *TextField) MaxLength() foundation.Opt[int] { return w.o.MaxLength }

// ObscureText is whether the text is hidden.
//
// Flutter: material/text_field.dart TextField.obscureText
func (w *TextField) ObscureText() bool { return w.o.ObscureText }

// ReadOnly is whether the text can't be edited.
//
// Flutter: material/text_field.dart TextField.readOnly
func (w *TextField) ReadOnly() bool { return w.o.ReadOnly }

// SelectionEnabled is whether the user can select text.
//
// Flutter: material/text_field.dart TextField.selectionEnabled
func (w *TextField) SelectionEnabled() bool { return w.o.EnableInteractiveSelection.Value() }

// MagnifierConfiguration is the configuration for the magnifier of this text
// field, or nil for the one of the platform (TextMagnifierAdaptiveMagnifierConfiguration).
//
// Flutter: material/text_field.dart TextField.magnifierConfiguration
func (w *TextField) MagnifierConfiguration() *widgets.TextMagnifierConfiguration {
	return w.o.MagnifierConfiguration
}

// SpellCheckConfiguration is the configuration of spell check; nil for none.
//
// Flutter: material/text_field.dart TextField.spellCheckConfiguration
func (w *TextField) SpellCheckConfiguration() *widgets.SpellCheckConfiguration {
	return w.o.SpellCheckConfiguration
}

// TextFieldMaterialMisspelledTextStyle is the TextStyle used to indicate misspelled
// words in the Material style: a red wavy underline.
//
// Flutter: material/text_field.dart TextField.materialMisspelledTextStyle
var TextFieldMaterialMisspelledTextStyle = painting.NewTextStyle(painting.TextStyleOpts{
	Decoration:      foundation.Some(ui.TextDecorationUnderline),
	DecorationColor: foundation.Some(Colors.Red.Primary),
	DecorationStyle: ui.TextDecorationStyleWavy,
})

// TextFieldDefaultSpellCheckSuggestionsToolbarBuilder is the default builder of the
// spell check suggestions toolbar of a TextField: the Material
// SpellCheckSuggestionsToolbar.
//
// Deviation: Flutter builds the Cupertino toolbar on iOS and macOS; there is no
// Cupertino library, so every platform gets the Material toolbar (SPEC §12.10).
//
// Flutter: material/text_field.dart TextField.defaultSpellCheckSuggestionsToolbarBuilder
func TextFieldDefaultSpellCheckSuggestionsToolbarBuilder(_ widgets.BuildContext, editableTextState *widgets.EditableTextState) widgets.Widget {
	return NewSpellCheckSuggestionsToolbarEditableText(editableTextState)
}

// TextFieldInferAndroidSpellCheckConfiguration returns a new configuration where
// the given one has had any missing values replaced with their defaults for the
// Android platform: the Material misspelled text style and the Material toolbar. A
// nil or disabled configuration gives the disabled one.
//
// Deviation: it is used on every platform, as there is no Cupertino configuration
// to infer on iOS and macOS (SPEC §12.10).
//
// Flutter: material/text_field.dart TextField.inferAndroidSpellCheckConfiguration
func TextFieldInferAndroidSpellCheckConfiguration(configuration *widgets.SpellCheckConfiguration) *widgets.SpellCheckConfiguration {
	if configuration == nil || configuration.Equal(widgets.NewSpellCheckConfigurationDisabled()) {
		return widgets.NewSpellCheckConfigurationDisabled()
	}
	misspelledTextStyle := configuration.MisspelledTextStyle()
	if misspelledTextStyle == nil {
		misspelledTextStyle = TextFieldMaterialMisspelledTextStyle
	}
	toolbarBuilder := configuration.SpellCheckSuggestionsToolbarBuilder()
	if toolbarBuilder == nil {
		toolbarBuilder = TextFieldDefaultSpellCheckSuggestionsToolbarBuilder
	}
	return configuration.CopyWith(widgets.SpellCheckConfigurationCopy{
		MisspelledTextStyle: misspelledTextStyle, SpellCheckSuggestionsToolbarBuilder: toolbarBuilder,
	})
}

// CreateState creates the state.
//
// Flutter: material/text_field.dart TextField.createState
func (w *TextField) CreateState() widgets.State { return &textFieldState{} }

// DebugFillProperties adds the main properties of the field.
//
// Flutter: material/text_field.dart TextField.debugFillProperties
func (w *TextField) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	o := w.o
	null := foundation.DiagnosticsPropertyOpts{DefaultValue: foundation.DefaultNull}
	p.Add(objectProp("controller", ptrOrNil(o.Controller)))
	p.Add(objectProp("focusNode", ptrOrNil(o.FocusNode)))
	p.Add(objectProp("undoController", ptrOrNil(o.UndoController)))
	p.Add(objectProp("enabled", optValue(o.Enabled)))
	p.Add(foundation.NewDiagnosticsProperty("keyboardType", optValue(o.KeyboardType), foundation.DiagnosticsPropertyOpts{DefaultValue: services.TextInputTypeText}))
	p.Add(objectProp("style", ptrOrNil(o.Style)))
	p.Add(foundation.NewDiagnosticsProperty("autofocus", o.Autofocus, foundation.DiagnosticsPropertyOpts{DefaultValue: false}))
	p.Add(foundation.NewDiagnosticsProperty("obscuringCharacter", o.ObscuringCharacter, foundation.DiagnosticsPropertyOpts{DefaultValue: "•"}))
	p.Add(foundation.NewDiagnosticsProperty("obscureText", o.ObscureText, foundation.DiagnosticsPropertyOpts{DefaultValue: false}))
	p.Add(objectProp("autocorrect", optValue(o.Autocorrect)))
	p.Add(foundation.NewIntProperty("maxLines", w.maxLines, foundation.NumberPropertyOpts{DefaultValue: 1}))
	p.Add(foundation.NewIntProperty("minLines", o.MinLines))
	p.Add(foundation.NewDiagnosticsProperty("expands", o.Expands, foundation.DiagnosticsPropertyOpts{DefaultValue: false}))
	p.Add(foundation.NewIntProperty("maxLength", o.MaxLength))
	p.Add(foundation.NewDoubleProperty("cursorWidth", o.CursorWidth, foundation.NumberPropertyOpts{DefaultValue: 2.0}))
	p.Add(painting.NewColorProperty("cursorColor", o.CursorColor, null))
	p.Add(painting.NewColorProperty("cursorErrorColor", o.CursorErrorColor, null))
	p.Add(foundation.NewFlagProperty("selectionEnabled", o.EnableInteractiveSelection, foundation.FlagPropertyOpts{DefaultValue: foundation.Some(true), IfFalse: "selection disabled"}))
}

// textFieldState is the state of a TextField.
//
// Flutter: material/text_field.dart _TextFieldState
type textFieldState struct {
	widgets.StateOf[*TextField]
	controller               *widgets.TextEditingController
	focusNode                *widgets.FocusNode
	isHovering               bool
	showSelectionHandles     bool
	selectionBuilder         *textFieldSelectionBuilder
	forcePressEnabled        bool
	editableTextKey          *widgets.GlobalKey[*widgets.EditableTextState]
	internalStatesController *widgets.WidgetStatesController
	statesListener           foundation.ListenerHandle
	focusListener            foundation.ListenerHandle
	focusListenerOn          *widgets.FocusNode
}

// EditableTextKey is the key of the EditableText of the field.
//
// Flutter: material/text_field.dart _TextFieldState.editableTextKey
func (s *textFieldState) EditableTextKey() *widgets.GlobalKey[*widgets.EditableTextState] {
	return s.editableTextKey
}

// ForcePressEnabled is whether a force press selects a word.
//
// Flutter: material/text_field.dart _TextFieldState.forcePressEnabled
func (s *textFieldState) ForcePressEnabled() bool { return s.forcePressEnabled }

// SelectionEnabled is whether the user can select text in the enabled field.
//
// Flutter: material/text_field.dart _TextFieldState.selectionEnabled
func (s *textFieldState) SelectionEnabled() bool {
	return s.Widget().SelectionEnabled() && s.isEnabled()
}

func (s *textFieldState) effectiveController() *widgets.TextEditingController {
	if c := s.Widget().o.Controller; c != nil {
		return c
	}
	return s.controller
}

func (s *textFieldState) effectiveFocusNode() *widgets.FocusNode {
	if f := s.Widget().o.FocusNode; f != nil {
		return f
	}
	if s.focusNode == nil {
		s.focusNode = widgets.NewFocusNode(widgets.FocusNodeOpts{})
	}
	return s.focusNode
}

func (s *textFieldState) statesController() *widgets.WidgetStatesController {
	if c := s.Widget().o.StatesController; c != nil {
		return c
	}
	return s.internalStatesController
}

func (s *textFieldState) effectiveMaxLengthEnforcement() services.MaxLengthEnforcement {
	if e := s.Widget().o.MaxLengthEnforcement; e != services.MaxLengthEnforcementUnspecified {
		return e
	}
	return services.LengthLimitingTextInputFormatterGetDefaultMaxLengthEnforcement(ThemeOf(s.Context()).Platform)
}

func (s *textFieldState) isEnabled() bool {
	w := s.Widget()
	if e, ok := w.o.Enabled.Get(); ok {
		return e
	}
	if d := w.Decoration(); d != nil {
		return !d.Disabled
	}
	return true
}

func (s *textFieldState) currentLength() int {
	return foundation.GraphemeCount(s.effectiveController().Text())
}

func (s *textFieldState) hasIntrinsicError() bool {
	m, ok := s.Widget().o.MaxLength.Get()
	return ok && m > 0 && s.currentLength() > m
}

func (s *textFieldState) hasError() bool {
	d := s.Widget().Decoration()
	return d != nil && (d.ErrorText.IsSome() || d.Error != nil) || s.hasIntrinsicError()
}

func (s *textFieldState) errorColor() ui.Color {
	if c, ok := s.Widget().o.CursorErrorColor.Get(); ok {
		return c
	}
	if st := s.effectiveDecoration().ErrorStyle; st != nil {
		if style := st.Resolve(s.statesController().Value()); style != nil {
			if c, ok := style.Color.Get(); ok {
				return c
			}
		}
	}
	return ThemeOf(s.Context()).ColorScheme.Error()
}

// effectiveDecoration is the decoration of the field with the defaults of the
// theme, the enabled flag and the counter.
//
// Flutter: material/text_field.dart _TextFieldState._getEffectiveDecoration
func (s *textFieldState) effectiveDecoration() InputDecoration {
	w := s.Widget()
	localizations := MaterialLocalizationsOf(s.Context())
	themeData := ThemeOf(s.Context())
	decorationTheme := InputDecorationThemeOf(s.Context())
	base := InputDecoration{}
	if d := w.Decoration(); d != nil {
		base = *d
	}
	hintMaxLines := base.HintMaxLines
	if hintMaxLines.IsNone() {
		hintMaxLines = decorationTheme.HintMaxLines
	}
	if hintMaxLines.IsNone() && w.maxLines.IsSome() {
		hintMaxLines = w.maxLines
	}
	effective := base.ApplyDefaults(decorationTheme).CopyWith(InputDecorationCopy{
		Disabled:     foundation.Some(!s.isEnabled()),
		HintMaxLines: hintMaxLines,
	})
	// No need to build anything if counter or counterText were given directly.
	if effective.Counter != nil || effective.CounterText.IsSome() {
		return effective
	}
	currentLength := s.currentLength()
	// If buildCounter was provided, use it to generate a counter widget.
	if w.o.BuildCounter != nil {
		isFocused := s.effectiveFocusNode().HasFocus()
		var counter widgets.Widget
		// If buildCounter returns nil, don't add a counter widget to the field.
		if built := w.o.BuildCounter(s.Context(), currentLength, w.o.MaxLength, isFocused); built != nil {
			counter = widgets.NewSemantics(built, widgets.SemanticsOpts{Container: true, LiveRegion: foundation.Some(isFocused)})
		}
		return effective.CopyWith(InputDecorationCopy{Counter: foundation.Some(counter)})
	}
	maxLength, ok := w.o.MaxLength.Get()
	if !ok {
		return effective // No counter widget
	}
	counterText := strconv.Itoa(currentLength)
	semanticCounterText := ""
	// Handle a real maxLength (positive number)
	if maxLength > 0 {
		// Show the maxLength in the counter
		counterText += "/" + strconv.Itoa(maxLength)
		remaining := min(max(maxLength-currentLength, 0), maxLength)
		semanticCounterText = localizations.RemainingTextFieldCharacterCount(remaining)
	}
	if s.hasIntrinsicError() {
		errorText := effective.ErrorText
		if errorText.IsNone() {
			errorText = foundation.Some("")
		}
		counterStyle := effective.ErrorStyle
		if counterStyle == nil {
			style := m3CounterErrorStyle(s.Context())
			if !themeData.UseMaterial3 {
				style = m2CounterErrorStyle(s.Context())
			}
			counterStyle = widgets.NewWidgetStatePropertyAll(style)
		}
		return effective.CopyWith(InputDecorationCopy{
			ErrorText:           errorText,
			CounterStyle:        foundation.Some(counterStyle),
			CounterText:         foundation.Some(counterText),
			SemanticCounterText: foundation.Some(semanticCounterText),
		})
	}
	return effective.CopyWith(InputDecorationCopy{
		CounterText:         foundation.Some(counterText),
		SemanticCounterText: foundation.Some(semanticCounterText),
	})
}

// InitState creates the local controller and the states controller, and listens
// to the focus.
//
// Flutter: material/text_field.dart _TextFieldState.initState
func (s *textFieldState) InitState() {
	s.editableTextKey = widgets.NewGlobalKey[*widgets.EditableTextState]()
	s.selectionBuilder = &textFieldSelectionBuilder{state: s}
	widgets.InitTextSelectionGestureDetectorBuilder(&s.selectionBuilder.TextSelectionGestureDetectorBuilder, s.selectionBuilder, s)
	if s.Widget().o.Controller == nil {
		s.controller = widgets.NewTextEditingController()
	}
	s.effectiveFocusNode().SetCanRequestFocus(!s.Widget().o.CannotRequestFocus && s.isEnabled())
	s.listenToFocus()
	s.initStatesController()
}

func (s *textFieldState) listenToFocus() {
	s.focusListenerOn = s.effectiveFocusNode()
	s.focusListener = s.focusListenerOn.AddListener(s.handleFocusChanged)
}

func (s *textFieldState) canRequestFocus() bool {
	mode, ok := widgets.MediaQueryMaybeNavigationModeOf(s.Context())
	if ok && mode == widgets.NavigationModeDirectional {
		return true
	}
	return !s.Widget().o.CannotRequestFocus && s.isEnabled()
}

// DidChangeDependencies updates whether the focus can be requested.
//
// Flutter: material/text_field.dart _TextFieldState.didChangeDependencies
func (s *textFieldState) DidChangeDependencies() {
	s.effectiveFocusNode().SetCanRequestFocus(s.canRequestFocus())
}

// DidUpdateWidget follows the controller, the focus node and the states
// controller of the new widget.
//
// Flutter: material/text_field.dart _TextFieldState.didUpdateWidget
func (s *textFieldState) DidUpdateWidget(old *TextField) {
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
	s.effectiveFocusNode().SetCanRequestFocus(s.canRequestFocus())
	if s.effectiveFocusNode().HasFocus() && w.o.ReadOnly != old.o.ReadOnly && s.isEnabled() {
		if s.effectiveController().Selection().IsCollapsed() {
			s.showSelectionHandles = !w.o.ReadOnly
		}
	}
	if w.o.StatesController == old.o.StatesController {
		s.updateStates()
	} else {
		if old.o.StatesController != nil {
			old.o.StatesController.RemoveListener(s.statesListener)
		} else if s.internalStatesController != nil {
			s.internalStatesController.RemoveListener(s.statesListener)
		}
		if w.o.StatesController != nil && s.internalStatesController != nil {
			s.internalStatesController.Dispose()
			s.internalStatesController = nil
		}
		s.initStatesController()
	}
}

func (s *textFieldState) updateStates() {
	c := s.statesController()
	c.Update(widgets.WidgetStateDisabled, !s.isEnabled())
	c.Update(widgets.WidgetStateHovered, s.isHovering)
	c.Update(widgets.WidgetStateFocused, s.effectiveFocusNode().HasFocus())
	c.Update(widgets.WidgetStateError, s.hasError())
}

func (s *textFieldState) initStatesController() {
	if s.Widget().o.StatesController == nil {
		s.internalStatesController = widgets.NewWidgetStatesController()
	}
	s.updateStates()
	s.statesListener = s.statesController().AddListener(func() {
		// Force a rebuild to resolve WidgetStateProperty properties.
		s.SetState(func() {})
	})
}

// Dispose releases what the state made.
//
// Flutter: material/text_field.dart _TextFieldState.dispose
func (s *textFieldState) Dispose() {
	s.focusListenerOn.RemoveListener(s.focusListener)
	if s.focusNode != nil {
		s.focusNode.Dispose()
	}
	if s.controller != nil {
		s.controller.Dispose()
	}
	s.statesController().RemoveListener(s.statesListener)
	if s.internalStatesController != nil {
		s.internalStatesController.Dispose()
	}
}

func (s *textFieldState) editableText() *widgets.EditableTextState {
	e, _ := s.editableTextKey.CurrentState()
	return e
}

func (s *textFieldState) requestKeyboard() {
	if e := s.editableText(); e != nil {
		e.RequestKeyboard()
	}
}

// shouldShowSelectionHandles is whether the handles are shown for a change of
// the selection.
//
// Flutter: material/text_field.dart _TextFieldState._shouldShowSelectionHandles
func (s *textFieldState) shouldShowSelectionHandles(cause services.SelectionChangedCause) bool {
	// When the text field is activated by something that doesn't trigger the
	// selection toolbar, we shouldn't show the handles either.
	b := &s.selectionBuilder.TextSelectionGestureDetectorBuilder
	if !b.ShouldShowSelectionToolbar() || !b.ShouldShowSelectionHandles() {
		return false
	}
	if cause == services.SelectionChangedCauseKeyboard {
		return false
	}
	if s.Widget().o.ReadOnly && s.effectiveController().Selection().IsCollapsed() {
		return false
	}
	if !s.isEnabled() {
		return false
	}
	if cause == services.SelectionChangedCauseLongPress || cause == services.SelectionChangedCauseStylusHandwriting {
		return true
	}
	return s.effectiveController().Text() != ""
}

func (s *textFieldState) handleFocusChanged() {
	// Rebuild the widget on focus change to show/hide the text selection
	// highlight.
	s.SetState(func() {})
	s.statesController().Update(widgets.WidgetStateFocused, s.effectiveFocusNode().HasFocus())
}

func (s *textFieldState) handleSelectionChanged(selection services.TextSelection, cause services.SelectionChangedCause) {
	willShowSelectionHandles := s.shouldShowSelectionHandles(cause)
	if willShowSelectionHandles != s.showSelectionHandles {
		s.SetState(func() { s.showSelectionHandles = willShowSelectionHandles })
	}
	if cause == services.SelectionChangedCauseLongPress {
		if e := s.editableText(); e != nil {
			e.BringIntoView(ui.NewTextPosition(selection.ExtentOffset))
		}
	}
	switch ThemeOf(s.Context()).Platform {
	case foundation.TargetPlatformMacOS, foundation.TargetPlatformLinux, foundation.TargetPlatformWindows:
		if cause == services.SelectionChangedCauseDrag {
			if e := s.editableText(); e != nil {
				e.HideToolbar(true)
			}
		}
	}
}

func (s *textFieldState) handleSelectionHandleTapped() {
	if s.effectiveController().Selection().IsCollapsed() {
		s.editableText().ToggleToolbar(true)
	}
}

func (s *textFieldState) handleHover(hovering bool) {
	if hovering != s.isHovering {
		s.SetState(func() { s.isHovering = hovering })
		s.statesController().Update(widgets.WidgetStateHovered, s.isHovering)
	}
}

// inputStyleForState merges the style that depends on the state of the field,
// the one of the theme, with the style that was given.
//
// Flutter: material/text_field.dart _TextFieldState._getInputStyleForState
func (s *textFieldState) inputStyleForState(style *painting.TextStyle) *painting.TextStyle {
	theme := ThemeOf(s.Context())
	states := s.statesController().Value()
	stateStyle := m2StateInputStyle(s.Context())
	if theme.UseMaterial3 {
		stateStyle = m3StateInputStyle(s.Context())
	}
	return style.Merge(stateStyle.Resolve(states))
}

// providedStyle is the style that was given, resolved for the states.
func (s *textFieldState) providedStyle() *painting.TextStyle {
	w := s.Widget()
	if w.o.Style != nil {
		return w.o.Style
	}
	if w.o.StateStyle != nil {
		return w.o.StateStyle.Resolve(s.statesController().Value())
	}
	return nil
}

// Build builds the EditableText inside the decorator, the gestures and the
// semantics.
//
// Flutter: material/text_field.dart _TextFieldState.build
func (s *textFieldState) Build(bc widgets.BuildContext) widgets.Widget {
	DebugCheckHasMaterial(bc)
	DebugCheckHasMaterialLocalizations(bc)
	widgets.DebugCheckHasDirectionality(bc)
	w := s.Widget()
	if foundation.DebugMode {
		if st := s.providedStyle(); st != nil && !st.Inherit && (st.FontSize.IsNone() || st.TextBaseline == ui.TextBaselineUnspecified) {
			panic(errors.New(tagTextFieldStyleNotInheriting, "an inherit false style must supply fontSize and textBaseline"))
		}
	}
	theme := ThemeOf(bc)
	selectionStyle := widgets.DefaultSelectionStyleOf(bc)
	provided := s.providedStyle()
	base := theme.TextTheme.TitleMedium
	if theme.UseMaterial3 {
		base = m3InputStyle(bc)
	}
	style := s.inputStyleForState(base).Merge(provided)
	keyboardAppearance := w.o.KeyboardAppearance
	if keyboardAppearance == ui.BrightnessUnspecified {
		keyboardAppearance = theme.Brightness()
	}
	controller := s.effectiveController()
	focusNode := s.effectiveFocusNode()
	var formatters []services.TextInputFormatter
	formatters = append(formatters, w.o.InputFormatters...)
	if w.o.MaxLength.IsSome() {
		formatters = append(formatters, services.NewLengthLimitingTextInputFormatter(w.o.MaxLength,
			services.LengthLimitingTextInputFormatterOpts{MaxLengthEnforcement: s.effectiveMaxLengthEnforcement()}))
	}
	textSelectionControls := w.o.SelectionControls
	var paintCursorAboveText bool
	cursorOpacityAnimates := w.o.CursorOpacityAnimates
	cursorOffset := foundation.None[ui.Offset]()
	var cursorColor, selectionColor ui.Color
	autocorrectionTextRectColor := foundation.None[ui.Color]()
	cursorRadius := w.o.CursorRadius
	var handleDidGainAccessibilityFocus, handleDidLoseAccessibilityFocus func()
	cursorOrDefault := func() ui.Color {
		if s.hasError() {
			return s.errorColor()
		}
		if c, ok := w.o.CursorColor.Get(); ok {
			return c
		}
		if c, ok := selectionStyle.CursorColor().Get(); ok {
			return c
		}
		return theme.ColorScheme.Primary()
	}
	selectionOrDefault := func() ui.Color {
		if c, ok := selectionStyle.SelectionColor().Get(); ok {
			return c
		}
		return theme.ColorScheme.Primary().WithOpacity(0.40)
	}
	activateOnAccessibilityFocus := func() {
		handleDidGainAccessibilityFocus = func() {
			// Automatically activate the TextField when it receives accessibility focus.
			if !focusNode.HasFocus() && focusNode.CanRequestFocus() {
				focusNode.RequestFocus()
			}
		}
		handleDidLoseAccessibilityFocus = func() { focusNode.Unfocus() }
	}
	// Deviation: the Apple platforms get the Material colors instead of the
	// ones of the Cupertino theme, which is not ported.
	switch theme.Platform {
	case foundation.TargetPlatformIOS:
		s.forcePressEnabled = true
		if textSelectionControls == nil {
			textSelectionControls = MaterialTextSelectionHandleControls
		}
		paintCursorAboveText = true
		cursorOpacityAnimates = foundation.Some(cursorOpacityAnimates.OrElse(true))
		cursorColor, selectionColor = cursorOrDefault(), selectionOrDefault()
		if cursorRadius.IsNone() {
			cursorRadius = foundation.Some(ui.NewRadiusCircular(2.0))
		}
		cursorOffset = foundation.Some(ui.Offset{Dx: iOSHorizontalOffset / widgets.MediaQueryDevicePixelRatioOf(bc)})
		autocorrectionTextRectColor = foundation.Some(selectionColor)
	case foundation.TargetPlatformMacOS:
		s.forcePressEnabled = false
		if textSelectionControls == nil {
			textSelectionControls = DesktopTextSelectionHandleControls
		}
		paintCursorAboveText = true
		cursorOpacityAnimates = foundation.Some(cursorOpacityAnimates.OrElse(false))
		cursorColor, selectionColor = cursorOrDefault(), selectionOrDefault()
		if cursorRadius.IsNone() {
			cursorRadius = foundation.Some(ui.NewRadiusCircular(2.0))
		}
		cursorOffset = foundation.Some(ui.Offset{Dx: iOSHorizontalOffset / widgets.MediaQueryDevicePixelRatioOf(bc)})
		activateOnAccessibilityFocus()
	case foundation.TargetPlatformAndroid, foundation.TargetPlatformFuchsia:
		s.forcePressEnabled = false
		if textSelectionControls == nil {
			textSelectionControls = MaterialTextSelectionHandleControls
		}
		cursorOpacityAnimates = foundation.Some(cursorOpacityAnimates.OrElse(false))
		cursorColor, selectionColor = cursorOrDefault(), selectionOrDefault()
	default: // linux, windows
		s.forcePressEnabled = false
		if textSelectionControls == nil {
			textSelectionControls = DesktopTextSelectionHandleControls
		}
		cursorOpacityAnimates = foundation.Some(cursorOpacityAnimates.OrElse(false))
		cursorColor, selectionColor = cursorOrDefault(), selectionOrDefault()
		activateOnAccessibilityFocus()
	}
	var controls widgets.TextSelectionControls
	if w.SelectionEnabled() {
		controls = textSelectionControls
	}
	var selectionColorOpt foundation.Opt[ui.Color]
	if focusNode.HasFocus() {
		// Only show the selection highlight when the text field is focused.
		selectionColorOpt = foundation.Some(selectionColor)
	}
	contextMenuBuilder := w.o.ContextMenuBuilder
	switch {
	case w.o.NoContextMenu:
		contextMenuBuilder = nil
	case contextMenuBuilder == nil:
		contextMenuBuilder = defaultTextFieldContextMenuBuilder
	}
	magnifierConfiguration := w.o.MagnifierConfiguration
	if magnifierConfiguration == nil {
		magnifierConfiguration = TextMagnifierAdaptiveMagnifierConfiguration()
	}
	maxLines := w.maxLines
	if maxLines.IsNone() {
		maxLines = foundation.Some(rendering.MaxLinesUnlimited)
	}
	// Set configuration as disabled if not otherwise specified. If specified,
	// ensure that configuration uses the correct style for misspelled words for
	// the current platform, unless a custom style is specified.
	spellCheckConfiguration := TextFieldInferAndroidSpellCheckConfiguration(w.o.SpellCheckConfiguration)
	var editable widgets.Widget = widgets.NewRepaintBoundary(widgets.NewEditableText(controller, focusNode, style, cursorColor, ui.NewColor(0xFF8E8E93),
		widgets.EditableTextOpts{
			Key:                            s.editableTextKey,
			ReadOnly:                       w.o.ReadOnly || !s.isEnabled(),
			ShowCursor:                     w.o.ShowCursor,
			ShowSelectionHandles:           s.showSelectionHandles,
			UndoController:                 w.o.UndoController,
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
			SelectionColor:                 selectionColorOpt,
			SelectionControls:              controls,
			OnChanged:                      w.o.OnChanged,
			OnSelectionChanged:             s.handleSelectionChanged,
			OnEditingComplete:              w.o.OnEditingComplete,
			OnSubmitted:                    w.o.OnSubmitted,
			OnAppPrivateCommand:            w.o.OnAppPrivateCommand,
			GroupID:                        w.o.GroupID,
			OnSelectionHandleTapped:        s.handleSelectionHandleTapped,
			OnTapOutside:                   w.o.OnTapOutside,
			OnTapUpOutside:                 w.o.OnTapUpOutside,
			InputFormatters:                formatters,
			RendererIgnoresPointer:         true,
			MouseCursor:                    services.MouseCursorDefer, // TextField will handle the cursor
			CursorWidth:                    w.o.CursorWidth,
			CursorHeight:                   w.o.CursorHeight,
			CursorRadius:                   cursorRadius,
			CursorOpacityAnimates:          cursorOpacityAnimates.Value(),
			CursorOffset:                   cursorOffset,
			PaintCursorAboveText:           paintCursorAboveText,
			SelectionHeightStyle:           w.o.SelectionHeightStyle,
			SelectionWidthStyle:            w.o.SelectionWidthStyle,
			ScrollPadding:                  w.o.ScrollPadding,
			KeyboardAppearance:             keyboardAppearance,
			EnableInteractiveSelection:     w.o.EnableInteractiveSelection,
			SelectAllOnFocus:               w.o.SelectAllOnFocus,
			DragStartBehavior:              w.o.DragStartBehavior,
			ScrollController:               w.o.ScrollController,
			ScrollPhysics:                  w.o.ScrollPhysics,
			AutofillHints:                  w.o.AutofillHints,
			AutocorrectionTextRectColor:    autocorrectionTextRectColor,
			ClipBehavior:                   w.o.ClipBehavior,
			DisableIMEPersonalizedLearning: w.o.DisableIMEPersonalizedLearning,
			EnableInlinePrediction:         w.o.EnableInlinePrediction,
			ContextMenuBuilder:             contextMenuBuilder,
			HintLocales:                    w.o.HintLocales,
			MagnifierConfiguration:         magnifierConfiguration,
			SpellCheckConfiguration:        spellCheckConfiguration,
		}))
	child := editable
	if w.Decoration() != nil {
		child = widgets.NewAnimatedBuilder(foundation.MergeListenables(focusNode, controller), func(_ widgets.BuildContext, built widgets.Widget) widgets.Widget {
			return NewInputDecorator(s.effectiveDecoration(), InputDecoratorOpts{
				BaseStyle:         w.o.Style,
				TextAlign:         w.o.TextAlign,
				TextAlignVertical: w.o.TextAlignVertical,
				IsHovering:        s.isHovering,
				IsFocused:         focusNode.HasFocus(),
				IsEmpty:           controller.Value().Text == "",
				Expands:           w.o.Expands,
				Child:             built,
			})
		}, editable)
	}
	cursorSource := w.o.MouseCursor
	if cursorSource == nil {
		cursorSource = widgets.WidgetStateMouseCursorTextable
	}
	effectiveMouseCursor := widgets.WidgetStateResolveAs[services.MouseCursor](cursorSource, s.statesController().Value())
	semanticsMaxValueLength := foundation.None[int]()
	if m, ok := w.o.MaxLength.Get(); ok && s.effectiveMaxLengthEnforcement() != services.MaxLengthEnforcementNone && m > 0 {
		semanticsMaxValueLength = foundation.Some(m)
	}
	ignoring := !s.isEnabled()
	if i, ok := w.o.IgnorePointers.Get(); ok {
		ignoring = i
	}
	enabled := s.isEnabled()
	return widgets.NewMouseRegion(
		widgets.NewTextFieldTapRegion(
			widgets.NewIgnorePointer(
				widgets.NewAnimatedBuilder(controller, func(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
					var onTap func()
					if !w.o.ReadOnly {
						onTap = func() {
							if !s.effectiveController().Selection().IsValid() {
								s.effectiveController().SetSelection(services.NewTextSelectionCollapsed(len(s.effectiveController().Text())))
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
								// If the platform requested focus, that means that previously the
								// platform believed that the text field did not have focus (even
								// though Flutter's widget system believed otherwise). This likely
								// means that the on-screen keyboard is hidden, or more generally,
								// there is no current editing session in this field. To correct
								// that, keyboard must be requested.
								s.requestKeyboard()
							}
						}
					}
					return widgets.NewSemantics(child, widgets.SemanticsOpts{
						Enabled:                     foundation.Some(enabled),
						MaxValueLength:              semanticsMaxValueLength,
						CurrentValueLength:          foundation.Some(s.currentLength()),
						OnTap:                       onTap,
						OnDidGainAccessibilityFocus: handleDidGainAccessibilityFocus,
						OnDidLoseAccessibilityFocus: handleDidLoseAccessibilityFocus,
						OnFocus:                     onFocus,
					})
				}, s.selectionBuilder.BuildGestureDetector(child, widgets.TextSelectionGestureDetectorBuildOpts{Behavior: rendering.HitTestBehaviorTranslucent})),
				widgets.IgnorePointerOpts{Ignoring: foundation.Some(ignoring)})),
		widgets.MouseRegionOpts{
			Cursor:  effectiveMouseCursor,
			OnEnter: func(*gestures.PointerEnterEvent) { s.handleHover(true) },
			OnExit:  func(*gestures.PointerExitEvent) { s.handleHover(false) },
		})
}

// defaultTextFieldContextMenuBuilder builds the toolbar of the platform.
//
// Flutter: material/text_field.dart TextField._defaultContextMenuBuilder
func defaultTextFieldContextMenuBuilder(_ widgets.BuildContext, state *widgets.EditableTextState) widgets.Widget {
	return NewAdaptiveTextSelectionToolbarEditableText(state)
}

// m2StateInputStyle is the text style that depends on the state, with Material 2.
//
// Flutter: material/text_field.dart _m2StateInputStyle
func m2StateInputStyle(bc widgets.BuildContext) widgets.WidgetStateProperty[*painting.TextStyle] {
	return widgets.WidgetStateTextStyleResolveWith(func(states widgets.WidgetStates) *painting.TextStyle {
		theme := ThemeOf(bc)
		if states.Has(widgets.WidgetStateDisabled) {
			return painting.NewTextStyle(painting.TextStyleOpts{Color: foundation.Some(theme.DisabledColor)})
		}
		return painting.NewTextStyle(painting.TextStyleOpts{Color: theme.TextTheme.TitleMedium.Color})
	})
}

// m2CounterErrorStyle is the style of the counter of a field with an error, with
// Material 2.
//
// Flutter: material/text_field.dart _m2CounterErrorStyle
func m2CounterErrorStyle(bc widgets.BuildContext) *painting.TextStyle {
	theme := ThemeOf(bc)
	return theme.TextTheme.BodySmall.CopyWith(painting.TextStyleCopy{Color: foundation.Some(theme.ColorScheme.Error())})
}

// m3StateInputStyle is the text style that depends on the state, with Material 3.
//
// Flutter: material/text_field.dart _m3StateInputStyle
func m3StateInputStyle(bc widgets.BuildContext) widgets.WidgetStateProperty[*painting.TextStyle] {
	return widgets.WidgetStateTextStyleResolveWith(func(states widgets.WidgetStates) *painting.TextStyle {
		color := ThemeOf(bc).TextTheme.BodyLarge.Color
		if states.Has(widgets.WidgetStateDisabled) {
			if c, ok := color.Get(); ok {
				color = foundation.Some(c.WithOpacity(0.38))
			}
		}
		return painting.NewTextStyle(painting.TextStyleOpts{Color: color})
	})
}

// m3InputStyle is the style of the text, with Material 3.
//
// Flutter: material/text_field.dart _m3InputStyle
func m3InputStyle(bc widgets.BuildContext) *painting.TextStyle {
	return ThemeOf(bc).TextTheme.BodyLarge
}

// m3CounterErrorStyle is the style of the counter of a field with an error, with
// Material 3.
//
// Flutter: material/text_field.dart _m3CounterErrorStyle
func m3CounterErrorStyle(bc widgets.BuildContext) *painting.TextStyle {
	theme := ThemeOf(bc)
	return theme.TextTheme.BodySmall.CopyWith(painting.TextStyleCopy{Color: foundation.Some(theme.ColorScheme.Error())})
}

// ptrOrNil is the pointer as an any, or untyped nil when it is nil.
func ptrOrNil[T any](p *T) any {
	if p == nil {
		return nil
	}
	return p
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagTextFieldExpandsWithLines              = "TextFieldExpandsWithLines"
	tagTextFieldMaxLengthInvalid              = "TextFieldMaxLengthInvalid"
	tagTextFieldMaxLinesInvalid               = "TextFieldMaxLinesInvalid"
	tagTextFieldMinLinesGreaterThanMaxLines   = "TextFieldMinLinesGreaterThanMaxLines"
	tagTextFieldMinLinesInvalid               = "TextFieldMinLinesInvalid"
	tagTextFieldNewlineActionWithTextKeyboard = "TextFieldNewlineActionWithTextKeyboard"
	tagTextFieldObscuredMultiline             = "TextFieldObscuredMultiline"
	tagTextFieldObscuringCharacterInvalid     = "TextFieldObscuringCharacterInvalid"
	tagTextFieldStyleNotInheriting            = "TextFieldStyleNotInheriting"
)
