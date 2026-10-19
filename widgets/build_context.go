package widgets

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
)

// ElementVisitor is called for each element a visit reaches.
//
// Flutter: widgets/framework.dart ElementVisitor
type ElementVisitor func(element Element)

// ConditionalElementVisitor is called for each element a walk reaches and
// returns whether the walk goes on.
//
// Flutter: widgets/framework.dart ConditionalElementVisitor
type ConditionalElementVisitor func(element Element) bool

// BuildContext is a handle to the location of a widget in the widget tree. An
// element is its own build context: the one that a widget's Build method
// receives is the element that builds it.
//
// Name a build context bc in Go code (ctx is for context.Context).
//
// Flutter: widgets/framework.dart BuildContext
type BuildContext interface {
	// Widget is the current configuration of the element that this context
	// belongs to. It panics with the tag ElementWidgetUnmounted when the
	// element has been unmounted.
	//
	// Flutter: widgets/framework.dart BuildContext.widget
	Widget() Widget
	// Owner is the build owner that manages the element, or nil before the
	// element is mounted.
	//
	// Flutter: widgets/framework.dart BuildContext.owner
	Owner() *BuildOwner
	// Mounted is whether the element is in the tree: false after it has been
	// unmounted.
	//
	// Flutter: widgets/framework.dart BuildContext.mounted
	Mounted() bool
	// DebugDoingBuild is whether the widget of this context is being built.
	//
	// Flutter: widgets/framework.dart BuildContext.debugDoingBuild
	DebugDoingBuild() bool
	// FindRenderObject is the render object of this element, or of the first
	// descendant that has one. It is valid only while the element is active.
	//
	// Flutter: widgets/framework.dart BuildContext.findRenderObject
	FindRenderObject() rendering.RenderObject
	// findAncestorRenderObject returns the render object of the closest
	// ancestor element for which accept returns true, or nil. The generic
	// FindAncestorRenderObjectOfType is the public form.
	findAncestorRenderObject(accept func(rendering.RenderObject) bool) rendering.RenderObject
	// VisitAncestorElements walks the ancestors from the closest to the root
	// until the visitor returns false.
	//
	// Flutter: widgets/framework.dart BuildContext.visitAncestorElements
	VisitAncestorElements(visitor ConditionalElementVisitor)
	// VisitChildElements calls the visitor for each child of the element.
	//
	// Flutter: widgets/framework.dart BuildContext.visitChildElements
	VisitChildElements(visitor ElementVisitor)
	// DescribeElement returns a node that describes the element, for an error.
	//
	// Flutter: widgets/framework.dart BuildContext.describeElement
	DescribeElement(name string, style ...foundation.DiagnosticsTreeStyle) foundation.DiagnosticsNode
	// DescribeWidget returns a node that describes the widget, for an error.
	//
	// Flutter: widgets/framework.dart BuildContext.describeWidget
	DescribeWidget(name string, style ...foundation.DiagnosticsTreeStyle) foundation.DiagnosticsNode
	// DescribeMissingAncestor returns nodes that describe an ancestor that
	// could not be found, for an error.
	//
	// Flutter: widgets/framework.dart BuildContext.describeMissingAncestor
	DescribeMissingAncestor(expectedAncestorType string) []foundation.DiagnosticsNode
	// DescribeOwnershipChain returns a node that lists the widgets that
	// created this one, for an error.
	//
	// Flutter: widgets/framework.dart BuildContext.describeOwnershipChain
	DescribeOwnershipChain(name string) foundation.DiagnosticsNode
}

// FindAncestorWidgetOfExactType returns the closest ancestor widget of type T
// (the exact type, not a type that embeds it), or the zero value and false.
// The context must be active.
//
// Flutter: widgets/framework.dart BuildContext.findAncestorWidgetOfExactType
func FindAncestorWidgetOfExactType[T Widget](bc BuildContext) (T, bool) {
	var found T
	ok := false
	bc.(Element).AsElement().debugCheckStateIsActiveForAncestorLookup()
	bc.VisitAncestorElements(func(e Element) bool {
		if w, isT := e.AsElement().widget.(T); isT {
			found, ok = w, true
			return false
		}
		return true
	})
	return found, ok
}

// FindAncestorStateOfType returns the state of the closest StatefulWidget
// ancestor whose state is of type S, or the zero value and false.
//
// Flutter: widgets/framework.dart BuildContext.findAncestorStateOfType
func FindAncestorStateOfType[S State](bc BuildContext) (S, bool) {
	var found S
	ok := false
	bc.(Element).AsElement().debugCheckStateIsActiveForAncestorLookup()
	bc.VisitAncestorElements(func(e Element) bool {
		if se, isStateful := e.(*StatefulElement); isStateful {
			if s, isS := se.state.(S); isS {
				found, ok = s, true
				return false
			}
		}
		return true
	})
	return found, ok
}

// FindRootAncestorStateOfType returns the state of the furthest StatefulWidget
// ancestor whose state is of type S, or the zero value and false.
//
// Flutter: widgets/framework.dart BuildContext.findRootAncestorStateOfType
func FindRootAncestorStateOfType[S State](bc BuildContext) (S, bool) {
	var found S
	ok := false
	bc.(Element).AsElement().debugCheckStateIsActiveForAncestorLookup()
	bc.VisitAncestorElements(func(e Element) bool {
		if se, isStateful := e.(*StatefulElement); isStateful {
			if s, isS := se.state.(S); isS {
				found, ok = s, true
			}
		}
		return true
	})
	return found, ok
}

// FindAncestorRenderObjectOfType returns the render object of the closest
// ancestor element that has one of type T, or the zero value and false.
//
// Flutter: widgets/framework.dart BuildContext.findAncestorRenderObjectOfType
func FindAncestorRenderObjectOfType[T rendering.RenderObject](bc BuildContext) (T, bool) {
	var zero T
	r := bc.findAncestorRenderObject(func(r rendering.RenderObject) bool {
		_, ok := r.(T)
		return ok
	})
	if r == nil {
		return zero, false
	}
	return r.(T), true
}
