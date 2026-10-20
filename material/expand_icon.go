// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"math"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// ExpandIconOpts are the optional parameters of NewExpandIcon.
//
// Flutter: material/expand_icon.dart ExpandIcon.new
type ExpandIconOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// IsExpanded is whether the icon points up (expanded) or down.
	IsExpanded bool
	// Size is the size of the icon; the default is 24.
	Size foundation.Opt[float64]
	// Padding is the padding around the icon; the default is 8 on each side.
	Padding painting.EdgeInsetsGeometry
	// Color is the color of the icon.
	Color foundation.Opt[ui.Color]
	// DisabledColor is the color of the icon when it is disabled.
	DisabledColor foundation.Opt[ui.Color]
	// ExpandedColor is the color of the icon when it is expanded.
	ExpandedColor foundation.Opt[ui.Color]
	// SplashColor is the splash color of the button.
	SplashColor foundation.Opt[ui.Color]
	// HighlightColor is the highlight color of the button.
	HighlightColor foundation.Opt[ui.Color]
}

// ExpandIcon is a widget representing a rotating expand/collapse button.
//
// Flutter: material/expand_icon.dart ExpandIcon
type ExpandIcon struct {
	widgets.StatefulWidget
	onPressed func(bool)
	o         ExpandIconOpts
}

// NewExpandIcon creates the icon; a nil onPressed disables it.
//
// Flutter: material/expand_icon.dart ExpandIcon.new
func NewExpandIcon(onPressed func(isExpanded bool), opts ...ExpandIconOpts) *ExpandIcon {
	o := firstOpts(opts)
	if o.Padding == nil {
		o.Padding = painting.NewEdgeInsetsAll(8)
	}
	return &ExpandIcon{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), onPressed: onPressed, o: o}
}

// CreateState creates the state.
//
// Flutter: material/expand_icon.dart ExpandIcon.createState
func (w *ExpandIcon) CreateState() widgets.State { return &expandIconState{} }

type expandIconState struct {
	widgets.StateOf[*ExpandIcon]
	widgets.SingleTickerProvider
	controller *animation.AnimationController
	iconTurns  animation.Animation[float64]
}

func (s *expandIconState) InitState() {
	s.controller = animation.NewAnimationController(s, animation.AnimationControllerOpts{Duration: foundation.Some(ThemeAnimationDuration)})
	s.iconTurns = animation.Drive[float64](animation.Drive[float64](s.controller, animation.NewCurveTween(animation.Curves.FastOutSlowIn)), animation.NewTween(0.0, 0.5))
	// Flutter sets the value to pi, which the controller clamps to 1.
	if s.Widget().o.IsExpanded {
		s.controller.SetValue(math.Pi)
	}
}

func (s *expandIconState) Dispose() { s.controller.Dispose() }

func (s *expandIconState) DidUpdateWidget(old *ExpandIcon) {
	if now := s.Widget().o.IsExpanded; now != old.o.IsExpanded {
		if now {
			s.controller.Forward()
		} else {
			s.controller.Reverse()
		}
	}
}

func (s *expandIconState) iconColor(bc widgets.BuildContext) ui.Color {
	o := s.Widget().o
	if c, ok := o.ExpandedColor.Get(); ok && o.IsExpanded {
		return c
	}
	if c, ok := o.Color.Get(); ok {
		return c
	}
	if ThemeBrightnessOf(bc) == ui.BrightnessLight {
		return Colors.Black54
	}
	return Colors.White60
}

// Build builds the icon button.
//
// Deviation: the semantics' onTapHint is not ported: the semantics widget has no
// tap hint (SPEC §12.10).
//
// Flutter: material/expand_icon.dart _ExpandIconState.build
func (s *expandIconState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	DebugCheckHasMaterial(bc)
	DebugCheckHasMaterialLocalizations(bc)
	var onPressed func()
	if w.onPressed != nil {
		onPressed = func() { w.onPressed(w.o.IsExpanded) }
	}
	return NewIconButton(onPressed, widgets.NewRotationTransition(s.iconTurns, widgets.NewIcon(Icons.ExpandMore())), IconButtonOpts{
		Padding: w.o.Padding, IconSize: foundation.Some(w.o.Size.OrElse(24)), HighlightColor: w.o.HighlightColor,
		SplashColor: w.o.SplashColor, Color: foundation.Some(s.iconColor(bc)), DisabledColor: w.o.DisabledColor,
	})
}
