//go:build !windows

package layouttest

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
)

// fakeRun returns a Checker whose compiler is found and whose run gives a fixed answer. It records
// the command line and the C file.
func fakeRun(out string, code int, runErr error, seen *[]string, source *string) Checker {
	return Checker{
		Compiler: "fake-gcc",
		LookPath: func(name string) (string, error) { return "/bin/" + name, nil },
		Run: func(_ context.Context, argv []string) ([]byte, int, error) {
			*seen = argv
			if data, err := os.ReadFile(argv[len(argv)-1]); err == nil {
				*source = string(data)
			}
			return []byte(out), code, runErr
		},
	}
}

func TestCheckerReportsWhatTheCompilerSays(t *testing.T) {
	spec := Spec{Headers: []string{"windows.h"}, Structs: []Struct{{CName: "RECT", Size: 16}}}
	tests := []struct {
		name      string
		out       string
		code      int
		wantTag   string // empty: no error
		wantRoute string
		wantMeta  string // a key that the metadata must have
	}{
		{name: "the headers agree", code: 0},
		{name: "a size is wrong", out: "layout.c:6:1: error: static assertion failed: \"RECT: size\"\n", code: 1,
			wantTag: TagLayoutMismatch, wantRoute: "Checker.Check.LayoutMismatch", wantMeta: "failed"},
		{name: "a header is missing", out: "layout.c:2:10: fatal error: d3d12.h: No such file or directory\n", code: 1,
			wantTag: TagLayoutHeaderMissing, wantRoute: "Checker.Check.LayoutHeaderMissing", wantMeta: "header"},
		{name: "another compile error", out: "layout.c:7:1: error: unknown type name 'RECTX'\n", code: 1,
			wantTag: TagLayoutCompileFailed, wantRoute: "Checker.Check.LayoutCompileFailed", wantMeta: "output"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var argv []string
			var source string
			err := fakeRun(tt.out, tt.code, nil, &argv, &source).Check(context.Background(), spec)
			if !strings.Contains(source, "_Static_assert(sizeof(RECT) == 16") {
				t.Errorf("the compiler was given %q", source)
			}
			if got := strings.Join(argv[:3], " "); got != "/bin/fake-gcc -fsyntax-only -x" {
				t.Errorf("command = %q", got)
			}
			if tt.wantTag == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T: %v", err, err)
			}
			if appErr.Tag != tt.wantTag || !appErr.HasRoute(tt.wantRoute) {
				t.Errorf("got %s via %s, want %s via %s", appErr.Tag, appErr.Route(), tt.wantTag, tt.wantRoute)
			}
			if _, ok := appErr.Metadata[tt.wantMeta]; !ok {
				t.Errorf("metadata = %v, want the key %q", appErr.Metadata, tt.wantMeta)
			}
		})
	}
}

func TestCheckerMismatchNamesTheStructure(t *testing.T) {
	var argv []string
	var source string
	out := "layout.c:6:1: error: static assertion failed: \"WNDCLASSEXW.hIcon: offset\"\n"
	err := fakeRun(out, 1, nil, &argv, &source).Check(context.Background(), Spec{})
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T", err)
	}
	if !strings.Contains(appErr.Message, "WNDCLASSEXW.hIcon: offset") {
		t.Errorf("message = %q, want the name of the structure and the field", appErr.Message)
	}
}

func TestCheckerFailures(t *testing.T) {
	tests := []struct {
		name      string
		checker   Checker
		setup     func(t *testing.T)
		wantTag   string
		wantRoute string
		wantMeta  string
	}{
		{name: "the compiler is not installed",
			checker: Checker{LookPath: func(string) (string, error) { return "", exec.ErrNotFound }},
			wantTag: TagLayoutCompilerMissing, wantRoute: "Checker.Check.LayoutCompilerMissing", wantMeta: "install"},
		{name: "the compiler cannot be started",
			checker: Checker{LookPath: func(string) (string, error) { return "/nonexistent/compiler", nil }},
			wantTag: TagLayoutCompilerRunFailed, wantRoute: "Checker.Check->runCommand.LayoutCompilerRunFailed", wantMeta: "command"},
		{name: "the C file cannot be written",
			checker: Checker{LookPath: func(string) (string, error) { return "/bin/true", nil }},
			setup:   func(t *testing.T) { t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "missing")) },
			wantTag: TagLayoutFileFailed, wantRoute: "Checker.Check.LayoutFileFailed"},
		{name: "the error of a runner is returned as it is",
			checker: Checker{LookPath: func(string) (string, error) { return "/bin/true", nil },
				Run: func(context.Context, []string) ([]byte, int, error) {
					return nil, -1, errors.New(tagRunnerBroke, "the runner broke")
				}},
			wantTag: tagRunnerBroke},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(t)
			}
			err := tt.checker.Check(context.Background(), Spec{})
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T: %v", err, err)
			}
			if appErr.Tag != tt.wantTag {
				t.Errorf("tag = %s, want %s (route %s)", appErr.Tag, tt.wantTag, appErr.Route())
			}
			if tt.wantRoute != "" && !appErr.HasRoute(tt.wantRoute) {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
			if tt.wantMeta != "" {
				if _, ok := appErr.Metadata[tt.wantMeta]; !ok {
					t.Errorf("metadata = %v, want the key %q", appErr.Metadata, tt.wantMeta)
				}
			}
		})
	}
}

// fixtureHeader is a C header that stands for the Windows headers: a structure and a vtable.
const fixtureHeader = `#include <stdint.h>
typedef struct { int32_t left; int32_t top; void *name; uint16_t tip[8]; } FixtureRect;
typedef struct FixtureVtbl {
  void *(*QueryInterface)(void *, const void *, void **);
  uint32_t (*AddRef)(void *);
  uint32_t (*Release)(void *);
  int32_t (*Extra)(void *, int32_t);
} FixtureVtbl;
#define FIXTURE_FAIL 0x80004005U
`

type rightRect struct {
	Left int32     `c:"left"`
	Top  int32     `c:"top"`
	Name *uint16   `c:"name"`
	Tip  [8]uint16 `c:"tip"`
}

type wrongRect struct { // Left is wider than the header says, so Top moves
	Left int64     `c:"left"`
	Name *uint16   `c:"name"`
	Tip  [8]uint16 `c:"tip"`
	Top  int32     `c:"top"`
}

// TestHostCompilerChecksAFixture runs the real compiler of the host (gcc or cc, not mingw-w64:
// the fixture has no Windows types) over a fixture header, so the harness is checked without the
// Windows headers: the right mirror passes, the wrong ones fail and name the structure.
func TestHostCompilerChecksAFixture(t *testing.T) {
	compiler := ""
	for _, name := range []string{"gcc", "cc"} {
		if _, err := exec.LookPath(name); err == nil {
			compiler = name
			break
		}
	}
	if compiler == "" {
		t.Skip("no C compiler on this machine (sudo dnf install gcc)")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "fixture.h"), []byte(fixtureHeader), 0o600); err != nil {
		t.Fatal(err)
	}
	base := Spec{Headers: []string{"fixture.h"}, IncludeDirs: []string{dir}}
	with := func(f func(*Spec)) Spec {
		s := base
		f(&s)
		return s
	}
	vtable := VTable{CName: "FixtureVtbl", Methods: []Method{{"QueryInterface", 0}, {"AddRef", 1}, {"Release", 2}, {"Extra", 3}}}
	tests := []struct {
		name       string
		spec       Spec
		wantFailed []string // nil: the check passes
	}{
		{"the right mirror", with(func(s *Spec) {
			s.Structs = []Struct{StructOf("FixtureRect", rightRect{})}
			s.VTables = []VTable{vtable}
			s.Consts = []Const{{"FIXTURE_FAIL", 0x80004005}}
		}), nil},
		{"a wrong size and offset", with(func(s *Spec) { s.Structs = []Struct{StructOf("FixtureRect", wrongRect{})} }),
			[]string{"FixtureRect: size", "FixtureRect.left: size", "FixtureRect.top: offset"}},
		{"a wrong method index", with(func(s *Spec) {
			s.VTables = []VTable{{CName: "FixtureVtbl", Methods: []Method{{"QueryInterface", 0}, {"Release", 1}}}}
		}), []string{"FixtureVtbl.Release: index"}},
		{"a wrong constant", with(func(s *Spec) { s.Consts = []Const{{"FIXTURE_FAIL", 0x80004006}} }), []string{"FIXTURE_FAIL: value"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Checker{Compiler: compiler}.Check(context.Background(), tt.spec)
			if tt.wantFailed == nil {
				if err != nil {
					t.Fatalf("the right mirror failed: %v", err)
				}
				return
			}
			appErr, ok := errors.FromError(err)
			if !ok || appErr.Tag != TagLayoutMismatch {
				t.Fatalf("got %v, want LayoutMismatch", err)
			}
			failed, _ := appErr.Metadata["failed"].([]string)
			for _, want := range tt.wantFailed {
				found := false
				for _, f := range failed {
					found = found || f == want
				}
				if !found {
					t.Errorf("failed = %v, want %q among them", failed, want)
				}
			}
		})
	}
	t.Run("a member the header does not have", func(t *testing.T) {
		err := Checker{Compiler: compiler}.Check(context.Background(),
			with(func(s *Spec) {
				s.Structs = []Struct{{CName: "FixtureRect", Size: 32, Fields: []Field{{CName: "nothing", Size: 4}}}}
			}))
		if appErr, ok := errors.FromError(err); !ok || appErr.Tag != TagLayoutCompileFailed {
			t.Errorf("got %v, want LayoutCompileFailed", err)
		}
	})
	t.Run("a header that is not there", func(t *testing.T) {
		err := Checker{Compiler: compiler}.Check(context.Background(), Spec{Headers: []string{"nothing_here.h"}})
		if appErr, ok := errors.FromError(err); !ok || appErr.Tag != TagLayoutHeaderMissing {
			t.Errorf("got %v, want LayoutHeaderMissing", err)
		}
	})
}

// The tag of the error that a fake runner makes.
const tagRunnerBroke = "RunnerBroke"
