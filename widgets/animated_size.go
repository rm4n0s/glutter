// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
)

// AnimatedSizeOpts are the optional parameters of NewAnimatedSize.
//
// Flutter: widgets/animated_size.dart AnimatedSize.new
type AnimatedSizeOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Alignment is how the child is aligned while the size animates; the
	// default is the center.
	Alignment painting.AlignmentGeometry
	// Curve is the curve of the animation; the default is Curves.Linear.
	Curve animation.Curve
	// ReverseDuration is the duration when the size shrinks; the default is the
	// duration.
	ReverseDuration foundation.Opt[time.Duration]
	// ClipBehavior is how the child is clipped while it overflows; the default is
	// ClipHardEdge.
	ClipBehavior ui.Clip
	// OnEnd is called every time the animation completes.
	OnEnd func()
}

// AnimatedSize animates its size to match the size of its child.
//
// Flutter: widgets/animated_size.dart AnimatedSize
type AnimatedSize struct {
	StatefulWidget
	child           Widget
	alignment       painting.AlignmentGeometry
	curve           animation.Curve
	duration        time.Duration
	reverseDuration foundation.Opt[time.Duration]
	clipBehavior    ui.Clip
	onEnd           func()
}

// NewAnimatedSize creates the widget; the child may be nil.
//
// Flutter: widgets/animated_size.dart AnimatedSize.new
func NewAnimatedSize(duration time.Duration, child Widget, opts ...AnimatedSizeOpts) *AnimatedSize {
	o := firstOpts(opts)
	if o.Alignment == nil {
		o.Alignment = painting.AlignmentCenter
	}
	if o.Curve == nil {
		o.Curve = animation.Curves.Linear
	}
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipHardEdge
	}
	return &AnimatedSize{NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), child, o.Alignment, o.Curve, duration, o.ReverseDuration, o.ClipBehavior, o.OnEnd}
}

// Child is the widget below this one in the tree.
//
// Flutter: widgets/animated_size.dart AnimatedSize.child
func (w *AnimatedSize) Child() Widget { return w.child }

// Alignment is how the child is aligned while the size animates.
//
// Flutter: widgets/animated_size.dart AnimatedSize.alignment
func (w *AnimatedSize) Alignment() painting.AlignmentGeometry { return w.alignment }

// Curve is the curve of the animation.
//
// Flutter: widgets/animated_size.dart AnimatedSize.curve
func (w *AnimatedSize) Curve() animation.Curve { return w.curve }

// Duration is the duration of the animation.
//
// Flutter: widgets/animated_size.dart AnimatedSize.duration
func (w *AnimatedSize) Duration() time.Duration { return w.duration }

// ReverseDuration is the duration when the size shrinks.
//
// Flutter: widgets/animated_size.dart AnimatedSize.reverseDuration
func (w *AnimatedSize) ReverseDuration() foundation.Opt[time.Duration] { return w.reverseDuration }

// ClipBehavior is how the child is clipped while it overflows.
//
// Flutter: widgets/animated_size.dart AnimatedSize.clipBehavior
func (w *AnimatedSize) ClipBehavior() ui.Clip { return w.clipBehavior }

// OnEnd is called every time the animation completes, or nil.
//
// Flutter: widgets/animated_size.dart AnimatedSize.onEnd
func (w *AnimatedSize) OnEnd() func() { return w.onEnd }

// CreateState creates the state.
//
// Flutter: widgets/animated_size.dart AnimatedSize.createState
func (w *AnimatedSize) CreateState() State { return &animatedSizeState{} }

type animatedSizeState struct {
	StateOf[*AnimatedSize]
	SingleTickerProvider
}

func (s *animatedSizeState) Build(BuildContext) Widget {
	w := s.Widget()
	return &animatedSizeRender{NewSingleChildRenderObjectWidgetBase(w.child), w.alignment, w.curve, w.duration, w.reverseDuration, s, w.clipBehavior, w.onEnd}
}

// animatedSizeRender is Flutter's _AnimatedSize.
type animatedSizeRender struct {
	SingleChildRenderObjectWidget
	alignment       painting.AlignmentGeometry
	curve           animation.Curve
	duration        time.Duration
	reverseDuration foundation.Opt[time.Duration]
	vsync           scheduler.TickerProvider
	clipBehavior    ui.Clip
	onEnd           func()
}

func (w *animatedSizeRender) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderAnimatedSize(w.vsync, w.duration, nil, rendering.RenderAnimatedSizeOpts{
		ReverseDuration: w.reverseDuration, Curve: w.curve, Alignment: w.alignment, TextDirection: maybeTextDirection(bc),
		ClipBehavior: w.clipBehavior, OnEnd: w.onEnd})
}

func (w *animatedSizeRender) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderAnimatedSize)
	r.SetAlignment(w.alignment)
	r.SetDuration(w.duration)
	r.SetReverseDuration(w.reverseDuration)
	r.SetCurve(w.curve)
	r.SetVsync(w.vsync)
	r.SetTextDirection(maybeTextDirection(bc))
	r.SetClipBehavior(w.clipBehavior)
	r.SetOnEnd(w.onEnd)
}
