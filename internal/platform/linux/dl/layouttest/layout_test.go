package layouttest

import (
	"fmt"
	"strings"
	"testing"
	"unsafe"
)

// recorder is a testing.TB that records Errorf calls instead of failing.
type recorder struct {
	testing.TB
	errors []string
}

func (r *recorder) Helper() {}
func (r *recorder) Errorf(format string, args ...any) {
	r.errors = append(r.errors, strings.TrimSpace(fmt.Sprintf(format, args...)))
}

// sample mirrors a C struct { uint8_t a; uint32_t b; uint64_t c; void *d; }
// on a 64-bit platform.
type sample struct {
	A uint8
	B uint32
	C uint64
	D *byte
}

type padded struct {
	A uint8
	_ [3]byte
	B uint32
}

func TestAssertLayout(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip("the sample layouts assume 64-bit pointers")
	}
	good := map[string]uintptr{"A": 0, "B": 4, "C": 8, "D": 16}

	tests := []struct {
		name    string
		size    uintptr
		offsets map[string]uintptr
		got     any
		want    []string // substrings of the reported errors; none means the layout is right
	}{
		{"matching layout", 24, good, sample{}, nil},
		{"padding fields are skipped", 8, map[string]uintptr{"A": 0, "B": 4}, padded{}, nil},
		{"wrong size", 32, good, sample{}, []string{"size is 24, want 32"}},
		{"wrong offset", 24, map[string]uintptr{"A": 0, "B": 8, "C": 8, "D": 16}, sample{}, []string{"offset of B is 4, want 8"}},
		{"field missing from the map", 24, map[string]uintptr{"A": 0, "B": 4, "C": 8}, sample{}, []string{"field D is not in the offsets map"}},
		{"map lists an unknown field", 24, map[string]uintptr{"A": 0, "B": 4, "C": 8, "D": 16, "E": 24}, sample{}, []string{"lists E"}},
		{"not a struct", 8, nil, 42, []string{"needs a struct value"}},
		{"nil", 8, nil, nil, []string{"needs a struct value"}},
		{"several problems are all reported", 1, map[string]uintptr{"A": 1, "B": 4, "C": 8, "D": 16}, sample{}, []string{"size is 24", "offset of A is 0"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &recorder{TB: t}
			AssertLayout(r, "sample", tt.size, tt.offsets, tt.got)
			if len(tt.want) == 0 {
				if len(r.errors) != 0 {
					t.Errorf("unexpected errors: %q", r.errors)
				}
				return
			}
			joined := strings.Join(r.errors, "\n")
			for _, want := range tt.want {
				if !strings.Contains(joined, want) {
					t.Errorf("errors %q do not contain %q", r.errors, want)
				}
			}
		})
	}
}
