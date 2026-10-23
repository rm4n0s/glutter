package widgets

import (
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// AutovalidateMode used to configure when a FormField will validate and
// generate its error message.
//
// Flutter: widgets/form.dart AutovalidateMode
type AutovalidateMode int

const (
	// AutovalidateModeUnspecified is the zero value: the default, which is
	// AutovalidateModeDisabled for a Form, and for a FormField.
	AutovalidateModeUnspecified AutovalidateMode = iota
	// AutovalidateModeDisabled is no auto validation, it will only validate
	// its form field when its Validate is called, or when the Form's Validate is
	// called.
	//
	// Flutter: widgets/form.dart AutovalidateMode.disabled
	AutovalidateModeDisabled
	// AutovalidateModeAlways always validates the form field, even before the
	// user has interacted with it.
	//
	// Flutter: widgets/form.dart AutovalidateMode.always
	AutovalidateModeAlways
	// AutovalidateModeOnUserInteraction validates the form field only after the
	// field has been interacted with by the user.
	//
	// Flutter: widgets/form.dart AutovalidateMode.onUserInteraction
	AutovalidateModeOnUserInteraction
	// AutovalidateModeOnUnfocus validates the form field when the field loses
	// focus.
	//
	// Flutter: widgets/form.dart AutovalidateMode.onUnfocus
	AutovalidateModeOnUnfocus
	// AutovalidateModeOnUserInteractionIfError validates the form field after
	// the user has interacted with it, and only while it has an error.
	//
	// Flutter: widgets/form.dart AutovalidateMode.onUserInteractionIfError
	AutovalidateModeOnUserInteractionIfError
)

// effective is the mode with the default resolved.
func (m AutovalidateMode) effective() AutovalidateMode {
	if m == AutovalidateModeUnspecified {
		return AutovalidateModeDisabled
	}
	return m
}

// Name is the name of the mode.
//
// Flutter: widgets/form.dart AutovalidateMode.name
func (m AutovalidateMode) Name() string {
	switch m {
	case AutovalidateModeDisabled:
		return "disabled"
	case AutovalidateModeAlways:
		return "always"
	case AutovalidateModeOnUserInteraction:
		return "onUserInteraction"
	case AutovalidateModeOnUnfocus:
		return "onUnfocus"
	case AutovalidateModeOnUserInteractionIfError:
		return "onUserInteractionIfError"
	}
	return "unspecified"
}

// String is "AutovalidateMode." and the name.
//
// Flutter: widgets/form.dart AutovalidateMode.toString
func (m AutovalidateMode) String() string { return "AutovalidateMode." + m.Name() }

// FormOpts are the optional parameters of NewForm.
//
// Deviation: the deprecated OnWillPop and OnPopInvoked are not ported; use
// CanPop and OnPopInvokedWithResult (SPEC §12.10).
//
// Flutter: widgets/form.dart Form.new
type FormOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// CanPop enables the form to veto attempts by the user to dismiss the route
	// that contains the form; when set, the form has a PopScope.
	CanPop foundation.Opt[bool]
	// OnPopInvokedWithResult is called after a route pop was handled.
	OnPopInvokedWithResult PopInvokedWithResultCallback
	// OnChanged is called when one of the form fields changes.
	OnChanged foundation.VoidCallback
	// AutovalidateMode used to enable/disable auto validation and capture the
	// state of the form fields; the default is AutovalidateModeDisabled.
	AutovalidateMode AutovalidateMode
}

// Form is an optional container for grouping together multiple form field
// widgets (e.g. TextField widgets).
//
// Each individual form field should be wrapped in a FormField widget, with the
// Form widget as a common ancestor of all of those. Call methods on FormState to
// save, reset, or validate each FormField that is a descendant of this Form. To
// obtain the FormState, you may use FormOf with a context whose ancestor is the
// Form, or pass a GlobalKey to the Form constructor and call
// GlobalKey.CurrentState.
//
// Flutter: widgets/form.dart Form
type Form struct {
	StatefulWidget
	child Widget
	o     FormOpts
}

// NewForm creates a container for form fields.
//
// Flutter: widgets/form.dart Form.new
func NewForm(child Widget, opts ...FormOpts) *Form {
	o := firstOpts(opts)
	o.AutovalidateMode = o.AutovalidateMode.effective()
	return &Form{StatefulWidget: NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), child: child, o: o}
}

// Child is the widget below this one in the tree.
//
// Flutter: widgets/form.dart Form.child
func (w *Form) Child() Widget { return w.child }

// AutovalidateMode is when the form fields validate.
//
// Flutter: widgets/form.dart Form.autovalidateMode
func (w *Form) AutovalidateMode() AutovalidateMode { return w.o.AutovalidateMode }

// FormMaybeOf returns the closest FormState which encloses the given context,
// or nil if there is none. Calling it creates a dependency on the closest form.
//
// Flutter: widgets/form.dart Form.maybeOf
func FormMaybeOf(bc BuildContext) *FormState {
	if scope, ok := DependOn[*formScope](bc); ok {
		return scope.formState
	}
	return nil
}

// FormOf returns the closest FormState which encloses the given context. It
// panics with the tag FormNotFound when there is none.
//
// Flutter: widgets/form.dart Form.of
func FormOf(bc BuildContext) *FormState {
	f := FormMaybeOf(bc)
	if f == nil {
		panic(errors.New(tagFormNotFound, "FormOf was called with a context that does not contain a Form widget"))
	}
	return f
}

// CreateState creates the state.
//
// Flutter: widgets/form.dart Form.createState
func (w *Form) CreateState() State { return &FormState{} }

// FormFieldStateBase is what a FormState needs of a FormFieldState of any type.
//
// Flutter: widgets/form.dart FormFieldState
type FormFieldStateBase interface {
	// Save calls the OnSaved of the field.
	Save()
	// Reset resets the field to its initial value.
	Reset()
	// Validate validates the field and says whether it is valid.
	Validate() bool
	// ErrorText is the current validation error, if any.
	ErrorText() foundation.Opt[string]
	// HasError is whether the field has an error.
	HasError() bool
	// HasInteractedByUser is whether the user has interacted with the field.
	HasInteractedByUser() bool
	clearErrorInternal()
	focusNodeOfField() *FocusNode
}

// FormState maintains the current state of the form. Typically obtained by
// FormOf.
//
// Flutter: widgets/form.dart FormState
type FormState struct {
	StateOf[*Form]
	generation          int
	hasInteractedByUser bool
	fields              []FormFieldStateBase
}

// Fields are the fields of the form, in the order they registered.
//
// Flutter: widgets/form.dart FormState.fields
func (s *FormState) Fields() []FormFieldStateBase { return s.fields }

func (s *FormState) fieldDidChange() {
	if f := s.Widget().o.OnChanged; f != nil {
		f()
	}
	s.hasInteractedByUser = false
	for _, f := range s.fields {
		if f.HasInteractedByUser() {
			s.hasInteractedByUser = true
			break
		}
	}
	s.forceRebuild()
}

func (s *FormState) forceRebuild() { s.SetState(func() { s.generation++ }) }

func (s *FormState) register(field FormFieldStateBase) {
	for _, f := range s.fields {
		if f == field {
			return
		}
	}
	s.fields = append(s.fields, field)
}

func (s *FormState) unregister(field FormFieldStateBase) {
	for i, f := range s.fields {
		if f == field {
			s.fields = append(s.fields[:i], s.fields[i+1:]...)
			return
		}
	}
}

// Build builds the form scope and validates the fields as the mode says.
//
// Flutter: widgets/form.dart FormState.build
func (s *FormState) Build(bc BuildContext) Widget {
	w := s.Widget()
	hasError := false
	for _, f := range s.fields {
		if f.HasError() {
			hasError = true
			break
		}
	}
	switch w.o.AutovalidateMode {
	case AutovalidateModeAlways:
		s.validate(nil)
	case AutovalidateModeOnUserInteraction:
		if s.hasInteractedByUser {
			s.validate(nil)
		}
	case AutovalidateModeOnUserInteractionIfError:
		if s.hasInteractedByUser && hasError {
			s.validate(nil)
		}
	}
	var form Widget = &formScope{InheritedWidget: NewInheritedWidgetBase(w.child), formState: s, generation: s.generation}
	if w.o.CanPop.IsSome() || w.o.OnPopInvokedWithResult != nil {
		form = NewPopScope(form, PopScopeOpts{CanPop: foundation.Some(w.o.CanPop.OrElse(true)), OnPopInvokedWithResult: w.o.OnPopInvokedWithResult})
	}
	return NewSemantics(form, SemanticsOpts{Container: true, ExplicitChildNodes: true, Role: foundation.Some(ui.SemanticsRoleForm)})
}

// Save saves every FormField that is a descendant of this Form.
//
// Flutter: widgets/form.dart FormState.save
func (s *FormState) Save() {
	for _, f := range append([]FormFieldStateBase(nil), s.fields...) {
		f.Save()
	}
}

// Reset resets every FormField that is a descendant of this Form back to its
// FormField initial value.
//
// Flutter: widgets/form.dart FormState.reset
func (s *FormState) Reset() {
	for _, f := range append([]FormFieldStateBase(nil), s.fields...) {
		f.Reset()
	}
	s.hasInteractedByUser = false
	s.fieldDidChange()
}

// ClearError clears the error of every FormField that is a descendant of this
// Form without validating.
//
// Flutter: widgets/form.dart FormState.clearError
func (s *FormState) ClearError() {
	for _, f := range s.fields {
		f.clearErrorInternal()
	}
	s.fieldDidChange()
}

// Validate validates every FormField that is a descendant of this Form, and
// returns true if there are no errors.
//
// Flutter: widgets/form.dart FormState.validate
func (s *FormState) Validate() bool {
	s.hasInteractedByUser = true
	s.forceRebuild()
	return s.validate(nil)
}

// ValidateGranularly validates every FormField that is a descendant of this
// Form, and returns the fields that are invalid.
//
// Flutter: widgets/form.dart FormState.validateGranularly
func (s *FormState) ValidateGranularly() []FormFieldStateBase {
	var invalid []FormFieldStateBase
	s.hasInteractedByUser = true
	s.forceRebuild()
	s.validate(&invalid)
	return invalid
}

// validate validates the fields. Deviation: Flutter announces the first error
// message to the user with the semantics service; the announcements are not
// ported yet (SPEC §12.10).
//
// Flutter: widgets/form.dart FormState._validate
func (s *FormState) validate(invalidFields *[]FormFieldStateBase) bool {
	hasError := false
	for _, f := range append([]FormFieldStateBase(nil), s.fields...) {
		// The field validates unless the mode is validating on unfocus and the
		// field has the focus; but Flutter's condition is always true.
		isFieldValid := f.Validate()
		hasError = hasError || !isFieldValid
		if invalidFields != nil && !isFieldValid {
			*invalidFields = append(*invalidFields, f)
		}
	}
	return !hasError
}

// formScope is Flutter's _FormScope.
//
// Flutter: widgets/form.dart _FormScope
type formScope struct {
	InheritedWidget
	formState  *FormState
	generation int
}

func (w *formScope) UpdateShouldNotify(old Widget) bool {
	return w.generation != old.(*formScope).generation
}

// FormFieldValidator is the signature of a function that validates the value of
// a form field and returns an error text, or none if the value is valid.
//
// Flutter: widgets/form.dart FormFieldValidator
type FormFieldValidator[T any] func(value foundation.Opt[T]) foundation.Opt[string]

// FormFieldErrorBuilder is the signature of a function that builds the widget
// that shows the error of a form field.
//
// Flutter: widgets/form.dart FormFieldErrorBuilder
type FormFieldErrorBuilder func(bc BuildContext, errorText string) Widget

// FormFieldSetter is the signature of a function that is called with the value
// of a form field when the form is saved.
//
// Flutter: widgets/form.dart FormFieldSetter
type FormFieldSetter[T any] func(newValue foundation.Opt[T])

// FormFieldBuilder is the signature of a function that builds the widget of a
// form field.
//
// Flutter: widgets/form.dart FormFieldBuilder
type FormFieldBuilder[T any] func(field *FormFieldState[T]) Widget

// FormFieldOpts are the optional parameters of NewFormField.
//
// Deviation: the restoration of the state (RestorationID) is not ported
// (SPEC §12.10).
//
// Flutter: widgets/form.dart FormField.new
type FormFieldOpts[T any] struct {
	// Key is the key of the widget.
	Key foundation.Key
	// OnSaved is an optional method to call with the final value when the form
	// is saved via FormState.Save.
	OnSaved FormFieldSetter[T]
	// OnReset is an optional method to call when the form field is reset via
	// FormFieldState.Reset.
	OnReset foundation.VoidCallback
	// OnDidChange is called at the end of FormFieldState.DidChange with the new
	// value.
	//
	// Deviation: a subclass of FormFieldState overrides didChange in Flutter, as
	// Material's TextFormField does to update its controller; Go state types
	// cannot be extended, so this callback stands in for the override (SPEC
	// §12.10).
	OnDidChange func(value foundation.Opt[T])
	// ForceErrorText forces an error to be shown, and skips the validator.
	ForceErrorText foundation.Opt[string]
	// Validator is an optional method that validates an input; it returns the
	// error text, or none if the input is valid.
	Validator FormFieldValidator[T]
	// ErrorBuilder builds the widget that shows the error.
	ErrorBuilder FormFieldErrorBuilder
	// InitialValue is the initial value of the form field.
	InitialValue foundation.Opt[T]
	// Disabled makes the field not validate on its own (Flutter's enabled is
	// true by default).
	Disabled bool
	// AutovalidateMode is when the field validates; the default is disabled.
	AutovalidateMode AutovalidateMode
}

// FormField is a single form field.
//
// This widget maintains the current state of the form field, so that updates
// and validation errors are visually reflected in the UI.
//
// When used inside a Form, you can use methods on FormState to query or
// manipulate the form data as a whole.
//
// Flutter: widgets/form.dart FormField
type FormField[T any] struct {
	StatefulWidget
	builder FormFieldBuilder[T]
	o       FormFieldOpts[T]
}

// NewFormField creates a single form field. The builder creates the widget of
// the field; it gets the state to read the value and the error, and to call
// DidChange.
//
// Flutter: widgets/form.dart FormField.new
func NewFormField[T any](builder FormFieldBuilder[T], opts ...FormFieldOpts[T]) *FormField[T] {
	o := firstOpts(opts)
	o.AutovalidateMode = o.AutovalidateMode.effective()
	return &FormField[T]{StatefulWidget: NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), builder: builder, o: o}
}

// ErrorBuilder builds the widget that shows the error, or nil.
//
// Flutter: widgets/form.dart FormField.errorBuilder
func (w *FormField[T]) ErrorBuilder() FormFieldErrorBuilder { return w.o.ErrorBuilder }

// Enabled is whether the field validates on its own.
//
// Flutter: widgets/form.dart FormField.enabled
func (w *FormField[T]) Enabled() bool { return !w.o.Disabled }

// AutovalidateMode is when the field validates.
//
// Flutter: widgets/form.dart FormField.autovalidateMode
func (w *FormField[T]) AutovalidateMode() AutovalidateMode { return w.o.AutovalidateMode }

// InitialValue is the initial value of the form field.
//
// Flutter: widgets/form.dart FormField.initialValue
func (w *FormField[T]) InitialValue() foundation.Opt[T] { return w.o.InitialValue }

// CreateState creates the state.
//
// Flutter: widgets/form.dart FormField.createState
func (w *FormField[T]) CreateState() State { return &FormFieldState[T]{} }

// FormFieldState is the current state of a FormField. Passed to the
// FormFieldBuilder method for use in constructing the form field's widget.
//
// Flutter: widgets/form.dart FormFieldState
type FormFieldState[T any] struct {
	StateOf[*FormField[T]]
	value               foundation.Opt[T]
	errorText           foundation.Opt[string]
	hasInteractedByUser bool
	focusNode           *FocusNode
}

// Value is the current value of the form field.
//
// Flutter: widgets/form.dart FormFieldState.value
func (s *FormFieldState[T]) Value() foundation.Opt[T] { return s.value }

// ErrorText is the current validation error returned by the validator, or none.
//
// Flutter: widgets/form.dart FormFieldState.errorText
func (s *FormFieldState[T]) ErrorText() foundation.Opt[string] { return s.errorText }

// HasError is true if the validator returned a validation error.
//
// Flutter: widgets/form.dart FormFieldState.hasError
func (s *FormFieldState[T]) HasError() bool { return s.errorText.IsSome() }

// HasInteractedByUser is true if this field has been interacted with by the
// user: DidChange was called, and the field has not been reset or its error
// cleared since.
//
// Flutter: widgets/form.dart FormFieldState.hasInteractedByUser
func (s *FormFieldState[T]) HasInteractedByUser() bool { return s.hasInteractedByUser }

// IsValid is true if the current value is valid, without setting the error.
//
// Flutter: widgets/form.dart FormFieldState.isValid
func (s *FormFieldState[T]) IsValid() bool {
	w := s.Widget()
	return w.o.ForceErrorText.IsNone() && (w.o.Validator == nil || w.o.Validator(s.value).IsNone())
}

func (s *FormFieldState[T]) focusNodeOfField() *FocusNode { return s.focusNode }

// Save calls the OnSaved of the field with the current value.
//
// Flutter: widgets/form.dart FormFieldState.save
func (s *FormFieldState[T]) Save() {
	if f := s.Widget().o.OnSaved; f != nil {
		f(s.value)
	}
}

// Reset resets the field to its initial value.
//
// Flutter: widgets/form.dart FormFieldState.reset
func (s *FormFieldState[T]) Reset() {
	s.SetState(func() {
		s.value = s.Widget().o.InitialValue
		s.clearErrorInternal()
	})
	if f := s.Widget().o.OnReset; f != nil {
		f()
	}
	if form := FormMaybeOf(s.Context()); form != nil {
		form.fieldDidChange()
	}
}

// ClearError clears the error without validating.
//
// Flutter: widgets/form.dart FormFieldState.clearError
func (s *FormFieldState[T]) ClearError() {
	s.SetState(s.clearErrorInternal)
	if form := FormMaybeOf(s.Context()); form != nil {
		form.fieldDidChange()
	}
}

// Validate calls the validator with the current value and sets the error, and
// returns true if there is no error.
//
// Flutter: widgets/form.dart FormFieldState.validate
func (s *FormFieldState[T]) Validate() bool {
	s.SetState(s.validateInternal)
	return !s.HasError()
}

func (s *FormFieldState[T]) clearErrorInternal() {
	s.errorText = foundation.None[string]()
	s.hasInteractedByUser = false
}

func (s *FormFieldState[T]) validateInternal() {
	w := s.Widget()
	if w.o.ForceErrorText.IsSome() {
		s.errorText = w.o.ForceErrorText
		// Skip validating if error is forced.
		return
	}
	if w.o.Validator != nil {
		s.errorText = w.o.Validator(s.value)
	} else {
		s.errorText = foundation.None[string]()
	}
}

// DidChange updates this field's state to the new value. Useful for
// responding to child widget changes, e.g. Slider's onChanged argument.
//
// Flutter: widgets/form.dart FormFieldState.didChange
func (s *FormFieldState[T]) DidChange(value foundation.Opt[T]) {
	s.SetState(func() {
		s.value = value
		s.hasInteractedByUser = true
	})
	if form := FormMaybeOf(s.Context()); form != nil {
		form.fieldDidChange()
	}
	if f := s.Widget().o.OnDidChange; f != nil {
		f(value)
	}
}

// SetValue sets the value associated with this form field, without telling the
// form or rebuilding. Use DidChange for the effects.
//
// Flutter: widgets/form.dart FormFieldState.setValue
func (s *FormFieldState[T]) SetValue(value foundation.Opt[T]) { s.value = value }

// InitState sets the initial value and the forced error.
//
// Flutter: widgets/form.dart FormFieldState.initState
func (s *FormFieldState[T]) InitState() {
	w := s.Widget()
	s.value = w.o.InitialValue
	s.errorText = w.o.ForceErrorText
	s.focusNode = NewFocusNode()
}

// DidUpdateWidget follows the forced error text.
//
// Flutter: widgets/form.dart FormFieldState.didUpdateWidget
func (s *FormFieldState[T]) DidUpdateWidget(old *FormField[T]) {
	if s.Widget().o.ForceErrorText != old.o.ForceErrorText {
		s.errorText = s.Widget().o.ForceErrorText
	}
}

// Deactivate unregisters the field from the form.
//
// Flutter: widgets/form.dart FormFieldState.deactivate
func (s *FormFieldState[T]) Deactivate() {
	if form := FormMaybeOf(s.Context()); form != nil {
		form.unregister(s)
	}
}

// DidChangeDependencies validates after the frame when the form always validates.
//
// Flutter: widgets/form.dart FormFieldState.didChangeDependencies
func (s *FormFieldState[T]) DidChangeDependencies() {
	if form := FormMaybeOf(s.Context()); form != nil && form.Widget().o.AutovalidateMode == AutovalidateModeAlways {
		s.binding().RunPostFrame(func(time.Duration) {
			// If the form is already validated, don't validate again.
			if s.Mounted() && !s.Widget().o.Disabled && !s.HasError() && !s.IsValid() {
				s.Validate()
			}
		})
	}
}

func (s *FormFieldState[T]) binding() *WidgetsBinding { return WidgetsBindingOf(s.Context()) }

// Dispose releases the focus node.
//
// Flutter: widgets/form.dart FormFieldState.dispose
func (s *FormFieldState[T]) Dispose() { s.focusNode.Dispose() }

// Build builds the field, validating as the mode says.
//
// Flutter: widgets/form.dart FormFieldState.build
func (s *FormFieldState[T]) Build(bc BuildContext) Widget {
	w := s.Widget()
	if !w.o.Disabled {
		switch w.o.AutovalidateMode {
		case AutovalidateModeAlways:
			s.validateInternal()
		case AutovalidateModeOnUserInteraction:
			if s.hasInteractedByUser {
				s.validateInternal()
			}
		case AutovalidateModeOnUserInteractionIfError:
			if s.hasInteractedByUser && s.HasError() {
				s.validateInternal()
			}
		}
	}
	form := FormMaybeOf(bc)
	if form != nil {
		form.register(s)
	}
	result := ui.SemanticsValidationResultValid
	if s.HasError() {
		result = ui.SemanticsValidationResultInvalid
	}
	child := Widget(NewSemantics(w.builder(s), SemanticsOpts{ValidationResult: result}))
	if (form != nil && form.Widget().o.AutovalidateMode == AutovalidateModeOnUnfocus && w.o.AutovalidateMode != AutovalidateModeAlways) ||
		w.o.AutovalidateMode == AutovalidateModeOnUnfocus {
		return NewFocus(child, FocusOpts{
			CanRequestFocus: foundation.Some(false), SkipTraversal: foundation.Some(true), FocusNode: s.focusNode,
			OnFocusChange: func(hasFocus bool) {
				if !hasFocus {
					s.SetState(s.validateInternal)
				}
			},
		})
	}
	return child
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagFormNotFound = "FormNotFound"
)
