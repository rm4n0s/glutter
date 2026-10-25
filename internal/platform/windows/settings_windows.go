//go:build windows

package windows

import (
	"reflect"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// The registry keys of the settings that no system call gives (SPEC §25.8).
const (
	accessibilityKey   = `Software\Microsoft\Accessibility`
	textScaleFactorVal = "TextScaleFactor"
)

// readSettings reads the settings from Windows: the registry, the system parameters, DWM and the
// user's locale. A value that cannot be read is left absent, and the settings have the default
// for it (a failure is logged once per read, at debug level: Wine has no accent color).
//
// Flutter: shell/platform/windows/settings_plugin.cc SettingsPlugin::SendSettings
func (p *Platform) readSettings() settingsInput {
	var in settingsInput
	var err error
	if in.AppsUseLightTheme, in.HasAppsUseLightTheme, err = win32.RegGetDWORD(win32.HKeyCurrentUser, personalizeKey, appsUseLightThemeValue); err != nil {
		p.log.Debug("the app mode of the system could not be read", "err", err)
	}
	if in.TextScalePercent, in.HasTextScale, err = win32.RegGetDWORD(win32.HKeyCurrentUser, accessibilityKey, textScaleFactorVal); err != nil {
		p.log.Debug("the text size of the system could not be read", "err", err)
	}
	if c, _, err := win32.DwmGetColorizationColor(); err == nil {
		in.Colorization, in.HasColorization = c, true
	} else {
		p.log.Debug("the accent color of the system could not be read", "err", err)
	}
	if flags, ok := win32.GetHighContrast(); ok {
		in.HighContrastFlags = flags
	}
	in.ClientAreaAnimation, in.HasClientAreaAnimation = win32.SystemParametersInfoBool(win32.SPIGetClientAreaAnimation)
	if in.Languages, err = win32.GetUserPreferredUILanguages(win32.MUILanguageName); err != nil {
		p.log.Debug("the languages of the user could not be read", "err", err)
	}
	if in.ShortTime, err = win32.GetLocaleInfoEx("", win32.LocaleSShortTime); err != nil {
		p.log.Debug("the time format of the user could not be read", "err", err)
	}
	in.CursorWidth = int(win32.GetSystemMetrics(win32.SMCxCursor))
	if smoothing, ok := win32.SystemParametersInfoBool(win32.SPIGetFontSmoothing); ok {
		in.FontSmoothing, in.HasFontSmoothing = smoothing, true
		if t, ok := win32.SystemParametersInfoUint(win32.SPIGetFontSmoothingType); ok {
			in.FontSmoothingType = t
		}
	}
	if m, ok := win32.GetNonClientMetrics(); ok {
		in.MessageFont = win32.UTF16ToString(m.MessageFont.FaceName[:])
		in.MessageFontHeight = m.MessageFont.Height
	}
	in.SystemDPI = win32.GetDpiForMonitor(win32.MonitorFromPoint(win32.Point{}, win32.MonitorDefaultToPrimary))
	in.CaretBlinkMillis = win32.GetCaretBlinkTime()
	in.DoubleClickMillis = win32.GetDoubleClickTime()
	return in
}

// Settings returns the settings of the system: read when it is first called, and again on every
// WM_SETTINGCHANGE, which arrives as Handler.OnSettings (SPEC §25.8).
//
// Flutter: shell/platform/windows/settings_plugin.cc SettingsPlugin::SendSettings
func (p *Platform) Settings() platform.Settings {
	if !p.settingsRead {
		p.settingsRead = true
		p.sys = p.readSettings()
		p.settings = settingsOf(p.sys)
		p.dark = darkTitleBar(p.sys.AppsUseLightTheme, p.sys.HasAppsUseLightTheme)
	}
	return p.settings
}

// settingChanged notes that a setting of the system may have changed (WM_SETTINGCHANGE reaches
// every top-level window, the service window and the app's windows, and says little about what
// changed). The settings are read again in the next turn of the loop, once, however many windows
// and messages there were (SPEC §25.8).
//
// Flutter: shell/platform/windows/settings_plugin.cc SettingsPlugin::StartWatching
func (p *Platform) settingChanged() {
	if p.settingsPending || p.closed {
		return
	}
	p.settingsPending = true
	p.Post(p.applySettings)
}

// applySettings reads the settings, makes the title bars follow the app mode, and tells the
// handler when something is different.
func (p *Platform) applySettings() {
	p.settingsPending = false
	if p.closed {
		return
	}
	in := p.readSettings()
	if p.settingsRead && reflect.DeepEqual(in, p.sys) {
		return
	}
	p.sys = in
	p.settingsRead = true
	p.settings = settingsOf(in)
	if dark := darkTitleBar(in.AppsUseLightTheme, in.HasAppsUseLightTheme); dark != p.dark {
		p.dark = dark
	}
	for _, w := range p.sortedWindows() {
		w.applyTitleBarTheme() // every window, live; the call is cheap and the same value is harmless
	}
	if p.handler != nil {
		p.handler.OnSettings(p.settings)
	}
}
