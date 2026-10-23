package rendering

import (
	"fmt"
	"math"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

// SliverLayoutDimensions are the dimensions that a sliver is laid out with, as
// ItemExtentBuilder sees them.
//
// Flutter: rendering/sliver.dart SliverLayoutDimensions
type SliverLayoutDimensions struct {
	// ScrollOffset is the scroll offset of the sliver, as in SliverConstraints.
	//
	// Flutter: rendering/sliver.dart SliverLayoutDimensions.scrollOffset
	ScrollOffset float64
	// PrecedingScrollExtent is the scroll extent of the slivers before this one.
	//
	// Flutter: rendering/sliver.dart SliverLayoutDimensions.precedingScrollExtent
	PrecedingScrollExtent float64
	// ViewportMainAxisExtent is the size of the viewport in the main axis.
	//
	// Flutter: rendering/sliver.dart SliverLayoutDimensions.viewportMainAxisExtent
	ViewportMainAxisExtent float64
	// CrossAxisExtent is the size of the sliver in the cross axis.
	//
	// Flutter: rendering/sliver.dart SliverLayoutDimensions.crossAxisExtent
	CrossAxisExtent float64
}

// String describes the dimensions.
//
// Flutter: rendering/sliver.dart SliverLayoutDimensions.toString
func (d SliverLayoutDimensions) String() string {
	return fmt.Sprintf("scrollOffset: %v precedingScrollExtent: %v viewportMainAxisExtent: %v crossAxisExtent: %v",
		dartDouble(d.ScrollOffset), dartDouble(d.PrecedingScrollExtent), dartDouble(d.ViewportMainAxisExtent), dartDouble(d.CrossAxisExtent))
}

// ItemExtentBuilder gives the main axis extent of the item at the index, or none
// to let the item decide.
//
// Flutter: rendering/sliver.dart ItemExtentBuilder
type ItemExtentBuilder func(index int, dimensions SliverLayoutDimensions) foundation.Opt[float64]

// GrowthDirection is the direction in which a sliver's contents are ordered,
// relative to the axis direction.
//
// Flutter: rendering/sliver.dart GrowthDirection
type GrowthDirection int

// The values of GrowthDirection follow Flutter's declaration order.
const (
	// GrowthDirectionUnspecified is the zero value; it is not a real direction.
	//
	// Flutter: no counterpart
	GrowthDirectionUnspecified GrowthDirection = iota
	// GrowthDirectionForward means the contents are ordered in the axis direction:
	// the first item at the start (the top or left).
	//
	// Flutter: rendering/sliver.dart GrowthDirection.forward
	GrowthDirectionForward
	// GrowthDirectionReverse means the contents are ordered against the axis
	// direction.
	//
	// Flutter: rendering/sliver.dart GrowthDirection.reverse
	GrowthDirectionReverse
)

// String is the name with the enum, such as "GrowthDirection.forward".
//
// Flutter: rendering/sliver.dart GrowthDirection
func (g GrowthDirection) String() string {
	switch g {
	case GrowthDirectionForward:
		return "GrowthDirection.forward"
	case GrowthDirectionReverse:
		return "GrowthDirection.reverse"
	}
	return "GrowthDirection.unspecified"
}

// ApplyGrowthDirectionToAxisDirection flips the axis direction for a reverse
// growth direction.
//
// Flutter: rendering/sliver.dart applyGrowthDirectionToAxisDirection
func ApplyGrowthDirectionToAxisDirection(axisDirection painting.AxisDirection, growthDirection GrowthDirection) painting.AxisDirection {
	if growthDirection == GrowthDirectionReverse {
		return painting.FlipAxisDirection(axisDirection)
	}
	return axisDirection
}

// ApplyGrowthDirectionToScrollDirection flips the scroll direction for a reverse
// growth direction.
//
// Flutter: rendering/sliver.dart applyGrowthDirectionToScrollDirection
func ApplyGrowthDirectionToScrollDirection(scrollDirection ScrollDirection, growthDirection GrowthDirection) ScrollDirection {
	if growthDirection == GrowthDirectionReverse {
		return FlipScrollDirection(scrollDirection)
	}
	return scrollDirection
}

// SliverConstraints are the immutable layout constraints for a RenderSliver:
// where the sliver is in the viewport and how much room it has.
//
// Flutter: rendering/sliver.dart SliverConstraints
type SliverConstraints struct {
	// AxisDirection is the direction in which the scroll offset increases.
	//
	// Flutter: rendering/sliver.dart SliverConstraints.axisDirection
	AxisDirection painting.AxisDirection
	// GrowthDirection is the direction in which the contents of slivers are
	// ordered, relative to the axis direction.
	//
	// Flutter: rendering/sliver.dart SliverConstraints.growthDirection
	GrowthDirection GrowthDirection
	// UserScrollDirection is the direction in which the user is scrolling.
	//
	// Flutter: rendering/sliver.dart SliverConstraints.userScrollDirection
	UserScrollDirection ScrollDirection
	// ScrollOffset is the number of pixels from where the sliver's scroll extent
	// starts to the leading edge of the viewport's visible area.
	//
	// Flutter: rendering/sliver.dart SliverConstraints.scrollOffset
	ScrollOffset float64
	// PrecedingScrollExtent is the scroll extent of everything before this
	// sliver.
	//
	// Flutter: rendering/sliver.dart SliverConstraints.precedingScrollExtent
	PrecedingScrollExtent float64
	// Overlap is the number of pixels of this sliver's leading edge that are
	// covered by a previous sliver that is pinned or floating.
	//
	// Flutter: rendering/sliver.dart SliverConstraints.overlap
	Overlap float64
	// RemainingPaintExtent is the number of pixels of the viewport that remain to
	// be painted, from the leading edge of this sliver.
	//
	// Flutter: rendering/sliver.dart SliverConstraints.remainingPaintExtent
	RemainingPaintExtent float64
	// CrossAxisExtent is the size of the viewport in the cross axis.
	//
	// Flutter: rendering/sliver.dart SliverConstraints.crossAxisExtent
	CrossAxisExtent float64
	// CrossAxisDirection is the direction in which children are laid out in the
	// cross axis.
	//
	// Flutter: rendering/sliver.dart SliverConstraints.crossAxisDirection
	CrossAxisDirection painting.AxisDirection
	// ViewportMainAxisExtent is the size of the viewport in the main axis.
	//
	// Flutter: rendering/sliver.dart SliverConstraints.viewportMainAxisExtent
	ViewportMainAxisExtent float64
	// RemainingCacheExtent is the number of pixels of the cache area that remain,
	// from the leading edge of this sliver (see CacheOrigin).
	//
	// Flutter: rendering/sliver.dart SliverConstraints.remainingCacheExtent
	RemainingCacheExtent float64
	// CacheOrigin is how many pixels before the leading edge of the viewport the
	// cache area starts; it is zero or negative.
	//
	// Flutter: rendering/sliver.dart SliverConstraints.cacheOrigin
	CacheOrigin float64
}

// SliverConstraintsCopy are the values CopyWith replaces.
//
// Flutter: rendering/sliver.dart SliverConstraints.copyWith
type SliverConstraintsCopy struct {
	AxisDirection          painting.AxisDirection
	GrowthDirection        GrowthDirection
	UserScrollDirection    ScrollDirection
	ScrollOffset           foundation.Opt[float64]
	PrecedingScrollExtent  foundation.Opt[float64]
	Overlap                foundation.Opt[float64]
	RemainingPaintExtent   foundation.Opt[float64]
	CrossAxisExtent        foundation.Opt[float64]
	CrossAxisDirection     painting.AxisDirection
	ViewportMainAxisExtent foundation.Opt[float64]
	RemainingCacheExtent   foundation.Opt[float64]
	CacheOrigin            foundation.Opt[float64]
}

// CopyWith returns the constraints with the values that are set replaced (a
// direction that is unspecified is kept).
//
// Flutter: rendering/sliver.dart SliverConstraints.copyWith
func (c SliverConstraints) CopyWith(o SliverConstraintsCopy) SliverConstraints {
	if o.AxisDirection != painting.AxisDirectionUnspecified {
		c.AxisDirection = o.AxisDirection
	}
	if o.GrowthDirection != GrowthDirectionUnspecified {
		c.GrowthDirection = o.GrowthDirection
	}
	if o.UserScrollDirection != ScrollDirectionUnspecified {
		c.UserScrollDirection = o.UserScrollDirection
	}
	c.ScrollOffset = o.ScrollOffset.OrElse(c.ScrollOffset)
	c.PrecedingScrollExtent = o.PrecedingScrollExtent.OrElse(c.PrecedingScrollExtent)
	c.Overlap = o.Overlap.OrElse(c.Overlap)
	c.RemainingPaintExtent = o.RemainingPaintExtent.OrElse(c.RemainingPaintExtent)
	c.CrossAxisExtent = o.CrossAxisExtent.OrElse(c.CrossAxisExtent)
	if o.CrossAxisDirection != painting.AxisDirectionUnspecified {
		c.CrossAxisDirection = o.CrossAxisDirection
	}
	c.ViewportMainAxisExtent = o.ViewportMainAxisExtent.OrElse(c.ViewportMainAxisExtent)
	c.RemainingCacheExtent = o.RemainingCacheExtent.OrElse(c.RemainingCacheExtent)
	c.CacheOrigin = o.CacheOrigin.OrElse(c.CacheOrigin)
	return c
}

// Axis is the axis of the main direction.
//
// Flutter: rendering/sliver.dart SliverConstraints.axis
func (c SliverConstraints) Axis() painting.Axis { return painting.AxisDirectionToAxis(c.AxisDirection) }

// NormalizedGrowthDirection is the growth direction with the reversal of the
// axis direction applied: forward when the sliver's children are laid out toward
// the larger coordinates.
//
// Flutter: rendering/sliver.dart SliverConstraints.normalizedGrowthDirection
func (c SliverConstraints) NormalizedGrowthDirection() GrowthDirection {
	if painting.AxisDirectionIsReversed(c.AxisDirection) {
		if c.GrowthDirection == GrowthDirectionForward {
			return GrowthDirectionReverse
		}
		return GrowthDirectionForward
	}
	return c.GrowthDirection
}

// IsTight is always false: sliver constraints never force one geometry.
//
// Flutter: rendering/sliver.dart SliverConstraints.isTight
func (c SliverConstraints) IsTight() bool { return false }

// IsNormalized is whether the offsets and extents are not negative and the two
// directions are along different axes.
//
// Flutter: rendering/sliver.dart SliverConstraints.isNormalized
func (c SliverConstraints) IsNormalized() bool {
	return c.ScrollOffset >= 0 && c.CrossAxisExtent >= 0 &&
		painting.AxisDirectionToAxis(c.AxisDirection) != painting.AxisDirectionToAxis(c.CrossAxisDirection) &&
		c.ViewportMainAxisExtent >= 0 && c.RemainingPaintExtent >= 0
}

// AsBoxConstraintsOpts are the optional parameters of AsBoxConstraints.
//
// Flutter: rendering/sliver.dart SliverConstraints.asBoxConstraints
type AsBoxConstraintsOpts struct {
	// MinExtent is the minimum extent in the main axis; the default is 0.
	MinExtent float64
	// MaxExtent is the maximum extent in the main axis; the default is infinite.
	MaxExtent foundation.Opt[float64]
	// CrossAxisExtent is the extent in the cross axis; the default is the cross
	// axis extent of the constraints.
	CrossAxisExtent foundation.Opt[float64]
}

// AsBoxConstraints makes box constraints for a child: tight in the cross axis
// and with the given range in the main axis.
//
// Flutter: rendering/sliver.dart SliverConstraints.asBoxConstraints
func (c SliverConstraints) AsBoxConstraints(opts ...AsBoxConstraintsOpts) BoxConstraints {
	o := firstOpts(opts)
	maxExtent := o.MaxExtent.OrElse(math.Inf(1))
	cross := o.CrossAxisExtent.OrElse(c.CrossAxisExtent)
	if c.Axis() == painting.AxisHorizontal {
		return BoxConstraints{MinWidth: o.MinExtent, MaxWidth: maxExtent, MinHeight: cross, MaxHeight: cross}
	}
	return BoxConstraints{MinWidth: cross, MaxWidth: cross, MinHeight: o.MinExtent, MaxHeight: maxExtent}
}

// DebugAssertIsValid panics in debug builds, with the tag
// SliverConstraintsInvalid, if a value is NaN or out of range, or the
// directions are along the same axis. It returns IsNormalized.
//
// Flutter: rendering/sliver.dart SliverConstraints.debugAssertIsValid
func (c SliverConstraints) DebugAssertIsValid(opts ...DebugAssertIsValidOpts) bool {
	o := firstOpts(opts)
	if foundation.DebugMode {
		var errs []string
		verify := func(check bool, message string) {
			if !check {
				errs = append(errs, "  "+message)
			}
		}
		verifyDouble := func(property float64, name string, mustBePositive, mustBeNegative bool) {
			switch {
			case math.IsNaN(property):
				additional := "."
				if mustBePositive {
					additional = ", expected greater than or equal to zero."
				} else if mustBeNegative {
					additional = ", expected less than or equal to zero."
				}
				verify(false, `The "`+name+`" is NaN`+additional)
			case mustBePositive:
				verify(property >= 0, `The "`+name+`" is negative.`)
			case mustBeNegative:
				verify(property <= 0, `The "`+name+`" is positive.`)
			}
		}
		verifyDouble(c.ScrollOffset, "scrollOffset", false, false)
		verifyDouble(c.Overlap, "overlap", false, false)
		verifyDouble(c.CrossAxisExtent, "crossAxisExtent", false, false)
		verifyDouble(c.ScrollOffset, "scrollOffset", true, false)
		verify(painting.AxisDirectionToAxis(c.AxisDirection) != painting.AxisDirectionToAxis(c.CrossAxisDirection),
			`The "axisDirection" and the "crossAxisDirection" are along the same axis.`)
		verifyDouble(c.ViewportMainAxisExtent, "viewportMainAxisExtent", true, false)
		verifyDouble(c.RemainingPaintExtent, "remainingPaintExtent", true, false)
		verifyDouble(c.RemainingCacheExtent, "remainingCacheExtent", true, false)
		verifyDouble(c.CacheOrigin, "cacheOrigin", false, true)
		verifyDouble(c.PrecedingScrollExtent, "precedingScrollExtent", true, false)
		verify(c.IsNormalized(), "The constraints are not normalized.")
		if len(errs) > 0 {
			message := "SliverConstraints is not valid:\n" + strings.Join(errs, "\n")
			if o.InformationCollector != nil {
				message += "\n" + strings.Join(o.InformationCollector(), "\n")
			}
			panic(errors.New(tagSliverConstraintsInvalid, message+"\nThe offending constraints were:\n  "+c.String()))
		}
	}
	return c.IsNormalized()
}

// String describes the constraints.
//
// Flutter: rendering/sliver.dart SliverConstraints.toString
func (c SliverConstraints) String() string {
	properties := []string{
		c.AxisDirection.String(), c.GrowthDirection.String(), c.UserScrollDirection.String(),
		"scrollOffset: " + fixed1(c.ScrollOffset),
		"precedingScrollExtent: " + fixed1(c.PrecedingScrollExtent),
		"remainingPaintExtent: " + fixed1(c.RemainingPaintExtent),
	}
	if c.Overlap != 0 {
		properties = append(properties, "overlap: "+fixed1(c.Overlap))
	}
	properties = append(properties,
		"crossAxisExtent: "+fixed1(c.CrossAxisExtent),
		"crossAxisDirection: "+c.CrossAxisDirection.String(),
		"viewportMainAxisExtent: "+fixed1(c.ViewportMainAxisExtent),
		"remainingCacheExtent: "+fixed1(c.RemainingCacheExtent),
		"cacheOrigin: "+fixed1(c.CacheOrigin))
	return "SliverConstraints(" + strings.Join(properties, ", ") + ")"
}

// SliverGeometry describes the geometry that a sliver chooses during layout:
// how much it scrolls, paints and occupies.
//
// Flutter: rendering/sliver.dart SliverGeometry
type SliverGeometry struct {
	// ScrollExtent is the (estimated) total scrollable extent of the sliver.
	//
	// Flutter: rendering/sliver.dart SliverGeometry.scrollExtent
	ScrollExtent float64
	// PaintOrigin is the offset from the leading edge of the sliver's area at
	// which it starts painting.
	//
	// Flutter: rendering/sliver.dart SliverGeometry.paintOrigin
	PaintOrigin float64
	// PaintExtent is how much of the viewport the sliver paints, from the leading
	// edge of the paint origin.
	//
	// Flutter: rendering/sliver.dart SliverGeometry.paintExtent
	PaintExtent float64
	// LayoutExtent is where the next sliver should start, relative to this one's
	// start; the default is PaintExtent.
	//
	// Flutter: rendering/sliver.dart SliverGeometry.layoutExtent
	LayoutExtent float64
	// MaxPaintExtent is the largest value PaintExtent can have.
	//
	// Flutter: rendering/sliver.dart SliverGeometry.maxPaintExtent
	MaxPaintExtent float64
	// MaxScrollObstructionExtent is how much of the viewport this sliver
	// obstructs when it is pinned (for example an app bar).
	//
	// Flutter: rendering/sliver.dart SliverGeometry.maxScrollObstructionExtent
	MaxScrollObstructionExtent float64
	// HitTestExtent is the distance from the leading edge that hit tests reach;
	// the default is PaintExtent.
	//
	// Flutter: rendering/sliver.dart SliverGeometry.hitTestExtent
	HitTestExtent float64
	// Visible is whether the sliver should be painted; the default is
	// PaintExtent > 0.
	//
	// Flutter: rendering/sliver.dart SliverGeometry.visible
	Visible bool
	// HasVisualOverflow is whether the sliver paints beyond its paint extent, so
	// the viewport must clip.
	//
	// Flutter: rendering/sliver.dart SliverGeometry.hasVisualOverflow
	HasVisualOverflow bool
	// ScrollOffsetCorrection, when it is some, asks the viewport to change its
	// scroll offset by this much and lay out again.
	//
	// Flutter: rendering/sliver.dart SliverGeometry.scrollOffsetCorrection
	ScrollOffsetCorrection foundation.Opt[float64]
	// CacheExtent is how much of the cache area the sliver consumes; the default
	// is LayoutExtent.
	//
	// Flutter: rendering/sliver.dart SliverGeometry.cacheExtent
	CacheExtent float64
	// CrossAxisExtent is the extent in the cross axis, if it differs from the
	// constraints'.
	//
	// Flutter: rendering/sliver.dart SliverGeometry.crossAxisExtent
	CrossAxisExtent foundation.Opt[float64]
}

// SliverGeometryOpts are the parameters of NewSliverGeometry; the ones that are
// none take the default described in the SliverGeometry fields.
//
// Flutter: rendering/sliver.dart SliverGeometry.new
type SliverGeometryOpts struct {
	ScrollExtent               float64
	PaintExtent                float64
	PaintOrigin                float64
	LayoutExtent               foundation.Opt[float64]
	MaxPaintExtent             float64
	MaxScrollObstructionExtent float64
	CrossAxisExtent            foundation.Opt[float64]
	HitTestExtent              foundation.Opt[float64]
	Visible                    foundation.Opt[bool]
	HasVisualOverflow          bool
	ScrollOffsetCorrection     foundation.Opt[float64]
	CacheExtent                foundation.Opt[float64]
}

// NewSliverGeometry creates the geometry. In debug builds a scroll offset
// correction of zero panics with the tag SliverGeometryInvalid.
//
// Flutter: rendering/sliver.dart SliverGeometry.new
func NewSliverGeometry(opts ...SliverGeometryOpts) SliverGeometry {
	o := firstOpts(opts)
	if foundation.DebugMode && o.ScrollOffsetCorrection.IsSome() && o.ScrollOffsetCorrection.Value() == 0 {
		panic(errors.New(tagSliverGeometryInvalid, "the scrollOffsetCorrection of a SliverGeometry must not be zero"))
	}
	layoutExtent := o.LayoutExtent.OrElse(o.PaintExtent)
	return SliverGeometry{
		ScrollExtent: o.ScrollExtent, PaintExtent: o.PaintExtent, PaintOrigin: o.PaintOrigin,
		LayoutExtent: layoutExtent, MaxPaintExtent: o.MaxPaintExtent, MaxScrollObstructionExtent: o.MaxScrollObstructionExtent,
		CrossAxisExtent: o.CrossAxisExtent, HitTestExtent: o.HitTestExtent.OrElse(o.PaintExtent),
		Visible:           o.Visible.OrElse(o.PaintExtent > 0),
		HasVisualOverflow: o.HasVisualOverflow, ScrollOffsetCorrection: o.ScrollOffsetCorrection,
		CacheExtent: o.CacheExtent.OrElse(layoutExtent),
	}
}

// SliverGeometryZero is the geometry of a sliver that takes no space.
//
// Flutter: rendering/sliver.dart SliverGeometry.zero
var SliverGeometryZero = SliverGeometry{}

// SliverGeometryCopy are the values CopyWith replaces.
//
// Flutter: rendering/sliver.dart SliverGeometry.copyWith
type SliverGeometryCopy struct {
	ScrollExtent               foundation.Opt[float64]
	PaintExtent                foundation.Opt[float64]
	PaintOrigin                foundation.Opt[float64]
	LayoutExtent               foundation.Opt[float64]
	MaxPaintExtent             foundation.Opt[float64]
	MaxScrollObstructionExtent foundation.Opt[float64]
	CrossAxisExtent            foundation.Opt[float64]
	HitTestExtent              foundation.Opt[float64]
	Visible                    foundation.Opt[bool]
	HasVisualOverflow          foundation.Opt[bool]
	CacheExtent                foundation.Opt[float64]
}

// CopyWith returns the geometry with the values that are set replaced. The scroll
// offset correction is dropped, as in Flutter.
//
// Flutter: rendering/sliver.dart SliverGeometry.copyWith
func (g SliverGeometry) CopyWith(o SliverGeometryCopy) SliverGeometry {
	return NewSliverGeometry(SliverGeometryOpts{
		ScrollExtent: o.ScrollExtent.OrElse(g.ScrollExtent), PaintExtent: o.PaintExtent.OrElse(g.PaintExtent),
		PaintOrigin:  o.PaintOrigin.OrElse(g.PaintOrigin),
		LayoutExtent: foundation.Some(o.LayoutExtent.OrElse(g.LayoutExtent)), MaxPaintExtent: o.MaxPaintExtent.OrElse(g.MaxPaintExtent),
		MaxScrollObstructionExtent: o.MaxScrollObstructionExtent.OrElse(g.MaxScrollObstructionExtent),
		CrossAxisExtent:            o.CrossAxisExtent, HitTestExtent: foundation.Some(o.HitTestExtent.OrElse(g.HitTestExtent)),
		Visible: foundation.Some(o.Visible.OrElse(g.Visible)), HasVisualOverflow: o.HasVisualOverflow.OrElse(g.HasVisualOverflow),
		CacheExtent: foundation.Some(o.CacheExtent.OrElse(g.CacheExtent)),
	})
}

// compareFloats says how two values differ, for an error message.
func compareFloats(labelA string, valueA float64, labelB string, valueB float64) string {
	if fixed1(valueA) != fixed1(valueB) {
		return "The " + labelA + " is " + fixed1(valueA) + ", but the " + labelB + " is " + fixed1(valueB) + "."
	}
	return "The " + labelA + " is " + dartDouble(valueA) + ", but the " + labelB + " is " + dartDouble(valueB) + ". " +
		"Maybe you have fallen prey to floating point rounding errors, and should explicitly apply the min() or max() functions, " +
		"or the clamp() method, to the " + labelB + "?"
}

// DebugAssertIsValid panics in debug builds, with the tag SliverGeometryInvalid,
// if an extent is negative or the paint extent exceeds the maximum. It returns
// true.
//
// Flutter: rendering/sliver.dart SliverGeometry.debugAssertIsValid
func (g SliverGeometry) DebugAssertIsValid(informationCollector ...InformationCollector) bool {
	if !foundation.DebugMode {
		return true
	}
	verify := func(check bool, summary string, details ...string) {
		if check {
			return
		}
		meta := []any{"geometry", g}
		if len(details) > 0 {
			meta = append(meta, "details", strings.Join(details, "\n"))
		}
		if len(informationCollector) > 0 && informationCollector[0] != nil {
			meta = append(meta, "information", strings.Join(informationCollector[0](), "\n"))
		}
		panic(errors.New(tagSliverGeometryInvalid, "SliverGeometry is not valid: "+summary, meta...))
	}
	verify(g.ScrollExtent >= 0, `The "scrollExtent" is negative.`)
	verify(g.PaintExtent >= 0, `The "paintExtent" is negative.`)
	verify(g.LayoutExtent >= 0, `The "layoutExtent" is negative.`)
	verify(g.CacheExtent >= 0, `The "cacheExtent" is negative.`)
	if g.LayoutExtent > g.PaintExtent {
		verify(false, `The "layoutExtent" exceeds the "paintExtent".`,
			compareFloats("paintExtent", g.PaintExtent, "layoutExtent", g.LayoutExtent))
	}
	if g.PaintExtent-g.MaxPaintExtent > foundation.PrecisionErrorTolerance {
		verify(false, `The "maxPaintExtent" is less than the "paintExtent".`,
			compareFloats("maxPaintExtent", g.MaxPaintExtent, "paintExtent", g.PaintExtent),
			"By definition, a sliver can't paint more than the maximum that it can paint!")
	}
	verify(g.HitTestExtent >= 0, `The "hitTestExtent" is negative.`)
	verify(!g.ScrollOffsetCorrection.IsSome() || g.ScrollOffsetCorrection.Value() != 0, `The "scrollOffsetCorrection" is zero.`)
	return true
}

// String describes the geometry.
//
// Flutter: rendering/sliver.dart SliverGeometry.toString
func (g SliverGeometry) String() string {
	parts := []string{"scrollExtent: " + dartDouble(g.ScrollExtent)}
	switch {
	case g.PaintExtent > 0:
		s := "paintExtent: " + dartDouble(g.PaintExtent)
		if !g.Visible {
			s += " but not painting"
		}
		parts = append(parts, s)
	case g.PaintExtent == 0:
		if g.Visible {
			parts = append(parts, "paintExtent: 0.0")
		} else {
			parts = append(parts, "hidden")
		}
	default:
		parts = append(parts, "paintExtent: "+dartDouble(g.PaintExtent)+"!")
	}
	if g.PaintOrigin != 0 {
		parts = append(parts, "paintOrigin: "+dartDouble(g.PaintOrigin))
	}
	if g.LayoutExtent != g.PaintExtent {
		parts = append(parts, "layoutExtent: "+dartDouble(g.LayoutExtent))
	}
	parts = append(parts, "maxPaintExtent: "+dartDouble(g.MaxPaintExtent))
	if g.HitTestExtent != g.PaintExtent {
		parts = append(parts, "hitTestExtent: "+dartDouble(g.HitTestExtent))
	}
	if g.HasVisualOverflow {
		parts = append(parts, "hasVisualOverflow: true")
	}
	if c, ok := g.ScrollOffsetCorrection.Get(); ok {
		parts = append(parts, "scrollOffsetCorrection: "+dartDouble(c))
	}
	if g.CacheExtent != 0 {
		parts = append(parts, "cacheExtent: "+dartDouble(g.CacheExtent))
	}
	return "SliverGeometry(" + strings.Join(parts, ", ") + ")"
}

// SliverHitTest hit tests a position in the coordinates of a sliver: the
// distances along and across the main axis from the sliver's origin.
//
// Flutter: rendering/sliver.dart SliverHitTest
type SliverHitTest func(result *SliverHitTestResult, mainAxisPosition, crossAxisPosition float64) bool

// SliverHitTestResult is a HitTestResult with the helper for the sliver
// protocol.
//
// Flutter: rendering/sliver.dart SliverHitTestResult
type SliverHitTestResult struct {
	*gestures.HitTestResult
}

// NewSliverHitTestResult creates an empty result.
//
// Flutter: rendering/sliver.dart SliverHitTestResult.new
func NewSliverHitTestResult() *SliverHitTestResult {
	return &SliverHitTestResult{gestures.NewHitTestResult()}
}

// NewSliverHitTestResultWrap makes a sliver result that adds to the given result.
//
// Flutter: rendering/sliver.dart SliverHitTestResult.wrap
func NewSliverHitTestResultWrap(result *gestures.HitTestResult) *SliverHitTestResult {
	return &SliverHitTestResult{gestures.NewHitTestResultWrap(result)}
}

// AddWithAxisOffset hit tests a child sliver that is painted at the offset (nil
// for none): the positions the child sees are shifted by the axis offsets.
//
// Flutter: rendering/sliver.dart SliverHitTestResult.addWithAxisOffset
func (r *SliverHitTestResult) AddWithAxisOffset(paintOffset *ui.Offset, mainAxisOffset, crossAxisOffset, mainAxisPosition, crossAxisPosition float64, hitTest SliverHitTest) bool {
	if paintOffset != nil {
		r.PushOffset(paintOffset.Neg())
	}
	isHit := hitTest(r, mainAxisPosition-mainAxisOffset, crossAxisPosition-crossAxisOffset)
	if paintOffset != nil {
		r.PopTransform()
	}
	return isHit
}

// SliverHitTestEntry is a hit test entry of a sliver: the target and the
// position of the hit along and across the main axis.
//
// Flutter: rendering/sliver.dart SliverHitTestEntry
type SliverHitTestEntry struct {
	gestures.HitTestEntryBase
	// MainAxisPosition is the distance in the main axis from the origin of the
	// sliver.
	//
	// Flutter: rendering/sliver.dart SliverHitTestEntry.mainAxisPosition
	MainAxisPosition float64
	// CrossAxisPosition is the distance in the cross axis from the origin.
	//
	// Flutter: rendering/sliver.dart SliverHitTestEntry.crossAxisPosition
	CrossAxisPosition float64
}

// NewSliverHitTestEntry creates an entry for the target at the positions.
//
// Flutter: rendering/sliver.dart SliverHitTestEntry.new
func NewSliverHitTestEntry(target Sliver, mainAxisPosition, crossAxisPosition float64) *SliverHitTestEntry {
	return &SliverHitTestEntry{HitTestEntryBase: *gestures.NewHitTestEntry(target), MainAxisPosition: mainAxisPosition, CrossAxisPosition: crossAxisPosition}
}

// String describes the entry.
//
// Flutter: rendering/sliver.dart SliverHitTestEntry.toString
func (e *SliverHitTestEntry) String() string {
	return fmt.Sprintf("%s@(mainAxis: %v, crossAxis: %v)", foundation.RuntimeTypeName(e.Target()), dartDouble(e.MainAxisPosition), dartDouble(e.CrossAxisPosition))
}

// SliverLogicalParentData is the parent data of a sliver's child that is
// positioned by its layout offset along the main axis.
//
// Flutter: rendering/sliver.dart SliverLogicalParentData
type SliverLogicalParentData struct {
	ParentDataBase
	// LayoutOffset is the offset of the child from the zero scroll offset of the
	// parent, along the main axis; none before the first layout.
	//
	// Flutter: rendering/sliver.dart SliverLogicalParentData.layoutOffset
	LayoutOffset foundation.Opt[float64]
}

// String is "layoutOffset=" and the offset.
//
// Flutter: rendering/sliver.dart SliverLogicalParentData.toString
func (d *SliverLogicalParentData) String() string {
	if o, ok := d.LayoutOffset.Get(); ok {
		return "layoutOffset=" + fixed1(o)
	}
	return "layoutOffset=None"
}

func (d *SliverLogicalParentData) sliverLogical() *SliverLogicalParentData { return d }

// sliverLogicalHolder is a ParentData that is or embeds a SliverLogicalParentData.
type sliverLogicalHolder interface {
	ParentData
	sliverLogical() *SliverLogicalParentData
}

// SliverLogicalContainerParentData is the parent data of the children of a sliver
// with a list of children laid out by a layout offset.
//
// Flutter: rendering/sliver.dart SliverLogicalContainerParentData
type SliverLogicalContainerParentData struct {
	SliverLogicalParentData
	ContainerParentData[Sliver]
}

// Detach panics in debug builds if the sibling links are still set.
//
// Flutter: rendering/object.dart ContainerParentDataMixin.detach
func (d *SliverLogicalContainerParentData) Detach() { d.checkDetach() }

// SliverPhysicalParentData is the parent data of a sliver's child that is
// positioned by a paint offset in the coordinates of the sliver.
//
// Flutter: rendering/sliver.dart SliverPhysicalParentData
type SliverPhysicalParentData struct {
	ParentDataBase
	// PaintOffset is the position of the child relative to the parent.
	//
	// Flutter: rendering/sliver.dart SliverPhysicalParentData.paintOffset
	PaintOffset ui.Offset
	// CrossAxisFlex is the flex factor in the cross axis (for a grid).
	//
	// Flutter: rendering/sliver.dart SliverPhysicalParentData.crossAxisFlex
	CrossAxisFlex foundation.Opt[int]
}

// ApplyPaintTransform translates the transform by the paint offset.
//
// Flutter: rendering/sliver.dart SliverPhysicalParentData.applyPaintTransform
func (d *SliverPhysicalParentData) ApplyPaintTransform(transform *ui.Matrix4) {
	transform.Translate(float32(d.PaintOffset.Dx), float32(d.PaintOffset.Dy), 0)
}

// String is "paintOffset=" and the offset.
//
// Flutter: rendering/sliver.dart SliverPhysicalParentData.toString
func (d *SliverPhysicalParentData) String() string { return "paintOffset=" + d.PaintOffset.String() }

func (d *SliverPhysicalParentData) sliverPhysical() *SliverPhysicalParentData { return d }

// sliverPhysicalHolder is a ParentData that is or embeds a SliverPhysicalParentData.
type sliverPhysicalHolder interface {
	ParentData
	sliverPhysical() *SliverPhysicalParentData
}

// SliverPhysicalContainerParentData is the parent data of the children of a sliver
// with a list of children positioned by a paint offset.
//
// Flutter: rendering/sliver.dart SliverPhysicalContainerParentData
type SliverPhysicalContainerParentData struct {
	SliverPhysicalParentData
	ContainerParentData[Sliver]
}

// Detach panics in debug builds if the sibling links are still set.
//
// Flutter: rendering/object.dart ContainerParentDataMixin.detach
func (d *SliverPhysicalContainerParentData) Detach() { d.checkDetach() }

// SliverLogicalParentDataOf is the SliverLogicalParentData of a child.
//
// Flutter: no counterpart (the cast `child.parentData! as SliverLogicalParentData`)
func SliverLogicalParentDataOf(child RenderObject) *SliverLogicalParentData {
	return child.ParentData().(sliverLogicalHolder).sliverLogical()
}

// SliverPhysicalParentDataOf is the SliverPhysicalParentData of a child.
//
// Flutter: no counterpart (the cast `child.parentData! as SliverPhysicalParentData`)
func SliverPhysicalParentDataOf(child RenderObject) *SliverPhysicalParentData {
	return child.ParentData().(sliverPhysicalHolder).sliverPhysical()
}

// Sliver is a render object that is laid out with SliverConstraints and chooses a
// SliverGeometry. A sliver type embeds RenderSliver (or a type that embeds it)
// and calls InitRenderSliver.
//
// Flutter: rendering/sliver.dart RenderSliver
type Sliver interface {
	RenderObject
	// AsRenderSliver returns the RenderSliver that the sliver embeds.
	AsRenderSliver() *RenderSliver
	// EnsureSemantics is whether the sliver keeps its semantics even when it is
	// not visible.
	//
	// Flutter: rendering/sliver.dart RenderSliver.ensureSemantics
	EnsureSemantics() bool
	// SliverConstraints are the constraints that the parent gave at the last
	// layout.
	//
	// Flutter: rendering/sliver.dart RenderSliver.constraints
	SliverConstraints() SliverConstraints
	// HasGeometry is whether the sliver has chosen its geometry.
	HasGeometry() bool
	// Geometry is the geometry that the sliver chose at the last layout. It
	// panics with SliverNotLaidOut in debug builds if there is none.
	//
	// Flutter: rendering/sliver.dart RenderSliver.geometry
	Geometry() SliverGeometry
	// SetGeometry sets the geometry. Only the sliver itself may call it, from
	// PerformLayout.
	//
	// Flutter: rendering/sliver.dart RenderSliver.geometry
	SetGeometry(value SliverGeometry)
	// CenterOffsetAdjustment is how far a viewport that has this sliver as its
	// center moves the zero scroll offset (see RenderSliverPersistentHeader).
	//
	// Flutter: rendering/sliver.dart RenderSliver.centerOffsetAdjustment
	CenterOffsetAdjustment() float64
	// HitTest adds the sliver and its descendants under the position to the
	// result.
	//
	// Flutter: rendering/sliver.dart RenderSliver.hitTest
	HitTest(result *SliverHitTestResult, mainAxisPosition, crossAxisPosition float64) bool
	// HitTestSelf is whether the sliver itself is hit at the position.
	//
	// Flutter: rendering/sliver.dart RenderSliver.hitTestSelf
	HitTestSelf(mainAxisPosition, crossAxisPosition float64) bool
	// HitTestChildren hit tests the children at the position.
	//
	// Flutter: rendering/sliver.dart RenderSliver.hitTestChildren
	HitTestChildren(result *SliverHitTestResult, mainAxisPosition, crossAxisPosition float64) bool
	// ChildMainAxisPosition is the position of the child in the main axis,
	// relative to the leading edge of this sliver's paint area.
	//
	// Flutter: rendering/sliver.dart RenderSliver.childMainAxisPosition
	ChildMainAxisPosition(child RenderObject) float64
	// ChildCrossAxisPosition is the position of the child in the cross axis.
	//
	// Flutter: rendering/sliver.dart RenderSliver.childCrossAxisPosition
	ChildCrossAxisPosition(child RenderObject) float64
	// ChildScrollOffset is the scroll offset of the child, relative to the start
	// of this sliver, or none if the sliver can not say.
	//
	// Flutter: rendering/sliver.dart RenderSliver.childScrollOffset
	ChildScrollOffset(child RenderObject) foundation.Opt[float64]
	// CalculatePaintOffset is how much of the range from..to (scroll offsets) is
	// visible in the viewport.
	//
	// Flutter: rendering/sliver.dart RenderSliver.calculatePaintOffset
	CalculatePaintOffset(constraints SliverConstraints, from, to float64) float64
	// CalculateCacheOffset is how much of the range from..to (scroll offsets) is
	// in the cache area.
	//
	// Flutter: rendering/sliver.dart RenderSliver.calculateCacheOffset
	CalculateCacheOffset(constraints SliverConstraints, from, to float64) float64
	// GetAbsoluteSizeRelativeToOrigin is the size of the sliver's paint area,
	// signed by the direction in which it extends from the origin.
	//
	// Flutter: rendering/sliver.dart RenderSliver.getAbsoluteSizeRelativeToOrigin
	GetAbsoluteSizeRelativeToOrigin() ui.Size
	// GetAbsoluteSize is the size of the sliver's paint area.
	//
	// Flutter: rendering/sliver.dart RenderSliver.getAbsoluteSize
	GetAbsoluteSize() ui.Size
	// GetMaxPaintRect is the largest rectangle the sliver could paint.
	//
	// Flutter: rendering/sliver.dart RenderSliver.getMaxPaintRect
	GetMaxPaintRect() ui.Rect
}

// RenderSliver is the base of the slivers: render objects that are portions of a
// scrollable area, laid out with SliverConstraints.
//
// Flutter: rendering/sliver.dart RenderSliver
type RenderSliver struct {
	RenderObjectBase
	geometry foundation.Opt[SliverGeometry]
}

// InitRenderSliver sets the outer object of a sliver.
//
// Flutter: no counterpart (the self pattern, SPEC §12.5)
func InitRenderSliver(s *RenderSliver, self Sliver) { InitRenderObject(&s.RenderObjectBase, self) }

// AsRenderSliver returns the sliver itself.
func (s *RenderSliver) AsRenderSliver() *RenderSliver { return s }

func (s *RenderSliver) debugSelf() Sliver { return s.self.(Sliver) }

// EnsureSemantics is false.
//
// Flutter: rendering/sliver.dart RenderSliver.ensureSemantics
func (s *RenderSliver) EnsureSemantics() bool { return false }

// SliverConstraints are the constraints that the parent gave at the last layout.
//
// Flutter: rendering/sliver.dart RenderSliver.constraints
func (s *RenderSliver) SliverConstraints() SliverConstraints {
	return s.Constraints().(SliverConstraints)
}

// HasGeometry is whether the sliver has chosen its geometry.
func (s *RenderSliver) HasGeometry() bool { return s.geometry.IsSome() }

// Geometry is the geometry that the sliver chose at the last layout. It panics
// with the tag SliverNotLaidOut in debug builds if there is none.
//
// Flutter: rendering/sliver.dart RenderSliver.geometry
func (s *RenderSliver) Geometry() SliverGeometry {
	g, ok := s.geometry.Get()
	if !ok && foundation.DebugMode {
		panic(errors.New(tagSliverNotLaidOut, "RenderSliver was not laid out: "+foundation.DescribeIdentity(s.self),
			"object", foundation.DescribeIdentity(s.self)))
	}
	return g
}

// SetGeometry sets the geometry. Only the sliver itself may call it, from
// PerformLayout (or PerformResize if it is SizedByParent); a debug build panics
// with SliverGeometrySetterInvalid anywhere else.
//
// Flutter: rendering/sliver.dart RenderSliver.geometry
func (s *RenderSliver) SetGeometry(value SliverGeometry) {
	if foundation.DebugMode {
		sized := s.self.SizedByParent()
		if s.debugDoingThisResize && s.debugDoingThisLayout {
			panic(errors.New(tagSliverGeometrySetterInvalid, "the sliver is resizing and laying out at the same time",
				"object", foundation.DescribeIdentity(s.self)))
		}
		if (!sized || !s.debugDoingThisResize) && (sized || !s.debugDoingThisLayout) {
			msg := "RenderSliver geometry setter called incorrectly. "
			if s.debugDoingThisLayout {
				msg += "It appears that the geometry setter was called from PerformLayout(). "
			} else {
				msg += "The geometry setter was called from outside layout (neither PerformResize() nor PerformLayout() were being run for this object). "
				if s.owner != nil && s.owner.debugDoingLayout {
					msg += "Only the object itself can set its geometry. It is a contract violation for other objects to set it. "
				}
			}
			if sized {
				msg += "Because this RenderSliver has SizedByParent set to true, it must set its geometry in PerformResize()."
			} else {
				msg += "Because this RenderSliver has SizedByParent set to false, it must set its geometry in PerformLayout()."
			}
			panic(errors.New(tagSliverGeometrySetterInvalid, msg, "object", foundation.DescribeIdentity(s.self)))
		}
	}
	s.geometry = foundation.Some(value)
}

// SemanticBounds are the paint bounds.
//
// Flutter: rendering/sliver.dart RenderSliver.semanticBounds
func (s *RenderSliver) SemanticBounds() ui.Rect { return s.self.PaintBounds() }

// PaintBounds are the bounds of the paint area: the paint extent along the main
// axis and the cross axis extent across it.
//
// Flutter: rendering/sliver.dart RenderSliver.paintBounds
func (s *RenderSliver) PaintBounds() ui.Rect {
	c := s.debugSelf().SliverConstraints()
	if c.Axis() == painting.AxisHorizontal {
		return ui.NewRectFromLTWH(0, 0, s.debugSelf().Geometry().PaintExtent, c.CrossAxisExtent)
	}
	return ui.NewRectFromLTWH(0, 0, c.CrossAxisExtent, s.debugSelf().Geometry().PaintExtent)
}

// DebugResetSize does nothing: slivers have no size.
//
// Flutter: rendering/sliver.dart RenderSliver.debugResetSize
func (s *RenderSliver) DebugResetSize() {}

// DebugAssertDoesMeetConstraints checks the geometry (SliverGeometryInvalid) and
// that it paints inside the remaining paint extent
// (SliverGeometryViolatesConstraints).
//
// Flutter: rendering/sliver.dart RenderSliver.debugAssertDoesMeetConstraints
func (s *RenderSliver) DebugAssertDoesMeetConstraints() {
	if !foundation.DebugMode {
		return
	}
	self := s.debugSelf()
	g := self.Geometry()
	g.DebugAssertIsValid(func() []string {
		return []string{"The RenderSliver that returned the offending geometry was: " + foundation.DescribeIdentity(s.self)}
	})
	c := self.SliverConstraints()
	if g.PaintOrigin+g.PaintExtent > c.RemainingPaintExtent {
		panic(errors.New(tagSliverGeometryViolatesConstraints,
			"SliverGeometry has a paintOffset that exceeds the remainingPaintExtent from the constraints. "+
				compareFloats("remainingPaintExtent", c.RemainingPaintExtent, "paintOrigin + paintExtent", g.PaintOrigin+g.PaintExtent)+
				" The paintOrigin and paintExtent must cause the child sliver to paint within the viewport, and so cannot exceed the remainingPaintExtent.",
			"object", foundation.DescribeIdentity(s.self)))
	}
}

// PerformResize panics: a sliver is never sized by its parent.
//
// Flutter: rendering/sliver.dart RenderSliver.performResize
func (s *RenderSliver) PerformResize() {
	panic(errors.New(tagSliverPerformResize, "a RenderSliver is not sized by its parent; PerformResize must not be called",
		"object", foundation.DescribeIdentity(s.self)))
}

// CenterOffsetAdjustment is zero.
//
// Flutter: rendering/sliver.dart RenderSliver.centerOffsetAdjustment
func (s *RenderSliver) CenterOffsetAdjustment() float64 { return 0 }

// HitTest adds the sliver to the result if the position is inside its hit test
// extent and it or a child is hit.
//
// Flutter: rendering/sliver.dart RenderSliver.hitTest
func (s *RenderSliver) HitTest(result *SliverHitTestResult, mainAxisPosition, crossAxisPosition float64) bool {
	self := s.debugSelf()
	if mainAxisPosition >= 0 && mainAxisPosition < self.Geometry().HitTestExtent &&
		crossAxisPosition >= 0 && crossAxisPosition < self.SliverConstraints().CrossAxisExtent {
		if self.HitTestChildren(result, mainAxisPosition, crossAxisPosition) || self.HitTestSelf(mainAxisPosition, crossAxisPosition) {
			result.Add(NewSliverHitTestEntry(self, mainAxisPosition, crossAxisPosition))
			return true
		}
	}
	return false
}

// HitTestSelf is false.
//
// Flutter: rendering/sliver.dart RenderSliver.hitTestSelf
func (s *RenderSliver) HitTestSelf(mainAxisPosition, crossAxisPosition float64) bool { return false }

// HitTestChildren is false.
//
// Flutter: rendering/sliver.dart RenderSliver.hitTestChildren
func (s *RenderSliver) HitTestChildren(*SliverHitTestResult, float64, float64) bool { return false }

// CalculatePaintOffset is how much of the range from..to (scroll offsets of
// this sliver) is visible in the viewport.
//
// Flutter: rendering/sliver.dart RenderSliver.calculatePaintOffset
func (s *RenderSliver) CalculatePaintOffset(constraints SliverConstraints, from, to float64) float64 {
	if foundation.DebugMode && from > to {
		panic(errors.New(tagSliverRangeInvalid, "CalculatePaintOffset needs from <= to", "from", from, "to", to))
	}
	a := constraints.ScrollOffset
	b := constraints.ScrollOffset + constraints.RemainingPaintExtent
	return ui.ClampDouble(ui.ClampDouble(to, a, b)-ui.ClampDouble(from, a, b), 0, constraints.RemainingPaintExtent)
}

// CalculateCacheOffset is how much of the range from..to (scroll offsets of this
// sliver) is in the cache area.
//
// Flutter: rendering/sliver.dart RenderSliver.calculateCacheOffset
func (s *RenderSliver) CalculateCacheOffset(constraints SliverConstraints, from, to float64) float64 {
	if foundation.DebugMode && from > to {
		panic(errors.New(tagSliverRangeInvalid, "CalculateCacheOffset needs from <= to", "from", from, "to", to))
	}
	a := constraints.ScrollOffset + constraints.CacheOrigin
	b := constraints.ScrollOffset + constraints.RemainingCacheExtent
	return ui.ClampDouble(ui.ClampDouble(to, a, b)-ui.ClampDouble(from, a, b), 0, constraints.RemainingCacheExtent)
}

// ChildMainAxisPosition panics in debug builds: a sliver with a child must
// override it.
//
// Flutter: rendering/sliver.dart RenderSliver.childMainAxisPosition
func (s *RenderSliver) ChildMainAxisPosition(RenderObject) float64 {
	if foundation.DebugMode {
		panic(errors.New(tagSliverChildPositionMissing, foundation.RuntimeTypeName(s.self)+" does not implement ChildMainAxisPosition.",
			"object", foundation.DescribeIdentity(s.self)))
	}
	return 0
}

// ChildCrossAxisPosition is zero.
//
// Flutter: rendering/sliver.dart RenderSliver.childCrossAxisPosition
func (s *RenderSliver) ChildCrossAxisPosition(RenderObject) float64 { return 0 }

// ChildScrollOffset is zero.
//
// Flutter: rendering/sliver.dart RenderSliver.childScrollOffset
func (s *RenderSliver) ChildScrollOffset(RenderObject) foundation.Opt[float64] {
	return foundation.Some(0.0)
}

// ApplyPaintTransform panics in debug builds: a sliver with a child must
// override it.
//
// Flutter: rendering/sliver.dart RenderSliver.applyPaintTransform
func (s *RenderSliver) ApplyPaintTransform(RenderObject, *ui.Matrix4) {
	if foundation.DebugMode {
		panic(errors.New(tagSliverPaintTransformMissing, foundation.RuntimeTypeName(s.self)+" does not implement ApplyPaintTransform.",
			"object", foundation.DescribeIdentity(s.self)))
	}
}

// GetAbsoluteSizeRelativeToOrigin is the size of the paint area, with the sign
// of the direction in which it extends from the origin.
//
// Flutter: rendering/sliver.dart RenderSliver.getAbsoluteSizeRelativeToOrigin
func (s *RenderSliver) GetAbsoluteSizeRelativeToOrigin() ui.Size {
	self := s.debugSelf()
	c, g := self.SliverConstraints(), self.Geometry()
	switch ApplyGrowthDirectionToAxisDirection(c.AxisDirection, c.GrowthDirection) {
	case painting.AxisDirectionUp:
		return ui.Size{Width: c.CrossAxisExtent, Height: -g.PaintExtent}
	case painting.AxisDirectionLeft:
		return ui.Size{Width: -g.PaintExtent, Height: c.CrossAxisExtent}
	case painting.AxisDirectionRight:
		return ui.Size{Width: g.PaintExtent, Height: c.CrossAxisExtent}
	}
	return ui.Size{Width: c.CrossAxisExtent, Height: g.PaintExtent}
}

// GetAbsoluteSize is the size of the paint area.
//
// Flutter: rendering/sliver.dart RenderSliver.getAbsoluteSize
func (s *RenderSliver) GetAbsoluteSize() ui.Size {
	self := s.debugSelf()
	c, g := self.SliverConstraints(), self.Geometry()
	switch c.AxisDirection {
	case painting.AxisDirectionUp, painting.AxisDirectionDown:
		return ui.Size{Width: c.CrossAxisExtent, Height: g.PaintExtent}
	}
	return ui.Size{Width: g.PaintExtent, Height: c.CrossAxisExtent}
}

// GetMaxPaintRect is the largest rectangle the sliver could paint, in its own
// coordinates, at the current scroll offset.
//
// Flutter: rendering/sliver.dart RenderSliver.getMaxPaintRect
func (s *RenderSliver) GetMaxPaintRect() ui.Rect {
	self := s.debugSelf()
	if !self.HasGeometry() || self.Geometry() == SliverGeometryZero {
		return ui.Rect{}
	}
	c, g := self.SliverConstraints(), self.Geometry()
	maxPaintExtent := g.MaxPaintExtent
	if math.IsInf(maxPaintExtent, 0) {
		maxPaintExtent = c.ScrollOffset + g.CacheExtent + c.CacheOrigin
	}
	paintExtent := g.PaintExtent
	leadingOffset := ui.ClampDouble(c.ScrollOffset, 0, g.ScrollExtent-g.MaxScrollObstructionExtent)
	crossAxisExtent := g.CrossAxisExtent.OrElse(c.CrossAxisExtent)
	var rect ui.Rect
	if c.Axis() == painting.AxisHorizontal {
		rect = ui.NewRectFromLTWH(-leadingOffset, 0, maxPaintExtent, crossAxisExtent)
	} else {
		rect = ui.NewRectFromLTWH(0, -leadingOffset, crossAxisExtent, maxPaintExtent)
	}
	switch ApplyGrowthDirectionToAxisDirection(c.AxisDirection, c.GrowthDirection) {
	case painting.AxisDirectionLeft:
		return ui.NewRectFromLTRB(paintExtent-rect.Right, rect.Top, paintExtent-rect.Left, rect.Bottom)
	case painting.AxisDirectionUp:
		return ui.NewRectFromLTRB(rect.Left, paintExtent-rect.Bottom, rect.Right, paintExtent-rect.Top)
	}
	return rect
}

func debugDrawArrow(canvas *ui.Canvas, paint *ui.Paint, p0, p1 ui.Offset, direction GrowthDirection) {
	if p0 == p1 {
		return
	}
	d := p1.Sub(p0).Distance() * 0.2
	dx1, dx2, dy1, dy2 := d, d, d, d
	if direction == GrowthDirectionReverse {
		p0, p1 = p1, p0
		dx1, dx2, dy1, dy2 = -d, -d, -d, -d
	}
	if p0.Dx == p1.Dx {
		dx2 = -dx2
	} else {
		dy2 = -dy2
	}
	path := ui.NewPath()
	path.MoveTo(p0.Dx, p0.Dy)
	path.LineTo(p1.Dx, p1.Dy)
	path.MoveTo(p1.Dx-dx1, p1.Dy-dy1)
	path.LineTo(p1.Dx, p1.Dy)
	path.LineTo(p1.Dx-dx2, p1.Dy-dy2)
	canvas.DrawPath(path, paint)
}

// DebugPaint paints the arrows that show the growth direction, if
// DebugOptions.PaintSizeEnabled is set.
//
// Flutter: rendering/sliver.dart RenderSliver.debugPaint
func (s *RenderSliver) DebugPaint(context *PaintingContext, offset ui.Offset) {
	if !foundation.DebugMode || !s.DebugOptions().PaintSizeEnabled {
		return
	}
	self := s.debugSelf()
	c, g := self.SliverConstraints(), self.Geometry()
	strokeWidth := math.Min(4, g.PaintExtent/30)
	paint := ui.NewPaint()
	paint.SetColor(ui.NewColor(0xFF33CC33))
	paint.SetStrokeWidth(strokeWidth)
	paint.SetStyle(ui.PaintingStyleStroke)
	paint.SetMaskFilter(ui.NewMaskFilterBlur(ui.BlurStyleSolid, strokeWidth))
	arrowExtent := g.PaintExtent
	padding := math.Max(2, strokeWidth)
	canvas := context.Canvas()
	canvas.DrawCircle(offset.Translate(padding, padding), padding*0.5, paint)
	if c.Axis() == painting.AxisVertical {
		canvas.DrawLine(offset, offset.Translate(c.CrossAxisExtent, 0), paint)
		debugDrawArrow(canvas, paint, offset.Translate(c.CrossAxisExtent*1/4, padding), offset.Translate(c.CrossAxisExtent*1/4, arrowExtent-padding), c.NormalizedGrowthDirection())
		debugDrawArrow(canvas, paint, offset.Translate(c.CrossAxisExtent*3/4, padding), offset.Translate(c.CrossAxisExtent*3/4, arrowExtent-padding), c.NormalizedGrowthDirection())
	} else {
		canvas.DrawLine(offset, offset.Translate(0, c.CrossAxisExtent), paint)
		debugDrawArrow(canvas, paint, offset.Translate(padding, c.CrossAxisExtent*1/4), offset.Translate(arrowExtent-padding, c.CrossAxisExtent*1/4), c.NormalizedGrowthDirection())
		debugDrawArrow(canvas, paint, offset.Translate(padding, c.CrossAxisExtent*3/4), offset.Translate(arrowExtent-padding, c.CrossAxisExtent*3/4), c.NormalizedGrowthDirection())
	}
}

// HandleEvent does nothing.
//
// Flutter: rendering/sliver.dart RenderSliver.handleEvent
func (s *RenderSliver) HandleEvent(gestures.PointerEvent, gestures.HitTestEntry) {}

// DebugFillProperties adds the geometry.
//
// Flutter: rendering/sliver.dart RenderSliver.debugFillProperties
func (s *RenderSliver) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	s.RenderObjectBase.DebugFillProperties(p)
	p.Add(foundation.NewDiagnosticsPropertyOpt("geometry", s.geometry))
}

// SliverHelpers are the helpers for a sliver that has a box child: hit testing
// the box and applying its paint transform. Embed nothing: the sliver calls the
// functions with itself.
//
// Flutter: rendering/sliver.dart RenderSliverHelpers

func sliverRightWayUp(c SliverConstraints) bool {
	reversed := painting.AxisDirectionIsReversed(c.AxisDirection)
	if c.GrowthDirection == GrowthDirectionForward {
		return !reversed
	}
	return reversed
}

// SliverHitTestBoxChild hit tests a box child of the sliver: it maps the
// positions of the sliver to the coordinates of the box.
//
// Flutter: rendering/sliver.dart RenderSliverHelpers.hitTestBoxChild
func SliverHitTestBoxChild(s Sliver, result *BoxHitTestResult, child Box, mainAxisPosition, crossAxisPosition float64) bool {
	c, g := s.SliverConstraints(), s.Geometry()
	rightWayUp := sliverRightWayUp(c)
	delta := s.ChildMainAxisPosition(child)
	crossAxisDelta := s.ChildCrossAxisPosition(child)
	absolutePosition := mainAxisPosition - delta
	absoluteCrossAxisPosition := crossAxisPosition - crossAxisDelta
	var paintOffset, transformedPosition ui.Offset
	if c.Axis() == painting.AxisHorizontal {
		if !rightWayUp {
			absolutePosition = child.Size().Width - absolutePosition
			delta = g.PaintExtent - child.Size().Width - delta
		}
		paintOffset = ui.Offset{Dx: delta, Dy: crossAxisDelta}
		transformedPosition = ui.Offset{Dx: absolutePosition, Dy: absoluteCrossAxisPosition}
	} else {
		if !rightWayUp {
			absolutePosition = child.Size().Height - absolutePosition
			delta = g.PaintExtent - child.Size().Height - delta
		}
		paintOffset = ui.Offset{Dx: crossAxisDelta, Dy: delta}
		transformedPosition = ui.Offset{Dx: absoluteCrossAxisPosition, Dy: absolutePosition}
	}
	return result.AddWithOutOfBandPosition(OutOfBandOpts{PaintOffset: &paintOffset}, func(r *BoxHitTestResult) bool {
		return child.HitTest(r, transformedPosition)
	})
}

// SliverApplyPaintTransformForBoxChild applies to the transform the translation
// that paints the box child of the sliver.
//
// Flutter: rendering/sliver.dart RenderSliverHelpers.applyPaintTransformForBoxChild
func SliverApplyPaintTransformForBoxChild(s Sliver, child Box, transform *ui.Matrix4) {
	c, g := s.SliverConstraints(), s.Geometry()
	rightWayUp := sliverRightWayUp(c)
	delta := s.ChildMainAxisPosition(child)
	crossAxisDelta := s.ChildCrossAxisPosition(child)
	if c.Axis() == painting.AxisHorizontal {
		if !rightWayUp {
			delta = g.PaintExtent - child.Size().Width - delta
		}
		transform.Translate(float32(delta), float32(crossAxisDelta), 0)
	} else {
		if !rightWayUp {
			delta = g.PaintExtent - child.Size().Height - delta
		}
		transform.Translate(float32(crossAxisDelta), float32(delta), 0)
	}
}

// RenderSliverSingleBoxAdapter is the base of the slivers that have one box
// child.
//
// Flutter: rendering/sliver.dart RenderSliverSingleBoxAdapter
type RenderSliverSingleBoxAdapter struct {
	RenderSliver
	RenderObjectWithChild[Box]
}

// InitRenderSliverSingleBoxAdapter sets the outer object and the child (nil for
// none).
//
// Flutter: rendering/sliver.dart RenderSliverSingleBoxAdapter.new
func InitRenderSliverSingleBoxAdapter(a *RenderSliverSingleBoxAdapter, self Sliver, child Box) {
	InitRenderSliver(&a.RenderSliver, self)
	InitRenderObjectWithChild(&a.RenderObjectWithChild, self)
	if !isNil(child) {
		a.SetChild(child)
	}
}

func (a *RenderSliverSingleBoxAdapter) hasChild() bool { return !isNil(a.Child()) }

// SetupParentData gives the child a SliverPhysicalParentData if it has none.
//
// Flutter: rendering/sliver.dart RenderSliverSingleBoxAdapter.setupParentData
func (a *RenderSliverSingleBoxAdapter) SetupParentData(child RenderObject) {
	if _, ok := child.ParentData().(sliverPhysicalHolder); !ok {
		child.AsRenderObject().SetParentData(&SliverPhysicalParentData{})
	}
}

// SetChildParentData sets the paint offset of the child from the constraints
// and the geometry.
//
// Flutter: rendering/sliver.dart RenderSliverSingleBoxAdapter.setChildParentData
func (a *RenderSliverSingleBoxAdapter) SetChildParentData(child RenderObject, constraints SliverConstraints, geometry SliverGeometry) {
	data := SliverPhysicalParentDataOf(child)
	switch ApplyGrowthDirectionToAxisDirection(constraints.AxisDirection, constraints.GrowthDirection) {
	case painting.AxisDirectionUp:
		data.PaintOffset = ui.Offset{Dy: geometry.PaintExtent + constraints.ScrollOffset - geometry.ScrollExtent}
	case painting.AxisDirectionLeft:
		data.PaintOffset = ui.Offset{Dx: geometry.PaintExtent + constraints.ScrollOffset - geometry.ScrollExtent}
	case painting.AxisDirectionRight:
		data.PaintOffset = ui.Offset{Dx: -constraints.ScrollOffset}
	default:
		data.PaintOffset = ui.Offset{Dy: -constraints.ScrollOffset}
	}
}

// HitTestChildren hit tests the box child.
//
// Flutter: rendering/sliver.dart RenderSliverSingleBoxAdapter.hitTestChildren
func (a *RenderSliverSingleBoxAdapter) HitTestChildren(result *SliverHitTestResult, mainAxisPosition, crossAxisPosition float64) bool {
	if a.hasChild() {
		return SliverHitTestBoxChild(a.debugSelf(), NewBoxHitTestResultWrap(result.HitTestResult), a.Child(), mainAxisPosition, crossAxisPosition)
	}
	return false
}

// ChildMainAxisPosition is minus the scroll offset.
//
// Flutter: rendering/sliver.dart RenderSliverSingleBoxAdapter.childMainAxisPosition
func (a *RenderSliverSingleBoxAdapter) ChildMainAxisPosition(RenderObject) float64 {
	return -a.debugSelf().SliverConstraints().ScrollOffset
}

// ApplyPaintTransform translates by the paint offset of the child.
//
// Flutter: rendering/sliver.dart RenderSliverSingleBoxAdapter.applyPaintTransform
func (a *RenderSliverSingleBoxAdapter) ApplyPaintTransform(child RenderObject, transform *ui.Matrix4) {
	SliverPhysicalParentDataOf(child).ApplyPaintTransform(transform)
}

// Paint paints the child at its paint offset if the sliver is visible.
//
// Flutter: rendering/sliver.dart RenderSliverSingleBoxAdapter.paint
func (a *RenderSliverSingleBoxAdapter) Paint(context *PaintingContext, offset ui.Offset) {
	if a.hasChild() && a.debugSelf().Geometry().Visible {
		context.PaintChild(a.Child(), offset.Add(SliverPhysicalParentDataOf(a.Child()).PaintOffset))
	}
}

// RenderSliverToBoxAdapter is a sliver that contains a single box widget.
//
// Flutter: rendering/sliver.dart RenderSliverToBoxAdapter
type RenderSliverToBoxAdapter struct {
	RenderSliverSingleBoxAdapter
}

// NewRenderSliverToBoxAdapter creates the sliver; the child may be nil.
//
// Flutter: rendering/sliver.dart RenderSliverToBoxAdapter.new
func NewRenderSliverToBoxAdapter(child Box) *RenderSliverToBoxAdapter {
	r := &RenderSliverToBoxAdapter{}
	InitRenderSliverSingleBoxAdapter(&r.RenderSliverSingleBoxAdapter, r, child)
	return r
}

// PerformLayout lays the child out with the cross axis tight and the main axis
// free, and sets the geometry from its extent.
//
// Flutter: rendering/sliver.dart RenderSliverToBoxAdapter.performLayout
func (r *RenderSliverToBoxAdapter) PerformLayout() {
	if !r.hasChild() {
		r.SetGeometry(SliverGeometryZero)
		return
	}
	c := r.SliverConstraints()
	child := r.Child()
	child.Layout(c.AsBoxConstraints(), LayoutOpts{ParentUsesSize: true})
	var childExtent float64
	if c.Axis() == painting.AxisHorizontal {
		childExtent = child.Size().Width
	} else {
		childExtent = child.Size().Height
	}
	paintedChildSize := r.CalculatePaintOffset(c, 0, childExtent)
	cacheExtent := r.CalculateCacheOffset(c, 0, childExtent)
	if foundation.DebugMode && (math.IsInf(paintedChildSize, 0) || math.IsNaN(paintedChildSize) || paintedChildSize < 0) {
		panic(errors.New(tagSliverPaintExtentInvalid, "the painted size of the child is not finite and not negative",
			"object", foundation.DescribeIdentity(r.self), "paintedChildSize", paintedChildSize))
	}
	r.SetGeometry(NewSliverGeometry(SliverGeometryOpts{
		ScrollExtent: childExtent, PaintExtent: paintedChildSize, CacheExtent: foundation.Some(cacheExtent),
		MaxPaintExtent: childExtent, HitTestExtent: foundation.Some(paintedChildSize),
		HasVisualOverflow: childExtent > c.RemainingPaintExtent || c.ScrollOffset > 0,
	}))
	r.SetChildParentData(child, c, r.Geometry())
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagSliverChildPositionMissing        = "SliverChildPositionMissing"
	tagSliverConstraintsInvalid          = "SliverConstraintsInvalid"
	tagSliverGeometryInvalid             = "SliverGeometryInvalid"
	tagSliverGeometrySetterInvalid       = "SliverGeometrySetterInvalid"
	tagSliverGeometryViolatesConstraints = "SliverGeometryViolatesConstraints"
	tagSliverNotLaidOut                  = "SliverNotLaidOut"
	tagSliverPaintExtentInvalid          = "SliverPaintExtentInvalid"
	tagSliverPaintTransformMissing       = "SliverPaintTransformMissing"
	tagSliverPerformResize               = "SliverPerformResize"
	tagSliverRangeInvalid                = "SliverRangeInvalid"
)
