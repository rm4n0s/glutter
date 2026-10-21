package gpu

import (
	"image"
	"math"
	"reflect"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/ui"
)

// fakeMasks is a MaskCache whose rasterizer covers the area of the shape's bounds with full
// coverage, and whose pages and uploads are recorded.
type fakeMasks struct {
	*MaskCache
	pages   int
	uploads []image.Rectangle
	areas   []image.Rectangle
}

func newFakeMasks(pageSize, maxPages int) *fakeMasks {
	f := &fakeMasks{MaskCache: NewMaskCache(pageSize, maxPages)}
	f.Rasterize = func(op ui.DisplayOp, m *ui.Matrix4, area image.Rectangle) (image.Rectangle, []float32) {
		f.areas = append(f.areas, area)
		cov := make([]float32, area.Dx()*area.Dy())
		for i := range cov {
			cov[i] = 1
		}
		return area, cov
	}
	f.NewPage = func() error { f.pages++; return nil }
	f.Upload = func(page, x, y, w, h int, data []byte) error {
		if len(data) != w*h {
			return errors.New("TestBadUpload", "the data is not w x h bytes")
		}
		f.uploads = append(f.uploads, image.Rect(x, y, x+w, y+h))
		return nil
	}
	return f
}

func TestMaskCacheTilesAndCaches(t *testing.T) {
	f := newFakeMasks(512, 8)
	op := ui.OpDrawRect{Rect: ui.NewRectFromLTWH(0, 0, 300, 100), Paint: *solidPaint(0xFF000000)}
	sh, ok := ShapeOf(op)
	if !ok {
		t.Fatal("not a shape")
	}
	m := ui.NewMatrix4Identity()
	m.Translate(100, 50, 0) // inside the view, so that no edge pixel is cut off by its border
	view := image.Rect(0, 0, 1000, 1000)
	tiles, err := f.Tiles(op, sh, m, 0, view)
	if err != nil {
		t.Fatal(err)
	}
	// 300 x 100 is two tiles of at most MaxAtlasTile (256) across.
	if len(tiles) != 2 || f.Rasterized != 2 || f.pages != f.Atlas().Pages() || f.pages < 1 || len(f.uploads) != 2 {
		t.Fatalf("tiles %d, rasterized %d, pages %d, uploads %d", len(tiles), f.Rasterized, f.pages, len(f.uploads))
	}
	if tiles[0].X0 != 100 || tiles[0].X1 != 100+256 || tiles[1].X0 != 100+256 || tiles[1].X1 != 100+300 {
		t.Errorf("tiles at %+v", tiles)
	}
	// Again, and at another whole-pixel offset: from the atlas.
	m2 := ui.NewMatrix4Identity()
	m2.Translate(140, 70, 0)
	for _, mm := range []*ui.Matrix4{m, m2} {
		f.BeginFrame()
		again, err := f.Tiles(op, sh, mm, 0, view)
		if err != nil {
			t.Fatal(err)
		}
		if len(again) != 2 || f.Rasterized != 2 {
			t.Fatalf("again: tiles %d, rasterized %d", len(again), f.Rasterized)
		}
	}
	// A fractional offset is another coverage.
	m3 := ui.NewMatrix4Identity()
	m3.Translate(100.5, 50, 0)
	f.BeginFrame()
	if _, err := f.Tiles(op, sh, m3, 0, view); err != nil || f.Rasterized != 4 {
		t.Errorf("fractional offset: rasterized %d, err %v", f.Rasterized, err)
	}
	// Only what shows is made.
	f2 := newFakeMasks(512, 8)
	tiles, err = f2.Tiles(op, sh, m, 0, image.Rect(0, 0, 200, 200))
	if err != nil || len(tiles) != 1 || f2.Rasterized != 1 {
		t.Errorf("a partly visible shape: %d tiles, rasterized %d, err %v", len(tiles), f2.Rasterized, err)
	}
	tiles, err = f2.Tiles(op, sh, m, 0, image.Rect(500, 500, 600, 600))
	if err != nil || len(tiles) != 0 {
		t.Errorf("an invisible shape: %d tiles, err %v", len(tiles), err)
	}
}

func TestMaskCacheEmptyCoverageAndErrors(t *testing.T) {
	f := newFakeMasks(256, 1)
	f.Rasterize = func(ui.DisplayOp, *ui.Matrix4, image.Rectangle) (image.Rectangle, []float32) {
		return image.Rectangle{}, nil
	}
	op := ui.OpDrawRect{Rect: ui.NewRectFromLTWH(0, 0, 10, 10), Paint: *solidPaint(0xFF000000)}
	sh, _ := ShapeOf(op)
	m := ui.NewMatrix4Identity()
	tiles, err := f.Tiles(op, sh, m, 1, image.Rect(0, 0, 50, 50))
	if err != nil || len(tiles) != 0 || f.pages != 0 || f.Rasterized != 1 {
		t.Errorf("a shape that covers nothing: %d tiles, %d pages, rasterized %d, err %v", len(tiles), f.pages, f.Rasterized, err)
	}
	// An empty tile is remembered too.
	if _, err := f.Tiles(op, sh, m, 1, image.Rect(0, 0, 50, 50)); err != nil || f.Rasterized != 1 {
		t.Errorf("an empty tile was made again: rasterized %d, err %v", f.Rasterized, err)
	}

	g := newFakeMasks(256, 1)
	g.NewPage = func() error { return errors.New("TestNoPage", "no page") }
	if _, err := g.Tiles(op, sh, m, 1, image.Rect(0, 0, 50, 50)); err == nil {
		t.Error("a page that cannot be made did not fail")
	} else if appErr, ok := errors.FromError(err); !ok || appErr.Tag != "TestNoPage" {
		t.Errorf("got %v, want the error of NewPage unchanged", err)
	}
}

func TestMaskMargin(t *testing.T) {
	fill := OpShape{Paint: solidPaint(0xFF000000)}
	thin := OpShape{Paint: linePaint(0xFF000000, 4, ui.StrokeCapButt, ui.StrokeJoinMiter), Stroke: true}
	hair := OpShape{Paint: linePaint(0xFF000000, 0, ui.StrokeCapButt, ui.StrokeJoinMiter), Stroke: true}
	tests := []struct {
		name  string
		sh    OpShape
		scale float64
		want  float64
	}{
		{"a fill reaches an edge pixel", fill, 1, 1},
		{"under a scale the pixel is smaller", fill, 4, 0.25},
		{"a stroke reaches half its width times the miter (4 / 2 * 4)", thin, 1, 1 + 2*4},
		{"a hairline is half a pixel", hair, 2, 0.5 + 0.25*4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MaskMargin(tt.sh, tt.scale); got != tt.want {
				t.Errorf("MaskMargin = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClipCoverage(t *testing.T) {
	area := image.Rect(0, 0, 4, 2)
	all := func(path *ui.Path, m *ui.Matrix4, aa bool, a image.Rectangle) (image.Rectangle, []float32) {
		cov := make([]float32, a.Dx()*a.Dy())
		for i := range cov {
			cov[i] = 1
		}
		return a, cov
	}
	left := func(path *ui.Path, m *ui.Matrix4, aa bool, a image.Rectangle) (image.Rectangle, []float32) {
		r := image.Rect(0, 0, 2, 2)
		cov := make([]float32, r.Dx()*r.Dy())
		for i := range cov {
			cov[i] = 1
		}
		return r, cov
	}
	none := func(*ui.Path, *ui.Matrix4, bool, image.Rectangle) (image.Rectangle, []float32) {
		return image.Rectangle{}, nil
	}
	rr := ClipShape{Kind: ClipShapeRRect, RRect: ui.NewRRectFromRectXY(ui.NewRectFromLTWH(0, 0, 4, 2), 1, 1), Transform: *ui.NewMatrix4Identity()}
	cutOut := rr
	cutOut.Difference = true
	tests := []struct {
		name      string
		shapes    []ClipShape
		rasterize func(*ui.Path, *ui.Matrix4, bool, image.Rectangle) (image.Rectangle, []float32)
		want      []byte
		empty     bool
	}{
		{"a shape that covers all", []ClipShape{rr}, all, []byte{255, 255, 255, 255, 255, 255, 255, 255}, false},
		{"a shape that covers the left half", []ClipShape{rr}, left, []byte{255, 255, 0, 0, 255, 255, 0, 0}, false},
		{"a shape that covers nothing", []ClipShape{rr}, none, nil, true},
		{"a difference of nothing leaves all", []ClipShape{cutOut}, none, []byte{255, 255, 255, 255, 255, 255, 255, 255}, false},
		{"a difference of the left half", []ClipShape{cutOut}, left, []byte{0, 0, 255, 255, 0, 0, 255, 255}, false},
		{"a difference of everything leaves nothing", []ClipShape{cutOut}, all, nil, true},
		{"two shapes multiply", []ClipShape{rr, cutOut}, left, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, empty := ClipCoverage(Clip{Bounds: ui.NewRectFromLTWH(0, 0, 4, 2), Shapes: tt.shapes}, area, tt.rasterize)
			if empty != tt.empty || !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ClipCoverage = %v, %v; want %v, %v", got, empty, tt.want, tt.empty)
			}
		})
	}
}

func TestClipKey(t *testing.T) {
	area := image.Rect(0, 0, 10, 10)
	mk := func(kind ClipShapeKind, aa bool, diff bool) []ClipShape {
		return []ClipShape{{Kind: kind, AntiAlias: aa, Difference: diff, RRect: ui.NewRRectFromRectXY(ui.NewRectFromLTWH(0, 0, 5, 5), 1, 1), Transform: *ui.NewMatrix4Identity()}}
	}
	base := ClipKey(area, mk(ClipShapeRRect, true, false))
	if ClipKey(area, mk(ClipShapeRRect, true, false)) != base {
		t.Error("the same clip has two keys")
	}
	for name, other := range map[string]uint64{
		"another area":     ClipKey(image.Rect(0, 0, 11, 10), mk(ClipShapeRRect, true, false)),
		"anti-aliasing":    ClipKey(area, mk(ClipShapeRRect, false, false)),
		"a difference":     ClipKey(area, mk(ClipShapeRRect, true, true)),
		"another kind":     ClipKey(area, mk(ClipShapeRSuperellipse, true, false)),
		"a second shape":   ClipKey(area, append(mk(ClipShapeRRect, true, false), mk(ClipShapeRRect, true, false)...)),
		"no shapes at all": ClipKey(area, nil),
	} {
		if other == base {
			t.Errorf("%s did not change the key", name)
		}
	}
}

func TestClipShapeOutline(t *testing.T) {
	r := ui.NewRectFromLTWH(1, 2, 30, 40)
	path := ui.NewPath()
	path.AddRect(r)
	tests := []struct {
		name  string
		shape ClipShape
	}{
		{"rect", ClipShape{Kind: ClipShapeRect, Rect: r}},
		{"rrect", ClipShape{Kind: ClipShapeRRect, RRect: ui.NewRRectFromRectXY(r, 4, 4)}},
		{"superellipse", ClipShape{Kind: ClipShapeRSuperellipse, RSuperellipse: ui.NewRSuperellipseFromRectXY(r, 4, 4)}},
		{"path", ClipShape{Kind: ClipShapePath, Path: path}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := tt.shape.Outline().GetBounds()
			near := func(a, b float64) bool { return math.Abs(a-b) < 1e-3 }
			if !near(b.Left, r.Left) || !near(b.Top, r.Top) || !near(b.Right, r.Right) || !near(b.Bottom, r.Bottom) {
				t.Errorf("bounds %v, want %v", b, r)
			}
		})
	}
}
