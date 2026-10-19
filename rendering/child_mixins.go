package rendering

import (
	"strconv"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
)

// RenderObjectWithChild holds one child of type C for a render object: embed it
// and call InitRenderObjectWithChild in the constructor. It adopts and drops the
// child, and takes part in attach, detach, redepth, visiting and diagnostics.
//
// Flutter: rendering/object.dart RenderObjectWithChildMixin
type RenderObjectWithChild[C RenderObject] struct {
	host  RenderObject
	child C
}

// InitRenderObjectWithChild registers the mixin with its render object.
//
// Flutter: no counterpart (the self pattern, SPEC §12.5)
func InitRenderObjectWithChild[C RenderObject](m *RenderObjectWithChild[C], host RenderObject) {
	m.host = host
	host.AsRenderObject().setChildModel(m)
}

// Child is the child, or the zero value (nil).
//
// Flutter: rendering/object.dart RenderObjectWithChildMixin.child
func (m *RenderObjectWithChild[C]) Child() C { return m.child }

// SetChild replaces the child: the old one is dropped, the new one adopted. Pass
// the zero value (nil) for none.
//
// Flutter: rendering/object.dart RenderObjectWithChildMixin.child
func (m *RenderObjectWithChild[C]) SetChild(value C) {
	if !isNil(m.child) {
		m.host.AsRenderObject().DropChild(m.child)
	}
	m.child = value
	if !isNil(m.child) {
		m.host.AsRenderObject().AdoptChild(m.child)
	}
}

func (m *RenderObjectWithChild[C]) attachChildren(owner *PipelineOwner) {
	if !isNil(m.child) {
		m.child.Attach(owner)
	}
}

func (m *RenderObjectWithChild[C]) detachChildren() {
	if !isNil(m.child) {
		m.child.Detach()
	}
}

func (m *RenderObjectWithChild[C]) redepthChildren() {
	if !isNil(m.child) {
		m.host.AsRenderObject().RedepthChild(m.child)
	}
}

func (m *RenderObjectWithChild[C]) visitChildren(visitor RenderObjectVisitor) {
	if !isNil(m.child) {
		visitor(m.child)
	}
}

func (m *RenderObjectWithChild[C]) describeChildren() []foundation.DiagnosticsNode {
	if isNil(m.child) {
		return nil
	}
	return []foundation.DiagnosticsNode{foundation.ToDiagnosticsNode(m.child, foundation.DiagnosticsNodeOpts{Name: foundation.Some("child")})}
}

// ContainerRenderObject holds a list of children of type C for a render object.
// The children's parent data must embed ContainerParentData[C]. Embed it and call
// InitContainerRenderObject in the constructor.
//
// Flutter: rendering/object.dart ContainerRenderObjectMixin
type ContainerRenderObject[C RenderObject] struct {
	host       RenderObject
	firstChild C
	lastChild  C
	childCount int
}

// InitContainerRenderObject registers the mixin with its render object.
//
// Flutter: no counterpart (the self pattern, SPEC §12.5)
func InitContainerRenderObject[C RenderObject](m *ContainerRenderObject[C], host RenderObject) {
	m.host = host
	host.AsRenderObject().setChildModel(m)
}

func (m *ContainerRenderObject[C]) data(child C) *ContainerParentData[C] {
	pd := child.AsRenderObject().parentData
	l, ok := pd.(containerParentDataLinks[C])
	if !ok {
		panic(errors.New("ContainerParentDataMissing", "a child of "+foundation.RuntimeTypeName(m.host)+
			" has parent data that does not embed ContainerParentData; the class should override SetupParentData",
			"object", foundation.DescribeIdentity(m.host), "parentData", pd))
	}
	return l.links()
}

// ChildCount is the number of children.
//
// Flutter: rendering/object.dart ContainerRenderObjectMixin.childCount
func (m *ContainerRenderObject[C]) ChildCount() int { return m.childCount }

// FirstChild is the first child, or nil.
//
// Flutter: rendering/object.dart ContainerRenderObjectMixin.firstChild
func (m *ContainerRenderObject[C]) FirstChild() C { return m.firstChild }

// LastChild is the last child, or nil.
//
// Flutter: rendering/object.dart ContainerRenderObjectMixin.lastChild
func (m *ContainerRenderObject[C]) LastChild() C { return m.lastChild }

func (m *ContainerRenderObject[C]) insertIntoChildList(child, after C) {
	cd := m.data(child)
	m.childCount++
	if isNil(after) {
		cd.NextSibling = m.firstChild
		if !isNil(m.firstChild) {
			m.data(m.firstChild).PreviousSibling = child
		}
		m.firstChild = child
		if isNil(m.lastChild) {
			m.lastChild = child
		}
		return
	}
	ad := m.data(after)
	if isNil(ad.NextSibling) {
		cd.PreviousSibling = after
		ad.NextSibling = child
		m.lastChild = child
		return
	}
	cd.NextSibling = ad.NextSibling
	cd.PreviousSibling = after
	m.data(cd.PreviousSibling).NextSibling = child
	m.data(cd.NextSibling).PreviousSibling = child
}

// Insert adds the child after another child, or at the start if after is nil. In
// debug builds it panics with ContainerInsertInvalid if the child is the object
// itself, is after itself, or is a child already.
//
// Flutter: rendering/object.dart ContainerRenderObjectMixin.insert
func (m *ContainerRenderObject[C]) Insert(child, after C) {
	if foundation.DebugMode {
		var msg string
		switch {
		case any(child) == any(m.host):
			msg = "a render object can not be inserted into itself"
		case any(after) == any(m.host):
			msg = "a render object can not be both the parent and the sibling of another render object"
		case !isNil(after) && any(child) == any(after):
			msg = "a render object can not be inserted after itself"
		case any(child) == any(m.firstChild) || any(child) == any(m.lastChild):
			msg = "the render object is a child already"
		}
		if msg != "" {
			panic(errors.New("ContainerInsertInvalid", msg, "object", foundation.DescribeIdentity(m.host),
				"child", foundation.DescribeIdentity(child)))
		}
	}
	m.host.AsRenderObject().AdoptChild(child)
	m.insertIntoChildList(child, after)
}

// Add adds the child at the end.
//
// Flutter: rendering/object.dart ContainerRenderObjectMixin.add
func (m *ContainerRenderObject[C]) Add(child C) { m.Insert(child, m.lastChild) }

// AddAll adds the children at the end, in order.
//
// Flutter: rendering/object.dart ContainerRenderObjectMixin.addAll
func (m *ContainerRenderObject[C]) AddAll(children []C) {
	for _, c := range children {
		m.Add(c)
	}
}

func (m *ContainerRenderObject[C]) removeFromChildList(child C) {
	cd := m.data(child)
	if isNil(cd.PreviousSibling) {
		m.firstChild = cd.NextSibling
	} else {
		m.data(cd.PreviousSibling).NextSibling = cd.NextSibling
	}
	if isNil(cd.NextSibling) {
		m.lastChild = cd.PreviousSibling
	} else {
		m.data(cd.NextSibling).PreviousSibling = cd.PreviousSibling
	}
	var zero C
	cd.PreviousSibling = zero
	cd.NextSibling = zero
	m.childCount--
}

// Remove removes the child.
//
// Flutter: rendering/object.dart ContainerRenderObjectMixin.remove
func (m *ContainerRenderObject[C]) Remove(child C) {
	m.removeFromChildList(child)
	m.host.AsRenderObject().DropChild(child)
}

// RemoveAll removes every child.
//
// Flutter: rendering/object.dart ContainerRenderObjectMixin.removeAll
func (m *ContainerRenderObject[C]) RemoveAll() {
	child := m.firstChild
	var zero C
	for !isNil(child) {
		cd := m.data(child)
		next := cd.NextSibling
		cd.PreviousSibling = zero
		cd.NextSibling = zero
		m.host.AsRenderObject().DropChild(child)
		child = next
	}
	m.firstChild = zero
	m.lastChild = zero
	m.childCount = 0
}

// Move moves the child to after another child (or the start), and marks the
// object as needing layout.
//
// Flutter: rendering/object.dart ContainerRenderObjectMixin.move
func (m *ContainerRenderObject[C]) Move(child, after C) {
	if foundation.DebugMode {
		hb := m.host.AsRenderObject()
		if any(child) == any(m.host) || any(after) == any(m.host) || any(child) == any(after) || child.AsRenderObject().parent != m.host {
			panic(errors.New("ContainerMoveInvalid", "a child can only be moved after another child of the same container",
				"object", foundation.DescribeIdentity(hb.self), "child", foundation.DescribeIdentity(child)))
		}
	}
	cd := m.data(child)
	if any(cd.PreviousSibling) == any(after) {
		return
	}
	m.removeFromChildList(child)
	m.insertIntoChildList(child, after)
	m.host.MarkNeedsLayout()
}

func (m *ContainerRenderObject[C]) attachChildren(owner *PipelineOwner) {
	for child := m.firstChild; !isNil(child); child = m.data(child).NextSibling {
		child.Attach(owner)
	}
}

func (m *ContainerRenderObject[C]) detachChildren() {
	for child := m.firstChild; !isNil(child); child = m.data(child).NextSibling {
		child.Detach()
	}
}

func (m *ContainerRenderObject[C]) redepthChildren() {
	for child := m.firstChild; !isNil(child); child = m.data(child).NextSibling {
		m.host.AsRenderObject().RedepthChild(child)
	}
}

func (m *ContainerRenderObject[C]) visitChildren(visitor RenderObjectVisitor) {
	for child := m.firstChild; !isNil(child); child = m.data(child).NextSibling {
		visitor(child)
	}
}

// ChildBefore is the child before the given child, or nil.
//
// Flutter: rendering/object.dart ContainerRenderObjectMixin.childBefore
func (m *ContainerRenderObject[C]) ChildBefore(child C) C {
	if foundation.DebugMode && child.AsRenderObject().parent != m.host {
		panic(errors.New("RenderObjectNotChild", "ChildBefore was called with a render object that is not a child",
			"child", foundation.DescribeIdentity(child)))
	}
	return m.data(child).PreviousSibling
}

// ChildAfter is the child after the given child, or nil.
//
// Flutter: rendering/object.dart ContainerRenderObjectMixin.childAfter
func (m *ContainerRenderObject[C]) ChildAfter(child C) C {
	if foundation.DebugMode && child.AsRenderObject().parent != m.host {
		panic(errors.New("RenderObjectNotChild", "ChildAfter was called with a render object that is not a child",
			"child", foundation.DescribeIdentity(child)))
	}
	return m.data(child).NextSibling
}

func (m *ContainerRenderObject[C]) describeChildren() []foundation.DiagnosticsNode {
	var out []foundation.DiagnosticsNode
	count := 1
	for child := m.firstChild; !isNil(child); child = m.data(child).NextSibling {
		out = append(out, foundation.ToDiagnosticsNode(child, foundation.DiagnosticsNodeOpts{Name: foundation.Some("child " + strconv.Itoa(count))}))
		count++
	}
	return out
}

// LayoutCallbackMixin is the state for a render object that runs a layout
// callback while it lays out (a LayoutBuilder's). Embed it, call
// InitLayoutCallbackMixin, and give the callback as the function.
//
// Flutter: rendering/object.dart RenderObjectWithLayoutCallbackMixin
type LayoutCallbackMixin struct {
	host         RenderObject
	needsRebuild bool
	callback     func()
}

// InitLayoutCallbackMixin registers the layout callback of the render object.
//
// Flutter: no counterpart (the self pattern, SPEC §12.5)
func InitLayoutCallbackMixin(m *LayoutCallbackMixin, host RenderObject, layoutCallback func()) {
	m.host = host
	m.callback = layoutCallback
	m.needsRebuild = true
}

// RunLayoutCallback runs the layout callback; call it from PerformLayout.
//
// Flutter: rendering/object.dart RenderObjectWithLayoutCallbackMixin.runLayoutCallback
func (m *LayoutCallbackMixin) RunLayoutCallback() {
	hb := m.host.AsRenderObject()
	if foundation.DebugMode && !hb.debugDoingThisLayout {
		panic(errors.New("RunLayoutCallbackInvalid", "the layout callback can only run during PerformLayout",
			"object", foundation.DescribeIdentity(m.host)))
	}
	hb.InvokeLayoutCallback(func(Constraints) { m.callback() })
	m.needsRebuild = false
}

// ScheduleLayoutCallback schedules the layout callback: the object is added to
// the owner's list to lay out, and marked as needing layout.
//
// Flutter: rendering/object.dart RenderObjectWithLayoutCallbackMixin.scheduleLayoutCallback
func (m *LayoutCallbackMixin) ScheduleLayoutCallback() {
	hb := m.host.AsRenderObject()
	if m.needsRebuild {
		return
	}
	m.needsRebuild = true
	if hb.owner != nil {
		hb.owner.nodesNeedingLayout = append(hb.owner.nodesNeedingLayout, m.host)
	}
	hb.MarkNeedsLayout()
}

// ChildObject is the child as a RenderObject, or nil. The widgets library uses it
// to reach the child of a render object without knowing its type.
//
// Flutter: no counterpart (RenderObjectWithChildMixin<RenderObject>.child)
func (m *RenderObjectWithChild[C]) ChildObject() RenderObject {
	if isNil(m.child) {
		return nil
	}
	return m.child
}

// SetChildObject replaces the child with a RenderObject that must be a C, or nil
// for none. It panics with the tag RenderChildTypeMismatch in debug builds when
// the child is of the wrong type.
//
// Flutter: widgets/framework.dart SingleChildRenderObjectElement.insertRenderObjectChild
func (m *RenderObjectWithChild[C]) SetChildObject(child RenderObject) {
	if child == nil {
		var zero C
		m.SetChild(zero)
		return
	}
	typed, ok := child.(C)
	if !ok {
		panic(errors.New("RenderChildTypeMismatch", "the render object is not of the type this render object accepts as a child",
			"parent", foundation.DescribeIdentity(m.host), "child", foundation.DescribeIdentity(child)))
	}
	m.SetChild(typed)
}

// InsertObject inserts the child after a sibling (nil for the start); both must
// be a C. It panics with the tag RenderChildTypeMismatch when one is of the
// wrong type.
//
// Flutter: widgets/framework.dart MultiChildRenderObjectElement.insertRenderObjectChild
func (m *ContainerRenderObject[C]) InsertObject(child, after RenderObject) {
	m.Insert(m.typed(child), m.typedOrZero(after))
}

// MoveObject moves the child after a sibling (nil for the start).
//
// Flutter: widgets/framework.dart MultiChildRenderObjectElement.moveRenderObjectChild
func (m *ContainerRenderObject[C]) MoveObject(child, after RenderObject) {
	m.Move(m.typed(child), m.typedOrZero(after))
}

// RemoveObject removes the child.
//
// Flutter: widgets/framework.dart MultiChildRenderObjectElement.removeRenderObjectChild
func (m *ContainerRenderObject[C]) RemoveObject(child RenderObject) { m.Remove(m.typed(child)) }

func (m *ContainerRenderObject[C]) typed(child RenderObject) C {
	typed, ok := child.(C)
	if !ok {
		panic(errors.New("RenderChildTypeMismatch", "the render object is not of the type this render object accepts as a child",
			"parent", foundation.DescribeIdentity(m.host), "child", foundation.DescribeIdentity(child)))
	}
	return typed
}

func (m *ContainerRenderObject[C]) typedOrZero(child RenderObject) C {
	var zero C
	if child == nil {
		return zero
	}
	return m.typed(child)
}
