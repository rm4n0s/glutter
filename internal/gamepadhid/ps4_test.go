package gamepadhid

import (
	"math"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/gamepaddb"
	"github.com/rm4n0s/glutter/internal/platform"
)

// ps4Packet is the state packet of a PlayStation 4 pad (PS4StatePacket_t of SDL, 54 bytes).
type ps4Packet struct {
	lx, ly, rx, ry, tl, tr uint8
	b0, b1, b2             uint8
	ts                     uint16
	gyro, accel            [3]int16
	battery                uint8
	t1, t2                 [4]byte
}

func putI16(b []byte, v int16) { b[0], b[1] = byte(v), byte(v>>8) }

func (p ps4Packet) bytes() []byte {
	b := make([]byte, ps4StateSize)
	b[0], b[1], b[2], b[3] = p.lx, p.ly, p.rx, p.ry
	b[4], b[5], b[6] = p.b0, p.b1, p.b2
	b[7], b[8] = p.tl, p.tr
	putI16(b[9:], int16(p.ts))
	for i := 0; i < 3; i++ {
		putI16(b[12+2*i:], p.gyro[i])
		putI16(b[18+2*i:], p.accel[i])
	}
	b[29] = p.battery
	copy(b[34:38], p.t1[:])
	copy(b[38:42], p.t2[:])
	return b
}

// usb is the input report of a pad on USB: the ID and the packet, 64 bytes.
func (p ps4Packet) usb() []byte {
	r := make([]byte, 64)
	r[0] = ps4ReportUSBState
	copy(r[1:], p.bytes())
	return r
}

// bluetooth is the input report of a pad on Bluetooth: the ID, two bytes of flags, the packet and the CRC, 78 bytes.
func (p ps4Packet) bluetooth() []byte {
	r := make([]byte, 78)
	r[0], r[1] = ps4ReportBluetoothState1, 0xC0
	copy(r[3:], p.bytes())
	return withCRC(0xA1, r)
}

func sonyPS4(bus Bus) DeviceInfo {
	return DeviceInfo{Vendor: vendorSony, Product: 0x09cc, Bus: bus, Name: "Wireless Controller"}
}

func startPS4(t *testing.T, info DeviceInfo, io *fakeIO) (*ps4, Caps) {
	t.Helper()
	p := newPS4(info).(*ps4)
	caps, err := p.Start(io)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	return p, caps
}

func TestPS4StartSendsTheEffectsThatSwitchToTheEnhancedMode(t *testing.T) {
	// The effects report with no flags: the pad's light is blue (player 0) and the motors are off.
	usb := make([]byte, 32)
	usb[0], usb[8] = 0x05, 0x40
	bt := make([]byte, 78)
	bt[0], bt[1], bt[10] = 0x11, 0xC4, 0x40
	putCRCBytes(bt)

	for _, tt := range []struct {
		name  string
		bus   Bus
		reads [][]byte
		want  []byte
	}{
		{"usb", BusUSB, nil, usb},
		{"bluetooth with enhanced reports", BusBluetooth, [][]byte{ps4Packet{}.bluetooth()}, bt},
		{"bluetooth with simple reports", BusBluetooth, [][]byte{{1, 0, 0, 0, 0, 0x08, 0, 0, 0, 0}}, bt},
	} {
		t.Run(tt.name, func(t *testing.T) {
			io := &fakeIO{reads: tt.reads}
			_, caps := startPS4(t, sonyPS4(tt.bus), io)
			if len(io.sends) != 1 {
				t.Fatalf("%d reports were sent, want 1", len(io.sends))
			}
			mustEqual(t, "effects report", io.sends[0], tt.want)
			want := Caps{Name: "PS4 Controller", Kind: gamepaddb.KindPS4, Rumble: true, LED: true, Sensors: true, Rate: 250, Touchpads: 1, Buttons: ps4Buttons}
			if caps != want {
				t.Errorf("caps = %+v, want %+v", caps, want)
			}
		})
	}
}

// putCRCBytes writes the CRC that the pad is sent (header 0xA2) in the last four bytes of the report.
func putCRCBytes(r []byte) {
	n := len(r) - 4
	c := slowCRC32([]byte{0xA2}, r[:n])
	r[n], r[n+1], r[n+2], r[n+3] = byte(c), byte(c>>8), byte(c>>16), byte(c>>24)
}

func TestPS4ReportsTheButtonsAndAxes(t *testing.T) {
	for _, tt := range []struct {
		name    string
		pk      ps4Packet
		buttons []gamepaddb.Button
		axes    map[gamepaddb.Axis]int16
	}{
		{"nothing is pressed", ps4Packet{lx: 128, ly: 128, rx: 128, ry: 128, b0: 0x08},
			nil, map[gamepaddb.Axis]int16{gamepaddb.LeftX: 128, gamepaddb.LeftY: 128, gamepaddb.RightX: 128, gamepaddb.RightY: 128}},
		{"the face buttons are the cross, circle, square and triangle", ps4Packet{b0: 0xF8},
			[]gamepaddb.Button{gamepaddb.South, gamepaddb.East, gamepaddb.West, gamepaddb.North}, nil},
		{"the cross alone", ps4Packet{b0: 0x28}, []gamepaddb.Button{gamepaddb.South}, nil},
		{"the hat up and right", ps4Packet{b0: 0x01}, []gamepaddb.Button{gamepaddb.DPadUp, gamepaddb.DPadRight}, nil},
		{"the hat left", ps4Packet{b0: 0x06}, []gamepaddb.Button{gamepaddb.DPadLeft}, nil},
		{"the shoulders, share, options and the sticks", ps4Packet{b0: 0x08, b1: 0xF3},
			[]gamepaddb.Button{gamepaddb.LeftShoulder, gamepaddb.RightShoulder, gamepaddb.Back, gamepaddb.Start, gamepaddb.LeftStick, gamepaddb.RightStick}, nil},
		{"the PS button and the touchpad", ps4Packet{b0: 0x08, b2: 0x03}, []gamepaddb.Button{gamepaddb.Guide, gamepaddb.Touchpad}, nil},
		{"the triggers", ps4Packet{b0: 0x08, tl: 255, tr: 128},
			nil, map[gamepaddb.Axis]int16{gamepaddb.LeftTrigger: 32767, gamepaddb.RightTrigger: 16447}},
		{"a trigger that is a button only", ps4Packet{b0: 0x08, b1: 0x04}, nil, map[gamepaddb.Axis]int16{gamepaddb.LeftTrigger: 32767}},
		{"the sticks at their ends", ps4Packet{b0: 0x08, lx: 0, ly: 255}, nil, map[gamepaddb.Axis]int16{gamepaddb.LeftX: -32768, gamepaddb.LeftY: 32767}},
	} {
		for _, bus := range []struct {
			name string
			bus  Bus
			rep  func(ps4Packet) []byte
		}{{"usb", BusUSB, ps4Packet.usb}, {"bluetooth", BusBluetooth, ps4Packet.bluetooth}} {
			t.Run(tt.name+" "+bus.name, func(t *testing.T) {
				io := &fakeIO{}
				if bus.bus == BusBluetooth {
					io.reads = [][]byte{ps4Packet{}.bluetooth()}
				}
				p, _ := startPS4(t, sonyPS4(bus.bus), io)
				var out recSink
				p.Report(time.Second, bus.rep(tt.pk), &out)
				s := out.lastState(t)
				want := map[gamepaddb.Button]bool{}
				for _, b := range tt.buttons {
					want[b] = true
				}
				for b := gamepaddb.Button(0); b < gamepaddb.ButtonCount; b++ {
					if s.Buttons[b] != want[b] {
						t.Errorf("button %d = %v, want %v", b, s.Buttons[b], want[b])
					}
				}
				for a, v := range tt.axes {
					if s.Axes[a] != padAxis(a, v) {
						t.Errorf("axis %d = %d, want %d", a, s.Axes[a], padAxis(a, v))
					}
				}
			})
		}
	}
}

// padAxis is the axis that a test expects: a stick is given as the byte the pad sends (0 to 255) except at its ends, a trigger as
// the axis.
func padAxis(a gamepaddb.Axis, v int16) int16 {
	switch a {
	case gamepaddb.LeftTrigger, gamepaddb.RightTrigger:
		return v
	}
	if v == -32768 || v == 32767 {
		return v
	}
	return int16(int(v)*257 - 32768)
}

func TestPS4SimpleReportOf10Bytes(t *testing.T) {
	io := &fakeIO{}
	p, _ := startPS4(t, sonyPS4(BusBluetooth), io)
	var out recSink
	// The report of a Bluetooth pad that is not in enhanced mode: 10 bytes, no touchpad, no sensors, no battery.
	p.Report(time.Second, []byte{1, 0, 255, 128, 128, 0x28, 0x01, 0x00, 0, 0}, &out)
	s := out.lastState(t)
	if !s.Buttons[gamepaddb.South] || s.Axes[gamepaddb.LeftY] != 32767 {
		t.Errorf("state = %+v", s)
	}
	if len(out.touches)+len(out.sensors)+len(out.batteries) != 0 {
		t.Errorf("a simple report held more than the buttons: %+v", out)
	}
}

func TestPS4Touchpad(t *testing.T) {
	io := &fakeIO{}
	p, _ := startPS4(t, sonyPS4(BusUSB), io)
	var out recSink
	// A finger at x=960 (half), y=460 (half) on the first slot; the second slot is up (bit 7 of its counter is set).
	x, y := 960, 460
	pk := ps4Packet{b0: 0x08, t1: [4]byte{0x05, byte(x), byte(x>>8)&0x0F | byte(y&0x0F)<<4, byte(y >> 4)}, t2: [4]byte{0x80}}
	p.Report(time.Second, pk.usb(), &out)
	if len(out.touches) != 1 {
		t.Fatalf("%d touch events, want 1: %+v", len(out.touches), out.touches)
	}
	got := out.touches[0]
	if !got.Down || got.Finger != 0 || got.Touchpad != 0 || math.Abs(float64(got.X)-0.5) > 0.001 || math.Abs(float64(got.Y)-0.5) > 0.001 || got.Pressure != 1 {
		t.Errorf("touch = %+v", got)
	}
	// The same report again changes nothing.
	p.Report(2*time.Second, pk.usb(), &out)
	if len(out.touches) != 1 {
		t.Errorf("a finger that did not move was reported again: %+v", out.touches)
	}
	// The finger leaves: it is reported up, at the place it was.
	pk.t1 = [4]byte{0x80, byte(x), byte(x>>8)&0x0F | byte(y&0x0F)<<4, byte(y >> 4)}
	p.Report(3*time.Second, pk.usb(), &out)
	if len(out.touches) != 2 || out.touches[1].Down || out.touches[1].Pressure != 0 || out.touches[1].X != got.X {
		t.Errorf("touches = %+v", out.touches)
	}
}

func TestPS4Battery(t *testing.T) {
	for _, tt := range []struct {
		name    string
		battery uint8
		state   platform.PadPowerState
		percent int
	}{
		{"on the battery, half", 0x04, platform.PadOnBattery, 45},
		{"on the battery, full", 0x08, platform.PadOnBattery, 85},
		{"on the battery, the highest level", 0x0B, platform.PadOnBattery, 100},
		{"charging", 0x15, platform.PadCharging, 55},
		{"charged", 0x1B, platform.PadCharged, 100},
		{"a level that means something else", 0x1E, platform.PadPowerUnknown, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p, _ := startPS4(t, sonyPS4(BusUSB), &fakeIO{})
			var out recSink
			p.Report(time.Second, ps4Packet{b0: 0x08, battery: tt.battery}.usb(), &out)
			if len(out.batteries) != 1 || out.batteries[0].State != tt.state || out.batteries[0].Percent != tt.percent {
				t.Errorf("batteries = %+v", out.batteries)
			}
			p.Report(2*time.Second, ps4Packet{b0: 0x08, battery: tt.battery}.usb(), &out)
			if len(out.batteries) != 1 {
				t.Errorf("an unchanged battery was reported again: %+v", out.batteries)
			}
		})
	}
}

// ps4Calibration makes the feature report of the calibration of a pad that is perfect: the scales are 1 and the biases 0.
func ps4Calibration(bluetooth bool) []byte {
	r := make([]byte, 37)
	r[0] = 0x02
	// Biases: 0. Plus and minus of each gyroscope axis: 8640 and -8640.
	put := func(o int, v int16) { putI16(r[o:], v) }
	if bluetooth {
		put(7, 8640)
		put(9, 8640)
		put(11, 8640)
		put(13, -8640)
		put(15, -8640)
		put(17, -8640)
	} else {
		put(7, 8640)
		put(9, -8640)
		put(11, 8640)
		put(13, -8640)
		put(15, 8640)
		put(17, -8640)
	}
	put(19, 540) // the speed, plus and minus
	put(21, 540)
	put(23, 8192) // the accelerometer, plus and minus of each axis
	put(25, -8192)
	put(27, 8192)
	put(29, -8192)
	put(31, 8192)
	put(33, -8192)
	return r
}

func TestPS4Sensors(t *testing.T) {
	const degree = math.Pi / 180
	for _, tt := range []struct {
		name string
		bus  Bus
		cal  map[byte][]byte
		// gyroScale and accelScale are what a raw number is worth in rad/s and m/s².
		gyroScale, accelScale float64
	}{
		{"usb with the pad's calibration", BusUSB, map[byte][]byte{0x02: ps4Calibration(false)}, degree / 16, standardGravity / 8192},
		{"bluetooth with the pad's calibration", BusBluetooth, map[byte][]byte{0x02: ps4Calibration(true), 0x05: ps4Calibration(true)}, degree / 16, standardGravity / 8192},
		{"a pad that does not give its calibration", BusUSB, nil, degree / 16, standardGravity / 8192},
	} {
		t.Run(tt.name, func(t *testing.T) {
			io := &fakeIO{features: tt.cal}
			if tt.bus == BusBluetooth {
				io.reads = [][]byte{ps4Packet{}.bluetooth()}
			}
			p, _ := startPS4(t, sonyPS4(tt.bus), io)
			var out recSink
			// Sensors are off until they are asked for.
			pk := ps4Packet{b0: 0x08, ts: 100, gyro: [3]int16{160, -320, 0}, accel: [3]int16{0, 8192, -4096}}
			report := func() []byte {
				if tt.bus == BusBluetooth {
					return pk.bluetooth()
				}
				return pk.usb()
			}
			p.Report(time.Second, report(), &out)
			if len(out.sensors) != 0 {
				t.Fatalf("sensors were reported before they were asked for: %+v", out.sensors)
			}
			if err := p.SetSensors(true); err != nil {
				t.Fatal(err)
			}
			pk.ts = 100 + 3*16 // 48 ticks of 5.33 µs
			p.Report(2*time.Second, report(), &out)
			if len(out.sensors) != 2 {
				t.Fatalf("%d sensor readings, want 2", len(out.sensors))
			}
			g, a := out.sensors[0], out.sensors[1]
			if g.Kind != platform.PadGyroscope || a.Kind != platform.PadAccelerometer {
				t.Fatalf("kinds = %d, %d", g.Kind, a.Kind)
			}
			for i, raw := range []float64{160, -320, 0} {
				if want := raw * tt.gyroScale; math.Abs(float64(g.Values[i])-want) > 1e-4 {
					t.Errorf("gyro[%d] = %v, want %v", i, g.Values[i], want)
				}
			}
			for i, raw := range []float64{0, 8192, -4096} {
				if want := raw * tt.accelScale; math.Abs(float64(a.Values[i])-want) > 1e-3 {
					t.Errorf("accel[%d] = %v, want %v", i, a.Values[i], want)
				}
			}
			// The sensor's clock: 148 ticks since the start, 5.33 µs each... the first report set the count at 100.
			if want := time.Duration(uint64(100+3*16) * 1000 * 16 / 3); g.SensorTime != want {
				t.Errorf("sensor time = %v, want %v", g.SensorTime, want)
			}
			if err := p.SetSensors(false); err != nil {
				t.Fatal(err)
			}
			p.Report(3*time.Second, report(), &out)
			if len(out.sensors) != 2 {
				t.Errorf("sensors were reported after they were turned off")
			}
		})
	}
}

func TestPS4BadCalibrationIsIgnored(t *testing.T) {
	cal := ps4Calibration(false)
	putI16(cal[1:], 4000) // a bias that is far off
	io := &fakeIO{features: map[byte][]byte{0x02: cal}}
	p, _ := startPS4(t, sonyPS4(BusUSB), io)
	if err := p.SetSensors(true); err != nil {
		t.Fatal(err)
	}
	if p.hardwareCalibration {
		t.Error("a calibration with a bias of 4000 was taken")
	}
	var out recSink
	p.Report(time.Second, ps4Packet{b0: 0x08, gyro: [3]int16{160, 0, 0}}.usb(), &out)
	// Without the calibration of the pad a raw number is worth a sixteenth of a degree a second.
	if got := float64(out.sensors[0].Values[0]); math.Abs(got-10*math.Pi/180) > 1e-4 {
		t.Errorf("gyro = %v", got)
	}
}

func TestPS4RumbleAndLightReports(t *testing.T) {
	io := &fakeIO{}
	p, _ := startPS4(t, sonyPS4(BusUSB), io)
	if err := p.Rumble(0xFFFF, 0x8000); err != nil {
		t.Fatal(err)
	}
	// USB: the report 5, the flag for the rumble, the right motor (the small one) first and then the left, the light.
	want := make([]byte, 32)
	want[0], want[1] = 0x05, 0x01
	want[4], want[5] = 0x80, 0xFF
	want[8] = 0x40
	mustEqual(t, "rumble report", io.lastSend(t), want)

	if err := p.SetLED(1, 2, 3); err != nil {
		t.Fatal(err)
	}
	want = make([]byte, 32)
	want[0], want[1] = 0x05, 0x02
	want[4], want[5] = 0x80, 0xFF
	want[6], want[7], want[8] = 1, 2, 3
	mustEqual(t, "light report", io.lastSend(t), want)

	if err := p.Rumble(0, 0); err != nil {
		t.Fatal(err)
	}
	want[1], want[4], want[5] = 0x01, 0, 0
	mustEqual(t, "stop report", io.lastSend(t), want)

	if err := p.RumbleTriggers(1, 1); err == nil {
		t.Error("a PlayStation 4 pad rumbled its triggers")
	}
}

func TestPS4BluetoothRumbleReportHasItsCRC(t *testing.T) {
	io := &fakeIO{reads: [][]byte{ps4Packet{}.bluetooth()}}
	p, _ := startPS4(t, sonyPS4(BusBluetooth), io)
	if err := p.Rumble(0x4000, 0xC000); err != nil {
		t.Fatal(err)
	}
	want := make([]byte, 78)
	want[0], want[1], want[3] = 0x11, 0xC4, 0x01
	want[6], want[7], want[10] = 0xC0, 0x40, 0x40
	putCRCBytes(want)
	mustEqual(t, "rumble report", io.lastSend(t), want)
}

func TestPS4BluetoothSilenceIsPoked(t *testing.T) {
	io := &fakeIO{reads: [][]byte{ps4Packet{}.bluetooth()}}
	p, _ := startPS4(t, sonyPS4(BusBluetooth), io)
	sent := len(io.sends)
	var out recSink
	p.Tick(100*time.Millisecond, &out)
	if len(io.sends) != sent {
		t.Fatal("a pad that was quiet for 100 ms was poked")
	}
	p.Tick(700*time.Millisecond, &out)
	if len(io.sends) != sent+1 {
		t.Fatalf("%d reports were sent, want %d", len(io.sends), sent+1)
	}
	// The poke has no CRC, so the pad ignores it.
	want := make([]byte, 78)
	want[0], want[1] = 0x11, 0xC0
	mustEqual(t, "poke", io.lastSend(t), want)
	p.Tick(800*time.Millisecond, &out)
	if len(io.sends) != sent+1 {
		t.Error("the poke was sent again at once")
	}
}

func TestPS4BluetoothChecksTheCRC(t *testing.T) {
	io := &fakeIO{reads: [][]byte{ps4Packet{}.bluetooth()}}
	p, _ := startPS4(t, sonyPS4(BusBluetooth), io)
	var out recSink
	good := ps4Packet{b0: 0x28}.bluetooth()
	bad := ps4Packet{b0: 0x48}.bluetooth()
	bad[77] ^= 0xFF
	// Before the pad has sent good reports a bad CRC is let through (some pads and Linux's own reports have none).
	p.Report(1*time.Second, bad, &out)
	if len(out.states) != 1 {
		t.Fatal("a report with a bad CRC from a pad that sent no good ones was dropped")
	}
	// Once it has sent several good ones (the count goes down by one for each bad one) a bad one is dropped.
	for i := 0; i < 5; i++ {
		p.Report(time.Second, good, &out)
	}
	n := len(out.states)
	p.Report(2*time.Second, bad, &out)
	if len(out.states) != n {
		t.Error("a report with a bad CRC was taken after good ones")
	}
	// A report that is not a state report, or too short, is dropped.
	p.Report(2*time.Second, []byte{0x55, 0, 0}, &out)
	p.Report(2*time.Second, good[:40], &out)
	if len(out.states) != n {
		t.Error("a report of another kind or a short one was taken")
	}
}

func TestPS4ThirdPartyPadAsksForItsCapabilities(t *testing.T) {
	// The capabilities report: 48 bytes, 0x27 in byte 2, the capabilities in byte 4 (0x02 sensors, 0x04 light bar, 0x08 vibration,
	// 0x40 touchpad), and the scales of the sensors in 10 to 17.
	feat := make([]byte, 48)
	feat[0], feat[2], feat[4] = 0x03, 0x27, 0x02|0x08
	putI16(feat[10:], 1)
	putI16(feat[12:], 32)
	putI16(feat[14:], 1)
	putI16(feat[16:], 4096)
	info := DeviceInfo{Vendor: 0x0f0d, Product: 0x0000, Bus: BusUSB, Name: "Fighting Pad"}
	io := &fakeIO{features: map[byte][]byte{0x03: feat}}
	p := newPS4(info).(*ps4)
	caps, err := p.Start(io)
	if err != nil {
		t.Fatal(err)
	}
	if caps.Name != "" || !caps.Rumble || caps.LED || !caps.Sensors || caps.Touchpads != 0 {
		t.Errorf("caps = %+v", caps)
	}
	if p.gyroDen != 32 || p.accelDen != 4096 {
		t.Errorf("scales = %d, %d", p.gyroDen, p.accelDen)
	}
	// A third party pad gets all the flags, as it may expect rumble and light together.
	if len(io.sends) != 1 || io.sends[0][1] != 0x07 {
		t.Errorf("effects = %x", io.sends)
	}
	if err := p.SetLED(1, 2, 3); err == nil {
		t.Error("a pad without a light bar took a colour")
	}
}

func TestPS4PadThatTakesNoEffects(t *testing.T) {
	// The Victrix FS Pro hangs when it is sent an effects report.
	info := DeviceInfo{Vendor: vendorPDP, Product: productVictrixFSPro, Bus: BusUSB, Name: "FS Pro"}
	feat := make([]byte, 48)
	feat[0], feat[2], feat[4] = 0x03, 0x27, 0x04|0x08
	io := &fakeIO{features: map[byte][]byte{0x03: feat}}
	p := newPS4(info).(*ps4)
	if _, err := p.Start(io); err != nil {
		t.Fatal(err)
	}
	if len(io.sends) != 0 {
		t.Errorf("a pad that takes no effects was sent %x", io.sends)
	}
	if err := p.Rumble(1, 1); err == nil {
		t.Error("a pad that takes no effects rumbled")
	}
}

// A report that asks for the light does not replace a queued one that asks for the rumble, or the rumble would be lost.
func TestPS4QueuedReportsAreMergedOnlyWhenTheyAskForTheSameEffects(t *testing.T) {
	for _, bus := range []Bus{BusUSB, BusBluetooth} {
		io := &fakeIO{}
		p, _ := startPS4(t, sonyPS4(bus), io)
		if err := p.Rumble(0xFFFF, 0); err != nil {
			t.Fatal(err)
		}
		rumble, rumbleMerge := io.lastSend(t), io.merges[len(io.merges)-1]
		if err := p.Rumble(0x8000, 0); err != nil {
			t.Fatal(err)
		}
		if !io.merges[len(io.merges)-1](rumble) {
			t.Errorf("bus %v: a rumble does not replace a queued rumble", bus)
		}
		if err := p.SetLED(1, 2, 3); err != nil {
			t.Fatal(err)
		}
		if io.merges[len(io.merges)-1](rumble) || rumbleMerge(io.lastSend(t)) {
			t.Errorf("bus %v: a report for the light replaces one for the rumble", bus)
		}
	}
}
