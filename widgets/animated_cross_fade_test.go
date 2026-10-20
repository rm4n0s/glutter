// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// Ported from packages/flutter/test/widgets/animated_cross_fade_test.dart. (The
// two focus tests are in focus_exclusion_test.go.)

func sizedBox(w, h float64, key ...foundation.Key) Widget {
	o := SizedBoxOpts{Width: foundation.Some(w), Height: foundation.Some(h)}
	if len(key) > 0 {
		o.Key = key[0]
	}
	return NewSizedBox(nil, o)
}

func crossFade(state CrossFadeState, opts ...AnimatedCrossFadeOpts) *AnimatedCrossFade {
	return NewAnimatedCrossFade(sizedBox(100, 100), sizedBox(200, 200), state, 200*time.Millisecond, opts...)
}

func ltr(child Widget) Widget { return NewDirectionality(ui.TextDirectionLTR, child) }

func TestAnimatedCrossFade(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(ltr(NewCenter(crossFade(CrossFadeStateShowFirst))))
	if n := ts.Count(gluttertest.ByType[*FadeTransition]()); n != 2 {
		t.Errorf("fade transitions = %d, want 2", n)
	}
	if size := ts.Size(gluttertest.ByType[*AnimatedCrossFade]()); size != (ui.Size{Width: 100, Height: 100}) {
		t.Errorf("size = %v", size)
	}
	ts.PumpWidget(ltr(NewCenter(crossFade(CrossFadeStateShowSecond))))
	ts.Pump(100 * time.Millisecond)
	if n := ts.Count(gluttertest.ByType[*FadeTransition]()); n != 2 {
		t.Errorf("fade transitions = %d, want 2", n)
	}
	if size := ts.Size(gluttertest.ByType[*AnimatedCrossFade]()); size != (ui.Size{Width: 150, Height: 150}) {
		t.Errorf("size = %v, want 150x150", size)
	}
}

func TestAnimatedCrossFadeShowSecond(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(ltr(NewCenter(crossFade(CrossFadeStateShowSecond))))
	if n := ts.Count(gluttertest.ByType[*FadeTransition]()); n != 2 {
		t.Errorf("fade transitions = %d, want 2", n)
	}
	if size := ts.Size(gluttertest.ByType[*AnimatedCrossFade]()); size != (ui.Size{Width: 200, Height: 200}) {
		t.Errorf("size = %v", size)
	}
}

func TestAnimatedCrossFadeAlignment(t *testing.T) {
	tests := []struct {
		name      string
		direction ui.TextDirection
		alignment painting.AlignmentGeometry
		want      ui.Offset
	}{
		{"VISUAL", ui.TextDirectionLTR, painting.AlignmentBottomRight, ui.Offset{Dx: 275, Dy: 175}},
		{"LTR", ui.TextDirectionLTR, painting.AlignmentDirectionalBottomEnd, ui.Offset{Dx: 275, Dy: 175}},
		{"RTL", ui.TextDirectionRTL, painting.AlignmentDirectionalBottomEnd, ui.Offset{Dx: 325, Dy: 175}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			first, second := foundation.NewUniqueKey(), foundation.NewUniqueKey()
			build := func(state CrossFadeState) Widget {
				return NewDirectionality(tc.direction, NewCenter(NewAnimatedCrossFade(sizedBox(100, 100, first), sizedBox(200, 200, second), state,
					200*time.Millisecond, AnimatedCrossFadeOpts{Alignment: tc.alignment})))
			}
			ts.PumpWidget(build(CrossFadeStateShowFirst))
			ts.PumpWidget(build(CrossFadeStateShowSecond))
			ts.Pump(100 * time.Millisecond)
			if got := ts.TopLeft(gluttertest.ByKey(first)); got != tc.want {
				t.Errorf("first at %v, want %v", got, tc.want)
			}
			if got := ts.TopLeft(gluttertest.ByKey(second)); got != tc.want {
				t.Errorf("second at %v, want %v", got, tc.want)
			}
		})
	}
}

// tickerWatchingWidget is Flutter's _TickerWatchingWidget: it has a running ticker.
type tickerWatchingWidget struct{ StatefulWidget }

func (w *tickerWatchingWidget) CreateState() State { return &tickerWatchingState{} }

type tickerWatchingState struct {
	StateOf[*tickerWatchingWidget]
	SingleTickerProvider
	ticker *scheduler.Ticker
}

func (s *tickerWatchingState) InitState() {
	s.ticker = s.CreateTicker(func(time.Duration) {})
	s.ticker.Start()
}
func (s *tickerWatchingState) Dispose()                  { s.ticker.Dispose() }
func (s *tickerWatchingState) Build(BuildContext) Widget { return NewContainer(nil) }

func crossFadeWithWatcher(towardsSecond bool) Widget {
	state := CrossFadeStateShowFirst
	if towardsSecond {
		state = CrossFadeStateShowSecond
	}
	return ltr(NewAnimatedCrossFade(&tickerWatchingWidget{NewStatefulWidgetBase()}, NewContainer(nil), state, 50*time.Millisecond))
}

func TestAnimatedCrossFadePreservesWidgetState(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(crossFadeWithWatcher(false))
	find := func() *tickerWatchingState {
		return gluttertest.StateOf[*tickerWatchingState](ts, gluttertest.ByType[*tickerWatchingWidget]())
	}
	state := find()
	ts.PumpWidget(crossFadeWithWatcher(true))
	for i := 0; i < 3; i++ {
		ts.Pump(25 * time.Millisecond)
		if find() != state {
			t.Fatalf("the state changed at pump %d", i)
		}
	}
}

func TestAnimatedCrossFadeSwitchesOffTickerModeAndSemanticsOnTheFadedOutWidget(t *testing.T) {
	ts := gluttertest.New(t)
	excluding := func() bool {
		f := gluttertest.Descendant(gluttertest.ByKey(foundation.NewValueKey(CrossFadeStateShowFirst)), gluttertest.ByType[*ExcludeSemantics]())
		return ts.Widget(f).(*ExcludeSemantics).Excluding()
	}
	ts.PumpWidget(crossFadeWithWatcher(false))
	state := gluttertest.StateOf[*tickerWatchingState](ts, gluttertest.ByType[*tickerWatchingWidget]())
	if state.ticker.Muted() || excluding() {
		t.Fatalf("at the start: muted %v, excluding %v", state.ticker.Muted(), excluding())
	}
	ts.PumpWidget(crossFadeWithWatcher(true))
	for i := 0; i < 2; i++ {
		ts.Pump(25 * time.Millisecond)
		if state.ticker.Muted() || !excluding() {
			t.Errorf("pump %d: muted %v, excluding %v, want false, true", i, state.ticker.Muted(), excluding())
		}
	}
	ts.Pump(25 * time.Millisecond)
	if !state.ticker.Muted() || !excluding() {
		t.Errorf("at the end: muted %v, excluding %v, want true, true", state.ticker.Muted(), excluding())
	}
}

func TestAnimatedCrossFadeLayoutBuilder(t *testing.T) {
	ts := gluttertest.New(t)
	text := func(s string) Widget { return NewText(s, TextOpts{TextDirection: ui.TextDirectionLTR}) }
	ts.PumpWidget(ltr(NewAnimatedCrossFade(text("AAA"), text("BBB"), CrossFadeStateShowFirst, 50*time.Millisecond)))
	if ts.Count(gluttertest.ByText("AAA")) != 1 || ts.Count(gluttertest.ByText("BBB")) != 1 {
		t.Error("the default layout does not show both children")
	}
	onlyTop := AnimatedCrossFadeOpts{LayoutBuilder: func(a Widget, _ foundation.Key, _ Widget, _ foundation.Key) Widget { return a }}
	ts.PumpWidget(ltr(NewAnimatedCrossFade(text("AAA"), text("BBB"), CrossFadeStateShowFirst, 50*time.Millisecond, onlyTop)))
	if ts.Count(gluttertest.ByText("AAA")) != 1 || ts.Count(gluttertest.ByText("BBB")) != 0 {
		t.Error("the custom layout does not show only the first child")
	}
	ts.PumpWidget(ltr(NewAnimatedCrossFade(text("AAA"), text("BBB"), CrossFadeStateShowSecond, 50*time.Millisecond, onlyTop)))
	if ts.Count(gluttertest.ByText("BBB")) != 1 || ts.Count(gluttertest.ByText("AAA")) != 0 {
		t.Error("the custom layout does not show only the second child")
	}
}

// tapButton is the TestButton of button_tester.dart: a detector of taps.
func tapButton(onPressed func(), child Widget) Widget {
	return NewGestureDetector(child, GestureDetectorOpts{OnTap: onPressed, Behavior: rendering.HitTestBehaviorOpaque})
}

func TestAnimatedCrossFadeSecondChildDoesNotReceiveTouchEvents(t *testing.T) {
	ts := gluttertest.New(t)
	touches := 0
	build := func(state CrossFadeState) Widget {
		return NewSizedBox(ltr(NewAnimatedCrossFade(NewText("AAA"),
			NewSizedBox(tapButton(func() { touches++ }, NewText("BBB")), SizedBoxOpts{Width: foundation.Some(1e9), Height: foundation.Some(600.0)}),
			state, 50*time.Millisecond)), SizedBoxOpts{Width: foundation.Some(300.0), Height: foundation.Some(600.0)})
	}
	touchSecondButton := func() {
		g := ts.StartGesture(ui.Offset{Dx: 150, Dy: 300})
		g.Up()
	}
	ts.PumpWidget(build(CrossFadeStateShowSecond))
	touchSecondButton()
	if touches != 1 {
		t.Fatalf("touches = %d, want 1", touches)
	}
	ts.PumpWidget(build(CrossFadeStateShowFirst))
	touchSecondButton()
	touchSecondButton()
	if touches != 1 {
		t.Errorf("touches = %d, want 1", touches)
	}
}

func TestAnimatedCrossFadeDoesNotCrashAtZeroArea(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(ltr(NewCenter(NewSizedBoxShrink(NewAnimatedCrossFade(NewText("X"), NewText("Y"), CrossFadeStateShowFirst, time.Second)))))
	if size := ts.Size(gluttertest.ByType[*AnimatedCrossFade]()); size != (ui.Size{}) {
		t.Errorf("size = %v", size)
	}
}

func TestAnimatedCrossFadeOnEnd(t *testing.T) {
	tests := []struct {
		name     string
		from, to CrossFadeState
	}{
		{"forward", CrossFadeStateShowFirst, CrossFadeStateShowSecond},
		{"reverse", CrossFadeStateShowSecond, CrossFadeStateShowFirst},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			calls := 0
			build := func(state CrossFadeState) Widget {
				return ltr(crossFade(state, AnimatedCrossFadeOpts{OnEnd: func() { calls++ }}))
			}
			ts.PumpWidget(build(tc.from))
			if calls != 0 {
				t.Fatalf("calls = %d, want 0", calls)
			}
			ts.PumpWidget(build(tc.to))
			ts.Pump(100 * time.Millisecond)
			if calls != 0 {
				t.Fatalf("calls halfway = %d, want 0", calls)
			}
			ts.PumpAndSettle()
			if calls != 1 {
				t.Errorf("calls = %d, want 1", calls)
			}
		})
	}
	t.Run("each animation cycle", func(t *testing.T) {
		ts := gluttertest.New(t)
		calls := 0
		build := func(state CrossFadeState) Widget {
			return ltr(crossFade(state, AnimatedCrossFadeOpts{OnEnd: func() { calls++ }}))
		}
		ts.PumpWidget(build(CrossFadeStateShowFirst))
		for i, state := range []CrossFadeState{CrossFadeStateShowSecond, CrossFadeStateShowFirst, CrossFadeStateShowSecond} {
			ts.PumpWidget(build(state))
			ts.PumpAndSettle()
			if calls != i+1 {
				t.Errorf("cycle %d: calls = %d, want %d", i, calls, i+1)
			}
		}
	})
}

func TestAnimatedCrossFadeClipBehavior(t *testing.T) {
	tests := []struct {
		name string
		opts AnimatedCrossFadeOpts
		want ui.Clip
	}{
		{"defaults to hardEdge", AnimatedCrossFadeOpts{}, ui.ClipHardEdge},
		{"is forwarded to ClipRect and AnimatedSize", AnimatedCrossFadeOpts{ClipBehavior: ui.ClipNone}, ui.ClipNone},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			ts.PumpWidget(ltr(NewCenter(crossFade(CrossFadeStateShowFirst, tc.opts))))
			if got := ts.Widget(gluttertest.ByType[*ClipRect]()).(*ClipRect).ClipBehavior(); got != tc.want {
				t.Errorf("ClipRect clipBehavior = %v, want %v", got, tc.want)
			}
			if got := ts.Widget(gluttertest.ByType[*AnimatedSize]()).(*AnimatedSize).ClipBehavior(); got != tc.want {
				t.Errorf("AnimatedSize clipBehavior = %v, want %v", got, tc.want)
			}
		})
	}
}
