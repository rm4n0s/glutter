package gamepadhid

import (
	"math"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gamepaddb"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/padstd"
)

// This file is translated to Go and changed from src/joystick/hidapi/SDL_hidapi_switch.c of SDL 3 (zlib license,
// third_party/sdl/LICENSE): the driver of the Nintendo Switch Pro controller, over Bluetooth and USB. Left out: the Joy-Cons,
// the Nintendo Online controllers, the pads that only read and cannot be written to, and SDL's hints (the enhanced reports are
// always on). A command sent after the pad is started is not waited for: SDL waits for the reply of each subcommand and sends
// it again up to five times, but here the reply would meet the goroutine that reads the pad.

const (
	switchInputSubcommandReply  = 0x21
	switchInputFullState        = 0x30
	switchInputFullAndMCUState  = 0x31
	switchInputSimpleState      = 0x3F
	switchInputCommandAck       = 0x81
	switchOutputRumbleAndSubcmd = 0x01
	switchOutputRumble          = 0x10
	switchOutputProprietary     = 0x80

	switchSubcmdRequestDeviceInfo = 0x02
	switchSubcmdSetInputReportMod = 0x03
	switchSubcmdSPIFlashRead      = 0x10
	switchSubcmdSetPlayerLights   = 0x30
	switchSubcmdSetHomeLight      = 0x38
	switchSubcmdEnableIMU         = 0x40
	switchSubcmdEnableVibration   = 0x48

	switchProprietaryStatus    = 0x01
	switchProprietaryHandshake = 0x02
	switchProprietaryHighSpeed = 0x03
	switchProprietaryForceUSB  = 0x04

	// switchPacketLength is the size of an output report over Bluetooth, and switchUSBPacketLength over USB.
	switchPacketLength    = 49
	switchUSBPacketLength = 64

	switchStickFactoryStart = 0x603D
	switchStickFactoryLen   = 0x604E - 0x603D + 1
	switchStickUserStart    = 0x8010
	switchStickUserLen      = 0x8025 - 0x8010 + 1
	switchIMUScaleStart     = 0x6020
	switchIMUScaleLen       = 0x6037 - 0x6020 + 1
	switchIMUUserStart      = 0x8026
	switchIMUUserLen        = 0x8039 - 0x8026 + 1

	switchGyroScale      = 14.2842
	switchAccelScale     = 4096.0
	switchGyroScaleMult  = 936.0
	switchAccelScaleMult = 4.0

	// switchRumbleWriteGap is how often rumble may be written (sooner can turn a pad off over Bluetooth), and
	// switchRumbleRefresh how often a long rumble is written again to keep the motors running.
	switchRumbleWriteGap = 30 * time.Millisecond
	switchRumbleRefresh  = 50 * time.Millisecond

	switchControllerTypePro    = 3
	switchControllerTypeLicPro = 6
)

type stickCal struct{ center, min, max int16 }

type stickExtents struct{ min, max int16 }

// The sensors of the pad give three readings in every full report, in these units.
type switchIMUScale struct {
	accel, gyro [3]float32
	gyroOffset  [3]int16
}

type switchPro struct {
	dev DeviceInfo
	io  IO

	bluetooth bool
	// controllerType is what the pad says it is (3 is the Pro controller).
	controllerType uint8
	initialMode    uint8
	currentMode    uint8
	counter        uint8
	rumblePacket   [10]byte // a common output packet: the type, the number and the two rumble blocks of four bytes
	ready          bool     // Start is over: commands are queued instead of written now

	rumbleActive      bool
	rumbleSent        time.Duration
	rumblePending     bool
	rumbleZeroPending bool
	rumbleValue       uint32

	reportSensors bool
	hasSensorData bool
	lastInput     time.Duration
	lastIMUReset  time.Duration
	imuSampleTime time.Duration
	imuSamples    uint32
	imuInterval   time.Duration
	timestamp     time.Duration
	power         batteryTracker
	snap          padstd.Snapshot
	cal           [2][2]stickCal
	extents       [2][2]stickExtents
	simpleExtents [2][2]stickExtents
	imu           switchIMUScale
}

func newSwitchPro(info DeviceInfo) Driver {
	return &switchPro{dev: info, bluetooth: info.Bus == BusBluetooth, imuInterval: 5 * time.Millisecond}
}

// switchNeutralRumble is the rumble block of a pad at rest (the standard neutral value).
var switchNeutralRumble = [4]byte{0x00, 0x01, 0x40, 0x40}

func (p *switchPro) setNeutralRumble() {
	copy(p.rumblePacket[2:6], switchNeutralRumble[:])
	copy(p.rumblePacket[6:10], switchNeutralRumble[:])
}

// writePacket writes a report: during the start at once, afterwards through the queue. A report shorter than the pad's output
// report is padded with zeros.
func (p *switchPro) writePacket(pkt []byte) error {
	size := switchUSBPacketLength
	if p.bluetooth {
		size = switchPacketLength
	}
	if len(pkt) > switchPacketLength {
		return errors.New(tagHidReportTooLong, "an output report is longer than the pad takes", "length", len(pkt))
	}
	buf := make([]byte, max(size, len(pkt)))
	copy(buf, pkt)
	buf = buf[:size]
	if !p.ready {
		_, err := p.io.Write(buf)
		return err
	}
	// Each report is sent as it is: reports are never replaced by others.
	return p.io.SendMerging(buf, func([]byte) bool { return false })
}

// subcommand makes the output report of a subcommand, which carries the rumble state as well.
func (p *switchPro) subcommand(id byte, data []byte) []byte {
	pkt := make([]byte, switchPacketLength)
	pkt[0] = switchOutputRumbleAndSubcmd
	pkt[1] = p.counter
	copy(pkt[2:10], p.rumblePacket[2:10])
	pkt[10] = id
	copy(pkt[11:], data)
	p.counter = (p.counter + 1) & 0xF
	return pkt
}

// subcommandReplyFor says whether an input report is the reply to the subcommand id (and, for an SPI read, to the read that
// request asks for), and returns the reply without the report ID: the state of the controller (12 bytes), the acknowledgement,
// the subcommand ID and the data.
func subcommandReplyFor(buf []byte, id byte, request []byte) []byte {
	if len(buf) < 15 || buf[0] != switchInputSubcommandReply {
		return nil
	}
	reply := buf[1:]
	if reply[13] != id || reply[12]&0x80 == 0 {
		return nil
	}
	if id == switchSubcmdSPIFlashRead {
		if len(request) < 5 || len(reply) < 14+5 {
			return nil
		}
		if ule32(reply[14:]) != ule32(request) || reply[18] < request[4] {
			// A reply for another read.
			return nil
		}
	}
	return reply
}

// readSubcommandReply waits for the reply to a subcommand, up to 100 ms (the average answer takes 30). A reply for another
// subcommand, or another SPI read, is skipped.
func (p *switchPro) readSubcommandReply(id byte, request []byte) []byte {
	deadline := p.io.Now() + 100*time.Millisecond
	buf := make([]byte, switchUSBPacketLength)
	for {
		n, err := p.io.Read(buf, 5*time.Millisecond)
		if err != nil {
			return nil
		}
		if n > 0 {
			if reply := subcommandReplyFor(buf[:n], id, request); reply != nil {
				return append([]byte(nil), reply...)
			}
		}
		if p.io.Now() >= deadline {
			return nil
		}
	}
}

// writeSubcommand sends a subcommand and waits for its reply, trying up to five times.
func (p *switchPro) writeSubcommand(id byte, data []byte) []byte {
	for try := 0; try < 5; try++ {
		if err := p.writePacket(p.subcommand(id, data)); err != nil {
			continue
		}
		if reply := p.readSubcommandReply(id, data); reply != nil {
			return reply
		}
	}
	return nil
}

// readProprietaryReply waits for the acknowledgement of a proprietary command, up to 100 ms.
func (p *switchPro) readProprietaryReply(cmd byte) []byte {
	deadline := p.io.Now() + 100*time.Millisecond
	buf := make([]byte, switchUSBPacketLength)
	for {
		n, err := p.io.Read(buf, 5*time.Millisecond)
		if err != nil {
			return nil
		}
		if n >= 2 && buf[0] == switchInputCommandAck && buf[1] == cmd {
			return append([]byte(nil), buf[:n]...)
		}
		if p.io.Now() >= deadline {
			return nil
		}
	}
}

// writeProprietary sends a command of the USB protocol and, when asked, waits for its acknowledgement (which it returns).
func (p *switchPro) writeProprietary(cmd byte, wait bool) (ok bool, reply []byte) {
	for try := 0; try < 5; try++ {
		pkt := make([]byte, switchPacketLength)
		pkt[0], pkt[1] = switchOutputProprietary, cmd
		if err := p.writePacket(pkt); err != nil {
			continue
		}
		if !wait {
			return true, nil
		}
		if r := p.readProprietaryReply(cmd); r != nil {
			return true, r
		}
	}
	return false, nil
}

func (p *switchPro) readDeviceInfo() bool {
	if p.bluetooth {
		reply := p.writeSubcommand(switchSubcmdRequestDeviceInfo, nil)
		if reply == nil || len(reply) < 14+3 {
			return false
		}
		// Byte 2 of the data is the kind of controller: 1 left Joy-Con, 2 right Joy-Con, 3 Pro.
		p.controllerType = reply[14+2]
		return true
	}
	ok, reply := p.writeProprietary(switchProprietaryStatus, true)
	if !ok || len(reply) < 4 {
		return false
	}
	p.controllerType = reply[3]
	return true
}

// trySetupUSB sends the handshake of the USB protocol. It does not work over Bluetooth, which is how the two are told apart.
func (p *switchPro) trySetupUSB() bool {
	if ok, _ := p.writeProprietary(switchProprietaryHandshake, true); !ok {
		return false
	}
	// Some third party pads do not answer these two, but work.
	p.writeProprietary(switchProprietaryHighSpeed, true)
	p.writeProprietary(switchProprietaryHandshake, true)
	ok, _ := p.writeProprietary(switchProprietaryForceUSB, false)
	return ok
}

func (p *switchPro) Start(io IO) (Caps, error) {
	p.io = io
	p.setNeutralRumble()
	// The pad is asked what it is; one that is not a Pro controller is not for this driver.
	p.readDeviceInfo()
	if p.controllerType == 0 {
		// The info could not be read: a pad that is not fully compliant. The product number says what it is.
		p.controllerType = switchControllerTypePro
	}
	if p.controllerType != switchControllerTypePro && p.controllerType != switchControllerTypeLicPro {
		return Caps{}, errors.New(tagHidDriverStartFailed, "the pad is not a Switch Pro controller", "type", int(p.controllerType),
			"vendor", p.dev.Vendor, "product", p.dev.Product)
	}

	// Guess the input mode the pad is in from a report.
	buf := make([]byte, switchUSBPacketLength)
	if n, _ := io.Read(buf, 16*time.Millisecond); n > 0 {
		switch buf[0] {
		case switchInputFullState, switchInputFullAndMCUState, switchInputSimpleState:
			p.initialMode = buf[0]
		}
	}
	p.currentMode = p.initialMode
	p.setNeutralRumble()

	if !p.bluetooth && !p.trySetupUSB() {
		return Caps{}, errors.New(tagHidDriverStartFailed, "the pad does not answer the handshake of USB", "vendor", p.dev.Vendor, "product", p.dev.Product)
	}
	p.loadStickCalibration()
	p.loadIMUCalibration()
	p.writeSubcommand(switchSubcmdEnableVibration, []byte{1})
	// The enhanced reports are always on: the full state, which has the sensors and the battery.
	p.setInputMode(p.defaultInputMode())
	if !p.bluetooth {
		// ForceUSB gets no acknowledgement.
		p.writeProprietary(switchProprietaryForceUSB, false)
	}
	// The light of the Home button stays as the pad has it; the player lights are off, as no player number is known.
	p.writeSubcommand(switchSubcmdSetPlayerLights, []byte{0})

	p.ready = true
	p.lastInput = io.Now()
	p.lastIMUReset = p.lastInput
	return Caps{
		Name: "Nintendo Switch Pro Controller", Kind: gamepaddb.KindSwitchPro, Rumble: true, LED: true, Sensors: true, Rate: 200,
		Buttons: buttonBit(gamepaddb.South, gamepaddb.East, gamepaddb.West, gamepaddb.North, gamepaddb.Back, gamepaddb.Guide,
			gamepaddb.Start, gamepaddb.LeftStick, gamepaddb.RightStick, gamepaddb.LeftShoulder, gamepaddb.RightShoulder,
			gamepaddb.DPadUp, gamepaddb.DPadDown, gamepaddb.DPadLeft, gamepaddb.DPadRight, gamepaddb.Misc1),
	}, nil
}

// defaultInputMode is the input mode that is wanted when the sensors are not: the full state (the simple state has no battery
// and the sticks are less exact; a wired pad does not report at all in the simple mode).
func (p *switchPro) defaultInputMode() uint8 {
	mode := p.initialMode
	if mode == 0 {
		if p.bluetooth {
			mode = switchInputSimpleState
		} else {
			mode = switchInputFullState
		}
	}
	if mode == switchInputSimpleState {
		mode = switchInputFullState
	}
	return mode
}

func (p *switchPro) setInputMode(mode uint8) {
	if mode == p.currentMode {
		return
	}
	p.currentMode = mode
	p.writeSubcommand(switchSubcmdSetInputReportMod, []byte{mode})
}

func (p *switchPro) loadStickCalibration() {
	var leftCal, rightCal [9]byte
	haveLeft, haveRight := false, false

	// The user's calibration is not readable on all pads, so a failure is ignored.
	var req [5]byte
	putLE32(req[:], switchStickUserStart)
	req[4] = switchStickUserLen
	user := p.writeSubcommand(switchSubcmdSPIFlashRead, req[:])
	// The data of an SPI read starts after the address and the length, 5 bytes into the data of the reply.
	spi := func(r []byte) []byte {
		if len(r) < 14+5 {
			return nil
		}
		return r[14+5:]
	}
	if d := spi(user); len(d) >= 22 {
		if d[0] == 0xB2 && d[1] == 0xA1 {
			haveLeft = true
			copy(leftCal[:], d[2:11])
		}
		if d[11] == 0xB2 && d[12] == 0xA1 {
			haveRight = true
			copy(rightCal[:], d[13:22])
		}
	}
	if !haveLeft || !haveRight {
		putLE32(req[:], switchStickFactoryStart)
		req[4] = switchStickFactoryLen
		if d := spi(p.writeSubcommand(switchSubcmdSPIFlashRead, req[:])); len(d) >= 18 {
			if !haveLeft {
				haveLeft = true
				copy(leftCal[:], d[0:9])
			}
			if !haveRight {
				haveRight = true
				copy(rightCal[:], d[9:18])
			}
		}
	}
	if !haveLeft {
		for i := range leftCal {
			leftCal[i] = 0xFF
		}
	}
	if !haveRight {
		for i := range rightCal {
			rightCal[i] = 0xFF
		}
	}

	// The calibration values are 12 bits each, packed. The fields come in a different order for each stick.
	// Left: x max, y max, x center, y center, x min, y min. Right: x center, y center, x min, y min, x max, y max.
	l, r := &leftCal, &rightCal
	pack := func(lo, mid, hi byte, first bool) int16 {
		if first {
			return int16((int(mid)<<8)&0xF00 | int(lo))
		}
		return int16(int(hi)<<4 | int(mid)>>4)
	}
	p.cal[0][0].max, p.cal[0][1].max = pack(l[0], l[1], 0, true), pack(0, l[1], l[2], false)
	p.cal[0][0].center, p.cal[0][1].center = pack(l[3], l[4], 0, true), pack(0, l[4], l[5], false)
	p.cal[0][0].min, p.cal[0][1].min = pack(l[6], l[7], 0, true), pack(0, l[7], l[8], false)
	p.cal[1][0].center, p.cal[1][1].center = pack(r[0], r[1], 0, true), pack(0, r[1], r[2], false)
	p.cal[1][0].min, p.cal[1][1].min = pack(r[3], r[4], 0, true), pack(0, r[4], r[5], false)
	p.cal[1][0].max, p.cal[1][1].max = pack(r[6], r[7], 0, true), pack(0, r[7], r[8], false)

	for s := range p.cal {
		for a := range p.cal[s] {
			c := &p.cal[s][a]
			// Values that were not set (0xFFF) are replaced.
			if c.center == 0xFFF {
				c.center = 2048
			}
			if c.max == 0xFFF {
				c.max = int16(float32(c.center) * 0.7)
			}
			if c.min == 0xFFF {
				c.min = int16(float32(c.center) * 0.7)
			}
		}
	}
	for s := range p.cal {
		for a := range p.cal[s] {
			p.extents[s][a].min = -int16(float32(p.cal[s][a].min) * 0.7)
			p.extents[s][a].max = int16(float32(p.cal[s][a].max) * 0.7)
			p.simpleExtents[s][a].min = math.MinInt16 / 2
			p.simpleExtents[s][a].max = math.MaxInt16 / 2
		}
	}
}

func putLE32(b []byte, v uint32) {
	b[0], b[1], b[2], b[3] = byte(v), byte(v>>8), byte(v>>16), byte(v>>24)
}

func (p *switchPro) loadIMUCalibration() {
	have := false
	var req [5]byte
	putLE32(req[:], switchIMUScaleStart)
	req[4] = switchIMUScaleLen
	if reply := p.writeSubcommand(switchSubcmdSPIFlashRead, req[:]); len(reply) >= 14+5+24 {
		sc := reply[14+5:]
		i16 := func(o int) int16 { return int16(uint16(sc[o]) | uint16(sc[o+1])<<8) }
		accelRaw := [3]int16{i16(0), i16(2), i16(4)}
		accelCoeff := [3]int16{i16(6), i16(8), i16(10)}
		gyroRaw := [3]int16{i16(12), i16(14), i16(16)}
		gyroCoeff := [3]int16{i16(18), i16(20), i16(22)}

		// A calibration of the user's, when it is set, replaces the factory settings.
		putLE32(req[:], switchIMUUserStart)
		req[4] = switchIMUUserLen
		if ur := p.writeSubcommand(switchSubcmdSPIFlashRead, req[:]); len(ur) >= 14+5+20 {
			u := ur[14+5:]
			if int(u[0])|int(u[1])<<8 == 0xA1B2 {
				ui16 := func(o int) int16 { return int16(uint16(u[o]) | uint16(u[o+1])<<8) }
				accelRaw = [3]int16{ui16(2), ui16(4), ui16(6)}
				gyroRaw = [3]int16{ui16(14), ui16(16), ui16(18)}
			}
		}
		if accelCoeff[0] != accelRaw[0] && accelCoeff[1] != accelRaw[1] && accelCoeff[2] != accelRaw[2] &&
			gyroCoeff[0] != gyroRaw[0] && gyroCoeff[1] != gyroRaw[1] && gyroCoeff[2] != gyroRaw[2] {
			for i := 0; i < 3; i++ {
				p.imu.gyroOffset[i] = gyroRaw[i]
				p.imu.accel[i] = switchAccelScaleMult / (float32(accelCoeff[i]) - float32(accelRaw[i])) * standardGravity
				p.imu.gyro[i] = switchGyroScaleMult / (float32(gyroCoeff[i]) - float32(gyroRaw[i])) * math.Pi / 180
			}
			have = true
		}
	}
	if !have {
		// The default values.
		accel := float32(standardGravity / switchAccelScale)
		gyro := float32(math.Pi / 180 / switchGyroScale)
		for i := 0; i < 3; i++ {
			p.imu.accel[i], p.imu.gyro[i], p.imu.gyroOffset[i] = accel, gyro, 0
		}
	}
}

// applyStickCalibration spreads a stick of 12 bits over the range of an axis, with the extents that the pad has been seen to
// reach: they grow when the stick goes further than the calibration says.
func (p *switchPro) applyStickCalibration(stick, axis int, raw int16) int16 {
	raw -= p.cal[stick][axis].center
	e := &p.extents[stick][axis]
	if raw >= 0 {
		if raw > e.max {
			e.max = raw
		}
		return remapToInt16(float32(raw), 0, float32(e.max), 0, math.MaxInt16)
	}
	if raw < e.min {
		e.min = raw
	}
	return remapToInt16(float32(raw), float32(e.min), 0, math.MinInt16, 0)
}

// applySimpleStickCalibration does the same for the sticks of the simple report, whose centre is 0x8000.
func (p *switchPro) applySimpleStickCalibration(stick, axis int, raw int16) int16 {
	raw = int16(uint16(raw) - 0x8000)
	e := &p.simpleExtents[stick][axis]
	if raw >= 0 {
		if raw > e.max {
			e.max = raw
		}
		return remapToInt16(float32(raw), 0, float32(e.max), 0, math.MaxInt16)
	}
	if raw < e.min {
		e.min = raw
	}
	return remapToInt16(float32(raw), float32(e.min), 0, math.MinInt16, 0)
}

func remapToInt16(val, inMin, inMax, outMin, outMax float32) int16 {
	if inMax == inMin {
		return 0
	}
	return int16(remap(val, inMin, inMax, outMin, outMax))
}

// The amplitudes of rumble, encoded as the pad takes them: a table of 101 steps for each band, as the reverse engineering of the
// pad's rumble data found them (https://github.com/dekuNukem/Nintendo_Switch_Reverse_Engineering/blob/master/rumble_data_table.md).
var switchHighFreqAmp = [101][2]uint16{
	{0, 0x0}, {514, 0x2}, {775, 0x4}, {921, 0x6}, {1096, 0x8}, {1303, 0x0a}, {1550, 0x0c}, {1843, 0x0e}, {2192, 0x10}, {2606, 0x12},
	{3100, 0x14}, {3686, 0x16}, {4383, 0x18}, {5213, 0x1a}, {6199, 0x1c}, {7372, 0x1e}, {7698, 0x20}, {8039, 0x22}, {8395, 0x24},
	{8767, 0x26}, {9155, 0x28}, {9560, 0x2a}, {9984, 0x2c}, {10426, 0x2e}, {10887, 0x30}, {11369, 0x32}, {11873, 0x34},
	{12398, 0x36}, {12947, 0x38}, {13520, 0x3a}, {14119, 0x3c}, {14744, 0x3e}, {15067, 0x40}, {15397, 0x42}, {15734, 0x44},
	{16079, 0x46}, {16431, 0x48}, {16790, 0x4a}, {17158, 0x4c}, {17534, 0x4e}, {17918, 0x50}, {18310, 0x52}, {18711, 0x54},
	{19121, 0x56}, {19540, 0x58}, {19967, 0x5a}, {20405, 0x5c}, {20851, 0x5e}, {21308, 0x60}, {21775, 0x62}, {22251, 0x64},
	{22739, 0x66}, {23236, 0x68}, {23745, 0x6a}, {24265, 0x6c}, {24797, 0x6e}, {25340, 0x70}, {25894, 0x72}, {26462, 0x74},
	{27041, 0x76}, {27633, 0x78}, {28238, 0x7a}, {28856, 0x7c}, {29488, 0x7e}, {30134, 0x80}, {30794, 0x82}, {31468, 0x84},
	{32157, 0x86}, {32861, 0x88}, {33581, 0x8a}, {34316, 0x8c}, {35068, 0x8e}, {35836, 0x90}, {36620, 0x92}, {37422, 0x94},
	{38242, 0x96}, {39079, 0x98}, {39935, 0x9a}, {40809, 0x9c}, {41703, 0x9e}, {42616, 0xa0}, {43549, 0xa2}, {44503, 0xa4},
	{45477, 0xa6}, {46473, 0xa8}, {47491, 0xaa}, {48531, 0xac}, {49593, 0xae}, {50679, 0xb0}, {51789, 0xb2}, {52923, 0xb4},
	{54082, 0xb6}, {55266, 0xb8}, {56476, 0xba}, {57713, 0xbc}, {58977, 0xbe}, {60268, 0xc0}, {61588, 0xc2}, {62936, 0xc4},
	{64315, 0xc6}, {65535, 0xc8},
}

var switchLowFreqAmp = [101][2]uint16{
	{0, 0x0040}, {514, 0x8040}, {775, 0x0041}, {921, 0x8041}, {1096, 0x0042}, {1303, 0x8042}, {1550, 0x0043}, {1843, 0x8043},
	{2192, 0x0044}, {2606, 0x8044}, {3100, 0x0045}, {3686, 0x8045}, {4383, 0x0046}, {5213, 0x8046}, {6199, 0x0047},
	{7372, 0x8047}, {7698, 0x0048}, {8039, 0x8048}, {8395, 0x0049}, {8767, 0x8049}, {9155, 0x004a}, {9560, 0x804a},
	{9984, 0x004b}, {10426, 0x804b}, {10887, 0x004c}, {11369, 0x804c}, {11873, 0x004d}, {12398, 0x804d}, {12947, 0x004e},
	{13520, 0x804e}, {14119, 0x004f}, {14744, 0x804f}, {15067, 0x0050}, {15397, 0x8050}, {15734, 0x0051}, {16079, 0x8051},
	{16431, 0x0052}, {16790, 0x8052}, {17158, 0x0053}, {17534, 0x8053}, {17918, 0x0054}, {18310, 0x8054}, {18711, 0x0055},
	{19121, 0x8055}, {19540, 0x0056}, {19967, 0x8056}, {20405, 0x0057}, {20851, 0x8057}, {21308, 0x0058}, {21775, 0x8058},
	{22251, 0x0059}, {22739, 0x8059}, {23236, 0x005a}, {23745, 0x805a}, {24265, 0x005b}, {24797, 0x805b}, {25340, 0x005c},
	{25894, 0x805c}, {26462, 0x005d}, {27041, 0x805d}, {27633, 0x005e}, {28238, 0x805e}, {28856, 0x005f}, {29488, 0x805f},
	{30134, 0x0060}, {30794, 0x8060}, {31468, 0x0061}, {32157, 0x8061}, {32861, 0x0062}, {33581, 0x8062}, {34316, 0x0063},
	{35068, 0x8063}, {35836, 0x0064}, {36620, 0x8064}, {37422, 0x0065}, {38242, 0x8065}, {39079, 0x0066}, {39935, 0x8066},
	{40809, 0x0067}, {41703, 0x8067}, {42616, 0x0068}, {43549, 0x8068}, {44503, 0x0069}, {45477, 0x8069}, {46473, 0x006a},
	{47491, 0x806a}, {48531, 0x006b}, {49593, 0x806b}, {50679, 0x006c}, {51789, 0x806c}, {52923, 0x006d}, {54082, 0x806d},
	{55266, 0x006e}, {56476, 0x806e}, {57713, 0x006f}, {58977, 0x806f}, {60268, 0x0070}, {61588, 0x8070}, {62936, 0x0071},
	{64315, 0x8071}, {65535, 0x0072},
}

func encodeRumbleHighAmplitude(amplitude uint16) uint8 {
	for _, e := range switchHighFreqAmp {
		if amplitude <= e[0] {
			return uint8(e[1])
		}
	}
	return uint8(switchHighFreqAmp[100][1])
}

func encodeRumbleLowAmplitude(amplitude uint16) uint16 {
	for _, e := range switchLowFreqAmp {
		if amplitude <= e[0] {
			return e[1]
		}
	}
	return switchLowFreqAmp[100][1]
}

// encodeRumble writes a rumble block of four bytes: the high band's frequency (9 bits) and amplitude, the low band's frequency and
// amplitude (9 bits), which take a bit from the neighbouring byte. Without any strength the block is the neutral one.
func (p *switchPro) encodeRumble(b []byte, highFreq uint16, highAmp, lowFreq uint8, lowAmp uint16) {
	if highAmp > 0 || lowAmp > 0 {
		b[0] = byte(highFreq & 0xFF)
		b[1] = highAmp | byte(highFreq>>8&0x01)
		b[2] = lowFreq | byte(lowAmp>>8&0x80)
		b[3] = byte(lowAmp & 0xFF)
		return
	}
	copy(b, switchNeutralRumble[:])
}

// writeRumble writes the rumble packet, which keeps the state of the motors for the subcommands that follow.
func (p *switchPro) writeRumble() error {
	p.rumblePacket[0] = switchOutputRumble
	p.rumblePacket[1] = p.counter
	p.counter = (p.counter + 1) & 0xF
	// The rumble is written again soon.
	p.rumbleSent = p.io.Now()
	pkt := make([]byte, 10)
	copy(pkt, p.rumblePacket[:])
	return p.writePacket(pkt)
}

func (p *switchPro) actuallyRumble(low, high uint16) error {
	// The values were found by experiment: tested pads take any strength as a switch for a constant rumble.
	const highFreq, lowFreq = 0x0074, 0x3D
	highAmp := encodeRumbleHighAmplitude(high)
	lowAmp := encodeRumbleLowAmplitude(low)
	if low != 0 || high != 0 {
		p.encodeRumble(p.rumblePacket[2:6], highFreq, highAmp, lowFreq, lowAmp)
		p.encodeRumble(p.rumblePacket[6:10], highFreq, highAmp, lowFreq, lowAmp)
	} else {
		p.setNeutralRumble()
	}
	p.rumbleActive = low != 0 || high != 0
	return p.writeRumble()
}

func (p *switchPro) sendPendingRumble() error {
	now := p.io.Now()
	if now < p.rumbleSent+switchRumbleWriteGap {
		return nil
	}
	if p.rumblePending {
		low, high := uint16(p.rumbleValue>>16), uint16(p.rumbleValue)
		p.rumblePending, p.rumbleValue = false, 0
		return p.actuallyRumble(low, high)
	}
	if p.rumbleZeroPending {
		p.rumbleZeroPending = false
		return p.actuallyRumble(0, 0)
	}
	return nil
}

func (p *switchPro) Rumble(low, high uint16) error {
	if p.rumblePending {
		if err := p.sendPendingRumble(); err != nil {
			return err
		}
	}
	if p.io.Now() < p.rumbleSent+switchRumbleWriteGap {
		if low != 0 || high != 0 {
			v := uint32(low)<<16 | uint32(high)
			// The strongest rumble of the interval is the one that is sent.
			if v > p.rumbleValue {
				p.rumbleValue = v
			}
			p.rumblePending, p.rumbleZeroPending = true, false
		} else {
			// When the rumble is over it is turned off.
			p.rumbleZeroPending = true
		}
		return nil
	}
	return p.actuallyRumble(low, high)
}

func (p *switchPro) RumbleTriggers(left, right uint16) error {
	return errors.New(platform.TagGamepadRumbleUnavailable, "the triggers of the pad have no motors", "vendor", p.dev.Vendor, "product", p.dev.Product)
}

// SetLED sets the light of the Home button, which has one level of brightness, from the strongest of the three colours.
func (p *switchPro) SetLED(r, g, b uint8) error {
	value := int(float32(max(r, g, b)) / 255 * 100) // the colours come from 0 to 255 and the brightness is from 0 to 100
	return p.setHomeLED(uint8(value))
}

func (p *switchPro) setHomeLED(brightness uint8) error {
	var intensity uint8
	if brightness > 0 {
		if brightness < 65 {
			intensity = (brightness + 5) / 10
		} else {
			intensity = uint8(math.Ceil(0xF * math.Pow(float64(brightness)/100, 2.13)))
		}
	}
	data := []byte{
		0x01,                   // no mini cycles besides the first, a cycle of 8 ms
		(intensity & 0xF) << 4, // the intensity the light starts at, and no cycles: it stays at it
		(intensity & 0xF) << 4, // the intensity of the first cycle, and 0 for the second
		0x00,                   // an 8 ms fade to the first cycle, and 8 ms for it
	}
	return p.writePacket(p.subcommand(switchSubcmdSetHomeLight, data))
}

// desiredInputMode is the input mode that is wanted now: the sensors need the full state.
func (p *switchPro) desiredInputMode() uint8 {
	if p.reportSensors {
		if p.initialMode == 0 || p.initialMode == switchInputSimpleState {
			return switchInputFullState
		}
		return p.initialMode
	}
	return p.defaultInputMode()
}

func (p *switchPro) SetSensors(on bool) error {
	p.reportSensors = on
	p.imuSamples = 0
	p.imuSampleTime = p.io.Now()
	p.setInputModeQueued(p.desiredInputMode())
	v := byte(0)
	if on {
		v = 1
	}
	return p.writePacket(p.subcommand(switchSubcmdEnableIMU, []byte{v}))
}

// setInputModeQueued asks for another input mode after the start: the command is queued and not waited for.
func (p *switchPro) setInputModeQueued(mode uint8) {
	if mode == p.currentMode {
		return
	}
	p.currentMode = mode
	_ = p.writePacket(p.subcommand(switchSubcmdSetInputReportMod, []byte{mode})) // a mode that is not set shows in the reports
}

func (p *switchPro) Close() {
	if !p.ready {
		return
	}
	// The simple input mode is put back for the other programs, over Bluetooth: over USB a pad in it does not report.
	if p.bluetooth && (p.initialMode == 0 || p.initialMode == switchInputSimpleState) {
		p.ready = false
		_ = p.writePacket(p.subcommand(switchSubcmdSetInputReportMod, []byte{switchInputSimpleState})) // the pad is going away
	}
}

func (p *switchPro) Report(now time.Duration, data []byte, out Sink) {
	if len(data) == 0 {
		return
	}
	p.lastInput = now
	if data[0] == switchInputSubcommandReply {
		return
	}
	p.currentMode = data[0]
	switch data[0] {
	case switchInputSimpleState:
		p.handleSimple(now, data[1:], out)
	case switchInputFullState, switchInputFullAndMCUState:
		p.handleFull(now, data[1:], out)
	}
}

func (p *switchPro) handleSimple(now time.Duration, body []byte, out Sink) {
	var pk [11]byte
	copy(pk[:], body)
	s := &p.snap
	b0 := pk[0]
	s.Buttons[gamepaddb.South] = b0&0x01 != 0
	s.Buttons[gamepaddb.East] = b0&0x02 != 0
	s.Buttons[gamepaddb.West] = b0&0x04 != 0
	s.Buttons[gamepaddb.North] = b0&0x08 != 0
	s.Buttons[gamepaddb.LeftShoulder] = b0&0x10 != 0
	s.Buttons[gamepaddb.RightShoulder] = b0&0x20 != 0
	b1 := pk[1]
	s.Buttons[gamepaddb.Back] = b1&0x01 != 0
	s.Buttons[gamepaddb.Start] = b1&0x02 != 0
	s.Buttons[gamepaddb.LeftStick] = b1&0x04 != 0
	s.Buttons[gamepaddb.RightStick] = b1&0x08 != 0
	s.Buttons[gamepaddb.Guide] = b1&0x10 != 0
	s.Buttons[gamepaddb.Misc1] = b1&0x20 != 0
	setHat(s, pk[2])
	s.Axes[gamepaddb.LeftTrigger] = digitalTrigger(b0&0x40 != 0)
	s.Axes[gamepaddb.RightTrigger] = digitalTrigger(b0&0x80 != 0 || b1&0x80 != 0)
	s.Axes[gamepaddb.LeftX] = p.applySimpleStickCalibration(0, 0, le16(pk[3:]))
	s.Axes[gamepaddb.LeftY] = p.applySimpleStickCalibration(0, 1, le16(pk[5:]))
	s.Axes[gamepaddb.RightX] = p.applySimpleStickCalibration(1, 0, le16(pk[7:]))
	s.Axes[gamepaddb.RightY] = p.applySimpleStickCalibration(1, 1, le16(pk[9:]))
	out.State(now, s)
}

// digitalTrigger is the axis of a trigger that is a button: all the way down or up.
func digitalTrigger(down bool) int16 {
	if down {
		return gamepaddb.AxisMax
	}
	return 0
}

// switchStateSize is the size of the packet of the full state: the state of the controller and three readings of the sensors.
const switchStateSize = 12 + 3*12

func (p *switchPro) handleFull(now time.Duration, body []byte, out Sink) {
	var pk [switchStateSize]byte
	copy(pk[:], body)
	s := &p.snap
	b0, b1, b2 := pk[2], pk[3], pk[4]
	s.Buttons[gamepaddb.South] = b0&0x04 != 0
	s.Buttons[gamepaddb.East] = b0&0x08 != 0
	s.Buttons[gamepaddb.West] = b0&0x01 != 0
	s.Buttons[gamepaddb.North] = b0&0x02 != 0
	s.Buttons[gamepaddb.RightShoulder] = b0&0x40 != 0
	s.Buttons[gamepaddb.Back] = b1&0x01 != 0
	s.Buttons[gamepaddb.Start] = b1&0x02 != 0
	s.Buttons[gamepaddb.RightStick] = b1&0x04 != 0
	s.Buttons[gamepaddb.LeftStick] = b1&0x08 != 0
	s.Buttons[gamepaddb.Guide] = b1&0x10 != 0
	s.Buttons[gamepaddb.Misc1] = b1&0x20 != 0
	setDPad(s, b2&0x02 != 0, b2&0x04 != 0, b2&0x01 != 0, b2&0x08 != 0)
	s.Buttons[gamepaddb.LeftShoulder] = b2&0x40 != 0
	s.Axes[gamepaddb.RightTrigger] = digitalTrigger(b0&0x80 != 0)
	s.Axes[gamepaddb.LeftTrigger] = digitalTrigger(b2&0x80 != 0)

	left, right := pk[5:8], pk[8:11]
	lx := int16(int(left[0]) | (int(left[1])&0xF)<<8)
	ly := int16(int(left[1]&0xF0)>>4 | int(left[2])<<4)
	rx := int16(int(right[0]) | (int(right[1])&0xF)<<8)
	ry := int16(int(right[1]&0xF0)>>4 | int(right[2])<<4)
	s.Axes[gamepaddb.LeftX] = p.applyStickCalibration(0, 0, lx)
	s.Axes[gamepaddb.LeftY] = ^p.applyStickCalibration(0, 1, ly)
	s.Axes[gamepaddb.RightX] = p.applyStickCalibration(1, 0, rx)
	s.Axes[gamepaddb.RightY] = ^p.applyStickCalibration(1, 1, ry)

	// The high nibble of the battery byte is the battery level, 0 (empty) to 8 (full) in steps of 2 of this nibble; its low bit
	// says charging. The low nibble is the state of the connection.
	charging := pk[1]&0x10 != 0
	level := int(pk[1]&0xE0) >> 4
	state := platform.PadOnBattery
	if charging {
		state = platform.PadCharging
		if level == 8 {
			state = platform.PadCharged
		}
	}
	p.power.send(out, state, int(math.Round(float64(level)/8*100)))

	if p.reportSensors {
		p.handleSensors(now, pk[12:], out)
	}
	out.State(now, s)
}

// handleSensors takes in the three readings of the sensors that a full report holds.
func (p *switchPro) handleSensors(now time.Duration, imu []byte, out Sink) {
	reading := func(i int) (accel, gyro [3]int16) {
		b := imu[i*12:]
		return [3]int16{le16(b[0:]), le16(b[2:]), le16(b[4:])}, [3]int16{le16(b[6:]), le16(b[8:]), le16(b[10:])}
	}
	a0, _ := reading(0)
	if a0[0] == 0 && a0[1] == 0 && a0[2] == 0 {
		if p.hasSensorData {
			// The sensors were turned off by someone: they are turned on again, but not more often than every three seconds.
			const resetDelay = 3 * time.Second
			if now >= p.lastIMUReset+resetDelay {
				_ = p.writePacket(p.subcommand(switchSubcmdEnableIMU, []byte{1})) // tried again at the next reset
				p.lastIMUReset = now
			}
		}
		return
	}
	p.hasSensorData = true
	// Three readings arrive in each report: the rate at which they do is measured over a thousand of them.
	const rateSamples = 1000
	p.imuSamples += 3
	if p.imuSamples >= rateSamples {
		if elapsed := now - p.imuSampleTime; elapsed > 0 {
			p.imuInterval = elapsed / time.Duration(p.imuSamples)
		}
		p.imuSamples, p.imuSampleTime = 0, now
	}
	var stamps [3]time.Duration
	for i := range stamps {
		p.timestamp += p.imuInterval
		stamps[i] = p.timestamp
	}
	// The oldest reading is the last one in the report.
	for i, idx := range [3]int{2, 1, 0} {
		accel, gyro := reading(idx)
		out.Sensor(now, platform.PadSensor{Kind: platform.PadGyroscope, SensorTime: stamps[i], Values: p.gyroValues(gyro)})
		out.Sensor(now, platform.PadSensor{Kind: platform.PadAccelerometer, SensorTime: stamps[i], Values: p.accelValues(accel)})
	}
}

// gyroValues and accelValues scale a reading and put its components in the order of the PlayStation pads, which is the order of
// everything here, with their signs.
func (p *switchPro) gyroValues(v [3]int16) [3]float32 {
	gx := float32(v[0] - p.imu.gyroOffset[0])
	gy := float32(v[1] - p.imu.gyroOffset[1])
	gz := float32(v[2] - p.imu.gyroOffset[2])
	return [3]float32{-(p.imu.gyro[1] * gy), p.imu.gyro[2] * gz, -(p.imu.gyro[0] * gx)}
}

func (p *switchPro) accelValues(v [3]int16) [3]float32 {
	return [3]float32{-(p.imu.accel[1] * float32(v[1])), p.imu.accel[2] * float32(v[2]), -(p.imu.accel[0] * float32(v[0]))}
}

func (p *switchPro) Tick(now time.Duration, out Sink) {
	if !p.ready {
		return
	}
	if !p.bluetooth {
		const inputWait = 100 * time.Millisecond
		if now >= p.lastInput+inputWait {
			// Something may have put the pad back into the mode where it does not report: it is told to report again.
			p.lastInput = now
			pkt := make([]byte, switchPacketLength)
			pkt[0], pkt[1] = switchOutputProprietary, switchProprietaryForceUSB
			_ = p.writePacket(pkt) // asked for again in 100 ms if it did not work
		}
	} else if p.currentMode != switchInputSimpleState {
		const silence = 3 * time.Second
		if now >= p.lastInput+silence {
			// The Bluetooth link may be gone.
			out.Lost()
			return
		}
	}
	if p.rumblePending || p.rumbleZeroPending {
		_ = p.sendPendingRumble() // a rumble that is not written is lost; the next one follows
	} else if p.rumbleActive && now >= p.rumbleSent+switchRumbleRefresh {
		_ = p.writeRumble() // the same
	}
}
