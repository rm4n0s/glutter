package gamepadhid

import (
	"math"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gamepaddb"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/padstd"
)

// This file is translated to Go and changed from src/joystick/hidapi/SDL_hidapi_ps4.c of SDL 3 (zlib license,
// third_party/sdl/LICENSE): the driver of the PlayStation 4 pad (DualShock 4) and of the pads that behave like it. Left out: the
// wireless adapter and the pads of the Nacon and Armor-X makers, the guitars, drums and other instruments that use the same
// reports, and SDL's hints (the enhanced mode is always on, the report interval is 4 ms).

const (
	ps4ReportUSBState        = 1
	ps4ReportUSBEffects      = 5
	ps4ReportBluetoothState1 = 17
	ps4ReportBluetoothState9 = 25
	ps4ReportBluetoothEffect = 17

	ps4FeatureGyroCalibrationUSB = 0x02
	ps4FeatureCapabilities       = 0x03
	ps4FeatureGyroCalibrationBT  = 0x05

	// ps4StateSize is the size of the packet of the state that follows the report ID (and the two bytes of a Bluetooth report).
	ps4StateSize = 54

	// ps4BluetoothSilence is how long a Bluetooth pad may be quiet before the driver pokes it.
	ps4BluetoothSilence = 500 * time.Millisecond
)

// The effects that an output report asks for: the validity flags of the report.
const (
	ps4EffectRumble = 1 << iota
	ps4EffectLED
	ps4EffectLEDBlink
)

// imuCal is the calibration of one axis of the gyroscope or the accelerometer.
type imuCal struct {
	bias  int16
	scale float32
}

type ps4 struct {
	dev DeviceInfo
	io  IO

	bluetooth, official                            bool
	sensorsOK, lightbarOK, vibrationOK, touchpadOK bool
	effectsOK                                      bool
	enhancedReports, enhancedMode                  bool
	reportInterval                                 uint8
	reportSensors, reportTouchpad, reportBattery   bool
	hardwareCalibration                            bool
	cal                                            [6]imuCal
	lastPacket                                     time.Duration
	rumbleLeft, rumbleRight                        uint8
	colorSet                                       bool
	led                                            [3]uint8
	gyroNum, gyroDen, accelNum, accelDen           uint16
	sensorTicks                                    uint64
	lastTick                                       uint16
	validCRC                                       uint16
	touch                                          touchTracker
	power                                          batteryTracker
	snap                                           padstd.Snapshot
}

func newPS4(info DeviceInfo) Driver {
	return &ps4{dev: info, bluetooth: info.Bus == BusBluetooth, reportInterval: 4, gyroNum: 1, gyroDen: 16, accelNum: 1, accelDen: 8192}
}

func (p *ps4) Start(io IO) (Caps, error) {
	p.io = io
	var data [usbPacketLength * 2]byte
	switch p.dev.Vendor {
	case vendorSony:
		if p.bluetooth {
			// Read a report to see whether the pad is in enhanced mode.
			n, _ := io.Read(data[:], 16*time.Millisecond)
			if n > 0 && data[0] >= ps4ReportBluetoothState1 && data[0] <= ps4ReportBluetoothState9 {
				p.enhancedReports = true
			}
		} else {
			p.enhancedReports = true
		}
	default:
		// Third party pads appear to all be wired.
		p.enhancedReports = true
	}

	if p.dev.Vendor == vendorSony {
		p.official = true
		p.sensorsOK, p.lightbarOK, p.vibrationOK, p.touchpadOK = true, true, true, true
	} else {
		// A third party pad is asked what it can do.
		var feat [usbPacketLength]byte
		n, _ := io.Feature(ps4FeatureCapabilities, feat[:])
		if n == 48 && feat[2] == 0x27 {
			caps := feat[4]
			p.sensorsOK = caps&0x02 != 0
			p.lightbarOK = caps&0x04 != 0
			p.vibrationOK = caps&0x08 != 0
			p.touchpadOK = caps&0x40 != 0
			if gn, gd := ule16(feat[10:]), ule16(feat[12:]); gn != 0 && gd != 0 {
				p.gyroNum, p.gyroDen = gn, gd
			}
			if an, ad := ule16(feat[14:]), ule16(feat[16:]); an != 0 && ad != 0 {
				p.accelNum, p.accelDen = an, ad
			}
		} else if p.dev.Vendor == vendorRazer {
			// The Razer Raiju does not answer the question, but has a touchpad and vibration.
			p.vibrationOK, p.touchpadOK = true, true
		}
	}
	p.effectsOK = p.lightbarOK || p.vibrationOK
	if p.dev.Vendor == vendorPDP && (p.dev.Product == productVictrixFSPro || p.dev.Product == productVictrixFSProV2) {
		// The Victrix FS Pro V2 says it has a light bar, but it does not answer the effects report and hangs on a reboot if it
		// is sent one.
		p.effectsOK = false
	}

	p.lastPacket = io.Now()
	// The enhanced mode is always on: the pad switches to the extended reports when the first effects report reaches it.
	p.setEnhancedModeAvailable()
	if err := p.setEnhancedMode(); err != nil && p.effectsOK {
		return Caps{}, err
	}
	c := Caps{Kind: gamepaddb.KindPS4, Rumble: p.vibrationOK, LED: p.lightbarOK, Sensors: p.sensorsOK, Rate: float32(1000 / int(p.reportInterval))}
	if p.official {
		c.Name = "PS4 Controller"
	}
	if p.touchpadOK {
		c.Touchpads = 1
	}
	c.Buttons = ps4Buttons
	if !p.touchpadOK {
		c.Buttons &^= buttonBit(gamepaddb.Touchpad)
	}
	return c, nil
}

// ps4Buttons are the buttons of the layout that a PlayStation 4 pad has.
var ps4Buttons = buttonBit(gamepaddb.South, gamepaddb.East, gamepaddb.West, gamepaddb.North, gamepaddb.Back, gamepaddb.Guide,
	gamepaddb.Start, gamepaddb.LeftStick, gamepaddb.RightStick, gamepaddb.LeftShoulder, gamepaddb.RightShoulder,
	gamepaddb.DPadUp, gamepaddb.DPadDown, gamepaddb.DPadLeft, gamepaddb.DPadRight, gamepaddb.Touchpad)

func (p *ps4) setEnhancedModeAvailable() {
	if p.touchpadOK {
		p.reportTouchpad = true
	}
	if p.official {
		p.reportBattery = true
	}
}

// setEnhancedMode switches the pad to the enhanced reports by sending it an effects report.
func (p *ps4) setEnhancedMode() error {
	p.enhancedMode = true
	return p.updateEffects(0)
}

// ps4Effects is the part of an output report that holds the rumble and the light (DS4EffectsState_t of SDL), 19 bytes.
type ps4Effects [19]byte

// ledColors are the colours of the light bar for the player numbers: the list that the kernel's hid-sony uses, whose first four
// the PlayStation 4 assigns. A pad has player number 0 here (SPEC §28.2 leaves the players out), which is blue.
var ps4LEDColors = [7][3]uint8{
	{0x00, 0x00, 0x40}, // blue
	{0x40, 0x00, 0x00}, // red
	{0x00, 0x40, 0x00}, // green
	{0x20, 0x00, 0x20}, // pink
	{0x02, 0x01, 0x00}, // orange
	{0x00, 0x01, 0x01}, // teal
	{0x01, 0x01, 0x01}, // white
}

func (p *ps4) updateEffects(mask byte) error {
	var e ps4Effects
	if p.vibrationOK {
		e[0], e[1] = p.rumbleRight, p.rumbleLeft
	}
	if p.lightbarOK {
		if p.colorSet {
			e[2], e[3], e[4] = p.led[0], p.led[1], p.led[2]
		} else {
			c := ps4LEDColors[0]
			e[2], e[3], e[4] = c[0], c[1], c[2]
		}
	}
	return p.sendEffect(e[:], mask)
}

func (p *ps4) sendEffect(effect []byte, mask byte) error {
	if !p.effectsOK {
		return errors.New(platform.TagGamepadRumbleUnavailable, "the pad takes no effects", "vendor", p.dev.Vendor, "product", p.dev.Product)
	}
	if !p.enhancedMode {
		return errors.New(platform.TagGamepadRumbleUnavailable, "the pad is not in enhanced mode", "vendor", p.dev.Vendor, "product", p.dev.Product)
	}
	var data [78]byte
	var size, offset int
	if p.bluetooth && p.official {
		data[0] = ps4ReportBluetoothEffect
		data[1] = 0xC0 | p.reportInterval // magic value: HID and CRC, and the update interval
		// Some third party pads expect rumble and light together, so only an official pad gets them apart.
		data[3] = mask
		size, offset = 78, 6
	} else {
		data[0] = ps4ReportUSBEffects
		data[1] = mask
		if !p.official {
			data[1] = ps4EffectRumble | ps4EffectLED | ps4EffectLEDBlink
		}
		size, offset = 32, 4
	}
	copy(data[offset:], effect)
	if p.bluetooth {
		putCRC(data[:size])
	}
	// A report that is queued already and asks for the same effects is replaced by this one; one that asks for others is not,
	// or the rumble would be lost to a change of the light.
	maskAt := 1
	if p.bluetooth && p.official {
		maskAt = 3
	}
	return p.io.SendMerging(data[:size], func(pending []byte) bool {
		return len(pending) == size && pending[0] == data[0] && pending[maskAt] == data[maskAt]
	})
}

func (p *ps4) Rumble(low, high uint16) error {
	if !p.vibrationOK {
		return errors.New(platform.TagGamepadRumbleUnavailable, "the pad has no motors", "vendor", p.dev.Vendor, "product", p.dev.Product)
	}
	p.rumbleLeft, p.rumbleRight = uint8(low>>8), uint8(high>>8)
	return p.updateEffects(ps4EffectRumble)
}

func (p *ps4) RumbleTriggers(left, right uint16) error {
	return errors.New(platform.TagGamepadRumbleUnavailable, "the triggers of the pad have no motors", "vendor", p.dev.Vendor, "product", p.dev.Product)
}

func (p *ps4) SetLED(r, g, b uint8) error {
	if !p.lightbarOK {
		return errors.New(platform.TagGamepadLEDUnavailable, "the pad has no light bar", "vendor", p.dev.Vendor, "product", p.dev.Product)
	}
	p.colorSet = true
	p.led = [3]uint8{r, g, b}
	return p.updateEffects(ps4EffectLED)
}

func (p *ps4) SetSensors(on bool) error {
	if !p.sensorsOK {
		return errors.New(platform.TagGamepadSensorsUnavailable, "the pad has no motion sensors", "vendor", p.dev.Vendor, "product", p.dev.Product)
	}
	if on {
		p.loadCalibration()
	}
	p.reportSensors = on
	return nil
}

func (p *ps4) Close() {}

// loadOfficialCalibration reads the calibration of the sensors that the pad keeps (feature reports 0x02 and, on Bluetooth, 0x05),
// and reports whether the numbers are usable.
func (p *ps4) loadOfficialCalibration() bool {
	if !p.official {
		return false
	}
	var data [usbPacketLength]byte
	var size int
	haveData := false
	for tries := 0; tries < 5; tries++ {
		// For a Bluetooth pad this report switches it into the enhanced mode.
		n, _ := p.io.Feature(ps4FeatureGyroCalibrationUSB, data[:])
		size = n
		if size < 35 {
			return false
		}
		if p.bluetooth {
			n, _ = p.io.Feature(ps4FeatureGyroCalibrationBT, data[:])
			size = n
			if size < 35 {
				return false
			}
		}
		// In some cases the report is all zeros.
		for _, b := range data[:size] {
			if b != 0 {
				haveData = true
				break
			}
		}
		if haveData {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}
	if !haveData {
		return false
	}
	pitchBias, yawBias, rollBias := int(le16(data[1:])), int(le16(data[3:])), int(le16(data[5:]))
	var pitchPlus, pitchMinus, yawPlus, yawMinus, rollPlus, rollMinus int
	if p.bluetooth {
		pitchPlus, yawPlus, rollPlus = int(le16(data[7:])), int(le16(data[9:])), int(le16(data[11:]))
		pitchMinus, yawMinus, rollMinus = int(le16(data[13:])), int(le16(data[15:])), int(le16(data[17:]))
	} else {
		pitchPlus, pitchMinus = int(le16(data[7:])), int(le16(data[9:]))
		yawPlus, yawMinus = int(le16(data[11:])), int(le16(data[13:]))
		rollPlus, rollMinus = int(le16(data[15:])), int(le16(data[17:]))
	}
	speedPlus, speedMinus := int(le16(data[19:])), int(le16(data[21:]))
	accXPlus, accXMinus := le16(data[23:]), le16(data[25:])
	accYPlus, accYMinus := le16(data[27:]), le16(data[29:])
	accZPlus, accZMinus := le16(data[31:]), le16(data[33:])

	numerator := float32(speedPlus+speedMinus) * float32(p.gyroDen) / float32(p.gyroNum)
	gyro := func(i, bias, plus, minus int) {
		den := float32(abs(plus-bias) + abs(minus-bias))
		if den != 0 {
			p.cal[i] = imuCal{int16(bias), numerator / den}
		}
	}
	gyro(0, pitchBias, pitchPlus, pitchMinus)
	gyro(1, yawBias, yawPlus, yawMinus)
	gyro(2, rollBias, rollPlus, rollMinus)
	accel := func(i int, plus, minus int16) {
		r := plus - minus
		p.cal[i] = imuCal{plus - r/2, (2 * float32(p.accelDen) / float32(p.accelNum)) / float32(r)}
	}
	accel(3, accXPlus, accXMinus)
	accel(4, accYPlus, accYMinus)
	accel(5, accZPlus, accZMinus)

	p.hardwareCalibration = true
	for _, c := range p.cal {
		// Some pads have a bad calibration.
		if abs(int(c.bias)) > 1024 || math.Abs(float64(1-c.scale)) > 0.5 {
			p.hardwareCalibration = false
		}
	}
	return p.hardwareCalibration
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// loadCalibration sets the calibration of the sensors: the pad's own when it is good, else none, and scales it to rad/s and m/s².
func (p *ps4) loadCalibration() {
	if !p.loadOfficialCalibration() {
		for i := range p.cal {
			p.cal[i] = imuCal{0, 1}
		}
	}
	for i := range p.cal {
		scale := float64(p.cal[i].scale)
		if i < 3 {
			scale *= (float64(p.gyroNum) / float64(p.gyroDen)) * math.Pi / 180
		} else {
			scale *= (float64(p.accelNum) / float64(p.accelDen)) * standardGravity
		}
		p.cal[i].scale = float32(scale)
	}
}

func (p *ps4) applyCalibration(i int, v int16) float32 {
	return (float32(v) - float32(p.cal[i].bias)) * p.cal[i].scale
}

// packetValid says whether an input report is one to use: it is of a known kind and, on Bluetooth, its CRC is good.
func (p *ps4) packetValid(data []byte) bool {
	size := len(data)
	switch {
	case data[0] == ps4ReportUSBState:
		if size == 10 {
			// The simple report of a pad that is not in enhanced mode.
			return true
		}
		// Bit 2 of byte 31 says, on the wireless adapter, whether a pad is connected (0); on the pad itself it is always 0.
		return size >= 64 && data[31]&0x04 == 0
	case data[0] >= ps4ReportBluetoothState1 && data[0] <= ps4ReportBluetoothState9:
		// A Bluetooth report has two more bytes at the start: the first says whether HID data is in it.
		if size >= 78 && data[1]&0x80 != 0 {
			if verifyCRC(data[:78]) {
				p.validCRC++
			} else {
				if p.validCRC > 0 {
					p.validCRC--
				}
				if p.validCRC >= 3 {
					// The CRC is generally good, but this one failed.
					return false
				}
			}
			return true
		}
	}
	return false
}

func (p *ps4) Report(now time.Duration, data []byte, out Sink) {
	if len(data) == 0 || !p.packetValid(data) {
		return
	}
	p.lastPacket = now
	switch data[0] {
	case ps4ReportUSBState:
		p.handleState(now, data[1:], out)
	default:
		// The extended report: the pad is in enhanced mode.
		p.enhancedReports = true
		if len(data) > 3 {
			p.handleState(now, data[3:], out)
		}
	}
}

const (
	ps4TouchpadScaleX = 5.20833333e-4 // 1 / 1920
	ps4TouchpadScaleY = 1.08695652e-3 // 1 / 920; the pad's height is noted as 944, but 920 feels better
)

func (p *ps4) handleState(now time.Duration, body []byte, out Sink) {
	size := len(body)
	var pk [ps4StateSize]byte
	copy(pk[:], body)
	s := &p.snap

	if size > 9 && p.reportTouchpad && p.enhancedReports {
		for finger := 0; finger < 2; finger++ {
			c := pk[34+finger*4]
			d := pk[35+finger*4 : 38+finger*4]
			down := c&0x80 == 0
			x := int(d[0]) | (int(d[1])&0x0F)<<8
			y := int(d[1]>>4) | int(d[2])<<4
			var pressure float32
			if down {
				pressure = 1
			}
			p.touch.send(out, now, 0, finger, down, float32(x)*ps4TouchpadScaleX, float32(y)*ps4TouchpadScaleY, pressure)
		}
	}

	b0 := pk[4]
	face := b0 >> 4
	s.Buttons[gamepaddb.West] = face&0x01 != 0
	s.Buttons[gamepaddb.South] = face&0x02 != 0
	s.Buttons[gamepaddb.East] = face&0x04 != 0
	s.Buttons[gamepaddb.North] = face&0x08 != 0
	setHat(s, b0&0x0F)

	b1 := pk[5]
	s.Buttons[gamepaddb.LeftShoulder] = b1&0x01 != 0
	s.Buttons[gamepaddb.RightShoulder] = b1&0x02 != 0
	s.Buttons[gamepaddb.Back] = b1&0x10 != 0
	s.Buttons[gamepaddb.Start] = b1&0x20 != 0
	s.Buttons[gamepaddb.LeftStick] = b1&0x40 != 0
	s.Buttons[gamepaddb.RightStick] = b1&0x80 != 0

	// Some fight sticks (the Victrix FS Pro) send only the digital trigger bits and not the analog values.
	left, right := pk[7], pk[8]
	if b1&0x0C != 0 {
		if b1&0x04 != 0 && left == 0 {
			left = 255
		}
		if b1&0x08 != 0 && right == 0 {
			right = 255
		}
	}

	b2 := pk[6]
	s.Buttons[gamepaddb.Guide] = b2&0x01 != 0
	s.Buttons[gamepaddb.Touchpad] = b2&0x02 != 0

	s.Axes[gamepaddb.LeftTrigger] = padstd.Trigger8(left)
	s.Axes[gamepaddb.RightTrigger] = padstd.Trigger8(right)
	s.Axes[gamepaddb.LeftX] = stick8(pk[0])
	s.Axes[gamepaddb.LeftY] = stick8(pk[1])
	s.Axes[gamepaddb.RightX] = stick8(pk[2])
	s.Axes[gamepaddb.RightY] = stick8(pk[3])

	if size > 9 && p.reportBattery && p.enhancedReports {
		level := pk[29] & 0x0F
		switch {
		case pk[29]&0x10 == 0:
			p.power.send(out, platform.PadOnBattery, powerPercent(level))
		case level <= 10:
			p.power.send(out, platform.PadCharging, powerPercent(level))
		case level == 11:
			p.power.send(out, platform.PadCharged, 100)
		default:
			p.power.send(out, platform.PadPowerUnknown, 0)
		}
	}

	if size > 9 && p.reportSensors {
		tick := ule16(pk[9:])
		var delta uint16
		if p.lastTick <= tick {
			delta = tick - p.lastTick
		} else {
			delta = math.MaxUint16 - p.lastTick + tick + 1
		}
		p.sensorTicks += uint64(delta)
		p.lastTick = tick
		// The sensor's clock runs in units of 5.33 µs.
		st := time.Duration(p.sensorTicks * 1000 * 16 / 3)
		out.Sensor(now, platform.PadSensor{Kind: platform.PadGyroscope, SensorTime: st, Values: [3]float32{
			p.applyCalibration(0, le16(pk[12:])), p.applyCalibration(1, le16(pk[14:])), p.applyCalibration(2, le16(pk[16:]))}})
		out.Sensor(now, platform.PadSensor{Kind: platform.PadAccelerometer, SensorTime: st, Values: [3]float32{
			p.applyCalibration(3, le16(pk[18:])), p.applyCalibration(4, le16(pk[20:])), p.applyCalibration(5, le16(pk[22:]))}})
	}
	out.State(now, s)
}

func (p *ps4) Tick(now time.Duration, out Sink) {
	if p.bluetooth && now >= p.lastPacket+ps4BluetoothSilence {
		// The link went quiet: an empty output report pokes the Bluetooth stack. Without the CRC the pad ignores it.
		p.tickleBluetooth()
		p.lastPacket = now
	}
}

func (p *ps4) tickleBluetooth() {
	if !p.enhancedReports {
		return
	}
	var data [78]byte
	data[0] = ps4ReportBluetoothEffect
	data[1] = 0xC0
	_ = p.io.Send(data[:]) // a poke that is not sent is not missed: the next one follows
}
