package gamepadhid

import (
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gamepaddb"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/padstd"
)

// This file is translated to Go and changed from src/joystick/hidapi/SDL_hidapi_xboxone.c of SDL 3 (zlib license,
// third_party/sdl/LICENSE): the driver of the Xbox One and Series pads over Bluetooth. Left out: the protocol of USB (GIP), which
// the system's drivers serve (XInput on Windows, xpad on Linux), the pads of other makers, the colour light of the Elite Series 2
// (it has none over Bluetooth) and the report descriptor that Linux makes for a pad (the reports are read at the offsets that
// SDL's code for Bluetooth has).

const (
	xboxReportState   = 0x01
	xboxReportGuide   = 0x02
	xboxReportBattery = 0x04

	// xboxRumbleBusy is how long after a rumble report the next one waits over Bluetooth (10 ms over USB).
	xboxRumbleBusy = 50 * time.Millisecond
)

// The Bluetooth pads of Microsoft (the product numbers).
const (
	productXboxOneAdaptiveBluetooth = 0x0b0c
	productXboxOneAdaptiveBLE       = 0x0b21
	productXboxOneSRev1Bluetooth    = 0x02e0
	productXboxOneSRev2Bluetooth    = 0x02fd
	productXboxOneSRev2BLE          = 0x0b20
	productXboxElite2Bluetooth      = 0x0b05
	productXboxElite2BLE            = 0x0b22
	productXboxSeriesXBLE           = 0x0b13
)

// isBluetoothXboxOne says whether the product is one of the Xbox One and Series pads of Microsoft that speak over Bluetooth.
func isBluetoothXboxOne(vendor, product uint16) bool {
	if vendor != vendorMicrosoft {
		return false
	}
	switch product {
	case productXboxOneAdaptiveBluetooth, productXboxOneAdaptiveBLE, productXboxOneSRev1Bluetooth, productXboxOneSRev2Bluetooth,
		productXboxOneSRev2BLE, productXboxElite2Bluetooth, productXboxElite2BLE, productXboxSeriesXBLE:
		return true
	}
	return false
}

type xboxOne struct {
	dev DeviceInfo
	io  IO

	hasGuidePacket bool
	hasPaddles     bool
	hasShare       bool

	low, high, left, right uint8
	rumblePending          bool
	rumbleBusy             bool
	rumbleTime             time.Duration
	power                  batteryTracker
	snap                   padstd.Snapshot
}

func newXboxOne(info DeviceInfo) Driver {
	return &xboxOne{
		dev: info,
		// The Elite Series 2 has paddles; the Series X has a Share button, as most pads that came after the Xbox One S do.
		hasPaddles: info.Product == productXboxElite2Bluetooth || info.Product == productXboxElite2BLE,
		hasShare:   info.Product == productXboxSeriesXBLE,
	}
}

func (x *xboxOne) Start(io IO) (Caps, error) {
	x.io = io
	// A pad over Bluetooth does not announce itself and needs no start sequence.
	buttons := buttonBit(gamepaddb.South, gamepaddb.East, gamepaddb.West, gamepaddb.North, gamepaddb.Back, gamepaddb.Guide,
		gamepaddb.Start, gamepaddb.LeftStick, gamepaddb.RightStick, gamepaddb.LeftShoulder, gamepaddb.RightShoulder,
		gamepaddb.DPadUp, gamepaddb.DPadDown, gamepaddb.DPadLeft, gamepaddb.DPadRight)
	if x.hasShare {
		buttons |= buttonBit(gamepaddb.Misc1)
	}
	if x.hasPaddles {
		buttons |= buttonBit(gamepaddb.Paddle1, gamepaddb.Paddle2, gamepaddb.Paddle3, gamepaddb.Paddle4)
	}
	return Caps{Kind: gamepaddb.KindXboxOne, Rumble: true, RumbleTriggers: true, Buttons: buttons}, nil
}

func (x *xboxOne) Close() {}

// updateRumble sends the rumble that is waiting, once the pad has had time for the one before.
func (x *xboxOne) updateRumble(now time.Duration) error {
	if x.rumbleBusy && now >= x.rumbleTime+xboxRumbleBusy {
		x.rumbleBusy = false
	}
	if !x.rumblePending || x.rumbleBusy {
		return nil
	}
	// It is not pending any more, even when the write fails.
	x.rumblePending = false
	packet := []byte{0x03, 0x0F, x.left, x.right, x.low, x.high, 0xFF, 0x00, 0xEB}
	x.rumbleBusy, x.rumbleTime = true, now
	return x.io.Send(packet)
}

// magnitude turns a strength (0 to 65535) into the magnitude of the pad, 0 to 100.
func magnitude(v uint16) uint8 { return uint8(v / 655) }

func (x *xboxOne) Rumble(low, high uint16) error {
	x.low, x.high = magnitude(low), magnitude(high)
	x.rumblePending = true
	return x.updateRumble(x.io.Now())
}

func (x *xboxOne) RumbleTriggers(left, right uint16) error {
	x.left, x.right = magnitude(left), magnitude(right)
	x.rumblePending = true
	return x.updateRumble(x.io.Now())
}

func (x *xboxOne) SetLED(r, g, b uint8) error {
	return errors.New(platform.TagGamepadLEDUnavailable, "the pad has no light that can be set over Bluetooth", "vendor", x.dev.Vendor, "product", x.dev.Product)
}

func (x *xboxOne) SetSensors(on bool) error {
	return errors.New(platform.TagGamepadSensorsUnavailable, "the pad has no motion sensors", "vendor", x.dev.Vendor, "product", x.dev.Product)
}

func (x *xboxOne) Tick(now time.Duration, out Sink) { _ = x.updateRumble(now) } // a rumble that is not sent is lost; the next one follows

func (x *xboxOne) Report(now time.Duration, data []byte, out Sink) {
	if len(data) == 0 {
		return
	}
	switch data[0] {
	case xboxReportState:
		if len(data) >= 16 {
			x.handleState(now, data, out)
		}
	case xboxReportGuide:
		if len(data) >= 2 {
			x.hasGuidePacket = true
			x.snap.Buttons[gamepaddb.Guide] = data[1]&0x01 != 0
			out.State(now, &x.snap)
		}
	case xboxReportBattery:
		if len(data) >= 2 {
			x.handleBattery(data[1], out)
		}
	}
}

// handleBattery reads the battery report: the low two bits are the level (as Microsoft's GameInput gives it in steps of a
// quarter), and bits 2 and 3 say where the power comes from (0 is USB, which charges).
func (x *xboxOne) handleBattery(flags byte, out Sink) {
	onUSB := (flags&0x0C)>>2 == 0
	percent := [4]int{10, 40, 70, 100}[flags&0x03]
	state := platform.PadOnBattery
	if onUSB {
		state = platform.PadCharging
	}
	x.power.send(out, state, percent)
}

func (x *xboxOne) handleState(now time.Duration, data []byte, out Sink) {
	size := len(data)
	var d [64]byte
	copy(d[:], data)
	s := &x.snap
	if size == 16 {
		// The original Xbox One S has a packet of 16 bytes, and its Guide button in a report of its own.
		b := d[14]
		s.Buttons[gamepaddb.South] = b&0x01 != 0
		s.Buttons[gamepaddb.East] = b&0x02 != 0
		s.Buttons[gamepaddb.West] = b&0x04 != 0
		s.Buttons[gamepaddb.North] = b&0x08 != 0
		s.Buttons[gamepaddb.LeftShoulder] = b&0x10 != 0
		s.Buttons[gamepaddb.RightShoulder] = b&0x20 != 0
		s.Buttons[gamepaddb.Back] = b&0x40 != 0
		s.Buttons[gamepaddb.Start] = b&0x80 != 0
		s.Buttons[gamepaddb.LeftStick] = d[15]&0x01 != 0
		s.Buttons[gamepaddb.RightStick] = d[15]&0x02 != 0
	} else {
		// Longer packets: the S with newer firmware has 17 bytes (Back and Guide in byte 15), the Series X has the Share button in
		// byte 17, and the Elite Series 2 has 19, 33 or 55 with its paddles.
		b := d[14]
		s.Buttons[gamepaddb.South] = b&0x01 != 0
		s.Buttons[gamepaddb.East] = b&0x02 != 0
		s.Buttons[gamepaddb.West] = b&0x08 != 0
		s.Buttons[gamepaddb.North] = b&0x10 != 0
		s.Buttons[gamepaddb.LeftShoulder] = b&0x40 != 0
		s.Buttons[gamepaddb.RightShoulder] = b&0x80 != 0
		b = d[15]
		if !x.hasGuidePacket {
			s.Buttons[gamepaddb.Guide] = b&0x10 != 0
		}
		s.Buttons[gamepaddb.Start] = b&0x08 != 0
		s.Buttons[gamepaddb.LeftStick] = b&0x20 != 0
		s.Buttons[gamepaddb.RightStick] = b&0x40 != 0
		if x.hasShare {
			s.Buttons[gamepaddb.Back] = b&0x04 != 0
			s.Buttons[gamepaddb.Misc1] = d[16]&0x01 != 0
		} else {
			s.Buttons[gamepaddb.Back] = b&0x04 != 0 || d[16]&0x01 != 0
		}
		if x.hasPaddles && (size == 20 || size == 39 || size == 55) {
			var index int
			var mapped bool
			switch size {
			case 55:
				// The first firmware of the Elite Series 2.
				index, mapped = 33, d[35] != 0
			case 39:
				index, mapped = 17, d[19] != 0
			default:
				// The firmware 5.13 and later.
				index, mapped = 19, d[17] != 0
			}
			p := d[index]
			if mapped {
				// The paddles are used for other controls: they are not passed on.
				p = 0
			}
			// The paddles: P1 is 0x01 (the B side), P2 0x02 (Y), P3 0x04 (A) and P4 0x08 (X). The layout numbers them as SDL's
			// mapping of the pad does: paddle1 is P1, paddle2 is P3, paddle3 is P2 and paddle4 is P4.
			s.Buttons[gamepaddb.Paddle1] = p&0x01 != 0
			s.Buttons[gamepaddb.Paddle3] = p&0x02 != 0
			s.Buttons[gamepaddb.Paddle2] = p&0x04 != 0
			s.Buttons[gamepaddb.Paddle4] = p&0x08 != 0
		}
	}

	if hat := d[13]; hat >= 1 && hat <= 8 {
		setHat(s, hat-1)
	} else {
		setHat(s, 255)
	}

	s.Axes[gamepaddb.LeftTrigger] = xboxTrigger(ule16(d[9:]))
	s.Axes[gamepaddb.RightTrigger] = xboxTrigger(ule16(d[11:]))
	s.Axes[gamepaddb.LeftX] = int16(int(ule16(d[1:])) - 0x8000)
	s.Axes[gamepaddb.LeftY] = int16(int(ule16(d[3:])) - 0x8000)
	s.Axes[gamepaddb.RightX] = int16(int(ule16(d[5:])) - 0x8000)
	s.Axes[gamepaddb.RightY] = int16(int(ule16(d[7:])) - 0x8000)
	out.State(now, s)
}

// xboxTrigger turns a trigger of 10 bits into the range of a trigger axis: the number is spread over the full range of an axis,
// as SDL does, and the top value is the maximum.
func xboxTrigger(raw uint16) int16 {
	v := int(raw)*64 - 32768
	if v == 32704 {
		v = 32767
	}
	// The axis of a trigger is 0 to 32767: the range of the raw axis is spread over it.
	return int16((v + 32768) * 32767 / 65535)
}
