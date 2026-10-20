// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package painting

import "github.com/rm4n0s/glutter/ui"

// ClipContext is a mixin that provides the clip-and-paint helpers to a class
// that has a canvas. The helpers are the functions ClipPathAndPaint,
// ClipRRectAndPaint, ClipRSuperellipseAndPaint and ClipRectAndPaint.
//
// Flutter: painting/clip.dart ClipContext
type ClipContext interface {
	// Canvas is the canvas the clipped painting is drawn on.
	Canvas() *ui.Canvas
}

func clipAndPaint(c ClipContext, clipCall func(doAntiAlias bool), clipBehavior ui.Clip, bounds ui.Rect, painter func()) {
	canvas := c.Canvas()
	canvas.Save()
	switch clipBehavior {
	case ui.ClipHardEdge:
		clipCall(false)
	case ui.ClipAntiAlias:
		clipCall(true)
	case ui.ClipAntiAliasWithSaveLayer:
		clipCall(true)
		canvas.SaveLayer(&bounds, ui.NewPaint())
	}
	painter()
	if clipBehavior == ui.ClipAntiAliasWithSaveLayer {
		canvas.Restore()
	}
	canvas.Restore()
}

// ClipPathAndPaint clips the canvas to the path and calls painter, as the
// clip behavior says.
//
// Flutter: painting/clip.dart ClipContext.clipPathAndPaint
func ClipPathAndPaint(c ClipContext, path *ui.Path, clipBehavior ui.Clip, bounds ui.Rect, painter func()) {
	clipAndPaint(c, func(aa bool) { c.Canvas().ClipPath(path, ui.ClipOpts{DoAntiAlias: &aa}) }, clipBehavior, bounds, painter)
}

// ClipRRectAndPaint clips the canvas to the rounded rectangle and calls
// painter, as the clip behavior says.
//
// Flutter: painting/clip.dart ClipContext.clipRRectAndPaint
func ClipRRectAndPaint(c ClipContext, rrect ui.RRect, clipBehavior ui.Clip, bounds ui.Rect, painter func()) {
	clipAndPaint(c, func(aa bool) { c.Canvas().ClipRRect(rrect, ui.ClipOpts{DoAntiAlias: &aa}) }, clipBehavior, bounds, painter)
}

// ClipRSuperellipseAndPaint clips the canvas to the rounded superellipse and
// calls painter, as the clip behavior says.
//
// Flutter: painting/clip.dart ClipContext.clipRSuperellipseAndPaint
func ClipRSuperellipseAndPaint(c ClipContext, rse ui.RSuperellipse, clipBehavior ui.Clip, bounds ui.Rect, painter func()) {
	clipAndPaint(c, func(aa bool) { c.Canvas().ClipRSuperellipse(rse, ui.ClipOpts{DoAntiAlias: &aa}) }, clipBehavior, bounds, painter)
}

// ClipRectAndPaint clips the canvas to the rectangle and calls painter, as the
// clip behavior says.
//
// Flutter: painting/clip.dart ClipContext.clipRectAndPaint
func ClipRectAndPaint(c ClipContext, rect ui.Rect, clipBehavior ui.Clip, bounds ui.Rect, painter func()) {
	clipAndPaint(c, func(aa bool) { c.Canvas().ClipRect(rect, ui.ClipRectOpts{DoAntiAlias: &aa}) }, clipBehavior, bounds, painter)
}
