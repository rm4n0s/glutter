package rendering

import (
	"fmt"
	"strconv"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/ui"
)

// RenderObjectVisitor is called with a render object, such as each child.
//
// Flutter: rendering/object.dart RenderObjectVisitor
type RenderObjectVisitor func(child RenderObject)

// LayoutCallback is called while a render object lays out, to build or change
// its children. The constraints are those of the render object.
//
// Flutter: rendering/object.dart LayoutCallback
type LayoutCallback func(constraints Constraints)

// LayoutOpts are the optional parameters of RenderObject.Layout.
//
// Flutter: rendering/object.dart RenderObject.layout
type LayoutOpts struct {
	// ParentUsesSize says that the parent reads the child's size, so the parent
	// has to lay out again when the child's size changes.
	ParentUsesSize bool
}

// RenderObject is a node of the render tree: it lays out, paints and hit tests
// a part of the UI. A render object type embeds RenderBox (or another base
// struct of this package) and calls the matching Init function, which gives the
// base the pointer to the outer object that its virtual calls go through
// (SPEC §12.5). The methods below are the ones that are overridden or called
// across types; RenderObjectBase gives them default implementations.
//
// Flutter: rendering/object.dart RenderObject
type RenderObject interface {
	foundation.DiagnosticableTree
	gestures.HitTestTarget

	// AsRenderObject returns the RenderObjectBase that the object embeds.
	AsRenderObject() *RenderObjectBase

	// SetupParentData gives the child the kind of parent data this object
	// needs.
	//
	// Flutter: rendering/object.dart RenderObject.setupParentData
	SetupParentData(child RenderObject)
	// VisitChildren calls the visitor with each child.
	//
	// Flutter: rendering/object.dart RenderObject.visitChildren
	VisitChildren(visitor RenderObjectVisitor)
	// Attach attaches the object and its children to the owner.
	//
	// Flutter: rendering/object.dart RenderObject.attach
	Attach(owner *PipelineOwner)
	// Detach detaches the object and its children from the owner.
	//
	// Flutter: rendering/object.dart RenderObject.detach
	Detach()
	// RedepthChildren updates the depth of the children.
	//
	// Flutter: rendering/object.dart RenderObject.redepthChildren
	RedepthChildren()
	// Dispose releases the object's resources.
	//
	// Flutter: rendering/object.dart RenderObject.dispose
	Dispose()
	// MarkNeedsLayout marks the object as needing layout.
	//
	// Flutter: rendering/object.dart RenderObject.markNeedsLayout
	MarkNeedsLayout()
	// SizedByParent says that the size depends only on the constraints, so
	// PerformResize sets it and PerformLayout only lays out the children.
	//
	// Flutter: rendering/object.dart RenderObject.sizedByParent
	SizedByParent() bool
	// PerformResize sets the size from the constraints, for an object that is
	// SizedByParent.
	//
	// Flutter: rendering/object.dart RenderObject.performResize
	PerformResize()
	// PerformLayout lays out the children and, unless SizedByParent, sets the
	// size.
	//
	// Flutter: rendering/object.dart RenderObject.performLayout
	PerformLayout()
	// DebugAssertDoesMeetConstraints panics in debug builds if the geometry
	// does not satisfy the constraints.
	//
	// Flutter: rendering/object.dart RenderObject.debugAssertDoesMeetConstraints
	DebugAssertDoesMeetConstraints()
	// DebugResetSize sets the size again, to check it against the new constraints.
	//
	// Flutter: rendering/object.dart RenderObject.debugResetSize
	DebugResetSize()
	// IsRepaintBoundary says whether the object paints into its own layer.
	//
	// Flutter: rendering/object.dart RenderObject.isRepaintBoundary
	IsRepaintBoundary() bool
	// AlwaysNeedsCompositing says whether the object must be composited, even
	// with no repaint boundary below it.
	//
	// Flutter: rendering/object.dart RenderObject.alwaysNeedsCompositing
	AlwaysNeedsCompositing() bool
	// UpdateCompositedLayer creates the layer of a repaint boundary, or updates
	// the old one (and returns it).
	//
	// Flutter: rendering/object.dart RenderObject.updateCompositedLayer
	UpdateCompositedLayer(oldLayer OffsetLayerLike) OffsetLayerLike
	// DebugRegisterRepaintBoundaryPaint is called when a repaint boundary
	// paints, for the benefit of tests.
	//
	// Flutter: rendering/object.dart RenderObject.debugRegisterRepaintBoundaryPaint
	DebugRegisterRepaintBoundaryPaint(includedParent, includedChild bool)
	// Paint paints the object at the offset.
	//
	// Flutter: rendering/object.dart RenderObject.paint
	Paint(context *PaintingContext, offset ui.Offset)
	// DebugPaint paints the debug overlays.
	//
	// Flutter: rendering/object.dart RenderObject.debugPaint
	DebugPaint(context *PaintingContext, offset ui.Offset)
	// PaintBounds are the bounds that the object paints in, in its own
	// coordinates.
	//
	// Flutter: rendering/object.dart RenderObject.paintBounds
	PaintBounds() ui.Rect
	// ApplyPaintTransform applies to the transform the transformation that
	// paints the child.
	//
	// Flutter: rendering/object.dart RenderObject.applyPaintTransform
	ApplyPaintTransform(child RenderObject, transform *ui.Matrix4)
	// PaintsChild says whether the child is painted.
	//
	// Flutter: rendering/object.dart RenderObject.paintsChild
	PaintsChild(child RenderObject) bool
	// DescribeApproximatePaintClip is the area that the child is clipped to.
	//
	// Flutter: rendering/object.dart RenderObject.describeApproximatePaintClip
	DescribeApproximatePaintClip(child RenderObject) foundation.Opt[ui.Rect]
	// SemanticBounds are the bounds of the object for semantics.
	//
	// Flutter: rendering/object.dart RenderObject.semanticBounds
	SemanticBounds() ui.Rect
	// DebugOptions are the debug flags of the owner.
	DebugOptions() *DebugOptions

	// Layout lays out the object for the constraints.
	//
	// Flutter: rendering/object.dart RenderObject.layout
	Layout(constraints Constraints, opts ...LayoutOpts)
	// MarkNeedsPaint marks the object as needing paint.
	//
	// Flutter: rendering/object.dart RenderObject.markNeedsPaint
	MarkNeedsPaint()
	// MarkParentNeedsLayout marks the object and its parent as needing layout.
	//
	// Flutter: rendering/object.dart RenderObject.markParentNeedsLayout
	MarkParentNeedsLayout()
	// MarkNeedsCompositingBitsUpdate marks the compositing bits as dirty.
	//
	// Flutter: rendering/object.dart RenderObject.markNeedsCompositingBitsUpdate
	MarkNeedsCompositingBitsUpdate()
	// Parent is the parent render object, or nil.
	//
	// Flutter: rendering/object.dart RenderObject.parent
	Parent() RenderObject
	// DebugLayoutParent is the parent in the layout protocol (debug builds);
	// it differs from Parent for a render object that is laid out by another
	// one than the one it is a child of.
	//
	// Flutter: rendering/object.dart RenderObject.debugLayoutParent
	DebugLayoutParent() RenderObject
	// ParentData is the data the parent stores on the object.
	//
	// Flutter: rendering/object.dart RenderObject.parentData
	ParentData() ParentData
	// Owner is the pipeline owner, or nil.
	//
	// Flutter: rendering/object.dart RenderObject.owner
	Owner() *PipelineOwner
	// Attached reports whether the object has an owner.
	//
	// Flutter: rendering/object.dart RenderObject.attached
	Attached() bool
	// Depth is the distance from the root.
	//
	// Flutter: rendering/object.dart RenderObject.depth
	Depth() int
	// DebugNeedsLayout reports whether the object needs layout (debug builds).
	//
	// Flutter: rendering/object.dart RenderObject.debugNeedsLayout
	DebugNeedsLayout() bool
	// NeedsCompositing reports whether the object needs compositing.
	//
	// Flutter: rendering/object.dart RenderObject.needsCompositing
	NeedsCompositing() bool
	// GetTransformTo is the transform to the target's coordinates.
	//
	// Flutter: rendering/object.dart RenderObject.getTransformTo
	GetTransformTo(target RenderObject) *ui.Matrix4
	// DebugDoingThisLayoutWithCallback reports whether a layout callback runs.
	//
	// Flutter: rendering/object.dart RenderObject.debugDoingThisLayoutWithCallback
	DebugDoingThisLayoutWithCallback() bool

	// MarkNeedsSemanticsUpdate marks the semantics of the object as changed.
	//
	// Flutter: rendering/object.dart RenderObject.markNeedsSemanticsUpdate
	MarkNeedsSemanticsUpdate()
	// DescribeSemanticsConfiguration fills the configuration with the
	// semantics of the object.
	//
	// Flutter: rendering/object.dart RenderObject.describeSemanticsConfiguration
	DescribeSemanticsConfiguration(config *semantics.SemanticsConfiguration)
	// VisitChildrenForSemantics visits the children that take part in
	// semantics, in paint order.
	//
	// Flutter: rendering/object.dart RenderObject.visitChildrenForSemantics
	VisitChildrenForSemantics(visitor RenderObjectVisitor)
	// AssembleSemanticsNode gives the node its configuration and children.
	//
	// Flutter: rendering/object.dart RenderObject.assembleSemanticsNode
	AssembleSemanticsNode(node *semantics.SemanticsNode, config *semantics.SemanticsConfiguration, children []*semantics.SemanticsNode)
	// DescribeSemanticsClip is the clip of the semantics of the child.
	//
	// Flutter: rendering/object.dart RenderObject.describeSemanticsClip
	DescribeSemanticsClip(child RenderObject) foundation.Opt[ui.Rect]
	// ClearSemantics throws away the semantics of the object and its subtree.
	//
	// Flutter: rendering/object.dart RenderObject.clearSemantics
	ClearSemantics()
	// ShowOnScreen asks the ancestors to scroll the object into view.
	//
	// Flutter: rendering/object.dart RenderObject.showOnScreen
	ShowOnScreen(opts ShowOnScreenOpts)
	// SendSemanticsEvent sends the event about the object.
	//
	// Flutter: rendering/object.dart RenderObject.sendSemanticsEvent
	SendSemanticsEvent(event semantics.SemanticsEvent)
}

// renderChildModel is how RenderObjectBase reaches the children that a mixin
// struct (RenderObjectWithChild, ContainerRenderObject) holds.
type renderChildModel interface {
	visitChildren(visitor RenderObjectVisitor)
	attachChildren(owner *PipelineOwner)
	detachChildren()
	redepthChildren()
	describeChildren() []foundation.DiagnosticsNode
}

// RenderObjectBase is the state and the default behavior that every render
// object embeds (through RenderBox).
//
// Flutter: rendering/object.dart RenderObject
type RenderObjectBase struct {
	self RenderObject

	parentData ParentData
	depth      int
	parent     RenderObject
	owner      *PipelineOwner
	model      renderChildModel

	needsLayout                 bool
	isRelayoutBoundary          foundation.Opt[bool]
	doingThisLayoutWithCallback bool
	constraints                 Constraints

	needsCompositingBitsUpdate bool
	needsCompositing           bool
	needsPaint                 bool
	needsCompositedLayerUpdate bool
	wasRepaintBoundary         bool
	layerHandle                LayerHandle[ContainerLayerLike]

	debugDisposed         bool
	debugDoingThisResize  bool
	debugDoingThisLayout  bool
	debugDoingThisPaint   bool
	debugCanParentUseSize foundation.Opt[bool]
	debugMutationsLocked  bool
	ownDebugState         *renderDebugState

	sem *renderObjectSemantics

	// DebugCreator is the object that created this render object, such as the
	// widget's element, for debug output.
	//
	// Flutter: rendering/object.dart RenderObject.debugCreator
	DebugCreator any
}

// InitRenderObject sets the outer object of a render object. The constructor of a
// type that embeds RenderObjectBase calls it (or the Init function of the struct
// it embeds, such as InitRenderBox).
//
// Flutter: no counterpart (the self pattern, SPEC §12.5)
func InitRenderObject(b *RenderObjectBase, self RenderObject) {
	b.self = self
	b.needsLayout = true
	b.needsPaint = true
	b.needsCompositing = self.IsRepaintBoundary() || self.AlwaysNeedsCompositing()
	b.wasRepaintBoundary = self.IsRepaintBoundary()
}

// AsRenderObject returns the base itself.
func (b *RenderObjectBase) AsRenderObject() *RenderObjectBase { return b }

func (b *RenderObjectBase) setChildModel(m renderChildModel) { b.model = m }

func (b *RenderObjectBase) dbg() *renderDebugState {
	if b.owner != nil {
		return b.owner.debugState
	}
	root := b
	for root.parent != nil {
		root = root.parent.AsRenderObject()
	}
	if root.ownDebugState == nil {
		root.ownDebugState = &renderDebugState{}
	}
	return root.ownDebugState
}

// DebugOptions are the debug flags of the owner; they are all off while the
// object has no owner.
//
// Flutter: rendering/debug.dart
func (b *RenderObjectBase) DebugOptions() *DebugOptions {
	if b.owner != nil {
		return b.owner.debug
	}
	return &DebugOptions{}
}

// Reassemble marks the object and its subtree as needing layout, compositing
// bits and paint, for a hot reload.
//
// Flutter: rendering/object.dart RenderObject.reassemble
func (b *RenderObjectBase) Reassemble() {
	b.self.MarkNeedsLayout()
	b.MarkNeedsCompositingBitsUpdate()
	b.MarkNeedsPaint()
	b.MarkNeedsSemanticsUpdate()
	b.self.VisitChildren(func(child RenderObject) { child.AsRenderObject().Reassemble() })
}

// DebugDisposed reports whether the object was disposed (debug builds only).
//
// Flutter: rendering/object.dart RenderObject.debugDisposed
func (b *RenderObjectBase) DebugDisposed() bool { return foundation.DebugMode && b.debugDisposed }

// Dispose lets go of the layer. A disposed object must not be used again; a
// second call panics with RenderObjectDisposedTwice in debug builds.
//
// Flutter: rendering/object.dart RenderObject.dispose
func (b *RenderObjectBase) Dispose() {
	if foundation.DebugMode && b.debugDisposed {
		panic(errors.New(tagRenderObjectDisposedTwice, "a render object was disposed twice",
			"object", foundation.DescribeIdentity(b.self)))
	}
	b.layerHandle.SetLayer(nil)
	if foundation.DebugMode {
		b.debugDisposed = true
	}
}

// ParentData is the data that the parent stores on this object.
//
// Flutter: rendering/object.dart RenderObject.parentData
func (b *RenderObjectBase) ParentData() ParentData { return b.parentData }

// SetParentData replaces the parent data. A parent calls it from SetupParentData.
//
// Flutter: rendering/object.dart RenderObject.parentData
func (b *RenderObjectBase) SetParentData(d ParentData) { b.parentData = d }

// SetupParentData gives the child a plain ParentData if it has none.
//
// Flutter: rendering/object.dart RenderObject.setupParentData
func (b *RenderObjectBase) SetupParentData(child RenderObject) {
	b.debugCheckCanPerformMutations()
	if child.AsRenderObject().parentData == nil {
		child.AsRenderObject().parentData = &ParentDataBase{}
	}
}

// Depth is the distance from the root of the tree; a parent has a smaller depth.
//
// Flutter: rendering/object.dart RenderObject.depth
func (b *RenderObjectBase) Depth() int { return b.depth }

// RedepthChild makes the child deeper than this object, if it is not.
//
// Flutter: rendering/object.dart RenderObject.redepthChild
func (b *RenderObjectBase) RedepthChild(child RenderObject) {
	cb := child.AsRenderObject()
	if foundation.DebugMode && cb.owner != b.owner {
		panic(errors.New(tagRenderObjectOwnerMismatch, "a child and its parent have different owners",
			"object", foundation.DescribeIdentity(b.self), "child", foundation.DescribeIdentity(child)))
	}
	if cb.depth <= b.depth {
		cb.depth = b.depth + 1
		child.RedepthChildren()
	}
}

// RedepthChildren updates the depth of the children, through the child model.
//
// Flutter: rendering/object.dart RenderObject.redepthChildren
func (b *RenderObjectBase) RedepthChildren() {
	if b.model != nil {
		b.model.redepthChildren()
	}
}

// Parent is the parent render object, or nil for the root.
//
// Flutter: rendering/object.dart RenderObject.parent
func (b *RenderObjectBase) Parent() RenderObject { return b.parent }

// AdoptChild makes the object the parent of the child: it sets up the parent
// data and marks the object for layout, compositing bits and semantics. Call it
// when a child is added. In debug builds it panics with RenderObjectCycle when
// the child is an ancestor, and with RenderObjectHasParent when it has a parent.
//
// Flutter: rendering/object.dart RenderObject.adoptChild
func (b *RenderObjectBase) AdoptChild(child RenderObject) {
	cb := child.AsRenderObject()
	if foundation.DebugMode {
		if cb.parent != nil {
			panic(errors.New(tagRenderObjectHasParent, "the child has a parent already",
				"object", foundation.DescribeIdentity(b.self), "child", foundation.DescribeIdentity(child)))
		}
		node := b.self
		for node.AsRenderObject().parent != nil {
			node = node.AsRenderObject().parent
		}
		if node == child {
			panic(errors.New(tagRenderObjectCycle, "adopting the child would make a cycle in the render tree",
				"object", foundation.DescribeIdentity(b.self), "child", foundation.DescribeIdentity(child)))
		}
	}
	b.self.SetupParentData(child)
	b.self.MarkNeedsLayout()
	b.MarkNeedsCompositingBitsUpdate()
	b.MarkNeedsSemanticsUpdate()
	cb.parent = b.self
	if b.Attached() {
		child.Attach(b.owner)
	}
	b.RedepthChild(child)
}

// DropChild removes the child from the object: it detaches the parent data and
// marks the object for layout, compositing bits and semantics. Call it when a
// child is removed.
//
// Flutter: rendering/object.dart RenderObject.dropChild
func (b *RenderObjectBase) DropChild(child RenderObject) {
	cb := child.AsRenderObject()
	if foundation.DebugMode {
		if cb.parent != b.self || cb.Attached() != b.Attached() || cb.parentData == nil {
			panic(errors.New(tagRenderObjectNotChild, "the render object is not a child of this one, or it has no parent data",
				"object", foundation.DescribeIdentity(b.self), "child", foundation.DescribeIdentity(child)))
		}
	}
	if !cb.isRelayoutBoundary.OrElse(true) {
		cb.isRelayoutBoundary = foundation.None[bool]()
	}
	cb.parentData.Detach()
	cb.parentData = nil
	cb.parent = nil
	if b.Attached() {
		child.Detach()
	}
	b.self.MarkNeedsLayout()
	b.MarkNeedsCompositingBitsUpdate()
	b.MarkNeedsSemanticsUpdate()
}

// VisitChildren calls the visitor with each child, through the child model.
//
// Flutter: rendering/object.dart RenderObject.visitChildren
func (b *RenderObjectBase) VisitChildren(visitor RenderObjectVisitor) {
	if b.model != nil {
		b.model.visitChildren(visitor)
	}
}

// DebugDoingThisResize reports whether PerformResize is running (debug builds).
//
// Flutter: rendering/object.dart RenderObject.debugDoingThisResize
func (b *RenderObjectBase) DebugDoingThisResize() bool { return b.debugDoingThisResize }

// DebugDoingThisLayout reports whether PerformLayout is running (debug builds).
//
// Flutter: rendering/object.dart RenderObject.debugDoingThisLayout
func (b *RenderObjectBase) DebugDoingThisLayout() bool { return b.debugDoingThisLayout }

// DebugActiveLayout is the render object whose layout is running, in the tree
// of this object (debug builds).
//
// Flutter: rendering/object.dart RenderObject.debugActiveLayout
func (b *RenderObjectBase) DebugActiveLayout() RenderObject { return b.dbg().activeLayout }

// DebugCanParentUseSize reports whether the parent said it uses the size
// (debug builds).
//
// Flutter: rendering/object.dart RenderObject.debugCanParentUseSize
func (b *RenderObjectBase) DebugCanParentUseSize() bool { return b.debugCanParentUseSize.OrElse(false) }

// DebugDoingThisLayoutWithCallback reports whether the object runs a layout
// callback.
//
// Flutter: rendering/object.dart RenderObject.debugDoingThisLayoutWithCallback
func (b *RenderObjectBase) DebugDoingThisLayoutWithCallback() bool {
	return b.doingThisLayoutWithCallback
}

func (b *RenderObjectBase) debugClosestMutationRoot() (RenderObject, bool, bool) {
	switch {
	case b.doingThisLayoutWithCallback:
		return b.self, true, true
	case b.owner != nil && b.owner.debugAllowMutations && b.needsLayout:
		return b.self, true, true
	case b.debugMutationsLocked:
		return b.self, false, true
	}
	if p := b.self.DebugLayoutParent(); p != nil {
		return p.AsRenderObject().debugClosestMutationRoot()
	}
	return nil, false, false
}

// DebugLayoutParent is the parent in the layout protocol (debug builds).
//
// Flutter: rendering/object.dart RenderObject.debugLayoutParent
func (b *RenderObjectBase) DebugLayoutParent() RenderObject {
	if !foundation.DebugMode {
		return nil
	}
	return b.parent
}

// debugCheckCanPerformMutations panics in debug builds if the object is
// disposed, or is changed in a way that layout forbids (tags
// RenderObjectDisposedMutated and RenderObjectMutatedDuringLayout).
func (b *RenderObjectBase) debugCheckCanPerformMutations() bool {
	if !foundation.DebugMode {
		return true
	}
	if b.debugDisposed {
		panic(errors.New(tagRenderObjectDisposedMutated, "a disposed render object was mutated",
			"object", foundation.DescribeIdentity(b.self)))
	}
	if b.owner == nil || !b.owner.debugDoingLayout {
		return true
	}
	root, allowed, found := b.debugClosestMutationRoot()
	if found && allowed {
		return true
	}
	active := b.dbg().activeLayout
	method := "performResize"
	if active != nil && active.AsRenderObject().debugDoingThisLayout {
		method = "performLayout"
	}
	who := "none"
	if active != nil {
		who = foundation.RuntimeTypeName(active) + "." + method
	}
	var msg string
	switch {
	case !found || root == nil:
		msg = "the render object was mutated when none of its ancestors is actively performing layout"
	case root == RenderObject(b.self):
		msg = "a render object must not re-dirty itself while still being laid out; consider using a LayoutBuilder to change a subtree during layout"
	case root == active:
		msg = "a render object must not mutate its descendants in its " + method + " method"
	default:
		msg = "a render object must not mutate another render object from a different render subtree in its " + method +
			" method; mutating the layout of another render object may cause some of the render objects in its subtree to be laid out more than once"
	}
	panic(errors.New(tagRenderObjectMutatedDuringLayout, foundation.RuntimeTypeName(b.self)+" was mutated in "+who+": "+msg,
		"object", foundation.DescribeIdentity(b.self), "mutatedBy", foundation.DescribeIdentity(active)))
}

// Owner is the pipeline owner that the object is attached to, or nil.
//
// Flutter: rendering/object.dart RenderObject.owner
func (b *RenderObjectBase) Owner() *PipelineOwner { return b.owner }

// Attached reports whether the object has an owner.
//
// Flutter: rendering/object.dart RenderObject.attached
func (b *RenderObjectBase) Attached() bool { return b.owner != nil }

// Attach attaches the object to the owner, and its children through the child
// model. Dirty flags that were set while detached are set again, so that the new
// owner hears of them.
//
// Flutter: rendering/object.dart RenderObject.attach
func (b *RenderObjectBase) Attach(owner *PipelineOwner) {
	if foundation.DebugMode {
		if b.debugDisposed {
			panic(errors.New(tagRenderObjectDisposedMutated, "a disposed render object was attached",
				"object", foundation.DescribeIdentity(b.self)))
		}
		if b.owner != nil {
			panic(errors.New(tagRenderObjectAttached, "the render object is attached to an owner already",
				"object", foundation.DescribeIdentity(b.self)))
		}
	}
	b.owner = owner
	if b.needsLayout && b.isRelayoutBoundary.IsSome() {
		b.needsLayout = false
		b.self.MarkNeedsLayout()
	}
	if b.needsCompositingBitsUpdate {
		b.needsCompositingBitsUpdate = false
		b.MarkNeedsCompositingBitsUpdate()
	}
	if b.needsPaint && !isNil(b.layerHandle.Layer()) {
		b.needsPaint = false
		b.MarkNeedsPaint()
	}
	if sem := b.semantics(); sem.configProv.effective().IsSemanticBoundary() && (sem.parentDataDirty() || !sem.built) {
		b.self.MarkNeedsSemanticsUpdate()
	}
	if b.model != nil {
		b.model.attachChildren(owner)
	}
}

// Detach clears the owner of the object and of its children.
//
// Flutter: rendering/object.dart RenderObject.detach
func (b *RenderObjectBase) Detach() {
	if foundation.DebugMode && b.owner == nil {
		panic(errors.New(tagRenderObjectNotAttached, "a render object that is not attached was detached",
			"object", foundation.DescribeIdentity(b.self)))
	}
	b.owner = nil
	if b.model != nil {
		b.model.detachChildren()
	}
}

// DebugNeedsLayout reports whether the object needs layout (debug builds only).
//
// Flutter: rendering/object.dart RenderObject.debugNeedsLayout
func (b *RenderObjectBase) DebugNeedsLayout() bool { return foundation.DebugMode && b.needsLayout }

// Constraints are the constraints that the parent gave at the last layout. It
// panics with RenderObjectNoConstraints before the first layout.
//
// Flutter: rendering/object.dart RenderObject.constraints
func (b *RenderObjectBase) Constraints() Constraints {
	if b.constraints == nil {
		panic(errors.New(tagRenderObjectNoConstraints, "a render object does not have any constraints before it has been laid out",
			"object", foundation.DescribeIdentity(b.self)))
	}
	return b.constraints
}

// DebugAssertDoesMeetConstraints does nothing; a box checks its size.
//
// Flutter: rendering/object.dart RenderObject.debugAssertDoesMeetConstraints
func (b *RenderObjectBase) DebugAssertDoesMeetConstraints() {}

// MarkNeedsLayout marks the object as needing layout. If it is a relayout
// boundary it asks its owner to lay it out; otherwise it marks its parent too.
//
// Flutter: rendering/object.dart RenderObject.markNeedsLayout
func (b *RenderObjectBase) MarkNeedsLayout() {
	if foundation.DebugMode && b.owner != nil {
		b.owner.debug.UIThread.AssertCurrent("RenderObject.MarkNeedsLayout")
	}
	b.debugCheckCanPerformMutations()
	if b.needsLayout {
		return
	}
	b.needsLayout = true
	if b.owner != nil && b.isRelayoutBoundary.OrElse(false) {
		if foundation.DebugMode && b.owner.debug.PrintMarkNeedsLayoutStacks {
			b.owner.debug.DebugPrint("markNeedsLayout() called for " + b.self.(foundation.ShortDescriber).ToStringShort())
		}
		b.owner.nodesNeedingLayout = append(b.owner.nodesNeedingLayout, b.self)
		b.owner.RequestVisualUpdate()
	} else if b.parent != nil {
		b.MarkParentNeedsLayout()
	}
}

// MarkParentNeedsLayout marks the object and its parent as needing layout.
//
// Flutter: rendering/object.dart RenderObject.markParentNeedsLayout
func (b *RenderObjectBase) MarkParentNeedsLayout() {
	b.debugCheckCanPerformMutations()
	b.needsLayout = true
	if foundation.DebugMode && b.parent == nil {
		panic(errors.New(tagRenderObjectNoParent, "MarkParentNeedsLayout was called on a render object without a parent",
			"object", foundation.DescribeIdentity(b.self)))
	}
	parent := b.parent
	if !b.doingThisLayoutWithCallback {
		parent.MarkNeedsLayout()
	}
}

// MarkNeedsLayoutForSizedByParentChange marks the object and its parent for
// layout; call it when SizedByParent changes.
//
// Flutter: rendering/object.dart RenderObject.markNeedsLayoutForSizedByParentChange
func (b *RenderObjectBase) MarkNeedsLayoutForSizedByParentChange() {
	b.self.MarkNeedsLayout()
	b.MarkParentNeedsLayout()
}

// ScheduleInitialLayout lays out the root of a tree for the first time, on the
// next flush. The object must be attached, have no parent, and have never been
// laid out.
//
// Flutter: rendering/object.dart RenderObject.scheduleInitialLayout
func (b *RenderObjectBase) ScheduleInitialLayout() {
	if foundation.DebugMode {
		if b.debugDisposed || !b.Attached() || b.parent != nil || b.owner.debugDoingLayout || b.isRelayoutBoundary.IsSome() {
			panic(errors.New(tagScheduleInitialLayoutInvalid,
				"the initial layout can only be scheduled for an attached root that has not been laid out",
				"object", foundation.DescribeIdentity(b.self)))
		}
		b.debugCanParentUseSize = foundation.Some(false)
	}
	b.isRelayoutBoundary = foundation.Some(true)
	b.owner.nodesNeedingLayout = append(b.owner.nodesNeedingLayout, b.self)
}

// layoutWithoutResize lays out a relayout boundary again with the same
// constraints, without calling PerformResize.
//
// A panic in PerformLayout is reported through the debug options' ReportError
// with the tag LayoutPanicked, and the layout goes on (SPEC §14.2).
func (b *RenderObjectBase) layoutWithoutResize() {
	var previousActive RenderObject
	if foundation.DebugMode {
		b.debugMutationsLocked = true
		b.debugDoingThisLayout = true
		previousActive = b.dbg().activeLayout
		b.dbg().activeLayout = b.self
		if b.DebugOptions().PrintLayouts {
			b.DebugOptions().DebugPrint("Laying out (without resize) " + foundation.ToStringShort(b.self))
		}
	}
	b.guard(foundation.LayoutPanicked, "PerformLayout", func() {
		b.self.PerformLayout()
		b.MarkNeedsSemanticsUpdate()
	})
	if foundation.DebugMode {
		b.dbg().activeLayout = previousActive
		b.debugDoingThisLayout = false
		b.debugMutationsLocked = false
	}
	b.needsLayout = false
	b.MarkNeedsPaint()
}

// guard runs fn and, when the debug options have a ReportError, recovers from a
// panic in it and reports it with the tag and the method that was running. The
// object is added to the report. Without a reporter the panic goes on.
//
// Flutter: rendering/object.dart RenderObject._reportException
func (b *RenderObjectBase) guard(tag, method string, fn func()) {
	foundation.Guard(b.DebugOptions().ReportError, tag, "during "+method+"()", func() []any {
		kv := []any{"library", "rendering library", "method", method, "object", foundation.DescribeIdentity(b.self)}
		if b.DebugCreator != nil {
			kv = append(kv, "creator", fmt.Sprint(b.DebugCreator))
		}
		return kv
	}, fn)
}

// Layout computes the layout of the object for the constraints. The parent
// calls it on each child from its own PerformLayout. Pass ParentUsesSize when
// the parent reads the size of the child.
//
// Layout is a recovery point (SPEC §14.2): a panic in PerformResize or
// PerformLayout is reported through the debug options' ReportError with the tag
// LayoutPanicked, and the layout goes on.
//
// Flutter: rendering/object.dart RenderObject.layout
func (b *RenderObjectBase) Layout(constraints Constraints, opts ...LayoutOpts) {
	o := firstOpts(opts)
	self := b.self
	if foundation.DebugMode {
		if b.debugDisposed {
			panic(errors.New(tagRenderObjectDisposedMutated, "a disposed render object was laid out",
				"object", foundation.DescribeIdentity(self)))
		}
		constraints.DebugAssertIsValid(DebugAssertIsValidOpts{
			IsAppliedConstraint: true,
			InformationCollector: func() []string {
				return []string{"These invalid constraints were provided to " + foundation.RuntimeTypeName(self) +
					"'s layout() function by its parent."}
			},
		})
		if b.debugDoingThisResize || b.debugDoingThisLayout {
			panic(errors.New(tagLayoutReentrant, "a render object was laid out while it was laying out",
				"object", foundation.DescribeIdentity(self)))
		}
		b.debugCanParentUseSize = foundation.Some(o.ParentUsesSize)
	}
	b.isRelayoutBoundary = foundation.Some(!o.ParentUsesSize || self.SizedByParent() || constraints.IsTight() || b.parent == nil)
	if !b.needsLayout && constraints == b.constraints {
		if foundation.DebugMode {
			b.debugDoingThisResize = self.SizedByParent()
			b.debugDoingThisLayout = !self.SizedByParent()
			previous := b.dbg().activeLayout
			b.dbg().activeLayout = self
			self.DebugResetSize()
			b.dbg().activeLayout = previous
			b.debugDoingThisLayout = false
			b.debugDoingThisResize = false
		}
		return
	}
	b.constraints = constraints
	if foundation.DebugMode {
		if b.debugMutationsLocked || b.doingThisLayoutWithCallback {
			panic(errors.New(tagLayoutReentrant, "a render object was laid out while its mutations were locked",
				"object", foundation.DescribeIdentity(self)))
		}
		b.debugMutationsLocked = true
		if b.DebugOptions().PrintLayouts {
			how := "with resize allowed"
			if self.SizedByParent() {
				how = "with separate resize"
			}
			b.DebugOptions().DebugPrint("Laying out (" + how + ") " + foundation.ToStringShort(self))
		}
	}
	if self.SizedByParent() {
		if foundation.DebugMode {
			b.debugDoingThisResize = true
		}
		b.guard(foundation.LayoutPanicked, "PerformResize", func() {
			self.PerformResize()
			if foundation.DebugMode {
				self.DebugAssertDoesMeetConstraints()
			}
		})
		if foundation.DebugMode {
			b.debugDoingThisResize = false
		}
	}
	var previousActive RenderObject
	if foundation.DebugMode {
		b.debugDoingThisLayout = true
		previousActive = b.dbg().activeLayout
		b.dbg().activeLayout = self
	}
	b.guard(foundation.LayoutPanicked, "PerformLayout", func() {
		self.PerformLayout()
		b.MarkNeedsSemanticsUpdate()
		if foundation.DebugMode {
			self.DebugAssertDoesMeetConstraints()
		}
	})
	if foundation.DebugMode {
		b.dbg().activeLayout = previousActive
		b.debugDoingThisLayout = false
		b.debugMutationsLocked = false
	}
	b.needsLayout = false
	b.MarkNeedsPaint()
}

// DebugResetSize does nothing; a box sets its size again to check it.
//
// Flutter: rendering/object.dart RenderObject.debugResetSize
func (b *RenderObjectBase) DebugResetSize() {}

// SizedByParent is false: the object computes its size in PerformLayout.
//
// Flutter: rendering/object.dart RenderObject.sizedByParent
func (b *RenderObjectBase) SizedByParent() bool { return false }

// PerformResize is abstract: it panics with RenderObjectAbstractMethod.
//
// Flutter: rendering/object.dart RenderObject.performResize
func (b *RenderObjectBase) PerformResize() {
	panic(errors.New(tagRenderObjectAbstractMethod, "the render object type does not implement PerformResize",
		"object", foundation.DescribeIdentity(b.self), "method", "PerformResize"))
}

// PerformLayout is abstract: it panics with RenderObjectAbstractMethod.
//
// Flutter: rendering/object.dart RenderObject.performLayout
func (b *RenderObjectBase) PerformLayout() {
	panic(errors.New(tagRenderObjectAbstractMethod, "the render object type does not implement PerformLayout",
		"object", foundation.DescribeIdentity(b.self), "method", "PerformLayout"))
}

// InvokeLayoutCallback runs the callback with the constraints, and lets it change
// the subtree of the object, which layout forbids otherwise. Call it from
// PerformLayout.
//
// Flutter: rendering/object.dart RenderObject.invokeLayoutCallback
func (b *RenderObjectBase) InvokeLayoutCallback(callback LayoutCallback) {
	if foundation.DebugMode && (!b.debugMutationsLocked || !b.debugDoingThisLayout || b.doingThisLayoutWithCallback) {
		panic(errors.New(tagInvokeLayoutCallbackInvalid, "a layout callback can only be invoked from PerformLayout, once",
			"object", foundation.DescribeIdentity(b.self)))
	}
	b.doingThisLayoutWithCallback = true
	defer func() { b.doingThisLayoutWithCallback = false }()
	b.owner.enableMutationsToDirtySubtrees(func() { callback(b.Constraints()) })
}

// DebugDoingThisPaint reports whether Paint is running (debug builds).
//
// Flutter: rendering/object.dart RenderObject.debugDoingThisPaint
func (b *RenderObjectBase) DebugDoingThisPaint() bool { return b.debugDoingThisPaint }

// DebugActivePaint is the render object that is painting, in the tree of this
// object (debug builds).
//
// Flutter: rendering/object.dart RenderObject.debugActivePaint
func (b *RenderObjectBase) DebugActivePaint() RenderObject { return b.dbg().activePaint }

// IsRepaintBoundary is false: the object paints into its parent's layer.
//
// Flutter: rendering/object.dart RenderObject.isRepaintBoundary
func (b *RenderObjectBase) IsRepaintBoundary() bool { return false }

// DebugRegisterRepaintBoundaryPaint does nothing.
//
// Flutter: rendering/object.dart RenderObject.debugRegisterRepaintBoundaryPaint
func (b *RenderObjectBase) DebugRegisterRepaintBoundaryPaint(includedParent, includedChild bool) {}

// AlwaysNeedsCompositing is false.
//
// Flutter: rendering/object.dart RenderObject.alwaysNeedsCompositing
func (b *RenderObjectBase) AlwaysNeedsCompositing() bool { return false }

// UpdateCompositedLayer returns the old layer, or a new OffsetLayer. An object
// that is a repaint boundary overrides it to set properties on its layer.
//
// Flutter: rendering/object.dart RenderObject.updateCompositedLayer
func (b *RenderObjectBase) UpdateCompositedLayer(oldLayer OffsetLayerLike) OffsetLayerLike {
	if foundation.DebugMode && !b.self.IsRepaintBoundary() {
		panic(errors.New(tagUpdateCompositedLayerNotBoundary, "UpdateCompositedLayer was called on a render object that is not a repaint boundary",
			"object", foundation.DescribeIdentity(b.self)))
	}
	if oldLayer != nil {
		return oldLayer
	}
	return NewOffsetLayer()
}

// Layer is the layer of the object: that of a repaint boundary, or the one
// that a non-boundary object keeps with SetLayer.
//
// Flutter: rendering/object.dart RenderObject.layer
func (b *RenderObjectBase) Layer() ContainerLayerLike {
	return b.layerHandle.Layer()
}

// SetLayer sets the layer of an object that is not a repaint boundary. Pass nil
// to let it go. A repaint boundary panics with RenderObjectBoundaryLayer in debug
// builds: the framework gives it its layer.
//
// Flutter: rendering/object.dart RenderObject.layer
func (b *RenderObjectBase) SetLayer(l ContainerLayerLike) {
	if foundation.DebugMode && b.self.IsRepaintBoundary() {
		panic(errors.New(tagRenderObjectBoundaryLayer,
			"attempted to set a layer to a repaint boundary render object; the framework creates and assigns an OffsetLayer to a repaint boundary automatically",
			"object", foundation.DescribeIdentity(b.self)))
	}
	b.layerHandle.SetLayer(l)
}

// DebugLayer is the layer of the object (debug builds only; nil otherwise).
//
// Flutter: rendering/object.dart RenderObject.debugLayer
func (b *RenderObjectBase) DebugLayer() ContainerLayerLike {
	if !foundation.DebugMode {
		return nil
	}
	return b.layerHandle.Layer()
}

// MarkNeedsCompositingBitsUpdate marks the object so that the next flush
// recomputes whether it needs compositing.
//
// Flutter: rendering/object.dart RenderObject.markNeedsCompositingBitsUpdate
func (b *RenderObjectBase) MarkNeedsCompositingBitsUpdate() {
	if foundation.DebugMode && b.debugDisposed {
		panic(errors.New(tagRenderObjectDisposedMutated, "a disposed render object was marked as needing compositing bits",
			"object", foundation.DescribeIdentity(b.self)))
	}
	if b.needsCompositingBitsUpdate {
		return
	}
	b.needsCompositingBitsUpdate = true
	if b.parent != nil {
		pb := b.parent.AsRenderObject()
		if pb.needsCompositingBitsUpdate {
			return
		}
		if (!b.wasRepaintBoundary || !b.self.IsRepaintBoundary()) && !b.parent.IsRepaintBoundary() {
			pb.MarkNeedsCompositingBitsUpdate()
			return
		}
	}
	if b.owner != nil {
		b.owner.nodesNeedingCompositing = append(b.owner.nodesNeedingCompositing, b.self)
	}
}

// NeedsCompositing reports whether the object or a descendant needs a layer; it
// panics with RenderObjectCompositingBitsDirty in debug builds if the bit is
// dirty.
//
// Flutter: rendering/object.dart RenderObject.needsCompositing
func (b *RenderObjectBase) NeedsCompositing() bool {
	if foundation.DebugMode && b.needsCompositingBitsUpdate {
		panic(errors.New(tagRenderObjectCompositingBitsDirty, "NeedsCompositing was read while the compositing bits were dirty",
			"object", foundation.DescribeIdentity(b.self)))
	}
	return b.needsCompositing
}

func (b *RenderObjectBase) updateCompositingBits() {
	if !b.needsCompositingBitsUpdate {
		return
	}
	old := b.needsCompositing
	b.needsCompositing = false
	b.self.VisitChildren(func(child RenderObject) {
		cb := child.AsRenderObject()
		cb.updateCompositingBits()
		if cb.NeedsCompositing() {
			b.needsCompositing = true
		}
	})
	if b.self.IsRepaintBoundary() || b.self.AlwaysNeedsCompositing() {
		b.needsCompositing = true
	}
	switch {
	case !b.self.IsRepaintBoundary() && b.wasRepaintBoundary:
		b.needsPaint = false
		b.needsCompositedLayerUpdate = false
		if b.owner != nil {
			nodes := b.owner.nodesNeedingPaint[:0]
			for _, n := range b.owner.nodesNeedingPaint {
				if n != b.self {
					nodes = append(nodes, n)
				}
			}
			b.owner.nodesNeedingPaint = nodes
		}
		b.needsCompositingBitsUpdate = false
		b.MarkNeedsPaint()
	case old != b.needsCompositing:
		b.needsCompositingBitsUpdate = false
		b.MarkNeedsPaint()
	default:
		b.needsCompositingBitsUpdate = false
	}
}

// DebugNeedsPaint reports whether the object needs paint (debug builds only).
//
// Flutter: rendering/object.dart RenderObject.debugNeedsPaint
func (b *RenderObjectBase) DebugNeedsPaint() bool { return foundation.DebugMode && b.needsPaint }

// DebugNeedsCompositedLayerUpdate reports whether the layer of the repaint
// boundary needs to be updated (debug builds only).
//
// Flutter: rendering/object.dart RenderObject.debugNeedsCompositedLayerUpdate
func (b *RenderObjectBase) DebugNeedsCompositedLayerUpdate() bool {
	return foundation.DebugMode && b.needsCompositedLayerUpdate
}

// MarkNeedsPaint marks the object as needing paint: the nearest repaint
// boundary above it (or itself) is added to the owner's list. A debug build
// panics with MarkNeedsPaintDuringPaint if the owner is painting.
//
// Flutter: rendering/object.dart RenderObject.markNeedsPaint
func (b *RenderObjectBase) MarkNeedsPaint() {
	if foundation.DebugMode {
		if b.debugDisposed {
			panic(errors.New(tagRenderObjectDisposedMutated, "a disposed render object was marked as needing paint",
				"object", foundation.DescribeIdentity(b.self)))
		}
		if b.owner != nil && b.owner.debugDoingPaint {
			panic(errors.New(tagMarkNeedsPaintDuringPaint, "MarkNeedsPaint was called while painting",
				"object", foundation.DescribeIdentity(b.self)))
		}
	}
	if b.needsPaint {
		return
	}
	b.needsPaint = true
	switch {
	case b.self.IsRepaintBoundary() && b.wasRepaintBoundary:
		if foundation.DebugMode {
			if b.DebugOptions().PrintMarkNeedsPaintStacks {
				b.DebugOptions().DebugPrint("markNeedsPaint() called for " + foundation.ToStringShort(b.self))
			}
		}
		if b.owner != nil {
			b.owner.nodesNeedingPaint = append(b.owner.nodesNeedingPaint, b.self)
			b.owner.RequestVisualUpdate()
		}
	case b.parent != nil:
		b.parent.AsRenderObject().MarkNeedsPaint()
	default:
		if foundation.DebugMode && b.DebugOptions().PrintMarkNeedsPaintStacks {
			b.DebugOptions().DebugPrint("markNeedsPaint() called for " + foundation.ToStringShort(b.self) + " (root of render tree)")
		}
		if b.owner != nil {
			b.owner.RequestVisualUpdate()
		}
	}
}

// MarkNeedsPaintQuietly marks the object as needing paint like MarkNeedsPaint, but
// does not ask for a frame: the next frame that something else asks for paints it. It
// is for what shows data that arrives between frames and must not keep the app drawing
// (the performance overlay).
//
// Flutter: no counterpart
func (b *RenderObjectBase) MarkNeedsPaintQuietly() {
	if b.owner == nil {
		b.MarkNeedsPaint()
		return
	}
	b.owner.quiet++
	defer func() { b.owner.quiet-- }()
	b.MarkNeedsPaint()
}

// MarkNeedsCompositedLayerUpdate marks a repaint boundary as needing only an
// update of its layer's properties, not a repaint.
//
// Flutter: rendering/object.dart RenderObject.markNeedsCompositedLayerUpdate
func (b *RenderObjectBase) MarkNeedsCompositedLayerUpdate() {
	if foundation.DebugMode {
		if b.debugDisposed {
			panic(errors.New(tagRenderObjectDisposedMutated, "a disposed render object was marked as needing a layer update",
				"object", foundation.DescribeIdentity(b.self)))
		}
		if b.owner != nil && b.owner.debugDoingPaint {
			panic(errors.New(tagMarkNeedsPaintDuringPaint, "MarkNeedsCompositedLayerUpdate was called while painting",
				"object", foundation.DescribeIdentity(b.self)))
		}
	}
	if b.needsCompositedLayerUpdate || b.needsPaint {
		return
	}
	b.needsCompositedLayerUpdate = true
	if b.self.IsRepaintBoundary() && b.wasRepaintBoundary {
		if b.owner != nil {
			b.owner.nodesNeedingPaint = append(b.owner.nodesNeedingPaint, b.self)
			b.owner.RequestVisualUpdate()
		}
	} else {
		b.MarkNeedsPaint()
	}
}

// skippedPaintingOnLayer is called when a repaint boundary was not painted
// because its layer is detached: the repaint boundaries above are marked so that
// they paint when the layer is attached again.
func (b *RenderObjectBase) skippedPaintingOnLayer() {
	node := b.parent
	for node != nil {
		nb := node.AsRenderObject()
		if node.IsRepaintBoundary() {
			l := nb.layerHandle.Layer()
			if isNil(l) || l.AsLayer().Attached() {
				break
			}
			nb.needsPaint = true
		}
		node = nb.parent
	}
}

// ScheduleInitialPaint paints the root of a tree for the first time, into the
// root layer, on the next flush.
//
// Flutter: rendering/object.dart RenderObject.scheduleInitialPaint
func (b *RenderObjectBase) ScheduleInitialPaint(rootLayer ContainerLayerLike) {
	if foundation.DebugMode && (!rootLayer.AsLayer().Attached() || !b.Attached() || b.parent != nil ||
		b.owner.debugDoingPaint || !b.self.IsRepaintBoundary() || !isNil(b.layerHandle.Layer()) || !b.needsPaint) {
		panic(errors.New(tagScheduleInitialPaintInvalid,
			"the initial paint can only be scheduled for an attached repaint boundary root that has not painted",
			"object", foundation.DescribeIdentity(b.self)))
	}
	b.layerHandle.SetLayer(rootLayer)
	b.owner.nodesNeedingPaint = append(b.owner.nodesNeedingPaint, b.self)
}

// ReplaceRootLayer replaces the layer of the root render object, after the first
// ScheduleInitialPaint.
//
// Flutter: rendering/object.dart RenderObject.replaceRootLayer
func (b *RenderObjectBase) ReplaceRootLayer(rootLayer OffsetLayerLike) {
	if foundation.DebugMode && (b.debugDisposed || !rootLayer.AsLayer().Attached() || !b.Attached() || b.parent != nil ||
		b.owner.debugDoingPaint || !b.self.IsRepaintBoundary() || isNil(b.layerHandle.Layer())) {
		panic(errors.New(tagReplaceRootLayerInvalid, "the root layer can only be replaced for an attached repaint boundary root that has a layer",
			"object", foundation.DescribeIdentity(b.self)))
	}
	b.layerHandle.Layer().Detach()
	b.layerHandle.SetLayer(rootLayer)
	b.MarkNeedsPaint()
}

// paintWithContext paints the object into the context, unless it needs layout.
//
// A panic in Paint is reported through the debug options' ReportError with the
// tag PaintPanicked, and the painting goes on (SPEC §14.2).
func (b *RenderObjectBase) paintWithContext(context *PaintingContext, offset ui.Offset) {
	self := b.self
	if foundation.DebugMode {
		if b.debugDisposed {
			panic(errors.New(tagRenderObjectDisposedMutated, "a disposed render object was painted",
				"object", foundation.DescribeIdentity(self)))
		}
		if b.debugDoingThisPaint {
			panic(errors.New(tagPaintReentrant, "tried to paint a render object reentrantly; this typically indicates an infinite recursion",
				"object", foundation.DescribeIdentity(self)))
		}
	}
	if b.needsLayout {
		return
	}
	if foundation.DebugMode && b.needsCompositingBitsUpdate {
		panic(errors.New(tagPaintBeforeCompositingBits, "tried to paint a render object before its compositing bits were updated; "+
			"the tree has not yet been properly configured for creating the layer tree",
			"object", foundation.DescribeIdentity(self)))
	}
	var lastActivePaint RenderObject
	if foundation.DebugMode {
		b.debugDoingThisPaint = true
		lastActivePaint = b.dbg().activePaint
		b.dbg().activePaint = self
	}
	b.needsPaint = false
	b.needsCompositedLayerUpdate = false
	b.wasRepaintBoundary = self.IsRepaintBoundary()
	b.guard(foundation.PaintPanicked, "paint", func() {
		self.Paint(context, offset)
		if foundation.DebugMode && (b.needsLayout || b.needsPaint) {
			panic(errors.New(tagPaintMarkedDirty, "Paint marked the render object as needing layout or paint again",
				"object", foundation.DescribeIdentity(self)))
		}
	})
	if foundation.DebugMode {
		self.DebugPaint(context, offset)
		b.dbg().activePaint = lastActivePaint
		b.debugDoingThisPaint = false
	}
}

// PaintBounds is abstract: it panics with RenderObjectAbstractMethod.
//
// Flutter: rendering/object.dart RenderObject.paintBounds
func (b *RenderObjectBase) PaintBounds() ui.Rect {
	panic(errors.New(tagRenderObjectAbstractMethod, "the render object type does not implement PaintBounds",
		"object", foundation.DescribeIdentity(b.self), "method", "PaintBounds"))
}

// SemanticBounds is abstract: it panics with RenderObjectAbstractMethod.
//
// Flutter: rendering/object.dart RenderObject.semanticBounds
func (b *RenderObjectBase) SemanticBounds() ui.Rect {
	panic(errors.New(tagRenderObjectAbstractMethod, "the render object type does not implement SemanticBounds",
		"object", foundation.DescribeIdentity(b.self), "method", "SemanticBounds"))
}

// DebugPaint does nothing; a box paints the debug overlays.
//
// Flutter: rendering/object.dart RenderObject.debugPaint
func (b *RenderObjectBase) DebugPaint(*PaintingContext, ui.Offset) {}

// Paint does nothing: an object with nothing to paint, or only children whose
// parent paints them.
//
// Flutter: rendering/object.dart RenderObject.paint
func (b *RenderObjectBase) Paint(*PaintingContext, ui.Offset) {}

// ApplyPaintTransform does nothing: the child is painted without a transform.
//
// Flutter: rendering/object.dart RenderObject.applyPaintTransform
func (b *RenderObjectBase) ApplyPaintTransform(child RenderObject, transform *ui.Matrix4) {
	if foundation.DebugMode && child.AsRenderObject().parent != b.self {
		panic(errors.New(tagRenderObjectNotChild, "ApplyPaintTransform was called with a render object that is not a child",
			"object", foundation.DescribeIdentity(b.self), "child", foundation.DescribeIdentity(child)))
	}
}

// PaintsChild is true: the child is painted.
//
// Flutter: rendering/object.dart RenderObject.paintsChild
func (b *RenderObjectBase) PaintsChild(child RenderObject) bool {
	if foundation.DebugMode && child.AsRenderObject().parent != b.self {
		panic(errors.New(tagRenderObjectNotChild, "PaintsChild was called with a render object that is not a child",
			"object", foundation.DescribeIdentity(b.self), "child", foundation.DescribeIdentity(child)))
	}
	return true
}

// GetTransformTo returns the transform from the coordinates of this object to
// those of the target, an ancestor or any object in the same tree; nil means the
// root. The object must be attached. If the transform can not be inverted it
// returns the zero matrix.
//
// Flutter: rendering/object.dart RenderObject.getTransformTo
func (b *RenderObjectBase) GetTransformTo(target RenderObject) *ui.Matrix4 {
	if foundation.DebugMode && !b.Attached() {
		panic(errors.New(tagGetTransformToDetached, "GetTransformTo needs an attached render object",
			"object", foundation.DescribeIdentity(b.self)))
	}
	var fromPath, toPath []RenderObject
	from := b.self
	var to RenderObject
	if target != nil {
		to = target
	} else {
		to = b.owner.rootNode
	}
	notSameTree := func() *errors.Error {
		return errors.New(tagRenderObjectsNotInSameTree, "the render objects are not in the same render tree",
			"target", foundation.DescribeIdentity(target), "object", foundation.DescribeIdentity(b.self))
	}
	for from != to {
		fd, td := from.AsRenderObject().depth, to.AsRenderObject().depth
		if fd >= td {
			fp := from.AsRenderObject().parent
			if fp == nil {
				panic(notSameTree())
			}
			if fromPath == nil {
				fromPath = []RenderObject{b.self}
			}
			fromPath = append(fromPath, fp)
			from = fp
		}
		if fd <= td {
			tp := to.AsRenderObject().parent
			if tp == nil {
				panic(notSameTree())
			}
			if target == nil {
				panic(notSameTree())
			}
			if toPath == nil {
				toPath = []RenderObject{target}
			}
			toPath = append(toPath, tp)
			to = tp
		}
	}
	var fromTransform *ui.Matrix4
	if fromPath != nil {
		fromTransform = ui.NewMatrix4Identity()
		last := len(fromPath) - 1
		if target == nil {
			last = len(fromPath) - 2
		}
		for i := last; i > 0; i-- {
			fromPath[i].ApplyPaintTransform(fromPath[i-1], fromTransform)
		}
	}
	if toPath == nil {
		if fromTransform != nil {
			return fromTransform
		}
		return ui.NewMatrix4Identity()
	}
	toTransform := ui.NewMatrix4Identity()
	for i := len(toPath) - 1; i > 0; i-- {
		toPath[i].ApplyPaintTransform(toPath[i-1], toTransform)
	}
	if toTransform.Invert() == 0 {
		return ui.NewMatrix4Zero()
	}
	if fromTransform != nil {
		fromTransform.MultiplyBy(toTransform)
		return fromTransform
	}
	return toTransform
}

// DescribeApproximatePaintClip is nothing: the child is not clipped.
//
// Flutter: rendering/object.dart RenderObject.describeApproximatePaintClip
func (b *RenderObjectBase) DescribeApproximatePaintClip(RenderObject) foundation.Opt[ui.Rect] {
	return foundation.None[ui.Rect]()
}

// HandleEvent does nothing; a render object that reacts to pointer events
// overrides it.
//
// Flutter: rendering/object.dart RenderObject.handleEvent
func (b *RenderObjectBase) HandleEvent(gestures.PointerEvent, gestures.HitTestEntry) {}

// ToStringShort describes the object and its state: "NEEDS-LAYOUT",
// "NEEDS-PAINT", its relayout boundary distance, and so on.
//
// Flutter: rendering/object.dart RenderObject.toStringShort
func (b *RenderObjectBase) ToStringShort() string {
	header := foundation.DescribeIdentity(b.self)
	if !foundation.DebugMode {
		return header
	}
	if b.debugDisposed {
		return header + " DISPOSED"
	}
	count := 0
	node := b.self
	for node != nil && !node.AsRenderObject().isRelayoutBoundary.OrElse(false) {
		if node.AsRenderObject().isRelayoutBoundary.IsNone() {
			count = -1
			break
		}
		count++
		node = node.AsRenderObject().parent
	}
	if count > 0 {
		header += " relayoutBoundary=up" + strconv.Itoa(count)
	}
	if b.needsLayout {
		header += " NEEDS-LAYOUT"
	}
	if b.needsPaint {
		header += " NEEDS-PAINT"
	}
	if b.needsCompositingBitsUpdate {
		header += " NEEDS-COMPOSITING-BITS-UPDATE"
	}
	if !b.Attached() {
		header += " DETACHED"
	}
	return header
}

// String is ToStringShort.
//
// Flutter: rendering/object.dart RenderObject.toString
func (b *RenderObjectBase) String() string { return b.self.(foundation.ShortDescriber).ToStringShort() }

// DebugFillProperties adds the compositing flag, the creator, the parent data,
// the constraints and the layer.
//
// Flutter: rendering/object.dart RenderObject.debugFillProperties
func (b *RenderObjectBase) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewFlagProperty("needsCompositing", foundation.Some(b.needsCompositing),
		foundation.FlagPropertyOpts{IfTrue: "needs compositing"}))
	p.Add(foundation.NewDiagnosticsProperty("creator", b.DebugCreator,
		foundation.DiagnosticsPropertyOpts{DefaultValue: foundation.DefaultNull, Level: foundation.DiagnosticLevelDebug}))
	tooltip := ""
	if b.debugCanParentUseSize.OrElse(false) {
		tooltip = "can use size"
	}
	p.Add(foundation.NewDiagnosticsProperty("parentData", b.parentData,
		foundation.DiagnosticsPropertyOpts{Tooltip: tooltip, MissingIfNull: true}))
	p.Add(foundation.NewDiagnosticsProperty("constraints", b.constraints, foundation.DiagnosticsPropertyOpts{MissingIfNull: true}))
	p.Add(foundation.NewDiagnosticsProperty("layer", b.layerHandle.Layer(),
		foundation.DiagnosticsPropertyOpts{DefaultValue: foundation.DefaultNull}))
	if node := b.DebugSemantics(); node != nil {
		p.Add(foundation.NewDiagnosticsProperty("semantics node", node))
	}
	eff := b.semantics().configProv.effective()
	p.Add(foundation.NewFlagProperty("isBlockingSemanticsOfPreviouslyPaintedNodes", foundation.Some(eff.IsBlockingSemanticsOfPreviouslyPaintedNodes),
		foundation.FlagPropertyOpts{IfTrue: "blocks semantics of earlier render objects below the common boundary"}))
	p.Add(foundation.NewFlagProperty("isSemanticBoundary", foundation.Some(eff.IsSemanticBoundary()),
		foundation.FlagPropertyOpts{IfTrue: "semantic boundary"}))
}

// DebugDescribeChildren describes the children through the child model.
//
// Flutter: rendering/object.dart RenderObject.debugDescribeChildren
func (b *RenderObjectBase) DebugDescribeChildren() []foundation.DiagnosticsNode {
	if b.model != nil {
		return b.model.describeChildren()
	}
	return nil
}

// DescribeForError describes the object as a node, for an error's metadata.
//
// Flutter: rendering/object.dart RenderObject.describeForError
func (b *RenderObjectBase) DescribeForError(name string, style foundation.DiagnosticsTreeStyle) foundation.DiagnosticsNode {
	return foundation.ToDiagnosticsNode(b.self, foundation.DiagnosticsNodeOpts{Name: foundation.Some(name), Style: style})
}

// ToStringDeep dumps the object and its descendants, with the debug-only layout
// state cleared, as Flutter does.
//
// Flutter: rendering/object.dart RenderObject.toStringDeep
func (b *RenderObjectBase) ToStringDeep(opts ...foundation.ToStringDeepOpts) string {
	prev := b.dbg().activeLayout
	b.dbg().activeLayout = nil
	defer func() { b.dbg().activeLayout = prev }()
	return foundation.ToStringDeep(b.self, opts...)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagGetTransformToDetached           = "GetTransformToDetached"
	tagInvokeLayoutCallbackInvalid      = "InvokeLayoutCallbackInvalid"
	tagLayoutReentrant                  = "LayoutReentrant"
	tagMarkNeedsPaintDuringPaint        = "MarkNeedsPaintDuringPaint"
	tagPaintBeforeCompositingBits       = "PaintBeforeCompositingBits"
	tagPaintMarkedDirty                 = "PaintMarkedDirty"
	tagPaintReentrant                   = "PaintReentrant"
	tagRenderObjectAbstractMethod       = "RenderObjectAbstractMethod"
	tagRenderObjectAttached             = "RenderObjectAttached"
	tagRenderObjectBoundaryLayer        = "RenderObjectBoundaryLayer"
	tagRenderObjectCompositingBitsDirty = "RenderObjectCompositingBitsDirty"
	tagRenderObjectCycle                = "RenderObjectCycle"
	tagRenderObjectDisposedMutated      = "RenderObjectDisposedMutated"
	tagRenderObjectDisposedTwice        = "RenderObjectDisposedTwice"
	tagRenderObjectHasParent            = "RenderObjectHasParent"
	tagRenderObjectMutatedDuringLayout  = "RenderObjectMutatedDuringLayout"
	tagRenderObjectNoConstraints        = "RenderObjectNoConstraints"
	tagRenderObjectNoParent             = "RenderObjectNoParent"
	tagRenderObjectNotAttached          = "RenderObjectNotAttached"
	tagRenderObjectOwnerMismatch        = "RenderObjectOwnerMismatch"
	tagRenderObjectsNotInSameTree       = "RenderObjectsNotInSameTree"
	tagReplaceRootLayerInvalid          = "ReplaceRootLayerInvalid"
	tagScheduleInitialLayoutInvalid     = "ScheduleInitialLayoutInvalid"
	tagScheduleInitialPaintInvalid      = "ScheduleInitialPaintInvalid"
	tagUpdateCompositedLayerNotBoundary = "UpdateCompositedLayerNotBoundary"
)
