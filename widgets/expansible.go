// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
)

// ExpansibleComponentBuilder builds the header or the body of an Expansible.
//
// Flutter: widgets/expansible.dart ExpansibleComponentBuilder
type ExpansibleComponentBuilder func(bc BuildContext, animation animation.Animation[float64]) Widget

// ExpansibleBuilder puts the header and the body of an Expansible together.
//
// Flutter: widgets/expansible.dart ExpansibleBuilder
type ExpansibleBuilder func(bc BuildContext, header, body Widget, animation animation.Animation[float64]) Widget

// ExpansibleController controls an Expansible: it expands and collapses it and
// tells whether it is expanded.
//
// Flutter: widgets/expansible.dart ExpansibleController
type ExpansibleController struct {
	foundation.ChangeNotifier
	isExpanded bool
}

// NewExpansibleController creates a controller that is collapsed.
//
// Flutter: widgets/expansible.dart ExpansibleController.new
func NewExpansibleController() *ExpansibleController { return &ExpansibleController{} }

func (c *ExpansibleController) setExpansionState(v bool) {
	if v != c.isExpanded {
		c.isExpanded = v
		c.NotifyListeners()
	}
}

// IsExpanded is whether the widget is expanded.
//
// Flutter: widgets/expansible.dart ExpansibleController.isExpanded
func (c *ExpansibleController) IsExpanded() bool { return c.isExpanded }

// Expand expands the widget; it does nothing when it is expanded already.
//
// Flutter: widgets/expansible.dart ExpansibleController.expand
func (c *ExpansibleController) Expand() { c.setExpansionState(true) }

// Collapse collapses the widget; it does nothing when it is collapsed already.
//
// Flutter: widgets/expansible.dart ExpansibleController.collapse
func (c *ExpansibleController) Collapse() { c.setExpansionState(false) }

// Toggle expands a collapsed widget and collapses an expanded one.
//
// Flutter: widgets/expansible.dart ExpansibleController.toggle
func (c *ExpansibleController) Toggle() {
	if c.isExpanded {
		c.Collapse()
	} else {
		c.Expand()
	}
}

// ExpansibleControllerOf is the controller of the closest Expansible above the
// context. In debug builds it panics with the tag ExpansibleControllerNoAncestor
// when there is none.
//
// Flutter: widgets/expansible.dart ExpansibleController.of
func ExpansibleControllerOf(bc BuildContext) *ExpansibleController {
	c := ExpansibleControllerMaybeOf(bc)
	if c == nil && foundation.DebugMode {
		panic(errors.New(tagExpansibleControllerNoAncestor,
			"ExpansibleControllerOf was called with a context that does not contain an Expansible; use a Builder to get a context that is under the Expansible, or give the Expansible a GlobalKey"))
	}
	return c
}

// ExpansibleControllerMaybeOf is the controller of the closest Expansible above
// the context, or nil.
//
// Flutter: widgets/expansible.dart ExpansibleController.maybeOf
func ExpansibleControllerMaybeOf(bc BuildContext) *ExpansibleController {
	if s, ok := FindAncestorStateOfType[*expansibleState](bc); ok {
		return s.Widget().controller
	}
	return nil
}

// ExpansibleOpts are the optional parameters of NewExpansible.
//
// Flutter: widgets/expansible.dart Expansible.new
type ExpansibleOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// ExpansibleBuilder puts the header and the body together; the default is a
	// column of the two.
	ExpansibleBuilder ExpansibleBuilder
	// AnimationStyle overrides the duration and the curves of the expansion.
	AnimationStyle *animation.AnimationStyle
	// Duration is how long the expansion takes; the default is 200 milliseconds.
	//
	// Flutter: deprecated after v3.38, use AnimationStyle.
	Duration foundation.Opt[time.Duration]
	// Curve is the curve of the expansion; the default is Curves.ease.
	//
	// Flutter: deprecated after v3.38, use AnimationStyle.
	Curve animation.Curve
	// ReverseCurve is the curve of the collapse; the default is the curve.
	//
	// Flutter: deprecated after v3.38, use AnimationStyle.
	ReverseCurve animation.Curve
	// NoMaintainState drops the body from the tree while the widget is collapsed:
	// Flutter's maintainState is false.
	NoMaintainState bool
}

// Expansible is a widget that expands and collapses to show or hide a body under
// a header, without a look of its own: ExpansionTile gives it one.
//
// Flutter: widgets/expansible.dart Expansible
type Expansible struct {
	StatefulWidget
	controller    *ExpansibleController
	headerBuilder ExpansibleComponentBuilder
	bodyBuilder   ExpansibleComponentBuilder
	o             ExpansibleOpts
}

// NewExpansible creates the widget.
//
// Flutter: widgets/expansible.dart Expansible.new
func NewExpansible(controller *ExpansibleController, headerBuilder, bodyBuilder ExpansibleComponentBuilder, opts ...ExpansibleOpts) *Expansible {
	o := firstOpts(opts)
	if o.ExpansibleBuilder == nil {
		o.ExpansibleBuilder = defaultExpansibleBuilder
	}
	if o.Curve == nil {
		o.Curve = animation.Curves.Ease
	}
	return &Expansible{StatefulWidget: NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), controller: controller, headerBuilder: headerBuilder, bodyBuilder: bodyBuilder, o: o}
}

func defaultExpansibleBuilder(_ BuildContext, header, body Widget, _ animation.Animation[float64]) Widget {
	return NewColumn([]Widget{header, body}, ColumnOpts{MainAxisSize: rendering.MainAxisSizeMin})
}

// CreateState creates the state.
//
// Flutter: widgets/expansible.dart Expansible.createState
func (w *Expansible) CreateState() State { return &expansibleState{} }

type expansibleState struct {
	StateOf[*Expansible]
	SingleTickerProvider
	animationController *animation.AnimationController
	heightFactor        *animation.CurvedAnimation
	listener            foundation.ListenerHandle
}

func expansibleDuration(w *Expansible) time.Duration {
	if w.o.AnimationStyle != nil {
		if d, ok := w.o.AnimationStyle.Duration.Get(); ok {
			return d
		}
	}
	return w.o.Duration.OrElse(200 * time.Millisecond)
}

func expansibleCurve(w *Expansible) animation.Curve {
	if w.o.AnimationStyle != nil && w.o.AnimationStyle.Curve != nil {
		return w.o.AnimationStyle.Curve
	}
	return w.o.Curve
}

func expansibleReverseCurve(w *Expansible) animation.Curve {
	if w.o.AnimationStyle != nil && w.o.AnimationStyle.ReverseCurve != nil {
		return w.o.AnimationStyle.ReverseCurve
	}
	return w.o.ReverseCurve
}

// InitState creates the animation and starts it from the stored state.
//
// Flutter: widgets/expansible.dart _ExpansibleState.initState
func (s *expansibleState) InitState() {
	w := s.Widget()
	s.animationController = animation.NewAnimationController(s, animation.AnimationControllerOpts{Duration: foundation.Some(expansibleDuration(w))})
	initiallyExpanded := w.controller.IsExpanded()
	if b := MaybePageStorageOf(s.Context()); b != nil {
		if v, ok := b.ReadState(s.Context()).(bool); ok {
			initiallyExpanded = v
		}
	}
	if initiallyExpanded {
		s.animationController.SetValue(1)
		w.controller.Expand()
	} else {
		w.controller.Collapse()
	}
	s.heightFactor = animation.NewCurvedAnimation(s.animationController, expansibleCurve(w), expansibleReverseCurve(w))
	s.listener = w.controller.AddListener(s.toggleExpansion)
}

// DidUpdateWidget follows the new controller and the new animation settings.
//
// Flutter: widgets/expansible.dart _ExpansibleState.didUpdateWidget
func (s *expansibleState) DidUpdateWidget(old *Expansible) {
	w := s.Widget()
	if c := expansibleCurve(w); c != expansibleCurve(old) {
		s.heightFactor.Curve = c
	}
	if c := expansibleReverseCurve(w); c != expansibleReverseCurve(old) {
		s.heightFactor.ReverseCurve = c
	}
	if d := expansibleDuration(w); d != expansibleDuration(old) {
		s.animationController.Duration = foundation.Some(d)
	}
	if w.controller != old.controller {
		old.controller.RemoveListener(s.listener)
		s.listener = w.controller.AddListener(s.toggleExpansion)
		if old.controller.IsExpanded() != w.controller.IsExpanded() {
			s.toggleExpansion()
		}
	}
}

// Dispose releases the animation.
//
// Flutter: widgets/expansible.dart _ExpansibleState.dispose
func (s *expansibleState) Dispose() {
	s.Widget().controller.RemoveListener(s.listener)
	s.animationController.Dispose()
	s.heightFactor.Dispose()
}

func (s *expansibleState) toggleExpansion() {
	s.SetState(func() {
		if s.Widget().controller.IsExpanded() {
			s.animationController.Forward()
		} else {
			s.animationController.Reverse().Then(func() {
				if !s.Mounted() {
					return
				}
				// Rebuild without the collapsed body.
				s.SetState(func() {})
			})
		}
		if b := MaybePageStorageOf(s.Context()); b != nil {
			b.WriteState(s.Context(), s.Widget().controller.IsExpanded())
		}
	})
}

// Build builds the header and the body that the animation shows.
//
// Flutter: widgets/expansible.dart _ExpansibleState.build
func (s *expansibleState) Build(bc BuildContext) Widget {
	w := s.Widget()
	closed := !w.controller.IsExpanded() && s.animationController.Status() == animation.AnimationStatusDismissed
	shouldRemoveBody := closed && w.o.NoMaintainState
	var result Widget
	if !shouldRemoveBody {
		result = NewOffstage(NewTickerMode(!closed, w.bodyBuilder(bc, s.animationController)), OffstageOpts{Offstage: foundation.Some(closed)})
	}
	return NewAnimatedBuilder(s.animationController, func(bc BuildContext, child Widget) Widget {
		header := w.headerBuilder(bc, s.animationController)
		var body Widget = NewClipRect(NewAlign(child, AlignOpts{HeightFactor: foundation.Some(s.heightFactor.Value())}))
		return w.o.ExpansibleBuilder(bc, header, body, s.animationController)
	}, result)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagExpansibleControllerNoAncestor = "ExpansibleControllerNoAncestor"
)
