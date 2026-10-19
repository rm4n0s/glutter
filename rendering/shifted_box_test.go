package rendering_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// mockBox is MockRenderBox (a baseline at 40) and MockRenderBoxNoBaseline.
type mockBox struct {
	rendering.RenderBox
	baseline foundation.Opt[float64]
}

func newMockBox(baseline foundation.Opt[float64]) *mockBox {
	b := &mockBox{baseline: baseline}
	rendering.InitRenderBox(&b.RenderBox, b)
	return b
}

func (b *mockBox) ComputeDryLayout(c rendering.BoxConstraints) ui.Size {
	return c.Constrain(ui.Size{Width: 50, Height: 50})
}
func (b *mockBox) ComputeDryBaseline(rendering.BoxConstraints, ui.TextBaseline) foundation.Opt[float64] {
	return b.baseline
}
func (b *mockBox) PerformLayout() { b.SetSize(b.ComputeDryLayout(b.BoxConstraints())) }

func TestShiftedBoxComputeDryBaseline(t *testing.T) {
	c100 := rendering.BoxConstraints{MaxWidth: 100, MaxHeight: 100}
	c200 := rendering.BoxConstraints{MaxWidth: 200, MaxHeight: 200}
	unbounded := rendering.BoxConstraints{MaxWidth: inf, MaxHeight: inf}
	topLeft := rendering.RenderPositionedBoxOpts{Alignment: painting.AlignmentTopLeft}
	baseline40 := foundation.Some(40.0)
	tests := []struct {
		name        string
		box         func() rendering.Box
		constraints rendering.BoxConstraints
		check       func(t *testing.T, got foundation.Opt[float64])
	}{
		{"positioned box without a child", func() rendering.Box { return rendering.NewRenderPositionedBox(nil) }, c100, wantNone},
		{"positioned box, child without baseline", func() rendering.Box {
			return rendering.NewRenderPositionedBox(newMockBox(foundation.None[float64]()))
		}, c100, wantNone},
		{"center alignment", func() rendering.Box { return rendering.NewRenderPositionedBox(newMockBox(baseline40)) }, c200,
			func(t *testing.T, got foundation.Opt[float64]) {
				if got.IsNone() || got.Value() <= 40 {
					t.Errorf("baseline = %v, want more than 40", got)
				}
			}},
		{"top left alignment", func() rendering.Box { return rendering.NewRenderPositionedBox(newMockBox(baseline40), topLeft) }, c200, wantValue(40)},
		{"bottom right alignment", func() rendering.Box {
			return rendering.NewRenderPositionedBox(newMockBox(baseline40), rendering.RenderPositionedBoxOpts{Alignment: painting.AlignmentBottomRight})
		}, c200, func(t *testing.T, got foundation.Opt[float64]) {
			if got.IsNone() || got.Value() <= 100 {
				t.Errorf("baseline = %v, want more than 100", got)
			}
		}},
		{"width and height factors", func() rendering.Box {
			return rendering.NewRenderPositionedBox(newMockBox(baseline40), rendering.RenderPositionedBoxOpts{
				Alignment: painting.AlignmentTopLeft, WidthFactor: some(2), HeightFactor: some(1.5)})
		}, c200, wantValue(40)},
		{"infinite constraints", func() rendering.Box { return rendering.NewRenderPositionedBox(newMockBox(baseline40), topLeft) }, unbounded, wantValue(40)},
		{"very small constraints", func() rendering.Box { return rendering.NewRenderPositionedBox(newMockBox(baseline40)) },
			rendering.BoxConstraints{MaxWidth: 1, MaxHeight: 1}, wantSome},
		{"zero constraints", func() rendering.Box { return rendering.NewRenderPositionedBox(newMockBox(baseline40)) },
			rendering.NewBoxConstraintsTight(ui.Size{}), wantSome},
		{"constrained overflow box", func() rendering.Box {
			return rendering.NewRenderConstrainedOverflowBox(newMockBox(baseline40), rendering.RenderConstrainedOverflowBoxOpts{Alignment: painting.AlignmentTopLeft})
		}, c100, wantSome},
		{"sized overflow box", func() rendering.Box {
			return rendering.NewRenderSizedOverflowBox(ui.Size{Width: 100, Height: 100}, newMockBox(baseline40),
				rendering.RenderSizedOverflowBoxOpts{Alignment: painting.AlignmentTopLeft})
		}, c200, wantSome},
		{"fractionally sized overflow box", func() rendering.Box {
			return rendering.NewRenderFractionallySizedOverflowBox(newMockBox(baseline40), rendering.RenderFractionallySizedOverflowBoxOpts{
				Alignment: painting.AlignmentTopLeft, WidthFactor: some(0.5), HeightFactor: some(0.5)})
		}, c200, wantSome},
		{"padding", func() rendering.Box {
			return rendering.NewRenderPadding(painting.NewEdgeInsetsAll(10), newMockBox(baseline40))
		}, c100, wantSome},
		{"baseline", func() rendering.Box {
			return rendering.NewRenderBaseline(30, ui.TextBaselineAlphabetic, newMockBox(baseline40))
		}, c100, wantSome},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			box := tc.box()
			tc.check(t, box.ComputeDryBaseline(tc.constraints, ui.TextBaselineAlphabetic))
		})
	}

	t.Run("alphabetic and ideographic agree for a mock", func(t *testing.T) {
		box := rendering.NewRenderPositionedBox(newMockBox(baseline40), topLeft)
		a := box.ComputeDryBaseline(c100, ui.TextBaselineAlphabetic)
		i := box.ComputeDryBaseline(c100, ui.TextBaselineIdeographic)
		if a.IsNone() || a != i {
			t.Errorf("alphabetic %v, ideographic %v", a, i)
		}
	})
}

func wantNone(t *testing.T, got foundation.Opt[float64]) {
	t.Helper()
	if got.IsSome() {
		t.Errorf("baseline = %v, want none", got)
	}
}

func wantSome(t *testing.T, got foundation.Opt[float64]) {
	t.Helper()
	if got.IsNone() {
		t.Error("baseline is none")
	}
}

func wantValue(v float64) func(*testing.T, foundation.Opt[float64]) {
	return func(t *testing.T, got foundation.Opt[float64]) {
		t.Helper()
		if got != foundation.Some(v) {
			t.Errorf("baseline = %v, want %v", got, v)
		}
	}
}
