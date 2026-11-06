package gamepadhid

import (
	"math"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/gamepaddb"
	"github.com/rm4n0s/glutter/internal/platform"
)

// ps5Packet is the packet of the state of a PlayStation 5 pad (PS5StatePacket_t of SDL, the full layout).
type ps5Packet struct {
	lx, ly, rx, ry, tl, tr uint8
	b0, b1, b2             uint8
	gyro, accel            [3]int16
	ts                     uint32
	t1, t2                 [4]byte
	battery                uint8
}

func (p ps5Packet) bytes() []byte {
	b := make([]byte, ps5StateSize)
	b[0], b[1], b[2], b[3], b[4], b[5] = p.lx, p.ly, p.rx, p.ry, p.tl, p.tr
	b[7], b[8], b[9] = p.b0, p.b1, p.b2
	for i := 0; i < 3; i++ {
		putI16(b[15+2*i:], p.gyro[i])
		putI16(b[21+2*i:], p.accel[i])
	}
	b[27], b[28], b[29], b[30] = byte(p.ts), byte(p.ts>>8), byte(p.ts>>16), byte(p.ts>>24)
	copy(b[32:36], p.t1[:])
	copy(b[36:40], p.t2[:])
	b[52] = p.battery
	return b
}

// usb is the report of a pad on USB: the ID and the packet, 64 bytes.
func (p ps5Packet) usb() []byte {
	r := make([]byte, 64)
	r[0] = ps5ReportState
	copy(r[1:], p.bytes())
	return r
}

// bluetooth is the report of a pad on Bluetooth: the ID 0x31, a byte, the packet, and the CRC at the end of 78 bytes.
func (p ps5Packet) bluetooth() []byte {
	r := make([]byte, 78)
	r[0] = ps5ReportBluetoothState
	copy(r[2:], p.bytes())
	return withCRC(0xA1, r)
}

func sonyPS5(bus Bus) DeviceInfo {
	return DeviceInfo{Vendor: vendorSony, Product: 0x0ce6, Bus: bus, Name: "DualSense Wireless Controller"}
}

// ps5Features are the feature reports of a DualSense: the serial number and the firmware (the version is bytes 44 and 45).
func ps5Features(firmware uint16) map[byte][]byte {
	fw := make([]byte, 64)
	fw[0], fw[44], fw[45] = ps5FeatureFirmware, byte(firmware), byte(firmware>>8)
	serial := make([]byte, 16)
	serial[0] = ps5FeatureSerial
	return map[byte][]byte{ps5FeatureSerial: serial, ps5FeatureFirmware: fw}
}

// ps5Calibration is the calibration of a pad that is perfect: the sensitivity of the gyroscope is 64 and that of the
// accelerometer 1, and the biases are 0.
func ps5Calibration() []byte {
	r := make([]byte, 41)
	r[0] = ps5FeatureCalibration
	put := func(o int, v int16) { putI16(r[o:], v) }
	for _, o := range []int{7, 11, 15} {
		put(o, 8640)
		put(o+2, -8640)
	}
	put(19, 540)
	put(21, 540)
	for _, o := range []int{23, 27, 31} {
		put(o, 8192)
		put(o+2, -8192)
	}
	return r
}

func startPS5(t *testing.T, info DeviceInfo, io *fakeIO) (*ps5, Caps) {
	t.Helper()
	p := newPS5(info).(*ps5)
	caps, err := p.Start(io)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	return p, caps
}

func startedPS5USB(t *testing.T, firmware uint16) (*ps5, *fakeIO) {
	t.Helper()
	io := &fakeIO{reads: [][]byte{make([]byte, 64)}, features: ps5Features(firmware)}
	p, _ := startPS5(t, sonyPS5(BusUSB), io)
	return p, io
}

func TestPS5StartOverUSB(t *testing.T) {
	io := &fakeIO{reads: [][]byte{make([]byte, 64)}, features: ps5Features(0x0224)}
	_, caps := startPS5(t, sonyPS5(BusUSB), io)
	want := Caps{Name: "DualSense Wireless Controller", Kind: gamepaddb.KindPS5, Rumble: true, RumbleTriggers: true, LED: true, Sensors: true,
		Rate: 250, Touchpads: 1, Buttons: ps5Buttons}
	if caps != want {
		t.Errorf("caps = %+v\nwant   %+v", caps, want)
	}
	if len(io.sends) != 2 {
		t.Fatalf("%d reports were sent, want 2", len(io.sends))
	}
	// The first report has no flags: it only switches the pad to the enhanced mode.
	mustEqual(t, "first report", io.sends[0], append([]byte{0x02}, make([]byte, 47)...))
	// The second sets the light (blue, player 0) and the lights of the touchpad (off).
	second := make([]byte, 48)
	second[0], second[2], second[1+43], second[1+46] = 0x02, 0x04|0x10, 0, 0x40
	mustEqual(t, "second report", io.sends[1], second)
}

func TestPS5StartOverBluetoothWaitsForTheConnectionLight(t *testing.T) {
	io := &fakeIO{reads: [][]byte{ps5Packet{}.bluetooth()}, features: ps5Features(0)}
	p, caps := startPS5(t, sonyPS5(BusBluetooth), io)
	if caps.Rate != 1000 {
		t.Errorf("rate = %v, want 1000 over Bluetooth", caps.Rate)
	}
	if len(io.sends) != 1 {
		t.Fatalf("%d reports were sent, want 1: the light waits", len(io.sends))
	}
	first := make([]byte, 78)
	first[0], first[2] = 0x31, 0x10
	putCRCBytes(first)
	mustEqual(t, "first report", io.sends[0], first)

	// A report whose timer says that the connection animation is not over yet changes nothing.
	var out recSink
	p.Report(time.Second, ps5Packet{ts: 1_000_000}.bluetooth(), &out)
	if len(io.sends) != 1 {
		t.Fatal("the light was set before the connection animation was over")
	}
	// When the timer passes 10200000 the light is reset and set.
	p.Report(2*time.Second, ps5Packet{ts: 10_300_000}.bluetooth(), &out)
	if len(io.sends) != 3 {
		t.Fatalf("%d reports were sent, want 3", len(io.sends))
	}
	reset := make([]byte, 78)
	reset[0], reset[2], reset[3+1] = 0x31, 0x10, 0x08
	putCRCBytes(reset)
	mustEqual(t, "reset report", io.sends[1], reset)
	light := make([]byte, 78)
	light[0], light[2], light[3+1], light[3+46] = 0x31, 0x10, 0x04|0x10, 0x40
	putCRCBytes(light)
	mustEqual(t, "light report", io.sends[2], light)
	// And not again.
	p.Report(3*time.Second, ps5Packet{ts: 10_400_000}.bluetooth(), &out)
	if len(io.sends) != 3 {
		t.Error("the light was set again")
	}
}

func TestPS5ReportsTheButtonsAndAxes(t *testing.T) {
	for _, tt := range []struct {
		name    string
		pk      ps5Packet
		buttons []gamepaddb.Button
		axes    map[gamepaddb.Axis]int16
	}{
		{"nothing is pressed", ps5Packet{lx: 128, b0: 0x08}, nil, map[gamepaddb.Axis]int16{gamepaddb.LeftX: 128}},
		{"the face buttons", ps5Packet{b0: 0xF8}, []gamepaddb.Button{gamepaddb.South, gamepaddb.East, gamepaddb.West, gamepaddb.North}, nil},
		{"the hat down and left", ps5Packet{b0: 0x05}, []gamepaddb.Button{gamepaddb.DPadDown, gamepaddb.DPadLeft}, nil},
		{"the shoulders and the middle buttons", ps5Packet{b0: 0x08, b1: 0xF3},
			[]gamepaddb.Button{gamepaddb.LeftShoulder, gamepaddb.RightShoulder, gamepaddb.Back, gamepaddb.Start, gamepaddb.LeftStick, gamepaddb.RightStick}, nil},
		{"the PS button, the touchpad and the microphone", ps5Packet{b0: 0x08, b2: 0x07},
			[]gamepaddb.Button{gamepaddb.Guide, gamepaddb.Touchpad, gamepaddb.Misc1}, nil},
		{"the buttons of the Edge", ps5Packet{b0: 0x08, b2: 0xF0},
			[]gamepaddb.Button{gamepaddb.Paddle4, gamepaddb.Paddle3, gamepaddb.Paddle2, gamepaddb.Paddle1}, nil},
		{"the triggers", ps5Packet{b0: 0x08, tl: 255, tr: 64}, nil, map[gamepaddb.Axis]int16{gamepaddb.LeftTrigger: 32767, gamepaddb.RightTrigger: 8223}},
		{"a trigger that is a button only", ps5Packet{b0: 0x08, b1: 0x08}, nil, map[gamepaddb.Axis]int16{gamepaddb.RightTrigger: 32767}},
		{"the sticks at their ends", ps5Packet{b0: 0x08, rx: 255, ry: 0}, nil, map[gamepaddb.Axis]int16{gamepaddb.RightX: 32767, gamepaddb.RightY: -32768}},
	} {
		for _, bus := range []struct {
			name string
			bus  Bus
			rep  func(ps5Packet) []byte
		}{{"usb", BusUSB, ps5Packet.usb}, {"bluetooth", BusBluetooth, ps5Packet.bluetooth}} {
			t.Run(tt.name+" "+bus.name, func(t *testing.T) {
				io := &fakeIO{reads: [][]byte{make([]byte, 64)}, features: ps5Features(0x0224)}
				if bus.bus == BusBluetooth {
					io.reads = [][]byte{ps5Packet{}.bluetooth()}
				}
				p, _ := startPS5(t, sonyPS5(bus.bus), io)
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

func TestPS5SimpleReport(t *testing.T) {
	io := &fakeIO{reads: [][]byte{{1, 0, 0, 0, 0, 0x08, 0, 0, 0, 0}}, features: ps5Features(0)}
	p, _ := startPS5(t, sonyPS5(BusBluetooth), io)
	var out recSink
	// The simple report of a Bluetooth pad: 10 bytes, with the sticks, the buttons and the triggers.
	p.Report(time.Second, []byte{1, 255, 0, 128, 128, 0x28, 0x10, 0x03, 0, 0}, &out)
	s := out.lastState(t)
	if !s.Buttons[gamepaddb.South] || !s.Buttons[gamepaddb.Back] || !s.Buttons[gamepaddb.Guide] || !s.Buttons[gamepaddb.Touchpad] {
		t.Errorf("buttons = %v", s.Buttons)
	}
	if s.Axes[gamepaddb.LeftX] != 32767 || s.Axes[gamepaddb.LeftY] != -32768 {
		t.Errorf("axes = %v", s.Axes)
	}
}

func TestPS5Touchpad(t *testing.T) {
	p, _ := startedPS5USB(t, 0x0224)
	var out recSink
	// Two fingers: the first at (1920, 1070) corner-ish values, the second at the origin.
	x, y := 480, 535
	finger := func(counter byte, x, y int) [4]byte {
		return [4]byte{counter, byte(x), byte(x>>8)&0x0F | byte(y&0x0F)<<4, byte(y >> 4)}
	}
	p.Report(time.Second, ps5Packet{b0: 0x08, t1: finger(0x01, x, y), t2: finger(0x02, 1, 1)}.usb(), &out)
	if len(out.touches) != 2 {
		t.Fatalf("touches = %+v", out.touches)
	}
	a, b := out.touches[0], out.touches[1]
	if !a.Down || a.Finger != 0 || math.Abs(float64(a.X)-0.25) > 0.001 || math.Abs(float64(a.Y)-0.5) > 0.001 {
		t.Errorf("first finger = %+v", a)
	}
	if !b.Down || b.Finger != 1 {
		t.Errorf("second finger = %+v", b)
	}
}

func TestPS5Battery(t *testing.T) {
	for _, tt := range []struct {
		name    string
		battery uint8
		state   platform.PadPowerState
		percent int
	}{
		{"on the battery", 0x05, platform.PadOnBattery, 55},
		{"charging", 0x17, platform.PadCharging, 75},
		{"charged", 0x2A, platform.PadCharged, 100},
		{"another state", 0x30, platform.PadPowerUnknown, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p, _ := startedPS5USB(t, 0x0224)
			var out recSink
			p.Report(time.Second, ps5Packet{b0: 0x08, battery: tt.battery}.usb(), &out)
			if len(out.batteries) != 1 || out.batteries[0].State != tt.state || out.batteries[0].Percent != tt.percent {
				t.Errorf("batteries = %+v", out.batteries)
			}
		})
	}
}

func TestPS5Sensors(t *testing.T) {
	const degree = math.Pi / 180
	for _, tt := range []struct {
		name     string
		features map[byte][]byte
		// gyro and accel are what a raw number is worth.
		gyro, accel float64
	}{
		{"with the pad's calibration", withFeature(ps5Features(0x0224), ps5FeatureCalibration, ps5Calibration()), degree / 16, standardGravity / 8192},
		{"without a calibration", ps5Features(0x0224), degree * 64 / 1024, standardGravity / 8192},
	} {
		t.Run(tt.name, func(t *testing.T) {
			io := &fakeIO{reads: [][]byte{make([]byte, 64)}, features: tt.features}
			p, _ := startPS5(t, sonyPS5(BusUSB), io)
			var out recSink
			pk := ps5Packet{b0: 0x08, ts: 3000, gyro: [3]int16{160, -320, 16}, accel: [3]int16{8192, 0, -4096}}
			p.Report(time.Second, pk.usb(), &out)
			if len(out.sensors) != 0 {
				t.Fatal("sensors were reported before they were asked for")
			}
			if err := p.SetSensors(true); err != nil {
				t.Fatal(err)
			}
			p.Report(2*time.Second, pk.usb(), &out)
			if len(out.sensors) != 2 {
				t.Fatalf("%d readings, want 2", len(out.sensors))
			}
			for i, raw := range []float64{160, -320, 16} {
				if want := raw * tt.gyro; math.Abs(float64(out.sensors[0].Values[i])-want) > 1e-4 {
					t.Errorf("gyro[%d] = %v, want %v", i, out.sensors[0].Values[i], want)
				}
			}
			for i, raw := range []float64{8192, 0, -4096} {
				if want := raw * tt.accel; math.Abs(float64(out.sensors[1].Values[i])-want) > 1e-3 {
					t.Errorf("accel[%d] = %v, want %v", i, out.sensors[1].Values[i], want)
				}
			}
			// The clock of the pad counts in units of 0.33 µs: 3000 of them are a millisecond.
			if got := out.sensors[0].SensorTime; got != time.Millisecond {
				t.Errorf("sensor time = %v, want 1ms", got)
			}
		})
	}
}

func withFeature(m map[byte][]byte, id byte, data []byte) map[byte][]byte {
	m[id] = data
	return m
}

func TestPS5RumbleReports(t *testing.T) {
	t.Run("firmware with the improved rumble", func(t *testing.T) {
		p, io := startedPS5USB(t, 0x0224)
		io.sends = nil
		if err := p.Rumble(0xFFFF, 0x4000); err != nil {
			t.Fatal(err)
		}
		if len(io.sends) != 2 {
			t.Fatalf("%d reports were sent, want 2", len(io.sends))
		}
		// The first only stops the audio haptics.
		start := make([]byte, 48)
		start[0], start[1] = 0x02, 0x02
		mustEqual(t, "rumble start", io.sends[0], start)
		// The second has the strengths (the big motor is the left one) and the bit of the improved rumble.
		run := make([]byte, 48)
		run[0], run[1], run[1+2], run[1+3], run[1+38] = 0x02, 0x02, 0x40, 0xFF, 0x04
		mustEqual(t, "rumble", io.sends[1], run)
		// The next change does not start again.
		io.sends = nil
		if err := p.Rumble(0, 0); err != nil {
			t.Fatal(err)
		}
		if len(io.sends) != 1 {
			t.Fatalf("%d reports were sent, want 1", len(io.sends))
		}
		stop := make([]byte, 48)
		stop[0] = 0x02
		mustEqual(t, "stop", io.sends[0], stop)
	})
	t.Run("older firmware, which is told to emulate the rumble", func(t *testing.T) {
		p, io := startedPS5USB(t, 0x0100)
		io.sends = nil
		if err := p.Rumble(0xFFFF, 0x4000); err != nil {
			t.Fatal(err)
		}
		run := make([]byte, 48)
		// Emulation (0x01) and no audio haptics (0x02), with the strengths halved.
		run[0], run[1], run[1+2], run[1+3] = 0x02, 0x03, 0x20, 0x7F
		mustEqual(t, "rumble", io.sends[1], run)
	})
}

func TestPS5TriggerRumble(t *testing.T) {
	p, io := startedPS5USB(t, 0x0224)
	io.sends = nil
	if err := p.RumbleTriggers(0xFFFF, 0); err != nil {
		t.Fatal(err)
	}
	want := make([]byte, 48)
	want[0] = 0x02
	// The flags of the effects of both triggers.
	want[1] = 0x04 | 0x08
	// The right trigger is off (mode 5); the left one vibrates (mode 0x26) in all ten zones, with the strongest amplitude, at
	// 40 hertz.
	want[1+10] = 0x05
	left := want[1+21:]
	left[0], left[1], left[2] = 0x26, 0xFF, 0x03
	left[3], left[4], left[5], left[6] = 0xFF, 0xFF, 0xFF, 0x3F
	left[9] = 40
	mustEqual(t, "trigger rumble", io.lastSend(t), want)

	if err := p.RumbleTriggers(0, 0); err != nil {
		t.Fatal(err)
	}
	off := make([]byte, 48)
	off[0], off[1], off[1+10], off[1+21] = 0x02, 0x0C, 0x05, 0x05
	mustEqual(t, "trigger rumble off", io.lastSend(t), off)
}

func TestPS5Light(t *testing.T) {
	p, io := startedPS5USB(t, 0x0224)
	io.sends = nil
	if err := p.SetLED(10, 20, 30); err != nil {
		t.Fatal(err)
	}
	want := make([]byte, 48)
	want[0], want[2] = 0x02, 0x04
	want[1+44], want[1+45], want[1+46] = 10, 20, 30
	mustEqual(t, "light", io.lastSend(t), want)
}

func TestPS5EdgeHasPaddles(t *testing.T) {
	info := DeviceInfo{Vendor: vendorSony, Product: productDualSenseEdge, Bus: BusUSB, Name: "DualSense Edge Wireless Controller"}
	io := &fakeIO{reads: [][]byte{make([]byte, 64)}, features: ps5Features(0)}
	_, caps := startPS5(t, info, io)
	if caps.Name != "DualSense Edge Wireless Controller" || caps.Rate != 1000 {
		t.Errorf("caps = %+v", caps)
	}
	for _, b := range []gamepaddb.Button{gamepaddb.Paddle1, gamepaddb.Paddle2, gamepaddb.Paddle3, gamepaddb.Paddle4} {
		if caps.Buttons&buttonBit(b) == 0 {
			t.Errorf("the Edge has no button %d", b)
		}
	}
}

func TestPS5BluetoothSilence(t *testing.T) {
	t.Run("a pad with enhanced reports is poked", func(t *testing.T) {
		io := &fakeIO{reads: [][]byte{ps5Packet{}.bluetooth()}, features: ps5Features(0)}
		p, _ := startPS5(t, sonyPS5(BusBluetooth), io)
		sent := len(io.sends)
		var out recSink
		p.Tick(time.Second, &out)
		if len(io.sends) != sent+1 {
			t.Fatalf("the pad was not poked")
		}
		want := make([]byte, 78)
		want[0], want[1] = 0x31, 0x02
		mustEqual(t, "poke", io.lastSend(t), want)
		if out.lost != 0 {
			t.Error("the pad was taken for gone")
		}
	})
	t.Run("a pad with simple reports is gone", func(t *testing.T) {
		io := &fakeIO{reads: [][]byte{{1, 0, 0, 0, 0, 0x08, 0, 0, 0, 0}}, features: ps5Features(0)}
		p, _ := startPS5(t, sonyPS5(BusBluetooth), io)
		var out recSink
		p.Tick(time.Second, &out)
		if out.lost != 1 {
			t.Errorf("lost = %d, want 1", out.lost)
		}
	})
}

func TestPS5ThirdPartyPadUsesTheAlternateReport(t *testing.T) {
	feat := make([]byte, 48)
	feat[0], feat[2], feat[4], feat[20] = 0x03, 0x28, 0x08|0x04, 0x80
	info := DeviceInfo{Vendor: 0x20d6, Product: 0x0001, Bus: BusUSB, Name: "Third party"}
	io := &fakeIO{reads: [][]byte{make([]byte, 64)}, features: map[byte][]byte{0x03: feat}}
	p := newPS5(info).(*ps5)
	caps, err := p.Start(io)
	if err != nil {
		t.Fatal(err)
	}
	if caps.Name != "" || !caps.Rumble || !caps.LED || caps.Sensors || caps.Touchpads != 0 {
		t.Errorf("caps = %+v", caps)
	}
	if !p.useAlternateReport {
		t.Fatal("the alternate report is not used")
	}
	// The alternate report has the battery at 29 and the touchpad at 31, and a timestamp of 16 bits.
	pk := make([]byte, 64)
	pk[0] = ps5ReportState
	pk[1+7] = 0x08
	pk[1+9] = 0x01
	pk[1+29] = 0x13 // charging, level 3
	var out recSink
	p.Report(time.Second, pk, &out)
	s := out.lastState(t)
	if !s.Buttons[gamepaddb.Guide] {
		t.Error("the PS button was not read")
	}
	if len(out.batteries) != 1 || out.batteries[0].State != platform.PadCharging || out.batteries[0].Percent != 35 {
		t.Errorf("batteries = %+v", out.batteries)
	}
}
