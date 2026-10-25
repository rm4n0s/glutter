//go:build windows

package ole

import (
	"io"
	"log/slog"
	"sync"

	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// Env is the Host that holds the COM objects of drag and drop that Go makes (SPEC §25.2, §25.10), and
// the vtables of the four interface types, made once (the callbacks of a process are never freed and
// there are about 2,000 of them: three of IUnknown and one for each method). A process has one Env,
// Shared; what depends on a platform (the guard of the methods, the log) is the Runtime that each
// object is made with.
//
// Flutter: no counterpart
type Env struct {
	host *win32.Host

	target, source, data, enum *win32.InterfaceType
}

// Runtime is what the methods of an object need from the platform that made it.
//
// Flutter: no counterpart
type Runtime struct {
	// Log gets what cannot be returned to Windows; nil discards it.
	Log *slog.Logger
	// Guard runs the body of every method of the object: it keeps a panic from unwinding through
	// Windows' frames and answers the value of a method that panicked. Nil runs the body as it is.
	Guard func(func() uintptr) uintptr
}

// log is the Log, or a logger that discards.
func (r Runtime) log() *slog.Logger {
	if r.Log == nil {
		return slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return r.Log
}

// run calls fn under the Guard.
func (r Runtime) run(fn func() uintptr) uintptr {
	if r.Guard == nil {
		return fn()
	}
	return r.Guard(fn)
}

// object is implemented by the Go values behind the COM objects of this package.
type object interface{ runtime() Runtime }

var shared = sync.OnceValues(newEnv)

// Shared is the Env of the process, made on first use. It fails with COMInterfaceInvalid and
// Win32CallFailed, and keeps failing with the same error.
//
// Flutter: no counterpart
func Shared() (*Env, error) { return shared() }

// newEnv makes the Host and the interface types.
func newEnv() (*Env, error) {
	host, err := win32.NewHost(nil)
	if err != nil {
		return nil, err
	}
	e := &Env{host: host}
	if e.target, err = host.NewInterface(win32.InterfaceDef{Name: "IDropTarget", IID: win32.IIDDropTarget, Methods: e.targetMethods()}); err != nil {
		return nil, err
	}
	if e.source, err = host.NewInterface(win32.InterfaceDef{Name: "IDropSource", IID: win32.IIDDropSource, Methods: e.sourceMethods()}); err != nil {
		return nil, err
	}
	if e.data, err = host.NewInterface(win32.InterfaceDef{Name: "IDataObject", IID: win32.IIDDataObject, Methods: e.dataMethods()}); err != nil {
		return nil, err
	}
	if e.enum, err = host.NewInterface(win32.InterfaceDef{Name: "IEnumFORMATETC", IID: win32.IIDEnumFormatEtc, Methods: e.enumMethods()}); err != nil {
		return nil, err
	}
	return e, nil
}

// Stats counts the objects of the Env that are alive.
//
// Flutter: no counterpart
func (e *Env) Stats() win32.HostStats { return e.host.Stats() }

// method builds the definition of a method whose body runs under the guard of the object's Runtime.
func (e *Env) method(name string, args int, fn func(impl any, a []uintptr) uintptr) win32.MethodDef {
	return win32.MethodDef{Name: name, Args: args, Fn: func(impl any, a []uintptr) uintptr {
		return impl.(object).runtime().run(func() uintptr { return fn(impl, a) })
	}}
}

// pointOf is the POINTL that a method got in a register: x in the low half, y in the high half.
func pointOf(v uintptr) win32.Point {
	return win32.Point{X: int32(uint32(v)), Y: int32(uint32(v >> 32))}
}
