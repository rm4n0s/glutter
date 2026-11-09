//go:build darwin

package dl

import (
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
)

func TestFrameworkPath(t *testing.T) {
	tests := []struct{ name, want string }{
		{"AppKit", "/System/Library/Frameworks/AppKit.framework/AppKit"},
		{"Metal", "/System/Library/Frameworks/Metal.framework/Metal"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FrameworkPath(tt.name); got != tt.want {
				t.Errorf("FrameworkPath(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestOpen(t *testing.T) {
	missing := "/usr/lib/libglutter-does-not-exist.dylib"
	tests := []struct {
		name     string
		paths    []string
		wantName string // loaded path; empty when Open must fail
	}{
		{"the first path loads", []string{libcPath}, libcPath},
		{"falls back to the second path", []string{missing, libcPath}, libcPath},
		{"skips every failing path before the good one", []string{missing, missing + ".1", libcPath}, libcPath},
		{"no path loads", []string{missing, missing + ".1"}, ""},
		{"no paths", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lib, err := Open(tt.paths...)
			if tt.wantName != "" {
				if err != nil {
					t.Fatalf("Open: %v", err)
				}
				if lib.Name() != tt.wantName {
					t.Errorf("Name() = %q, want %q", lib.Name(), tt.wantName)
				}
				return
			}
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T", err)
			}
			if !appErr.HasRoute("Open.LibraryLoadFailed") {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
			names, _ := appErr.Metadata["names"].([]string)
			if strings.Join(names, ",") != strings.Join(tt.paths, ",") {
				t.Errorf("Metadata[names] = %v, want %v", names, tt.paths)
			}
			if got, ok := appErr.Metadata["dlerror"].(string); !ok || (len(tt.paths) > 0 && got == "") {
				t.Errorf("Metadata[dlerror] = %v, want the message of dlerror()", appErr.Metadata["dlerror"])
			}
		})
	}
}

func TestOpenFramework(t *testing.T) {
	t.Run("a framework that every Mac has", func(t *testing.T) {
		lib, err := OpenFramework("Foundation")
		if err != nil {
			t.Fatalf("OpenFramework(Foundation): %v", err)
		}
		if lib.Name() != FrameworkPath("Foundation") {
			t.Errorf("Name() = %q", lib.Name())
		}
		if !lib.Has("NSLog") {
			t.Error("Foundation has no NSLog")
		}
	})
	t.Run("a framework that does not exist", func(t *testing.T) {
		_, err := OpenFramework("GlutterNoSuchFramework")
		appErr, ok := errors.FromError(err)
		if !ok {
			t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
		}
		if !appErr.HasRoute("OpenFramework.MacOSFrameworkMissing") {
			t.Errorf("unexpected failure path: %s", appErr.Route())
		}
		if got, _ := appErr.Metadata["path"].(string); got != FrameworkPath("GlutterNoSuchFramework") {
			t.Errorf("Metadata[path] = %q", got)
		}
	})
}

func TestBindAndSymbol(t *testing.T) {
	lib, err := Open(libcPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("binds a function and calls it", func(t *testing.T) {
		var abs func(int32) int32
		if err := lib.Bind(&abs, "abs"); err != nil {
			t.Fatal(err)
		}
		if got := abs(-7); got != 7 {
			t.Errorf("abs(-7) = %d, want 7", got)
		}
	})
	t.Run("a missing symbol", func(t *testing.T) {
		var f func()
		err := lib.Bind(&f, "glutter_no_such_symbol")
		appErr, ok := errors.FromError(err)
		if !ok {
			t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
		}
		if !appErr.HasRoute("Lib.Bind.SymbolMissing") {
			t.Errorf("unexpected failure path: %s", appErr.Route())
		}
		if appErr.Metadata["symbol"] != "glutter_no_such_symbol" || appErr.Metadata["library"] != libcPath {
			t.Errorf("metadata = %v", appErr.Metadata)
		}
	})
	t.Run("Has and Symbol", func(t *testing.T) {
		if !lib.Has("malloc") || lib.Has("glutter_no_such_symbol") {
			t.Error("Has gave a wrong answer")
		}
		if addr, err := lib.Symbol("malloc"); err != nil || addr == 0 {
			t.Errorf("Symbol(malloc) = %#x, %v", addr, err)
		}
		_, err := lib.Symbol("glutter_no_such_symbol")
		if appErr, ok := errors.FromError(err); !ok || !appErr.HasRoute("Lib.Symbol.SymbolMissing") {
			t.Errorf("Symbol of a missing name: %v", err)
		}
	})
}

func TestCallbackLimit(t *testing.T) {
	registered := 0
	r := &callbackRegistry{limit: 3, register: func(any) uintptr { registered++; return uintptr(registered) }}
	for i := range 3 {
		if got := r.add(func() {}); got != uintptr(i+1) {
			t.Fatalf("callback %d = %d", i, got)
		}
	}
	defer func() {
		appErr, ok := errors.FromError(recover().(error))
		if !ok || appErr.Tag != tagCallbackLimit {
			t.Fatalf("want a panic with CallbackLimit, got %v", appErr)
		}
		if registered != 3 {
			t.Errorf("registered = %d, want 3 (the fourth must not reach purego)", registered)
		}
	}()
	r.add(func() {})
}

func TestReserveCallbacks(t *testing.T) {
	r := &callbackRegistry{limit: 5, register: func(any) uintptr { return 1 }}
	r.reserve(2)
	r.add(func() {})
	if r.count != 3 {
		t.Fatalf("count = %d, want 3", r.count)
	}
	func() {
		defer func() {
			appErr, ok := errors.FromError(recover().(error))
			if !ok || appErr.Tag != tagCallbackLimit {
				t.Fatalf("want a panic with CallbackLimit, got %v", appErr)
			}
		}()
		r.reserve(3) // 3 + 3 > 5
	}()
	if r.count != 3 {
		t.Errorf("a reservation that was refused counted %d callbacks, want 3", r.count)
	}
	r.reserve(2) // exactly the limit
	if r.count != 5 {
		t.Errorf("count = %d, want 5", r.count)
	}
}

func TestCallbackCountStaysUnderTheLimit(t *testing.T) {
	if got := CallbackCount(); got > MaxCallbacks {
		t.Errorf("CallbackCount = %d, over MaxCallbacks %d", got, MaxCallbacks)
	}
}

func TestLiveBlocks(t *testing.T) {
	tests := []struct {
		name  string
		alloc func() []uintptr
		want  int64
	}{
		{"nothing", func() []uintptr { return nil }, 0},
		{"malloc, calloc and a string", func() []uintptr { return []uintptr{Malloc(8), Calloc(2, 4), CString("hello")} }, 3},
		{"many strings", func() []uintptr {
			var out []uintptr
			for range 50 {
				out = append(out, CString("x"))
			}
			return out
		}, 50},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := LiveBlocks()
			ptrs := tt.alloc()
			if got := LiveBlocks() - before; got != tt.want {
				t.Fatalf("%d blocks alive after the allocations, want %d", got, tt.want)
			}
			for _, p := range ptrs {
				Free(p)
			}
			Free(0) // does nothing, and is not counted
			if got := LiveBlocks() - before; got != 0 {
				t.Errorf("%d blocks alive after Free, want 0", got)
			}
		})
	}
}

func TestCStrings(t *testing.T) {
	tests := []struct{ name, s, want string }{
		{"ascii", "hello", "hello"},
		{"empty", "", ""},
		{"utf-8", "héllo ☃", "héllo ☃"},
		{"a NUL cuts the string on the C side", "ab\x00cd", "ab"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := CString(tt.s)
			defer Free(p)
			if got := GoString(p); got != tt.want {
				t.Errorf("GoString(CString(%q)) = %q, want %q", tt.s, got, tt.want)
			}
			if got := Strlen(p); got != uintptr(len(tt.want)) {
				t.Errorf("Strlen = %d, want %d", got, len(tt.want))
			}
		})
	}
	if got := GoString(0); got != "" {
		t.Errorf("GoString(0) = %q", got)
	}
	if got := Bytes(0, 4); got != nil {
		t.Errorf("Bytes(0, 4) = %v", got)
	}
	p := Calloc(1, 4)
	defer Free(p)
	if b := Bytes(p, 4); len(b) != 4 || b[0]|b[1]|b[2]|b[3] != 0 {
		t.Errorf("Bytes of calloc'd memory = %v", b)
	}
}
