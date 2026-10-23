package widgets

import (
	"fmt"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
)

// multiChildComponent is the widget behind ViewCollection and ViewAnchor: an
// optional child that is rendered into the surrounding view, and any number of
// views (the "view slot"), each of which starts a render tree of its own.
//
// Flutter: widgets/view.dart _MultiChildComponentWidget
type multiChildComponent struct {
	WidgetBase
	views []Widget
	child Widget
}

// component is the widget itself; the widgets that embed it have it too.
func (w *multiChildComponent) component() *multiChildComponent { return w }

// multiChildWidget is a multiChildComponent or a widget that embeds one.
type multiChildWidget interface {
	Widget
	component() *multiChildComponent
}

// CreateElement creates the element, which builds the child and the views.
//
// Flutter: widgets/view.dart _MultiChildComponentWidget.createElement
func (w *multiChildComponent) CreateElement() Element {
	e := &multiChildComponentElement{}
	InitElement(&e.ElementBase, e, w)
	return e
}

// CreateElement creates the element of the collection.
//
// Flutter: widgets/view.dart _MultiChildComponentWidget.createElement
func (w *ViewCollection) CreateElement() Element {
	e := &multiChildComponentElement{}
	InitElement(&e.ElementBase, e, w)
	return e
}

// ViewCollection is a collection of sibling Views. It can be used where a View is
// allowed, in a non-rendering zone of the widget tree: at the root of the tree
// outside of any View, as a child of another ViewCollection, or in the view slot of
// a ViewAnchor. Other widgets that are not render object widgets (such as an
// InheritedWidget) may appear in between. The views of the collection must be
// Views, wrapped or not in such widgets.
//
// Flutter: widgets/view.dart ViewCollection
type ViewCollection struct {
	multiChildComponent
}

// ViewCollectionOpts are the optional parameters of NewViewCollection.
//
// Flutter: widgets/view.dart ViewCollection.new
type ViewCollectionOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
}

// NewViewCollection creates a collection of the views.
//
// Flutter: widgets/view.dart ViewCollection.new
func NewViewCollection(views []Widget, opts ...ViewCollectionOpts) *ViewCollection {
	o := firstOpts(opts)
	return &ViewCollection{multiChildComponent{WidgetBase: NewWidgetBase(WidgetOpts(o)), views: views}}
}

// Views are the View descendants of the collection.
//
// Flutter: widgets/view.dart ViewCollection.views
func (w *ViewCollection) Views() []Widget { return w.views }

// ViewAnchor decorates a child with a side View. It needs a View above it, into
// which the child is rendered; the widget in the view slot (typically a View or
// a ViewCollection) starts new render trees, and has access to the inherited
// widgets above the anchor. A tooltip that extends beyond the bounds of the main
// view is an example: the tooltip is a View anchored to the button.
//
// Flutter: widgets/view.dart ViewAnchor
type ViewAnchor struct {
	StatelessWidget
	view  Widget
	child Widget
}

// ViewAnchorOpts are the optional parameters of NewViewAnchor.
//
// Flutter: widgets/view.dart ViewAnchor.new
type ViewAnchorOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// View is the widget that defines the view anchored to the child, typically a
	// View or a ViewCollection; nil for none.
	View Widget
}

// NewViewAnchor creates an anchor that renders the child into the surrounding
// view and the View option into a view of its own.
//
// Flutter: widgets/view.dart ViewAnchor.new
func NewViewAnchor(child Widget, opts ...ViewAnchorOpts) *ViewAnchor {
	o := firstOpts(opts)
	return &ViewAnchor{StatelessWidget: NewStatelessWidgetBase(WidgetOpts{Key: o.Key}), view: o.View, child: child}
}

// ViewSlot is the widget that defines the view anchored to the child.
//
// Flutter: widgets/view.dart ViewAnchor.view
func (w *ViewAnchor) ViewSlot() Widget { return w.view }

// Child is the widget rendered into the surrounding view.
//
// Flutter: widgets/view.dart ViewAnchor.child
func (w *ViewAnchor) Child() Widget { return w.child }

// Build makes the component that has the child and the view.
//
// Flutter: widgets/view.dart ViewAnchor.build
func (w *ViewAnchor) Build(BuildContext) Widget {
	var views []Widget
	if w.view != nil {
		views = []Widget{NewLookupBoundary(w.view)}
	}
	return &multiChildComponent{WidgetBase: NewWidgetBase(), views: views, child: w.child}
}

// viewSlotValue is the slot of the elements of the views of a
// multiChildComponentElement: they attach no render object to an ancestor.
type viewSlotValue struct{}

// viewSlot is the one slot of the view elements.
//
// Flutter: widgets/view.dart _MultiChildComponentElement._viewSlot
var viewSlot = &viewSlotValue{}

// multiChildComponentElement is the element of a multiChildComponent: its child
// is in the tree of the surrounding view, and each view in a tree of its own.
//
// Flutter: widgets/view.dart _MultiChildComponentElement
type multiChildComponentElement struct {
	ElementBase
	viewElements []Element
	forgotten    map[Element]struct{}
	childElement Element
}

func (e *multiChildComponentElement) component() *multiChildComponent {
	return e.widget.(multiChildWidget).component()
}

// AttachRenderObject attaches the child and checks that the slot may hold a
// render object.
//
// Flutter: widgets/view.dart _MultiChildComponentElement.attachRenderObject
func (e *multiChildComponentElement) AttachRenderObject(newSlot any) {
	e.ElementBase.AttachRenderObject(newSlot)
	e.debugCheckMustAttachRenderObject(newSlot)
}

// Mount mounts the element and builds the child and the views.
//
// Flutter: widgets/view.dart _MultiChildComponentElement.mount
func (e *multiChildComponentElement) Mount(parent Element, newSlot any) {
	e.ElementBase.Mount(parent, newSlot)
	e.debugCheckMustAttachRenderObject(newSlot)
	e.Rebuild()
}

// UpdateSlot moves the element to a new slot.
//
// Flutter: widgets/view.dart _MultiChildComponentElement.updateSlot
func (e *multiChildComponentElement) UpdateSlot(newSlot any) {
	e.ElementBase.UpdateSlot(newSlot)
	e.debugCheckMustAttachRenderObject(newSlot)
}

// debugCheckMustAttachRenderObject reports a view collection that a render
// object ancestor expects to give it a render object: the views make render trees
// of their own, which cannot be attached to the ancestor. It applies only to the
// ViewCollection form, the one with no child.
func (e *multiChildComponentElement) debugCheckMustAttachRenderObject(slot any) {
	if !foundation.DebugMode || e.component().child != nil {
		return
	}
	hasAncestorRenderObjectElement := false
	ancestorWantsRenderObject := true
	e.VisitAncestorElements(func(ancestor Element) bool {
		if !ancestor.DebugExpectsRenderObjectForSlot(slot) {
			ancestorWantsRenderObject = false
			return false
		}
		if _, ok := ancestor.(RenderObjectElementSelf); ok {
			hasAncestorRenderObjectElement = true
			return false
		}
		return true
	})
	if hasAncestorRenderObjectElement && ancestorWantsRenderObject {
		err := errors.New(tagViewCollectionInRenderTree,
			fmt.Sprintf("the Element for %s cannot be inserted into slot %v of its ancestor; this element allows the creation of multiple independent render trees, which cannot be attached to an ancestor in an existing render tree, yet an ancestor render object is expecting a child; try moving the subtree that contains the widget into the view property of a ViewAnchor widget or to the root of the widget tree, where it is not expected to attach its render object", foundation.ToStringShort(e.self), slot),
			"creatorChain", e.DebugGetCreatorChain(10))
		if e.owner != nil && e.owner.OnError != nil {
			e.owner.OnError(err)
			return
		}
		panic(err)
	}
}

// Update replaces the widget and rebuilds.
//
// Flutter: widgets/view.dart _MultiChildComponentElement.update
func (e *multiChildComponentElement) Update(newWidget Widget) {
	e.ElementBase.Update(newWidget)
	e.Rebuild(RebuildOpts{Force: true})
}

// DebugExpectsRenderObjectForSlot is false for the slot of the views.
//
// Flutter: widgets/view.dart _MultiChildComponentElement.debugExpectsRenderObjectForSlot
func (e *multiChildComponentElement) DebugExpectsRenderObjectForSlot(slot any) bool {
	return slot != any(viewSlot)
}

// PerformRebuild updates the child and the views.
//
// Flutter: widgets/view.dart _MultiChildComponentElement.performRebuild
func (e *multiChildComponentElement) PerformRebuild() {
	w := e.component()
	e.childElement = e.UpdateChild(e.childElement, w.child, e.slot)

	slots := make([]any, len(w.views))
	for i := range slots {
		slots[i] = viewSlot
	}
	e.viewElements = e.UpdateChildren(e.viewElements, w.views, UpdateChildrenOpts{ForgottenChildren: e.forgotten, Slots: slots})
	clear(e.forgotten)

	e.ElementBase.PerformRebuild() // clears the dirty flag
}

// ForgetChild drops the child, or marks the view element as one to leave out of
// the next update.
//
// Flutter: widgets/view.dart _MultiChildComponentElement.forgetChild
func (e *multiChildComponentElement) ForgetChild(child Element) {
	if child == e.childElement {
		e.childElement = nil
	} else {
		if e.forgotten == nil {
			e.forgotten = map[Element]struct{}{}
		}
		e.forgotten[child] = struct{}{}
	}
	e.ElementBase.ForgetChild(child)
}

// VisitChildren visits the child, then the views.
//
// Flutter: widgets/view.dart _MultiChildComponentElement.visitChildren
func (e *multiChildComponentElement) VisitChildren(visitor ElementVisitor) {
	if e.childElement != nil {
		visitor(e.childElement)
	}
	for _, child := range e.viewElements {
		if _, gone := e.forgotten[child]; !gone {
			visitor(child)
		}
	}
}

// DebugDoingBuild is false: this element has no build phase.
//
// Flutter: widgets/view.dart _MultiChildComponentElement.debugDoingBuild
func (e *multiChildComponentElement) DebugDoingBuild() bool { return false }

// RenderObjectAttachingChild is the child that is rendered into the surrounding
// view; the views are not attached to it.
//
// Flutter: widgets/view.dart _MultiChildComponentElement.renderObjectAttachingChild
func (e *multiChildComponentElement) RenderObjectAttachingChild() Element { return e.childElement }

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagViewCollectionInRenderTree = "ViewCollectionInRenderTree"
)
