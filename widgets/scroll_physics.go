package widgets

import (
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/physics"
	"github.com/rm4n0s/glutter/ui"
)

// ScrollDecelerationRate is how quickly a BouncingScrollPhysics decelerates.
//
// Flutter: widgets/scroll_physics.dart ScrollDecelerationRate
type ScrollDecelerationRate int

const (
	// ScrollDecelerationRateUnspecified is the zero value: the physics'
	// default, which is ScrollDecelerationRateNormal.
	//
	// Flutter: no counterpart
	ScrollDecelerationRateUnspecified ScrollDecelerationRate = iota
	// ScrollDecelerationRateNormal is the default rate, used on iOS.
	//
	// Flutter: widgets/scroll_physics.dart ScrollDecelerationRate.normal
	ScrollDecelerationRateNormal
	// ScrollDecelerationRateFast decelerates quicker; macOS uses it.
	//
	// Flutter: widgets/scroll_physics.dart ScrollDecelerationRate.fast
	ScrollDecelerationRateFast
)

// String is the name with the enum, such as "ScrollDecelerationRate.fast".
//
// Flutter: widgets/scroll_physics.dart ScrollDecelerationRate
func (r ScrollDecelerationRate) String() string {
	switch r {
	case ScrollDecelerationRateFast:
		return "ScrollDecelerationRate.fast"
	case ScrollDecelerationRateNormal:
		return "ScrollDecelerationRate.normal"
	}
	return "ScrollDecelerationRate.unspecified"
}

// ScrollPhysics determines the physics of a scrollable: how user gestures move
// it, how it comes to rest, where it stops and how it reacts to a change of its
// dimensions. Physics are immutable and chain: each one has a parent that it
// defers to for what it does not decide.
//
// Implement it by embedding ScrollPhysicsBase (InitScrollPhysicsBase) and
// overriding the methods that differ, and give the type an ApplyTo that builds
// a copy of it on top of its ancestor.
//
// Flutter: widgets/scroll_physics.dart ScrollPhysics
type ScrollPhysics interface {
	// Parent is the physics this one defers to, or nil.
	//
	// Flutter: widgets/scroll_physics.dart ScrollPhysics.parent
	Parent() ScrollPhysics
	// ApplyTo returns a copy of these physics with the ancestor at the end of
	// their chain of parents.
	//
	// Flutter: widgets/scroll_physics.dart ScrollPhysics.applyTo
	ApplyTo(ancestor ScrollPhysics) ScrollPhysics
	// ApplyPhysicsToUserOffset is the number of pixels the scroll position moves
	// for a drag of offset pixels.
	//
	// Flutter: widgets/scroll_physics.dart ScrollPhysics.applyPhysicsToUserOffset
	ApplyPhysicsToUserOffset(position ScrollMetrics, offset float64) float64
	// ShouldAcceptUserOffset is whether the scrollable should let the user
	// scroll it.
	//
	// Flutter: widgets/scroll_physics.dart ScrollPhysics.shouldAcceptUserOffset
	ShouldAcceptUserOffset(position ScrollMetrics) bool
	// RecommendDeferredLoading is whether the scrollable is moving so fast that
	// it is better to not load the images of the content yet.
	//
	// Flutter: widgets/scroll_physics.dart ScrollPhysics.recommendDeferredLoading
	RecommendDeferredLoading(velocity float64, metrics ScrollMetrics, bc BuildContext) bool
	// ApplyBoundaryConditions is the part of the change to value that the
	// position must not apply: the overscroll.
	//
	// Flutter: widgets/scroll_physics.dart ScrollPhysics.applyBoundaryConditions
	ApplyBoundaryConditions(position ScrollMetrics, value float64) float64
	// AdjustPositionForNewDimensions is the scroll position to use after the
	// dimensions of the content or the viewport changed.
	//
	// Flutter: widgets/scroll_physics.dart ScrollPhysics.adjustPositionForNewDimensions
	AdjustPositionForNewDimensions(oldPosition, newPosition ScrollMetrics, isScrolling bool, velocity float64) float64
	// CreateBallisticSimulation is the simulation that carries the scrollable
	// from where it is, at the velocity, to rest; nil if it is to stay put.
	//
	// Flutter: widgets/scroll_physics.dart ScrollPhysics.createBallisticSimulation
	CreateBallisticSimulation(position ScrollMetrics, velocity float64) physics.Simulation
	// Spring is the spring that pulls an overscrolled position back.
	//
	// Flutter: widgets/scroll_physics.dart ScrollPhysics.spring
	Spring() physics.SpringDescription
	// ToleranceFor is the tolerance of the simulations of the position.
	//
	// Flutter: widgets/scroll_physics.dart ScrollPhysics.toleranceFor
	ToleranceFor(metrics ScrollMetrics) physics.Tolerance
	// MinFlingDistance is the minimum distance of a gesture to be a fling.
	//
	// Flutter: widgets/scroll_physics.dart ScrollPhysics.minFlingDistance
	MinFlingDistance() float64
	// MinFlingVelocity is the minimum velocity of a gesture to be a fling.
	//
	// Flutter: widgets/scroll_physics.dart ScrollPhysics.minFlingVelocity
	MinFlingVelocity() float64
	// MaxFlingVelocity is the maximum velocity of a fling.
	//
	// Flutter: widgets/scroll_physics.dart ScrollPhysics.maxFlingVelocity
	MaxFlingVelocity() float64
	// CarriedMomentum is the velocity a new drag gets from the velocity the
	// scrollable had when the finger came down.
	//
	// Flutter: widgets/scroll_physics.dart ScrollPhysics.carriedMomentum
	CarriedMomentum(existingVelocity float64) float64
	// DragStartDistanceMotionThreshold is the distance the pointer has to move
	// before a drag that started while the scrollable was stopped moves it; none
	// for no threshold.
	//
	// Flutter: widgets/scroll_physics.dart ScrollPhysics.dragStartDistanceMotionThreshold
	DragStartDistanceMotionThreshold() foundation.Opt[float64]
	// AllowImplicitScrolling is whether a viewport may build more of its content
	// when the semantics ask for it.
	//
	// Flutter: widgets/scroll_physics.dart ScrollPhysics.allowImplicitScrolling
	AllowImplicitScrolling() bool
	// AllowUserScrolling is whether the user may scroll.
	//
	// Flutter: widgets/scroll_physics.dart ScrollPhysics.allowUserScrolling
	AllowUserScrolling() bool
	// String describes the chain of physics, such as
	// "ClampingScrollPhysics -> RangeMaintainingScrollPhysics".
	//
	// Flutter: widgets/scroll_physics.dart ScrollPhysics.toString
	String() string
}

var defaultScrollSpring = physics.NewSpringDescriptionWithDampingRatio(0.5, 100.0, physics.SpringDescriptionWithDampingRatioOpts{Ratio: foundation.Some(1.1)})

// ScrollPhysicsBase implements ScrollPhysics with the behavior of Flutter's
// ScrollPhysics class: it defers to the parent, or has the default when there is
// none. Embed it and call InitScrollPhysicsBase with the outer value so that the
// methods call the outer type's overrides.
//
// Flutter: widgets/scroll_physics.dart ScrollPhysics
type ScrollPhysicsBase struct {
	self   ScrollPhysics
	parent ScrollPhysics
}

// InitScrollPhysicsBase sets the outer value and the parent (nil for none).
//
// Flutter: widgets/scroll_physics.dart ScrollPhysics.new
func InitScrollPhysicsBase(b *ScrollPhysicsBase, self, parent ScrollPhysics) {
	b.self, b.parent = self, parent
}

// Parent is the physics this one defers to, or nil.
//
// Flutter: widgets/scroll_physics.dart ScrollPhysics.parent
func (b *ScrollPhysicsBase) Parent() ScrollPhysics { return b.parent }

// BuildParent applies the parent of these physics to the ancestor, or returns
// the ancestor if there is no parent. ApplyTo implementations use it to build
// the parent of the copy.
//
// Flutter: widgets/scroll_physics.dart ScrollPhysics.buildParent
func (b *ScrollPhysicsBase) BuildParent(ancestor ScrollPhysics) ScrollPhysics {
	if b.parent != nil {
		return b.parent.ApplyTo(ancestor)
	}
	return ancestor
}

// ApplyTo returns plain ScrollPhysics with the ancestor at the end of the
// chain.
//
// Flutter: widgets/scroll_physics.dart ScrollPhysics.applyTo
func (b *ScrollPhysicsBase) ApplyTo(ancestor ScrollPhysics) ScrollPhysics {
	return NewPlainScrollPhysics(b.BuildParent(ancestor))
}

// ApplyPhysicsToUserOffset defers to the parent; without one the offset is
// used as it is.
//
// Flutter: widgets/scroll_physics.dart ScrollPhysics.applyPhysicsToUserOffset
func (b *ScrollPhysicsBase) ApplyPhysicsToUserOffset(position ScrollMetrics, offset float64) float64 {
	if b.parent != nil {
		return b.parent.ApplyPhysicsToUserOffset(position, offset)
	}
	return offset
}

// ShouldAcceptUserOffset is false if user scrolling is not allowed. Otherwise
// it defers to the parent; without one, it is whether the content can scroll.
//
// Flutter: widgets/scroll_physics.dart ScrollPhysics.shouldAcceptUserOffset
func (b *ScrollPhysicsBase) ShouldAcceptUserOffset(position ScrollMetrics) bool {
	if !b.self.AllowUserScrolling() {
		return false
	}
	if b.parent == nil {
		return position.Pixels() != 0 || position.MinScrollExtent() != position.MaxScrollExtent()
	}
	return b.parent.ShouldAcceptUserOffset(position)
}

// RecommendDeferredLoading defers to the parent; without one it is whether the
// velocity is larger than the longest side of the view in physical pixels per
// second.
//
// Flutter: widgets/scroll_physics.dart ScrollPhysics.recommendDeferredLoading
func (b *ScrollPhysicsBase) RecommendDeferredLoading(velocity float64, metrics ScrollMetrics, bc BuildContext) bool {
	if b.parent == nil {
		maxPhysicalPixels := ViewOf(bc).PhysicalSize().LongestSide()
		return math.Abs(velocity) > maxPhysicalPixels
	}
	return b.parent.RecommendDeferredLoading(velocity, metrics, bc)
}

// ApplyBoundaryConditions defers to the parent; without one there is no
// overscroll.
//
// Flutter: widgets/scroll_physics.dart ScrollPhysics.applyBoundaryConditions
func (b *ScrollPhysicsBase) ApplyBoundaryConditions(position ScrollMetrics, value float64) float64 {
	if b.parent != nil {
		return b.parent.ApplyBoundaryConditions(position, value)
	}
	return 0
}

// AdjustPositionForNewDimensions defers to the parent; without one it keeps the
// new position.
//
// Flutter: widgets/scroll_physics.dart ScrollPhysics.adjustPositionForNewDimensions
func (b *ScrollPhysicsBase) AdjustPositionForNewDimensions(oldPosition, newPosition ScrollMetrics, isScrolling bool, velocity float64) float64 {
	if b.parent == nil {
		return newPosition.Pixels()
	}
	return b.parent.AdjustPositionForNewDimensions(oldPosition, newPosition, isScrolling, velocity)
}

// CreateBallisticSimulation defers to the parent; without one it is nil.
//
// Flutter: widgets/scroll_physics.dart ScrollPhysics.createBallisticSimulation
func (b *ScrollPhysicsBase) CreateBallisticSimulation(position ScrollMetrics, velocity float64) physics.Simulation {
	if b.parent != nil {
		return b.parent.CreateBallisticSimulation(position, velocity)
	}
	return nil
}

// Spring defers to the parent; without one it is a spring with mass 0.5,
// stiffness 100 and damping ratio 1.1.
//
// Flutter: widgets/scroll_physics.dart ScrollPhysics.spring
func (b *ScrollPhysicsBase) Spring() physics.SpringDescription {
	if b.parent != nil {
		return b.parent.Spring()
	}
	return defaultScrollSpring
}

// ToleranceFor defers to the parent; without one it is a distance of one
// physical pixel and a velocity of a physical pixel in 50 ms.
//
// Flutter: widgets/scroll_physics.dart ScrollPhysics.toleranceFor
func (b *ScrollPhysicsBase) ToleranceFor(metrics ScrollMetrics) physics.Tolerance {
	if b.parent != nil {
		return b.parent.ToleranceFor(metrics)
	}
	return physics.NewTolerance(physics.ToleranceOpts{
		Velocity: foundation.Some(1 / (0.050 * metrics.DevicePixelRatio())), // logical pixels per second
		Distance: foundation.Some(1 / metrics.DevicePixelRatio()),           // logical pixels
	})
}

// MinFlingDistance defers to the parent; without one it is the touch slop.
//
// Flutter: widgets/scroll_physics.dart ScrollPhysics.minFlingDistance
func (b *ScrollPhysicsBase) MinFlingDistance() float64 {
	if b.parent != nil {
		return b.parent.MinFlingDistance()
	}
	return gestures.TouchSlop
}

// MinFlingVelocity defers to the parent; without one it is the minimum fling
// velocity of the gestures.
//
// Flutter: widgets/scroll_physics.dart ScrollPhysics.minFlingVelocity
func (b *ScrollPhysicsBase) MinFlingVelocity() float64 {
	if b.parent != nil {
		return b.parent.MinFlingVelocity()
	}
	return gestures.MinFlingVelocity
}

// MaxFlingVelocity defers to the parent; without one it is the maximum fling
// velocity of the gestures.
//
// Flutter: widgets/scroll_physics.dart ScrollPhysics.maxFlingVelocity
func (b *ScrollPhysicsBase) MaxFlingVelocity() float64 {
	if b.parent != nil {
		return b.parent.MaxFlingVelocity()
	}
	return gestures.MaxFlingVelocity
}

// CarriedMomentum defers to the parent; without one it is zero.
//
// Flutter: widgets/scroll_physics.dart ScrollPhysics.carriedMomentum
func (b *ScrollPhysicsBase) CarriedMomentum(existingVelocity float64) float64 {
	if b.parent != nil {
		return b.parent.CarriedMomentum(existingVelocity)
	}
	return 0
}

// DragStartDistanceMotionThreshold defers to the parent; without one it is
// none.
//
// Flutter: widgets/scroll_physics.dart ScrollPhysics.dragStartDistanceMotionThreshold
func (b *ScrollPhysicsBase) DragStartDistanceMotionThreshold() foundation.Opt[float64] {
	if b.parent != nil {
		return b.parent.DragStartDistanceMotionThreshold()
	}
	return foundation.Opt[float64]{}
}

// AllowImplicitScrolling is true.
//
// Flutter: widgets/scroll_physics.dart ScrollPhysics.allowImplicitScrolling
func (b *ScrollPhysicsBase) AllowImplicitScrolling() bool { return true }

// AllowUserScrolling is true.
//
// Flutter: widgets/scroll_physics.dart ScrollPhysics.allowUserScrolling
func (b *ScrollPhysicsBase) AllowUserScrolling() bool { return true }

// String is the name of the type of the physics, then " -> " and the parent if
// there is one.
//
// Flutter: widgets/scroll_physics.dart ScrollPhysics.toString
func (b *ScrollPhysicsBase) String() string {
	name := foundation.RuntimeTypeName(b.self)
	if b.parent == nil {
		return name
	}
	return name + " -> " + b.parent.String()
}

// PlainScrollPhysics is the ScrollPhysics class itself: physics that decide
// nothing and defer to the parent.
//
// Flutter: widgets/scroll_physics.dart ScrollPhysics
type PlainScrollPhysics struct {
	ScrollPhysicsBase
}

// NewPlainScrollPhysics creates physics with the parent (nil for none).
//
// Flutter: widgets/scroll_physics.dart ScrollPhysics.new
func NewPlainScrollPhysics(parent ScrollPhysics) *PlainScrollPhysics {
	p := &PlainScrollPhysics{}
	InitScrollPhysicsBase(&p.ScrollPhysicsBase, p, parent)
	return p
}

// String is "ScrollPhysics", then " -> " and the parent if there is one.
//
// Flutter: widgets/scroll_physics.dart ScrollPhysics.toString
func (p *PlainScrollPhysics) String() string {
	if p.parent == nil {
		return "ScrollPhysics"
	}
	return "ScrollPhysics -> " + p.parent.String()
}

// RangeMaintainingScrollPhysics keeps the scroll position within the range of
// the content when its dimensions change, and keeps an overscroll that was
// there before the change.
//
// Flutter: widgets/scroll_physics.dart RangeMaintainingScrollPhysics
type RangeMaintainingScrollPhysics struct {
	ScrollPhysicsBase
}

// NewRangeMaintainingScrollPhysics creates the physics with the parent (nil for
// none).
//
// Flutter: widgets/scroll_physics.dart RangeMaintainingScrollPhysics.new
func NewRangeMaintainingScrollPhysics(parent ScrollPhysics) *RangeMaintainingScrollPhysics {
	p := &RangeMaintainingScrollPhysics{}
	InitScrollPhysicsBase(&p.ScrollPhysicsBase, p, parent)
	return p
}

// ApplyTo returns a copy on top of the ancestor.
//
// Flutter: widgets/scroll_physics.dart RangeMaintainingScrollPhysics.applyTo
func (p *RangeMaintainingScrollPhysics) ApplyTo(ancestor ScrollPhysics) ScrollPhysics {
	return NewRangeMaintainingScrollPhysics(p.BuildParent(ancestor))
}

// AdjustPositionForNewDimensions keeps the overscroll and the boundary when the
// extents change while the scrollable is at rest.
//
// Flutter: widgets/scroll_physics.dart RangeMaintainingScrollPhysics.adjustPositionForNewDimensions
func (p *RangeMaintainingScrollPhysics) AdjustPositionForNewDimensions(oldPosition, newPosition ScrollMetrics, isScrolling bool, velocity float64) float64 {
	maintainOverscroll := true
	enforceBoundary := true
	if velocity != 0 {
		// Don't try to adjust an animating position, the jumping around would be
		// distracting.
		maintainOverscroll = false
		enforceBoundary = false
	}
	if oldPosition.MinScrollExtent() == newPosition.MinScrollExtent() &&
		oldPosition.MaxScrollExtent() == newPosition.MaxScrollExtent() {
		// If the extents haven't changed then ignore overscroll.
		maintainOverscroll = false
	}
	if oldPosition.Pixels() != newPosition.Pixels() {
		// If the position has been changed already, then it might have been
		// preserved by the framework, or changed by the app.
		maintainOverscroll = false
		if isFinite(oldPosition.MinScrollExtent()) && isFinite(oldPosition.MaxScrollExtent()) &&
			isFinite(newPosition.MinScrollExtent()) && isFinite(newPosition.MaxScrollExtent()) {
			// Without finite extents, the previous position can't be enforced.
			enforceBoundary = false
		}
	}
	if oldPosition.Pixels() < oldPosition.MinScrollExtent() || oldPosition.Pixels() > oldPosition.MaxScrollExtent() {
		// If the old position was out of range, then we should not enforce the
		// boundary.
		enforceBoundary = false
	}
	if maintainOverscroll {
		// Force the new position to be no more out of range than it was before,
		// if:
		//  * it was overscrolled, and
		//  * the extents have decreased, meaning that some content was removed.
		if oldPosition.Pixels() < oldPosition.MinScrollExtent() && newPosition.MinScrollExtent() > oldPosition.MinScrollExtent() {
			oldDelta := oldPosition.MinScrollExtent() - oldPosition.Pixels()
			return newPosition.MinScrollExtent() - oldDelta
		}
		if oldPosition.Pixels() > oldPosition.MaxScrollExtent() && newPosition.MaxScrollExtent() < oldPosition.MaxScrollExtent() {
			oldDelta := oldPosition.Pixels() - oldPosition.MaxScrollExtent()
			return newPosition.MaxScrollExtent() + oldDelta
		}
	}
	result := p.ScrollPhysicsBase.AdjustPositionForNewDimensions(oldPosition, newPosition, isScrolling, velocity)
	if enforceBoundary {
		result = ui.ClampDouble(result, newPosition.MinScrollExtent(), newPosition.MaxScrollExtent())
	}
	return result
}

func isFinite(x float64) bool { return !math.IsInf(x, 0) && !math.IsNaN(x) }

// BouncingScrollPhysics is the scroll physics of iOS and macOS: the content
// goes past its extents with friction, and a spring pulls it back.
//
// Flutter: widgets/scroll_physics.dart BouncingScrollPhysics
type BouncingScrollPhysics struct {
	ScrollPhysicsBase
	decelerationRate ScrollDecelerationRate
}

// BouncingScrollPhysicsOpts are the optional parameters of
// NewBouncingScrollPhysics.
//
// Flutter: widgets/scroll_physics.dart BouncingScrollPhysics.new
type BouncingScrollPhysicsOpts struct {
	// DecelerationRate is how quickly the physics decelerate; the default is
	// ScrollDecelerationRateNormal.
	DecelerationRate ScrollDecelerationRate
	// Parent is the physics to defer to.
	Parent ScrollPhysics
}

// NewBouncingScrollPhysics creates the physics.
//
// Flutter: widgets/scroll_physics.dart BouncingScrollPhysics.new
func NewBouncingScrollPhysics(opts ...BouncingScrollPhysicsOpts) *BouncingScrollPhysics {
	o := firstOpts(opts)
	if o.DecelerationRate == ScrollDecelerationRateUnspecified {
		o.DecelerationRate = ScrollDecelerationRateNormal
	}
	p := &BouncingScrollPhysics{decelerationRate: o.DecelerationRate}
	InitScrollPhysicsBase(&p.ScrollPhysicsBase, p, o.Parent)
	return p
}

// DecelerationRate is how quickly the physics decelerate.
//
// Flutter: widgets/scroll_physics.dart BouncingScrollPhysics.decelerationRate
func (p *BouncingScrollPhysics) DecelerationRate() ScrollDecelerationRate { return p.decelerationRate }

// ApplyTo returns a copy on top of the ancestor.
//
// Flutter: widgets/scroll_physics.dart BouncingScrollPhysics.applyTo
func (p *BouncingScrollPhysics) ApplyTo(ancestor ScrollPhysics) ScrollPhysics {
	return NewBouncingScrollPhysics(BouncingScrollPhysicsOpts{Parent: p.BuildParent(ancestor), DecelerationRate: p.decelerationRate})
}

// FrictionFactor is the fraction of a drag that moves the position while it is
// overscrolled by the fraction of the viewport; the farther it goes, the less
// it moves.
//
// Flutter: widgets/scroll_physics.dart BouncingScrollPhysics.frictionFactor
func (p *BouncingScrollPhysics) FrictionFactor(overscrollFraction float64) float64 {
	rate := 0.52
	if p.decelerationRate == ScrollDecelerationRateFast {
		rate = 0.26
	}
	remaining := 1 - overscrollFraction
	return remaining * remaining * rate
}

// ApplyPhysicsToUserOffset applies the friction of the overscroll to a drag.
//
// Flutter: widgets/scroll_physics.dart BouncingScrollPhysics.applyPhysicsToUserOffset
func (p *BouncingScrollPhysics) ApplyPhysicsToUserOffset(position ScrollMetrics, offset float64) float64 {
	if foundation.DebugMode {
		if offset == 0 {
			panic(errors.New("ScrollOffsetZero", "the offset of a drag handed to the bouncing physics must not be zero"))
		}
		if position.MinScrollExtent() > position.MaxScrollExtent() {
			panic(errors.New("ScrollMetricsExtentsInvalid", "the minimum scroll extent must not be past the maximum",
				"minScrollExtent", position.MinScrollExtent(), "maxScrollExtent", position.MaxScrollExtent()))
		}
	}
	if !position.OutOfRange() {
		return offset
	}
	overscrollPastStart := math.Max(position.MinScrollExtent()-position.Pixels(), 0)
	overscrollPastEnd := math.Max(position.Pixels()-position.MaxScrollExtent(), 0)
	overscrollPast := math.Max(overscrollPastStart, overscrollPastEnd)
	easing := (overscrollPastStart > 0 && offset < 0) || (overscrollPastEnd > 0 && offset > 0)
	var friction float64
	if easing {
		// Apply less resistance when easing the overscroll vs tensioning.
		friction = p.FrictionFactor((overscrollPast - math.Abs(offset)) / position.ViewportDimension())
	} else {
		// Apply more resistance when tensioning the overscroll.
		friction = p.FrictionFactor(overscrollPast / position.ViewportDimension())
	}
	direction := math.Copysign(1, offset)
	if offset == 0 {
		direction = 0
	}
	if easing && p.decelerationRate == ScrollDecelerationRateFast {
		return direction * math.Abs(offset)
	}
	return direction * applyBouncingFriction(overscrollPast, math.Abs(offset), friction)
}

func applyBouncingFriction(extentOutside, absDelta, gamma float64) float64 {
	total := 0.0
	if extentOutside > 0 {
		deltaToLimit := extentOutside / gamma
		if absDelta < deltaToLimit {
			return absDelta * gamma
		}
		total += extentOutside
		absDelta -= deltaToLimit
	}
	return total + absDelta
}

// ApplyBoundaryConditions has no boundary: the content can go past its
// extents.
//
// Flutter: widgets/scroll_physics.dart BouncingScrollPhysics.applyBoundaryConditions
func (p *BouncingScrollPhysics) ApplyBoundaryConditions(ScrollMetrics, float64) float64 { return 0 }

// CreateBallisticSimulation is a BouncingScrollSimulation when the position is
// out of range or the velocity is above the tolerance; otherwise nil.
//
// Flutter: widgets/scroll_physics.dart BouncingScrollPhysics.createBallisticSimulation
func (p *BouncingScrollPhysics) CreateBallisticSimulation(position ScrollMetrics, velocity float64) physics.Simulation {
	tolerance := p.ToleranceFor(position)
	if math.Abs(velocity) >= tolerance.Velocity || position.OutOfRange() {
		constantDeceleration := 0.0
		if p.decelerationRate == ScrollDecelerationRateFast {
			constantDeceleration = 1400
		}
		return physics.NewBouncingScrollSimulation(position.Pixels(), velocity, position.MinScrollExtent(), position.MaxScrollExtent(),
			p.Spring(), physics.BouncingScrollSimulationOpts{Tolerance: foundation.Some(tolerance), ConstantDeceleration: constantDeceleration})
	}
	return nil
}

// MinFlingVelocity is twice that of the other physics, so that a slow drag does
// not fling.
//
// Flutter: widgets/scroll_physics.dart BouncingScrollPhysics.minFlingVelocity
func (p *BouncingScrollPhysics) MinFlingVelocity() float64 { return gestures.MinFlingVelocity * 2 }

// CarriedMomentum is the momentum of the fling that a new touch carries over;
// the formula is empirical.
//
// Flutter: widgets/scroll_physics.dart BouncingScrollPhysics.carriedMomentum
func (p *BouncingScrollPhysics) CarriedMomentum(existingVelocity float64) float64 {
	sign := math.Copysign(1, existingVelocity)
	if existingVelocity == 0 {
		sign = 0
	}
	return sign * math.Min(0.000816*math.Pow(math.Abs(existingVelocity), 1.967), 40000.0)
}

// DragStartDistanceMotionThreshold is 3.5 logical pixels, so that a finger that
// stops a fling does not move the content.
//
// Flutter: widgets/scroll_physics.dart BouncingScrollPhysics.dragStartDistanceMotionThreshold
func (p *BouncingScrollPhysics) DragStartDistanceMotionThreshold() foundation.Opt[float64] {
	return foundation.Some(3.5)
}

// MaxFlingVelocity is eight times the usual for the fast deceleration rate.
//
// Flutter: widgets/scroll_physics.dart BouncingScrollPhysics.maxFlingVelocity
func (p *BouncingScrollPhysics) MaxFlingVelocity() float64 {
	if p.decelerationRate == ScrollDecelerationRateFast {
		return gestures.MaxFlingVelocity * 8
	}
	return p.ScrollPhysicsBase.MaxFlingVelocity()
}

// Spring is a stiffer spring for the fast deceleration rate.
//
// Flutter: widgets/scroll_physics.dart BouncingScrollPhysics.spring
func (p *BouncingScrollPhysics) Spring() physics.SpringDescription {
	if p.decelerationRate == ScrollDecelerationRateFast {
		return physics.NewSpringDescriptionWithDampingRatio(0.3, 75.0, physics.SpringDescriptionWithDampingRatioOpts{Ratio: foundation.Some(1.3)})
	}
	return p.ScrollPhysicsBase.Spring()
}

// ClampingScrollPhysics is the scroll physics of Android, Linux and Windows: the
// content stops at its extents.
//
// Flutter: widgets/scroll_physics.dart ClampingScrollPhysics
type ClampingScrollPhysics struct {
	ScrollPhysicsBase
}

// NewClampingScrollPhysics creates the physics with the parent (nil for none).
//
// Flutter: widgets/scroll_physics.dart ClampingScrollPhysics.new
func NewClampingScrollPhysics(parent ScrollPhysics) *ClampingScrollPhysics {
	p := &ClampingScrollPhysics{}
	InitScrollPhysicsBase(&p.ScrollPhysicsBase, p, parent)
	return p
}

// ApplyTo returns a copy on top of the ancestor.
//
// Flutter: widgets/scroll_physics.dart ClampingScrollPhysics.applyTo
func (p *ClampingScrollPhysics) ApplyTo(ancestor ScrollPhysics) ScrollPhysics {
	return NewClampingScrollPhysics(p.BuildParent(ancestor))
}

// ApplyBoundaryConditions is the part of the change that would take the position
// past its extents. In debug builds it panics with the tag
// ScrollBoundaryRedundant when value is the current position.
//
// Flutter: widgets/scroll_physics.dart ClampingScrollPhysics.applyBoundaryConditions
func (p *ClampingScrollPhysics) ApplyBoundaryConditions(position ScrollMetrics, value float64) float64 {
	if foundation.DebugMode && value == position.Pixels() {
		panic(errors.New("ScrollBoundaryRedundant",
			"ClampingScrollPhysics.applyBoundaryConditions() was called redundantly; the proposed new position is exactly equal to the current position of the given scroll metrics; the method should only be called when the value is going to actually change the pixels",
			"value", value, "physics", p.String(), "position", position))
	}
	if value < position.Pixels() && position.Pixels() <= position.MinScrollExtent() {
		// Underscroll.
		return value - position.Pixels()
	}
	if position.MaxScrollExtent() <= position.Pixels() && position.Pixels() < value {
		// Overscroll.
		return value - position.Pixels()
	}
	if value < position.MinScrollExtent() && position.MinScrollExtent() < position.Pixels() {
		// Hit top edge.
		return value - position.MinScrollExtent()
	}
	if position.Pixels() < position.MaxScrollExtent() && position.MaxScrollExtent() < value {
		// Hit bottom edge.
		return value - position.MaxScrollExtent()
	}
	return 0
}

// CreateBallisticSimulation is a spring back into range when the position is
// out of range, a ClampingScrollSimulation for a fling inside the range, and nil
// when the velocity is too small or points away from the content.
//
// Flutter: widgets/scroll_physics.dart ClampingScrollPhysics.createBallisticSimulation
func (p *ClampingScrollPhysics) CreateBallisticSimulation(position ScrollMetrics, velocity float64) physics.Simulation {
	tolerance := p.ToleranceFor(position)
	if position.OutOfRange() {
		end := math.NaN()
		if position.Pixels() > position.MaxScrollExtent() {
			end = position.MaxScrollExtent()
		}
		if position.Pixels() < position.MinScrollExtent() {
			end = position.MinScrollExtent()
		}
		return physics.NewScrollSpringSimulation(p.Spring(), position.Pixels(), end, math.Min(0, velocity),
			physics.SpringSimulationOpts{Tolerance: foundation.Some(tolerance)})
	}
	if math.Abs(velocity) < tolerance.Velocity {
		return nil
	}
	if velocity > 0 && position.Pixels() >= position.MaxScrollExtent() {
		return nil
	}
	if velocity < 0 && position.Pixels() <= position.MinScrollExtent() {
		return nil
	}
	return physics.NewClampingScrollSimulation(position.Pixels(), velocity, physics.ClampingScrollSimulationOpts{Tolerance: foundation.Some(tolerance)})
}

// AlwaysScrollableScrollPhysics accepts user scrolling even when the content is
// smaller than the viewport.
//
// Flutter: widgets/scroll_physics.dart AlwaysScrollableScrollPhysics
type AlwaysScrollableScrollPhysics struct {
	ScrollPhysicsBase
}

// NewAlwaysScrollableScrollPhysics creates the physics with the parent (nil for
// none).
//
// Flutter: widgets/scroll_physics.dart AlwaysScrollableScrollPhysics.new
func NewAlwaysScrollableScrollPhysics(parent ScrollPhysics) *AlwaysScrollableScrollPhysics {
	p := &AlwaysScrollableScrollPhysics{}
	InitScrollPhysicsBase(&p.ScrollPhysicsBase, p, parent)
	return p
}

// ApplyTo returns a copy on top of the ancestor.
//
// Flutter: widgets/scroll_physics.dart AlwaysScrollableScrollPhysics.applyTo
func (p *AlwaysScrollableScrollPhysics) ApplyTo(ancestor ScrollPhysics) ScrollPhysics {
	return NewAlwaysScrollableScrollPhysics(p.BuildParent(ancestor))
}

// ShouldAcceptUserOffset is always true.
//
// Flutter: widgets/scroll_physics.dart AlwaysScrollableScrollPhysics.shouldAcceptUserOffset
func (p *AlwaysScrollableScrollPhysics) ShouldAcceptUserOffset(ScrollMetrics) bool { return true }

// NeverScrollableScrollPhysics does not let the user scroll; the scrollable can
// still be moved by a controller.
//
// Flutter: widgets/scroll_physics.dart NeverScrollableScrollPhysics
type NeverScrollableScrollPhysics struct {
	ScrollPhysicsBase
}

// NewNeverScrollableScrollPhysics creates the physics with the parent (nil for
// none).
//
// Flutter: widgets/scroll_physics.dart NeverScrollableScrollPhysics.new
func NewNeverScrollableScrollPhysics(parent ScrollPhysics) *NeverScrollableScrollPhysics {
	p := &NeverScrollableScrollPhysics{}
	InitScrollPhysicsBase(&p.ScrollPhysicsBase, p, parent)
	return p
}

// ApplyTo returns a copy on top of the ancestor.
//
// Flutter: widgets/scroll_physics.dart NeverScrollableScrollPhysics.applyTo
func (p *NeverScrollableScrollPhysics) ApplyTo(ancestor ScrollPhysics) ScrollPhysics {
	return NewNeverScrollableScrollPhysics(p.BuildParent(ancestor))
}

// AllowUserScrolling is false.
//
// Flutter: widgets/scroll_physics.dart NeverScrollableScrollPhysics.allowUserScrolling
func (p *NeverScrollableScrollPhysics) AllowUserScrolling() bool { return false }

// AllowImplicitScrolling is false.
//
// Flutter: widgets/scroll_physics.dart NeverScrollableScrollPhysics.allowImplicitScrolling
func (p *NeverScrollableScrollPhysics) AllowImplicitScrolling() bool { return false }
