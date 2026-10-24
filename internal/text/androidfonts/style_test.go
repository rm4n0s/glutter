package androidfonts

import "testing"

func TestMatchStyle(t *testing.T) {
	roboto := []face{
		{path: "thin", weight: 100}, {path: "thin-i", weight: 100, italic: true},
		{path: "light", weight: 300}, {path: "regular", weight: 400}, {path: "italic", weight: 400, italic: true},
		{path: "medium", weight: 500}, {path: "bold", weight: 700}, {path: "bold-i", weight: 700, italic: true},
		{path: "black", weight: 900},
	}
	tests := []struct {
		name   string
		fonts  []face
		weight int
		italic bool
		want   string
	}{
		{"regular", roboto, 400, false, "regular"},
		{"bold", roboto, 700, false, "bold"},
		{"bold italic", roboto, 700, true, "bold-i"},
		{"italic", roboto, 400, true, "italic"},
		{"medium", roboto, 500, false, "medium"},
		{"semibold goes to the heavier", roboto, 600, false, "bold"},
		{"800 goes to the heavier", roboto, 800, false, "black"},
		{"1000 is the heaviest", roboto, 1000, false, "black"},
		{"200 goes to the lighter", roboto, 200, false, "thin"},
		{"350 goes to the lighter", roboto, 350, false, "light"},
		{"100", roboto, 100, false, "thin"},
		{"450 goes to 400", roboto, 450, false, "regular"},
		{"the slant counts more than the weight", roboto, 300, true, "thin-i"},
		{"500 with no 500 takes 400", []face{{path: "r", weight: 400}, {path: "b", weight: 700}}, 500, false, "r"},
		{"400 with no 400 takes 500", []face{{path: "m", weight: 500}, {path: "b", weight: 700}}, 400, false, "m"},
		{"400 with only lighter and heavier takes the lighter", []face{{path: "l", weight: 300}, {path: "b", weight: 900}}, 400, false, "l"},
		{"700 with only lighter takes the heaviest lighter", []face{{path: "r", weight: 400}, {path: "l", weight: 300}}, 700, false, "r"},
		{"italic wanted, only upright", []face{{path: "r", weight: 400}, {path: "b", weight: 700}}, 700, true, "b"},
		{"upright wanted, only italic", []face{{path: "i", weight: 400, italic: true}}, 400, false, "i"},
		{"the first of equal fonts", []face{{path: "a", weight: 400}, {path: "b", weight: 400}}, 400, false, "a"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchStyle(tt.fonts, tt.weight, tt.italic)
			if got == nil || got.path != tt.want {
				t.Errorf("matchStyle(weight %d, italic %t) = %v, want %s", tt.weight, tt.italic, got, tt.want)
			}
		})
	}
	if got := matchStyle(nil, 400, false); got != nil {
		t.Errorf("matchStyle of no fonts = %v, want nil", got)
	}
}
