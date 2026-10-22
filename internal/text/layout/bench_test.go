package layout

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/internal/text"
	"github.com/rm4n0s/glutter/material/fonts"
	"github.com/rm4n0s/glutter/ui"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/gofont/gomonoitalic"
)

// The benchmarks of a code editor's text: a line of 100 columns in a monospace font, in 5, 10
// or 20 styled spans (keywords, names, strings and comments), as a syntax highlighter makes them.

const codeColumns = 100

// codeSource is what the lines are cut from; it is long enough for every line.
const codeSource = `func (e *Engine) glyphPath(face *font.Face, gid font.GID, size float64) *ui.Path { // the outline of a glyph, cached ` +
	`if p, ok := e.paths.get(key); ok { return p } var path *ui.Path = nil for i := 0; i < len(segments); i++ { x, y := pt(i) } `

// codeStyles are the styles of the spans, in turn: text, a keyword (bold), a string, a comment
// (italic), a number.
var codeStyles = []ui.TextStyleOpts{
	{Color: ptrColor(0xFF212121)},
	{Color: ptrColor(0xFF0033B3), FontWeight: ui.FontWeightW700},
	{Color: ptrColor(0xFF067D17)},
	{Color: ptrColor(0xFF8C8C8C), FontStyle: ui.FontStyleItalic},
	{Color: ptrColor(0xFF1750EB)},
}

func ptrColor(v uint32) *ui.Color { c := ui.NewColor(v); return &c }

// codeEngine is a text engine with Go Mono in regular, bold and italic.
func codeEngine(tb testing.TB) *Engine {
	tb.Helper()
	c := text.NewCollection(text.CollectionOpts{Matcher: noSystemFonts{}})
	if err := fonts.Register(c); err != nil {
		tb.Fatal(err)
	}
	e := NewEngine(context.Background(), c)
	for _, data := range [][]byte{gomono.TTF, gomonobold.TTF, gomonoitalic.TTF} {
		if err := e.LoadFontFromList(data, "Go Mono"); err != nil {
			tb.Fatal(err)
		}
	}
	return e
}

// codeSpans cuts a line of codeColumns characters, starting at offset into codeSource, into n spans
// of nearly the same length.
func codeSpans(offset, n int) (spans []string) {
	line := (codeSource + codeSource)[offset : offset+codeColumns]
	for i := 0; i < n; i++ {
		spans = append(spans, line[i*codeColumns/n:(i+1)*codeColumns/n])
	}
	return spans
}

// buildCode builds the paragraph of a line of code.
func buildCode(e *Engine, spans []string) *ui.Paragraph {
	b := ui.NewParagraphBuilder(e, ui.NewParagraphStyle(ui.ParagraphStyleOpts{FontFamily: "Go Mono", FontSize: ui.Ptr(14.0), MaxLines: ui.Ptr(1)}))
	for i, s := range spans {
		st := codeStyles[i%len(codeStyles)]
		st.FontFamily = "Go Mono"
		b.PushStyle(ui.NewTextStyle(st))
		b.AddText(s)
		b.Pop()
	}
	return b.Build()
}

var spanCounts = []int{5, 10, 20}

func BenchmarkCodeLineBuild(b *testing.B) {
	for _, n := range spanCounts {
		b.Run(fmt.Sprintf("spans=%d", n), func(b *testing.B) {
			e := codeEngine(b)
			spans := codeSpans(0, n)
			buildCode(e, spans) // the first build loads the fonts
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				buildCode(e, spans)
			}
		})
	}
}

func BenchmarkCodeLineLayout(b *testing.B) {
	for _, n := range spanCounts {
		b.Run(fmt.Sprintf("spans=%d", n), func(b *testing.B) {
			e := codeEngine(b)
			p := buildCode(e, codeSpans(0, n))
			p.Layout(ui.NewParagraphConstraints(2000))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				p.Layout(ui.NewParagraphConstraints(2000 + float64(i%2))) // another width, so that it is laid out again
			}
		})
	}
}

func BenchmarkCodeLinePaint(b *testing.B) {
	for _, n := range spanCounts {
		b.Run(fmt.Sprintf("spans=%d", n), func(b *testing.B) {
			e := codeEngine(b)
			p := buildCode(e, codeSpans(0, n))
			p.Layout(ui.NewParagraphConstraints(2000))
			rec := ui.NewPictureRecorder()
			c := ui.NewCanvas(rec, ui.CanvasOpts{CullRect: ui.Ptr(ui.NewRectFromLTWH(0, 0, 2000, 40))})
			c.DrawParagraph(p, ui.Offset{}) // the first paint makes the outlines
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				rec := ui.NewPictureRecorder()
				c := ui.NewCanvas(rec, ui.CanvasOpts{CullRect: ui.Ptr(ui.NewRectFromLTWH(0, 0, 2000, 40))})
				c.DrawParagraph(p, ui.Offset{})
				rec.EndRecording()
			}
		})
	}
}

// BenchmarkCodeLine is the whole way of a line of code that changed: build, layout and paint.
func BenchmarkCodeLine(b *testing.B) {
	for _, n := range spanCounts {
		b.Run(fmt.Sprintf("spans=%d", n), func(b *testing.B) {
			e := codeEngine(b)
			spans := codeSpans(0, n)
			p := buildCode(e, spans)
			p.Layout(ui.NewParagraphConstraints(2000))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				p := buildCode(e, spans)
				p.Layout(ui.NewParagraphConstraints(2000))
				rec := ui.NewPictureRecorder()
				c := ui.NewCanvas(rec, ui.CanvasOpts{CullRect: ui.Ptr(ui.NewRectFromLTWH(0, 0, 2000, 40))})
				c.DrawParagraph(p, ui.Offset{})
				rec.EndRecording()
			}
		})
	}
}

const pageLines = 50

// BenchmarkCodePage is a page of 50 lines, each of 10 spans, built, laid out and painted, as the
// first frame of a code view or a frame after a big change is.
func BenchmarkCodePage(b *testing.B) {
	e := codeEngine(b)
	page := func() *ui.Picture {
		rec := ui.NewPictureRecorder()
		c := ui.NewCanvas(rec, ui.CanvasOpts{CullRect: ui.Ptr(ui.NewRectFromLTWH(0, 0, 900, pageLines*20))})
		for i := 0; i < pageLines; i++ {
			p := buildCode(e, codeSpans(i*7, 10))
			p.Layout(ui.NewParagraphConstraints(900))
			c.DrawParagraph(p, ui.Offset{Dy: float64(i) * 20})
		}
		return rec.EndRecording()
	}
	page()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		page()
	}
}

// BenchmarkCodePageRaster is the software renderer drawing the page (a scene of 50 paragraphs): not a
// GPU number, but it shows how much a page of text costs to rasterize.
func BenchmarkCodePageRaster(b *testing.B) {
	e := codeEngine(b)
	rec := ui.NewPictureRecorder()
	c := ui.NewCanvas(rec, ui.CanvasOpts{CullRect: ui.Ptr(ui.NewRectFromLTWH(0, 0, 900, pageLines*20))})
	c.DrawColor(ui.NewColor(0xFFFFFFFF), ui.BlendModeSrcOver)
	for i := 0; i < pageLines; i++ {
		p := buildCode(e, codeSpans(i*7, 10))
		p.Layout(ui.NewParagraphConstraints(900))
		c.DrawParagraph(p, ui.Offset{Dy: float64(i) * 20})
	}
	sb := ui.NewSceneBuilder()
	sb.AddPicture(ui.Offset{}, rec.EndRecording())
	scene := sb.Build()
	if _, err := soft.Render(scene, 900, pageLines*20); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := soft.Render(scene, 900, pageLines*20); err != nil {
			b.Fatal(err)
		}
	}
}

// TestCodeLineBenchmarkInput checks what the benchmarks measure: a line is 100 columns in the
// monospace font, in n spans, on one line.
func TestCodeLineBenchmarkInput(t *testing.T) {
	e := codeEngine(t)
	for _, n := range spanCounts {
		spans := codeSpans(13, n)
		if len(spans) != n || len(strings.Join(spans, "")) != codeColumns {
			t.Fatalf("%d spans of %d characters, want %d spans of %d", len(spans), len(strings.Join(spans, "")), n, codeColumns)
		}
		p := buildCode(e, spans)
		p.Layout(ui.NewParagraphConstraints(2000))
		advance := 1229.0 / 2048.0 * 14
		if want := float64(codeColumns) * advance; p.LongestLine() < want-0.5 || p.LongestLine() > want+0.5 {
			t.Errorf("a line of %d spans is %v wide, want %v (100 columns of Go Mono)", n, p.LongestLine(), want)
		}
		if p.Height() > 25 {
			t.Errorf("the line wrapped: height %v", p.Height())
		}
	}
}
