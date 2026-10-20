// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"fmt"
	"math"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

const (
	snackBarSingleLineVerticalPadding = 14.0
	snackBarTransitionDuration        = 250 * time.Millisecond
	snackBarDisplayDuration           = 4000 * time.Millisecond
)

// The curves of the snack bar animations.
//
// Flutter: material/snack_bar.dart _snackBarHeightCurve, _snackBarM3HeightCurve, _snackBarFadeInCurve, _snackBarM3FadeInCurve, _snackBarFadeOutCurve
func snackBarCurves() (height, m3Height, fadeIn, m3FadeIn, fadeOut animation.Curve) {
	return animation.Curves.FastOutSlowIn, animation.Curves.EaseInOutQuart, animation.NewInterval(0.4, 1.0),
		animation.NewInterval(0.4, 0.6, animation.Curves.EaseInCirc), animation.NewInterval(0.72, 1.0, animation.Curves.FastOutSlowIn)
}

// SnackBarClosedReason specifies how a SnackBar was closed.
//
// Flutter: material/snack_bar.dart SnackBarClosedReason
type SnackBarClosedReason int

const (
	// SnackBarClosedReasonUnspecified is the zero value.
	//
	// Flutter: no counterpart
	SnackBarClosedReasonUnspecified SnackBarClosedReason = iota
	// SnackBarClosedReasonAction means the snack bar was closed after the user
	// tapped a SnackBarAction.
	//
	// Flutter: material/snack_bar.dart SnackBarClosedReason.action
	SnackBarClosedReasonAction
	// SnackBarClosedReasonDismiss means the snack bar was closed through a
	// semantics dismiss action or its close icon.
	//
	// Flutter: material/snack_bar.dart SnackBarClosedReason.dismiss
	SnackBarClosedReasonDismiss
	// SnackBarClosedReasonSwipe means the snack bar was closed by a user's swipe.
	//
	// Flutter: material/snack_bar.dart SnackBarClosedReason.swipe
	SnackBarClosedReasonSwipe
	// SnackBarClosedReasonHide means the snack bar was closed by
	// ScaffoldMessengerState.hideCurrentSnackBar.
	//
	// Flutter: material/snack_bar.dart SnackBarClosedReason.hide
	SnackBarClosedReasonHide
	// SnackBarClosedReasonRemove means the snack bar was closed by
	// ScaffoldMessengerState.removeCurrentSnackBar.
	//
	// Flutter: material/snack_bar.dart SnackBarClosedReason.remove
	SnackBarClosedReasonRemove
	// SnackBarClosedReasonTimeout means the snack bar was closed because its
	// timer expired.
	//
	// Flutter: material/snack_bar.dart SnackBarClosedReason.timeout
	SnackBarClosedReasonTimeout
)

// Name returns the Dart enum name.
//
// Flutter: material/snack_bar.dart SnackBarClosedReason.name
func (r SnackBarClosedReason) Name() string {
	return enumName(int(r), "action", "dismiss", "swipe", "hide", "remove", "timeout")
}

// String returns the name Dart prints, such as "SnackBarClosedReason.hide".
//
// Flutter: material/snack_bar.dart SnackBarClosedReason.toString
func (r SnackBarClosedReason) String() string { return "SnackBarClosedReason." + r.Name() }

// SnackBarActionOpts are the optional parameters of NewSnackBarAction.
//
// Deviation: the colors are properties of the widget state, where Flutter's are
// a Color that may be a WidgetStateColor; a plain color is
// widgets.NewWidgetStatePropertyAll(color) (SPEC §12.10).
//
// Flutter: material/snack_bar.dart SnackBarAction.new
type SnackBarActionOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// TextColor is the button's text color.
	TextColor widgets.WidgetStateProperty[ui.Color]
	// DisabledTextColor is the button's text color when disabled.
	DisabledTextColor foundation.Opt[ui.Color]
	// BackgroundColor is the button's background color.
	BackgroundColor widgets.WidgetStateProperty[ui.Color]
	// DisabledBackgroundColor is the button's background color when disabled.
	DisabledBackgroundColor foundation.Opt[ui.Color]
}

// SnackBarAction is a button for a SnackBar, known as an "action". Snack bar
// actions are always enabled; instead of disabling a snack bar action, avoid
// including it in the snack bar in the first place.
//
// TODO(step 13.3): the button is a stand-in for TextButton.
//
// Flutter: material/snack_bar.dart SnackBarAction
type SnackBarAction struct {
	widgets.StatefulWidget
	label     string
	onPressed func()
	o         SnackBarActionOpts
}

// NewSnackBarAction creates the action.
//
// Flutter: material/snack_bar.dart SnackBarAction.new
func NewSnackBarAction(label string, onPressed func(), opts ...SnackBarActionOpts) *SnackBarAction {
	o := firstOpts(opts)
	return &SnackBarAction{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), label: label, onPressed: onPressed, o: o}
}

// Label is the button label.
//
// Flutter: material/snack_bar.dart SnackBarAction.label
func (w *SnackBarAction) Label() string { return w.label }

// OnPressed is the callback that is called when the action is pressed.
//
// Flutter: material/snack_bar.dart SnackBarAction.onPressed
func (w *SnackBarAction) OnPressed() func() { return w.onPressed }

// CreateState creates the state.
//
// Flutter: material/snack_bar.dart SnackBarAction.createState
func (w *SnackBarAction) CreateState() widgets.State { return &snackBarActionState{} }

type snackBarActionState struct {
	widgets.StateOf[*SnackBarAction]
	haveTriggeredAction bool
}

func (s *snackBarActionState) handlePressed() {
	if s.haveTriggeredAction {
		return
	}
	s.SetState(func() { s.haveTriggeredAction = true })
	s.Widget().onPressed()
	ScaffoldMessengerOf(s.Context()).HideCurrentSnackBar(SnackBarClosedReasonAction)
}

func (s *snackBarActionState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	defaults := snackBarDefaults(bc)
	snackBarTheme := SnackBarThemeOf(bc)

	foreground := func() widgets.WidgetStateProperty[ui.Color] {
		return widgets.WidgetStateColorResolveWith(func(states widgets.WidgetStates) ui.Color {
			if states.Has(widgets.WidgetStateDisabled) {
				if c, ok := w.o.DisabledTextColor.Get(); ok {
					return c
				}
				if c, ok := snackBarTheme.DisabledActionTextColor.Get(); ok {
					return c
				}
				return defaults.DisabledActionTextColor.Value()
			}
			for _, p := range []widgets.WidgetStateProperty[ui.Color]{w.o.TextColor, snackBarTheme.ActionTextColor, defaults.ActionTextColor} {
				if p != nil {
					return p.Resolve(states)
				}
			}
			return ui.Color{}
		})
	}()
	background := func() widgets.WidgetStateProperty[ui.Color] {
		return widgets.WidgetStateColorResolveWith(func(states widgets.WidgetStates) ui.Color {
			if states.Has(widgets.WidgetStateDisabled) {
				if c, ok := w.o.DisabledBackgroundColor.Get(); ok {
					return c
				}
				if c, ok := snackBarTheme.DisabledActionBackgroundColor.Get(); ok {
					return c
				}
				return Colors.Transparent
			}
			for _, p := range []widgets.WidgetStateProperty[ui.Color]{w.o.BackgroundColor, snackBarTheme.ActionBackgroundColor} {
				if p != nil {
					return p.Resolve(states)
				}
			}
			return Colors.Transparent
		})
	}()
	var onPressed func()
	if !s.haveTriggeredAction {
		onPressed = s.handlePressed
	}
	return &tempTextButton{StatelessWidget: widgets.NewStatelessWidgetBase(), label: w.label, onPressed: onPressed,
		foreground: foreground, background: background, horizontalInset: 12}
}

// SnackBarOpts are the optional parameters of NewSnackBar.
//
// Flutter: material/snack_bar.dart SnackBar.new
type SnackBarOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// BackgroundColor is the snack bar's background color.
	BackgroundColor foundation.Opt[ui.Color]
	// Elevation is the z-coordinate at which to place the snack bar; it must not
	// be negative.
	Elevation foundation.Opt[float64]
	// Margin is the outer margin around the snack bar; it can only be used with
	// the floating behavior and not together with Width.
	Margin painting.EdgeInsetsGeometry
	// Padding is the amount of padding to apply to the snack bar's content and
	// optional action.
	Padding painting.EdgeInsetsGeometry
	// Width is the width of the snack bar; it can only be used with the floating
	// behavior and not together with Margin.
	Width foundation.Opt[float64]
	// Shape is the shape of the snack bar's Material.
	Shape painting.ShapeBorder
	// HitTestBehavior defines how the snack bar area is hit tested.
	//
	// TODO: it is used by Dismissible, which is not ported.
	HitTestBehavior rendering.HitTestBehavior
	// Behavior is where the snack bar appears; the default is fixed.
	Behavior SnackBarBehavior
	// Action is an optional action that the user can take based on the snack bar.
	Action *SnackBarAction
	// ActionOverflowThreshold is the fraction of the width above which the action
	// moves to its own row; it is between 0 and 1.
	ActionOverflowThreshold foundation.Opt[float64]
	// ShowCloseIcon says whether to show an icon to close the snack bar.
	ShowCloseIcon foundation.Opt[bool]
	// CloseIconColor is the color of the close icon.
	CloseIconColor foundation.Opt[ui.Color]
	// Duration is how long the snack bar is shown; the default is 4 seconds.
	Duration foundation.Opt[time.Duration]
	// Persist says whether the snack bar stays until it is dismissed; the default
	// is true when there is an action.
	Persist foundation.Opt[bool]
	// Animation is the animation driving the entrance and exit of the snack bar;
	// ScaffoldMessenger supplies it.
	Animation animation.Animation[float64]
	// OnVisible is called the first time that the snack bar is visible.
	OnVisible func()
	// DismissDirection is the direction in which the snack bar can be dismissed.
	DismissDirection widgets.DismissDirection
	// ClipBehavior is how the content is clipped; the default is ClipHardEdge.
	ClipBehavior ui.Clip
}

// SnackBar is a lightweight message with an optional action which briefly
// displays at the bottom of the screen. Show it with
// ScaffoldMessengerState.ShowSnackBar.
//
// TODO(step 13.3): the close icon and the action are stand-ins for IconButton and
// TextButton. TODO: swiping the snack bar away needs Dismissible, which is not
// ported; the dismiss direction is stored but has no effect, and only the
// semantics dismiss action and the close icon close the bar by hand.
//
// Flutter: material/snack_bar.dart SnackBar
type SnackBar struct {
	widgets.StatefulWidget
	content widgets.Widget
	o       SnackBarOpts
}

// NewSnackBar creates a snack bar. In debug builds it panics with the tag
// SnackBarElevationNegative when the elevation is negative,
// SnackBarWidthAndMargin when both a width and a margin are given, and
// SnackBarOverflowThreshold when the overflow threshold is not between 0 and 1.
//
// Flutter: material/snack_bar.dart SnackBar.new
func NewSnackBar(content widgets.Widget, opts ...SnackBarOpts) *SnackBar {
	o := firstOpts(opts)
	if foundation.DebugMode {
		if o.Elevation.OrElse(0) < 0 {
			panic(errors.New("SnackBarElevationNegative", "the elevation of a SnackBar must not be negative", "elevation", o.Elevation.Value()))
		}
		if o.Width.IsSome() && o.Margin != nil {
			panic(errors.New("SnackBarWidthAndMargin", "Width and margin can not be used together"))
		}
		if t, ok := o.ActionOverflowThreshold.Get(); ok && (t < 0 || t > 1) {
			panic(errors.New("SnackBarOverflowThreshold", "Action overflow threshold must be between 0 and 1 inclusive", "threshold", t))
		}
	}
	if o.Persist.IsNone() {
		o.Persist = foundation.Some(o.Action != nil)
	}
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipHardEdge
	}
	return &SnackBar{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), content: content, o: o}
}

// Content is the primary content of the snack bar.
//
// Flutter: material/snack_bar.dart SnackBar.content
func (w *SnackBar) Content() widgets.Widget { return w.content }

// Duration is how long the snack bar is shown.
//
// Flutter: material/snack_bar.dart SnackBar.duration
func (w *SnackBar) Duration() time.Duration { return w.o.Duration.OrElse(snackBarDisplayDuration) }

// Persist is whether the snack bar stays until it is dismissed.
//
// Flutter: material/snack_bar.dart SnackBar.persist
func (w *SnackBar) Persist() bool { return w.o.Persist.Value() }

// Behavior is where the snack bar appears, when it was set.
//
// Flutter: material/snack_bar.dart SnackBar.behavior
func (w *SnackBar) Behavior() SnackBarBehavior { return w.o.Behavior }

// Width is the width of the snack bar, when it was set.
//
// Flutter: material/snack_bar.dart SnackBar.width
func (w *SnackBar) Width() foundation.Opt[float64] { return w.o.Width }

// Action is the action of the snack bar.
//
// Flutter: material/snack_bar.dart SnackBar.action
func (w *SnackBar) Action() *SnackBarAction { return w.o.Action }

// Animation is the animation of the snack bar, when ScaffoldMessenger gave one.
//
// Flutter: material/snack_bar.dart SnackBar.animation
func (w *SnackBar) Animation() animation.Animation[float64] { return w.o.Animation }

// SnackBarCreateAnimationController creates the animation controller that
// ScaffoldMessengerState.showSnackBar uses: the duration is the given one or
// 250 ms.
//
// Flutter: material/snack_bar.dart SnackBar.createAnimationController
func SnackBarCreateAnimationController(vsync scheduler.TickerProvider, duration, reverseDuration foundation.Opt[time.Duration]) *animation.AnimationController {
	return animation.NewAnimationController(vsync, animation.AnimationControllerOpts{
		Duration: foundation.Some(duration.OrElse(snackBarTransitionDuration)), ReverseDuration: reverseDuration, DebugLabel: "SnackBar"})
}

// WithAnimation creates a copy of this snack bar with the given animation and the
// key, if it has none, of the fallback.
//
// Flutter: material/snack_bar.dart SnackBar.withAnimation
func (w *SnackBar) WithAnimation(newAnimation animation.Animation[float64], fallbackKey foundation.Key) *SnackBar {
	o := w.o
	o.Animation = newAnimation
	if o.Key == nil {
		o.Key = fallbackKey
	}
	n := *w
	n.o = o
	n.StatefulWidget = widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key})
	return &n
}

// CreateState creates the state.
//
// Flutter: material/snack_bar.dart SnackBar.createState
func (w *SnackBar) CreateState() widgets.State { return &snackBarState{} }

type snackBarState struct {
	widgets.StateOf[*SnackBar]
	wasVisible     bool
	handle         animation.StatusListenerHandle
	height         *animation.CurvedAnimation
	fadeIn         *animation.CurvedAnimation
	fadeInM3       *animation.CurvedAnimation
	fadeOut        *animation.CurvedAnimation
	heightM3       *animation.CurvedAnimation
	dismissibleKey foundation.Key
}

func (s *snackBarState) InitState() {
	s.dismissibleKey = foundation.NewUniqueKey()
	s.handle = s.Widget().o.Animation.AddStatusListener(s.onAnimationStatusChanged)
	s.setAnimations()
}

func (s *snackBarState) DidUpdateWidget(old *SnackBar) {
	if s.Widget().o.Animation != old.o.Animation {
		old.o.Animation.RemoveStatusListener(s.handle)
		s.handle = s.Widget().o.Animation.AddStatusListener(s.onAnimationStatusChanged)
		s.disposeAnimations()
		s.setAnimations()
	}
}

func (s *snackBarState) setAnimations() {
	parent := s.Widget().o.Animation
	height, m3Height, fadeIn, m3FadeIn, fadeOut := snackBarCurves()
	s.height = animation.NewCurvedAnimation(parent, height)
	s.fadeIn = animation.NewCurvedAnimation(parent, fadeIn)
	s.fadeInM3 = animation.NewCurvedAnimation(parent, m3FadeIn)
	s.fadeOut = animation.NewCurvedAnimation(parent, fadeOut, animation.NewThreshold(0.0))
	// Material 3 Animation has a height animation on entry, but a direct fade out on exit.
	s.heightM3 = animation.NewCurvedAnimation(parent, m3Height, animation.NewThreshold(0.0))
}

func (s *snackBarState) disposeAnimations() {
	for _, a := range []*animation.CurvedAnimation{s.height, s.fadeIn, s.fadeInM3, s.fadeOut, s.heightM3} {
		if a != nil {
			a.Dispose()
		}
	}
	s.height, s.fadeIn, s.fadeInM3, s.fadeOut, s.heightM3 = nil, nil, nil, nil, nil
}

func (s *snackBarState) Dispose() {
	s.Widget().o.Animation.RemoveStatusListener(s.handle)
	s.disposeAnimations()
}

func (s *snackBarState) onAnimationStatusChanged(status animation.AnimationStatus) {
	if status.IsCompleted() {
		if f := s.Widget().o.OnVisible; f != nil && !s.wasVisible {
			f()
		}
		s.wasVisible = true
	}
}

// Build builds the snack bar.
//
// Flutter: material/snack_bar.dart _SnackBarState.build
func (s *snackBarState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	o := w.o
	widgets.DebugCheckHasMediaQuery(bc)
	accessibleNavigation := widgets.MediaQueryAccessibleNavigationOf(bc)
	theme := ThemeOf(bc)
	colorScheme := theme.ColorScheme
	snackBarTheme := SnackBarThemeOf(bc)
	isThemeDark := theme.Brightness() == ui.BrightnessDark
	buttonColor := colorScheme.Secondary()
	if isThemeDark {
		buttonColor = colorScheme.Primary()
	}
	defaults := snackBarDefaults(bc)

	// SnackBar uses a theme that is the opposite brightness from the surrounding
	// theme.
	brightness := ui.BrightnessDark
	if isThemeDark {
		brightness = ui.BrightnessLight
	}
	// Invert the theme values for Material 2. Material 3 values are tokenized to
	// pre-inverted values.
	effectiveTheme := theme
	if !theme.UseMaterial3 {
		effectiveTheme = theme.CopyWith(ThemeDataCopy{ColorScheme: foundation.Some(NewColorScheme(brightness,
			colorScheme.OnPrimary(), colorScheme.Primary(), buttonColor, colorScheme.Secondary(),
			colorScheme.OnError(), colorScheme.Error(), colorScheme.OnSurface(), colorScheme.Surface()))})
	}

	contentTextStyle := snackBarTheme.ContentTextStyle
	if contentTextStyle == nil {
		contentTextStyle = defaults.ContentTextStyle
	}
	behavior := o.Behavior
	if behavior == SnackBarBehaviorUnspecified {
		behavior = snackBarTheme.Behavior
	}
	if behavior == SnackBarBehaviorUnspecified {
		behavior = defaults.Behavior
	}
	width := o.Width
	if width.IsNone() {
		width = snackBarTheme.Width
	}
	if foundation.DebugMode && behavior != SnackBarBehaviorFloating {
		// Whether the behavior is set through the constructor or the theme, assert
		// that other properties are configured properly.
		message := func(parameter string) error {
			prefix := parameter + " can only be used with floating behavior."
			switch {
			case o.Behavior != SnackBarBehaviorUnspecified:
				prefix += " SnackBarBehavior.fixed was set in the SnackBar constructor."
			case snackBarTheme.Behavior != SnackBarBehaviorUnspecified:
				prefix += " SnackBarBehavior.fixed was set by the inherited SnackBarThemeData."
			default:
				prefix += " SnackBarBehavior.fixed was set by default."
			}
			return errors.New("SnackBarFloatingOnly", prefix, "parameter", parameter)
		}
		if o.Margin != nil {
			panic(message("Margin"))
		}
		if width.IsSome() {
			panic(message("Width"))
		}
	}
	showCloseIcon := o.ShowCloseIcon.OrElse(snackBarTheme.ShowCloseIcon.OrElse(defaults.ShowCloseIcon.Value()))
	isFloating := behavior == SnackBarBehaviorFloating
	horizontalPadding := 24.0
	if isFloating {
		horizontalPadding = 16.0
	}
	padding := o.Padding
	if padding == nil {
		end := horizontalPadding
		if o.Action != nil || showCloseIcon {
			end = 0
		}
		padding = painting.NewEdgeInsetsDirectionalFromSTEB(horizontalPadding, 0, end, 0)
	}
	paddingRight := horizontalPadding
	if o.Padding != nil {
		paddingRight = o.Padding.Resolve(ui.TextDirectionLTR).Right
	}
	actionHorizontalMargin := paddingRight / 2
	iconHorizontalMargin := paddingRight / 12.0

	var iconButton widgets.Widget
	const closeIconSize = 24.0
	if showCloseIcon {
		iconButton = &tempIconButton{StatelessWidget: widgets.NewStatelessWidgetBase(), icon: Icons.Close(), size: closeIconSize,
			color:     foundation.Some(o.CloseIconColor.OrElse(snackBarTheme.CloseIconColor.OrElse(defaults.CloseIconColor.Value()))),
			onPressed: func() { ScaffoldMessengerOf(bc).HideCurrentSnackBar(SnackBarClosedReasonDismiss) },
			tooltip:   MaterialLocalizationsOf(bc).CloseButtonTooltip()}
	}

	// Calculate combined width of Action, Icon, and their padding, if they are present.
	actionLabel := ""
	if o.Action != nil {
		actionLabel = o.Action.label
	}
	actionTextWidth := measureTextWidth(bc, actionLabel, theme.TextTheme.LabelLarge)
	actionAndIconWidth := actionTextWidth
	if o.Action != nil {
		actionAndIconWidth += actionHorizontalMargin
	}
	if showCloseIcon {
		// Dart's precedence: iconSize ?? (0 + iconHorizontalMargin) is the icon size.
		actionAndIconWidth += closeIconSize
	}
	_ = iconHorizontalMargin

	var margin painting.EdgeInsets
	switch {
	case o.Margin != nil:
		margin = o.Margin.Resolve(ui.TextDirectionLTR)
	case snackBarTheme.InsetPadding.IsSome():
		margin = snackBarTheme.InsetPadding.Value()
	default:
		margin = defaults.InsetPadding.Value()
	}
	snackBarWidth := width.OrElse(widgets.MediaQueryWidthOf(bc) - (margin.Left + margin.Right))
	actionOverflowThreshold := o.ActionOverflowThreshold.OrElse(snackBarTheme.ActionOverflowThreshold.OrElse(defaults.ActionOverflowThreshold.Value()))
	willOverflowAction := actionAndIconWidth/snackBarWidth > actionOverflowThreshold

	var maybeActionAndIcon []widgets.Widget
	if o.Action != nil {
		maybeActionAndIcon = append(maybeActionAndIcon, widgets.NewPadding(painting.NewEdgeInsetsSymmetric(actionHorizontalMargin, 0),
			&snackBarActionPadding{StatelessWidget: widgets.NewStatelessWidgetBase(), action: o.Action, foreground: buttonColor, horizontalPadding: horizontalPadding}))
	}
	if showCloseIcon {
		maybeActionAndIcon = append(maybeActionAndIcon, widgets.NewPadding(painting.NewEdgeInsetsSymmetric(iconHorizontalMargin, 0), iconButton))
	}

	contentPadding := painting.EdgeInsetsGeometry(painting.NewEdgeInsetsSymmetric(0, snackBarSingleLineVerticalPadding))
	if o.Padding != nil {
		contentPadding = painting.EdgeInsetsZero
	}
	row := []widgets.Widget{widgets.NewExpanded(widgets.NewPadding(contentPadding, widgets.NewDefaultTextStyle(contentTextStyle, w.content)))}
	if !willOverflowAction {
		row = append(row, maybeActionAndIcon...)
	} else {
		row = append(row, widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(snackBarWidth * 0.4)}))
	}
	wrapChildren := []widgets.Widget{widgets.NewRow(row)}
	if willOverflowAction {
		wrapChildren = append(wrapChildren, widgets.NewPadding(painting.EdgeInsets{Bottom: snackBarSingleLineVerticalPadding},
			widgets.NewRow(maybeActionAndIcon, widgets.RowOpts{MainAxisAlignment: rendering.MainAxisAlignmentEnd})))
	}
	var snackBar widgets.Widget = widgets.NewPadding(padding, widgets.NewWrap(wrapChildren))
	if !isFloating {
		snackBar = widgets.NewSafeArea(snackBar, widgets.SafeAreaOpts{Top: foundation.Some(false)})
	}

	elevation := o.Elevation.OrElse(snackBarTheme.Elevation.OrElse(defaults.Elevation.Value()))
	backgroundColor := o.BackgroundColor.OrElse(snackBarTheme.BackgroundColor.OrElse(defaults.BackgroundColor.Value()))
	shape := o.Shape
	if shape == nil {
		shape = snackBarTheme.Shape
	}
	if shape == nil && isFloating {
		shape = defaults.Shape
	}

	themed := widgets.Widget(snackBar)
	if !accessibleNavigation && !theme.UseMaterial3 {
		themed = widgets.NewFadeTransition(s.fadeOut, snackBar)
	}
	snackBar = NewMaterial(NewTheme(effectiveTheme, themed),
		MaterialOpts{Shape: shape, Elevation: elevation, Color: foundation.Some(backgroundColor), ClipBehavior: o.ClipBehavior})

	if isFloating {
		// If width is provided, do not include horizontal margins.
		if width.IsSome() {
			snackBar = widgets.NewPadding(painting.EdgeInsets{Top: margin.Top, Bottom: margin.Bottom},
				widgets.NewSizedBox(snackBar, widgets.SizedBoxOpts{Width: width}))
		} else {
			snackBar = widgets.NewPadding(margin, snackBar)
		}
		snackBar = widgets.NewSafeArea(snackBar, widgets.SafeAreaOpts{Top: foundation.Some(false), Bottom: foundation.Some(false)})
	}

	// TODO: a Dismissible with the dismiss direction wraps the snack bar here
	// (Flutter's widgets/dismissible.dart); it is not ported.
	snackBar = widgets.NewSemantics(snackBar, widgets.SemanticsOpts{
		Container: true, LiveRegion: foundation.Some(true),
		OnDismiss: func() { ScaffoldMessengerOf(bc).RemoveCurrentSnackBar(SnackBarClosedReasonDismiss) },
	})

	var transition widgets.Widget
	switch {
	case accessibleNavigation:
		transition = snackBar
	case isFloating && !theme.UseMaterial3:
		transition = widgets.NewFadeTransition(s.fadeIn, snackBar)
	case isFloating && theme.UseMaterial3:
		transition = widgets.NewFadeTransition(s.fadeInM3, widgets.NewValueListenableBuilder[float64](s.heightM3,
			func(_ widgets.BuildContext, value float64, child widgets.Widget) widgets.Widget {
				return widgets.NewAlign(child, widgets.AlignOpts{Alignment: painting.AlignmentBottomLeft, HeightFactor: foundation.Some(value)})
			}, snackBar))
	default:
		transition = widgets.NewValueListenableBuilder[float64](s.height,
			func(_ widgets.BuildContext, value float64, child widgets.Widget) widgets.Widget {
				return widgets.NewAlign(child, widgets.AlignOpts{Alignment: painting.AlignmentDirectionalTopStart, HeightFactor: foundation.Some(value)})
			}, snackBar)
	}
	return widgets.NewHero(fmt.Sprintf("<SnackBar Hero tag - %v>", w.content),
		widgets.NewClipRect(transition, widgets.ClipRectOpts{ClipBehavior: o.ClipBehavior}),
		widgets.HeroOpts{TransitionOnUserGestures: true})
}

// snackBarActionPadding applies the text color of the snack bar action's theme
// (the TextButtonTheme of Flutter's SnackBar), through the action's own theme.
//
// TODO(step 13.3): TextButtonTheme.
type snackBarActionPadding struct {
	widgets.StatelessWidget
	action            *SnackBarAction
	foreground        ui.Color
	horizontalPadding float64
}

func (w *snackBarActionPadding) Build(widgets.BuildContext) widgets.Widget {
	return w.action
}

// measureTextWidth is the width of the text in one line with the style, which
// SnackBar uses to decide whether the action overflows.
func measureTextWidth(bc widgets.BuildContext, text string, style *painting.TextStyle) float64 {
	if text == "" {
		return 0
	}
	var engine ui.ParagraphFactory
	if b := widgets.MaybeWidgetsBindingOf(bc); b != nil {
		engine = b.PlatformDispatcher().TextEngine()
	}
	tp := painting.NewTextPainter(painting.TextPainterOpts{
		TextEngine:    engine,
		Text:          painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some(text), Style: style}),
		MaxLines:      foundation.Some(1),
		TextDirection: ui.TextDirectionLTR,
	})
	defer tp.Dispose()
	tp.Layout(0, math.Inf(1))
	return tp.Size().Width
}

// snackBarDefaults returns the defaults of the snack bar of the theme.
func snackBarDefaults(bc widgets.BuildContext) SnackBarThemeData {
	theme := ThemeOf(bc)
	if theme.UseMaterial3 {
		return snackBarDefaultsM3(theme)
	}
	return snackBarDefaultsM2(theme)
}

// snackBarDefaultsM2 are the hand coded defaults of the snack bar for Material 2.
//
// Flutter: material/snack_bar.dart _SnackbarDefaultsM2
func snackBarDefaultsM2(theme *ThemeData) SnackBarThemeData {
	colors := theme.ColorScheme
	light := theme.Brightness() == ui.BrightnessLight
	background := colors.OnSurface()
	if light {
		background = ui.ColorAlphaBlend(colors.OnSurface().WithOpacity(0.80), colors.Surface())
	}
	contentBrightness := ui.BrightnessLight
	if light {
		contentBrightness = ui.BrightnessDark
	}
	disabledOpacity := 0.3
	if light {
		disabledOpacity = 0.38
	}
	return SnackBarThemeData{
		Elevation:               foundation.Some(6.0),
		BackgroundColor:         foundation.Some(background),
		ContentTextStyle:        NewThemeData(ThemeDataOpts{UseMaterial3: foundation.Some(theme.UseMaterial3), Brightness: contentBrightness}).TextTheme.TitleMedium,
		Behavior:                SnackBarBehaviorFixed,
		ActionTextColor:         widgets.NewWidgetStatePropertyAll(colors.Secondary()),
		DisabledActionTextColor: foundation.Some(colors.OnSurface().WithOpacity(disabledOpacity)),
		Shape:                   painting.NewRoundedRectangleBorder(painting.RoundedRectangleBorderOpts{BorderRadius: painting.NewBorderRadiusAll(ui.Radius{X: 4, Y: 4})}),
		InsetPadding:            foundation.Some(painting.NewEdgeInsetsFromLTRB(15.0, 5.0, 15.0, 10.0)),
		ShowCloseIcon:           foundation.Some(false),
		CloseIconColor:          foundation.Some(colors.OnSurface()),
		ActionOverflowThreshold: foundation.Some(0.25),
	}
}

// snackBarDefaultsM3 are the Material 3 defaults of the snack bar (generated from
// the Material token database in Flutter).
//
// Flutter: material/snack_bar.dart _SnackbarDefaultsM3
func snackBarDefaultsM3(theme *ThemeData) SnackBarThemeData {
	colors := theme.ColorScheme
	return SnackBarThemeData{
		BackgroundColor:         foundation.Some(colors.InverseSurface()),
		ActionTextColor:         widgets.WidgetStateColorResolveWith(func(widgets.WidgetStates) ui.Color { return colors.InversePrimary() }),
		DisabledActionTextColor: foundation.Some(colors.InversePrimary()),
		ContentTextStyle:        theme.TextTheme.BodyMedium.CopyWith(painting.TextStyleCopy{Color: foundation.Some(colors.OnInverseSurface())}),
		Elevation:               foundation.Some(6.0),
		Shape:                   painting.NewRoundedRectangleBorder(painting.RoundedRectangleBorderOpts{BorderRadius: painting.NewBorderRadiusAll(ui.Radius{X: 4, Y: 4})}),
		Behavior:                SnackBarBehaviorFixed,
		InsetPadding:            foundation.Some(painting.NewEdgeInsetsFromLTRB(15.0, 5.0, 15.0, 10.0)),
		ShowCloseIcon:           foundation.Some(false),
		CloseIconColor:          foundation.Some(colors.OnInverseSurface()),
		ActionOverflowThreshold: foundation.Some(0.25),
	}
}
