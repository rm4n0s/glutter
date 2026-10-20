// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"fmt"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/internal/dartfmt"
	"github.com/rm4n0s/glutter/rendering"
)

// ViewportNotification is a notification that a viewport counts: each viewport
// it bubbles through adds one to its depth, so that a listener can tell the
// notifications of its own scrollable (depth 0) from the ones of nested
// scrollables.
//
// Flutter: widgets/scroll_notification.dart ViewportNotificationMixin
type ViewportNotification interface {
	// Depth is the number of viewports the notification has bubbled through.
	//
	// Flutter: widgets/scroll_notification.dart ViewportNotificationMixin.depth
	Depth() int
	// IncrementDepth adds one to the depth. A viewport's element calls it
	// (ViewportElementOnNotification).
	//
	// Flutter: widgets/scroll_notification.dart ViewportElementMixin.onNotification
	IncrementDepth()
}

// ViewportNotificationMixin implements ViewportNotification; a notification
// that bubbles through viewports embeds it.
//
// Flutter: widgets/scroll_notification.dart ViewportNotificationMixin
type ViewportNotificationMixin struct {
	depth int
}

// Depth is the number of viewports the notification has bubbled through.
//
// Flutter: widgets/scroll_notification.dart ViewportNotificationMixin.depth
func (m *ViewportNotificationMixin) Depth() int { return m.depth }

// IncrementDepth adds one to the depth.
//
// Flutter: widgets/scroll_notification.dart ViewportElementMixin.onNotification
func (m *ViewportNotificationMixin) IncrementDepth() { m.depth++ }

func (m *ViewportNotificationMixin) describeDepth() string {
	kind := "remote"
	if m.depth == 0 {
		kind = "local"
	}
	return fmt.Sprintf("depth: %d (%s)", m.depth, kind)
}

// ViewportElementOnNotification is what the element of a viewport does with a
// notification that bubbles through it: it counts it if it is a
// ViewportNotification, and returns false so that the notification keeps
// bubbling. An element that implements NotifiableElement calls it from
// OnNotification.
//
// Flutter: widgets/scroll_notification.dart ViewportElementMixin.onNotification
func ViewportElementOnNotification(notification Notification) bool {
	if n, ok := notification.(ViewportNotification); ok {
		n.IncrementDepth()
	}
	return false
}

// ScrollNotification is a notification about a change to a scrollable: the
// scroll started, updated, overscrolled or ended, or the user's scroll
// direction changed.
//
// Flutter: widgets/scroll_notification.dart ScrollNotification
type ScrollNotification interface {
	ViewportNotification
	// Metrics is a snapshot of the scroll metrics when the notification was
	// created.
	//
	// Flutter: widgets/scroll_notification.dart ScrollNotification.metrics
	Metrics() ScrollMetrics
	// Context is the build context of the scrollable that dispatched the
	// notification (the one of its notification context).
	//
	// Flutter: widgets/scroll_notification.dart ScrollNotification.context
	Context() BuildContext
	// IsScrollNotification marks the notifications that are scroll
	// notifications. ScrollMetricsNotification has the same methods as the rest
	// but is not one, as in Flutter, so the interface needs the mark.
	//
	// Flutter: no counterpart
	IsScrollNotification()
}

// ScrollNotificationBase is the part that every ScrollNotification has. The
// concrete notifications embed it.
//
// Deviation: Flutter's ScrollNotification extends LayoutChangedNotification, so
// a NotificationListener<LayoutChangedNotification> receives it; Go has no
// inheritance of types, so it does not (SPEC §12.10).
//
// Flutter: widgets/scroll_notification.dart ScrollNotification
type ScrollNotificationBase struct {
	LayoutChangedNotification
	ViewportNotificationMixin
	metrics ScrollMetrics
	context BuildContext
}

// Metrics is a snapshot of the scroll metrics when the notification was
// created.
//
// Flutter: widgets/scroll_notification.dart ScrollNotification.metrics
func (n *ScrollNotificationBase) Metrics() ScrollMetrics { return n.metrics }

// Context is the build context of the scrollable that dispatched the
// notification.
//
// Flutter: widgets/scroll_notification.dart ScrollNotification.context
func (n *ScrollNotificationBase) Context() BuildContext { return n.context }

// IsScrollNotification marks the notification as a ScrollNotification.
//
// Flutter: no counterpart
func (n *ScrollNotificationBase) IsScrollNotification() {}

func (n *ScrollNotificationBase) describe(extra ...string) string {
	return strings.Join(append([]string{n.describeDepth(), fmt.Sprint(n.metrics)}, extra...), ", ")
}

// ScrollStartNotification indicates that a descendant scrollable has begun
// scrolling.
//
// Flutter: widgets/scroll_notification.dart ScrollStartNotification
type ScrollStartNotification struct {
	ScrollNotificationBase
	// DragDetails is the details of the drag that started the scroll, if the
	// user's drag did.
	//
	// Flutter: widgets/scroll_notification.dart ScrollStartNotification.dragDetails
	DragDetails foundation.Opt[gestures.DragStartDetails]
}

// NewScrollStartNotification creates the notification.
//
// Flutter: widgets/scroll_notification.dart ScrollStartNotification.new
func NewScrollStartNotification(metrics ScrollMetrics, context BuildContext, dragDetails foundation.Opt[gestures.DragStartDetails]) *ScrollStartNotification {
	return &ScrollStartNotification{ScrollNotificationBase{metrics: metrics, context: context}, dragDetails}
}

// String describes the notification.
//
// Flutter: widgets/scroll_notification.dart ScrollStartNotification.debugFillDescription
func (n *ScrollStartNotification) String() string {
	var extra []string
	if d, ok := n.DragDetails.Get(); ok {
		extra = append(extra, fmt.Sprint(d))
	}
	return "ScrollStartNotification(" + n.describe(extra...) + ")"
}

// ScrollUpdateNotification indicates that a descendant scrollable has changed
// its scroll position.
//
// Flutter: widgets/scroll_notification.dart ScrollUpdateNotification
type ScrollUpdateNotification struct {
	ScrollNotificationBase
	// DragDetails is the details of the drag that moved the scrollable, if the
	// user's drag did.
	//
	// Flutter: widgets/scroll_notification.dart ScrollUpdateNotification.dragDetails
	DragDetails foundation.Opt[gestures.DragUpdateDetails]
	// ScrollDelta is the distance by which the scrollable was scrolled, in
	// logical pixels; none for a notification made from a metrics change.
	//
	// Flutter: widgets/scroll_notification.dart ScrollUpdateNotification.scrollDelta
	ScrollDelta foundation.Opt[float64]
}

// ScrollUpdateNotificationOpts are the optional parameters of
// NewScrollUpdateNotification.
//
// Flutter: widgets/scroll_notification.dart ScrollUpdateNotification.new
type ScrollUpdateNotificationOpts struct {
	// DragDetails is the details of the drag that moved the scrollable.
	DragDetails foundation.Opt[gestures.DragUpdateDetails]
	// ScrollDelta is the distance by which the scrollable moved.
	ScrollDelta foundation.Opt[float64]
	// Depth is the initial depth of the notification.
	Depth foundation.Opt[int]
}

// NewScrollUpdateNotification creates the notification.
//
// Flutter: widgets/scroll_notification.dart ScrollUpdateNotification.new
func NewScrollUpdateNotification(metrics ScrollMetrics, context BuildContext, opts ...ScrollUpdateNotificationOpts) *ScrollUpdateNotification {
	o := firstOpts(opts)
	n := &ScrollUpdateNotification{ScrollNotificationBase: ScrollNotificationBase{metrics: metrics, context: context}, DragDetails: o.DragDetails, ScrollDelta: o.ScrollDelta}
	if d, ok := o.Depth.Get(); ok {
		n.depth = d
	}
	return n
}

// String describes the notification.
//
// Flutter: widgets/scroll_notification.dart ScrollUpdateNotification.debugFillDescription
func (n *ScrollUpdateNotification) String() string {
	extra := []string{"scrollDelta: " + describeOptFloat(n.ScrollDelta)}
	if d, ok := n.DragDetails.Get(); ok {
		extra = append(extra, fmt.Sprint(d))
	}
	return "ScrollUpdateNotification(" + n.describe(extra...) + ")"
}

func describeOptFloat(v foundation.Opt[float64]) string {
	if f, ok := v.Get(); ok {
		return dartfmt.Double(f)
	}
	return "null"
}

// OverscrollNotification indicates that a descendant scrollable has not changed
// its scroll position because the change would have caused its scroll position
// to go outside its scroll bounds.
//
// Flutter: widgets/scroll_notification.dart OverscrollNotification
type OverscrollNotification struct {
	ScrollNotificationBase
	// DragDetails is the details of the drag that caused the overscroll, if the
	// user's drag did.
	//
	// Flutter: widgets/scroll_notification.dart OverscrollNotification.dragDetails
	DragDetails foundation.Opt[gestures.DragUpdateDetails]
	// Overscroll is the number of logical pixels that the scrollable did not
	// scroll; it is never zero.
	//
	// Flutter: widgets/scroll_notification.dart OverscrollNotification.overscroll
	Overscroll float64
	// Velocity is the velocity when an overscroll happens in a ballistic
	// scroll, zero otherwise.
	//
	// Flutter: widgets/scroll_notification.dart OverscrollNotification.velocity
	Velocity float64
}

// OverscrollNotificationOpts are the optional parameters of
// NewOverscrollNotification.
//
// Flutter: widgets/scroll_notification.dart OverscrollNotification.new
type OverscrollNotificationOpts struct {
	// DragDetails is the details of the drag that caused the overscroll.
	DragDetails foundation.Opt[gestures.DragUpdateDetails]
	// Velocity is the velocity of the ballistic scroll.
	Velocity float64
}

// NewOverscrollNotification creates the notification. In debug builds it panics
// with the tag OverscrollNotificationInvalid when overscroll is zero or not
// finite.
//
// Flutter: widgets/scroll_notification.dart OverscrollNotification.new
func NewOverscrollNotification(metrics ScrollMetrics, context BuildContext, overscroll float64, opts ...OverscrollNotificationOpts) *OverscrollNotification {
	if foundation.DebugMode && (overscroll == 0 || overscroll != overscroll || overscroll > 1.7976931348623157e308 || overscroll < -1.7976931348623157e308) {
		panic(errors.New("OverscrollNotificationInvalid", "the overscroll of an OverscrollNotification must be finite and not zero",
			"overscroll", overscroll))
	}
	o := firstOpts(opts)
	return &OverscrollNotification{ScrollNotificationBase{metrics: metrics, context: context}, o.DragDetails, overscroll, o.Velocity}
}

// String describes the notification.
//
// Flutter: widgets/scroll_notification.dart OverscrollNotification.debugFillDescription
func (n *OverscrollNotification) String() string {
	extra := []string{"overscroll: " + dartfmt.ToStringAsFixed(n.Overscroll, 1), "velocity: " + dartfmt.ToStringAsFixed(n.Velocity, 1)}
	if d, ok := n.DragDetails.Get(); ok {
		extra = append(extra, fmt.Sprint(d))
	}
	return "OverscrollNotification(" + n.describe(extra...) + ")"
}

// ScrollEndNotification indicates that a descendant scrollable has ended
// scrolling.
//
// Flutter: widgets/scroll_notification.dart ScrollEndNotification
type ScrollEndNotification struct {
	ScrollNotificationBase
	// DragDetails is the details of the drag that ended the scroll, if the
	// user's drag did.
	//
	// Flutter: widgets/scroll_notification.dart ScrollEndNotification.dragDetails
	DragDetails foundation.Opt[gestures.DragEndDetails]
}

// NewScrollEndNotification creates the notification.
//
// Flutter: widgets/scroll_notification.dart ScrollEndNotification.new
func NewScrollEndNotification(metrics ScrollMetrics, context BuildContext, dragDetails foundation.Opt[gestures.DragEndDetails]) *ScrollEndNotification {
	return &ScrollEndNotification{ScrollNotificationBase{metrics: metrics, context: context}, dragDetails}
}

// String describes the notification.
//
// Flutter: widgets/scroll_notification.dart ScrollEndNotification.debugFillDescription
func (n *ScrollEndNotification) String() string {
	var extra []string
	if d, ok := n.DragDetails.Get(); ok {
		extra = append(extra, fmt.Sprint(d))
	}
	return "ScrollEndNotification(" + n.describe(extra...) + ")"
}

// UserScrollNotification indicates that the user has changed the direction in
// which they are scrolling.
//
// Flutter: widgets/scroll_notification.dart UserScrollNotification
type UserScrollNotification struct {
	ScrollNotificationBase
	// Direction is the direction in which the user is scrolling.
	//
	// Flutter: widgets/scroll_notification.dart UserScrollNotification.direction
	Direction rendering.ScrollDirection
}

// NewUserScrollNotification creates the notification.
//
// Flutter: widgets/scroll_notification.dart UserScrollNotification.new
func NewUserScrollNotification(metrics ScrollMetrics, context BuildContext, direction rendering.ScrollDirection) *UserScrollNotification {
	return &UserScrollNotification{ScrollNotificationBase{metrics: metrics, context: context}, direction}
}

// String describes the notification.
//
// Flutter: widgets/scroll_notification.dart UserScrollNotification.debugFillDescription
func (n *UserScrollNotification) String() string {
	return "UserScrollNotification(" + n.describe("direction: "+n.Direction.String()) + ")"
}

// ScrollMetricsNotification indicates that the metrics of a descendant
// scrollable changed without the scroll position changing (for instance when the
// content grows).
//
// Flutter: widgets/scroll_position.dart ScrollMetricsNotification
type ScrollMetricsNotification struct {
	ViewportNotificationMixin
	metrics ScrollMetrics
	context BuildContext
}

// NewScrollMetricsNotification creates the notification.
//
// Flutter: widgets/scroll_position.dart ScrollMetricsNotification.new
func NewScrollMetricsNotification(metrics ScrollMetrics, context BuildContext) *ScrollMetricsNotification {
	return &ScrollMetricsNotification{metrics: metrics, context: context}
}

// Metrics is a snapshot of the scroll metrics when the notification was
// created.
//
// Flutter: widgets/scroll_position.dart ScrollMetricsNotification.metrics
func (n *ScrollMetricsNotification) Metrics() ScrollMetrics { return n.metrics }

// Context is the build context of the scrollable that dispatched the
// notification.
//
// Flutter: widgets/scroll_position.dart ScrollMetricsNotification.context
func (n *ScrollMetricsNotification) Context() BuildContext { return n.context }

// AsScrollUpdate is the notification as a ScrollUpdateNotification with the
// same metrics, context and depth.
//
// Flutter: widgets/scroll_position.dart ScrollMetricsNotification.asScrollUpdate
func (n *ScrollMetricsNotification) AsScrollUpdate() *ScrollUpdateNotification {
	return NewScrollUpdateNotification(n.metrics, n.context, ScrollUpdateNotificationOpts{Depth: foundation.Some(n.depth)})
}

// String describes the notification.
//
// Flutter: widgets/scroll_position.dart ScrollMetricsNotification.debugFillDescription
func (n *ScrollMetricsNotification) String() string {
	return "ScrollMetricsNotification(" + n.describeDepth() + ", " + fmt.Sprint(n.metrics) + ")"
}

// ScrollNotificationPredicate decides whether a scroll notification is handled
// by a widget, for instance only the ones of its own scrollable.
//
// Flutter: widgets/scroll_notification.dart ScrollNotificationPredicate
type ScrollNotificationPredicate func(notification ScrollNotification) bool

// DefaultScrollNotificationPredicate accepts the notifications that were
// dispatched by the closest scrollable (depth 0).
//
// Flutter: widgets/scroll_notification.dart defaultScrollNotificationPredicate
func DefaultScrollNotificationPredicate(notification ScrollNotification) bool {
	return notification.Depth() == 0
}
