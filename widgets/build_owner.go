package widgets

import (
	"slices"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
)

// BuildScope is a set of dirty elements that are built together, in order of
// depth. A BuildOwner builds one scope at a time; a scope lets part of the tree
// (for example a LayoutBuilder's subtree) be built at another time than the
// rest.
//
// Flutter: widgets/framework.dart BuildScope
type BuildScope struct {
	scheduleRebuild func()
	// buildScheduled is whether scheduleRebuild has been called and the scope has
	// not been built since.
	buildScheduled bool
	// building is whether BuildOwner.BuildScope is running for this scope.
	building bool
	// dirtyElementsNeedsResorting is nil while the scope is not flushing, and
	// says whether the dirty list must be sorted again otherwise.
	dirtyElementsNeedsResorting *bool
	dirtyElements               []Element
}

// BuildScopeOpts are the optional parameters of NewBuildScope.
//
// Flutter: widgets/framework.dart BuildScope.new
type BuildScopeOpts struct {
	// ScheduleRebuild is called when an element of the scope becomes dirty
	// and the scope is not being built.
	ScheduleRebuild func()
}

// NewBuildScope creates a build scope.
//
// Flutter: widgets/framework.dart BuildScope.new
func NewBuildScope(opts ...BuildScopeOpts) *BuildScope {
	return &BuildScope{scheduleRebuild: firstOpts(opts).ScheduleRebuild}
}

func (s *BuildScope) scheduleBuildFor(element Element) {
	eb := element.AsElement()
	if !eb.inDirtyList {
		s.dirtyElements = append(s.dirtyElements, element)
		eb.inDirtyList = true
	}
	if !s.buildScheduled && !s.building {
		s.buildScheduled = true
		if s.scheduleRebuild != nil {
			s.scheduleRebuild()
		}
	}
	if s.dirtyElementsNeedsResorting != nil {
		*s.dirtyElementsNeedsResorting = true
	}
}

// sortElements orders elements by depth, with the clean ones before the dirty
// ones at the same depth. The sort is stable.
func sortElements(elements []Element) {
	slices.SortStableFunc(elements, compareElements)
}

func compareElements(a, b Element) int {
	ab, bb := a.AsElement(), b.AsElement()
	if diff := ab.depth - bb.depth; diff != 0 {
		return diff
	}
	if ab.dirty != bb.dirty {
		if bb.dirty {
			return -1
		}
		return 1
	}
	return 0
}

func (s *BuildScope) tryRebuild(element Element) {
	// TODO(step 7.3): recover from a panic here, report it and go on with the
	// next dirty element.
	element.Rebuild()
}

func (s *BuildScope) flushDirtyElements() {
	needsResorting := false
	sortElements(s.dirtyElements)
	s.dirtyElementsNeedsResorting = &needsResorting
	defer func() {
		for _, element := range s.dirtyElements {
			if element.BuildScope() == s {
				element.AsElement().inDirtyList = false
			}
		}
		s.dirtyElements = nil
		s.dirtyElementsNeedsResorting = nil
		s.buildScheduled = false
	}()
	for index := 0; index < len(s.dirtyElements); index = s.dirtyElementIndexAfter(index) {
		element := s.dirtyElements[index]
		if element.BuildScope() == s {
			s.tryRebuild(element)
		}
	}
}

func (s *BuildScope) dirtyElementIndexAfter(index int) int {
	if !*s.dirtyElementsNeedsResorting {
		return index + 1
	}
	index++
	sortElements(s.dirtyElements)
	*s.dirtyElementsNeedsResorting = false
	// Previously dirty but inactive elements can move right in the list, so the
	// index moves left until it is just after the right-most clean element.
	for index > 0 && s.dirtyElements[index-1].Dirty() {
		index--
	}
	return index
}

// BuildOwner manages the widgets framework: it tracks which elements need
// rebuilding, rebuilds them in order of depth, and finalizes the tree by
// unmounting the elements that were deactivated.
//
// Flutter: widgets/framework.dart BuildOwner
type BuildOwner struct {
	// OnBuildScheduled is called when the first element becomes dirty; the
	// owner of a binding uses it to schedule a frame.
	//
	// Flutter: widgets/framework.dart BuildOwner.onBuildScheduled
	OnBuildScheduled func()

	inactive                    inactiveElements
	scheduledFlushDirtyElements bool
	debugStateLockLevel         int
	debugBuilding               bool
	debugCurrentBuildTarget     Element
}

// BuildOwnerOpts are the optional parameters of NewBuildOwner.
//
// Flutter: widgets/framework.dart BuildOwner.new
type BuildOwnerOpts struct {
	// OnBuildScheduled is called when the first element becomes dirty.
	OnBuildScheduled func()
}

// NewBuildOwner creates a build owner.
//
// Flutter: widgets/framework.dart BuildOwner.new
func NewBuildOwner(opts ...BuildOwnerOpts) *BuildOwner {
	return &BuildOwner{OnBuildScheduled: firstOpts(opts).OnBuildScheduled}
}

func (o *BuildOwner) debugStateLocked() bool { return o.debugStateLockLevel > 0 }

// DebugBuilding is whether the owner is in BuildScope. It is always false in
// release builds.
//
// Flutter: widgets/framework.dart BuildOwner.debugBuilding
func (o *BuildOwner) DebugBuilding() bool { return o.debugBuilding }

// ScheduleBuildFor adds the element to the dirty list of its build scope; the
// element must already be marked dirty. In debug builds it panics with the tag
// ScheduleBuildForNotDirty for an element that is not dirty, and with
// ScheduleBuildForInappropriately for an element that is already in the list
// outside of a build.
//
// Flutter: widgets/framework.dart BuildOwner.scheduleBuildFor
func (o *BuildOwner) ScheduleBuildFor(element Element) {
	eb := element.AsElement()
	if foundation.DebugMode {
		if !eb.dirty {
			panic(errors.New("ScheduleBuildForNotDirty",
				"ScheduleBuildFor was called for a widget that is not marked as dirty; set the dirty flag before calling it",
				"element", element.String()))
		}
		if !o.debugBuilding && eb.inDirtyList {
			panic(errors.New("ScheduleBuildForInappropriately",
				"BuildOwner.ScheduleBuildFor was called on an element that is already in the dirty list",
				"element", element.String()))
		}
	}
	scope := element.BuildScope()
	if !o.scheduledFlushDirtyElements && o.OnBuildScheduled != nil {
		o.scheduledFlushDirtyElements = true
		o.OnBuildScheduled()
	}
	scope.scheduleBuildFor(element)
}

// LockState runs the callback with the tree locked: in debug builds, marking
// an element dirty inside it panics with MarkNeedsBuildWhileLocked.
//
// Flutter: widgets/framework.dart BuildOwner.lockState
func (o *BuildOwner) LockState(callback func()) {
	if foundation.DebugMode {
		o.debugStateLockLevel++
		defer func() { o.debugStateLockLevel-- }()
	}
	callback()
}

// BuildScope builds the dirty elements of the scope of context, from the
// shallowest to the deepest, and elements that become dirty on the way if they
// are below. It first runs the callback, if there is one, with the tree locked.
// Pass a nil callback to just build.
//
// Deviation: it is a method with the callback as a plain argument, where
// Flutter's is an optional positional one (SPEC §12.10).
//
// Flutter: widgets/framework.dart BuildOwner.buildScope
func (o *BuildOwner) BuildScope(context Element, callback func()) {
	scope := context.BuildScope()
	if callback == nil && len(scope.dirtyElements) == 0 {
		return
	}
	if foundation.DebugMode {
		o.debugStateLockLevel++
		o.debugBuilding = true
	}
	defer func() {
		scope.building = false
		o.scheduledFlushDirtyElements = false
		if foundation.DebugMode {
			o.debugBuilding = false
			o.debugStateLockLevel--
		}
	}()
	o.scheduledFlushDirtyElements = true
	scope.building = true
	if callback != nil {
		var previous Element
		if foundation.DebugMode {
			previous = o.debugCurrentBuildTarget
			o.debugCurrentBuildTarget = context
		}
		func() {
			defer func() {
				if foundation.DebugMode {
					o.debugCurrentBuildTarget = previous
					o.debugElementWasRebuilt(context)
				}
			}()
			callback()
		}()
	}
	scope.flushDirtyElements()
}

// debugElementWasRebuilt is told that an element was built. The GlobalKey
// bookkeeping of step 7.2 uses it.
func (o *BuildOwner) debugElementWasRebuilt(Element) {}

// FinalizeTree completes the build pass by unmounting the elements that are no
// longer active. The binding calls it at the end of a frame.
//
// Flutter: widgets/framework.dart BuildOwner.finalizeTree
func (o *BuildOwner) FinalizeTree() {
	o.LockState(o.inactive.unmountAll)
	// TODO(step 7.2): verify the global keys (reservations, duplicates).
}

// inactiveElements holds the elements that were removed from the tree during a
// build. If they are not put back by the end of the frame, they are unmounted.
// The set keeps the order of insertion, so that the unmount order does not
// depend on a map.
//
// Flutter: widgets/framework.dart _InactiveElements
type inactiveElements struct {
	locked   bool
	elements []Element
	set      map[Element]struct{}
}

func (ie *inactiveElements) contains(e Element) bool {
	_, ok := ie.set[e]
	return ok
}

func unmountInactive(element Element) {
	element.VisitChildren(unmountInactive)
	element.Unmount()
}

func (ie *inactiveElements) unmountAll() {
	ie.locked = true
	elements := ie.elements
	ie.elements, ie.set = nil, nil
	sortElements(elements)
	defer func() { ie.locked = false }()
	for i := len(elements) - 1; i >= 0; i-- {
		unmountInactive(elements[i])
	}
}

func deactivateRecursively(element Element) {
	func() {
		defer func() {
			if r := recover(); r != nil {
				deactivateFailedSubtreeRecursively(element)
				panic(r)
			}
		}()
		element.Deactivate()
	}()
	element.VisitChildren(deactivateRecursively)
	element.DebugDeactivated()
}

func (ie *inactiveElements) add(element Element) {
	eb := element.AsElement()
	switch eb.lifecycle {
	case lifecycleActive:
		deactivateRecursively(element)
		// The element is added only if the whole subtree was deactivated.
		ie.put(element)
	case lifecycleInactive:
		ie.put(element)
	}
}

func (ie *inactiveElements) put(element Element) {
	if ie.set == nil {
		ie.set = map[Element]struct{}{}
	}
	if _, ok := ie.set[element]; !ok {
		ie.set[element] = struct{}{}
		ie.elements = append(ie.elements, element)
	}
}

func (ie *inactiveElements) remove(element Element) {
	delete(ie.set, element)
	ie.elements = slices.DeleteFunc(ie.elements, func(e Element) bool { return e == element })
}

// DebugInactiveElementsContain is whether the element is in the inactive list.
// It is false in release builds.
//
// Flutter: widgets/framework.dart _InactiveElements.debugContains
func (o *BuildOwner) DebugInactiveElementsContain(element Element) bool {
	return foundation.DebugMode && o.inactive.contains(element)
}
