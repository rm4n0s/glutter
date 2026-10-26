package widgets

import (
	"github.com/rm4n0s/glutter/foundation"
)

// ErrorBoundaryOpts are the optional parameters of NewErrorBoundary.
//
// Flutter: no counterpart
type ErrorBoundaryOpts struct {
	// Key is the key of the widget. A new key makes a new boundary, which tries
	// its child again after the old one failed.
	Key foundation.Key
	// OnError is called with each error that a widget below the boundary
	// reported while it was built (a panic in Build, in InitState,
	// DidUpdateWidget or DidChangeDependencies of a state below, in the builder
	// of a LayoutBuilder, in a lazily built child of a sliver), before the
	// error goes to the application's ErrorReporter.
	//
	// It runs on the UI thread, inside the build pass (or the layout pass), so
	// it must not call SetState on an ancestor of the boundary; it may post
	// work with the binding's Post. A panic in it is an error of the boundary's
	// own element: the next boundary above takes it, or the application's
	// ErrorReporter when there is none.
	//
	// Flutter: no counterpart
	OnError func(err error)
}

// ErrorBoundary takes over the build errors of its subtree. When a widget below
// it fails to build (a panic in Build, in the InitState, DidUpdateWidget or
// DidChangeDependencies of a state, in the builder of a LayoutBuilder, or in a
// lazily built child of a sliver such as the item of ListView.builder), the
// boundary is told first (ErrorBoundaryOpts.OnError), and it builds the fallback
// in place of its whole child, in the same frame: the failed subtree is unmounted
// (its states are disposed) and never reaches a painted frame. The application's
// ErrorReporter still gets every error, after the boundary.
//
// A failed boundary stays failed: a rebuild of it, or a new child from its
// parent, builds the fallback again, and the old child is not tried again. A new
// Key makes a new boundary, which is the way to try again. The fallback gets the
// first error; the errors that come after are only reported.
//
// Boundaries nest: the nearest one above the failing element takes the error. A
// panic in a fallback, or in anything the fallback builds, goes to the next
// boundary above, or to the application's ErrorReporter when there is none, as a
// panic in OnError does. Without a boundary nothing changes: the ErrorWidget
// takes the place of the widget that failed. Like the recovery points
// themselves (SPEC §14.2), a boundary acts only when the build owner has an
// error reporter, which the binding always sets.
//
// The boundary needs no Flutter counterpart in the framework: Flutter shows
// the ErrorWidget in the place of the failing widget only (SPEC §12.10).
//
// Flutter: no counterpart
type ErrorBoundary struct {
	WidgetBase
	fallback func(bc BuildContext, err error) Widget
	child    Widget
	onError  func(err error)
}

// NewErrorBoundary creates a boundary around the child. The fallback builds
// what is shown instead of the child once a widget below failed; it gets the
// first error. A nil fallback means the build owner's error widget
// (BuildOwner.ErrorWidgetBuilder, or ErrorWidget).
//
// Flutter: no counterpart
func NewErrorBoundary(fallback func(bc BuildContext, err error) Widget, child Widget, opts ...ErrorBoundaryOpts) *ErrorBoundary {
	o := firstOpts(opts)
	return &ErrorBoundary{WidgetBase: NewWidgetBase(WidgetOpts{Key: o.Key}), fallback: fallback, child: child, onError: o.OnError}
}

// Child is the widget below this one in the tree.
//
// Flutter: no counterpart
func (w *ErrorBoundary) Child() Widget { return w.child }

// CreateElement creates the element of the boundary.
//
// Flutter: no counterpart
func (w *ErrorBoundary) CreateElement() Element {
	e := &errorBoundaryElement{}
	InitComponentElement(&e.ComponentElement, e, w)
	return e
}

// errorBoundaryFallbackKey keeps the fallback from updating the failed child
// when both are widgets of the same type: the whole subtree is replaced.
type errorBoundaryFallbackKey struct{}

// errorBoundaryElement is the element of an ErrorBoundary.
type errorBoundaryElement struct {
	ComponentElement
	// failed is whether a widget below reported an error; failure is the first.
	failed  bool
	failure error
	// showsFallback is whether the last Build returned the fallback. While it
	// is true the boundary takes no error: what fails is the fallback or
	// something it built, and that goes to the boundary above.
	showsFallback bool
	// queued is whether the boundary is in BuildOwner.failedBoundaries.
	queued bool
}

// Mount mounts the element and builds the first child. The owner learns that it
// has a boundary, so that the binding settles failures between layout and paint.
func (e *errorBoundaryElement) Mount(parent Element, slot any) {
	e.ElementBase.Mount(parent, slot)
	e.owner.hasErrorBoundary = true
	e.cself.firstBuild()
}

// Build is the child, or the fallback once the boundary failed.
func (e *errorBoundaryElement) Build() Widget {
	w := e.widget.(*ErrorBoundary)
	if !e.failed {
		e.showsFallback = false
		return w.child
	}
	e.showsFallback = true
	if w.fallback == nil {
		return e.owner.errorWidgetFor(e.failure)
	}
	return NewKeyedSubtree(w.fallback(e, e.failure), WidgetOpts{Key: foundation.NewValueKey(errorBoundaryFallbackKey{})})
}

// Update replaces the widget and rebuilds.
func (e *errorBoundaryElement) Update(newWidget Widget) {
	e.ComponentElement.Update(newWidget)
	e.Rebuild(RebuildOpts{Force: true})
}

// take is the part of the boundary in an error of its subtree: it remembers the
// first error, tells OnError, and asks for the rebuild that shows the fallback.
func (e *errorBoundaryElement) take(err error) {
	if !e.failed {
		e.failed = true
		e.failure = err
	}
	e.owner.queueErrorBoundary(e)
	if onError := e.widget.(*ErrorBoundary).onError; onError != nil {
		var callbackErr error
		foundation.GuardLazy(func(err error) { callbackErr = err }, foundation.CallbackPanicked,
			func() string { return "while calling ErrorBoundary.OnError of " + e.String() }, e.panicInfo,
			func() { onError(err) })
		if callbackErr != nil {
			// The boundary cannot take the error of its own callback.
			e.owner.reportBuildError(e.parent, callbackErr)
		}
	}
}

// nearestErrorBoundary is the first boundary at or above the element that does
// not show its fallback, or nil. It walks the parent links: it registers no
// dependency, and it is not a lookup that the debug checks of
// VisitAncestorElements would refuse in the middle of a build.
func nearestErrorBoundary(from Element) *errorBoundaryElement {
	for e := from; e != nil; e = e.AsElement().parent {
		if b, ok := e.(*errorBoundaryElement); ok && !b.showsFallback {
			return b
		}
	}
	return nil
}

// reportBuildError reports an error of the element from, or of one of its
// descendants, that the framework recovered: the nearest ErrorBoundary at or above
// from is told first, and then the application's OnError. It returns whether a
// boundary took the error. The caller has checked that OnError is set.
func (o *BuildOwner) reportBuildError(from Element, err error) (taken bool) {
	b := nearestErrorBoundary(from)
	if b != nil {
		b.take(err)
	}
	o.OnError(err)
	return b != nil
}

// queueErrorBoundary adds a boundary that failed to the list that the binding
// rebuilds before the frame is laid out or painted.
func (o *BuildOwner) queueErrorBoundary(b *errorBoundaryElement) {
	if b.queued || b.showsFallback {
		return
	}
	b.queued = true
	o.failedBoundaries = append(o.failedBoundaries, b)
}

// rebuildErrorBoundaries rebuilds the boundaries that failed, so that each
// builds its fallback in place of its child. Each rebuild is a build scope of its
// own, so the checks of a build pass hold; the boundary is put into the dirty list
// directly, which is the way into a pass that MarkNeedsBuild does not give to an
// ancestor of the element being built. A boundary whose fallback fails sends the
// error to the boundary above, which joins the list, so the loop ends when no
// boundary is left. The binding calls it after the build pass and again after
// each layout, so that no failed subtree is laid out again or painted.
func (o *BuildOwner) rebuildErrorBoundaries() {
	for len(o.failedBoundaries) > 0 {
		b := o.failedBoundaries[0]
		o.failedBoundaries = o.failedBoundaries[1:]
		b.queued = false
		if b.lifecycle != lifecycleActive || b.showsFallback {
			continue
		}
		o.BuildScope(b, func() {
			if !b.dirty {
				b.dirty = true
				o.ScheduleBuildFor(b)
			}
		})
	}
}

// dropStaleChild deactivates the child that an update left half way, when a
// boundary took the error: the boundary unmounts the whole subtree, and the state
// of a child that nothing refers to any more would never be disposed. Without a
// boundary the child is left to the tree, as in Flutter.
func dropStaleChild(parent Element, child Element) {
	if child == nil || child.AsElement().lifecycle != lifecycleActive || child.AsElement().parent != parent {
		return
	}
	parent.DeactivateChild(child)
}
