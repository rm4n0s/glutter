package gamepaddb

import (
	"reflect"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
)

func TestCRC16(t *testing.T) {
	// The mapping of the table says that "Defender Joystick Cobra R4" has the CRC c77a.
	tests := []struct {
		name string
		want uint16
	}{
		{"Defender Joystick Cobra R4", 0xc77a},
		{"", 0},
	}
	for _, tt := range tests {
		if got := crc16(0, tt.name); got != tt.want {
			t.Errorf("crc16(%q) = %04x, want %04x", tt.name, got, tt.want)
		}
	}
}

func TestGUID(t *testing.T) {
	tests := []struct {
		name                              string
		guid                              GUID
		wantString                        string
		wantVendor, wantProduct, wantVers uint16
		wantCRC                           uint16
	}{
		{
			name:       "an Xbox 360 pad on USB without names",
			guid:       NewGUID(BusUSB, 0x045e, 0x028e, 0x0114, "", "", 0, 0),
			wantString: "030000005e0400008e02000014010000",
			wantVendor: 0x045e, wantProduct: 0x028e, wantVers: 0x0114,
		},
		{
			name:       "an XInput pad",
			guid:       NewGUID(BusUSB, 0x045e, 0x028e, 0x0114, "", "", 'x', 1),
			wantString: "030000005e0400008e02000014017801"[:28] + "7801",
			wantVendor: 0x045e, wantProduct: 0x028e, wantVers: 0x0114,
		},
		{
			name:       "a pad with a vendor name and a product name",
			guid:       NewGUID(BusUSB, 0x1234, 0x5678, 1, "Acme", "Pad", 0, 0),
			wantString: "0300" + "cd4f" + "34120000" + "78560000" + "01000000",
			wantVendor: 0x1234, wantProduct: 0x5678, wantVers: 1,
			wantCRC: crc16(crc16(crc16(0, "Acme"), " "), "Pad"),
		},
		{
			name:       "a pad that only has a name",
			guid:       NewGUIDForName("Old Pad"),
			wantString: "00007b5f" + "4f6c64205061640000000000"[:24],
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.guid
			// The CRC bytes depend on the names; the expectations above are checked after the CRC is read back.
			info := got.Info()
			if tt.wantCRC != 0 && info.CRC != tt.wantCRC {
				t.Errorf("CRC = %04x, want %04x", info.CRC, tt.wantCRC)
			}
			if info.Vendor != tt.wantVendor || info.Product != tt.wantProduct || info.Version != tt.wantVers {
				t.Errorf("Info() = %+v, want vendor %04x product %04x version %04x", info, tt.wantVendor, tt.wantProduct, tt.wantVers)
			}
			if rt := ParseGUID(got.String()); rt != got {
				t.Errorf("ParseGUID(String()) = %v, want %v", rt, got)
			}
		})
	}
	if got := NewGUID(BusUSB, 0x045e, 0x028e, 0x0114, "", "", 0, 0).String(); got != "030000005e0400008e02000014010000" {
		t.Errorf("GUID = %s", got)
	}
	if got := NewGUIDForName("Old Pad").String(); !strings.HasPrefix(got[8:], "4f6c64205061640000") {
		t.Errorf("a GUID of a name has the name in it: %s", got)
	}
}

func TestParseGUIDIsForgiving(t *testing.T) {
	tests := []struct {
		in   string
		want GUID
	}{
		{"", GUID{}},
		{"zz", GUID{}},
		{"0Aff", GUID{0x0a, 0xff}},
		{"abc", GUID{0xab}},
		{strings.Repeat("11", 20), GUID{0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11}},
	}
	for _, tt := range tests {
		if got := ParseGUID(tt.in); got != tt.want {
			t.Errorf("ParseGUID(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestDeviceName(t *testing.T) {
	tests := []struct {
		name                    string
		vendor, product         uint16
		vendorName, productName string
		want                    string
	}{
		{"the list gives a name", 0x045e, 0x028e, "", "Microsoft X-Box 360 pad", "Xbox 360 Controller"},
		{"both names are joined", 0x1234, 0x5678, "Acme", "Pad", "Acme Pad"},
		{"only a product name", 0x1234, 0x5678, "", "Pad", "Pad"},
		{"leading spaces go and double spaces are one", 0x1234, 0x5678, "  Acme", "Big   Pad  ", "Acme Big Pad"},
		{"a maker's name is shortened", 0x1234, 0x5678, "8BitDo Tech Ltd", "SN30", "8BitDo SN30"},
		{"unknown is dropped", 0x1234, 0x5678, "", "Unknown Pad", "Pad"},
		{"a DualSense without a name", 0x054c, 0x0ce6, "", "", "DualSense Wireless Controller"},
		{"numbers when nothing else is known", 0x1234, 0x5678, "", "", "0x1234/0x5678"},
		{"the default", 0, 0, "", "", "Controller"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DeviceName(tt.vendor, tt.product, tt.vendorName, tt.productName); got != tt.want {
				t.Errorf("DeviceName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestKindOf(t *testing.T) {
	tests := []struct {
		name            string
		vendor, product uint16
		padName         string
		forUI           bool
		want            Kind
	}{
		{"Xbox 360", 0x045e, 0x028e, "", true, KindXbox360},
		{"Xbox One", 0x045e, 0x02ea, "", true, KindXboxOne},
		{"DualShock 3", 0x054c, 0x0268, "", true, KindPS3},
		{"DualShock 4", 0x054c, 0x09cc, "", true, KindPS4},
		{"DualSense", 0x054c, 0x0ce6, "", true, KindPS5},
		{"Switch Pro", 0x057e, 0x2009, "", true, KindSwitchPro},
		{"Joy-Con left", 0x057e, 0x2006, "", true, KindJoyConLeft},
		{"Joy-Con right", 0x057e, 0x2007, "", true, KindJoyConRight},
		{"the NES pad of Nintendo Online", 0x057e, 0x2007, "Nintendo Switch Online NES Controller", true, KindStandard},
		{"Joy-Con grip, left", 0x057e, 0x200e, "Joy-Con (L)", true, KindJoyConLeft},
		{"Joy-Con grip, pair", 0x057e, 0x200e, "Joy-Con (L/R) L+R", true, KindJoyConPair},
		{"Joy-Con grip, right", 0x057e, 0x200e, "Joy-Con (R)", true, KindJoyConRight},
		{"Joy-Con pair", 0x057e, 0x2008, "", true, KindJoyConPair},
		{"a GameCube adapter for the user", 0x057e, 0x0337, "", true, KindGameCube},
		{"a GameCube adapter for the mapping", 0x057e, 0x0337, "", false, KindStandard},
		{"a clone known by its name", 0, 0, "Lic Pro Controller", true, KindSwitchPro},
		{"a pad nobody knows", 0x1234, 0x5678, "", true, KindStandard},
		{"no numbers and no name", 0, 0, "", true, KindStandard},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := KindOf(tt.vendor, tt.product, tt.padName, tt.forUI); got != tt.want {
				t.Errorf("KindOf() = %v, want %v", got, tt.want)
			}
		})
	}
	if got := KindOfGUID(NewGUID(BusUSB, 0x1234, 0x5678, 1, "", "", 'x', 0), ""); got != KindXboxOne {
		t.Errorf("an XInput pad of no known model is %v, want KindXboxOne", got)
	}
}

func TestKindStrings(t *testing.T) {
	for k := KindUnknown; k < kindCount; k++ {
		if got := KindFromString(k.String()); got != k {
			t.Errorf("KindFromString(%q) = %v, want %v", k.String(), got, k)
		}
	}
	if KindFromString("+ps5") != KindPS5 || KindFromString("nonsense") != KindUnknown {
		t.Error("KindFromString must skip a sign and not know nonsense")
	}
	if FaceStyleOf(KindPS4) != FaceSony || FaceStyleOf(KindSwitchPro) != FaceBAYX || FaceStyleOf(KindGameCube) != FaceAXBY || FaceStyleOf(KindXbox360) != FaceABXY {
		t.Error("FaceStyleOf is wrong")
	}
}

func TestControllersAreSorted(t *testing.T) {
	for i := 1; i < len(Controllers); i++ {
		a, b := Controllers[i-1], Controllers[i]
		if uint32(a.Vendor)<<16|uint32(a.Product) >= uint32(b.Vendor)<<16|uint32(b.Product) {
			t.Fatalf("Controllers[%d] = %04x:%04x is not after %04x:%04x", i, b.Vendor, b.Product, a.Vendor, a.Product)
		}
	}
	if len(Controllers) < 400 {
		t.Errorf("the controller list has %d entries", len(Controllers))
	}
}

func TestParseBindings(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []Binding
	}{
		{
			name: "a button and an axis",
			text: "a:b0,leftx:a0",
			want: []Binding{
				{In: bindButton, Out: bindButton, InIndex: 0, OutButton: South},
				{In: bindAxis, Out: bindAxis, InIndex: 0, InMin: AxisMin, InMax: AxisMax, OutAxis: LeftX, OutMin: AxisMin, OutMax: AxisMax},
			},
		},
		{
			name: "an inverted axis",
			text: "lefty:a1~",
			want: []Binding{{In: bindAxis, Out: bindAxis, InIndex: 1, InMin: AxisMax, InMax: AxisMin, OutAxis: LeftY, OutMin: AxisMin, OutMax: AxisMax}},
		},
		{
			name: "half axes in and out",
			text: "+leftx:+a3,-lefty:-a4~",
			want: []Binding{
				{In: bindAxis, Out: bindAxis, InIndex: 3, InMin: 0, InMax: AxisMax, OutAxis: LeftX, OutMin: 0, OutMax: AxisMax},
				{In: bindAxis, Out: bindAxis, InIndex: 4, InMin: AxisMin, InMax: 0, OutAxis: LeftY, OutMin: 0, OutMax: AxisMin},
			},
		},
		{
			name: "a trigger is always a half axis",
			text: "lefttrigger:a2",
			want: []Binding{{In: bindAxis, Out: bindAxis, InIndex: 2, InMin: AxisMin, InMax: AxisMax, OutAxis: LeftTrigger, OutMin: 0, OutMax: AxisMax}},
		},
		{
			name: "a hat and a button on an axis",
			text: "dpup:h0.1,dpright:h1.2,righttrigger:b7",
			want: []Binding{
				{In: bindHat, Out: bindButton, InIndex: 0, HatMask: 1, OutButton: DPadUp},
				{In: bindHat, Out: bindButton, InIndex: 1, HatMask: 2, OutButton: DPadRight},
				{In: bindButton, Out: bindAxis, InIndex: 7, OutAxis: RightTrigger, OutMin: 0, OutMax: AxisMax},
			},
		},
		{
			name: "fields that are not bindings, and names that are not known, are skipped",
			text: "crc:1234,type:xbox360,face:abxy,platform:Linux,hint:!SDL_GAMECONTROLLER_USE_BUTTON_LABELS:=1,sdk>=:29,bogus:b3,a:q9,b:b1",
			want: []Binding{{In: bindButton, Out: bindButton, InIndex: 1, OutButton: East}},
		},
		{
			name: "the same binding twice is one",
			text: "a:b0,a:b0",
			want: []Binding{{In: bindButton, Out: bindButton, InIndex: 0, OutButton: South}},
		},
		{
			name: "spaces are ignored and a trailing comma is fine",
			text: " a : b0 ,",
			want: []Binding{{In: bindButton, Out: bindButton, InIndex: 0, OutButton: South}},
		},
		{
			name: "the names are not case sensitive",
			text: "A:b0,LeftX:a0",
			want: []Binding{
				{In: bindButton, Out: bindButton, InIndex: 0, OutButton: South},
				{In: bindAxis, Out: bindAxis, InIndex: 0, InMin: AxisMin, InMax: AxisMax, OutAxis: LeftX, OutMin: AxisMin, OutMax: AxisMax},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseBindings(tt.text)
			if err != nil {
				t.Fatalf("ParseBindings() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseBindings(%q)\n got %+v\nwant %+v", tt.text, got, tt.want)
			}
		})
	}
}

func TestParseBindingsTooLong(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{"a button name", strings.Repeat("x", 20) + ":b0"},
		{"an input name", "a:" + strings.Repeat("b", 128)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseBindings(tt.text)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if !appErr.HasRoute("ParseBindings.GamepadMappingInvalid") {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
		})
	}
}

func TestBuiltinTableParses(t *testing.T) {
	sections := map[string][]string{"XInput": XInput, "WGI": WGI, "DInput": DInput, "Linux": Linux, "Android": Android, "Web": Web}
	for name, lines := range sections {
		table := NewTable(lines)
		if table.Len() == 0 {
			t.Errorf("%s: no mapping was added", name)
		}
		for _, m := range table.Mappings() {
			b, err := ParseBindings(m.Text)
			if err != nil {
				t.Errorf("%s: %q: %v", name, m.Name, err)
			}
			if len(b) == 0 && !strings.HasPrefix(m.Text, "platform") {
				t.Errorf("%s: %q has no bindings: %q", name, m.Name, m.Text)
			}
		}
	}
	if NewTable(XInput).xinput == nil {
		t.Error("the XInput section has no xinput mapping")
	}
	if NewTable(Web).Default() == nil {
		t.Error("the Web section has no default mapping")
	}
	if n := NewTable(Linux).Len(); n < 250 {
		t.Errorf("the Linux section has %d mappings", n)
	}
}

func TestTableMatching(t *testing.T) {
	const (
		versioned = "030000001234000056780000aa010000" // version 0x01aa
		other     = "030000001234000056780000bb010000" // the same pad, another version
	)
	table := NewTable()
	for _, line := range []string{
		"03000000123400005678000000000000,Any Version,a:b2,",
		versioned + ",Versioned,a:b1,",
		"0300000099990000aaaa000000000000,With CRC,crc:abcd,a:b3,",
		"0300000099990000aaaa000000000000,Without CRC,a:b4,",
	} {
		if n, err := table.Add(line); err != nil || n != 1 {
			t.Fatalf("Add(%q) = %d, %v", line, n, err)
		}
	}
	tests := []struct {
		name string
		guid string
		want string
	}{
		{"the exact version", versioned, "Versioned"},
		{"another version falls back to version zero", other, "Any Version"},
		{"nothing known", "030000000000000000000000ffff0000", ""},
		{"a CRC that matches", "0300cdab99990000aaaa000000000000", "With CRC"},
		{"a CRC that does not match falls back to the mapping without one", "0300ffff99990000aaaa000000000000", "Without CRC"},
		{"no CRC at all prefers the mapping without one", "0300000099990000aaaa000000000000", "Without CRC"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ""
			if m := table.Find(ParseGUID(tt.guid)); m != nil {
				got = m.Name
			}
			if got != tt.want {
				t.Errorf("Find(%s) = %q, want %q", tt.guid, got, tt.want)
			}
		})
	}
}

func TestTablePriorityAndReplacement(t *testing.T) {
	const guid = "03000000123400005678000001000000"
	table := NewTable()
	steps := []struct {
		line     string
		priority Priority
		wantNew  int
		wantName string
	}{
		{guid + ",First,a:b0,", PriorityDefault, 1, "First"},
		{guid + ",Second,a:b1,", PriorityAPI, 0, "Second"},
		{guid + ",Third,a:b2,", PriorityDefault, 0, "Second"},
		{guid + ",Fourth,a:b3,", PriorityUser, 0, "Fourth"},
	}
	for _, s := range steps {
		n, err := table.add(s.line, s.priority)
		if err != nil || n != s.wantNew {
			t.Fatalf("add(%q) = %d, %v, want %d", s.line, n, err, s.wantNew)
		}
		if got := table.Find(ParseGUID(guid)).Name; got != s.wantName {
			t.Errorf("after %q the mapping is %q, want %q", s.line, got, s.wantName)
		}
	}
	if table.Len() != 1 {
		t.Errorf("Len() = %d, want 1", table.Len())
	}
}

func TestTableSpecialGUIDs(t *testing.T) {
	table := NewTable()
	if _, err := table.Add("xinput,*,a:b0,"); err != nil {
		t.Fatal(err)
	}
	if _, err := table.Add("default,*,a:b1,"); err != nil {
		t.Fatal(err)
	}
	if table.Default() == nil || table.Default().Text != "a:b1," {
		t.Errorf("Default() = %+v", table.Default())
	}
	xinputPad := NewGUID(BusUSB, 0x045e, 0x02ea, 1, "", "", 'x', 0)
	if m := table.Find(xinputPad); m == nil || m.Text != "a:b0," {
		t.Errorf("an XInput pad gets %+v, want the xinput mapping", m)
	}
	if m := table.Find(NewGUID(BusUSB, 0x1234, 0x5678, 1, "", "", 0, 0)); m != nil {
		t.Errorf("a pad that is not XInput got %+v", m)
	}
}

func TestTableHints(t *testing.T) {
	env := map[string]string{"GLUTTER_TEST_HINT": "1"}
	getenv := func(k string) string { return env[k] }
	tests := []struct {
		name     string
		line     string
		wantNew  int
		wantText string
	}{
		{"labels are swapped to positions", "03000000123400005678000001000000,P,a:b0,b:b1,x:b2,y:b3,hint:SDL_GAMECONTROLLER_USE_BUTTON_LABELS:=1,", 1, "b:b0,a:b1,y:b2,x:b3,hint:!SDL_GAMECONTROLLER_USE_BUTTON_LABELS:=1,"},
		{"a hint that says positions is kept", "03000000123400005678000001000000,P,a:b0,b:b1,hint:!SDL_GAMECONTROLLER_USE_BUTTON_LABELS:=1,", 1, "a:b0,b:b1,hint:!SDL_GAMECONTROLLER_USE_BUTTON_LABELS:=1,"},
		{"GameCube labels swap B and X", "03000000123400005678000001000000,P,a:b0,b:b1,x:b2,hint:SDL_GAMECONTROLLER_USE_GAMECUBE_LABELS:=1,", 1, "a:b0,x:b1,b:b2,hint:!SDL_GAMECONTROLLER_USE_GAMECUBE_LABELS:=1,"},
		{"a hint that is on", "03000000123400005678000001000000,P,a:b0,hint:GLUTTER_TEST_HINT:=0,", 1, "a:b0,hint:GLUTTER_TEST_HINT:=0,"},
		{"a hint that is off turns the mapping off", "03000000123400005678000001000000,P,a:b0,hint:GLUTTER_TEST_OFF:=0,", 0, ""},
		{"a hint that is off by a negation", "03000000123400005678000001000000,P,a:b0,hint:!GLUTTER_TEST_HINT:=0,", 0, ""},
		{"a GameCube adapter gets the GameCube labels", "030000007e0500003703000001000000,GC,a:b0,b:b1,x:b2,", 1, "a:b0,x:b1,b:b2,hint:!SDL_GAMECONTROLLER_USE_GAMECUBE_LABELS:=1,"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table := NewTable()
			table.Getenv = getenv
			n, err := table.Add(tt.line)
			if err != nil || n != tt.wantNew {
				t.Fatalf("Add() = %d, %v, want %d", n, err, tt.wantNew)
			}
			if tt.wantNew == 0 {
				return
			}
			if got := table.Mappings()[0].Text; got != tt.wantText {
				t.Errorf("Text = %q, want %q", got, tt.wantText)
			}
		})
	}
}

func TestTableCRCInTheGUIDMovesToTheText(t *testing.T) {
	table := NewTable()
	// The CRC is the second 16 bits of the GUID, little-endian: cdab is the CRC abcd.
	if _, err := table.Add("0300cdab123400005678000001000000,P,a:b0,"); err != nil {
		t.Fatal(err)
	}
	m := table.Mappings()[0]
	if m.GUID.Info().CRC != 0 || !strings.HasPrefix(m.Text, "a:b0,crc:abcd,") {
		t.Errorf("mapping = %+v, want the CRC in the text", m)
	}
}

func TestAddText(t *testing.T) {
	const text = "# a comment\n" +
		"03000000123400005678000001000000,Linux Pad,a:b0,platform:Linux,\n" +
		"03000000123400005678000002000000,Windows Pad,a:b0,platform:Windows,\r\n" +
		"03000000123400005678000003000000,No Platform,a:b0,\n" +
		"03000000123400005678000004000000,Last,a:b0,platform:linux\n" +
		"03000000123400005678000005000000,Also Linux,a:b0,platform:Linux,"
	table := NewTable()
	n, err := table.AddText(text, "Linux")
	if err != nil || n != 2 {
		t.Fatalf("AddText() = %d, %v, want 2 mappings", n, err)
	}
	var names []string
	for _, m := range table.Mappings() {
		names = append(names, m.Name)
	}
	if want := []string{"Linux Pad", "Also Linux"}; !reflect.DeepEqual(names, want) {
		t.Errorf("mappings = %v, want %v", names, want)
	}
}

func TestAddInvalid(t *testing.T) {
	tests := []struct {
		name  string
		line  string
		route string
	}{
		{"no comma", "nothing", "Table.Add->Table.add.GamepadMappingInvalid"},
		{"no name", "03000000123400005678000001000000,OnlyAName", "Table.Add->Table.add.GamepadMappingInvalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewTable().Add(tt.line)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if appErr.Tag != TagMappingInvalid || !appErr.HasRoute(tt.route) {
				t.Errorf("unexpected failure: tag %s, path %s", appErr.Tag, appErr.Route())
			}
		})
	}
	table := &Table{SDKVersion: 28}
	if _, err := table.Add("03000000123400005678000001000000,P,a:b0,sdk>=:29,"); err == nil {
		t.Error("a mapping for a newer Android was added")
	}
	table.SDKVersion = 30
	if n, err := table.Add("03000000123400005678000001000000,P,a:b0,sdk>=:29,"); err != nil || n != 1 {
		t.Errorf("a mapping for Android 29 on Android 30 = %d, %v", n, err)
	}
}

func TestSetAuto(t *testing.T) {
	table := NewTable()
	guid := NewGUID(BusUSB, 0x1234, 0x5678, 1, "", "Some, Pad", 0, 0)
	m := table.SetAuto(guid, "Some, Pad", "a:b0,")
	if m.Name != "Some  Pad" || m.Priority != PriorityDefault {
		t.Errorf("SetAuto() = %+v", m)
	}
	if got := table.Find(guid); got != m {
		t.Errorf("Find() = %+v, want the made mapping", got)
	}
	if again := table.SetAuto(guid, "Other", "a:b9,"); again != m {
		t.Errorf("a second SetAuto replaced the mapping")
	}
	// A mapping that the app adds for the same pad wins over the made one.
	if _, err := table.Add(guid.WithCRC(0).String() + ",Mine,a:b2,"); err != nil {
		t.Fatal(err)
	}
	if got := table.Find(guid).Name; got != "Mine" {
		t.Errorf("Find() = %q, want the app's mapping", got)
	}
}

type recorder struct {
	events []string
}

func (r *recorder) ButtonChanged(_ int64, b Button, down bool) {
	r.events = append(r.events, b.Name()+map[bool]string{true: " down", false: " up"}[down])
}

func (r *recorder) AxisChanged(_ int64, a Axis, v int16) {
	r.events = append(r.events, a.Name()+"="+itoa(int(v)))
}

func (r *recorder) take() []string {
	e := r.events
	r.events = nil
	return e
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var b []byte
	for ; v > 0; v /= 10 {
		b = append([]byte{byte('0' + v%10)}, b...)
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}

func newTestDevice(t *testing.T, text string, axes, buttons, hats int) (*Device, *recorder) {
	t.Helper()
	r := &recorder{}
	d, err := NewDevice(axes, buttons, hats, &Mapping{Text: text}, r)
	if err != nil {
		t.Fatalf("NewDevice() error = %v", err)
	}
	return d, r
}

func TestDeviceButtonsAndHats(t *testing.T) {
	d, r := newTestDevice(t, "a:b0,b:b1,dpup:h0.1,dpleft:h0.8,rightshoulder:b5,dpdown:b6", 0, 8, 1)
	steps := []struct {
		do   func()
		want []string
	}{
		{func() { d.RawButton(1, 0, true) }, []string{"a down"}},
		{func() { d.RawButton(2, 0, true) }, nil},
		{func() { d.RawButton(3, 0, false) }, []string{"a up"}},
		{func() { d.RawButton(4, 7, true) }, nil},
		{func() { d.RawButton(4, 99, true) }, nil},
		{func() { d.RawHat(5, 0, HatUp) }, []string{"dpup down"}},
		{func() { d.RawHat(6, 0, HatUp|HatLeft) }, []string{"dpleft down"}},
		{func() { d.RawHat(7, 0, HatLeft) }, []string{"dpup up"}},
		{func() { d.RawHat(8, 0, HatCentered) }, []string{"dpleft up"}},
		{func() { d.RawHat(9, 3, HatUp) }, nil},
		{func() { d.RawButton(10, 6, true); d.RawButton(10, 5, true) }, []string{"dpdown down", "rightshoulder down"}},
	}
	for i, s := range steps {
		s.do()
		if got := r.take(); !reflect.DeepEqual(got, s.want) {
			t.Errorf("step %d: events = %v, want %v", i, got, s.want)
		}
	}
	if !d.Button(DPadDown) || d.Button(South) || !d.HasButton(DPadUp) || d.HasButton(Misc1) {
		t.Error("Button/HasButton are wrong")
	}
	d.Recenter(11)
	if got := r.take(); !reflect.DeepEqual(got, []string{"rightshoulder up", "dpdown up"}) && !reflect.DeepEqual(got, []string{"dpdown up", "rightshoulder up"}) {
		t.Errorf("Recenter() events = %v", got)
	}
}

func TestDeviceAxes(t *testing.T) {
	// Axis 0 and 1 are a stick; axis 2 is a trigger that rests at the bottom of its range; axis 3 is a stick axis inverted; the
	// right trigger is a button on a full axis; axis 5 gives two buttons by its halves.
	d, r := newTestDevice(t, "leftx:a0,lefty:a1,lefttrigger:a2,rightx:a3~,righttrigger:b9,dpleft:-a5,dpright:+a5,rightshoulder:a4", 6, 10, 0)
	steps := []struct {
		name string
		do   func()
		want []string
	}{
		{"the first report only sets the rest value", func() { d.RawAxis(1, 0, 100) }, nil},
		{"jitter is dropped", func() { d.RawAxis(2, 0, 150) }, nil},
		{"movement is sent after the rest value", func() { d.RawAxis(3, 0, 20000) }, []string{"leftx=100", "leftx=20000"}},
		{"the same value again is dropped", func() { d.RawAxis(4, 0, 20000) }, nil},
		{"the rest value of a trigger", func() { d.RawAxis(5, 2, -32768) }, nil},
		{"a trigger half pulled", func() { d.RawAxis(6, 2, -16384) }, []string{"lefttrigger=8191"}},
		{"a trigger fully pulled", func() { d.RawAxis(7, 2, 32767) }, []string{"lefttrigger=32767"}},
		// The scaling truncates, as SDL's does: the middle of an inverted axis is -1, not 0.
		{"an inverted axis", func() { d.RawAxis(8, 3, 0); d.RawAxis(9, 3, 10000) }, []string{"rightx=-1", "rightx=-10001"}},
		{"an axis that gives a button", func() { d.RawAxis(10, 4, -32768); d.RawAxis(11, 4, 32767) }, []string{"rightshoulder down"}},
		{"and lets go", func() { d.RawAxis(12, 4, -32000) }, []string{"rightshoulder up"}},
		{"a button on the left half of an axis", func() { d.RawAxis(13, 5, 0); d.RawAxis(14, 5, -30000) }, []string{"dpleft down"}},
		{"the other half", func() { d.RawAxis(15, 5, 30000) }, []string{"dpleft up", "dpright down"}},
		{"back to the middle", func() { d.RawAxis(16, 5, 0) }, []string{"dpright up"}},
		{"an axis the pad does not have", func() { d.RawAxis(17, 9, 5) }, nil},
	}
	for _, s := range steps {
		s.do()
		got := r.take()
		// The right-hand side of an equals sign is checked as a number, so leave out the order of the initial value.
		if !reflect.DeepEqual(got, s.want) {
			t.Errorf("%s: events = %v, want %v", s.name, got, s.want)
		}
	}
	if !d.HasAxis(LeftTrigger) || d.HasAxis(RightY) || d.Axis(LeftTrigger) != 32767 || d.Axis(AxisCount) != 0 {
		t.Error("Axis/HasAxis are wrong")
	}
	d.Recenter(20)
	if got := r.take(); len(got) == 0 {
		t.Error("Recenter() did not centre the axes")
	}
	if d.Axis(LeftX) != 0 || d.Axis(LeftTrigger) != 0 {
		t.Error("the axes are not centred after Recenter()")
	}
}

func TestDeviceFromTheTable(t *testing.T) {
	table := NewTable(Linux)
	guid := NewGUID(BusUSB, 0x054c, 0x0ce6, 0x8111, "", "Sony Interactive Entertainment Wireless Controller", 0, 0)
	m := table.Find(guid)
	if m == nil || m.Name != "PS5 Controller" {
		t.Fatalf("Find() = %+v, want the PS5 mapping", m)
	}
	if m.Kind() != KindUnknown {
		t.Errorf("Kind() = %v, want the kind to come from the numbers", m.Kind())
	}
	r := &recorder{}
	d, err := NewDevice(6, 13, 1, m, r)
	if err != nil {
		t.Fatal(err)
	}
	// This mapping (the Bluetooth variant of the DualSense) has a:b0 and b:b1.
	d.RawButton(1, 1, true)
	d.RawButton(2, 0, true)
	if got := r.take(); !reflect.DeepEqual(got, []string{"b down", "a down"}) {
		t.Errorf("events = %v", got)
	}
	for _, b := range []Button{South, East, West, North, Back, Guide, Start, LeftStick, RightStick, LeftShoulder, RightShoulder, DPadUp, DPadDown, DPadLeft, DPadRight} {
		if !d.HasButton(b) {
			t.Errorf("the PS5 mapping has no %s", b.Name())
		}
	}
	for _, a := range []Axis{LeftX, LeftY, RightX, RightY, LeftTrigger, RightTrigger} {
		if !d.HasAxis(a) {
			t.Errorf("the PS5 mapping has no %s", a.Name())
		}
	}
}

func TestNamesRoundTrip(t *testing.T) {
	for b := Button(0); b < ButtonCount; b++ {
		got, ok := buttonFromName(b.Name())
		if !ok || got != b {
			t.Errorf("button %d: %q reads back as %d, %v", b, b.Name(), got, ok)
		}
	}
	for a := Axis(0); a < AxisCount; a++ {
		got, ok := axisFromName(a.Name())
		if !ok || got != a {
			t.Errorf("axis %d: %q reads back as %d, %v", a, a.Name(), got, ok)
		}
	}
	if Button(200).Name() != "" || Axis(200).Name() != "" {
		t.Error("a number out of range has a name")
	}
}

func TestMappingFields(t *testing.T) {
	m := &Mapping{Text: "a:b0,type:ps5,face:sony,crc:1f2e,hint:!SDL_GAMECONTROLLER_USE_BUTTON_LABELS:=1,"}
	if m.Field("type") != "ps5" || m.Kind() != KindPS5 || m.Field("nothing") != "" {
		t.Errorf("Field/Kind are wrong: %q %v", m.Field("type"), m.Kind())
	}
	if m.FaceStyle() != FaceSony {
		t.Errorf("FaceStyle() = %v", m.FaceStyle())
	}
	if crc, ok := m.crc(); !ok || crc != 0x1f2e {
		t.Errorf("crc() = %04x, %v", crc, ok)
	}
	labels := &Mapping{Text: "a:b0,hint:SDL_GAMECONTROLLER_USE_GAMECUBE_LABELS:=1,"}
	if labels.FaceStyle() != FaceAXBY {
		t.Errorf("a GameCube hint gives %v", labels.FaceStyle())
	}
}
