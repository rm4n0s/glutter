// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package services

import (
	"fmt"
	"slices"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/opts"
	"github.com/rm4n0s/glutter/ui"
)

// SmartDashesType is whether the input method replaces two dashes with a long
// dash. The zero value, SmartDashesTypeUnspecified, is disabled for a field with
// obscured text and enabled for any other.
//
// Flutter: services/text_input.dart SmartDashesType
type SmartDashesType int

// The values of SmartDashesType follow Flutter's declaration order.
const (
	// SmartDashesTypeUnspecified is the zero value: the default for the field.
	//
	// Flutter: no counterpart
	SmartDashesTypeUnspecified SmartDashesType = iota
	// SmartDashesTypeDisabled leaves the dashes as they are.
	//
	// Flutter: services/text_input.dart SmartDashesType.disabled
	SmartDashesTypeDisabled
	// SmartDashesTypeEnabled replaces two dashes with a long dash.
	//
	// Flutter: services/text_input.dart SmartDashesType.enabled
	SmartDashesTypeEnabled
)

// SmartQuotesType is whether the input method replaces straight quotes with
// curly quotes. The zero value, SmartQuotesTypeUnspecified, is disabled for a
// field with obscured text and enabled for any other.
//
// Flutter: services/text_input.dart SmartQuotesType
type SmartQuotesType int

// The values of SmartQuotesType follow Flutter's declaration order.
const (
	// SmartQuotesTypeUnspecified is the zero value: the default for the field.
	//
	// Flutter: no counterpart
	SmartQuotesTypeUnspecified SmartQuotesType = iota
	// SmartQuotesTypeDisabled leaves the quotes as they are.
	//
	// Flutter: services/text_input.dart SmartQuotesType.disabled
	SmartQuotesTypeDisabled
	// SmartQuotesTypeEnabled replaces straight quotes with curly quotes.
	//
	// Flutter: services/text_input.dart SmartQuotesType.enabled
	SmartQuotesTypeEnabled
)

// TextInputType is the type of information a text field edits, so the platform
// can offer a fitting keyboard. The zero value is TextInputTypeText.
//
// Flutter: services/text_input.dart TextInputType
type TextInputType struct {
	index   int
	signed  foundation.Opt[bool]
	decimal foundation.Opt[bool]
}

// The kinds of TextInputType.
var (
	// TextInputTypeText is plain text.
	//
	// Flutter: services/text_input.dart TextInputType.text
	TextInputTypeText = TextInputType{index: 0}
	// TextInputTypeMultiline is multiline text, where the Enter key adds a line.
	//
	// Flutter: services/text_input.dart TextInputType.multiline
	TextInputTypeMultiline = TextInputType{index: 1}
	// TextInputTypeNumber is an unsigned integer.
	//
	// Flutter: services/text_input.dart TextInputType.number
	TextInputTypeNumber = NewTextInputTypeNumberWithOptions()
	// TextInputTypePhone is a phone number.
	//
	// Flutter: services/text_input.dart TextInputType.phone
	TextInputTypePhone = TextInputType{index: 3}
	// TextInputTypeDatetime is a date and time.
	//
	// Flutter: services/text_input.dart TextInputType.datetime
	TextInputTypeDatetime = TextInputType{index: 4}
	// TextInputTypeEmailAddress is an email address.
	//
	// Flutter: services/text_input.dart TextInputType.emailAddress
	TextInputTypeEmailAddress = TextInputType{index: 5}
	// TextInputTypeURL is a URL.
	//
	// Flutter: services/text_input.dart TextInputType.url
	TextInputTypeURL = TextInputType{index: 6}
	// TextInputTypeVisiblePassword is a password that is shown.
	//
	// Flutter: services/text_input.dart TextInputType.visiblePassword
	TextInputTypeVisiblePassword = TextInputType{index: 7}
	// TextInputTypeName is the name of a person.
	//
	// Flutter: services/text_input.dart TextInputType.name
	TextInputTypeName = TextInputType{index: 8}
	// TextInputTypeStreetAddress is a postal address.
	//
	// Flutter: services/text_input.dart TextInputType.streetAddress
	TextInputTypeStreetAddress = TextInputType{index: 9}
	// TextInputTypeNone is no keyboard: the app provides its own input.
	//
	// Flutter: services/text_input.dart TextInputType.none
	TextInputTypeNone = TextInputType{index: 10}
	// TextInputTypeWebSearch is a search term for the web.
	//
	// Flutter: services/text_input.dart TextInputType.webSearch
	TextInputTypeWebSearch = TextInputType{index: 11}
	// TextInputTypeTwitter is a post on a microblogging service.
	//
	// Flutter: services/text_input.dart TextInputType.twitter
	TextInputTypeTwitter = TextInputType{index: 12}
)

var textInputTypeNames = [...]string{
	"text", "multiline", "number", "phone", "datetime", "emailAddress", "url",
	"visiblePassword", "name", "address", "none", "webSearch", "twitter",
}

// TextInputTypeNumberOpts are the optional parameters of
// NewTextInputTypeNumberWithOptions.
//
// Flutter: services/text_input.dart TextInputType.numberWithOptions
type TextInputTypeNumberOpts struct {
	// Signed is whether the number can have a sign.
	Signed bool
	// Decimal is whether the number can have a decimal separator.
	Decimal bool
}

// NewTextInputTypeNumberWithOptions creates the type of a number, signed and
// with a decimal separator or not.
//
// Flutter: services/text_input.dart TextInputType.numberWithOptions
func NewTextInputTypeNumberWithOptions(o ...TextInputTypeNumberOpts) TextInputType {
	op := opts.First(o)
	return TextInputType{index: 2, signed: foundation.Some(op.Signed), decimal: foundation.Some(op.Decimal)}
}

// TextInputTypeValues returns all the kinds of TextInputType, in the order of
// Flutter's list.
//
// Flutter: services/text_input.dart TextInputType.values
func TextInputTypeValues() []TextInputType {
	return []TextInputType{
		TextInputTypeText, TextInputTypeMultiline, TextInputTypeNumber, TextInputTypePhone,
		TextInputTypeDatetime, TextInputTypeEmailAddress, TextInputTypeURL, TextInputTypeVisiblePassword,
		TextInputTypeName, TextInputTypeStreetAddress, TextInputTypeNone, TextInputTypeWebSearch,
		TextInputTypeTwitter,
	}
}

// Index is the position of the type in TextInputTypeValues.
//
// Flutter: services/text_input.dart TextInputType.index
func (t TextInputType) Index() int { return t.index }

// Signed is whether a number can have a sign; it is none for the other types.
//
// Flutter: services/text_input.dart TextInputType.signed
func (t TextInputType) Signed() foundation.Opt[bool] { return t.signed }

// Decimal is whether a number can have a decimal separator; it is none for the
// other types.
//
// Flutter: services/text_input.dart TextInputType.decimal
func (t TextInputType) Decimal() foundation.Opt[bool] { return t.decimal }

// String returns what Dart prints, such as "TextInputType(name: TextInputType.text, signed: null, decimal: null)".
func (t TextInputType) String() string {
	return fmt.Sprintf("TextInputType(name: TextInputType.%s, signed: %s, decimal: %s)",
		textInputTypeNames[t.index], optBool(t.signed), optBool(t.decimal))
}

func optBool(o foundation.Opt[bool]) string {
	if v, ok := o.Get(); ok {
		return fmt.Sprint(v)
	}
	return "null"
}

// TextInputAction is the action the Enter key (or its onscreen counterpart)
// does in a text field. The zero value, TextInputActionUnspecified, means the
// default of the field: TextInputActionDone.
//
// Deviation: Flutter's own value `unspecified`, which lets the platform choose
// the action, is TextInputActionPlatformDefault here, because the zero value of
// an enum is the "unspecified" of Go's options (SPEC §12.10).
//
// Flutter: services/text_input.dart TextInputAction
type TextInputAction int

// The values of TextInputAction follow Flutter's declaration order.
const (
	// TextInputActionUnspecified is the zero value: the default (done).
	//
	// Flutter: no counterpart
	TextInputActionUnspecified TextInputAction = iota
	// TextInputActionNone means there is no action; the Enter key does what it
	// does when it is just a key.
	//
	// Flutter: services/text_input.dart TextInputAction.none
	TextInputActionNone
	// TextInputActionPlatformDefault lets the platform choose the action.
	//
	// Flutter: services/text_input.dart TextInputAction.unspecified
	TextInputActionPlatformDefault
	// TextInputActionDone means the user is done with the input.
	//
	// Flutter: services/text_input.dart TextInputAction.done
	TextInputActionDone
	// TextInputActionGo takes the user to the target of the text.
	//
	// Flutter: services/text_input.dart TextInputAction.go
	TextInputActionGo
	// TextInputActionSearch runs a search for the text.
	//
	// Flutter: services/text_input.dart TextInputAction.search
	TextInputActionSearch
	// TextInputActionSend sends the text.
	//
	// Flutter: services/text_input.dart TextInputAction.send
	TextInputActionSend
	// TextInputActionNext moves the focus to the next field.
	//
	// Flutter: services/text_input.dart TextInputAction.next
	TextInputActionNext
	// TextInputActionPrevious moves the focus to the previous field.
	//
	// Flutter: services/text_input.dart TextInputAction.previous
	TextInputActionPrevious
	// TextInputActionContinueAction continues to the next step.
	//
	// Flutter: services/text_input.dart TextInputAction.continueAction
	TextInputActionContinueAction
	// TextInputActionJoin joins something.
	//
	// Flutter: services/text_input.dart TextInputAction.join
	TextInputActionJoin
	// TextInputActionRoute shows a route to the text.
	//
	// Flutter: services/text_input.dart TextInputAction.route
	TextInputActionRoute
	// TextInputActionEmergencyCall places an emergency call.
	//
	// Flutter: services/text_input.dart TextInputAction.emergencyCall
	TextInputActionEmergencyCall
	// TextInputActionNewline adds a line break.
	//
	// Flutter: services/text_input.dart TextInputAction.newline
	TextInputActionNewline
)

var textInputActionNames = [...]string{
	"TextInputActionUnspecified", "TextInputActionNone", "TextInputActionPlatformDefault", "TextInputActionDone",
	"TextInputActionGo", "TextInputActionSearch", "TextInputActionSend", "TextInputActionNext",
	"TextInputActionPrevious", "TextInputActionContinueAction", "TextInputActionJoin", "TextInputActionRoute",
	"TextInputActionEmergencyCall", "TextInputActionNewline",
}

// String returns the name of the value, such as "TextInputActionDone".
func (a TextInputAction) String() string {
	if a >= 0 && int(a) < len(textInputActionNames) {
		return textInputActionNames[a]
	}
	return fmt.Sprintf("TextInputAction(%d)", int(a))
}

// TextCapitalization is how the input method capitalizes the text. The zero
// value, TextCapitalizationUnspecified, is TextCapitalizationNone.
//
// Flutter: services/text_input.dart TextCapitalization
type TextCapitalization int

// The values of TextCapitalization follow Flutter's declaration order.
const (
	// TextCapitalizationUnspecified is the zero value: none.
	//
	// Flutter: no counterpart
	TextCapitalizationUnspecified TextCapitalization = iota
	// TextCapitalizationWords capitalizes the first letter of every word.
	//
	// Flutter: services/text_input.dart TextCapitalization.words
	TextCapitalizationWords
	// TextCapitalizationSentences capitalizes the first letter of every sentence.
	//
	// Flutter: services/text_input.dart TextCapitalization.sentences
	TextCapitalizationSentences
	// TextCapitalizationCharacters capitalizes every character.
	//
	// Flutter: services/text_input.dart TextCapitalization.characters
	TextCapitalizationCharacters
	// TextCapitalizationNone doesn't capitalize anything.
	//
	// Flutter: services/text_input.dart TextCapitalization.none
	TextCapitalizationNone
)

// TextInputConfiguration describes how the platform input method should work
// for a text field. Make one with NewTextInputConfiguration, which sets the
// defaults of Flutter; a TextInputConfiguration{} literal has false for
// Autocorrect, EnableSuggestions and the other defaults that are true.
//
// Flutter: services/text_input.dart TextInputConfiguration
type TextInputConfiguration struct {
	// ViewID is the view of the text field, or none for the implicit view.
	//
	// Flutter: services/text_input.dart TextInputConfiguration.viewId
	ViewID foundation.Opt[int]
	// InputType is the type of information the field edits.
	//
	// Flutter: services/text_input.dart TextInputConfiguration.inputType
	InputType TextInputType
	// ReadOnly is whether the field only shows text.
	//
	// Flutter: services/text_input.dart TextInputConfiguration.readOnly
	ReadOnly bool
	// ObscureText is whether the text is hidden, as for a password.
	//
	// Flutter: services/text_input.dart TextInputConfiguration.obscureText
	ObscureText bool
	// Autocorrect is whether the input method corrects the text.
	//
	// Flutter: services/text_input.dart TextInputConfiguration.autocorrect
	Autocorrect bool
	// SmartDashesType is whether two dashes become a long dash.
	//
	// Flutter: services/text_input.dart TextInputConfiguration.smartDashesType
	SmartDashesType SmartDashesType
	// SmartQuotesType is whether straight quotes become curly quotes.
	//
	// Flutter: services/text_input.dart TextInputConfiguration.smartQuotesType
	SmartQuotesType SmartQuotesType
	// EnableSuggestions is whether the input method suggests text.
	//
	// Flutter: services/text_input.dart TextInputConfiguration.enableSuggestions
	EnableSuggestions bool
	// EnableInteractiveSelection is whether the user can select text.
	//
	// Flutter: services/text_input.dart TextInputConfiguration.enableInteractiveSelection
	EnableInteractiveSelection bool
	// ActionLabel is the text of the action button of the keyboard, if any.
	//
	// Flutter: services/text_input.dart TextInputConfiguration.actionLabel
	ActionLabel foundation.Opt[string]
	// InputAction is what the Enter key does.
	//
	// Flutter: services/text_input.dart TextInputConfiguration.inputAction
	InputAction TextInputAction
	// KeyboardAppearance is the brightness of the keyboard.
	//
	// Flutter: services/text_input.dart TextInputConfiguration.keyboardAppearance
	KeyboardAppearance ui.Brightness
	// TextCapitalization is how the text is capitalized.
	//
	// Flutter: services/text_input.dart TextInputConfiguration.textCapitalization
	TextCapitalization TextCapitalization
	// AutofillConfiguration is how the field takes part in autofill.
	//
	// Flutter: services/text_input.dart TextInputConfiguration.autofillConfiguration
	AutofillConfiguration AutofillConfiguration
	// EnableIMEPersonalizedLearning is whether the input method may learn from
	// what the user writes.
	//
	// Flutter: services/text_input.dart TextInputConfiguration.enableIMEPersonalizedLearning
	EnableIMEPersonalizedLearning bool
	// AllowedMimeTypes are the MIME types of the content the field accepts from
	// the input method (images, for example).
	//
	// Flutter: services/text_input.dart TextInputConfiguration.allowedMimeTypes
	AllowedMimeTypes []string
	// EnableDeltaModel is whether the platform reports the changes as
	// TextEditingDeltas, to a DeltaTextInputClient, not as whole values.
	//
	// Flutter: services/text_input.dart TextInputConfiguration.enableDeltaModel
	EnableDeltaModel bool
	// HintLocales are the locales the input method should prefer.
	//
	// Flutter: services/text_input.dart TextInputConfiguration.hintLocales
	HintLocales []ui.Locale
	// EnableInlinePrediction is whether the input method may show a prediction
	// inline; none leaves it to the platform.
	//
	// Flutter: services/text_input.dart TextInputConfiguration.enableInlinePrediction
	EnableInlinePrediction foundation.Opt[bool]
}

// TextInputConfigurationOpts are the optional parameters of
// NewTextInputConfiguration.
//
// Flutter: services/text_input.dart TextInputConfiguration.new
type TextInputConfigurationOpts struct {
	ViewID foundation.Opt[int]
	// InputType defaults to TextInputTypeText.
	InputType   TextInputType
	ReadOnly    bool
	ObscureText bool
	// DisableAutocorrect turns the autocorrection off; it is on by default.
	DisableAutocorrect bool
	// SmartDashesType defaults to disabled for an obscured text, enabled for any
	// other.
	SmartDashesType SmartDashesType
	// SmartQuotesType defaults like SmartDashesType.
	SmartQuotesType SmartQuotesType
	// DisableSuggestions turns the suggestions off; they are on by default.
	DisableSuggestions bool
	// DisableInteractiveSelection stops the user from selecting text; they can by
	// default.
	DisableInteractiveSelection bool
	ActionLabel                 foundation.Opt[string]
	// InputAction defaults to TextInputActionDone.
	InputAction TextInputAction
	// KeyboardAppearance defaults to light.
	KeyboardAppearance ui.Brightness
	// TextCapitalization defaults to none.
	TextCapitalization TextCapitalization
	// AutofillConfiguration defaults to AutofillConfigurationDisabled.
	AutofillConfiguration foundation.Opt[AutofillConfiguration]
	// DisableIMEPersonalizedLearning turns the learning of the input method
	// off; it is on by default.
	DisableIMEPersonalizedLearning bool
	AllowedMimeTypes               []string
	EnableDeltaModel               bool
	HintLocales                    []ui.Locale
	EnableInlinePrediction         foundation.Opt[bool]
}

// NewTextInputConfiguration creates a configuration with Flutter's defaults.
//
// Flutter: services/text_input.dart TextInputConfiguration.new
func NewTextInputConfiguration(o ...TextInputConfigurationOpts) TextInputConfiguration {
	op := opts.First(o)
	c := TextInputConfiguration{
		ViewID:                        op.ViewID,
		InputType:                     op.InputType,
		ReadOnly:                      op.ReadOnly,
		ObscureText:                   op.ObscureText,
		Autocorrect:                   !op.DisableAutocorrect,
		SmartDashesType:               op.SmartDashesType,
		SmartQuotesType:               op.SmartQuotesType,
		EnableSuggestions:             !op.DisableSuggestions,
		EnableInteractiveSelection:    !op.DisableInteractiveSelection,
		ActionLabel:                   op.ActionLabel,
		InputAction:                   op.InputAction,
		KeyboardAppearance:            op.KeyboardAppearance,
		TextCapitalization:            op.TextCapitalization,
		AutofillConfiguration:         op.AutofillConfiguration.OrElse(AutofillConfigurationDisabled),
		EnableIMEPersonalizedLearning: !op.DisableIMEPersonalizedLearning,
		AllowedMimeTypes:              op.AllowedMimeTypes,
		EnableDeltaModel:              op.EnableDeltaModel,
		HintLocales:                   op.HintLocales,
		EnableInlinePrediction:        op.EnableInlinePrediction,
	}
	if c.SmartDashesType == SmartDashesTypeUnspecified {
		c.SmartDashesType = SmartDashesTypeEnabled
		if c.ObscureText {
			c.SmartDashesType = SmartDashesTypeDisabled
		}
	}
	if c.SmartQuotesType == SmartQuotesTypeUnspecified {
		c.SmartQuotesType = SmartQuotesTypeEnabled
		if c.ObscureText {
			c.SmartQuotesType = SmartQuotesTypeDisabled
		}
	}
	if c.InputAction == TextInputActionUnspecified {
		c.InputAction = TextInputActionDone
	}
	if c.KeyboardAppearance == ui.BrightnessUnspecified {
		c.KeyboardAppearance = ui.BrightnessLight
	}
	if c.TextCapitalization == TextCapitalizationUnspecified {
		c.TextCapitalization = TextCapitalizationNone
	}
	if c.InputType == (TextInputType{}) {
		c.InputType = TextInputTypeText
	}
	return c
}

// TextInputConfigurationCopy holds the values TextInputConfiguration.CopyWith
// replaces. An unspecified enum keeps the old value.
//
// Flutter: services/text_input.dart TextInputConfiguration.copyWith
type TextInputConfigurationCopy struct {
	ViewID                        foundation.Opt[int]
	InputType                     foundation.Opt[TextInputType]
	ReadOnly                      foundation.Opt[bool]
	ObscureText                   foundation.Opt[bool]
	Autocorrect                   foundation.Opt[bool]
	SmartDashesType               SmartDashesType
	SmartQuotesType               SmartQuotesType
	EnableSuggestions             foundation.Opt[bool]
	EnableInteractiveSelection    foundation.Opt[bool]
	ActionLabel                   foundation.Opt[string]
	InputAction                   TextInputAction
	KeyboardAppearance            ui.Brightness
	TextCapitalization            TextCapitalization
	EnableIMEPersonalizedLearning foundation.Opt[bool]
	AllowedMimeTypes              foundation.Opt[[]string]
	AutofillConfiguration         foundation.Opt[AutofillConfiguration]
	EnableDeltaModel              foundation.Opt[bool]
	HintLocales                   foundation.Opt[[]ui.Locale]
	EnableInlinePrediction        foundation.Opt[bool]
}

// CopyWith returns a copy with the given values replaced.
//
// Flutter: services/text_input.dart TextInputConfiguration.copyWith
func (c TextInputConfiguration) CopyWith(r TextInputConfigurationCopy) TextInputConfiguration {
	out := c
	if r.ViewID.IsSome() {
		out.ViewID = r.ViewID
	}
	out.InputType = r.InputType.OrElse(c.InputType)
	out.ReadOnly = r.ReadOnly.OrElse(c.ReadOnly)
	out.ObscureText = r.ObscureText.OrElse(c.ObscureText)
	out.Autocorrect = r.Autocorrect.OrElse(c.Autocorrect)
	if r.SmartDashesType != SmartDashesTypeUnspecified {
		out.SmartDashesType = r.SmartDashesType
	}
	if r.SmartQuotesType != SmartQuotesTypeUnspecified {
		out.SmartQuotesType = r.SmartQuotesType
	}
	out.EnableSuggestions = r.EnableSuggestions.OrElse(c.EnableSuggestions)
	out.EnableInteractiveSelection = r.EnableInteractiveSelection.OrElse(c.EnableInteractiveSelection)
	if r.ActionLabel.IsSome() {
		out.ActionLabel = r.ActionLabel
	}
	if r.InputAction != TextInputActionUnspecified {
		out.InputAction = r.InputAction
	}
	if r.KeyboardAppearance != ui.BrightnessUnspecified {
		out.KeyboardAppearance = r.KeyboardAppearance
	}
	if r.TextCapitalization != TextCapitalizationUnspecified {
		out.TextCapitalization = r.TextCapitalization
	}
	out.EnableIMEPersonalizedLearning = r.EnableIMEPersonalizedLearning.OrElse(c.EnableIMEPersonalizedLearning)
	out.AllowedMimeTypes = r.AllowedMimeTypes.OrElse(c.AllowedMimeTypes)
	out.AutofillConfiguration = r.AutofillConfiguration.OrElse(c.AutofillConfiguration)
	out.EnableDeltaModel = r.EnableDeltaModel.OrElse(c.EnableDeltaModel)
	out.HintLocales = r.HintLocales.OrElse(c.HintLocales)
	if r.EnableInlinePrediction.IsSome() {
		out.EnableInlinePrediction = r.EnableInlinePrediction
	}
	return out
}

// Equal reports whether two configurations are the same.
//
// Flutter: services/text_input.dart TextInputConfiguration.==
func (c TextInputConfiguration) Equal(o TextInputConfiguration) bool {
	return c.ViewID == o.ViewID && c.InputType == o.InputType && c.ReadOnly == o.ReadOnly &&
		c.ObscureText == o.ObscureText && c.Autocorrect == o.Autocorrect &&
		c.SmartDashesType == o.SmartDashesType && c.SmartQuotesType == o.SmartQuotesType &&
		c.EnableSuggestions == o.EnableSuggestions && c.EnableInteractiveSelection == o.EnableInteractiveSelection &&
		c.ActionLabel == o.ActionLabel && c.InputAction == o.InputAction &&
		c.KeyboardAppearance == o.KeyboardAppearance && c.TextCapitalization == o.TextCapitalization &&
		c.AutofillConfiguration.Equal(o.AutofillConfiguration) &&
		c.EnableIMEPersonalizedLearning == o.EnableIMEPersonalizedLearning &&
		slices.Equal(c.AllowedMimeTypes, o.AllowedMimeTypes) && c.EnableDeltaModel == o.EnableDeltaModel &&
		slices.Equal(c.HintLocales, o.HintLocales) && c.EnableInlinePrediction == o.EnableInlinePrediction
}

// String returns what Dart prints for the configuration.
func (c TextInputConfiguration) String() string {
	var d []string
	if v, ok := c.ViewID.Get(); ok {
		d = append(d, fmt.Sprintf("viewId: %d", v))
	}
	d = append(d,
		"inputType: "+c.InputType.String(),
		fmt.Sprintf("readOnly: %v", c.ReadOnly),
		fmt.Sprintf("obscureText: %v", c.ObscureText),
		fmt.Sprintf("autocorrect: %v", c.Autocorrect),
		fmt.Sprintf("smartDashesType: %d", c.SmartDashesType),
		fmt.Sprintf("smartQuotesType: %d", c.SmartQuotesType),
		fmt.Sprintf("enableSuggestions: %v", c.EnableSuggestions),
		fmt.Sprintf("enableInteractiveSelection: %v", c.EnableInteractiveSelection))
	if v, ok := c.ActionLabel.Get(); ok {
		d = append(d, "actionLabel: "+v)
	}
	d = append(d,
		"inputAction: "+c.InputAction.String(),
		"keyboardAppearance: "+c.KeyboardAppearance.String(),
		fmt.Sprintf("textCapitalization: %d", c.TextCapitalization),
		"autofillConfiguration: "+c.AutofillConfiguration.String(),
		fmt.Sprintf("enableIMEPersonalizedLearning: %v", c.EnableIMEPersonalizedLearning),
		fmt.Sprintf("allowedMimeTypes: %v", c.AllowedMimeTypes),
		fmt.Sprintf("enableDeltaModel: %v", c.EnableDeltaModel))
	if c.HintLocales != nil {
		d = append(d, fmt.Sprintf("hintLocales: %v", c.HintLocales))
	}
	if v, ok := c.EnableInlinePrediction.Get(); ok {
		d = append(d, fmt.Sprintf("enableInlinePrediction: %v", v))
	}
	return "TextInputConfiguration(" + strings.Join(d, ", ") + ")"
}

// FloatingCursorDragState is the state of a floating cursor drag. The zero
// value is FloatingCursorDragStateUnspecified.
//
// Flutter: services/text_input.dart FloatingCursorDragState
type FloatingCursorDragState int

// The values of FloatingCursorDragState follow Flutter's declaration order.
const (
	// FloatingCursorDragStateUnspecified is the zero value.
	//
	// Flutter: no counterpart
	FloatingCursorDragStateUnspecified FloatingCursorDragState = iota
	// FloatingCursorDragStateStart is the start of a drag.
	//
	// Flutter: services/text_input.dart FloatingCursorDragState.Start
	FloatingCursorDragStateStart
	// FloatingCursorDragStateUpdate is a movement of the cursor during a drag.
	//
	// Flutter: services/text_input.dart FloatingCursorDragState.Update
	FloatingCursorDragStateUpdate
	// FloatingCursorDragStateEnd is the end of a drag.
	//
	// Flutter: services/text_input.dart FloatingCursorDragState.End
	FloatingCursorDragStateEnd
)

// FloatingCursorStartLocation is where a floating cursor drag started: a point
// of the editable and the text position there.
//
// Flutter: services/text_input.dart RawFloatingCursorPoint.startLocation
type FloatingCursorStartLocation struct {
	// Offset is the point in the editable.
	Offset ui.Offset
	// Position is the text position at the point.
	Position ui.TextPosition
}

// RawFloatingCursorPoint is a state and a position of the floating cursor that
// some keyboards show.
//
// Flutter: services/text_input.dart RawFloatingCursorPoint
type RawFloatingCursorPoint struct {
	// Offset is the raw position of the floating cursor as determined by the
	// platform; it is set for FloatingCursorDragStateUpdate.
	//
	// Flutter: services/text_input.dart RawFloatingCursorPoint.offset
	Offset foundation.Opt[ui.Offset]
	// StartLocation is where the drag started, if the platform says.
	//
	// Flutter: services/text_input.dart RawFloatingCursorPoint.startLocation
	StartLocation foundation.Opt[FloatingCursorStartLocation]
	// State is the state of the drag.
	//
	// Flutter: services/text_input.dart RawFloatingCursorPoint.state
	State FloatingCursorDragState
}

// RawFloatingCursorPointOpts are the optional parameters of
// NewRawFloatingCursorPoint.
//
// Flutter: services/text_input.dart RawFloatingCursorPoint.new
type RawFloatingCursorPointOpts struct {
	Offset        foundation.Opt[ui.Offset]
	StartLocation foundation.Opt[FloatingCursorStartLocation]
}

// NewRawFloatingCursorPoint creates a point. An update without an offset panics
// in a debug build with the tag RawFloatingCursorPointOffsetMissing.
//
// Flutter: services/text_input.dart RawFloatingCursorPoint.new
func NewRawFloatingCursorPoint(state FloatingCursorDragState, o ...RawFloatingCursorPointOpts) RawFloatingCursorPoint {
	op := opts.First(o)
	if foundation.DebugMode && state == FloatingCursorDragStateUpdate && op.Offset.IsNone() {
		panic(errors.New("RawFloatingCursorPointOffsetMissing", "a floating cursor update needs an offset"))
	}
	return RawFloatingCursorPoint{Offset: op.Offset, StartLocation: op.StartLocation, State: state}
}

// SelectionRect is the rectangle of a character of the text, in the field's
// coordinates, so the platform can place its own selection.
//
// Flutter: services/text_input.dart SelectionRect
type SelectionRect struct {
	// Position is the offset of the character in the text.
	//
	// Flutter: services/text_input.dart SelectionRect.position
	Position int
	// Bounds is the rectangle of the character.
	//
	// Flutter: services/text_input.dart SelectionRect.bounds
	Bounds ui.Rect
	// Direction is the direction of the text at the character; it is left to
	// right unless it is set.
	//
	// Flutter: services/text_input.dart SelectionRect.direction
	Direction ui.TextDirection
}

// NewSelectionRect creates the rectangle of the character at position.
//
// Flutter: services/text_input.dart SelectionRect.new
func NewSelectionRect(position int, bounds ui.Rect, direction ...ui.TextDirection) SelectionRect {
	r := SelectionRect{Position: position, Bounds: bounds}
	if len(direction) > 0 {
		r.Direction = direction[0]
	}
	return r
}

// String returns what Dart prints for the rectangle.
func (r SelectionRect) String() string {
	return fmt.Sprintf("SelectionRect(%d, %v)", r.Position, r.Bounds)
}

// TextInputStyle is the style of the text of the field that the platform shows
// in its own widgets (the composing text, for example).
//
// Flutter: services/text_input.dart TextInputStyle
type TextInputStyle struct {
	// FontFamily is the font family of the text.
	FontFamily foundation.Opt[string]
	// FontSize is the font size of the text.
	FontSize foundation.Opt[float64]
	// FontWeight is the weight of the font; the zero value is none.
	FontWeight ui.FontWeight
	// TextDirection is the direction of the text.
	TextDirection ui.TextDirection
	// TextAlign is the alignment of the text.
	TextAlign ui.TextAlign
	// LetterSpacing is the space between letters.
	LetterSpacing foundation.Opt[float64]
	// WordSpacing is the space between words.
	WordSpacing foundation.Opt[float64]
	// LineHeight is the height of a line as a multiple of the font size.
	LineHeight foundation.Opt[float64]
}

// TextInputStyleOpts are the optional parameters of NewTextInputStyle.
//
// Flutter: services/text_input.dart TextInputStyle.new
type TextInputStyleOpts struct {
	FontFamily    foundation.Opt[string]
	FontSize      foundation.Opt[float64]
	FontWeight    ui.FontWeight
	LetterSpacing foundation.Opt[float64]
	WordSpacing   foundation.Opt[float64]
	LineHeight    foundation.Opt[float64]
}

// NewTextInputStyle creates a style.
//
// Flutter: services/text_input.dart TextInputStyle.new
func NewTextInputStyle(textDirection ui.TextDirection, textAlign ui.TextAlign, o ...TextInputStyleOpts) TextInputStyle {
	op := opts.First(o)
	return TextInputStyle{op.FontFamily, op.FontSize, op.FontWeight, textDirection, textAlign,
		op.LetterSpacing, op.WordSpacing, op.LineHeight}
}

// DebugFillProperties adds the properties of the style.
func (s TextInputStyle) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	null := foundation.DefaultNull
	p.Add(foundation.NewStringProperty("fontFamily", s.FontFamily, foundation.StringPropertyOpts{DefaultValue: null}))
	p.Add(foundation.NewDoubleProperty("fontSize", s.FontSize, foundation.NumberPropertyOpts{DefaultValue: null}))
	weight := foundation.None[ui.FontWeight]()
	if s.FontWeight != 0 {
		weight = foundation.Some(s.FontWeight)
	}
	p.Add(foundation.NewDiagnosticsPropertyOpt("fontWeight", weight, foundation.DiagnosticsPropertyOpts{DefaultValue: null}))
	p.Add(foundation.NewEnumProperty("textDirection", foundation.Some(s.TextDirection)))
	p.Add(foundation.NewEnumProperty("textAlign", foundation.Some(s.TextAlign)))
	p.Add(foundation.NewDoubleProperty("letterSpacing", s.LetterSpacing, foundation.NumberPropertyOpts{DefaultValue: null}))
	p.Add(foundation.NewDoubleProperty("wordSpacing", s.WordSpacing, foundation.NumberPropertyOpts{DefaultValue: null}))
	p.Add(foundation.NewDoubleProperty("lineHeight", s.LineHeight, foundation.NumberPropertyOpts{DefaultValue: null}))
}
