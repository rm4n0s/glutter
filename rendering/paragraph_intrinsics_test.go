package rendering_test

import (
	"math"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/material/fonts"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
)

func TestListBodyAndParagraphIntrinsics(t *testing.T) {
	span := painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some("Hello World"), Style: painting.NewTextStyle(painting.TextStyleOpts{
		FontFamily: fonts.RobotoFamily, FontSize: foundation.Some(14.0), Height: foundation.Some(1.0)})})
	paragraph := newParagraph(span)
	block := rendering.NewRenderListBody([]rendering.Box{paragraph})

	textWidth := paragraph.GetMaxIntrinsicWidth(inf)
	oneLine := paragraph.GetMinIntrinsicHeight(inf)
	constrained := textWidth * 0.9
	wrappedWidth := paragraph.GetMinIntrinsicWidth(inf)
	twoLines := paragraph.GetMinIntrinsicHeight(constrained)
	manyLines := paragraph.GetMinIntrinsicHeight(0)

	if !(wrappedWidth > 0 && wrappedWidth < textWidth && oneLine < twoLines && twoLines < oneLine*3 && manyLines > twoLines) {
		t.Fatalf("widths %v, %v; heights %v, %v, %v", wrappedWidth, textWidth, oneLine, twoLines, manyLines)
	}
	for _, tc := range []struct {
		width, want float64
	}{{inf, oneLine}, {constrained, twoLines}, {0, manyLines}} {
		if got := paragraph.GetMaxIntrinsicHeight(tc.width); got != tc.want {
			t.Errorf("paragraph max intrinsic height at %v = %v, want %v", tc.width, got, tc.want)
		}
	}
	for _, direction := range []painting.AxisDirection{painting.AxisDirectionDown, painting.AxisDirectionRight} {
		block.SetAxisDirection(direction)
		checks := []struct {
			name      string
			got, want float64
		}{
			{"min width at infinity", block.GetMinIntrinsicWidth(inf), wrappedWidth},
			{"max width at infinity", block.GetMaxIntrinsicWidth(inf), textWidth},
			{"min height at infinity", block.GetMinIntrinsicHeight(inf), oneLine},
			{"min height, constrained", block.GetMinIntrinsicHeight(constrained), twoLines},
			{"max height at infinity", block.GetMaxIntrinsicHeight(inf), oneLine},
			{"max height, constrained", block.GetMaxIntrinsicHeight(constrained), twoLines},
			{"min width at 0", block.GetMinIntrinsicWidth(0), wrappedWidth},
			{"max width at 0", block.GetMaxIntrinsicWidth(0), textWidth},
			{"min height, wrapped", block.GetMinIntrinsicHeight(wrappedWidth), paragraph.GetMinIntrinsicHeight(wrappedWidth)},
			{"max height, wrapped", block.GetMaxIntrinsicHeight(wrappedWidth), paragraph.GetMaxIntrinsicHeight(wrappedWidth)},
			{"min height at 0", block.GetMinIntrinsicHeight(0), manyLines},
			{"max height at 0", block.GetMaxIntrinsicHeight(0), manyLines},
		}
		for _, c := range checks {
			if c.got != c.want {
				t.Errorf("%v: %s = %v, want %v", direction, c.name, c.got, c.want)
			}
		}
	}
}

func TestParagraphIntrinsicsDependOnTheProperties(t *testing.T) {
	t.Run("textScaler", func(t *testing.T) {
		p := newParagraph(textSpan("Hello World", 10))
		before := p.GetMaxIntrinsicWidth(inf)
		p.SetTextScaler(painting.NewTextScalerLinear(2))
		if got := p.GetMaxIntrinsicWidth(inf); math.Abs(got-2*before) > 0.01*before {
			t.Errorf("scaled width = %v, want about %v", got, 2*before)
		}
	})
	t.Run("maxLines", func(t *testing.T) {
		p := newParagraph(textSpan(strings.Join([]string{"A", "A", "A", "A", "A"}, "\n"), 10))
		line := p.GetMaxIntrinsicHeight(inf) / 5
		p.SetMaxLines(foundation.Some(1))
		if got := p.GetMaxIntrinsicHeight(inf); math.Abs(got-line) > 1e-9 {
			t.Errorf("height with one line = %v, want %v", got, line)
		}
	})
	t.Run("strutStyle", func(t *testing.T) {
		p := newParagraph(textSpan("Hello World", 10))
		if p.GetMaxIntrinsicHeight(inf) >= 100 {
			t.Fatal("the text is not smaller than the strut")
		}
		p.SetStrutStyle(painting.NewStrutStyle(painting.StrutStyleOpts{FontFamily: fonts.RobotoFamily, FontSize: foundation.Some(100.0), ForceStrutHeight: foundation.Some(true)}))
		if got := p.GetMaxIntrinsicHeight(inf); got < 100 {
			t.Errorf("height with a strut of 100 = %v, want at least 100", got)
		}
	})
}
