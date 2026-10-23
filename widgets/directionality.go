package widgets

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/ui"
)

// Directionality is the inherited widget that says in which direction the text
// and the widgets that depend on it are laid out (left to right or right to
// left).
//
// Deviation: Flutter's is a "ubiquitous" inherited widget that finds its
// dependents by walking the subtree; this one keeps the dependents like any
// other inherited widget, which notifies the same elements (SPEC §12.10).
//
// Flutter: widgets/basic.dart Directionality
type Directionality struct {
	InheritedWidget
	textDirection ui.TextDirection
}

// NewDirectionality creates the widget that gives the text direction to the
// widgets below it.
//
// Flutter: widgets/basic.dart Directionality.new
func NewDirectionality(textDirection ui.TextDirection, child Widget, opts ...WidgetOpts) *Directionality {
	return &Directionality{NewInheritedWidgetBase(child, opts...), textDirection}
}

// TextDirection is the direction the widget gives.
//
// Flutter: widgets/basic.dart Directionality.textDirection
func (w *Directionality) TextDirection() ui.TextDirection { return w.textDirection }

// UpdateShouldNotify is whether the direction changed.
//
// Flutter: widgets/basic.dart Directionality.updateShouldNotify
func (w *Directionality) UpdateShouldNotify(old Widget) bool {
	return w.textDirection != old.(*Directionality).textDirection
}

// MaybeDirectionalityOf is the direction of the closest Directionality above
// the context. The context depends on it.
//
// Flutter: widgets/basic.dart Directionality.maybeOf
func MaybeDirectionalityOf(bc BuildContext) (ui.TextDirection, bool) {
	w, ok := DependOn[*Directionality](bc)
	if !ok {
		return ui.TextDirectionUnspecified, false
	}
	return w.textDirection, true
}

// DirectionalityOf is the direction of the closest Directionality above the
// context. It panics with the tag DirectionalityMissing when there is none.
//
// Flutter: widgets/basic.dart Directionality.of
func DirectionalityOf(bc BuildContext) ui.TextDirection {
	d, ok := MaybeDirectionalityOf(bc)
	if !ok {
		DebugCheckHasDirectionality(bc)
	}
	return d
}

// DebugCheckHasDirectionality panics with the tag DirectionalityMissing, naming
// the widget that needs it, when there is no Directionality above the context
// (or at it). Pass the empty string for the name when the widget is not known.
// It returns true so it can be used in a condition.
//
// Flutter: widgets/debug.dart debugCheckHasDirectionality
func DebugCheckHasDirectionality(bc BuildContext) bool {
	if _, ok := bc.Widget().(*Directionality); ok {
		return true
	}
	if GetElementForInheritedWidget[*Directionality](bc) != nil {
		return true
	}
	panic(errors.New(tagDirectionalityMissing,
		"no Directionality widget found; the widget needs a Directionality ancestor to know the text direction; to introduce one, wrap the widget (or the whole app) in a Directionality widget with a text direction",
		"widget", bc.(Element).String()))
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagDirectionalityMissing = "DirectionalityMissing"
)
