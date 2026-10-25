// Package win32 is the one package that holds the bindings and the helpers for calling Windows
// without cgo (SPEC §25.2): System32-only DLL loading, the procedure tables, UTF-16 strings, the
// HRESULT and GetLastError errors, the COM calls over a vtable, the COM objects that Go implements,
// the callbacks, and the first kernel32 bindings. It uses the standard library's syscall (SyscallN
// and NewCallback), never purego and never golang.org/x/sys/windows (D25).
//
// The Go mirrors of Win32 and COM structures, the GUID and the method indexes of the COM vtables
// are in files with no OS tag, so that their layout tests run on the Linux host against the
// mingw-w64 headers (package layouttest). The calls themselves are in *_windows.go files.
//
// Flutter: no counterpart (Flutter's Windows embedder calls Win32 from C++,
// engine/src/flutter/shell/platform/windows).
package win32
