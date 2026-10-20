// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets_test

import (
	"math"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/physics"
	. "github.com/rm4n0s/glutter/widgets"
)

// Ported from packages/flutter/test/widgets/scroll_physics_test.dart. The cases
// with PageScrollPhysics and NestedScrollView wait for step 9.3.

// testScrollPhysics is TestScrollPhysics of the Dart test: physics with a name.
type testScrollPhysics struct {
	ScrollPhysicsBase
	name string
}

func newTestScrollPhysics(name string, parent ScrollPhysics) *testScrollPhysics {
	p := &testScrollPhysics{name: name}
	InitScrollPhysicsBase(&p.ScrollPhysicsBase, p, parent)
	return p
}

func (p *testScrollPhysics) ApplyTo(ancestor ScrollPhysics) ScrollPhysics {
	parent := ancestor
	if p.Parent() != nil {
		parent = p.Parent().ApplyTo(ancestor)
	}
	return newTestScrollPhysics(p.name, parent)
}

func (p *testScrollPhysics) names() string {
	if p.Parent() == nil {
		return p.name
	}
	return p.name + " " + p.Parent().(*testScrollPhysics).names()
}

func metricsAt(min, max, pixels, viewport float64) ScrollMetrics {
	return NewFixedScrollMetrics(foundation.Some(min), foundation.Some(max), foundation.Some(pixels), foundation.Some(viewport),
		painting.AxisDirectionDown, 3.0)
}

func moreOrLessEquals(a, b float64) bool {
	return math.Abs(a-b) < 1e-10*math.Max(1, math.Abs(b)) || math.Abs(a-b) < 1e-4
}

func TestScrollPhysicsApplyTo(t *testing.T) {
	a, b, c := newTestScrollPhysics("a", nil), newTestScrollPhysics("b", nil), newTestScrollPhysics("c", nil)
	d, e := newTestScrollPhysics("d", nil), newTestScrollPhysics("e", nil)
	if a.Parent() != nil || b.Parent() != nil || c.Parent() != nil {
		t.Fatal("a fresh physics must have no parent")
	}
	ab := a.ApplyTo(b).(*testScrollPhysics)
	if ab.names() != "a b" {
		t.Errorf("ab = %q", ab.names())
	}
	abc := ab.ApplyTo(c).(*testScrollPhysics)
	if abc.names() != "a b c" {
		t.Errorf("abc = %q", abc.names())
	}
	de := d.ApplyTo(e).(*testScrollPhysics)
	if de.names() != "d e" {
		t.Errorf("de = %q", de.names())
	}
	abcde := abc.ApplyTo(de).(*testScrollPhysics)
	if abcde.names() != "a b c d e" {
		t.Errorf("abcde = %q", abcde.names())
	}
}

func TestScrollPhysicsSubclassesApplyTo(t *testing.T) {
	bounce := ScrollPhysics(NewBouncingScrollPhysics())
	clamp := ScrollPhysics(NewClampingScrollPhysics(nil))
	never := ScrollPhysics(NewNeverScrollableScrollPhysics(nil))
	always := ScrollPhysics(NewAlwaysScrollableScrollPhysics(nil))
	rangeMaintaining := ScrollPhysics(NewRangeMaintainingScrollPhysics(nil))
	types := func(p ScrollPhysics) string {
		out := foundation.RuntimeTypeName(p)
		for p = p.Parent(); p != nil; p = p.Parent() {
			out += " " + foundation.RuntimeTypeName(p)
		}
		return out
	}
	tests := []struct {
		name string
		got  ScrollPhysics
		want string
	}{
		{"bounce first", bounce.ApplyTo(clamp.ApplyTo(never.ApplyTo(always.ApplyTo(rangeMaintaining)))),
			"BouncingScrollPhysics ClampingScrollPhysics NeverScrollableScrollPhysics AlwaysScrollableScrollPhysics RangeMaintainingScrollPhysics"},
		{"clamp first", clamp.ApplyTo(never.ApplyTo(always.ApplyTo(rangeMaintaining.ApplyTo(bounce)))),
			"ClampingScrollPhysics NeverScrollableScrollPhysics AlwaysScrollableScrollPhysics RangeMaintainingScrollPhysics BouncingScrollPhysics"},
		{"never first", never.ApplyTo(always.ApplyTo(rangeMaintaining.ApplyTo(bounce.ApplyTo(clamp)))),
			"NeverScrollableScrollPhysics AlwaysScrollableScrollPhysics RangeMaintainingScrollPhysics BouncingScrollPhysics ClampingScrollPhysics"},
		{"always first", always.ApplyTo(rangeMaintaining.ApplyTo(bounce.ApplyTo(clamp.ApplyTo(never)))),
			"AlwaysScrollableScrollPhysics RangeMaintainingScrollPhysics BouncingScrollPhysics ClampingScrollPhysics NeverScrollableScrollPhysics"},
		{"range maintaining first", rangeMaintaining.ApplyTo(bounce.ApplyTo(clamp.ApplyTo(never.ApplyTo(always)))),
			"RangeMaintainingScrollPhysics BouncingScrollPhysics ClampingScrollPhysics NeverScrollableScrollPhysics AlwaysScrollableScrollPhysics"},
	}
	for _, tt := range tests {
		if got := types(tt.got); got != tt.want {
			t.Errorf("%s: types = %q, want %q", tt.name, got, tt.want)
		}
	}
	desktop := NewBouncingScrollPhysics(BouncingScrollPhysicsOpts{DecelerationRate: ScrollDecelerationRateFast})
	if rate := desktop.ApplyTo(always).(*BouncingScrollPhysics).DecelerationRate(); rate != ScrollDecelerationRateFast {
		t.Errorf("the deceleration rate after applyTo = %v, want fast", rate)
	}
	if got := ScrollPhysics(NewPlainScrollPhysics(nil)).ApplyTo(clamp).String(); got != "ScrollPhysics -> ClampingScrollPhysics" {
		t.Errorf("plain physics String = %q", got)
	}
}

func TestScrollPhysicsBallisticSimulationKeepsVelocityAtTimeZero(t *testing.T) {
	// Calls to createBallisticSimulation may happen on every frame (i.e. when the
	// maxScrollExtent changes). Changing velocity for time 0 may cause a sudden,
	// unwanted damping/speedup effect.
	position := NewFixedScrollMetrics(foundation.Some(0.0), foundation.Some(100.0), foundation.Some(20.0), foundation.Some(500.0),
		painting.AxisDirectionDown, 3.0)
	tests := []struct {
		name    string
		physics ScrollPhysics
	}{
		{"bouncing", NewBouncingScrollPhysics()},
		{"clamping", NewClampingScrollPhysics(nil)},
	}
	for _, tt := range tests {
		sim := tt.physics.CreateBallisticSimulation(position, 1000)
		if sim == nil {
			t.Fatalf("%s: no simulation", tt.name)
		}
		if dx := sim.Dx(0); !moreOrLessEquals(dx, 1000) {
			t.Errorf("%s: dx(0) = %v, want 1000", tt.name, dx)
		}
	}
}

func TestBouncingScrollPhysics(t *testing.T) {
	physicsUnderTest := NewBouncingScrollPhysics()
	t.Run("overscroll is progressively harder", func(t *testing.T) {
		less := physicsUnderTest.ApplyPhysicsToUserOffset(metricsAt(0, 1000, -20, 100), 10)
		more := physicsUnderTest.ApplyPhysicsToUserOffset(metricsAt(0, 1000, -40, 100), 10)
		if !(less > 1 && less < 20) || !(more > 1 && more < 20) {
			t.Errorf("applied offsets %v and %v must be between 1 and 20", less, more)
		}
		// Scrolling from a more overscrolled position meets more resistance.
		if !(math.Abs(less) > math.Abs(more)) {
			t.Errorf("less = %v must be more than more = %v", less, more)
		}
	})
	t.Run("easing an overscroll still has resistance", func(t *testing.T) {
		easing := physicsUnderTest.ApplyPhysicsToUserOffset(metricsAt(0, 1000, -20, 100), -10)
		if !(easing < -1 && easing > -10) {
			t.Errorf("easing = %v, want between -10 and -1", easing)
		}
	})
	t.Run("no resistance when not overscrolled", func(t *testing.T) {
		position := metricsAt(0, 1000, 300, 100)
		if got := physicsUnderTest.ApplyPhysicsToUserOffset(position, 10); got != 10 {
			t.Errorf("forward = %v, want 10", got)
		}
		if got := physicsUnderTest.ApplyPhysicsToUserOffset(position, -10); got != -10 {
			t.Errorf("backward = %v, want -10", got)
		}
	})
	t.Run("easing an overscroll meets less resistance than tensioning", func(t *testing.T) {
		position := metricsAt(0, 1000, -20, 100)
		easing := physicsUnderTest.ApplyPhysicsToUserOffset(position, -10)
		tensioning := physicsUnderTest.ApplyPhysicsToUserOffset(position, 10)
		if !(math.Abs(easing) > math.Abs(tensioning)) {
			t.Errorf("easing %v must be more than tensioning %v", easing, tensioning)
		}
	})
	t.Run("no easing resistance for ScrollDecelerationRate.fast", func(t *testing.T) {
		desktop := NewBouncingScrollPhysics(BouncingScrollPhysicsOpts{DecelerationRate: ScrollDecelerationRateFast})
		position := metricsAt(0, 1000, -20, 100)
		easing := desktop.ApplyPhysicsToUserOffset(position, -10)
		tensioning := desktop.ApplyPhysicsToUserOffset(position, 10)
		if !(math.Abs(tensioning) < math.Abs(easing)) || easing != -10 {
			t.Errorf("easing = %v, tensioning = %v", easing, tensioning)
		}
	})
	t.Run("overscroll a small list and a big list works the same way", func(t *testing.T) {
		small := physicsUnderTest.ApplyPhysicsToUserOffset(metricsAt(0, 10, -20, 100), 10)
		big := physicsUnderTest.ApplyPhysicsToUserOffset(metricsAt(0, 1000, -20, 100), 10)
		if small != big || !(small > 1 && small < 20) {
			t.Errorf("small = %v, big = %v", small, big)
		}
	})
	t.Run("frictionFactor", func(t *testing.T) {
		mobile := NewBouncingScrollPhysics()
		desktop := NewBouncingScrollPhysics(BouncingScrollPhysicsOpts{DecelerationRate: ScrollDecelerationRateFast})
		tests := []struct {
			fraction        float64
			desktop, mobile float64
		}{{0, 0.26, 0.52}, {0.4, 0.0936, 0.1872}, {0.8, 0.0104, 0.0208}}
		for _, tt := range tests {
			if got := desktop.FrictionFactor(tt.fraction); !moreOrLessEquals(got, tt.desktop) {
				t.Errorf("desktop.frictionFactor(%v) = %v, want %v", tt.fraction, got, tt.desktop)
			}
			if got := mobile.FrictionFactor(tt.fraction); !moreOrLessEquals(got, tt.mobile) {
				t.Errorf("mobile.frictionFactor(%v) = %v, want %v", tt.fraction, got, tt.mobile)
			}
		}
	})
}

func TestClampingScrollPhysicsRedundantBoundaryConditions(t *testing.T) {
	skipUnlessDebug(t)
	pixels := 500.0
	position := NewFixedScrollMetrics(foundation.Some(0.0), foundation.Some(1000.0), foundation.Some(pixels), foundation.Some(0.0),
		painting.AxisDirectionDown, 3.0)
	if position.Pixels() != pixels {
		t.Fatalf("pixels = %v", position.Pixels())
	}
	physics := NewClampingScrollPhysics(nil)
	requireTag(t, "ScrollBoundaryRedundant", func() { physics.ApplyBoundaryConditions(position, pixels) })
	err := panicOf(func() { physics.ApplyBoundaryConditions(position, pixels) }).(error)
	appErr, _ := errors.FromError(err)
	if appErr.Metadata["physics"] != "ClampingScrollPhysics" || appErr.Metadata["position"].(ScrollMetrics).Pixels() != pixels {
		t.Errorf("metadata = %v", appErr.Metadata)
	}
}

func TestClampingScrollPhysicsBoundaryConditions(t *testing.T) {
	// The four cases of ClampingScrollPhysics.applyBoundaryConditions.
	physics := NewClampingScrollPhysics(nil)
	tests := []struct {
		name                     string
		min, max, pixels, target float64
		want                     float64
	}{
		{"inside", 0, 100, 50, 60, 0},
		{"underscroll", 0, 100, 0, -10, -10},
		{"overscroll", 0, 100, 100, 110, 10},
		{"hit the top edge", 0, 100, 20, -10, -10},
		{"hit the bottom edge", 0, 100, 80, 110, 10},
	}
	for _, tt := range tests {
		if got := physics.ApplyBoundaryConditions(metricsAt(tt.min, tt.max, tt.pixels, 50), tt.target); got != tt.want {
			t.Errorf("%s: overscroll = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestClampingScrollPhysicsBallistic(t *testing.T) {
	clamping := NewClampingScrollPhysics(nil)
	tests := []struct {
		name       string
		position   ScrollMetrics
		velocity   float64
		wantNil    bool
		wantSpring bool
	}{
		{"fling inside", metricsAt(0, 1000, 500, 100), 1000, false, false},
		{"too slow", metricsAt(0, 1000, 500, 100), 0.1, true, false},
		{"fling past the end", metricsAt(0, 1000, 1000, 100), 1000, true, false},
		{"fling past the start", metricsAt(0, 1000, 0, 100), -1000, true, false},
		{"out of range at the end springs back", metricsAt(0, 1000, 1100, 100), 0, false, true},
		{"out of range at the start springs back", metricsAt(0, 1000, -100, 100), 0, false, true},
	}
	for _, tt := range tests {
		sim := clamping.CreateBallisticSimulation(tt.position, tt.velocity)
		if (sim == nil) != tt.wantNil {
			t.Errorf("%s: simulation = %v, wantNil %v", tt.name, sim, tt.wantNil)
			continue
		}
		if tt.wantSpring {
			if _, ok := sim.(*physics.ScrollSpringSimulation); !ok {
				t.Errorf("%s: simulation is %T, want a scroll spring", tt.name, sim)
			}
		}
	}
}

func TestAlwaysAndNeverScrollablePhysics(t *testing.T) {
	fits := metricsAt(0, 0, 0, 100)
	scrolls := metricsAt(0, 100, 0, 100)
	tests := []struct {
		name                                string
		physics                             ScrollPhysics
		acceptsWhenFits, acceptsWhenScrolls bool
		implicit                            bool
	}{
		{"clamping", NewClampingScrollPhysics(nil), false, true, true},
		{"always", NewAlwaysScrollableScrollPhysics(nil), true, true, true},
		{"never", NewNeverScrollableScrollPhysics(nil), false, false, false},
		{"always over never", NewAlwaysScrollableScrollPhysics(NewNeverScrollableScrollPhysics(nil)), true, true, true},
		{"never over always", NewNeverScrollableScrollPhysics(NewAlwaysScrollableScrollPhysics(nil)), false, false, false},
	}
	for _, tt := range tests {
		if got := tt.physics.ShouldAcceptUserOffset(fits); got != tt.acceptsWhenFits {
			t.Errorf("%s: ShouldAcceptUserOffset(content fits) = %v, want %v", tt.name, got, tt.acceptsWhenFits)
		}
		if got := tt.physics.ShouldAcceptUserOffset(scrolls); got != tt.acceptsWhenScrolls {
			t.Errorf("%s: ShouldAcceptUserOffset(content scrolls) = %v, want %v", tt.name, got, tt.acceptsWhenScrolls)
		}
		if got := tt.physics.AllowImplicitScrolling(); got != tt.implicit {
			t.Errorf("%s: AllowImplicitScrolling = %v, want %v", tt.name, got, tt.implicit)
		}
	}
}

func TestRangeMaintainingScrollPhysics(t *testing.T) {
	physics := NewRangeMaintainingScrollPhysics(nil)
	tests := []struct {
		name         string
		old, new_    ScrollMetrics
		isScrolling  bool
		velocity     float64
		want         float64
		wantDescribe string
	}{
		{"unchanged", metricsAt(0, 100, 50, 100), metricsAt(0, 100, 50, 100), false, 0, 50, "the position stays"},
		{"content shrank below the position", metricsAt(0, 100, 80, 100), metricsAt(0, 50, 80, 100), false, 0, 50, "clamped to the new maximum"},
		{"overscroll at the start is kept when the minimum grows", metricsAt(0, 100, -10, 100), metricsAt(20, 100, -10, 100), false, 0, 10, "keeps the 10 pixel overscroll"},
		{"overscroll at the end is kept when the maximum shrinks", metricsAt(0, 100, 110, 100), metricsAt(0, 60, 110, 100), false, 0, 70, "keeps the 10 pixel overscroll"},
		{"not enforced while moving", metricsAt(0, 100, 50, 100), metricsAt(0, 40, 50, 100), true, 100, 50, "no clamp"},
	}
	for _, tt := range tests {
		got := physics.AdjustPositionForNewDimensions(tt.old, tt.new_, tt.isScrolling, tt.velocity)
		if got != tt.want {
			t.Errorf("%s: position = %v, want %v (%s)", tt.name, got, tt.want, tt.wantDescribe)
		}
	}
}

func TestScrollPhysicsDefaults(t *testing.T) {
	base := NewPlainScrollPhysics(nil)
	metrics := NewFixedScrollMetrics(foundation.Some(0.0), foundation.Some(100.0), foundation.Some(20.0), foundation.Some(500.0),
		painting.AxisDirectionDown, 3.0)
	tolerance := base.ToleranceFor(metrics)
	if !moreOrLessEquals(tolerance.Distance, 1.0/3.0) || !moreOrLessEquals(tolerance.Velocity, 1.0/(0.05*3.0)) {
		t.Errorf("tolerance = %+v", tolerance)
	}
	if base.MinFlingDistance() != 18 || base.MinFlingVelocity() != 50 || base.MaxFlingVelocity() != 8000 {
		t.Errorf("fling constants = %v %v %v", base.MinFlingDistance(), base.MinFlingVelocity(), base.MaxFlingVelocity())
	}
	if base.CarriedMomentum(100) != 0 || base.DragStartDistanceMotionThreshold().IsSome() {
		t.Error("the base physics carry no momentum and have no start threshold")
	}
	if base.ApplyBoundaryConditions(metrics, 200) != 0 || base.ApplyPhysicsToUserOffset(metrics, 3) != 3 {
		t.Error("the base physics apply no boundary and no friction")
	}
	if base.CreateBallisticSimulation(metrics, 1000) != nil {
		t.Error("the base physics have no simulation")
	}
	spring := base.Spring()
	if !moreOrLessEquals(spring.Mass, 0.5) || !moreOrLessEquals(spring.Stiffness, 100) {
		t.Errorf("spring = %v", spring)
	}
	bounce := NewBouncingScrollPhysics()
	if bounce.MinFlingVelocity() != 100 || bounce.DragStartDistanceMotionThreshold().Value() != 3.5 {
		t.Errorf("bouncing constants = %v %v", bounce.MinFlingVelocity(), bounce.DragStartDistanceMotionThreshold())
	}
	if got := bounce.CarriedMomentum(-1000); !(got < 0 && math.Abs(got) < 40000) {
		t.Errorf("carried momentum = %v", got)
	}
	fast := NewBouncingScrollPhysics(BouncingScrollPhysicsOpts{DecelerationRate: ScrollDecelerationRateFast})
	if fast.MaxFlingVelocity() != 8000*8 || fast.Spring().Mass != 0.3 {
		t.Errorf("fast physics: max fling %v, spring %v", fast.MaxFlingVelocity(), fast.Spring())
	}
}

func TestScrollPhysicsTags(t *testing.T) {
	skipUnlessDebug(t)
	t.Run("bouncing physics reject a zero offset", func(t *testing.T) {
		requireTag(t, "ScrollOffsetZero", func() { NewBouncingScrollPhysics().ApplyPhysicsToUserOffset(metricsAt(0, 100, 0, 50), 0) })
	})
	t.Run("metrics extents past each other", func(t *testing.T) {
		requireTag(t, "ScrollMetricsExtentsInvalid", func() { metricsAt(100, 0, 0, 50).ExtentInside() })
		requireTag(t, "ScrollMetricsExtentsInvalid", func() {
			NewBouncingScrollPhysics().ApplyPhysicsToUserOffset(metricsAt(100, 0, 0, 50), 3)
		})
	})
	t.Run("missing metrics values", func(t *testing.T) {
		empty := NewFixedScrollMetrics(foundation.Opt[float64]{}, foundation.Opt[float64]{}, foundation.Opt[float64]{}, foundation.Opt[float64]{},
			painting.AxisDirectionDown, 1)
		for name, read := range map[string]func(){
			"min": func() { empty.MinScrollExtent() }, "max": func() { empty.MaxScrollExtent() },
			"pixels": func() { empty.Pixels() }, "viewport": func() { empty.ViewportDimension() },
		} {
			t.Run(name, func(t *testing.T) { requireTag(t, "ScrollMetricsValueMissing", read) })
		}
		if empty.HasContentDimensions() || empty.HasPixels() || empty.HasViewportDimension() {
			t.Error("the empty metrics have nothing")
		}
	})
}
