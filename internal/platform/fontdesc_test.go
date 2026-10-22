package platform

import "testing"

func TestParseFontDescription(t *testing.T) {
	tests := []struct {
		desc   string
		family string
		size   float64
		ok     bool
	}{
		{"Cantarell 11", "Cantarell", 11, true},
		{"Adwaita Mono 11", "Adwaita Mono", 11, true},
		{"Source Code Pro Semi-Bold 10", "Source Code Pro", 10, true},
		{"Noto Sans Mono Bold Italic 12", "Noto Sans Mono", 12, true},
		{"DejaVu Sans Mono 10.5", "DejaVu Sans Mono", 10.5, true},
		{"Monospace 10", "Monospace", 10, true},
		{"Ubuntu Light 9", "Ubuntu", 9, true},
		{"Inter Display Condensed Black 14", "Inter Display", 14, true},
		{"Roboto 12px", "Roboto", 9, true},
		{"Roboto 11 @wght=500", "Roboto", 11, true},
		{"Noto Sans, Sans 11", "Noto Sans", 11, true},
		{"'Fira Code' 11", "Fira Code", 11, true},
		{"  Cantarell   11  ", "Cantarell", 11, true},
		{"Cantarell", "Cantarell", 0, true},
		{"Bold", "Bold", 0, true},
		{"Sans Bold", "Sans", 0, true},
		{"Noto Sans Regular", "Noto Sans", 0, true},
		{"Font 2000 11", "Font 2000", 11, true},
		{"11", "11", 0, true},
		{"", "", 0, false},
		{"   ", "", 0, false},
		{"@wght=500", "", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			family, size, ok := ParseFontDescription(tt.desc)
			if family != tt.family || size != tt.size || ok != tt.ok {
				t.Errorf("ParseFontDescription(%q) = %q, %v, %v; want %q, %v, %v", tt.desc, family, size, ok, tt.family, tt.size, tt.ok)
			}
		})
	}
}
