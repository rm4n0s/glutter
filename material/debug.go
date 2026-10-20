package material

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/widgets"
)

// DebugCheckHasMaterial asserts that the given context has a Material ancestor
// within the closest LookupBoundary. It panics with the tag MaterialMissing in
// debug builds, naming the widget that needs a Material, and returns true so
// it can be used in a condition. Used by many material widgets to make sure that
// they are only used in contexts where they can print ink onto some material.
//
// Flutter: material/debug.dart debugCheckHasMaterial
func DebugCheckHasMaterial(bc widgets.BuildContext) bool {
	if !foundation.DebugMode {
		return true
	}
	if _, ok := widgets.LookupBoundaryFindAncestorWidgetOfExactType[*Material](bc); ok {
		return true
	}
	hiddenByBoundary := widgets.LookupBoundaryDebugIsHidingAncestorWidgetOfExactType[*Material](bc)
	summary := "No Material widget found."
	if hiddenByBoundary {
		summary = "No Material widget found within the closest LookupBoundary. There is an ancestor Material widget, but it is hidden by a LookupBoundary."
	}
	panic(errors.New("MaterialMissing", summary+" "+foundation.RuntimeTypeName(bc.Widget())+" widgets require a Material widget ancestor within the closest LookupBoundary.\n"+
		"In Material Design, most widgets are conceptually \"printed\" on a sheet of material. In Flutter's material library, that material is represented by the Material widget. "+
		"It is the Material widget that renders ink splashes, for instance. Because of this, many material library widgets require that there be a Material widget in the tree above them.\n"+
		"To introduce a Material widget, you can either directly include one, or use a widget that contains Material itself, such as a Card, Dialog, Drawer, or Scaffold.",
		"widget", foundation.RuntimeTypeName(bc.Widget()), "hiddenByLookupBoundary", hiddenByBoundary))
}

// DebugCheckHasMaterialLocalizations asserts that the given context has a
// Localizations ancestor that contains a MaterialLocalizations delegate. It
// panics with the tag MaterialLocalizationsMissing in debug builds, and returns
// true so it can be used in a condition.
//
// Flutter: material/debug.dart debugCheckHasMaterialLocalizations
func DebugCheckHasMaterialLocalizations(bc widgets.BuildContext) bool {
	if !foundation.DebugMode {
		return true
	}
	if _, ok := MaterialLocalizationsMaybeOf(bc); ok {
		return true
	}
	panic(errors.New("MaterialLocalizationsMissing", "No MaterialLocalizations found. "+foundation.RuntimeTypeName(bc.Widget())+
		" widgets require MaterialLocalizations to be provided by a Localizations widget ancestor.\n"+
		"The material library uses Localizations to generate messages, labels, and abbreviations.\n"+
		"To introduce a MaterialLocalizations, either use a MaterialApp at the root of your application to include them automatically, "+
		"or add a Localization widget with a MaterialLocalizations delegate.",
		"widget", foundation.RuntimeTypeName(bc.Widget())))
}
