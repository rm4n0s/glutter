//go:build windows

package winetest

import "github.com/rm4n0s/glutter/internal/platform/windows/win32"

// UnderWine is whether the program runs under Wine: ntdll.dll of Wine exports wine_get_version, which
// Windows' ntdll does not. A test uses it to skip a check that only a real Windows can answer (or
// that Wine answers differently), and to say where it ran.
//
// Flutter: no counterpart
func UnderWine() bool {
	ntdll, err := win32.LoadSystemDLL("ntdll.dll")
	if err != nil {
		return false
	}
	_, ok := ntdll.OptionalProc("wine_get_version")
	return ok
}
