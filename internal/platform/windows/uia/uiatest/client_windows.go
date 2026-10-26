//go:build windows

package uiatest

import (
	"math"
	"unsafe"

	"github.com/rm4n0s/glutter/internal/platform/windows/uia"
	"github.com/rm4n0s/glutter/internal/platform/windows/uia/callshim"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// TreeScopeSubtree is TreeScope_Subtree: the element, its children and all its descendants.
//
// Flutter: no counterpart
const TreeScopeSubtree = 7

// Client is the UI Automation client (IUIAutomation). The thread that uses it is in a multithreaded
// apartment (InitMTA) or a single-threaded one that pumps messages.
//
// Flutter: no counterpart
type Client struct {
	auto win32.Object
}

// InitMTA makes the calling thread a multithreaded COM apartment and returns the function that undoes it.
// The thread must stay the same (runtime.LockOSThread) until then.
//
// Flutter: no counterpart
func InitMTA() (func(), error) {
	if err := win32.CoInitializeEx(0 /* COINIT_MULTITHREADED */); err != nil {
		return nil, err
	}
	return win32.CoUninitialize, nil
}

// NewClient creates the client: CUIAutomation. It fails with
// COMCallFailed when the class is not there (the HRESULT in the metadata says why).
//
// Flutter: no counterpart
func NewClient() (*Client, error) {
	auto, err := win32.CoCreateInstance(&CLSIDAutomation, &IfaceAutomation.IID)
	if err != nil {
		return nil, err
	}
	c := &Client{auto: auto}
	return c, nil
}

// Close releases the client.
//
// Flutter: no counterpart
func (c *Client) Close() {
	c.auto.Release()
}

// call calls the method of the interface.
//
//go:uintptrescapes
func call(o win32.Object, iface uia.Interface, method string, args ...uintptr) error {
	return o.CallHR(iface.Name+"::"+method, iface.Index(method), args...)
}

// Element is an element of the tree (IUIAutomationElement).
//
// Flutter: no counterpart
type Element struct {
	c   *Client
	obj win32.Object
}

// Release drops the element.
//
// Flutter: no counterpart
func (e *Element) Release() {
	if e != nil && !e.obj.IsNil() {
		e.obj.Release()
		e.obj = win32.Object{}
	}
}

// ElementFromHandle is the element of the window: the one UI Automation makes of the HWND, merged with
// the provider that the window answered to WM_GETOBJECT.
//
// Flutter: no counterpart
func (c *Client) ElementFromHandle(hwnd uintptr) (*Element, error) {
	var out uintptr
	if err := call(c.auto, IfaceAutomation, "ElementFromHandle", hwnd, uintptr(unsafe.Pointer(&out))); err != nil {
		return nil, err
	}
	return &Element{c: c, obj: win32.ObjectFromPtr(out)}, nil
}

// ElementFromPoint is the element at a point of the screen.
//
// Flutter: no counterpart
func (c *Client) ElementFromPoint(x, y int32) (*Element, error) {
	var out uintptr
	pt := uintptr(uint32(x)) | uintptr(uint32(y))<<32 // POINT is 8 bytes: one register
	if err := call(c.auto, IfaceAutomation, "ElementFromPoint", pt, uintptr(unsafe.Pointer(&out))); err != nil {
		return nil, err
	}
	return &Element{c: c, obj: win32.ObjectFromPtr(out)}, nil
}

// FocusedElement is the element that has the keyboard focus.
//
// Flutter: no counterpart
func (c *Client) FocusedElement() (*Element, error) {
	var out uintptr
	if err := call(c.auto, IfaceAutomation, "GetFocusedElement", uintptr(unsafe.Pointer(&out))); err != nil {
		return nil, err
	}
	return &Element{c: c, obj: win32.ObjectFromPtr(out)}, nil
}

// Children are the children of the element: FindAll over the true condition with the scope of the
// children (Wine's client has no tree walkers). The caller releases each.
//
// Flutter: no counterpart
func (e *Element) Children() ([]*Element, error) { return e.findAll(2 /* TreeScope_Children */) }

// Descendants are all the elements below the element, in tree order.
//
// Flutter: no counterpart
func (e *Element) Descendants() ([]*Element, error) { return e.findAll(4 /* TreeScope_Descendants */) }

func (e *Element) findAll(scope uintptr) ([]*Element, error) {
	var cond uintptr
	if err := call(e.c.auto, IfaceAutomation, "CreateTrueCondition", uintptr(unsafe.Pointer(&cond))); err != nil {
		return nil, err
	}
	condition := win32.ObjectFromPtr(cond)
	defer condition.Release()
	var arr uintptr
	if err := call(e.obj, IfaceElement, "FindAll", scope, cond, uintptr(unsafe.Pointer(&arr))); err != nil {
		return nil, err
	}
	array := win32.ObjectFromPtr(arr)
	defer array.Release()
	var n int32
	if err := call(array, IfaceElementArray, "get_Length", uintptr(unsafe.Pointer(&n))); err != nil {
		return nil, err
	}
	out := make([]*Element, 0, n)
	for i := range n {
		var el uintptr
		if err := call(array, IfaceElementArray, "GetElement", uintptr(i), uintptr(unsafe.Pointer(&el))); err != nil {
			for _, e := range out {
				e.Release()
			}
			return nil, err
		}
		out = append(out, &Element{c: e.c, obj: win32.ObjectFromPtr(el)})
	}
	return out, nil
}

func (e *Element) bstr(method string) (string, error) {
	var b uintptr
	if err := call(e.obj, IfaceElement, method, uintptr(unsafe.Pointer(&b))); err != nil {
		return "", err
	}
	s := win32.BSTRToString(b)
	win32.SysFreeString(b)
	return s, nil
}

func (e *Element) boolean(method string) (bool, error) {
	var v int32
	if err := call(e.obj, IfaceElement, method, uintptr(unsafe.Pointer(&v))); err != nil {
		return false, err
	}
	return v != 0, nil
}

// Name is CurrentName.
//
// Flutter: no counterpart
func (e *Element) Name() (string, error) { return e.bstr("get_CurrentName") }

// HelpText is CurrentHelpText.
//
// Flutter: no counterpart
func (e *Element) HelpText() (string, error) { return e.bstr("get_CurrentHelpText") }

// AutomationID is CurrentAutomationId.
//
// Flutter: no counterpart
func (e *Element) AutomationID() (string, error) { return e.bstr("get_CurrentAutomationId") }

// ClassName is CurrentClassName.
//
// Flutter: no counterpart
func (e *Element) ClassName() (string, error) { return e.bstr("get_CurrentClassName") }

// FrameworkID is CurrentFrameworkId.
//
// Flutter: no counterpart
func (e *Element) FrameworkID() (string, error) { return e.bstr("get_CurrentFrameworkId") }

// IsEnabled is CurrentIsEnabled.
//
// Flutter: no counterpart
func (e *Element) IsEnabled() (bool, error) { return e.boolean("get_CurrentIsEnabled") }

// HasKeyboardFocus is CurrentHasKeyboardFocus.
//
// Flutter: no counterpart
func (e *Element) HasKeyboardFocus() (bool, error) { return e.boolean("get_CurrentHasKeyboardFocus") }

// IsKeyboardFocusable is CurrentIsKeyboardFocusable.
//
// Flutter: no counterpart
func (e *Element) IsKeyboardFocusable() (bool, error) {
	return e.boolean("get_CurrentIsKeyboardFocusable")
}

// IsOffscreen is CurrentIsOffscreen.
//
// Flutter: no counterpart
func (e *Element) IsOffscreen() (bool, error) { return e.boolean("get_CurrentIsOffscreen") }

// IsPassword is CurrentIsPassword.
//
// Flutter: no counterpart
func (e *Element) IsPassword() (bool, error) { return e.boolean("get_CurrentIsPassword") }

// IsControlElement is CurrentIsControlElement.
//
// Flutter: no counterpart
func (e *Element) IsControlElement() (bool, error) { return e.boolean("get_CurrentIsControlElement") }

// ControlType is CurrentControlType.
//
// Flutter: no counterpart
func (e *Element) ControlType() (uia.ControlType, error) {
	var v int32
	if err := call(e.obj, IfaceElement, "get_CurrentControlType", uintptr(unsafe.Pointer(&v))); err != nil {
		return 0, err
	}
	return uia.ControlType(v), nil
}

// BoundingRectangle is CurrentBoundingRectangle: left, top, right and bottom on the screen.
//
// Flutter: no counterpart
func (e *Element) BoundingRectangle() (win32.Rect, error) {
	var r win32.Rect
	if err := call(e.obj, IfaceElement, "get_CurrentBoundingRectangle", uintptr(unsafe.Pointer(&r))); err != nil {
		return r, err
	}
	return r, nil
}

// SetFocus is IUIAutomationElement::SetFocus.
//
// Flutter: no counterpart
func (e *Element) SetFocus() error { return call(e.obj, IfaceElement, "SetFocus") }

// RuntimeID is GetRuntimeId.
//
// Flutter: no counterpart
func (e *Element) RuntimeID() ([]int32, error) {
	var sa uintptr
	if err := call(e.obj, IfaceElement, "GetRuntimeId", uintptr(unsafe.Pointer(&sa))); err != nil {
		return nil, err
	}
	defer win32.SafeArrayDestroy(sa)
	lb, err := win32.SafeArrayGetLBound(sa)
	if err != nil {
		return nil, err
	}
	ub, err := win32.SafeArrayGetUBound(sa)
	if err != nil {
		return nil, err
	}
	var out []int32
	for i := lb; i <= ub; i++ {
		var v int32
		if err := win32.SafeArrayGetElement(sa, i, unsafe.Pointer(&v)); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// Property is GetCurrentPropertyValue.
//
// Flutter: no counterpart
func (e *Element) Property(id uia.PropertyID) (uia.Value, error) {
	var v win32.Variant
	if err := call(e.obj, IfaceElement, "GetCurrentPropertyValue", uintptr(id), uintptr(unsafe.Pointer(&v))); err != nil {
		return uia.Value{}, err
	}
	defer win32.VariantClear(&v)
	switch v.Vt {
	case win32.VTBool:
		return uia.Bool(uint16(v.Val) != 0), nil
	case win32.VTI4:
		return uia.Int(int(int32(v.Val))), nil
	case win32.VTR8:
		return uia.Float(math.Float64frombits(v.Val)), nil
	case win32.VTBSTR:
		return uia.String(win32.BSTRToString(uintptr(v.Val))), nil
	}
	return uia.Value{}, nil
}

// Pattern is GetCurrentPattern asked for the client interface of the pattern; ok is false when the element
// does not support it. The caller releases the object.
//
// Flutter: no counterpart
func (e *Element) Pattern(id uia.PatternID, iface uia.Interface) (obj win32.Object, ok bool, err error) {
	var p uintptr
	if err := call(e.obj, IfaceElement, "GetCurrentPattern", uintptr(id), uintptr(unsafe.Pointer(&p))); err != nil {
		return win32.Object{}, false, err
	}
	if p == 0 {
		return win32.Object{}, false, nil
	}
	unk := win32.ObjectFromPtr(p)
	defer unk.Release()
	q, err := unk.QueryInterface(&iface.IID)
	if err != nil {
		return win32.Object{}, false, err
	}
	return q, true, nil
}

// with gets a pattern and runs fn on it. A pattern the element does not support fails with
// COMCallFailed for UIA_E_NOTSUPPORTED.
func (e *Element) with(id uia.PatternID, iface uia.Interface, fn func(p win32.Object) error) error {
	p, ok, err := e.Pattern(id, iface)
	if err != nil {
		return err
	}
	if !ok {
		// What UI Automation answers when an element lacks the pattern: UIA_E_NOTSUPPORTED.
		notSupported := uint32(uia.ErrNotSupported)
		return win32.HResultError("IUIAutomationElement::GetCurrentPattern", win32.HRESULT(int32(notSupported)))
	}
	defer p.Release()
	return fn(p)
}

// Invoke is IUIAutomationInvokePattern::Invoke.
//
// Flutter: no counterpart
func (e *Element) Invoke() error {
	return e.with(uia.PatternInvoke, IfaceInvoke, func(p win32.Object) error { return call(p, IfaceInvoke, "Invoke") })
}

// Toggle is IUIAutomationTogglePattern::Toggle.
//
// Flutter: no counterpart
func (e *Element) Toggle() error {
	return e.with(uia.PatternToggle, IfaceToggle, func(p win32.Object) error { return call(p, IfaceToggle, "Toggle") })
}

// ToggleState is CurrentToggleState.
//
// Flutter: no counterpart
func (e *Element) ToggleState() (state int, err error) {
	err = e.with(uia.PatternToggle, IfaceToggle, func(p win32.Object) error {
		var v int32
		if err := call(p, IfaceToggle, "get_CurrentToggleState", uintptr(unsafe.Pointer(&v))); err != nil {
			return err
		}
		state = int(v)
		return nil
	})
	return state, err
}

// SetValue is IUIAutomationValuePattern::SetValue.
//
// Flutter: no counterpart
func (e *Element) SetValue(text string) error {
	return e.with(uia.PatternValue, IfaceValue, func(p win32.Object) error {
		b, err := win32.SysAllocString(text)
		if err != nil {
			return err
		}
		defer win32.SysFreeString(b)
		return call(p, IfaceValue, "SetValue", b)
	})
}

// Value is CurrentValue of the Value pattern.
//
// Flutter: no counterpart
func (e *Element) Value() (text string, err error) {
	err = e.with(uia.PatternValue, IfaceValue, func(p win32.Object) error {
		var b uintptr
		if err := call(p, IfaceValue, "get_CurrentValue", uintptr(unsafe.Pointer(&b))); err != nil {
			return err
		}
		text = win32.BSTRToString(b)
		win32.SysFreeString(b)
		return nil
	})
	return text, err
}

// IsReadOnly is CurrentIsReadOnly of the Value pattern.
//
// Flutter: no counterpart
func (e *Element) IsReadOnly() (ro bool, err error) {
	err = e.with(uia.PatternValue, IfaceValue, func(p win32.Object) error {
		var v int32
		if err := call(p, IfaceValue, "get_CurrentIsReadOnly", uintptr(unsafe.Pointer(&v))); err != nil {
			return err
		}
		ro = v != 0
		return nil
	})
	return ro, err
}

// SetRangeValue is IUIAutomationRangeValuePattern::SetValue, which takes a double: it is called through
// the shim of package callshim.
//
// Flutter: no counterpart
func (e *Element) SetRangeValue(v float64) error {
	return e.with(uia.PatternRangeValue, IfaceRangeValue, func(p win32.Object) error {
		fn := p.Method(IfaceRangeValue.Index("SetValue"))
		return win32.HResultError("IUIAutomationRangeValuePattern::SetValue", win32.FromResult(callshim.Call(fn, p.Ptr(), v, 0, 0)))
	})
}

// RangeValue is CurrentValue of the RangeValue pattern, with the minimum and the maximum.
//
// Flutter: no counterpart
func (e *Element) RangeValue() (value, minimum, maximum float64, err error) {
	err = e.with(uia.PatternRangeValue, IfaceRangeValue, func(p win32.Object) error {
		for method, out := range map[string]*float64{"get_CurrentValue": &value, "get_CurrentMinimum": &minimum, "get_CurrentMaximum": &maximum} {
			if err := call(p, IfaceRangeValue, method, uintptr(unsafe.Pointer(out))); err != nil {
				return err
			}
		}
		return nil
	})
	return value, minimum, maximum, err
}

// Scroll is IUIAutomationScrollPattern::Scroll.
//
// Flutter: no counterpart
func (e *Element) Scroll(horizontal, vertical int) error {
	return e.with(uia.PatternScroll, IfaceScroll, func(p win32.Object) error {
		return call(p, IfaceScroll, "Scroll", uintptr(horizontal), uintptr(vertical))
	})
}

// SetScrollPercent is IUIAutomationScrollPattern::SetScrollPercent, through the shim (two doubles).
//
// Flutter: no counterpart
func (e *Element) SetScrollPercent(horizontal, vertical float64) error {
	return e.with(uia.PatternScroll, IfaceScroll, func(p win32.Object) error {
		fn := p.Method(IfaceScroll.Index("SetScrollPercent"))
		return win32.HResultError("IUIAutomationScrollPattern::SetScrollPercent", win32.FromResult(callshim.Call(fn, p.Ptr(), horizontal, vertical, 0)))
	})
}

// ScrollPercent is the vertical and horizontal scroll percentages (-1 for an axis that does not scroll).
//
// Flutter: no counterpart
func (e *Element) ScrollPercent() (horizontal, vertical float64, err error) {
	err = e.with(uia.PatternScroll, IfaceScroll, func(p win32.Object) error {
		if err := call(p, IfaceScroll, "get_CurrentHorizontalScrollPercent", uintptr(unsafe.Pointer(&horizontal))); err != nil {
			return err
		}
		return call(p, IfaceScroll, "get_CurrentVerticalScrollPercent", uintptr(unsafe.Pointer(&vertical)))
	})
	return horizontal, vertical, err
}

// Select is IUIAutomationSelectionItemPattern::Select.
//
// Flutter: no counterpart
func (e *Element) Select() error {
	return e.with(uia.PatternSelectionItem, IfaceSelectionItem, func(p win32.Object) error { return call(p, IfaceSelectionItem, "Select") })
}

// IsSelected is CurrentIsSelected.
//
// Flutter: no counterpart
func (e *Element) IsSelected() (sel bool, err error) {
	err = e.with(uia.PatternSelectionItem, IfaceSelectionItem, func(p win32.Object) error {
		var v int32
		if err := call(p, IfaceSelectionItem, "get_CurrentIsSelected", uintptr(unsafe.Pointer(&v))); err != nil {
			return err
		}
		sel = v != 0
		return nil
	})
	return sel, err
}

// Expand is IUIAutomationExpandCollapsePattern::Expand.
//
// Flutter: no counterpart
func (e *Element) Expand() error {
	return e.with(uia.PatternExpandCollapse, IfaceExpandCollapse, func(p win32.Object) error { return call(p, IfaceExpandCollapse, "Expand") })
}

// Collapse is IUIAutomationExpandCollapsePattern::Collapse.
//
// Flutter: no counterpart
func (e *Element) Collapse() error {
	return e.with(uia.PatternExpandCollapse, IfaceExpandCollapse, func(p win32.Object) error { return call(p, IfaceExpandCollapse, "Collapse") })
}

// ExpandCollapseState is CurrentExpandCollapseState.
//
// Flutter: no counterpart
func (e *Element) ExpandCollapseState() (state int, err error) {
	err = e.with(uia.PatternExpandCollapse, IfaceExpandCollapse, func(p win32.Object) error {
		var v int32
		if err := call(p, IfaceExpandCollapse, "get_CurrentExpandCollapseState", uintptr(unsafe.Pointer(&v))); err != nil {
			return err
		}
		state = int(v)
		return nil
	})
	return state, err
}

// TextRange is a range of the Text pattern (IUIAutomationTextRange).
//
// Flutter: no counterpart
type TextRange struct{ obj win32.Object }

// Release drops the range.
//
// Flutter: no counterpart
func (r TextRange) Release() { r.obj.Release() }

// DocumentRange is the range of all the text of the element.
//
// Flutter: no counterpart
func (e *Element) DocumentRange() (r TextRange, err error) {
	err = e.with(uia.PatternText, IfaceText, func(p win32.Object) error {
		var out uintptr
		if err := call(p, IfaceText, "get_DocumentRange", uintptr(unsafe.Pointer(&out))); err != nil {
			return err
		}
		r = TextRange{win32.ObjectFromPtr(out)}
		return nil
	})
	return r, err
}

// SelectionRanges are the ranges of the selection, with the text of each.
//
// Flutter: no counterpart
func (e *Element) SelectionRanges() (texts []string, err error) {
	err = e.with(uia.PatternText, IfaceText, func(p win32.Object) error {
		var arr uintptr
		if err := call(p, IfaceText, "GetSelection", uintptr(unsafe.Pointer(&arr))); err != nil {
			return err
		}
		var n uintptr
		if err := call(win32.ObjectFromPtr(arr), IfaceTextRangeArray, "get_Length", uintptr(unsafe.Pointer(&n))); err != nil {
			return err
		}
		for i := uintptr(0); i < n; i++ {
			var rp uintptr
			if err := call(win32.ObjectFromPtr(arr), IfaceTextRangeArray, "GetElement", i, uintptr(unsafe.Pointer(&rp))); err != nil {
				return err
			}
			r := TextRange{win32.ObjectFromPtr(rp)}
			s, err := r.Text(-1)
			r.Release()
			if err != nil {
				return err
			}
			texts = append(texts, s)
		}
		win32.ObjectFromPtr(arr).Release()
		return nil
	})
	return texts, err
}

// Text is GetText, cut to maxLength units (-1 for no limit).
//
// Flutter: no counterpart
func (r TextRange) Text(maxLength int) (string, error) {
	var b uintptr
	if err := call(r.obj, IfaceTextRange, "GetText", uintptr(maxLength), uintptr(unsafe.Pointer(&b))); err != nil {
		return "", err
	}
	s := win32.BSTRToString(b)
	win32.SysFreeString(b)
	return s, nil
}

// Select is ITextRange::Select.
//
// Flutter: no counterpart
func (r TextRange) Select() error { return call(r.obj, IfaceTextRange, "Select") }

// Move is ITextRange::Move; it returns the number of units moved.
//
// Flutter: no counterpart
func (r TextRange) Move(unit, count int) (int, error) {
	var moved int32
	if err := call(r.obj, IfaceTextRange, "Move", uintptr(unit), uintptr(count), uintptr(unsafe.Pointer(&moved))); err != nil {
		return 0, err
	}
	return int(moved), nil
}

// ExpandToEnclosingUnit is ITextRange::ExpandToEnclosingUnit.
//
// Flutter: no counterpart
func (r TextRange) ExpandToEnclosingUnit(unit int) error {
	return call(r.obj, IfaceTextRange, "ExpandToEnclosingUnit", uintptr(unit))
}

// MoveEndpointByUnit is ITextRange::MoveEndpointByUnit.
//
// Flutter: no counterpart
func (r TextRange) MoveEndpointByUnit(endpoint, unit, count int) (int, error) {
	var moved int32
	if err := call(r.obj, IfaceTextRange, "MoveEndpointByUnit", uintptr(endpoint), uintptr(unit), uintptr(count), uintptr(unsafe.Pointer(&moved))); err != nil {
		return 0, err
	}
	return int(moved), nil
}
