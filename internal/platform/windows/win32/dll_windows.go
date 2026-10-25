//go:build windows

package win32

import (
	"strings"
	"syscall"
	"unsafe"

	"github.com/rm4n0s/errors"
)

// loadLibrarySearchSystem32 is LOAD_LIBRARY_SEARCH_SYSTEM32: search the System32 directory only.
const loadLibrarySearchSystem32 = 0x00000800

// bootstrap is kernel32.dll, loaded the way Go loads it. kernel32 is a known DLL, which Windows
// takes from System32 whatever files lie around, so it is the one library that does not go through
// LoadSystemDLL: it holds LoadLibraryExW itself (SPEC §25.2). Its procedures are looked up when
// first called.
var (
	bootstrap          = syscall.NewLazyDLL("kernel32.dll")
	procLoadLibraryEx  = bootstrap.NewProc("LoadLibraryExW")
	procGetProcAddress = bootstrap.NewProc("GetProcAddress")
)

// Library is a DLL loaded from System32.
//
// Flutter: no counterpart
type Library struct {
	name   string
	handle uintptr
}

// Proc is a function of a Library, found with GetProcAddress.
//
// Flutter: no counterpart
type Proc struct {
	name string
	addr uintptr
}

// LoadSystemDLL loads a DLL with LoadLibraryExW and LOAD_LIBRARY_SEARCH_SYSTEM32, so that only
// System32 is searched: never the directory of the program, the current directory or PATH,
// whatever files are there (SPEC §25.2). name is a file name such as "user32.dll", with no
// directory. It fails with LibraryLoadFailed, with the metadata "name" and "code" (the Win32
// error), when the DLL is not there or cannot be loaded, and also for a name with a path in it.
//
// Flutter: no counterpart
func LoadSystemDLL(name string) (*Library, error) {
	if name == "" || strings.ContainsAny(name, `\/:`) {
		return nil, errors.New(TagLibraryLoadFailed, "the name of a system DLL is a file name with no path", "name", name)
	}
	p, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return nil, errors.NewErr(TagLibraryLoadFailed, err).SetMetadata("name", name)
	}
	h, _, errno := syscall.SyscallN(procLoadLibraryEx.Addr(), uintptr(unsafe.Pointer(p)), 0, loadLibrarySearchSystem32)
	if h == 0 {
		return nil, errors.NewErr(TagLibraryLoadFailed, errno).SetMetadata("name", name, "code", int(errno))
	}
	return &Library{name: name, handle: h}, nil
}

// Name is the file name the library was loaded by.
//
// Flutter: no counterpart
func (l *Library) Name() string { return l.name }

// Handle is the HMODULE of the library.
//
// Flutter: no counterpart
func (l *Library) Handle() uintptr { return l.handle }

// Proc finds a function by name (GetProcAddress). It fails with SymbolMissing, with the metadata
// "library" and "symbol", when the library does not export it.
//
// Flutter: no counterpart
func (l *Library) Proc(name string) (*Proc, error) {
	b, err := syscall.BytePtrFromString(name)
	if err != nil {
		return nil, errors.NewErr(TagSymbolMissing, err).SetMetadata("library", l.name, "symbol", name)
	}
	addr, _, errno := syscall.SyscallN(procGetProcAddress.Addr(), l.handle, uintptr(unsafe.Pointer(b)))
	if addr == 0 {
		return nil, errors.NewErr(TagSymbolMissing, errno).SetMetadata("library", l.name, "symbol", name, "code", int(errno))
	}
	return &Proc{name: name, addr: addr}, nil
}

// OptionalProc is Proc for a function that only newer Windows has (GetDpiForWindow, the dark
// title bar attribute): ok is false when the library does not export it, and the caller handles
// the absence.
//
// Flutter: no counterpart
func (l *Library) OptionalProc(name string) (p *Proc, ok bool) {
	b, err := syscall.BytePtrFromString(name)
	if err != nil {
		return nil, false
	}
	addr, _, _ := syscall.SyscallN(procGetProcAddress.Addr(), l.handle, uintptr(unsafe.Pointer(b)))
	if addr == 0 {
		return nil, false
	}
	return &Proc{name: name, addr: addr}, true
}

// Name is the name of the function.
//
// Flutter: no counterpart
func (p *Proc) Name() string { return p.name }

// Addr is the address of the function.
//
// Flutter: no counterpart
func (p *Proc) Addr() uintptr { return p.addr }

// Call calls the function with uintptr arguments (syscall.SyscallN) and returns the value of RAX
// and the last Win32 error (GetLastError) that the call left. A structure is passed by pointer,
// converted in the argument list (uintptr(unsafe.Pointer(&s))), which keeps it alive for the call; a
// structure of 8 bytes or less goes in the register as one uintptr. A function that takes a float
// or double by value cannot be called (SPEC §25.2).
//
// Flutter: no counterpart
//
//go:uintptrescapes
func (p *Proc) Call(args ...uintptr) (r uintptr, errno syscall.Errno) {
	r, _, errno = syscall.SyscallN(p.addr, args...)
	return r, errno
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	// TagLibraryLoadFailed is the tag of the error for a DLL that cannot be loaded from System32;
	// the metadata are "name" and "code".
	TagLibraryLoadFailed = "LibraryLoadFailed"
	// TagSymbolMissing is the tag of the error for a function that a library does not export;
	// the metadata are "library" and "symbol".
	TagSymbolMissing = "SymbolMissing"
)
