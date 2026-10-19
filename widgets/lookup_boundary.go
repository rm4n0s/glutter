package widgets

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
)

// LookupBoundary hides the widgets above it from the lookups of the widgets
// below it that go through the LookupBoundary functions. A widget that is
// hidden by a boundary cannot be found by them, so a subtree can be kept apart
// from the ancestors it is built under. The ordinary lookups (DependOn,
// FindAncestorWidgetOfExactType, ...) ignore it.
//
// Flutter: widgets/lookup_boundary.dart LookupBoundary
type LookupBoundary struct {
	InheritedWidget
}

// NewLookupBoundary creates a boundary around the child.
//
// Flutter: widgets/lookup_boundary.dart LookupBoundary.new
func NewLookupBoundary(child Widget, opts ...WidgetOpts) *LookupBoundary {
	return &LookupBoundary{NewInheritedWidgetBase(child, opts...)}
}

// UpdateShouldNotify is false: the boundary has no data to notify about.
//
// Flutter: widgets/lookup_boundary.dart LookupBoundary.updateShouldNotify
func (w *LookupBoundary) UpdateShouldNotify(Widget) bool { return false }

func isLookupBoundary(e Element) bool {
	_, ok := e.AsElement().widget.(*LookupBoundary)
	return ok
}

// LookupBoundaryDependOnInheritedWidgetOfExactType is DependOn that does not
// look past a LookupBoundary. The context depends on the boundary even when
// nothing is found, so that its dependencies change when it moves in the tree.
//
// Flutter: widgets/lookup_boundary.dart LookupBoundary.dependOnInheritedWidgetOfExactType
func LookupBoundaryDependOnInheritedWidgetOfExactType[T InheritedWidgetInterface](bc BuildContext, aspect ...any) (T, bool) {
	var zero T
	var a any
	if len(aspect) > 0 {
		a = aspect[0]
	}
	// This call makes sure that the context depends on something, so that
	// DidChangeDependencies is called when the context moves in the tree even
	// when the request is not fulfilled.
	DependOn[*LookupBoundary](bc)
	candidate := LookupBoundaryGetElementForInheritedWidgetOfExactType[T](bc)
	if candidate == nil {
		return zero, false
	}
	bc.DependOnInheritedElement(candidate, a)
	return candidate.AsElement().widget.(T), true
}

// LookupBoundaryGetInheritedWidgetOfExactType is GetInherited that does not look
// past a LookupBoundary.
//
// Flutter: widgets/lookup_boundary.dart LookupBoundary.getInheritedWidgetOfExactType
func LookupBoundaryGetInheritedWidgetOfExactType[T InheritedWidgetInterface](bc BuildContext) (T, bool) {
	var zero T
	candidate := LookupBoundaryGetElementForInheritedWidgetOfExactType[T](bc)
	if candidate == nil {
		return zero, false
	}
	return candidate.AsElement().widget.(T), true
}

// LookupBoundaryGetElementForInheritedWidgetOfExactType is
// GetElementForInheritedWidget that does not look past a LookupBoundary. It
// returns nil when there is no such element, or when a boundary is closer.
//
// Flutter: widgets/lookup_boundary.dart LookupBoundary.getElementForInheritedWidgetOfExactType
func LookupBoundaryGetElementForInheritedWidgetOfExactType[T InheritedWidgetInterface](bc BuildContext) InheritedElementSelf {
	candidate := GetElementForInheritedWidget[T](bc)
	if candidate == nil {
		return nil
	}
	if boundary := GetElementForInheritedWidget[*LookupBoundary](bc); boundary != nil && boundary.Depth() > candidate.Depth() {
		return nil
	}
	return candidate
}

// LookupBoundaryFindAncestorWidgetOfExactType is FindAncestorWidgetOfExactType
// that does not look past a LookupBoundary.
//
// Flutter: widgets/lookup_boundary.dart LookupBoundary.findAncestorWidgetOfExactType
func LookupBoundaryFindAncestorWidgetOfExactType[T Widget](bc BuildContext) (T, bool) {
	var found T
	ok := false
	bc.VisitAncestorElements(func(ancestor Element) bool {
		if w, isT := ancestor.AsElement().widget.(T); isT {
			found, ok = w, true
			return false
		}
		return !isLookupBoundary(ancestor)
	})
	return found, ok
}

// LookupBoundaryFindAncestorStateOfType is FindAncestorStateOfType that does not
// look past a LookupBoundary.
//
// Flutter: widgets/lookup_boundary.dart LookupBoundary.findAncestorStateOfType
func LookupBoundaryFindAncestorStateOfType[S State](bc BuildContext) (S, bool) {
	var found S
	ok := false
	bc.VisitAncestorElements(func(ancestor Element) bool {
		if se, isStateful := ancestor.(*StatefulElement); isStateful {
			if s, isS := se.state.(S); isS {
				found, ok = s, true
				return false
			}
		}
		return !isLookupBoundary(ancestor)
	})
	return found, ok
}

// LookupBoundaryFindRootAncestorStateOfType is FindRootAncestorStateOfType that
// does not look past a LookupBoundary.
//
// Flutter: widgets/lookup_boundary.dart LookupBoundary.findRootAncestorStateOfType
func LookupBoundaryFindRootAncestorStateOfType[S State](bc BuildContext) (S, bool) {
	var found S
	ok := false
	bc.VisitAncestorElements(func(ancestor Element) bool {
		if se, isStateful := ancestor.(*StatefulElement); isStateful {
			if s, isS := se.state.(S); isS {
				found, ok = s, true
			}
		}
		return !isLookupBoundary(ancestor)
	})
	return found, ok
}

// LookupBoundaryFindAncestorRenderObjectOfType is FindAncestorRenderObjectOfType
// that does not look past a LookupBoundary.
//
// Flutter: widgets/lookup_boundary.dart LookupBoundary.findAncestorRenderObjectOfType
func LookupBoundaryFindAncestorRenderObjectOfType[T rendering.RenderObject](bc BuildContext) (T, bool) {
	var found T
	ok := false
	bc.VisitAncestorElements(func(ancestor Element) bool {
		if ro, has := ancestor.ownRenderObject(); has {
			if typed, isT := ro.(T); isT {
				found, ok = typed, true
				return false
			}
		}
		return !isLookupBoundary(ancestor)
	})
	return found, ok
}

// LookupBoundaryVisitAncestorElements walks the ancestors from the closest to
// the root until the visitor returns false or a LookupBoundary has been visited.
//
// Flutter: widgets/lookup_boundary.dart LookupBoundary.visitAncestorElements
func LookupBoundaryVisitAncestorElements(bc BuildContext, visitor ConditionalElementVisitor) {
	bc.VisitAncestorElements(func(ancestor Element) bool {
		return visitor(ancestor) && !isLookupBoundary(ancestor)
	})
}

// LookupBoundaryVisitChildElements visits the children of the context except the
// ones that are a LookupBoundary.
//
// Flutter: widgets/lookup_boundary.dart LookupBoundary.visitChildElements
func LookupBoundaryVisitChildElements(bc BuildContext, visitor ElementVisitor) {
	bc.VisitChildElements(func(child Element) {
		if !isLookupBoundary(child) {
			visitor(child)
		}
	})
}

func debugIsHidingAncestor(bc BuildContext, found func(Element) bool) bool {
	result := false
	if foundation.DebugMode {
		hiddenByBoundary := false
		ancestorFound := false
		bc.VisitAncestorElements(func(ancestor Element) bool {
			if found(ancestor) {
				ancestorFound = true
				return false
			}
			hiddenByBoundary = hiddenByBoundary || isLookupBoundary(ancestor)
			return true
		})
		result = ancestorFound && hiddenByBoundary
	}
	return result
}

// LookupBoundaryDebugIsHidingAncestorWidgetOfExactType is whether a widget of
// type T is above the context but hidden from it by a LookupBoundary. In debug
// builds only; release builds report false.
//
// Flutter: widgets/lookup_boundary.dart LookupBoundary.debugIsHidingAncestorWidgetOfExactType
func LookupBoundaryDebugIsHidingAncestorWidgetOfExactType[T Widget](bc BuildContext) bool {
	return debugIsHidingAncestor(bc, func(e Element) bool {
		_, ok := e.AsElement().widget.(T)
		return ok
	})
}

// LookupBoundaryDebugIsHidingAncestorStateOfType is whether a state of type S is
// above the context but hidden from it by a LookupBoundary. In debug builds
// only; release builds report false.
//
// Flutter: widgets/lookup_boundary.dart LookupBoundary.debugIsHidingAncestorStateOfType
func LookupBoundaryDebugIsHidingAncestorStateOfType[S State](bc BuildContext) bool {
	return debugIsHidingAncestor(bc, func(e Element) bool {
		if se, ok := e.(*StatefulElement); ok {
			_, isS := se.state.(S)
			return isS
		}
		return false
	})
}

// LookupBoundaryDebugIsHidingAncestorRenderObjectOfType is whether a render
// object of type T is above the context but hidden from it by a LookupBoundary.
// In debug builds only; release builds report false.
//
// Flutter: widgets/lookup_boundary.dart LookupBoundary.debugIsHidingAncestorRenderObjectOfType
func LookupBoundaryDebugIsHidingAncestorRenderObjectOfType[T rendering.RenderObject](bc BuildContext) bool {
	return debugIsHidingAncestor(bc, func(e Element) bool {
		if ro, ok := e.ownRenderObject(); ok {
			_, isT := ro.(T)
			return isT
		}
		return false
	})
}
