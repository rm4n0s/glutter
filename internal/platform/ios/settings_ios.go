//go:build ios

package ios

import (
	"reflect"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/darwin/dl"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
)

// This file reads the system settings from UIKit and watches them (SPEC §30.7). The conversion into platform.Settings is settings.go.

// The notifications that say a setting changed (the strings are the names of the constants).
var settingsNotifications = []string{
	"UIContentSizeCategoryDidChangeNotification",
	"UIAccessibilityVoiceOverStatusDidChangeNotification",
	"UIAccessibilitySwitchControlStatusDidChangeNotification",
	"UIAccessibilitySpeakScreenStatusDidChangeNotification",
	"UIAccessibilityReduceMotionStatusDidChangeNotification",
	"UIAccessibilityBoldTextStatusDidChangeNotification",
	"UIAccessibilityDarkerSystemColorsStatusDidChangeNotification",
	"UIAccessibilityReduceTransparencyStatusDidChangeNotification",
	"UIAccessibilityOnOffSwitchLabelsDidChangeNotification",
	"UIAccessibilityShouldDifferentiateWithoutColorDidChangeNotification",
	"NSCurrentLocaleDidChangeNotification",
}

// dateTemplateHourOnly is the template of a time with the hour only, whose format says whether the locale has an am/pm marker.
const dateTemplateHourOnly = "j"

// accessibilityFlags are the C functions of UIKit that say how the accessibility switches are.
type accessibilityFlags struct {
	voiceOver, switchControl, speakScreen, boldText, reduceMotion, darkerColors, reduceTransparency, onOffLabels, differentiate func() bool
}

// loadAccessibilityFlags binds the functions.
func loadAccessibilityFlags() (accessibilityFlags, error) {
	var f accessibilityFlags
	uikit, err := dl.OpenFramework("UIKit")
	if err != nil {
		return f, err
	}
	for fn, name := range map[*func() bool]string{
		&f.voiceOver:          "UIAccessibilityIsVoiceOverRunning",
		&f.switchControl:      "UIAccessibilityIsSwitchControlRunning",
		&f.speakScreen:        "UIAccessibilityIsSpeakScreenEnabled",
		&f.boldText:           "UIAccessibilityIsBoldTextEnabled",
		&f.reduceMotion:       "UIAccessibilityIsReduceMotionEnabled",
		&f.darkerColors:       "UIAccessibilityDarkerSystemColorsEnabled",
		&f.reduceTransparency: "UIAccessibilityIsReduceTransparencyEnabled",
		&f.onOffLabels:        "UIAccessibilityIsOnOffSwitchLabelsEnabled",
		&f.differentiate:      "UIAccessibilityShouldDifferentiateWithoutColor",
	} {
		if err := uikit.Bind(fn, name); err != nil {
			return f, err
		}
	}
	return f, nil
}

// registerSettingsObserver registers the class of the observer that the notification center calls when a setting changed.
func registerSettingsObserver() (*objcx.RegisteredClass, error) {
	return objcx.RegisterClass(objcx.ClassDef{
		Name: "SettingsObserver", Super: "NSObject",
		Methods: []objcx.Method{
			{Name: "settingsChanged:", Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
				if p, ok := objcx.LookupAs[*Platform](objcx.HandleOf(self)); ok && !p.closed {
					p.guard(p.settingsChanged)
				}
			}},
		},
	})
}

// observeSettings starts listening to the notifications of the settings.
func (p *Platform) observeSettings() {
	p.settingsObserver = p.s.settingsObserver.New(p.handle)
	center := class("NSNotificationCenter").Send(sel("defaultCenter"))
	for _, n := range settingsNotifications {
		name := objcx.NewString(n)
		center.Send(sel("addObserver:selector:name:object:"), p.settingsObserver, sel("settingsChanged:"), name, objc.ID(0))
		objcx.Release(name)
	}
}

// stopObservingSettings stops listening and releases the observer.
func (p *Platform) stopObservingSettings() {
	if p.settingsObserver == 0 {
		return
	}
	class("NSNotificationCenter").Send(sel("defaultCenter")).Send(sel("removeObserver:"), p.settingsObserver)
	objcx.Release(p.settingsObserver)
	p.settingsObserver = 0
}

// readSettings reads what the settings are made of from UIKit and Foundation.
func (p *Platform) readSettings() (in settingsInput) {
	objcx.WithPool(func() {
		f := p.s.access
		screen := class("UIScreen").Send(sel("mainScreen"))
		traits := send[objc.ID](screen, "traitCollection")
		in.Dark = send[int64](traits, "userInterfaceStyle") == userInterfaceStyleDark
		in.IncreaseContrast = f.darkerColors() || send[int64](traits, "accessibilityContrast") == accessibilityContrastHigh
		in.Category = objcx.String(send[objc.ID](class("UIApplication").Send(sel("sharedApplication")), "preferredContentSizeCategory"))
		in.BoldText, in.ReduceMotion, in.ReduceTransparency, in.OnOffLabels = f.boldText(), f.reduceMotion(), f.reduceTransparency(), f.onOffLabels()
		in.DifferentiateWithoutColor = f.differentiate()
		in.VoiceOver, in.SwitchControl, in.SpeakScreen = f.voiceOver(), f.switchControl(), f.speakScreen()

		tmpl := objcx.NewString(dateTemplateHourOnly)
		format := class("NSDateFormatter").Send(sel("dateFormatFromTemplate:options:locale:"), tmpl, uint64(0),
			class("NSLocale").Send(sel("currentLocale")))
		objcx.Release(tmpl)
		in.TimeFormat = objcx.String(format)

		langs := class("NSLocale").Send(sel("preferredLanguages"))
		for i := range objcx.ArrayCount(langs) {
			in.Languages = append(in.Languages, objcx.String(objcx.ArrayAt(langs, i)))
		}
	})
	return in
}

// The values of UIUserInterfaceStyle and UIAccessibilityContrast that the settings look for.
const (
	userInterfaceStyleDark    int64 = 2
	accessibilityContrastHigh int64 = 1
)

// scaleFontSize scales a font size the way Dynamic Type does: UIFontMetrics scales a large size less than a small one (the scale of body
// text is the textScaleOf of the category, and a title grows by less). It is called on the UI thread.
//
// Deviation: Flutter's iOS embedding gives a linear text scale from a table of body sizes (textScaleOf); Glutter gives that table as the
// text scale and the metrics of UIKit as the scale of a font size, as it does on Android 14 (SPEC §12.10).
func (p *Platform) scaleFontSize(size float64) float64 {
	var out float64
	objcx.WithPool(func() {
		metrics := class("UIFontMetrics").Send(sel("defaultMetrics"))
		out = send[float64](metrics, "scaledValueForValue:", size)
	})
	if out <= 0 {
		return size
	}
	return out
}

// currentSettings returns the settings, reading them the first time.
func (p *Platform) currentSettings() platform.Settings {
	if !p.settingsHave {
		p.settingsLast, p.settingsHave = p.readSettings(), true
	}
	s := settingsOf(p.settingsLast)
	s.ScaleFontSize = p.scaleFontSize
	return s
}

// settingsChanged rereads the settings and gives the handler the new ones when something differs from what it was told.
func (p *Platform) settingsChanged() {
	in := p.readSettings()
	if p.settingsHave && reflect.DeepEqual(in, p.settingsLast) {
		return
	}
	p.settingsLast, p.settingsHave = in, true
	if p.acc != nil {
		p.acc.statusChanged()
	}
	if p.handler != nil {
		s := settingsOf(in)
		s.ScaleFontSize = p.scaleFontSize
		p.handler.OnSettings(s)
	}
}

// Settings returns the system settings (SPEC §30.7): the dark mode of the screen's trait collection, the Dynamic Type size as the text
// scale, Bold Text, Reduce Motion, Increase Contrast, VoiceOver and Switch Control, the 24-hour clock and the preferred languages.
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterViewController.mm onUserSettingsChanged
func (p *Platform) Settings() platform.Settings { return p.currentSettings() }
