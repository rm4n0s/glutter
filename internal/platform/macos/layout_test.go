package macos

import (
	"testing"
)

// layoutOf makes the lookup of a layout from one of the tables of layouts_test.go.
func layoutOf(table []uint32) func(uint16, bool) layoutClue {
	return func(keyCode uint16, shift bool) layoutClue {
		i := int(keyCode) * 2
		if shift {
			i++
		}
		if i >= len(table) {
			return layoutClue{}
		}
		v := table[i]
		return layoutClue{character: v & 0xffff, dead: v&0x10000 != 0}
	}
}

// TestLogicalKeysOfLayouts is Flutter's CorrectLogicalKeyForLayouts: the logical key of a key down on the
// US, French, Russian and Khmer layouts.
//
// Flutter: FlutterKeyboardManagerTest.mm testCorrectLogicalKeyForLayouts
func TestLogicalKeysOfLayouts(t *testing.T) {
	const (
		kvA, kvT, kvM, kvQ        = 0x00, 0x11, 0x2e, 0x0c
		kv1, kv2                  = 0x12, 0x13
		kvMinus, kvLeftBracket    = 0x1b, 0x21
		kvSemicolon               = 0x29
		a, tt, m, q               = 0x61, 0x74, 0x6d, 0x71
		digit1, digit2, bracketLt = 0x31, 0x32, 0x5b
	)
	type tap struct {
		keyCode          uint16
		chars, unmodChar string
		logical          uint64
	}
	tests := []struct {
		name   string
		layout []uint32
		taps   []tap
	}{
		{"US", usLayout, []tap{
			{kvA, "a", "a", a},
			{kvA, "A", "A", a},
			{kvA, "å", "a", a},
			{kvT, "t", "t", tt},
			{kv1, "1", "1", digit1},
			{kv1, "!", "!", digit1},
			{kvMinus, "-", "-", '-'},
			{kvMinus, "=", "=", '='},
		}},
		{"French", frenchLayout, []tap{
			{kvA, "q", "q", q},
			{kvA, "Q", "Q", q},
			{kvSemicolon, "m", "m", m},
			{kvM, ",", ",", ','},
			{kv1, "&", "&", digit1},
			{kv1, "1", "1", digit1},
			{kvMinus, ")", ")", ')'},
			{kvMinus, "°", "°", 0xb0},
		}},
		{"Russian", russianLayout, []tap{
			{kvA, "ф", "ф", a},
			{kv1, "1", "1", digit1},
			{kvLeftBracket, "х", "х", bracketLt},
		}},
		// https://github.com/flutter/flutter/issues/108729
		{"Khmer", khmerLayout, []tap{
			{kv2, "២", "២", digit2},
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r, rec := newKeyRecorder()
			r.layout = buildLayout(layoutGoals, layoutOf(tc.layout))
			for i, tp := range tc.taps {
				r.handle(evKey(keyKindDown, 0x100, tp.chars, tp.unmodChar, false, tp.keyCode))
				r.handle(evKey(keyKindUp, 0x100, tp.chars, tp.unmodChar, false, tp.keyCode))
				if len(rec.events) != 2 {
					t.Fatalf("tap %d: %d events, want a down and an up", i, len(rec.events))
				}
				down, up := rec.events[0], rec.events[1]
				rec.events = nil
				if down.Logical != tp.logical || up.Logical != tp.logical {
					t.Errorf("tap %d (key code %#x, %q): logical key %#x (up %#x), want %#x", i, tp.keyCode, tp.chars, down.Logical, up.Logical, tp.logical)
				}
				if down.Character != tp.chars {
					t.Errorf("tap %d: character %q, want %q", i, down.Character, tp.chars)
				}
			}
		})
	}
}

func TestBuildLayout(t *testing.T) {
	goals := []layoutGoal{
		{keyCode: 0x00, logical: 'a', mandatory: true},
		{keyCode: 0x01, logical: 's', mandatory: true},
		{keyCode: 0x1b, logical: '-', mandatory: false},
		{keyCode: 0x2b, logical: ',', mandatory: false},
	}
	// clue makes a layout out of a map: key code to the two characters, a negative character is a dead key.
	clue := func(m map[uint16][2]int32) func(uint16, bool) layoutClue {
		return func(k uint16, shift bool) layoutClue {
			i := 0
			if shift {
				i = 1
			}
			c := m[k][i]
			if c < 0 {
				return layoutClue{character: uint32(-c), dead: true}
			}
			return layoutClue{character: uint32(c)}
		}
	}
	tests := []struct {
		name   string
		layout map[uint16][2]int32
		want   map[uint16]uint64
	}{
		{"the US layout needs nothing of the goals but the mandatory ones that match", map[uint16][2]int32{
			0x00: {'a', 'A'}, 0x01: {'s', 'S'}, 0x1b: {'-', '_'}, 0x2b: {',', '<'},
		}, map[uint16]uint64{0x00: 'a', 0x01: 's'}},
		{"a mandatory character that is on another key goes to that key", map[uint16][2]int32{
			0x00: {'q', 'Q'}, 0x01: {'s', 'S'}, 0x02: {'a', 'A'},
		}, map[uint16]uint64{0x01: 's', 0x02: 'a'}},
		{"a mandatory character that no key makes goes to its key of the US layout", map[uint16][2]int32{
			0x00: {0x444, 0x424}, 0x01: {0x44b, 0x42b},
		}, map[uint16]uint64{0x00: 'a', 0x01: 's'}},
		{"a key of an alphabet that is not Latin gets the key of the US layout", map[uint16][2]int32{
			0x00: {'a', 'A'}, 0x01: {'s', 'S'}, 0x1b: {0x430, 0x410},
		}, map[uint16]uint64{0x00: 'a', 0x01: 's', 0x1b: '-'}},
		{"a key that has one Latin character keeps it", map[uint16][2]int32{
			0x00: {'a', 'A'}, 0x01: {'s', 'S'}, 0x2b: {0x430, ','},
		}, map[uint16]uint64{0x00: 'a', 0x01: 's'}},
		{"a dead key does not count as Latin", map[uint16][2]int32{
			0x00: {'a', 'A'}, 0x01: {'s', 'S'}, 0x2b: {-0x5e, -0xa8},
		}, map[uint16]uint64{0x00: 'a', 0x01: 's', 0x2b: ','}},
		{"a dead key does not meet a goal", map[uint16][2]int32{
			0x00: {-'a', 'A'}, 0x01: {'s', 'S'},
		}, map[uint16]uint64{0x00: 'a', 0x01: 's'}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildLayout(goals, clue(tt.layout))
			if len(got) != len(tt.want) {
				t.Fatalf("layout = %#v, want %#v", got, tt.want)
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("key code %#x: %#x, want %#x", k, got[k], v)
				}
			}
		})
	}
}

// TestLayoutGoalsOfTheTable checks the generated goals against what a layout needs: every digit and every
// letter is mandatory, and no two goals are for one key.
func TestLayoutGoalsOfTheTable(t *testing.T) {
	seen := map[uint16]bool{}
	mandatory := 0
	for _, g := range layoutGoals {
		if seen[g.keyCode] {
			t.Errorf("key code %#x has two goals", g.keyCode)
		}
		seen[g.keyCode] = true
		if g.keyCode > lastTypewriterKeyCode {
			t.Errorf("goal %+v is for a key after the typewriter keys", g)
		}
		isAlnum := g.logical >= '0' && g.logical <= '9' || g.logical >= 'a' && g.logical <= 'z'
		if g.mandatory != isAlnum {
			t.Errorf("goal %+v: mandatory = %v, want %v", g, g.mandatory, isAlnum)
		}
		if g.mandatory {
			mandatory++
		}
	}
	if mandatory != 36 {
		t.Errorf("%d mandatory goals, want 36 (ten digits and the letters)", mandatory)
	}
}
