package svg

import (
	"bytes"
	"flag"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/rm4n0s/glutter/internal/goldens"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/ui"
)

var updateReferences = flag.Bool("update-references", false, "remake the reference PNGs of testdata/reference with rsvg-convert")

// referenceWidth is the width in pixels of every reference image; the height follows from the document's aspect
// ratio.
const referenceWidth = 200

// renderDocument draws a document with the software renderer into an image of w x h pixels, scaled to fill it.
func renderDocument(t testing.TB, d *Document, w, h int) *image.RGBA {
	t.Helper()
	size := d.Size()
	sx, sy := float64(w)/size.Width, float64(h)/size.Height
	rec := ui.NewPictureRecorder()
	c := ui.NewCanvas(rec)
	c.Scale(sx, sy)
	c.DrawPicture(d.PictureAt(sx))
	b := ui.NewSceneBuilder()
	b.AddPicture(ui.Offset{}, rec.EndRecording())
	img, err := soft.Render(b.Build(), w, h)
	if err != nil {
		t.Fatalf("soft.Render: %v", err)
	}
	return img
}

// referenceCases are the documents of testdata/reference, each with the tolerance its comparison with librsvg's
// render allows: Tolerance is the largest difference in one channel of one pixel (out of 255) that counts as equal,
// and Pixels how many pixels may differ by more than that. Both are as small as the document passes with. The edges of
// shapes are anti-aliased by exact coverage here and by cairo's rasterizer in librsvg, and the two differ by a few
// levels along an edge, more where an edge is thin or curved; Why says what a document that needs more has.
var referenceCases = []struct {
	Name      string
	Tolerance int
	Pixels    int
	// Via names a document of testdata/reference/equivalent that librsvg draws as the document itself should be drawn,
	// for a feature that librsvg does not have: the reference is made from it.
	Via string
	Why string
}{
	{Name: "clip-basic", Tolerance: 8, Pixels: 18, Why: "anti-aliased curves: cairo flattens a curve with a tolerance of 0.1 px, so the pixels along it differ by up to a quarter of their coverage"},                                                                          // 12 pixels, largest 15
	{Name: "clip-group-opacity", Tolerance: 8, Pixels: 10, Why: "anti-aliased edges: cairo's rasterizer and Glutter's cover edge pixels a few levels apart"},                                                                                                                   // 5 pixels, largest 10
	{Name: "clip-nested", Tolerance: 8, Pixels: 3, Via: "clip-nested", Why: "librsvg ignores clip-path on a clipPath and on its children, which the specification applies; the reference is made from an equivalent document (testdata/reference/equivalent/clip-nested.svg)"}, // 0 pixels, largest 7
	{Name: "clip-rule", Tolerance: 8, Pixels: 8, Why: "anti-aliased edges: cairo's rasterizer and Glutter's cover edge pixels a few levels apart"},                                                                                                                             // 4 pixels, largest 45
	{Name: "clip-transform", Tolerance: 16, Pixels: 3, Why: "anti-aliased edges: cairo's rasterizer and Glutter's cover edge pixels a few levels apart"},                                                                                                                       // 0 pixels, largest 14
	{Name: "clip-units-bbox", Tolerance: 16, Pixels: 6, Why: "anti-aliased edges: cairo's rasterizer and Glutter's cover edge pixels a few levels apart"},                                                                                                                      // 2 pixels, largest 18
	{Name: "dash-closed", Tolerance: 32, Pixels: 28, Why: "the corners where dashes turn: the join of a dash at a corner differs by a pixel, and the dash ends are anti-aliased"},                                                                                              // 20 pixels, largest 79
	{Name: "dash-curve", Tolerance: 64, Pixels: 5, Why: "dash ends along a curve: cairo measures a curve by its own flattening, so a dash ends up to half a pixel from where Glutter's does, and at a butt end a column of pixels is then in or out"},                          // 1 pixels, largest 71
	{Name: "dash-offset", Tolerance: 2, Pixels: 3, Why: "the same pixels: nothing is anti-aliased differently here"},                                                                                                                                                           // 0 pixels, largest 0
	{Name: "dash-round-dots", Tolerance: 8, Pixels: 3, Why: "anti-aliased curves: cairo flattens a curve with a tolerance of 0.1 px, so the pixels along it differ by up to a quarter of their coverage"},                                                                      // 0 pixels, largest 8
	{Name: "dash-simple", Tolerance: 2, Pixels: 3, Why: "the same pixels: nothing is anti-aliased differently here"},                                                                                                                                                           // 0 pixels, largest 0
	{Name: "fill-rule", Tolerance: 16, Pixels: 12, Why: "anti-aliased edges: cairo's rasterizer and Glutter's cover edge pixels a few levels apart"},                                                                                                                           // 7 pixels, largest 42
	{Name: "fill-stroke-opacity", Tolerance: 32, Pixels: 22, Why: "anti-aliased curves: cairo flattens a curve with a tolerance of 0.1 px, so the pixels along it differ by up to a quarter of their coverage"},                                                                // 15 pixels, largest 39
	{Name: "gradient-degenerate", Tolerance: 2, Pixels: 3, Why: "the same pixels: nothing is anti-aliased differently here"},                                                                                                                                                   // 0 pixels, largest 1
	{Name: "gradient-href", Tolerance: 2, Pixels: 3, Why: "the same pixels: nothing is anti-aliased differently here"},                                                                                                                                                         // 0 pixels, largest 1
	{Name: "gradient-linear-bbox", Tolerance: 8, Pixels: 11, Why: "anti-aliased edges: cairo's rasterizer and Glutter's cover edge pixels a few levels apart"},                                                                                                                 // 6 pixels, largest 15
	{Name: "gradient-linear-transform", Tolerance: 2, Pixels: 3, Why: "the same pixels: nothing is anti-aliased differently here"},                                                                                                                                             // 0 pixels, largest 1
	{Name: "gradient-linear-userspace", Tolerance: 2, Pixels: 3, Why: "the same pixels: nothing is anti-aliased differently here"},                                                                                                                                             // 0 pixels, largest 1
	{Name: "gradient-radial", Tolerance: 2, Pixels: 3, Why: "the same pixels: nothing is anti-aliased differently here"},                                                                                                                                                       // 0 pixels, largest 1
	{Name: "gradient-radial-focal", Tolerance: 8, Pixels: 11, Why: "anti-aliased curves: cairo flattens a curve with a tolerance of 0.1 px, so the pixels along it differ by up to a quarter of their coverage"},                                                               // 6 pixels, largest 15
	{Name: "gradient-spread", Tolerance: 2, Pixels: 3, Why: "the same pixels: nothing is anti-aliased differently here"},                                                                                                                                                       // 0 pixels, largest 1
	{Name: "gradient-stop-opacity", Tolerance: 2, Pixels: 3, Why: "the same pixels: nothing is anti-aliased differently here"},                                                                                                                                                 // 0 pixels, largest 2
	{Name: "gradient-stroke", Tolerance: 64, Pixels: 16, Why: "dash ends along a circle: the same as dash-curve"},                                                                                                                                                              // 10 pixels, largest 104
	{Name: "group-opacity", Tolerance: 2, Pixels: 3, Why: "the same pixels: nothing is anti-aliased differently here"},                                                                                                                                                         // 0 pixels, largest 0
	{Name: "group-opacity-nested", Tolerance: 8, Pixels: 6, Why: "anti-aliased edges: cairo's rasterizer and Glutter's cover edge pixels a few levels apart"},                                                                                                                  // 2 pixels, largest 12
	{Name: "image-aspect", Tolerance: 2, Pixels: 5, Why: "anti-aliased edges: cairo's rasterizer and Glutter's cover edge pixels a few levels apart"},                                                                                                                          // 1 pixels, largest 3
	{Name: "image-raster", Tolerance: 2, Pixels: 3, Why: "the same pixels: nothing is anti-aliased differently here"},                                                                                                                                                          // 0 pixels, largest 2
	{Name: "image-svg", Tolerance: 8, Pixels: 12, Why: "anti-aliased curves: cairo flattens a curve with a tolerance of 0.1 px, so the pixels along it differ by up to a quarter of their coverage"},                                                                           // 7 pixels, largest 12
	{Name: "pattern-bbox-units", Tolerance: 8, Pixels: 20, Why: "anti-aliased curves: cairo flattens a curve with a tolerance of 0.1 px, so the pixels along it differ by up to a quarter of their coverage"},                                                                  // 13 pixels, largest 15
	{Name: "pattern-href-stroke", Tolerance: 2, Pixels: 3, Why: "the same pixels: nothing is anti-aliased differently here"},                                                                                                                                                   // 0 pixels, largest 0
	{Name: "pattern-transform", Tolerance: 64, Pixels: 3, Why: "the tile of a rotated or skewed pattern is resampled: librsvg and Glutter filter it differently, so the edges inside the tile differ"},                                                                         // 0 pixels, largest 57
	{Name: "pattern-userspace", Tolerance: 16, Pixels: 3, Why: "anti-aliased curves: cairo flattens a curve with a tolerance of 0.1 px, so the pixels along it differ by up to a quarter of their coverage"},                                                                   // 0 pixels, largest 16
	{Name: "pattern-viewbox", Tolerance: 8, Pixels: 11, Why: "anti-aliased curves: cairo flattens a curve with a tolerance of 0.1 px, so the pixels along it differ by up to a quarter of their coverage"},                                                                     // 6 pixels, largest 10
	{Name: "shape-circle-ellipse", Tolerance: 16, Pixels: 8, Why: "anti-aliased curves: cairo flattens a curve with a tolerance of 0.1 px, so the pixels along it differ by up to a quarter of their coverage"},                                                                // 4 pixels, largest 21
	{Name: "shape-line-polys", Tolerance: 16, Pixels: 10, Why: "anti-aliased edges: cairo's rasterizer and Glutter's cover edge pixels a few levels apart"},                                                                                                                    // 5 pixels, largest 31
	{Name: "shape-opacity-fill-stroke", Tolerance: 2, Pixels: 5, Why: "anti-aliased edges: cairo's rasterizer and Glutter's cover edge pixels a few levels apart"},                                                                                                             // 1 pixels, largest 5
	{Name: "shape-path-arcs", Tolerance: 64, Pixels: 8, Why: "anti-aliased curves: cairo flattens a curve with a tolerance of 0.1 px, so the pixels along it differ by up to a quarter of their coverage"},                                                                     // 4 pixels, largest 72
	{Name: "shape-path-curves", Tolerance: 32, Pixels: 23, Why: "anti-aliased curves: cairo flattens a curve with a tolerance of 0.1 px, so the pixels along it differ by up to a quarter of their coverage"},                                                                  // 16 pixels, largest 59
	{Name: "shape-rect-radii", Tolerance: 8, Pixels: 16, Why: "anti-aliased curves: cairo flattens a curve with a tolerance of 0.1 px, so the pixels along it differ by up to a quarter of their coverage"},                                                                    // 10 pixels, largest 15
	{Name: "shape-rendering-crisp", Tolerance: 8, Pixels: 6, Why: "crispEdges: the two renderers pick different pixels at a corner of the square (2 pixels)"},                                                                                                                  // 2 pixels, largest 255
	{Name: "stroke-caps", Tolerance: 8, Pixels: 3, Why: "anti-aliased edges: cairo's rasterizer and Glutter's cover edge pixels a few levels apart"},                                                                                                                           // 0 pixels, largest 7
	{Name: "stroke-closed-open", Tolerance: 16, Pixels: 3, Why: "anti-aliased edges: cairo's rasterizer and Glutter's cover edge pixels a few levels apart"},                                                                                                                   // 0 pixels, largest 9
	{Name: "stroke-gradient-userspace-opacity", Tolerance: 32, Pixels: 22, Why: "anti-aliased curves and the round caps, as in the curves, on a stroke that is also painted with a gradient"},                                                                                  // 15 pixels, largest 89
	{Name: "stroke-joins", Tolerance: 8, Pixels: 3, Why: "anti-aliased edges: cairo's rasterizer and Glutter's cover edge pixels a few levels apart"},                                                                                                                          // 0 pixels, largest 8
	{Name: "stroke-miterlimit", Tolerance: 16, Pixels: 8, Why: "anti-aliased edges: cairo's rasterizer and Glutter's cover edge pixels a few levels apart"},                                                                                                                    // 4 pixels, largest 41
	{Name: "stroke-non-scaling", Tolerance: 2, Pixels: 8, Why: "vector-effect: non-scaling-stroke: a few pixels at the ends of the dashes and at the corners of the strokes"},                                                                                                  // 4 pixels, largest 63
	{Name: "style-cascade", Tolerance: 8, Pixels: 12, Why: "anti-aliased edges: cairo's rasterizer and Glutter's cover edge pixels a few levels apart"},                                                                                                                        // 7 pixels, largest 15
	{Name: "transform-nested", Tolerance: 16, Pixels: 3, Why: "anti-aliased edges at an angle: cairo's rasterizer and Glutter's cover them a few levels apart, more where a rotated viewport clips"},                                                                           // 0 pixels, largest 15
	{Name: "use-nested-svg", Tolerance: 64, Pixels: 18, Why: "rotated clip rectangles of nested viewports: cairo clips at an angle with a different anti-aliasing"},                                                                                                            // 12 pixels, largest 75
	{Name: "use-transform", Tolerance: 16, Pixels: 7, Why: "anti-aliased edges at an angle: cairo's rasterizer and Glutter's cover them a few levels apart, more where a rotated viewport clips"},                                                                              // 3 pixels, largest 17
	{Name: "viewport-meet-max-min", Tolerance: 8, Pixels: 5, Why: "anti-aliased edges: cairo's rasterizer and Glutter's cover edge pixels a few levels apart"},                                                                                                                 // 1 pixels, largest 11
	{Name: "viewport-meet-mid", Tolerance: 8, Pixels: 5, Why: "anti-aliased edges: cairo's rasterizer and Glutter's cover edge pixels a few levels apart"},                                                                                                                     // 1 pixels, largest 11
	{Name: "viewport-meet-min-max", Tolerance: 8, Pixels: 5, Why: "anti-aliased edges: cairo's rasterizer and Glutter's cover edge pixels a few levels apart"},                                                                                                                 // 1 pixels, largest 11
	{Name: "viewport-nested-clip", Tolerance: 2, Pixels: 3, Why: "the same pixels: nothing is anti-aliased differently here"},                                                                                                                                                  // 0 pixels, largest 1
	{Name: "viewport-nested-overflow", Tolerance: 8, Pixels: 5, Why: "anti-aliased edges: cairo's rasterizer and Glutter's cover edge pixels a few levels apart"},                                                                                                              // 1 pixels, largest 9
	{Name: "viewport-none", Tolerance: 16, Pixels: 3, Why: "anti-aliased edges: cairo's rasterizer and Glutter's cover edge pixels a few levels apart"},                                                                                                                        // 0 pixels, largest 15
	{Name: "viewport-percent-nested", Tolerance: 8, Pixels: 3, Why: "anti-aliased edges: cairo's rasterizer and Glutter's cover edge pixels a few levels apart"},                                                                                                               // 0 pixels, largest 7
	{Name: "viewport-size-only", Tolerance: 2, Pixels: 3, Why: "the same pixels: nothing is anti-aliased differently here"},                                                                                                                                                    // 0 pixels, largest 0
	{Name: "viewport-slice", Tolerance: 8, Pixels: 12, Why: "anti-aliased edges: cairo's rasterizer and Glutter's cover edge pixels a few levels apart"},                                                                                                                       // 7 pixels, largest 13
	{Name: "viewport-slice-min", Tolerance: 8, Pixels: 12, Why: "anti-aliased edges: cairo's rasterizer and Glutter's cover edge pixels a few levels apart"},                                                                                                                   // 7 pixels, largest 13
	{Name: "viewport-symbol", Tolerance: 64, Pixels: 5, Why: "anti-aliased edges at an angle: cairo's rasterizer and Glutter's cover them a few levels apart, more where a rotated viewport clips"},                                                                            // 1 pixels, largest 65
	{Name: "viewport-symbol-overflow", Tolerance: 8, Pixels: 6, Why: "anti-aliased curves: cairo flattens a curve with a tolerance of 0.1 px, so the pixels along it differ by up to a quarter of their coverage"},                                                             // 3 pixels, largest 11
	{Name: "viewport-viewbox-offset", Tolerance: 16, Pixels: 23, Why: "anti-aliased edges: cairo's rasterizer and Glutter's cover edge pixels a few levels apart"},                                                                                                             // 16 pixels, largest 24
	{Name: "visibility-display", Tolerance: 2, Pixels: 3, Why: "the same pixels: nothing is anti-aliased differently here"},                                                                                                                                                    // 0 pixels, largest 0
}

func TestReferences(t *testing.T) {
	files, err := filepath.Glob("testdata/reference/*.svg")
	if err != nil {
		t.Fatal(err)
	}
	listed := map[string]bool{}
	for _, c := range referenceCases {
		listed[c.Name] = true
	}
	for _, f := range files {
		name := filepath.Base(f[:len(f)-len(".svg")])
		if !listed[name] {
			t.Errorf("testdata/reference/%s.svg is not in referenceCases", name)
		}
	}
	if len(files) != len(referenceCases) {
		t.Errorf("%d documents in testdata/reference, %d in referenceCases", len(files), len(referenceCases))
	}
	if len(referenceCases) < 40 {
		t.Errorf("only %d reference documents, want at least 40", len(referenceCases))
	}
	for _, tt := range referenceCases {
		t.Run(tt.Name, func(t *testing.T) {
			src := filepath.Join("testdata", "reference", tt.Name+".svg")
			ref := filepath.Join("testdata", "reference", tt.Name+".png")
			if *updateReferences {
				from := src
				if tt.Via != "" {
					from = filepath.Join("testdata", "reference", "equivalent", tt.Via+".svg")
				}
				makeReference(t, from, ref)
			}
			data, err := os.ReadFile(src)
			if err != nil {
				t.Fatal(err)
			}
			d, err := Parse(data)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			want, err := goldens.ReadPNG(ref)
			if err != nil {
				t.Fatalf("%v (make the references with: go test ./internal/svg -run TestReferences -update-references)", err)
			}
			got := renderDocument(t, d, want.Bounds().Dx(), want.Bounds().Dy())
			res := goldens.Compare(want, got, tt.Tolerance)
			if res.SizeMismatch {
				t.Fatalf("size mismatch")
			}
			if res.Pixels > tt.Pixels {
				dir := os.Getenv("SVG_REFERENCE_OUT")
				if dir == "" {
					dir = t.TempDir()
				}
				_ = goldens.WritePNG(filepath.Join(dir, tt.Name+".actual.png"), got)
				_ = goldens.WritePNG(filepath.Join(dir, tt.Name+".diff.png"), res.Diff)
				t.Errorf("%d pixels differ by more than %d (largest difference %d), %d are allowed; images in %s",
					res.Pixels, tt.Tolerance, res.MaxDiff, tt.Pixels, dir)
			} else {
				t.Logf("%d pixels differ by more than %d (largest difference %d)", res.Pixels, tt.Tolerance, res.MaxDiff)
			}
		})
	}
}

// makeReference renders an SVG file with librsvg into a PNG of referenceWidth pixels, or skips the test if
// rsvg-convert is not installed.
func makeReference(t *testing.T, src, dst string) {
	t.Helper()
	exe, err := exec.LookPath("rsvg-convert")
	if err != nil {
		t.Skip("rsvg-convert is not installed; install it with: sudo dnf install librsvg2-tools")
	}
	cmd := exec.Command(exe, "--width", strconv.Itoa(referenceWidth), "--format", "png", src)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("rsvg-convert %s: %v", src, err)
	}
	if _, err := png.Decode(bytes.NewReader(out)); err != nil {
		t.Fatalf("rsvg-convert %s wrote no PNG: %v", src, err)
	}
	if err := os.WriteFile(dst, out, 0o644); err != nil {
		t.Fatal(err)
	}
}
