package cupertino

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// CupertinoTextFormFieldRowOpts are the optional parameters of NewCupertinoTextFormFieldRow: the ones of a
// CupertinoFormRow, of a borderless CupertinoTextField and of a FormField.
//
// Flutter: cupertino/text_form_field_row.dart CupertinoTextFormFieldRow.new
type CupertinoTextFormFieldRowOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Prefix is shown at the start of the row, before the field; it is usually a label.
	Prefix widgets.Widget
	// Padding is the padding of the row; the default is the one of CupertinoFormRow.
	Padding painting.EdgeInsetsGeometry
	// Controller is the controller of the text; it can't be given with an InitialValue.
	Controller *widgets.TextEditingController
	// InitialValue is the initial text; it can't be given with a Controller (the text of the controller is the
	// initial one then).
	InitialValue foundation.Opt[string]
	// FocusNode is the focus node of the field.
	FocusNode *widgets.FocusNode
	// Decoration is painted around the field; the default is none.
	Decoration *painting.BoxDecoration
	// KeyboardType is the type of keyboard.
	KeyboardType foundation.Opt[services.TextInputType]
	// TextCapitalization is how the keyboard capitalizes.
	TextCapitalization services.TextCapitalization
	// TextInputAction is the action of the keyboard's action button.
	TextInputAction services.TextInputAction
	// Style is the style of the text.
	Style *painting.TextStyle
	// StrutStyle is the strut style of the text.
	StrutStyle *painting.StrutStyle
	// TextDirection is the direction of the text.
	TextDirection ui.TextDirection
	// TextAlign is how the text is aligned horizontally.
	TextAlign ui.TextAlign
	// TextAlignVertical is how the text is aligned vertically.
	TextAlignVertical foundation.Opt[painting.TextAlignVertical]
	// Autofocus asks for the focus when the field is built.
	Autofocus bool
	// ReadOnly says that the text cannot be edited.
	ReadOnly bool
	// ShowCursor is whether the cursor is shown; the default is when the field is focused.
	ShowCursor foundation.Opt[bool]
	// ObscuringCharacter replaces each character when the text is obscured; the default is a bullet.
	ObscuringCharacter string
	// ObscureText hides the text.
	ObscureText bool
	// Autocorrect is whether the keyboard corrects the text; the default is true.
	Autocorrect foundation.Opt[bool]
	// SmartDashesType is how dashes are made smart.
	SmartDashesType services.SmartDashesType
	// SmartQuotesType is how quotes are made smart.
	SmartQuotesType services.SmartQuotesType
	// DisableSuggestions turns off the suggestions of the keyboard (Flutter's enableSuggestions is true by
	// default).
	DisableSuggestions bool
	// MaxLines is the maximum number of lines; the default is 1.
	MaxLines foundation.Opt[int]
	// MinLines is the minimum number of lines.
	MinLines foundation.Opt[int]
	// Expands makes the field fill its parent.
	Expands bool
	// MaxLength is the maximum number of characters.
	MaxLength foundation.Opt[int]
	// OnChanged is called when the text changes.
	OnChanged func(text string)
	// OnTap is called when the field is tapped.
	OnTap func()
	// OnEditingComplete is called when the user finishes editing.
	OnEditingComplete func()
	// OnFieldSubmitted is called with the text when the user submits it.
	OnFieldSubmitted func(text string)
	// OnSaved is called with the final value when the form is saved.
	OnSaved widgets.FormFieldSetter[string]
	// Validator validates the text; it returns the error text, or none if the text is valid.
	Validator widgets.FormFieldValidator[string]
	// InputFormatters change the text as the user types.
	InputFormatters []services.TextInputFormatter
	// Enabled is whether the field answers input; the default is true.
	Enabled foundation.Opt[bool]
	// CursorWidth is the width of the cursor; the default is 2.
	CursorWidth foundation.Opt[float64]
	// CursorHeight is the height of the cursor; the default is the height of the font.
	CursorHeight foundation.Opt[float64]
	// CursorColor is the color of the cursor.
	CursorColor ui.ColorValue
	// KeyboardAppearance is the brightness of the keyboard.
	KeyboardAppearance ui.Brightness
	// ScrollPadding is how far from the edges the field is scrolled when it is brought into view; the default is
	// 20.
	ScrollPadding foundation.Opt[painting.EdgeInsets]
	// EnableInteractiveSelection is whether the user can select the text; the default is true.
	EnableInteractiveSelection foundation.Opt[bool]
	// SelectionControls are the handles of the selection.
	SelectionControls widgets.TextSelectionControls
	// ScrollPhysics is the physics of the scroll of the text.
	ScrollPhysics widgets.ScrollPhysics
	// AutofillHints are the hints for the autofill service.
	AutofillHints []string
	// AutovalidateMode is when the field validates; the default is disabled.
	AutovalidateMode widgets.AutovalidateMode
	// Placeholder is the text shown while the field is empty.
	Placeholder foundation.Opt[string]
	// PlaceholderStyle is the style of the placeholder; the default is the weight 400 in the placeholder color.
	PlaceholderStyle *painting.TextStyle
	// ContextMenuBuilder builds the context menu; the default is the adaptive toolbar of the platform.
	ContextMenuBuilder widgets.EditableTextContextMenuBuilder
	// NoContextMenu shows no context menu (Flutter's contextMenuBuilder is null).
	NoContextMenu bool
	// SpellCheckConfiguration is the configuration of the spell check.
	SpellCheckConfiguration *widgets.SpellCheckConfiguration
	// SelectionHeightStyle is the height style of the selection boxes.
	SelectionHeightStyle ui.BoxHeightStyle
	// SelectionWidthStyle is the width style of the selection boxes.
	SelectionWidthStyle ui.BoxWidthStyle
}

// CupertinoTextFormFieldRow is a CupertinoFormRow with a borderless CupertinoTextField in it, in a FormField: the text
// is the value of the form field, it is validated, and the error is the one of the row.
//
// Deviation: Dart's CupertinoTextFormFieldRow is a FormField<String> with its own state; this one is a stateful widget
// that builds a FormField<String> and keeps the controller in sync with it, as TextFormField does (SPEC §12.10). The
// restoration of the state and the deprecated toolbarOptions are not ported.
//
// Flutter: cupertino/text_form_field_row.dart CupertinoTextFormFieldRow
type CupertinoTextFormFieldRow struct {
	widgets.StatefulWidget
	o CupertinoTextFormFieldRowOpts
}

// NewCupertinoTextFormFieldRow creates the row. In debug builds it panics with the tag
// CupertinoTextFormFieldRowInitialValueWithController when it has both an initial value and a controller, and with
// the tags of NewCupertinoTextFieldBorderless for the other parameters.
//
// Flutter: cupertino/text_form_field_row.dart CupertinoTextFormFieldRow.new
func NewCupertinoTextFormFieldRow(opts ...CupertinoTextFormFieldRowOpts) *CupertinoTextFormFieldRow {
	o := firstOf(opts)
	if foundation.DebugMode && o.InitialValue.IsSome() && o.Controller != nil {
		panic(errors.New(tagCupertinoTextFormFieldRowInitialValueWithController, "a CupertinoTextFormFieldRow can't have both an initialValue and a controller"))
	}
	w := &CupertinoTextFormFieldRow{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), o: o}
	// Check the parameters of the text field now, as Flutter's constructor does.
	NewCupertinoTextFieldBorderless(w.fieldOpts(nil, nil))
	return w
}

// Prefix is shown at the start of the row.
//
// Flutter: cupertino/text_form_field_row.dart CupertinoTextFormFieldRow.prefix
func (w *CupertinoTextFormFieldRow) Prefix() widgets.Widget { return w.o.Prefix }

// Padding is the padding of the row.
//
// Flutter: cupertino/text_form_field_row.dart CupertinoTextFormFieldRow.padding
func (w *CupertinoTextFormFieldRow) Padding() painting.EdgeInsetsGeometry { return w.o.Padding }

// Controller is the controller of the text; nil when the row makes its own.
//
// Flutter: cupertino/text_form_field_row.dart CupertinoTextFormFieldRow.controller
func (w *CupertinoTextFormFieldRow) Controller() *widgets.TextEditingController {
	return w.o.Controller
}

// InitialValue is the text that the row starts with, and is reset to.
//
// Flutter: cupertino/text_form_field_row.dart CupertinoTextFormFieldRow.initialValue
func (w *CupertinoTextFormFieldRow) InitialValue() string {
	if w.o.Controller != nil {
		return w.o.Controller.Text()
	}
	return w.o.InitialValue.OrElse("")
}

// fieldOpts are the parameters of the text field inside the row.
func (w *CupertinoTextFormFieldRow) fieldOpts(controller *widgets.TextEditingController, onChanged func(string)) CupertinoTextFieldOpts {
	o := w.o
	return CupertinoTextFieldOpts{
		Controller: controller, FocusNode: o.FocusNode, Decoration: o.Decoration,
		KeyboardType: o.KeyboardType, TextInputAction: o.TextInputAction, Style: o.Style, StrutStyle: o.StrutStyle,
		TextAlign: o.TextAlign, TextAlignVertical: o.TextAlignVertical, TextCapitalization: o.TextCapitalization,
		TextDirection: o.TextDirection, Autofocus: o.Autofocus, ReadOnly: o.ReadOnly, ShowCursor: o.ShowCursor,
		ObscuringCharacter: o.ObscuringCharacter, ObscureText: o.ObscureText, Autocorrect: o.Autocorrect,
		SmartDashesType: o.SmartDashesType, SmartQuotesType: o.SmartQuotesType, DisableSuggestions: o.DisableSuggestions,
		MaxLines: o.MaxLines, MinLines: o.MinLines, Expands: o.Expands, MaxLength: o.MaxLength,
		OnChanged: onChanged, OnTap: o.OnTap, OnEditingComplete: o.OnEditingComplete, OnSubmitted: o.OnFieldSubmitted,
		InputFormatters: o.InputFormatters, Enabled: foundation.Some(o.Enabled.OrElse(true)),
		CursorWidth: o.CursorWidth, CursorHeight: o.CursorHeight, CursorColor: o.CursorColor,
		ScrollPadding: o.ScrollPadding, ScrollPhysics: o.ScrollPhysics, KeyboardAppearance: o.KeyboardAppearance,
		EnableInteractiveSelection: o.EnableInteractiveSelection, SelectionControls: o.SelectionControls,
		AutofillHints: o.AutofillHints, Placeholder: o.Placeholder, PlaceholderStyle: o.PlaceholderStyle,
		ContextMenuBuilder: o.ContextMenuBuilder, NoContextMenu: o.NoContextMenu,
		SpellCheckConfiguration: o.SpellCheckConfiguration, SelectionHeightStyle: o.SelectionHeightStyle,
		SelectionWidthStyle: o.SelectionWidthStyle,
	}
}

// CreateState creates the state.
//
// Flutter: cupertino/text_form_field_row.dart CupertinoTextFormFieldRow.createState
func (w *CupertinoTextFormFieldRow) CreateState() widgets.State { return &textFormFieldRowState{} }

// textFormFieldRowState keeps the controller of the text and the form field in sync.
//
// Flutter: cupertino/text_form_field_row.dart _CupertinoTextFormFieldRowState
type textFormFieldRowState struct {
	widgets.StateOf[*CupertinoTextFormFieldRow]
	controller   *widgets.TextEditingController
	field        *widgets.FormFieldState[string]
	initialValue string
	listeningTo  *widgets.TextEditingController
	handle       foundation.ListenerHandle
}

func (s *textFormFieldRowState) effectiveController() *widgets.TextEditingController {
	if c := s.Widget().o.Controller; c != nil {
		return c
	}
	return s.controller
}

// InitState creates the local controller, or listens to the given one.
//
// Flutter: cupertino/text_form_field_row.dart _CupertinoTextFormFieldRowState.initState
func (s *textFormFieldRowState) InitState() {
	w := s.Widget()
	if w.o.Controller == nil {
		s.controller = widgets.NewTextEditingController(w.o.InitialValue.OrElse(""))
	} else {
		s.listenTo(w.o.Controller)
	}
	s.initialValue = w.InitialValue()
}

func (s *textFormFieldRowState) listenTo(c *widgets.TextEditingController) {
	s.listeningTo = c
	s.handle = c.AddListener(s.handleControllerChanged)
}

func (s *textFormFieldRowState) stopListening() {
	if s.listeningTo != nil {
		s.listeningTo.RemoveListener(s.handle)
		s.listeningTo = nil
	}
}

// DidUpdateWidget follows the controller of the new widget.
//
// Flutter: cupertino/text_form_field_row.dart _CupertinoTextFormFieldRowState.didUpdateWidget
func (s *textFormFieldRowState) DidUpdateWidget(old *CupertinoTextFormFieldRow) {
	w := s.Widget()
	if w.o.Controller == old.o.Controller {
		return
	}
	s.stopListening()
	if w.o.Controller != nil {
		s.listenTo(w.o.Controller)
	}
	if old.o.Controller != nil && w.o.Controller == nil {
		s.controller = widgets.NewTextEditingControllerFromValue(old.o.Controller.Value())
	}
	if w.o.Controller != nil {
		if s.field != nil {
			s.field.SetValue(foundation.Some(w.o.Controller.Text()))
		}
		if old.o.Controller == nil && s.controller != nil {
			s.controller.Dispose()
			s.controller = nil
		}
	}
}

// Dispose stops listening and releases the local controller.
//
// Flutter: cupertino/text_form_field_row.dart _CupertinoTextFormFieldRowState.dispose
func (s *textFormFieldRowState) Dispose() {
	s.stopListening()
	if s.controller != nil {
		s.controller.Dispose()
	}
}

func (s *textFormFieldRowState) handleDidChange(value foundation.Opt[string]) {
	c := s.effectiveController()
	if text := value.OrElse(""); c.Text() != text {
		c.SetValue(services.NewTextEditingValue(services.TextEditingValueOpts{Text: text}))
	}
}

func (s *textFormFieldRowState) handleReset() {
	c := s.effectiveController()
	c.SetValue(services.NewTextEditingValue(services.TextEditingValueOpts{Text: s.initialValue}))
	if s.field != nil {
		s.field.SetValue(foundation.Some(s.initialValue))
	}
	if f := s.Widget().o.OnChanged; f != nil {
		f(c.Text())
	}
}

func (s *textFormFieldRowState) handleControllerChanged() {
	if s.field == nil {
		return
	}
	if text := s.effectiveController().Text(); s.field.Value() != foundation.Some(text) {
		s.field.DidChange(foundation.Some(text))
	}
}

// Build builds the form field.
//
// Flutter: cupertino/text_form_field_row.dart CupertinoTextFormFieldRow
func (s *textFormFieldRowState) Build(widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	return widgets.NewFormField(s.buildField, widgets.FormFieldOpts[string]{
		OnSaved:          w.o.OnSaved,
		OnReset:          s.handleReset,
		OnDidChange:      s.handleDidChange,
		Validator:        w.o.Validator,
		InitialValue:     foundation.Some(w.InitialValue()),
		Disabled:         !w.o.Enabled.OrElse(true),
		AutovalidateMode: w.o.AutovalidateMode,
	})
}

func (s *textFormFieldRowState) buildField(field *widgets.FormFieldState[string]) widgets.Widget {
	s.field = field
	w := s.Widget()
	var errorWidget widgets.Widget
	if errorText, ok := field.ErrorText().Get(); ok {
		errorWidget = widgets.NewText(errorText)
	}
	onChanged := func(value string) {
		field.DidChange(foundation.Some(value))
		if w.o.OnChanged != nil {
			w.o.OnChanged(value)
		}
	}
	return NewCupertinoFormRow(NewCupertinoTextFieldBorderless(w.fieldOpts(s.effectiveController(), onChanged)), CupertinoFormRowOpts{
		Prefix: w.o.Prefix, Padding: w.o.Padding, Error: errorWidget,
	})
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagCupertinoTextFormFieldRowInitialValueWithController = "CupertinoTextFormFieldRowInitialValueWithController"
)
