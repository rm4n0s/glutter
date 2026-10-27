package svg

import (
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

// docAt renders a document body (in the 100 x 100 viewBox of svgOpen) at scale 1.
func docAt(t *testing.T, body string) (get func(x, y int) [4]uint8) {
	t.Helper()
	d := mustParse(t, svgOpen+body+`</svg>`)
	img := render(t, d, 1)
	return func(x, y int) [4]uint8 {
		p := pixel(img, x, y)
		return [4]uint8{p.R, p.G, p.B, p.A}
	}
}

func TestMasks(t *testing.T) {
	const white = `<rect width="100" height="100" fill="#fff"/>`
	tests := []struct {
		name string
		body string
		at   [2]int
		want [4]uint8
		tol  int
	}{
		{"white luminance lets everything through", white + `<mask id="m" maskUnits="userSpaceOnUse" x="0" y="0" width="100" height="100"><rect width="100" height="100" fill="#fff"/></mask><rect width="100" height="100" fill="#f00" mask="url(#m)"/>`, [2]int{50, 50}, [4]uint8{255, 0, 0, 255}, 1},
		{"black luminance hides", white + `<mask id="m" maskUnits="userSpaceOnUse" x="0" y="0" width="100" height="100"><rect width="100" height="100" fill="#000"/></mask><rect width="100" height="100" fill="#f00" mask="url(#m)"/>`, [2]int{50, 50}, [4]uint8{255, 255, 255, 255}, 0},
		{"gray luminance is half", white + `<mask id="m" maskUnits="userSpaceOnUse" x="0" y="0" width="100" height="100"><rect width="100" height="100" fill="#808080"/></mask><rect width="100" height="100" fill="#f00" mask="url(#m)"/>`, [2]int{50, 50}, [4]uint8{255, 127, 127, 255}, 2},
		{"a transparent mask hides", white + `<mask id="m" maskUnits="userSpaceOnUse" x="0" y="0" width="100" height="100"><rect width="100" height="100" fill="#fff" fill-opacity="0"/></mask><rect width="100" height="100" fill="#f00" mask="url(#m)"/>`, [2]int{50, 50}, [4]uint8{255, 255, 255, 255}, 0},
		{"half transparent white is half", white + `<mask id="m" maskUnits="userSpaceOnUse" x="0" y="0" width="100" height="100"><rect width="100" height="100" fill="#fff" fill-opacity="0.5"/></mask><rect width="100" height="100" fill="#f00" mask="url(#m)"/>`, [2]int{50, 50}, [4]uint8{255, 127, 127, 255}, 2},
		{"mask-type alpha uses the alpha", white + `<mask id="m" mask-type="alpha" maskUnits="userSpaceOnUse" x="0" y="0" width="100" height="100"><rect width="100" height="100" fill="#000" fill-opacity="0.5"/></mask><rect width="100" height="100" fill="#f00" mask="url(#m)"/>`, [2]int{50, 50}, [4]uint8{255, 127, 127, 255}, 2},
		{"outside the mask region nothing is drawn", white + `<mask id="m" maskUnits="userSpaceOnUse" x="10" y="10" width="20" height="20"><rect width="100" height="100" fill="#fff"/></mask><rect width="100" height="100" fill="#f00" mask="url(#m)"/>`, [2]int{50, 50}, [4]uint8{255, 255, 255, 255}, 0},
		{"inside the mask region", white + `<mask id="m" maskUnits="userSpaceOnUse" x="10" y="10" width="20" height="20"><rect width="100" height="100" fill="#fff"/></mask><rect width="100" height="100" fill="#f00" mask="url(#m)"/>`, [2]int{20, 20}, [4]uint8{255, 0, 0, 255}, 0},
		{"the default region is the box with a margin", white + `<mask id="m"><rect width="100" height="100" fill="#fff"/></mask><rect x="20" y="20" width="50" height="50" fill="#f00" mask="url(#m)"/>`, [2]int{25, 25}, [4]uint8{255, 0, 0, 255}, 0},
		{"content in object bounding box units", white + `<mask id="m" maskContentUnits="objectBoundingBox"><rect width="0.5" height="1" fill="#fff"/></mask><rect x="20" y="20" width="60" height="60" fill="#f00" mask="url(#m)"/>`, [2]int{30, 50}, [4]uint8{255, 0, 0, 255}, 0},
		{"content in object bounding box units, the other half", white + `<mask id="m" maskContentUnits="objectBoundingBox"><rect width="0.5" height="1" fill="#fff"/></mask><rect x="20" y="20" width="60" height="60" fill="#f00" mask="url(#m)"/>`, [2]int{70, 50}, [4]uint8{255, 255, 255, 255}, 0},
		{"a mask of a mask multiplies", white + `<mask id="a" maskUnits="userSpaceOnUse" x="0" y="0" width="100" height="100"><rect width="100" height="100" fill="#808080"/></mask><mask id="m" maskUnits="userSpaceOnUse" x="0" y="0" width="100" height="100" mask="url(#a)"><rect width="100" height="100" fill="#808080"/></mask><rect width="100" height="100" fill="#f00" mask="url(#m)"/>`, [2]int{50, 50}, [4]uint8{255, 191, 191, 255}, 3},
		{"a mask on a group with opacity: both apply", white + `<mask id="m" maskUnits="userSpaceOnUse" x="0" y="0" width="100" height="100"><rect width="100" height="100" fill="#808080"/></mask><g opacity="0.5" mask="url(#m)"><rect width="100" height="100" fill="#f00"/></g>`, [2]int{50, 50}, [4]uint8{255, 191, 191, 255}, 3},
		{"a mask on a group with opacity: the children do not show through each other", white + `<mask id="m" maskUnits="userSpaceOnUse" x="0" y="0" width="100" height="100"><rect width="100" height="100" fill="#fff"/></mask><g opacity="0.5" mask="url(#m)"><rect width="60" height="60" fill="#f00"/><rect x="30" y="30" width="60" height="60" fill="#00f"/></g>`, [2]int{45, 45}, [4]uint8{127, 127, 255, 255}, 2},
		{"luminance of a color uses the weights", white + `<mask id="m" maskUnits="userSpaceOnUse" x="0" y="0" width="100" height="100"><rect width="100" height="100" fill="#00ff00"/></mask><rect width="100" height="100" fill="#000" mask="url(#m)"/>`, [2]int{50, 50}, [4]uint8{73, 73, 73, 255}, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := docAt(t, tt.body)(tt.at[0], tt.at[1])
			if !nearPx(got, tt.want, tt.tol) {
				t.Errorf("pixel %v = %v, want %v", tt.at, got, tt.want)
			}
		})
	}
	t.Run("a mask in object bounding box units on a line is not drawn", func(t *testing.T) {
		got := docAt(t, white+`<mask id="m"><rect width="100" height="100" fill="#fff"/></mask><path d="M 10 50 L 90 50" stroke="#f00" stroke-width="10" mask="url(#m)"/>`)(50, 50)
		if got != [4]uint8{255, 255, 255, 255} {
			t.Errorf("pixel = %v, want the background", got)
		}
	})
	t.Run("a reference to a mask that does not exist is ignored", func(t *testing.T) {
		got := docAt(t, white+`<rect width="100" height="100" fill="#f00" mask="url(#nothing)"/>`)(50, 50)
		if got != [4]uint8{255, 0, 0, 255} {
			t.Errorf("pixel = %v, want red", got)
		}
	})
}

func TestFilterGraph(t *testing.T) {
	const white = `<rect width="100" height="100" fill="#fff"/>`
	// A filter that uses a region of the user space whole picture, so that nothing is cut off.
	region := `filterUnits="userSpaceOnUse" x="0" y="0" width="100" height="100"`
	tests := []struct {
		name string
		body string
		at   [2]int
		want [4]uint8
		tol  int
	}{
		{"an offset moves the element", white + `<filter id="f" ` + region + `><feOffset dx="20" dy="10"/></filter><rect x="10" y="10" width="10" height="10" fill="#f00" filter="url(#f)"/>`, [2]int{35, 25}, [4]uint8{255, 0, 0, 255}, 0},
		{"an offset leaves nothing behind", white + `<filter id="f" ` + region + `><feOffset dx="20" dy="10"/></filter><rect x="10" y="10" width="10" height="10" fill="#f00" filter="url(#f)"/>`, [2]int{15, 15}, [4]uint8{255, 255, 255, 255}, 0},
		{"a flood fills the region", white + `<filter id="f" ` + region + `><feFlood flood-color="#0f0"/></filter><rect x="10" y="10" width="10" height="10" fill="#f00" filter="url(#f)"/>`, [2]int{90, 90}, [4]uint8{0, 255, 0, 255}, 0},
		{"a flood with opacity", white + `<filter id="f" ` + region + `><feFlood flood-color="#000" flood-opacity="0.5"/></filter><rect x="10" y="10" width="10" height="10" fill="#f00" filter="url(#f)"/>`, [2]int{90, 90}, [4]uint8{127, 127, 127, 255}, 2},
		{"a flood in a subregion", white + `<filter id="f" ` + region + `><feFlood flood-color="#0f0" x="30" y="30" width="20" height="20"/></filter><rect x="10" y="10" width="10" height="10" fill="#f00" filter="url(#f)"/>`, [2]int{40, 40}, [4]uint8{0, 255, 0, 255}, 0},
		{"outside the subregion of a flood", white + `<filter id="f" ` + region + `><feFlood flood-color="#0f0" x="30" y="30" width="20" height="20"/></filter><rect x="10" y="10" width="10" height="10" fill="#f00" filter="url(#f)"/>`, [2]int{60, 60}, [4]uint8{255, 255, 255, 255}, 0},
		{"named results: a flood in the shape of the source", white + `<filter id="f" ` + region + `><feFlood flood-color="#00f" result="c"/><feComposite in="c" in2="SourceAlpha" operator="in"/></filter><rect x="10" y="10" width="30" height="30" fill="#f00" filter="url(#f)"/>`, [2]int{20, 20}, [4]uint8{0, 0, 255, 255}, 0},
		{"named results: nothing outside the shape", white + `<filter id="f" ` + region + `><feFlood flood-color="#00f" result="c"/><feComposite in="c" in2="SourceAlpha" operator="in"/></filter><rect x="10" y="10" width="30" height="30" fill="#f00" filter="url(#f)"/>`, [2]int{60, 60}, [4]uint8{255, 255, 255, 255}, 0},
		{"named results: a result used after another primitive", white + `<filter id="f" ` + region + `><feFlood flood-color="#0f0" result="a"/><feOffset in="SourceGraphic" dx="30" result="b"/><feMerge><feMergeNode in="a"/><feMergeNode in="b"/></feMerge></filter><rect x="10" y="10" width="10" height="10" fill="#f00" filter="url(#f)"/>`, [2]int{45, 15}, [4]uint8{255, 0, 0, 255}, 0},
		{"named results: the green shows where the red is not", white + `<filter id="f" ` + region + `><feFlood flood-color="#0f0" result="a"/><feOffset in="SourceGraphic" dx="30" result="b"/><feMerge><feMergeNode in="a"/><feMergeNode in="b"/></feMerge></filter><rect x="10" y="10" width="10" height="10" fill="#f00" filter="url(#f)"/>`, [2]int{15, 15}, [4]uint8{0, 255, 0, 255}, 0},
		{"the previous result is the default input", white + `<filter id="f" ` + region + `><feOffset dx="10"/><feOffset dx="10"/></filter><rect x="10" y="10" width="10" height="10" fill="#f00" filter="url(#f)"/>`, [2]int{35, 15}, [4]uint8{255, 0, 0, 255}, 0},
		{"SourceAlpha is black", white + `<filter id="f" ` + region + `><feOffset in="SourceAlpha" dx="0"/></filter><rect x="10" y="10" width="10" height="10" fill="#f00" filter="url(#f)"/>`, [2]int{15, 15}, [4]uint8{0, 0, 0, 255}, 0},
		{"BackgroundImage is empty", white + `<filter id="f" ` + region + `><feOffset in="BackgroundImage" dx="0"/></filter><rect x="10" y="10" width="10" height="10" fill="#f00" filter="url(#f)"/>`, [2]int{15, 15}, [4]uint8{255, 255, 255, 255}, 0},
		{"an unknown name is the previous result", white + `<filter id="f" ` + region + `><feOffset in="nothing" dx="10"/></filter><rect x="10" y="10" width="10" height="10" fill="#f00" filter="url(#f)"/>`, [2]int{25, 15}, [4]uint8{255, 0, 0, 255}, 0},
		{"primitive units: objectBoundingBox offsets are fractions of the box", white + `<filter id="f" primitiveUnits="objectBoundingBox" ` + region + `><feOffset dx="1" dy="0"/></filter><rect x="10" y="10" width="20" height="10" fill="#f00" filter="url(#f)"/>`, [2]int{35, 15}, [4]uint8{255, 0, 0, 255}, 0},
		{"the offset follows the transform of the element", white + `<filter id="f" ` + region + `><feOffset dx="5"/></filter><g transform="scale(2)"><rect x="5" y="5" width="5" height="5" fill="#f00" filter="url(#f)"/></g>`, [2]int{25, 15}, [4]uint8{255, 0, 0, 255}, 0},
		{"an unknown primitive: the element is drawn as if it had no filter", white + `<filter id="f"><feTurbulence baseFrequency="0.1"/></filter><rect x="10" y="10" width="10" height="10" fill="#f00" filter="url(#f)"/>`, [2]int{15, 15}, [4]uint8{255, 0, 0, 255}, 0},
		{"a reference to a filter that is not there is ignored", white + `<rect x="10" y="10" width="10" height="10" fill="#f00" filter="url(#nothing)"/>`, [2]int{15, 15}, [4]uint8{255, 0, 0, 255}, 0},
		{"two filters in order: an offset in x, then in y", white + `<filter id="a" ` + region + `><feOffset dx="20"/></filter><filter id="b" ` + region + `><feOffset dy="20"/></filter><rect x="10" y="10" width="10" height="10" fill="#f00" filter="url(#a) url(#b)"/>`, [2]int{35, 35}, [4]uint8{255, 0, 0, 255}, 0},
		{"composite arithmetic in sRGB averages the colors", white + `<filter id="f" color-interpolation-filters="sRGB" ` + region + `><feFlood flood-color="#f00" result="a"/><feFlood flood-color="#00f" result="b"/><feComposite in="a" in2="b" operator="arithmetic" k2="0.5" k3="0.5"/></filter><rect x="10" y="10" width="10" height="10" fill="#000" filter="url(#f)"/>`, [2]int{50, 50}, [4]uint8{128, 0, 128, 255}, 1},
		{"composite arithmetic in linearRGB is brighter", white + `<filter id="f" ` + region + `><feFlood flood-color="#f00" result="a"/><feFlood flood-color="#00f" result="b"/><feComposite in="a" in2="b" operator="arithmetic" k2="0.5" k3="0.5"/></filter><rect x="10" y="10" width="10" height="10" fill="#000" filter="url(#f)"/>`, [2]int{50, 50}, [4]uint8{188, 0, 188, 255}, 2},
		{"drop shadow keeps the element on top", white + `<filter id="f" ` + region + `><feDropShadow dx="10" dy="0" stdDeviation="0" flood-color="#00f"/></filter><rect x="10" y="10" width="10" height="10" fill="#f00" filter="url(#f)"/>`, [2]int{15, 15}, [4]uint8{255, 0, 0, 255}, 0},
		{"drop shadow draws the shadow", white + `<filter id="f" ` + region + `><feDropShadow dx="10" dy="0" stdDeviation="0" flood-color="#00f"/></filter><rect x="10" y="10" width="10" height="10" fill="#f00" filter="url(#f)"/>`, [2]int{25, 15}, [4]uint8{0, 0, 255, 255}, 0},
		{"morphology dilates", white + `<filter id="f" ` + region + `><feMorphology operator="dilate" radius="3"/></filter><rect x="10" y="10" width="10" height="10" fill="#f00" filter="url(#f)"/>`, [2]int{7, 15}, [4]uint8{255, 0, 0, 255}, 0},
		{"morphology erodes", white + `<filter id="f" ` + region + `><feMorphology operator="erode" radius="3"/></filter><rect x="10" y="10" width="10" height="10" fill="#f00" filter="url(#f)"/>`, [2]int{11, 15}, [4]uint8{255, 255, 255, 255}, 0},
		{"a blend of multiply", white + `<filter id="f" ` + region + `><feFlood flood-color="#808080" result="a"/><feBlend in="SourceGraphic" in2="a" mode="multiply"/></filter><rect x="10" y="10" width="10" height="10" fill="#f00" filter="url(#f)"/>`, [2]int{15, 15}, [4]uint8{128, 0, 0, 255}, 1},
		{"a color matrix", white + `<filter id="f" ` + region + `><feColorMatrix type="matrix" values="0 0 1 0 0  0 1 0 0 0  1 0 0 0 0  0 0 0 1 0"/></filter><rect x="10" y="10" width="10" height="10" fill="#f00" filter="url(#f)"/>`, [2]int{15, 15}, [4]uint8{0, 0, 255, 255}, 0},
		{"a component transfer", white + `<filter id="f" ` + region + `><feComponentTransfer><feFuncR type="table" tableValues="1 0"/></feComponentTransfer></filter><rect x="10" y="10" width="10" height="10" fill="#f00" filter="url(#f)"/>`, [2]int{15, 15}, [4]uint8{0, 0, 0, 255}, 0},
		{"the filter applies before the opacity", white + `<filter id="f" ` + region + `><feOffset dx="20"/></filter><rect x="10" y="10" width="10" height="10" fill="#f00" opacity="0.5" filter="url(#f)"/>`, [2]int{35, 15}, [4]uint8{255, 127, 127, 255}, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := docAt(t, tt.body)(tt.at[0], tt.at[1])
			if !nearPx(got, tt.want, tt.tol) {
				t.Errorf("pixel %v = %v, want %v", tt.at, got, tt.want)
			}
		})
	}
}

func TestFilterBlurSpreadsWithTheScale(t *testing.T) {
	body := `<filter id="f" x="-1" y="-1" width="3" height="3"><feGaussianBlur stdDeviation="4"/></filter><rect x="40" y="40" width="20" height="20" fill="#000" filter="url(#f)"/>`
	spread := func(scale float64) int {
		d := mustParse(t, svgOpen+body+`</svg>`)
		img := render(t, d, scale)
		// The distance, in document units, from the box edge to where the blur has faded to under 2%.
		w := img.Bounds().Dx()
		y := w / 2
		far := 0
		for x := 0; x < w/2; x++ {
			if pixel(img, x, y).A > 5 {
				far = int(float64(w/2-x) / scale)
				break
			}
		}
		return far
	}
	one, three := spread(1), spread(3)
	if one < 15 || three < 15 {
		t.Fatalf("the blur is not wide: %d and %d", one, three)
	}
	// At each scale the blur reaches about as far in document units: the deviation is in user units.
	if d := one - three; d < -3 || d > 3 {
		t.Errorf("the blur reaches %d units at scale 1 and %d at scale 3", one, three)
	}
}

func TestFilterOnTurnedElement(t *testing.T) {
	got := docAt(t, `<rect width="100" height="100" fill="#fff"/><filter id="f" filterUnits="userSpaceOnUse" x="0" y="0" width="100" height="100"><feOffset dx="10"/></filter><g transform="rotate(90 50 50)"><rect x="20" y="40" width="20" height="10" fill="#f00" filter="url(#f)"/></g>`)
	// The rectangle turned by a quarter stands at x 50 to 60 and y 20 to 40; the offset of 10 along the turned x
	// axis, which points down, moves it to y 30 to 50.
	if p := got(55, 45); p != [4]uint8{255, 0, 0, 255} {
		t.Errorf("pixel (55, 45) = %v, want red", p)
	}
	if p := got(55, 25); p != [4]uint8{255, 255, 255, 255} {
		t.Errorf("pixel (55, 25) = %v, want the background", p)
	}
}

func TestFilterRegionClipsATurnedElement(t *testing.T) {
	// The region is a rectangle in the turned user space: what lies outside it is cut, even where the box of the
	// region on the device would let it through.
	got := docAt(t, `<rect width="100" height="100" fill="#fff"/><filter id="f" filterUnits="userSpaceOnUse" x="40" y="40" width="20" height="20"><feFlood flood-color="#f00"/></filter><g transform="rotate(45 50 50)"><rect x="0" y="0" width="100" height="100" fill="#00f" filter="url(#f)"/></g>`)
	if p := got(50, 50); p != [4]uint8{255, 0, 0, 255} {
		t.Errorf("center = %v, want red", p)
	}
	// A corner of the box of the turned region is outside the region itself.
	if p := got(38, 38); p[1] != 255 {
		t.Errorf("corner of the box = %v, want the background", p)
	}
}

func TestMarkerVertices(t *testing.T) {
	tests := []struct {
		name string
		d    string
		want int
		// angle is the bisect angle of the vertex idx.
		idx   int
		angle float64
	}{
		{"a line", "M 0 0 L 10 0", 2, 0, 0},
		{"a corner turns half the way", "M 0 0 L 10 0 L 10 10", 3, 1, 45},
		{"the end has the incoming direction", "M 0 0 L 10 0 L 10 10", 3, 2, 90},
		{"curves use the tangents", "M 0 0 C 0 10 10 10 10 0", 2, 0, 90},
		{"an arc is one segment", "M 0 0 A 10 10 0 0 1 20 0 L 20 10", 3, 2, 90},
		{"two arcs", "M 0 0 A 10 10 0 1 1 20 0 A 10 10 0 1 1 0 0", 3, 0, -90},
		{"a closed path has a vertex at the close", "M 0 0 L 10 0 L 10 10 Z", 4, 3, -67.5},
		{"the start of a closed path turns by both sides", "M 0 0 L 10 0 L 10 10 Z", 4, 0, 112.5 - 180},
		{"subpaths each have their ends", "M 0 0 L 10 0 M 20 0 L 30 0", 4, 2, 0},
		{"a move and nothing", "M 5 5", 1, 0, 0},
		{"horizontal and vertical lines", "M 0 0 H 10 V 10", 3, 1, 45},
		{"relative commands", "m 0 0 l 10 0 l 0 10", 3, 1, 45},
		{"a segment with no length has no direction", "M 0 0 L 0 0 L 10 0", 3, 2, 0},
		{"quadratic", "M 0 0 Q 10 0 10 10", 2, 1, 90},
		{"smooth curves", "M 0 0 C 0 10 10 10 10 0 S 20 -10 20 0", 3, 1, -90},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vc := newVertexCollector()
			appendPathData(ui.NewPath(), tt.d, vc)
			if len(vc.verts) != tt.want {
				t.Fatalf("%d vertices, want %d: %+v", len(vc.verts), tt.want, vc.verts)
			}
			if got := vc.verts[tt.idx].bisect(); got-tt.angle > 0.5 && got-tt.angle < 359.5 || got-tt.angle < -0.5 && got-tt.angle > -359.5 {
				t.Errorf("angle of vertex %d = %.1f, want %.1f", tt.idx, got, tt.angle)
			}
		})
	}
}

func TestMarkerDrawing(t *testing.T) {
	const white = `<rect width="100" height="100" fill="#fff"/>`
	box := func(attrs string) string {
		return `<marker id="m" markerWidth="10" markerHeight="10" refX="5" refY="5" ` + attrs + `><rect width="10" height="10" fill="#f00"/></marker>`
	}
	bg := [4]uint8{255, 255, 255, 255}
	red := [4]uint8{255, 0, 0, 255}
	tests := []struct {
		name string
		body string
		at   [2]int
		want [4]uint8
	}{
		{"the reference point is on the vertex", white + `<defs>` + box(`markerUnits="userSpaceOnUse"`) + `</defs><path d="M 20 20 L 80 20" stroke="none" fill="none" marker-start="url(#m)"/>`, [2]int{16, 16}, red},
		{"the marker covers its size around the vertex", white + `<defs>` + box(`markerUnits="userSpaceOnUse"`) + `</defs><path d="M 20 20 L 80 20" stroke="none" fill="none" marker-start="url(#m)"/>`, [2]int{26, 20}, bg},
		{"marker-end is at the last vertex", white + `<defs>` + box(`markerUnits="userSpaceOnUse"`) + `</defs><path d="M 20 20 L 80 20" stroke="none" fill="none" marker-end="url(#m)"/>`, [2]int{80, 20}, red},
		{"marker-end is not at the first vertex", white + `<defs>` + box(`markerUnits="userSpaceOnUse"`) + `</defs><path d="M 20 20 L 80 20" stroke="none" fill="none" marker-end="url(#m)"/>`, [2]int{20, 20}, bg},
		{"marker-mid is at the vertices between", white + `<defs>` + box(`markerUnits="userSpaceOnUse"`) + `</defs><path d="M 20 20 L 50 50 L 80 20" stroke="none" fill="none" marker-mid="url(#m)"/>`, [2]int{50, 50}, red},
		{"marker-mid is not at the ends", white + `<defs>` + box(`markerUnits="userSpaceOnUse"`) + `</defs><path d="M 20 20 L 50 50 L 80 20" stroke="none" fill="none" marker-mid="url(#m)"/>`, [2]int{20, 20}, bg},
		{"the stroke width scales the marker", white + `<defs>` + box(`markerUnits="strokeWidth"`) + `</defs><path d="M 50 50 L 80 50" stroke="#000" stroke-width="4" marker-start="url(#m)"/>`, [2]int{34, 40}, red},
		{"the stroke width scales the marker: beyond it", white + `<defs>` + box(`markerUnits="strokeWidth"`) + `</defs><path d="M 50 50 L 80 50" stroke="#000" stroke-width="4" marker-start="url(#m)"/>`, [2]int{28, 40}, bg},
		{"userSpaceOnUse ignores the stroke width", white + `<defs>` + box(`markerUnits="userSpaceOnUse"`) + `</defs><path d="M 50 50 L 80 50" stroke="#000" stroke-width="4" marker-start="url(#m)"/>`, [2]int{34, 40}, bg},
		{"orient auto turns the marker", white + `<defs><marker id="m" markerWidth="20" markerHeight="4" refX="0" refY="2" orient="auto" markerUnits="userSpaceOnUse"><rect width="20" height="4" fill="#f00"/></marker></defs><path d="M 50 20 L 50 80" stroke="none" fill="none" marker-start="url(#m)"/>`, [2]int{50, 30}, red},
		{"orient auto: not along the old direction", white + `<defs><marker id="m" markerWidth="20" markerHeight="4" refX="0" refY="2" orient="auto" markerUnits="userSpaceOnUse"><rect width="20" height="4" fill="#f00"/></marker></defs><path d="M 50 20 L 50 80" stroke="none" fill="none" marker-start="url(#m)"/>`, [2]int{65, 22}, bg},
		{"a fixed angle", white + `<defs><marker id="m" markerWidth="20" markerHeight="4" refX="0" refY="2" orient="90" markerUnits="userSpaceOnUse"><rect width="20" height="4" fill="#f00"/></marker></defs><path d="M 50 20 L 80 20" stroke="none" fill="none" marker-start="url(#m)"/>`, [2]int{50, 30}, red},
		{"auto-start-reverse turns the start marker around", white + `<defs><marker id="m" markerWidth="20" markerHeight="4" refX="0" refY="2" orient="auto-start-reverse" markerUnits="userSpaceOnUse"><rect width="20" height="4" fill="#f00"/></marker></defs><path d="M 50 20 L 80 20" stroke="none" fill="none" marker-start="url(#m)"/>`, [2]int{40, 20}, red},
		{"auto-start-reverse leaves the end marker", white + `<defs><marker id="m" markerWidth="20" markerHeight="4" refX="0" refY="2" orient="auto-start-reverse" markerUnits="userSpaceOnUse"><rect width="20" height="4" fill="#f00"/></marker></defs><path d="M 50 20 L 80 20" stroke="none" fill="none" marker-end="url(#m)"/>`, [2]int{90, 20}, red},
		{"the viewBox scales the content", white + `<defs><marker id="m" markerWidth="10" markerHeight="10" viewBox="0 0 2 2" refX="1" refY="1" markerUnits="userSpaceOnUse"><rect width="2" height="2" fill="#f00"/></marker></defs><path d="M 50 50 L 80 50" stroke="none" fill="none" marker-start="url(#m)"/>`, [2]int{46, 46}, red},
		{"the content is clipped to the marker viewport", white + `<defs><marker id="m" markerWidth="4" markerHeight="4" refX="2" refY="2" markerUnits="userSpaceOnUse"><rect x="-10" y="-10" width="30" height="30" fill="#f00"/></marker></defs><path d="M 50 50 L 80 50" stroke="none" fill="none" marker-start="url(#m)"/>`, [2]int{58, 50}, bg},
		{"overflow visible does not clip", white + `<defs><marker id="m" markerWidth="4" markerHeight="4" refX="2" refY="2" markerUnits="userSpaceOnUse" overflow="visible"><rect x="-10" y="-10" width="30" height="30" fill="#f00"/></marker></defs><path d="M 50 50 L 80 50" stroke="none" fill="none" marker-start="url(#m)"/>`, [2]int{58, 50}, red},
		{"a rect has no markers", white + `<defs>` + box(`markerUnits="userSpaceOnUse"`) + `</defs><rect x="30" y="30" width="40" height="40" fill="none" marker-start="url(#m)" marker-mid="url(#m)" marker-end="url(#m)"/>`, [2]int{30, 30}, bg},
		{"markers inherit through a group", white + `<defs>` + box(`markerUnits="userSpaceOnUse"`) + `</defs><g marker-end="url(#m)"><path d="M 20 20 L 80 20" stroke="none" fill="none"/></g>`, [2]int{80, 20}, red},
		{"a polyline", white + `<defs>` + box(`markerUnits="userSpaceOnUse"`) + `</defs><polyline points="20,20 50,50 80,20" fill="none" marker-mid="url(#m)"/>`, [2]int{50, 50}, red},
		{"a line", white + `<defs>` + box(`markerUnits="userSpaceOnUse"`) + `</defs><line x1="20" y1="20" x2="80" y2="20" marker-end="url(#m)"/>`, [2]int{80, 20}, red},
		{"a polygon has the closing vertex", white + `<defs>` + box(`markerUnits="userSpaceOnUse"`) + `</defs><polygon points="20,20 80,20 80,80" fill="none" marker-mid="url(#m)"/>`, [2]int{80, 20}, red},
		{"paint-order: markers first, then the stroke over them", white + `<defs>` + box(`markerUnits="userSpaceOnUse"`) + `</defs><path d="M 20 50 L 80 50" stroke="#00f" stroke-width="6" marker-start="url(#m)" paint-order="markers stroke fill"/>`, [2]int{25, 50}, [4]uint8{0, 0, 255, 255}},
		{"paint-order by default: markers over the stroke", white + `<defs>` + box(`markerUnits="userSpaceOnUse"`) + `</defs><path d="M 20 50 L 80 50" stroke="#00f" stroke-width="6" marker-start="url(#m)"/>`, [2]int{22, 50}, red},
		{"opacity covers the markers with the shape", white + `<defs>` + box(`markerUnits="userSpaceOnUse"`) + `</defs><path d="M 20 50 L 80 50" stroke="#00f" stroke-width="6" opacity="0.5" marker-start="url(#m)"/>`, [2]int{22, 50}, [4]uint8{255, 127, 127, 255}},
		{"a marker that does not exist is ignored", white + `<path d="M 20 20 L 80 20" stroke="#000" stroke-width="4" marker-end="url(#nothing)"/>`, [2]int{50, 20}, [4]uint8{0, 0, 0, 255}},
		{"marker: none", white + `<defs>` + box(`markerUnits="userSpaceOnUse"`) + `</defs><g marker-end="url(#m)"><path d="M 20 20 L 80 20" marker-end="none"/></g>`, [2]int{80, 20}, bg},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := docAt(t, tt.body)(tt.at[0], tt.at[1])
			if !nearPx(got, tt.want, 3) {
				t.Errorf("pixel %v = %v, want %v", tt.at, got, tt.want)
			}
		})
	}
}

func TestMarkerCountIsBounded(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("M 0 0")
	for i := 0; i < 3*maxMarkers; i++ {
		sb.WriteString(" l 0.001 0")
	}
	d := mustParse(t, svgOpen+`<defs><marker id="m" markerWidth="1" markerHeight="1"><rect width="1" height="1"/></marker></defs><path d="`+sb.String()+`" marker-mid="url(#m)"/></svg>`)
	// Each marker clips and draws: a clip and a rect at least; the pictures grows with the count, which is capped.
	ops := len(opsOf(d))
	if ops > 20*maxMarkers {
		t.Errorf("%d operations for a path of %d vertices", ops, 3*maxMarkers)
	}
	if ops < maxMarkers {
		t.Errorf("only %d operations: markers were not drawn", ops)
	}
}
