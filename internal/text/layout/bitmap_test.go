package layout

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"math"
	"testing"

	"github.com/go-text/typesetting/font"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/goldens"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/internal/text"
	"github.com/rm4n0s/glutter/material/fonts"
	"github.com/rm4n0s/glutter/ui"
)

// cbdtFamily is the family the test font is loaded as.
const cbdtFamily = "CBDT Test"

// cbdtFace parses the test font into a face.
func cbdtFace(t *testing.T, strikes []testStrike) *font.Face {
	t.Helper()
	face, err := font.ParseTTF(bytes.NewReader(buildCBDTFont(strikes)))
	if err != nil {
		t.Fatalf("the test font does not parse: %v", err)
	}
	return face
}

// cbdtEngine is an engine with Roboto and the test font as the family cbdtFamily. The font errors
// go to onErr.
func cbdtEngine(t *testing.T, strikes []testStrike, onErr func(error)) *Engine {
	t.Helper()
	c := text.NewCollection(text.CollectionOpts{Matcher: noSystemFonts{}})
	if err := fonts.Register(c); err != nil {
		t.Fatal(err)
	}
	e := NewEngine(context.Background(), c, EngineOpts{OnFontError: onErr})
	if err := e.LoadFontFromList(buildCBDTFont(strikes), cbdtFamily); err != nil {
		t.Fatalf("LoadFontFromList: %v", err)
	}
	return e
}

// imageRectOf returns the image and the rectangles of the OpDrawImageRect that a picture draws.
func imageRectOf(t *testing.T, pic *ui.Picture) ui.OpDrawImageRect {
	t.Helper()
	if pic == nil {
		t.Fatal("no picture")
	}
	var found []ui.OpDrawImageRect
	for _, op := range pic.DisplayList().Ops() {
		if v, ok := op.(ui.OpDrawImageRect); ok {
			found = append(found, v)
		}
	}
	if len(found) != 1 {
		t.Fatalf("%d images drawn, want 1", len(found))
	}
	return found[0]
}

// pixelAt returns the premultiplied pixel of an image.
func pixelAt(img *ui.Image, x, y int) [4]uint8 {
	p := img.Pixels()[4*(y*img.Width()+x):]
	return [4]uint8{p[0], p[1], p[2], p[3]}
}

func TestCBDTFontLoadsInGoText(t *testing.T) {
	face := cbdtFace(t, testStrikes)
	sizes := face.BitmapSizes()
	if len(sizes) != 2 || sizes[0].YPpem != 109 || sizes[1].YPpem != 32 {
		t.Fatalf("strikes = %+v, want ppem 109 and 32", sizes)
	}
	for _, tt := range []struct {
		r    rune
		gid  font.GID
		w, h int
	}{{'A', 1, 109, 109}, {0x1F600, 2, 109, 109}, {0x1F601, 3, 109, 109}} {
		gid, ok := face.NominalGlyph(tt.r)
		if !ok || gid != tt.gid {
			t.Fatalf("%U: gid %d %v, want %d", tt.r, gid, ok, tt.gid)
		}
		bm, _, ok := bitmapOf(face, gid, sizes[0])
		if !ok || bm.Format != font.PNG || bm.Width != tt.w || bm.Height != tt.h {
			t.Errorf("%U: bitmap %+v %v", tt.r, bm.Format, ok)
		}
	}
	if x, y := face.Ppem(); x != 0 || y != 0 {
		t.Errorf("the face keeps ppem %d, %d after the bitmaps are read, want 0, 0", x, y)
	}
}

func TestChooseBitmapStrike(t *testing.T) {
	strikes := func(ppems ...uint16) []font.BitmapSize {
		out := make([]font.BitmapSize, len(ppems))
		for i, p := range ppems {
			out[i] = font.BitmapSize{XPpem: p, YPpem: p}
		}
		return out
	}
	tests := []struct {
		name    string
		strikes []font.BitmapSize
		size    float64
		want    int // the index of the strike, -1 for none
	}{
		{"no strikes", nil, 20, -1},
		{"an exact match", strikes(109, 32, 64), 64, 2},
		{"the smallest bigger strike", strikes(109, 32, 64), 40, 2},
		{"the smallest bigger strike, in any order", strikes(32, 64, 109), 40, 1},
		{"a size between strikes goes up", strikes(32, 109), 32.5, 1},
		{"a size below every strike takes the smallest", strikes(109, 32), 8, 1},
		{"a tiny size takes the smallest", strikes(32, 109), 0.5, 0},
		{"a size above every strike takes the biggest", strikes(32, 109, 64), 300, 1},
		{"one strike", strikes(109), 500, 0},
		{"nothing can be drawn at size 0", strikes(32), 0, -1},
	}
	for _, tt := range tests {
		if got := chooseBitmapStrike(tt.strikes, tt.size); got != tt.want {
			t.Errorf("%s: strike %d, want %d", tt.name, got, tt.want)
		}
	}
}

func TestBitmapGlyphPicksTheStrikeOfTheSize(t *testing.T) {
	face := cbdtFace(t, testStrikes)
	e := bundledEngine(t)
	tests := []struct {
		size       float64
		wantSize   int // pixels of the image: the strike
		wantMarker [4]uint8
	}{
		{1, 32, [4]uint8{255, 255, 255, 255}},
		{20, 32, [4]uint8{255, 255, 255, 255}},
		{32, 32, [4]uint8{255, 255, 255, 255}},
		{32.1, 109, [4]uint8{0, 0, 0, 255}},
		{40, 109, [4]uint8{0, 0, 0, 255}},
		{109, 109, [4]uint8{0, 0, 0, 255}},
		{400, 109, [4]uint8{0, 0, 0, 255}},
	}
	for _, tt := range tests {
		op := imageRectOf(t, e.bitmapGlyph(face, 2, tt.size))
		if op.Image.Width() != tt.wantSize || op.Image.Height() != tt.wantSize {
			t.Errorf("size %v: image %dx%d, want %d", tt.size, op.Image.Width(), op.Image.Height(), tt.wantSize)
			continue
		}
		m := tt.wantSize/8 + 2 // inside the marker
		if got := pixelAt(op.Image, m, m); got != tt.wantMarker {
			t.Errorf("size %v: marker pixel %v, want %v", tt.size, got, tt.wantMarker)
		}
		// The body of the glyph is red, the border half transparent: the pixels are premultiplied.
		if got := pixelAt(op.Image, tt.wantSize/2, tt.wantSize/2); got != [4]uint8{255, 0, 0, 255} {
			t.Errorf("size %v: body pixel %v", tt.size, got)
		}
		if got := pixelAt(op.Image, 0, tt.wantSize/2); got != [4]uint8{128, 0, 0, 128} {
			t.Errorf("size %v: border pixel %v, want red at half alpha, premultiplied", tt.size, got)
		}
		if op.Paint.FilterQuality() != ui.FilterQualityMedium {
			t.Errorf("size %v: filter quality %v", tt.size, op.Paint.FilterQuality())
		}
	}
	// The pictures are made once per strike, not once per size.
	if a, b := e.bitmapGlyph(face, 2, 20), e.bitmapGlyph(face, 2, 31); a != b {
		t.Error("two sizes with one strike have two pictures")
	}
	if a, b := e.bitmapGlyph(face, 2, 20), e.bitmapGlyph(face, 3, 20); a == b {
		t.Error("two glyphs have one picture")
	}
	if a, b := e.bitmapGlyph(face, 2, 20), e.bitmapGlyph(face, 2, 40); a == b {
		t.Error("two strikes have one picture")
	}
	if x, y := face.Ppem(); x != 0 || y != 0 {
		t.Errorf("the face keeps ppem %d, %d, want 0, 0", x, y)
	}
}

func TestBitmapGlyphPlacement(t *testing.T) {
	// The picture has font units, y up, and the bitmap is drawn with y down under a flip; a pixel
	// of a strike is upem/ppem units. Glyph 2 has small metrics, glyph 3 big ones.
	tests := []struct {
		name string
		gid  font.GID
		size float64
		want ui.Rect // the destination, in font units, y down from the baseline
	}{
		// ppem 32: 64 units a pixel; bearing (2, 28); 32 pixels square.
		{"small metrics, strike 32", 2, 20, ui.Rect{Left: 2 * 64, Top: -28 * 64, Right: 34 * 64, Bottom: 4 * 64}},
		{"big metrics, strike 32", 3, 20, ui.Rect{Left: 2 * 64, Top: -28 * 64, Right: 34 * 64, Bottom: 4 * 64}},
		// ppem 109: 2048/109 units a pixel; bearing (7, 95); 109 pixels square.
		{"small metrics, strike 109", 2, 40, ui.Rect{Left: 7 * 2048.0 / 109, Top: -95 * 2048.0 / 109, Right: 116 * 2048.0 / 109, Bottom: 14 * 2048.0 / 109}},
		{"big metrics, strike 109", 3, 40, ui.Rect{Left: 7 * 2048.0 / 109, Top: -95 * 2048.0 / 109, Right: 116 * 2048.0 / 109, Bottom: 14 * 2048.0 / 109}},
	}
	face := cbdtFace(t, testStrikes)
	e := bundledEngine(t)
	for _, tt := range tests {
		pic := e.bitmapGlyph(face, tt.gid, tt.size)
		op := imageRectOf(t, pic)
		near := func(a, b float64) bool { return math.Abs(a-b) < 0.01 }
		got := op.Dst
		if !near(got.Left, tt.want.Left) || !near(got.Top, tt.want.Top) || !near(got.Right, tt.want.Right) || !near(got.Bottom, tt.want.Bottom) {
			t.Errorf("%s: destination %v, want %v", tt.name, got, tt.want)
		}
		if op.Src != (ui.Rect{Right: float64(op.Image.Width()), Bottom: float64(op.Image.Height())}) {
			t.Errorf("%s: source %v is not the whole image", tt.name, op.Src)
		}
	}
}

// bitmapLine is a text of the test font at a size and a place.
type bitmapLine struct {
	text string
	size float64
	at   ui.Offset
}

// renderLine renders paragraphs of the test font on white: each text at its size and offset.
func renderLine(t *testing.T, e *Engine, w, h int, lines []bitmapLine) (*image.RGBA, []*paragraph) {
	t.Helper()
	rec := ui.NewPictureRecorder()
	c := ui.NewCanvas(rec)
	bg := ui.NewPaint()
	bg.SetColor(ui.NewColor(0xFFFFFFFF))
	c.DrawRect(ui.Rect{Right: float64(w), Bottom: float64(h)}, bg)
	var paras []*paragraph
	for _, l := range lines {
		b := ui.NewParagraphBuilder(e, nil)
		b.PushStyle(ui.NewTextStyle(ui.TextStyleOpts{FontFamily: cbdtFamily, FontSize: ui.Ptr(l.size)}))
		b.AddText(l.text)
		para := b.Build()
		para.Layout(ui.NewParagraphConstraints(float64(w)))
		c.DrawParagraph(para, l.at)
		paras = append(paras, para.Layouter().(*paragraph))
	}
	sb := ui.NewSceneBuilder()
	sb.AddPicture(ui.Offset{}, rec.EndRecording())
	img, err := soft.Render(sb.Build(), w, h)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	return img, paras
}

// boundsOf returns the bounding box of the pixels that pick accepts, or an empty rectangle.
func boundsOf(img *image.RGBA, pick func(c color.RGBA) bool) image.Rectangle {
	var r image.Rectangle
	for y := img.Rect.Min.Y; y < img.Rect.Max.Y; y++ {
		for x := img.Rect.Min.X; x < img.Rect.Max.X; x++ {
			if pick(img.RGBAAt(x, y)) {
				r = r.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	return r
}

func isRed(c color.RGBA) bool   { return int(c.R) > int(c.G)+60 && int(c.R) > int(c.B)+60 }
func isGreen(c color.RGBA) bool { return int(c.G) > int(c.R)+60 && int(c.G) > int(c.B)+60 }
func isBlue(c color.RGBA) bool  { return int(c.B) > int(c.R)+60 && int(c.B) > int(c.G)+60 }

func TestBitmapGlyphsAreDrawnWhereTheBearingsPutThem(t *testing.T) {
	e := cbdtEngine(t, testStrikes, func(err error) { t.Errorf("font error: %v", err) })
	// Each glyph advances one em, so the glyphs of "A<1F600><1F601>" start at 0, size and 2 size.
	tests := []struct {
		size   float64
		strike int // pixels of the strike
		bx, by int // bearings, in strike pixels
	}{
		{20, 32, 2, 28},
		{30, 32, 2, 28},
		{40, 109, 7, 95},
		{64, 109, 7, 95},
	}
	for _, tt := range tests {
		img, paras := renderLine(t, e, 260, 140, []bitmapLine{{"A\U0001F600\U0001F601", tt.size, ui.Offset{Dx: 10, Dy: 10}}})
		draws := paintOf(paras[0])
		if len(draws) != 3 {
			t.Fatalf("size %v: %d pieces, want 3", tt.size, len(draws))
		}
		scale := tt.size / float64(tt.strike) // a pixel of the strike, in pixels
		for i, pick := range []func(color.RGBA) bool{isBlue, isRed, isGreen} {
			origin := ui.Offset{Dx: 10 + draws[i].Offset.Dx, Dy: 10 + draws[i].Offset.Dy}
			want := image.Rect(
				int(math.Round(origin.Dx+float64(tt.bx)*scale)),
				int(math.Round(origin.Dy-float64(tt.by)*scale)),
				int(math.Round(origin.Dx+float64(tt.bx+tt.strike)*scale)),
				int(math.Round(origin.Dy-float64(tt.by-tt.strike)*scale)),
			)
			got := boundsOf(img, pick)
			// The border is half transparent and filtered: allow a pixel.
			if got.Empty() || abs(got.Min.X-want.Min.X) > 1 || abs(got.Min.Y-want.Min.Y) > 1 || abs(got.Max.X-want.Max.X) > 1 || abs(got.Max.Y-want.Max.Y) > 1 {
				t.Errorf("size %v, glyph %d: drawn in %v, want %v", tt.size, i, got, want)
			}
		}
	}
}

func TestBitmapGlyphGolden(t *testing.T) {
	e := cbdtEngine(t, testStrikes, func(err error) { t.Errorf("font error: %v", err) })
	img, _ := renderLine(t, e, 230, 110, []bitmapLine{
		{"A\U0001F600\U0001F601", 20, ui.Offset{Dx: 8, Dy: 8}},
		{"A\U0001F600\U0001F601", 40, ui.Offset{Dx: 8, Dy: 40}},
	})
	goldens.Check(t, "bitmap-glyphs", img)
}

func TestBitmapGlyphColorIsTheOwnColor(t *testing.T) {
	// The color of the text does not change a bitmap glyph.
	e := cbdtEngine(t, testStrikes, func(err error) { t.Errorf("font error: %v", err) })
	var got []*ui.Picture
	for _, c := range []ui.Color{ui.NewColor(0xFF000000), ui.NewColor(0xFF00FF00)} {
		b := ui.NewParagraphBuilder(e, nil)
		b.PushStyle(ui.NewTextStyle(ui.TextStyleOpts{FontFamily: cbdtFamily, FontSize: ui.Ptr(20.0), Color: &c}))
		b.AddText("\U0001F600")
		para := b.Build()
		para.Layout(ui.NewParagraphConstraints(100))
		draws := paintOf(para.Layouter())
		if len(draws) != 1 {
			t.Fatalf("%d pieces, want 1", len(draws))
		}
		pic, ok := draws[0].Op.(ui.OpDrawPicture)
		if !ok {
			t.Fatalf("op %T, want a picture", draws[0].Op)
		}
		got = append(got, pic.Picture)
		// The picture is drawn under the scale of size/upem with y up, like the COLR glyphs.
		s := float32(20.0 / testUpem)
		if m := draws[0].Matrix; m == nil || m.Storage()[0] != s || m.Storage()[5] != -s {
			t.Errorf("matrix %v, want a scale of %v with y flipped", m, s)
		}
	}
	if got[0] != got[1] {
		t.Error("the color of the text makes another picture of a bitmap glyph")
	}
}

func TestBitmapGlyphBrokenDataDrawsNothing(t *testing.T) {
	wide := image.NewNRGBA(image.Rect(0, 0, maxBitmapGlyphSize+1, 1))
	var tooWide bytes.Buffer
	if err := png.Encode(&tooWide, wide); err != nil {
		t.Fatal(err)
	}
	good := testGlyphPNG(32, testRed, testStrikes[1].marker)
	tests := []struct {
		name string
		data []byte
	}{
		{"not an image", []byte("this is not a PNG")},
		{"empty", []byte{}},
		{"a PNG cut short", good[:len(good)/2]},
		{"a PNG that is too big", tooWide.Bytes()},
	}
	for _, tt := range tests {
		strikes := []testStrike{{ppem: 32, size: 32, bearingX: 2, bearingY: 28, png: map[int][]byte{2: tt.data}}}

		// The failure has a tag and a route.
		e := bundledEngine(t)
		face := cbdtFace(t, strikes)
		_, err := e.recordBitmap(face, 2, face.BitmapSizes()[0])
		appErr, ok := errors.FromError(err)
		if !ok {
			t.Fatalf("%s: expected *errors.Error, got %T %v", tt.name, err, err)
		}
		if !appErr.HasRoute("Engine.recordBitmap->decodeGlyphImage.BitmapGlyphInvalid") {
			t.Errorf("%s: unexpected failure path: %s", tt.name, appErr.Route())
		}

		// The paragraph draws the other glyphs, reports the broken one once, and does not panic.
		var reported []error
		e = cbdtEngine(t, strikes, func(err error) { reported = append(reported, err) })
		img, paras := renderLine(t, e, 120, 60, []bitmapLine{{"\U0001F600A\U0001F600", 20, ui.Offset{Dx: 4, Dy: 4}}})
		if len(reported) != 1 {
			t.Errorf("%s: %d font errors reported, want 1: %v", tt.name, len(reported), reported)
		}
		if !boundsOf(img, isRed).Empty() {
			t.Errorf("%s: the broken glyph drew %v", tt.name, boundsOf(img, isRed))
		}
		if boundsOf(img, isBlue).Empty() {
			t.Errorf("%s: the good glyph next to it was not drawn", tt.name)
		}
		if len(paintOf(paras[0])) != 1 {
			t.Errorf("%s: %d pieces, want only the good glyph", tt.name, len(paintOf(paras[0])))
		}
	}
}

func TestBitmapGlyphMissingDrawsNothing(t *testing.T) {
	e := bundledEngine(t)
	face := cbdtFace(t, testStrikes)
	for _, gid := range []font.GID{0, 4, 99, 70000} {
		if pic := e.bitmapGlyph(face, gid, 20); pic != nil {
			t.Errorf("glyph %d has a picture", gid)
		}
	}
	// A font whose bitmap tables have no strike has no color glyphs.
	empty := cbdtFace(t, nil)
	if pic := e.colorGlyph(empty, 2, ui.NewColor(0xFF000000), 20); pic != nil {
		t.Error("a font without strikes has a picture")
	}
	if pic := e.bitmapGlyph(empty, 2, 20); pic != nil {
		t.Error("a font without strikes has a bitmap picture")
	}
	// Roboto has no bitmaps and no COLR table.
	r := e.fontsFor([]string{fonts.RobotoFamily}, ui.FontWeightW400, false, ui.Locale{}, nil).ResolveFace('H')
	gid, _ := r.NominalGlyph('H')
	if pic := e.colorGlyph(r, gid, ui.NewColor(0xFF000000), 20); pic != nil {
		t.Error("an outline glyph has a color picture")
	}
}
