//go:build windows

package win32

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/rm4n0s/errors"
)

func TestLoadSystemDLLFindsTheSystemLibraries(t *testing.T) {
	tests := []struct {
		dll, proc string
	}{
		{"kernel32.dll", "GetCurrentProcessId"},
		{"user32.dll", "GetSystemMetrics"},
		{"ntdll.dll", "NtClose"},
		{"d3dcompiler_47.dll", "D3DCompile"},
	}
	for _, tt := range tests {
		t.Run(tt.dll, func(t *testing.T) {
			lib, err := LoadSystemDLL(tt.dll)
			if err != nil {
				t.Fatal(err)
			}
			if lib.Name() != tt.dll || lib.Handle() == 0 {
				t.Errorf("library = %q handle %#x", lib.Name(), lib.Handle())
			}
			proc, err := lib.Proc(tt.proc)
			if err != nil {
				t.Fatal(err)
			}
			if proc.Name() != tt.proc || proc.Addr() == 0 {
				t.Errorf("proc = %q at %#x", proc.Name(), proc.Addr())
			}
		})
	}
}

func TestProcCall(t *testing.T) {
	lib, err := LoadSystemDLL("kernel32.dll")
	if err != nil {
		t.Fatal(err)
	}
	proc, err := lib.Proc("GetCurrentProcessId")
	if err != nil {
		t.Fatal(err)
	}
	if pid, _ := proc.Call(); int(pid) != os.Getpid() {
		t.Errorf("GetCurrentProcessId = %d, want %d", pid, os.Getpid())
	}
	user32, err := LoadSystemDLL("user32.dll")
	if err != nil {
		t.Fatal(err)
	}
	metrics, err := user32.Proc("GetSystemMetrics")
	if err != nil {
		t.Fatal(err)
	}
	if w, _ := metrics.Call(0); w == 0 { // SM_CXSCREEN
		t.Errorf("GetSystemMetrics(SM_CXSCREEN) = 0")
	}
}

func TestLoadSystemDLLFailures(t *testing.T) {
	tests := []struct {
		name string
		dll  string
	}{
		{"a DLL that does not exist", "glutter_no_such_library.dll"},
		{"a path", `C:\Windows\System32\kernel32.dll`},
		{"a relative path", `.\kernel32.dll`},
		{"a path with a slash", "system32/kernel32.dll"},
		{"an empty name", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadSystemDLL(tt.dll)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T: %v", err, err)
			}
			if appErr.Tag != "LibraryLoadFailed" || !appErr.HasRoute("LoadSystemDLL.LibraryLoadFailed") {
				t.Errorf("got %s via %s, want LibraryLoadFailed via LoadSystemDLL", appErr.Tag, appErr.Route())
			}
			if appErr.Metadata["name"] != tt.dll {
				t.Errorf("metadata = %v, want the name", appErr.Metadata)
			}
		})
	}
}

func TestMissingSymbol(t *testing.T) {
	lib, err := LoadSystemDLL("kernel32.dll")
	if err != nil {
		t.Fatal(err)
	}
	_, err = lib.Proc("GlutterNoSuchFunction")
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T: %v", err, err)
	}
	if appErr.Tag != "SymbolMissing" || !appErr.HasRoute("Library.Proc.SymbolMissing") {
		t.Errorf("got %s via %s, want SymbolMissing via Library.Proc", appErr.Tag, appErr.Route())
	}
	if appErr.Metadata["library"] != "kernel32.dll" || appErr.Metadata["symbol"] != "GlutterNoSuchFunction" {
		t.Errorf("metadata = %v", appErr.Metadata)
	}
	if p, ok := lib.OptionalProc("GlutterNoSuchFunction"); ok || p != nil {
		t.Errorf("OptionalProc found a function that is not there: %v", p)
	}
	if p, ok := lib.OptionalProc("GetCurrentProcessId"); !ok || p == nil {
		t.Errorf("OptionalProc did not find GetCurrentProcessId")
	}
}

// TestLoadSystemDLLIgnoresPlantedLibraries plants a library in the current directory and next to the
// test binary, under a name that System32 does not have, and checks that LoadSystemDLL does not
// take it, while the default search of LoadLibrary does (SPEC §25.2).
func TestLoadSystemDLLIgnoresPlantedLibraries(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(self) // a program image can be loaded as a module
	if err != nil {
		t.Fatal(err)
	}
	const name = "glutter_planted.dll"
	tests := []struct {
		name string
		dir  func(t *testing.T) string
	}{
		{"the current directory", func(t *testing.T) string {
			dir := t.TempDir()
			t.Chdir(dir)
			return dir
		}},
		{"next to the program", func(t *testing.T) string { return filepath.Dir(self) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.dir(t)
			planted := filepath.Join(dir, name)
			if err := os.WriteFile(planted, data, 0o755); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Remove(planted) })
			if _, err := LoadSystemDLL(name); err == nil {
				t.Fatalf("LoadSystemDLL loaded %s from %s", name, dir)
			} else if appErr, ok := errors.FromError(err); !ok || appErr.Tag != "LibraryLoadFailed" {
				t.Errorf("got %v, want LibraryLoadFailed", err)
			}
			// The control: the default search does find it, so the planted file is a library
			// that could have been loaded.
			h, err := syscall.LoadLibrary(name)
			if err != nil {
				t.Logf("the control did not load the planted file by its bare name (%v); the check above is weaker on this system", err)
				return
			}
			_ = syscall.FreeLibrary(h)
		})
	}
}
