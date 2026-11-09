package layout

import (
	"bytes"
	"image/color"
	"math"
	"testing"

	"github.com/go-text/typesetting/font"
	"github.com/rm4n0s/glutter/internal/goldens"
	"github.com/rm4n0s/glutter/ui"
)

// This file builds a tiny sfnt font with an sbix table, the color bitmap font of Apple (Apple Color Emoji),
// from the base tables of the CBDT test font, so that the tests need no font file and no license. Its glyphs:
//
//	gid 1  'A'        a blue square
//	gid 2  U+1F600    a red square
//	gid 3  U+1F601    a 'dupe' of gid 2, which sbix uses for glyphs that share an image
//
// Each strike has its own pixels per em and its own marker square, as the strikes of the CBDT font have.

// sbixStrike describes a strike of the test font: the glyphs are squares of size pixels, placed
// originX, originY pixels from the origin of the glyph to the left and bottom edges of the bitmap.
type sbixStrike struct {
	ppem, size         int
	originX, originY   int
	marker             color.NRGBA
	graphicType        string // "png " when empty; "tiff" makes a strike that is not drawn
	noGlyph3, noGlyph1 bool
}

// sbixStrikes are the two strikes of the default sbix test font: 160 is Apple Color Emoji's biggest, 32 a
// small one. The origins scale with the ppem, so that every strike places the glyph the same.
var sbixStrikes = []sbixStrike{
	{ppem: 160, size: 160, originX: 10, originY: -20, marker: color.NRGBA{A: 255}},
	{ppem: 32, size: 32, originX: 2, originY: -4, marker: color.NRGBA{R: 255, G: 255, B: 255, A: 255}},
}

// buildSbixFont makes the font with the strikes.
func buildSbixFont(strikes []sbixStrike) []byte {
	const numGlyphs = 4
	var sbix be
	sbix.u16(1) // version
	sbix.u16(1) // flags: draw the outlines too (there are none)
	sbix.u32(len(strikes))
	base := 8 + 4*len(strikes)
	var data be
	for _, s := range strikes {
		sbix.u32(base + len(data))
		graphic := s.graphicType
		if graphic == "" {
			graphic = "png "
		}
		var glyphs be // the glyph data, after the offsets
		offsets := make([]int, numGlyphs+1)
		put := func(gid int, payload []byte, kind string) {
			offsets[gid] = len(glyphs)
			glyphs.u16(s.originX)
			glyphs.u16(s.originY)
			glyphs.raw([]byte(kind))
			glyphs.raw(payload)
		}
		var dupe be
		dupe.u16(2)
		blue, red := testGlyphPNG(s.size, testBlue, s.marker), testGlyphPNG(s.size, testRed, s.marker)
		offsets[0] = 0
		if !s.noGlyph1 {
			put(1, blue, graphic)
		} else {
			offsets[1] = len(glyphs)
		}
		put(2, red, graphic)
		if !s.noGlyph3 {
			put(3, dupe, "dupe")
		} else {
			offsets[3] = len(glyphs)
		}
		offsets[numGlyphs] = len(glyphs)
		// A glyph with no data has the same offset as the next one.
		for gid := numGlyphs - 1; gid >= 0; gid-- {
			if gid == 0 || (gid == 1 && s.noGlyph1) || (gid == 3 && s.noGlyph3) {
				offsets[gid] = offsets[gid+1]
			}
		}
		offsets[0] = offsets[1]
		var strike be
		strike.u16(s.ppem)
		strike.u16(72) // ppi
		headerLen := 4 + 4*(numGlyphs+1)
		for _, o := range offsets {
			strike.u32(headerLen + o)
		}
		strike.raw(glyphs)
		data.raw(strike)
	}
	sbix.raw(data)

	tables := testBaseTables()
	tables["sbix"] = sbix
	return buildSfnt(tables)
}

func sbixFace(t *testing.T, strikes []sbixStrike) *font.Face {
	t.Helper()
	face, err := font.ParseTTF(bytes.NewReader(buildSbixFont(strikes)))
	if err != nil {
		t.Fatalf("the sbix test font does not parse: %v", err)
	}
	return face
}

func TestSbixFontLoadsInGoText(t *testing.T) {
	face := sbixFace(t, sbixStrikes)
	sizes := face.BitmapSizes()
	if len(sizes) != 2 || sizes[0].YPpem != 160 || sizes[1].YPpem != 32 {
		t.Fatalf("strikes = %+v, want ppem 160 and 32", sizes)
	}
	for _, tt := range []struct {
		r   rune
		gid font.GID
	}{{'A', 1}, {0x1F600, 2}, {0x1F601, 3}} {
		gid, ok := face.NominalGlyph(tt.r)
		if !ok || gid != tt.gid {
			t.Fatalf("%U: gid %d %v, want %d", tt.r, gid, ok, tt.gid)
		}
		bm, ext, ok := bitmapOf(face, gid, sizes[0])
		if !ok || bm.Format != font.PNG || bm.Width != 160 || bm.Height != 160 {
			t.Errorf("%U: bitmap %+v %v", tt.r, bm.Format, ok)
		}
		if ext.Height >= 0 || ext.Width <= 0 {
			t.Errorf("%U: extents %+v", tt.r, ext)
		}
	}
	if x, y := face.Ppem(); x != 0 || y != 0 {
		t.Errorf("the face keeps ppem %d, %d after the bitmaps are read, want 0, 0", x, y)
	}
}

func TestSbixGlyphPicksTheStrikeOfTheSizeAndFollowsDupe(t *testing.T) {
	face := sbixFace(t, sbixStrikes)
	e := bundledEngine(t)
	tests := []struct {
		size     float64
		wantSize int
		marker   [4]uint8
	}{
		{1, 32, [4]uint8{255, 255, 255, 255}},
		{32, 32, [4]uint8{255, 255, 255, 255}},
		{32.1, 160, [4]uint8{0, 0, 0, 255}},
		{200, 160, [4]uint8{0, 0, 0, 255}},
	}
	for _, tt := range tests {
		// Glyph 3 is a dupe of glyph 2: both draw the red square.
		for _, gid := range []font.GID{2, 3} {
			op := imageRectOf(t, e.bitmapGlyph(face, gid, tt.size))
			if op.Image.Width() != tt.wantSize {
				t.Errorf("size %v, glyph %d: image %d, want %d", tt.size, gid, op.Image.Width(), tt.wantSize)
				continue
			}
			m := tt.wantSize/8 + 2
			if got := pixelAt(op.Image, m, m); got != tt.marker {
				t.Errorf("size %v, glyph %d: marker %v, want %v", tt.size, gid, got, tt.marker)
			}
			if got := pixelAt(op.Image, tt.wantSize/2, tt.wantSize/2); got != [4]uint8{255, 0, 0, 255} {
				t.Errorf("size %v, glyph %d: body %v, want red", tt.size, gid, got)
			}
		}
	}
	if x, y := face.Ppem(); x != 0 || y != 0 {
		t.Errorf("the face keeps ppem %d, %d, want 0, 0", x, y)
	}
}

func TestSbixGlyphPlacement(t *testing.T) {
	// The bottom edge of the bitmap is originY pixels above the baseline and its left edge originX pixels
	// right of the glyph's origin: a pixel of a strike is upem/ppem units.
	near := func(a, b float64) bool { return math.Abs(a-b) < 0.01 }
	tests := []struct {
		name string
		size float64
		want ui.Rect
	}{
		{"strike 32", 20, ui.Rect{Left: 2 * 64, Top: -28 * 64, Right: 34 * 64, Bottom: 4 * 64}},
		{"strike 160", 100, ui.Rect{Left: 10 * 12.8, Top: -140 * 12.8, Right: 170 * 12.8, Bottom: 20 * 12.8}},
	}
	face := sbixFace(t, sbixStrikes)
	e := bundledEngine(t)
	for _, tt := range tests {
		got := imageRectOf(t, e.bitmapGlyph(face, 2, tt.size)).Dst
		if !near(got.Left, tt.want.Left) || !near(got.Top, tt.want.Top) || !near(got.Right, tt.want.Right) || !near(got.Bottom, tt.want.Bottom) {
			t.Errorf("%s: destination %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestSbixGlyphGolden(t *testing.T) {
	e := cbdtEngineOf(t, buildSbixFont(sbixStrikes))
	img, _ := renderLine(t, e, 230, 150, []bitmapLine{
		{"A\U0001F600\U0001F601", 20, ui.Offset{Dx: 8, Dy: 8}},
		{"A\U0001F600\U0001F601", 40, ui.Offset{Dx: 8, Dy: 40}},
	})
	goldens.Check(t, "sbix-glyphs", img)
}

func TestSbixGlyphOfAnotherGraphicTypeDrawsNothing(t *testing.T) {
	// Apple Color Emoji has PNG strikes; a strike of TIFF images is not drawn and is not an error.
	strikes := []sbixStrike{{ppem: 32, size: 32, originX: 2, originY: -4, graphicType: "tiff", noGlyph3: true}}
	face := sbixFace(t, strikes)
	e := bundledEngine(t)
	if pic := e.bitmapGlyph(face, 2, 20); pic != nil {
		t.Errorf("a glyph of a TIFF strike has a picture")
	}
}
