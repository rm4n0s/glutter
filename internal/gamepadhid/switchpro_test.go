package gamepadhid

import (
	"math"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gamepaddb"
	"github.com/rm4n0s/glutter/internal/platform"
)

// switchPad is a scripted Switch Pro controller: it answers the subcommands and the proprietary commands that the driver sends.
type switchPad struct {
	io *fakeIO
	// deviceType is what the pad says it is (3 is the Pro controller).
	deviceType byte
	// spi is the flash of the pad: the data at an address.
	spi map[uint32][]byte
	// usb makes the pad answer the commands of the USB protocol, as a pad on USB does (and not over Bluetooth).
	usb bool
	// ids are the subcommands (and proprietary commands, with 0x100 added) that were sent, in order.
	ids []int
}

func newSwitchPad(usb bool) (*switchPad, *fakeIO) {
	io := &fakeIO{}
	p := &switchPad{io: io, deviceType: 3, usb: usb, spi: map[uint32][]byte{}}
	io.onWrite = p.onWrite
	io.onSend = p.onSend
	return p, io
}

func (p *switchPad) onSend(data []byte) { p.record(data) }

func (p *switchPad) onWrite(data []byte) {
	p.record(data)
	switch data[0] {
	case switchOutputProprietary:
		if !p.usb {
			return
		}
		cmd := data[1]
		r := make([]byte, 64)
		r[0], r[1] = switchInputCommandAck, cmd
		if cmd == switchProprietaryStatus {
			r[3] = p.deviceType
			copy(r[4:10], []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66})
		}
		p.io.reads = append(p.io.reads, r)
	case switchOutputRumbleAndSubcmd:
		id := data[10]
		r := make([]byte, 64)
		r[0], r[13], r[14] = switchInputSubcommandReply, 0x80, id
		switch id {
		case switchSubcmdRequestDeviceInfo:
			r[15+2] = p.deviceType
		case switchSubcmdSPIFlashRead:
			addr := ule32(data[11:])
			n := int(data[15])
			copy(r[15:19], data[11:15])
			r[19] = data[15]
			copy(r[20:20+n], p.spi[addr])
		}
		p.io.reads = append(p.io.reads, r)
	}
}

func (p *switchPad) record(data []byte) {
	switch data[0] {
	case switchOutputRumbleAndSubcmd:
		p.ids = append(p.ids, int(data[10]))
	case switchOutputProprietary:
		p.ids = append(p.ids, 0x100+int(data[1]))
	case switchOutputRumble:
		p.ids = append(p.ids, 0x200)
	}
}

// packPair packs two values of 12 bits into three bytes, as the calibration of the sticks and the reports of the state hold them.
func packPair(a, b int) [3]byte {
	return [3]byte{byte(a), byte(a>>8)&0x0F | byte(b&0x0F)<<4, byte(b >> 4)}
}

// factoryStickCalibration makes the calibration of the two sticks as the flash holds it: the left stick as x and y of the maximum,
// the center and the minimum, the right stick as the center, the minimum and the maximum.
func factoryStickCalibration(center, extent int) []byte {
	var out []byte
	for _, v := range []int{extent, center, extent} { // left: max, center, min
		p := packPair(v, v)
		out = append(out, p[:]...)
	}
	for _, v := range []int{center, extent, extent} { // right: center, min, max
		p := packPair(v, v)
		out = append(out, p[:]...)
	}
	return out
}

func switchProInfo(bus Bus) DeviceInfo {
	return DeviceInfo{Vendor: vendorNintendo, Product: productSwitchPro, Bus: bus, Name: "Pro Controller"}
}

func startSwitch(t *testing.T, bus Bus, configure func(*switchPad)) (*switchPro, *switchPad, *fakeIO) {
	t.Helper()
	pad, io := newSwitchPad(bus == BusUSB)
	pad.spi[switchStickFactoryStart] = factoryStickCalibration(2048, 1500)
	if configure != nil {
		configure(pad)
	}
	d := newSwitchPro(switchProInfo(bus)).(*switchPro)
	if _, err := d.Start(io); err != nil {
		t.Fatalf("Start: %v", err)
	}
	return d, pad, io
}

func TestSwitchProStartOverBluetoothTalksInOrder(t *testing.T) {
	_, pad, io := startSwitch(t, BusBluetooth, nil)
	// The device info, the calibration of the sticks (the user's, then the factory's because the user's has no magic number), the
	// calibration of the sensors (the factory's and the user's), vibration on, the full input mode, and the player lights.
	want := []int{
		switchSubcmdRequestDeviceInfo, switchSubcmdSPIFlashRead, switchSubcmdSPIFlashRead, switchSubcmdSPIFlashRead,
		switchSubcmdSPIFlashRead, switchSubcmdEnableVibration, switchSubcmdSetInputReportMod, switchSubcmdSetPlayerLights,
	}
	if len(pad.ids) != len(want) {
		t.Fatalf("commands = %x, want %x", pad.ids, want)
	}
	for i := range want {
		if pad.ids[i] != want[i] {
			t.Fatalf("commands = %x, want %x", pad.ids, want)
		}
	}
	// The first report byte for byte: the report 1, the counter 0, the neutral rumble twice, the subcommand 2.
	first := make([]byte, 49)
	copy(first, []byte{0x01, 0x00, 0x00, 0x01, 0x40, 0x40, 0x00, 0x01, 0x40, 0x40, 0x02})
	mustEqual(t, "first report", io.writes[0], first)
	// The counter goes up by one in every report and wraps at 16.
	if io.writes[1][1] != 1 || io.writes[2][1] != 2 {
		t.Errorf("counters = %d, %d", io.writes[1][1], io.writes[2][1])
	}
	// The mode report: subcommand 3 with the full state.
	mode := io.writes[6]
	if mode[10] != 0x03 || mode[11] != switchInputFullState {
		t.Errorf("mode report = %x", mode[:12])
	}
	// The user's calibration is asked for first, at 0x8010 for 22 bytes.
	spi := io.writes[1]
	if ule32(spi[11:]) != 0x8010 || spi[15] != 22 {
		t.Errorf("spi read = %x", spi[:16])
	}
	// The vibration is turned on.
	if v := io.writes[5]; v[10] != 0x48 || v[11] != 1 {
		t.Errorf("vibration report = %x", v[:12])
	}
}

func TestSwitchProStartOverUSBMakesTheHandshake(t *testing.T) {
	_, pad, io := startSwitch(t, BusUSB, nil)
	want := []int{
		0x100 + switchProprietaryStatus, 0x100 + switchProprietaryHandshake, 0x100 + switchProprietaryHighSpeed,
		0x100 + switchProprietaryHandshake, 0x100 + switchProprietaryForceUSB,
		switchSubcmdSPIFlashRead, switchSubcmdSPIFlashRead, switchSubcmdSPIFlashRead, switchSubcmdSPIFlashRead,
		switchSubcmdEnableVibration, switchSubcmdSetInputReportMod, 0x100 + switchProprietaryForceUSB, switchSubcmdSetPlayerLights,
	}
	if len(pad.ids) != len(want) {
		t.Fatalf("commands = %x, want %x", pad.ids, want)
	}
	for i := range want {
		if pad.ids[i] != want[i] {
			t.Fatalf("commands = %x, want %x", pad.ids, want)
		}
	}
	// A report over USB is 64 bytes.
	if len(io.writes[0]) != 64 {
		t.Errorf("first report is %d bytes", len(io.writes[0]))
	}
	status := make([]byte, 64)
	status[0], status[1] = 0x80, 0x01
	mustEqual(t, "status request", io.writes[0], status)
}

func TestSwitchProThatIsNotAProController(t *testing.T) {
	pad, io := newSwitchPad(false)
	pad.deviceType = 1 // the left Joy-Con
	d := newSwitchPro(switchProInfo(BusBluetooth)).(*switchPro)
	_, err := d.Start(io)
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if !appErr.HasRoute("switchPro.Start.HidDriverStartFailed") {
		t.Errorf("unexpected failure path: %s", appErr.Route())
	}
}

func TestSwitchProThatDoesNotAnswerIsAProController(t *testing.T) {
	// A pad that does not answer the request for its info is taken for a Pro controller by its product number.
	io := &fakeIO{}
	d := newSwitchPro(switchProInfo(BusBluetooth)).(*switchPro)
	if _, err := d.Start(io); err != nil {
		t.Fatal(err)
	}
	if d.controllerType != switchControllerTypePro {
		t.Errorf("type = %d", d.controllerType)
	}
}

// fullReport makes a report of the full state: the counter, the battery byte, the three bytes of buttons, the two sticks of three
// bytes (12 bits each) and the three readings of the sensors.
func fullReport(battery byte, buttons [3]byte, left, right [2]int, imu [3][6]int16) []byte {
	r := make([]byte, 49)
	r[0] = switchInputFullState
	r[2] = battery
	copy(r[3:6], buttons[:])
	l := packPair(left[0], left[1])
	copy(r[6:9], l[:])
	rr := packPair(right[0], right[1])
	copy(r[9:12], rr[:])
	for i := range imu {
		for j := range imu[i] {
			putI16(r[13+12*i+2*j:], imu[i][j])
		}
	}
	return r
}

func TestSwitchProFullReport(t *testing.T) {
	for _, tt := range []struct {
		name    string
		buttons [3]byte
		want    []gamepaddb.Button
		axes    map[gamepaddb.Axis]int16
	}{
		{"nothing", [3]byte{}, nil, nil},
		{"the face buttons by position", [3]byte{0x0F, 0, 0}, []gamepaddb.Button{gamepaddb.South, gamepaddb.East, gamepaddb.West, gamepaddb.North}, nil},
		{"B at the bottom", [3]byte{0x04, 0, 0}, []gamepaddb.Button{gamepaddb.South}, nil},
		{"the right shoulder and trigger", [3]byte{0xC0, 0, 0}, []gamepaddb.Button{gamepaddb.RightShoulder}, map[gamepaddb.Axis]int16{gamepaddb.RightTrigger: 32767}},
		{"minus, plus, the sticks, home and capture", [3]byte{0, 0x3F, 0},
			[]gamepaddb.Button{gamepaddb.Back, gamepaddb.Start, gamepaddb.RightStick, gamepaddb.LeftStick, gamepaddb.Guide, gamepaddb.Misc1}, nil},
		{"the D-pad", [3]byte{0, 0, 0x0F}, []gamepaddb.Button{gamepaddb.DPadUp, gamepaddb.DPadRight, gamepaddb.DPadDown, gamepaddb.DPadLeft}, nil},
		{"the left shoulder and trigger", [3]byte{0, 0, 0xC0}, []gamepaddb.Button{gamepaddb.LeftShoulder}, map[gamepaddb.Axis]int16{gamepaddb.LeftTrigger: 32767}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			d, _, _ := startSwitch(t, BusBluetooth, nil)
			var out recSink
			d.Report(time.Second, fullReport(0, tt.buttons, [2]int{2048, 2048}, [2]int{2048, 2048}, [3][6]int16{}), &out)
			s := out.lastState(t)
			want := map[gamepaddb.Button]bool{}
			for _, b := range tt.want {
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

func TestSwitchProSticksUseTheCalibration(t *testing.T) {
	d, _, _ := startSwitch(t, BusBluetooth, nil)
	// The stick can go 1500 below and above the centre of 2048, and the range that counts is 70% of that: 1050.
	for _, tt := range []struct {
		name        string
		left, right [2]int
		lx, ly      int16
		rx, ry      int16
	}{
		{"at rest", [2]int{2048, 2048}, [2]int{2048, 2048}, 0, -1, 0, -1},
		{"to the right and up", [2]int{2048 + 1050, 2048 + 1050}, [2]int{2048 - 1050, 2048 - 1050}, 32767, -32768, -32768, 32767},
		{"half way", [2]int{2048 + 525, 2048}, [2]int{2048, 2048 + 525}, 16383, -1, 0, -16384},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var out recSink
			d.Report(time.Second, fullReport(0, [3]byte{}, tt.left, tt.right, [3][6]int16{}), &out)
			s := out.lastState(t)
			if s.Axes[gamepaddb.LeftX] != tt.lx || s.Axes[gamepaddb.LeftY] != tt.ly || s.Axes[gamepaddb.RightX] != tt.rx || s.Axes[gamepaddb.RightY] != tt.ry {
				t.Errorf("sticks = %d %d %d %d, want %d %d %d %d", s.Axes[gamepaddb.LeftX], s.Axes[gamepaddb.LeftY], s.Axes[gamepaddb.RightX],
					s.Axes[gamepaddb.RightY], tt.lx, tt.ly, tt.rx, tt.ry)
			}
		})
	}
}

func TestSwitchProStickThatGoesFurtherThanItsCalibration(t *testing.T) {
	d, _, _ := startSwitch(t, BusBluetooth, nil)
	var out recSink
	// 2048 + 1400 is beyond the extent of 1050: the extent grows to it, so the stick reads its full range and not more.
	d.Report(time.Second, fullReport(0, [3]byte{}, [2]int{2048 + 1400, 2048}, [2]int{2048, 2048}, [3][6]int16{}), &out)
	if got := out.lastState(t).Axes[gamepaddb.LeftX]; got != 32767 {
		t.Errorf("left x = %d, want 32767", got)
	}
	d.Report(time.Second, fullReport(0, [3]byte{}, [2]int{2048 + 700, 2048}, [2]int{2048, 2048}, [3][6]int16{}), &out)
	if got := out.lastState(t).Axes[gamepaddb.LeftX]; got != 16383 {
		t.Errorf("left x = %d, want 16383 (700 of 1400)", got)
	}
}

func TestSwitchProSimpleReport(t *testing.T) {
	d, _, _ := startSwitch(t, BusBluetooth, nil)
	var out recSink
	r := make([]byte, 12)
	r[0] = switchInputSimpleState
	r[1], r[2], r[3] = 0x01|0x10|0x40, 0x02|0x20, 0x02 // A (bottom), left shoulder, ZL; plus, capture; hat right
	putI16(r[4:], int16(-32768+0x8000-0x8000))         // left x at the centre (0x8000 as a signed number is -32768)
	putI16(r[6:], -32768)
	putI16(r[8:], -32768)
	putI16(r[10:], -32768)
	d.Report(time.Second, r, &out)
	s := out.lastState(t)
	if !s.Buttons[gamepaddb.South] || !s.Buttons[gamepaddb.LeftShoulder] || !s.Buttons[gamepaddb.Start] || !s.Buttons[gamepaddb.Misc1] || !s.Buttons[gamepaddb.DPadRight] {
		t.Errorf("buttons = %v", s.Buttons)
	}
	if s.Axes[gamepaddb.LeftTrigger] != 32767 || s.Axes[gamepaddb.RightTrigger] != 0 {
		t.Errorf("triggers = %d, %d", s.Axes[gamepaddb.LeftTrigger], s.Axes[gamepaddb.RightTrigger])
	}
	if s.Axes[gamepaddb.LeftX] != 0 {
		t.Errorf("left x = %d, want 0", s.Axes[gamepaddb.LeftX])
	}
}

func TestSwitchProBattery(t *testing.T) {
	for _, tt := range []struct {
		name    string
		battery byte
		state   platform.PadPowerState
		percent int
	}{
		{"full on the battery", 0x80, platform.PadOnBattery, 100},
		{"half", 0x40, platform.PadOnBattery, 50},
		{"charging", 0x70 | 0x10, platform.PadCharging, 75},
		{"charged", 0x80 | 0x10, platform.PadCharged, 100},
	} {
		t.Run(tt.name, func(t *testing.T) {
			d, _, _ := startSwitch(t, BusBluetooth, nil)
			var out recSink
			d.Report(time.Second, fullReport(tt.battery, [3]byte{}, [2]int{2048, 2048}, [2]int{2048, 2048}, [3][6]int16{}), &out)
			if len(out.batteries) != 1 || out.batteries[0].State != tt.state || out.batteries[0].Percent != tt.percent {
				t.Errorf("batteries = %+v", out.batteries)
			}
		})
	}
}

func TestSwitchProSensors(t *testing.T) {
	d, pad, io := startSwitch(t, BusBluetooth, nil)
	pad.ids = nil
	if err := d.SetSensors(true); err != nil {
		t.Fatal(err)
	}
	// The pad is in the full mode already; only the sensors are turned on.
	if len(pad.ids) != 1 || pad.ids[0] != switchSubcmdEnableIMU {
		t.Fatalf("commands = %x", pad.ids)
	}
	if got := io.lastSend(t); got[11] != 1 {
		t.Errorf("enable report = %x", got[:12])
	}
	var out recSink
	// Three readings, the newest first: only the first one has an accelerometer value that is checked here.
	imu := [3][6]int16{{4096, 0, 0, 0, 0, 0}, {0, 0, 0, 0, 0, 0}, {0, 4096, 0, 0, 0, 142}}
	d.Report(time.Second, fullReport(0, [3]byte{}, [2]int{2048, 2048}, [2]int{2048, 2048}, imu), &out)
	if len(out.sensors) != 6 {
		t.Fatalf("%d readings, want 6 (three gyro and accel pairs)", len(out.sensors))
	}
	// They come oldest first: the last reading of the report is the oldest.
	oldestGyro, oldestAccel := out.sensors[0], out.sensors[1]
	if oldestGyro.Kind != platform.PadGyroscope || oldestAccel.Kind != platform.PadAccelerometer {
		t.Fatalf("kinds = %d, %d", oldestGyro.Kind, oldestAccel.Kind)
	}
	// The order and the signs follow the PlayStation pads: the accelerometer's y of 4096 is a force of gravity on the first
	// component, negated, and the gyroscope's z of 142 is a degree a second on the second.
	if got := float64(oldestAccel.Values[0]); math.Abs(got+standardGravity) > 1e-3 {
		t.Errorf("accel[0] = %v, want %v", got, -standardGravity)
	}
	if got := float64(oldestGyro.Values[1]); math.Abs(got-142*math.Pi/180/switchGyroScale) > 1e-4 {
		t.Errorf("gyro[1] = %v", got)
	}
	// The newest reading: accel x = 4096 is the third component, negated.
	newestAccel := out.sensors[5]
	if got := float64(newestAccel.Values[2]); math.Abs(got+standardGravity) > 1e-3 {
		t.Errorf("accel[2] = %v, want %v", got, -standardGravity)
	}
	// Their clocks advance by the interval (5 ms at first), three readings to a report.
	if out.sensors[2].SensorTime-out.sensors[0].SensorTime != 5*time.Millisecond {
		t.Errorf("readings are %v apart", out.sensors[2].SensorTime-out.sensors[0].SensorTime)
	}
}

func TestSwitchProRumbleReports(t *testing.T) {
	d, _, io := startSwitch(t, BusBluetooth, nil)
	io.now = time.Second
	io.sends = nil
	if err := d.Rumble(0xFFFF, 0); err != nil {
		t.Fatal(err)
	}
	if len(io.sends) != 1 {
		t.Fatalf("%d reports were sent", len(io.sends))
	}
	got := io.sends[0]
	// The report 0x10, the counter, and the rumble of both sides: the high band at 0x0074 with no amplitude, the low band at 0x3D
	// with the last amplitude of the table, 0x0072.
	want := make([]byte, 49)
	copy(want, []byte{0x10, got[1], 0x74, 0x00, 0x3D, 0x72, 0x74, 0x00, 0x3D, 0x72})
	mustEqual(t, "rumble", got, want)

	// The high band at full strength is the last value of its table, 0xC8; the low band at none is the first of its table,
	// 0x0040.
	io.now = 2 * time.Second
	if err := d.Rumble(0, 0xFFFF); err != nil {
		t.Fatal(err)
	}
	copy(want, []byte{0x10, io.lastSend(t)[1], 0x74, 0xC8, 0x3D, 0x40, 0x74, 0xC8, 0x3D, 0x40})
	mustEqual(t, "high rumble", io.lastSend(t), want)

	// A rumble of 0 gives the neutral value back.
	io.now = 3 * time.Second
	if err := d.Rumble(0, 0); err != nil {
		t.Fatal(err)
	}
	copy(want, []byte{0x10, io.lastSend(t)[1], 0x00, 0x01, 0x40, 0x40, 0x00, 0x01, 0x40, 0x40})
	mustEqual(t, "stop", io.lastSend(t), want)
}

func TestSwitchProRumbleIsThrottled(t *testing.T) {
	d, _, io := startSwitch(t, BusBluetooth, nil)
	io.now = time.Second
	io.sends = nil
	var out recSink
	if err := d.Rumble(0x2000, 0); err != nil {
		t.Fatal(err)
	}
	// Within 30 ms of the last write, a change waits; the strongest of the changes is the one that is sent.
	io.now += 10 * time.Millisecond
	_ = d.Rumble(0x4000, 0)
	_ = d.Rumble(0x1000, 0)
	if len(io.sends) != 1 {
		t.Fatalf("%d reports were sent in the interval, want 1", len(io.sends))
	}
	d.Tick(io.now, &out)
	if len(io.sends) != 1 {
		t.Fatal("the pending rumble was sent too soon")
	}
	io.now += 30 * time.Millisecond
	d.Tick(io.now, &out)
	if len(io.sends) != 2 {
		t.Fatalf("%d reports were sent, want 2", len(io.sends))
	}
	if got := io.lastSend(t); got[2] != 0x74 || got[5] != byte(encodeRumbleLowAmplitude(0x4000)) {
		t.Errorf("rumble = %x", got[:10])
	}
	// A rumble that goes on is written again every 50 ms.
	n := len(io.sends)
	io.now += 20 * time.Millisecond
	d.Tick(io.now, &out)
	if len(io.sends) != n {
		t.Error("a rumble was refreshed after 20 ms")
	}
	io.now += 40 * time.Millisecond
	d.Tick(io.now, &out)
	if len(io.sends) != n+1 {
		t.Error("a rumble was not refreshed after 60 ms")
	}
	// And a stop inside the interval is sent when the time has come.
	_ = d.Rumble(0, 0)
	io.now += 40 * time.Millisecond
	d.Tick(io.now, &out)
	last := io.lastSend(t)
	if last[2] != 0x00 || last[3] != 0x01 {
		t.Errorf("the last report is not a stop: %x", last[:10])
	}
}

func TestEncodeRumbleAmplitudes(t *testing.T) {
	for _, tt := range []struct {
		amp  uint16
		high uint8
		low  uint16
	}{
		{0, 0x00, 0x0040},
		{1, 0x02, 0x8040},
		{514, 0x02, 0x8040},
		{515, 0x04, 0x0041},
		{16000, 0x46, 0x8051},
		{65535, 0xc8, 0x0072},
	} {
		if got := encodeRumbleHighAmplitude(tt.amp); got != tt.high {
			t.Errorf("high(%d) = %#x, want %#x", tt.amp, got, tt.high)
		}
		if got := encodeRumbleLowAmplitude(tt.amp); got != tt.low {
			t.Errorf("low(%d) = %#x, want %#x", tt.amp, got, tt.low)
		}
	}
	if len(switchHighFreqAmp) != 101 || len(switchLowFreqAmp) != 101 {
		t.Error("the tables have the wrong size")
	}
}

func TestSwitchProHomeLight(t *testing.T) {
	d, _, io := startSwitch(t, BusBluetooth, nil)
	for _, tt := range []struct {
		name    string
		r, g, b uint8
		data    [4]byte
	}{
		{"full", 255, 0, 0, [4]byte{0x01, 0xF0, 0xF0, 0x00}},
		{"off", 0, 0, 0, [4]byte{0x01, 0x00, 0x00, 0x00}},
		{"half", 0, 128, 0, [4]byte{0x01, 0x50, 0x50, 0x00}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			io.sends = nil
			if err := d.SetLED(tt.r, tt.g, tt.b); err != nil {
				t.Fatal(err)
			}
			got := io.lastSend(t)
			if got[10] != switchSubcmdSetHomeLight || [4]byte(got[11:15]) != tt.data {
				t.Errorf("report = %x, want subcommand 0x38 with %x", got[:15], tt.data)
			}
		})
	}
}

func TestSwitchProBluetoothSilence(t *testing.T) {
	d, _, io := startSwitch(t, BusBluetooth, nil)
	var out recSink
	// The pad is in the full mode (reports come as it moves): three seconds without any is a lost link.
	d.Report(time.Second, fullReport(0, [3]byte{}, [2]int{2048, 2048}, [2]int{2048, 2048}, [3][6]int16{}), &out)
	d.Tick(2*time.Second, &out)
	if out.lost != 0 {
		t.Fatal("lost after one second")
	}
	d.Tick(5*time.Second, &out)
	if out.lost != 1 {
		t.Errorf("lost = %d, want 1", out.lost)
	}
	_ = io
}

func TestSwitchProCloseRestoresTheSimpleMode(t *testing.T) {
	d, _, io := startSwitch(t, BusBluetooth, nil)
	n := len(io.writes)
	d.Close()
	if len(io.writes) != n+1 {
		t.Fatalf("%d reports were written, want 1", len(io.writes)-n)
	}
	got := io.writes[n]
	if got[10] != switchSubcmdSetInputReportMod || got[11] != switchInputSimpleState {
		t.Errorf("close report = %x", got[:12])
	}
}

func TestSwitchProUSBAsksForReportsAgain(t *testing.T) {
	d, _, io := startSwitch(t, BusUSB, nil)
	io.sends = nil
	var out recSink
	d.Tick(io.now+50*time.Millisecond, &out)
	if len(io.sends) != 0 {
		t.Fatal("asked again after 50 ms")
	}
	d.Tick(io.now+500*time.Millisecond, &out)
	if len(io.sends) != 1 || io.sends[0][0] != 0x80 || io.sends[0][1] != 0x04 {
		t.Errorf("sends = %x", io.sends)
	}
}
