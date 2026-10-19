package widgets

import (
	"reflect"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
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
	// Size is the size of the render box of this element; false when there is no
	// box. The element must be active, laid out and not in a build.
	//
	// Flutter: widgets/framework.dart BuildContext.size
	Size() (ui.Size, bool)
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
	// DependOnInheritedElement registers the context as a dependent of the
	// inherited element, with an aspect for the models that use it, and returns
	// the inherited widget. The context is rebuilt when the widget changes.
	// Look inherited widgets up with DependOn instead.
	//
	// Flutter: widgets/framework.dart BuildContext.dependOnInheritedElement
	DependOnInheritedElement(ancestor InheritedElementSelf, aspect any) Widget
	// dependOnInheritedType registers a dependency on the closest inherited
	// widget of the type and returns it, or nil.
	dependOnInheritedType(t reflect.Type, aspect any) Widget
	// inheritedElementOfType is the closest inherited element of the type, or
	// nil, without registering a dependency.
	inheritedElementOfType(t reflect.Type) Element
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

// DependOn returns the closest inherited widget of type T above the context
// and registers the context as a dependent of it: the context is rebuilt when
// the widget changes. The optional aspect is for inherited models. It returns
// the zero value and false when there is none. The context must be active.
//
// Flutter: widgets/framework.dart BuildContext.dependOnInheritedWidgetOfExactType
func DependOn[T InheritedWidgetInterface](bc BuildContext, aspect ...any) (T, bool) {
	var a any
	if len(aspect) > 0 {
		a = aspect[0]
	}
	w := bc.dependOnInheritedType(reflect.TypeFor[T](), a)
	if w == nil {
		var zero T
		return zero, false
	}
	return w.(T), true
}

// GetInherited returns the closest inherited widget of type T above the context
// without registering a dependency: the context is not rebuilt when it changes.
// Use it from callbacks and from InitState.
//
// Flutter: widgets/framework.dart BuildContext.getInheritedWidgetOfExactType
func GetInherited[T InheritedWidgetInterface](bc BuildContext) (T, bool) {
	e := bc.inheritedElementOfType(reflect.TypeFor[T]())
	if e == nil {
		var zero T
		return zero, false
	}
	return e.AsElement().widget.(T), true
}

// GetElementForInheritedWidget returns the closest inherited element for
// widgets of type T above the context, or nil, without registering a
// dependency.
//
// Flutter: widgets/framework.dart BuildContext.getElementForInheritedWidgetOfExactType
func GetElementForInheritedWidget[T InheritedWidgetInterface](bc BuildContext) InheritedElementSelf {
	e := bc.inheritedElementOfType(reflect.TypeFor[T]())
	if e == nil {
		return nil
	}
	return e.(InheritedElementSelf)
}
