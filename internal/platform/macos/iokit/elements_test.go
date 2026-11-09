package iokit

import (
	"reflect"
	"testing"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		name  string
		typ   ElementType
		page  uint32
		usage uint32
		want  Kind
	}{
		{"x axis", ElementInputAxis, pageGenericDesktop, usageX, KindAxis},
		{"wheel", ElementInputMisc, pageGenericDesktop, usageWheel, KindAxis},
		{"dial", ElementInputMisc, pageGenericDesktop, usageDial, KindAxis},
		{"hat", ElementInputMisc, pageGenericDesktop, usageHatswitch, KindHat},
		{"dpad up", ElementInputButton, pageGenericDesktop, usageDPadUp, KindButton},
		{"start", ElementInputButton, pageGenericDesktop, usageStart, KindButton},
		{"main menu", ElementInputButton, pageGenericDesktop, usageSystemMainMenu, KindButton},
		{"a generic desktop usage that is none", ElementInputMisc, pageGenericDesktop, 0x01, KindNone},
		{"throttle", ElementInputAxis, pageSimulation, usageThrottle, KindAxis},
		{"brake", ElementInputAxis, pageSimulation, usageBrake, KindAxis},
		{"another simulation usage", ElementInputAxis, pageSimulation, 0x01, KindNone},
		{"button page", ElementInputButton, pageButton, 7, KindButton},
		{"consumer page", ElementInputButton, pageConsumer, 0x40, KindButton},
		{"led page", ElementInputButton, 0x08, 1, KindNone},
		{"a collection", ElementCollection, pageGenericDesktop, usageX, KindNone},
		{"an output element", ElementType(129), pageButton, 1, KindNone},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Classify(tt.typ, tt.page, tt.usage); got != tt.want {
				t.Errorf("Classify = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestBuildLayoutSortsByUsageAndDropsDuplicates(t *testing.T) {
	raw := []RawElement{
		{Cookie: 5, Type: ElementInputAxis, Page: pageGenericDesktop, Usage: usageY, Max: 255},
		{Cookie: 4, Type: ElementInputAxis, Page: pageGenericDesktop, Usage: usageX, Max: 255},
		{Cookie: 4, Type: ElementInputAxis, Page: pageGenericDesktop, Usage: usageX, Max: 255}, // the same cookie again
		{Cookie: 9, Type: ElementInputButton, Page: pageButton, Usage: 2},
		{Cookie: 8, Type: ElementInputButton, Page: pageButton, Usage: 1},
		{Cookie: 10, Type: ElementInputButton, Page: pageButton, Usage: 2}, // the same usage keeps the order of the device
		{Cookie: 11, Type: ElementInputMisc, Page: pageGenericDesktop, Usage: usageHatswitch, Max: 7},
		{Cookie: 12, Type: ElementInputMisc, Page: 0xFF00, Usage: 1}, // vendor page: not used
	}
	l := BuildLayout(raw)
	cookies := func(es []*Element) []uint32 {
		var out []uint32
		for _, e := range es {
			out = append(out, e.Cookie)
		}
		return out
	}
	if got := cookies(l.Axes); !reflect.DeepEqual(got, []uint32{4, 5}) {
		t.Errorf("axes = %v", got)
	}
	if got := cookies(l.Buttons); !reflect.DeepEqual(got, []uint32{8, 9, 10}) {
		t.Errorf("buttons = %v", got)
	}
	if got := cookies(l.Hats); !reflect.DeepEqual(got, []uint32{11}) {
		t.Errorf("hats = %v", got)
	}
	if l.Count() != 6 {
		t.Errorf("count = %d", l.Count())
	}
	if p := l.byCookie[9]; p.kind != KindButton || p.index != 1 {
		t.Errorf("position of 9 = %+v", p)
	}
}

func TestAxisValueScalesAndCalibrates(t *testing.T) {
	tests := []struct {
		name     string
		min, max int32
		reads    []int32
		want     []int16
	}{
		{"unsigned byte", 0, 255, []int32{0, 255, 128}, []int16{-32768, 32767, 128}},
		{"signed", -128, 127, []int32{-128, 0, 127}, []int16{-32768, 128, 32767}},
		{"a trigger", 0, 1023, []int32{0, 1023}, []int16{-32768, 32767}},
		// A pad that goes beyond its logical limits fills the range by what it has done: 300 stretches the range.
		{"beyond the limits", 0, 255, []int32{300, 0}, []int16{32767, -32768}},
		{"a single value", 5, 5, []int32{5}, []int16{5}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Element{RawElement: RawElement{Min: tt.min, Max: tt.max}, minReport: tt.min, maxReport: tt.max}
			for i, v := range tt.reads {
				if got := e.AxisValue(v); got != tt.want[i] {
					t.Errorf("read %d: AxisValue(%d) = %d, want %d", i, v, got, tt.want[i])
				}
			}
		})
	}
}

func TestHatValue(t *testing.T) {
	eight := &Element{RawElement: RawElement{Min: 0, Max: 7}}
	eightFromOne := &Element{RawElement: RawElement{Min: 1, Max: 8}}
	four := &Element{RawElement: RawElement{Min: 0, Max: 3}}
	odd := &Element{RawElement: RawElement{Min: 0, Max: 5}}
	tests := []struct {
		name string
		e    *Element
		v    int32
		want int
	}{
		{"8: up", eight, 0, hatUp},
		{"8: up right", eight, 1, hatUp | hatRight},
		{"8: right", eight, 2, hatRight},
		{"8: right down", eight, 3, hatRight | hatDown},
		{"8: down", eight, 4, hatDown},
		{"8: down left", eight, 5, hatDown | hatLeft},
		{"8: left", eight, 6, hatLeft},
		{"8: left up", eight, 7, hatLeft | hatUp},
		{"8: the null state", eight, 8, 0},
		{"8 from 1: up", eightFromOne, 1, hatUp},
		{"8 from 1: left up", eightFromOne, 8, hatLeft | hatUp},
		{"8 from 1: the null state", eightFromOne, 0, 0},
		{"4: up", four, 0, hatUp},
		{"4: right", four, 1, hatRight},
		{"4: down", four, 2, hatDown},
		{"4: left", four, 3, hatLeft},
		{"4: the null state", four, 4, 0},
		{"4: below the range", four, -1, 0},
		{"6 positions is centered", odd, 2, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.e.HatValue(tt.v); got != tt.want {
				t.Errorf("HatValue(%d) = %d, want %d", tt.v, got, tt.want)
			}
		})
	}
}

func TestButtonValue(t *testing.T) {
	e := &Element{}
	for v, want := range map[int32]int{0: 0, 1: 1, 2: 1, -1: 1} {
		if got := e.ButtonValue(v); got != want {
			t.Errorf("ButtonValue(%d) = %d, want %d", v, got, want)
		}
	}
}

func TestIsPad(t *testing.T) {
	tests := []struct {
		page, usage uint32
		want        bool
	}{
		{pageGenericDesktop, usageJoystick, true},
		{pageGenericDesktop, usageGamePad, true},
		{pageGenericDesktop, usageMultiAxisController, true},
		{pageGenericDesktop, 0x06, false}, // a keyboard
		{pageGenericDesktop, 0x02, false}, // a mouse
		{pageConsumer, usageGamePad, false},
	}
	for _, tt := range tests {
		if got := IsPad(tt.page, tt.usage); got != tt.want {
			t.Errorf("IsPad(%#x, %#x) = %v, want %v", tt.page, tt.usage, got, tt.want)
		}
	}
	if len(MatchingPages()) != 3 {
		t.Errorf("MatchingPages = %v", MatchingPages())
	}
}
