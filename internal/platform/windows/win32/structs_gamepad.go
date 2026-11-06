package win32

// The Go mirrors of the structures of XInput, Raw Input and the HID parser that the gamepad service of Windows uses (step 31.2,
// SPEC §28.2). They have no OS tag, so that the layout tests check them on the Linux host against the mingw-w64 headers.

// XInputGamepad is the Go mirror of XINPUT_GAMEPAD.
//
// Flutter: no counterpart
type XInputGamepad struct {
	Buttons      uint16 `c:"wButtons"`
	LeftTrigger  uint8  `c:"bLeftTrigger"`
	RightTrigger uint8  `c:"bRightTrigger"`
	ThumbLX      int16  `c:"sThumbLX"`
	ThumbLY      int16  `c:"sThumbLY"`
	ThumbRX      int16  `c:"sThumbRX"`
	ThumbRY      int16  `c:"sThumbRY"`
}

// XInputState is the Go mirror of XINPUT_STATE.
//
// Flutter: no counterpart
type XInputState struct {
	PacketNumber uint32        `c:"dwPacketNumber"`
	Gamepad      XInputGamepad `c:"Gamepad"`
}

// XInputVibration is the Go mirror of XINPUT_VIBRATION: the speeds of the two motors, from 0 to 65535.
//
// Flutter: no counterpart
type XInputVibration struct {
	LeftMotorSpeed  uint16 `c:"wLeftMotorSpeed"`
	RightMotorSpeed uint16 `c:"wRightMotorSpeed"`
}

// XInputCapabilities is the Go mirror of XINPUT_CAPABILITIES.
//
// Flutter: no counterpart
type XInputCapabilities struct {
	Type      uint8           `c:"Type"`
	SubType   uint8           `c:"SubType"`
	Flags     uint16          `c:"Flags"`
	Gamepad   XInputGamepad   `c:"Gamepad"`
	Vibration XInputVibration `c:"Vibration"`
}

// XInputCapabilitiesEx is the Go mirror of XINPUT_CAPABILITIES_EX, which XInputGetCapabilitiesEx (an undocumented function of
// xinput1_4.dll, ordinal 108) fills: the capabilities and the vendor, product and version of the pad.
//
// Flutter: no counterpart
type XInputCapabilitiesEx struct {
	Capabilities  XInputCapabilities `c:"Capabilities"`
	VendorID      uint16             `c:"VendorId"`
	ProductID     uint16             `c:"ProductId"`
	VersionNumber uint16             `c:"VersionNumber"`
	_             uint16
	_             uint32
}

// XInputBatteryInformation is the Go mirror of XINPUT_BATTERY_INFORMATION.
//
// Flutter: no counterpart
type XInputBatteryInformation struct {
	BatteryType  uint8 `c:"BatteryType"`
	BatteryLevel uint8 `c:"BatteryLevel"`
}

// RawInputDeviceList is the Go mirror of RAWINPUTDEVICELIST (GetRawInputDeviceList).
//
// Flutter: no counterpart
type RawInputDeviceList struct {
	Device uintptr `c:"hDevice"`
	Type   uint32  `c:"dwType"`
}

// RidDeviceInfoHID is the Go mirror of RID_DEVICE_INFO_HID.
//
// Flutter: no counterpart
type RidDeviceInfoHID struct {
	VendorID      uint32 `c:"dwVendorId"`
	ProductID     uint32 `c:"dwProductId"`
	VersionNumber uint32 `c:"dwVersionNumber"`
	UsagePage     uint16 `c:"usUsagePage"`
	Usage         uint16 `c:"usUsage"`
}

// RidDeviceInfo is the Go mirror of RID_DEVICE_INFO for a HID device: the union holds the largest member (a keyboard's) and the
// HID member is the first 20 bytes of it.
//
// Flutter: no counterpart
type RidDeviceInfo struct {
	Size uint32           `c:"cbSize"`
	Type uint32           `c:"dwType"`
	HID  RidDeviceInfoHID `c:"hid"`
	_    [8]byte
}

// HidpCaps is the Go mirror of HIDP_CAPS.
//
// Flutter: no counterpart
type HidpCaps struct {
	Usage                     uint16     `c:"Usage"`
	UsagePage                 uint16     `c:"UsagePage"`
	InputReportByteLength     uint16     `c:"InputReportByteLength"`
	OutputReportByteLength    uint16     `c:"OutputReportByteLength"`
	FeatureReportByteLength   uint16     `c:"FeatureReportByteLength"`
	Reserved                  [17]uint16 `c:"Reserved"`
	NumberLinkCollectionNodes uint16     `c:"NumberLinkCollectionNodes"`
	NumberInputButtonCaps     uint16     `c:"NumberInputButtonCaps"`
	NumberInputValueCaps      uint16     `c:"NumberInputValueCaps"`
	NumberInputDataIndices    uint16     `c:"NumberInputDataIndices"`
	NumberOutputButtonCaps    uint16     `c:"NumberOutputButtonCaps"`
	NumberOutputValueCaps     uint16     `c:"NumberOutputValueCaps"`
	NumberOutputDataIndices   uint16     `c:"NumberOutputDataIndices"`
	NumberFeatureButtonCaps   uint16     `c:"NumberFeatureButtonCaps"`
	NumberFeatureValueCaps    uint16     `c:"NumberFeatureValueCaps"`
	NumberFeatureDataIndices  uint16     `c:"NumberFeatureDataIndices"`
}

// HidpButtonCaps is the Go mirror of HIDP_BUTTON_CAPS. Its union of Range and NotRange (16 bytes) is Union; the methods read it.
//
// Flutter: no counterpart
type HidpButtonCaps struct {
	UsagePage       uint16     `c:"UsagePage"`
	ReportID        uint8      `c:"ReportID"`
	IsAlias         uint8      `c:"IsAlias"`
	BitField        uint16     `c:"BitField"`
	LinkCollection  uint16     `c:"LinkCollection"`
	LinkUsage       uint16     `c:"LinkUsage"`
	LinkUsagePage   uint16     `c:"LinkUsagePage"`
	IsRange         uint8      `c:"IsRange"`
	IsStringRange   uint8      `c:"IsStringRange"`
	IsDesignatorRng uint8      `c:"IsDesignatorRange"`
	IsAbsolute      uint8      `c:"IsAbsolute"`
	Reserved        [10]uint32 `c:"Reserved"`
	Union           [16]byte   `c:"Range"`
}

// UsageMin is the first usage of a range (Range.UsageMin), or the usage (NotRange.Usage), which is at the same place.
//
// Flutter: no counterpart
func (c *HidpButtonCaps) UsageMin() uint16 { return unionU16(&c.Union, 0) }

// UsageMax is the last usage of a range (Range.UsageMax); it is meaningful when IsRange is set.
//
// Flutter: no counterpart
func (c *HidpButtonCaps) UsageMax() uint16 { return unionU16(&c.Union, 2) }

// DataIndexMin is the data index of the first usage of a range (Range.DataIndexMin), or the data index (NotRange.DataIndex).
//
// Flutter: no counterpart
func (c *HidpButtonCaps) DataIndexMin() uint16 { return unionU16(&c.Union, 12) }

// DataIndexMax is the data index of the last usage of a range (Range.DataIndexMax).
//
// Flutter: no counterpart
func (c *HidpButtonCaps) DataIndexMax() uint16 { return unionU16(&c.Union, 14) }

// HidpValueCaps is the Go mirror of HIDP_VALUE_CAPS.
//
// Flutter: no counterpart
type HidpValueCaps struct {
	UsagePage       uint16 `c:"UsagePage"`
	ReportID        uint8  `c:"ReportID"`
	IsAlias         uint8  `c:"IsAlias"`
	BitField        uint16 `c:"BitField"`
	LinkCollection  uint16 `c:"LinkCollection"`
	LinkUsage       uint16 `c:"LinkUsage"`
	LinkUsagePage   uint16 `c:"LinkUsagePage"`
	IsRange         uint8  `c:"IsRange"`
	IsStringRange   uint8  `c:"IsStringRange"`
	IsDesignatorRng uint8  `c:"IsDesignatorRange"`
	IsAbsolute      uint8  `c:"IsAbsolute"`
	HasNull         uint8  `c:"HasNull"`
	_               uint8
	BitSize         uint16    `c:"BitSize"`
	ReportCount     uint16    `c:"ReportCount"`
	Reserved2       [5]uint16 `c:"Reserved2"`
	UnitsExp        uint32    `c:"UnitsExp"`
	Units           uint32    `c:"Units"`
	LogicalMin      int32     `c:"LogicalMin"`
	LogicalMax      int32     `c:"LogicalMax"`
	PhysicalMin     int32     `c:"PhysicalMin"`
	PhysicalMax     int32     `c:"PhysicalMax"`
	Union           [16]byte  `c:"Range"`
}

// UsageMin is the first usage of a range (Range.UsageMin), or the usage (NotRange.Usage), which is at the same place.
//
// Flutter: no counterpart
func (c *HidpValueCaps) UsageMin() uint16 { return unionU16(&c.Union, 0) }

// UsageMax is the last usage of a range (Range.UsageMax); it is meaningful when IsRange is set.
//
// Flutter: no counterpart
func (c *HidpValueCaps) UsageMax() uint16 { return unionU16(&c.Union, 2) }

// DataIndexMin is the data index of the first usage of a range (Range.DataIndexMin), or the data index (NotRange.DataIndex).
//
// Flutter: no counterpart
func (c *HidpValueCaps) DataIndexMin() uint16 { return unionU16(&c.Union, 12) }

// DataIndexMax is the data index of the last usage of a range (Range.DataIndexMax).
//
// Flutter: no counterpart
func (c *HidpValueCaps) DataIndexMax() uint16 { return unionU16(&c.Union, 14) }

// HidpData is the Go mirror of HIDP_DATA: a data index and the value (RawValue, or the On flag of a button in its low byte).
//
// Flutter: no counterpart
type HidpData struct {
	DataIndex uint16 `c:"DataIndex"`
	_         uint16
	RawValue  uint32 `c:"RawValue"`
}

// On says that the button of this item is down.
//
// Flutter: no counterpart
func (d HidpData) On() bool { return byte(d.RawValue) != 0 }

func unionU16(u *[16]byte, off int) uint16 { return uint16(u[off]) | uint16(u[off+1])<<8 }
