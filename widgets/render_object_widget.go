package widgets

import (
	"slices"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
)

// RenderObjectWidget is the base of the widgets that have a render object.
// Embed LeafRenderObjectWidget, SingleChildRenderObjectWidget or
// MultiChildRenderObjectWidget, and implement CreateRenderObject. Optional
// methods are UpdateRenderObject(bc, renderObject) and
// DidUnmountRenderObject(renderObject).
//
// Flutter: widgets/framework.dart RenderObjectWidget
type RenderObjectWidget struct{ WidgetBase }

// NewRenderObjectWidgetBase creates the base of a render object widget.
//
// Flutter: widgets/framework.dart RenderObjectWidget.new
func NewRenderObjectWidgetBase(opts ...WidgetOpts) RenderObjectWidget {
	return RenderObjectWidget{NewWidgetBase(opts...)}
}

func (*RenderObjectWidget) isRenderObjectWidget() {}

// renderObjectWidget is any widget with a RenderObjectWidget base that creates
// its render object.
type renderObjectWidget interface {
	Widget
	// CreateRenderObject creates the render object for this widget at this
	// location. The context is the element that is being mounted.
	//
	// Flutter: widgets/framework.dart RenderObjectWidget.createRenderObject
	CreateRenderObject(bc BuildContext) rendering.RenderObject
	isRenderObjectWidget()
}

type renderObjectUpdater interface {
	// UpdateRenderObject copies the configuration of the widget to the render
	// object, when the widget is updated.
	//
	// Flutter: widgets/framework.dart RenderObjectWidget.updateRenderObject
	UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject)
}

type renderObjectUnmountListener interface {
	// DidUnmountRenderObject is called when the render object is removed from the
	// tree for good, before it is disposed.
	//
	// Flutter: widgets/framework.dart RenderObjectWidget.didUnmountRenderObject
	DidUnmountRenderObject(renderObject rendering.RenderObject)
}

// LeafRenderObjectWidget is a render object widget that has no children.
//
// Flutter: widgets/framework.dart LeafRenderObjectWidget
type LeafRenderObjectWidget struct{ RenderObjectWidget }

// NewLeafRenderObjectWidgetBase creates the base of a leaf render object
// widget.
//
// Flutter: widgets/framework.dart LeafRenderObjectWidget.new
func NewLeafRenderObjectWidgetBase(opts ...WidgetOpts) LeafRenderObjectWidget {
	return LeafRenderObjectWidget{NewRenderObjectWidgetBase(opts...)}
}

func (*LeafRenderObjectWidget) isLeafRenderObjectWidget() {}

// SingleChildRenderObjectWidget is a render object widget that has one optional
// child.
//
// Flutter: widgets/framework.dart SingleChildRenderObjectWidget
type SingleChildRenderObjectWidget struct {
	RenderObjectWidget
	child Widget
}

// NewSingleChildRenderObjectWidgetBase creates the base of a render object
// widget with a child, which may be nil.
//
// Flutter: widgets/framework.dart SingleChildRenderObjectWidget.new
func NewSingleChildRenderObjectWidgetBase(child Widget, opts ...WidgetOpts) SingleChildRenderObjectWidget {
	return SingleChildRenderObjectWidget{NewRenderObjectWidgetBase(opts...), child}
}

// Child is the widget below this one in the tree, or nil.
//
// Flutter: widgets/framework.dart SingleChildRenderObjectWidget.child
func (w *SingleChildRenderObjectWidget) Child() Widget { return w.child }

func (*SingleChildRenderObjectWidget) isSingleChildRenderObjectWidget() {}

// MultiChildRenderObjectWidget is a render object widget that has a list of
// children.
//
// Flutter: widgets/framework.dart MultiChildRenderObjectWidget
type MultiChildRenderObjectWidget struct {
	RenderObjectWidget
	children []Widget
}

// NewMultiChildRenderObjectWidgetBase creates the base of a render object
// widget with children.
//
// Flutter: widgets/framework.dart MultiChildRenderObjectWidget.new
func NewMultiChildRenderObjectWidgetBase(children []Widget, opts ...WidgetOpts) MultiChildRenderObjectWidget {
	return MultiChildRenderObjectWidget{NewRenderObjectWidgetBase(opts...), children}
}

// Children are the widgets below this one in the tree, in order.
//
// Flutter: widgets/framework.dart MultiChildRenderObjectWidget.children
func (w *MultiChildRenderObjectWidget) Children() []Widget { return w.children }

func (*MultiChildRenderObjectWidget) isMultiChildRenderObjectWidget() {}

type leafRenderObjectWidget interface {
	renderObjectWidget
	isLeafRenderObjectWidget()
}

type singleChildRenderObjectWidget interface {
	renderObjectWidget
	Child() Widget
	isSingleChildRenderObjectWidget()
}

type multiChildRenderObjectWidget interface {
	renderObjectWidget
	Children() []Widget
	isMultiChildRenderObjectWidget()
}

// DebugCreator is the creator of a render object, set on the render objects the
// elements create so that their description can say where they came from.
//
// Flutter: widgets/framework.dart DebugCreator
type DebugCreator struct {
	// Element is the element that created the render object.
	Element Element
}

// String is the chain of the creator's ancestors.
//
// Flutter: widgets/framework.dart DebugCreator.toString
func (c DebugCreator) String() string { return c.Element.AsElement().DebugGetCreatorChain(12) }

// RenderObjectElementSelf is what a render object element needs from the outer
// type: where to put the children's render objects.
type RenderObjectElementSelf interface {
	Element
	// InsertRenderObjectChild puts the render object of a child into this
	// element's render object, in the slot.
	//
	// Flutter: widgets/framework.dart RenderObjectElement.insertRenderObjectChild
	InsertRenderObjectChild(child rendering.RenderObject, slot any)
	// MoveRenderObjectChild moves the render object of a child to another slot.
	//
	// Flutter: widgets/framework.dart RenderObjectElement.moveRenderObjectChild
	MoveRenderObjectChild(child rendering.RenderObject, oldSlot, newSlot any)
	// RemoveRenderObjectChild takes the render object of a child out of this
	// element's render object.
	//
	// Flutter: widgets/framework.dart RenderObjectElement.removeRenderObjectChild
	RemoveRenderObjectChild(child rendering.RenderObject, slot any)
	asRenderObjectElement() *RenderObjectElement
}

// renderObjectElementLike is an element that has a render object.
type renderObjectElementLike interface {
	RenderObjectElementSelf
	updateParentData(w parentDataWidget)
}

// RenderObjectElement is the element of a RenderObjectWidget. It creates the
// render object and keeps it in sync with the widget, and it puts the render
// objects of its descendants into its own.
//
// Flutter: widgets/framework.dart RenderObjectElement
type RenderObjectElement struct {
	ElementBase
	rself                RenderObjectElementSelf
	renderObject         rendering.RenderObject
	debugDoingBuild      bool
	ancestorRenderObject RenderObjectElementSelf
}

// InitRenderObjectElement sets up a render object element; self is the outer
// element.
//
// Flutter: widgets/framework.dart RenderObjectElement.new
func InitRenderObjectElement(e *RenderObjectElement, self RenderObjectElementSelf, widget Widget) {
	InitElement(&e.ElementBase, self, widget)
	e.rself = self
}

func (e *RenderObjectElement) asRenderObjectElement() *RenderObjectElement { return e }

func (e *RenderObjectElement) ownRenderObject() (rendering.RenderObject, bool) {
	return e.renderObject, true
}

// DebugDoingBuild is whether the element is creating or updating its render
// object.
//
// Flutter: widgets/framework.dart RenderObjectElement.debugDoingBuild
func (e *RenderObjectElement) DebugDoingBuild() bool { return e.debugDoingBuild }

// RenderObjectAttachingChild is nil: a render object element holds the render
// objects of its subtree itself.
//
// Flutter: widgets/framework.dart RenderObjectElement.renderObjectAttachingChild
func (e *RenderObjectElement) RenderObjectAttachingChild() Element { return nil }

func (e *RenderObjectElement) findAncestorRenderObjectElement() RenderObjectElementSelf {
	ancestor := e.parent
	for ancestor != nil {
		if r, ok := ancestor.(RenderObjectElementSelf); ok {
			if foundation.DebugMode && !ancestor.DebugExpectsRenderObjectForSlot(e.slot) {
				return nil
			}
			return r
		}
		// In debug builds the ancestor is asked whether it accepts render objects,
		// for a better error message; release builds assume correct trees.
		if foundation.DebugMode && !ancestor.DebugExpectsRenderObjectForSlot(e.slot) {
			return nil
		}
		ancestor = ancestor.AsElement().parent
	}
	return nil
}

func (e *RenderObjectElement) findAncestorParentDataElements() []*ParentDataElement {
	var result []*ParentDataElement
	var parentDataTypes orderedSet[string]
	var culprits []string
	ancestor := e.parent
	// More than one ParentDataWidget can contribute parent data, but each must
	// be of a different widget type and write a different parent data type, and
	// the parent data must be compatible with all of them (checked in
	// UpdateParentData).
	for ancestor != nil {
		if _, ok := ancestor.(RenderObjectElementSelf); ok {
			break
		}
		if pde, ok := ancestor.(*ParentDataElement); ok {
			if foundation.DebugMode {
				// Flutter's ParentDataElement<T> is a different element type for
				// each T, so a repeated element type and a repeated parent data
				// type are the same thing here.
				if !parentDataTypes.add(pde.DebugParentDataType()) {
					culprits = append(culprits, pde.DebugParentDataType())
				}
			}
			result = append(result, pde)
		}
		ancestor = ancestor.AsElement().parent
	}
	if foundation.DebugMode && len(result) > 0 && ancestor != nil {
		e.debugCheckCompetingAncestors(result, culprits)
	}
	return result
}

func (e *RenderObjectElement) debugCheckCompetingAncestors(result []*ParentDataElement, culprits []string) {
	if len(culprits) == 0 {
		return
	}
	var msg strings.Builder
	msg.WriteString("Incorrect use of ParentDataWidget.\nCompeting ParentDataWidgets are providing parent data to the same RenderObject:")
	for _, a := range result {
		if !slices.Contains(culprits, a.DebugParentDataType()) {
			continue
		}
		msg.WriteString("\n- " + widgetString(a.widget) + ", which writes ParentData of type " + a.DebugParentDataType() +
			", (typically placed directly inside a " + typicalAncestor(a.widget) + " widget)")
	}
	msg.WriteString("\nA RenderObject can receive parent data from multiple ParentDataWidgets, but the Type of ParentData must be unique to prevent one overwriting another." +
		"\nUsually, this indicates that one or more of the offending ParentDataWidgets listed above isn't placed inside a dedicated compatible ancestor widget that it isn't sharing with another ParentDataWidget of the same type." +
		"\nOtherwise, separating aspects of ParentData to prevent conflicts can be done using mixins, mixing them all in on the full ParentData Object, such as KeepAlive does with KeepAliveParentDataMixin." +
		"\nThe ownership chain for the RenderObject that received the parent data was:\n  " + e.DebugGetCreatorChain(10))
	e.owner.reportError(errors.New("ParentDataWidgetsCompeting", msg.String()))
}

// widgetString describes a widget the way Flutter's Widget.toString does: its
// type and key, and its properties in parentheses.
func widgetString(w Widget) string {
	return foundation.DiagnosticableString(&widgetDescription{w})
}

// typicalAncestor is the name of the widget a parent data widget is typically
// placed in, for messages; the optional method DebugTypicalAncestorWidgetClass
// gives it.
func typicalAncestor(w Widget) string {
	if t, ok := w.(interface{ DebugTypicalAncestorWidgetClass() string }); ok {
		if d, ok := w.(interface{ DebugTypicalAncestorWidgetDescription() string }); ok {
			return d.DebugTypicalAncestorWidgetDescription()
		}
		return t.DebugTypicalAncestorWidgetClass()
	}
	return "its typical ancestor"
}

// Mount creates the render object and attaches it to the render tree.
//
// Flutter: widgets/framework.dart RenderObjectElement.mount
func (e *RenderObjectElement) Mount(parent Element, newSlot any) {
	e.ElementBase.Mount(parent, newSlot)
	e.debugDoingBuild = true
	e.renderObject = e.widget.(renderObjectWidget).CreateRenderObject(e.rself)
	e.debugDoingBuild = false
	e.debugUpdateRenderObjectOwner()
	e.rself.AttachRenderObject(newSlot)
	e.ElementBase.PerformRebuild() // clears the dirty flag
}

// Update replaces the widget and updates the render object from it.
//
// Flutter: widgets/framework.dart RenderObjectElement.update
func (e *RenderObjectElement) Update(newWidget Widget) {
	e.ElementBase.Update(newWidget)
	e.debugUpdateRenderObjectOwner()
	e.updateRenderObject()
}

func (e *RenderObjectElement) debugUpdateRenderObjectOwner() {
	if foundation.DebugMode {
		e.renderObject.AsRenderObject().DebugCreator = DebugCreator{e.self}
	}
}

// PerformRebuild updates the render object from the widget.
//
// Flutter: widgets/framework.dart RenderObjectElement.performRebuild
func (e *RenderObjectElement) PerformRebuild() { e.updateRenderObject() }

func (e *RenderObjectElement) updateRenderObject() {
	e.debugDoingBuild = true
	if u, ok := e.widget.(renderObjectUpdater); ok {
		u.UpdateRenderObject(e.rself, e.renderObject)
	}
	e.debugDoingBuild = false
	e.ElementBase.PerformRebuild() // clears the dirty flag
}

// Unmount tells the widget, then disposes the render object.
//
// Flutter: widgets/framework.dart RenderObjectElement.unmount
func (e *RenderObjectElement) Unmount() {
	oldWidget := e.widget
	e.ElementBase.Unmount()
	if l, ok := oldWidget.(renderObjectUnmountListener); ok {
		l.DidUnmountRenderObject(e.renderObject)
	}
	e.renderObject.Dispose()
	e.renderObject = nil
}

// DebugFillProperties adds the render object.
//
// Flutter: widgets/framework.dart RenderObjectElement.debugFillProperties
func (e *RenderObjectElement) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	e.ElementBase.DebugFillProperties(properties)
	properties.Add(foundation.NewDiagnosticsProperty("renderObject", e.renderObject,
		foundation.DiagnosticsPropertyOpts{DefaultValue: foundation.DefaultNull}))
}

func (e *RenderObjectElement) updateParentData(w parentDataWidget) {
	apply := true
	if foundation.DebugMode && !w.DebugIsValidRenderObject(e.renderObject) {
		apply = false
		name := foundation.RuntimeTypeName(w)
		var msg strings.Builder
		msg.WriteString("Incorrect use of ParentDataWidget.\nThe ParentDataWidget " + widgetString(w) + " wants to apply ParentData of type " +
			w.DebugParentDataType() + " to a RenderObject, ")
		if data := e.renderObject.ParentData(); data == nil {
			msg.WriteString("which has not been set up to receive any ParentData.")
		} else {
			msg.WriteString("which has been set up to accept ParentData of incompatible type " + foundation.RuntimeTypeName(data) + ".")
		}
		msg.WriteString("\nUsually, this means that the " + name + " widget has the wrong ancestor RenderObjectWidget. Typically, " +
			name + " widgets are placed directly inside " + typicalAncestor(w) + " widgets.")
		if e.ancestorRenderObject != nil {
			msg.WriteString("\nThe offending " + name + " is currently placed inside a " +
				foundation.RuntimeTypeName(e.ancestorRenderObject.AsElement().widget) + " widget.")
		}
		msg.WriteString("\nThe ownership chain for the RenderObject that received the incompatible parent data was:\n  " + e.DebugGetCreatorChain(10))
		e.owner.reportError(errors.New("ParentDataWidgetIncorrectUse", msg.String()))
	}
	if apply {
		w.ApplyParentData(e.renderObject)
	}
}

// UpdateSlot moves the render object to the new slot of the ancestor's render
// object.
//
// Flutter: widgets/framework.dart RenderObjectElement.updateSlot
func (e *RenderObjectElement) UpdateSlot(newSlot any) {
	oldSlot := e.slot
	e.ElementBase.UpdateSlot(newSlot)
	if e.ancestorRenderObject != nil {
		e.ancestorRenderObject.MoveRenderObjectChild(e.renderObject, oldSlot, e.slot)
	}
}

// AttachRenderObject inserts the render object into the closest ancestor that
// has one and applies the parent data of the widgets in between. In debug
// builds an element without such an ancestor reports the error
// RenderObjectAncestorMissing.
//
// Flutter: widgets/framework.dart RenderObjectElement.attachRenderObject
func (e *RenderObjectElement) AttachRenderObject(newSlot any) {
	e.slot = newSlot
	e.ancestorRenderObject = e.findAncestorRenderObjectElement()
	if foundation.DebugMode && e.ancestorRenderObject == nil {
		e.owner.reportError(errors.New("RenderObjectAncestorMissing",
			"the render object for "+e.ToStringShort()+" cannot find an ancestor render object to attach to; try wrapping your widget in a View widget or any other widget that is backed by a render tree root element to serve as the root of the render tree",
			"ownership chain", e.DebugGetCreatorChain(10)))
	}
	if e.ancestorRenderObject != nil {
		e.ancestorRenderObject.InsertRenderObjectChild(e.renderObject, newSlot)
	}
	for _, pde := range e.findAncestorParentDataElements() {
		e.updateParentData(pde.widget.(parentDataWidget))
	}
}

// DetachRenderObject takes the render object out of the ancestor's.
//
// Flutter: widgets/framework.dart RenderObjectElement.detachRenderObject
func (e *RenderObjectElement) DetachRenderObject() {
	if e.ancestorRenderObject != nil {
		e.ancestorRenderObject.RemoveRenderObjectChild(e.renderObject, e.slot)
		e.ancestorRenderObject = nil
	}
	e.slot = nil
}

// LeafRenderObjectElement is the element of a LeafRenderObjectWidget.
//
// Flutter: widgets/framework.dart LeafRenderObjectElement
type LeafRenderObjectElement struct {
	RenderObjectElement
}

// NewLeafRenderObjectElement creates the element of a leaf render object
// widget.
//
// Flutter: widgets/framework.dart LeafRenderObjectElement.new
func NewLeafRenderObjectElement(w leafRenderObjectWidget) *LeafRenderObjectElement {
	e := &LeafRenderObjectElement{}
	InitRenderObjectElement(&e.RenderObjectElement, e, w)
	return e
}

func leafHasNoChildren(op string) error {
	return errors.New("LeafRenderObjectHasNoChildren", "a leaf render object element has no children", "operation", op)
}

// InsertRenderObjectChild panics: a leaf has no children.
//
// Flutter: widgets/framework.dart LeafRenderObjectElement.insertRenderObjectChild
func (e *LeafRenderObjectElement) InsertRenderObjectChild(rendering.RenderObject, any) {
	panic(leafHasNoChildren("insert"))
}

// MoveRenderObjectChild panics: a leaf has no children.
//
// Flutter: widgets/framework.dart LeafRenderObjectElement.moveRenderObjectChild
func (e *LeafRenderObjectElement) MoveRenderObjectChild(rendering.RenderObject, any, any) {
	panic(leafHasNoChildren("move"))
}

// RemoveRenderObjectChild panics: a leaf has no children.
//
// Flutter: widgets/framework.dart LeafRenderObjectElement.removeRenderObjectChild
func (e *LeafRenderObjectElement) RemoveRenderObjectChild(rendering.RenderObject, any) {
	panic(leafHasNoChildren("remove"))
}

// ForgetChild panics: a leaf has no children.
//
// Flutter: widgets/framework.dart LeafRenderObjectElement.forgetChild
func (e *LeafRenderObjectElement) ForgetChild(Element) { panic(leafHasNoChildren("forget")) }

// SingleChildRenderObjectElement is the element of a
// SingleChildRenderObjectWidget.
//
// Flutter: widgets/framework.dart SingleChildRenderObjectElement
type SingleChildRenderObjectElement struct {
	RenderObjectElement
	child Element
}

// NewSingleChildRenderObjectElement creates the element of a render object
// widget with one child.
//
// Flutter: widgets/framework.dart SingleChildRenderObjectElement.new
func NewSingleChildRenderObjectElement(w singleChildRenderObjectWidget) *SingleChildRenderObjectElement {
	e := &SingleChildRenderObjectElement{}
	InitRenderObjectElement(&e.RenderObjectElement, e, w)
	return e
}

// VisitChildren visits the child.
//
// Flutter: widgets/framework.dart SingleChildRenderObjectElement.visitChildren
func (e *SingleChildRenderObjectElement) VisitChildren(visitor ElementVisitor) {
	if e.child != nil {
		visitor(e.child)
	}
}

// ForgetChild drops the child.
//
// Flutter: widgets/framework.dart SingleChildRenderObjectElement.forgetChild
func (e *SingleChildRenderObjectElement) ForgetChild(child Element) {
	e.child = nil
	e.ElementBase.ForgetChild(child)
}

// Mount creates the render object and the child.
//
// Flutter: widgets/framework.dart SingleChildRenderObjectElement.mount
func (e *SingleChildRenderObjectElement) Mount(parent Element, newSlot any) {
	e.RenderObjectElement.Mount(parent, newSlot)
	e.child = e.UpdateChild(e.child, e.widget.(singleChildRenderObjectWidget).Child(), nil)
}

// Update updates the render object and the child.
//
// Flutter: widgets/framework.dart SingleChildRenderObjectElement.update
func (e *SingleChildRenderObjectElement) Update(newWidget Widget) {
	e.RenderObjectElement.Update(newWidget)
	e.child = e.UpdateChild(e.child, e.widget.(singleChildRenderObjectWidget).Child(), nil)
}

// singleChildHolder is a render object that holds one child of any type.
type singleChildHolder interface {
	ChildObject() rendering.RenderObject
	SetChildObject(child rendering.RenderObject)
}

// InsertRenderObjectChild sets the child of the render object.
//
// Flutter: widgets/framework.dart SingleChildRenderObjectElement.insertRenderObjectChild
func (e *SingleChildRenderObjectElement) InsertRenderObjectChild(child rendering.RenderObject, _ any) {
	e.renderObject.(singleChildHolder).SetChildObject(child)
}

// MoveRenderObjectChild panics: a single child has no other slot.
//
// Flutter: widgets/framework.dart SingleChildRenderObjectElement.moveRenderObjectChild
func (e *SingleChildRenderObjectElement) MoveRenderObjectChild(rendering.RenderObject, any, any) {
	panic(errors.New("SingleChildRenderObjectMove", "a render object with a single child has no slot to move the child to"))
}

// RemoveRenderObjectChild clears the child of the render object.
//
// Flutter: widgets/framework.dart SingleChildRenderObjectElement.removeRenderObjectChild
func (e *SingleChildRenderObjectElement) RemoveRenderObjectChild(rendering.RenderObject, any) {
	e.renderObject.(singleChildHolder).SetChildObject(nil)
}

// MultiChildRenderObjectElement is the element of a
// MultiChildRenderObjectWidget.
//
// Flutter: widgets/framework.dart MultiChildRenderObjectElement
type MultiChildRenderObjectElement struct {
	RenderObjectElement
	children  []Element
	forgotten map[Element]struct{}
}

// NewMultiChildRenderObjectElement creates the element of a render object widget
// with children. In debug builds it panics with the tag
// ChildrenHaveDuplicateKeys when two children have the same key.
//
// Flutter: widgets/framework.dart MultiChildRenderObjectElement.new
func NewMultiChildRenderObjectElement(w multiChildRenderObjectWidget) *MultiChildRenderObjectElement {
	e := &MultiChildRenderObjectElement{}
	InitMultiChildRenderObjectElement(e, e, w)
	return e
}

// InitMultiChildRenderObjectElement sets up an element that embeds
// MultiChildRenderObjectElement, with the outer element as self.
//
// Flutter: widgets/framework.dart MultiChildRenderObjectElement.new
func InitMultiChildRenderObjectElement(e *MultiChildRenderObjectElement, self RenderObjectElementSelf, w multiChildRenderObjectWidget) {
	DebugChildrenHaveDuplicateKeys(w, w.Children())
	e.forgotten = map[Element]struct{}{}
	InitRenderObjectElement(&e.RenderObjectElement, self, w)
}

// Children are the child elements, except the ones that were forgotten.
//
// Flutter: widgets/framework.dart MultiChildRenderObjectElement.children
func (e *MultiChildRenderObjectElement) Children() []Element {
	var out []Element
	for _, c := range e.children {
		if _, ok := e.forgotten[c]; !ok {
			out = append(out, c)
		}
	}
	return out
}

type containerHolder interface {
	InsertObject(child, after rendering.RenderObject)
	MoveObject(child, after rendering.RenderObject)
	RemoveObject(child rendering.RenderObject)
}

func slotSibling(slot any) rendering.RenderObject {
	s, ok := slot.(IndexedSlot)
	if !ok || s.Value == nil {
		return nil
	}
	return s.Value.RenderObject()
}

// InsertRenderObjectChild inserts the child after the render object of the
// previous sibling.
//
// Flutter: widgets/framework.dart MultiChildRenderObjectElement.insertRenderObjectChild
func (e *MultiChildRenderObjectElement) InsertRenderObjectChild(child rendering.RenderObject, slot any) {
	e.renderObject.(containerHolder).InsertObject(child, slotSibling(slot))
}

// MoveRenderObjectChild moves the child after the render object of the new
// previous sibling.
//
// Flutter: widgets/framework.dart MultiChildRenderObjectElement.moveRenderObjectChild
func (e *MultiChildRenderObjectElement) MoveRenderObjectChild(child rendering.RenderObject, _, newSlot any) {
	e.renderObject.(containerHolder).MoveObject(child, slotSibling(newSlot))
}

// RemoveRenderObjectChild removes the child.
//
// Flutter: widgets/framework.dart MultiChildRenderObjectElement.removeRenderObjectChild
func (e *MultiChildRenderObjectElement) RemoveRenderObjectChild(child rendering.RenderObject, _ any) {
	e.renderObject.(containerHolder).RemoveObject(child)
}

// VisitChildren visits the children that were not forgotten.
//
// Flutter: widgets/framework.dart MultiChildRenderObjectElement.visitChildren
func (e *MultiChildRenderObjectElement) VisitChildren(visitor ElementVisitor) {
	for _, c := range e.children {
		if _, ok := e.forgotten[c]; !ok {
			visitor(c)
		}
	}
}

// ForgetChild remembers that the child went to another parent.
//
// Flutter: widgets/framework.dart MultiChildRenderObjectElement.forgetChild
func (e *MultiChildRenderObjectElement) ForgetChild(child Element) {
	e.forgotten[child] = struct{}{}
	e.ElementBase.ForgetChild(child)
}

// InflateWidget inflates the child, and in debug builds reports the error
// MultiChildNeedsRenderObject when it has no render object.
//
// Flutter: widgets/framework.dart MultiChildRenderObjectElement.inflateWidget
func (e *MultiChildRenderObjectElement) InflateWidget(newWidget Widget, newSlot any) Element {
	newChild := e.ElementBase.InflateWidget(newWidget, newSlot)
	if foundation.DebugMode && newChild.RenderObject() == nil {
		e.owner.reportError(errors.New("MultiChildNeedsRenderObject",
			"the children of a MultiChildRenderObjectElement must each have an associated render object; this typically means that the widget or its children are not render object widgets",
			"widget", foundation.RuntimeTypeName(newChild.AsElement().widget)))
	}
	return newChild
}

// Mount creates the render object and inflates the children.
//
// Flutter: widgets/framework.dart MultiChildRenderObjectElement.mount
func (e *MultiChildRenderObjectElement) Mount(parent Element, newSlot any) {
	e.RenderObjectElement.Mount(parent, newSlot)
	widgets := e.widget.(multiChildRenderObjectWidget).Children()
	children := make([]Element, len(widgets))
	var previous Element
	for i, w := range widgets {
		child := e.self.InflateWidget(w, IndexedSlot{Index: i, Value: previous})
		children[i] = child
		previous = child
	}
	e.children = children
}

// Update updates the render object and the children.
//
// Flutter: widgets/framework.dart MultiChildRenderObjectElement.update
func (e *MultiChildRenderObjectElement) Update(newWidget Widget) {
	e.RenderObjectElement.Update(newWidget)
	w := e.widget.(multiChildRenderObjectWidget)
	DebugChildrenHaveDuplicateKeys(w, w.Children())
	e.children = e.UpdateChildren(e.children, w.Children(), UpdateChildrenOpts{ForgottenChildren: e.forgotten})
	clear(e.forgotten)
}

// RenderTreeRootElement is the element of a render object that is the root of a
// render tree of its own, such as a view's: it does not attach itself to an
// ancestor's render object.
//
// Flutter: widgets/framework.dart RenderTreeRootElement
type RenderTreeRootElement struct {
	RenderObjectElement
}

// InitRenderTreeRootElement sets up a render tree root element.
//
// Flutter: widgets/framework.dart RenderTreeRootElement.new
func InitRenderTreeRootElement(e *RenderTreeRootElement, self RenderObjectElementSelf, widget Widget) {
	InitRenderObjectElement(&e.RenderObjectElement, self, widget)
}

// AttachRenderObject only sets the slot.
//
// Flutter: widgets/framework.dart RenderTreeRootElement.attachRenderObject
func (e *RenderTreeRootElement) AttachRenderObject(newSlot any) {
	e.slot = newSlot
	e.debugCheckMustNotAttachRenderObjectToAncestor()
}

// DetachRenderObject only clears the slot.
//
// Flutter: widgets/framework.dart RenderTreeRootElement.detachRenderObject
func (e *RenderTreeRootElement) DetachRenderObject() { e.slot = nil }

// UpdateSlot sets the slot and checks, in debug builds, that there is still no
// ancestor to attach to.
//
// Flutter: widgets/framework.dart RenderTreeRootElement.updateSlot
func (e *RenderTreeRootElement) UpdateSlot(newSlot any) {
	e.RenderObjectElement.UpdateSlot(newSlot)
	e.debugCheckMustNotAttachRenderObjectToAncestor()
}

func (e *RenderTreeRootElement) debugCheckMustNotAttachRenderObjectToAncestor() {
	if foundation.DebugMode && e.findAncestorRenderObjectElement() != nil {
		panic(errors.New("RenderTreeRootHasAncestor",
			"the RenderObject for "+e.ToStringShort()+" cannot maintain an independent render tree at its current location; this RenderObject is the root of an independent render tree and it cannot attach itself to an ancestor in an existing tree, but the ancestor RenderObject expects that a child will be attached; try moving the subtree that contains the widget to a location where it is not expected to attach its RenderObject to a parent",
			"ownership chain", e.DebugGetCreatorChain(10)))
	}
}

// SlottedRenderObjectWidget is a render object widget whose children are in
// named slots, such as the parts of an input decorator. Embed
// SlottedMultiChildRenderObjectWidget and implement Slots, ChildForSlot,
// CreateRenderObject and UpdateRenderObject; the render object embeds
// rendering.SlottedContainer and implements SetSlotChild, MoveSlotChild and
// SlotChildObject.
//
// Flutter: widgets/slotted_render_object_widget.dart SlottedMultiChildRenderObjectWidget
type SlottedRenderObjectWidget interface {
	renderObjectWidget
	// Slots are the slots of the widget; they must not change.
	//
	// Flutter: widgets/slotted_render_object_widget.dart SlottedMultiChildRenderObjectWidgetMixin.slots
	Slots() []any
	// ChildForSlot is the widget in the slot, or nil.
	//
	// Flutter: widgets/slotted_render_object_widget.dart SlottedMultiChildRenderObjectWidgetMixin.childForSlot
	ChildForSlot(slot any) Widget
}

// SlottedMultiChildRenderObjectWidget is the base of a render object widget
// whose children are in slots.
//
// Flutter: widgets/slotted_render_object_widget.dart SlottedMultiChildRenderObjectWidget
type SlottedMultiChildRenderObjectWidget struct{ RenderObjectWidget }

// NewSlottedMultiChildRenderObjectWidgetBase creates the base of a render
// object widget with slots.
//
// Flutter: widgets/slotted_render_object_widget.dart SlottedMultiChildRenderObjectWidget.new
func NewSlottedMultiChildRenderObjectWidgetBase(opts ...WidgetOpts) SlottedMultiChildRenderObjectWidget {
	return SlottedMultiChildRenderObjectWidget{NewRenderObjectWidgetBase(opts...)}
}

// slottedHolder is a render object that holds children in slots.
type slottedHolder interface {
	SlotChild(slot any) rendering.RenderObject
	SetSlotChild(child rendering.RenderObject, slot any)
	MoveSlotChild(child rendering.RenderObject, slot, oldSlot any)
}

// SlottedRenderObjectElement is the element of a SlottedMultiChildRenderObjectWidget.
//
// Flutter: widgets/slotted_render_object_widget.dart SlottedRenderObjectElement
type SlottedRenderObjectElement struct {
	RenderObjectElement
	slotToChild   map[any]Element
	keyedChildren map[foundation.Key]Element
	previousSlots []any
}

// NewSlottedRenderObjectElement creates the element of a render object widget
// with slots.
//
// Flutter: widgets/slotted_render_object_widget.dart SlottedRenderObjectElement.new
func NewSlottedRenderObjectElement(w SlottedRenderObjectWidget) *SlottedRenderObjectElement {
	e := &SlottedRenderObjectElement{slotToChild: map[any]Element{}, keyedChildren: map[foundation.Key]Element{}}
	InitRenderObjectElement(&e.RenderObjectElement, e, w)
	return e
}

// VisitChildren visits the children.
//
// Flutter: widgets/slotted_render_object_widget.dart SlottedRenderObjectElement.visitChildren
func (e *SlottedRenderObjectElement) VisitChildren(visitor ElementVisitor) {
	for _, slot := range e.widget.(SlottedRenderObjectWidget).Slots() {
		if c, ok := e.slotToChild[slot]; ok {
			visitor(c)
		}
	}
}

// ForgetChild forgets the child, which went to another parent.
//
// Flutter: widgets/slotted_render_object_widget.dart SlottedRenderObjectElement.forgetChild
func (e *SlottedRenderObjectElement) ForgetChild(child Element) {
	delete(e.slotToChild, child.Slot())
	e.ElementBase.ForgetChild(child)
}

// Mount creates the render object and the children.
//
// Flutter: widgets/slotted_render_object_widget.dart SlottedRenderObjectElement.mount
func (e *SlottedRenderObjectElement) Mount(parent Element, newSlot any) {
	e.RenderObjectElement.Mount(parent, newSlot)
	e.updateChildren()
}

// Update updates the render object and the children.
//
// Flutter: widgets/slotted_render_object_widget.dart SlottedRenderObjectElement.update
func (e *SlottedRenderObjectElement) Update(newWidget Widget) {
	e.RenderObjectElement.Update(newWidget)
	e.updateChildren()
}

func (e *SlottedRenderObjectElement) updateChildren() {
	w := e.widget.(SlottedRenderObjectWidget)
	slots := w.Slots()
	if foundation.DebugMode {
		if e.previousSlots == nil {
			e.previousSlots = append([]any(nil), slots...)
		}
		if !slices.Equal(e.previousSlots, slots) {
			panic(errors.New("SlottedSlotsChanged", "the slots of a slotted render object widget must not change",
				"widget", foundation.RuntimeTypeName(w)))
		}
		seen := map[any]struct{}{}
		for _, s := range slots {
			if _, dup := seen[s]; dup {
				panic(errors.New("SlottedSlotsNotUnique", "the slots of a slotted render object widget must be unique",
					"widget", foundation.RuntimeTypeName(w), "slot", s))
			}
			seen[s] = struct{}{}
		}
	}
	oldKeyed := e.keyedChildren
	e.keyedChildren = map[foundation.Key]Element{}
	oldSlotToChild := e.slotToChild
	e.slotToChild = map[any]Element{}
	var duplicateKey foundation.Key
	for _, slot := range slots {
		child := w.ChildForSlot(slot)
		var newKey foundation.Key
		if child != nil {
			newKey = child.Key()
		}
		oldSlotChild := oldSlotToChild[slot]
		var oldKeyChild Element
		if newKey != nil {
			oldKeyChild = oldKeyed[newKey]
		}
		// Try to find the element that the widget should update. If key matching
		// fails, resort to the child of the same slot.
		var from Element
		switch {
		case oldKeyChild != nil:
			from = oldSlotToChild[oldKeyChild.Slot()]
			delete(oldSlotToChild, oldKeyChild.Slot())
		case oldSlotChild == nil || oldSlotChild.AsElement().widget.Key() == nil:
			from = oldSlotChild
			delete(oldSlotToChild, slot)
		}
		newChild := e.UpdateChild(from, child, slot)
		if newChild != nil {
			e.slotToChild[slot] = newChild
			if newKey != nil {
				if _, dup := e.keyedChildren[newKey]; dup && foundation.DebugMode {
					duplicateKey = newKey
				}
				e.keyedChildren[newKey] = newChild
			}
		}
	}
	for _, c := range oldSlotToChild {
		e.DeactivateChild(c)
	}
	if foundation.DebugMode && duplicateKey != nil {
		panic(errors.New("SlottedDuplicateKeys", "multiple widgets used the same key in a slotted render object widget; a key can only be specified on one widget at a time in the same parent widget",
			"widget", foundation.RuntimeTypeName(w), "key", duplicateKey.String()))
	}
}

// InsertRenderObjectChild puts the child in the slot of the render object.
//
// Flutter: widgets/slotted_render_object_widget.dart SlottedRenderObjectElement.insertRenderObjectChild
func (e *SlottedRenderObjectElement) InsertRenderObjectChild(child rendering.RenderObject, slot any) {
	e.renderObject.(slottedHolder).SetSlotChild(child, slot)
}

// RemoveRenderObjectChild takes the child out of the slot of the render object.
//
// Flutter: widgets/slotted_render_object_widget.dart SlottedRenderObjectElement.removeRenderObjectChild
func (e *SlottedRenderObjectElement) RemoveRenderObjectChild(child rendering.RenderObject, slot any) {
	holder := e.renderObject.(slottedHolder)
	if holder.SlotChild(slot) == child {
		holder.SetSlotChild(nil, slot)
	}
}

// MoveRenderObjectChild moves the child to another slot of the render object.
//
// Flutter: widgets/slotted_render_object_widget.dart SlottedRenderObjectElement.moveRenderObjectChild
func (e *SlottedRenderObjectElement) MoveRenderObjectChild(child rendering.RenderObject, oldSlot, newSlot any) {
	e.renderObject.(slottedHolder).MoveSlotChild(child, newSlot, oldSlot)
}
