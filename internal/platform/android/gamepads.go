package android

import (
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gamepaddb"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/padstd"
)

// The gamepads of Android (SPEC §28.2, step 31.2). The Java side (GamepadInput) finds the input devices that are gamepads or
// joysticks, tells Go of each one that comes and goes, and sends the key events and the joystick motion of those devices here
// instead of to the keyboard, so that each press arrives once and the pad manager makes the key events of the framework (SPEC §28.2
// "Pads as keys"). Android names the buttons and axes already (KEYCODE_BUTTON_A, AXIS_X): a pad is a pad of the standard layout,
// as the mapping that SDL 3 makes for an Android pad is (src/joystick/android/SDL_sysjoystick.c and the mapping of
// SDL_CreateMappingForAndroidGamepad in SDL_gamepad.c, translated to Go and changed). Everything here runs on the UI thread.

// The values of MotionEvent.AXIS_*.
const (
	axisX        = 0
	axisY        = 1
	axisZ        = 11
	axisRX       = 12
	axisRY       = 13
	axisRZ       = 14
	axisHatX     = 15
	axisHatY     = 16
	axisLTrigger = 17
	axisRTrigger = 18
	axisGas      = 22
	axisBrake    = 23
)

// The indices of the axes in the array of a motion event of the Java side, which is in the order of GamepadInput.AXES: X, Y, Z,
// RZ, RX, RY, LTRIGGER, RTRIGGER, BRAKE, GAS, HAT_X and HAT_Y.
const (
	slotX = iota
	slotY
	slotZ
	slotRZ
	slotRX
	slotRY
	slotLTrigger
	slotRTrigger
	slotBrake
	slotGas
	slotHatX
	slotHatY
	numPadAxes
)

// The flags of a pad that the Java side reports.
const (
	// padFlagRumble says that the device has a vibrator (VibratorManager, Android 12).
	padFlagRumble = 1
)

// The operation of the activity's command call that makes a pad rumble: A is the Android device id, B and C the strengths of the
// low and the high frequency motor from 0 to 65535, D the duration in milliseconds (0 stops the motors).
const opPadRumble = 28

// padDescriptor is what the Java side says of an input device that is a pad.
type padDescriptor struct {
	DeviceID int
	Name     string
	// Descriptor is InputDevice.getDescriptor, the stable name of the device, which the GUID is made of.
	Descriptor string
	Vendor     int
	Product    int
	Flags      int
	// Axes are the MotionEvent.AXIS_* the device has, and Ranges their minimum and maximum, two numbers for each.
	Axes   []int
	Ranges []float32
}

// axisPlan says which axes of the device are the sticks and the triggers.
type axisPlan struct {
	leftX, leftY       int // slots of a motion event, -1 for none
	rightX, rightY     int
	leftTrig, rightTrg int
	hat                bool
	// trigMin and trigMax are the range of the trigger axes, to spread over 0 to 32767.
	trigMin, trigMax [2]float32
}

// planAxes finds the sticks and triggers of a device from the axes it has: the left stick is X and Y; the right stick is Z and RZ
// (the convention of Android's gamepads) or, where those are not there, RX and RY; the triggers are LTRIGGER and RTRIGGER, or
// BRAKE and GAS, or, where the right stick is on Z and RZ, RX and RY (a DualShock 4's).
func planAxes(axes []int, ranges []float32) axisPlan {
	has := map[int]int{}
	for i, a := range axes {
		has[a] = i
	}
	rng := func(axis int) (lo, hi float32) {
		i, ok := has[axis]
		if !ok || 2*i+1 >= len(ranges) {
			return 0, 1
		}
		return ranges[2*i], ranges[2*i+1]
	}
	slotOf := map[int]int{axisX: slotX, axisY: slotY, axisZ: slotZ, axisRZ: slotRZ, axisRX: slotRX, axisRY: slotRY,
		axisLTrigger: slotLTrigger, axisRTrigger: slotRTrigger, axisBrake: slotBrake, axisGas: slotGas}
	both := func(a, b int) bool { _, x := has[a]; _, y := has[b]; return x && y }
	plan := axisPlan{leftX: -1, leftY: -1, rightX: -1, rightY: -1, leftTrig: -1, rightTrg: -1}
	if both(axisX, axisY) {
		plan.leftX, plan.leftY = slotX, slotY
	}
	rightOnZ := false
	switch {
	case both(axisZ, axisRZ):
		plan.rightX, plan.rightY, rightOnZ = slotZ, slotRZ, true
	case both(axisRX, axisRY):
		plan.rightX, plan.rightY = slotRX, slotRY
	}
	var trigAxes [2]int
	switch {
	case both(axisLTrigger, axisRTrigger):
		trigAxes = [2]int{axisLTrigger, axisRTrigger}
	case both(axisBrake, axisGas):
		trigAxes = [2]int{axisBrake, axisGas}
	case rightOnZ && both(axisRX, axisRY):
		trigAxes = [2]int{axisRX, axisRY}
	}
	if trigAxes != ([2]int{}) {
		plan.leftTrig, plan.rightTrg = slotOf[trigAxes[0]], slotOf[trigAxes[1]]
		plan.trigMin[0], plan.trigMax[0] = rng(trigAxes[0])
		plan.trigMin[1], plan.trigMax[1] = rng(trigAxes[1])
	}
	_, hx := has[axisHatX]
	_, hy := has[axisHatY]
	plan.hat = hx || hy
	return plan
}

// triggerValue spreads the value of a trigger axis over 0 to 32767: from its minimum (at rest) to its maximum.
func triggerValue(v, lo, hi float32) int16 {
	if hi <= lo {
		return 0
	}
	return padstd.Unit(float64((v - lo) / (hi - lo)))
}

// The values of KeyEvent.KEYCODE_*.
const (
	keycodeBack         = 4
	keycodeDpadUp       = 19
	keycodeDpadDown     = 20
	keycodeDpadLeft     = 21
	keycodeDpadRight    = 22
	keycodeDpadCenter   = 23
	keycodeButtonA      = 96
	keycodeButtonB      = 97
	keycodeButtonX      = 99
	keycodeButtonY      = 100
	keycodeButtonL1     = 102
	keycodeButtonR1     = 103
	keycodeButtonL2     = 104
	keycodeButtonR2     = 105
	keycodeButtonThumbL = 106
	keycodeButtonThumbR = 107
	keycodeButtonStart  = 108
	keycodeButtonSelect = 109
	keycodeButtonMode   = 110
	keycodeMediaRecord  = 130
)

// IsPadKey says whether a key of a gamepad device is one that the pad service takes: a button of the pad or its D-pad. Back and the
// other keys that a pad sends are left to the keyboard. The Java side asks the same of its own table (GamepadInput.isPadKey).
//
// Flutter: no counterpart
func IsPadKey(keyCode int) bool {
	_, button := padButtonOfKey(keyCode)
	return button || keyCode == keycodeButtonL2 || keyCode == keycodeButtonR2
}

// padButtonOfKey is the button of the standard layout that a key code is. DPAD_CENTER is the A button, as SDL takes it, and the
// record key of some pads the extra button. The two lower triggers are not here: they are axes (padService.key).
func padButtonOfKey(keyCode int) (gamepaddb.Button, bool) {
	switch keyCode {
	case keycodeButtonA, keycodeDpadCenter:
		return gamepaddb.South, true
	case keycodeButtonB:
		return gamepaddb.East, true
	case keycodeButtonX:
		return gamepaddb.West, true
	case keycodeButtonY:
		return gamepaddb.North, true
	case keycodeButtonL1:
		return gamepaddb.LeftShoulder, true
	case keycodeButtonR1:
		return gamepaddb.RightShoulder, true
	case keycodeButtonThumbL:
		return gamepaddb.LeftStick, true
	case keycodeButtonThumbR:
		return gamepaddb.RightStick, true
	case keycodeButtonStart:
		return gamepaddb.Start, true
	case keycodeButtonSelect:
		return gamepaddb.Back, true
	case keycodeButtonMode:
		return gamepaddb.Guide, true
	case keycodeDpadUp:
		return gamepaddb.DPadUp, true
	case keycodeDpadDown:
		return gamepaddb.DPadDown, true
	case keycodeDpadLeft:
		return gamepaddb.DPadLeft, true
	case keycodeDpadRight:
		return gamepaddb.DPadRight, true
	case keycodeMediaRecord:
		return gamepaddb.Misc1, true
	}
	return 0, false
}

// androidPad is a pad that is connected.
type androidPad struct {
	desc    padDescriptor
	id      platform.PadID
	plan    axisPlan
	tracker *padstd.Tracker
	snap    padstd.Snapshot
	// reported says that PadAdded was told to the handler (the service had one when the pad came, or since).
	reported bool
}

// padService is platform.Gamepads for Android. The Java side's events reach it through the bridge, on the UI thread.
type padService struct {
	// Everything here belongs to the UI thread.
	host   host
	h      platform.PadHandler
	pads   map[int]*androidPad
	order  []int
	nextID platform.PadID
}

var _ platform.Gamepads = (*padService)(nil)

func newPadService(h host) *padService { return &padService{host: h, pads: map[int]*androidPad{}} }

// MappingPlatform implements platform.Gamepads.
func (s *padService) MappingPlatform() string { return "Android" }

// Start implements platform.Gamepads: the pads that are there are reported.
func (s *padService) Start(h platform.PadHandler) error {
	s.h = h
	for _, dev := range s.order {
		s.report(s.pads[dev])
	}
	return nil
}

// Stop implements platform.Gamepads.
func (s *padService) Stop() {
	s.h = nil
	for _, p := range s.pads {
		p.reported = false
	}
}

func (s *padService) info(p *androidPad) platform.PadInfo {
	d := p.desc
	vendor, product := uint16(d.Vendor), uint16(d.Product)
	raw := d.Name
	return platform.PadInfo{
		ID:   p.id,
		Name: gamepaddb.DeviceName(vendor, product, "", raw),
		// SDL makes the GUID of an Android pad from the descriptor, on the Bluetooth bus.
		GUID:     gamepaddb.NewGUID(gamepaddb.BusBluetooth, vendor, product, 0, "", d.Descriptor, 0, 0),
		Path:     d.Descriptor,
		Axes:     int(gamepaddb.AxisCount),
		Buttons:  int(gamepaddb.ButtonCount),
		Standard: true,
		Rumble:   d.Flags&padFlagRumble != 0,
		Battery:  -1,
	}
}

func (s *padService) report(p *androidPad) {
	if s.h == nil || p.reported {
		return
	}
	p.reported = true
	s.h.PadAdded(s.info(p))
}

// added is a pad that the Java side found (the device list at the start, or a device that came). One that is there already is left
// as it is.
func (s *padService) added(d padDescriptor) {
	if _, ok := s.pads[d.DeviceID]; ok {
		return
	}
	s.nextID++
	p := &androidPad{desc: d, id: s.nextID, plan: planAxes(d.Axes, d.Ranges), tracker: padstd.NewTracker(s.nextID)}
	s.pads[d.DeviceID] = p
	s.order = append(s.order, d.DeviceID)
	s.report(p)
}

// removed is a device that went.
func (s *padService) removed(device int) {
	p, ok := s.pads[device]
	if !ok {
		return
	}
	delete(s.pads, device)
	for i, dev := range s.order {
		if dev == device {
			s.order = append(s.order[:i], s.order[i+1:]...)
			break
		}
	}
	if s.h != nil && p.reported {
		s.h.PadRemoved(p.id)
	}
}

// sync drops the pads that are not among the devices the Java side has now: it lists them when an activity starts, because a pad
// may have gone while there was none to hear of it.
func (s *padService) sync(devices []int) {
	present := map[int]bool{}
	for _, d := range devices {
		present[d] = true
	}
	for _, dev := range append([]int(nil), s.order...) {
		if !present[dev] {
			s.removed(dev)
		}
	}
}

// update gives the handler the events of what a report changed.
func (s *padService) update(p *androidPad, eventTime time.Duration) {
	if s.h == nil || !p.reported {
		return
	}
	if events := p.tracker.Update(eventTime, &p.snap, nil); len(events) > 0 {
		s.h.PadEvents(events)
	}
}

// key is a key event of a pad (down is true for ACTION_DOWN); eventTime is KeyEvent.getEventTime in milliseconds. It reports whether
// the service took it: a pad it does not know, and a key that is not a pad's, are left to the keyboard.
func (s *padService) key(device int, down bool, keyCode int, eventTime int64) bool {
	p, ok := s.pads[device]
	if !ok {
		return false
	}
	t := time.Duration(eventTime) * time.Millisecond
	switch keyCode {
	case keycodeButtonL2, keycodeButtonR2:
		// The triggers are axes where the pad has them; a pad that has only the keys has them as keys.
		slot, trigger := p.plan.leftTrig, gamepaddb.LeftTrigger
		if keyCode == keycodeButtonR2 {
			slot, trigger = p.plan.rightTrg, gamepaddb.RightTrigger
		}
		if slot >= 0 {
			return true
		}
		if down {
			p.snap.Axes[trigger] = gamepaddb.AxisMax
		} else {
			p.snap.Axes[trigger] = 0
		}
		s.update(p, t)
		return true
	}
	button, ok := padButtonOfKey(keyCode)
	if !ok {
		return false
	}
	p.snap.Buttons[button] = down
	s.update(p, t)
	return true
}

// motion is the joystick motion of a pad: the values of the axes in the order of the slots. A pad that is not known is ignored.
func (s *padService) motion(device int, vals []float32, eventTime int64) {
	p, ok := s.pads[device]
	if !ok || len(vals) < numPadAxes {
		return
	}
	pl := p.plan
	sig := func(slot int) int16 {
		if slot < 0 {
			return 0
		}
		return padstd.Signed(float64(vals[slot]))
	}
	p.snap.Axes[gamepaddb.LeftX], p.snap.Axes[gamepaddb.LeftY] = sig(pl.leftX), sig(pl.leftY)
	p.snap.Axes[gamepaddb.RightX], p.snap.Axes[gamepaddb.RightY] = sig(pl.rightX), sig(pl.rightY)
	if pl.leftTrig >= 0 {
		p.snap.Axes[gamepaddb.LeftTrigger] = triggerValue(vals[pl.leftTrig], pl.trigMin[0], pl.trigMax[0])
		p.snap.Axes[gamepaddb.RightTrigger] = triggerValue(vals[pl.rightTrg], pl.trigMin[1], pl.trigMax[1])
	}
	if pl.hat {
		x, y := vals[slotHatX], vals[slotHatY]
		p.snap.Buttons[gamepaddb.DPadLeft], p.snap.Buttons[gamepaddb.DPadRight] = x < -0.5, x > 0.5
		p.snap.Buttons[gamepaddb.DPadUp], p.snap.Buttons[gamepaddb.DPadDown] = y < -0.5, y > 0.5
	}
	s.update(p, time.Duration(eventTime)*time.Millisecond)
}

// deviceOf finds the Android device of a pad.
func (s *padService) deviceOf(id platform.PadID) (int, *androidPad, bool) {
	for dev, p := range s.pads {
		if p.id == id {
			return dev, p, true
		}
	}
	return 0, nil, false
}

// Rumble implements platform.Gamepads: the activity makes the device's vibrators run (Android 12 and later), and a duration of 0
// stops them.
func (s *padService) Rumble(id platform.PadID, low, high uint16, d time.Duration) error {
	dev, p, ok := s.deviceOf(id)
	if !ok || p.desc.Flags&padFlagRumble == 0 {
		return errors.New(platform.TagGamepadRumbleUnavailable, "the pad has no vibrator (Android 12 and later) or is not there", "pad", int64(id))
	}
	ms := int64(0)
	if d > 0 && (low != 0 || high != 0) {
		ms = max(1, int64(d/time.Millisecond))
	}
	s.host.Command(command{Op: opPadRumble, A: int64(dev), B: int64(low), C: int64(high), D: ms})
	return nil
}

// RumbleTriggers implements platform.Gamepads: Android has no motors for the triggers.
func (s *padService) RumbleTriggers(id platform.PadID, left, right uint16, d time.Duration) error {
	return errors.New(platform.TagGamepadRumbleUnavailable, "the triggers of a pad cannot rumble on Android", "pad", int64(id))
}

// SetLED implements platform.Gamepads: the lights of a pad are not set on Android.
func (s *padService) SetLED(id platform.PadID, r, g, b uint8) error {
	return errors.New(platform.TagGamepadLEDUnavailable, "the light of a pad cannot be set on Android", "pad", int64(id))
}

// The bridge's side: the events of the Java side go to the UI thread, where the service is.

// padAdded is a device that the Java side found.
func (b *bridge) padAdded(d padDescriptor) { b.post(func(p *Platform) { p.pads.added(d) }) }

// padRemoved is a device that went.
func (b *bridge) padRemoved(device int) { b.post(func(p *Platform) { p.pads.removed(device) }) }

// padSync is the list of the pad devices the Java side has now.
func (b *bridge) padSync(devices []int) { b.post(func(p *Platform) { p.pads.sync(devices) }) }

// padKey is a key event of a pad.
func (b *bridge) padKey(device int, down bool, keyCode int, eventTime int64) {
	b.post(func(p *Platform) { p.pads.key(device, down, keyCode, eventTime) })
}

// padMotion is the joystick motion of a pad.
func (b *bridge) padMotion(device int, vals []float32, eventTime int64) {
	cp := append([]float32(nil), vals...)
	b.post(func(p *Platform) { p.pads.motion(device, cp, eventTime) })
}
