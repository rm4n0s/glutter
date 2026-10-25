//go:build windows

package win32

import (
	"sync"
	"syscall"
)

// dllTable is the table of the functions of one System32 library: the library is loaded and each
// function is looked up the first time it is called, and kept (SPEC §25.2). The bindings of a
// library (user32_windows.go and the others) call through it, so a function that an old Windows
// lacks costs nothing until something calls it.
type dllTable struct {
	name string

	mu    sync.Mutex
	lib   *Library
	procs map[string]*Proc
}

// proc finds the function, loading the library first. It fails with LibraryLoadFailed or
// SymbolMissing.
func (t *dllTable) proc(name string) (*Proc, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if p, ok := t.procs[name]; ok {
		return p, nil
	}
	if t.lib == nil {
		lib, err := LoadSystemDLL(t.name)
		if err != nil {
			return nil, err
		}
		t.lib = lib
		t.procs = map[string]*Proc{}
	}
	p, err := t.lib.Proc(name)
	if err != nil {
		return nil, err
	}
	t.procs[name] = p
	return p, nil
}

// optional finds a function that only newer Windows has, and returns nil when the library or the
// function is not there.
func (t *dllTable) optional(name string) *Proc {
	t.mu.Lock()
	defer t.mu.Unlock()
	if p, ok := t.procs[name]; ok {
		return p
	}
	if t.lib == nil {
		lib, err := LoadSystemDLL(t.name)
		if err != nil {
			return nil
		}
		t.lib = lib
		t.procs = map[string]*Proc{}
	}
	p, ok := t.lib.OptionalProc(name)
	if !ok {
		return nil
	}
	t.procs[name] = p
	return p
}

// call calls the function with uintptr arguments (see Proc.Call). The pointers converted in the
// argument list stay alive for the call.
//
//go:uintptrescapes
func (t *dllTable) call(name string, args ...uintptr) (uintptr, syscall.Errno, error) {
	p, err := t.proc(name)
	if err != nil {
		return 0, 0, err
	}
	r, errno := p.Call(args...)
	return r, errno, nil
}

// boolInt is a Go bool as the BOOL of a Win32 function.
func boolInt(b bool) uintptr {
	if b {
		return 1
	}
	return 0
}

// View returns a pointer to memory that Windows owns, at addr: the lParam of a window message
// that points to a structure (WM_GETMINMAXINFO, WM_DPICHANGED), valid while the message is
// handled. T must be a mirror of the structure that addr holds.
//
// Flutter: no counterpart
func View[T any](addr uintptr) *T { return (*T)(at(addr)) }
