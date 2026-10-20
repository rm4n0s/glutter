// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package ui

// StringAttribute is an immutable annotation of a range of a string, which
// semantics sends to the accessibility service.
//
// Flutter: ui/semantics.dart StringAttribute
type StringAttribute interface {
	// Range is the range of the string the attribute applies to (UTF-8 byte
	// offsets, SPEC §12.10).
	//
	// Flutter: ui/semantics.dart StringAttribute.range
	Range() TextRange
	// CopyWithRange returns the attribute for another range.
	//
	// Flutter: ui/semantics.dart StringAttribute.copy
	CopyWithRange(r TextRange) StringAttribute
}

// SpellOutStringAttribute says that the range should be spoken character by
// character.
//
// Flutter: ui/semantics.dart SpellOutStringAttribute
type SpellOutStringAttribute struct{ rng TextRange }

// NewSpellOutStringAttribute creates an attribute for the range.
//
// Flutter: ui/semantics.dart SpellOutStringAttribute.new
func NewSpellOutStringAttribute(r TextRange) SpellOutStringAttribute {
	return SpellOutStringAttribute{r}
}

// Range is the range the attribute applies to.
//
// Flutter: ui/semantics.dart StringAttribute.range
func (a SpellOutStringAttribute) Range() TextRange { return a.rng }

// CopyWithRange returns the attribute for another range.
//
// Flutter: ui/semantics.dart SpellOutStringAttribute.copy
func (a SpellOutStringAttribute) CopyWithRange(r TextRange) StringAttribute {
	return SpellOutStringAttribute{r}
}

// LocaleStringAttribute says that the range is in the locale.
//
// Flutter: ui/semantics.dart LocaleStringAttribute
type LocaleStringAttribute struct {
	rng    TextRange
	locale Locale
}

// NewLocaleStringAttribute creates an attribute for the range and the locale.
//
// Flutter: ui/semantics.dart LocaleStringAttribute.new
func NewLocaleStringAttribute(r TextRange, locale Locale) LocaleStringAttribute {
	return LocaleStringAttribute{r, locale}
}

// Range is the range the attribute applies to.
//
// Flutter: ui/semantics.dart StringAttribute.range
func (a LocaleStringAttribute) Range() TextRange { return a.rng }

// Locale is the locale of the range.
//
// Flutter: ui/semantics.dart LocaleStringAttribute.locale
func (a LocaleStringAttribute) Locale() Locale { return a.locale }

// CopyWithRange returns the attribute for another range.
//
// Flutter: ui/semantics.dart LocaleStringAttribute.copy
func (a LocaleStringAttribute) CopyWithRange(r TextRange) StringAttribute {
	return LocaleStringAttribute{r, a.locale}
}
