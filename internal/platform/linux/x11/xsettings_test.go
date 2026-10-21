//go:build linux

package x11

import (
	"encoding/binary"
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// xsettingsData builds the property of a settings daemon.
// byteOrder is a byte order that also appends.
type byteOrder interface {
	binary.ByteOrder
	binary.AppendByteOrder
}

type xsettingsData struct {
	order byteOrder
	buf   []byte
	count uint32
}

func newXSettingsData(order byteOrder) *xsettingsData {
	return &xsettingsData{order: order}
}

func (d *xsettingsData) u16(v uint16) { d.buf = d.order.AppendUint16(d.buf, v) }
func (d *xsettingsData) u32(v uint32) { d.buf = d.order.AppendUint32(d.buf, v) }
func (d *xsettingsData) name(typ byte, name string) {
	d.buf = append(d.buf, typ, 0)
	d.u16(uint16(len(name)))
	d.buf = append(d.buf, name...)
	for len(d.buf)%4 != 0 {
		d.buf = append(d.buf, 0)
	}
	d.u32(1) // the serial of the last change
	d.count++
}

func (d *xsettingsData) Int(name string, v int32) *xsettingsData {
	d.name(0, name)
	d.u32(uint32(v))
	return d
}

func (d *xsettingsData) String(name, v string) *xsettingsData {
	d.name(1, name)
	d.u32(uint32(len(v)))
	d.buf = append(d.buf, v...)
	for len(d.buf)%4 != 0 {
		d.buf = append(d.buf, 0)
	}
	return d
}

func (d *xsettingsData) Color(name string) *xsettingsData {
	d.name(2, name)
	for range 4 {
		d.u16(0xffff)
	}
	return d
}

// Bytes is the property: the header, then the settings.
func (d *xsettingsData) Bytes() []byte {
	order := byte(0)
	if d.order == binary.BigEndian {
		order = 1
	}
	head := []byte{order, 0, 0, 0}
	head = d.order.AppendUint32(head, 7) // the serial of the settings
	head = d.order.AppendUint32(head, d.count)
	return append(head, d.buf...)
}

func TestParseXSettings(t *testing.T) {
	for _, order := range []byteOrder{binary.LittleEndian, binary.BigEndian} {
		data := newXSettingsData(order).
			Int("Gtk/EnableAnimations", 0).
			String("Net/ThemeName", "Adwaita-dark").
			Color("Some/Color").
			String("Gtk/CursorThemeName", "Bibata"). // after a color: the position must be right
			Int("Gtk/CursorThemeSize", 36).
			String("Odd", "ab"). // padded to four bytes
			Int("Xft/Antialias", 1).Bytes()
		got := parseXSettings(data)
		want := map[string]xsetting{
			"Gtk/EnableAnimations": {Type: xsettingInt, Int: 0},
			"Net/ThemeName":        {Type: xsettingString, String: "Adwaita-dark"},
			"Some/Color":           {Type: xsettingColor},
			"Gtk/CursorThemeName":  {Type: xsettingString, String: "Bibata"},
			"Gtk/CursorThemeSize":  {Type: xsettingInt, Int: 36},
			"Odd":                  {Type: xsettingString, String: "ab"},
			"Xft/Antialias":        {Type: xsettingInt, Int: 1},
		}
		if len(got) != len(want) {
			t.Fatalf("%v: parsed %v, want %v", order, got, want)
		}
		for name, w := range want {
			if got[name] != w {
				t.Errorf("%v: %s = %+v, want %+v", order, name, got[name], w)
			}
		}
	}
}

func TestParseXSettingsToleratesBadData(t *testing.T) {
	good := newXSettingsData(binary.LittleEndian).Int("A", 1).String("B", "text").Int("C", 3).Bytes()
	if n := len(parseXSettings(good)); n != 3 {
		t.Fatalf("parsed %d settings of the good data", n)
	}
	// Every cut of the data gives a prefix of the settings, and never a panic.
	for i := range len(good) {
		got := parseXSettings(good[:i])
		if i < 12 && got != nil {
			t.Errorf("a cut at %d parsed %v", i, got)
		}
		if len(got) > 3 {
			t.Errorf("a cut at %d parsed %v", i, got)
		}
	}
	tests := []struct {
		name string
		data []byte
	}{
		{"nil", nil},
		{"unknown byte order", append([]byte{9, 0, 0, 0}, make([]byte, 8)...)},
		{"more settings than data", append([]byte{0, 0, 0, 0, 1, 0, 0, 0, 0xff, 0xff, 0xff, 0x7f}, 0, 0, 1, 0)},
		{"a string longer than the data", newXSettingsData(binary.LittleEndian).String("A", "x").Bytes()[:30]},
		{"unknown type", append(newXSettingsData(binary.LittleEndian).Bytes()[:8], 1, 0, 0, 0, 9, 0, 1, 0, 'a', 0, 0, 0, 1, 0, 0, 0)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { _ = parseXSettings(tt.data) })
	}
}

func TestApplyXSettings(t *testing.T) {
	base := platform.Settings{PlatformBrightness: ui.BrightnessLight, TextScaleFactor: 1, CursorSize: 24}
	tests := []struct {
		name string
		data *xsettingsData
		want func(*platform.Settings)
	}{
		{"nothing", newXSettingsData(binary.LittleEndian), func(*platform.Settings) {}},
		{"a dark theme", newXSettingsData(binary.LittleEndian).String("Net/ThemeName", "Yaru-Dark"),
			func(s *platform.Settings) { s.PlatformBrightness = ui.BrightnessDark }},
		{"a light theme", newXSettingsData(binary.LittleEndian).String("Net/ThemeName", "Adwaita"), func(*platform.Settings) {}},
		{"prefer dark", newXSettingsData(binary.LittleEndian).Int("Gtk/ApplicationPreferDarkTheme", 1),
			func(s *platform.Settings) { s.PlatformBrightness = ui.BrightnessDark }},
		{"cursor", newXSettingsData(binary.LittleEndian).String("Gtk/CursorThemeName", "Bibata").Int("Gtk/CursorThemeSize", 48),
			func(s *platform.Settings) { s.CursorTheme, s.CursorSize = "Bibata", 48 }},
		{"a cursor size of zero is ignored", newXSettingsData(binary.LittleEndian).Int("Gtk/CursorThemeSize", 0), func(*platform.Settings) {}},
		{"animations off", newXSettingsData(binary.LittleEndian).Int("Gtk/EnableAnimations", 0),
			func(s *platform.Settings) { s.DisableAnimations = true }},
		{"animations on", newXSettingsData(binary.LittleEndian).Int("Gtk/EnableAnimations", 1), func(*platform.Settings) {}},
		{"no antialiasing", newXSettingsData(binary.LittleEndian).Int("Xft/Antialias", 0),
			func(s *platform.Settings) { s.FontAntialiasing = platform.FontAntialiasingNone }},
		{"subpixel antialiasing", newXSettingsData(binary.LittleEndian).Int("Xft/Antialias", 1).String("Xft/RGBA", "rgb"),
			func(s *platform.Settings) { s.FontAntialiasing = platform.FontAntialiasingRGBA }},
		{"grayscale antialiasing", newXSettingsData(binary.LittleEndian).Int("Xft/Antialias", 1).String("Xft/RGBA", "none"),
			func(s *platform.Settings) { s.FontAntialiasing = platform.FontAntialiasingGrayscale }},
		{"hinting", newXSettingsData(binary.LittleEndian).String("Xft/HintStyle", "hintslight"),
			func(s *platform.Settings) { s.FontHinting = platform.FontHintingSlight }},
		{"a value of the wrong type is ignored", newXSettingsData(binary.LittleEndian).Int("Net/ThemeName", 1).String("Gtk/CursorThemeSize", "big"), func(*platform.Settings) {}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := base
			tt.want(&want)
			if got := applyXSettings(base, parseXSettings(tt.data.Bytes())); !reflect.DeepEqual(got, want) {
				t.Errorf("settings = %+v, want %+v", got, want)
			}
		})
	}
}

func TestXSettingsScale(t *testing.T) {
	two := parseXSettings(newXSettingsData(binary.LittleEndian).Int("Gdk/WindowScalingFactor", 2).Bytes())
	one := parseXSettings(newXSettingsData(binary.LittleEndian).Int("Gdk/WindowScalingFactor", 1).Bytes())
	if xsettingsScale(two) != 2 || xsettingsScale(one) != 0 || xsettingsScale(nil) != 0 {
		t.Errorf("scales: %v %v %v", xsettingsScale(two), xsettingsScale(one), xsettingsScale(nil))
	}
}
