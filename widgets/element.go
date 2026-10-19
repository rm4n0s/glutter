package widgets

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
)

// elementLifecycle is where an element is in its life.
//
// Flutter: widgets/framework.dart _ElementLifecycle
type elementLifecycle int

const (
	lifecycleInitial elementLifecycle = iota
	lifecycleActive
	lifecycleInactive
	lifecycleFailed
	lifecycleDefunct
)

var lifecycleNames = [...]string{"initial", "active", "inactive", "failed", "defunct"}

func (l elementLifecycle) String() string { return lifecycleNames[l] }

// IndexedSlot is the slot of an element whose parent keeps its children in a
// list: the index of the element and the element before it. It is a value, so
// two slots are equal when both parts are.
//
// Flutter: widgets/framework.dart IndexedSlot
type IndexedSlot struct {
	// Index is the position of the element in the list.
	//
	// Flutter: widgets/framework.dart IndexedSlot.index
	Index int
	// Value is the previous sibling, or nil for the first.
	//
	// Flutter: widgets/framework.dart IndexedSlot.value
	Value Element
}

// Element is an instantiation of a Widget at a particular location in the
// tree. It is also the BuildContext of its widget.
//
// Element types embed ElementBase and call InitElement in their constructor, so
// that the base can make the virtual calls Flutter makes (SPEC §12.5). A
// method that overrides one of the base methods calls the base method on the
// embedded ElementBase where Flutter calls super.
//
// Flutter: widgets/framework.dart Element
type Element interface {
	BuildContext
	foundation.DiagnosticableTree
	fmt.Stringer
	foundation.ShortDescriber
	// AsElement is the embedded base.
	AsElement() *ElementBase

	// Slot is the location of the element in its parent's child model.
	//
	// Flutter: widgets/framework.dart Element.slot
	Slot() any
	// Depth is the distance from the root; it panics with the tag
	// ElementDepthBeforeMount in debug builds until the element is mounted.
	//
	// Flutter: widgets/framework.dart Element.depth
	Depth() int
	// Dirty is whether the element needs to be rebuilt.
	//
	// Flutter: widgets/framework.dart Element.dirty
	Dirty() bool
	// BuildScope is the build scope that builds this element; it is the one of
	// the parent unless an element overrides it.
	//
	// Flutter: widgets/framework.dart Element.buildScope
	BuildScope() *BuildScope
	// VisitChildren calls the visitor for each child. The default has none.
	//
	// Flutter: widgets/framework.dart Element.visitChildren
	VisitChildren(visitor ElementVisitor)
	// DebugVisitOnstageChildren calls the visitor for each child that is
	// visible; it is VisitChildren by default.
	//
	// Flutter: widgets/framework.dart Element.debugVisitOnstageChildren
	DebugVisitOnstageChildren(visitor ElementVisitor)
	// RenderObjectAttachingChild is the one child that holds the render
	// objects of this element's subtree, or nil.
	//
	// Flutter: widgets/framework.dart Element.renderObjectAttachingChild
	RenderObjectAttachingChild() Element
	// Mount adds the element to the tree under the parent, in the slot.
	//
	// Flutter: widgets/framework.dart Element.mount
	Mount(parent Element, newSlot any)
	// Update changes the widget of the element to a new one that CanUpdate.
	//
	// Flutter: widgets/framework.dart Element.update
	Update(newWidget Widget)
	// UpdateSlot changes the slot of the element.
	//
	// Flutter: widgets/framework.dart Element.updateSlot
	UpdateSlot(newSlot any)
	// DetachRenderObject removes the render objects of the subtree from the
	// render tree.
	//
	// Flutter: widgets/framework.dart Element.detachRenderObject
	DetachRenderObject()
	// AttachRenderObject puts the render objects of the subtree into the render
	// tree, in the slot.
	//
	// Flutter: widgets/framework.dart Element.attachRenderObject
	AttachRenderObject(newSlot any)
	// ForgetChild is told that the child is leaving for another parent.
	//
	// Flutter: widgets/framework.dart Element.forgetChild
	ForgetChild(child Element)
	// Activate moves the element back into the tree from the inactive list.
	//
	// Flutter: widgets/framework.dart Element.activate
	Activate()
	// Deactivate moves the element out of the tree to the inactive list.
	//
	// Flutter: widgets/framework.dart Element.deactivate
	Deactivate()
	// DebugDeactivated is called after the whole subtree is deactivated.
	//
	// Flutter: widgets/framework.dart Element.debugDeactivated
	DebugDeactivated()
	// Unmount removes the element for good.
	//
	// Flutter: widgets/framework.dart Element.unmount
	Unmount()
	// DidChangeDependencies is called when a dependency of the element
	// changes.
	//
	// Flutter: widgets/framework.dart Element.didChangeDependencies
	DidChangeDependencies()
	// PerformRebuild does the work of a rebuild.
	//
	// Flutter: widgets/framework.dart Element.performRebuild
	PerformRebuild()
	// Reassemble marks the subtree dirty, after a hot reload.
	//
	// Flutter: widgets/framework.dart Element.reassemble
	Reassemble()
	// DebugExpectsRenderObjectForSlot is whether the slot holds a render
	// object.
	//
	// Flutter: widgets/framework.dart Element.debugExpectsRenderObjectForSlot
	DebugExpectsRenderObjectForSlot(slot any) bool
	// MarkNeedsBuild marks the element dirty and schedules it to be built.
	//
	// Flutter: widgets/framework.dart Element.markNeedsBuild
	MarkNeedsBuild()
	// Rebuild builds the element if it is dirty.
	//
	// Flutter: widgets/framework.dart Element.rebuild
	Rebuild(opts ...RebuildOpts)
	// UpdateChild updates a child to a new widget; see ElementBase.UpdateChild.
	//
	// Flutter: widgets/framework.dart Element.updateChild
	UpdateChild(child Element, newWidget Widget, newSlot any) Element
	// InflateWidget makes a new child element; see ElementBase.InflateWidget.
	//
	// Flutter: widgets/framework.dart Element.inflateWidget
	InflateWidget(newWidget Widget, newSlot any) Element
	// DeactivateChild moves a child to the inactive list.
	//
	// Flutter: widgets/framework.dart Element.deactivateChild
	DeactivateChild(child Element)
	// RenderObject is the render object of this element, or of its first
	// descendant that has one; nil when there is none.
	//
	// Flutter: widgets/framework.dart Element.renderObject
	RenderObject() rendering.RenderObject
	// ownRenderObject is the render object this element creates itself, if it
	// is a render object element.
	ownRenderObject() (rendering.RenderObject, bool)
	// updateInheritance sets the inherited elements this element sees: those
	// of its parent, plus itself for an inherited element.
	updateInheritance()
}

// RebuildOpts are the optional parameters of Element.Rebuild.
//
// Flutter: widgets/framework.dart Element.rebuild
type RebuildOpts struct {
	// Force rebuilds the element even when it is not dirty.
	Force bool
}

// ElementBase is the part of every element that the framework ports from
// Flutter's Element class. Embed it and call InitElement.
//
// Flutter: widgets/framework.dart Element
type ElementBase struct {
	self        Element
	parent      Element
	slot        any
	depth       int
	widget      Widget
	owner       *BuildOwner
	parentScope *BuildScope
	lifecycle   elementLifecycle
	dirty       bool
	inDirtyList bool

	// The inherited widgets above this element, by type, and the ones this
	// element depends on.
	inheritedElements          map[reflect.Type]Element
	dependencies               map[Element]struct{}
	hadUnsatisfiedDependencies bool
	// forgottenWithGlobalKey are the children with a global key that moved to
	// another parent; their reservations are released when this element updates.
	forgottenWithGlobalKey orderedSet[Element]
}

// InitElement sets up the base of an element: self is the outer element and
// widget is its first configuration.
//
// Flutter: widgets/framework.dart Element.new
func InitElement(b *ElementBase, self Element, widget Widget) {
	b.self = self
	b.widget = widget
	b.dirty = true
}

// AsElement returns the base.
func (b *ElementBase) AsElement() *ElementBase { return b }

// Slot is the location of the element in its parent's child model.
//
// Flutter: widgets/framework.dart Element.slot
func (b *ElementBase) Slot() any { return b.slot }

// Depth is the distance from the root.
//
// Flutter: widgets/framework.dart Element.depth
func (b *ElementBase) Depth() int {
	if foundation.DebugMode && b.lifecycle == lifecycleInitial {
		panic(errors.New("ElementDepthBeforeMount", "depth is only available when the element has been mounted"))
	}
	return b.depth
}

// Dirty is whether the element needs to be rebuilt.
//
// Flutter: widgets/framework.dart Element.dirty
func (b *ElementBase) Dirty() bool { return b.dirty }

// Widget is the current configuration of the element. It panics with the tag
// ElementWidgetUnmounted when the element has been unmounted.
//
// Flutter: widgets/framework.dart Element.widget
func (b *ElementBase) Widget() Widget {
	if b.widget == nil {
		panic(errors.New("ElementWidgetUnmounted", "the widget of an element is gone once the element is unmounted",
			"element", b.ToStringShort()))
	}
	return b.widget
}

// Mounted is whether the element is in the tree.
//
// Flutter: widgets/framework.dart Element.mounted
func (b *ElementBase) Mounted() bool { return b.widget != nil }

// DebugIsDefunct is whether the element has been unmounted. It is false in
// release builds.
//
// Flutter: widgets/framework.dart Element.debugIsDefunct
func (b *ElementBase) DebugIsDefunct() bool {
	return foundation.DebugMode && b.lifecycle == lifecycleDefunct
}

// DebugIsActive is whether the element is in the tree. It is false in release
// builds.
//
// Flutter: widgets/framework.dart Element.debugIsActive
func (b *ElementBase) DebugIsActive() bool {
	return foundation.DebugMode && b.lifecycle == lifecycleActive
}

// Owner is the build owner of the element; nil until it is mounted.
//
// Flutter: widgets/framework.dart Element.owner
func (b *ElementBase) Owner() *BuildOwner { return b.owner }

// BuildScope is the build scope of the parent, cached when the element is
// mounted or reparented.
//
// Flutter: widgets/framework.dart Element.buildScope
func (b *ElementBase) BuildScope() *BuildScope { return b.parentScope }

// AssignOwner gives a root element its owner and a build scope of its own. Call
// it on the root element before mounting it.
//
// Deviation: it is a method of every element, where Flutter has it in
// RootElementMixin, because Go cannot mix two bases into one element (SPEC
// §12.10).
//
// Flutter: widgets/framework.dart RootElementMixin.assignOwner
func (b *ElementBase) AssignOwner(owner *BuildOwner) {
	b.owner = owner
	b.parentScope = NewBuildScope()
}

// Parent is the parent element, or nil for the root and for an element that is
// not in the tree.
//
// Flutter: widgets/framework.dart Element._parent
func (b *ElementBase) Parent() Element { return b.parent }

// Reassemble marks the element and its descendants dirty.
//
// Flutter: widgets/framework.dart Element.reassemble
func (b *ElementBase) Reassemble() {
	b.self.MarkNeedsBuild()
	b.self.VisitChildren(func(child Element) { child.Reassemble() })
}

func (b *ElementBase) isDescendantOf(target Element) bool {
	e := b.self
	for e != nil && e.AsElement().depth > target.AsElement().depth {
		e = e.AsElement().parent
	}
	return e == target
}

// ownRenderObject is the render object created by this element; only render
// object elements have one.
func (b *ElementBase) ownRenderObject() (rendering.RenderObject, bool) { return nil, false }

// RenderObject is the render object of this element, or of its first
// descendant that has one. It is nil when there is none or the element is
// unmounted.
//
// Flutter: widgets/framework.dart Element.renderObject
func (b *ElementBase) RenderObject() rendering.RenderObject {
	current := b.self
	for current != nil {
		cb := current.AsElement()
		if cb.lifecycle == lifecycleDefunct {
			break
		}
		if ro, ok := current.ownRenderObject(); ok {
			return ro
		}
		current = current.RenderObjectAttachingChild()
	}
	return nil
}

// RenderObjectAttachingChild is the only child, found by visiting the
// children. An element with more than one child overrides it.
//
// Flutter: widgets/framework.dart Element.renderObjectAttachingChild
func (b *ElementBase) RenderObjectAttachingChild() Element {
	var next Element
	b.self.VisitChildren(func(child Element) { next = child })
	return next
}

// DescribeMissingAncestor describes this element and its ancestors for an error
// about an ancestor of the expected type that does not exist.
//
// Flutter: widgets/framework.dart Element.describeMissingAncestor
func (b *ElementBase) DescribeMissingAncestor(expectedAncestorType string) []foundation.DiagnosticsNode {
	var ancestors []Element
	b.self.VisitAncestorElements(func(e Element) bool {
		ancestors = append(ancestors, e)
		return true
	})
	info := []foundation.DiagnosticsNode{
		foundation.NewDiagnosticsProperty("The specific widget that could not find a "+expectedAncestorType+" ancestor was", b.self,
			foundation.DiagnosticsPropertyOpts{Style: foundation.DiagnosticsTreeStyleErrorProperty}),
	}
	if len(ancestors) > 0 {
		info = append(info, DescribeElements("The ancestors of this widget were", ancestors))
	} else {
		info = append(info, foundation.NewMessageProperty("",
			"This widget is the root of the tree, so it has no ancestors, let alone a \""+expectedAncestorType+"\" ancestor."))
	}
	return info
}

// DescribeElements describes a list of elements as a block, for an error.
//
// Flutter: widgets/framework.dart Element.describeElements
func DescribeElements(name string, elements []Element) foundation.DiagnosticsNode {
	children := make([]foundation.DiagnosticsNode, len(elements))
	for i, e := range elements {
		children[i] = foundation.NewDiagnosticsProperty("", e)
	}
	return foundation.NewDiagnosticsBlock(foundation.DiagnosticsBlockOpts{
		Name: foundation.Some(name), Children: children, AllowTruncate: true,
	})
}

// DescribeElement describes this element as a property, for an error.
//
// Flutter: widgets/framework.dart Element.describeElement
func (b *ElementBase) DescribeElement(name string, style ...foundation.DiagnosticsTreeStyle) foundation.DiagnosticsNode {
	return foundation.NewDiagnosticsProperty(name, b.self, foundation.DiagnosticsPropertyOpts{Style: errorStyle(style)})
}

// DescribeWidget describes this element as a property, for an error.
//
// Flutter: widgets/framework.dart Element.describeWidget
func (b *ElementBase) DescribeWidget(name string, style ...foundation.DiagnosticsTreeStyle) foundation.DiagnosticsNode {
	return foundation.NewDiagnosticsProperty(name, b.self, foundation.DiagnosticsPropertyOpts{Style: errorStyle(style)})
}

func errorStyle(style []foundation.DiagnosticsTreeStyle) foundation.DiagnosticsTreeStyle {
	if len(style) > 0 {
		return style[0]
	}
	return foundation.DiagnosticsTreeStyleErrorProperty
}

// DescribeOwnershipChain lists the elements from this one up to ten parents.
//
// Flutter: widgets/framework.dart Element.describeOwnershipChain
func (b *ElementBase) DescribeOwnershipChain(name string) foundation.DiagnosticsNode {
	return foundation.NewStringProperty(name, foundation.Some(b.DebugGetCreatorChain(10)))
}

// VisitChildren has no children by default.
//
// Flutter: widgets/framework.dart Element.visitChildren
func (b *ElementBase) VisitChildren(ElementVisitor) {}

// DebugVisitOnstageChildren is VisitChildren by default.
//
// Flutter: widgets/framework.dart Element.debugVisitOnstageChildren
func (b *ElementBase) DebugVisitOnstageChildren(visitor ElementVisitor) {
	b.self.VisitChildren(visitor)
}

// VisitChildElements calls the visitor for each child. In debug builds it
// panics with the tag VisitChildElementsDuringBuild while the tree is locked.
//
// Flutter: widgets/framework.dart Element.visitChildElements
func (b *ElementBase) VisitChildElements(visitor ElementVisitor) {
	if foundation.DebugMode && b.owner != nil && b.owner.debugStateLocked() {
		panic(errors.New("VisitChildElementsDuringBuild",
			"VisitChildElements was called during build; the child list is still being updated at that point, so the children might not be constructed yet, or might be old children that are going to be replaced"))
	}
	b.self.VisitChildren(visitor)
}

// DebugDoingBuild is whether the element is in its build function. Only
// component elements build.
//
// Flutter: widgets/framework.dart Element.debugDoingBuild
func (b *ElementBase) DebugDoingBuild() bool { return false }

// DebugExpectsRenderObjectForSlot is true by default.
//
// Flutter: widgets/framework.dart Element.debugExpectsRenderObjectForSlot
func (b *ElementBase) DebugExpectsRenderObjectForSlot(any) bool { return true }

// DetachRenderObject detaches the descendants and clears the slot.
//
// Flutter: widgets/framework.dart Element.detachRenderObject
func (b *ElementBase) DetachRenderObject() {
	b.self.VisitChildren(func(child Element) { child.DetachRenderObject() })
	b.slot = nil
}

// AttachRenderObject attaches the descendants and sets the slot.
//
// Flutter: widgets/framework.dart Element.attachRenderObject
func (b *ElementBase) AttachRenderObject(newSlot any) {
	b.self.VisitChildren(func(child Element) { child.AttachRenderObject(newSlot) })
	b.slot = newSlot
}

// FindRenderObject is the render object of this element or of the first
// descendant that has one. In debug builds it panics with the tag
// FindRenderObjectInactive when the element is not active.
//
// Flutter: widgets/framework.dart Element.findRenderObject
func (b *ElementBase) FindRenderObject() rendering.RenderObject {
	if foundation.DebugMode && b.lifecycle != lifecycleActive {
		panic(errors.New("FindRenderObjectInactive",
			"in order for an element to have a valid render object, it must be active, which means it is part of the tree; if you called this method from a State, consider guarding it with Mounted",
			"lifecycle", b.lifecycle.String(), "element", b.ToStringShort()))
	}
	return b.RenderObject()
}

func (b *ElementBase) findAncestorRenderObject(accept func(rendering.RenderObject) bool) rendering.RenderObject {
	b.debugCheckStateIsActiveForAncestorLookup()
	for a := b.parent; a != nil; a = a.AsElement().parent {
		if ro, ok := a.ownRenderObject(); ok && accept(ro) {
			return ro
		}
	}
	return nil
}

func (b *ElementBase) debugCheckStateIsActiveForAncestorLookup() {
	if foundation.DebugMode && b.lifecycle != lifecycleActive {
		panic(errors.New("AncestorLookupOnInactiveElement",
			"looking up a deactivated widget's ancestor is unsafe; at this point the state of the widget's element tree is no longer stable; to safely refer to a widget's ancestor in its Dispose method, save a reference to the ancestor by looking it up in DidChangeDependencies",
			"lifecycle", b.lifecycle.String()))
	}
}

// VisitAncestorElements walks the ancestors from the closest to the root until
// the visitor returns false.
//
// Flutter: widgets/framework.dart Element.visitAncestorElements
func (b *ElementBase) VisitAncestorElements(visitor ConditionalElementVisitor) {
	b.debugCheckStateIsActiveForAncestorLookup()
	for a := b.parent; a != nil && visitor(a); a = a.AsElement().parent {
	}
}

// DebugGetCreatorChain describes this element and up to limit parents, such as
// "Padding ← Center ← ⋯".
//
// Flutter: widgets/framework.dart Element.debugGetCreatorChain
func (b *ElementBase) DebugGetCreatorChain(limit int) string {
	var chain []string
	node := b.self
	for len(chain) < limit && node != nil {
		chain = append(chain, foundation.ToStringShort(node))
		node = node.AsElement().parent
	}
	if node != nil {
		chain = append(chain, "⋯")
	}
	return strings.Join(chain, " ← ")
}

// DebugGetDiagnosticChain is this element followed by its ancestors.
//
// Flutter: widgets/framework.dart Element.debugGetDiagnosticChain
func (b *ElementBase) DebugGetDiagnosticChain() []Element {
	chain := []Element{b.self}
	for n := b.parent; n != nil; n = n.AsElement().parent {
		chain = append(chain, n)
	}
	return chain
}

// ToStringShort is the short description of the widget, or the identity of the
// element with "(DEFUNCT)" once the widget is gone.
//
// Flutter: widgets/framework.dart Element.toStringShort
func (b *ElementBase) ToStringShort() string {
	if b.widget != nil {
		return WidgetToStringShort(b.widget)
	}
	return foundation.DescribeIdentity(b.self) + "(DEFUNCT)"
}

// String describes the element and its properties on one line.
//
// Flutter: foundation/diagnostics.dart DiagnosticableTree.toString
func (b *ElementBase) String() string { return foundation.DiagnosticableString(b.self) }

// DebugFillProperties adds the depth, the widget, its key and properties, the
// dirty flag and the dependencies.
//
// Flutter: widgets/framework.dart Element.debugFillProperties
func (b *ElementBase) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	properties.DefaultDiagnosticsTreeStyle = foundation.DiagnosticsTreeStyleDense
	if b.lifecycle != lifecycleInitial {
		properties.Add(foundation.NewObjectFlagProperty("depth", b.depth, foundation.ObjectFlagPropertyOpts{IfNull: "no depth"}))
	}
	properties.Add(foundation.NewObjectFlagProperty("widget", b.widget, foundation.ObjectFlagPropertyOpts{IfNull: "no widget"}))
	var key foundation.Key
	if b.widget != nil {
		key = b.widget.Key()
	}
	properties.Add(foundation.NewDiagnosticsProperty("key", key, foundation.DiagnosticsPropertyOpts{
		ShowName: foundation.Some(false), DefaultValue: foundation.DefaultNull, Level: foundation.DiagnosticLevelHidden,
	}))
	if d, ok := b.widget.(foundation.Diagnosticable); ok {
		d.DebugFillProperties(properties)
	}
	properties.Add(foundation.NewFlagProperty("dirty", foundation.Some(b.dirty), foundation.FlagPropertyOpts{IfTrue: "dirty"}))
	if len(b.dependencies) > 0 {
		deps := make([]Element, 0, len(b.dependencies))
		for d := range b.dependencies {
			deps = append(deps, d)
		}
		sortElementsByShortName(deps)
		descriptions := make([]string, len(deps))
		for i, d := range deps {
			descriptions[i] = foundation.ToStringDeep(&widgetDescriptionTree{d.AsElement().widget}, foundation.ToStringDeepOpts{})
		}
		properties.Add(foundation.NewDiagnosticsProperty("dependencies", deps, foundation.DiagnosticsPropertyOpts{
			Description: "[" + strings.Join(descriptions, ", ") + "]",
		}))
	}
}

func sortElementsByShortName(es []Element) {
	for i := 1; i < len(es); i++ {
		for j := i; j > 0 && es[j].ToStringShort() < es[j-1].ToStringShort(); j-- {
			es[j], es[j-1] = es[j-1], es[j]
		}
	}
}

// widgetDescriptionTree prints a widget as a tree node without children.
type widgetDescriptionTree struct{ w Widget }

func (d *widgetDescriptionTree) ToStringShort() string { return WidgetToStringShort(d.w) }

func (d *widgetDescriptionTree) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	if dd, ok := d.w.(foundation.Diagnosticable); ok {
		dd.DebugFillProperties(p)
	}
}

func (d *widgetDescriptionTree) DebugDescribeChildren() []foundation.DiagnosticsNode { return nil }

// DebugDescribeChildren describes the children in visiting order.
//
// Flutter: widgets/framework.dart Element.debugDescribeChildren
func (b *ElementBase) DebugDescribeChildren() []foundation.DiagnosticsNode {
	var children []foundation.DiagnosticsNode
	b.self.VisitChildren(func(child Element) {
		children = append(children, foundation.ToDiagnosticsNode(child))
	})
	return children
}

// MarkNeedsBuild marks the element as dirty and adds it to the list of active
// elements that the owner will rebuild. In debug builds it panics with the tag
// MarkNeedsBuildDuringBuild when the element is not below the one being built,
// with MarkNeedsBuildWhileLocked while the tree is locked, and with
// MarkNeedsBuildOnDefunctElement for an element that has been unmounted.
//
// Flutter: widgets/framework.dart Element.markNeedsBuild
func (b *ElementBase) MarkNeedsBuild() {
	if foundation.DebugMode && b.lifecycle == lifecycleDefunct {
		panic(errors.New("MarkNeedsBuildOnDefunctElement",
			"SetState or MarkNeedsBuild was called on an element that has been unmounted",
			"element", b.ToStringShort()))
	}
	if b.lifecycle != lifecycleActive {
		return
	}
	if foundation.DebugMode {
		if b.owner.debugBuilding {
			if b.isDescendantOf(b.owner.debugCurrentBuildTarget) {
				// A descendant of the element being built is built later.
			} else {
				panic(errors.New("MarkNeedsBuildDuringBuild",
					"SetState or MarkNeedsBuild was called during build; a widget can be marked as needing to be built during the build phase only if one of its ancestors is currently building, because the framework builds parent widgets before children, which means a dirty descendant will always be built; otherwise, the framework might not visit this widget during this build phase",
					"widget", foundation.RuntimeTypeName(b.widget),
					"element", b.self.String(),
					"building", b.owner.debugCurrentBuildTarget.String()))
			}
		} else if b.owner.debugStateLocked() {
			panic(errors.New("MarkNeedsBuildWhileLocked",
				"SetState or MarkNeedsBuild was called when the widget tree was locked; the widget cannot be marked as needing to build because the framework is locked",
				"widget", foundation.RuntimeTypeName(b.widget), "element", b.self.String()))
		}
	}
	if b.dirty {
		return
	}
	b.dirty = true
	b.owner.ScheduleBuildFor(b.self)
}

// Rebuild builds the element if it is dirty (or always with Force) and clears
// the dirty flag.
//
// Flutter: widgets/framework.dart Element.rebuild
func (b *ElementBase) Rebuild(opts ...RebuildOpts) {
	force := firstOpts(opts).Force
	if b.lifecycle != lifecycleActive || (!b.dirty && !force) {
		return
	}
	var previous Element
	if foundation.DebugMode {
		previous = b.owner.debugCurrentBuildTarget
		b.owner.debugCurrentBuildTarget = b.self
	}
	defer func() {
		if foundation.DebugMode {
			b.owner.debugElementWasRebuilt(b.self)
			b.owner.debugCurrentBuildTarget = previous
		}
	}()
	b.self.PerformRebuild()
}

// PerformRebuild clears the dirty flag; elements that build override it and
// call this one.
//
// Flutter: widgets/framework.dart Element.performRebuild
func (b *ElementBase) PerformRebuild() { b.dirty = false }

// DidChangeDependencies is called when a dependency of this element changes: it
// marks the element for a rebuild. In debug builds it panics with the tag
// DidChangeDependenciesOutsideBuild when no build is going on.
//
// Flutter: widgets/framework.dart Element.didChangeDependencies
func (b *ElementBase) DidChangeDependencies() {
	if foundation.DebugMode && b.owner.debugCurrentBuildTarget == nil {
		panic(errors.New("DidChangeDependenciesOutsideBuild",
			"DidChangeDependencies was called at an inappropriate time; it may only be called while the widgets are being built (not from a network or file I/O event, a timer, or a goroutine)",
			"widget", foundation.RuntimeTypeName(b.widget)))
	}
	b.self.MarkNeedsBuild()
}
