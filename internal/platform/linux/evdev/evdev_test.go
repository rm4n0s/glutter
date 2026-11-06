package evdev

import (
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl/layouttest"
)

func bitsOf(codes ...int) bits {
	var b bits
	for _, c := range codes {
		b = b.set(c)
	}
	return b
}

// padCaps makes the capabilities of a pad.
func padCaps(vendor, product uint16, name string, keys, abs []int, info map[int]absInfo) *caps {
	c := &caps{ID: inputID{Bus: 3, Vendor: vendor, Product: product, Version: 0x0110}, Name: name, Key: bitsOf(keys...), Abs: bitsOf(abs...)}
	for code, i := range info {
		c.Info[code] = i
	}
	return c
}

var stick = absInfo{Minimum: -32768, Maximum: 32767, Fuzz: 16, Flat: 128}
var trigger = absInfo{Minimum: 0, Maximum: 255}
var digitalHat = absInfo{Minimum: -1, Maximum: 1}

// xbox360 is what the xpad driver reports for an Xbox 360 pad.
func xbox360() *caps {
	return padCaps(0x045e, 0x028e, "Microsoft X-Box 360 pad",
		[]int{btnSouth, btnEast, btnNorth, btnWest, btnTL, btnTR, btnSelect, btnStart, btnMode, btnThumbL, btnThumbR},
		[]int{absX, absY, absZ, absRX, absRY, absRZ, absHat0X, absHat0Y},
		map[int]absInfo{absX: stick, absY: stick, absRX: stick, absRY: stick, absZ: trigger, absRZ: trigger, absHat0X: digitalHat, absHat0Y: digitalHat})
}

// dualShock4 is what the hid-sony driver reports for a DualShock 4.
func dualShock4() *caps {
	c := padCaps(vendorSony, 0x09cc, "Sony Interactive Entertainment Wireless Controller",
		[]int{btnSouth, btnEast, btnNorth, btnWest, btnTL, btnTR, btnTL2, btnTR2, btnSelect, btnStart, btnMode, btnThumbL, btnThumbR},
		[]int{absX, absY, absZ, absRX, absRY, absRZ, absHat0X, absHat0Y},
		map[int]absInfo{absX: {Minimum: 0, Maximum: 255, Flat: 0}, absY: {Maximum: 255}, absRX: {Maximum: 255}, absRY: {Maximum: 255},
			absZ: {Maximum: 255}, absRZ: {Maximum: 255}, absHat0X: digitalHat, absHat0Y: digitalHat})
	return c
}

func TestParseCapabilities(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []int
	}{
		{"empty", "0\n", nil},
		{"one word", "30000\n", []int{16, 17}},
		{"two words, the highest first", "1 8000000000000001\n", []int{0, 63, 64}},
		{"junk is skipped", "zz 3\n", []int{0, 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := parseCapabilities(tt.in)
			var got []int
			for i := 0; i < 8*len(b)+8; i++ {
				if b.has(i) {
					got = append(got, i)
				}
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("bits = %v, want %v", got, tt.want)
			}
		})
	}
	if bits(nil).has(3) || bitsOf(1).has(-1) || bitsOf(1).has(1000) {
		t.Error("has() must be false outside the set")
	}
}

type fakeFiles map[string]string

func (f fakeFiles) read(rel string) (string, error) {
	if v, ok := f[rel]; ok {
		return v, nil
	}
	return "", errNoFile
}

type noFile struct{}

func (noFile) Error() string { return "no file" }

var errNoFile = noFile{}

func TestProbe(t *testing.T) {
	files := fakeFiles{
		"device/id/bustype": "0003\n", "device/id/vendor": "045e\n", "device/id/product": "028e\n", "device/id/version": "0114\n",
		"device/name": "Microsoft X-Box 360 pad\n", "device/capabilities/key": "7cdb000000000000 0 0 0 0\n",
		"device/capabilities/abs": "3003f\n", "device/capabilities/rel": "0\n", "device/capabilities/ff": "0\n",
	}
	c, ok := probe(files)
	if !ok {
		t.Fatal("probe() failed")
	}
	if c.ID != (inputID{Bus: 3, Vendor: 0x045e, Product: 0x028e, Version: 0x0114}) || c.Name != "Microsoft X-Box 360 pad" {
		t.Errorf("id = %+v, name = %q", c.ID, c.Name)
	}
	if !c.Abs.has(absX) || !c.Abs.has(absHat0X) || !c.Abs.has(absRX) || c.Abs.has(absThrot) {
		t.Errorf("abs bits are wrong: %v", c.Abs)
	}
	delete(files, "device/capabilities/ff")
	if _, ok := probe(files); ok {
		t.Error("probe() works with a missing file; the device may be too new for sysfs")
	}
	files["device/capabilities/ff"] = "0"
	files["device/id/vendor"] = "xyz"
	if _, ok := probe(files); ok {
		t.Error("probe() works with a bad id")
	}
}

func TestIsJoystick(t *testing.T) {
	tests := []struct {
		name string
		c    *caps
		want bool
	}{
		{"an Xbox 360 pad", xbox360(), true},
		{"a DualShock 4", dualShock4(), true},
		{"a joystick with a trigger button", padCaps(0x046d, 0xc215, "Logitech Extreme 3D", []int{btnTrigger}, []int{absX, absY, absRZ, absThrot}, nil), true},
		{"pedals with only axes", padCaps(0x1234, 1, "Pedals", nil, []int{absX, absY, absRudd, absBrake, absGas}, nil), true},
		{"a pad with buttons and no axes", padCaps(0x1234, 2, "Buttons", []int{btnSouth, btnEast}, nil, nil), true},
		{"a mouse", &caps{Key: bitsOf(btnLeft, btnLeft+1), Rel: bitsOf(relX, relY)}, false},
		{"a keyboard", &caps{Key: bitsOf(keyEsc, keyEnter, keySpace, 30, 31)}, false},
		{"a touchpad", &caps{Key: bitsOf(btnLeft, btnToolFing, btnTouch), Abs: bitsOf(absX, absY, absMTX, absMTY)}, false},
		{"a touch screen", &caps{Key: bitsOf(btnTouch), Abs: bitsOf(absX, absY, absMTX, absMTY), Prop: bitsOf(propDirect)}, false},
		{"a tablet", &caps{Key: bitsOf(btnToolPen, btnStylus, btnTouch), Abs: bitsOf(absX, absY, absRX)}, false},
		{"the motion sensors of a pad", &caps{Abs: bitsOf(absX, absY, absZ, absRX, absRY, absRZ), Prop: bitsOf(propAccelerometer)}, false},
		{"a pad's touchpad", &caps{Key: bitsOf(btnLeft, btnToolFing, btnTouch), Abs: bitsOf(absMTX, absMTY, absX, absY)}, false},
		{"a power button", &caps{Key: bitsOf(116)}, false},
		{"nothing", &caps{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.isJoystick(); got != tt.want {
				t.Errorf("isJoystick() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLayoutNumbering(t *testing.T) {
	c := xbox360()
	l := newLayout(c)
	if l.buttons != 11 || l.axes != 6 || l.hats != 1 {
		t.Fatalf("buttons %d, axes %d, hats %d, want 11, 6, 1", l.buttons, l.axes, l.hats)
	}
	for code, want := range map[int]int16{btnSouth: 0, btnEast: 1, btnNorth: 2, btnWest: 3, btnTL: 4, btnTR: 5, btnSelect: 6, btnStart: 7, btnMode: 8, btnThumbL: 9, btnThumbR: 10, btnTL2: -1} {
		if l.keyMap[code] != want {
			t.Errorf("key 0x%x is button %d, want %d", code, l.keyMap[code], want)
		}
	}
	for code, want := range map[int]int16{absX: 0, absY: 1, absZ: 2, absRX: 3, absRY: 4, absRZ: 5, absHat0X: -1, absHat0Y: -1} {
		if l.absMap[code] != want {
			t.Errorf("axis 0x%x is axis %d, want %d", code, l.absMap[code], want)
		}
	}
	if l.hatIndex[0] != 0 || l.hatIndex[1] != -1 {
		t.Errorf("hat indices = %v", l.hatIndex)
	}

	// Keys below BTN_JOYSTICK come after those from it up.
	c = padCaps(1, 1, "Mixed", []int{btn0, btnTrigger, btnSouth, 5}, []int{absX}, map[int]absInfo{absX: stick})
	l = newLayout(c)
	if l.keyMap[btnTrigger] != 0 || l.keyMap[btnSouth] != 1 || l.keyMap[5] != 2 || l.keyMap[btn0] != 3 {
		t.Errorf("the order of the buttons is wrong: trigger %d, south %d, key 5 %d, btn0 %d", l.keyMap[btnTrigger], l.keyMap[btnSouth], l.keyMap[5], l.keyMap[btn0])
	}
}

func TestHatsThatAreAnalogAreAxes(t *testing.T) {
	analog := absInfo{Minimum: -32768, Maximum: 32767, Fuzz: 16, Flat: 128, Resolution: 1}
	l := newLayout(padCaps(1, 1, "Analog hat", []int{btnSouth}, []int{absX, absHat0X, absHat0Y}, map[int]absInfo{absX: stick, absHat0X: analog, absHat0Y: analog}))
	if l.hats != 0 || l.hatIndex[0] != -1 || l.absMap[absHat0X] < 0 || l.absMap[absHat0Y] < 0 {
		t.Errorf("an analog hat is a hat: hats %d, hatIndex %v", l.hats, l.hatIndex)
	}
	l = newLayout(padCaps(1, 1, "One axis hat", []int{btnSouth}, []int{absX, absHat0X}, map[int]absInfo{absX: stick, absHat0X: digitalHat}))
	if l.hats != 1 || l.correct()[0].min[1] != -1 {
		t.Errorf("a hat with one axis is not a hat: %d, %+v", l.hats, l.correct()[0])
	}
}

func (l *layout) correct() [4]hatCorrect { return l.hatCorrect }

func TestGuessAxesAreDigitalHat(t *testing.T) {
	tests := []struct {
		name string
		x, y *absInfo
		want bool
	}{
		{"neither", nil, nil, false},
		{"minus one to one", &absInfo{Minimum: -1, Maximum: 1, Fuzz: 5}, &absInfo{Minimum: -1, Maximum: 1}, true},
		{"one missing", nil, &absInfo{Minimum: -1, Maximum: 1}, true},
		{"no fuzz, flat or resolution", &absInfo{Minimum: -127, Maximum: 127}, &absInfo{Minimum: -127, Maximum: 127}, true},
		{"fuzz on one", &absInfo{Minimum: -127, Maximum: 127, Fuzz: 3}, &absInfo{Minimum: -127, Maximum: 127}, false},
		{"flat on the other", &absInfo{Minimum: -127, Maximum: 127}, &absInfo{Minimum: -127, Maximum: 127, Flat: 15}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := guessAxesAreDigitalHat(tt.x, tt.y); got != tt.want {
				t.Errorf("guessAxesAreDigitalHat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func fields(text string) []string {
	f := strings.Split(strings.TrimSuffix(text, ","), ",")
	sort.Strings(f)
	return f
}

func TestMappingOf(t *testing.T) {
	tests := []struct {
		name string
		c    *caps
		want string
	}{
		{
			name: "an Xbox 360 pad is the mapping SDL's table has for it",
			c:    xbox360(),
			want: "a:b0,b:b1,back:b6,dpdown:h0.4,dpleft:h0.8,dpright:h0.2,dpup:h0.1,guide:b8,leftshoulder:b4,leftstick:b9,lefttrigger:a2,leftx:a0,lefty:a1,rightshoulder:b5,rightstick:b10,righttrigger:a5,rightx:a3,righty:a4,start:b7,x:b2,y:b3,",
		},
		{
			name: "a DualShock 4 has its square and triangle the other way",
			c:    dualShock4(),
			want: "a:b0,b:b1,back:b8,dpdown:h0.4,dpleft:h0.8,dpright:h0.2,dpup:h0.1,guide:b10,leftshoulder:b4,leftstick:b11,lefttrigger:a2,leftx:a0,lefty:a1,rightshoulder:b5,rightstick:b12,righttrigger:a5,rightx:a3,righty:a4,start:b9,x:b3,y:b2,",
		},
		{
			name: "buttons for the D-pad and the triggers, and Android's axes",
			c: padCaps(0x2563, 0x0575, "Bluetooth Pad",
				[]int{btnSouth, btnEast, btnNorth, btnWest, btnTL, btnTR, btnDpadUp, btnDpadDown, btnDpadLeft, btnDpadRite, btnTL2, btnTR2},
				[]int{absX, absY, absZ, absRZ, absBrake, absGas}, map[int]absInfo{absX: stick, absY: stick, absZ: stick, absRZ: stick, absBrake: trigger, absGas: trigger}),
			want: "a:b0,b:b1,x:b2,y:b3,leftshoulder:b4,rightshoulder:b5,lefttrigger:a5,righttrigger:a4,dpup:b8,dpdown:b9,dpleft:b10,dpright:b11,leftx:a0,lefty:a1,rightx:a2,righty:a3,",
		},
		{
			name: "triggers that are only buttons",
			c:    padCaps(0x2563, 0x0576, "Digital Pad", []int{btnSouth, btnTL, btnTR, btnTL2, btnTR2}, []int{absX, absY}, map[int]absInfo{absX: stick, absY: stick}),
			want: "a:b0,leftshoulder:b1,rightshoulder:b2,lefttrigger:b3,righttrigger:b4,leftx:a0,lefty:a1,",
		},
		{
			name: "an analog D-pad is the halves of an axis",
			c: padCaps(0x2563, 0x0577, "Analog Dpad Pad", []int{btnSouth}, []int{absX, absY, absHat0X, absHat0Y},
				map[int]absInfo{absX: stick, absY: stick, absHat0X: {Minimum: -127, Maximum: 127, Fuzz: 4, Flat: 8}, absHat0Y: {Minimum: -127, Maximum: 127, Fuzz: 4, Flat: 8}}),
			want: "a:b0,dpleft:-a2,dpright:+a2,dpup:-a3,dpdown:+a3,leftx:a0,lefty:a1,",
		},
		{
			name: "xpad reports the D-pad as buttons",
			c: func() *caps {
				c := padCaps(0x045e, 0x028e, "xpad", []int{btnSouth, btnTrigHap1, btnTrigHap2, btnTrigHap3, btnTrigHap4}, []int{absX, absY}, map[int]absInfo{absX: stick, absY: stick})
				c.Driver = "xpad"
				return c
			}(),
			want: "a:b0,dpleft:b1,dpright:b2,dpup:b3,dpdown:b4,leftx:a0,lefty:a1,",
		},
		{
			name: "the paddles and the share button",
			c:    padCaps(0x1234, 3, "Paddles", []int{btnSouth, btnGripL, btnGripR, btnGripL2, btnGripR2, keyRecord}, []int{absX, absY}, map[int]absInfo{absX: stick, absY: stick}),
			want: "a:b0,paddle1:b2,paddle2:b1,paddle3:b4,paddle4:b3,misc1:b5,leftx:a0,lefty:a1,",
		},
		{
			name: "an Xbox Elite pad of old drivers has its paddles as trigger buttons",
			c:    padCaps(vendorMicrosoft, 0x02e3, "Elite", []int{btnSouth, btnTrigHap5, btnTrigHap6, btnTrigHap7, btnTrigHap8}, []int{absX, absY}, map[int]absInfo{absX: stick, absY: stick}),
			want: "a:b0,paddle1:b1,paddle2:b3,paddle3:b2,paddle4:b4,leftx:a0,lefty:a1,",
		},
		{
			name: "a device that is not a gamepad by the specification has no mapping",
			c:    padCaps(0x046d, 0xc215, "Logitech Extreme 3D", []int{btnTrigger, btnTrigger + 1}, []int{absX, absY}, map[int]absInfo{absX: stick, absY: stick}),
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mappingOf(tt.c, newLayout(tt.c))
			if !reflect.DeepEqual(fields(got), fields(tt.want)) && (got != "" || tt.want != "") {
				t.Errorf("mapping\n got %v\nwant %v", fields(got), fields(tt.want))
			}
		})
	}
}

func TestRemoveField(t *testing.T) {
	if got := removeField("a:b0,paddle1:b2,b:b1,", "paddle1"); got != "a:b0,b:b1," {
		t.Errorf("removeField() = %q", got)
	}
	if got := removeField("paddle1:b2,", "paddle1"); got != "" {
		t.Errorf("removeField() = %q, want nothing", got)
	}
}

// fakeSource is a device whose state the decoder asks for.
type fakeSource struct {
	abs  map[int]int32
	keys bits
}

func (f fakeSource) absValue(code int) (int32, bool) { v, ok := f.abs[code]; return v, ok }
func (f fakeSource) keyState() (bits, bool)          { return f.keys, f.keys != nil }

func ev(sec int, typ, code uint16, value int32) inputEvent {
	return inputEvent{Time: time.Duration(sec) * time.Second, Type: typ, Code: code, Value: value}
}

func decode(d *decoder, src source, events ...inputEvent) []platform.PadEvent {
	for _, e := range events {
		d.handle(e, 99*time.Second, src)
	}
	return d.take()
}

func pe(sec int, kind platform.PadEventKind, index, value int) platform.PadEvent {
	return platform.PadEvent{Pad: 7, Time: time.Duration(sec) * time.Second, Kind: kind, Index: index, Value: value}
}

func TestDecodeButtonsAndAxes(t *testing.T) {
	l := newLayout(xbox360())
	d := newDecoder(7, l)
	tests := []struct {
		name   string
		events []inputEvent
		want   []platform.PadEvent
	}{
		{"a button goes down and up", []inputEvent{ev(1, evKey, btnSouth, 1), ev(2, evKey, btnSouth, 0)},
			[]platform.PadEvent{pe(1, platform.PadButton, 0, 1), pe(2, platform.PadButton, 0, 0)}},
		{"a key that is repeated counts as down", []inputEvent{ev(1, evKey, btnWest, 2)}, []platform.PadEvent{pe(1, platform.PadButton, 3, 1)}},
		{"a key the pad does not have is dropped", []inputEvent{ev(1, evKey, btnTL2, 1), ev(1, evKey, keyMax, 1)}, nil},
		{"an axis is scaled to the full range", []inputEvent{ev(3, evAbs, absX, 32767), ev(3, evAbs, absY, -32768), ev(3, evAbs, absX, 0)},
			[]platform.PadEvent{pe(3, platform.PadAxis, 0, 32767), pe(3, platform.PadAxis, 1, -32768), pe(3, platform.PadAxis, 0, 0)}},
		{"a trigger from 0 to 255", []inputEvent{ev(3, evAbs, absZ, 0), ev(3, evAbs, absZ, 255), ev(3, evAbs, absZ, 128)},
			[]platform.PadEvent{pe(3, platform.PadAxis, 2, -32768), pe(3, platform.PadAxis, 2, 32767), pe(3, platform.PadAxis, 2, 128)}},
		{"an axis the pad does not have is dropped", []inputEvent{ev(3, evAbs, absWheel, 5)}, nil},
		{"events of other kinds are dropped", []inputEvent{ev(3, evRel, relX, 5), ev(3, evMsc, 5, 5), ev(3, evSyn, synReport, 0)}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decode(d, fakeSource{}, tt.events...)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("reports\n got %+v\nwant %+v", got, tt.want)
			}
		})
	}
}

func TestDecodeHats(t *testing.T) {
	d := newDecoder(7, newLayout(xbox360()))
	const (
		up    = 1
		right = 2
		down  = 4
		left  = 8
	)
	steps := []struct {
		e    inputEvent
		want []int
	}{
		{ev(1, evAbs, absHat0Y, -1), []int{up}},
		{ev(2, evAbs, absHat0X, 1), []int{up | right}},
		{ev(3, evAbs, absHat0Y, 0), []int{right}},
		{ev(4, evAbs, absHat0X, 0), []int{0}},
		{ev(5, evAbs, absHat0X, -1), []int{left}},
		{ev(6, evAbs, absHat0Y, 1), []int{left | down}},
		{ev(7, evAbs, absHat0Y, 1), nil},
		{ev(8, evAbs, absHat1X, 1), nil},
	}
	for i, s := range steps {
		got := decode(d, fakeSource{}, s.e)
		var values []int
		for _, r := range got {
			if r.Kind != platform.PadHat || r.Index != 0 {
				t.Fatalf("step %d: report %+v is not hat 0", i, r)
			}
			values = append(values, r.Value)
		}
		if !reflect.DeepEqual(values, s.want) {
			t.Errorf("step %d: hat = %v, want %v", i, values, s.want)
		}
	}
}

func TestDecodeHatThatIsReallyAnalogHasDeadZones(t *testing.T) {
	// Taken for a hat by its range, but its values run over a range wider than -1..1: small values are centre.
	c := padCaps(1, 1, "Wide hat", []int{btnSouth}, []int{absX, absHat0X}, map[int]absInfo{absX: stick, absHat0X: {Minimum: -90, Maximum: 90}})
	d := newDecoder(7, newLayout(c))
	steps := []struct {
		value int32
		want  []int
	}{
		{20, nil},      // within a third of the range: centred
		{-20, nil},     // the same on the other side
		{60, []int{2}}, // beyond a third: right
		{0, []int{0}},
		{-80, []int{8}},
		{-90, nil}, // the minimum is reached: still left
	}
	for i, s := range steps {
		var got []int
		for _, r := range decode(d, fakeSource{}, ev(i, evAbs, absHat0X, s.value)) {
			got = append(got, r.Value)
		}
		if !reflect.DeepEqual(got, s.want) {
			t.Errorf("step %d (%d): %v, want %v", i, s.value, got, s.want)
		}
	}
}

func TestDecodeDroppedEvents(t *testing.T) {
	l := newLayout(xbox360())
	d := newDecoder(7, l)
	src := fakeSource{
		abs:  map[int]int32{absX: 32767, absY: 0, absZ: 255, absRX: 0, absRY: 0, absRZ: 0, absHat0X: 1, absHat0Y: 0},
		keys: bitsOf(btnSouth, btnMode),
	}
	got := decode(d, src,
		ev(1, evKey, btnEast, 1), // before the drop: reported
		ev(2, evSyn, synDropped, 0),
		ev(2, evKey, btnNorth, 1), // the rest of the packet is ignored
		ev(2, evAbs, absX, 5),
		ev(2, evSyn, synReport, 0), // the end of the packet: the state is read again
	)
	if got[0] != pe(1, platform.PadButton, 1, 1) {
		t.Fatalf("the first report = %+v", got[0])
	}
	var axes, hats, buttons = map[int]int{}, map[int]int{}, map[int]int{}
	for _, r := range got[1:] {
		if r.Time != 99*time.Second {
			t.Errorf("report %+v: the reports of a re-read have the time of the read", r)
		}
		switch r.Kind {
		case platform.PadAxis:
			axes[r.Index] = r.Value
		case platform.PadHat:
			hats[r.Index] = r.Value
		case platform.PadButton:
			buttons[r.Index] = r.Value
		}
	}
	if axes[0] != 32767 || axes[2] != 32767 || axes[1] != 0 || len(axes) != 6 {
		t.Errorf("axes after the drop = %v", axes)
	}
	if hats[0] != 2 {
		t.Errorf("hat after the drop = %v, want right", hats)
	}
	if buttons[0] != 1 || buttons[8] != 1 || buttons[1] != 0 || buttons[2] != 0 || len(buttons) != 11 {
		t.Errorf("buttons after the drop = %v", buttons)
	}
	// After the report the events flow again.
	if got := decode(d, src, ev(3, evKey, btnNorth, 1)); len(got) != 1 {
		t.Errorf("events after the recovery = %+v", got)
	}
}

func TestPollAllIsWhatTheFirstReportIs(t *testing.T) {
	d := newDecoder(7, newLayout(xbox360()))
	d.pollAll(5*time.Second, fakeSource{abs: map[int]int32{absX: 0}, keys: bits{0}})
	got := d.take()
	if len(got) != 1+11 {
		t.Fatalf("got %d reports, want the one axis that could be read and the 11 buttons: %+v", len(got), got)
	}
	for _, r := range got {
		if r.Time != 5*time.Second || r.Pad != 7 {
			t.Errorf("report %+v", r)
		}
	}
	if got[0].Kind != platform.PadAxis || got[0].Value != 0 {
		t.Errorf("first report = %+v", got[0])
	}
}

func TestAxisValue(t *testing.T) {
	l := newLayout(padCaps(1, 1, "Axes", []int{btnSouth}, []int{absX, absY, absZ, absRX},
		map[int]absInfo{absX: {Minimum: 0, Maximum: 255}, absY: {Minimum: -100, Maximum: 100}, absZ: {Minimum: 5, Maximum: 5}, absRX: {Minimum: -32768, Maximum: 32767}}))
	tests := []struct {
		code, in, want int
	}{
		{absX, 0, -32768}, {absX, 255, 32767}, {absX, 128, 128}, {absX, 300, 32767}, {absX, -50, -32768},
		{absY, -100, -32768}, {absY, 100, 32767}, {absY, 50, 16383}, {absY, 0, -1}, // 0 is -1: the float32 scale of SDL truncates
		{absZ, 5, 5}, {absZ, 70000, 32767},
		{absRX, 12345, 12345}, {absRX, -32768, -32768},
	}
	for _, tt := range tests {
		if got := l.axisValue(tt.code, tt.in); got != tt.want {
			t.Errorf("axisValue(0x%x, %d) = %d, want %d", tt.code, tt.in, got, tt.want)
		}
	}
}

func TestFixupForMapping(t *testing.T) {
	elite := &caps{ID: inputID{Vendor: 0x045e, Product: 0x0b05, Version: 0x0903}, Key: bitsOf(0x2c0)}
	fixupForMapping(elite)
	if elite.ID.Version != 0x0902 {
		t.Errorf("Elite 2 duplicating firmware: version = %#x, want 0x902", elite.ID.Version)
	}
	elite = &caps{ID: inputID{Vendor: 0x045e, Product: 0x0b05, Version: 0x0903}}
	fixupForMapping(elite)
	if elite.ID.Version != 0x0903 {
		t.Errorf("Elite 2 without the duplicates: version = %#x, want 0x903", elite.ID.Version)
	}
	atari := &caps{ID: inputID{Vendor: 0x3250, Product: 0x1002, Version: 0x0300}}
	fixupForMapping(atari)
	if atari.ID.Version != 0 {
		t.Errorf("Atari VCS: version = %#x, want 0", atari.ID.Version)
	}
	if !ignoreDevice(&caps{ID: inputID{Vendor: 0x0b05, Product: 0x18e3}}) || ignoreDevice(&caps{ID: inputID{Vendor: 0x045e, Product: 0x028e}}) {
		t.Error("ignoreDevice() is wrong")
	}
}

func TestStructLayouts(t *testing.T) {
	// The sizes and offsets are those of linux/input.h (struct input_absinfo and struct input_id).
	layouttest.AssertLayout(t, "input_absinfo", 24, map[string]uintptr{"Value": 0, "Minimum": 4, "Maximum": 8, "Fuzz": 12, "Flat": 16, "Resolution": 20}, absInfo{})
	layouttest.AssertLayout(t, "input_id", 8, map[string]uintptr{"Bus": 0, "Vendor": 2, "Product": 4, "Version": 6}, inputID{})
}
