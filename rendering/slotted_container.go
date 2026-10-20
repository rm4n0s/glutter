package rendering

import (
	"fmt"

	"github.com/rm4n0s/glutter/foundation"
)

// SlottedContainer holds children that each have a name, a slot (an enum value,
// or any comparable value), for a render object: embed it and call
// InitSlottedContainer in the constructor. It adopts and drops the children, and
// takes part in attach, detach, redepth, visiting and diagnostics.
//
// Flutter: widgets/slotted_render_object_widget.dart SlottedContainerRenderObjectMixin
type SlottedContainer struct {
	host     RenderObject
	slots    []any
	children map[any]RenderObject
	// DebugNameForSlot gives the name of a slot in the diagnostics; the default
	// is the String method of the slot, or its default format.
	DebugNameForSlot func(slot any) string
}

// InitSlottedContainer registers the mixin with its render object.
//
// Flutter: no counterpart (the self pattern, SPEC §12.5)
func InitSlottedContainer(m *SlottedContainer, host RenderObject) {
	m.host = host
	m.children = map[any]RenderObject{}
	host.AsRenderObject().setChildModel(m)
}

// SlotChild is the child in the slot, or nil.
//
// Flutter: widgets/slotted_render_object_widget.dart SlottedContainerRenderObjectMixin.childForSlot
func (m *SlottedContainer) SlotChild(slot any) RenderObject {
	c, ok := m.children[slot]
	if !ok {
		return nil
	}
	return c
}

// SlotChildren are the children, in the order they were set.
//
// Flutter: widgets/slotted_render_object_widget.dart SlottedContainerRenderObjectMixin.children
func (m *SlottedContainer) SlotChildren() []RenderObject {
	out := make([]RenderObject, 0, len(m.slots))
	for _, s := range m.slots {
		out = append(out, m.children[s])
	}
	return out
}

// SetSlotChild puts the child (nil for none) in the slot: the child there is
// dropped, the new one adopted.
//
// Flutter: widgets/slotted_render_object_widget.dart SlottedContainerRenderObjectMixin._setChild
func (m *SlottedContainer) SetSlotChild(child RenderObject, slot any) {
	if old, ok := m.children[slot]; ok {
		m.host.AsRenderObject().DropChild(old)
		delete(m.children, slot)
		for i, s := range m.slots {
			if s == slot {
				m.slots = append(m.slots[:i], m.slots[i+1:]...)
				break
			}
		}
	}
	if !isNil(child) {
		m.children[slot] = child
		m.slots = append(m.slots, slot)
		m.host.AsRenderObject().AdoptChild(child)
	}
}

// MoveSlotChild moves the child from the old slot to the new one.
//
// Flutter: widgets/slotted_render_object_widget.dart SlottedContainerRenderObjectMixin._moveChild
func (m *SlottedContainer) MoveSlotChild(child RenderObject, slot, oldSlot any) {
	if old, ok := m.children[oldSlot]; ok && old == child {
		m.SetSlotChild(nil, oldSlot)
	}
	m.SetSlotChild(child, slot)
}

func (m *SlottedContainer) attachChildren(owner *PipelineOwner) {
	for _, c := range m.SlotChildren() {
		c.Attach(owner)
	}
}

func (m *SlottedContainer) detachChildren() {
	for _, c := range m.SlotChildren() {
		c.Detach()
	}
}

func (m *SlottedContainer) redepthChildren() {
	for _, c := range m.SlotChildren() {
		m.host.AsRenderObject().RedepthChild(c)
	}
}

func (m *SlottedContainer) visitChildren(visitor RenderObjectVisitor) {
	for _, c := range m.SlotChildren() {
		visitor(c)
	}
}

func (m *SlottedContainer) describeChildren() []foundation.DiagnosticsNode {
	var out []foundation.DiagnosticsNode
	for _, s := range m.slots {
		name := ""
		if m.DebugNameForSlot != nil {
			name = m.DebugNameForSlot(s)
		} else if st, ok := s.(fmt.Stringer); ok {
			name = st.String()
		} else {
			name = fmt.Sprint(s)
		}
		out = append(out, foundation.ToDiagnosticsNode(m.children[s], foundation.DiagnosticsNodeOpts{Name: foundation.Some(name)}))
	}
	return out
}
