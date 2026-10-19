package layout

import (
	"testing"

	"github.com/go-text/typesetting/font/opentype/tables"
	"github.com/rm4n0s/glutter/ui"
)

func TestColrTransforms(t *testing.T) {
	tests := []struct {
		name    string
		m       *ui.Matrix4
		in, out ui.Offset
	}{
		// Angles are in units of 180 degrees, counter-clockwise with y up.
		{"rotation by a quarter turn", rotation(0.5), ui.Offset{Dx: 1, Dy: 0}, ui.Offset{Dx: 0, Dy: 1}},
		{"rotation by a half turn", rotation(1), ui.Offset{Dx: 1, Dy: 2}, ui.Offset{Dx: -1, Dy: -2}},
		{"skew of x", skew(0.25, 0), ui.Offset{Dx: 0, Dy: 1}, ui.Offset{Dx: -1, Dy: 1}},
		{"skew of y", skew(0, 0.25), ui.Offset{Dx: 1, Dy: 0}, ui.Offset{Dx: 1, Dy: 1}},
		{"affine", affine(2, 0, 0, 3, 5, 7), ui.Offset{Dx: 1, Dy: 1}, ui.Offset{Dx: 7, Dy: 10}},
		{"around a point", around(rotation(1), 10, 10), ui.Offset{Dx: 12, Dy: 10}, ui.Offset{Dx: 8, Dy: 10}},
	}
	for _, tt := range tests {
		got := tt.m.TransformPoint(tt.in)
		if !near(got.Dx, tt.out.Dx) || !near(got.Dy, tt.out.Dy) {
			t.Errorf("%s: %v -> %v, want %v", tt.name, tt.in, got, tt.out)
		}
	}
}

func TestColrFixed214(t *testing.T) {
	for in, want := range map[tables.Fixed214]float64{0: 0, 1 << 14: 1, 1 << 13: 0.5, -(1 << 14): -1} {
		if got := fixed214(in); got != want {
			t.Errorf("%d = %v, want %v", in, got, want)
		}
	}
}

func TestColrTileModesAndBlends(t *testing.T) {
	for in, want := range map[tables.Extend]ui.TileMode{
		tables.ExtendPad: ui.TileModeClamp, tables.ExtendRepeat: ui.TileModeRepeated, tables.ExtendReflect: ui.TileModeMirror,
	} {
		if got := tileOf(in); got != want {
			t.Errorf("extend %d = %v, want %v", in, got, want)
		}
	}
	modes := map[tables.CompositeMode]ui.BlendMode{
		tables.CompositeClear: ui.BlendModeClear, tables.CompositeSrc: ui.BlendModeSrc, tables.CompositeDest: ui.BlendModeDst,
		tables.CompositeSrcOver: ui.BlendModeSrcOver, tables.CompositeDestOver: ui.BlendModeDstOver,
		tables.CompositeSrcIn: ui.BlendModeSrcIn, tables.CompositeDestIn: ui.BlendModeDstIn,
		tables.CompositeSrcOut: ui.BlendModeSrcOut, tables.CompositeDestOut: ui.BlendModeDstOut,
		tables.CompositeSrcAtop: ui.BlendModeSrcATop, tables.CompositeDestAtop: ui.BlendModeDstATop,
		tables.CompositeXor: ui.BlendModeXor, tables.CompositePlus: ui.BlendModePlus,
		tables.CompositeScreen: ui.BlendModeScreen, tables.CompositeOverlay: ui.BlendModeOverlay,
		tables.CompositeDarken: ui.BlendModeDarken, tables.CompositeLighten: ui.BlendModeLighten,
		tables.CompositeColorDodge: ui.BlendModeColorDodge, tables.CompositeColorBurn: ui.BlendModeColorBurn,
		tables.CompositeHardLight: ui.BlendModeHardLight, tables.CompositeSoftLight: ui.BlendModeSoftLight,
		tables.CompositeDifference: ui.BlendModeDifference, tables.CompositeExclusion: ui.BlendModeExclusion,
		tables.CompositeMultiply: ui.BlendModeMultiply, tables.CompositeHslHue: ui.BlendModeHue,
		tables.CompositeHslSaturation: ui.BlendModeSaturation, tables.CompositeHslColor: ui.BlendModeColor,
		tables.CompositeHslLuminosity: ui.BlendModeLuminosity,
	}
	for in, want := range modes {
		if got := compositeBlend(in); got != want {
			t.Errorf("composite %d = %v, want %v", in, got, want)
		}
	}
	if got := compositeBlend(tables.CompositeMode(200)); got != ui.BlendModeSrcOver {
		t.Errorf("an unknown mode is srcOver, got %v", got)
	}
}

func TestColrVarStops(t *testing.T) {
	in := []tables.VarColorStop{{StopOffset: 1 << 13, PaletteIndex: 3, Alpha: 1 << 14}}
	got := varStops(in)
	if len(got) != 1 || got[0].StopOffset != 1<<13 || got[0].PaletteIndex != 3 || got[0].Alpha != 1<<14 {
		t.Errorf("%+v", got)
	}
}

func TestColrColorLine(t *testing.T) {
	cp := &colrPainter{fg: ui.NewColor(0xFF102030)}
	// The stops come out sorted, with the offsets scaled to 0..1 around the original range.
	stops := []tables.ColorStop{
		{StopOffset: 3 << 12, PaletteIndex: 0xFFFF, Alpha: 1 << 14},
		{StopOffset: 1 << 12, PaletteIndex: 0xFFFF, Alpha: 1 << 13},
	}
	colors, offsets, lo, hi := cp.colorLine(stops)
	if len(colors) != 2 || !near(offsets[0], 0) || !near(offsets[1], 1) || !near(lo, 0.25) || !near(hi, 0.75) {
		t.Errorf("offsets %v lo %v hi %v", offsets, lo, hi)
	}
	if !near(float64(colors[0].A), 0.5) || !near(float64(colors[1].A), 1) {
		t.Errorf("the alpha of a stop scales the text color's: %v", colors)
	}
	if c, o, _, _ := cp.colorLine(nil); c != nil || o != nil {
		t.Error("no stops, nothing")
	}
	// One offset repeated: the last stop is at 1.
	_, o, _, _ := cp.colorLine([]tables.ColorStop{{StopOffset: 0, PaletteIndex: 0xFFFF}, {StopOffset: 0, PaletteIndex: 0xFFFF}})
	if o[0] != 0 || o[1] != 1 {
		t.Errorf("equal offsets %v", o)
	}
}

func TestColrColorPaletteAndForeground(t *testing.T) {
	cp := &colrPainter{fg: ui.NewColor(0x80FF0000)}
	if c := cp.color(0xFFFF, 0.5); !near(float64(c.A), 0x80/255.0*0.5) || c.R != 1 {
		t.Errorf("foreground at half alpha: %v", c)
	}

}
