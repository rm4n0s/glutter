package platform

import "time"

// This file holds the interface of the gamepad service of a platform (SPEC §28.2). A backend reports the pads it finds and
// their raw buttons, axes and hats; the mapping to the standard layout, the state and the key events are made above it (package
// gamepaddb and the pad manager), so every backend gets them the same. No Linux, Windows or browser type crosses it.

// PadID names a pad while it is connected. The platform makes it; it is never reused during a run.
//
// Flutter: no counterpart
type PadID int64

// PadInfo is what a backend knows of a pad when it reports it.
//
// Flutter: no counterpart
type PadInfo struct {
	ID PadID
	// Name is the name of the pad, as gamepaddb.DeviceName makes it.
	Name string
	// GUID identifies the model (gamepaddb.GUID).
	GUID [16]byte
	// Path says where the pad is on the system ("/dev/input/event7"); it is empty where there is no such thing.
	Path string
	// Axes, Buttons and Hats are the numbers of raw axes, buttons and hats; the indices of PadEvent count from zero in them.
	Axes, Buttons, Hats int
	// Mapping is a mapping text ("a:b0,b:b1,...") that the backend made from what the pad reports, used when the table of
	// mappings has none for the GUID; empty when the backend made none.
	Mapping string
	// Standard says that the pad reports the standard layout already: its raw buttons and axes are in the order of
	// gamepaddb's Button and Axis, and no mapping is applied (a browser's "standard" mapping, XInput).
	Standard bool
	// Rumble, RumbleTriggers and LED say what the pad can do.
	Rumble, RumbleTriggers, LED bool
	// Battery is the charge of the pad from 0 to 100, or -1 when it is not known.
	Battery int
	// ButtonMask says which buttons of the layout a Standard pad has, one bit for each gamepaddb.Button; 0 means all of them.
	// A pad that reports raw inputs gets its buttons from its mapping instead.
	ButtonMask uint32
	// Touchpads is the number of touchpads of the pad (a PlayStation pad has one); each reports two fingers.
	Touchpads int
	// Sensors says that the pad has a gyroscope and an accelerometer, and SensorRate is how many readings of each it makes in
	// a second (0 when it does not say).
	Sensors    bool
	SensorRate float32
}

// PadEventKind says what a PadEvent reports.
//
// Flutter: no counterpart
type PadEventKind uint8

const (
	// PadAxis is a raw axis: Value is from -32768 to 32767.
	//
	// Flutter: no counterpart
	PadAxis PadEventKind = iota
	// PadButton is a raw button: Value is 1 when it is down and 0 when it is up.
	//
	// Flutter: no counterpart
	PadButton
	// PadHat is a raw hat: Value is a mask of 1 (up), 2 (right), 4 (down) and 8 (left).
	//
	// Flutter: no counterpart
	PadHat
)

// PadEvent is a report of one input of a pad.
//
// Flutter: no counterpart
type PadEvent struct {
	Pad PadID
	// Time is when the report was made, on the system's monotonic clock.
	Time  time.Duration
	Kind  PadEventKind
	Index int
	Value int
}

// PadHandler receives what a Gamepads backend finds. Its methods are called on the UI thread, in the order of the events, and
// never from inside Start.
//
// Flutter: no counterpart
type PadHandler interface {
	// PadAdded tells that a pad is connected. The pads that are there when Start is called are reported first.
	PadAdded(info PadInfo)
	// PadRemoved tells that the pad is gone.
	PadRemoved(id PadID)
	// PadEvents gives the reports of the pads, oldest first. The slice is valid during the call only.
	PadEvents(events []PadEvent)
}

// PadTouch is a finger on a touchpad of a pad, reported by the backends whose pads have one (the HID drivers of PlayStation
// pads, SPEC §28.2). X and Y are from 0 to 1, from the top left corner; Down says that the finger is on the pad.
//
// Flutter: no counterpart
type PadTouch struct {
	Pad PadID
	// Time is when the report was made, on the system's monotonic clock.
	Time time.Duration
	// Touchpad and Finger number the touchpad and the finger, from 0.
	Touchpad, Finger int
	Down             bool
	X, Y, Pressure   float32
}

// PadSensorKind is the kind of a sensor of a pad.
//
// Flutter: no counterpart
type PadSensorKind uint8

const (
	// PadGyroscope is the gyroscope: Values are in radians a second.
	//
	// Flutter: no counterpart
	PadGyroscope PadSensorKind = iota
	// PadAccelerometer is the accelerometer: Values are in metres a second squared, gravity included.
	//
	// Flutter: no counterpart
	PadAccelerometer
)

// PadSensor is a reading of a motion sensor of a pad. The axes are x to the right, y up and z out of the face of the pad when it
// lies flat with its sticks up, with the signs of the PlayStation pads, to which the others are made to agree.
//
// Flutter: no counterpart
type PadSensor struct {
	Pad PadID
	// Time is when the report that held the reading was made, on the system's monotonic clock; SensorTime is when the sensor
	// made the reading, on the sensor's own clock (it only means something between two readings of the same sensor).
	Time, SensorTime time.Duration
	Kind             PadSensorKind
	Values           [3]float32
}

// PadPowerState says what a pad does with its battery.
//
// Flutter: no counterpart
type PadPowerState uint8

const (
	// PadPowerUnknown is a pad that does not say.
	//
	// Flutter: no counterpart
	PadPowerUnknown PadPowerState = iota
	// PadOnBattery is a pad that runs on its battery.
	//
	// Flutter: no counterpart
	PadOnBattery
	// PadCharging is a pad whose battery is charging.
	//
	// Flutter: no counterpart
	PadCharging
	// PadCharged is a pad whose battery is full while it is plugged in.
	//
	// Flutter: no counterpart
	PadCharged
)

// PadBattery is the battery state that a pad reported.
//
// Flutter: no counterpart
type PadBattery struct {
	Pad   PadID
	State PadPowerState
	// Percent is the charge from 0 to 100, or -1 when the pad does not say.
	Percent int
}

// PadExtrasHandler is implemented by a PadHandler that takes more than the buttons and axes: the touchpad, the motion sensors
// and the battery of the pads that have them. A backend whose pads have none never looks for it. The methods are called on the
// UI thread, in order with the events, like the methods of PadHandler.
//
// Flutter: no counterpart
type PadExtrasHandler interface {
	PadTouch(t PadTouch)
	PadSensor(s PadSensor)
	PadBattery(b PadBattery)
}

// PadSensorControl is implemented by the Gamepads of a backend whose pads have motion sensors: they report only while an app
// wants them, which saves the pad's battery and the bandwidth of its link.
//
// Flutter: no counterpart
type PadSensorControl interface {
	// SetPadSensors turns the sensors of the pad on or off. It fails with GamepadSensorsUnavailable when the pad has none.
	SetPadSensors(id PadID, on bool) error
}

// Gamepads is the gamepad service of a platform.
//
// Flutter: no counterpart
type Gamepads interface {
	// MappingPlatform names the platform for the table of mappings: "Linux", "Windows", "Android" or "Web". It decides
	// which built-in mappings are loaded and which platform: fields of an app's mapping files apply.
	MappingPlatform() string
	// Start begins to look for pads and to report them to h. It may be called again after Stop.
	Start(h PadHandler) error
	// Stop ends it: no more calls reach the handler and what the backend holds is released.
	Stop()
	// Rumble makes the pad's motors turn for d (0 stops them); the strengths are from 0 to 65535. It fails with
	// GamepadRumbleUnavailable when the pad or the platform cannot, and with a tag of the backend's own when the system refuses.
	Rumble(id PadID, low, high uint16, d time.Duration) error
	// RumbleTriggers does the same for the motors of the triggers, and fails the same way.
	RumbleTriggers(id PadID, left, right uint16, d time.Duration) error
	// SetLED sets the colour of the pad's light. It fails with GamepadLEDUnavailable when the pad or the platform cannot.
	SetLED(id PadID, r, g, b uint8) error
}

// GamepadHost is implemented by the host of the dispatcher (the engine) and found through ui.PlatformDispatcher.HostAs: it
// gives access to the gamepad service of the platform, or nil when the platform has none.
//
// Flutter: no counterpart
type GamepadHost interface {
	Gamepads() Gamepads
}

// The tags of the errors a Gamepads service returns.
const (
	// TagGamepadRumbleUnavailable is the tag of the error of a rumble that the pad or the platform cannot make.
	//
	// Flutter: no counterpart
	TagGamepadRumbleUnavailable = "GamepadRumbleUnavailable"
	// TagGamepadLEDUnavailable is the tag of the error of a light that the pad or the platform cannot set.
	//
	// Flutter: no counterpart
	TagGamepadLEDUnavailable = "GamepadLEDUnavailable"
	// TagGamepadSensorsUnavailable is the tag of the error of motion sensors that the pad or the platform does not have.
	//
	// Flutter: no counterpart
	TagGamepadSensorsUnavailable = "GamepadSensorsUnavailable"
)
