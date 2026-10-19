package widgets

import (
	"slices"
	"strconv"
	"strings"

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

// tryRebuild rebuilds the element. It is a recovery point (SPEC §14.2): with a
// reporter a panic is reported and the next dirty element is built. A panic in
// Build itself never gets here, because PerformRebuild turns it into an
// ErrorWidget; this catches the other failures of a rebuild, such as the
// lifecycle hooks of a state.
//
// Flutter: widgets/framework.dart BuildScope._tryRebuild
func (s *BuildScope) tryRebuild(element Element) {
	var report foundation.ErrorReporter
	if o := element.AsElement().owner; o != nil && o.OnError != nil {
		report = o.OnError
	}
	foundation.Guard(report, foundation.BuildPanicked, "while rebuilding dirty elements", func() []any {
		return []any{"library", "widgets library", "element", element.String(),
			"creatorChain", element.AsElement().DebugGetCreatorChain(12)}
	}, func() { element.Rebuild() })
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
	// OnError is the framework's error reporter: it receives the errors the
	// framework reports instead of throwing, such as the problems it finds while
	// it finalizes the tree or applies parent data, and the panics it recovers
	// from in Build and in the lifecycle hooks of a state (SPEC §14.2). Without
	// it these errors panic, because they must not pass unnoticed.
	//
	// Flutter: foundation/assertions.dart FlutterError.onError
	OnError foundation.ErrorReporter
	// Post runs a function on the UI thread, from any goroutine; the binding
	// passes its own. SetState called from another goroutine and the errors of
	// State.Go go through it (SPEC §13). Without it, only the UI thread may call
	// SetState.
	//
	// Flutter: no counterpart
	Post func(fn func())
	// UIThread says which goroutine is the UI thread. Debug builds check it in
	// MarkNeedsBuild, and SetState uses it to tell the UI thread from the
	// goroutines that must post. Without it nothing is checked.
	//
	// Flutter: no counterpart
	UIThread *foundation.UIThread
	// ErrorWidgetBuilder builds the widget that takes the place of a widget that
	// panicked in Build. The default is NewErrorWidgetWithDetails, with the text
	// of the error as the message in debug builds and an empty message in
	// release builds.
	//
	// Flutter: widgets/framework.dart ErrorWidget.builder
	ErrorWidgetBuilder func(err error) Widget

	inactive                    inactiveElements
	scheduledFlushDirtyElements bool
	debugStateLockLevel         int
	debugBuilding               bool
	debugCurrentBuildTarget     Element

	globalKeyRegistry map[AnyGlobalKey]Element
	// The debug checks of global keys: the elements that lost their key to
	// another element, which parent reserves which key for which child, and the
	// parents that lost a child to another parent and must update.
	debugIllFatedElements      orderedSet[Element]
	debugGlobalKeyReservations orderedMap[Element, *orderedMap[Element, AnyGlobalKey]]
	debugToRebuildForKeys      orderedMap[Element, *orderedSet[AnyGlobalKey]]
}

// BuildOwnerOpts are the optional parameters of NewBuildOwner.
//
// Flutter: widgets/framework.dart BuildOwner.new
type BuildOwnerOpts struct {
	// OnBuildScheduled is called when the first element becomes dirty.
	OnBuildScheduled func()
	// OnError is the framework's error reporter (see BuildOwner.OnError).
	OnError foundation.ErrorReporter
	// Post runs a function on the UI thread, from any goroutine (see
	// BuildOwner.Post).
	Post func(fn func())
	// UIThread is the UI thread (see BuildOwner.UIThread).
	UIThread *foundation.UIThread
	// ErrorWidgetBuilder replaces the default ErrorWidget builder.
	ErrorWidgetBuilder func(err error) Widget
}

// NewBuildOwner creates a build owner.
//
// Flutter: widgets/framework.dart BuildOwner.new
func NewBuildOwner(opts ...BuildOwnerOpts) *BuildOwner {
	o := firstOpts(opts)
	return &BuildOwner{OnBuildScheduled: o.OnBuildScheduled, OnError: o.OnError, Post: o.Post, UIThread: o.UIThread,
		ErrorWidgetBuilder: o.ErrorWidgetBuilder}
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

// debugElementWasRebuilt is told that an element was built: it is not a parent
// that must still update after a global key moved a child away.
func (o *BuildOwner) debugElementWasRebuilt(node Element) {
	o.debugToRebuildForKeys.remove(node)
}

// reportFromAnyGoroutine gives the error to OnError on the UI thread: it posts it
// when the caller is another goroutine and Post is set.
func (o *BuildOwner) reportFromAnyGoroutine(err error) {
	if o.Post != nil && !o.UIThread.IsCurrent() {
		o.Post(func() { o.OnError(err) })
		return
	}
	o.OnError(err)
}

// errorWidgetFor builds the widget that replaces one that failed to build.
func (o *BuildOwner) errorWidgetFor(err error) Widget {
	if o.ErrorWidgetBuilder != nil {
		return o.ErrorWidgetBuilder(err)
	}
	message := ""
	if foundation.DebugMode {
		message = err.Error()
	}
	return NewErrorWidgetWithDetails(message, err)
}

// reportError hands the error to OnError, or panics with it when there is none.
func (o *BuildOwner) reportError(err error) {
	if o.OnError == nil {
		panic(err)
	}
	o.OnError(err)
}

// FinalizeTree completes the build pass by unmounting the elements that are no
// longer active. In debug builds it also checks the global keys: it reports
// (see OnError) a key that is used by more than one widget, with the tags
// GlobalKeyUsedByMultipleWidgets, GlobalKeyUsedByMultipleParents and
// GlobalKeyDuplicateDetected. The binding calls it at the end of a frame.
//
// Flutter: widgets/framework.dart BuildOwner.finalizeTree
func (o *BuildOwner) FinalizeTree() {
	// The error is reported directly instead of building an ErrorWidget: the tree
	// is in a broken state and more widgets would only cause more errors.
	r := recoverValue(func() {
		o.LockState(o.inactive.unmountAll)
		if foundation.DebugMode {
			o.debugVerifyGlobalKeyReservation()
			o.debugVerifyIllFatedPopulation()
			o.debugVerifyKeyShenanigans()
		}
	})
	o.debugToRebuildForKeys.clear()
	if r != nil {
		o.reportPanic(r)
	}
}

// recoverValue runs fn and returns the value it panicked with, or nil.
func recoverValue(fn func()) (r any) {
	defer func() { r = recover() }()
	fn()
	return nil
}

// reportPanic reports a recovered panic value: to OnError as an error, or, when
// there is none, by panicking with the same value again.
func (o *BuildOwner) reportPanic(r any) {
	if o.OnError == nil {
		panic(r)
	}
	if err, ok := r.(error); ok {
		o.OnError(err)
		return
	}
	o.OnError(errors.New("UnexpectedPanic", "a panic that is not an error", "value", r))
}

// GlobalKeyCount is the number of global keys that are in use.
//
// Flutter: widgets/framework.dart BuildOwner.globalKeyCount
func (o *BuildOwner) GlobalKeyCount() int { return len(o.globalKeyRegistry) }

func (o *BuildOwner) registerGlobalKey(key AnyGlobalKey, element Element) {
	if foundation.DebugMode {
		if old, ok := o.globalKeyRegistry[key]; ok {
			o.debugIllFatedElements.add(old)
		}
	}
	if o.globalKeyRegistry == nil {
		o.globalKeyRegistry = map[AnyGlobalKey]Element{}
	}
	key.bind(o)
	o.globalKeyRegistry[key] = element
}

func (o *BuildOwner) unregisterGlobalKey(key AnyGlobalKey, element Element) {
	if o.globalKeyRegistry[key] == element {
		delete(o.globalKeyRegistry, key)
	}
}

func (o *BuildOwner) debugReserveGlobalKeyFor(parent, child Element, key AnyGlobalKey) {
	if !foundation.DebugMode {
		return
	}
	reservations, ok := o.debugGlobalKeyReservations.get(parent)
	if !ok {
		reservations = &orderedMap[Element, AnyGlobalKey]{}
		o.debugGlobalKeyReservations.put(parent, reservations)
	}
	reservations.put(child, key)
}

func (o *BuildOwner) debugRemoveGlobalKeyReservationFor(parent, child Element) {
	if !foundation.DebugMode {
		return
	}
	if reservations, ok := o.debugGlobalKeyReservations.get(parent); ok {
		reservations.remove(child)
	}
}

func (o *BuildOwner) debugTrackElementThatWillNeedToBeRebuiltDueToGlobalKeyShenanigans(node Element, key AnyGlobalKey) {
	keys, ok := o.debugToRebuildForKeys.get(node)
	if !ok {
		keys = &orderedSet[AnyGlobalKey]{}
		o.debugToRebuildForKeys.put(node, keys)
	}
	keys.add(key)
}

// debugVerifyGlobalKeyReservation checks that no global key is reserved by two
// parents, and fixes the tree for the error to end the test cleanly.
func (o *BuildOwner) debugVerifyGlobalKeyReservation() {
	keyToParent := map[AnyGlobalKey]Element{}
	var failure error
	for _, parent := range o.debugGlobalKeyReservations.keys {
		childToKey, _ := o.debugGlobalKeyReservations.get(parent)
		// Parents that are unmounted or detached are ignored.
		pb := parent.AsElement()
		if pb.lifecycle == lifecycleDefunct {
			continue
		}
		if ro := parent.RenderObject(); ro != nil && !ro.AsRenderObject().Attached() {
			continue
		}
		for _, child := range childToKey.keys {
			key, _ := childToKey.get(child)
			// A child without a parent was deactivated and not put back.
			if child.AsElement().parent == nil {
				continue
			}
			older, seen := keyToParent[key]
			if !seen || older == parent {
				keyToParent[key] = parent
				continue
			}
			newer := parent
			if failure == nil {
				if older.String() != newer.String() {
					failure = errors.New("GlobalKeyUsedByMultipleParents",
						"Multiple widgets used the same GlobalKey.\nThe key "+key.String()+" was used by multiple widgets. The parents of those widgets were:\n- "+
							older.String()+"\n- "+newer.String()+"\nA GlobalKey can only be specified on one widget at a time in the widget tree.",
						"key", key.String())
				} else {
					failure = errors.New("GlobalKeyUsedByMultipleParents",
						"Multiple widgets used the same GlobalKey.\nThe key "+key.String()+" was used by multiple widgets. The parents of those widgets were different widgets that both had the following description:\n  "+
							parent.String()+"\nA GlobalKey can only be specified on one widget at a time in the widget tree.",
						"key", key.String())
				}
			}
			// Fix the tree by removing the duplicated child from one of its parents.
			if child.AsElement().parent != older {
				older.VisitChildren(func(current Element) {
					if current == child {
						older.ForgetChild(child)
					}
				})
			}
			if child.AsElement().parent != newer {
				newer.VisitChildren(func(current Element) {
					if current == child {
						newer.ForgetChild(child)
					}
				})
			}
			panic(failure)
		}
	}
	o.debugGlobalKeyReservations.clear()
}

func (o *BuildOwner) debugVerifyIllFatedPopulation() {
	var duplicates orderedMap[AnyGlobalKey, *orderedSet[Element]]
	for _, element := range o.debugIllFatedElements.items {
		eb := element.AsElement()
		if eb.lifecycle == lifecycleDefunct {
			continue
		}
		key, ok := eb.widget.Key().(AnyGlobalKey)
		if !ok {
			continue
		}
		elements, ok := duplicates.get(key)
		if !ok {
			elements = &orderedSet[Element]{}
			duplicates.put(key, elements)
		}
		elements.add(element)
		if current, ok := o.globalKeyRegistry[key]; ok {
			elements.add(current)
		}
	}
	o.debugIllFatedElements.clear()
	if duplicates.len() == 0 {
		return
	}
	var msg strings.Builder
	msg.WriteString("Multiple widgets used the same GlobalKey.")
	var keys []string
	for _, key := range duplicates.keys {
		elements, _ := duplicates.get(key)
		keys = append(keys, key.String())
		msg.WriteString("\nThe key " + key.String() + " was used by " + strconv.Itoa(elements.len()) + " widgets:")
		for _, e := range elements.items {
			msg.WriteString("\n- " + e.String())
		}
	}
	msg.WriteString("\nA GlobalKey can only be specified on one widget at a time in the widget tree.")
	panic(errors.New("GlobalKeyUsedByMultipleWidgets", msg.String(), "keys", strings.Join(keys, ", ")))
}

// debugVerifyKeyShenanigans checks that the parents that lost a child to a
// global key updated afterwards, which they did not if the key is duplicated.
func (o *BuildOwner) debugVerifyKeyShenanigans() {
	if o.debugToRebuildForKeys.len() == 0 {
		return
	}
	var keys orderedSet[AnyGlobalKey]
	for _, element := range o.debugToRebuildForKeys.keys {
		if element.AsElement().lifecycle != lifecycleDefunct {
			ks, _ := o.debugToRebuildForKeys.get(element)
			for _, k := range ks.items {
				keys.add(k)
			}
		}
	}
	if keys.len() == 0 {
		return
	}
	keyLabels := countLabels(mapStrings(keys.items, func(k AnyGlobalKey) string { return k.String() }),
		"different affected keys had this toString representation")
	elementLabels := countLabels(mapStrings(o.debugToRebuildForKeys.keys, func(e Element) string { return e.String() }),
		"different affected elements had this toString representation")
	the, s, were, their, those := "", "s", "were", "their", "those"
	if keys.len() == 1 {
		the, s, were, their, those = " the", "", "was", "its", "that"
	}
	respective, s2, those2, they, think, are := " respective", "s", "those", "they", "think", "are"
	if len(elementLabels) == 1 {
		respective, s2, those2, they, think, are = "", "", "that", "it", "thinks", "is"
	}
	panic(errors.New("GlobalKeyDuplicateDetected",
		"Duplicate GlobalKey"+s+" detected in widget tree.\n"+
			"The following GlobalKey"+s+" "+were+" specified multiple times in the widget tree. This will lead to parts of the widget tree being truncated unexpectedly, because the second time a key is seen, the previous instance is moved to the new location. The key"+s+" "+were+":\n"+
			"- "+strings.Join(keyLabels, "\n  ")+"\n"+
			"This was determined by noticing that after"+the+" widget"+s+" with the above global key"+s+" "+were+" moved out of "+their+respective+" previous parent"+s2+", "+those2+" previous parent"+s2+" never updated during this frame, meaning that "+they+" either did not update at all or updated before the widget"+s+" "+were+" moved, in either case implying that "+they+" still "+think+" that "+they+" should have a child with "+those+" global key"+s+".\n"+
			"The specific parent"+s2+" that did not update after having one or more children forcibly removed due to GlobalKey reparenting "+are+":\n"+
			"- "+strings.Join(elementLabels, "\n  ")+
			"\nA GlobalKey can only be specified on one widget at a time in the widget tree.",
		"keys", strings.Join(keyLabels, ", ")))
}

func mapStrings[T any](items []T, f func(T) string) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = f(it)
	}
	return out
}

// countLabels merges equal strings: a string that occurs more than once gets a
// count and the suffix.
func countLabels(items []string, suffix string) []string {
	var order []string
	counts := map[string]int{}
	for _, it := range items {
		if counts[it] == 0 {
			order = append(order, it)
		}
		counts[it]++
	}
	out := make([]string, len(order))
	for i, it := range order {
		if counts[it] == 1 {
			out[i] = it
		} else {
			out[i] = it + " (" + strconv.Itoa(counts[it]) + " " + suffix + ")"
		}
	}
	return out
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
