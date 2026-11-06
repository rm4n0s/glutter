package gamepadhid

import "testing"

func TestModelOf(t *testing.T) {
	for _, tt := range []struct {
		name string
		info DeviceInfo
		want Model
	}{
		{"a DualShock 4", DeviceInfo{Vendor: 0x054c, Product: 0x09cc, Bus: BusUSB, Name: "Wireless Controller"}, ModelPS4},
		{"a DualShock 4 over Bluetooth", DeviceInfo{Vendor: 0x054c, Product: 0x05c4, Bus: BusBluetooth}, ModelPS4},
		{"the wireless adapter of the DualShock 4", DeviceInfo{Vendor: 0x054c, Product: 0x0ba0}, ModelNone},
		{"a DualSense", DeviceInfo{Vendor: 0x054c, Product: 0x0ce6, Bus: BusUSB}, ModelPS5},
		{"a DualSense Edge", DeviceInfo{Vendor: 0x054c, Product: 0x0df2, Bus: BusBluetooth}, ModelPS5},
		{"a Switch Pro controller", DeviceInfo{Vendor: 0x057e, Product: 0x2009, Bus: BusBluetooth}, ModelSwitchPro},
		{"a Joy-Con", DeviceInfo{Vendor: 0x057e, Product: 0x2006, Bus: BusBluetooth}, ModelNone},
		{"an Xbox Series pad over Bluetooth", DeviceInfo{Vendor: 0x045e, Product: 0x0b13, Bus: BusBluetooth}, ModelXboxOneBluetooth},
		{"an Xbox Series pad over USB", DeviceInfo{Vendor: 0x045e, Product: 0x0b12, Bus: BusUSB}, ModelNone},
		{"a keyboard", DeviceInfo{Vendor: 0x046d, Product: 0xc31c}, ModelNone},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := ModelOf(tt.info); got != tt.want {
				t.Errorf("ModelOf = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestGUIDOfAPadThatADriverServes(t *testing.T) {
	info := DeviceInfo{Vendor: 0x054c, Product: 0x09cc, Version: 0x8111, Bus: BusBluetooth, Name: "Wireless Controller"}
	g := GUID(info, Caps{Name: "PS4 Controller"})
	i := g.Info()
	if g[0] != 0x05 || i.Vendor != 0x054c || i.Product != 0x09cc || i.Version != 0x8111 {
		t.Errorf("info = %+v", i)
	}
	// The driver signature is 'h', which is what the mapping table of SDL has for these pads.
	if g[14] != 'h' || g[15] != 0 {
		t.Errorf("signature = %c %d", g[14], g[15])
	}
	// The CRC is of the name that the driver gave, so that two pads with the same numbers and other names differ.
	if g == GUID(info, Caps{}) {
		t.Error("the name of the driver changed nothing")
	}
}
