// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
)

// ScrollContext is what a ScrollPosition asks of the scrollable that owns it: its
// place in the widget tree, a ticker provider for its animations, its axis, and
// a way to tell it that the user can drag it, that pointers are to be ignored
// while it scrolls, and what its semantics can do. ScrollableState implements
// it.
//
// Flutter: widgets/scroll_context.dart ScrollContext
type ScrollContext interface {
	// NotificationContext is the context the scroll notifications are
	// dispatched from, or nil when the scrollable is not built.
	//
	// Flutter: widgets/scroll_context.dart ScrollContext.notificationContext
	NotificationContext() BuildContext
	// StorageContext is the context that identifies the scrollable in a
	// PageStorage, and where the position gets the widgets binding.
	//
	// Flutter: widgets/scroll_context.dart ScrollContext.storageContext
	StorageContext() BuildContext
	// Vsync is the ticker provider the position's animations use.
	//
	// Flutter: widgets/scroll_context.dart ScrollContext.vsync
	Vsync() scheduler.TickerProvider
	// AxisDirection is the direction in which the scroll offset increases.
	//
	// Flutter: widgets/scroll_context.dart ScrollContext.axisDirection
	AxisDirection() painting.AxisDirection
	// DevicePixelRatio is the device pixel ratio of the display the scrollable
	// is on.
	//
	// Flutter: widgets/scroll_context.dart ScrollContext.devicePixelRatio
	DevicePixelRatio() float64
	// SetIgnorePointer makes the scrollable ignore (or stop ignoring) pointer
	// events.
	//
	// Flutter: widgets/scroll_context.dart ScrollContext.setIgnorePointer
	SetIgnorePointer(value bool)
	// SetCanDrag says whether the user can drag the scrollable.
	//
	// Flutter: widgets/scroll_context.dart ScrollContext.setCanDrag
	SetCanDrag(value bool)
	// SetSemanticsActions sets the scroll actions the semantics of the
	// scrollable offers.
	//
	// Flutter: widgets/scroll_context.dart ScrollContext.setSemanticsActions
	SetSemanticsActions(actions map[ui.SemanticsAction]struct{})
	// SaveOffset saves the scroll offset so that it can be restored.
	//
	// Flutter: widgets/scroll_context.dart ScrollContext.saveOffset
	SaveOffset(offset float64)
}
