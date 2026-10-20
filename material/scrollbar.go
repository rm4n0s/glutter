// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

const (
	scrollbarThickness          = 8.0
	scrollbarThicknessWithTrack = 12.0
	scrollbarMargin             = 2.0
	scrollbarMinLength          = 48.0
	scrollbarFadeDuration       = 300 * time.Millisecond
	scrollbarTimeToFade         = 600 * time.Millisecond
)

// ScrollbarOpts are the optional parameters of NewScrollbar.
//
// Flutter: material/scrollbar.dart Scrollbar
type ScrollbarOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Controller is the scroll controller of the scrollable the scrollbar is for.
	Controller widgets.ScrollController
	// ThumbVisibility is whether the thumb is always visible.
	ThumbVisibility foundation.Opt[bool]
	// TrackVisibility is whether the track is visible when the thumb is.
	TrackVisibility foundation.Opt[bool]
	// Thickness is the thickness of the thumb.
	Thickness foundation.Opt[float64]
	// Radius is the radius of the corners of the thumb.
	Radius foundation.Opt[ui.Radius]
	// NotificationPredicate says which scroll notifications update the scrollbar;
	// the default is DefaultScrollNotificationPredicate.
	NotificationPredicate widgets.ScrollNotificationPredicate
	// Interactive is whether the thumb can be dragged.
	Interactive foundation.Opt[bool]
	// ScrollbarOrientation is where the scrollbar sits.
	ScrollbarOrientation widgets.ScrollbarOrientation
}

// Scrollbar is a Material Design scrollbar: a thumb that shows where the
// content is, which fades out when the scrolling stops, grows and darkens under
// the mouse and can be dragged.
//
// Flutter: material/scrollbar.dart Scrollbar
type Scrollbar struct {
	widgets.StatelessWidget
	child widgets.Widget
	o     ScrollbarOpts
}

// NewScrollbar creates a scrollbar around the child.
//
// Deviation: on iOS Flutter builds a CupertinoScrollbar; Glutter has no
// Cupertino library, so the Material scrollbar is built on every platform
// (SPEC §12.10).
//
// Flutter: material/scrollbar.dart Scrollbar.new
func NewScrollbar(child widgets.Widget, opts ...ScrollbarOpts) *Scrollbar {
	o := firstOpts(opts)
	return &Scrollbar{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), child: child, o: o}
}

// Child is the widget below the scrollbar.
//
// Flutter: material/scrollbar.dart Scrollbar.child
func (w *Scrollbar) Child() widgets.Widget { return w.child }

// Build builds the RawScrollbar that has the Material look.
//
// Flutter: material/scrollbar.dart Scrollbar.build
func (w *Scrollbar) Build(bc widgets.BuildContext) widgets.Widget {
	o := w.o
	return widgets.NewRawScrollbar(w.child, widgets.RawScrollbarOpts{
		Controller:            o.Controller,
		ThumbVisibility:       o.ThumbVisibility,
		TrackVisibility:       o.TrackVisibility,
		Thickness:             o.Thickness,
		Radius:                o.Radius,
		Interactive:           o.Interactive,
		ScrollbarOrientation:  o.ScrollbarOrientation,
		FadeDuration:          foundation.Some(scrollbarFadeDuration),
		TimeToFade:            foundation.Some(scrollbarTimeToFade),
		NotificationPredicate: o.NotificationPredicate,
		Subclass: func(base *widgets.RawScrollbarState) widgets.RawScrollbarSubclass {
			return &materialScrollbarState{base: base}
		},
	})
}

// materialScrollbarState is the part of _MaterialScrollbarState that is not the
// RawScrollbarState it extends in Flutter.
//
// Flutter: material/scrollbar.dart _MaterialScrollbarState
type materialScrollbarState struct {
	base                *widgets.RawScrollbarState
	hoverAnimation      *animation.AnimationController
	dragIsActive        bool
	hoverIsActive       bool
	colorScheme         *ColorScheme
	theme               *ThemeData
	scrollbarTheme      ScrollbarThemeData
	useAndroidScrollbar bool
}

func (s *materialScrollbarState) options() widgets.RawScrollbarOpts { return s.base.Widget().Options() }

func (s *materialScrollbarState) states() widgets.WidgetStates {
	var states widgets.WidgetStates
	if s.dragIsActive {
		states = states.With(widgets.WidgetStateDragged)
	}
	if s.hoverIsActive {
		states = states.With(widgets.WidgetStateHovered)
	}
	return states
}

func resolveOptBool(p widgets.WidgetStateProperty[foundation.Opt[bool]], states widgets.WidgetStates) foundation.Opt[bool] {
	if p == nil {
		return foundation.None[bool]()
	}
	return p.Resolve(states)
}

func resolveOptDouble(p widgets.WidgetStateProperty[foundation.Opt[float64]], states widgets.WidgetStates) foundation.Opt[float64] {
	if p == nil {
		return foundation.None[float64]()
	}
	return p.Resolve(states)
}

func resolveOptColor(p widgets.WidgetStateProperty[foundation.Opt[ui.Color]], states widgets.WidgetStates) foundation.Opt[ui.Color] {
	if p == nil {
		return foundation.None[ui.Color]()
	}
	return p.Resolve(states)
}

// ShowScrollbar is whether the thumb is always shown.
//
// Flutter: material/scrollbar.dart _MaterialScrollbarState.showScrollbar
func (s *materialScrollbarState) ShowScrollbar() bool {
	if v, ok := s.options().ThumbVisibility.Get(); ok {
		return v
	}
	return resolveOptBool(s.scrollbarTheme.ThumbVisibility, s.states()).OrElse(false)
}

// EnableGestures is whether the thumb responds to the pointer.
//
// Flutter: material/scrollbar.dart _MaterialScrollbarState.enableGestures
func (s *materialScrollbarState) EnableGestures() bool {
	if v, ok := s.options().Interactive.Get(); ok {
		return v
	}
	return s.scrollbarTheme.Interactive.OrElse(!s.useAndroidScrollbar)
}

func (s *materialScrollbarState) trackVisible(states widgets.WidgetStates) bool {
	if v, ok := s.options().TrackVisibility.Get(); ok {
		return v
	}
	return resolveOptBool(s.scrollbarTheme.TrackVisibility, states).OrElse(false)
}

func (s *materialScrollbarState) thumbColor(states widgets.WidgetStates) ui.Color {
	onSurface := s.colorScheme.OnSurface()
	var dragColor, hoverColor, idleColor ui.Color
	switch s.colorScheme.Brightness() {
	case ui.BrightnessDark:
		dragColor, hoverColor = onSurface.WithOpacity(0.75), onSurface.WithOpacity(0.65)
		idleColor = onSurface.WithOpacity(0.3)
	default:
		dragColor, hoverColor = onSurface.WithOpacity(0.6), onSurface.WithOpacity(0.5)
		idleColor = onSurface.WithOpacity(0.1)
	}
	if s.useAndroidScrollbar {
		idleColor = s.theme.HighlightColor.WithOpacity(1.0)
	}
	themed := resolveOptColor(s.scrollbarTheme.ThumbColor, states)
	if states.Has(widgets.WidgetStateDragged) {
		return themed.OrElse(dragColor)
	}
	if s.trackVisible(states) {
		return themed.OrElse(hoverColor)
	}
	return lerpColor(themed.OrElse(idleColor), themed.OrElse(hoverColor), s.hoverAnimation.Value())
}

func (s *materialScrollbarState) trackColor(states widgets.WidgetStates, themed foundation.Opt[ui.Color], light, dark float64) ui.Color {
	if s.ShowScrollbar() && s.trackVisible(states) {
		if c, ok := themed.Get(); ok {
			return c
		}
		onSurface := s.colorScheme.OnSurface()
		if s.colorScheme.Brightness() == ui.BrightnessDark {
			return onSurface.WithOpacity(dark)
		}
		return onSurface.WithOpacity(light)
	}
	return ui.NewColor(0x00000000)
}

func (s *materialScrollbarState) thickness(states widgets.WidgetStates) float64 {
	o := s.options()
	themed := resolveOptDouble(s.scrollbarTheme.Thickness, states)
	if states.Has(widgets.WidgetStateHovered) && s.trackVisible(states) {
		return firstOpt(o.Thickness, themed, foundation.Some(scrollbarThicknessWithTrack)).Value()
	}
	idle := scrollbarThickness
	if s.useAndroidScrollbar {
		idle /= 2
	}
	return firstOpt(o.Thickness, themed, foundation.Some(idle)).Value()
}

// InitState creates the controller of the hover animation.
//
// Flutter: material/scrollbar.dart _MaterialScrollbarState.initState
func (s *materialScrollbarState) InitState() {
	s.hoverAnimation = animation.NewAnimationController(s.base, animation.AnimationControllerOpts{Duration: foundation.Some(200 * time.Millisecond)})
	s.hoverAnimation.AddListener(func() { s.UpdateScrollbarPainter() })
}

// BeforeDidChangeDependencies reads the theme and the platform.
//
// Flutter: material/scrollbar.dart _MaterialScrollbarState.didChangeDependencies
func (s *materialScrollbarState) BeforeDidChangeDependencies() {
	s.theme = ThemeOf(s.base.Context())
	s.colorScheme = s.theme.ColorScheme
	s.scrollbarTheme = ScrollbarThemeOf(s.base.Context())
	s.useAndroidScrollbar = s.theme.Platform == foundation.TargetPlatformAndroid
}

// UpdateScrollbarPainter gives the painter the colors, the thickness, the radius
// and the margins.
//
// Flutter: material/scrollbar.dart _MaterialScrollbarState.updateScrollbarPainter
func (s *materialScrollbarState) UpdateScrollbarPainter() {
	if s.theme == nil {
		return
	}
	o := s.options()
	states := s.states()
	p := s.base.ScrollbarPainter()
	p.SetColor(s.thumbColor(states))
	p.SetTrackColor(s.trackColor(states, resolveOptColor(s.scrollbarTheme.TrackColor, states), 0.03, 0.05))
	p.SetTrackBorderColor(s.trackColor(states, resolveOptColor(s.scrollbarTheme.TrackBorderColor, states), 0.1, 0.25))
	p.SetTextDirection(directionOf(s.base.Context()))
	p.SetThickness(s.thickness(states))
	radius := firstOpt(o.Radius, s.scrollbarTheme.Radius)
	if radius.IsNone() && !s.useAndroidScrollbar {
		radius = foundation.Some(ui.NewRadiusCircular(8.0))
	}
	p.SetRadius(radius)
	cross := scrollbarMargin
	if s.useAndroidScrollbar {
		cross = 0
	}
	p.SetCrossAxisMargin(s.scrollbarTheme.CrossAxisMargin.OrElse(cross))
	p.SetMainAxisMargin(s.scrollbarTheme.MainAxisMargin.OrElse(0))
	p.SetMinLength(s.scrollbarTheme.MinThumbLength.OrElse(scrollbarMinLength))
	p.SetPadding(widgets.MediaQueryPaddingOf(s.base.Context()))
	p.SetScrollbarOrientation(o.ScrollbarOrientation)
	p.SetIgnorePointer(!s.EnableGestures())
}

// HandleThumbPressStart marks the thumb as dragged.
//
// Flutter: material/scrollbar.dart _MaterialScrollbarState.handleThumbPressStart
func (s *materialScrollbarState) HandleThumbPressStart(localPosition ui.Offset) {
	s.base.HandleThumbPressStart(localPosition)
	s.base.SetState(func() { s.dragIsActive = true })
}

// HandleThumbPressEnd marks the thumb as released.
//
// Flutter: material/scrollbar.dart _MaterialScrollbarState.handleThumbPressEnd
func (s *materialScrollbarState) HandleThumbPressEnd(localPosition ui.Offset, velocity gestures.Velocity) {
	s.base.HandleThumbPressEnd(localPosition, velocity)
	s.base.SetState(func() { s.dragIsActive = false })
}

// HandleHover darkens the thumb while the mouse is over it.
//
// Flutter: material/scrollbar.dart _MaterialScrollbarState.handleHover
func (s *materialScrollbarState) HandleHover(event *gestures.PointerHoverEvent) {
	s.base.HandleHover(event)
	if s.base.IsPointerOverScrollbar(event.Position(), event.Kind()) {
		s.base.SetState(func() { s.hoverIsActive = true })
		s.hoverAnimation.Forward()
	} else if s.hoverIsActive {
		s.base.SetState(func() { s.hoverIsActive = false })
		s.hoverAnimation.Reverse()
	}
}

// HandleHoverExit lightens the thumb again.
//
// Flutter: material/scrollbar.dart _MaterialScrollbarState.handleHoverExit
func (s *materialScrollbarState) HandleHoverExit(event *gestures.PointerExitEvent) {
	s.base.HandleHoverExit(event)
	s.base.SetState(func() { s.hoverIsActive = false })
	s.hoverAnimation.Reverse()
}

// BeforeDispose disposes the hover animation.
//
// Flutter: material/scrollbar.dart _MaterialScrollbarState.dispose
func (s *materialScrollbarState) BeforeDispose() { s.hoverAnimation.Dispose() }
