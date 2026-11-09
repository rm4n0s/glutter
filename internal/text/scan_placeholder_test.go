package text

import "testing"

func TestIsPlaceholderFont(t *testing.T) {
	tests := []struct {
		name string
		file string
		want bool
	}{
		{"Apple's Last Resort font", "/System/Library/Fonts/LastResort.otf", true},
		{"its name in another case", "/x/lastresort.ttf", true},
		{"a font whose name only starts like it", "/x/LastResortSerif.ttf", false},
		{"an ordinary font", "/usr/share/fonts/DejaVuSans.ttf", false},
		{"no file", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isPlaceholderFont(tt.file); got != tt.want {
				t.Errorf("isPlaceholderFont(%q) = %v, want %v", tt.file, got, tt.want)
			}
		})
	}
}
