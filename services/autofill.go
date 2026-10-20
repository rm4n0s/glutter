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
)

// The autofill hints, for AutofillConfiguration.AutofillHints: what a text
// field is for, so a password manager or an operating system can fill it.
//
// Flutter: services/autofill.dart AutofillHints
const (
	AutofillHintAddressCity                     = "addressCity"
	AutofillHintAddressCityAndState             = "addressCityAndState"
	AutofillHintAddressState                    = "addressState"
	AutofillHintBirthday                        = "birthday"
	AutofillHintBirthdayDay                     = "birthdayDay"
	AutofillHintBirthdayMonth                   = "birthdayMonth"
	AutofillHintBirthdayYear                    = "birthdayYear"
	AutofillHintCountryCode                     = "countryCode"
	AutofillHintCountryName                     = "countryName"
	AutofillHintCreditCardExpirationDate        = "creditCardExpirationDate"
	AutofillHintCreditCardExpirationDay         = "creditCardExpirationDay"
	AutofillHintCreditCardExpirationMonth       = "creditCardExpirationMonth"
	AutofillHintCreditCardExpirationYear        = "creditCardExpirationYear"
	AutofillHintCreditCardFamilyName            = "creditCardFamilyName"
	AutofillHintCreditCardGivenName             = "creditCardGivenName"
	AutofillHintCreditCardMiddleName            = "creditCardMiddleName"
	AutofillHintCreditCardName                  = "creditCardName"
	AutofillHintCreditCardNumber                = "creditCardNumber"
	AutofillHintCreditCardSecurityCode          = "creditCardSecurityCode"
	AutofillHintCreditCardType                  = "creditCardType"
	AutofillHintEmail                           = "email"
	AutofillHintFamilyName                      = "familyName"
	AutofillHintFullStreetAddress               = "fullStreetAddress"
	AutofillHintGender                          = "gender"
	AutofillHintGivenName                       = "givenName"
	AutofillHintImpp                            = "impp"
	AutofillHintJobTitle                        = "jobTitle"
	AutofillHintLanguage                        = "language"
	AutofillHintLocation                        = "location"
	AutofillHintMiddleInitial                   = "middleInitial"
	AutofillHintMiddleName                      = "middleName"
	AutofillHintName                            = "name"
	AutofillHintNamePrefix                      = "namePrefix"
	AutofillHintNameSuffix                      = "nameSuffix"
	AutofillHintNewPassword                     = "newPassword"
	AutofillHintNewUsername                     = "newUsername"
	AutofillHintNickname                        = "nickname"
	AutofillHintOneTimeCode                     = "oneTimeCode"
	AutofillHintOrganizationName                = "organizationName"
	AutofillHintPassword                        = "password"
	AutofillHintPhoto                           = "photo"
	AutofillHintPostalAddress                   = "postalAddress"
	AutofillHintPostalAddressExtended           = "postalAddressExtended"
	AutofillHintPostalAddressExtendedPostalCode = "postalAddressExtendedPostalCode"
	AutofillHintPostalCode                      = "postalCode"
	AutofillHintStreetAddressLevel1             = "streetAddressLevel1"
	AutofillHintStreetAddressLevel2             = "streetAddressLevel2"
	AutofillHintStreetAddressLevel3             = "streetAddressLevel3"
	AutofillHintStreetAddressLevel4             = "streetAddressLevel4"
	AutofillHintStreetAddressLine1              = "streetAddressLine1"
	AutofillHintStreetAddressLine2              = "streetAddressLine2"
	AutofillHintStreetAddressLine3              = "streetAddressLine3"
	AutofillHintSublocality                     = "sublocality"
	AutofillHintTelephoneNumber                 = "telephoneNumber"
	AutofillHintTelephoneNumberAreaCode         = "telephoneNumberAreaCode"
	AutofillHintTelephoneNumberCountryCode      = "telephoneNumberCountryCode"
	AutofillHintTelephoneNumberDevice           = "telephoneNumberDevice"
	AutofillHintTelephoneNumberExtension        = "telephoneNumberExtension"
	AutofillHintTelephoneNumberLocal            = "telephoneNumberLocal"
	AutofillHintTelephoneNumberLocalPrefix      = "telephoneNumberLocalPrefix"
	AutofillHintTelephoneNumberLocalSuffix      = "telephoneNumberLocalSuffix"
	AutofillHintTelephoneNumberNational         = "telephoneNumberNational"
	AutofillHintTransactionAmount               = "transactionAmount"
	AutofillHintTransactionCurrency             = "transactionCurrency"
	AutofillHintURL                             = "url"
	AutofillHintUsername                        = "username"
)

// AutofillConfiguration is how a text field takes part in autofill: whether it
// does, what it is for, and its current value.
//
// Flutter: services/autofill.dart AutofillConfiguration
type AutofillConfiguration struct {
	// Enabled is whether the field can be autofilled.
	//
	// Flutter: services/autofill.dart AutofillConfiguration.enabled
	Enabled bool
	// UniqueIdentifier identifies the field in its AutofillScope.
	//
	// Flutter: services/autofill.dart AutofillConfiguration.uniqueIdentifier
	UniqueIdentifier string
	// AutofillHints say what the field is for, from the AutofillHint constants.
	//
	// Flutter: services/autofill.dart AutofillConfiguration.autofillHints
	AutofillHints []string
	// CurrentEditingValue is the value of the field.
	//
	// Flutter: services/autofill.dart AutofillConfiguration.currentEditingValue
	CurrentEditingValue TextEditingValue
	// HintText is the placeholder text of the field.
	//
	// Flutter: services/autofill.dart AutofillConfiguration.hintText
	HintText foundation.Opt[string]
}

// AutofillConfigurationDisabled is the configuration of a field that can't be
// autofilled.
//
// Flutter: services/autofill.dart AutofillConfiguration.disabled
var AutofillConfigurationDisabled = AutofillConfiguration{CurrentEditingValue: TextEditingValueEmpty}

// AutofillConfigurationOpts are the optional parameters of
// NewAutofillConfiguration.
//
// Flutter: services/autofill.dart AutofillConfiguration.new
type AutofillConfigurationOpts struct {
	// HintText is the placeholder text of the field.
	HintText foundation.Opt[string]
}

// NewAutofillConfiguration creates the configuration of a field that can be
// autofilled.
//
// Flutter: services/autofill.dart AutofillConfiguration.new
func NewAutofillConfiguration(uniqueIdentifier string, autofillHints []string, currentEditingValue TextEditingValue, o ...AutofillConfigurationOpts) AutofillConfiguration {
	return AutofillConfiguration{
		Enabled: true, UniqueIdentifier: uniqueIdentifier, AutofillHints: autofillHints,
		CurrentEditingValue: currentEditingValue, HintText: first(o).HintText,
	}
}

// Equal reports whether two configurations are the same.
//
// Flutter: services/autofill.dart AutofillConfiguration.==
func (c AutofillConfiguration) Equal(o AutofillConfiguration) bool {
	return c.Enabled == o.Enabled && c.UniqueIdentifier == o.UniqueIdentifier &&
		slices.Equal(c.AutofillHints, o.AutofillHints) && c.CurrentEditingValue.Equal(o.CurrentEditingValue) &&
		c.HintText == o.HintText
}

// String returns what Dart prints for the configuration.
func (c AutofillConfiguration) String() string {
	d := []string{
		fmt.Sprintf("enabled: %v", c.Enabled),
		"uniqueIdentifier: " + c.UniqueIdentifier,
		fmt.Sprintf("autofillHints: [%s]", strings.Join(c.AutofillHints, ", ")),
		"currentEditingValue: " + c.CurrentEditingValue.String(),
	}
	if v, ok := c.HintText.Get(); ok {
		d = append(d, "hintText: "+v)
	}
	return "AutofillConfiguration(" + strings.Join(d, ", ") + ")"
}

// AutofillClient is a text field that can be autofilled.
//
// Flutter: services/autofill.dart AutofillClient
type AutofillClient interface {
	// AutofillID is the identifier of the field in its scope.
	//
	// Flutter: services/autofill.dart AutofillClient.autofillId
	AutofillID() string
	// TextInputConfiguration is the configuration of the field.
	//
	// Flutter: services/autofill.dart AutofillClient.textInputConfiguration
	TextInputConfiguration() TextInputConfiguration
	// Autofill fills the field with the value.
	//
	// Flutter: services/autofill.dart AutofillClient.autofill
	Autofill(newEditingValue TextEditingValue)
}

// AutofillScope is a group of fields that are autofilled together, such as a
// username and a password.
//
// Flutter: services/autofill.dart AutofillScope
type AutofillScope interface {
	// GetAutofillClient returns the field with the identifier, or nil.
	//
	// Flutter: services/autofill.dart AutofillScope.getAutofillClient
	GetAutofillClient(autofillID string) AutofillClient
	// AutofillClients returns the fields of the scope.
	//
	// Flutter: services/autofill.dart AutofillScope.autofillClients
	AutofillClients() []AutofillClient
	// Attach connects the client that triggered the input to the text input of
	// the platform, with the configuration of the whole scope.
	//
	// Flutter: services/autofill.dart AutofillScope.attach
	Attach(textInput *TextInput, trigger TextInputClient, configuration TextInputConfiguration) *TextInputConnection
}

// AutofillScopeAttach is the Attach of a scope: it checks that every field of
// the scope enables autofill, and attaches the trigger client to the text
// input. Embed it in the Attach of a scope.
//
// A field without autofill in the scope panics in a debug build with the tag
// AutofillClientDisabled.
//
// Deviation: Flutter sends the configurations of all the fields of the scope to
// the platform; no platform of Glutter uses them, so only the configuration of
// the trigger is attached (SPEC §12.10).
//
// Flutter: services/autofill.dart AutofillScopeMixin.attach
func AutofillScopeAttach(scope AutofillScope, textInput *TextInput, trigger TextInputClient, configuration TextInputConfiguration) *TextInputConnection {
	if foundation.DebugMode {
		for _, c := range scope.AutofillClients() {
			if !c.TextInputConfiguration().AutofillConfiguration.Enabled {
				panic(errors.New("AutofillClientDisabled", "every client in AutofillScope.AutofillClients must enable autofill",
					"autofillId", c.AutofillID()))
			}
		}
	}
	return textInput.Attach(trigger, configuration)
}
