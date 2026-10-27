package layout

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func TestClusters(t *testing.T) {
	e := bundledEngine(t)
	size := 20.0
	roboto := ui.NewTextStyle(ui.TextStyleOpts{FontFamily: "Roboto", FontSize: &size})
	tests := []struct {
		name string
		text string
		// groups is the number of clusters wanted.
		groups int
	}{
		{"one letter", "a", 1},
		{"letters", "abc", 3},
		{"a space is a cluster", "a b", 3},
		{"a letter and its accent are one", "é", 1},
		{"emoji with a modifier", "\U0001F44D\U0001F3FD", 1},
		{"empty", "", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := build(e, tt.text, roboto, nil)
			cl, ok := p.(Clusterer)
			if !ok {
				t.Fatal("the paragraph is not a Clusterer")
			}
			cs := cl.Clusters()
			if len(cs) != tt.groups {
				t.Fatalf("%d clusters, want %d: %+v", len(cs), tt.groups, cs)
			}
			end := 0
			for i, c := range cs {
				if c.Start != end || c.End <= c.Start {
					t.Errorf("cluster %d covers %d..%d, want it to start at %d", i, c.Start, c.End, end)
				}
				end = c.End
				if c.X1 < c.X0 {
					t.Errorf("cluster %d has a negative advance", i)
				}
			}
			if tt.groups > 0 && end != len(tt.text) {
				t.Errorf("clusters end at %d of %d bytes", end, len(tt.text))
			}
		})
	}
	t.Run("the advances add up to the width of the paragraph", func(t *testing.T) {
		p := build(e, "Hello world", roboto, nil)
		cs := p.(Clusterer).Clusters()
		sum := 0.0
		for _, c := range cs {
			sum += c.X1 - c.X0
		}
		if w := p.(*paragraph).MaxIntrinsicWidth(); math.Abs(sum-w) > 1e-6 {
			t.Errorf("advances add to %v, the paragraph is %v wide", sum, w)
		}
		for i := 1; i < len(cs); i++ {
			if math.Abs(cs[i].X0-cs[i-1].X1) > 1e-6 {
				t.Errorf("cluster %d starts at %v but the one before ends at %v", i, cs[i].X0, cs[i-1].X1)
			}
		}
	})
	t.Run("glyphs have outlines at the baseline", func(t *testing.T) {
		cs := build(e, "Hx", roboto, nil).(Clusterer).Clusters()
		for i, c := range cs {
			if len(c.Glyphs) != 1 || c.Glyphs[0].Path == nil || c.Glyphs[0].Path.IsEmpty() {
				t.Fatalf("cluster %d has no outline", i)
			}
			if c.Glyphs[0].Origin.Dy != c.Baseline {
				t.Errorf("origin y = %v, baseline = %v", c.Glyphs[0].Origin.Dy, c.Baseline)
			}
			if c.Ascent <= 0 || c.Descent <= 0 || c.Underline <= 0 || c.UnderlineThickness <= 0 || c.Strike >= 0 {
				t.Errorf("metrics of cluster %d: %+v", i, c)
			}
		}
	})
	t.Run("a space has a cluster but no outline", func(t *testing.T) {
		cs := build(e, " ", roboto, nil).(Clusterer).Clusters()
		if len(cs) != 1 || len(cs[0].Glyphs) != 0 || cs[0].X1 <= cs[0].X0 {
			t.Errorf("clusters = %+v", cs)
		}
	})
	t.Run("right to left text is in logical order with the first on the right", func(t *testing.T) {
		cs := build(e, "אבג", roboto, nil).(Clusterer).Clusters()
		if len(cs) != 3 {
			t.Fatalf("%d clusters", len(cs))
		}
		for i := 1; i < len(cs); i++ {
			if !(cs[i].X0 < cs[i-1].X0) || !cs[i].RTL {
				t.Errorf("cluster %d: X0 %v after %v, RTL %v", i, cs[i].X0, cs[i-1].X0, cs[i].RTL)
			}
		}
	})
	t.Run("a style with letter spacing widens each cluster", func(t *testing.T) {
		ls := 4.0
		spaced := ui.NewTextStyle(ui.TextStyleOpts{FontFamily: "Roboto", FontSize: &size, LetterSpacing: &ls})
		a := build(e, "ab", roboto, nil).(Clusterer).Clusters()
		b := build(e, "ab", spaced, nil).(Clusterer).Clusters()
		if d := (b[0].X1 - b[0].X0) - (a[0].X1 - a[0].X0); math.Abs(d-4) > 1e-6 {
			t.Errorf("the advance grew by %v, want 4", d)
		}
	})
	t.Run("clusters of two styles", func(t *testing.T) {
		big := 40.0
		b := ui.NewParagraphBuilder(e, nil)
		b.PushStyle(roboto)
		b.AddText("a")
		b.Pop()
		b.PushStyle(ui.NewTextStyle(ui.TextStyleOpts{FontFamily: "Roboto", FontSize: &big}))
		b.AddText("b")
		cs := b.Build().Layouter().(Clusterer).Clusters()
		if len(cs) != 2 || cs[0].Size != 20 || cs[1].Size != 40 {
			t.Errorf("clusters = %+v", cs)
		}
	})
}
