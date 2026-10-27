package svg

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/ui"
)

// paintedPixels renders a document into an image of w x h and counts the pixels that are not transparent.
func paintedPixels(t *testing.T, d *Document, w, h int) int {
	t.Helper()
	img := renderDocument(t, d, w, h)
	n := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if c := img.RGBAAt(x, y); c.A != 0 {
				n++
			}
		}
	}
	return n
}

func TestFilterRobustness(t *testing.T) {
	prims := func(n int) string {
		return strings.Repeat(`<feOffset dx="0.1" dy="0"/>`, n)
	}
	tests := []struct {
		name string
		defs string
		// wantPainted says whether something must be drawn: 1 yes, 0 no, -1 either (the result is bounded, and what it
		// shows is not specified).
		wantPainted int
	}{
		{"a filter region of a million user units", `<filter id="f" x="-500000" y="-500000" width="1000000" height="1000000"><feGaussianBlur stdDeviation="3"/></filter>`, -1},
		{"a blur of a huge deviation", `<filter id="f" x="-5" y="-5" width="10" height="10"><feGaussianBlur stdDeviation="100000000"/></filter>`, -1},
		{"a morphology of a huge radius", `<filter id="f" x="-5" y="-5" width="10" height="10"><feMorphology operator="dilate" radius="100000000"/></filter>`, 1},
		{"a chain of 90 primitives", `<filter id="f" x="-1" y="-1" width="3" height="3">` + prims(90) + `</filter>`, 1},
		{"a graph of 5000 primitives is not drawn", `<filter id="f">` + prims(5000) + `</filter>`, 0},
		{"a negative deviation leaves the element out", `<filter id="f"><feGaussianBlur stdDeviation="-1"/></filter>`, 0},
		{"a filter with no primitive leaves the element out", `<filter id="f"></filter>`, 0},
		{"a filter with an unknown primitive is skipped", `<filter id="f"><feTurbulence baseFrequency="0.1"/></filter>`, 1},
		{"a result that nothing is named", `<filter id="f"><feOffset in="nothing" dx="1"/></filter>`, 1},
		{"a region with no width", `<filter id="f" x="0" y="0" width="0" height="1"><feOffset/></filter>`, 0},
		{"NaN-like values", `<filter id="f"><feGaussianBlur stdDeviation="1e999"/></filter>`, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := mustParse(t, svgOpen+`<defs>`+tt.defs+`</defs><rect x="20" y="20" width="40" height="40" fill="#f00" filter="url(#f)"/></svg>`)
			var before, after runtime.MemStats
			runtime.ReadMemStats(&before)
			start := time.Now()
			n := paintedPixels(t, d, 100, 100)
			elapsed := time.Since(start)
			runtime.ReadMemStats(&after)
			if elapsed > 10*time.Second {
				t.Errorf("took %v", elapsed)
			}
			if mb := (after.TotalAlloc - before.TotalAlloc) >> 20; mb > 900 {
				t.Errorf("allocated %d MB", mb)
			}
			if got := n > 0; tt.wantPainted >= 0 && got != (tt.wantPainted == 1) {
				t.Errorf("painted pixels = %d, want painted %d", n, tt.wantPainted)
			}
		})
	}
}

func TestFilterInsideFilterIsBounded(t *testing.T) {
	// Each group has a filter and holds the next group: the source graphic of a filter is drawn through the filters
	// inside it, but only so deep.
	var sb strings.Builder
	sb.WriteString(svgOpen + `<defs><filter id="f" x="-1" y="-1" width="3" height="3"><feOffset dx="1"/></filter></defs>`)
	for i := 0; i < 40; i++ {
		sb.WriteString(`<g filter="url(#f)">`)
	}
	sb.WriteString(`<rect x="10" y="10" width="20" height="20" fill="#00f"/>`)
	sb.WriteString(strings.Repeat(`</g>`, 40) + `</svg>`)
	d := mustParse(t, sb.String())
	start := time.Now()
	paintedPixels(t, d, 100, 100)
	if time.Since(start) > 10*time.Second {
		t.Errorf("took %v", time.Since(start))
	}
}

func TestMasksThatReferToThemselves(t *testing.T) {
	tests := []struct {
		name string
		defs string
	}{
		{"a mask whose content uses the mask", `<mask id="m"><rect width="100" height="100" fill="#fff" mask="url(#m)"/></mask>`},
		{"two masks that use each other", `<mask id="m"><rect width="100" height="100" fill="#fff" mask="url(#n)"/></mask><mask id="n"><rect width="100" height="100" fill="#fff" mask="url(#m)"/></mask>`},
		{"a mask with a mask that is itself", `<mask id="m" mask="url(#m)"><rect width="100" height="100" fill="#fff"/></mask>`},
		{"a marker that uses itself", `<marker id="m" markerWidth="10" markerHeight="10"><path d="M0 0 L10 10" stroke="#000" marker-end="url(#m)"/></marker>`},
		{"a filter inside a mask that uses the mask", `<filter id="f"><feOffset/></filter><mask id="m"><rect width="100" height="100" fill="#fff" filter="url(#f)" mask="url(#m)"/></mask>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := mustParse(t, svgOpen+`<defs>`+tt.defs+`</defs><rect width="60" height="60" fill="#f00" mask="url(#m)" marker-end="url(#m)"/><path d="M10 80 L90 80" stroke="#00f" marker-start="url(#m)" marker-end="url(#m)"/></svg>`)
			done := make(chan struct{})
			go func() {
				defer close(done)
				d.PictureAt(1)
			}()
			select {
			case <-done:
			case <-time.After(10 * time.Second):
				t.Fatal("drawing did not end")
			}
		})
	}
}

func TestHugeTextIsBounded(t *testing.T) {
	text := strings.Repeat("Lorem ipsum dolor sit amet ", 5000) // 135,000 characters
	d := mustParse(t, svgOpen+`<text x="1" y="50" font-family="Roboto" font-size="4">`+text+`</text></svg>`)
	n := mustFind2(t, d)
	start := time.Now()
	p := newPainter(ui.NewCanvas(ui.NewPictureRecorder()), 1)
	p.doc = d
	tb := p.textBlockOf(n)
	if tb == nil {
		t.Fatal("no layout")
	}
	if len(tb.items) == 0 || len(tb.items) > maxTextRunes {
		t.Errorf("%d clusters, want between 1 and %d", len(tb.items), maxTextRunes)
	}
	if !raceEnabled && time.Since(start) > 20*time.Second {
		t.Errorf("took %v", time.Since(start))
	}
	t.Logf("%d characters laid out in %v", len(tb.items), time.Since(start))
}

func mustFind2(t *testing.T, d *Document) *Node {
	t.Helper()
	var found *Node
	var walk func(n *Node)
	walk = func(n *Node) {
		if n.Kind == NodeText && found == nil {
			found = n
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(d.Root)
	if found == nil {
		t.Fatal("no text node")
	}
	return found
}

func TestMarkersOnHugePathAreBounded(t *testing.T) {
	var sb strings.Builder
	sb.WriteString(`M 0 0`)
	for i := 0; i < 300000; i++ {
		fmt.Fprintf(&sb, " L %d %d", i%90, (i/90)%90)
	}
	d := mustParse(t, svgOpen+`<defs><marker id="m" markerWidth="2" markerHeight="2"><rect width="2" height="2"/></marker></defs><path d="`+sb.String()+`" fill="none" stroke="#000" stroke-width="0.01" marker-mid="url(#m)"/></svg>`)
	start := time.Now()
	d.PictureAt(1)
	if time.Since(start) > 20*time.Second {
		t.Errorf("took %v", time.Since(start))
	}
}
