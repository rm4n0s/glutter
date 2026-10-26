package uintptrcheck

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
)

// write makes a module root with the files, and returns it.
func write(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, content := range files {
		path := filepath.Join(root, "win", name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

const header = `package win

import (
	"syscall"
	"unsafe"
)

// Call is marked.
//
//go:uintptrescapes
func Call(args ...uintptr) uintptr { r, _, _ := syscall.SyscallN(0, args...); return r }

type T struct{}

// call is marked too.
//
//go:uintptrescapes
func (t *T) call(name string, args ...uintptr) uintptr { return Call(args...) }

// plain is not marked.
func plain(args ...uintptr) uintptr { return 0 }
`

func TestCheck(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []string
	}{
		{name: "conversion in the arguments of a marked function", body: `
func A(x *int) uintptr { return Call(uintptr(unsafe.Pointer(x)), 1) }`},
		{name: "conversion in the arguments of a marked method", body: `
func A(t *T, x *int) uintptr { return t.call("n", uintptr(unsafe.Pointer(x))) }`},
		{name: "conversion in SyscallN", body: `
func A(x *int) { syscall.SyscallN(0, uintptr(unsafe.Pointer(x))) }`},
		{name: "arithmetic on the conversion in an argument", body: `
func A(x *int) uintptr { return Call((uintptr(unsafe.Pointer(x)) + 8), 2) }`},
		{name: "a variable that stays unsafe.Pointer", body: `
func A(x *int) uintptr {
	var p unsafe.Pointer
	if x != nil {
		p = unsafe.Pointer(x)
	}
	return Call(uintptr(p))
}`},
		{name: "conversion kept in a variable", body: `
func A(x *int) uintptr {
	p := uintptr(unsafe.Pointer(x))
	return Call(p)
}`, want: []string{"ptr-escape win/a.go:A"}},
		{name: "conversion chosen by an if", body: `
func (t *T) B(x []int) uintptr {
	var p uintptr
	if len(x) > 0 {
		p = uintptr(unsafe.Pointer(&x[0]))
	}
	return t.call("n", p)
}`, want: []string{"ptr-escape win/a.go:T.B"}},
		{name: "conversion in a struct field", body: `
type S struct{ P uintptr }

var v int

var s = S{P: uintptr(unsafe.Pointer(&v))}`, want: []string{"ptr-escape win/a.go:-"}},
		{name: "conversion in the arguments of an unmarked function", body: `
func C(x *int) uintptr { return plain(uintptr(unsafe.Pointer(x))) }`, want: []string{"ptr-escape win/a.go:C"}},
		{name: "conversion in a function literal inside a safe call", body: `
func D(x *int) uintptr {
	f := func() uintptr { return uintptr(unsafe.Pointer(x)) }
	return Call(f())
}`, want: []string{"ptr-escape win/a.go:D"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := write(t, map[string]string{"a.go": header + tt.body})
			vs, err := Check(root, []string{"win"})
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, v := range vs {
				got = append(got, v.Key())
			}
			sort.Strings(got)
			if strings.Join(got, "\n") != strings.Join(tt.want, "\n") {
				t.Errorf("violations = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRun(t *testing.T) {
	root := write(t, map[string]string{"a.go": header + `
func A(x *int) uintptr {
	p := uintptr(unsafe.Pointer(x))
	return Call(p)
}`})
	allow := filepath.Join(t.TempDir(), "allow")
	for _, tt := range []struct {
		name      string
		allow     string
		wantRoute string
		wantOut   string
	}{
		{name: "a violation", allow: "", wantRoute: "run.UintptrCheckViolations", wantOut: "[ptr-escape] A"},
		{name: "an accepted violation", allow: "ptr-escape win/a.go:A # reviewed\n", wantOut: "0 violations, 1 accepted"},
		{name: "a stale allow line", allow: "ptr-escape win/a.go:A\nptr-escape win/a.go:Gone\n", wantRoute: "run.UintptrCheckViolations", wantOut: "a violation that is gone"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := os.WriteFile(allow, []byte(tt.allow), 0o644); err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			err := run([]string{"-root", root, "-dirs", "win", "-allow", allow}, &out)
			if !strings.Contains(out.String(), tt.wantOut) {
				t.Errorf("output = %q, want it to contain %q", out.String(), tt.wantOut)
			}
			if tt.wantRoute == "" {
				if err != nil {
					t.Fatalf("run failed: %v", err)
				}
				return
			}
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T", err)
			}
			if !appErr.HasRoute(tt.wantRoute) {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
		})
	}
}

func TestRunErrors(t *testing.T) {
	root := write(t, map[string]string{"a.go": header, "bad.go": "package win\nfunc ("})
	good := write(t, map[string]string{"a.go": header})
	badAllow := filepath.Join(t.TempDir(), "allow")
	if err := os.WriteFile(badAllow, []byte("one two three\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name      string
		args      []string
		wantRoute string
	}{
		{name: "bad flags", args: []string{"-nope"}, wantRoute: "run.UintptrCheckBadFlags"},
		{name: "allow file missing", args: []string{"-root", good, "-dirs", "win", "-allow", filepath.Join(good, "none")}, wantRoute: "run->ReadAllow.UintptrCheckAllowUnreadable"},
		{name: "allow line malformed", args: []string{"-root", good, "-dirs", "win", "-allow", badAllow}, wantRoute: "run->ReadAllow.UintptrCheckAllowMalformed"},
		{name: "a file that does not parse", args: []string{"-root", root, "-dirs", "win"}, wantRoute: "run->Check.UintptrCheckParseFailed"},
		{name: "a directory that is missing", args: []string{"-root", good, "-dirs", "nowhere"}, wantRoute: "run->Check.UintptrCheckWalkFailed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			err := run(tt.args, &out)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if !appErr.HasRoute(tt.wantRoute) {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
		})
	}
}
