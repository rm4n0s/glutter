package gamepadhid

import (
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
)

// checkRoute fails the test unless err is an error of the module with the route.
func checkRoute(t *testing.T, err error, route string) {
	t.Helper()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if !appErr.HasRoute(route) {
		t.Errorf("unexpected failure path: %s (want %s)", appErr.Route(), route)
	}
}

func TestDriversRefuseWhatThePadCannotDo(t *testing.T) {
	ps4Pad, _ := startPS4(t, sonyPS4(BusUSB), &fakeIO{})
	ps5Pad, _ := startedPS5USB(t, 0x0224)
	switchPad, _, _ := startSwitch(t, BusBluetooth, nil)
	xbox, _, _ := startXbox(t, productXboxSeriesXBLE)

	for _, tt := range []struct {
		name  string
		err   error
		route string
	}{
		{"triggers of a PlayStation 4 pad", ps4Pad.RumbleTriggers(1, 1), "ps4.RumbleTriggers." + platform.TagGamepadRumbleUnavailable},
		{"triggers of a Switch Pro controller", switchPad.RumbleTriggers(1, 1), "switchPro.RumbleTriggers." + platform.TagGamepadRumbleUnavailable},
		{"the light of an Xbox pad", xbox.SetLED(1, 1, 1), "xboxOne.SetLED." + platform.TagGamepadLEDUnavailable},
		{"the sensors of an Xbox pad", xbox.SetSensors(true), "xboxOne.SetSensors." + platform.TagGamepadSensorsUnavailable},
		{"a PlayStation 5 pad that takes no effects", func() error {
			d := *ps5Pad
			d.vibrationOK = false
			return d.Rumble(1, 1)
		}(), "ps5.Rumble." + platform.TagGamepadRumbleUnavailable},
		{"the sensors of a pad that has none", func() error {
			d := *ps4Pad
			d.sensorsOK = false
			return d.SetSensors(true)
		}(), "ps4.SetSensors." + platform.TagGamepadSensorsUnavailable},
		{"the light of a pad that has none", func() error {
			d := *ps4Pad
			d.lightbarOK = false
			return d.SetLED(1, 1, 1)
		}(), "ps4.SetLED." + platform.TagGamepadLEDUnavailable},
	} {
		t.Run(tt.name, func(t *testing.T) { checkRoute(t, tt.err, tt.route) })
	}
}

func TestSwitchProReportThatIsTooLong(t *testing.T) {
	d, _, _ := startSwitch(t, BusBluetooth, nil)
	checkRoute(t, d.writePacket(make([]byte, 60)), "switchPro.writePacket.HidReportTooLong")
}

func TestPadThatTakesNoEffectsRefusesThemWithARoute(t *testing.T) {
	info := DeviceInfo{Vendor: vendorPDP, Product: productVictrixFSPro, Bus: BusUSB, Name: "FS Pro"}
	feat := make([]byte, 48)
	feat[0], feat[2], feat[4] = 0x03, 0x27, 0x04|0x08
	d := newPS4(info).(*ps4)
	if _, err := d.Start(&fakeIO{features: map[byte][]byte{0x03: feat}}); err != nil {
		t.Fatal(err)
	}
	checkRoute(t, d.Rumble(1, 1), "ps4.Rumble->ps4.updateEffects->ps4.sendEffect."+platform.TagGamepadRumbleUnavailable)
}

func TestDeviceThatIsNotReadyRefusesItsCommands(t *testing.T) {
	tr, h := newFakeTransport(), newRecHandler()
	drv := &stubDriver{startErr: errors.New(tagStubStartFailed, "no answer")}
	d := startWith(drv, DeviceInfo{}, tr, h, fastOptions(nil))
	_ = within(t, "a failure", h.failed) // the error is not what this test looks at
	d.Close()
	checkRoute(t, d.Rumble(1, 1, 1), "Device.Rumble."+platform.TagGamepadRumbleUnavailable)
	checkRoute(t, d.RumbleTriggers(1, 1, 1), "Device.RumbleTriggers."+platform.TagGamepadRumbleUnavailable)
	checkRoute(t, d.SetLED(1, 1, 1), "Device.SetLED."+platform.TagGamepadLEDUnavailable)
	checkRoute(t, d.SetSensors(true), "Device.SetSensors."+platform.TagGamepadSensorsUnavailable)
}

// tagStubStartFailed is the tag of the failure of the stub driver.
const tagStubStartFailed = "StubStartFailed"
