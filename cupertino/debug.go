package cupertino

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/widgets"
)

// DebugCheckHasCupertinoLocalizations asserts that the given context has a Localizations
// ancestor that contains a CupertinoLocalizations delegate. It panics with the tag
// CupertinoLocalizationsMissing in debug builds, naming the widget that needs them, and
// returns true so it can be used in a condition.
//
// Flutter: cupertino/debug.dart debugCheckHasCupertinoLocalizations
func DebugCheckHasCupertinoLocalizations(bc widgets.BuildContext) bool {
	if !foundation.DebugMode {
		return true
	}
	if _, ok := CupertinoLocalizationsMaybeOf(bc); ok {
		return true
	}
	panic(errors.New(tagCupertinoLocalizationsMissing, "No CupertinoLocalizations found. "+foundation.RuntimeTypeName(bc.Widget())+
		" widgets require CupertinoLocalizations to be provided by a Localizations widget ancestor.\n"+
		"The cupertino library uses Localizations to generate messages, labels, and abbreviations.\n"+
		"To introduce a CupertinoLocalizations, either use a CupertinoApp at the root of your application to include them automatically, "+
		"or add a Localization widget with a CupertinoLocalizations delegate.",
		"widget", foundation.RuntimeTypeName(bc.Widget())))
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagCupertinoLocalizationsMissing = "CupertinoLocalizationsMissing"
)
