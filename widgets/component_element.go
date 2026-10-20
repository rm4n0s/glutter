package widgets

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
)

// ComponentElementSelf is what a component element needs from the outer type:
// the Build method, which is Flutter's abstract ComponentElement.build.
//
// Flutter: no counterpart
type ComponentElementSelf interface {
	Element
	// Build describes the widget this element builds.
	//
	// Flutter: widgets/framework.dart ComponentElement.build
	Build() Widget
	firstBuild()
}

// ComponentElement is an element that is composed of other elements: it builds
// one child widget. Embed it, call InitComponentElement and implement Build.
//
// Flutter: widgets/framework.dart ComponentElement
type ComponentElement struct {
	ElementBase
	cself           ComponentElementSelf
	child           Element
	debugDoingBuild bool
}

// InitComponentElement sets up a component element; self is the outer element.
//
// Flutter: widgets/framework.dart ComponentElement.new
func InitComponentElement(c *ComponentElement, self ComponentElementSelf, widget Widget) {
	InitElement(&c.ElementBase, self, widget)
	c.cself = self
}

// DebugDoingBuild is whether the element is in its Build method.
//
// Flutter: widgets/framework.dart ComponentElement.debugDoingBuild
func (c *ComponentElement) DebugDoingBuild() bool { return c.debugDoingBuild }

// RenderObjectAttachingChild is the built child.
//
// Flutter: widgets/framework.dart ComponentElement.renderObjectAttachingChild
func (c *ComponentElement) RenderObjectAttachingChild() Element { return c.child }

// Mount mounts the element and builds the first child.
//
// Flutter: widgets/framework.dart ComponentElement.mount
func (c *ComponentElement) Mount(parent Element, newSlot any) {
	c.ElementBase.Mount(parent, newSlot)
	c.cself.firstBuild()
}

// firstBuild rebuilds; a stateful element also initializes its state first.
//
// Flutter: widgets/framework.dart ComponentElement._firstBuild
func (c *ComponentElement) firstBuild() { c.self.Rebuild() }

// PerformRebuild calls Build, updates the child to what it returned and clears
// the dirty flag.
//
// Flutter: widgets/framework.dart ComponentElement.performRebuild
func (c *ComponentElement) PerformRebuild() {
	owner := c.owner
	var built Widget
	func() {
		// The element is marked clean only after Build, so that attempts to
		// MarkNeedsBuild during Build are ignored.
		defer c.ElementBase.PerformRebuild()
		if foundation.DebugMode {
			c.debugDoingBuild = true
			defer func() { c.debugDoingBuild = false }()
		}
		// Build is a recovery point (SPEC §14.2): with a reporter, a panic is
		// reported and the ErrorWidget takes the place of the built widget.
		var buildErr error
		report := func(err error) { buildErr = err }
		if owner == nil || owner.OnError == nil {
			report = nil
		}
		foundation.GuardLazy(report, foundation.BuildPanicked, c.buildContext, c.panicInfo, func() {
			built = c.cself.Build()
			if foundation.DebugMode && built == nil {
				panic(errors.New("BuildReturnedNil",
					"a build function returned nil; build functions must never return nil, to return an empty space that causes the building widget to fill available room, return a Container{}; to return an empty space that takes as little room as possible, return a SizedBox{}",
					"widget", foundation.RuntimeTypeName(c.widget)))
			}
		})
		if buildErr != nil {
			c.debugDoingBuild = false
			owner.OnError(buildErr)
			built = owner.errorWidgetFor(buildErr)
		}
	}()
	if owner == nil || owner.OnError == nil {
		c.child = c.UpdateChild(c.child, built, c.slot)
		return
	}
	var updateErr error
	foundation.GuardLazy(func(err error) { updateErr = err }, foundation.BuildPanicked, c.buildContext, c.panicInfo, func() {
		c.child = c.UpdateChild(c.child, built, c.slot)
	})
	if updateErr != nil {
		owner.OnError(updateErr)
		// As in Flutter the old child is left to the tree: the update stopped
		// half way, and the error widget is built from scratch.
		c.child = c.UpdateChild(nil, owner.errorWidgetFor(updateErr), c.slot)
	}
}

// buildContext says what was happening when a build panicked.
func (c *ComponentElement) buildContext() string { return "while building " + c.String() }

// panicInfo is the metadata of a report about this element: the widget and the
// chain of the elements that created it.
func (c *ComponentElement) panicInfo() []any {
	return []any{"library", "widgets library", "element", c.String(), "widget", foundation.RuntimeTypeName(c.widget),
		"creatorChain", c.DebugGetCreatorChain(12)}
}

// VisitChildren visits the built child.
//
// Flutter: widgets/framework.dart ComponentElement.visitChildren
func (c *ComponentElement) VisitChildren(visitor ElementVisitor) {
	if c.child != nil {
		visitor(c.child)
	}
}

// ForgetChild drops the child.
//
// Flutter: widgets/framework.dart ComponentElement.forgetChild
func (c *ComponentElement) ForgetChild(child Element) {
	c.child = nil
	c.ElementBase.ForgetChild(child)
}

// StatelessElement is the element of a StatelessWidget.
//
// Flutter: widgets/framework.dart StatelessElement
type StatelessElement struct {
	ComponentElement
}

// NewStatelessElement creates the element of a widget that has a Build method.
//
// Flutter: widgets/framework.dart StatelessElement.new
func NewStatelessElement(w statelessWidget) *StatelessElement {
	e := &StatelessElement{}
	InitComponentElement(&e.ComponentElement, e, w)
	return e
}

// Build calls the Build method of the widget with this element as the context.
//
// Flutter: widgets/framework.dart StatelessElement.build
func (e *StatelessElement) Build() Widget { return e.widget.(statelessWidget).Build(e) }

// Update replaces the widget and rebuilds.
//
// Flutter: widgets/framework.dart StatelessElement.update
func (e *StatelessElement) Update(newWidget Widget) {
	e.ComponentElement.Update(newWidget)
	e.Rebuild(RebuildOpts{Force: true})
}

// StatefulElement is the element of a StatefulWidget. It owns the State.
//
// Flutter: widgets/framework.dart StatefulElement
type StatefulElement struct {
	ComponentElement
	state               State
	didChangeDependency bool
}

// NewStatefulElement creates the element of a widget that has CreateState and
// the state the widget creates. It panics with the tag CreateStateReturnedNil
// when CreateState returns nil, with CreateStateWrongType when the state is for
// another widget type, and with CreateStateReturnedUsedState when the state
// already belongs to an element.
//
// Flutter: widgets/framework.dart StatefulElement.new
func NewStatefulElement(w statefulWidget) *StatefulElement {
	e := &StatefulElement{}
	InitComponentElement(&e.ComponentElement, e, w)
	state := w.CreateState()
	if state == nil {
		panic(errors.New("CreateStateReturnedNil", "CreateState returned nil",
			"widget", foundation.RuntimeTypeName(w)))
	}
	si := state.stateInternals()
	sd := si.data()
	if sd.element != nil || si.widgetAny() != nil {
		panic(errors.New("CreateStateReturnedUsedState",
			"CreateState returned an old or invalid state instance, which already belongs to a widget; it must return a new State every time",
			"widget", foundation.RuntimeTypeName(w), "state", foundation.RuntimeTypeName(state)))
	}
	if !si.bind(w) {
		panic(errors.New("CreateStateWrongType",
			"CreateState must return a State for the type of its widget, but it returned a state of another type",
			"widget", foundation.RuntimeTypeName(w), "state", foundation.RuntimeTypeName(state)))
	}
	sd.self = state
	sd.element = e
	sd.lifecycle = StateLifecycleCreated
	e.state = state
	sd.mixins = collectStateMixins(state)
	for _, m := range sd.mixins {
		m.attachStateMixin(state)
	}
	return e
}

// State is the state of the widget. It panics with the tag
// ElementStateUnmounted once the element is unmounted.
//
// Flutter: widgets/framework.dart StatefulElement.state
func (e *StatefulElement) State() State {
	if e.state == nil {
		panic(errors.New("ElementStateUnmounted", "the state of an element is gone once the element is unmounted"))
	}
	return e.state
}

// Build calls the Build method of the state.
//
// Flutter: widgets/framework.dart StatefulElement.build
func (e *StatefulElement) Build() Widget { return e.state.Build(e) }

// Reassemble calls the Reassemble hook of the state and marks the subtree
// dirty.
//
// Flutter: widgets/framework.dart StatefulElement.reassemble
func (e *StatefulElement) Reassemble() {
	if h, ok := e.state.(reassembleStater); ok {
		h.Reassemble()
	}
	e.ComponentElement.Reassemble()
}

// firstBuild runs InitState and DidChangeDependencies, then builds.
//
// Flutter: widgets/framework.dart StatefulElement._firstBuild
func (e *StatefulElement) firstBuild() {
	sd := e.state.stateInternals().data()
	sd.owner = e.owner
	for _, m := range sd.mixins {
		if h, ok := m.(stateInitMixin); ok {
			h.initStateMixin()
		}
	}
	if h, ok := e.state.(initStater); ok {
		h.InitState()
	}
	sd.lifecycle = StateLifecycleInitialized
	if h, ok := e.state.(didChangeDependenciesStater); ok {
		h.DidChangeDependencies()
	}
	sd.lifecycle = StateLifecycleReady
	e.ComponentElement.firstBuild()
}

// PerformRebuild calls DidChangeDependencies on the state first if a
// dependency changed.
//
// Flutter: widgets/framework.dart StatefulElement.performRebuild
func (e *StatefulElement) PerformRebuild() {
	if e.didChangeDependency {
		if h, ok := e.state.(didChangeDependenciesStater); ok {
			h.DidChangeDependencies()
		}
		e.didChangeDependency = false
	}
	e.ComponentElement.PerformRebuild()
}

// Update gives the new widget to the state, calls its DidUpdateWidget hook with
// the old one, and rebuilds.
//
// Flutter: widgets/framework.dart StatefulElement.update
func (e *StatefulElement) Update(newWidget Widget) {
	e.ComponentElement.Update(newWidget)
	old := e.state.stateInternals().swapWidget(newWidget)
	e.state.stateInternals().didUpdateWidget(old)
	e.Rebuild(RebuildOpts{Force: true})
}

// Activate calls the Activate hook of the state and rebuilds, so that the state
// can allocate the resources that Deactivate released.
//
// Flutter: widgets/framework.dart StatefulElement.activate
func (e *StatefulElement) Activate() {
	e.ComponentElement.Activate()
	for _, m := range e.state.stateInternals().data().mixins {
		m.activateStateMixin()
	}
	if h, ok := e.state.(activateStater); ok {
		h.Activate()
	}
	e.MarkNeedsBuild()
}

// Deactivate calls the Deactivate hook of the state.
//
// Flutter: widgets/framework.dart StatefulElement.deactivate
func (e *StatefulElement) Deactivate() {
	for _, m := range e.state.stateInternals().data().mixins {
		if h, ok := m.(stateDeactivateMixin); ok {
			h.deactivateStateMixin()
		}
	}
	if h, ok := e.state.(deactivateStater); ok {
		h.Deactivate()
	}
	e.ComponentElement.Deactivate()
}

// Unmount calls the Dispose hook of the state and cancels its Lifetime.
//
// Flutter: widgets/framework.dart StatefulElement.unmount
func (e *StatefulElement) Unmount() {
	e.ComponentElement.Unmount()
	sd := e.state.stateInternals().data()
	if h, ok := e.state.(disposeStater); ok {
		h.Dispose()
	}
	for _, m := range sd.mixins {
		m.disposeStateMixin()
	}
	sd.lifecycle = StateLifecycleDefunct
	if sd.cancel != nil {
		sd.cancel()
	}
	sd.element = nil
	// Release the state to reduce the severity of memory leaks caused by
	// defunct, but accidentally retained elements.
	e.state = nil
}

// DidChangeDependencies marks the element dirty and remembers to tell the
// state.
//
// Flutter: widgets/framework.dart StatefulElement.didChangeDependencies
func (e *StatefulElement) DidChangeDependencies() {
	e.ComponentElement.DidChangeDependencies()
	e.didChangeDependency = true
}

// DebugFillProperties adds the state.
//
// Flutter: widgets/framework.dart StatefulElement.debugFillProperties
func (e *StatefulElement) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	e.ComponentElement.DebugFillProperties(properties)
	properties.Add(foundation.NewDiagnosticsProperty("state", e.state,
		foundation.DiagnosticsPropertyOpts{DefaultValue: foundation.DefaultNull}))
}
