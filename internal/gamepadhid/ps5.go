package gamepadhid

import (
	"math"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gamepaddb"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/padstd"
)

// This file is translated to Go and changed from src/joystick/hidapi/SDL_hidapi_ps5.c of SDL 3 (zlib license,
// third_party/sdl/LICENSE): the driver of the PlayStation 5 pad (DualSense, DualSense Edge) and of the pads that behave like it.
// Left out: the wireless adapters and the Razer and Nacon pads that need their own cases, the guitars, and SDL's hints (the
// enhanced mode is always on, the player lights are on). The trigger rumble is not SDL's: SDL sends adaptive trigger effects
// only through its effect call, so the vibration effect of the triggers is made here (see ps5TriggerVibration).

const (
	ps5ReportState            = 0x01
	ps5ReportUSBEffects       = 0x02
	ps5ReportBluetoothEffects = 0x31
	ps5ReportBluetoothState   = 0x31

	ps5FeatureCapabilities = 0x03
	ps5FeatureCalibration  = 0x05
	ps5FeatureSerial       = 0x09
	ps5FeatureFirmware     = 0x20

	ps5GyroResPerDegree = 1024.0
	ps5AccelResPerG     = 8192.0

	ps5BluetoothSilence = 500 * time.Millisecond
	// ps5LEDConnectionComplete is the value of the pad's timer (0.33 µs units) after which the light animation that plays when a
	// Bluetooth connection is made is over.
	ps5LEDConnectionComplete = 10200000
)

// The effects that an output report asks for (EDS5Effect of SDL).
const (
	ps5EffectRumbleStart = 1 << iota
	ps5EffectRumble
	ps5EffectLEDReset
	ps5EffectLED
	ps5EffectPadLights
	ps5EffectMicLight
	ps5EffectTriggers
)

// ps5Effects is the part of an output report that holds the effects (DS5EffectsState_t of SDL), 47 bytes.
type ps5Effects [47]byte

// The offsets in ps5Effects.
const (
	ps5EnableBits1   = 0
	ps5EnableBits2   = 1
	ps5RumbleRight   = 2
	ps5RumbleLeft    = 3
	ps5MicLightMode  = 8
	ps5RightTrigger  = 10
	ps5LeftTrigger   = 21
	ps5EnableBits3   = 38
	ps5PadLights     = 43
	ps5LEDRed        = 44
	ps5TriggerLength = 11
)

type ps5 struct {
	dev DeviceInfo
	io  IO

	bluetooth                                      bool
	useAlternateReport                             bool
	sensorsOK, lightbarOK, vibrationOK, playerLEDs bool
	touchpadOK, effectsOK                          bool
	enhancedReports, enhancedMode                  bool
	reportSensors, reportTouchpad, reportBattery   bool
	hardwareCalibration                            bool
	cal                                            [6]struct {
		bias        int16
		sensitivity float32
	}
	firmware        uint16
	lastPacket      time.Duration
	playerLights    bool
	enhancedRumble  bool
	rumbleLeft      uint8
	rumbleRight     uint8
	triggerLeft     uint16
	triggerRight    uint16
	colorSet        bool
	led             [3]uint8
	ledReset        int
	sensorTicks     uint64
	lastTick        uint32
	lastSensorTicks uint32
	touch           touchTracker
	power           batteryTracker
	snap            padstd.Snapshot
	edge            bool
	buttons         uint32
}

const (
	ledResetNone = iota
	ledResetPending
	ledResetComplete
)

func newPS5(info DeviceInfo) Driver {
	return &ps5{dev: info, bluetooth: info.Bus == BusBluetooth, playerLights: true, edge: info.Vendor == vendorSony && info.Product == productDualSenseEdge}
}

func (p *ps5) Start(io IO) (Caps, error) {
	p.io = io
	var data [usbPacketLength * 2]byte
	// Read a report to see what mode the pad is in.
	n, _ := io.Read(data[:], 16*time.Millisecond)
	switch {
	case n == 64:
		// Connected over USB.
		p.enhancedReports = true
	case n > 0 && data[0] == ps5ReportBluetoothEffects:
		// Connected over Bluetooth, using the enhanced reports.
		p.enhancedReports = true
	}
	// Otherwise it is connected over Bluetooth, using the simple reports.

	if p.dev.Vendor == vendorSony && p.enhancedReports {
		// Reading these reports also switches a Bluetooth pad to the enhanced reports.
		var feat [usbPacketLength * 2]byte
		_, _ = io.Feature(ps5FeatureSerial, feat[:]) // the serial number (the Bluetooth address) is not used
		var fw [usbPacketLength]byte
		if n, _ := io.Feature(ps5FeatureFirmware, fw[:]); n >= 46 {
			p.firmware = uint16(fw[44]) | uint16(fw[45])<<8
		}
	}
	if p.dev.Vendor == vendorSony && (p.edge || p.firmware == 0 || p.firmware >= 0x0224) {
		// Firmware 2.24 and later, and the Edge, have improved rumble emulation; no firmware number over Bluetooth is taken for
		// updated firmware.
		p.enhancedRumble = true
	}

	if p.dev.Vendor == vendorSony {
		p.sensorsOK, p.lightbarOK, p.vibrationOK, p.playerLEDs, p.touchpadOK = true, true, true, true, true
	} else {
		var feat [usbPacketLength]byte
		n, _ := io.Feature(ps5FeatureCapabilities, feat[:])
		if n == 48 && feat[2] == 0x28 {
			caps, caps2 := feat[4], feat[20]
			p.sensorsOK = caps&0x02 != 0
			p.lightbarOK = caps&0x04 != 0
			p.vibrationOK = caps&0x08 != 0
			p.touchpadOK = caps&0x40 != 0
			p.playerLEDs = caps2&0x80 != 0
			p.useAlternateReport = true
			p.reportBattery = true
		}
	}
	p.effectsOK = p.lightbarOK || p.vibrationOK || p.playerLEDs

	p.lastPacket = io.Now()
	p.ledReset = ledResetNone
	// The enhanced mode is always on.
	p.setEnhancedModeAvailable()
	p.enhancedMode = true
	if err := p.updateEffects(0); err != nil && p.effectsOK {
		return Caps{}, err
	}
	_ = p.updateEffects(ps5EffectLED | ps5EffectPadLights) // the light is set when the connection animation is over; a pad that refuses still reads
	c := Caps{Kind: gamepaddb.KindPS5, Rumble: p.vibrationOK, RumbleTriggers: p.vibrationOK, LED: p.lightbarOK, Sensors: p.sensorsOK}
	if p.dev.Vendor == vendorSony {
		c.Name = "DualSense Wireless Controller"
		if p.edge {
			c.Name = "DualSense Edge Wireless Controller"
		}
	}
	if p.sensorsOK {
		// The sensors report 250 times a second over USB and 1000 over Bluetooth; the Edge does 1000 over USB too.
		c.Rate = 250
		if p.bluetooth || p.edge {
			c.Rate = 1000
		}
	}
	if p.touchpadOK {
		c.Touchpads = 1
	}
	c.Buttons = ps5Buttons
	if p.edge {
		c.Buttons |= buttonBit(gamepaddb.Paddle1, gamepaddb.Paddle2, gamepaddb.Paddle3, gamepaddb.Paddle4)
	}
	if !p.touchpadOK {
		c.Buttons &^= buttonBit(gamepaddb.Touchpad)
	}
	p.buttons = c.Buttons
	return c, nil
}

// ps5Buttons are the buttons of the layout that a PlayStation 5 pad has (the Edge has paddles as well).
var ps5Buttons = buttonBit(gamepaddb.South, gamepaddb.East, gamepaddb.West, gamepaddb.North, gamepaddb.Back, gamepaddb.Guide,
	gamepaddb.Start, gamepaddb.LeftStick, gamepaddb.RightStick, gamepaddb.LeftShoulder, gamepaddb.RightShoulder,
	gamepaddb.DPadUp, gamepaddb.DPadDown, gamepaddb.DPadLeft, gamepaddb.DPadRight, gamepaddb.Touchpad, gamepaddb.Misc1)

func (p *ps5) setEnhancedModeAvailable() {
	if p.touchpadOK {
		p.reportTouchpad = true
	}
	p.reportBattery = true
}

// ps5LEDColors are the colours of the light bar for the player numbers (the first four are the PlayStation's).
var ps5LEDColors = [7][3]uint8{
	{0x00, 0x00, 0x40}, // blue
	{0x40, 0x00, 0x00}, // red
	{0x00, 0x40, 0x00}, // green
	{0x20, 0x00, 0x20}, // pink
	{0x20, 0x10, 0x00}, // orange
	{0x00, 0x10, 0x10}, // teal
	{0x10, 0x10, 0x10}, // white
}

func (p *ps5) updateEffects(mask int) error {
	// The Bluetooth connection sequence has to finish before the light is changed.
	if p.bluetooth && p.enhancedReports && mask&(ps5EffectLED|ps5EffectPadLights) != 0 {
		if p.ledReset != ledResetComplete {
			p.ledReset = ledResetPending
			return nil
		}
	}
	var e ps5Effects
	if p.vibrationOK {
		if p.rumbleLeft != 0 || p.rumbleRight != 0 {
			if p.enhancedRumble {
				e[ps5EnableBits3] |= 0x04 // improved rumble emulation, firmware 2.24 and newer
				e[ps5RumbleLeft], e[ps5RumbleRight] = p.rumbleLeft, p.rumbleRight
			} else {
				e[ps5EnableBits1] |= 0x01 // rumble emulation
				// Shifted, to bring the strength down to what an Xbox pad gives.
				e[ps5RumbleLeft], e[ps5RumbleRight] = p.rumbleLeft>>1, p.rumbleRight>>1
			}
			e[ps5EnableBits1] |= 0x02 // no audio haptics
		}
		// Rumble emulation bits that stay off give the audio haptics back.
		if mask&ps5EffectRumbleStart != 0 {
			e[ps5EnableBits1] |= 0x02
		}
	}
	if p.lightbarOK {
		if mask&ps5EffectLEDReset != 0 {
			e[ps5EnableBits2] |= 0x08 // reset the light
		}
		if mask&ps5EffectLED != 0 {
			e[ps5EnableBits2] |= 0x04 // set the colour
			if p.colorSet {
				e[ps5LEDRed], e[ps5LEDRed+1], e[ps5LEDRed+2] = p.led[0], p.led[1], p.led[2]
			} else {
				c := ps5LEDColors[0]
				e[ps5LEDRed], e[ps5LEDRed+1], e[ps5LEDRed+2] = c[0], c[1], c[2]
			}
		}
	}
	if p.playerLEDs && mask&ps5EffectPadLights != 0 {
		e[ps5EnableBits2] |= 0x10 // the lights of the touchpad
		// No player number: the lights are off.
		e[ps5PadLights] = 0
	}
	if mask&ps5EffectMicLight != 0 {
		e[ps5EnableBits2] |= 0x01 // the microphone light, off
		e[ps5MicLightMode] = 0
	}
	if mask&ps5EffectTriggers != 0 && p.vibrationOK {
		// The trigger motors: an effect for each trigger, which stays set until it is replaced.
		e[ps5EnableBits1] |= 0x04 | 0x08
		ps5TriggerVibration(e[ps5RightTrigger:ps5RightTrigger+ps5TriggerLength], p.triggerRight)
		ps5TriggerVibration(e[ps5LeftTrigger:ps5LeftTrigger+ps5TriggerLength], p.triggerLeft)
	}
	return p.sendEffect(e[:])
}

// ps5TriggerVibration makes the effect of a trigger motor in the 11 bytes the report has for it. DualSense pads have no motors
// in the triggers: what the app gets is the vibration mode of their adaptive triggers, which makes the resistance buzz. The
// mode is 0x26, with ten zones of three bits of strength that are on from the start of the travel, and the frequency in hertz in
// the tenth byte; a strength of 0 is the off mode, 0x05. The strength (0 to 65535) becomes the amplitude 1 to 8.
func ps5TriggerVibration(b []byte, strength uint16) {
	clear(b)
	if strength == 0 {
		b[0] = 0x05
		return
	}
	amp := uint32(strength)*8/65536 + 1 // 1 to 8
	var zones uint32
	var active uint16
	for i := 0; i < 10; i++ {
		zones |= (amp - 1) << (3 * i)
		active |= 1 << i
	}
	b[0] = 0x26
	b[1], b[2] = byte(active), byte(active>>8)
	b[3], b[4], b[5], b[6] = byte(zones), byte(zones>>8), byte(zones>>16), byte(zones>>24)
	b[9] = 40 // hertz
}

func (p *ps5) sendEffect(effect []byte) error {
	if !p.effectsOK {
		return errors.New(platform.TagGamepadRumbleUnavailable, "the pad takes no effects", "vendor", p.dev.Vendor, "product", p.dev.Product)
	}
	if !p.enhancedMode {
		return errors.New(platform.TagGamepadRumbleUnavailable, "the pad is not in enhanced mode", "vendor", p.dev.Vendor, "product", p.dev.Product)
	}
	var data [78]byte
	var size, offset int
	if p.bluetooth {
		data[0] = ps5ReportBluetoothEffects
		data[1] = 0x00 // tag and sequence
		data[2] = 0x10 // magic value
		size, offset = 78, 3
	} else {
		data[0] = ps5ReportUSBEffects
		size, offset = 48, 1
	}
	copy(data[offset:], effect)
	if p.bluetooth {
		putCRC(data[:size])
	}
	// A report that is queued already and asks for the same effects is replaced by this one.
	return p.io.SendMerging(data[:size], func(pending []byte) bool {
		return len(pending) == size && pending[offset+ps5EnableBits1] == data[offset+ps5EnableBits1] &&
			pending[offset+ps5EnableBits2] == data[offset+ps5EnableBits2]
	})
}

func (p *ps5) Rumble(low, high uint16) error {
	if !p.vibrationOK {
		return errors.New(platform.TagGamepadRumbleUnavailable, "the pad has no motors", "vendor", p.dev.Vendor, "product", p.dev.Product)
	}
	if p.rumbleLeft == 0 && p.rumbleRight == 0 {
		if err := p.updateEffects(ps5EffectRumbleStart); err != nil {
			return err
		}
	}
	p.rumbleLeft, p.rumbleRight = uint8(low>>8), uint8(high>>8)
	return p.updateEffects(ps5EffectRumble)
}

func (p *ps5) RumbleTriggers(left, right uint16) error {
	if !p.vibrationOK {
		return errors.New(platform.TagGamepadRumbleUnavailable, "the pad has no triggers that vibrate", "vendor", p.dev.Vendor, "product", p.dev.Product)
	}
	p.triggerLeft, p.triggerRight = left, right
	return p.updateEffects(ps5EffectTriggers)
}

func (p *ps5) SetLED(r, g, b uint8) error {
	if !p.lightbarOK {
		return errors.New(platform.TagGamepadLEDUnavailable, "the pad has no light bar", "vendor", p.dev.Vendor, "product", p.dev.Product)
	}
	p.colorSet = true
	p.led = [3]uint8{r, g, b}
	return p.updateEffects(ps5EffectLED)
}

func (p *ps5) SetSensors(on bool) error {
	if !p.sensorsOK {
		return errors.New(platform.TagGamepadSensorsUnavailable, "the pad has no motion sensors", "vendor", p.dev.Vendor, "product", p.dev.Product)
	}
	if on {
		p.loadCalibration()
	}
	p.reportSensors = on
	return nil
}

func (p *ps5) Close() {}

// loadCalibration reads the calibration of the sensors that the pad keeps (feature report 0x05).
func (p *ps5) loadCalibration() {
	var data [usbPacketLength]byte
	n, _ := p.io.Feature(ps5FeatureCalibration, data[:])
	if n < 35 {
		return
	}
	pitchBias, yawBias, rollBias := le16(data[1:]), le16(data[3:]), le16(data[5:])
	pitchPlus, pitchMinus := int(le16(data[7:])), int(le16(data[9:]))
	yawPlus, yawMinus := int(le16(data[11:])), int(le16(data[13:]))
	rollPlus, rollMinus := int(le16(data[15:])), int(le16(data[17:]))
	speedPlus, speedMinus := int(le16(data[19:])), int(le16(data[21:]))
	accXPlus, accXMinus := le16(data[23:]), le16(data[25:])
	accYPlus, accYMinus := le16(data[27:]), le16(data[29:])
	accZPlus, accZMinus := le16(data[31:]), le16(data[33:])

	numerator := float32(speedPlus+speedMinus) * ps5GyroResPerDegree
	gyro := func(i int, bias int16, plus, minus int) {
		p.cal[i].bias = bias
		p.cal[i].sensitivity = numerator / float32(plus-minus)
	}
	gyro(0, pitchBias, pitchPlus, pitchMinus)
	gyro(1, yawBias, yawPlus, yawMinus)
	gyro(2, rollBias, rollPlus, rollMinus)
	accel := func(i int, plus, minus int16) {
		r := plus - minus
		p.cal[i].bias = plus - r/2
		p.cal[i].sensitivity = 2 * ps5AccelResPerG / float32(r)
	}
	accel(3, accXPlus, accXMinus)
	accel(4, accYPlus, accYMinus)
	accel(5, accZPlus, accZMinus)

	p.hardwareCalibration = true
	for i, c := range p.cal {
		divisor := float32(1)
		if i < 3 {
			divisor = 64
		}
		// Some pads have a bad calibration.
		if abs(int(c.bias)) > 1024 || math.Abs(float64(1-c.sensitivity/divisor)) > 0.5 {
			p.hardwareCalibration = false
		}
	}
}

func (p *ps5) applyCalibration(i int, v int16) float32 {
	var r float32
	switch {
	case p.hardwareCalibration:
		r = (float32(v) - float32(p.cal[i].bias)) * p.cal[i].sensitivity
	case i < 3:
		r = float32(v) * 64
	default:
		r = float32(v)
	}
	// The units of the sensors: rad/s and m/s².
	if i < 3 {
		return (r / ps5GyroResPerDegree) * math.Pi / 180
	}
	return (r / ps5AccelResPerG) * standardGravity
}

func (p *ps5) packetValid(data []byte) bool {
	switch data[0] {
	case ps5ReportState:
		return true
	case ps5ReportBluetoothState:
		return len(data) > 4 && verifyCRC(data)
	}
	return false
}

func (p *ps5) Report(now time.Duration, data []byte, out Sink) {
	if len(data) == 0 || !p.packetValid(data) {
		return
	}
	p.lastPacket = now
	switch data[0] {
	case ps5ReportState:
		if len(data) == 10 || len(data) == 78 {
			p.handleSimple(now, data[1:], out)
		} else {
			p.handleEnhanced(now, data[1:], out)
		}
	case ps5ReportBluetoothState:
		// The enhanced report: effects are allowed now.
		p.enhancedReports = true
		if len(data) > 2 {
			p.handleEnhanced(now, data[2:], out)
		}
		if p.ledReset == ledResetPending {
			p.checkPendingLEDReset()
		}
	}
}

// ps5StateSize is the size of the packet of the enhanced state (the full layout is the longest).
const ps5StateSize = 54

func (p *ps5) handleSimple(now time.Duration, body []byte, out Sink) {
	var pk [9]byte
	copy(pk[:], body)
	s := &p.snap
	face := pk[4] >> 4
	s.Buttons[gamepaddb.West] = face&0x01 != 0
	s.Buttons[gamepaddb.South] = face&0x02 != 0
	s.Buttons[gamepaddb.East] = face&0x04 != 0
	s.Buttons[gamepaddb.North] = face&0x08 != 0
	setHat(s, pk[4]&0x0F)
	b1 := pk[5]
	s.Buttons[gamepaddb.LeftShoulder] = b1&0x01 != 0
	s.Buttons[gamepaddb.RightShoulder] = b1&0x02 != 0
	s.Buttons[gamepaddb.Back] = b1&0x10 != 0
	s.Buttons[gamepaddb.Start] = b1&0x20 != 0
	s.Buttons[gamepaddb.LeftStick] = b1&0x40 != 0
	s.Buttons[gamepaddb.RightStick] = b1&0x80 != 0
	b2 := pk[6] & 0x03
	s.Buttons[gamepaddb.Guide] = b2&0x01 != 0
	s.Buttons[gamepaddb.Touchpad] = b2&0x02 != 0
	left, right := pk[7], pk[8]
	s.Axes[gamepaddb.LeftTrigger] = trigger(left, b1&0x04 != 0)
	s.Axes[gamepaddb.RightTrigger] = trigger(right, b1&0x08 != 0)
	s.Axes[gamepaddb.LeftX] = stick8(pk[0])
	s.Axes[gamepaddb.LeftY] = stick8(pk[1])
	s.Axes[gamepaddb.RightX] = stick8(pk[2])
	s.Axes[gamepaddb.RightY] = stick8(pk[3])
	out.State(now, s)
}

// trigger is the axis of a trigger of one byte, with its digital button: a trigger that reads 0 and whose button is down is all
// the way down.
func trigger(v uint8, digital bool) int16 {
	if v == 0 && digital {
		return gamepaddb.AxisMax
	}
	return padstd.Trigger8(v)
}

const (
	ps5TouchpadScaleX = 5.20833333e-4 // 1 / 1920
	ps5TouchpadScaleY = 9.34579439e-4 // 1 / 1070
)

// handleEnhanced takes in the enhanced state; the layout of its packet depends on the pad (use_alternate_report).
func (p *ps5) handleEnhanced(now time.Duration, body []byte, out Sink) {
	var pk [ps5StateSize]byte
	copy(pk[:], body)
	// The offsets that differ between the layouts: the battery, the touchpad, and the width of the sensor timestamp.
	battery, touch1 := 52, 32
	if p.useAlternateReport {
		battery, touch1 = 29, 31
	}
	if p.reportTouchpad {
		for finger := 0; finger < 2; finger++ {
			c := pk[touch1+finger*4]
			d := pk[touch1+1+finger*4 : touch1+4+finger*4]
			down := c&0x80 == 0
			x := int(d[0]) | (int(d[1])&0x0F)<<8
			y := int(d[1]>>4) | int(d[2])<<4
			var pressure float32
			if down {
				pressure = 1
			}
			p.touch.send(out, now, 0, finger, down, float32(x)*ps5TouchpadScaleX, float32(y)*ps5TouchpadScaleY, pressure)
		}
	}
	if p.reportBattery {
		status, level := pk[battery]>>4&0x0F, pk[battery]&0x0F
		// 0x0C means that a pad does not report its battery.
		if !p.useAlternateReport || level != 0x0C {
			switch status {
			case 0:
				p.power.send(out, platform.PadOnBattery, powerPercent(level))
			case 1:
				p.power.send(out, platform.PadCharging, powerPercent(level))
			case 2:
				p.power.send(out, platform.PadCharged, 100)
			default:
				p.power.send(out, platform.PadPowerUnknown, 0)
			}
		}
	}
	p.lastSensorTicks = ule32(pk[27:])
	p.handleCommon(now, &pk, out)
}

func (p *ps5) handleCommon(now time.Duration, pk *[ps5StateSize]byte, out Sink) {
	s := &p.snap
	b0 := pk[7]
	face := b0 >> 4
	s.Buttons[gamepaddb.West] = face&0x01 != 0
	s.Buttons[gamepaddb.South] = face&0x02 != 0
	s.Buttons[gamepaddb.East] = face&0x04 != 0
	s.Buttons[gamepaddb.North] = face&0x08 != 0
	setHat(s, b0&0x0F)
	b1 := pk[8]
	s.Buttons[gamepaddb.LeftShoulder] = b1&0x01 != 0
	s.Buttons[gamepaddb.RightShoulder] = b1&0x02 != 0
	s.Buttons[gamepaddb.Back] = b1&0x10 != 0
	s.Buttons[gamepaddb.Start] = b1&0x20 != 0
	s.Buttons[gamepaddb.LeftStick] = b1&0x40 != 0
	s.Buttons[gamepaddb.RightStick] = b1&0x80 != 0
	b2 := pk[9]
	s.Buttons[gamepaddb.Guide] = b2&0x01 != 0
	s.Buttons[gamepaddb.Touchpad] = b2&0x02 != 0
	s.Buttons[gamepaddb.Misc1] = b2&0x04 != 0
	// The Edge's function buttons and paddles, numbered as SDL's mapping of the pad numbers them.
	s.Buttons[gamepaddb.Paddle4] = b2&0x10 != 0 // left function
	s.Buttons[gamepaddb.Paddle3] = b2&0x20 != 0 // right function
	s.Buttons[gamepaddb.Paddle2] = b2&0x40 != 0 // left paddle
	s.Buttons[gamepaddb.Paddle1] = b2&0x80 != 0 // right paddle
	s.Axes[gamepaddb.LeftTrigger] = trigger(pk[4], b1&0x04 != 0)
	s.Axes[gamepaddb.RightTrigger] = trigger(pk[5], b1&0x08 != 0)
	s.Axes[gamepaddb.LeftX] = stick8(pk[0])
	s.Axes[gamepaddb.LeftY] = stick8(pk[1])
	s.Axes[gamepaddb.RightX] = stick8(pk[2])
	s.Axes[gamepaddb.RightY] = stick8(pk[3])

	if p.reportSensors {
		var st time.Duration
		if p.useAlternateReport {
			// A timestamp of 16 bits, in microseconds.
			tick := uint32(ule16(pk[27:]))
			var delta uint32
			if p.lastTick <= tick {
				delta = tick - p.lastTick
			} else {
				delta = math.MaxUint16 - p.lastTick + tick + 1
			}
			p.lastTick = tick
			p.sensorTicks += uint64(delta)
			st = time.Duration(p.sensorTicks) * time.Microsecond
		} else {
			// A timestamp of 32 bits, in units of 0.33 µs.
			tick := ule32(pk[27:])
			var delta uint32
			if p.lastTick < tick {
				delta = tick - p.lastTick
			} else {
				delta = math.MaxUint32 - p.lastTick + tick + 1
			}
			p.lastTick = tick
			p.sensorTicks += uint64(delta)
			st = time.Duration(p.sensorTicks * 1000 / 3)
		}
		out.Sensor(now, platform.PadSensor{Kind: platform.PadGyroscope, SensorTime: st, Values: [3]float32{
			p.applyCalibration(0, le16(pk[15:])), p.applyCalibration(1, le16(pk[17:])), p.applyCalibration(2, le16(pk[19:]))}})
		out.Sensor(now, platform.PadSensor{Kind: platform.PadAccelerometer, SensorTime: st, Values: [3]float32{
			p.applyCalibration(3, le16(pk[21:])), p.applyCalibration(4, le16(pk[23:])), p.applyCalibration(5, le16(pk[25:]))}})
	}
	out.State(now, s)
}

// checkPendingLEDReset sends the light settings that waited for the end of the connection animation of a Bluetooth pad.
func (p *ps5) checkPendingLEDReset() {
	complete := true
	if p.enhancedReports && p.sensorsOK && !p.useAlternateReport {
		// The timer of the pad says whether the animation is over.
		complete = p.lastSensorTicks >= ps5LEDConnectionComplete
	}
	if !complete {
		return
	}
	_ = p.updateEffects(ps5EffectLEDReset) // a refused light setting is not worth stopping for
	p.ledReset = ledResetComplete
	_ = p.updateEffects(ps5EffectLED | ps5EffectPadLights)
}

func (p *ps5) Tick(now time.Duration, out Sink) {
	if p.bluetooth && now >= p.lastPacket+ps5BluetoothSilence {
		if p.enhancedReports {
			// A report that does nothing, as it has no CRC, pokes the Bluetooth stack.
			var data [78]byte
			data[0] = ps5ReportBluetoothEffects
			data[1] = 0x02 // magic value
			_ = p.io.Send(data[:])
		} else {
			// Not even an invalid effects report may be sent, or the pad changes its mode: the pad is taken for gone.
			out.Lost()
		}
		p.lastPacket = now
	}
}
