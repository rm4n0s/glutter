package ole

import (
	"reflect"
	"testing"
)

func TestFormatEnumNextSkipResetClone(t *testing.T) {
	e := NewFormatEnum([]uint32{10, 11, 12, 13, 14})
	steps := []struct {
		name string
		do   func() any
		want any
	}{
		{"two formats", func() any { return e.Next(2) }, []uint32{10, 11}},
		{"skip one", func() any { return e.Skip(1) }, true},
		{"the next one is the fourth", func() any { return e.Next(1) }, []uint32{13}},
		{"clone at the position", func() any { return e.Clone().Next(5) }, []uint32{14}},
		{"the clone did not move the original", func() any { return e.Next(5) }, []uint32{14}},
		{"at the end", func() any { return e.Next(1) }, []uint32(nil)},
		{"skip at the end", func() any { return e.Skip(1) }, false},
		{"reset", func() any { e.Reset(); return e.Next(1) }, []uint32{10}},
		{"skip past the end goes to the end", func() any { return e.Skip(99) }, false},
		{"nothing after that", func() any { return e.Next(1) }, []uint32(nil)},
		{"skip nothing", func() any { return e.Skip(0) }, true},
		{"a negative skip", func() any { return e.Skip(-1) }, false},
		{"zero formats", func() any { return e.Next(0) }, []uint32(nil)},
	}
	for _, s := range steps {
		if got := s.do(); !reflect.DeepEqual(got, s.want) {
			t.Errorf("%s: got %v, want %v", s.name, got, s.want)
		}
	}
}

func TestFormatEnumCloneMovesOnItsOwn(t *testing.T) {
	e := NewFormatEnum([]uint32{1, 2, 3})
	e.Next(1)
	c := e.Clone()
	if got := c.Next(1); !reflect.DeepEqual(got, []uint32{2}) {
		t.Errorf("clone Next = %v", got)
	}
	if got := e.Next(2); !reflect.DeepEqual(got, []uint32{2, 3}) {
		t.Errorf("original Next = %v", got)
	}
	// The enumerator works on a copy of the formats.
	formats := []uint32{5, 6}
	e2 := NewFormatEnum(formats)
	formats[0] = 99
	if got := e2.Next(1); got[0] != 5 {
		t.Errorf("the enumerator changed with the slice it was made from: %v", got)
	}
}
