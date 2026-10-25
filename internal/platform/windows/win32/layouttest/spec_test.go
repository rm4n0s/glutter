package layouttest

import (
	"fmt"
	"strings"
	"testing"
)

type sample struct {
	Left  int32   `c:"left"`
	Pad   [4]byte `c:"-"`
	Name  *uint16 `c:"name"`
	_     [4]byte
	Count uint32 `c:"count"`
}

func TestStructOf(t *testing.T) {
	got := StructOf("SAMPLE", sample{})
	want := Struct{CName: "SAMPLE", Size: 24, Fields: []Field{
		{CName: "left", Offset: 0, Size: 4},
		{CName: "name", Offset: 8, Size: 8},
		{CName: "count", Offset: 20, Size: 4},
	}}
	if got.CName != want.CName || got.Size != want.Size || fmt.Sprint(got.Fields) != fmt.Sprint(want.Fields) {
		t.Errorf("StructOf = %+v, want %+v", got, want)
	}
}

func TestStructOfPanicsOnMistakes(t *testing.T) {
	type untagged struct{ A int32 }
	tests := []struct {
		name string
		v    any
		want string
	}{
		{"a field without a c tag", untagged{}, "has no c tag"},
		{"not a struct", 42, "needs a struct value"},
		{"nil", nil, "needs a struct value"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if r == nil || !strings.Contains(fmt.Sprint(r), tt.want) {
					t.Errorf("recovered %v, want a panic containing %q", r, tt.want)
				}
			}()
			StructOf("X", tt.v)
		})
	}
}

func TestSource(t *testing.T) {
	spec := Spec{
		Headers: []string{"windows.h", "d3d12.h"},
		Structs: []Struct{{CName: "RECT", Size: 16, Fields: []Field{{CName: "left", Offset: 0, Size: 4}, {CName: "top", Offset: 4, Size: 4}}}},
		VTables: []VTable{{CName: "IUnknownVtbl", Methods: []Method{{Name: "QueryInterface", Index: 0}, {Name: "Release", Index: 2}}}},
		Consts:  []Const{{Name: "E_FAIL", Value: 0x80004005}},
	}
	want := `/* Written by internal/platform/windows/win32/layouttest. */
#include <windows.h>
#include <d3d12.h>
#include <stddef.h>

_Static_assert(sizeof(RECT) == 16, "RECT: size");
_Static_assert(offsetof(RECT, left) == 0, "RECT.left: offset");
_Static_assert(sizeof(((RECT *)0)->left) == 4, "RECT.left: size");
_Static_assert(offsetof(RECT, top) == 4, "RECT.top: offset");
_Static_assert(sizeof(((RECT *)0)->top) == 4, "RECT.top: size");
_Static_assert(offsetof(IUnknownVtbl, QueryInterface) / sizeof(void *) == 0, "IUnknownVtbl.QueryInterface: index");
_Static_assert(offsetof(IUnknownVtbl, Release) / sizeof(void *) == 2, "IUnknownVtbl.Release: index");
_Static_assert((unsigned int)(E_FAIL) == 0x80004005U, "E_FAIL: value");
`
	if got := spec.Source(); got != want {
		t.Errorf("Source =\n%s\nwant\n%s", got, want)
	}
}

func TestCommand(t *testing.T) {
	tests := []struct {
		name string
		spec Spec
		want string
	}{
		{"the defaults", Spec{}, "cc -fsyntax-only -x c -std=gnu11 -DCOBJMACROS -D_WIN32_WINNT=0x0A00 layout.c"},
		{"defines and include directories", Spec{Defines: []string{"INITGUID"}, IncludeDirs: []string{"/inc"}},
			"cc -fsyntax-only -x c -std=gnu11 -DCOBJMACROS -D_WIN32_WINNT=0x0A00 -DINITGUID -I/inc layout.c"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := strings.Join(Command("cc", tt.spec, "layout.c"), " "); got != tt.want {
				t.Errorf("Command = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseOutput(t *testing.T) {
	tests := []struct {
		name       string
		out        string
		wantFailed []string
		wantHeader string
	}{
		{"nothing", "", nil, ""},
		{"two assertions", "layout.c:5:1: error: static assertion failed: \"RECT: size\"\nlayout.c:6:1: error: static assertion failed: \"RECT.top: offset\"\n", []string{"RECT: size", "RECT.top: offset"}, ""},
		{"a missing header", "layout.c:2:10: fatal error: d3d13.h: No such file or directory\n", nil, "d3d13.h"},
		{"another error", "layout.c:5:1: error: unknown type name 'FOO'\n", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			failed, header := parseOutput([]byte(tt.out))
			if fmt.Sprint(failed) != fmt.Sprint(tt.wantFailed) || header != tt.wantHeader {
				t.Errorf("parseOutput = %v, %q; want %v, %q", failed, header, tt.wantFailed, tt.wantHeader)
			}
		})
	}
}
