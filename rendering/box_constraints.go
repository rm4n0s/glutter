package rendering

import (
	"math"
	"strconv"
	"strings"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

// Constraints is an immutable layout constraint that a parent gives to a child.
// Implementations must be comparable with ==: the pipeline compares the new
// constraints with the old ones to skip layout.
//
// Flutter: rendering/object.dart Constraints
type Constraints interface {
	// IsTight reports whether there is exactly one size that satisfies the
	// constraints.
	//
	// Flutter: rendering/object.dart Constraints.isTight
	IsTight() bool
	// IsNormalized reports whether the constraints are in canonical form:
	// every minimum is at most its maximum, and none is negative.
	//
	// Flutter: rendering/object.dart Constraints.isNormalized
	IsNormalized() bool
	// DebugAssertIsValid panics in a debug build with a tag that says what is
	// wrong if the constraints are invalid. It returns IsNormalized.
	//
	// Flutter: rendering/object.dart Constraints.debugAssertIsValid
	DebugAssertIsValid(opts ...DebugAssertIsValidOpts) bool
}

// DebugAssertIsValidOpts are the optional parameters of
// Constraints.DebugAssertIsValid.
//
// Flutter: rendering/object.dart Constraints.debugAssertIsValid
type DebugAssertIsValidOpts struct {
	// IsAppliedConstraint is true if the constraints are applied to a render
	// object: constraints that force an infinite size are then invalid.
	IsAppliedConstraint bool
	// InformationCollector adds lines to the panic's metadata.
	InformationCollector InformationCollector
}

// InformationCollector returns lines that add context to a failure.
//
// Deviation: Flutter's collector returns DiagnosticsNodes; this returns
// strings (SPEC §12.10).
//
// Flutter: foundation/assertions.dart InformationCollector
type InformationCollector func() []string

// BoxConstraints are immutable layout constraints for a RenderBox: a minimum
// and maximum width and height. A render box picks a size inside them.
//
// The zero value is the tight constraints of size zero; use NewBoxConstraints
// for Flutter's defaults (unconstrained).
//
// Flutter: rendering/box.dart BoxConstraints
type BoxConstraints struct {
	// MinWidth is the minimum width that satisfies the constraints.
	MinWidth float64
	// MaxWidth is the maximum width that satisfies the constraints.
	MaxWidth float64
	// MinHeight is the minimum height that satisfies the constraints.
	MinHeight float64
	// MaxHeight is the maximum height that satisfies the constraints.
	MaxHeight float64
}

// BoxConstraintsOpts are the optional parameters of NewBoxConstraints.
//
// Flutter: rendering/box.dart BoxConstraints.new
type BoxConstraintsOpts struct {
	// MinWidth defaults to 0.
	MinWidth foundation.Opt[float64]
	// MaxWidth defaults to infinity.
	MaxWidth foundation.Opt[float64]
	// MinHeight defaults to 0.
	MinHeight foundation.Opt[float64]
	// MaxHeight defaults to infinity.
	MaxHeight foundation.Opt[float64]
}

// NewBoxConstraints creates box constraints; an unset value is 0 for a minimum
// and infinity for a maximum.
//
// Flutter: rendering/box.dart BoxConstraints.new
func NewBoxConstraints(opts ...BoxConstraintsOpts) BoxConstraints {
	o := firstOpts(opts)
	return BoxConstraints{
		MinWidth:  o.MinWidth.OrElse(0),
		MaxWidth:  o.MaxWidth.OrElse(math.Inf(1)),
		MinHeight: o.MinHeight.OrElse(0),
		MaxHeight: o.MaxHeight.OrElse(math.Inf(1)),
	}
}

// NewBoxConstraintsTight creates constraints that are tight on the size.
//
// Flutter: rendering/box.dart BoxConstraints.tight
func NewBoxConstraintsTight(size ui.Size) BoxConstraints {
	return BoxConstraints{size.Width, size.Width, size.Height, size.Height}
}

// TightForOpts are the optional parameters of NewBoxConstraintsTightFor.
//
// Flutter: rendering/box.dart BoxConstraints.tightFor
type TightForOpts struct {
	// Width is the exact width; unset leaves the width unconstrained.
	Width foundation.Opt[float64]
	// Height is the exact height; unset leaves the height unconstrained.
	Height foundation.Opt[float64]
}

// NewBoxConstraintsTightFor creates constraints with the given width and
// height as the only value when set, and unconstrained otherwise.
//
// Flutter: rendering/box.dart BoxConstraints.tightFor
func NewBoxConstraintsTightFor(opts ...TightForOpts) BoxConstraints {
	o := firstOpts(opts)
	return BoxConstraints{
		MinWidth:  o.Width.OrElse(0),
		MaxWidth:  o.Width.OrElse(math.Inf(1)),
		MinHeight: o.Height.OrElse(0),
		MaxHeight: o.Height.OrElse(math.Inf(1)),
	}
}

// TightForFiniteOpts are the optional parameters of
// NewBoxConstraintsTightForFinite.
//
// Flutter: rendering/box.dart BoxConstraints.tightForFinite
type TightForFiniteOpts struct {
	// Width is tight if it is set and finite.
	Width foundation.Opt[float64]
	// Height is tight if it is set and finite.
	Height foundation.Opt[float64]
}

// NewBoxConstraintsTightForFinite creates constraints that are tight on each
// dimension that is set and finite, and unconstrained on the others.
//
// Flutter: rendering/box.dart BoxConstraints.tightForFinite
func NewBoxConstraintsTightForFinite(opts ...TightForFiniteOpts) BoxConstraints {
	o := firstOpts(opts)
	w, h := o.Width.OrElse(math.Inf(1)), o.Height.OrElse(math.Inf(1))
	c := BoxConstraints{0, math.Inf(1), 0, math.Inf(1)}
	if w != math.Inf(1) {
		c.MinWidth, c.MaxWidth = w, w
	}
	if h != math.Inf(1) {
		c.MinHeight, c.MaxHeight = h, h
	}
	return c
}

// NewBoxConstraintsLoose creates constraints that forbid sizes larger than
// the given size.
//
// Flutter: rendering/box.dart BoxConstraints.loose
func NewBoxConstraintsLoose(size ui.Size) BoxConstraints {
	return BoxConstraints{0, size.Width, 0, size.Height}
}

// ExpandOpts are the optional parameters of NewBoxConstraintsExpand.
//
// Flutter: rendering/box.dart BoxConstraints.expand
type ExpandOpts struct {
	// Width is the exact width; unset means as large as possible.
	Width foundation.Opt[float64]
	// Height is the exact height; unset means as large as possible.
	Height foundation.Opt[float64]
}

// NewBoxConstraintsExpand creates constraints that expand to fill another box
// constraints: each dimension is the given value, or infinity.
//
// Flutter: rendering/box.dart BoxConstraints.expand
func NewBoxConstraintsExpand(opts ...ExpandOpts) BoxConstraints {
	o := firstOpts(opts)
	w, h := o.Width.OrElse(math.Inf(1)), o.Height.OrElse(math.Inf(1))
	return BoxConstraints{w, w, h, h}
}

// NewBoxConstraintsFromViewConstraints creates box constraints from the
// constraints of a view.
//
// Flutter: rendering/box.dart BoxConstraints.fromViewConstraints
func NewBoxConstraintsFromViewConstraints(c ui.ViewConstraints) BoxConstraints {
	return BoxConstraints{c.MinWidth, c.MaxWidth, c.MinHeight, c.MaxHeight}
}

// BoxConstraintsCopy are the values CopyWith replaces.
//
// Flutter: rendering/box.dart BoxConstraints.copyWith
type BoxConstraintsCopy struct {
	// MinWidth replaces the minimum width if set.
	MinWidth foundation.Opt[float64]
	// MaxWidth replaces the maximum width if set.
	MaxWidth foundation.Opt[float64]
	// MinHeight replaces the minimum height if set.
	MinHeight foundation.Opt[float64]
	// MaxHeight replaces the maximum height if set.
	MaxHeight foundation.Opt[float64]
}

// CopyWith returns the constraints with the set values replaced.
//
// Flutter: rendering/box.dart BoxConstraints.copyWith
func (c BoxConstraints) CopyWith(o BoxConstraintsCopy) BoxConstraints {
	return BoxConstraints{
		MinWidth:  o.MinWidth.OrElse(c.MinWidth),
		MaxWidth:  o.MaxWidth.OrElse(c.MaxWidth),
		MinHeight: o.MinHeight.OrElse(c.MinHeight),
		MaxHeight: o.MaxHeight.OrElse(c.MaxHeight),
	}
}

// Deflate returns constraints with the insets removed from the maximums, and
// from the minimums down to zero.
//
// Flutter: rendering/box.dart BoxConstraints.deflate
func (c BoxConstraints) Deflate(edges painting.EdgeInsetsGeometry) BoxConstraints {
	c.DebugAssertIsValid()
	horizontal, vertical := edges.Horizontal(), edges.Vertical()
	minW := math.Max(0, c.MinWidth-horizontal)
	minH := math.Max(0, c.MinHeight-vertical)
	return BoxConstraints{
		MinWidth:  minW,
		MaxWidth:  math.Max(minW, c.MaxWidth-horizontal),
		MinHeight: minH,
		MaxHeight: math.Max(minH, c.MaxHeight-vertical),
	}
}

// Loosen returns constraints with the minimums removed and the same maximums.
//
// Flutter: rendering/box.dart BoxConstraints.loosen
func (c BoxConstraints) Loosen() BoxConstraints {
	c.DebugAssertIsValid()
	return BoxConstraints{0, c.MaxWidth, 0, c.MaxHeight}
}

// Enforce returns constraints that respect the given constraints as much as
// these allow.
//
// Flutter: rendering/box.dart BoxConstraints.enforce
func (c BoxConstraints) Enforce(o BoxConstraints) BoxConstraints {
	return BoxConstraints{
		MinWidth:  ui.ClampDouble(c.MinWidth, o.MinWidth, o.MaxWidth),
		MaxWidth:  ui.ClampDouble(c.MaxWidth, o.MinWidth, o.MaxWidth),
		MinHeight: ui.ClampDouble(c.MinHeight, o.MinHeight, o.MaxHeight),
		MaxHeight: ui.ClampDouble(c.MaxHeight, o.MinHeight, o.MaxHeight),
	}
}

// TightenOpts are the optional parameters of BoxConstraints.Tighten.
//
// Flutter: rendering/box.dart BoxConstraints.tighten
type TightenOpts struct {
	// Width is the width to tighten to, clamped to the constraints; unset keeps
	// the width constraints.
	Width foundation.Opt[float64]
	// Height is the height to tighten to, clamped to the constraints; unset
	// keeps the height constraints.
	Height foundation.Opt[float64]
}

// Tighten returns constraints that are tight on the given width and height,
// clamped to these constraints.
//
// Flutter: rendering/box.dart BoxConstraints.tighten
func (c BoxConstraints) Tighten(opts ...TightenOpts) BoxConstraints {
	o := firstOpts(opts)
	if w, ok := o.Width.Get(); ok {
		w = ui.ClampDouble(w, c.MinWidth, c.MaxWidth)
		c.MinWidth, c.MaxWidth = w, w
	}
	if h, ok := o.Height.Get(); ok {
		h = ui.ClampDouble(h, c.MinHeight, c.MaxHeight)
		c.MinHeight, c.MaxHeight = h, h
	}
	return c
}

// Flipped returns constraints with the width and height swapped.
//
// Flutter: rendering/box.dart BoxConstraints.flipped
func (c BoxConstraints) Flipped() BoxConstraints {
	return BoxConstraints{c.MinHeight, c.MaxHeight, c.MinWidth, c.MaxWidth}
}

// WidthConstraints returns constraints with only the width constraints kept.
//
// Flutter: rendering/box.dart BoxConstraints.widthConstraints
func (c BoxConstraints) WidthConstraints() BoxConstraints {
	return BoxConstraints{c.MinWidth, c.MaxWidth, 0, math.Inf(1)}
}

// HeightConstraints returns constraints with only the height constraints kept.
//
// Flutter: rendering/box.dart BoxConstraints.heightConstraints
func (c BoxConstraints) HeightConstraints() BoxConstraints {
	return BoxConstraints{0, math.Inf(1), c.MinHeight, c.MaxHeight}
}

// ConstrainWidth returns the width closest to the given one that satisfies the
// constraints. Pass math.Inf(1) for the largest.
//
// Flutter: rendering/box.dart BoxConstraints.constrainWidth
func (c BoxConstraints) ConstrainWidth(width float64) float64 {
	c.DebugAssertIsValid()
	return ui.ClampDouble(width, c.MinWidth, c.MaxWidth)
}

// ConstrainHeight returns the height closest to the given one that satisfies
// the constraints. Pass math.Inf(1) for the largest.
//
// Flutter: rendering/box.dart BoxConstraints.constrainHeight
func (c BoxConstraints) ConstrainHeight(height float64) float64 {
	c.DebugAssertIsValid()
	return ui.ClampDouble(height, c.MinHeight, c.MaxHeight)
}

// Constrain returns the size closest to the given one that satisfies the
// constraints.
//
// Flutter: rendering/box.dart BoxConstraints.constrain
func (c BoxConstraints) Constrain(size ui.Size) ui.Size {
	return ui.Size{Width: c.ConstrainWidth(size.Width), Height: c.ConstrainHeight(size.Height)}
}

// ConstrainDimensions returns the size closest to the given width and height
// that satisfies the constraints.
//
// Flutter: rendering/box.dart BoxConstraints.constrainDimensions
func (c BoxConstraints) ConstrainDimensions(width, height float64) ui.Size {
	return ui.Size{Width: c.ConstrainWidth(width), Height: c.ConstrainHeight(height)}
}

// ConstrainSizeAndAttemptToPreserveAspectRatio returns the size that satisfies
// the constraints and keeps the aspect ratio of the given size where it can.
//
// Flutter: rendering/box.dart BoxConstraints.constrainSizeAndAttemptToPreserveAspectRatio
func (c BoxConstraints) ConstrainSizeAndAttemptToPreserveAspectRatio(size ui.Size) ui.Size {
	if c.IsTight() {
		return c.Smallest()
	}
	if size.IsEmpty() {
		return c.Constrain(size)
	}
	width, height := size.Width, size.Height
	aspectRatio := width / height
	if width > c.MaxWidth {
		width = c.MaxWidth
		height = width / aspectRatio
	}
	if height > c.MaxHeight {
		height = c.MaxHeight
		width = height * aspectRatio
	}
	if width < c.MinWidth {
		width = c.MinWidth
		height = width / aspectRatio
	}
	if height < c.MinHeight {
		height = c.MinHeight
		width = height * aspectRatio
	}
	return ui.Size{Width: c.ConstrainWidth(width), Height: c.ConstrainHeight(height)}
}

// Biggest is the biggest size that satisfies the constraints.
//
// Flutter: rendering/box.dart BoxConstraints.biggest
func (c BoxConstraints) Biggest() ui.Size {
	return ui.Size{Width: c.ConstrainWidth(math.Inf(1)), Height: c.ConstrainHeight(math.Inf(1))}
}

// Smallest is the smallest size that satisfies the constraints.
//
// Flutter: rendering/box.dart BoxConstraints.smallest
func (c BoxConstraints) Smallest() ui.Size {
	return ui.Size{Width: c.ConstrainWidth(0), Height: c.ConstrainHeight(0)}
}

// HasTightWidth reports whether there is exactly one width that satisfies the
// constraints.
//
// Flutter: rendering/box.dart BoxConstraints.hasTightWidth
func (c BoxConstraints) HasTightWidth() bool { return c.MinWidth >= c.MaxWidth }

// HasTightHeight reports whether there is exactly one height that satisfies
// the constraints.
//
// Flutter: rendering/box.dart BoxConstraints.hasTightHeight
func (c BoxConstraints) HasTightHeight() bool { return c.MinHeight >= c.MaxHeight }

// IsTight reports whether there is exactly one size that satisfies the
// constraints.
//
// Flutter: rendering/box.dart BoxConstraints.isTight
func (c BoxConstraints) IsTight() bool { return c.HasTightWidth() && c.HasTightHeight() }

// HasBoundedWidth reports whether the maximum width is finite.
//
// Flutter: rendering/box.dart BoxConstraints.hasBoundedWidth
func (c BoxConstraints) HasBoundedWidth() bool { return c.MaxWidth < math.Inf(1) }

// HasBoundedHeight reports whether the maximum height is finite.
//
// Flutter: rendering/box.dart BoxConstraints.hasBoundedHeight
func (c BoxConstraints) HasBoundedHeight() bool { return c.MaxHeight < math.Inf(1) }

// HasInfiniteWidth reports whether the constraints force an infinite width.
//
// Flutter: rendering/box.dart BoxConstraints.hasInfiniteWidth
func (c BoxConstraints) HasInfiniteWidth() bool { return c.MinWidth >= math.Inf(1) }

// HasInfiniteHeight reports whether the constraints force an infinite height.
//
// Flutter: rendering/box.dart BoxConstraints.hasInfiniteHeight
func (c BoxConstraints) HasInfiniteHeight() bool { return c.MinHeight >= math.Inf(1) }

// IsSatisfiedBy reports whether the size satisfies the constraints.
//
// Flutter: rendering/box.dart BoxConstraints.isSatisfiedBy
func (c BoxConstraints) IsSatisfiedBy(size ui.Size) bool {
	c.DebugAssertIsValid()
	return c.MinWidth <= size.Width && size.Width <= c.MaxWidth &&
		c.MinHeight <= size.Height && size.Height <= c.MaxHeight
}

// Mul scales every value by the factor.
//
// Flutter: rendering/box.dart BoxConstraints.operator*
func (c BoxConstraints) Mul(factor float64) BoxConstraints {
	return BoxConstraints{c.MinWidth * factor, c.MaxWidth * factor, c.MinHeight * factor, c.MaxHeight * factor}
}

// Div divides every value by the factor.
//
// Flutter: rendering/box.dart BoxConstraints.operator/
func (c BoxConstraints) Div(factor float64) BoxConstraints {
	return BoxConstraints{c.MinWidth / factor, c.MaxWidth / factor, c.MinHeight / factor, c.MaxHeight / factor}
}

// TruncDiv divides every value by the factor and truncates the result.
//
// Flutter: rendering/box.dart BoxConstraints.operator~/
func (c BoxConstraints) TruncDiv(factor float64) BoxConstraints {
	t := func(v float64) float64 { return math.Trunc(v / factor) }
	return BoxConstraints{t(c.MinWidth), t(c.MaxWidth), t(c.MinHeight), t(c.MaxHeight)}
}

// Mod takes the remainder of every value divided by value, like Dart's %: it is
// never negative for a positive divisor.
//
// Flutter: rendering/box.dart BoxConstraints.operator%
func (c BoxConstraints) Mod(value float64) BoxConstraints {
	return BoxConstraints{dartMod(c.MinWidth, value), dartMod(c.MaxWidth, value),
		dartMod(c.MinHeight, value), dartMod(c.MaxHeight, value)}
}

// dartMod is Dart's double %: the result is never negative for a positive
// divisor.
func dartMod(a, b float64) float64 {
	r := math.Mod(a, b)
	if r < 0 {
		if b < 0 {
			r -= b
		} else {
			r += b
		}
	}
	return r
}

// BoxConstraintsLerp linearly interpolates between two constraints; nil is
// treated as zero constraints scaled by t.
//
// Flutter: rendering/box.dart BoxConstraints.lerp
func BoxConstraintsLerp(a, b *BoxConstraints, t float64) *BoxConstraints {
	if a == b {
		return a
	}
	if a == nil {
		r := b.Mul(t)
		return &r
	}
	if b == nil {
		r := a.Mul(1 - t)
		return &r
	}
	if foundation.DebugMode {
		a.DebugAssertIsValid()
		b.DebugAssertIsValid()
		pairs := [][2]float64{{a.MinWidth, b.MinWidth}, {a.MaxWidth, b.MaxWidth}, {a.MinHeight, b.MinHeight}, {a.MaxHeight, b.MaxHeight}}
		for _, p := range pairs {
			if !((isFinite(p[0]) && isFinite(p[1])) || (math.IsInf(p[0], 1) && math.IsInf(p[1], 1))) {
				panic(errors.New(tagBoxConstraintsLerpUnbounded,
					"cannot interpolate between finite constraints and unbounded constraints", "a", *a, "b", *b))
			}
		}
	}
	l := func(x, y float64) float64 {
		if isFinite(x) {
			return ui.LerpDouble(x, y, t)
		}
		return math.Inf(1)
	}
	return &BoxConstraints{l(a.MinWidth, b.MinWidth), l(a.MaxWidth, b.MaxWidth),
		l(a.MinHeight, b.MinHeight), l(a.MaxHeight, b.MaxHeight)}
}

func isFinite(v float64) bool { return !math.IsInf(v, 0) && !math.IsNaN(v) }

// IsNormalized reports whether the minimums are non-negative and at most the
// maximums.
//
// Flutter: rendering/box.dart BoxConstraints.isNormalized
func (c BoxConstraints) IsNormalized() bool {
	return c.MinWidth >= 0 && c.MinWidth <= c.MaxWidth && c.MinHeight >= 0 && c.MinHeight <= c.MaxHeight
}

// DebugAssertIsValid panics in a debug build if the constraints are NaN,
// negative or not normalized (and, for applied constraints, infinite). The tags
// are BoxConstraintsNaN, BoxConstraintsNegative, BoxConstraintsNotNormalized
// and BoxConstraintsInfinite. It returns IsNormalized.
//
// Flutter: rendering/box.dart BoxConstraints.debugAssertIsValid
func (c BoxConstraints) DebugAssertIsValid(opts ...DebugAssertIsValidOpts) bool {
	if foundation.DebugMode {
		o := firstOpts(opts)
		fail := func(tag, msg string) {
			meta := []any{"constraints", c}
			if o.InformationCollector != nil {
				meta = append(meta, "information", strings.Join(o.InformationCollector(), "\n"))
			}
			panic(errors.New(tag, msg, meta...))
		}
		if math.IsNaN(c.MinWidth) || math.IsNaN(c.MaxWidth) || math.IsNaN(c.MinHeight) || math.IsNaN(c.MaxHeight) {
			var fields []string
			for _, f := range []struct {
				n string
				v float64
			}{{"minWidth", c.MinWidth}, {"maxWidth", c.MaxWidth}, {"minHeight", c.MinHeight}, {"maxHeight", c.MaxHeight}} {
				if math.IsNaN(f.v) {
					fields = append(fields, f.n)
				}
			}
			var which string
			switch {
			case len(fields) == 1:
				which = fields[0]
			case len(fields) == 2:
				which = fields[0] + " and " + fields[1]
			default:
				which = strings.Join(fields[:len(fields)-1], ", ") + ", and " + fields[len(fields)-1]
			}
			noun := "a NaN value"
			if len(fields) > 1 {
				noun = "NaN values"
			}
			fail("BoxConstraintsNaN", "BoxConstraints has "+noun+" in "+which+".")
		}
		switch {
		case c.MinWidth < 0 && c.MinHeight < 0:
			fail("BoxConstraintsNegative", "BoxConstraints has both a negative minimum width and a negative minimum height.")
		case c.MinWidth < 0:
			fail("BoxConstraintsNegative", "BoxConstraints has a negative minimum width.")
		case c.MinHeight < 0:
			fail("BoxConstraintsNegative", "BoxConstraints has a negative minimum height.")
		}
		switch {
		case c.MaxWidth < c.MinWidth && c.MaxHeight < c.MinHeight:
			fail("BoxConstraintsNotNormalized", "BoxConstraints has both width and height constraints non-normalized.")
		case c.MaxWidth < c.MinWidth:
			fail("BoxConstraintsNotNormalized", "BoxConstraints has non-normalized width constraints.")
		case c.MaxHeight < c.MinHeight:
			fail("BoxConstraintsNotNormalized", "BoxConstraints has non-normalized height constraints.")
		}
		if o.IsAppliedConstraint {
			wi, hi := math.IsInf(c.MinWidth, 0), math.IsInf(c.MinHeight, 0)
			switch {
			case wi && hi:
				fail("BoxConstraintsInfinite", "BoxConstraints forces an infinite width and infinite height.")
			case wi:
				fail("BoxConstraintsInfinite", "BoxConstraints forces an infinite width.")
			case hi:
				fail("BoxConstraintsInfinite", "BoxConstraints forces an infinite height.")
			}
		}
	}
	return c.IsNormalized()
}

// Normalize returns constraints with the same values made normalized: no
// negative minimum, and no maximum below its minimum.
//
// Flutter: rendering/box.dart BoxConstraints.normalize
func (c BoxConstraints) Normalize() BoxConstraints {
	if c.IsNormalized() {
		return c
	}
	minW := c.MinWidth
	if !(minW >= 0) {
		minW = 0
	}
	minH := c.MinHeight
	if !(minH >= 0) {
		minH = 0
	}
	maxW, maxH := c.MaxWidth, c.MaxHeight
	if minW > maxW {
		maxW = minW
	}
	if minH > maxH {
		maxH = minH
	}
	return BoxConstraints{minW, maxW, minH, maxH}
}

// String describes the constraints, such as "BoxConstraints(0.0<=w<=100.0, h=50.0)".
//
// Flutter: rendering/box.dart BoxConstraints.toString
func (c BoxConstraints) String() string {
	annotation := ""
	if !c.IsNormalized() {
		annotation = "; NOT NORMALIZED"
	}
	inf := math.Inf(1)
	if c.MinWidth == inf && c.MinHeight == inf {
		return "BoxConstraints(biggest" + annotation + ")"
	}
	if c.MinWidth == 0 && c.MaxWidth == inf && c.MinHeight == 0 && c.MaxHeight == inf {
		return "BoxConstraints(unconstrained" + annotation + ")"
	}
	describe := func(min, max float64, dim string) string {
		if min == max {
			return dim + "=" + fixed1(min)
		}
		return fixed1(min) + "<=" + dim + "<=" + fixed1(max)
	}
	return "BoxConstraints(" + describe(c.MinWidth, c.MaxWidth, "w") + ", " + describe(c.MinHeight, c.MaxHeight, "h") + annotation + ")"
}

// fixed1 is Dart's toStringAsFixed(1).
func fixed1(v float64) string {
	switch {
	case math.IsInf(v, 1):
		return "Infinity"
	case math.IsInf(v, -1):
		return "-Infinity"
	case math.IsNaN(v):
		return "NaN"
	}
	return strconv.FormatFloat(v, 'f', 1, 64)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagBoxConstraintsLerpUnbounded = "BoxConstraintsLerpUnbounded"
)
