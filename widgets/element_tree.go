package widgets

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
)

// UpdateChild updates a child element to a new widget and slot, and returns
// the element that now stands for the child: the same element when the new
// widget CanUpdate it, a new one otherwise, or nil when newWidget is nil.
//
//   - newWidget nil, child nil: nothing happens, the result is nil.
//   - newWidget nil, child not nil: the child is removed (deactivated).
//   - newWidget not nil, child nil: a new element is inflated.
//   - newWidget not nil, child not nil: the child is updated when it can be and
//     replaced with a new element when it cannot.
//
// Flutter: widgets/framework.dart Element.updateChild
func (b *ElementBase) UpdateChild(child Element, newWidget Widget, newSlot any) Element {
	if newWidget == nil {
		if child != nil {
			b.self.DeactivateChild(child)
		}
		return nil
	}
	var newChild Element
	switch {
	case child == nil:
		newChild = b.self.InflateWidget(newWidget, newSlot)
	case child.AsElement().widget == newWidget:
		if child.Slot() != newSlot {
			b.UpdateSlotForChild(child, newSlot)
		}
		newChild = child
	case CanUpdate(child.AsElement().widget, newWidget):
		if child.Slot() != newSlot {
			b.UpdateSlotForChild(child, newSlot)
		}
		child.Update(newWidget)
		if foundation.DebugMode {
			child.AsElement().owner.debugElementWasRebuilt(child)
		}
		newChild = child
	default:
		b.self.DeactivateChild(child)
		newChild = b.self.InflateWidget(newWidget, newSlot)
	}
	if foundation.DebugMode {
		if child != nil {
			b.debugRemoveGlobalKeyReservation(child)
		}
		if key, ok := newWidget.Key().(AnyGlobalKey); ok {
			b.owner.debugReserveGlobalKeyFor(b.self, newChild, key)
		}
	}
	return newChild
}

// UpdateChildrenOpts are the optional parameters of ElementBase.UpdateChildren.
//
// Flutter: widgets/framework.dart Element.updateChildren
type UpdateChildrenOpts struct {
	// ForgottenChildren are old children that have moved to another parent, so
	// they must not be updated or deactivated here.
	ForgottenChildren map[Element]struct{}
	// Slots are the slots of the new children, one for each new widget. When it
	// is nil the slot of a child is an IndexedSlot of its index and previous
	// sibling.
	Slots []any
}

// UpdateChildren updates the list of old child elements to the list of new
// widgets and returns the new list of elements. It syncs the lists from the
// top, then from the bottom, then matches keyed widgets in the middle, and
// removes the old children that are left over. A widget without a key might be
// synced with an old child but there is no guarantee. The caller is expected to
// update its render object accordingly.
//
// Flutter: widgets/framework.dart Element.updateChildren
func (b *ElementBase) UpdateChildren(oldChildren []Element, newWidgets []Widget, opts ...UpdateChildrenOpts) []Element {
	o := firstOpts(opts)
	forgotten := o.ForgottenChildren
	slots := o.Slots

	replaceWithNilIfForgotten := func(child Element) Element {
		if _, ok := forgotten[child]; ok {
			return nil
		}
		return child
	}
	slotFor := func(newChildIndex int, previousChild Element) any {
		if slots != nil {
			return slots[newChildIndex]
		}
		return IndexedSlot{Index: newChildIndex, Value: previousChild}
	}

	newChildrenTop := 0
	oldChildrenTop := 0
	newChildrenBottom := len(newWidgets) - 1
	oldChildrenBottom := len(oldChildren) - 1

	newChildren := make([]Element, len(newWidgets))
	var previousChild Element

	// Update the top of the list.
	for oldChildrenTop <= oldChildrenBottom && newChildrenTop <= newChildrenBottom {
		oldChild := replaceWithNilIfForgotten(oldChildren[oldChildrenTop])
		newWidget := newWidgets[newChildrenTop]
		if oldChild == nil || !CanUpdate(oldChild.AsElement().widget, newWidget) {
			break
		}
		newChild := b.self.UpdateChild(oldChild, newWidget, slotFor(newChildrenTop, previousChild))
		newChildren[newChildrenTop] = newChild
		previousChild = newChild
		newChildrenTop++
		oldChildrenTop++
	}

	// Scan the bottom of the list.
	for oldChildrenTop <= oldChildrenBottom && newChildrenTop <= newChildrenBottom {
		oldChild := replaceWithNilIfForgotten(oldChildren[oldChildrenBottom])
		newWidget := newWidgets[newChildrenBottom]
		if oldChild == nil || !CanUpdate(oldChild.AsElement().widget, newWidget) {
			break
		}
		oldChildrenBottom--
		newChildrenBottom--
	}

	// Scan the old children in the middle of the list.
	haveOldChildren := oldChildrenTop <= oldChildrenBottom
	var oldKeyedChildren map[foundation.Key]Element
	if haveOldChildren {
		oldKeyedChildren = map[foundation.Key]Element{}
		for oldChildrenTop <= oldChildrenBottom {
			oldChild := replaceWithNilIfForgotten(oldChildren[oldChildrenTop])
			if oldChild != nil {
				if key := oldChild.AsElement().widget.Key(); key != nil {
					oldKeyedChildren[key] = oldChild
				} else {
					b.self.DeactivateChild(oldChild)
				}
			}
			oldChildrenTop++
		}
	}

	// Update the middle of the list.
	for newChildrenTop <= newChildrenBottom {
		var oldChild Element
		newWidget := newWidgets[newChildrenTop]
		if haveOldChildren {
			if key := newWidget.Key(); key != nil {
				oldChild = oldKeyedChildren[key]
				if oldChild != nil {
					if CanUpdate(oldChild.AsElement().widget, newWidget) {
						// We found a match; remove it so we don't unsync it later.
						delete(oldKeyedChildren, key)
					} else {
						// Not a match, let's pretend we didn't see it for now.
						oldChild = nil
					}
				}
			}
		}
		newChild := b.self.UpdateChild(oldChild, newWidget, slotFor(newChildrenTop, previousChild))
		newChildren[newChildrenTop] = newChild
		previousChild = newChild
		newChildrenTop++
	}

	// We've scanned the whole list.
	newChildrenBottom = len(newWidgets) - 1
	oldChildrenBottom = len(oldChildren) - 1

	// Update the bottom of the list.
	for oldChildrenTop <= oldChildrenBottom && newChildrenTop <= newChildrenBottom {
		oldChild := oldChildren[oldChildrenTop]
		newWidget := newWidgets[newChildrenTop]
		newChild := b.self.UpdateChild(oldChild, newWidget, slotFor(newChildrenTop, previousChild))
		newChildren[newChildrenTop] = newChild
		previousChild = newChild
		newChildrenTop++
		oldChildrenTop++
	}

	// Clean up any of the remaining middle nodes from the old list, in the order
	// of the old list so that the result does not depend on the map.
	if haveOldChildren && len(oldKeyedChildren) > 0 {
		for _, oldChild := range oldChildren {
			key := oldChild.AsElement().widget.Key()
			if key == nil || oldKeyedChildren[key] != oldChild {
				continue
			}
			if _, isForgotten := forgotten[oldChild]; !isForgotten {
				b.self.DeactivateChild(oldChild)
			}
		}
	}
	return newChildren
}

// Mount adds the element to the tree as a child of parent, in the given slot.
// A root element has no parent and gets its owner from AssignOwner.
//
// Flutter: widgets/framework.dart Element.mount
func (b *ElementBase) Mount(parent Element, newSlot any) {
	b.parent = parent
	b.slot = newSlot
	b.lifecycle = lifecycleActive
	b.depth = 1
	if parent != nil {
		pb := parent.AsElement()
		b.depth += pb.depth
		// Only assign ownership if the parent is not nil. If it is nil (the root),
		// the owner should have been assigned by AssignOwner.
		b.owner = pb.owner
		b.parentScope = parent.BuildScope()
	}
	if key, ok := b.widget.Key().(AnyGlobalKey); ok {
		b.owner.registerGlobalKey(key, b.self)
	}
	b.self.updateInheritance()
}

func (b *ElementBase) debugRemoveGlobalKeyReservation(child Element) {
	b.owner.debugRemoveGlobalKeyReservationFor(b.self, child)
}

// Update changes the configuration of the element to a new widget that
// CanUpdate the old one. Subclasses override it, update what depends on the
// widget and call this one.
//
// Flutter: widgets/framework.dart Element.update
func (b *ElementBase) Update(newWidget Widget) {
	// This element was told to update, so the global key reservations of its
	// forgotten children can go. They could not go earlier: the forgotten
	// children are still a duplicate of their key if this element never updates.
	if foundation.DebugMode {
		for _, child := range b.forgottenWithGlobalKey.items {
			b.debugRemoveGlobalKeyReservation(child)
		}
		b.forgottenWithGlobalKey.clear()
	}
	b.widget = newWidget
}

// UpdateSlotForChild changes the slot of the child and of the descendants that
// share its render object.
//
// Flutter: widgets/framework.dart Element.updateSlotForChild
func (b *ElementBase) UpdateSlotForChild(child Element, newSlot any) {
	var visit func(Element)
	visit = func(e Element) {
		e.UpdateSlot(newSlot)
		if d := e.RenderObjectAttachingChild(); d != nil {
			visit(d)
		}
	}
	visit(child)
}

// UpdateSlot changes the slot.
//
// Flutter: widgets/framework.dart Element.updateSlot
func (b *ElementBase) UpdateSlot(newSlot any) { b.slot = newSlot }

func (b *ElementBase) updateDepth(parentDepth int) {
	expected := parentDepth + 1
	if b.depth < expected {
		b.depth = expected
		b.self.VisitChildren(func(child Element) { child.AsElement().updateDepth(expected) })
	}
}

func (b *ElementBase) updateBuildScopeRecursively() {
	var parentScope *BuildScope
	if b.parent != nil {
		parentScope = b.parent.BuildScope()
	}
	if b.self.BuildScope() == parentScope {
		return
	}
	// Unset the flag so this element can be added to the dirty list of the new
	// build scope if it is dirty.
	b.inDirtyList = false
	b.parentScope = parentScope
	b.self.VisitChildren(func(child Element) { child.AsElement().updateBuildScopeRecursively() })
}

// InflateWidget makes an element for the widget, mounts it as a child of this
// element and returns it. If the mount fails it cleans up on a best-effort
// basis and the panic goes on.
//
// Flutter: widgets/framework.dart Element.inflateWidget
func (b *ElementBase) InflateWidget(newWidget Widget, newSlot any) Element {
	var inactiveChild Element
	if key, ok := newWidget.Key().(AnyGlobalKey); ok {
		inactiveChild = b.retakeInactiveElement(key, newWidget)
	}
	newChild := inactiveChild
	if newChild == nil {
		newChild = CreateElement(newWidget)
	}
	completed := false
	defer func() {
		if !completed {
			// Clean up on a best-effort basis when activation or mount fails.
			b.deactivateFailedChildSilently(newChild)
		}
	}()
	if inactiveChild != nil {
		inactiveChild.AsElement().activateWithParent(b.self, newSlot)
		updated := b.self.UpdateChild(inactiveChild, newWidget, newSlot)
		completed = true
		return updated
	}
	newChild.Mount(b.self, newSlot)
	completed = true
	return newChild
}

// retakeInactiveElement takes the element that has the global key from where it
// is: from its parent, which is told to forget it, or from the inactive list.
// It returns nil when there is no such element or its widget cannot be updated
// to the new one.
//
// Flutter: widgets/framework.dart Element._retakeInactiveElement
func (b *ElementBase) retakeInactiveElement(key AnyGlobalKey, newWidget Widget) Element {
	// The element may be forward-looking inactive: if it is taken from a parent
	// that still has it as a child, that parent will soon not have it. The only
	// way that is wrong is a duplicated key, which is tracked below.
	element := key.currentElementIn(b.owner)
	if element == nil || !CanUpdate(element.AsElement().widget, newWidget) {
		return nil
	}
	if parent := element.AsElement().parent; parent != nil {
		if foundation.DebugMode {
			if parent == b.self {
				panic(errors.New("GlobalKeyUsedTwiceInChildList",
					"a GlobalKey was used multiple times inside one widget's child list; a GlobalKey can only be specified on one widget at a time in the widget tree",
					"key", key.String(), "parent", parent.String(), "first child", element.String(), "second widget", foundation.RuntimeTypeName(newWidget)))
			}
			b.owner.debugTrackElementThatWillNeedToBeRebuiltDueToGlobalKeyShenanigans(parent, key)
		}
		parent.ForgetChild(element)
		parent.DeactivateChild(element)
	}
	b.owner.inactive.remove(element)
	return element
}

// DeactivateChild moves the child to the inactive list of the owner: it is
// detached from this element and from the render tree, and deactivated.
//
// Flutter: widgets/framework.dart Element.deactivateChild
func (b *ElementBase) DeactivateChild(child Element) {
	cb := child.AsElement()
	cb.parent = nil
	child.DetachRenderObject()
	b.owner.inactive.add(child)
}

func (b *ElementBase) deactivateFailedChildSilently(child Element) {
	// The subtree has already panicked with a different error and the framework
	// is cleaning up on a best-effort basis, so what goes wrong here is dropped.
	_ = tryCall(func() {
		child.AsElement().parent = nil
		child.DetachRenderObject()
		deactivateFailedSubtreeRecursively(child)
	})
}

// deactivateFailedSubtreeRecursively deactivates the subtree rooted at element,
// whatever panics, and marks it failed.
func deactivateFailedSubtreeRecursively(element Element) {
	if tryCall(element.Deactivate) {
		element.AsElement().ensureDeactivated()
	}
	element.AsElement().lifecycle = lifecycleFailed
	_ = tryCall(func() { element.VisitChildren(deactivateFailedSubtreeRecursively) })
}

// tryCall runs fn and reports whether it panicked. The panic is dropped; the
// caller is cleaning up after another failure.
func tryCall(fn func()) (panicked bool) {
	defer func() {
		if recover() != nil {
			panicked = true
		}
	}()
	fn()
	return false
}

// ForgetChild is called on the old parent when the child, which has a global
// key, is given a new parent. The reservation of the key cannot be released
// here: the forgotten child is not removed from the tree until this element
// updates, and if it never does, the child is still a duplicate of the key.
//
// Flutter: widgets/framework.dart Element.forgetChild
func (b *ElementBase) ForgetChild(child Element) {
	if foundation.DebugMode {
		if _, ok := child.AsElement().widget.Key().(AnyGlobalKey); ok {
			b.forgottenWithGlobalKey.add(child)
		}
	}
}

func (b *ElementBase) activateWithParent(parent Element, newSlot any) {
	b.parent = parent
	b.owner = parent.AsElement().owner
	b.updateDepth(b.parent.AsElement().depth)
	b.updateBuildScopeRecursively()
	activateRecursively(b.self)
	b.self.AttachRenderObject(newSlot)
}

func activateRecursively(element Element) {
	element.Activate()
	element.VisitChildren(activateRecursively)
}

// Activate moves the element back into the tree: it clears the dependencies
// that Deactivate unregistered, and rebuilds if it was dirty or had
// dependencies.
//
// Flutter: widgets/framework.dart Element.activate
func (b *ElementBase) Activate() {
	hadDependencies := len(b.dependencies) > 0 || b.hadUnsatisfiedDependencies
	b.lifecycle = lifecycleActive
	// The dependencies were unregistered in Deactivate but never cleared; the
	// element is reused now.
	clear(b.dependencies)
	b.hadUnsatisfiedDependencies = false
	b.self.updateInheritance()
	if b.dirty {
		b.owner.ScheduleBuildFor(b.self)
	}
	if hadDependencies {
		b.self.DidChangeDependencies()
	}
}

// Deactivate takes the element out of the tree: it unregisters from the
// inherited elements it depends on.
//
// Flutter: widgets/framework.dart Element.deactivate
func (b *ElementBase) Deactivate() { b.ensureDeactivated() }

func (b *ElementBase) ensureDeactivated() {
	for dependency := range b.dependencies {
		if d, ok := dependency.(interface{ removeDependent(Element) }); ok {
			d.removeDependent(b.self)
		}
	}
	// The list is not cleared, even though it no longer says what the element
	// is registered with: Activate uses it to decide whether to rebuild.
	b.inheritedElements = nil
	b.lifecycle = lifecycleInactive
}

// DebugDeactivated is called after the element and its subtree have been
// deactivated.
//
// Flutter: widgets/framework.dart Element.debugDeactivated
func (b *ElementBase) DebugDeactivated() {}

// Unmount removes the element from the tree for good. It releases the widget,
// so that a retained element does not keep it alive.
//
// Flutter: widgets/framework.dart Element.unmount
func (b *ElementBase) Unmount() {
	if b.widget != nil {
		if key, ok := b.widget.Key().(AnyGlobalKey); ok {
			b.owner.unregisterGlobalKey(key, b.self)
		}
	}
	b.widget = nil
	b.dependencies = nil
	b.lifecycle = lifecycleDefunct
}

// updateInheritance copies the inherited elements of the parent. An
// InheritedElement adds itself.
//
// Flutter: widgets/framework.dart Element._updateInheritance
func (b *ElementBase) updateInheritance() {
	if b.parent != nil {
		b.inheritedElements = b.parent.AsElement().inheritedElements
	} else {
		b.inheritedElements = nil
	}
}
