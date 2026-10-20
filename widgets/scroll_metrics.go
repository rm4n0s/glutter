// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/dartfmt"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

// ScrollMetrics describes the state of a scrollable: how far it is scrolled,
// how large its viewport and its content are, and in which direction it scrolls.
// A ScrollPosition implements it, and the notifications carry a snapshot of it
// (FixedScrollMetrics).
//
// Flutter: widgets/scroll_metrics.dart ScrollMetrics
type ScrollMetrics interface {
	// CopyWith returns a snapshot of the metrics with the given values changed.
	//
	// Flutter: widgets/scroll_metrics.dart ScrollMetrics.copyWith
	CopyWith(changes ...ScrollMetricsChanges) ScrollMetrics
	// MinScrollExtent is the minimum in-range value for Pixels. It panics with
	// the tag ScrollMetricsValueMissing when HasContentDimensions is false.
	//
	// Flutter: widgets/scroll_metrics.dart ScrollMetrics.minScrollExtent
	MinScrollExtent() float64
	// MaxScrollExtent is the maximum in-range value for Pixels. It panics with
	// the tag ScrollMetricsValueMissing when HasContentDimensions is false.
	//
	// Flutter: widgets/scroll_metrics.dart ScrollMetrics.maxScrollExtent
	MaxScrollExtent() float64
	// HasContentDimensions is whether MinScrollExtent and MaxScrollExtent are
	// known.
	//
	// Flutter: widgets/scroll_metrics.dart ScrollMetrics.hasContentDimensions
	HasContentDimensions() bool
	// Pixels is the current scroll position, in logical pixels along the axis.
	// It panics with the tag ScrollMetricsValueMissing when HasPixels is false.
	//
	// Flutter: widgets/scroll_metrics.dart ScrollMetrics.pixels
	Pixels() float64
	// HasPixels is whether Pixels is known.
	//
	// Flutter: widgets/scroll_metrics.dart ScrollMetrics.hasPixels
	HasPixels() bool
	// ViewportDimension is the extent of the viewport along the axis. It panics
	// with the tag ScrollMetricsValueMissing when HasViewportDimension is false.
	//
	// Flutter: widgets/scroll_metrics.dart ScrollMetrics.viewportDimension
	ViewportDimension() float64
	// HasViewportDimension is whether ViewportDimension is known.
	//
	// Flutter: widgets/scroll_metrics.dart ScrollMetrics.hasViewportDimension
	HasViewportDimension() bool
	// AxisDirection is the direction in which the scroll offset increases.
	//
	// Flutter: widgets/scroll_metrics.dart ScrollMetrics.axisDirection
	AxisDirection() painting.AxisDirection
	// DevicePixelRatio is the ratio of physical pixels to logical pixels of the
	// display the scrollable is on.
	//
	// Flutter: widgets/scroll_metrics.dart ScrollMetrics.devicePixelRatio
	DevicePixelRatio() float64

	// Axis is the axis of AxisDirection.
	//
	// Flutter: widgets/scroll_metrics.dart ScrollMetrics.axis
	Axis() painting.Axis
	// OutOfRange is whether Pixels is outside MinScrollExtent..MaxScrollExtent.
	//
	// Flutter: widgets/scroll_metrics.dart ScrollMetrics.outOfRange
	OutOfRange() bool
	// AtEdge is whether Pixels is at an end of the scroll extent.
	//
	// Flutter: widgets/scroll_metrics.dart ScrollMetrics.atEdge
	AtEdge() bool
	// ExtentBefore is the extent of the content before the viewport.
	//
	// Flutter: widgets/scroll_metrics.dart ScrollMetrics.extentBefore
	ExtentBefore() float64
	// ExtentInside is the extent of the content that is visible in the
	// viewport.
	//
	// Flutter: widgets/scroll_metrics.dart ScrollMetrics.extentInside
	ExtentInside() float64
	// ExtentAfter is the extent of the content after the viewport.
	//
	// Flutter: widgets/scroll_metrics.dart ScrollMetrics.extentAfter
	ExtentAfter() float64
	// ExtentTotal is the extent of all the content, including what is before
	// and after the viewport.
	//
	// Flutter: widgets/scroll_metrics.dart ScrollMetrics.extentTotal
	ExtentTotal() float64
}

// ScrollMetricsChanges are the values that ScrollMetrics.CopyWith changes; the
// ones left out keep the value of the metrics. Only the first value passed to
// CopyWith is used.
//
// Flutter: widgets/scroll_metrics.dart ScrollMetrics.copyWith
type ScrollMetricsChanges struct {
	// MinScrollExtent replaces the minimum scroll extent.
	MinScrollExtent foundation.Opt[float64]
	// MaxScrollExtent replaces the maximum scroll extent.
	MaxScrollExtent foundation.Opt[float64]
	// Pixels replaces the scroll position.
	Pixels foundation.Opt[float64]
	// ViewportDimension replaces the viewport extent.
	ViewportDimension foundation.Opt[float64]
	// AxisDirection replaces the axis direction; AxisDirectionUnspecified keeps
	// it.
	AxisDirection painting.AxisDirection
	// DevicePixelRatio replaces the device pixel ratio; zero keeps it.
	DevicePixelRatio float64
}

// scrollMetricsCopy builds the snapshot that CopyWith returns.
func scrollMetricsCopy(m ScrollMetrics, changes []ScrollMetricsChanges) *FixedScrollMetrics {
	var c ScrollMetricsChanges
	if len(changes) > 0 {
		c = changes[0]
	}
	current := func(has bool, v func() float64) foundation.Opt[float64] {
		if has {
			return foundation.Some(v())
		}
		return foundation.Opt[float64]{}
	}
	min, max := c.MinScrollExtent, c.MaxScrollExtent
	if !min.IsSome() {
		min = current(m.HasContentDimensions(), m.MinScrollExtent)
	}
	if !max.IsSome() {
		max = current(m.HasContentDimensions(), m.MaxScrollExtent)
	}
	pixels := c.Pixels
	if !pixels.IsSome() {
		pixels = current(m.HasPixels(), m.Pixels)
	}
	viewport := c.ViewportDimension
	if !viewport.IsSome() {
		viewport = current(m.HasViewportDimension(), m.ViewportDimension)
	}
	dir := c.AxisDirection
	if dir == painting.AxisDirectionUnspecified {
		dir = m.AxisDirection()
	}
	ratio := c.DevicePixelRatio
	if ratio == 0 {
		ratio = m.DevicePixelRatio()
	}
	return NewFixedScrollMetrics(min, max, pixels, viewport, dir, ratio)
}

// ScrollMetricsMixin gives a type that implements the primitive methods of
// ScrollMetrics the derived ones (Axis, OutOfRange, AtEdge, ExtentBefore,
// ExtentInside, ExtentAfter, ExtentTotal) and CopyWith. Embed it and call
// InitScrollMetricsMixin with the outer value.
//
// Flutter: widgets/scroll_metrics.dart ScrollMetrics
type ScrollMetricsMixin struct {
	self ScrollMetrics
}

// InitScrollMetricsMixin sets the outer value that the mixin's derived methods
// read the primitives from.
//
// Flutter: widgets/scroll_metrics.dart ScrollMetrics
func InitScrollMetricsMixin(m *ScrollMetricsMixin, self ScrollMetrics) { m.self = self }

// CopyWith returns a snapshot of the metrics with the given values changed.
//
// Flutter: widgets/scroll_metrics.dart ScrollMetrics.copyWith
func (m *ScrollMetricsMixin) CopyWith(changes ...ScrollMetricsChanges) ScrollMetrics {
	return scrollMetricsCopy(m.self, changes)
}

// Axis is the axis of AxisDirection.
//
// Flutter: widgets/scroll_metrics.dart ScrollMetrics.axis
func (m *ScrollMetricsMixin) Axis() painting.Axis {
	return painting.AxisDirectionToAxis(m.self.AxisDirection())
}

// OutOfRange is whether Pixels is outside MinScrollExtent..MaxScrollExtent.
//
// Flutter: widgets/scroll_metrics.dart ScrollMetrics.outOfRange
func (m *ScrollMetricsMixin) OutOfRange() bool {
	return m.self.Pixels() < m.self.MinScrollExtent() || m.self.Pixels() > m.self.MaxScrollExtent()
}

// AtEdge is whether Pixels is at an end of the scroll extent.
//
// Flutter: widgets/scroll_metrics.dart ScrollMetrics.atEdge
func (m *ScrollMetricsMixin) AtEdge() bool {
	return m.self.Pixels() == m.self.MinScrollExtent() || m.self.Pixels() == m.self.MaxScrollExtent()
}

// ExtentBefore is the extent of the content before the viewport.
//
// Flutter: widgets/scroll_metrics.dart ScrollMetrics.extentBefore
func (m *ScrollMetricsMixin) ExtentBefore() float64 {
	return math.Max(m.self.Pixels()-m.self.MinScrollExtent(), 0)
}

// ExtentInside is the extent of the content that is visible in the viewport. In
// debug builds it panics with the tag ScrollMetricsExtentsInvalid when the
// minimum scroll extent is past the maximum.
//
// Flutter: widgets/scroll_metrics.dart ScrollMetrics.extentInside
func (m *ScrollMetricsMixin) ExtentInside() float64 {
	min, max := m.self.MinScrollExtent(), m.self.MaxScrollExtent()
	if foundation.DebugMode && min > max {
		panic(errors.New("ScrollMetricsExtentsInvalid", "the minimum scroll extent must not be past the maximum",
			"minScrollExtent", min, "maxScrollExtent", max))
	}
	viewport := m.self.ViewportDimension()
	pixels := m.self.Pixels()
	return viewport -
		// "above" overscroll value
		ui.ClampDouble(min-pixels, 0, viewport) -
		// "below" overscroll value
		ui.ClampDouble(pixels-max, 0, viewport)
}

// ExtentAfter is the extent of the content after the viewport.
//
// Flutter: widgets/scroll_metrics.dart ScrollMetrics.extentAfter
func (m *ScrollMetricsMixin) ExtentAfter() float64 {
	return math.Max(m.self.MaxScrollExtent()-m.self.Pixels(), 0)
}

// ExtentTotal is the extent of all the content, including what is before and
// after the viewport.
//
// Flutter: widgets/scroll_metrics.dart ScrollMetrics.extentTotal
func (m *ScrollMetricsMixin) ExtentTotal() float64 {
	return m.self.MaxScrollExtent() - m.self.MinScrollExtent() + m.self.ViewportDimension()
}

// FixedScrollMetrics is an immutable snapshot of ScrollMetrics. The scroll
// notifications carry one.
//
// Flutter: widgets/scroll_metrics.dart FixedScrollMetrics
type FixedScrollMetrics struct {
	ScrollMetricsMixin
	minScrollExtent, maxScrollExtent foundation.Opt[float64]
	pixels                           foundation.Opt[float64]
	viewportDimension                foundation.Opt[float64]
	axisDirection                    painting.AxisDirection
	devicePixelRatio                 float64
}

// NewFixedScrollMetrics creates a snapshot. A value that is none is missing:
// reading it panics with the tag ScrollMetricsValueMissing.
//
// Flutter: widgets/scroll_metrics.dart FixedScrollMetrics.new
func NewFixedScrollMetrics(minScrollExtent, maxScrollExtent, pixels, viewportDimension foundation.Opt[float64], axisDirection painting.AxisDirection, devicePixelRatio float64) *FixedScrollMetrics {
	m := &FixedScrollMetrics{
		minScrollExtent: minScrollExtent, maxScrollExtent: maxScrollExtent, pixels: pixels,
		viewportDimension: viewportDimension, axisDirection: axisDirection, devicePixelRatio: devicePixelRatio,
	}
	InitScrollMetricsMixin(&m.ScrollMetricsMixin, m)
	return m
}

func scrollMetricsValue(v foundation.Opt[float64], name string) float64 {
	x, ok := v.Get()
	if !ok {
		panic(errors.New("ScrollMetricsValueMissing", "the scroll metrics do not have a value for "+name+" (it is read before the viewport laid out)",
			"value", name))
	}
	return x
}

// MinScrollExtent is the minimum in-range value for Pixels.
//
// Flutter: widgets/scroll_metrics.dart FixedScrollMetrics.minScrollExtent
func (m *FixedScrollMetrics) MinScrollExtent() float64 {
	return scrollMetricsValue(m.minScrollExtent, "minScrollExtent")
}

// MaxScrollExtent is the maximum in-range value for Pixels.
//
// Flutter: widgets/scroll_metrics.dart FixedScrollMetrics.maxScrollExtent
func (m *FixedScrollMetrics) MaxScrollExtent() float64 {
	return scrollMetricsValue(m.maxScrollExtent, "maxScrollExtent")
}

// HasContentDimensions is whether both scroll extents are known.
//
// Flutter: widgets/scroll_metrics.dart FixedScrollMetrics.hasContentDimensions
func (m *FixedScrollMetrics) HasContentDimensions() bool {
	return m.minScrollExtent.IsSome() && m.maxScrollExtent.IsSome()
}

// Pixels is the scroll position.
//
// Flutter: widgets/scroll_metrics.dart FixedScrollMetrics.pixels
func (m *FixedScrollMetrics) Pixels() float64 { return scrollMetricsValue(m.pixels, "pixels") }

// HasPixels is whether the scroll position is known.
//
// Flutter: widgets/scroll_metrics.dart FixedScrollMetrics.hasPixels
func (m *FixedScrollMetrics) HasPixels() bool { return m.pixels.IsSome() }

// ViewportDimension is the extent of the viewport.
//
// Flutter: widgets/scroll_metrics.dart FixedScrollMetrics.viewportDimension
func (m *FixedScrollMetrics) ViewportDimension() float64 {
	return scrollMetricsValue(m.viewportDimension, "viewportDimension")
}

// HasViewportDimension is whether the viewport extent is known.
//
// Flutter: widgets/scroll_metrics.dart FixedScrollMetrics.hasViewportDimension
func (m *FixedScrollMetrics) HasViewportDimension() bool { return m.viewportDimension.IsSome() }

// AxisDirection is the direction in which the scroll offset increases.
//
// Flutter: widgets/scroll_metrics.dart FixedScrollMetrics.axisDirection
func (m *FixedScrollMetrics) AxisDirection() painting.AxisDirection { return m.axisDirection }

// DevicePixelRatio is the device pixel ratio of the display.
//
// Flutter: widgets/scroll_metrics.dart FixedScrollMetrics.devicePixelRatio
func (m *FixedScrollMetrics) DevicePixelRatio() float64 { return m.devicePixelRatio }

// String describes the extents before, inside and after the viewport.
//
// Flutter: widgets/scroll_metrics.dart FixedScrollMetrics.toString
func (m *FixedScrollMetrics) String() string {
	return "FixedScrollMetrics(" + dartfmt.ToStringAsFixed(m.ExtentBefore(), 1) + "..[" + dartfmt.ToStringAsFixed(m.ExtentInside(), 1) + "].." + dartfmt.ToStringAsFixed(m.ExtentAfter(), 1) + ")"
}
