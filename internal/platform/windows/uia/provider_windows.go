//go:build windows

package uia

import (
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// elem is the Go value behind the COM object of a node of the semantics tree, or of the fragment root of
// a window (its ID is FragmentRootID): IRawElementProviderSimple and IRawElementProviderFragment, the
// pattern interfaces for a node, IRawElementProviderFragmentRoot for the root. A node's object implements
// every pattern interface, and GetPatternProvider says which ones the node supports.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc AXPlatformNodeWin
type elem struct {
	b   *Bridge
	ref NodeRef
}

func (el *elem) isRoot() bool { return el.ref.ID == FragmentRootID }

var errGone = hresult(ErrElementNotAvailable)

// with runs fn on the node of the element under the core's lock. fn only reads the model.
func (el *elem) with(fn func(m *Model, n *Node) win32.HRESULT) win32.HRESULT {
	hr := errGone
	el.b.core.Read(func(m *Model) {
		if n := m.Node(el.ref); n != nil {
			hr = fn(m, n)
		}
	})
	return hr
}

// patternInterfaces maps a pattern to the interface that GetPatternProvider gives.
var patternInterfaces = map[PatternID]Interface{
	PatternInvoke: IfaceInvoke, PatternToggle: IfaceToggle, PatternValue: IfaceValue, PatternRangeValue: IfaceRangeValue,
	PatternScroll: IfaceScroll, PatternText: IfaceText, PatternExpandCollapse: IfaceExpandCollapse,
	PatternSelection: IfaceSelection, PatternSelectionItem: IfaceSelectionItem,
}

// hand gives Windows a new reference to the interface of the node, or null for no node.
func (el *elem) hand(out uintptr, ref NodeRef, ok bool, iface Interface) win32.HRESULT {
	if out == 0 {
		return win32.EPointer
	}
	*win32.View[uintptr](out) = 0
	if !ok {
		return hrOK
	}
	p, err := el.b.pointer(ref, iface)
	if err != nil {
		el.b.log.Debug("a provider could not be handed to UI Automation", "err", err)
		return win32.EFail
	}
	*win32.View[uintptr](out) = p
	return hrOK
}

// elementMethods are the methods of the interfaces of an element.
func elementMethods(e *Env) map[string]map[string]win32.MethodDef {
	return map[string]map[string]win32.MethodDef{
		IfaceSimple.Name:         defsOf(simpleMethods()),
		IfaceFragment.Name:       defsOf(fragmentMethods()),
		IfaceFragmentRoot.Name:   defsOf(rootMethods(e)),
		IfaceInvoke.Name:         defsOf(invokeMethods()),
		IfaceToggle.Name:         defsOf(toggleMethods()),
		IfaceValue.Name:          defsOf(valueMethods()),
		IfaceRangeValue.Name:     defsOf(rangeValueMethods(e)),
		IfaceScroll.Name:         defsOf(scrollMethods(e)),
		IfaceExpandCollapse.Name: defsOf(expandCollapseMethods()),
		IfaceSelection.Name:      defsOf(selectionMethods()),
		IfaceSelectionItem.Name:  defsOf(selectionItemMethods()),
		IfaceText.Name:           defsOf(textMethods()),
	}
}

func defsOf(list []win32.MethodDef) map[string]win32.MethodDef {
	m := make(map[string]win32.MethodDef, len(list))
	for _, d := range list {
		m[d.Name] = d
	}
	return m
}

// simpleMethods are the methods of IRawElementProviderSimple.
func simpleMethods() []win32.MethodDef {
	return []win32.MethodDef{
		// ProviderOptions: the provider is called on the UI thread, through COM's apartment (SPEC §25.11).
		elementMethod("get_ProviderOptions", 1, func(el *elem, a []uintptr) win32.HRESULT {
			return outInt32(a[0], ProviderServerSide|ProviderUseCOMThreading)
		}),
		elementMethod("GetPatternProvider", 2, func(el *elem, a []uintptr) win32.HRESULT {
			if a[1] == 0 {
				return win32.EPointer
			}
			*win32.View[uintptr](a[1]) = 0
			iface, known := patternInterfaces[PatternID(int32(a[0]))]
			if el.isRoot() || !known {
				return hrOK
			}
			var has bool
			if hr := el.with(func(m *Model, n *Node) win32.HRESULT {
				has = m.Patterns(n).Has(PatternID(int32(a[0])))
				return hrOK
			}); hr.Failed() {
				return hr
			}
			return el.hand(a[1], el.ref, has, iface)
		}),
		elementMethod("GetPropertyValue", 2, func(el *elem, a []uintptr) win32.HRESULT {
			id := PropertyID(int32(a[0]))
			if el.isRoot() {
				// UI Automation has a provider for the window that gives most properties; leave them to it
				// so that no blank value overrides them, and say which of the fragment roots are controls.
				switch id {
				case PropIsControlElement, PropIsContentElement:
					return putVariant(a[1], Bool(true))
				}
				return putVariant(a[1], Value{})
			}
			var v Value
			if hr := el.with(func(m *Model, n *Node) win32.HRESULT {
				v = m.Property(n, id)
				return hrOK
			}); hr.Failed() {
				return hr
			}
			return putVariant(a[1], v)
		}),
		elementMethod("get_HostRawElementProvider", 1, func(el *elem, a []uintptr) win32.HRESULT {
			if a[0] == 0 {
				return win32.EPointer
			}
			*win32.View[uintptr](a[0]) = 0
			if !el.isRoot() {
				return hrOK
			}
			host, err := win32.UiaHostProviderFromHwnd(el.b.core.HWND(el.ref.View))
			if err != nil {
				el.b.log.Debug("the host provider of the window is not available", "err", err)
				return win32.EFail
			}
			*win32.View[uintptr](a[0]) = host.Ptr()
			return hrOK
		}),
	}
}

// fragmentMethods are the methods of IRawElementProviderFragment.
func fragmentMethods() []win32.MethodDef {
	return []win32.MethodDef{
		elementMethod("Navigate", 2, func(el *elem, a []uintptr) win32.HRESULT {
			var target NodeRef
			var found bool
			dir := int(int32(a[0]))
			read := func(m *Model) win32.HRESULT {
				if el.isRoot() {
					root := m.Root(el.ref.View)
					if root != nil && (dir == NavigateFirstChild || dir == NavigateLastChild) {
						target, found = root.Ref(), true
					}
					return hrOK
				}
				n := m.Node(el.ref)
				if n == nil {
					return errGone
				}
				var t *Node
				switch dir {
				case NavigateParent:
					if n.ID == RootID {
						target, found = NodeRef{n.View, FragmentRootID}, true
						return hrOK
					}
					t = m.ParentOf(n)
				case NavigateNextSibling:
					t = m.Sibling(n, 1)
				case NavigatePreviousSibling:
					t = m.Sibling(n, -1)
				case NavigateFirstChild, NavigateLastChild:
					if kids := m.Children(n); len(kids) > 0 {
						k := kids[0]
						if dir == NavigateLastChild {
							k = kids[len(kids)-1]
						}
						t = m.Node(k)
					}
				}
				if t != nil {
					target, found = t.Ref(), true
				}
				return hrOK
			}
			var hr win32.HRESULT
			el.b.core.Read(func(m *Model) { hr = read(m) })
			if hr.Failed() {
				return hr
			}
			return el.hand(a[1], target, found, IfaceFragment)
		}),
		elementMethod("GetRuntimeId", 1, func(el *elem, a []uintptr) win32.HRESULT {
			if a[0] == 0 {
				return win32.EPointer
			}
			*win32.View[uintptr](a[0]) = 0
			id := RootRuntimeID()
			if !el.isRoot() {
				id = RuntimeID(el.ref.ID)
			}
			sa, err := int32Array(id)
			if err != nil {
				return win32.EOutOfMemory
			}
			*win32.View[uintptr](a[0]) = sa
			return hrOK
		}),
		elementMethod("get_BoundingRectangle", 1, func(el *elem, a []uintptr) win32.HRESULT {
			if a[0] == 0 {
				return win32.EPointer
			}
			ox, oy, w, h, ok := el.b.core.ClientRect(el.ref.View)
			if !ok {
				return errGone
			}
			rect := win32.UiaRect{Left: ox, Top: oy, Width: w, Height: h}
			if !el.isRoot() {
				if hr := el.with(func(m *Model, n *Node) win32.HRESULT {
					r := m.ScreenRect(n, ox, oy)
					rect = win32.UiaRect{Left: r.X, Top: r.Y, Width: r.Width, Height: r.Height}
					return hrOK
				}); hr.Failed() {
					return hr
				}
			}
			*win32.View[win32.UiaRect](a[0]) = rect
			return hrOK
		}),
		elementMethod("GetEmbeddedFragmentRoots", 1, func(el *elem, a []uintptr) win32.HRESULT {
			return outPtr(a[0], 0)
		}),
		elementMethod("SetFocus", 0, func(el *elem, a []uintptr) win32.HRESULT {
			if el.isRoot() {
				return hrOK
			}
			return statusHR(el.b.core.Request(func(m *Model) ([]ui.SemanticsActionEvent, Status) { return m.Focus(el.ref) }))
		}),
		elementMethod("get_FragmentRoot", 1, func(el *elem, a []uintptr) win32.HRESULT {
			return el.hand(a[0], NodeRef{el.ref.View, FragmentRootID}, true, IfaceFragmentRoot)
		}),
	}
}

// rootMethods are the methods of IRawElementProviderFragmentRoot. ElementProviderFromPoint takes two
// doubles: its vtable entry is a thunk (thunk_windows.go), and fromPoint is what the thunk's callback runs.
func rootMethods(e *Env) []win32.MethodDef {
	point, _, _ := thunkEntries()
	return []win32.MethodDef{
		{Name: "ElementProviderFromPoint", Args: 3, Raw: point},
		elementMethod("GetFocus", 1, func(el *elem, a []uintptr) win32.HRESULT {
			var target NodeRef
			var found bool
			el.b.core.Read(func(m *Model) {
				if n := m.Focused(el.ref.View); n != nil {
					target, found = n.Ref(), true
				}
			})
			return el.hand(a[0], target, found, IfaceFragment)
		}),
	}
}

// fromPoint is ElementProviderFromPoint for a point on the screen in physical pixels: the node of the
// window's tree under it, or null.
func (el *elem) fromPoint(x, y float64, out uintptr) win32.HRESULT {
	if out == 0 {
		return win32.EPointer
	}
	*win32.View[uintptr](out) = 0
	if !el.isRoot() || !finite(x) || !finite(y) {
		return hrOK
	}
	ox, oy, _, _, ok := el.b.core.ClientRect(el.ref.View)
	if !ok {
		return hrOK
	}
	var target NodeRef
	var found bool
	el.b.core.Read(func(m *Model) {
		if root := m.Root(el.ref.View); root != nil {
			if hit := m.HitTest(root, x-ox, y-oy); hit != nil {
				target, found = hit.Ref(), true
			}
		}
	})
	return el.hand(out, target, found, IfaceFragment)
}

// request runs a request of a pattern on the element's node.
func (el *elem) request(fn func(m *Model, ref NodeRef) ([]ui.SemanticsActionEvent, Status)) win32.HRESULT {
	return statusHR(el.b.core.Request(func(m *Model) ([]ui.SemanticsActionEvent, Status) { return fn(m, el.ref) }))
}

func invokeMethods() []win32.MethodDef {
	return []win32.MethodDef{
		elementMethod("Invoke", 0, func(el *elem, a []uintptr) win32.HRESULT {
			return el.request(func(m *Model, ref NodeRef) ([]ui.SemanticsActionEvent, Status) { return m.Invoke(ref) })
		}),
	}
}

func toggleMethods() []win32.MethodDef {
	return []win32.MethodDef{
		elementMethod("Toggle", 0, func(el *elem, a []uintptr) win32.HRESULT {
			return el.request(func(m *Model, ref NodeRef) ([]ui.SemanticsActionEvent, Status) { return m.Toggle(ref) })
		}),
		elementMethod("get_ToggleState", 1, func(el *elem, a []uintptr) win32.HRESULT {
			return el.property(a[0], PropToggleToggleState)
		}),
	}
}

// property writes the value of a property that a pattern's getter reads into the out parameter, which is
// an int32 (an enumeration), a BOOL, a double or a BSTR as the value says.
func (el *elem) property(out uintptr, id PropertyID) win32.HRESULT {
	var v Value
	if hr := el.with(func(m *Model, n *Node) win32.HRESULT {
		v = m.Property(n, id)
		return hrOK
	}); hr.Failed() {
		return hr
	}
	switch v.Kind {
	case ValueBool:
		return outBool(out, v.B)
	case ValueInt:
		return outInt32(out, v.I)
	case ValueFloat:
		return outFloat(out, v.F)
	case ValueString:
		return outBSTR(out, v.S)
	}
	return hresult(ErrNotSupported)
}

func valueMethods() []win32.MethodDef {
	return []win32.MethodDef{
		elementMethod("SetValue", 1, func(el *elem, a []uintptr) win32.HRESULT {
			if a[0] == 0 {
				return win32.EPointer
			}
			text := win32.UTF16PtrToString((*uint16)(at(a[0])))
			return el.request(func(m *Model, ref NodeRef) ([]ui.SemanticsActionEvent, Status) { return m.SetValue(ref, text) })
		}),
		elementMethod("get_Value", 1, func(el *elem, a []uintptr) win32.HRESULT {
			return el.property(a[0], PropValueValue)
		}),
		elementMethod("get_IsReadOnly", 1, func(el *elem, a []uintptr) win32.HRESULT {
			return el.property(a[0], PropValueIsReadOnly)
		}),
	}
}

// rangeValueMethods are the methods of IRangeValueProvider. SetValue takes a double: its entry is a thunk.
func rangeValueMethods(e *Env) []win32.MethodDef {
	_, _, rng := thunkEntries()
	prop := func(name string, id PropertyID) win32.MethodDef {
		return elementMethod(name, 1, func(el *elem, a []uintptr) win32.HRESULT { return el.property(a[0], id) })
	}
	return []win32.MethodDef{
		{Name: "SetValue", Args: 1, Raw: rng},
		prop("get_Value", PropRangeValueValue),
		prop("get_IsReadOnly", PropRangeValueIsReadOnly),
		prop("get_Maximum", PropRangeValueMaximum),
		prop("get_Minimum", PropRangeValueMinimum),
		prop("get_LargeChange", PropRangeValueLargeChange),
		prop("get_SmallChange", PropRangeValueSmallChange),
	}
}

// setRangeValue is IRangeValueProvider::SetValue.
func (el *elem) setRangeValue(v float64) win32.HRESULT {
	if !finite(v) {
		return win32.EInvalidArg
	}
	return el.request(func(m *Model, ref NodeRef) ([]ui.SemanticsActionEvent, Status) { return m.SetRangeValue(ref, v) })
}

// scrollMethods are the methods of IScrollProvider. SetScrollPercent takes two doubles: its entry is a thunk.
func scrollMethods(e *Env) []win32.MethodDef {
	_, scroll, _ := thunkEntries()
	prop := func(name string, id PropertyID) win32.MethodDef {
		return elementMethod(name, 1, func(el *elem, a []uintptr) win32.HRESULT { return el.property(a[0], id) })
	}
	return []win32.MethodDef{
		elementMethod("Scroll", 2, func(el *elem, a []uintptr) win32.HRESULT {
			h, v := int(int32(a[0])), int(int32(a[1]))
			return el.request(func(m *Model, ref NodeRef) ([]ui.SemanticsActionEvent, Status) { return m.Scroll(ref, h, v) })
		}),
		{Name: "SetScrollPercent", Args: 2, Raw: scroll},
		prop("get_HorizontalScrollPercent", PropScrollHorizontalScrollPercent),
		prop("get_VerticalScrollPercent", PropScrollVerticalScrollPercent),
		prop("get_HorizontalViewSize", PropScrollHorizontalViewSize),
		prop("get_VerticalViewSize", PropScrollVerticalViewSize),
		prop("get_HorizontallyScrollable", PropScrollHorizontallyScrollable),
		prop("get_VerticallyScrollable", PropScrollVerticallyScrollable),
	}
}

// setScrollPercent is IScrollProvider::SetScrollPercent. UIA_ScrollPatternNoScroll (-1) leaves an axis.
func (el *elem) setScrollPercent(h, v float64) win32.HRESULT {
	if !finite(h) || !finite(v) {
		return win32.EInvalidArg
	}
	return el.request(func(m *Model, ref NodeRef) ([]ui.SemanticsActionEvent, Status) { return m.SetScrollPercent(ref, h, v) })
}

func expandCollapseMethods() []win32.MethodDef {
	return []win32.MethodDef{
		elementMethod("Expand", 0, func(el *elem, a []uintptr) win32.HRESULT {
			return el.request(func(m *Model, ref NodeRef) ([]ui.SemanticsActionEvent, Status) { return m.Expand(ref) })
		}),
		elementMethod("Collapse", 0, func(el *elem, a []uintptr) win32.HRESULT {
			return el.request(func(m *Model, ref NodeRef) ([]ui.SemanticsActionEvent, Status) { return m.Collapse(ref) })
		}),
		elementMethod("get_ExpandCollapseState", 1, func(el *elem, a []uintptr) win32.HRESULT {
			return el.property(a[0], PropExpandCollapseExpandCollapseState)
		}),
	}
}

func selectionMethods() []win32.MethodDef {
	return []win32.MethodDef{
		elementMethod("GetSelection", 1, func(el *elem, a []uintptr) win32.HRESULT {
			if a[0] == 0 {
				return win32.EPointer
			}
			*win32.View[uintptr](a[0]) = 0
			var refs []NodeRef
			if hr := el.with(func(m *Model, n *Node) win32.HRESULT {
				refs = m.SelectedChildren(n)
				return hrOK
			}); hr.Failed() {
				return hr
			}
			ptrs := make([]uintptr, 0, len(refs))
			for _, r := range refs {
				p, err := el.b.pointer(r, IfaceSimple)
				if err != nil {
					return win32.EFail
				}
				ptrs = append(ptrs, p)
			}
			sa, err := unknownArray(ptrs)
			for _, p := range ptrs { // the array took its own references
				win32.ObjectFromPtr(p).Release()
			}
			if err != nil {
				return win32.EOutOfMemory
			}
			*win32.View[uintptr](a[0]) = sa
			return hrOK
		}),
		elementMethod("get_CanSelectMultiple", 1, func(el *elem, a []uintptr) win32.HRESULT {
			return el.property(a[0], PropSelectionCanSelectMultiple)
		}),
		elementMethod("get_IsSelectionRequired", 1, func(el *elem, a []uintptr) win32.HRESULT {
			return el.property(a[0], PropSelectionIsSelectionRequired)
		}),
	}
}

func selectionItemMethods() []win32.MethodDef {
	return []win32.MethodDef{
		elementMethod("Select", 0, func(el *elem, a []uintptr) win32.HRESULT {
			return el.request(func(m *Model, ref NodeRef) ([]ui.SemanticsActionEvent, Status) { return m.Select(ref) })
		}),
		elementMethod("AddToSelection", 0, func(el *elem, a []uintptr) win32.HRESULT {
			return el.request(func(m *Model, ref NodeRef) ([]ui.SemanticsActionEvent, Status) { return m.AddToSelection(ref) })
		}),
		elementMethod("RemoveFromSelection", 0, func(el *elem, a []uintptr) win32.HRESULT {
			return el.request(func(m *Model, ref NodeRef) ([]ui.SemanticsActionEvent, Status) { return m.RemoveFromSelection(ref) })
		}),
		elementMethod("get_IsSelected", 1, func(el *elem, a []uintptr) win32.HRESULT {
			return el.property(a[0], PropSelectionItemIsSelected)
		}),
		elementMethod("get_SelectionContainer", 1, func(el *elem, a []uintptr) win32.HRESULT {
			var target NodeRef
			var found bool
			if hr := el.with(func(m *Model, n *Node) win32.HRESULT {
				if c := m.SelectionContainer(n); c != nil {
					target, found = c.Ref(), true
				}
				return hrOK
			}); hr.Failed() {
				return hr
			}
			return el.hand(a[0], target, found, IfaceSimple)
		}),
	}
}
