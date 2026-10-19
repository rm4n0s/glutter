package ui

import (
	"math"
	"testing"
)

func TestMatrix4DecomposeAndCompose(t *testing.T) {
	m := NewMatrix4Identity()
	m.Translate(6, -8, 2)
	m.RotateZ(0.7)
	m.Scale(0.5, 2, 5)
	translation, rotation, scale := m.Decompose()
	if translation != (Vector3{6, -8, 2}) {
		t.Errorf("translation = %v", translation)
	}
	if math.Abs(float64(scale.X)-0.5) > 1e-6 || math.Abs(float64(scale.Y)-2) > 1e-6 || math.Abs(float64(scale.Z)-5) > 1e-6 {
		t.Errorf("scale = %v", scale)
	}
	if math.Abs(rotation.Length()-1) > 1e-9 {
		t.Errorf("rotation %v is not a unit quaternion", rotation)
	}
	back := NewMatrix4Compose(translation, rotation, scale)
	for i := 0; i < 16; i++ {
		if math.Abs(float64(back.storage[i]-m.storage[i])) > 1e-5 {
			t.Fatalf("compose(decompose(m)) = %v, want %v", back.storage, m.storage)
		}
	}
}

func TestQuaternionFromRotationBranches(t *testing.T) {
	// A half turn around each axis exercises the branches with a trace <= 0.
	for name, m := range map[string]*Matrix4{
		"x": NewMatrix4RotationX(math.Pi), "y": NewMatrix4RotationY(math.Pi), "z": NewMatrix4RotationZ(math.Pi),
	} {
		_, q, _ := m.Decompose()
		if math.Abs(q.Length()-1) > 1e-6 {
			t.Errorf("%s: quaternion %v is not a unit", name, q)
		}
		back := NewMatrix4Compose(Vector3{}, q, Vector3{1, 1, 1})
		for i := 0; i < 16; i++ {
			if math.Abs(float64(back.storage[i]-m.storage[i])) > 1e-5 {
				t.Errorf("%s: compose(decompose(m)) = %v, want %v", name, back.storage, m.storage)
				break
			}
		}
	}
	q := NewQuaternionIdentity().Scaled(2).Add(Quaternion{X: 1})
	if got := q.Normalized().Length(); math.Abs(got-1) > 1e-12 {
		t.Errorf("normalized length = %v", got)
	}
	if (Quaternion{}).Normalized() != (Quaternion{}) {
		t.Error("the zero quaternion stays zero")
	}
}
