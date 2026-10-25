//go:build windows

package win32

import (
	"unsafe"
)

// The bindings of the system settings (SPEC §25.8): the system parameters, the colorization color
// of the desktop, the user's languages and locale, and the registry.

// GetCaretBlinkTime is the time the caret of the system stays on, and then off, in milliseconds;
// 0xFFFFFFFF (INFINITE) when the user turned the blinking off, and 0 when the call failed.
//
// Flutter: no counterpart
func GetCaretBlinkTime() uint32 {
	r, _, err := user32.call("GetCaretBlinkTime")
	if err != nil {
		return 0
	}
	return uint32(r)
}

// GetDoubleClickTime is the longest time between the two clicks of a double click, in milliseconds.
//
// Flutter: no counterpart
func GetDoubleClickTime() uint32 {
	r, _, err := user32.call("GetDoubleClickTime")
	if err != nil {
		return 0
	}
	return uint32(r)
}

// SystemParametersInfoBool reads a system parameter that is a BOOL (SPI_GET*). ok is false when the
// call failed.
//
// Flutter: no counterpart
func SystemParametersInfoBool(action uint32) (value, ok bool) {
	var v int32
	r, _, err := user32.call("SystemParametersInfoW", uintptr(action), 0, uintptr(unsafe.Pointer(&v)), 0)
	return v != 0, err == nil && r != 0
}

// GetHighContrast reads SPI_GETHIGHCONTRAST: the flags (HCFHighContrastOn and the others).
//
// Flutter: shell/platform/windows/system_utils.cc (the accessibility features)
func GetHighContrast() (flags uint32, ok bool) {
	hc := HighContrast{Size: uint32(unsafe.Sizeof(HighContrast{}))}
	r, _, err := user32.call("SystemParametersInfoW", SPIGetHighContrast, uintptr(hc.Size), uintptr(unsafe.Pointer(&hc)), 0)
	if err != nil || r == 0 {
		return 0, false
	}
	return hc.Flags, true
}

// GetNonClientMetrics reads SPI_GETNONCLIENTMETRICS: the fonts of the title bar, the menus, the
// status bars and the message boxes.
//
// Flutter: no counterpart
func GetNonClientMetrics() (NonClientMetrics, bool) {
	m := NonClientMetrics{Size: uint32(unsafe.Sizeof(NonClientMetrics{}))}
	r, _, err := user32.call("SystemParametersInfoW", SPIGetNonClientMetrics, uintptr(m.Size), uintptr(unsafe.Pointer(&m)), 0)
	if err != nil || r == 0 {
		return NonClientMetrics{}, false
	}
	return m, true
}

// DwmGetColorizationColor is the accent color of the desktop as 0xAARRGGBB, and whether it is
// opaque. It fails with COMCallFailed when composition is off or Windows lacks the function.
//
// Flutter: no counterpart
func DwmGetColorizationColor() (color uint32, opaque bool, err error) {
	var c uint32
	var blend int32
	r, _, err := dwmapi.call("DwmGetColorizationColor", uintptr(unsafe.Pointer(&c)), uintptr(unsafe.Pointer(&blend)))
	if err != nil {
		return 0, false, err
	}
	if herr := HResultError("DwmGetColorizationColor", FromResult(r)); herr != nil {
		return 0, false, herr
	}
	return c, blend != 0, nil
}

// GetUserPreferredUILanguages lists the languages of the user, most preferred first, in the
// format of the flags (MUILanguageName: BCP 47 names such as "en-US" and "zh-Hans-CN"). The list
// is empty when the user has none.
//
// Flutter: shell/platform/windows/system_utils.cc GetPreferredLanguageInfo
func GetUserPreferredUILanguages(flags uint32) ([]string, error) {
	var count, size uint32
	r, errno, err := kern32.call("GetUserPreferredUILanguages", uintptr(flags), uintptr(unsafe.Pointer(&count)), 0, uintptr(unsafe.Pointer(&size)))
	if err != nil {
		return nil, err
	}
	if r == 0 {
		return nil, Win32Error("GetUserPreferredUILanguages", uint32(errno))
	}
	if size == 0 {
		return nil, nil
	}
	buf := make([]uint16, size)
	r, errno, err = kern32.call("GetUserPreferredUILanguages", uintptr(flags), uintptr(unsafe.Pointer(&count)), uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	if err != nil {
		return nil, err
	}
	if r == 0 {
		return nil, Win32Error("GetUserPreferredUILanguages", uint32(errno))
	}
	return SplitMultiSZ(buf[:min(int(size), len(buf))]), nil
}

// SplitMultiSZ splits a list of strings that are each ended by a NUL, and the list by one more NUL.
//
// Flutter: no counterpart
func SplitMultiSZ(buf []uint16) []string {
	var out []string
	start := 0
	for i, c := range buf {
		if c != 0 {
			continue
		}
		if i == start { // the empty string that ends the list
			break
		}
		out = append(out, UTF16ToString(buf[start:i]))
		start = i + 1
	}
	return out
}

// GetLocaleInfoEx reads one piece of information (LOCALE_*) about the locale, as text. An empty
// locale name is the user's locale.
//
// Flutter: shell/platform/windows/system_utils.cc (the locale of the user)
func GetLocaleInfoEx(locale string, lctype uint32) (string, error) {
	var name uintptr
	if locale != "" {
		p, err := UTF16PtrFromString(locale)
		if err != nil {
			return "", err
		}
		name = uintptr(unsafe.Pointer(p))
	}
	n, errno, err := kern32.call("GetLocaleInfoEx", name, uintptr(lctype), 0, 0)
	if err != nil {
		return "", err
	}
	if n == 0 {
		return "", Win32Error("GetLocaleInfoEx", uint32(errno))
	}
	buf := make([]uint16, n)
	n, errno, err = kern32.call("GetLocaleInfoEx", name, uintptr(lctype), uintptr(unsafe.Pointer(&buf[0])), n)
	if err != nil {
		return "", err
	}
	if n == 0 {
		return "", Win32Error("GetLocaleInfoEx", uint32(errno))
	}
	return UTF16ToString(buf), nil
}

// RegGetString reads a text value of the registry (REG_SZ, or REG_EXPAND_SZ with its variables
// expanded). found is false when the key or the value is not there.
//
// Flutter: no counterpart
func RegGetString(root uintptr, subKey, value string) (s string, found bool, err error) {
	ps, err := UTF16PtrFromString(subKey)
	if err != nil {
		return "", false, err
	}
	pv, err := UTF16PtrFromString(value)
	if err != nil {
		return "", false, err
	}
	const flags = RRFRTRegSZ | RRFRTRegExpandSZ
	var size uint32
	r, _, err := advapi32.call("RegGetValueW", root, uintptr(unsafe.Pointer(ps)), uintptr(unsafe.Pointer(pv)), flags, 0, 0, uintptr(unsafe.Pointer(&size)))
	if err != nil {
		return "", false, err
	}
	if r != 0 || size == 0 {
		return "", false, nil
	}
	buf := make([]uint16, size/2+1)
	size = uint32(len(buf) * 2)
	r, _, err = advapi32.call("RegGetValueW", root, uintptr(unsafe.Pointer(ps)), uintptr(unsafe.Pointer(pv)), flags, 0,
		uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	if err != nil {
		return "", false, err
	}
	if r != 0 {
		return "", false, nil
	}
	return UTF16ToString(buf), true, nil
}

// RegValue is a text value of a registry key: its name and its data.
//
// Flutter: no counterpart
type RegValue struct {
	Name, Data string
}

// RegEnumStrings lists the text values (REG_SZ and REG_EXPAND_SZ) of a key, in the order the
// registry gives them. A key that is not there has none. It fails with Win32CallFailed when
// the key exists and cannot be read.
//
// Flutter: no counterpart
func RegEnumStrings(root uintptr, subKey string) ([]RegValue, error) {
	ps, err := UTF16PtrFromString(subKey)
	if err != nil {
		return nil, err
	}
	var key uintptr
	r, _, err := advapi32.call("RegOpenKeyExW", root, uintptr(unsafe.Pointer(ps)), 0, KeyRead, uintptr(unsafe.Pointer(&key)))
	if err != nil {
		return nil, err
	}
	const errorFileNotFound = 2
	if r == errorFileNotFound {
		return nil, nil
	}
	if r != 0 {
		return nil, Win32Error("RegOpenKeyExW", uint32(r))
	}
	defer func() { _, _, _ = advapi32.call("RegCloseKey", key) }()

	const (
		regSZ            = 1
		regExpandSZ      = 2
		errorNoMoreItems = 259
		errorMoreData    = 234
	)
	var out []RegValue
	name := make([]uint16, 16384) // the longest value name
	data := make([]uint16, 2048)
	for i := uint32(0); ; i++ {
		nameLen := uint32(len(name))
		dataLen := uint32(len(data) * 2)
		var typ uint32
		for {
			r, _, err = advapi32.call("RegEnumValueW", key, uintptr(i), uintptr(unsafe.Pointer(&name[0])), uintptr(unsafe.Pointer(&nameLen)), 0,
				uintptr(unsafe.Pointer(&typ)), uintptr(unsafe.Pointer(&data[0])), uintptr(unsafe.Pointer(&dataLen)))
			if err != nil {
				return out, err
			}
			if r == errorMoreData && int(dataLen) > len(data)*2 {
				data = make([]uint16, dataLen/2+1)
				nameLen = uint32(len(name))
				dataLen = uint32(len(data) * 2)
				continue
			}
			break
		}
		if r == errorNoMoreItems {
			return out, nil
		}
		if r != 0 {
			return out, Win32Error("RegEnumValueW", uint32(r))
		}
		if typ != regSZ && typ != regExpandSZ {
			continue
		}
		out = append(out, RegValue{Name: UTF16ToString(name[:nameLen]), Data: UTF16ToString(data[:min(int(dataLen/2), len(data))])})
	}
}

// ExpandEnvironmentStrings expands the %VARIABLES% of a string.
//
// Flutter: no counterpart
func ExpandEnvironmentStrings(s string) (string, error) {
	p, err := UTF16PtrFromString(s)
	if err != nil {
		return "", err
	}
	n, errno, err := kern32.call("ExpandEnvironmentStringsW", uintptr(unsafe.Pointer(p)), 0, 0)
	if err != nil {
		return "", err
	}
	if n == 0 {
		return "", Win32Error("ExpandEnvironmentStringsW", uint32(errno))
	}
	buf := make([]uint16, n)
	n, errno, err = kern32.call("ExpandEnvironmentStringsW", uintptr(unsafe.Pointer(p)), uintptr(unsafe.Pointer(&buf[0])), n)
	if err != nil {
		return "", err
	}
	if n == 0 {
		return "", Win32Error("ExpandEnvironmentStringsW", uint32(errno))
	}
	return UTF16ToString(buf), nil
}

// BroadcastSettingChange sends WM_SETTINGCHANGE to every top-level window, as Windows does when a
// user changes a setting; area is the lParam text ("ImmersiveColorSet" for the app mode,
// "WindowsThemeElement", "Environment"). The call waits for each window up to two seconds. The tests
// use it after they changed a value in the registry.
//
// Flutter: no counterpart
func BroadcastSettingChange(area string) error {
	p, err := UTF16PtrFromString(area)
	if err != nil {
		return err
	}
	const (
		hwndBroadcast   = 0xFFFF
		smtoAbortIfHung = 0x0002
	)
	var result uintptr
	r, errno, err := user32.call("SendMessageTimeoutW", hwndBroadcast, WMSettingChange, 0, uintptr(unsafe.Pointer(p)), smtoAbortIfHung, 2000, uintptr(unsafe.Pointer(&result)))
	if err != nil {
		return err
	}
	if r == 0 {
		return Win32Error("SendMessageTimeoutW", uint32(errno))
	}
	return nil
}
