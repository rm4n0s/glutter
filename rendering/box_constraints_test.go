package rendering_test

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

var inf = math.Inf(1)

func TestBoxConstraintsToString(t *testing.T) {
	tests := []struct {
		name string
		c    rendering.BoxConstraints
		want string
	}{
		{"expand is biggest", rendering.NewBoxConstraintsExpand(), "BoxConstraints(biggest)"},
		{"default is unconstrained", rendering.NewBoxConstraints(), "BoxConstraints(unconstrained)"},
		{"tight width", rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{Width: foundation.Some(50.0)}), "BoxConstraints(w=50.0, 0.0<=h<=Infinity)"},
		{"range", rendering.BoxConstraints{MinWidth: 1, MaxWidth: 2, MinHeight: 3, MaxHeight: 3}, "BoxConstraints(1.0<=w<=2.0, h=3.0)"},
		{"not normalized", rendering.BoxConstraints{MinWidth: 3, MaxWidth: 2, MinHeight: 0, MaxHeight: 1}, "BoxConstraints(3.0<=w<=2.0, 0.0<=h<=1.0; NOT NORMALIZED)"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.c.String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestBoxConstraintsConstructors(t *testing.T) {
	size := ui.Size{Width: 30, Height: 40}
	tests := []struct {
		name string
		got  rendering.BoxConstraints
		want rendering.BoxConstraints
	}{
		{"tight", rendering.NewBoxConstraintsTight(size), rendering.BoxConstraints{30, 30, 40, 40}},
		{"loose", rendering.NewBoxConstraintsLoose(size), rendering.BoxConstraints{0, 30, 0, 40}},
		{"tightFor height", rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{Height: foundation.Some(7.0)}), rendering.BoxConstraints{0, inf, 7, 7}},
		{"tightForFinite finite", rendering.NewBoxConstraintsTightForFinite(rendering.TightForFiniteOpts{Width: foundation.Some(5.0)}), rendering.BoxConstraints{5, 5, 0, inf}},
		{"tightForFinite infinite", rendering.NewBoxConstraintsTightForFinite(rendering.TightForFiniteOpts{Width: foundation.Some(inf)}), rendering.BoxConstraints{0, inf, 0, inf}},
		{"expand", rendering.NewBoxConstraintsExpand(rendering.ExpandOpts{Width: foundation.Some(9.0)}), rendering.BoxConstraints{9, 9, inf, inf}},
		{"fromViewConstraints", rendering.NewBoxConstraintsFromViewConstraints(ui.ViewConstraints{MinWidth: 1, MaxWidth: 2, MinHeight: 3, MaxHeight: 4}), rendering.BoxConstraints{1, 2, 3, 4}},
		{"unconstrained view", rendering.NewBoxConstraintsFromViewConstraints(ui.ViewConstraints{MaxWidth: inf, MaxHeight: inf}), rendering.NewBoxConstraints()},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("got %v, want %v", tc.got, tc.want)
			}
		})
	}
}

func TestBoxConstraintsCopyWith(t *testing.T) {
	c := rendering.BoxConstraints{MinWidth: 3, MaxWidth: 7, MinHeight: 11, MaxHeight: 17}
	if got := c.CopyWith(rendering.BoxConstraintsCopy{}); got != c {
		t.Errorf("empty copy = %v, want %v", got, c)
	}
	got := c.CopyWith(rendering.BoxConstraintsCopy{
		MinWidth: foundation.Some(13.0), MaxWidth: foundation.Some(17.0),
		MinHeight: foundation.Some(111.0), MaxHeight: foundation.Some(117.0),
	})
	if want := (rendering.BoxConstraints{MinWidth: 13, MaxWidth: 17, MinHeight: 111, MaxHeight: 117}); got != want {
		t.Errorf("copy = %v, want %v", got, want)
	}
	if got == c {
		t.Error("the copy equals the original")
	}
}

func TestBoxConstraintsOperators(t *testing.T) {
	c := rendering.BoxConstraints{MinWidth: 3, MaxWidth: 7, MinHeight: 11, MaxHeight: 17}
	tests := []struct {
		name string
		got  rendering.BoxConstraints
		want rendering.BoxConstraints
	}{
		{"mul", c.Mul(2), rendering.BoxConstraints{6, 14, 22, 34}},
		{"div", c.Mul(2).Div(2), c},
		{"truncDiv", c.TruncDiv(2), rendering.BoxConstraints{1, 3, 5, 8}},
		{"mod", c.Mod(3), rendering.BoxConstraints{0, 1, 2, 2}},
		{"mod negative", rendering.BoxConstraints{MinWidth: -1, MaxWidth: 1}.Mod(3), rendering.BoxConstraints{MinWidth: 2, MaxWidth: 1}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("got %v, want %v", tc.got, tc.want)
			}
		})
	}
}

func near(a, b float64) bool { return a == b || math.Abs(a-b) < 1e-9 }

func TestBoxConstraintsLerp(t *testing.T) {
	c := &rendering.BoxConstraints{MinWidth: 3, MaxWidth: 7, MinHeight: 11, MaxHeight: 17}
	tests := []struct {
		name string
		a, b *rendering.BoxConstraints
		t    float64
		want rendering.BoxConstraints
	}{
		{"from nil", nil, c, 0.5, rendering.BoxConstraints{1.5, 3.5, 5.5, 8.5}},
		{"to nil", c, nil, 0.5, rendering.BoxConstraints{1.5, 3.5, 5.5, 8.5}},
		{"between", &rendering.BoxConstraints{MinWidth: 13, MaxWidth: 17, MinHeight: 111, MaxHeight: 117}, c, 0.2, rendering.BoxConstraints{11, 15, 91, 97}},
		{"unbounded width", &rendering.BoxConstraints{MinWidth: inf, MaxWidth: inf, MinHeight: 10, MaxHeight: 20},
			&rendering.BoxConstraints{MinWidth: inf, MaxWidth: inf, MinHeight: 20, MaxHeight: 30}, 0.5,
			rendering.BoxConstraints{MinWidth: inf, MaxWidth: inf, MinHeight: 15, MaxHeight: 25}},
		{"unbounded height", &rendering.BoxConstraints{MinWidth: 10, MaxWidth: 20, MinHeight: inf, MaxHeight: inf},
			&rendering.BoxConstraints{MinWidth: 20, MaxWidth: 30, MinHeight: inf, MaxHeight: inf}, 0.5,
			rendering.BoxConstraints{MinWidth: 15, MaxWidth: 25, MinHeight: inf, MaxHeight: inf}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := rendering.BoxConstraintsLerp(tc.a, tc.b, tc.t)
			if got == nil || !near(got.MinWidth, tc.want.MinWidth) || !near(got.MaxWidth, tc.want.MaxWidth) ||
				!near(got.MinHeight, tc.want.MinHeight) || !near(got.MaxHeight, tc.want.MaxHeight) {
				t.Errorf("lerp = %v, want %v", got, tc.want)
			}
		})
	}
	if got := rendering.BoxConstraintsLerp(nil, nil, 0.5); got != nil {
		t.Errorf("lerp(nil, nil) = %v, want nil", got)
	}
	same := &rendering.BoxConstraints{MaxWidth: inf, MaxHeight: inf}
	if got := rendering.BoxConstraintsLerp(same, same, 0.5); got != same {
		t.Error("lerp of identical constraints is not the same pointer")
	}
}

func TestBoxConstraintsNormalize(t *testing.T) {
	tests := []struct {
		name string
		c    rendering.BoxConstraints
		want rendering.BoxConstraints
	}{
		{"max below min", rendering.BoxConstraints{MinWidth: 3, MaxWidth: 2, MinHeight: 11, MaxHeight: 18}, rendering.BoxConstraints{3, 3, 11, 18}},
		{"negative min", rendering.BoxConstraints{MinWidth: -1, MaxWidth: 5, MinHeight: -2, MaxHeight: -1}, rendering.BoxConstraints{0, 5, 0, 0}},
		{"already normal", rendering.BoxConstraints{1, 2, 3, 4}, rendering.BoxConstraints{1, 2, 3, 4}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.c.Normalize(); got != tc.want || !got.IsNormalized() {
				t.Errorf("Normalize() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestBoxConstraintsAspectRatio(t *testing.T) {
	c := rendering.BoxConstraints{MinWidth: 10, MaxWidth: 20, MinHeight: 10, MaxHeight: 20}
	tests := []struct {
		name string
		c    rendering.BoxConstraints
		in   ui.Size
		want ui.Size
	}{
		{"empty size", c, ui.Size{Width: 15, Height: 0}, ui.Size{Width: 15, Height: 10}},
		{"tight", rendering.NewBoxConstraintsTight(ui.Size{Width: 5, Height: 6}), ui.Size{Width: 50, Height: 10}, ui.Size{Width: 5, Height: 6}},
		{"too wide", c, ui.Size{Width: 40, Height: 20}, ui.Size{Width: 20, Height: 10}},
		{"too tall", c, ui.Size{Width: 20, Height: 40}, ui.Size{Width: 10, Height: 20}},
		{"too small", c, ui.Size{Width: 2, Height: 1}, ui.Size{Width: 20, Height: 10}},
		{"fits", c, ui.Size{Width: 12, Height: 14}, ui.Size{Width: 12, Height: 14}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.c.ConstrainSizeAndAttemptToPreserveAspectRatio(tc.in); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestBoxConstraintsMethods(t *testing.T) {
	c := rendering.BoxConstraints{MinWidth: 10, MaxWidth: 100, MinHeight: 20, MaxHeight: 200}
	tests := []struct {
		name string
		got  any
		want any
	}{
		{"deflate", c.Deflate(painting.NewEdgeInsetsFromLTRB(5, 10, 15, 30)), rendering.BoxConstraints{0, 80, 0, 160}},
		{"deflate keeps ordering", rendering.BoxConstraints{5, 10, 5, 10}.Deflate(painting.NewEdgeInsetsAll(20)), rendering.BoxConstraints{0, 0, 0, 0}},
		{"loosen", c.Loosen(), rendering.BoxConstraints{0, 100, 0, 200}},
		{"enforce", rendering.BoxConstraints{0, 500, 0, 500}.Enforce(c), c},
		{"tighten", c.Tighten(rendering.TightenOpts{Width: foundation.Some(500.0), Height: foundation.Some(1.0)}), rendering.BoxConstraints{100, 100, 20, 20}},
		{"tighten none", c.Tighten(), c},
		{"flipped", c.Flipped(), rendering.BoxConstraints{20, 200, 10, 100}},
		{"widthConstraints", c.WidthConstraints(), rendering.BoxConstraints{10, 100, 0, inf}},
		{"heightConstraints", c.HeightConstraints(), rendering.BoxConstraints{0, inf, 20, 200}},
		{"constrainWidth", c.ConstrainWidth(1000), 100.0},
		{"constrainHeight", c.ConstrainHeight(0), 20.0},
		{"constrain", c.Constrain(ui.Size{Width: 1, Height: 1000}), ui.Size{Width: 10, Height: 200}},
		{"constrainDimensions", c.ConstrainDimensions(50, 50), ui.Size{Width: 50, Height: 50}},
		{"biggest", c.Biggest(), ui.Size{Width: 100, Height: 200}},
		{"smallest", c.Smallest(), ui.Size{Width: 10, Height: 20}},
		{"hasTightWidth", rendering.NewBoxConstraintsTight(ui.Size{Width: 1, Height: 2}).HasTightWidth(), true},
		{"hasTightHeight", c.HasTightHeight(), false},
		{"isTight", rendering.NewBoxConstraintsTight(ui.Size{Width: 1, Height: 2}).IsTight(), true},
		{"hasBoundedWidth", c.HasBoundedWidth(), true},
		{"hasBoundedHeight", rendering.NewBoxConstraints().HasBoundedHeight(), false},
		{"hasInfiniteWidth", rendering.NewBoxConstraintsExpand().HasInfiniteWidth(), true},
		{"hasInfiniteHeight", c.HasInfiniteHeight(), false},
		{"isSatisfiedBy yes", c.IsSatisfiedBy(ui.Size{Width: 10, Height: 200}), true},
		{"isSatisfiedBy no", c.IsSatisfiedBy(ui.Size{Width: 9, Height: 200}), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("got %v, want %v", tc.got, tc.want)
			}
		})
	}
}
