// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// DialogRouteOpts are the optional parameters of NewDialogRoute.
//
// Flutter: material/dialog.dart DialogRoute.new
type DialogRouteOpts struct {
	// Themes are the themes of the context that shows the dialog; the dialog
	// is wrapped in them, as the route is above them.
	Themes *widgets.CapturedThemes
	// BarrierColor is the color of the barrier; the default is Colors.black54.
	BarrierColor foundation.Opt[ui.Color]
	// BarrierDismissible is whether tapping the barrier pops the dialog; the
	// default is true.
	BarrierDismissible foundation.Opt[bool]
	// BarrierLabel is the semantic label of the barrier; the default is the
	// modal barrier dismiss label of the localizations.
	BarrierLabel string
	// UseSafeArea keeps the dialog out of the system intrusions; the default is
	// true.
	UseSafeArea foundation.Opt[bool]
	// RouteSettings are the settings of the route.
	RouteSettings widgets.RouteSettings
	// RequestFocus is whether the dialog requests the focus.
	RequestFocus foundation.Opt[bool]
	// TraversalEdgeBehavior is how the focus moves past the edge of the dialog;
	// the default is closed loop.
	TraversalEdgeBehavior widgets.TraversalEdgeBehavior
	// FullscreenDialog is whether the dialog is fullscreen.
	FullscreenDialog bool
	// AnimationStyle overrides the duration (150 ms) and the curves (ease out)
	// of the fade.
	AnimationStyle *animation.AnimationStyle
}

// NewDialogRoute creates a route for a Material dialog, which fades in and out.
//
// Deviation: Flutter's DialogRoute is a subclass of RawDialogRoute; the route
// here is a RawDialogRoute whose transition is the fade (SPEC §12.10).
//
// Flutter: material/dialog.dart DialogRoute.new
func NewDialogRoute(bc widgets.BuildContext, builder widgets.WidgetBuilder, opts ...DialogRouteOpts) *widgets.RawDialogRoute {
	o := firstOpts(opts)
	useSafeArea := o.UseSafeArea.OrElse(true)
	barrierLabel := o.BarrierLabel
	if barrierLabel == "" {
		barrierLabel = MaterialLocalizationsOf(bc).ModalBarrierDismissLabel()
	}
	traversal := o.TraversalEdgeBehavior
	style := o.AnimationStyle
	duration := 150 * time.Millisecond
	curve, reverseCurve := animation.Curve(animation.Curves.EaseOut), animation.Curve(animation.Curves.EaseOut)
	if style != nil {
		if style.Duration.IsSome() {
			duration = style.Duration.Value()
		}
		if style.Curve != nil {
			curve = style.Curve
		}
		if style.ReverseCurve != nil {
			reverseCurve = style.ReverseCurve
		}
	}
	var curved *animation.CurvedAnimation
	var curvedParent animation.Animation[float64]
	return widgets.NewRawDialogRoute(func(_ widgets.BuildContext, _, _ animation.Animation[float64]) widgets.Widget {
		var dialog widgets.Widget = widgets.NewBuilder(builder)
		if o.Themes != nil {
			dialog = o.Themes.Wrap(dialog)
		}
		if useSafeArea {
			dialog = widgets.NewSafeArea(dialog)
		}
		return widgets.NewSemantics(dialog, widgets.SemanticsOpts{HitTestBehavior: foundation.Some(ui.SemanticsHitTestBehaviorOpaque)})
	}, widgets.RawDialogRouteOpts{
		ModalRouteOpts: widgets.ModalRouteOpts{
			RouteOpts:             widgets.RouteOpts{Settings: o.RouteSettings, RequestFocus: o.RequestFocus},
			TraversalEdgeBehavior: traversal,
		},
		BarrierColor:       foundation.Some(o.BarrierColor.OrElse(Colors.Black54)),
		BarrierDismissible: o.BarrierDismissible,
		BarrierLabel:       barrierLabel,
		TransitionDuration: foundation.Some(duration),
		FullscreenDialog:   o.FullscreenDialog,
		TransitionBuilder: func(_ widgets.BuildContext, a, _ animation.Animation[float64], child widgets.Widget) widgets.Widget {
			if curved == nil || curvedParent != a {
				if curved != nil {
					curved.Dispose()
				}
				curvedParent = a
				curved = animation.NewCurvedAnimation(a, curve, reverseCurve)
			}
			return widgets.NewFadeTransition(curved, child)
		},
	})
}

// ShowDialogOpts are the optional parameters of ShowDialog.
//
// Flutter: material/dialog.dart showDialog
type ShowDialogOpts struct {
	// BarrierDismissible is whether tapping the barrier pops the dialog; the
	// default is true.
	BarrierDismissible foundation.Opt[bool]
	// BarrierColor is the color of the barrier; the default comes from the dialog
	// theme, then is Colors.black54.
	BarrierColor foundation.Opt[ui.Color]
	// BarrierLabel is the semantic label of the barrier.
	BarrierLabel string
	// UseSafeArea keeps the dialog out of the system intrusions; the default is
	// true.
	UseSafeArea foundation.Opt[bool]
	// UseRootNavigator pushes the dialog onto the root navigator; the default is
	// true.
	UseRootNavigator foundation.Opt[bool]
	// RouteSettings are the settings of the route.
	RouteSettings widgets.RouteSettings
	// TraversalEdgeBehavior is how the focus moves past the edge of the dialog.
	TraversalEdgeBehavior widgets.TraversalEdgeBehavior
	// FullscreenDialog is whether the dialog is fullscreen.
	FullscreenDialog bool
	// RequestFocus is whether the dialog requests the focus.
	RequestFocus foundation.Opt[bool]
	// AnimationStyle overrides the duration and the curves of the fade.
	AnimationStyle *animation.AnimationStyle
}

// ShowDialog displays a Material dialog above the current contents of the app,
// with Material entrance and exit animations, modal barrier color, and modal
// barrier behavior. It returns a channel that gets the value the dialog is
// popped with, then is closed; the zero value of T when it is popped with
// nothing or with a value of another type.
//
// In debug builds it panics with the tag ShowDialogContextInactive when the
// context is no longer in the tree.
//
// Deviation: the dialog is always a route of the navigator; Flutter's separate
// windows for dialogs are not ported (SPEC §12.10).
//
// Flutter: material/dialog.dart showDialog
func ShowDialog[T any](bc widgets.BuildContext, builder widgets.WidgetBuilder, opts ...ShowDialogOpts) <-chan T {
	return convertChan[T](showDialogAny(bc, builder, firstOpts(opts)))
}

// ShowAdaptiveDialog displays either a Material or a Cupertino dialog depending
// on the platform.
//
// Deviation: Glutter has no Cupertino library; the dialog is the Material one,
// and it is not dismissible by a tap on the barrier on iOS and macOS, as
// Flutter's Cupertino dialog is (SPEC §12.10).
//
// Flutter: material/dialog.dart showAdaptiveDialog
func ShowAdaptiveDialog[T any](bc widgets.BuildContext, builder widgets.WidgetBuilder, opts ...ShowDialogOpts) <-chan T {
	o := firstOpts(opts)
	if o.BarrierDismissible.IsNone() {
		switch ThemeOf(bc).Platform {
		case foundation.TargetPlatformIOS, foundation.TargetPlatformMacOS:
			o.BarrierDismissible = foundation.Some(false)
		}
	}
	return convertChan[T](showDialogAny(bc, builder, o))
}

// showDialogAny pushes the dialog route and returns the channel of its result.
func showDialogAny(bc widgets.BuildContext, builder widgets.WidgetBuilder, o ShowDialogOpts) <-chan any {
	debugCheckContextActive(bc)
	DebugCheckHasMaterialLocalizations(bc)
	navigator := widgets.NavigatorOf(bc, widgets.NavigatorOfOpts{RootNavigator: o.UseRootNavigator.OrElse(true)})
	themes := widgets.CaptureThemes(bc, navigator.Context())
	return navigator.Push(NewDialogRoute(bc, builder, DialogRouteOpts{
		Themes:                themes,
		BarrierColor:          firstOpt(o.BarrierColor, DialogThemeOf(bc).BarrierColor, ThemeOf(bc).DialogTheme.BarrierColor),
		BarrierDismissible:    o.BarrierDismissible,
		BarrierLabel:          o.BarrierLabel,
		UseSafeArea:           o.UseSafeArea,
		RouteSettings:         o.RouteSettings,
		RequestFocus:          o.RequestFocus,
		TraversalEdgeBehavior: firstNonZero(o.TraversalEdgeBehavior, widgets.TraversalEdgeBehaviorClosedLoop),
		FullscreenDialog:      o.FullscreenDialog,
		AnimationStyle:        o.AnimationStyle,
	}))
}

// debugCheckContextActive panics with the tag ShowDialogContextInactive when the
// context is an element that is not in the tree, which happens when a dialog is
// shown after the widget was disposed.
//
// Flutter: material/dialog.dart _debugIsActive
func debugCheckContextActive(bc widgets.BuildContext) {
	if !foundation.DebugMode {
		return
	}
	if e, ok := bc.(interface{ DebugIsActive() bool }); ok && !e.DebugIsActive() {
		panic(errors.New("ShowDialogContextInactive", "this BuildContext is no longer valid: the showDialog function was called after the widget was disposed; use a parent context instead"))
	}
}

// convertChan makes a channel of T from a channel of results. A result of
// another type, and nil, arrive as the zero value of T.
func convertChan[T any](in <-chan any) <-chan T {
	out := make(chan T, 1)
	go func() {
		defer close(out)
		if v, ok := <-in; ok {
			t, _ := v.(T)
			out <- t
		}
	}()
	return out
}
