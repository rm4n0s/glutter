package material_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func TestIcons(t *testing.T) {
	tests := []struct {
		name   string
		icon   widgets.IconData
		code   int
		mirror bool
	}{
		{"add", Icons.Add(), 0xe047, false},
		{"menu", Icons.Menu(), 0xe3dc, false},
		{"arrow back mirrors", Icons.ArrowBack(), 0xe092, true},
		{"arrow forward mirrors", Icons.ArrowForward(), 0xe09b, true},
		{"home outlined", Icons.HomeOutlined(), 0xf107, false},
		{"astral code point", Icons.Onetwothree(), 0xf04b5, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.icon.CodePoint != tt.code || tt.icon.MatchTextDirection != tt.mirror || tt.icon.FontFamily != MaterialIconsFamily {
				t.Errorf("icon = %+v, want code %#x, mirror %v", tt.icon, tt.code, tt.mirror)
			}
		})
	}
}

func TestIconsRenderFromTheBundledFont(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(materialApp(nil, widgets.NewColoredBox(white, widgets.NewCenter(widgets.NewIcon(Icons.Add(), widgets.IconOpts{Size: foundation.Some(48.0), Color: foundation.Some(ui.NewColor(0xFF000000))})))))
	// The glyph of "add" is a plus: its center and its arms are dark, the area
	// between the arms is not. (In the font of Google's repository, whose code
	// points differ from Flutter's, 0xe047 is a filled square.)
	dark := func(x, y int) bool { c := pixel(ts, x, y); return c.R < 128 }
	if !dark(400, 300) || !dark(400, 289) || !dark(400, 311) || !dark(389, 300) || !dark(411, 300) || dark(391, 291) {
		t.Error("the add icon is not drawn as a plus")
	}
}
