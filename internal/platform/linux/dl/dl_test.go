package dl

import (
	"path/filepath"
	"strings"
	"testing"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/rm4n0s/errors"
)

func mustOpenLibc(t *testing.T) *Lib {
	t.Helper()
	lib, err := Open(libcName)
	if err != nil {
		t.Fatalf("Open(%q): %v", libcName, err)
	}
	return lib
}

func TestOpen(t *testing.T) {
	missing := "libglutter-does-not-exist.so.9"
	tests := []struct {
		name     string
		names    []string
		wantName string // loaded name; empty when Open must fail
	}{
		{"first name loads", []string{libcName}, libcName},
		{"falls back to the second name", []string{missing, libcName}, libcName},
		{"skips every failing name before the good one", []string{missing, missing + ".1", libcName}, libcName},
		{"no name loads", []string{missing, missing + ".1"}, ""},
		{"no names", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lib, err := Open(tt.names...)
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
			if strings.Join(names, ",") != strings.Join(tt.names, ",") {
				t.Errorf("Metadata[names] = %v, want %v", names, tt.names)
			}
			if _, ok := appErr.Metadata["dlerror"].(string); !ok {
				t.Errorf("Metadata[dlerror] = %v, want a string", appErr.Metadata["dlerror"])
			}
			if len(tt.names) > 0 && appErr.Metadata["dlerror"] == "" {
				t.Error("Metadata[dlerror] is empty, want the message from dlerror()")
			}
		})
	}
}

func TestOpenOverride(t *testing.T) {
	const env = "GLUTTER_TEST_LIBRARY_OVERRIDE"
	absMissing := filepath.Join(t.TempDir(), "libmissing.so")

	tests := []struct {
		name     string
		value    string
		wantName string // loaded name; empty when OpenOverride must fail
		wantTag  string
		wantRte  string
	}{
		{"unset uses the names", "", libcName, "", ""},
		{"override that exists wins over the names", "@libc", "@libc", "", ""},
		{"override that is missing fails and does not fall back", absMissing, "", "LibraryLoadFailed", "OpenOverride->Open.LibraryLoadFailed"},
		{"relative override is rejected", "lib/libc.so.6", "", "LibraryOverrideInvalid", "OpenOverride.LibraryOverrideInvalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := tt.value
			if value == "@libc" {
				// The absolute path of the real libc, found through an open handle's name.
				value = absolutePathOfLibc(t)
				tt.wantName = value
			}
			t.Setenv(env, value)

			lib, err := OpenOverride(env, libcName)
			if tt.wantName != "" {
				if err != nil {
					t.Fatalf("OpenOverride: %v", err)
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
			if appErr.Tag != tt.wantTag {
				t.Errorf("Tag = %q, want %q", appErr.Tag, tt.wantTag)
			}
			if !appErr.HasRoute(tt.wantRte) {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
		})
	}
}

// absolutePathOfLibc returns the absolute path of the loaded libc, taken from
// the dynamic linker's own record (dladdr) of where strlen lives.
func absolutePathOfLibc(t *testing.T) string {
	t.Helper()
	lib := mustOpenLibc(t)
	var dladdr func(addr uintptr, info uintptr) int32
	if err := lib.Bind(&dladdr, "dladdr"); err != nil {
		t.Skipf("dladdr is not available: %v", err)
	}
	fn, err := purego.Dlsym(lib.handle, "strlen")
	if err != nil {
		t.Fatal(err)
	}
	// Dl_info is { const char *dli_fname; void *dli_fbase; const char *dli_sname; void *dli_saddr; }.
	var info [4]uintptr
	if dladdr(fn, uintptr(unsafe.Pointer(&info[0]))) == 0 {
		t.Skip("dladdr did not resolve strlen")
	}
	path := GoString(info[0])
	if !filepath.IsAbs(path) {
		t.Skipf("dladdr returned %q, not an absolute path", path)
	}
	return path
}

func TestBindAndHas(t *testing.T) {
	lib := mustOpenLibc(t)

	t.Run("bind strlen and call it", func(t *testing.T) {
		var strlen func(s string) uintptr
		if err := lib.Bind(&strlen, "strlen"); err != nil {
			t.Fatalf("Bind: %v", err)
		}
		tests := []struct {
			in   string
			want uintptr
		}{{"", 0}, {"a", 1}, {"glutter", 7}, {"héllo", 6}}
		for _, tt := range tests {
			if got := strlen(tt.in); got != tt.want {
				t.Errorf("strlen(%q) = %d, want %d", tt.in, got, tt.want)
			}
		}
	})

	t.Run("missing symbol", func(t *testing.T) {
		var fn func()
		err := lib.Bind(&fn, "glutter_no_such_symbol")
		appErr, ok := errors.FromError(err)
		if !ok {
			t.Fatalf("expected *errors.Error, got %T", err)
		}
		if !appErr.HasRoute("Lib.Bind.SymbolMissing") {
			t.Errorf("unexpected failure path: %s", appErr.Route())
		}
		if appErr.Metadata["library"] != libcName || appErr.Metadata["symbol"] != "glutter_no_such_symbol" {
			t.Errorf("Metadata = %v, want library %q and symbol glutter_no_such_symbol", appErr.Metadata, libcName)
		}
	})

	t.Run("Has", func(t *testing.T) {
		tests := []struct {
			symbol string
			want   bool
		}{{"strlen", true}, {"malloc", true}, {"glutter_no_such_symbol", false}, {"", false}}
		for _, tt := range tests {
			if got := lib.Has(tt.symbol); got != tt.want {
				t.Errorf("Has(%q) = %v, want %v", tt.symbol, got, tt.want)
			}
		}
	})
}

func TestCStrings(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{"empty", ""},
		{"ascii", "wl_compositor"},
		{"unicode", "Glutter 🎨 — héllo"},
		{"long", strings.Repeat("x", 100000)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ptr := CString(tt.in)
			if ptr == 0 {
				t.Fatal("CString returned 0")
			}
			defer Free(ptr)
			if got := Strlen(ptr); got != uintptr(len(tt.in)) {
				t.Errorf("Strlen = %d, want %d", got, len(tt.in))
			}
			if got := GoString(ptr); got != tt.in {
				t.Errorf("GoString = %q, want %q", got, tt.in)
			}
		})
	}

	t.Run("GoString of 0 is empty", func(t *testing.T) {
		if got := GoString(0); got != "" {
			t.Errorf("GoString(0) = %q, want empty", got)
		}
	})
	t.Run("Free of 0 does nothing", func(t *testing.T) {
		Free(0)
	})
}

func TestMallocAndCalloc(t *testing.T) {
	ptr := Calloc(16, 4)
	if ptr == 0 {
		t.Fatal("Calloc returned 0")
	}
	defer Free(ptr)
	for i, b := range unsafe.Slice((*byte)(pointer(ptr)), 64) {
		if b != 0 {
			t.Fatalf("Calloc memory is not zeroed at byte %d: %d", i, b)
		}
	}

	m := Malloc(8)
	if m == 0 {
		t.Fatal("Malloc returned 0")
	}
	defer Free(m)
	buf := unsafe.Slice((*byte)(pointer(m)), 8)
	copy(buf, "abcdefg\x00")
	if got := GoString(m); got != "abcdefg" {
		t.Errorf("GoString after writing to Malloc memory = %q, want abcdefg", got)
	}
}

func TestPointerAndBytes(t *testing.T) {
	ptr := Calloc(1, 8)
	defer Free(ptr)

	view := Bytes(ptr, 8)
	if len(view) != 8 {
		t.Fatalf("len(Bytes) = %d, want 8", len(view))
	}
	copy(view, "abcdefg\x00")
	if got := GoString(ptr); got != "abcdefg" {
		t.Errorf("GoString after writing through Bytes = %q, want abcdefg", got)
	}
	*(*byte)(Pointer(ptr)) = 'X'
	if view[0] != 'X' {
		t.Errorf("a write through Pointer is not visible through Bytes: %q", view[0])
	}
	if got := Bytes(0, 8); got != nil {
		t.Errorf("Bytes(0, 8) = %v, want nil", got)
	}
	if got := Bytes(ptr, 0); got != nil {
		t.Errorf("Bytes(ptr, 0) = %v, want nil", got)
	}
}

func TestCallbackRegistryLimit(t *testing.T) {
	var registered int
	r := &callbackRegistry{limit: 3, register: func(any) uintptr {
		registered++
		return uintptr(registered)
	}}

	for i := 1; i <= 3; i++ {
		if got := r.add(func() {}); got != uintptr(i) {
			t.Errorf("callback %d: got pointer %d, want %d", i, got, i)
		}
	}
	if r.count != 3 {
		t.Errorf("count = %d, want 3", r.count)
	}

	defer func() {
		v := recover()
		err, ok := v.(error)
		if !ok {
			t.Fatalf("panic value = %v (%T), want an error", v, v)
		}
		appErr, ok := errors.FromError(err)
		if !ok {
			t.Fatalf("panic value is %T, want *errors.Error", err)
		}
		if appErr.Tag != "CallbackLimit" || !appErr.HasRoute("callbackRegistry.add.CallbackLimit") {
			t.Errorf("Tag = %q, route = %s, want CallbackLimit via callbackRegistry.add", appErr.Tag, appErr.Route())
		}
		if appErr.Metadata["limit"] != 3 {
			t.Errorf("Metadata[limit] = %v, want 3", appErr.Metadata["limit"])
		}
		if registered != 3 || r.count != 3 {
			t.Errorf("a callback was registered past the limit: registered=%d count=%d", registered, r.count)
		}
	}()
	r.add(func() {})
	t.Fatal("add past the limit did not panic")
}

func TestCallbackLimitIsBelowPuregos(t *testing.T) {
	if MaxCallbacks >= 2000 {
		t.Errorf("MaxCallbacks = %d, want a margin below purego's 2000", MaxCallbacks)
	}
}

// TestCallbackFromC sorts an array in C memory with libc's qsort, which calls
// a Go comparator through a callback created by Callback.
func TestCallbackFromC(t *testing.T) {
	lib := mustOpenLibc(t)
	var qsort func(base uintptr, n, size uintptr, cmp uintptr)
	if err := lib.Bind(&qsort, "qsort"); err != nil {
		t.Fatal(err)
	}

	var calls int
	before := CallbackCount()
	cmp := Callback(func(a, b uintptr) int32 {
		calls++
		x := *(*int32)(pointer(a))
		y := *(*int32)(pointer(b))
		switch {
		case x < y:
			return -1
		case x > y:
			return 1
		}
		return 0
	})
	if got := CallbackCount(); got != before+1 {
		t.Errorf("CallbackCount() = %d, want %d", got, before+1)
	}

	tests := []struct {
		name string
		in   []int32
		want []int32
	}{
		{"unsorted", []int32{5, 2, 9, -1, 0, 7}, []int32{-1, 0, 2, 5, 7, 9}},
		{"already sorted", []int32{1, 2, 3}, []int32{1, 2, 3}},
		{"duplicates", []int32{4, 4, 1, 4}, []int32{1, 4, 4, 4}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := Malloc(uintptr(len(tt.in)) * 4)
			defer Free(base)
			arr := unsafe.Slice((*int32)(pointer(base)), len(tt.in))
			copy(arr, tt.in)

			calls = 0
			qsort(base, uintptr(len(tt.in)), 4, cmp)

			for i := range tt.want {
				if arr[i] != tt.want[i] {
					t.Fatalf("sorted = %v, want %v", arr, tt.want)
				}
			}
			if len(tt.in) > 1 && calls == 0 {
				t.Error("the Go comparator was never called")
			}
		})
	}
}
