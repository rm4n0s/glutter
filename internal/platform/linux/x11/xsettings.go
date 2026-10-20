package x11

import (
	"encoding/binary"
	"log/slog"
	"strings"

	"github.com/rm4n0s/glutter/internal/dbus"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
	"github.com/rm4n0s/glutter/ui"
)

// XSETTINGS (https://specifications.freedesktop.org/xsettings-spec/): a settings
// daemon, such as GNOME's or KDE's, owns the selection _XSETTINGS_S<screen> and
// keeps the settings in the property _XSETTINGS_SETTINGS of the owner's window. The
// settings are the fallback when the Settings portal is missing (SPEC §8.8).

// xsettingType is the type of the value of a setting.
type xsettingType uint8

const (
	xsettingInt xsettingType = iota
	xsettingString
	xsettingColor
)

// xsetting is one setting; Int is the value of an integer setting, String of a
// string setting.
type xsetting struct {
	Type   xsettingType
	Int    int32
	String string
}

// parseXSettings reads the property of the settings daemon. It returns nil for data
// that is not in the format; a setting that is cut off ends the list.
func parseXSettings(data []byte) map[string]xsetting {
	if len(data) < 12 {
		return nil
	}
	var order binary.ByteOrder
	switch data[0] {
	case 0:
		order = binary.LittleEndian
	case 1:
		order = binary.BigEndian
	default:
		return nil
	}
	count := order.Uint32(data[8:])
	out := map[string]xsetting{}
	pos := 12
	pad := func(n int) int { return (n + 3) &^ 3 }
	for range count {
		if pos+4 > len(data) {
			break
		}
		typ := xsettingType(data[pos])
		nameLen := int(order.Uint16(data[pos+2:]))
		pos += 4
		if pos+pad(nameLen)+4 > len(data) {
			break
		}
		name := string(data[pos : pos+nameLen])
		pos += pad(nameLen) + 4 // the name and the serial of the last change
		var s xsetting
		s.Type = typ
		switch typ {
		case xsettingInt:
			if pos+4 > len(data) {
				return out
			}
			s.Int = int32(order.Uint32(data[pos:]))
			pos += 4
		case xsettingString:
			if pos+4 > len(data) {
				return out
			}
			n := int(order.Uint32(data[pos:]))
			pos += 4
			if n < 0 || n > len(data) || pos+pad(n) > len(data) {
				return out
			}
			s.String = string(data[pos : pos+n])
			pos += pad(n)
		case xsettingColor:
			pos += 8
			if pos > len(data) {
				return out
			}
		default:
			return out // unknown type: the size is unknown
		}
		out[name] = s
	}
	return out
}

// applyXSettings sets the fields of s that the settings of the daemon say. The
// names are those of GTK, which every daemon fills in.
func applyXSettings(s platform.Settings, xs map[string]xsetting) platform.Settings {
	if v, ok := xs["Net/ThemeName"]; ok && v.Type == xsettingString && strings.Contains(strings.ToLower(v.String), "dark") {
		s.PlatformBrightness = ui.BrightnessDark
	}
	if v, ok := xs["Gtk/ApplicationPreferDarkTheme"]; ok && v.Type == xsettingInt && v.Int == 1 {
		s.PlatformBrightness = ui.BrightnessDark
	}
	if v, ok := xs["Gtk/CursorThemeName"]; ok && v.Type == xsettingString && v.String != "" {
		s.CursorTheme = v.String
	}
	if v, ok := xs["Gtk/CursorThemeSize"]; ok && v.Type == xsettingInt && v.Int > 0 {
		s.CursorSize = int(v.Int)
	}
	if v, ok := xs["Gtk/EnableAnimations"]; ok && v.Type == xsettingInt {
		s.DisableAnimations = v.Int == 0
	}
	if v, ok := xs["Xft/Antialias"]; ok && v.Type == xsettingInt {
		if v.Int == 0 {
			s.FontAntialiasing = platform.FontAntialiasingNone
		} else {
			s.FontAntialiasing = platform.FontAntialiasingGrayscale
			if rgba, ok := xs["Xft/RGBA"]; ok && rgba.Type == xsettingString && rgba.String != "" && rgba.String != "none" {
				s.FontAntialiasing = platform.FontAntialiasingRGBA
			}
		}
	}
	if v, ok := xs["Xft/HintStyle"]; ok && v.Type == xsettingString {
		switch v.String {
		case "hintnone":
			s.FontHinting = platform.FontHintingNone
		case "hintslight":
			s.FontHinting = platform.FontHintingSlight
		case "hintmedium":
			s.FontHinting = platform.FontHintingMedium
		case "hintfull":
			s.FontHinting = platform.FontHintingFull
		}
	}
	return s
}

// xsettingsScale is the integer scale that the daemon asks for (Gdk/WindowScalingFactor),
// or 0.
func xsettingsScale(xs map[string]xsetting) float64 {
	if v, ok := xs["Gdk/WindowScalingFactor"]; ok && v.Type == xsettingInt && v.Int > 1 {
		return float64(v.Int)
	}
	return 0
}

// xsettingsState is what the platform knows of the daemon.
type xsettingsState struct {
	owner    uint64 // the window that owns the selection; 0 when nobody does
	settings map[string]xsetting
}

// readProperty reads a property of a window as bytes, up to maxWords 32-bit words
// (the data is copied out of the memory of Xlib); nil when it is missing.
func (p *Platform) readProperty(window, property, typ uint64, maxWords int64) []byte {
	mem := dl.Calloc(1, 8+4+8+8+8)
	defer dl.Free(mem)
	actualType, actualFormat, nitems, bytesAfter, prop := mem, mem+8, mem+16, mem+24, mem+32
	status := p.x.getWindowProperty(p.display, window, property, 0, maxWords, 0, typ,
		actualType, actualFormat, nitems, bytesAfter, prop)
	data := *view[uintptr](prop)
	if data != 0 {
		defer p.x.free(data)
	}
	if status != 0 || data == 0 || *view[int32](actualFormat) != 8 {
		return nil
	}
	return append([]byte(nil), dl.Bytes(data, int(*view[uint64](nitems)))...)
}

// findXSettings looks for the owner of the settings selection and reads its
// settings. It follows the owner's window for changes.
func (p *Platform) findXSettings() {
	if p.xs == nil {
		p.xs = &xsettingsState{}
	}
	owner := p.x.getSelectionOwner(p.display, p.atoms.xsettingsSelection)
	if owner != p.xs.owner && owner != 0 {
		p.x.selectInput(p.display, owner, propertyChangeMask|structureNotifyMask)
	}
	p.xs.owner = owner
	p.xs.settings = nil
	if owner != 0 {
		p.xs.settings = parseXSettings(p.readProperty(owner, p.atoms.xsettingsProperty, p.atoms.xsettingsProperty, 64*1024))
	}
	p.x.flush(p.display)
}

// xsettingsChanged follows a change of the daemon's settings, or of the daemon: the
// scale of the monitors and the settings of the app are worked out again.
func (p *Platform) xsettingsChanged() {
	p.findXSettings()
	p.monitorsChanged()
	if p.settings != nil && p.xsettingsInUse {
		p.settings.Set(applyXSettings(dbus.FallbackSettings(p.getenv), p.xs.settings))
	}
}

// withXSettings is the fallback settings with what the daemon says, and starts
// following the daemon. It is used when the portal is missing.
func (p *Platform) withXSettings(s platform.Settings) platform.Settings {
	p.xsettingsInUse = true
	if p.xs == nil || p.xs.owner == 0 {
		p.findXSettings()
	}
	if len(p.xs.settings) == 0 {
		slog.Debug("no XSETTINGS daemon: the settings come from the environment")
		return s
	}
	return applyXSettings(s, p.xs.settings)
}
