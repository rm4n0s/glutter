// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"reflect"
	"slices"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// LocaleListResolutionCallback resolves the locale of an app from the list of
// the locales the user prefers and the locales the app supports. Returning false
// as the second result falls back to LocaleResolutionCallback and then to
// BasicLocaleListResolution.
//
// Deviation: Flutter's returns a nullable Locale; Go returns the second result
// (SPEC §12.10).
//
// Flutter: widgets/app.dart LocaleListResolutionCallback
type LocaleListResolutionCallback func(locales []ui.Locale, supportedLocales []ui.Locale) (ui.Locale, bool)

// LocaleResolutionCallback resolves the locale of an app from the first locale
// the user prefers (false when there is none) and the locales the app supports.
// Returning false as the second result falls back to BasicLocaleListResolution.
//
// Deviation: Flutter's takes and returns a nullable Locale; Go uses a second
// result for the presence of the argument and of the answer (SPEC §12.10).
//
// Flutter: widgets/app.dart LocaleResolutionCallback
type LocaleResolutionCallback func(locale ui.Locale, hasLocale bool, supportedLocales []ui.Locale) (ui.Locale, bool)

// BasicLocaleListResolution is the default algorithm that picks the locale of an
// app: the first of the preferred locales that the app supports in full, or by
// language and script, or by language and country, or by language, or by country,
// or the first supported locale. supportedLocales must not be empty.
//
// Flutter: widgets/app.dart basicLocaleListResolution
func BasicLocaleListResolution(preferredLocales []ui.Locale, supportedLocales []ui.Locale) ui.Locale {
	if len(preferredLocales) == 0 {
		return supportedLocales[0]
	}
	all := map[ui.Locale]bool{}
	languageAndCountry := map[[2]string]ui.Locale{}
	languageAndScript := map[[2]string]ui.Locale{}
	language := map[string]ui.Locale{}
	country := map[string]ui.Locale{}
	setIfAbsent := func(m map[string]ui.Locale, k string, l ui.Locale) {
		if _, ok := m[k]; !ok {
			m[k] = l
		}
	}
	setIfAbsentPair := func(m map[[2]string]ui.Locale, k [2]string, l ui.Locale) {
		if _, ok := m[k]; !ok {
			m[k] = l
		}
	}
	for _, l := range supportedLocales {
		all[l] = true
		setIfAbsentPair(languageAndScript, [2]string{l.LanguageCode, l.ScriptCode}, l)
		setIfAbsentPair(languageAndCountry, [2]string{l.LanguageCode, l.CountryCode}, l)
		setIfAbsent(language, l.LanguageCode, l)
		setIfAbsent(country, l.CountryCode, l)
	}

	// A language-only match is not returned at once, as it can be a low quality
	// one: the next preferred locale may have a better match.
	var matchesLanguage, matchesCountry ui.Locale
	var hasLanguage, hasCountry bool
	for i, user := range preferredLocales {
		if all[user] {
			return user
		}
		if user.ScriptCode != "" {
			if m, ok := languageAndScript[[2]string{user.LanguageCode, user.ScriptCode}]; ok {
				return m
			}
		}
		if user.CountryCode != "" {
			if m, ok := languageAndCountry[[2]string{user.LanguageCode, user.CountryCode}]; ok {
				return m
			}
		}
		if hasLanguage {
			return matchesLanguage
		}
		if m, ok := language[user.LanguageCode]; ok {
			matchesLanguage, hasLanguage = m, true
			// The first locale is usually highly preferred, so a language-only
			// match is returned at once unless the next preferred locale has the
			// same language.
			if i == 0 && (i+1 >= len(preferredLocales) || preferredLocales[i+1].LanguageCode != user.LanguageCode) {
				return matchesLanguage
			}
		}
		if !hasCountry && user.CountryCode != "" {
			if m, ok := country[user.CountryCode]; ok {
				matchesCountry, hasCountry = m, true
			}
		}
	}
	if hasLanguage {
		return matchesLanguage
	}
	if hasCountry {
		return matchesCountry
	}
	return supportedLocales[0]
}

// LocalizationsResolver resolves the locale of an app from the locales of the
// system, its supported locales and the callbacks, and tells its listeners when
// the system locales change the result. WidgetsApp keeps one.
//
// Flutter: widgets/localizations.dart LocalizationsResolver
type LocalizationsResolver struct {
	foundation.ChangeNotifier
	WidgetsBindingObserverBase
	binding                *WidgetsBinding
	locale                 foundation.Opt[ui.Locale]
	localeListResolution   LocaleListResolutionCallback
	localeResolution       LocaleResolutionCallback
	delegates              []LocalizationsDelegate
	supportedLocales       []ui.Locale
	resolvedLocale         ui.Locale
	reportedUnsupportedFor ui.Locale
}

// LocalizationsResolverOpts are the optional parameters of
// NewLocalizationsResolver.
//
// Flutter: widgets/localizations.dart LocalizationsResolver.new
type LocalizationsResolverOpts struct {
	// Locale forces the locale of the app, still resolved against the supported
	// locales.
	Locale foundation.Opt[ui.Locale]
	// LocaleListResolutionCallback resolves the locale from the preferred ones.
	LocaleListResolutionCallback LocaleListResolutionCallback
	// LocaleResolutionCallback resolves the locale from the first preferred one.
	LocaleResolutionCallback LocaleResolutionCallback
	// LocalizationsDelegates are the delegates of the app; the default widgets
	// delegate is always added after them.
	LocalizationsDelegates []LocalizationsDelegate
}

// NewLocalizationsResolver creates the resolver and starts to listen to the
// locales of the system through the binding. supportedLocales must not be empty.
//
// Flutter: widgets/localizations.dart LocalizationsResolver.new
func NewLocalizationsResolver(binding *WidgetsBinding, supportedLocales []ui.Locale, opts ...LocalizationsResolverOpts) *LocalizationsResolver {
	o := firstOpts(opts)
	r := &LocalizationsResolver{
		binding:              binding,
		locale:               o.Locale,
		localeListResolution: o.LocaleListResolutionCallback,
		localeResolution:     o.LocaleResolutionCallback,
		delegates:            o.LocalizationsDelegates,
		supportedLocales:     supportedLocales,
	}
	r.SetErrorReporter(binding.ErrorReporter())
	r.resolvedLocale = r.resolveLocales(binding.PlatformDispatcher().Locales(), supportedLocales)
	binding.AddObserver(r)
	return r
}

// Dispose stops listening to the system locales. The resolver can not be used
// afterwards.
//
// Flutter: widgets/localizations.dart LocalizationsResolver.dispose
func (r *LocalizationsResolver) Dispose() {
	r.binding.RemoveObserver(r)
	r.ChangeNotifier.Dispose()
}

// Update replaces the configuration of the resolver, and resolves the locale
// again when the supported locales changed.
//
// Flutter: widgets/localizations.dart LocalizationsResolver.update
func (r *LocalizationsResolver) Update(supportedLocales []ui.Locale, opts LocalizationsResolverOpts) {
	r.locale = opts.Locale
	r.localeListResolution = opts.LocaleListResolutionCallback
	r.localeResolution = opts.LocaleResolutionCallback
	r.delegates = opts.LocalizationsDelegates
	if !slices.Equal(r.supportedLocales, supportedLocales) {
		r.supportedLocales = supportedLocales
		r.updateResolvedLocale(r.binding.PlatformDispatcher().Locales())
	}
}

// Locale is the locale of the app: the forced one, resolved against the
// supported locales, or the one resolved from the system locales. In debug builds
// a locale that some delegate does not support is reported once with the tag
// LocaleNotSupported.
//
// Flutter: widgets/localizations.dart LocalizationsResolver.locale
func (r *LocalizationsResolver) Locale() ui.Locale {
	appLocale := r.resolvedLocale
	if l, ok := r.locale.Get(); ok {
		appLocale = r.resolveLocales([]ui.Locale{l}, r.supportedLocales)
	}
	if foundation.DebugMode {
		r.debugCheckLocalizations(appLocale)
	}
	return appLocale
}

// LocalizationsDelegates are the delegates of the app followed by the default
// delegate of the widgets. Only the first delegate of a resource type is loaded,
// so the delegates of the app can override the default one.
//
// Flutter: widgets/localizations.dart LocalizationsResolver.localizationsDelegates
func (r *LocalizationsResolver) LocalizationsDelegates() []LocalizationsDelegate {
	return append(slices.Clone(r.delegates), DefaultWidgetsLocalizationsDelegate)
}

// SupportedLocales are the locales the app supports.
//
// Flutter: widgets/localizations.dart LocalizationsResolver.supportedLocales
func (r *LocalizationsResolver) SupportedLocales() []ui.Locale { return r.supportedLocales }

// DidChangeLocales resolves the locale again.
//
// Flutter: widgets/localizations.dart LocalizationsResolver.didChangeLocales
func (r *LocalizationsResolver) DidChangeLocales(locales []ui.Locale) {
	r.updateResolvedLocale(locales)
}

func (r *LocalizationsResolver) updateResolvedLocale(preferred []ui.Locale) {
	newLocale := r.resolveLocales(preferred, r.supportedLocales)
	if newLocale != r.resolvedLocale {
		r.resolvedLocale = newLocale
		r.NotifyListeners()
	}
}

func (r *LocalizationsResolver) resolveLocales(preferred, supported []ui.Locale) ui.Locale {
	if r.localeListResolution != nil {
		if l, ok := r.localeListResolution(preferred, supported); ok {
			return l
		}
	}
	if r.localeResolution != nil {
		var first ui.Locale
		if len(preferred) > 0 {
			first = preferred[0]
		}
		if l, ok := r.localeResolution(first, len(preferred) > 0, supported); ok {
			return l
		}
	}
	return BasicLocaleListResolution(preferred, supported)
}

// String names the resolver.
//
// Flutter: widgets/localizations.dart LocalizationsResolver.toString
func (r *LocalizationsResolver) String() string { return "LocalizationsResolver" }

func (r *LocalizationsResolver) debugCheckLocalizations(locale ui.Locale) {
	if r.reportedUnsupportedFor == locale {
		return
	}
	delegates := r.LocalizationsDelegates()
	unsupported := map[reflect.Type]bool{}
	for _, d := range delegates {
		unsupported[d.ResourceType()] = true
	}
	for _, d := range delegates {
		if !unsupported[d.ResourceType()] {
			continue
		}
		if d.IsSupported(locale) {
			delete(unsupported, d.ResourceType())
		}
	}
	if len(unsupported) == 0 {
		return
	}
	r.reportedUnsupportedFor = locale
	var types []string
	for t := range unsupported {
		types = append(types, "• A "+t.String()+" delegate that supports the "+locale.String()+" locale was not found.")
	}
	slices.Sort(types)
	var supported []string
	for _, l := range r.supportedLocales {
		supported = append(supported, l.String())
	}
	r.binding.ErrorReporter()(errors.New("LocaleNotSupported",
		"this application's locale, "+locale.String()+", is not supported by all of its localization delegates",
		"unsupported", strings.Join(types, "\n"),
		"supportedLocales", strings.Join(supported, ", ")))
}
