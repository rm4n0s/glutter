package win32

// The constants of XInput, Raw Input and the HID parser that the gamepad service of Windows uses (step 31.2).

// The buttons of XINPUT_GAMEPAD.wButtons. Guide is not in the headers: only the undocumented XInputGetStateEx reports it.
const (
	// XInputGamepadDPadUp is XINPUT_GAMEPAD_DPAD_UP.
	//
	// Flutter: no counterpart
	XInputGamepadDPadUp = 0x0001
	// XInputGamepadDPadDown is XINPUT_GAMEPAD_DPAD_DOWN.
	//
	// Flutter: no counterpart
	XInputGamepadDPadDown = 0x0002
	// XInputGamepadDPadLeft is XINPUT_GAMEPAD_DPAD_LEFT.
	//
	// Flutter: no counterpart
	XInputGamepadDPadLeft = 0x0004
	// XInputGamepadDPadRight is XINPUT_GAMEPAD_DPAD_RIGHT.
	//
	// Flutter: no counterpart
	XInputGamepadDPadRight = 0x0008
	// XInputGamepadStart is XINPUT_GAMEPAD_START.
	//
	// Flutter: no counterpart
	XInputGamepadStart = 0x0010
	// XInputGamepadBack is XINPUT_GAMEPAD_BACK.
	//
	// Flutter: no counterpart
	XInputGamepadBack = 0x0020
	// XInputGamepadLeftThumb is XINPUT_GAMEPAD_LEFT_THUMB.
	//
	// Flutter: no counterpart
	XInputGamepadLeftThumb = 0x0040
	// XInputGamepadRightThumb is XINPUT_GAMEPAD_RIGHT_THUMB.
	//
	// Flutter: no counterpart
	XInputGamepadRightThumb = 0x0080
	// XInputGamepadLeftShoulder is XINPUT_GAMEPAD_LEFT_SHOULDER.
	//
	// Flutter: no counterpart
	XInputGamepadLeftShoulder = 0x0100
	// XInputGamepadRightShoulder is XINPUT_GAMEPAD_RIGHT_SHOULDER.
	//
	// Flutter: no counterpart
	XInputGamepadRightShoulder = 0x0200
	// XInputGamepadGuide is the Guide button, which XInputGetStateEx reports and XINPUT_GAMEPAD has no name for.
	//
	// Flutter: no counterpart
	XInputGamepadGuide = 0x0400
	// XInputGamepadA is XINPUT_GAMEPAD_A.
	//
	// Flutter: no counterpart
	XInputGamepadA = 0x1000
	// XInputGamepadB is XINPUT_GAMEPAD_B.
	//
	// Flutter: no counterpart
	XInputGamepadB = 0x2000
	// XInputGamepadX is XINPUT_GAMEPAD_X.
	//
	// Flutter: no counterpart
	XInputGamepadX = 0x4000
	// XInputGamepadY is XINPUT_GAMEPAD_Y.
	//
	// Flutter: no counterpart
	XInputGamepadY = 0x8000
)

// Other constants of XInput.
const (
	// XUserMaxCount is XUSER_MAX_COUNT, the number of pads XInput serves.
	//
	// Flutter: no counterpart
	XUserMaxCount = 4
	// XInputFlagGamepad is XINPUT_FLAG_GAMEPAD, the flag of XInputGetCapabilities that asks for gamepads only.
	//
	// Flutter: no counterpart
	XInputFlagGamepad = 0x00000001
	// XInputDevTypeGamepad is XINPUT_DEVTYPE_GAMEPAD.
	//
	// Flutter: no counterpart
	XInputDevTypeGamepad = 0x01
	// XInputDevSubTypeGamepad is XINPUT_DEVSUBTYPE_GAMEPAD.
	//
	// Flutter: no counterpart
	XInputDevSubTypeGamepad = 0x01
	// XInputCapsWireless is XINPUT_CAPS_WIRELESS.
	//
	// Flutter: no counterpart
	XInputCapsWireless = 0x0002
	// XInputBatteryTypeDisconnected is BATTERY_TYPE_DISCONNECTED.
	//
	// Flutter: no counterpart
	XInputBatteryTypeDisconnected = 0x00
	// XInputBatteryTypeWired is BATTERY_TYPE_WIRED.
	//
	// Flutter: no counterpart
	XInputBatteryTypeWired = 0x01
	// XInputBatteryTypeUnknown is BATTERY_TYPE_UNKNOWN.
	//
	// Flutter: no counterpart
	XInputBatteryTypeUnknown = 0xFF
	// XInputBatteryLevelEmpty is BATTERY_LEVEL_EMPTY.
	//
	// Flutter: no counterpart
	XInputBatteryLevelEmpty = 0x00
	// XInputBatteryLevelLow is BATTERY_LEVEL_LOW.
	//
	// Flutter: no counterpart
	XInputBatteryLevelLow = 0x01
	// XInputBatteryLevelMedium is BATTERY_LEVEL_MEDIUM.
	//
	// Flutter: no counterpart
	XInputBatteryLevelMedium = 0x02
	// XInputBatteryLevelFull is BATTERY_LEVEL_FULL.
	//
	// Flutter: no counterpart
	XInputBatteryLevelFull = 0x03
	// ErrorSuccessCode is ERROR_SUCCESS, which XInput returns for a pad that is there.
	//
	// Flutter: no counterpart
	ErrorSuccessCode = 0
	// ErrorDeviceNotConnected is ERROR_DEVICE_NOT_CONNECTED, which XInput returns for an empty slot.
	//
	// Flutter: no counterpart
	ErrorDeviceNotConnected = 1167
)

// The constants of Raw Input for the pads and of the HID parser.
const (
	// WMInputDeviceChange is WM_INPUT_DEVICE_CHANGE, the message of a device that arrives or leaves.
	//
	// Flutter: no counterpart
	WMInputDeviceChange = 0x00FE
	// GIDCArrival is GIDC_ARRIVAL, the wParam of WM_INPUT_DEVICE_CHANGE for a device that arrived.
	//
	// Flutter: no counterpart
	GIDCArrival = 1
	// GIDCRemoval is GIDC_REMOVAL, the wParam of WM_INPUT_DEVICE_CHANGE for a device that left.
	//
	// Flutter: no counterpart
	GIDCRemoval = 2
	// RIDEVInputSink is RIDEV_INPUTSINK: the window gets the input even when it is not in front.
	//
	// Flutter: no counterpart
	RIDEVInputSink = 0x00000100
	// RIDEVDevNotify is RIDEV_DEVNOTIFY: the window gets WM_INPUT_DEVICE_CHANGE.
	//
	// Flutter: no counterpart
	RIDEVDevNotify = 0x00002000
	// RIDIPreparsedData is RIDI_PREPARSEDDATA, the command of GetRawInputDeviceInfo that gives the parsed report descriptor.
	//
	// Flutter: no counterpart
	RIDIPreparsedData = 0x20000005
	// RIDIDeviceName is RIDI_DEVICENAME, the command that gives the path of the device.
	//
	// Flutter: no counterpart
	RIDIDeviceName = 0x20000007
	// RIDIDeviceInfo is RIDI_DEVICEINFO, the command that gives a RID_DEVICE_INFO.
	//
	// Flutter: no counterpart
	RIDIDeviceInfo = 0x2000000b
	// RIMTypeHID is RIM_TYPEHID, the type of a raw input that is a HID report.
	//
	// Flutter: no counterpart
	RIMTypeHID = 2
	// HIDUsageGenericJoystick is HID_USAGE_GENERIC_JOYSTICK.
	//
	// Flutter: no counterpart
	HIDUsageGenericJoystick = 0x04
	// HIDUsageGenericGamepad is HID_USAGE_GENERIC_GAMEPAD.
	//
	// Flutter: no counterpart
	HIDUsageGenericGamepad = 0x05
	// HIDUsageGenericMultiAxis is HID_USAGE_GENERIC_MULTI_AXIS_CONTROLLER.
	//
	// Flutter: no counterpart
	HIDUsageGenericMultiAxis = 0x08
	// HIDUsageGenericHatSwitch is HID_USAGE_GENERIC_HATSWITCH.
	//
	// Flutter: no counterpart
	HIDUsageGenericHatSwitch = 0x39
	// HIDUsageGenericZ is HID_USAGE_GENERIC_Z.
	//
	// Flutter: no counterpart
	HIDUsageGenericZ = 0x32
	// HIDUsagePageButton is HID_USAGE_PAGE_BUTTON.
	//
	// Flutter: no counterpart
	HIDUsagePageButton = 0x09
	// HidPInput is HidP_Input, the report type of the input reports.
	//
	// Flutter: no counterpart
	HidPInput = 0
	// HidPStatusSuccess is HIDP_STATUS_SUCCESS.
	//
	// Flutter: no counterpart
	HidPStatusSuccess = 0x00110000
	// GenericRead is GENERIC_READ.
	//
	// Flutter: no counterpart
	GenericRead = 0x80000000
	// FileShareRead is FILE_SHARE_READ.
	//
	// Flutter: no counterpart
	FileShareRead = 0x00000001
	// FileShareWrite is FILE_SHARE_WRITE.
	//
	// Flutter: no counterpart
	FileShareWrite = 0x00000002
	// OpenExisting is OPEN_EXISTING.
	//
	// Flutter: no counterpart
	OpenExisting = 3
)
