package viewport3d

import (
	"math"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

// labelKey tells the labels of a cache apart.
type labelKey struct {
	text  string
	color ui.Color
}

// labels are the text painters of the labels that the axes and the view cube draw, laid out once each: a label is made when it is
// first drawn and kept until the overlay is disposed.
type labels struct {
	engine ui.ParagraphFactory
	// base is the text style of the surroundings (the font), which the labels add their color, size and weight to.
	base     *painting.TextStyle
	size     float64
	weight   ui.FontWeight
	painters map[labelKey]*painting.TextPainter
}

func newLabels(engine ui.ParagraphFactory, base *painting.TextStyle, size float64, weight ui.FontWeight) *labels {
	return &labels{engine: engine, base: base, size: size, weight: weight, painters: map[labelKey]*painting.TextPainter{}}
}

// setBase changes the style of the surroundings; the labels are made again.
func (l *labels) setBase(base *painting.TextStyle) {
	if base != l.base {
		l.dispose()
		l.base = base
	}
}

// get returns the painter of a text in a color, laid out.
func (l *labels) get(text string, color ui.Color) *painting.TextPainter {
	k := labelKey{text, color}
	if p, ok := l.painters[k]; ok {
		return p
	}
	p := painting.NewTextPainter(painting.TextPainterOpts{
		TextEngine: l.engine, TextAlign: ui.TextAlignCenter, TextDirection: ui.TextDirectionLTR,
		Text: painting.NewTextSpan(painting.TextSpanOpts{
			Text:  foundation.Some(text),
			Style: l.base.Merge(painting.NewTextStyle(painting.TextStyleOpts{Color: foundation.Some(color), FontSize: foundation.Some(l.size), FontWeight: l.weight})),
		}),
	})
	p.Layout(0, math.Inf(1))
	l.painters[k] = p
	return p
}

// paintCentered draws a label with its center at a point.
func (l *labels) paintCentered(canvas *ui.Canvas, text string, color ui.Color, at ui.Offset) {
	p := l.get(text, color)
	p.Paint(canvas, ui.Offset{Dx: at.Dx - p.Width()/2, Dy: at.Dy - p.Height()/2})
}

// dispose releases the painters.
func (l *labels) dispose() {
	for k, p := range l.painters {
		p.Dispose()
		delete(l.painters, k)
	}
}
