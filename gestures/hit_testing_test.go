package gestures

import (
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

// Ported from packages/flutter/test/gestures/hit_test_test.dart.

type dummyHitTestTarget struct{}

func (dummyHitTestTarget) HandleEvent(PointerEvent, HitTestEntry) {}

func newDummyEntry() *HitTestEntryBase { return NewHitTestEntry(dummyHitTestTarget{}) }

func sameEntries(got []HitTestEntry, want ...HitTestEntry) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestWrappedHitTestResult(t *testing.T) {
	entry1, entry2, entry3 := newDummyEntry(), newDummyEntry(), newDummyEntry()
	transform := ui.NewMatrix4TranslationValues(40, 150, 0)
	wrapped := NewHitTestResult()
	wrapped.PushTransform(transform)
	wrapped.Add(entry1)
	if !sameEntries(wrapped.Path(), entry1) {
		t.Fatalf("wrapped path = %v", wrapped.Path())
	}
	if entry1.Transform() != transform && !entry1.Transform().Equals(transform) {
		t.Errorf("entry1 transform = %v, want %v", entry1.Transform().Storage(), transform.Storage())
	}
	wrapping := NewHitTestResultWrap(wrapped)
	if !sameEntries(wrapping.Path(), entry1) {
		t.Fatalf("wrapping path = %v", wrapping.Path())
	}
	wrapping.Add(entry2)
	if !sameEntries(wrapping.Path(), entry1, entry2) || !sameEntries(wrapped.Path(), entry1, entry2) {
		t.Fatalf("paths after adding to the wrapper: %v and %v", wrapping.Path(), wrapped.Path())
	}
	if !entry2.Transform().Equals(transform) {
		t.Errorf("entry2 transform = %v", entry2.Transform().Storage())
	}
	wrapped.Add(entry3)
	if !sameEntries(wrapping.Path(), entry1, entry2, entry3) || !sameEntries(wrapped.Path(), entry1, entry2, entry3) {
		t.Fatalf("paths after adding to the wrapped: %v and %v", wrapping.Path(), wrapped.Path())
	}
	if !entry3.Transform().Equals(transform) {
		t.Errorf("entry3 transform = %v", entry3.Transform().Storage())
	}
}

// matrixNear compares matrices with a tolerance: the storage is float32 (the
// Dart tests compare float64 values exactly), so a product that is grouped
// differently rounds differently.
func matrixNear(a, b *ui.Matrix4) bool {
	for i, v := range a.Storage() {
		if d := v - b.Storage()[i]; d > 1e-4 || d < -1e-4 {
			return false
		}
	}
	return true
}

func currentTransform(result *HitTestResult) *ui.Matrix4 {
	entry := newDummyEntry()
	result.Add(entry)
	return entry.Transform()
}

func TestHitTestResultPushAndPopTransforms(t *testing.T) {
	result := NewHitTestResult()
	m1 := ui.NewMatrix4TranslationValues(10, 20, 0)
	m2 := ui.NewMatrix4RotationZ(1)
	m3 := ui.NewMatrix4Diagonal3Values(1.1, 1.2, 1.0)
	mul := func(ms ...*ui.Matrix4) *ui.Matrix4 {
		r := ms[0]
		for _, m := range ms[1:] {
			r = r.Multiply(m)
		}
		return r
	}
	steps := []struct {
		name string
		do   func() *ui.Matrix4
		want *ui.Matrix4
	}{
		{"push m1", func() *ui.Matrix4 { result.PushTransform(m1); return currentTransform(result) }, m1},
		{"push m2", func() *ui.Matrix4 { result.PushTransform(m2); return currentTransform(result) }, mul(m2, m1)},
		{"repeated add", func() *ui.Matrix4 { return currentTransform(result) }, mul(m2, m1)},
		{"wrapped at [m1, m2]", func() *ui.Matrix4 { return currentTransform(NewHitTestResultWrap(result)) }, mul(m2, m1)},
		{"push m3", func() *ui.Matrix4 { result.PushTransform(m3); return currentTransform(result) }, mul(m3, m2, m1)},
		{"pop twice", func() *ui.Matrix4 { result.PopTransform(); result.PopTransform(); return currentTransform(result) }, m1},
		{"pop, push m3", func() *ui.Matrix4 { result.PopTransform(); result.PushTransform(m3); return currentTransform(result) }, m3},
		{"push m2", func() *ui.Matrix4 { result.PushTransform(m2); return currentTransform(result) }, mul(m2, m3)},
	}
	for _, s := range steps {
		if got := s.do(); !matrixNear(got, s.want) {
			t.Errorf("%s: got %v, want %v", s.name, got.Storage(), s.want.Storage())
		}
	}
}

func TestHitTestResultPushAndPopOffsets(t *testing.T) {
	result := NewHitTestResult()
	m1 := ui.NewMatrix4RotationZ(1)
	m2 := ui.NewMatrix4Diagonal3Values(1.1, 1.2, 1.0)
	o3 := ui.Offset{Dx: 10, Dy: 20}
	m3 := ui.NewMatrix4TranslationValues(10, 20, 0)
	mul := func(ms ...*ui.Matrix4) *ui.Matrix4 {
		r := ms[0]
		for _, m := range ms[1:] {
			r = r.Multiply(m)
		}
		return r
	}
	steps := []struct {
		name string
		do   func() *ui.Matrix4
		want *ui.Matrix4
	}{
		{"offset first", func() *ui.Matrix4 { result.PushOffset(o3); return currentTransform(result) }, m3},
		{"pop, offset, m1", func() *ui.Matrix4 {
			result.PopTransform()
			result.PushOffset(o3)
			result.PushTransform(m1)
			return currentTransform(result)
		}, mul(m1, m3)},
		{"repeated add", func() *ui.Matrix4 { return currentTransform(result) }, mul(m1, m3)},
		{"wrapped", func() *ui.Matrix4 { return currentTransform(NewHitTestResultWrap(result)) }, mul(m1, m3)},
		{"push m2", func() *ui.Matrix4 { result.PushTransform(m2); return currentTransform(result) }, mul(m2, m1, m3)},
		{"pop all", func() *ui.Matrix4 {
			result.PopTransform()
			result.PopTransform()
			result.PopTransform()
			return currentTransform(result)
		}, ui.NewMatrix4Identity()},
		{"m2, offset, m1", func() *ui.Matrix4 {
			result.PushTransform(m2)
			result.PushOffset(o3)
			result.PushTransform(m1)
			return currentTransform(result)
		}, mul(m1, m3, m2)},
		{"pop m1", func() *ui.Matrix4 { result.PopTransform(); return currentTransform(result) }, mul(m3, m2)},
	}
	for _, s := range steps {
		if got := s.do(); !matrixNear(got, s.want) {
			t.Errorf("%s: got %v, want %v", s.name, got.Storage(), s.want.Storage())
		}
	}
}

func TestHitTestResultString(t *testing.T) {
	r := NewHitTestResult()
	if got := r.String(); got != "HitTestResult(<empty path>)" {
		t.Errorf("String() = %q", got)
	}
	r.Add(newDummyEntry())
	if got := r.String(); len(got) < len("HitTestResult(HitTestEntryBase#") {
		t.Errorf("String() = %q", got)
	}
}
