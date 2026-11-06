package gamepadhid

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/gamepaddb"
	"github.com/rm4n0s/glutter/internal/platform"
)

func xboxInfo(product uint16) DeviceInfo {
	return DeviceInfo{Vendor: vendorMicrosoft, Product: product, Bus: BusBluetooth, Name: "Xbox Wireless Controller"}
}

func startXbox(t *testing.T, product uint16) (*xboxOne, *fakeIO, Caps) {
	t.Helper()
	io := &fakeIO{}
	d := newXboxOne(xboxInfo(product)).(*xboxOne)
	caps, err := d.Start(io)
	if err != nil {
		t.Fatal(err)
	}
	return d, io, caps
}

// xboxReport makes the state report of a pad of the given size: the sticks (16 bits each), the triggers (10 bits), the hat and the
// two bytes of buttons that SDL's code reads at 14 and 15.
func xboxReport(size int, lx, ly, rx, ry, lt, rt uint16, hat, b14, b15 byte) []byte {
	r := make([]byte, size)
	r[0] = xboxReportState
	for i, v := range []uint16{lx, ly, rx, ry, lt, rt} {
		r[1+2*i], r[2+2*i] = byte(v), byte(v>>8)
	}
	r[13], r[14], r[15] = hat, b14, b15
	return r
}

func TestXboxOneStateReport(t *testing.T) {
	for _, tt := range []struct {
		name    string
		product uint16
		report  []byte
		buttons []gamepaddb.Button
		axes    map[gamepaddb.Axis]int16
	}{
		{"a pad at rest", productXboxSeriesXBLE, xboxReport(17, 0x8000, 0x8000, 0x8000, 0x8000, 0, 0, 0, 0, 0), nil,
			map[gamepaddb.Axis]int16{gamepaddb.LeftX: 0, gamepaddb.RightY: 0}},
		{"the face buttons", productXboxSeriesXBLE, xboxReport(17, 0x8000, 0x8000, 0x8000, 0x8000, 0, 0, 0, 0x01|0x02|0x08|0x10, 0),
			[]gamepaddb.Button{gamepaddb.South, gamepaddb.East, gamepaddb.West, gamepaddb.North}, nil},
		{"the shoulders", productXboxSeriesXBLE, xboxReport(17, 0x8000, 0x8000, 0x8000, 0x8000, 0, 0, 0, 0x40|0x80, 0),
			[]gamepaddb.Button{gamepaddb.LeftShoulder, gamepaddb.RightShoulder}, nil},
		{"view, menu, sticks and the Guide in the state", productXboxSeriesXBLE, xboxReport(17, 0x8000, 0x8000, 0x8000, 0x8000, 0, 0, 0, 0, 0x04|0x08|0x10|0x20|0x40),
			[]gamepaddb.Button{gamepaddb.Back, gamepaddb.Start, gamepaddb.Guide, gamepaddb.LeftStick, gamepaddb.RightStick}, nil},
		{"the Share button of the Series X", productXboxSeriesXBLE, func() []byte {
			r := xboxReport(18, 0x8000, 0x8000, 0x8000, 0x8000, 0, 0, 0, 0, 0)
			r[16] = 0x01
			return r
		}(), []gamepaddb.Button{gamepaddb.Misc1}, nil},
		{"an Xbox One S has no Share: the byte is the view button", productXboxOneSRev2BLE, func() []byte {
			r := xboxReport(17, 0x8000, 0x8000, 0x8000, 0x8000, 0, 0, 0, 0, 0)
			r[16] = 0x01
			return r
		}(), []gamepaddb.Button{gamepaddb.Back}, nil},
		{"the hat", productXboxSeriesXBLE, xboxReport(17, 0x8000, 0x8000, 0x8000, 0x8000, 0, 0, 2, 0, 0),
			[]gamepaddb.Button{gamepaddb.DPadUp, gamepaddb.DPadRight}, nil},
		{"the hat left", productXboxSeriesXBLE, xboxReport(17, 0x8000, 0x8000, 0x8000, 0x8000, 0, 0, 7, 0, 0),
			[]gamepaddb.Button{gamepaddb.DPadLeft}, nil},
		{"the triggers", productXboxSeriesXBLE, xboxReport(17, 0x8000, 0x8000, 0x8000, 0x8000, 1023, 512, 0, 0, 0), nil,
			map[gamepaddb.Axis]int16{gamepaddb.LeftTrigger: 32767, gamepaddb.RightTrigger: 16383}},
		{"the sticks at their ends", productXboxSeriesXBLE, xboxReport(17, 0, 0xFFFF, 0xFFFF, 0, 0, 0, 0, 0, 0), nil,
			map[gamepaddb.Axis]int16{gamepaddb.LeftX: -32768, gamepaddb.LeftY: 32767, gamepaddb.RightX: 32767, gamepaddb.RightY: -32768}},
		{"the first Xbox One S: a packet of 16 bytes", productXboxOneSRev1Bluetooth, func() []byte {
			r := xboxReport(16, 0x8000, 0x8000, 0x8000, 0x8000, 0, 0, 0, 0x01|0x04|0x40|0x80, 0x03)
			return r
		}(), []gamepaddb.Button{gamepaddb.South, gamepaddb.West, gamepaddb.Back, gamepaddb.Start, gamepaddb.LeftStick, gamepaddb.RightStick}, nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			d, _, _ := startXbox(t, tt.product)
			var out recSink
			d.Report(time.Second, tt.report, &out)
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
				if s.Axes[a] != v {
					t.Errorf("axis %d = %d, want %d", a, s.Axes[a], v)
				}
			}
		})
	}
}

func TestXboxOneGuideComesInItsOwnReport(t *testing.T) {
	d, _, _ := startXbox(t, productXboxOneSRev2Bluetooth)
	var out recSink
	d.Report(time.Second, []byte{xboxReportGuide, 0x01}, &out)
	if !out.lastState(t).Buttons[gamepaddb.Guide] {
		t.Fatal("the Guide button was not read from its report")
	}
	// From then on the state's own bit is not the Guide button.
	d.Report(time.Second, xboxReport(17, 0x8000, 0x8000, 0x8000, 0x8000, 0, 0, 0, 0, 0x10), &out)
	if !out.lastState(t).Buttons[gamepaddb.Guide] {
		t.Error("the Guide button was taken from the state after its own report")
	}
	d.Report(time.Second, []byte{xboxReportGuide, 0x00}, &out)
	if out.lastState(t).Buttons[gamepaddb.Guide] {
		t.Error("the Guide button did not go up")
	}
}

func TestXboxOneElitePaddles(t *testing.T) {
	for _, tt := range []struct {
		name    string
		size    int
		index   int
		mapped  int
		paddles byte
		want    []gamepaddb.Button
	}{
		{"firmware 5.13 and later", 20, 19, 17, 0x0F, []gamepaddb.Button{gamepaddb.Paddle1, gamepaddb.Paddle2, gamepaddb.Paddle3, gamepaddb.Paddle4}},
		{"one paddle", 20, 19, 17, 0x04, []gamepaddb.Button{gamepaddb.Paddle2}},
		{"the updated firmware", 39, 17, 19, 0x01, []gamepaddb.Button{gamepaddb.Paddle1}},
		{"the first firmware", 55, 33, 35, 0x02, []gamepaddb.Button{gamepaddb.Paddle3}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			d, _, caps := startXbox(t, productXboxElite2BLE)
			if caps.Buttons&buttonBit(gamepaddb.Paddle1) == 0 {
				t.Fatal("the Elite has no paddles in its caps")
			}
			r := xboxReport(tt.size, 0x8000, 0x8000, 0x8000, 0x8000, 0, 0, 0, 0, 0)
			r[tt.index] = tt.paddles
			var out recSink
			d.Report(time.Second, r, &out)
			s := out.lastState(t)
			want := map[gamepaddb.Button]bool{}
			for _, b := range tt.want {
				want[b] = true
			}
			for _, b := range []gamepaddb.Button{gamepaddb.Paddle1, gamepaddb.Paddle2, gamepaddb.Paddle3, gamepaddb.Paddle4} {
				if s.Buttons[b] != want[b] {
					t.Errorf("paddle button %d = %v, want %v", b, s.Buttons[b], want[b])
				}
			}
			// Paddles that are mapped to other controls are not passed on.
			r[tt.mapped] = 1
			d.Report(time.Second, r, &out)
			s = out.lastState(t)
			for _, b := range []gamepaddb.Button{gamepaddb.Paddle1, gamepaddb.Paddle2, gamepaddb.Paddle3, gamepaddb.Paddle4} {
				if s.Buttons[b] {
					t.Errorf("a paddle that is mapped was passed on (%d)", b)
				}
			}
		})
	}
}

func TestXboxOneBattery(t *testing.T) {
	for _, tt := range []struct {
		flags   byte
		state   platform.PadPowerState
		percent int
	}{
		{0x04, platform.PadOnBattery, 10},
		{0x04 | 0x01, platform.PadOnBattery, 40},
		{0x08 | 0x02, platform.PadOnBattery, 70},
		{0x04 | 0x03, platform.PadOnBattery, 100},
		{0x00 | 0x02, platform.PadCharging, 70},
	} {
		d, _, _ := startXbox(t, productXboxSeriesXBLE)
		var out recSink
		d.Report(time.Second, []byte{xboxReportBattery, tt.flags}, &out)
		if len(out.batteries) != 1 || out.batteries[0].State != tt.state || out.batteries[0].Percent != tt.percent {
			t.Errorf("flags %#x: batteries = %+v", tt.flags, out.batteries)
		}
	}
}

func TestXboxOneRumbleReports(t *testing.T) {
	d, io, caps := startXbox(t, productXboxSeriesXBLE)
	if !caps.Rumble || !caps.RumbleTriggers {
		t.Fatalf("caps = %+v", caps)
	}
	io.now = time.Second
	if err := d.Rumble(0xFFFF, 0x8000); err != nil {
		t.Fatal(err)
	}
	// The report 3, the flags 0x0F (all four motors), the trigger motors, the two main motors (0 to 100), the length and the
	// pause.
	mustEqual(t, "rumble", io.lastSend(t), []byte{0x03, 0x0F, 0x00, 0x00, 100, 50, 0xFF, 0x00, 0xEB})
	// A report that follows in less than 50 ms waits for the pad.
	io.now += 10 * time.Millisecond
	if err := d.RumbleTriggers(0x4000, 0x8000); err != nil {
		t.Fatal(err)
	}
	if len(io.sends) != 1 {
		t.Fatalf("%d reports were sent, want 1", len(io.sends))
	}
	var out recSink
	io.now += 60 * time.Millisecond
	d.Tick(io.now, &out)
	if len(io.sends) != 2 {
		t.Fatalf("%d reports were sent, want 2", len(io.sends))
	}
	mustEqual(t, "trigger rumble", io.lastSend(t), []byte{0x03, 0x0F, 25, 50, 100, 50, 0xFF, 0x00, 0xEB})
	if err := d.SetLED(1, 2, 3); err == nil {
		t.Error("a pad over Bluetooth took a colour")
	}
	if err := d.SetSensors(true); err == nil {
		t.Error("a pad without sensors turned them on")
	}
}

func TestBluetoothXboxOneProducts(t *testing.T) {
	for product, want := range map[uint16]bool{
		0x0b13: true, 0x0b05: true, 0x0b22: true, 0x02e0: true, 0x02fd: true, 0x0b20: true, 0x0b0c: true, 0x0b21: true,
		0x0b12: false, 0x02ea: false, 0x0b00: false,
	} {
		if got := isBluetoothXboxOne(vendorMicrosoft, product); got != want {
			t.Errorf("product %#x: %v, want %v", product, got, want)
		}
	}
	if isBluetoothXboxOne(0x1234, 0x0b13) {
		t.Error("a product of another maker was taken")
	}
}
