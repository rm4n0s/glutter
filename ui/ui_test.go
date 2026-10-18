package ui

import "testing"

func TestKeyDataString(t *testing.T) {
	tests := []struct {
		name string
		data KeyData
		want string
	}{
		{
			name: "unicode letter down",
			data: KeyData{Type: KeyEventTypeDown, Physical: 0x70004, Logical: 0x61, Character: "a"},
			want: `KeyData(Key Down, physical: 0x70004, logical: 0x61 (Unicode), character: "a" (0x61)`,
		},
		{
			name: "key up without character",
			data: KeyData{Type: KeyEventTypeUp, Physical: 0x70004, Logical: 0x61},
			want: `KeyData(Key Up, physical: 0x70004, logical: 0x61 (Unicode), character: <none>`,
		},
		{
			name: "GTK plane, repeat, synthesized",
			data: KeyData{Type: KeyEventTypeRepeat, Physical: 0x700e1, Logical: 0x1500000001, Synthesized: true},
			want: `KeyData(Key Repeat, physical: 0x700e1, logical: 0x1500000001 (GTK), character: <none>, synthesized`,
		},
		{
			name: "unprintable plane",
			data: KeyData{Type: KeyEventTypeDown, Physical: 0x70028, Logical: 0x100000000},
			want: `KeyData(Key Down, physical: 0x70028, logical: 0x100000000 (Unprintable), character: <none>`,
		},
		{
			name: "unknown plane has no description",
			data: KeyData{Type: KeyEventTypeDown, Logical: 0x9900000001},
			want: `KeyData(Key Down, physical: 0x0, logical: 0x9900000001, character: <none>`,
		},
		{
			name: "newline is escaped",
			data: KeyData{Type: KeyEventTypeDown, Physical: 0x70028, Logical: 0x10000000d, Character: "\n"},
			want: `KeyData(Key Down, physical: 0x70028, logical: 0x10000000d (Unprintable), character: "\n" (0x0a)`,
		},
		{
			name: "character outside the BMP uses surrogate pairs",
			data: KeyData{Type: KeyEventTypeDown, Logical: 0x1f600, Character: "😀"},
			want: `KeyData(Key Down, physical: 0x0, logical: 0x1f600 (Unicode), character: "😀" (0xd83d de00)`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.data.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEnumStrings(t *testing.T) {
	tests := []struct {
		got  string
		want string
	}{
		{PointerChangeUnspecified.String(), "PointerChangeUnspecified"},
		{PointerChangePanZoomEnd.String(), "PointerChangePanZoomEnd"},
		{PointerChange(99).String(), "PointerChange(99)"},
		{PointerDeviceKindMouse.String(), "PointerDeviceKindMouse"},
		{PointerDeviceKind(-1).String(), "PointerDeviceKind(-1)"},
		{PointerSignalKindScrollInertiaCancel.String(), "PointerSignalKindScrollInertiaCancel"},
		{KeyEventTypeRepeat.String(), "Key Repeat"},
		{KeyEventDeviceTypeDirectionalPad.String(), "Directional Pad"},
		{AppLifecycleStateHidden.String(), "AppLifecycleStateHidden"},
		{AppLifecycleState(42).String(), "AppLifecycleState(42)"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("got %q, want %q", tt.got, tt.want)
		}
	}
}

func TestEnumOrderMatchesFlutter(t *testing.T) {
	// Flutter's declaration order, after the Unspecified zero value.
	if PointerChangeCancel != 1 || PointerChangePanZoomEnd != 10 {
		t.Errorf("PointerChange values are out of order: cancel=%d panZoomEnd=%d", PointerChangeCancel, PointerChangePanZoomEnd)
	}
	if AppLifecycleStateDetached != 1 || AppLifecycleStatePaused != 5 {
		t.Errorf("AppLifecycleState values are out of order")
	}
}

func TestPointerDataRespond(t *testing.T) {
	tests := []struct {
		name  string
		allow bool
	}{{"allow", true}, {"deny", false}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got *bool
			d := PointerData{OnRespond: func(allow bool) { got = &allow }}
			d.Respond(tt.allow)
			if got == nil || *got != tt.allow {
				t.Errorf("OnRespond got %v, want %v", got, tt.allow)
			}
		})
	}
	t.Run("nil OnRespond is a no-op", func(t *testing.T) {
		PointerData{}.Respond(true)
	})
}

func TestPointerDataString(t *testing.T) {
	d := PointerData{ViewID: 2, PhysicalX: 1.5, PhysicalY: 3}
	if got, want := d.String(), "PointerData(viewId: 2, x: 1.5, y: 3)"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}
