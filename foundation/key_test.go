package foundation

import (
	"testing"

	"github.com/rm4n0s/errors"
)

func TestKeyEquality(t *testing.T) {
	type thing struct{ n int }
	a, b := &thing{1}, &thing{1}
	u1, u2 := NewUniqueKey(), NewUniqueKey()
	tests := []struct {
		name string
		x, y Key
		want bool
	}{
		{"same string", NewKey("a"), NewKey("a"), true},
		{"other string", NewKey("a"), NewKey("b"), false},
		{"same int", NewValueKey(3), NewValueKey(3), true},
		{"int and string", NewValueKey(3), NewValueKey("3"), false},
		{"int and int64", NewValueKey(3), NewValueKey(int64(3)), false},
		{"unique with itself", u1, u1, true},
		{"unique with another", u1, u2, false},
		{"object same pointer", NewObjectKey(a), NewObjectKey(a), true},
		{"object other pointer", NewObjectKey(a), NewObjectKey(b), false},
		{"object and value key", NewObjectKey("a"), NewKey("a"), false},
		{"object with a plain value", NewObjectKey(5), NewObjectKey(5), true},
		{"object nil", NewObjectKey(nil), NewObjectKey(nil), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.x == tt.y; got != tt.want {
				t.Errorf("%v == %v is %v, want %v", tt.x, tt.y, got, tt.want)
			}
		})
	}
}

func TestKeysWorkAsMapKeys(t *testing.T) {
	m := map[Key]string{}
	m[NewKey("a")] = "first"
	m[NewKey("a")] = "second"
	m[NewValueKey(1)] = "int"
	u := NewUniqueKey()
	m[u] = "unique"
	if len(m) != 3 || m[NewKey("a")] != "second" || m[u] != "unique" {
		t.Errorf("map = %v", m)
	}
}

func TestKeyKinds(t *testing.T) {
	var _ LocalKey = NewKey("a")
	var _ LocalKey = NewValueKey(1)
	var _ LocalKey = NewObjectKey(1)
	var _ LocalKey = NewUniqueKey()
	if got := NewValueKey(3).Value(); got != 3 {
		t.Errorf("Value = %v", got)
	}
	if got := NewObjectKey(7).Value(); got != 7 {
		t.Errorf("Value = %v", got)
	}
}

func TestKeyString(t *testing.T) {
	type thing struct{}
	tests := []struct {
		name string
		key  Key
		want string
	}{
		{"string", NewKey("title"), "[<'title'>]"},
		{"int", NewValueKey(3), "[<3>]"},
		{"double", NewValueKey(2.0), "[<2.0>]"},
		{"bool", NewValueKey(true), "[<true>]"},
		{"object", NewObjectKey(&thing{}), "[thing#00000]"},
		{"unique", NewUniqueKey(), "[#00000]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ignoringHashCodes(tt.key.String()); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
	if a, b := NewUniqueKey().String(), NewUniqueKey().String(); a == b {
		t.Logf("two unique keys have the same short hash %s (possible, 1 in a million)", a)
	}
}

func TestObjectKeyNotComparable(t *testing.T) {
	tests := []struct {
		name  string
		value any
	}{
		{"slice", []int{1}},
		{"map", map[string]int{}},
		{"func", func() {}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := recoverError(t, func() { NewObjectKey(tt.value) })
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T", err)
			}
			if !appErr.HasRoute("NewObjectKey.ObjectKeyNotComparable") {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
		})
	}
}
