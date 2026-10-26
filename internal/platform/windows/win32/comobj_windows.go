//go:build windows

package win32

import (
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/rm4n0s/errors"
)

// COM objects made in Go (SPEC §25.2): IDropTarget, IDropSource, IDataObject, the UI Automation
// providers. Windows keeps a pointer to them, so each is memory from HeapAlloc, not Go memory.
//
//   - For each interface that an object implements there is a block of two words from HeapAlloc: a
//     pointer to the vtable of the interface type, and a handle (a number) that finds the Go object in
//     the Host's table. The pointer to the block is the interface pointer that Windows gets.
//   - The vtable of an interface type is also HeapAlloc memory, filled once, when the type is made,
//     with callbacks: the three methods of IUnknown (shared by every type) and one callback per
//     method of the type. A callback finds the object by the handle in the block it is called with.
//     No callback is made per object, so the number of callbacks is fixed at startup (CallbackLimit).
//   - QueryInterface answers by IID with the block of the interface asked for; AddRef and Release
//     count atomically for the whole object; the last Release frees the blocks and drops the object
//     from the table.

// blockSize is the size of the block of one interface of one object: the vtable pointer and the
// handle.
const blockSize = 2 * unsafe.Sizeof(uintptr(0))

// maxMethodArgs is the most arguments (besides this) that a method of a Go-made interface takes.
const maxMethodArgs = 6

// Result is the value that a method of a Go-made COM object returns for an HRESULT.
//
// Flutter: no counterpart
func Result(hr HRESULT) uintptr { return uintptr(uint32(hr)) }

// MethodDef describes one method of a COM interface that Go implements.
//
// Flutter: no counterpart
type MethodDef struct {
	// Name is the name of the method, for error messages.
	Name string
	// Args is the number of arguments after this, from 0 to 6. Every argument is one 64-bit
	// register: a structure of 8 bytes or less (POINTL) comes as one value, a larger one as a
	// pointer, and a floating-point argument is not supported here (SPEC §25.2).
	Args int
	// Fn runs the method on the object that the interface pointer belongs to (the impl given to
	// NewObject) and returns the value for RAX, usually Result(hr). It runs on the thread that
	// Windows called from, which is the UI thread for the COM objects of the window.
	Fn func(impl any, args []uintptr) uintptr
	// Raw, when it is not 0, is the address of the function that the vtable gets instead of a
	// callback made from Fn: a thunk in assembly that Windows calls with a floating-point argument
	// (SPEC §25.2), which hands the argument's bits to a callback of its own. Fn is then not used.
	Raw uintptr
}

// InterfaceDef describes a COM interface that Go implements: its IID and its own methods, in
// vtable order after the three methods of IUnknown.
//
// Flutter: no counterpart
type InterfaceDef struct {
	// Name is the name of the interface, such as "IDropTarget".
	Name string
	// IID is the interface ID.
	IID GUID
	// Methods are the methods after IUnknown's, in vtable order.
	Methods []MethodDef
}

// InterfaceType is an interface of a Host: its vtable, made once.
//
// Flutter: no counterpart
type InterfaceType struct {
	def  InterfaceDef
	vtbl uintptr
	host *Host
}

// Name is the name of the interface.
//
// Flutter: no counterpart
func (t *InterfaceType) Name() string { return t.def.Name }

// goObject is an object made in Go: the implementation, its interface types and their blocks.
type goObject struct {
	impl   any
	types  []*InterfaceType
	blocks []uintptr
	handle uint64
	refs   atomic.Int32
}

// HostStats counts the COM objects of a Host, for tests and diagnostics.
//
// Flutter: no counterpart
type HostStats struct {
	// Objects is the number of objects that are alive.
	Objects int
	// Blocks is the number of blocks made so far.
	Blocks int
	// FreedBlocks is the number of blocks given back to the heap.
	FreedBlocks int
}

// Host makes COM objects out of Go values. It owns the table that finds the Go object of an
// interface pointer by its handle, and the three callbacks of IUnknown. A process makes one Host at
// startup and shares it; a second one costs three more callbacks.
//
// Flutter: no counterpart
type Host struct {
	heap uintptr
	log  *slog.Logger

	queryInterface, addRef, release uintptr

	mu      sync.Mutex
	next    uint64
	objects map[uint64]*goObject
	// vtbls are the vtables of the interface types, to tell the pointer of one of this Host's objects
	// from any other (ImplOf).
	vtbls map[uintptr]bool

	blocks, freed atomic.Int64
}

// NewHost makes a Host: it finds the process heap and makes the callbacks of IUnknown. log gets
// the errors that cannot be returned because Windows called (a block that could not be freed); nil
// discards them.
//
// Flutter: no counterpart
func NewHost(log *slog.Logger) (*Host, error) {
	heap, err := ProcessHeap()
	if err != nil {
		return nil, err
	}
	if log == nil {
		log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	h := &Host{heap: heap, log: log, next: 1, objects: map[uint64]*goObject{}, vtbls: map[uintptr]bool{}}
	h.queryInterface = NewCallback(h.queryInterfaceMethod)
	h.addRef = NewCallback(h.addRefMethod)
	h.release = NewCallback(h.releaseMethod)
	return h, nil
}

// NewInterface makes the vtable of an interface type, with a callback for each of its methods. It
// fails with COMInterfaceInvalid for a method with more than six arguments or without a function,
// and with Win32CallFailed when the heap has no memory.
//
// Flutter: no counterpart
func (h *Host) NewInterface(def InterfaceDef) (*InterfaceType, error) {
	for _, m := range def.Methods {
		if (m.Fn == nil && m.Raw == 0) || m.Args < 0 || m.Args > maxMethodArgs {
			return nil, errors.New(tagCOMInterfaceInvalid, "a method of a COM interface needs a function and 0 to 6 arguments",
				"interface", def.Name, "method", m.Name, "args", m.Args)
		}
	}
	size := uintptr(3+len(def.Methods)) * unsafe.Sizeof(uintptr(0))
	vtbl, err := HeapAlloc(h.heap, HeapZeroMemory, size)
	if err != nil {
		return nil, err
	}
	entries := []uintptr{h.queryInterface, h.addRef, h.release}
	for _, m := range def.Methods {
		if m.Raw != 0 {
			entries = append(entries, m.Raw)
			continue
		}
		entries = append(entries, h.trampoline(m))
	}
	for i, e := range entries {
		storePtr(vtbl+uintptr(i)*unsafe.Sizeof(uintptr(0)), e)
	}
	h.mu.Lock()
	h.vtbls[vtbl] = true
	h.mu.Unlock()
	return &InterfaceType{def: def, vtbl: vtbl, host: h}, nil
}

// NewObject makes an object that implements the given interface types, with impl as the Go value
// that their methods get. The result is the interface pointer of the first type, with one
// reference; QueryInterface gives the others. It fails with COMInterfaceInvalid when there is no
// type or a type belongs to another Host, and with Win32CallFailed when the heap has no memory.
//
// Flutter: no counterpart
func (h *Host) NewObject(impl any, types ...*InterfaceType) (Object, error) {
	if len(types) == 0 {
		return Object{}, errors.New(tagCOMInterfaceInvalid, "a COM object needs at least one interface")
	}
	for _, t := range types {
		if t.host != h {
			return Object{}, errors.New(tagCOMInterfaceInvalid, "the interface type belongs to another Host", "interface", t.def.Name)
		}
	}
	o := &goObject{impl: impl, types: types}
	for _, t := range types {
		block, err := HeapAlloc(h.heap, HeapZeroMemory, blockSize)
		if err != nil {
			h.freeBlocks(o.blocks)
			return Object{}, err
		}
		h.blocks.Add(1)
		storePtr(block, t.vtbl)
		o.blocks = append(o.blocks, block)
	}
	o.refs.Store(1)
	h.mu.Lock()
	o.handle = h.next
	h.next++
	h.objects[o.handle] = o
	h.mu.Unlock()
	for _, block := range o.blocks {
		storePtr(block+unsafe.Sizeof(uintptr(0)), uintptr(o.handle))
	}
	return Object{ptr: o.blocks[0]}, nil
}

// ImplOf returns the Go value (the impl given to NewObject) behind an interface pointer, or nil when the
// pointer is not one of this Host's live objects: a pointer that a client passes back to a method (a text
// range given to Compare) may be a proxy or another program's object, so the vtable pointer is checked
// before the handle is trusted.
//
// Flutter: no counterpart
func (h *Host) ImplOf(ptr uintptr) any {
	if ptr == 0 {
		return nil
	}
	vtbl := loadPtr(ptr)
	h.mu.Lock()
	known := h.vtbls[vtbl]
	h.mu.Unlock()
	if !known {
		return nil
	}
	if o := h.find(ptr); o != nil {
		return o.impl
	}
	return nil
}

// Stats counts the objects and blocks of the Host.
//
// Flutter: no counterpart
func (h *Host) Stats() HostStats {
	h.mu.Lock()
	defer h.mu.Unlock()
	return HostStats{Objects: len(h.objects), Blocks: int(h.blocks.Load()), FreedBlocks: int(h.freed.Load())}
}

// find returns the object that an interface pointer (the address of a block) belongs to, or nil
// when it was released.
func (h *Host) find(this uintptr) *goObject {
	if this == 0 {
		return nil
	}
	handle := uint64(loadPtr(this + unsafe.Sizeof(uintptr(0))))
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.objects[handle]
}

// freeBlocks gives blocks back to the heap and counts them. A failure cannot be returned to
// Windows, so it is logged.
func (h *Host) freeBlocks(blocks []uintptr) {
	for _, b := range blocks {
		if err := HeapFree(h.heap, 0, b); err != nil {
			h.log.Error("freeing the block of a COM object failed", "err", err)
			continue
		}
		h.freed.Add(1)
	}
}

// queryInterfaceMethod is IUnknown::QueryInterface of every Go-made object.
func (h *Host) queryInterfaceMethod(this, riid, ppv uintptr) uintptr {
	if ppv == 0 {
		return Result(EPointer)
	}
	storePtr(ppv, 0)
	o := h.find(this)
	if riid == 0 || o == nil {
		return Result(ENoInterface)
	}
	iid := *(*GUID)(at(riid))
	if iid == IIDUnknown {
		o.refs.Add(1)
		storePtr(ppv, o.blocks[0])
		return Result(SOK)
	}
	for i, t := range o.types {
		if t.def.IID == iid {
			o.refs.Add(1)
			storePtr(ppv, o.blocks[i])
			return Result(SOK)
		}
	}
	return Result(ENoInterface)
}

// addRefMethod is IUnknown::AddRef of every Go-made object.
func (h *Host) addRefMethod(this uintptr) uintptr {
	o := h.find(this)
	if o == nil {
		return 0
	}
	return uintptr(o.refs.Add(1))
}

// releaseMethod is IUnknown::Release of every Go-made object. The last reference frees the blocks
// and drops the object from the table.
func (h *Host) releaseMethod(this uintptr) uintptr {
	o := h.find(this)
	if o == nil {
		return 0
	}
	n := o.refs.Add(-1)
	if n == 0 {
		h.mu.Lock()
		delete(h.objects, o.handle)
		h.mu.Unlock()
		h.freeBlocks(o.blocks)
	}
	return uintptr(max(n, 0))
}

// trampoline makes the callback of one method: it finds the object by the handle in this and calls
// the method's function with the arguments. A call on an object that was released returns E_FAIL.
func (h *Host) trampoline(m MethodDef) uintptr {
	call := func(this uintptr, args []uintptr) uintptr {
		o := h.find(this)
		if o == nil {
			return Result(EFail)
		}
		return m.Fn(o.impl, args)
	}
	switch m.Args {
	case 0:
		return NewCallback(func(this uintptr) uintptr { return call(this, nil) })
	case 1:
		return NewCallback(func(this, a1 uintptr) uintptr { return call(this, []uintptr{a1}) })
	case 2:
		return NewCallback(func(this, a1, a2 uintptr) uintptr { return call(this, []uintptr{a1, a2}) })
	case 3:
		return NewCallback(func(this, a1, a2, a3 uintptr) uintptr { return call(this, []uintptr{a1, a2, a3}) })
	case 4:
		return NewCallback(func(this, a1, a2, a3, a4 uintptr) uintptr { return call(this, []uintptr{a1, a2, a3, a4}) })
	case 5:
		return NewCallback(func(this, a1, a2, a3, a4, a5 uintptr) uintptr { return call(this, []uintptr{a1, a2, a3, a4, a5}) })
	default:
		return NewCallback(func(this, a1, a2, a3, a4, a5, a6 uintptr) uintptr {
			return call(this, []uintptr{a1, a2, a3, a4, a5, a6})
		})
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagCOMInterfaceInvalid = "COMInterfaceInvalid"
)
