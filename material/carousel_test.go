// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material_test

import (
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// Ported from packages/flutter/test/material/carousel_test.dart.

func carouselItems(n int) []widgets.Widget {
	items := make([]widgets.Widget, n)
	for i := range items {
		items[i] = widgets.NewSizedBox(widgets.NewText("Item "+strconv.Itoa(i)), widgets.SizedBoxOpts{Key: foundation.NewValueKey("item " + strconv.Itoa(i))})
	}
	return items
}

func pumpCarousel(t *testing.T, carousel widgets.Widget) *gluttertest.Tester {
	t.Helper()
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: standardTheme(true), Home: NewScaffold(ScaffoldOpts{
		Body: widgets.NewCenter(widgets.NewSizedBox(carousel, widgets.SizedBoxOpts{Height: foundation.Some(200.0)}))})}))
	ts.PumpAndSettle()
	return ts
}

// itemWidth is the width of the item, with the padding of 4 on each side.
func itemWidth(ts *gluttertest.Tester, i int) float64 {
	return ts.Size(gluttertest.ByKey(foundation.NewValueKey("item "+strconv.Itoa(i)))).Width + 8
}

func TestCarouselFixedExtentLayout(t *testing.T) {
	tests := []struct {
		name       string
		extent     float64
		shrink     float64
		wantWidths map[int]float64
	}{
		// 800 / 300 leaves 200: the last visible item takes it. The padding is part of the item.
		{"fixed", 300, 0, map[int]float64{0: 300, 1: 300, 2: 200}},
		{"fixed bigger than the viewport", 1000, 0, map[int]float64{0: 800}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := pumpCarousel(t, NewCarouselView(tt.extent, carouselItems(10), CarouselViewOpts{ShrinkExtent: tt.shrink}))
			for i, want := range tt.wantWidths {
				if got := itemWidth(ts, i); math.Abs(got-want) > 0.01 {
					t.Errorf("item %d is %v wide, want %v", i, got, want)
				}
			}
		})
	}
}

func TestCarouselFixedExtentShrinksWhileScrolling(t *testing.T) {
	ts := pumpCarousel(t, NewCarouselView(300, carouselItems(10)))
	ts.DragFrom(ui.Offset{Dx: 400, Dy: 300}, ui.Offset{Dx: -100, Dy: 0})
	ts.Pump()
	if got := itemWidth(ts, 0); got >= 300 || got < 150 {
		t.Errorf("the first item is %v wide after the drag, want it to shrink from 300", got)
	}
}

func TestCarouselWeightedLayout(t *testing.T) {
	tests := []struct {
		name    string
		weights []int
		want    []float64
	}{
		// With consumeMaxWeight there is room before item 0 for a shrunk item, so item 0
		// has the largest weight; the rest of the viewport goes to item 1.
		{"hero", []int{1, 7, 1}, []float64{800.0 * 7 / 9, 800.0 / 9}},
		{"multi browse", []int{1, 2, 1}, []float64{400, 200}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := pumpCarousel(t, NewCarouselViewWeighted(tt.weights, carouselItems(10)))
			// With consumeMaxWeight the first items are the leading small ones.
			for i, want := range tt.want {
				got := itemWidth(ts, i)
				if math.Abs(got-want) > 0.01 {
					t.Errorf("item %d is %v wide, want %v", i, got, want)
				}
			}
		})
	}
}

func TestCarouselWeightedFirstItemPosition(t *testing.T) {
	tests := []struct {
		name         string
		weights      []int
		wantLeft     float64
		wantWidth    float64
		noConsumeMax bool
	}{
		{"hero keeps room for a small item", []int{1, 7, 1}, 800.0 / 9, 800.0 * 7 / 9, false},
		{"the first item is the small one", []int{1, 7, 1}, 0, 800.0 / 9, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := pumpCarousel(t, NewCarouselViewWeighted(tt.weights, carouselItems(10), CarouselViewOpts{NoConsumeMaxWeight: tt.noConsumeMax}))
			key := gluttertest.ByKey(foundation.NewValueKey("item 0"))
			size := ts.Size(key)
			left := ts.Center(key).Dx - size.Width/2 - 4
			if math.Abs(left-tt.wantLeft) > 0.1 || math.Abs(size.Width+8-tt.wantWidth) > 0.1 {
				t.Errorf("item 0 is at %v and %v wide, want %v and %v", left, size.Width+8, tt.wantLeft, tt.wantWidth)
			}
		})
	}
}

func TestCarouselControllerAnimateToItem(t *testing.T) {
	c := NewCarouselController()
	var reported []int
	ts := pumpCarousel(t, NewCarouselView(300, carouselItems(10), CarouselViewOpts{Controller: c, OnIndexChanged: func(i int) { reported = append(reported, i) }}))
	if c.LeadingItem() != 0 {
		t.Errorf("leading item %d, want 0", c.LeadingItem())
	}
	c.AnimateToItem(4, AnimateToItemOpts{Duration: foundation.Some(200 * time.Millisecond)})
	ts.PumpAndSettle()
	if c.LeadingItem() != 4 {
		t.Errorf("leading item %d after animateToItem(4), want 4", c.LeadingItem())
	}
	if len(reported) == 0 || reported[len(reported)-1] != 4 {
		t.Errorf("reported %v, want it to end with 4", reported)
	}
	if !ts.Exists(gluttertest.ByText("Item 4")) {
		t.Error("item 4 is not shown")
	}
}

func TestCarouselInitialItem(t *testing.T) {
	c := NewCarouselController(CarouselControllerOpts{InitialItem: 3})
	pumpCarousel(t, NewCarouselView(300, carouselItems(10), CarouselViewOpts{Controller: c}))
	if c.LeadingItem() != 3 {
		t.Errorf("leading item %d, want 3", c.LeadingItem())
	}
}

func TestCarouselItemSnapping(t *testing.T) {
	c := NewCarouselController()
	ts := pumpCarousel(t, NewCarouselView(300, carouselItems(10), CarouselViewOpts{Controller: c, ItemSnapping: true}))
	ts.DragFrom(ui.Offset{Dx: 600, Dy: 300}, ui.Offset{Dx: -200, Dy: 0})
	ts.PumpAndSettle()
	pos := c.Offset()
	if math.Abs(math.Mod(pos, 300)) > 0.01 {
		t.Errorf("the carousel settled at %v, want a multiple of 300", pos)
	}
}

func TestCarouselOnTap(t *testing.T) {
	var tapped []int
	ts := pumpCarousel(t, NewCarouselView(300, carouselItems(10), CarouselViewOpts{OnTap: func(i int) { tapped = append(tapped, i) }}))
	ts.Tap(gluttertest.ByText("Item 1"))
	if len(tapped) != 1 || tapped[0] != 1 {
		t.Errorf("tapped %v, want [1]", tapped)
	}
}

func TestCarouselBuilder(t *testing.T) {
	ts := pumpCarousel(t, NewCarouselViewBuilder(300, func(bc widgets.BuildContext, i int) widgets.Widget {
		return widgets.NewText("Built " + strconv.Itoa(i))
	}, CarouselViewOpts{ItemCount: foundation.Some(5)}))
	if !ts.Exists(gluttertest.ByText("Built 0")) || ts.Exists(gluttertest.ByText("Built 4")) {
		t.Error("the builder items are not the visible ones")
	}
}

func TestCarouselInfiniteWraps(t *testing.T) {
	c := NewCarouselController()
	ts := pumpCarousel(t, NewCarouselViewBuilder(300, func(bc widgets.BuildContext, i int) widgets.Widget {
		return widgets.NewText("Cycle " + strconv.Itoa(i))
	}, CarouselViewOpts{Controller: c, Infinite: true, ItemCount: foundation.Some(3)}))
	if !ts.Exists(gluttertest.ByText("Cycle 0")) {
		t.Error("the infinite carousel does not show item 0")
	}
	c.AnimateToItem(2)
	ts.PumpAndSettle()
	if got := c.LeadingItem(); got != 2 {
		t.Errorf("leading item %d, want 2", got)
	}
}

func TestCarouselRouteTags(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name  string
		build func(ts *gluttertest.Tester)
		route string
	}{
		{"invalid weights", func(ts *gluttertest.Tester) {
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: NewCarouselViewWeighted([]int{1, 0}, carouselItems(3))})}))
		}, "CarouselViewFlexWeightsInvalid"},
		{"controller not attached", func(ts *gluttertest.Tester) {
			NewCarouselController().LeadingItem()
		}, "CarouselControllerNotAttached"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
			err := recoverError(func() { tt.build(ts); takeAndPanic(ts) })
			wantRoute(t, err, tt.route)
		})
	}
}

func itemSpan(ts *gluttertest.Tester, i int) (left, width float64) {
	key := gluttertest.ByKey(foundation.NewValueKey("item " + strconv.Itoa(i)))
	size := ts.Size(key)
	return ts.Center(key).Dx - size.Width/2 - 4, size.Width + 8
}

func TestCarouselWeightedSnappingRespectsConsumeMaxWeight(t *testing.T) {
	ts := pumpCarousel(t, NewCarouselViewWeighted([]int{1, 2, 4, 2, 1}, carouselItems(10), CarouselViewOpts{ItemSnapping: true}))
	if left, width := itemSpan(ts, 0); math.Abs(left-240) > 0.1 || math.Abs(width-320) > 0.1 {
		t.Errorf("item 0 is at %v and %v wide, want 240 and 320", left, width)
	}
	tests := []struct {
		name      string
		drags     int
		wantLeft  float64
		wantWidth float64
	}{
		{"item 9 is the last of the viewport", 7, 720, 80},
		{"item 9 grows to the largest weight", 2, 240, 320},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for range tt.drags {
				ts.DragFrom(ui.Offset{Dx: 400, Dy: 300}, ui.Offset{Dx: -80, Dy: 0})
				ts.PumpAndSettle()
			}
			left, width := itemSpan(ts, 9)
			if math.Abs(left-tt.wantLeft) > 0.1 || math.Abs(width-tt.wantWidth) > 0.1 {
				t.Errorf("item 9 is at %v and %v wide, want %v and %v", left, width, tt.wantLeft, tt.wantWidth)
			}
		})
	}
}
