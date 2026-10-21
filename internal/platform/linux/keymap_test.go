//go:build linux

package linux

import "testing"

func TestPhysicalKey(t *testing.T) {
	tests := []struct {
		name  string
		xkb   uint32
		want  uint64
		found bool
	}{
		{"Escape", 9, 0x70029, true},
		{"KeyA", 38, 0x70004, true},
		{"Enter", 36, 0x70028, true},
		{"Space", 65, 0x7002c, true},
		{"left Shift", 50, 0x700e1, true},
		{"keycode below the keyboard range", 0, 0, false},
		{"keycode that has no key", 4000, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := PhysicalKey(tt.xkb)
			if ok != tt.found || got != tt.want {
				t.Errorf("PhysicalKey(%d) = 0x%x, %v; want 0x%x, %v", tt.xkb, got, ok, tt.want, tt.found)
			}
		})
	}
}

func TestKeysymLogicalKey(t *testing.T) {
	tests := []struct {
		name  string
		sym   uint32
		want  uint64
		found bool
	}{
		{"Return", 0xff0d, 0x10000000d, true},
		{"Escape", 0xff1b, 0x10000001b, true},
		{"Shift_L", 0xffe1, 0x200000102, true},
		{"KP_1 is Numpad1", 0xffb1, 0x200000231, true},
		{"F1", 0xffbe, 0x100000801, true},
		{"a is a character, not in the table", 'a', 0, false},
		{"a keysym nobody maps", 0x1234567, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := KeysymLogicalKey(tt.sym)
			if ok != tt.found || got != tt.want {
				t.Errorf("KeysymLogicalKey(0x%x) = 0x%x, %v; want 0x%x, %v", tt.sym, got, ok, tt.want, tt.found)
			}
		})
	}
}

func TestLayoutGoals(t *testing.T) {
	goals := LayoutGoals()
	if len(goals) != 49 {
		t.Fatalf("%d layout goals, want the 49 of layout_goals.json", len(goals))
	}
	byKey := map[uint32]LayoutGoal{}
	for i, g := range goals {
		if i > 0 && goals[i-1].Keycode >= g.Keycode {
			t.Errorf("goals are not ordered by keycode at %d", i)
		}
		byKey[g.Keycode] = g
	}
	tests := []struct {
		name      string
		xkb       uint32
		logical   uint64
		mandatory bool
	}{
		{"KeyA", 38, 'a', true},
		{"Digit1", 10, '1', true},
		{"Comma", 59, ',', false},
		{"Space", 65, ' ', false},
		{"Backquote", 49, '`', false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, ok := byKey[tt.xkb]
			if !ok {
				t.Fatalf("no goal for keycode %d", tt.xkb)
			}
			if g.Default != tt.logical || g.Mandatory != tt.mandatory {
				t.Errorf("goal = %+v, want default 0x%x mandatory %v", g, tt.logical, tt.mandatory)
			}
		})
	}
}

func TestPlanes(t *testing.T) {
	// The planes agree with the table: Enter is in the "unprintable" plane 1, a GTK
	// keysym is in plane 0x15.
	if id, _ := KeysymLogicalKey(0xff0d); id>>32 != 1 {
		t.Errorf("Enter is in plane %d, want 1", id>>32)
	}
	if GTKPlane>>32 != 0x15 || UnicodePlane != 0 {
		t.Errorf("GTKPlane = 0x%x, UnicodePlane = 0x%x", GTKPlane, UnicodePlane)
	}
}
