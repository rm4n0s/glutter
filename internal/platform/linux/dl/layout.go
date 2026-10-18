package dl

import (
	"reflect"
	"sort"
	"testing"
)

// AssertLayout is a test helper for the layout test that every Go mirror of a C
// struct needs (SPEC §8.1 rule 3). It checks that the struct value got has the
// given size and the given field offsets, as taken from the C headers and
// recorded in the test. Every named field of got must be listed in offsets, so
// that a field added later cannot escape the test. Fields named "_" are
// padding and are skipped. It reports every difference with t.Errorf.
//
// Flutter: no counterpart
func AssertLayout(t testing.TB, name string, size uintptr, offsets map[string]uintptr, got any) {
	t.Helper()
	typ := reflect.TypeOf(got)
	if typ == nil || typ.Kind() != reflect.Struct {
		t.Errorf("%s: AssertLayout needs a struct value, got %T", name, got)
		return
	}
	if typ.Size() != size {
		t.Errorf("%s: size is %d, want %d (the C size)", name, typ.Size(), size)
	}
	fields := make(map[string]bool, typ.NumField())
	for i := range typ.NumField() {
		field := typ.Field(i)
		if field.Name == "_" {
			continue
		}
		fields[field.Name] = true
		want, listed := offsets[field.Name]
		switch {
		case !listed:
			t.Errorf("%s: field %s is not in the offsets map", name, field.Name)
		case field.Offset != want:
			t.Errorf("%s: offset of %s is %d, want %d (the C offset)", name, field.Name, field.Offset, want)
		}
	}
	var unknown []string
	for field := range offsets {
		if !fields[field] {
			unknown = append(unknown, field)
		}
	}
	sort.Strings(unknown)
	for _, field := range unknown {
		t.Errorf("%s: the offsets map lists %s, which the Go struct does not have", name, field)
	}
}
