//go:build windows

package uia

import (
	"fmt"
	"math"
	"runtime/debug"
	"sync"
	"unsafe"

	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// Env is the Host that holds the COM objects of the providers that Go makes (SPEC §25.2, §25.11), and the
// vtables of the interface types, made once (the callbacks of a process are never freed and there are
// about 2,000 of them: three of IUnknown and one for each method, 140 or so here). A process has one
// Env, Shared; what depends on a window (the model, the log) is the Bridge that each object is made with.
//
// Flutter: no counterpart
type Env struct {
	host  *win32.Host
	types map[string]*win32.InterfaceType

	// The interface types of each kind of object, in the order QueryInterface looks for them: the
	// first is the one the object is made with.
	elementTypes []*win32.InterfaceType
	rootTypes    []*win32.InterfaceType
	rangeTypes   []*win32.InterfaceType
}

var shared = sync.OnceValues(newEnv)

// Shared is the Env of the process, made on first use. It fails with COMInterfaceInvalid and
// Win32CallFailed, and keeps failing with the same error.
//
// Flutter: no counterpart
func Shared() (*Env, error) { return shared() }

// Stats counts the objects of the Env that are alive.
//
// Flutter: no counterpart
func (e *Env) Stats() win32.HostStats { return e.host.Stats() }

// newEnv makes the Host, the thunks and the interface types.
func newEnv() (*Env, error) {
	host, err := win32.NewHost(nil)
	if err != nil {
		return nil, err
	}
	e := &Env{host: host, types: map[string]*win32.InterfaceType{}}
	e.makeThunks()
	for _, iface := range Interfaces {
		t, err := host.NewInterface(win32.InterfaceDef{Name: iface.Name, IID: iface.IID, Methods: e.methodsOf(iface)})
		if err != nil {
			return nil, err
		}
		e.types[iface.Name] = t
	}
	pick := func(ifaces ...Interface) []*win32.InterfaceType {
		out := make([]*win32.InterfaceType, len(ifaces))
		for i, f := range ifaces {
			out[i] = e.types[f.Name]
		}
		return out
	}
	e.elementTypes = pick(IfaceSimple, IfaceFragment, IfaceInvoke, IfaceToggle, IfaceValue, IfaceRangeValue, IfaceScroll,
		IfaceExpandCollapse, IfaceSelection, IfaceSelectionItem, IfaceText)
	e.rootTypes = pick(IfaceSimple, IfaceFragment, IfaceFragmentRoot)
	e.rangeTypes = pick(IfaceTextRange)
	return e, nil
}

// makeThunks makes the callbacks that the assembly thunks jump to (thunk_windows.go).
func (e *Env) makeThunks() {
	thunkTargets[thunkIndexPoint] = win32.NewCallback(func(this, xbits, ybits, ret uintptr) uintptr {
		el, ok := e.host.ImplOf(this).(*elem)
		if !ok {
			return win32.Result(win32.EFail)
		}
		return el.guard("ElementProviderFromPoint", func() win32.HRESULT {
			return el.fromPoint(math.Float64frombits(uint64(xbits)), math.Float64frombits(uint64(ybits)), ret)
		})
	})
	thunkTargets[thunkIndexScroll] = win32.NewCallback(func(this, hbits, vbits, _ uintptr) uintptr {
		el, ok := e.host.ImplOf(this).(*elem)
		if !ok {
			return win32.Result(win32.EFail)
		}
		return el.guard("SetScrollPercent", func() win32.HRESULT {
			return el.setScrollPercent(math.Float64frombits(uint64(hbits)), math.Float64frombits(uint64(vbits)))
		})
	})
	thunkTargets[thunkIndexRange] = win32.NewCallback(func(this, vbits, _, _ uintptr) uintptr {
		el, ok := e.host.ImplOf(this).(*elem)
		if !ok {
			return win32.Result(win32.EFail)
		}
		return el.guard("SetValue", func() win32.HRESULT {
			return el.setRangeValue(math.Float64frombits(uint64(vbits)))
		})
	})
}

// ThunkEntries are the addresses of the three thunks, for the test that calls them with known doubles:
// ElementProviderFromPoint, SetScrollPercent and the range SetValue. They are the vtable entries of
// those methods.
//
// Flutter: no counterpart
func ThunkEntries() (point, scroll, rng uintptr) { return thunkEntries() }

// guard runs the body of a method. A panic in it must not unwind through Windows' frames: it is logged
// and the method answers E_FAIL, so a bug in a provider costs the client one answer and not the app.
func (el *elem) guard(method string, fn func() win32.HRESULT) (result uintptr) {
	defer func() {
		if v := recover(); v != nil {
			el.b.log.Error("a method of a UI Automation provider panicked", "method", method, "panic", fmt.Sprint(v), "stack", string(debug.Stack()))
			result = win32.Result(win32.EFail)
		}
	}()
	return win32.Result(fn())
}

// guardRange is guard for the objects of a text range.
func (r *rangeObj) guard(method string, fn func() win32.HRESULT) (result uintptr) {
	defer func() {
		if v := recover(); v != nil {
			r.b.log.Error("a method of a UI Automation text range panicked", "method", method, "panic", fmt.Sprint(v), "stack", string(debug.Stack()))
			result = win32.Result(win32.EFail)
		}
	}()
	return win32.Result(fn())
}

// method makes the definition of a method of an element: its body runs under the guard.
func elementMethod(name string, args int, fn func(el *elem, a []uintptr) win32.HRESULT) win32.MethodDef {
	return win32.MethodDef{Name: name, Args: args, Fn: func(impl any, a []uintptr) uintptr {
		el := impl.(*elem)
		return el.guard(name, func() win32.HRESULT { return fn(el, a) })
	}}
}

// rangeMethod makes the definition of a method of a text range.
func rangeMethod(name string, args int, fn func(r *rangeObj, a []uintptr) win32.HRESULT) win32.MethodDef {
	return win32.MethodDef{Name: name, Args: args, Fn: func(impl any, a []uintptr) uintptr {
		r := impl.(*rangeObj)
		return r.guard(name, func() win32.HRESULT { return fn(r, a) })
	}}
}

// methodsOf builds the methods of an interface in vtable order. A method with no function is left
// without one, and NewInterface fails with COMInterfaceInvalid, naming it.
func (e *Env) methodsOf(iface Interface) []win32.MethodDef {
	defs := elementMethods(e)[iface.Name]
	if iface.Name == IfaceTextRange.Name {
		defs = rangeMethods(e)
	}
	out := make([]win32.MethodDef, len(iface.Methods))
	for i, name := range iface.Methods {
		out[i] = win32.MethodDef{Name: name}
		if d, ok := defs[name]; ok {
			out[i] = d
		}
	}
	return out
}

// Small helpers for the out parameters of the methods.

// The HRESULTs of the providers, as win32.HRESULT values.
const (
	hrOK = win32.SOK
)

func hresult(code uint32) win32.HRESULT { return win32.HRESULT(int32(code)) }

// statusHR turns the outcome of a request into the HRESULT a client sees.
func statusHR(st Status) win32.HRESULT {
	switch st {
	case StatusOK:
		return hrOK
	case StatusGone:
		return hresult(ErrElementNotAvailable)
	case StatusDisabled:
		return hresult(ErrElementNotEnabled)
	case StatusNotSupported:
		return hresult(ErrNotSupported)
	}
	return win32.EInvalidArg
}

func outPtr(ptr, v uintptr) win32.HRESULT {
	if ptr == 0 {
		return win32.EPointer
	}
	*win32.View[uintptr](ptr) = v
	return hrOK
}

func outInt32(ptr uintptr, v int32) win32.HRESULT {
	if ptr == 0 {
		return win32.EPointer
	}
	*win32.View[int32](ptr) = v
	return hrOK
}

func outFloat(ptr uintptr, v float64) win32.HRESULT {
	if ptr == 0 {
		return win32.EPointer
	}
	*win32.View[float64](ptr) = v
	return hrOK
}

func outBool(ptr uintptr, v bool) win32.HRESULT {
	if ptr == 0 {
		return win32.EPointer
	}
	b := int32(0)
	if v {
		b = 1
	}
	*win32.View[int32](ptr) = b
	return hrOK
}

func outBSTR(ptr uintptr, s string) win32.HRESULT {
	if ptr == 0 {
		return win32.EPointer
	}
	*win32.View[uintptr](ptr) = 0
	bstr, err := win32.SysAllocString(s)
	if err != nil {
		return win32.EOutOfMemory
	}
	*win32.View[uintptr](ptr) = bstr
	return hrOK
}

// putVariant writes a Value into the VARIANT at ptr. A string needs a BSTR that the client frees with
// VariantClear.
func putVariant(ptr uintptr, v Value) win32.HRESULT {
	if ptr == 0 {
		return win32.EPointer
	}
	out := win32.View[win32.Variant](ptr)
	*out = win32.Variant{}
	return fillVariant(out, v)
}

func fillVariant(out *win32.Variant, v Value) win32.HRESULT {
	switch v.Kind {
	case ValueBool:
		out.Vt = win32.VTBool
		if v.B {
			out.Val = win32.VariantTrue
		}
	case ValueInt:
		out.Vt = win32.VTI4
		out.Val = uint64(uint32(v.I))
	case ValueFloat:
		out.Vt = win32.VTR8
		out.Val = math.Float64bits(v.F)
	case ValueString:
		bstr, err := win32.SysAllocString(v.S)
		if err != nil {
			return win32.EOutOfMemory
		}
		out.Vt = win32.VTBSTR
		out.Val = uint64(bstr)
	default:
		out.Vt = win32.VTEmpty
	}
	return hrOK
}

// int32Array makes a SAFEARRAY of the numbers (a runtime ID).
func int32Array(nums []int32) (uintptr, error) {
	sa, err := win32.SafeArrayCreateVector(win32.VTI4, 0, uint32(len(nums)))
	if err != nil {
		return 0, err
	}
	for i := range nums {
		if err := win32.SafeArrayPutElement(sa, int32(i), unsafe.Pointer(&nums[i])); err != nil {
			win32.SafeArrayDestroy(sa)
			return 0, err
		}
	}
	return sa, nil
}

// float64Array makes a SAFEARRAY of doubles (the rectangles of a text range).
func float64Array(nums []float64) (uintptr, error) {
	sa, err := win32.SafeArrayCreateVector(win32.VTR8, 0, uint32(len(nums)))
	if err != nil {
		return 0, err
	}
	for i := range nums {
		if err := win32.SafeArrayPutElement(sa, int32(i), unsafe.Pointer(&nums[i])); err != nil {
			win32.SafeArrayDestroy(sa)
			return 0, err
		}
	}
	return sa, nil
}

// unknownArray makes a SAFEARRAY of interface pointers; the array takes a reference of each.
func unknownArray(ptrs []uintptr) (uintptr, error) {
	sa, err := win32.SafeArrayCreateVector(win32.VTUnknown, 0, uint32(len(ptrs)))
	if err != nil {
		return 0, err
	}
	for i, p := range ptrs {
		if err := win32.SafeArrayPutElement(sa, int32(i), at(p)); err != nil {
			win32.SafeArrayDestroy(sa)
			return 0, err
		}
	}
	return sa, nil
}

// at turns an address that Windows owns into a pointer.
func at(addr uintptr) unsafe.Pointer { return *(*unsafe.Pointer)(unsafe.Pointer(&addr)) }
