//go:build windows

package uia

import (
	"bytes"
	"log/slog"
	"math"
	"reflect"
	"strings"
	"sync"
	"testing"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/windows/uia/callshim"
	"github.com/rm4n0s/glutter/internal/platform/windows/uia/semtree"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// These tests call the providers through their vtables, the way UI Automation does, without UI
// Automation: the COM objects made in Go, the thunks of the methods that take a double, and the
// guard of the methods. They run inside Wine with the other unit tests of the packages that build for
// Windows (scripts/test-windows.sh).

// testWindow is the window of the tests: a client area at a place on the screen.
type testWindow struct{ x, y, w, h float64 }

func (f testWindow) ClientRect(ui.ViewID) (x, y, w, h float64, ok bool) {
	return f.x, f.y, f.w, f.h, true
}
func (f testWindow) HWND(ui.ViewID) uintptr { return 0x1234 }
func (f testWindow) Post(fn func())         { fn() }

// testHost records what the bridge asks of the framework.
type testHost struct {
	mu      sync.Mutex
	changes []bool
	actions []ui.SemanticsActionEvent
}

func (h *testHost) ScreenReaderChanged(on bool) {
	h.mu.Lock()
	h.changes = append(h.changes, on)
	h.mu.Unlock()
}
func (h *testHost) PerformAction(ev ui.SemanticsActionEvent) {
	h.mu.Lock()
	h.actions = append(h.actions, ev)
	h.mu.Unlock()
}
func (h *testHost) take() []ui.SemanticsActionEvent {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := h.actions
	h.actions = nil
	return out
}

// bridgeOf makes a bridge that is on, with the tree.
func bridgeOf(t *testing.T, win testWindow, update *ui.SemanticsUpdate) (*Bridge, *testHost) {
	t.Helper()
	env, err := Shared()
	if err != nil {
		t.Fatal(err)
	}
	b := NewBridge(env, win, nil)
	t.Cleanup(b.Close)
	host := &testHost{}
	b.Core().Start(host)
	if !b.Core().Ask(RootObjectID) {
		t.Fatal("the bridge did not turn on")
	}
	b.SetWindow(1, "Test", true)
	b.UpdateSemantics(1, update)
	return b, host
}

// vcall calls a method of an interface of an element by name and returns the HRESULT.
//
//go:uintptrescapes
func vcall(o win32.Object, iface Interface, method string, args ...uintptr) win32.HRESULT {
	return win32.FromResult(o.Call(iface.Index(method), args...))
}

// iface returns the interface pointer of the element as an object for vcall.
func iface(t *testing.T, b *Bridge, ref NodeRef, i Interface) win32.Object {
	t.Helper()
	p, err := b.pointer(ref, i)
	if err != nil {
		t.Fatalf("%s of %v: %v", i.Name, ref, err)
	}
	o := win32.ObjectFromPtr(p)
	t.Cleanup(func() { o.Release() })
	return o
}

func testTree() *ui.SemanticsUpdate {
	root := semtree.N(0, "", 0, 0, 400, 300, 1, 2, 3, 4, 5, 6, 7, 8)
	button := semtree.N(1, "Press", 10, 10, 80, 30)
	button.Flags.IsButton = true
	button.Actions = ui.SemanticsActionTap
	label := semtree.N(2, "Hello", 10, 50, 80, 30)
	check := semtree.N(3, "Agree", 100, 10, 80, 30)
	check.Flags.IsChecked = ui.CheckedStateIsTrue
	check.Actions = ui.SemanticsActionTap
	slider := semtree.N(4, "Level", 10, 100, 100, 20)
	slider.Flags.IsSlider = true
	slider.Value = "50%"
	slider.Actions = ui.SemanticsActionIncrease | ui.SemanticsActionDecrease
	list := semtree.N(5, "Items", 200, 100, 100, 100)
	list.Actions = ui.SemanticsActionScrollUp | ui.SemanticsActionScrollDown | ui.SemanticsActionScrollToOffset
	list.ScrollPosition, list.ScrollExtentMin, list.ScrollExtentMax = 0, 0, 200
	branch := semtree.N(6, "Branch", 300, 10, 80, 30)
	branch.Flags.IsExpanded = ui.TristateIsFalse
	branch.Actions = ui.SemanticsActionExpand
	radio := semtree.N(7, "Option", 300, 50, 80, 30)
	radio.Flags.IsChecked = ui.CheckedStateIsFalse
	radio.Flags.IsInMutuallyExclusiveGroup = true
	radio.Actions = ui.SemanticsActionTap
	group := semtree.N(8, "Group", 300, 100, 80, 80, 9, 10)
	chosen := semtree.N(9, "Chosen", 300, 100, 80, 30)
	chosen.Flags.IsSelected = ui.TristateIsTrue
	other := semtree.N(10, "Other", 300, 130, 80, 30)
	other.Flags.IsSelected = ui.TristateIsFalse
	other.Actions = ui.SemanticsActionTap
	field := semtree.N(11, "Name", 10, 200, 200, 30)
	field.Flags.IsTextField = true
	field.Flags.IsFocused = ui.TristateIsFalse
	field.Value = "héllo wörld"
	field.Actions = ui.SemanticsActionFocus | ui.SemanticsActionSetSelection
	field.TextSelectionBase, field.TextSelectionExtent = 1, 4
	root.ChildrenInTraversalOrder = append(root.ChildrenInTraversalOrder, 11)
	root.ChildrenInHitTestOrder = root.ChildrenInTraversalOrder
	return semtree.Update(root, button, label, check, slider, list, branch, radio, group, chosen, other, field)
}

func TestThunkElementProviderFromPointGetsTheDoubles(t *testing.T) {
	// The window is at (1000, 500) on the screen; the point is on the screen too.
	b, _ := bridgeOf(t, testWindow{x: 1000, y: 500, w: 400, h: 300}, testTree())
	root := iface(t, b, NodeRef{1, FragmentRootID}, IfaceFragmentRoot)
	point, _, _ := ThunkEntries()
	if got := root.Method(IfaceFragmentRoot.Index("ElementProviderFromPoint")); got != point {
		t.Fatalf("the vtable entry is %#x, want the thunk %#x", got, point)
	}
	tests := []struct {
		name string
		x, y float64
		want int // the node, -1 for none
	}{
		{"inside the button", 1010 + 5, 500 + 10 + 5, 1},
		{"the top left corner of the button", 1010, 510, 1},
		{"a fraction before the right edge", 1010 + 79.75, 520, 1},
		{"the right edge is outside the button", 1010 + 80, 520, 0},
		{"a fraction past the bottom edge", 1010 + 5, 510 + 30.25, 0},
		{"the label", 1010 + 40.5, 500 + 50.5 + 10, 2},
		{"a point in the root only", 1000 + 399.5, 500 + 299.5, 0},
		{"left of the window", 999.5, 520, -1},
		{"below the window", 1010, 500 + 300, -1},
		{"negative coordinates", -10.5, -20.25, -1},
		{"not a number", math.NaN(), 520, -1},
		{"infinity", math.Inf(1), 520, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out uintptr = 0xDEAD
			hr := win32.FromResult(callshim.Call(point, root.Ptr(), tt.x, tt.y, uintptr(unsafe.Pointer(&out))))
			if hr != win32.SOK {
				t.Fatalf("hr = %v", hr)
			}
			if tt.want < 0 {
				if out != 0 {
					t.Errorf("got %#x, want null", out)
				}
				return
			}
			want := iface(t, b, NodeRef{1, tt.want}, IfaceFragment)
			if out != want.Ptr() {
				t.Errorf("got %#x, want the provider of node %d (%#x)", out, tt.want, want.Ptr())
			}
			win32.ObjectFromPtr(out).Release()
		})
	}
	// A null result pointer is E_POINTER.
	if hr := win32.FromResult(callshim.Call(point, root.Ptr(), 1.5, 2.5, 0)); hr != win32.EPointer {
		t.Errorf("a null result pointer gave %v, want E_POINTER", hr)
	}
}

func TestThunkRangeValueSetValueGetsTheDouble(t *testing.T) {
	b, host := bridgeOf(t, testWindow{w: 400, h: 300}, testTree())
	rv := iface(t, b, NodeRef{1, 4}, IfaceRangeValue)
	_, _, entry := ThunkEntries()
	if got := rv.Method(IfaceRangeValue.Index("SetValue")); got != entry {
		t.Fatalf("the vtable entry is %#x, want the thunk %#x", got, entry)
	}
	inc := []ui.SemanticsActionEvent{{ViewID: 1, NodeID: 4, Type: ui.SemanticsActionIncrease}}
	dec := []ui.SemanticsActionEvent{{ViewID: 1, NodeID: 4, Type: ui.SemanticsActionDecrease}}
	tests := []struct {
		name string
		v    float64
		hr   win32.HRESULT
		want []ui.SemanticsActionEvent
	}{
		{"exactly the current value", 50, win32.SOK, nil},
		{"a hair above", 50.0000001, win32.SOK, inc},
		{"a hair below", 49.9999999, win32.SOK, dec},
		{"far above", 99, win32.SOK, inc},
		{"far below", 1.5, win32.SOK, dec},
		{"the maximum", 100, win32.SOK, inc},
		{"over the maximum", 100.5, win32.EInvalidArg, nil},
		{"negative", -0.25, win32.EInvalidArg, nil},
		{"not a number", math.NaN(), win32.EInvalidArg, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hr := win32.FromResult(callshim.Call(rv.Method(IfaceRangeValue.Index("SetValue")), rv.Ptr(), tt.v, 0, 0))
			if hr != tt.hr {
				t.Errorf("hr = %v, want %v", hr, tt.hr)
			}
			if got := host.take(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("actions = %+v, want %+v", got, tt.want)
			}
		})
	}
	// The other methods of the interface, with ordinary arguments, still work after a thunk was called.
	var v float64
	if hr := vcall(rv, IfaceRangeValue, "get_Value", uintptr(unsafe.Pointer(&v))); hr != win32.SOK || v != 50 {
		t.Errorf("get_Value = %v, %v", v, hr)
	}
}

func TestThunkSetScrollPercentGetsBothDoubles(t *testing.T) {
	b, host := bridgeOf(t, testWindow{w: 400, h: 300}, testTree())
	sp := iface(t, b, NodeRef{1, 5}, IfaceScroll)
	_, scroll, _ := ThunkEntries()
	if got := sp.Method(IfaceScroll.Index("SetScrollPercent")); got != scroll {
		t.Fatalf("the vtable entry is %#x, want the thunk %#x", got, scroll)
	}
	tests := []struct {
		name string
		h, v float64
		hr   win32.HRESULT
		want []float64 // the offset asked for, nil for no action
	}{
		{"a quarter", NoScroll, 25, win32.SOK, []float64{0, 50}},
		{"a fraction", NoScroll, 37.5, win32.SOK, []float64{0, 75}},
		{"the end", NoScroll, 100, win32.SOK, []float64{0, 200}},
		{"the start", NoScroll, 0, win32.SOK, []float64{0, 0}},
		{"over", NoScroll, 100.5, win32.EInvalidArg, nil},
		{"the horizontal axis, where the list does not scroll", 10, NoScroll, win32.EInvalidArg, nil},
		{"not a number", NoScroll, math.NaN(), win32.EInvalidArg, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hr := win32.FromResult(callshim.Call(scroll, sp.Ptr(), tt.h, tt.v, 0))
			if hr != tt.hr {
				t.Errorf("hr = %v, want %v", hr, tt.hr)
			}
			got := host.take()
			if tt.want == nil {
				if len(got) != 0 {
					t.Errorf("actions = %+v, want none", got)
				}
				return
			}
			if len(got) != 1 || got[0].Type != ui.SemanticsActionScrollToOffset || !reflect.DeepEqual(got[0].Arguments, tt.want) {
				t.Errorf("actions = %+v, want an offset of %v", got, tt.want)
			}
		})
	}
}

func TestProviderMethodsThroughTheVtable(t *testing.T) {
	b, host := bridgeOf(t, testWindow{x: 100, y: 200, w: 400, h: 300}, testTree())
	button := iface(t, b, NodeRef{1, 1}, IfaceSimple)
	frag := iface(t, b, NodeRef{1, 1}, IfaceFragment)

	var opts int32
	if hr := vcall(button, IfaceSimple, "get_ProviderOptions", uintptr(unsafe.Pointer(&opts))); hr != win32.SOK || opts != ProviderServerSide|ProviderUseCOMThreading {
		t.Errorf("provider options = %#x, %v; want the server side and COM threading", opts, hr)
	}
	// Properties.
	var v win32.Variant
	if hr := vcall(button, IfaceSimple, "GetPropertyValue", uintptr(PropName), uintptr(unsafe.Pointer(&v))); hr != win32.SOK || v.Vt != win32.VTBSTR || win32.BSTRToString(uintptr(v.Val)) != "Press" {
		t.Errorf("Name = %+v, %v", v, hr)
	}
	win32.VariantClear(&v)
	v = win32.Variant{}
	if hr := vcall(button, IfaceSimple, "GetPropertyValue", uintptr(PropControlType), uintptr(unsafe.Pointer(&v))); hr != win32.SOK || v.Vt != win32.VTI4 || int32(v.Val) != int32(ControlTypeButton) {
		t.Errorf("ControlType = %+v, %v", v, hr)
	}
	v = win32.Variant{}
	if hr := vcall(button, IfaceSimple, "GetPropertyValue", uintptr(PropIsEnabled), uintptr(unsafe.Pointer(&v))); hr != win32.SOK || v.Vt != win32.VTBool || uint16(v.Val) != win32.VariantTrue {
		t.Errorf("IsEnabled = %+v, %v", v, hr)
	}
	v = win32.Variant{}
	if hr := vcall(button, IfaceSimple, "GetPropertyValue", uintptr(PropFullDescription), uintptr(unsafe.Pointer(&v))); hr != win32.SOK || v.Vt != win32.VTEmpty {
		t.Errorf("a property it has no value for = %+v, %v; want empty", v, hr)
	}
	// Patterns.
	for _, tt := range []struct {
		ref     NodeRef
		pattern PatternID
		want    bool
	}{
		{NodeRef{1, 1}, PatternInvoke, true},
		{NodeRef{1, 1}, PatternToggle, false},
		{NodeRef{1, 2}, PatternInvoke, false},
		{NodeRef{1, 3}, PatternToggle, true},
		{NodeRef{1, 4}, PatternRangeValue, true},
		{NodeRef{1, 5}, PatternScroll, true},
		{NodeRef{1, 5}, PatternInvoke, false},
	} {
		simple := iface(t, b, tt.ref, IfaceSimple)
		var p uintptr
		if hr := vcall(simple, IfaceSimple, "GetPatternProvider", uintptr(tt.pattern), uintptr(unsafe.Pointer(&p))); hr != win32.SOK {
			t.Errorf("GetPatternProvider(%v, %d): %v", tt.ref, tt.pattern, hr)
		}
		if (p != 0) != tt.want {
			t.Errorf("GetPatternProvider(%v, %d) = %#x, want a provider: %v", tt.ref, tt.pattern, p, tt.want)
		}
		if p != 0 {
			win32.ObjectFromPtr(p).Release()
		}
	}
	// Invoke reaches the framework.
	invoke := iface(t, b, NodeRef{1, 1}, IfaceInvoke)
	if hr := vcall(invoke, IfaceInvoke, "Invoke"); hr != win32.SOK {
		t.Errorf("Invoke: %v", hr)
	}
	if got := host.take(); !reflect.DeepEqual(got, []ui.SemanticsActionEvent{{ViewID: 1, NodeID: 1, Type: ui.SemanticsActionTap}}) {
		t.Errorf("Invoke asked the framework for %+v", got)
	}
	// Navigate.
	nav := func(from NodeRef, dir int) uintptr {
		o := iface(t, b, from, IfaceFragment)
		var out uintptr
		if hr := vcall(o, IfaceFragment, "Navigate", uintptr(dir), uintptr(unsafe.Pointer(&out))); hr != win32.SOK {
			t.Fatalf("Navigate(%v, %d): %v", from, dir, hr)
		}
		return out
	}
	same := func(got uintptr, ref NodeRef, i Interface) bool {
		if got == 0 {
			return false
		}
		defer win32.ObjectFromPtr(got).Release()
		return got == iface(t, b, ref, i).Ptr()
	}
	if !same(nav(NodeRef{1, 1}, NavigateNextSibling), NodeRef{1, 2}, IfaceFragment) {
		t.Error("the next sibling of the button is not the label")
	}
	if got := nav(NodeRef{1, 1}, NavigatePreviousSibling); got != 0 {
		t.Errorf("the first child has a previous sibling %#x", got)
	}
	if !same(nav(NodeRef{1, 0}, NavigateFirstChild), NodeRef{1, 1}, IfaceFragment) || !same(nav(NodeRef{1, 0}, NavigateLastChild), NodeRef{1, 11}, IfaceFragment) {
		t.Error("the first and last children of the root are wrong")
	}
	if !same(nav(NodeRef{1, 3}, NavigateParent), NodeRef{1, 0}, IfaceFragment) {
		t.Error("the parent of the check box is not the root")
	}
	if !same(nav(NodeRef{1, 0}, NavigateParent), NodeRef{1, FragmentRootID}, IfaceFragment) {
		t.Error("the parent of the root node is not the fragment root")
	}
	rootFrag := NodeRef{1, FragmentRootID}
	if !same(nav(rootFrag, NavigateFirstChild), NodeRef{1, 0}, IfaceFragment) || nav(rootFrag, NavigateParent) != 0 || nav(rootFrag, NavigateNextSibling) != 0 {
		t.Error("navigating from the fragment root is wrong")
	}
	// The fragment root of an element, its runtime id and rectangle.
	var fr uintptr
	if hr := vcall(frag, IfaceFragment, "get_FragmentRoot", uintptr(unsafe.Pointer(&fr))); hr != win32.SOK || !same(fr, rootFrag, IfaceFragmentRoot) {
		t.Errorf("get_FragmentRoot = %#x, %v", fr, hr)
	}
	var rect win32.UiaRect
	if hr := vcall(frag, IfaceFragment, "get_BoundingRectangle", uintptr(unsafe.Pointer(&rect))); hr != win32.SOK || rect != (win32.UiaRect{Left: 110, Top: 210, Width: 80, Height: 30}) {
		t.Errorf("BoundingRectangle = %+v, %v; want the rectangle of the button on the screen", rect, hr)
	}
	var sa uintptr
	if hr := vcall(frag, IfaceFragment, "GetRuntimeId", uintptr(unsafe.Pointer(&sa))); hr != win32.SOK || sa == 0 {
		t.Fatalf("GetRuntimeId: %v", hr)
	}
	var id [2]int32
	for i := range id {
		if err := win32.SafeArrayGetElement(sa, int32(i), unsafe.Pointer(&id[i])); err != nil {
			t.Fatal(err)
		}
	}
	win32.SafeArrayDestroy(sa)
	if id != [2]int32{AppendRuntimeID, 2} {
		t.Errorf("runtime id = %v, want [UiaAppendRuntimeId 2]", id)
	}
	// A null out parameter is E_POINTER, not a crash.
	if hr := vcall(frag, IfaceFragment, "get_BoundingRectangle", 0); hr != win32.EPointer {
		t.Errorf("a null out parameter gave %v", hr)
	}
	// The root fragment says its host is the window's provider (a window that does not exist fails cleanly).
	rootSimple := iface(t, b, rootFrag, IfaceSimple)
	var host2 uintptr
	if hr := vcall(rootSimple, IfaceSimple, "get_HostRawElementProvider", uintptr(unsafe.Pointer(&host2))); hr.Succeeded() && host2 != 0 {
		win32.ObjectFromPtr(host2).Release()
	}
}

func TestAMethodOnANodeThatLeftTheTreeSaysTheElementIsGone(t *testing.T) {
	b, _ := bridgeOf(t, testWindow{w: 400, h: 300}, testTree())
	label := iface(t, b, NodeRef{1, 2}, IfaceFragment)
	b.UpdateSemantics(1, semtree.Update(semtree.N(0, "", 0, 0, 400, 300, 1), testTree().Nodes[1]))
	var rect win32.UiaRect
	hr := vcall(label, IfaceFragment, "get_BoundingRectangle", uintptr(unsafe.Pointer(&rect)))
	if uint32(hr) != ErrElementNotAvailable {
		t.Errorf("hr = %v, want UIA_E_ELEMENTNOTAVAILABLE", hr)
	}
	// The bridge let go of the node, but the client's reference keeps the object, and it is still callable.
	var sa uintptr
	if hr := vcall(label, IfaceFragment, "GetRuntimeId", uintptr(unsafe.Pointer(&sa))); hr != win32.SOK {
		t.Errorf("GetRuntimeId of a disconnected node: %v", hr)
	}
	win32.SafeArrayDestroy(sa)
}

func TestTheObjectsAreFreedWhenTheLastReferenceGoes(t *testing.T) {
	env, err := Shared()
	if err != nil {
		t.Fatal(err)
	}
	before := env.Stats()
	b := NewBridge(env, testWindow{w: 400, h: 300}, nil)
	host := &testHost{}
	b.Core().Start(host)
	b.Core().Ask(RootObjectID)
	b.UpdateSemantics(1, testTree())
	for id := 0; id <= 5; id++ {
		p, err := b.pointer(NodeRef{1, id}, IfaceFragment)
		if err != nil {
			t.Fatal(err)
		}
		win32.ObjectFromPtr(p).Release()
	}
	// The bridge holds one object for each node it made, and for the fragment root if an event needed it.
	b.mu.Lock()
	made := len(b.objs)
	b.mu.Unlock()
	if got := env.Stats().Objects - before.Objects; got != made || made < 6 {
		t.Fatalf("%d objects alive, the bridge made %d (at least the 6 nodes asked for)", got, made)
	}
	b.Close()
	if got := env.Stats().Objects - before.Objects; got != 0 {
		t.Errorf("%d objects alive after Close, want 0", got)
	}
}

func TestTheCallbacksOfAProcessStayUnderTheLimit(t *testing.T) {
	if _, err := Shared(); err != nil {
		t.Fatal(err)
	}
	n := win32.CallbackCount()
	if n > win32.CallbackLimit {
		t.Errorf("%d callbacks, over the limit of %d", n, win32.CallbackLimit)
	}
	t.Logf("%d callbacks of the process after the providers' vtables were made", n)
}

func TestAnElementOfAClosedBridgeFailsWithItsTag(t *testing.T) {
	env, err := Shared()
	if err != nil {
		t.Fatal(err)
	}
	b := NewBridge(env, testWindow{w: 400, h: 300}, nil)
	b.Close()
	_, err = b.element(NodeRef{1, 1})
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T", err)
	}
	if !appErr.HasRoute("Bridge.element.WindowsUIAFailed") {
		t.Errorf("unexpected failure path: %s", appErr.Route())
	}
}

func TestUiaReturnRawElementProviderFailsWithItsTag(t *testing.T) {
	b, _ := bridgeOf(t, testWindow{w: 400, h: 300}, testTree())
	root, err := b.element(NodeRef{1, FragmentRootID})
	if err != nil {
		t.Fatal(err)
	}
	// There is no window 1 to answer for: UI Automation gives no result.
	_, err = win32.UiaReturnRawElementProvider(1, 0, uintptr(uint32(0xFFFFFFE7)), root)
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if !appErr.HasRoute("UiaReturnRawElementProvider.WindowsUIAFailed") {
		t.Errorf("unexpected failure path: %s", appErr.Route())
	}
	if appErr.Metadata["function"] != "UiaReturnRawElementProvider" {
		t.Errorf("metadata = %v", appErr.Metadata)
	}
}

func TestAPanicInAMethodIsAnE_FAILAndTheNextCallWorks(t *testing.T) {
	b, _ := bridgeOf(t, testWindow{w: 400, h: 300}, testTree())
	frag := iface(t, b, NodeRef{1, 1}, IfaceFragment)
	// A bad address for the result makes the method fault in Go, a runtime panic that the guard catches.
	hr := vcall(frag, IfaceFragment, "get_BoundingRectangle", 8)
	if hr != win32.EFail {
		t.Errorf("hr = %v, want E_FAIL", hr)
	}
	var rect win32.UiaRect
	if hr := vcall(frag, IfaceFragment, "get_BoundingRectangle", uintptr(unsafe.Pointer(&rect))); hr != win32.SOK {
		t.Errorf("the next call: %v", hr)
	}
}

// outs are the out parameters of the pattern getters.
type outs struct {
	i32 int32
	f64 float64
	ptr uintptr
	sa  uintptr
}

func TestPatternsThroughTheVtables(t *testing.T) {
	b, host := bridgeOf(t, testWindow{x: 100, y: 200, w: 400, h: 300}, testTree())
	act := func(node int, a ui.SemanticsAction, args any) []ui.SemanticsActionEvent {
		return []ui.SemanticsActionEvent{{ViewID: 1, NodeID: node, Type: a, Arguments: args}}
	}
	var o outs
	getI32 := func(obj win32.Object, i Interface, m string) int32 {
		t.Helper()
		o.i32 = -77
		if hr := vcall(obj, i, m, uintptr(unsafe.Pointer(&o.i32))); hr != win32.SOK {
			t.Errorf("%s::%s: %v", i.Name, m, hr)
		}
		return o.i32
	}
	getF64 := func(obj win32.Object, i Interface, m string) float64 {
		t.Helper()
		o.f64 = math.NaN()
		if hr := vcall(obj, i, m, uintptr(unsafe.Pointer(&o.f64))); hr != win32.SOK {
			t.Errorf("%s::%s: %v", i.Name, m, hr)
		}
		return o.f64
	}
	getBSTR := func(obj win32.Object, i Interface, m string) string {
		t.Helper()
		o.ptr = 0
		if hr := vcall(obj, i, m, uintptr(unsafe.Pointer(&o.ptr))); hr != win32.SOK {
			t.Errorf("%s::%s: %v", i.Name, m, hr)
		}
		s := win32.BSTRToString(o.ptr)
		win32.SysFreeString(o.ptr)
		return s
	}
	wantActions := func(name string, want []ui.SemanticsActionEvent) {
		t.Helper()
		if got := host.take(); !reflect.DeepEqual(got, want) {
			t.Errorf("%s: the framework was asked for %+v, want %+v", name, got, want)
		}
	}

	t.Run("Toggle", func(t *testing.T) {
		toggle := iface(t, b, NodeRef{1, 3}, IfaceToggle)
		if got := getI32(toggle, IfaceToggle, "get_ToggleState"); got != ToggleOn {
			t.Errorf("state = %d, want on", got)
		}
		if hr := vcall(toggle, IfaceToggle, "Toggle"); hr != win32.SOK {
			t.Errorf("Toggle: %v", hr)
		}
		wantActions("Toggle", act(3, ui.SemanticsActionTap, nil))
		// The button has no Toggle pattern.
		notToggle := iface(t, b, NodeRef{1, 1}, IfaceToggle)
		if hr := vcall(notToggle, IfaceToggle, "Toggle"); uint32(hr) != ErrNotSupported {
			t.Errorf("Toggle of a button: %v, want UIA_E_NOTSUPPORTED", hr)
		}
	})

	t.Run("Value", func(t *testing.T) {
		value := iface(t, b, NodeRef{1, 11}, IfaceValue)
		if got := getBSTR(value, IfaceValue, "get_Value"); got != "héllo wörld" {
			t.Errorf("value = %q", got)
		}
		o.i32 = -1
		if hr := vcall(value, IfaceValue, "get_IsReadOnly", uintptr(unsafe.Pointer(&o.i32))); hr != win32.SOK || o.i32 != 0 {
			t.Errorf("IsReadOnly = %d, %v", o.i32, hr)
		}
		text, _ := win32.UTF16PtrFromString("typed")
		if hr := vcall(value, IfaceValue, "SetValue", uintptr(unsafe.Pointer(text))); hr != win32.SOK {
			t.Errorf("SetValue: %v", hr)
		}
		// The field lacks the action to set its text: it is focused first, then the text is set when the
		// framework offers the action.
		wantActions("SetValue", act(11, ui.SemanticsActionFocus, nil))
		field := testTree().Nodes[len(testTree().Nodes)-1]
		field.Flags.IsFocused = ui.TristateIsTrue
		field.Actions |= ui.SemanticsActionSetText
		b.UpdateSemantics(1, semtree.Update(field))
		wantActions("SetValue after the field took the focus", act(11, ui.SemanticsActionSetText, "typed"))
		if hr := vcall(value, IfaceValue, "SetValue", 0); hr != win32.EPointer {
			t.Errorf("SetValue(nil) = %v, want E_POINTER", hr)
		}
	})

	t.Run("Scroll", func(t *testing.T) {
		scroll := iface(t, b, NodeRef{1, 5}, IfaceScroll)
		if hr := vcall(scroll, IfaceScroll, "Scroll", ScrollNoAmount, ScrollLargeIncrement); hr != win32.SOK {
			t.Errorf("Scroll: %v", hr)
		}
		wantActions("Scroll", act(5, ui.SemanticsActionScrollUp, nil))
		if hr := vcall(scroll, IfaceScroll, "Scroll", ScrollSmallIncrement, ScrollNoAmount); uint32(hr) != ErrNotSupported {
			t.Errorf("Scroll on the axis the list does not scroll: %v", hr)
		}
		if got := getF64(scroll, IfaceScroll, "get_VerticalScrollPercent"); got != 0 {
			t.Errorf("vertical percent = %v", got)
		}
		if got := getF64(scroll, IfaceScroll, "get_HorizontalScrollPercent"); got != NoScroll {
			t.Errorf("horizontal percent = %v, want UIA_ScrollPatternNoScroll", got)
		}
		if got := getF64(scroll, IfaceScroll, "get_VerticalViewSize"); math.Abs(got-100.0*100/(100+200)) > 1e-9 {
			t.Errorf("vertical view size = %v", got)
		}
		o.i32 = -1
		if hr := vcall(scroll, IfaceScroll, "get_VerticallyScrollable", uintptr(unsafe.Pointer(&o.i32))); hr != win32.SOK || o.i32 != 1 {
			t.Errorf("VerticallyScrollable = %d, %v", o.i32, hr)
		}
		if hr := vcall(scroll, IfaceScroll, "get_HorizontallyScrollable", uintptr(unsafe.Pointer(&o.i32))); hr != win32.SOK || o.i32 != 0 {
			t.Errorf("HorizontallyScrollable = %d, %v", o.i32, hr)
		}
	})

	t.Run("ExpandCollapse", func(t *testing.T) {
		ec := iface(t, b, NodeRef{1, 6}, IfaceExpandCollapse)
		if got := getI32(ec, IfaceExpandCollapse, "get_ExpandCollapseState"); got != ExpandCollapseCollapsed {
			t.Errorf("state = %d, want collapsed", got)
		}
		if hr := vcall(ec, IfaceExpandCollapse, "Expand"); hr != win32.SOK {
			t.Errorf("Expand: %v", hr)
		}
		wantActions("Expand", act(6, ui.SemanticsActionExpand, nil))
		if hr := vcall(ec, IfaceExpandCollapse, "Collapse"); uint32(hr) != ErrNotSupported {
			t.Errorf("Collapse of a collapsed node: %v, want UIA_E_NOTSUPPORTED", hr)
		}
	})

	t.Run("SelectionItem", func(t *testing.T) {
		radio := iface(t, b, NodeRef{1, 7}, IfaceSelectionItem)
		o.i32 = -1
		if hr := vcall(radio, IfaceSelectionItem, "get_IsSelected", uintptr(unsafe.Pointer(&o.i32))); hr != win32.SOK || o.i32 != 0 {
			t.Errorf("IsSelected = %d, %v", o.i32, hr)
		}
		if hr := vcall(radio, IfaceSelectionItem, "Select"); hr != win32.SOK {
			t.Errorf("Select: %v", hr)
		}
		wantActions("Select", act(7, ui.SemanticsActionTap, nil))
		chosen := iface(t, b, NodeRef{1, 9}, IfaceSelectionItem)
		if hr := vcall(chosen, IfaceSelectionItem, "Select"); hr != win32.SOK {
			t.Errorf("Select of a selected item: %v", hr)
		}
		wantActions("Select of a selected item", nil)
		other := iface(t, b, NodeRef{1, 10}, IfaceSelectionItem)
		o.ptr = 0
		if hr := vcall(other, IfaceSelectionItem, "get_SelectionContainer", uintptr(unsafe.Pointer(&o.ptr))); hr != win32.SOK || o.ptr != iface(t, b, NodeRef{1, 8}, IfaceSimple).Ptr() {
			t.Errorf("SelectionContainer = %#x, %v; want the group", o.ptr, hr)
		}
		win32.ObjectFromPtr(o.ptr).Release()
	})

	t.Run("Selection", func(t *testing.T) {
		sel := iface(t, b, NodeRef{1, 8}, IfaceSelection)
		o.sa = 0
		if hr := vcall(sel, IfaceSelection, "GetSelection", uintptr(unsafe.Pointer(&o.sa))); hr != win32.SOK || o.sa == 0 {
			t.Fatalf("GetSelection: %v", hr)
		}
		defer win32.SafeArrayDestroy(o.sa)
		ub, err := win32.SafeArrayGetUBound(o.sa)
		if err != nil || ub != 0 {
			t.Fatalf("the selection has upper bound %d, %v; want one element", ub, err)
		}
		var item uintptr
		if err := win32.SafeArrayGetElement(o.sa, 0, unsafe.Pointer(&item)); err != nil {
			t.Fatal(err)
		}
		if want := iface(t, b, NodeRef{1, 9}, IfaceSimple).Ptr(); item != want {
			t.Errorf("the selected item is %#x, want node 9 (%#x)", item, want)
		}
		win32.ObjectFromPtr(item).Release()
		o.i32 = -1
		if hr := vcall(sel, IfaceSelection, "get_CanSelectMultiple", uintptr(unsafe.Pointer(&o.i32))); hr != win32.SOK || o.i32 != 0 {
			t.Errorf("CanSelectMultiple = %d, %v", o.i32, hr)
		}
	})

	t.Run("RangeValue", func(t *testing.T) {
		rv := iface(t, b, NodeRef{1, 4}, IfaceRangeValue)
		for _, c := range []struct {
			m    string
			want float64
		}{{"get_Value", 50}, {"get_Minimum", 0}, {"get_Maximum", 100}, {"get_SmallChange", 1}, {"get_LargeChange", 10}} {
			if got := getF64(rv, IfaceRangeValue, c.m); got != c.want {
				t.Errorf("%s = %v, want %v", c.m, got, c.want)
			}
		}
		o.i32 = -1
		if hr := vcall(rv, IfaceRangeValue, "get_IsReadOnly", uintptr(unsafe.Pointer(&o.i32))); hr != win32.SOK || o.i32 != 0 {
			t.Errorf("IsReadOnly = %d, %v", o.i32, hr)
		}
	})

	t.Run("Fragment SetFocus and GetFocus", func(t *testing.T) {
		field := iface(t, b, NodeRef{1, 11}, IfaceFragment)
		if hr := vcall(field, IfaceFragment, "SetFocus"); hr != win32.SOK {
			t.Errorf("SetFocus: %v", hr)
		}
		wantActions("SetFocus", act(11, ui.SemanticsActionFocus, nil))
		if hr := vcall(iface(t, b, NodeRef{1, 2}, IfaceFragment), IfaceFragment, "SetFocus"); uint32(hr) != ErrNotSupported {
			t.Errorf("SetFocus of a label: %v, want UIA_E_NOTSUPPORTED", hr)
		}
		root := iface(t, b, NodeRef{1, FragmentRootID}, IfaceFragmentRoot)
		o.ptr = 0xDEAD
		if hr := vcall(root, IfaceFragmentRoot, "GetFocus", uintptr(unsafe.Pointer(&o.ptr))); hr != win32.SOK {
			t.Errorf("GetFocus: %v", hr)
		}
		// The field took the focus in the update above, and the window is active.
		if want := field.Ptr(); o.ptr != want {
			t.Errorf("GetFocus = %#x, want the text field (%#x)", o.ptr, want)
		}
		win32.ObjectFromPtr(o.ptr).Release()
	})
}

func TestTextPatternThroughTheVtables(t *testing.T) {
	b, host := bridgeOf(t, testWindow{x: 100, y: 200, w: 400, h: 300}, testTree())
	text := iface(t, b, NodeRef{1, 11}, IfaceText)
	var out uintptr
	docPtr := func() win32.Object {
		t.Helper()
		out = 0
		if hr := vcall(text, IfaceText, "get_DocumentRange", uintptr(unsafe.Pointer(&out))); hr != win32.SOK || out == 0 {
			t.Fatalf("DocumentRange: %v", hr)
		}
		o := win32.ObjectFromPtr(out)
		t.Cleanup(func() { o.Release() })
		return o
	}
	getText := func(r win32.Object, max int32) string {
		t.Helper()
		var b uintptr
		if hr := vcall(r, IfaceTextRange, "GetText", uintptr(max), uintptr(unsafe.Pointer(&b))); hr != win32.SOK {
			t.Fatalf("GetText: %v", hr)
		}
		s := win32.BSTRToString(b)
		win32.SysFreeString(b)
		return s
	}
	var sel int32
	if hr := vcall(text, IfaceText, "get_SupportedTextSelection", uintptr(unsafe.Pointer(&sel))); hr != win32.SOK || sel != SupportedTextSelectionSingle {
		t.Errorf("SupportedTextSelection = %d, %v", sel, hr)
	}
	doc := docPtr()
	if got := getText(doc, -1); got != "héllo wörld" {
		t.Errorf("document = %q", got)
	}
	if got := getText(doc, 4); got != "héll" {
		t.Errorf("first 4 units = %q", got)
	}
	// The selection: bytes 1 to 4 of "héllo wörld" are the units 1 to 3.
	var sa uintptr
	if hr := vcall(text, IfaceText, "GetSelection", uintptr(unsafe.Pointer(&sa))); hr != win32.SOK || sa == 0 {
		t.Fatalf("GetSelection: %v", hr)
	}
	var rp uintptr
	if err := win32.SafeArrayGetElement(sa, 0, unsafe.Pointer(&rp)); err != nil {
		t.Fatal(err)
	}
	selRange := win32.ObjectFromPtr(rp)
	if got := getText(selRange, -1); got != "él" {
		t.Errorf("selection = %q, want él", got)
	}
	selRange.Release()
	win32.SafeArrayDestroy(sa)

	// Clone, compare, endpoints.
	var clone uintptr
	if hr := vcall(doc, IfaceTextRange, "Clone", uintptr(unsafe.Pointer(&clone))); hr != win32.SOK || clone == 0 {
		t.Fatalf("Clone: %v", hr)
	}
	cl := win32.ObjectFromPtr(clone)
	defer cl.Release()
	var same int32
	if hr := vcall(doc, IfaceTextRange, "Compare", cl.Ptr(), uintptr(unsafe.Pointer(&same))); hr != win32.SOK || same != 1 {
		t.Errorf("Compare with the clone = %d, %v", same, hr)
	}
	var moved int32
	if hr := vcall(cl, IfaceTextRange, "ExpandToEnclosingUnit", TextUnitWord); hr != win32.SOK {
		t.Errorf("ExpandToEnclosingUnit: %v", hr)
	}
	if got := getText(cl, -1); got != "héllo wörld" {
		t.Errorf("the word range of a document that starts a word = %q", got)
	}
	if hr := vcall(cl, IfaceTextRange, "Move", TextUnitWord, 1, uintptr(unsafe.Pointer(&moved))); hr != win32.SOK {
		t.Errorf("Move: %v", hr)
	}
	// The clone is the whole first word now? Expand made it "héllo " first.
	if hr := vcall(doc, IfaceTextRange, "Compare", cl.Ptr(), uintptr(unsafe.Pointer(&same))); hr != win32.SOK {
		t.Errorf("Compare: %v", hr)
	}
	var cmp int32
	if hr := vcall(doc, IfaceTextRange, "CompareEndpoints", EndpointStart, cl.Ptr(), EndpointStart, uintptr(unsafe.Pointer(&cmp))); hr != win32.SOK {
		t.Errorf("CompareEndpoints: %v", hr)
	}
	// Move the start of the clone by one character, and select it.
	if hr := vcall(cl, IfaceTextRange, "MoveEndpointByUnit", EndpointStart, TextUnitCharacter, 1, uintptr(unsafe.Pointer(&moved))); hr != win32.SOK || moved != 1 {
		t.Errorf("MoveEndpointByUnit = %d, %v", moved, hr)
	}
	if hr := vcall(cl, IfaceTextRange, "Select"); hr != win32.SOK {
		t.Errorf("Select: %v", hr)
	}
	if got := host.take(); len(got) != 1 || got[0].Type != ui.SemanticsActionSetSelection || got[0].NodeID != 11 {
		t.Errorf("Select asked the framework for %+v, want a set-selection", got)
	}
	// FindText.
	needle, _ := win32.SysAllocString("wör")
	var found uintptr
	if hr := vcall(doc, IfaceTextRange, "FindText", needle, 0, 0, uintptr(unsafe.Pointer(&found))); hr != win32.SOK || found == 0 {
		t.Errorf("FindText: %v", hr)
	} else {
		fr := win32.ObjectFromPtr(found)
		if got := getText(fr, -1); got != "wör" {
			t.Errorf("found %q", got)
		}
		fr.Release()
	}
	win32.SysFreeString(needle)
	// The rectangle, the enclosing element, the children.
	var arr uintptr
	if hr := vcall(doc, IfaceTextRange, "GetBoundingRectangles", uintptr(unsafe.Pointer(&arr))); hr != win32.SOK || arr == 0 {
		t.Errorf("GetBoundingRectangles: %v", hr)
	} else {
		var v [4]float64
		for i := range v {
			if err := win32.SafeArrayGetElement(arr, int32(i), unsafe.Pointer(&v[i])); err != nil {
				t.Errorf("rectangle element %d: %v", i, err)
			}
		}
		win32.SafeArrayDestroy(arr)
		if v != [4]float64{110, 400, 200, 30} {
			t.Errorf("rectangle = %v, want the field's on the screen", v)
		}
	}
	var enc uintptr
	if hr := vcall(doc, IfaceTextRange, "GetEnclosingElement", uintptr(unsafe.Pointer(&enc))); hr != win32.SOK || enc != iface(t, b, NodeRef{1, 11}, IfaceSimple).Ptr() {
		t.Errorf("GetEnclosingElement = %#x, %v", enc, hr)
	}
	win32.ObjectFromPtr(enc).Release()
	var ch uintptr
	if hr := vcall(doc, IfaceTextRange, "GetChildren", uintptr(unsafe.Pointer(&ch))); hr != win32.SOK || ch == 0 {
		t.Errorf("GetChildren: %v", hr)
	}
	win32.SafeArrayDestroy(ch)
	// What is not done says so.
	var attr win32.Variant
	if hr := vcall(doc, IfaceTextRange, "GetAttributeValue", 40001, uintptr(unsafe.Pointer(&attr))); uint32(hr) != ErrNotSupported {
		t.Errorf("GetAttributeValue = %v, want UIA_E_NOTSUPPORTED", hr)
	}
	if hr := vcall(text, IfaceText, "RangeFromPoint", 0, 0, 0); uint32(hr) != ErrNotSupported {
		t.Errorf("RangeFromPoint = %v, want UIA_E_NOTSUPPORTED", hr)
	}
	// A range that is not ours is refused.
	if hr := vcall(doc, IfaceTextRange, "Compare", 0, uintptr(unsafe.Pointer(&same))); hr != win32.EInvalidArg {
		t.Errorf("Compare with nothing = %v, want E_INVALIDARG", hr)
	}
}

func TestPropertiesThroughGetPropertyValue(t *testing.T) {
	b, _ := bridgeOf(t, testWindow{x: 100, y: 200, w: 400, h: 300}, testTree())
	tests := []struct {
		name string
		ref  NodeRef
		prop PropertyID
		want Value
	}{
		{"the name of a button", NodeRef{1, 1}, PropName, String("Press")},
		{"its control type", NodeRef{1, 1}, PropControlType, Int(int(ControlTypeButton))},
		{"the control type of a check box", NodeRef{1, 3}, PropControlType, Int(int(ControlTypeCheckBox))},
		{"the control type of a radio button", NodeRef{1, 7}, PropControlType, Int(int(ControlTypeRadioButton))},
		{"the control type of a text field", NodeRef{1, 11}, PropControlType, Int(int(ControlTypeEdit))},
		{"the control type of a slider", NodeRef{1, 4}, PropControlType, Int(int(ControlTypeSlider))},
		{"is it enabled", NodeRef{1, 1}, PropIsEnabled, Bool(true)},
		{"is the field focusable", NodeRef{1, 11}, PropIsKeyboardFocusable, Bool(true)},
		{"does the button have the focus", NodeRef{1, 1}, PropHasKeyboardFocus, Bool(false)},
		{"the automation id", NodeRef{1, 1}, PropAutomationId, String("1")},
		{"the framework", NodeRef{1, 1}, PropFrameworkId, String("Glutter")},
		{"is it a control", NodeRef{1, 1}, PropIsControlElement, Bool(true)},
		{"the toggle state", NodeRef{1, 3}, PropToggleToggleState, Int(ToggleOn)},
		{"the value of the field", NodeRef{1, 11}, PropValueValue, String("héllo wörld")},
		{"the range value", NodeRef{1, 4}, PropRangeValueValue, Float(50)},
		{"the expand state", NodeRef{1, 6}, PropExpandCollapseExpandCollapseState, Int(ExpandCollapseCollapsed)},
		{"the selection item", NodeRef{1, 9}, PropSelectionItemIsSelected, Bool(true)},
		{"the scroll percent", NodeRef{1, 5}, PropScrollVerticalScrollPercent, Float(0)},
		{"a property it has no value for", NodeRef{1, 1}, PropFullDescription, Value{}},
		{"the fragment root says nothing of the name", NodeRef{1, FragmentRootID}, PropName, Value{}},
		{"the fragment root is a control", NodeRef{1, FragmentRootID}, PropIsControlElement, Bool(true)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			simple := iface(t, b, tt.ref, IfaceSimple)
			v := win32.Variant{Vt: 0xFFFF, Val: 0xFFFF}
			if hr := vcall(simple, IfaceSimple, "GetPropertyValue", uintptr(tt.prop), uintptr(unsafe.Pointer(&v))); hr != win32.SOK {
				t.Fatalf("hr = %v", hr)
			}
			defer win32.VariantClear(&v)
			var got Value
			switch v.Vt {
			case win32.VTBool:
				got = Bool(uint16(v.Val) != 0)
			case win32.VTI4:
				got = Int(int(int32(v.Val)))
			case win32.VTR8:
				got = Float(math.Float64frombits(v.Val))
			case win32.VTBSTR:
				got = String(win32.BSTRToString(uintptr(v.Val)))
			case win32.VTEmpty:
			default:
				t.Fatalf("variant type %d", v.Vt)
			}
			if got != tt.want {
				t.Errorf("value = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestEveryKindOfEventIsRaisedWithoutAnError(t *testing.T) {
	env, err := Shared()
	if err != nil {
		t.Fatal(err)
	}
	var logs bytes.Buffer
	b := NewBridge(env, testWindow{w: 400, h: 300}, slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(b.Close)
	host := &testHost{}
	b.Core().Start(host)
	b.Core().Ask(RootObjectID)
	b.SetWindow(1, "Test", true)
	b.UpdateSemantics(1, testTree())
	if !win32.UiaClientsAreListening() {
		t.Log("UiaClientsAreListening is false: the events below are not raised, and the test checks only the calls that are made")
	}
	nodes := testTree().Nodes
	field := nodes[len(nodes)-1]
	check := nodes[3]
	button := nodes[1]
	field.Flags.IsFocused = ui.TristateIsTrue // focus
	field.Value = "héllo wörld!"              // text, value property
	field.TextSelectionBase, field.TextSelectionExtent = 2, 2
	check.Flags.IsChecked = ui.CheckedStateIsFalse // toggle property
	button.Label = "Renamed"                       // name property
	live := nodes[2]
	live.Flags.IsLiveRegion = true
	live.Label = "Saved" // live region
	root := nodes[0]
	root.ChildrenInTraversalOrder = []int{1, 2, 3, 4, 5, 6, 7, 8}
	b.UpdateSemantics(1, semtree.Update(field, check, button, live, root))
	b.SetWindow(1, "Renamed window", true) // the name of the root
	b.SetWindow(1, "Renamed window", false)
	b.SetWindow(1, "Renamed window", true) // the focus again
	if strings.Contains(logs.String(), "was not raised") || strings.Contains(logs.String(), "no provider") {
		t.Errorf("an event failed:\n%s", logs.String())
	}
}
