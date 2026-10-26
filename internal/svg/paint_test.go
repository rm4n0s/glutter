package svg

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/ui"
)

// render draws a document at the scale (pixels to a unit of the document) into an image of its size.
func render(t testing.TB, d *Document, scale float64) *image.RGBA {
	t.Helper()
	size := d.Size()
	w, h := int(math.Round(size.Width*scale)), int(math.Round(size.Height*scale))
	rec := ui.NewPictureRecorder()
	c := ui.NewCanvas(rec)
	c.Scale(scale, scale)
	c.DrawPicture(d.PictureAt(scale))
	b := ui.NewSceneBuilder()
	b.AddPicture(ui.Offset{}, rec.EndRecording())
	img, err := soft.Render(b.Build(), w, h)
	if err != nil {
		t.Fatalf("soft.Render: %v", err)
	}
	return img
}

// pixel returns the color at x, y as non-premultiplied bytes.
func pixel(img *image.RGBA, x, y int) color.NRGBA {
	return color.NRGBAModel.Convert(img.RGBAAt(x, y)).(color.NRGBA)
}

func near(a, b color.NRGBA, tol int) bool {
	d := func(x, y uint8) int { return int(math.Abs(float64(x) - float64(y))) }
	return d(a.R, b.R) <= tol && d(a.G, b.G) <= tol && d(a.B, b.B) <= tol && d(a.A, b.A) <= tol
}

func rgba(r, g, b, a uint8) color.NRGBA { return color.NRGBA{R: r, G: g, B: b, A: a} }

// opsOf returns the operations of the picture of a document.
func opsOf(d *Document) []ui.DisplayOp { return d.Picture().DisplayList().Ops() }

func countOps[T ui.DisplayOp](ops []ui.DisplayOp) int {
	n := 0
	for _, op := range ops {
		if _, ok := op.(T); ok {
			n++
		}
	}
	return n
}

func TestViewBoxTransform(t *testing.T) {
	wide := ViewBox{X: 10, Y: 20, Width: 50, Height: 25} // in 100 x 100: meet scales by 2, slice by 4
	tall := ViewBox{X: 0, Y: 0, Width: 25, Height: 50}   // in 100 x 100: meet scales by 2
	tests := []struct {
		name string
		vb   ViewBox
		par  string
		w, h float64
		want Affine
	}{
		// meet: the scale is 2, the content is 100 x 50 and 50 is left on y.
		{"meet xMinYMin", wide, "xMinYMin meet", 100, 100, Affine{A: 2, D: 2, E: -20, F: -40}},
		{"meet xMidYMin", wide, "xMidYMin", 100, 100, Affine{A: 2, D: 2, E: -20, F: -40}},
		{"meet xMaxYMin", wide, "xMaxYMin meet", 100, 100, Affine{A: 2, D: 2, E: -20, F: -40}},
		{"meet xMinYMid", wide, "xMinYMid", 100, 100, Affine{A: 2, D: 2, E: -20, F: -15}},
		{"meet xMidYMid (the default)", wide, "", 100, 100, Affine{A: 2, D: 2, E: -20, F: -15}},
		{"meet xMaxYMid", wide, "xMaxYMid meet", 100, 100, Affine{A: 2, D: 2, E: -20, F: -15}},
		{"meet xMinYMax", wide, "xMinYMax meet", 100, 100, Affine{A: 2, D: 2, E: -20, F: 10}},
		{"meet xMidYMax", wide, "xMidYMax", 100, 100, Affine{A: 2, D: 2, E: -20, F: 10}},
		{"meet xMaxYMax", wide, "xMaxYMax meet", 100, 100, Affine{A: 2, D: 2, E: -20, F: 10}},
		// the other way round: the content is 50 x 100 and 50 is left on x.
		{"tall meet xMinYMid", tall, "xMinYMid meet", 100, 100, Affine{A: 2, D: 2}},
		{"tall meet xMidYMid", tall, "xMidYMid meet", 100, 100, Affine{A: 2, D: 2, E: 25}},
		{"tall meet xMaxYMid", tall, "xMaxYMid meet", 100, 100, Affine{A: 2, D: 2, E: 50}},
		// slice: the scale is 4, the content is 200 x 100 and 100 is cut on x.
		{"slice xMinYMin", wide, "xMinYMin slice", 100, 100, Affine{A: 4, D: 4, E: -40, F: -80}},
		{"slice xMidYMid", wide, "xMidYMid slice", 100, 100, Affine{A: 4, D: 4, E: -90, F: -80}},
		{"slice xMaxYMax", wide, "xMaxYMax slice", 100, 100, Affine{A: 4, D: 4, E: -140, F: -80}},
		{"slice xMinYMax", wide, "xMinYMax slice", 100, 100, Affine{A: 4, D: 4, E: -40, F: -80}},
		{"slice xMaxYMin", wide, "xMaxYMin slice", 100, 100, Affine{A: 4, D: 4, E: -140, F: -80}},
		// none stretches: 2 on x and 4 on y.
		{"none", wide, "none", 100, 100, Affine{A: 2, D: 4, E: -20, F: -80}},
		{"none slice is ignored", wide, "none slice", 100, 100, Affine{A: 2, D: 4, E: -20, F: -80}},
		// a viewport that is not the same shape as the box.
		{"meet in a wide viewport", tall, "xMidYMid meet", 200, 100, Affine{A: 2, D: 2, E: 75}},
		{"slice in a wide viewport", tall, "xMidYMid slice", 200, 100, Affine{A: 8, D: 8, F: -150}},
		{"identity", ViewBox{Width: 100, Height: 100}, "", 100, 100, Identity},
		{"an empty box draws at the origin", ViewBox{Width: 0, Height: 10}, "", 100, 100, Identity},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			par := PreserveAspectRatio{}
			if tt.par != "" {
				var ok bool
				if par, ok = ParsePreserveAspectRatio(tt.par); !ok {
					t.Fatalf("cannot parse %q", tt.par)
				}
			}
			got := ViewBoxTransform(tt.vb, par, tt.w, tt.h)
			if !affEq(got, tt.want) {
				t.Errorf("ViewBoxTransform = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestDashPath(t *testing.T) {
	circle := ui.NewPath()
	circle.AddOval(ui.NewRectFromLTWH(0, 0, 20, 20))
	curve := ui.NewPath()
	curve.MoveTo(0, 0)
	curve.CubicTo(0, 50, 100, 50, 100, 0)
	square := ui.NewPath()
	square.AddRect(ui.NewRectFromLTWH(0, 0, 10, 10))
	line := ui.NewPath()
	line.MoveTo(0, 0)
	line.LineTo(100, 0)

	lengths := func(p *ui.Path) []float64 {
		var out []float64
		for _, m := range p.ComputeMetrics() {
			out = append(out, m.Length)
		}
		return out
	}

	t.Run("along a curve", func(t *testing.T) {
		// Each dash is 20 long and starts 30 further along the curve than the one before it, and its end points are
		// on the curve where the distance says.
		orig := curve.ComputeMetrics()[0]
		dashed := dashPath(curve, []float64{20, 10}, 0).ComputeMetrics()
		want := int(orig.Length / 30)
		if orig.Length-float64(want)*30 > 0 {
			want++
		}
		if len(dashed) != want {
			t.Fatalf("%d dashes on a curve of length %.2f, want %d", len(dashed), orig.Length, want)
		}
		for i, m := range dashed {
			start := float64(i) * 30
			wantLen := math.Min(20, orig.Length-start)
			if math.Abs(m.Length-wantLen) > 0.02 {
				t.Errorf("dash %d is %.3f long, want %.3f", i, m.Length, wantLen)
			}
			a, _ := m.GetTangentForOffset(0)
			b, _ := orig.GetTangentForOffset(start)
			if math.Hypot(a.Position.Dx-b.Position.Dx, a.Position.Dy-b.Position.Dy) > 0.02 {
				t.Errorf("dash %d starts at %v, the curve is at %v at %v", i, a.Position, b.Position, start)
			}
		}
	})

	t.Run("a circle", func(t *testing.T) {
		total := circle.ComputeMetrics()[0].Length
		got := lengths(dashPath(circle, []float64{10, 5}, 0))
		// The circle is 62.83 long: dashes of 10 at 0, 15, 30, 45, and 2.83 of the one at 60, which joins the first (a
		// closed contour that starts with a dash), so there are four, and the last is 12.83 long.
		want := []float64{10, 10, 10, total - 60 + 10}
		if len(got) != len(want) {
			t.Fatalf("dash lengths = %v on a circle of %.3f", got, total)
		}
		for i := range want {
			if math.Abs(got[i]-want[i]) > 1e-3 {
				t.Errorf("dash lengths = %v, want %v", got, want)
				break
			}
		}
		// An offset of 10 starts in the gap (5 of it is left): the dashes are at 5, 20, 35, 50 and nothing joins.
		got = lengths(dashPath(circle, []float64{10, 5}, 10))
		if len(got) != 4 {
			t.Fatalf("dash lengths with an offset = %v", got)
		}
		for _, l := range got {
			if math.Abs(l-10) > 1e-3 {
				t.Errorf("dash lengths with an offset = %v, want 4 of 10", got)
				break
			}
		}
	})

	tests := []struct {
		name   string
		path   *ui.Path
		dash   []float64
		offset float64
		want   []float64
	}{
		{"line", line, []float64{10, 10}, 0, []float64{10, 10, 10, 10, 10}},
		{"offset in a dash", line, []float64{10, 10}, 5, []float64{5, 10, 10, 10, 10, 5}},
		{"offset in a gap", line, []float64{10, 10}, 15, []float64{10, 10, 10, 10, 10}},
		{"negative offset", line, []float64{10, 10}, -5, []float64{10, 10, 10, 10, 10}},
		{"offset beyond a period", line, []float64{10, 10}, 25, []float64{5, 10, 10, 10, 10, 5}},
		{"long dashes", line, []float64{30, 10}, 0, []float64{30, 30, 20}},
		{"dash longer than the line", line, []float64{300, 10}, 0, []float64{100}},
		{"closed shape, dash ends at the start", square, []float64{10, 10}, 0, []float64{10, 10}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lengths(dashPath(tt.path, tt.dash, tt.offset))
			if len(got) != len(tt.want) {
				t.Fatalf("lengths = %v, want %v", got, tt.want)
			}
			for i := range got {
				if math.Abs(got[i]-tt.want[i]) > 0.01 {
					t.Fatalf("lengths = %v, want %v", got, tt.want)
				}
			}
		})
	}

	t.Run("a dash of length 0 is a point", func(t *testing.T) {
		dashed := dashPath(line, []float64{0, 25}, 0)
		points := 0
		for _, s := range dashed.Segments() {
			if s.Verb == ui.PathVerbLine && s.P0 == s.P1 {
				points++
			}
		}
		if points != 4 {
			t.Errorf("%d points, want 4 (at 0, 25, 50 and 75)", points)
		}
	})

	t.Run("too many dashes", func(t *testing.T) {
		long := ui.NewPath()
		long.MoveTo(0, 0)
		long.LineTo(1e6, 0)
		if got := dashPath(long, []float64{0.01, 0.01}, 0); got != long {
			t.Error("a million dashes of a hundredth should leave the path as it is")
		}
	})
	t.Run("invalid", func(t *testing.T) {
		for _, d := range [][]float64{{0, 0}, {-1, 2}, {math.NaN(), 1}} {
			if got := dashPath(line, d, 0); got != line {
				t.Errorf("dash %v changed the path", d)
			}
		}
	})
}

func TestTransformAttributeSyntax(t *testing.T) {
	tests := []struct {
		attr string
		want Affine
	}{
		{`transform="rotate(90 10 10)"`, Affine{0, 1, -1, 0, 20, 0}},
		{`transform="translate(5,6) scale(2)"`, Affine{2, 0, 0, 2, 5, 6}},
		{`style="transform: rotate(90deg)"`, Affine{0, 1, -1, 0, 0, 0}},
		{`transform="rotate(0.25turn)"`, Affine{0, 1, -1, 0, 0, 0}},
	}
	for _, tt := range tests {
		t.Run(tt.attr, func(t *testing.T) {
			d := mustParse(t, svgOpen+`<rect id="r" width="5" height="5" `+tt.attr+`/></svg>`)
			if got := mustFind(t, d, "r").Transform; !affEq(got, tt.want) {
				t.Errorf("transform = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestOpacityLayers(t *testing.T) {
	tests := []struct {
		name   string
		body   string
		layers int
		alpha  int // the alpha of the fill of the first path drawn, if the opacity is folded into it
	}{
		{"no opacity", `<rect width="10" height="10" fill="red"/>`, 0, 255},
		{"a shape with only a fill folds the opacity", `<rect width="10" height="10" fill="red" opacity="0.5"/>`, 0, 128},
		{"a shape with only a stroke folds the opacity", `<rect width="10" height="10" fill="none" stroke="red" opacity="0.5"/>`, 0, 128},
		{"fill and stroke overlap", `<rect width="10" height="10" fill="red" stroke="blue" stroke-width="4" opacity="0.5"/>`, 1, 255},
		{"a group of one shape folds it", `<g opacity="0.5"><rect width="10" height="10" fill="red"/></g>`, 0, 128},
		{"nested opacity multiplies", `<g opacity="0.5"><rect width="10" height="10" fill="red" opacity="0.5"/></g>`, 1, 128},
		{"two shapes may overlap", `<g opacity="0.5"><rect width="10" height="10" fill="red"/><rect width="10" height="10" fill="blue"/></g>`, 1, 255},
		{"a group in a group", `<g opacity="0.5"><g><rect width="10" height="10" fill="red"/><circle r="4" fill="blue"/></g></g>`, 1, 255},
		{"a pattern needs the layer", `<pattern id="p" width="4" height="4" patternUnits="userSpaceOnUse"><rect width="2" height="2"/></pattern><rect width="10" height="10" fill="url(#p)" opacity="0.5"/>`, 1, 255},
		{"opacity 1 makes none", `<g opacity="1"><rect width="10" height="10" fill="red"/><circle r="4" fill="blue"/></g>`, 0, 255},
		{"opacity 0 draws nothing", `<g opacity="0"><rect width="10" height="10" fill="red"/><circle r="4" fill="blue"/></g>`, 0, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := mustParse(t, svgOpen+tt.body+`</svg>`)
			ops := opsOf(d)
			if got := countOps[ui.OpSaveLayer](ops); got != tt.layers {
				t.Errorf("%d save layers, want %d", got, tt.layers)
			}
			if tt.alpha < 0 {
				if n := countOps[ui.OpDrawPath](ops); n != 0 {
					t.Errorf("%d paths drawn, want none", n)
				}
				return
			}
			for _, op := range ops {
				if p, ok := op.(ui.OpDrawPath); ok {
					if got := p.Paint.Color().Alpha(); got != tt.alpha {
						t.Errorf("alpha of the first path = %d, want %d", got, tt.alpha)
					}
					break
				}
			}
		})
	}
	t.Run("saves and restores balance", func(t *testing.T) {
		for _, tt := range tests {
			d := mustParse(t, svgOpen+tt.body+`</svg>`)
			ops := opsOf(d)
			saves := countOps[ui.OpSave](ops) + countOps[ui.OpSaveLayer](ops)
			if restores := countOps[ui.OpRestore](ops); saves != restores {
				t.Errorf("%s: %d saves, %d restores", tt.name, saves, restores)
			}
		}
	})
}

func TestPaintGradientEdgeCases(t *testing.T) {
	const stops = `<stop offset="0" stop-color="#ff0000"/><stop offset="1" stop-color="#0000ff"/>`
	tests := []struct {
		name string
		defs string
		fill string
		want color.NRGBA // at the center of a 20 x 20 rect
	}{
		// SVG 1.1 says the area is painted with the last stop when the vector or the radius is empty.
		{"linear, the vector is a point", `<linearGradient id="g" x1="0.5" y1="0.5" x2="0.5" y2="0.5">` + stops + `</linearGradient>`, "url(#g)", rgba(0, 0, 255, 255)},
		{"radial, radius 0", `<radialGradient id="g" r="0">` + stops + `</radialGradient>`, "url(#g)", rgba(0, 0, 255, 255)},
		{"one stop", `<linearGradient id="g"><stop offset="0.3" stop-color="#00ff00"/></linearGradient>`, "url(#g)", rgba(0, 255, 0, 255)},
		{"a singular gradientTransform", `<linearGradient id="g" gradientTransform="scale(0 1)">` + stops + `</linearGradient>`, "url(#g)", rgba(0, 0, 255, 255)},
		{"the stops stop before the end", `<linearGradient id="g"><stop offset="0.2" stop-color="#ff0000"/><stop offset="0.4" stop-color="#00ff00"/></linearGradient>`, "url(#g)", rgba(0, 255, 0, 255)},
		{"a missing gradient takes the fallback", ``, "url(#nope) #123456", rgba(0x12, 0x34, 0x56, 255)},
		{"a missing gradient without a fallback is none", ``, "url(#nope)", rgba(0, 0, 0, 0)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := mustParse(t, `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20"><defs>`+tt.defs+`</defs><rect width="20" height="20" fill="`+tt.fill+`"/></svg>`)
			if got := pixel(render(t, d, 1), 10, 10); !near(got, tt.want, 2) {
				t.Errorf("center = %v, want %v", got, tt.want)
			}
		})
	}

	// A gradient in objectBoundingBox units on a shape with no area paints nothing; in userSpaceOnUse units it does.
	t.Run("no bounding box", func(t *testing.T) {
		d := mustParse(t, `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20"><defs>
			<linearGradient id="a">`+stops+`</linearGradient>
			<linearGradient id="b" gradientUnits="userSpaceOnUse" x2="20">`+stops+`</linearGradient></defs>
			<path d="M0 5H20" stroke="url(#a)" stroke-width="4"/><path d="M0 15H20" stroke="url(#b)" stroke-width="4"/></svg>`)
		img := render(t, d, 1)
		if got := pixel(img, 10, 5); got.A != 0 {
			t.Errorf("objectBoundingBox on a line = %v, want nothing", got)
		}
		if got := pixel(img, 10, 15); got.A != 255 || got.R < 100 || got.B < 100 {
			t.Errorf("userSpaceOnUse on a line = %v, want the middle of the gradient", got)
		}
	})
}

func TestPaintOrder(t *testing.T) {
	// The stroke's inner half is over the fill, or under it when paint-order says so.
	for _, tt := range []struct {
		name  string
		style string
		want  color.NRGBA
	}{
		{"fill then stroke", ``, rgba(0, 0, 255, 255)},
		{"stroke then fill", `paint-order="stroke"`, rgba(255, 0, 0, 255)},
		{"stroke fill markers", `paint-order="stroke fill markers"`, rgba(255, 0, 0, 255)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			d := mustParse(t, `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20"><rect x="5" y="5" width="10" height="10" fill="#f00" stroke="#00f" stroke-width="6" `+tt.style+`/></svg>`)
			// (6, 10) is 1 unit inside the rectangle's edge: stroke over fill, or fill over stroke.
			if got := pixel(render(t, d, 1), 6, 10); !near(got, tt.want, 2) {
				t.Errorf("pixel = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNonScalingStroke(t *testing.T) {
	// A stroke of 2 under a scale of 4 is 8 wide, or 2 device pixels wide with non-scaling-stroke, at any scale of the
	// picture. A row across the left edge of the rectangle counts the opaque pixels.
	for _, scale := range []float64{1, 2} {
		t.Run(fmt.Sprintf("scale %g", scale), func(t *testing.T) {
			count := func(effect string) int {
				d := mustParse(t, `<svg xmlns="http://www.w3.org/2000/svg" width="100" height="50"><g transform="scale(4)"><rect x="10" y="3" width="10" height="6" fill="none" stroke="#000" stroke-width="2" `+effect+`/></g></svg>`)
				img := render(t, d, scale)
				n := 0
				for x := 0; x < img.Bounds().Dx()/2; x++ {
					if pixel(img, x, int(20*scale)).A > 200 {
						n++
					}
				}
				return n
			}
			if got, want := count(""), int(8*scale); got < want-1 || got > want+1 {
				t.Errorf("scaling stroke: %d opaque pixels, want %d", got, want)
			}
			if got, want := count(`vector-effect="non-scaling-stroke"`), 2; got < want-1 || got > want+1 {
				t.Errorf("non-scaling stroke: %d opaque pixels, want about %d", got, want)
			}
		})
	}
}

func TestUseOfViewports(t *testing.T) {
	// A use of an svg element or a symbol makes a viewport that its width and height set.
	const defs = `<defs>
		<svg id="s" width="20" height="20"><rect width="100%" height="100%" fill="#f00"/></svg>
		<symbol id="y" viewBox="0 0 10 10"><rect width="10" height="10" fill="#00f"/></symbol></defs>`
	tests := []struct {
		name string
		use  string
		in   [][2]int
		out  [][2]int
	}{
		{"svg with its own size", `<use href="#s" x="10" y="10"/>`, [][2]int{{15, 15}, {29, 29}}, [][2]int{{35, 35}, {5, 5}}},
		{"svg with the use's size", `<use href="#s" x="10" y="10" width="40" height="30"/>`, [][2]int{{45, 15}, {15, 35}}, [][2]int{{55, 15}, {15, 45}}},
		{"symbol, 100% by default", `<use href="#y" x="10" y="10"/>`, [][2]int{{15, 15}, {95, 95}}, [][2]int{{5, 5}}},
		{"symbol with a size", `<use href="#y" x="10" y="10" width="30" height="30"/>`, [][2]int{{15, 15}, {35, 35}}, [][2]int{{45, 45}, {5, 5}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := mustParse(t, `<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100" viewBox="0 0 100 100">`+defs+tt.use+`</svg>`)
			img := render(t, d, 1)
			for _, p := range tt.in {
				if pixel(img, p[0], p[1]).A != 255 {
					t.Errorf("(%d, %d) is not drawn", p[0], p[1])
				}
			}
			for _, p := range tt.out {
				if pixel(img, p[0], p[1]).A != 0 {
					t.Errorf("(%d, %d) is drawn", p[0], p[1])
				}
			}
		})
	}
}

func TestClipPathEdges(t *testing.T) {
	const defs = `<defs>
		<clipPath id="none"/>
		<clipPath id="box" clipPathUnits="objectBoundingBox"><rect width="0.5" height="1"/></clipPath>
		<clipPath id="two"><rect width="12" height="20"/><rect x="8" width="12" height="20"/></clipPath>
		<clipPath id="rev"><path d="M0 0H12V20H0Z"/><path d="M8 0V20H20V0Z"/></clipPath></defs>`
	tests := []struct {
		name string
		el   string
		in   [][2]int
		out  [][2]int
	}{
		{"an empty clip path hides the element", `<rect width="20" height="20" clip-path="url(#none)"/>`, nil, [][2]int{{10, 10}}},
		{"objectBoundingBox units", `<rect x="4" y="0" width="12" height="20" clip-path="url(#box)"/>`, [][2]int{{8, 10}}, [][2]int{{14, 10}, {2, 10}}},
		{"objectBoundingBox on a line is not drawn", `<path d="M0 10H20" stroke="#000" stroke-width="4" clip-path="url(#box)"/>`, nil, [][2]int{{4, 10}, {16, 10}}},
		{"the union of two shapes", `<rect width="20" height="20" clip-path="url(#two)"/>`, [][2]int{{2, 10}, {10, 10}, {18, 10}}, nil},
		{"shapes that overlap with opposite windings", `<rect width="20" height="20" clip-path="url(#rev)"/>`, [][2]int{{2, 10}, {10, 10}, {18, 10}}, nil},
		{"a missing clip path is ignored", `<rect width="20" height="20" clip-path="url(#nope)"/>`, [][2]int{{2, 10}, {18, 10}}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := mustParse(t, `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20">`+defs+tt.el+`</svg>`)
			img := render(t, d, 1)
			for _, p := range tt.in {
				if pixel(img, p[0], p[1]).A != 255 {
					t.Errorf("(%d, %d) is not drawn", p[0], p[1])
				}
			}
			for _, p := range tt.out {
				if pixel(img, p[0], p[1]).A != 0 {
					t.Errorf("(%d, %d) is drawn", p[0], p[1])
				}
			}
		})
	}
}

func TestPatternTileScale(t *testing.T) {
	// The tile is rasterized with the pixels the device will show: a 10 x 10 tile at a scale of 3 is 30 x 30 pixels.
	d := mustParse(t, `<svg xmlns="http://www.w3.org/2000/svg" width="40" height="40"><defs>
		<pattern id="p" width="10" height="10" patternUnits="userSpaceOnUse"><circle cx="5" cy="5" r="4"/></pattern></defs>
		<rect width="40" height="40" fill="url(#p)"/></svg>`)
	for _, scale := range []float64{1, 2, 3} {
		tile := 0
		for _, op := range d.PictureAt(scale).DisplayList().Ops() {
			if p, ok := op.(ui.OpDrawPath); ok {
				if sh, ok := p.Paint.Shader().(*ui.ImageShader); ok {
					tile = sh.Image().Width()
					if sh.Image().Height() != tile {
						t.Errorf("tile is %dx%d, want a square", sh.Image().Width(), sh.Image().Height())
					}
				}
			}
		}
		if want := int(10 * scale); tile != want {
			t.Errorf("scale %g: tile is %d pixels wide, want %d", scale, tile, want)
		}
	}
	// A tile used by two shapes is made once.
	two := mustParse(t, `<svg xmlns="http://www.w3.org/2000/svg" width="40" height="40"><defs>
		<pattern id="p" width="10" height="10" patternUnits="userSpaceOnUse"><circle cx="5" cy="5" r="4"/></pattern></defs>
		<rect width="20" height="40" fill="url(#p)"/><rect x="20" width="20" height="40" fill="url(#p)"/></svg>`)
	var images []*ui.Image
	for _, op := range two.Picture().DisplayList().Ops() {
		if p, ok := op.(ui.OpDrawPath); ok {
			if sh, ok := p.Paint.Shader().(*ui.ImageShader); ok {
				images = append(images, sh.Image())
			}
		}
	}
	if len(images) != 2 || images[0] != images[1] {
		t.Errorf("%d tile images, want 2 shaders of one image", len(images))
	}
}

// pngDataURL makes a data URL of a w x h PNG filled with the color.
func pngDataURL(t *testing.T, w, h int, c color.NRGBA) string {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

func TestPaintImages(t *testing.T) {
	red := pngDataURL(t, 4, 2, rgba(255, 0, 0, 255))
	svgImage := "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString(
		[]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="10" height="10"><rect width="10" height="10" fill="#0000ff"/></svg>`))
	tests := []struct {
		name string
		el   string
		in   [][2]int
		out  [][2]int
	}{
		{"fits in its rectangle (meet)", `<image x="10" y="10" width="40" height="40" href="` + red + `"/>`, [][2]int{{30, 30}, {12, 22}, {48, 38}}, [][2]int{{30, 15}, {30, 45}, {5, 5}}},
		{"stretches with none", `<image x="10" y="10" width="40" height="40" preserveAspectRatio="none" href="` + red + `"/>`, [][2]int{{30, 12}, {30, 48}, {12, 12}}, [][2]int{{5, 5}, {55, 30}}},
		{"covers with slice and is clipped", `<image x="10" y="10" width="40" height="40" preserveAspectRatio="xMidYMid slice" href="` + red + `"/>`, [][2]int{{12, 12}, {48, 48}}, [][2]int{{5, 30}, {55, 30}}},
		{"takes its size from the image", `<image x="10" y="10" href="` + red + `"/>`, [][2]int{{12, 11}, {13, 11}}, [][2]int{{15, 11}, {12, 13}}},
		{"takes one side from the other", `<image x="10" y="10" width="40" href="` + red + `"/>`, [][2]int{{30, 15}, {12, 28}}, [][2]int{{30, 35}}},
		{"an SVG image", `<image x="10" y="10" width="20" height="20" href="` + svgImage + `"/>`, [][2]int{{12, 12}, {28, 28}}, [][2]int{{35, 35}}},
		{"hidden", `<image x="10" y="10" width="40" height="40" visibility="hidden" href="` + red + `"/>`, nil, [][2]int{{30, 30}}},
		{"not an image", `<image x="10" y="10" width="40" height="40" href="data:image/png;base64,AAAA"/>`, nil, [][2]int{{30, 30}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := mustParse(t, `<svg xmlns="http://www.w3.org/2000/svg" width="60" height="60">`+tt.el+`</svg>`)
			img := render(t, d, 1)
			for _, p := range tt.in {
				if pixel(img, p[0], p[1]).A < 200 {
					t.Errorf("(%d, %d) is not drawn", p[0], p[1])
				}
			}
			for _, p := range tt.out {
				if pixel(img, p[0], p[1]).A != 0 {
					t.Errorf("(%d, %d) is drawn", p[0], p[1])
				}
			}
		})
	}
	t.Run("an SVG image has the right colors", func(t *testing.T) {
		d := mustParse(t, `<svg xmlns="http://www.w3.org/2000/svg" width="60" height="60"><image width="20" height="20" opacity="0.5" href="`+svgImage+`"/></svg>`)
		if got := pixel(render(t, d, 1), 10, 10); !near(got, rgba(0, 0, 255, 128), 2) {
			t.Errorf("pixel = %v, want half-transparent blue", got)
		}
	})
	t.Run("images nest only so deep", func(t *testing.T) {
		inner := `<svg xmlns="http://www.w3.org/2000/svg" width="10" height="10"><rect width="10" height="10" fill="#00f"/></svg>`
		for i := 0; i < maxImageDepth+2; i++ {
			inner = `<svg xmlns="http://www.w3.org/2000/svg" width="10" height="10"><image width="10" height="10" href="data:image/svg+xml;base64,` +
				base64.StdEncoding.EncodeToString([]byte(inner)) + `"/></svg>`
		}
		d := mustParse(t, inner)
		if got := pixel(render(t, d, 1), 5, 5); got.A != 0 {
			t.Errorf("an image nested %d deep is drawn: %v", maxImageDepth+2, got)
		}
		shallow := `<svg xmlns="http://www.w3.org/2000/svg" width="10" height="10"><rect width="10" height="10" fill="#00f"/></svg>`
		for i := 0; i < maxImageDepth; i++ {
			shallow = `<svg xmlns="http://www.w3.org/2000/svg" width="10" height="10"><image width="10" height="10" href="data:image/svg+xml;base64,` +
				base64.StdEncoding.EncodeToString([]byte(shallow)) + `"/></svg>`
		}
		if got := pixel(render(t, mustParse(t, shallow), 1), 5, 5); got.A != 255 {
			t.Errorf("an image nested %d deep is not drawn: %v", maxImageDepth, got)
		}
	})
}

func TestPictureRobustness(t *testing.T) {
	// Documents that ask for the impossible must draw (or not) without a panic, and quickly.
	huge := strings.Repeat(`<g>`, 200) + `<rect width="5" height="5"/>` + strings.Repeat(`</g>`, 200)
	tests := []struct{ name, body string }{
		{"empty", ``},
		{"a pattern with a tile of a trillion pixels", `<pattern id="p" width="1e12" height="1e12" patternUnits="userSpaceOnUse"><rect width="5" height="5"/></pattern><rect width="10" height="10" fill="url(#p)"/>`},
		{"a pattern with a tile of nothing", `<pattern id="p" width="1e-9" height="1e-9" patternUnits="userSpaceOnUse"><rect width="5" height="5"/></pattern><rect width="10" height="10" fill="url(#p)"/>`},
		{"a pattern that uses itself", `<pattern id="p" width="5" height="5" patternUnits="userSpaceOnUse"><rect width="5" height="5" fill="url(#p)"/></pattern><rect width="10" height="10" fill="url(#p)"/>`},
		{"a clip path that uses itself", `<clipPath id="c" clip-path="url(#c)"><rect width="5" height="5"/></clipPath><rect width="10" height="10" clip-path="url(#c)"/>`},
		{"a million dashes", `<path d="M0 0H100000" stroke="#000" stroke-dasharray="0.01"/>`},
		{"a singular transform", `<g transform="scale(0)"><rect width="10" height="10" stroke="#000" vector-effect="non-scaling-stroke"/></g>`},
		{"deep groups with opacity", huge},
		{"a zero-size viewport", `<svg width="0" height="0"><rect width="10" height="10"/></svg>`},
		{"NaN-like values", `<rect width="10" height="10" stroke-width="1e999" stroke="#000" stroke-dashoffset="1e999" stroke-dasharray="1 1"/>`},
		{"a stroke of no width", `<rect width="10" height="10" fill="none" stroke="#000" stroke-width="0"/>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := mustParse(t, `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="10" height="10">`+tt.body+`</svg>`)
			_ = d.Picture()
			_ = d.PictureAt(0)
			_ = d.PictureAt(math.NaN())
			_ = d.PictureAt(3)
		})
	}
	t.Run("a document with no root", func(t *testing.T) {
		if p := (&Document{}).Picture(); p == nil {
			t.Error("no picture")
		}
		var d *Document
		if p := d.Picture(); p == nil {
			t.Error("no picture for a nil document")
		}
	})
}

func TestPictureOfTestdata(t *testing.T) {
	// Every document of testdata (the parser's and the references') draws without a panic.
	for _, pattern := range []string{"testdata/*.svg", "testdata/reference/*.svg", "testdata/reference/equivalent/*.svg"} {
		files := globOrFail(t, pattern)
		for _, f := range files {
			t.Run(f, func(t *testing.T) {
				d := mustParse(t, readFileOrFail(t, f))
				if p := d.Picture(); len(p.DisplayList().Ops()) == 0 && countNodes(d.Root) > 1 {
					t.Error("a document with content recorded nothing")
				}
			})
		}
	}
}

func globOrFail(t *testing.T, pattern string) []string {
	t.Helper()
	files, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func readFileOrFail(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
