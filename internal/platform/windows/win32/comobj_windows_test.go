//go:build windows

package win32

import (
	"testing"
	"unsafe"

	"github.com/rm4n0s/errors"
)

var (
	iidFoo = MustParseGUID("{6C0F2E41-3B1D-4D7E-9A57-0F1E2D3C4B5A}")
	iidBar = MustParseGUID("{6C0F2E42-3B1D-4D7E-9A57-0F1E2D3C4B5A}")
	iidNot = MustParseGUID("{6C0F2E43-3B1D-4D7E-9A57-0F1E2D3C4B5A}")
)

// The indexes of the test interfaces: the three methods of IUnknown, then their own.
const (
	fooAdd  = 3
	fooPing = 4
	fooSix  = 5
	barName = 3
)

// thing is the Go value behind the COM objects of these tests.
type thing struct{ value uintptr }

func newTestHost(t *testing.T) (host *Host, foo, bar *InterfaceType) {
	t.Helper()
	host, err := NewHost(nil)
	if err != nil {
		t.Fatal(err)
	}
	foo, err = host.NewInterface(InterfaceDef{Name: "IFoo", IID: iidFoo, Methods: []MethodDef{
		{Name: "Add", Args: 2, Fn: func(_ any, a []uintptr) uintptr { return a[0] + a[1] }},
		{Name: "Ping", Args: 0, Fn: func(impl any, _ []uintptr) uintptr { return impl.(*thing).value }},
		{Name: "Six", Args: 6, Fn: func(_ any, a []uintptr) uintptr { return a[0] + a[1] + a[2] + a[3] + a[4] + a[5] }},
	}})
	if err != nil {
		t.Fatal(err)
	}
	bar, err = host.NewInterface(InterfaceDef{Name: "IBar", IID: iidBar, Methods: []MethodDef{
		{Name: "Name", Args: 1, Fn: func(impl any, a []uintptr) uintptr {
			storePtr(a[0], impl.(*thing).value+1)
			return Result(SOK)
		}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	return host, foo, bar
}

func TestGoObjectIsCalledThroughItsVTable(t *testing.T) {
	host, foo, bar := newTestHost(t)
	obj, err := host.NewObject(&thing{value: 42}, foo, bar)
	if err != nil {
		t.Fatal(err)
	}
	if obj.IsNil() || obj.Ptr() == 0 {
		t.Fatal("the object is nil")
	}
	calls := []struct {
		name  string
		index int
		args  []uintptr
		want  uintptr
	}{
		{"two arguments", fooAdd, []uintptr{40, 2}, 42},
		{"no arguments, the Go value", fooPing, nil, 42},
		{"six arguments", fooSix, []uintptr{1, 2, 3, 4, 5, 6}, 21},
		{"a large argument", fooAdd, []uintptr{1 << 40, 1}, 1<<40 + 1},
	}
	for _, c := range calls {
		if got := obj.Call(c.index, c.args...); got != c.want {
			t.Errorf("%s: Call = %d, want %d", c.name, got, c.want)
		}
	}
	if obj.Release() != 0 {
		t.Errorf("the last Release did not reach zero")
	}
}

func TestGoObjectQueryInterfaceAndReferences(t *testing.T) {
	host, foo, bar := newTestHost(t)
	obj, err := host.NewObject(&thing{value: 7}, foo, bar)
	if err != nil {
		t.Fatal(err)
	}
	// The other interface of the object has its own block and the same Go value.
	other, err := obj.QueryInterface(&iidBar)
	if err != nil {
		t.Fatal(err)
	}
	if other.Ptr() == obj.Ptr() {
		t.Error("IBar has the block of IFoo")
	}
	var name uintptr
	if hr := FromResult(other.Call(barName, uintptr(unsafe.Pointer(&name)))); hr.Failed() || name != 8 {
		t.Errorf("IBar.Name = %v, %d; want S_OK, 8", hr, name)
	}
	// IUnknown is the first interface.
	identity, err := obj.QueryInterface(&IIDUnknown)
	if err != nil {
		t.Fatal(err)
	}
	if identity.Ptr() != obj.Ptr() {
		t.Errorf("IUnknown = %#x, want the block of IFoo %#x", identity.Ptr(), obj.Ptr())
	}
	// The same interface again.
	again, err := other.QueryInterface(&iidFoo)
	if err != nil {
		t.Fatal(err)
	}
	if again.Ptr() != obj.Ptr() {
		t.Errorf("IFoo = %#x, want %#x", again.Ptr(), obj.Ptr())
	}
	// 1 (the new object) + 3 queries.
	if n := obj.AddRef(); n != 5 {
		t.Errorf("AddRef = %d, want 5", n)
	}
	for want := uint32(4); want >= 1; want-- {
		if n := obj.Release(); n != want {
			t.Errorf("Release = %d, want %d", n, want)
		}
	}
	if s := host.Stats(); s.Objects != 1 || s.FreedBlocks != 0 {
		t.Errorf("the object is alive but Stats = %+v", s)
	}
	if n := other.Release(); n != 0 {
		t.Errorf("the last Release = %d, want 0", n)
	}
	if s := host.Stats(); s.Objects != 0 || s.Blocks != 2 || s.FreedBlocks != 2 {
		t.Errorf("after the last Release Stats = %+v, want no objects and both blocks freed", s)
	}
}

func TestGoObjectRefusesAnUnknownInterface(t *testing.T) {
	host, foo, _ := newTestHost(t)
	obj, err := host.NewObject(&thing{}, foo)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { obj.Release() })
	out, err := obj.QueryInterface(&iidNot)
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T: %v", err, err)
	}
	if appErr.Tag != "COMCallFailed" || !appErr.HasRoute("Object.QueryInterface->Object.CallHR->HResultError.COMCallFailed") {
		t.Errorf("got %s via %s", appErr.Tag, appErr.Route())
	}
	if appErr.Metadata["hresult"] != "0x80004002" || appErr.Metadata["method"] != "IUnknown::QueryInterface" {
		t.Errorf("metadata = %v", appErr.Metadata)
	}
	if !out.IsNil() {
		t.Errorf("the result is %#x, want nil", out.Ptr())
	}
	// A null out pointer is E_POINTER, and the answer is the same for a null IID.
	if hr := FromResult(obj.Call(IUnknownQueryInterface, uintptr(unsafe.Pointer(&iidFoo)), 0)); hr != EPointer {
		t.Errorf("QueryInterface with no out pointer = %v, want E_POINTER", hr)
	}
	var p uintptr = 99
	if hr := FromResult(obj.Call(IUnknownQueryInterface, 0, uintptr(unsafe.Pointer(&p)))); hr != ENoInterface || p != 0 {
		t.Errorf("QueryInterface with no IID = %v, out %d; want E_NOINTERFACE, 0", hr, p)
	}
}

func TestManyObjectsMakeNoCallbacks(t *testing.T) {
	host, foo, bar := newTestHost(t)
	before := CallbackCount()
	var objs []Object
	for range 200 {
		obj, err := host.NewObject(&thing{}, foo, bar)
		if err != nil {
			t.Fatal(err)
		}
		objs = append(objs, obj)
	}
	if after := CallbackCount(); after != before {
		t.Errorf("200 objects made %d callbacks, want none", after-before)
	}
	if s := host.Stats(); s.Objects != 200 {
		t.Errorf("Stats = %+v, want 200 objects", s)
	}
	for _, obj := range objs {
		obj.Release()
	}
	if s := host.Stats(); s.Objects != 0 || s.Blocks != s.FreedBlocks {
		t.Errorf("Stats = %+v, want no object and every block freed", s)
	}
}

func TestCallbackCountStaysUnderTheLimit(t *testing.T) {
	// The packages of this test binary have been initialized; the tests above made their hosts.
	newTestHost(t)
	if n := CallbackCount(); n <= 0 || n > CallbackLimit {
		t.Errorf("CallbackCount = %d, want between 1 and %d", n, CallbackLimit)
	}
}

func TestNewInterfaceAndNewObjectRefuseMistakes(t *testing.T) {
	host, foo, _ := newTestHost(t)
	other, _, _ := newTestHost(t)
	tests := []struct {
		name string
		call func() error
		path string
	}{
		{"a method with seven arguments", func() error {
			_, err := host.NewInterface(InterfaceDef{Name: "I", Methods: []MethodDef{{Name: "M", Args: 7, Fn: func(any, []uintptr) uintptr { return 0 }}}})
			return err
		}, "Host.NewInterface.COMInterfaceInvalid"},
		{"a method without a function", func() error {
			_, err := host.NewInterface(InterfaceDef{Name: "I", Methods: []MethodDef{{Name: "M"}}})
			return err
		}, "Host.NewInterface.COMInterfaceInvalid"},
		{"an object with no interface", func() error { _, err := host.NewObject(&thing{}); return err }, "Host.NewObject.COMInterfaceInvalid"},
		{"an interface of another host", func() error { _, err := other.NewObject(&thing{}, foo); return err }, "Host.NewObject.COMInterfaceInvalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appErr, ok := errors.FromError(tt.call())
			if !ok {
				t.Fatal("expected *errors.Error")
			}
			if appErr.Tag != "COMInterfaceInvalid" || !appErr.HasRoute(tt.path) {
				t.Errorf("got %s via %s, want %s", appErr.Tag, appErr.Route(), tt.path)
			}
		})
	}
}

func TestMethodOfANilObjectPanics(t *testing.T) {
	defer func() {
		err, _ := recover().(error)
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != "COMNilObject" || !appErr.HasRoute("Object.Call.COMNilObject") {
			t.Errorf("recovered %v, want COMNilObject via Object.Call", err)
		}
	}()
	Object{}.Call(0)
}
