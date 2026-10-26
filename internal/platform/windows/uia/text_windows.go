//go:build windows

package uia

import (
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// rangeObj is the Go value behind a COM text range: a range of the text of the field of ref, in UTF-16
// units. It is a snapshot of the offsets; the text is read from the model each time, and the range is
// cut to it.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_textrangeprovider_win.cc AXPlatformNodeTextRangeProviderWin
type rangeObj struct {
	b   *Bridge
	ref NodeRef
	r   TextRange
}

// newRange makes the COM object of a text range and returns a new reference to its interface.
func (b *Bridge) newRange(ref NodeRef, r TextRange) (uintptr, error) {
	o, err := b.env.host.NewObject(&rangeObj{b: b, ref: ref, r: r}, b.env.rangeTypes...)
	if err != nil {
		return 0, err
	}
	return o.Ptr(), nil // the object's one reference goes to the caller
}

// text reads the text of the field under the lock; ok is false when the field is gone.
func (r *rangeObj) text() (t Text, ok bool) {
	r.b.core.Read(func(m *Model) {
		if n := m.Node(r.ref); n != nil {
			t, ok = NewText(textOf(n)), true
		}
	})
	return t, ok
}

// outRange gives Windows a new text range.
func (b *Bridge) outRange(out uintptr, ref NodeRef, r TextRange) win32.HRESULT {
	if out == 0 {
		return win32.EPointer
	}
	*win32.View[uintptr](out) = 0
	p, err := b.newRange(ref, r)
	if err != nil {
		return win32.EOutOfMemory
	}
	*win32.View[uintptr](out) = p
	return hrOK
}

// textMethods are the methods of ITextProvider: the document range of a text field, its selection and
// the ranges that are visible (all of the text). RangeFromPoint is not done (SPEC §25.11): its argument
// is a structure of two doubles, which the calling conventions of amd64 and arm64 pass differently.
func textMethods() []win32.MethodDef {
	return []win32.MethodDef{
		elementMethod("GetSelection", 1, func(el *elem, a []uintptr) win32.HRESULT {
			return el.rangeArray(a[0], func(t Text, n *Node) []TextRange {
				if sel, ok := Selection(n); ok {
					return []TextRange{t.Clamp(sel)}
				}
				return nil
			})
		}),
		elementMethod("GetVisibleRanges", 1, func(el *elem, a []uintptr) win32.HRESULT {
			return el.rangeArray(a[0], func(t Text, _ *Node) []TextRange { return []TextRange{t.Document()} })
		}),
		elementMethod("RangeFromChild", 1, func(el *elem, a []uintptr) win32.HRESULT {
			return hresult(ErrNotSupported)
		}),
		elementMethod("RangeFromPoint", 0, func(el *elem, a []uintptr) win32.HRESULT {
			return hresult(ErrNotSupported)
		}),
		elementMethod("get_DocumentRange", 1, func(el *elem, a []uintptr) win32.HRESULT {
			var doc TextRange
			if hr := el.with(func(m *Model, n *Node) win32.HRESULT {
				doc = NewText(textOf(n)).Document()
				return hrOK
			}); hr.Failed() {
				return hr
			}
			return el.b.outRange(a[0], el.ref, doc)
		}),
		elementMethod("get_SupportedTextSelection", 1, func(el *elem, a []uintptr) win32.HRESULT {
			return outInt32(a[0], SupportedTextSelectionSingle)
		}),
	}
}

// rangeArray gives Windows a SAFEARRAY of text ranges that pick chooses from the field's text.
func (el *elem) rangeArray(out uintptr, pick func(t Text, n *Node) []TextRange) win32.HRESULT {
	if out == 0 {
		return win32.EPointer
	}
	*win32.View[uintptr](out) = 0
	var ranges []TextRange
	if hr := el.with(func(m *Model, n *Node) win32.HRESULT {
		ranges = pick(NewText(textOf(n)), n)
		return hrOK
	}); hr.Failed() {
		return hr
	}
	ptrs := make([]uintptr, 0, len(ranges))
	for _, r := range ranges {
		p, err := el.b.newRange(el.ref, r)
		if err != nil {
			return win32.EOutOfMemory
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
	*win32.View[uintptr](out) = sa
	return hrOK
}

// other reads a text range that a client passed back: it must be one of ours, of the same field.
func (r *rangeObj) other(ptr uintptr) (*rangeObj, bool) {
	o, ok := r.b.env.host.ImplOf(ptr).(*rangeObj)
	if !ok || o.ref != r.ref {
		return nil, false
	}
	return o, true
}

// rangeMethods are the methods of ITextRangeProvider.
func rangeMethods(e *Env) map[string]win32.MethodDef {
	defs := []win32.MethodDef{
		rangeMethod("Clone", 1, func(r *rangeObj, a []uintptr) win32.HRESULT { return r.b.outRange(a[0], r.ref, r.r) }),
		rangeMethod("Compare", 2, func(r *rangeObj, a []uintptr) win32.HRESULT {
			o, ok := r.other(a[0])
			if !ok {
				return win32.EInvalidArg
			}
			return outBool(a[1], r.r == o.r)
		}),
		rangeMethod("CompareEndpoints", 4, func(r *rangeObj, a []uintptr) win32.HRESULT {
			o, ok := r.other(a[1])
			if !ok {
				return win32.EInvalidArg
			}
			return outInt32(a[3], int32(CompareEndpoints(r.r, int(int32(a[0])), o.r, int(int32(a[2])))))
		}),
		rangeMethod("ExpandToEnclosingUnit", 1, func(r *rangeObj, a []uintptr) win32.HRESULT {
			t, ok := r.text()
			if !ok {
				return errGone
			}
			r.r = t.Expand(r.r, int(int32(a[0])))
			return hrOK
		}),
		rangeMethod("FindAttribute", 0, func(r *rangeObj, a []uintptr) win32.HRESULT { return hresult(ErrNotSupported) }),
		rangeMethod("FindText", 4, func(r *rangeObj, a []uintptr) win32.HRESULT {
			t, ok := r.text()
			if !ok {
				return errGone
			}
			if a[0] == 0 || a[3] == 0 {
				return win32.EPointer
			}
			*win32.View[uintptr](a[3]) = 0
			needle := win32.BSTRToString(a[0])
			found, ok := t.Find(r.r, needle, int32(a[1]) != 0, int32(a[2]) != 0)
			if !ok {
				return hrOK
			}
			return r.b.outRange(a[3], r.ref, found)
		}),
		rangeMethod("GetAttributeValue", 2, func(r *rangeObj, a []uintptr) win32.HRESULT {
			// Text attributes are not done (SPEC §25.11): the answer is the "not supported" value.
			if a[1] == 0 {
				return win32.EPointer
			}
			*win32.View[win32.Variant](a[1]) = win32.Variant{}
			return hresult(ErrNotSupported)
		}),
		rangeMethod("GetBoundingRectangles", 1, func(r *rangeObj, a []uintptr) win32.HRESULT {
			if a[0] == 0 {
				return win32.EPointer
			}
			*win32.View[uintptr](a[0]) = 0
			ox, oy, _, _, ok := r.b.core.ClientRect(r.ref.View)
			if !ok {
				return errGone
			}
			var rect Rect
			if hr := r.b.withNode(r.ref, func(m *Model, n *Node) win32.HRESULT {
				rect = m.ScreenRect(n, ox, oy)
				return hrOK
			}); hr.Failed() {
				return hr
			}
			// The layout of the text is not known to the semantics tree: a range that is not empty has the
			// rectangle of the field.
			var nums []float64
			if r.r.End > r.r.Start {
				nums = []float64{rect.X, rect.Y, rect.Width, rect.Height}
			}
			sa, err := float64Array(nums)
			if err != nil {
				return win32.EOutOfMemory
			}
			*win32.View[uintptr](a[0]) = sa
			return hrOK
		}),
		rangeMethod("GetEnclosingElement", 1, func(r *rangeObj, a []uintptr) win32.HRESULT {
			p, err := r.b.pointer(r.ref, IfaceSimple)
			if err != nil {
				return errGone
			}
			return outPtr(a[0], p)
		}),
		rangeMethod("GetText", 2, func(r *rangeObj, a []uintptr) win32.HRESULT {
			t, ok := r.text()
			if !ok {
				return errGone
			}
			max := int(int32(a[0]))
			if max < -1 {
				return win32.EInvalidArg
			}
			return outBSTR(a[1], t.Substring(r.r, max))
		}),
		rangeMethod("Move", 3, func(r *rangeObj, a []uintptr) win32.HRESULT {
			t, ok := r.text()
			if !ok {
				return errGone
			}
			nr, moved := t.Move(r.r, int(int32(a[0])), int(int32(a[1])))
			r.r = nr
			return outInt32(a[2], int32(moved))
		}),
		rangeMethod("MoveEndpointByUnit", 4, func(r *rangeObj, a []uintptr) win32.HRESULT {
			t, ok := r.text()
			if !ok {
				return errGone
			}
			nr, moved := t.MoveEndpoint(r.r, int(int32(a[0])), int(int32(a[1])), int(int32(a[2])))
			r.r = nr
			return outInt32(a[3], int32(moved))
		}),
		rangeMethod("MoveEndpointByRange", 3, func(r *rangeObj, a []uintptr) win32.HRESULT {
			o, ok := r.other(a[1])
			if !ok {
				return win32.EInvalidArg
			}
			r.r = MoveEndpointToEndpoint(r.r, int(int32(a[0])), o.r, int(int32(a[2])))
			return hrOK
		}),
		rangeMethod("Select", 0, func(r *rangeObj, a []uintptr) win32.HRESULT {
			rng := r.r
			return statusHR(r.b.core.Request(func(m *Model) ([]ui.SemanticsActionEvent, Status) { return m.SelectRange(r.ref, rng) }))
		}),
		rangeMethod("AddToSelection", 0, func(r *rangeObj, a []uintptr) win32.HRESULT { return hresult(ErrInvalidOperation) }),
		rangeMethod("RemoveFromSelection", 0, func(r *rangeObj, a []uintptr) win32.HRESULT { return hresult(ErrInvalidOperation) }),
		rangeMethod("ScrollIntoView", 1, func(r *rangeObj, a []uintptr) win32.HRESULT {
			return statusHR(r.b.core.Request(func(m *Model) ([]ui.SemanticsActionEvent, Status) { return m.ShowOnScreen(r.ref) }))
		}),
		rangeMethod("GetChildren", 1, func(r *rangeObj, a []uintptr) win32.HRESULT {
			if a[0] == 0 {
				return win32.EPointer
			}
			sa, err := unknownArray(nil)
			if err != nil {
				return win32.EOutOfMemory
			}
			*win32.View[uintptr](a[0]) = sa
			return hrOK
		}),
	}
	return defsOf(defs)
}

// withNode runs fn on a node under the core's lock.
func (b *Bridge) withNode(ref NodeRef, fn func(m *Model, n *Node) win32.HRESULT) win32.HRESULT {
	hr := errGone
	b.core.Read(func(m *Model) {
		if n := m.Node(ref); n != nil {
			hr = fn(m, n)
		}
	})
	return hr
}
