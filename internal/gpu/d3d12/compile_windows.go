//go:build windows

package d3d12

import (
	"path/filepath"
	"syscall"
	"unsafe"

	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// compilerDLLName is the library with D3DCompile and D3DReflect: part of Windows 10 and of Wine,
// loaded from System32 only (SPEC §9.8 "Loading").
const compilerDLLName = "d3dcompiler_47.dll"

// sourceName is the name D3DCompile puts before the line and column of its messages.
const sourceName = "shader.hlsl"

// CompilerDLL is d3dcompiler_47.dll, loaded: D3DCompile and D3DReflect, and the file version of the
// library (part of the cache key). It implements Backend.
//
// Flutter: no counterpart
type CompilerDLL struct {
	lib     *win32.Library
	compile *win32.Proc
	reflect *win32.Proc
	version string
}

// LoadCompilerDLL loads d3dcompiler_47.dll from System32 and finds D3DCompile and D3DReflect. It
// fails with the errors of win32: LibraryLoadFailed for a library that is not there and
// SymbolMissing for a function it lacks. The file version is read for the cache key; a library
// without a version resource has the empty version, and then nothing is cached.
//
// Flutter: no counterpart
func LoadCompilerDLL() (*CompilerDLL, error) {
	lib, err := win32.LoadSystemDLL(compilerDLLName)
	if err != nil {
		return nil, err
	}
	compile, err := lib.Proc("D3DCompile")
	if err != nil {
		return nil, err
	}
	reflect, err := lib.Proc("D3DReflect")
	if err != nil {
		return nil, err
	}
	d := &CompilerDLL{lib: lib, compile: compile, reflect: reflect}
	if dir, err := win32.SystemDirectory(); err == nil {
		if v, err := win32.FileVersionInfo(filepath.Join(dir, compilerDLLName)); err == nil {
			d.version = v.String()
		}
	}
	return d, nil
}

// Version implements Backend: the file version of d3dcompiler_47.dll, such as "10.0.19041.1".
//
// Flutter: no counterpart
func (d *CompilerDLL) Version() string { return d.version }

// Compile implements Backend with D3DCompile: the entry point of the HLSL for the target (vs_5_1,
// ps_5_1) and the flags. The messages are the text of the error blob (the compiler's lines, such as
// "shader.hlsl(3,5): error X3004: undeclared identifier 'x'"). It returns a negative HRESULT when
// D3DCompile fails, and E_INVALIDARG for an entry point or target with a NUL in it.
//
// Flutter: no counterpart
func (d *CompilerDLL) Compile(hlsl, entry, target string, flags uint32) (dxbc []byte, messages string, hresult int32) {
	pEntry, err := syscall.BytePtrFromString(entry)
	if err != nil {
		return nil, "the entry point has a NUL in it", int32(win32.EInvalidArg)
	}
	pTarget, err := syscall.BytePtrFromString(target)
	if err != nil {
		return nil, "the target has a NUL in it", int32(win32.EInvalidArg)
	}
	pName, err := syscall.BytePtrFromString(sourceName)
	if err != nil {
		return nil, "the source name has a NUL in it", int32(win32.EInvalidArg)
	}
	src := append([]byte(hlsl), 0) // the size passed is len(hlsl); the NUL keeps &src[0] valid for ""
	var codePtr, errsPtr uintptr
	r, _ := d.compile.Call(
		uintptr(unsafe.Pointer(&src[0])), uintptr(len(hlsl)), uintptr(unsafe.Pointer(pName)),
		0, 0, // no macros, no include handler
		uintptr(unsafe.Pointer(pEntry)), uintptr(unsafe.Pointer(pTarget)),
		uintptr(flags), 0,
		uintptr(unsafe.Pointer(&codePtr)), uintptr(unsafe.Pointer(&errsPtr)))
	code, errs := win32.ObjectFromPtr(codePtr), win32.ObjectFromPtr(errsPtr)
	if !errs.IsNil() {
		messages = blobText(errs)
		errs.Release()
	}
	hr := win32.FromResult(r)
	if !code.IsNil() {
		if hr.Succeeded() {
			dxbc = append([]byte(nil), blobBytes(code)...)
		}
		code.Release()
	}
	return dxbc, messages, int32(hr)
}

// blobBytes is the contents of an ID3DBlob, valid until the blob is released.
func blobBytes(b win32.Object) []byte {
	return win32.Bytes(b.Call(blobGetBufferPointer), int(b.Call(blobGetBufferSize)))
}

// blobText is the contents of an ID3DBlob of messages as text, without the NUL that ends it.
func blobText(b win32.Object) string {
	data := blobBytes(b)
	for len(data) > 0 && data[len(data)-1] == 0 {
		data = data[:len(data)-1]
	}
	return string(data)
}

var _ Backend = (*CompilerDLL)(nil)
