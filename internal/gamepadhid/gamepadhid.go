// Package gamepadhid holds the HID drivers of the pads that report more through HID than the system's own driver shows: the
// PlayStation 4 and 5 pads, the Nintendo Switch Pro controller and the Xbox One and Series pads over Bluetooth (SPEC §28.1,
// §28.2). A driver reads the pad's input reports and makes the standard layout of them (internal/platform/padstd), the
// touchpad, the motion sensors and the battery, and it writes the output reports of rumble, trigger rumble and the lights.
//
// The drivers are plain Go over a small Transport (read a report, write a report, feature reports): the transport of Linux is
// /dev/hidraw*, that of Windows is hid.dll, and the tests use scripted ones, so the drivers are tested on any host. A Device
// runs one driver on a goroutine of its own and tells a Handler what it finds; the platform backends make the Handler.
//
// The drivers are translated to Go and changed from src/joystick/hidapi/SDL_hidapi_ps4.c, SDL_hidapi_ps5.c,
// SDL_hidapi_switch.c, SDL_hidapi_xboxone.c and SDL_hidapi_rumble.c of SDL 3 (zlib license, third_party/sdl/LICENSE).
//
// Flutter: no counterpart
package gamepadhid

import (
	"time"

	"github.com/rm4n0s/glutter/internal/gamepaddb"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/padstd"
)

// Bus is how a pad is connected.
//
// Flutter: no counterpart
type Bus uint8

const (
	// BusUnknown is a pad whose connection is not known.
	//
	// Flutter: no counterpart
	BusUnknown Bus = iota
	// BusUSB is a pad on USB.
	//
	// Flutter: no counterpart
	BusUSB
	// BusBluetooth is a pad on Bluetooth.
	//
	// Flutter: no counterpart
	BusBluetooth
)

// DeviceInfo is what the platform knows of a HID device before a driver looks at it.
//
// Flutter: no counterpart
type DeviceInfo struct {
	Vendor, Product, Version uint16
	Bus                      Bus
	// Name is the product name that the device reports.
	Name string
	// Path says where the device is on the system.
	Path string
}

// Transport is the link to a HID device: the reports that go in and out and the feature reports. The first byte of a report is
// its report ID, as on the wire. Read and Write may be called from different goroutines at the same time; Close ends them.
//
// Flutter: no counterpart
type Transport interface {
	// Read waits up to timeout for an input report and puts it in buf. It returns 0 and no error when the time ran out, and an
	// error when the device is gone.
	Read(buf []byte, timeout time.Duration) (int, error)
	// Write sends an output report and returns the number of bytes sent.
	Write(data []byte) (int, error)
	// SendFeature sends a feature report (data[0] is its ID) and returns the number of bytes sent.
	SendFeature(data []byte) (int, error)
	// GetFeature asks for a feature report: data[0] is the ID of the report, and the report is put in data, with its ID in
	// the first byte. It returns the number of bytes, ID included.
	GetFeature(data []byte) (int, error)
	// Close ends the link; reads and writes that wait return an error.
	Close() error
}

// IO is what a driver sees of its device: the transport, with the output queue of the rumble thread of SDL (a report that is
// sent is written on another goroutine, a pause after another, and a report of the same kind that has not been written yet is
// replaced by the new one).
//
// Flutter: no counterpart
type IO interface {
	// Read waits up to timeout for an input report, as Transport.Read does.
	Read(buf []byte, timeout time.Duration) (int, error)
	// Write sends an output report now and returns when it is sent.
	Write(data []byte) (int, error)
	// Send queues an output report. A report that is queued and not written yet, with the same length and report ID, is
	// replaced by this one.
	Send(data []byte) error
	// SendMerging queues an output report like Send, but replaces a queued one of the same length when merge says so.
	SendMerging(data []byte, merge func(pending []byte) bool) error
	// Feature asks for the feature report id: buf is zeroed and filled by the answer (ID first). It returns the number of
	// bytes, or an error.
	Feature(id byte, buf []byte) (int, error)
	// SendFeature sends a feature report (data[0] is its ID).
	SendFeature(data []byte) (int, error)
	// Now is the monotonic time.
	Now() time.Duration
}

// Sink receives what a driver reads from the pad. It is called on the goroutine of the device, with its lock held, so it must
// not call the device back.
//
// Flutter: no counterpart
type Sink interface {
	// State gives the buttons and axes of the layout after a report.
	State(now time.Duration, s *padstd.Snapshot)
	// Touch gives a finger of a touchpad.
	Touch(now time.Duration, t platform.PadTouch)
	// Sensor gives a reading of a motion sensor.
	Sensor(now time.Duration, s platform.PadSensor)
	// Battery gives the battery state of the pad.
	Battery(b platform.PadBattery)
	// Lost says that the pad is gone although the link is still there (a wireless pad whose dongle is, for instance).
	Lost()
}

// Caps is what a driver knows of its pad once it has started.
//
// Flutter: no counterpart
type Caps struct {
	// Name is the name that the driver gives the pad, or "" to keep the one of the device.
	Name string
	Kind gamepaddb.Kind
	// Rumble, RumbleTriggers and LED say what the pad can do (the light is a light bar or the Home light).
	Rumble, RumbleTriggers, LED bool
	// Sensors says that the pad has a gyroscope and an accelerometer that SetSensors turns on, reading Rate times a second.
	Sensors bool
	Rate    float32
	// Touchpads is the number of touchpads.
	Touchpads int
	// Buttons says which buttons of the layout the pad has: a bit for each gamepaddb.Button.
	Buttons uint32
}

// Driver is the code of one model of pad. It is used by one goroutine at a time (a Device holds a lock), so it keeps its state
// without any of its own.
//
// Flutter: no counterpart
type Driver interface {
	// Start talks to the pad (feature reports, the first report) and returns what it can do. An error means the pad is not
	// for this driver and the platform may read it another way.
	Start(io IO) (Caps, error)
	// Report takes in an input report.
	Report(now time.Duration, data []byte, out Sink)
	// Tick is called after every report and every time a read waits in vain: it keeps what needs the time (a Bluetooth link
	// that went quiet, a rumble that has to be refreshed).
	Tick(now time.Duration, out Sink)
	// Rumble sets the strengths of the two motors (0 to 65535); 0 and 0 stops them.
	Rumble(low, high uint16) error
	// RumbleTriggers sets the strengths of the two trigger motors; 0 and 0 stops them.
	RumbleTriggers(left, right uint16) error
	// SetLED sets the colour of the light.
	SetLED(r, g, b uint8) error
	// SetSensors turns the sensors on or off.
	SetSensors(on bool) error
	// Close restores what the driver changed that other programs would meet.
	Close()
}

// Model is the driver that a device needs.
//
// Flutter: no counterpart
type Model uint8

const (
	// ModelNone is a device that no driver here serves.
	//
	// Flutter: no counterpart
	ModelNone Model = iota
	// ModelPS4 is a PlayStation 4 pad (DualShock 4) or one that behaves like it.
	//
	// Flutter: no counterpart
	ModelPS4
	// ModelPS5 is a PlayStation 5 pad (DualSense, DualSense Edge) or one that behaves like it.
	//
	// Flutter: no counterpart
	ModelPS5
	// ModelSwitchPro is the Nintendo Switch Pro controller.
	//
	// Flutter: no counterpart
	ModelSwitchPro
	// ModelXboxOneBluetooth is an Xbox One or Series pad over Bluetooth.
	//
	// Flutter: no counterpart
	ModelXboxOneBluetooth
)

// The vendor and product numbers that the drivers know by themselves.
const (
	vendorSony      = 0x054c
	vendorMicrosoft = 0x045e
	vendorNintendo  = 0x057e
	vendorRazer     = 0x1532
	vendorPDP       = 0x0e6f

	productDS4Dongle      = 0x0ba0
	productDS4Strikepad   = 0x05c5
	productDualSenseEdge  = 0x0df2
	productSwitchPro      = 0x2009
	productVictrixFSPro   = 0x0203
	productVictrixFSProV2 = 0x0207
)

// ModelOf says which driver serves a device, or ModelNone. The choice follows the pad's kind in SDL's controller list; the
// Switch Pro driver serves only the controller of Nintendo, and the Xbox driver only the Bluetooth pads of Microsoft, which it
// knows by their product numbers (over USB the system's drivers serve the Xbox pads, XInput on Windows and xpad on Linux).
//
// Flutter: no counterpart
func ModelOf(info DeviceInfo) Model {
	switch gamepaddb.KindOf(info.Vendor, info.Product, info.Name, false) {
	case gamepaddb.KindPS4:
		if info.Vendor == vendorSony && info.Product == productDS4Dongle {
			// The wireless adapter makes a pad of its own when a pad connects to it; that needs more than this driver does.
			return ModelNone
		}
		return ModelPS4
	case gamepaddb.KindPS5:
		return ModelPS5
	case gamepaddb.KindSwitchPro:
		if info.Vendor == vendorNintendo && info.Product == productSwitchPro {
			return ModelSwitchPro
		}
	case gamepaddb.KindXboxOne:
		if isBluetoothXboxOne(info.Vendor, info.Product) {
			return ModelXboxOneBluetooth
		}
	}
	return ModelNone
}

// GUID makes the GUID of a pad that a driver serves, as SDL makes it for a pad of its HID drivers: the bus, the vendor, the
// product and the version of the device, the CRC of the name the driver gave it and the driver signature 'h'.
//
// Flutter: no counterpart
func GUID(info DeviceInfo, caps Caps) gamepaddb.GUID {
	bus := gamepaddb.BusUSB
	if info.Bus == BusBluetooth {
		bus = gamepaddb.BusBluetooth
	}
	name := info.Name
	if caps.Name != "" {
		name = caps.Name
	}
	return gamepaddb.NewGUID(bus, info.Vendor, info.Product, info.Version, "", name, 'h', 0)
}

// newDriver makes the driver of a model.
func newDriver(m Model, info DeviceInfo) Driver {
	switch m {
	case ModelPS4:
		return newPS4(info)
	case ModelPS5:
		return newPS5(info)
	case ModelSwitchPro:
		return newSwitchPro(info)
	case ModelXboxOneBluetooth:
		return newXboxOne(info)
	}
	return nil
}

// buttonBit is the bit of a button in Caps.Buttons.
func buttonBit(buttons ...gamepaddb.Button) uint32 {
	var m uint32
	for _, b := range buttons {
		m |= 1 << b
	}
	return m
}

// The tags of the errors that this package makes (rm4n0s/errors).
const (
	// tagHidDriverStartFailed is the tag of the error of a driver that could not start: the pad is not what the driver serves,
	// or it does not answer.
	tagHidDriverStartFailed = "HidDriverStartFailed"
	// tagHidReportTooLong is the tag of the error of an output report that is longer than the pad takes.
	tagHidReportTooLong = "HidReportTooLong"
)
