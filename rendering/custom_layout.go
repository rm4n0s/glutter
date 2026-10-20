package rendering

import (
	"fmt"
	"strings"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// MultiChildLayoutParentData is the parent data of the children of a
// RenderCustomMultiChildLayoutBox: the id that the delegate knows the child by.
//
// Flutter: rendering/custom_layout.dart MultiChildLayoutParentData
type MultiChildLayoutParentData struct {
	ContainerBoxParentData[Box]
	// ID is the identifier used to refer to the child. It must be comparable.
	//
	// Flutter: rendering/custom_layout.dart MultiChildLayoutParentData.id
	ID any
}

// String is the offset and the id.
//
// Flutter: rendering/custom_layout.dart MultiChildLayoutParentData.toString
func (d *MultiChildLayoutParentData) String() string {
	return d.BoxParentData.String() + "; id=" + fmt.Sprint(d.ID)
}

// MultiChildLayoutDelegate controls the layout and the position of the children
// of a RenderCustomMultiChildLayoutBox. Embed MultiChildLayoutDelegateBase and
// implement PerformLayout and ShouldRelayout.
//
// Flutter: rendering/custom_layout.dart MultiChildLayoutDelegate
type MultiChildLayoutDelegate interface {
	foundation.Listenable
	// Base is the embedded base, where the state of a layout is kept.
	Base() *MultiChildLayoutDelegateBase
	// GetSize is the size of the box for the constraints.
	//
	// Flutter: rendering/custom_layout.dart MultiChildLayoutDelegate.getSize
	GetSize(constraints BoxConstraints) ui.Size
	// PerformLayout lays out and positions the children, each exactly once,
	// with the base's LayoutChild and PositionChild.
	//
	// Flutter: rendering/custom_layout.dart MultiChildLayoutDelegate.performLayout
	PerformLayout(size ui.Size)
	// ShouldRelayout says whether the box should be laid out again, given the
	// delegate of the same type that it replaces.
	//
	// Flutter: rendering/custom_layout.dart MultiChildLayoutDelegate.shouldRelayout
	ShouldRelayout(oldDelegate MultiChildLayoutDelegate) bool
}

// MultiChildLayoutDelegateBase has the relayout listenable, the state of a
// layout and the methods a delegate uses to lay its children out.
//
// Flutter: rendering/custom_layout.dart MultiChildLayoutDelegate
type MultiChildLayoutDelegateBase struct {
	relayout              foundation.Listenable
	self                  MultiChildLayoutDelegate
	idToChild             map[any]Box
	debugChildrenNeedLays map[Box]bool
}

// NewMultiChildLayoutDelegateBase creates the base; the box is laid out again
// when the relayout listenable (nil for none) notifies.
//
// Flutter: rendering/custom_layout.dart MultiChildLayoutDelegate.new
func NewMultiChildLayoutDelegateBase(relayout foundation.Listenable) MultiChildLayoutDelegateBase {
	return MultiChildLayoutDelegateBase{relayout: relayout}
}

// Base returns the base itself.
//
// Flutter: no counterpart (the self pattern, SPEC §12.5)
func (d *MultiChildLayoutDelegateBase) Base() *MultiChildLayoutDelegateBase { return d }

// AddListener adds the listener to the relayout listenable.
//
// Flutter: rendering/custom_layout.dart MultiChildLayoutDelegate._relayout
func (d *MultiChildLayoutDelegateBase) AddListener(listener foundation.VoidCallback) foundation.ListenerHandle {
	if d.relayout == nil {
		return foundation.ListenerHandle{}
	}
	return d.relayout.AddListener(listener)
}

// RemoveListener removes the listener from the relayout listenable.
//
// Flutter: rendering/custom_layout.dart MultiChildLayoutDelegate._relayout
func (d *MultiChildLayoutDelegateBase) RemoveListener(handle foundation.ListenerHandle) {
	if d.relayout != nil {
		d.relayout.RemoveListener(handle)
	}
}

// GetSize is the biggest size the constraints allow.
//
// Flutter: rendering/custom_layout.dart MultiChildLayoutDelegate.getSize
func (*MultiChildLayoutDelegateBase) GetSize(constraints BoxConstraints) ui.Size {
	return constraints.Biggest()
}

// HasChild reports whether there is a child with the id.
//
// Flutter: rendering/custom_layout.dart MultiChildLayoutDelegate.hasChild
func (d *MultiChildLayoutDelegateBase) HasChild(childID any) bool {
	return !isNil(d.idToChild[childID])
}

func (d *MultiChildLayoutDelegateBase) name() string { return foundation.RuntimeTypeName(d.self) }

// LayoutChild lays out the child with the id with the constraints and returns
// its size. In debug builds it panics with MultiChildLayoutNoSuchChild if there
// is no such child, and with MultiChildLayoutChildTwice if the child was laid
// out already.
//
// Flutter: rendering/custom_layout.dart MultiChildLayoutDelegate.layoutChild
func (d *MultiChildLayoutDelegateBase) LayoutChild(childID any, constraints BoxConstraints) ui.Size {
	child := d.idToChild[childID]
	if foundation.DebugMode {
		if isNil(child) {
			panic(errors.New("MultiChildLayoutNoSuchChild", "The "+d.name()+" custom multichild layout delegate tried to lay out a "+
				"non-existent child. There is no child with the id \""+fmt.Sprint(childID)+"\".", "id", childID))
		}
		if !d.debugChildrenNeedLays[child] {
			panic(errors.New("MultiChildLayoutChildTwice", "The "+d.name()+" custom multichild layout delegate tried to lay out the "+
				"child with id \""+fmt.Sprint(childID)+"\" more than once. Each child must be laid out exactly once.", "id", childID))
		}
		delete(d.debugChildrenNeedLays, child)
		constraints.DebugAssertIsValid(DebugAssertIsValidOpts{IsAppliedConstraint: true})
	}
	child.Layout(constraints, LayoutOpts{ParentUsesSize: true})
	return child.Size()
}

// PositionChild sets the offset of the child with the id. In debug builds it
// panics with MultiChildLayoutNoSuchChild if there is no such child.
//
// Flutter: rendering/custom_layout.dart MultiChildLayoutDelegate.positionChild
func (d *MultiChildLayoutDelegateBase) PositionChild(childID any, offset ui.Offset) {
	child := d.idToChild[childID]
	if foundation.DebugMode && isNil(child) {
		panic(errors.New("MultiChildLayoutNoSuchChild", "The "+d.name()+" custom multichild layout delegate tried to position out a "+
			"non-existent child: There is no child with the id \""+fmt.Sprint(childID)+"\".", "id", childID))
	}
	child.ParentData().(*MultiChildLayoutParentData).Offset = offset
}

// callPerformLayout runs the delegate's PerformLayout with the children by id.
//
// Flutter: rendering/custom_layout.dart MultiChildLayoutDelegate._callPerformLayout
func (d *MultiChildLayoutDelegateBase) callPerformLayout(delegate MultiChildLayoutDelegate, size ui.Size, firstChild Box) {
	previousIDToChild, previousNeeding, previousSelf := d.idToChild, d.debugChildrenNeedLays, d.self
	d.self = delegate
	if foundation.DebugMode {
		d.debugChildrenNeedLays = map[Box]bool{}
	}
	defer func() {
		d.idToChild, d.debugChildrenNeedLays, d.self = previousIDToChild, previousNeeding, previousSelf
	}()
	d.idToChild = map[any]Box{}
	var order []Box
	for child := firstChild; !isNil(child); {
		data := child.ParentData().(*MultiChildLayoutParentData)
		if foundation.DebugMode && data.ID == nil {
			panic(errors.New("MultiChildLayoutMissingID", "Every child of a RenderCustomMultiChildLayoutBox must have an ID in its "+
				"parent data. The following child has no ID: "+foundation.DescribeIdentity(child), "child", foundation.DescribeIdentity(child)))
		}
		d.idToChild[data.ID] = child
		if foundation.DebugMode {
			d.debugChildrenNeedLays[child] = true
			order = append(order, child)
		}
		child = data.NextSibling
	}
	delegate.PerformLayout(size)
	if foundation.DebugMode && len(d.debugChildrenNeedLays) > 0 {
		var forgotten []string
		for _, child := range order {
			if d.debugChildrenNeedLays[child] {
				forgotten = append(forgotten, fmt.Sprint(child.ParentData().(*MultiChildLayoutParentData).ID)+": "+foundation.DescribeIdentity(child))
			}
		}
		noun := "child"
		if len(forgotten) > 1 {
			noun = "children"
		}
		panic(errors.New("MultiChildLayoutChildNotLaidOut", "Each child must be laid out exactly once. The "+d.name()+
			" custom multichild layout delegate forgot to lay out the following "+noun+": "+strings.Join(forgotten, ", "),
			"children", forgotten))
	}
}

// RenderCustomMultiChildLayoutBox uses a delegate to size and position multiple
// children.
//
// Flutter: rendering/custom_layout.dart RenderCustomMultiChildLayoutBox
type RenderCustomMultiChildLayoutBox struct {
	RenderBox
	ContainerRenderObject[Box]
	delegate       MultiChildLayoutDelegate
	delegateHandle foundation.ListenerHandle
}

// NewRenderCustomMultiChildLayoutBox creates the box with the children (nil for
// none).
//
// Flutter: rendering/custom_layout.dart RenderCustomMultiChildLayoutBox.new
func NewRenderCustomMultiChildLayoutBox(delegate MultiChildLayoutDelegate, children []Box) *RenderCustomMultiChildLayoutBox {
	r := &RenderCustomMultiChildLayoutBox{delegate: delegate}
	InitRenderBox(&r.RenderBox, r)
	InitContainerRenderObject(&r.ContainerRenderObject, r)
	r.AddAll(children)
	return r
}

// SetupParentData gives the child a MultiChildLayoutParentData if it has none.
//
// Flutter: rendering/custom_layout.dart RenderCustomMultiChildLayoutBox.setupParentData
func (r *RenderCustomMultiChildLayoutBox) SetupParentData(child RenderObject) {
	if _, ok := child.ParentData().(*MultiChildLayoutParentData); !ok {
		child.AsRenderObject().SetParentData(&MultiChildLayoutParentData{})
	}
}

// Delegate controls the layout of the children.
//
// Flutter: rendering/custom_layout.dart RenderCustomMultiChildLayoutBox.delegate
func (r *RenderCustomMultiChildLayoutBox) Delegate() MultiChildLayoutDelegate { return r.delegate }

// SetDelegate changes the delegate. The box needs layout if the new delegate is
// of another type or says so.
//
// Flutter: rendering/custom_layout.dart RenderCustomMultiChildLayoutBox.delegate
func (r *RenderCustomMultiChildLayoutBox) SetDelegate(newDelegate MultiChildLayoutDelegate) {
	if any(r.delegate) == any(newDelegate) {
		return
	}
	old := r.delegate
	if foundation.RuntimeTypeName(newDelegate) != foundation.RuntimeTypeName(old) || newDelegate.ShouldRelayout(old) {
		r.MarkNeedsLayout()
	}
	r.delegate = newDelegate
	if r.Attached() {
		old.RemoveListener(r.delegateHandle)
		r.delegateHandle = newDelegate.AddListener(r.MarkNeedsLayout)
	}
}

// Attach starts listening to the delegate's relayout listenable.
//
// Flutter: rendering/custom_layout.dart RenderCustomMultiChildLayoutBox.attach
func (r *RenderCustomMultiChildLayoutBox) Attach(owner *PipelineOwner) {
	r.RenderBox.Attach(owner)
	r.delegateHandle = r.delegate.AddListener(r.MarkNeedsLayout)
}

// Detach stops listening to the delegate's relayout listenable.
//
// Flutter: rendering/custom_layout.dart RenderCustomMultiChildLayoutBox.detach
func (r *RenderCustomMultiChildLayoutBox) Detach() {
	r.delegate.RemoveListener(r.delegateHandle)
	r.delegateHandle = foundation.ListenerHandle{}
	r.RenderBox.Detach()
}

func (r *RenderCustomMultiChildLayoutBox) getSize(constraints BoxConstraints) ui.Size {
	if foundation.DebugMode {
		constraints.DebugAssertIsValid()
	}
	return constraints.Constrain(r.delegate.GetSize(constraints))
}

// ComputeMinIntrinsicWidth is the delegate's width for the height, or 0 if it
// is not finite.
//
// Flutter: rendering/custom_layout.dart RenderCustomMultiChildLayoutBox.computeMinIntrinsicWidth
func (r *RenderCustomMultiChildLayoutBox) ComputeMinIntrinsicWidth(height float64) float64 {
	return finiteOrZero(r.getSize(NewBoxConstraintsTightForFinite(TightForFiniteOpts{Height: foundation.Some(height)})).Width)
}

// ComputeMaxIntrinsicWidth is the delegate's width for the height, or 0 if it
// is not finite.
//
// Flutter: rendering/custom_layout.dart RenderCustomMultiChildLayoutBox.computeMaxIntrinsicWidth
func (r *RenderCustomMultiChildLayoutBox) ComputeMaxIntrinsicWidth(height float64) float64 {
	return r.ComputeMinIntrinsicWidth(height)
}

// ComputeMinIntrinsicHeight is the delegate's height for the width, or 0 if it
// is not finite.
//
// Flutter: rendering/custom_layout.dart RenderCustomMultiChildLayoutBox.computeMinIntrinsicHeight
func (r *RenderCustomMultiChildLayoutBox) ComputeMinIntrinsicHeight(width float64) float64 {
	return finiteOrZero(r.getSize(NewBoxConstraintsTightForFinite(TightForFiniteOpts{Width: foundation.Some(width)})).Height)
}

// ComputeMaxIntrinsicHeight is the delegate's height for the width, or 0 if it
// is not finite.
//
// Flutter: rendering/custom_layout.dart RenderCustomMultiChildLayoutBox.computeMaxIntrinsicHeight
func (r *RenderCustomMultiChildLayoutBox) ComputeMaxIntrinsicHeight(width float64) float64 {
	return r.ComputeMinIntrinsicHeight(width)
}

// ComputeDryLayout is the size the delegate gives.
//
// Flutter: rendering/custom_layout.dart RenderCustomMultiChildLayoutBox.computeDryLayout
func (r *RenderCustomMultiChildLayoutBox) ComputeDryLayout(constraints BoxConstraints) ui.Size {
	return r.getSize(constraints)
}

// PerformLayout sizes the box and lets the delegate lay the children out.
//
// Flutter: rendering/custom_layout.dart RenderCustomMultiChildLayoutBox.performLayout
func (r *RenderCustomMultiChildLayoutBox) PerformLayout() {
	r.SetSize(r.getSize(r.BoxConstraints()))
	r.delegate.Base().callPerformLayout(r.delegate, r.Size(), r.FirstChild())
}

// Paint paints the children in order.
//
// Flutter: rendering/custom_layout.dart RenderCustomMultiChildLayoutBox.paint
func (r *RenderCustomMultiChildLayoutBox) Paint(context *PaintingContext, offset ui.Offset) {
	DefaultPaint(&r.ContainerRenderObject, context, offset)
}

// HitTestChildren hit tests the children from the front.
//
// Flutter: rendering/custom_layout.dart RenderCustomMultiChildLayoutBox.hitTestChildren
func (r *RenderCustomMultiChildLayoutBox) HitTestChildren(result *BoxHitTestResult, position ui.Offset) bool {
	return DefaultHitTestChildren(&r.ContainerRenderObject, result, position)
}
