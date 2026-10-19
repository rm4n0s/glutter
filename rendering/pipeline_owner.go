package rendering

import (
	"sort"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/semantics"
)

// PipelineManifold is the connection between a tree of pipeline owners and the
// rest of the framework: it asks for frames and says whether semantics are
// wanted. It is a Listenable that notifies when SemanticsEnabled changes.
//
// Flutter: rendering/object.dart PipelineManifold
type PipelineManifold interface {
	foundation.Listenable
	// SemanticsEnabled reports whether semantics are needed.
	//
	// Flutter: rendering/object.dart PipelineManifold.semanticsEnabled
	SemanticsEnabled() bool
	// RequestVisualUpdate asks for a frame.
	//
	// Flutter: rendering/object.dart PipelineManifold.requestVisualUpdate
	RequestVisualUpdate()
}

// PipelineOwnerOpts are the optional parameters of NewPipelineOwner.
//
// Flutter: rendering/object.dart PipelineOwner.new
type PipelineOwnerOpts struct {
	// OnNeedVisualUpdate is called when a render object asks for a frame; the
	// default asks the manifold.
	OnNeedVisualUpdate func()
	// DisallowRootNode makes SetRootNode with a render object panic with the
	// tag RootNodeNotAllowed: the root pipeline owner of the binding manages
	// child owners, not a root of its own.
	DisallowRootNode bool
	// Debug are the debug flags. The default is a fresh set; a child owner
	// uses its parent's.
	Debug *DebugOptions

	// OnSemanticsOwnerCreated is called when the owner creates its semantics
	// owner, because the manifold wants semantics.
	OnSemanticsOwnerCreated func()
	// OnSemanticsUpdate receives the semantics updates. A tree whose manifold
	// enables semantics needs it (the tag SemanticsUpdateCallbackMissing).
	OnSemanticsUpdate semantics.SemanticsUpdateCallback
	// OnSemanticsOwnerDisposed is called when the owner disposes its semantics
	// owner.
	OnSemanticsOwnerDisposed func()
}

// renderDebugState is the debug state that Flutter keeps in static variables of
// RenderObject and RenderBox: it is shared by all the owners of a tree.
type renderDebugState struct {
	activeLayout              RenderObject
	activePaint               RenderObject
	checkingIntrinsics        bool
	doingBaseline             bool
	dryLayoutCalculationValid bool
}

// PipelineOwner is the interface between the render tree and the pipeline that
// runs layout, compositing bits, paint and semantics. A tree of owners has one
// owner per root render object (such as a RenderView).
//
// Flutter: rendering/object.dart PipelineOwner
type PipelineOwner struct {
	onNeedVisualUpdate func()
	debug              *DebugOptions
	debugState         *renderDebugState

	rootNode                RenderObject
	shouldMergeDirtyNodes   bool
	nodesNeedingLayout      []RenderObject
	debugDoingLayout        bool
	debugDoingChildLayout   bool
	debugAllowMutations     bool
	nodesNeedingCompositing []RenderObject
	nodesNeedingPaint       []RenderObject
	debugDoingPaint         bool
	children                []*PipelineOwner
	manifold                PipelineManifold
	debugParent             *PipelineOwner
	disallowRootNode        bool
	semanticsListener       foundation.ListenerHandle

	onSemanticsOwnerCreated  func()
	onSemanticsUpdate        semantics.SemanticsUpdateCallback
	onSemanticsOwnerDisposed func()
	semanticsOwner           *semantics.SemanticsOwner
	debugDoingSemantics      bool

	nodesNeedingSemanticsUpdate         orderedSet[RenderObject]
	nodesNeedingSemanticsGeometryUpdate orderedSet[RenderObject]
}

// NewPipelineOwner creates a pipeline owner.
//
// Flutter: rendering/object.dart PipelineOwner.new
func NewPipelineOwner(opts ...PipelineOwnerOpts) *PipelineOwner {
	o := firstOpts(opts)
	if o.Debug == nil {
		o.Debug = NewDebugOptions()
	}
	p := &PipelineOwner{onNeedVisualUpdate: o.OnNeedVisualUpdate, debug: o.Debug, debugState: &renderDebugState{}, disallowRootNode: o.DisallowRootNode,
		onSemanticsOwnerCreated: o.OnSemanticsOwnerCreated, onSemanticsUpdate: o.OnSemanticsUpdate, onSemanticsOwnerDisposed: o.OnSemanticsOwnerDisposed}
	return p
}

// DebugOptions are the debug flags that the owner and its render objects read.
//
// Flutter: rendering/debug.dart
func (p *PipelineOwner) DebugOptions() *DebugOptions { return p.debug }

// RequestVisualUpdate asks for a frame: it calls OnNeedVisualUpdate, or asks the
// manifold.
//
// Flutter: rendering/object.dart PipelineOwner.requestVisualUpdate
func (p *PipelineOwner) RequestVisualUpdate() {
	if p.onNeedVisualUpdate != nil {
		p.onNeedVisualUpdate()
	} else if p.manifold != nil {
		p.manifold.RequestVisualUpdate()
	}
}

// RootNode is the render object at the root of the tree this owner manages.
//
// Flutter: rendering/object.dart PipelineOwner.rootNode
func (p *PipelineOwner) RootNode() RenderObject { return p.rootNode }

// SetRootNode replaces the root: the old root is detached and the new one
// attached. Pass nil to remove the root.
//
// Flutter: rendering/object.dart PipelineOwner.rootNode
func (p *PipelineOwner) SetRootNode(value RenderObject) {
	if p.disallowRootNode && value != nil {
		panic(errors.New("RootNodeNotAllowed", "cannot set a rootNode on the default root pipeline owner: "+
			"it manages child pipeline owners, which have the root nodes; override the creation of the root pipeline owner to change this"))
	}
	if p.rootNode == value {
		return
	}
	if p.rootNode != nil {
		p.rootNode.Detach()
	}
	p.rootNode = value
	if p.rootNode != nil {
		p.rootNode.Attach(p)
	}
}

// NodesNeedingLayout are the render objects that need layout.
//
// Flutter: rendering/object.dart PipelineOwner.nodesNeedingLayout
func (p *PipelineOwner) NodesNeedingLayout() []RenderObject { return p.nodesNeedingLayout }

// DebugDoingLayout reports whether FlushLayout is running.
//
// Flutter: rendering/object.dart PipelineOwner.debugDoingLayout
func (p *PipelineOwner) DebugDoingLayout() bool { return p.debugDoingLayout }

func byDepthAscending(nodes []RenderObject) {
	sort.SliceStable(nodes, func(i, j int) bool { return nodes[i].AsRenderObject().depth < nodes[j].AsRenderObject().depth })
}

// FlushLayout lays out the render objects that need it, shallowest first, and
// then those of the child owners.
//
// Flutter: rendering/object.dart PipelineOwner.flushLayout
func (p *PipelineOwner) FlushLayout() {
	if foundation.DebugMode {
		p.debugDoingLayout = true
	}
	defer func() {
		p.shouldMergeDirtyNodes = false
		if foundation.DebugMode {
			p.debugDoingLayout = false
			p.debugDoingChildLayout = false
		}
	}()
	for len(p.nodesNeedingLayout) > 0 {
		dirty := p.nodesNeedingLayout
		p.nodesNeedingLayout = nil
		byDepthAscending(dirty)
		for i := 0; i < len(dirty); i++ {
			if p.shouldMergeDirtyNodes {
				p.shouldMergeDirtyNodes = false
				if len(p.nodesNeedingLayout) > 0 {
					p.nodesNeedingLayout = append(p.nodesNeedingLayout, dirty[i:]...)
					break
				}
			}
			node := dirty[i]
			nb := node.AsRenderObject()
			if nb.needsLayout && nb.owner == p {
				nb.layoutWithoutResize()
			}
		}
		p.shouldMergeDirtyNodes = false
	}
	if foundation.DebugMode {
		p.debugDoingChildLayout = true
	}
	for _, child := range append([]*PipelineOwner(nil), p.children...) {
		child.FlushLayout()
	}
}

func (p *PipelineOwner) enableMutationsToDirtySubtrees(callback func()) {
	old := p.debugAllowMutations
	if foundation.DebugMode {
		p.debugAllowMutations = true
	}
	defer func() {
		p.shouldMergeDirtyNodes = true
		if foundation.DebugMode {
			p.debugAllowMutations = old
		}
	}()
	callback()
}

// FlushCompositingBits updates the compositing bits of the render objects that
// need it, shallowest first, and then those of the child owners.
//
// Flutter: rendering/object.dart PipelineOwner.flushCompositingBits
func (p *PipelineOwner) FlushCompositingBits() {
	byDepthAscending(p.nodesNeedingCompositing)
	for _, node := range p.nodesNeedingCompositing {
		nb := node.AsRenderObject()
		if nb.needsCompositingBitsUpdate && nb.owner == p {
			nb.updateCompositingBits()
		}
	}
	p.nodesNeedingCompositing = nil
	for _, child := range p.children {
		child.FlushCompositingBits()
	}
}

// NodesNeedingPaint are the repaint boundaries that need to paint.
//
// Flutter: rendering/object.dart PipelineOwner.nodesNeedingPaint
func (p *PipelineOwner) NodesNeedingPaint() []RenderObject { return p.nodesNeedingPaint }

// DebugDoingPaint reports whether FlushPaint is running.
//
// Flutter: rendering/object.dart PipelineOwner.debugDoingPaint
func (p *PipelineOwner) DebugDoingPaint() bool { return p.debugDoingPaint }

// FlushPaint repaints the repaint boundaries that need it, deepest first, and
// then those of the child owners.
//
// Flutter: rendering/object.dart PipelineOwner.flushPaint
func (p *PipelineOwner) FlushPaint() {
	if foundation.DebugMode {
		p.debugDoingPaint = true
	}
	defer func() {
		if foundation.DebugMode {
			p.debugDoingPaint = false
		}
	}()
	dirty := p.nodesNeedingPaint
	p.nodesNeedingPaint = nil
	sort.SliceStable(dirty, func(i, j int) bool { return dirty[i].AsRenderObject().depth > dirty[j].AsRenderObject().depth })
	for _, node := range dirty {
		nb := node.AsRenderObject()
		if (nb.needsPaint || nb.needsCompositedLayerUpdate) && nb.owner == p {
			if nb.layerHandle.Layer().AsLayer().Attached() {
				if nb.needsPaint {
					RepaintCompositedChild(node, false)
				} else {
					UpdateLayerProperties(node)
				}
			} else {
				nb.skippedPaintingOnLayer()
			}
		}
	}
	for _, child := range p.children {
		child.FlushPaint()
	}
}

// SemanticsOwner is the owner of the semantics tree, or nil while the manifold
// does not want semantics. A nil owner makes the pipeline skip all the
// semantics steps.
//
// Flutter: rendering/object.dart PipelineOwner.semanticsOwner
func (p *PipelineOwner) SemanticsOwner() *semantics.SemanticsOwner { return p.semanticsOwner }

// DebugDoingSemantics reports whether FlushSemantics is running.
//
// Flutter: rendering/object.dart PipelineOwner._debugDoingSemantics
func (p *PipelineOwner) DebugDoingSemantics() bool { return p.debugDoingSemantics }

func (p *PipelineOwner) updateSemanticsOwner() {
	if p.manifold != nil && p.manifold.SemanticsEnabled() {
		if p.semanticsOwner == nil {
			if p.onSemanticsUpdate == nil {
				panic(errors.New("SemanticsUpdateCallbackMissing", "attempted to enable semantics without configuring an OnSemanticsUpdate callback"))
			}
			p.semanticsOwner = semantics.NewSemanticsOwner(p.onSemanticsUpdate)
			if p.onSemanticsOwnerCreated != nil {
				p.onSemanticsOwnerCreated()
			}
		}
	} else if p.semanticsOwner != nil {
		p.semanticsOwner.Dispose()
		p.semanticsOwner = nil
		if p.onSemanticsOwnerDisposed != nil {
			p.onSemanticsOwnerDisposed()
		}
	}
}

// FlushSemantics updates the semantics tree, if there is one, and then those of
// the child owners. Initially only the root, as scheduled by
// ScheduleInitialSemantics, needs an update.
//
// Flutter: rendering/object.dart PipelineOwner.flushSemantics
func (p *PipelineOwner) FlushSemantics() {
	if p.semanticsOwner == nil {
		return
	}
	if foundation.DebugMode {
		p.debugDoingSemantics = true
	}
	defer func() {
		if foundation.DebugMode {
			p.debugDoingSemantics = false
		}
	}()
	// Top-down order: a change of a parent can invalidate the whole subtree.
	var nodesToProcess []RenderObject
	for _, o := range p.nodesNeedingSemanticsUpdate.take() {
		if ob := o.AsRenderObject(); !ob.needsLayout && ob.owner == p {
			nodesToProcess = append(nodesToProcess, o)
		}
	}
	sort.SliceStable(nodesToProcess, func(i, j int) bool { return nodesToProcess[i].Depth() < nodesToProcess[j].Depth() })
	for _, node := range nodesToProcess {
		sem := node.AsRenderObject().semantics()
		if sem.parentDataDirty() {
			// The node is blocked by a sibling or hidden by its parent; the
			// parent data must not be updated now, or the gap of render
			// objects with dirty parent data would remain when the branch
			// rejoins the tree.
			continue
		}
		sem.updateChildren()
	}

	// updateChildren can add nodes needing geometry updates.
	var nodesToProcessGeometry []RenderObject
	for _, o := range p.nodesNeedingSemanticsGeometryUpdate.take() {
		if ob := o.AsRenderObject(); !ob.needsLayout && ob.owner == p && !ob.semantics().parentDataDirty() {
			nodesToProcessGeometry = append(nodesToProcessGeometry, o)
		}
	}
	for _, node := range nodesToProcessGeometry {
		sem := node.AsRenderObject().semantics()
		if sem.shouldFormSemanticsNode() && sem.geometryDirty() {
			continue
		}
		if sem.shouldFormSemanticsNode() && node.AsRenderObject().isRelayoutBoundary.OrElse(false) {
			sem.geometry = nil
			continue
		}
		if !sem.contributesToSemanticsTree() {
			for _, f := range sem.mergeUp {
				child, ok := f.(*renderObjectSemantics)
				if !ok {
					continue
				}
				if child.shouldFormSemanticsNode() {
					child.geometry = nil
				} else {
					for _, inSubtree := range child.children {
						inSubtree.geometry = nil
					}
				}
			}
			continue
		}
		for _, child := range sem.children {
			child.geometry = nil
		}
	}

	// The token invalidates the cache of the closest ancestor with clean geometry.
	treeShapeToken := new(struct{})
	var ensure orderedSet[*renderObjectSemantics]
	for _, node := range nodesToProcessGeometry {
		sem := node.AsRenderObject().semantics()
		sem.computeAncestorInfo(treeShapeToken)
		if sem.firstAncestorNodeWithCleanGeometry != nil {
			ensure.add(sem.firstAncestorNodeWithCleanGeometry)
		}
	}
	toEnsure := ensure.take()
	sort.SliceStable(toEnsure, func(i, j int) bool { return toEnsure[i].renderObject.Depth() < toEnsure[j].renderObject.Depth() })
	for _, node := range toEnsure {
		node.ensureGeometry()
	}

	for i := len(nodesToProcess) - 1; i >= 0; i-- {
		sem := nodesToProcess[i].AsRenderObject().semantics()
		sem.computeAncestorInfo(treeShapeToken)
		var targets []*renderObjectSemantics
		if sem.geometryDirty() {
			if sem.firstAncestorNodeWithCleanGeometry != nil {
				targets = append(targets, sem.firstAncestorNodeWithCleanGeometry)
			}
		} else {
			// A node that is a semantics boundary and a layout boundary can
			// become invisible after ensureGeometry: its parent must then
			// update its subtree to drop the node.
			if !sem.geometry.isVisible() && !sem.isRoot() {
				if parentInTree := sem.parentInSemanticsTree; parentInTree != nil {
					if !parentInTree.geometryDirty() {
						targets = append(targets, parentInTree)
					} else if first := parentInTree.firstAncestorNodeWithCleanGeometry; first != nil {
						// Nil if this is a blocked branch.
						targets = append(targets, first)
					}
				}
			}
			targets = append(targets, sem)
		}
		for _, target := range targets {
			if target.parentDataDirty() {
				// The node is not in the tree.
				continue
			}
			target.ensureSemanticsNode()
		}
	}

	p.semanticsOwner.SendSemanticsUpdate()
	for _, child := range p.children {
		child.FlushSemantics()
	}
}

// DebugDescribeChildren is the child owners.
//
// Flutter: rendering/object.dart PipelineOwner.debugDescribeChildren
func (p *PipelineOwner) DebugDescribeChildren() []foundation.DiagnosticsNode {
	out := make([]foundation.DiagnosticsNode, 0, len(p.children))
	for _, c := range p.children {
		out = append(out, foundation.ToDiagnosticsNode(c))
	}
	return out
}

// DebugFillProperties adds the root node.
//
// Flutter: rendering/object.dart PipelineOwner.debugFillProperties
func (p *PipelineOwner) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	b.Add(foundation.NewDiagnosticsProperty("rootNode", p.rootNode, foundation.DiagnosticsPropertyOpts{DefaultValue: foundation.DefaultNull}))
}

// String describes the owner.
func (p *PipelineOwner) String() string { return foundation.DiagnosticableString(p) }

// Attach connects the owner and its children to the manifold.
//
// Flutter: rendering/object.dart PipelineOwner.attach
func (p *PipelineOwner) Attach(manifold PipelineManifold) {
	if foundation.DebugMode && p.manifold != nil {
		panic(errors.New("PipelineOwnerAttached", "the pipeline owner is attached to a manifold already"))
	}
	p.manifold = manifold
	p.semanticsListener = manifold.AddListener(p.updateSemanticsOwner)
	p.updateSemanticsOwner()
	for _, child := range p.children {
		child.Attach(manifold)
	}
}

// Detach disconnects the owner and its children from the manifold.
//
// Flutter: rendering/object.dart PipelineOwner.detach
func (p *PipelineOwner) Detach() {
	if foundation.DebugMode && p.manifold == nil {
		panic(errors.New("PipelineOwnerNotAttached", "the pipeline owner is not attached to a manifold"))
	}
	p.manifold.RemoveListener(p.semanticsListener)
	p.manifold = nil
	// The semantics owner stays, so as not to disrupt its clients in case of a
	// re-attach; Attach updates it, and Dispose disposes it.
	for _, child := range p.children {
		child.Detach()
	}
}

// AdoptChild makes the owner a child of this one: it is flushed after this
// owner, and attached to the same manifold. A debug build panics with
// PipelineOwnerChildListLocked if called during layout, paint or semantics.
//
// Flutter: rendering/object.dart PipelineOwner.adoptChild
func (p *PipelineOwner) AdoptChild(child *PipelineOwner) {
	if foundation.DebugMode {
		if child.debugParent != nil {
			panic(errors.New("PipelineOwnerHasParent", "the pipeline owner has a parent already"))
		}
		for _, c := range p.children {
			if c == child {
				panic(errors.New("PipelineOwnerHasParent", "the pipeline owner is a child already"))
			}
		}
		if p.debugDoingChildLayout || p.debugDoingPaint {
			panic(errors.New("PipelineOwnerChildListLocked", "cannot modify the child list after layout"))
		}
	}
	p.children = append(p.children, child)
	if foundation.DebugMode {
		child.debugParent = p
	}
	child.shareDebugWith(p)
	if p.manifold != nil {
		child.Attach(p.manifold)
	}
}

func (p *PipelineOwner) shareDebugWith(parent *PipelineOwner) {
	p.debug = parent.debug
	p.debugState = parent.debugState
	for _, c := range p.children {
		c.shareDebugWith(p)
	}
}

// DropChild removes a child owner.
//
// Flutter: rendering/object.dart PipelineOwner.dropChild
func (p *PipelineOwner) DropChild(child *PipelineOwner) {
	if foundation.DebugMode {
		if child.debugParent != p {
			panic(errors.New("PipelineOwnerNotChild", "the pipeline owner is not a child of this one"))
		}
		if p.debugDoingChildLayout || p.debugDoingPaint {
			panic(errors.New("PipelineOwnerChildListLocked", "cannot modify the child list after layout"))
		}
	}
	for i, c := range p.children {
		if c == child {
			p.children = append(p.children[:i], p.children[i+1:]...)
			break
		}
	}
	if foundation.DebugMode {
		child.debugParent = nil
	}
	if p.manifold != nil {
		child.Detach()
	}
}

// VisitChildren calls the visitor with every child owner.
//
// Flutter: rendering/object.dart PipelineOwner.visitChildren
func (p *PipelineOwner) VisitChildren(visitor func(child *PipelineOwner)) {
	for _, c := range append([]*PipelineOwner(nil), p.children...) {
		visitor(c)
	}
}

// Dispose releases the owner. It must have no children, no root and no manifold.
//
// Flutter: rendering/object.dart PipelineOwner.dispose
func (p *PipelineOwner) Dispose() {
	if foundation.DebugMode && (len(p.children) > 0 || p.rootNode != nil || p.manifold != nil || p.debugParent != nil) {
		panic(errors.New("PipelineOwnerDisposeBusy", "a pipeline owner can only be disposed when it has no children, root, manifold or parent"))
	}
	if p.semanticsOwner != nil {
		p.semanticsOwner.Dispose()
		p.semanticsOwner = nil
	}
	p.nodesNeedingLayout = nil
	p.nodesNeedingCompositing = nil
	p.nodesNeedingPaint = nil
	p.nodesNeedingSemanticsUpdate.take()
}
