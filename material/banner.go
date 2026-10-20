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
	materialBannerTransitionDuration = 250 * time.Millisecond
	// maxContentTextScaleFactor is the biggest text scale factor of the content.
	//
	// Flutter: material/banner.dart _kMaxContentTextScaleFactor
	maxContentTextScaleFactor = 1.5
)

// MaterialBannerClosedReason specifies how a MaterialBanner was closed.
//
// Flutter: material/banner.dart MaterialBannerClosedReason
type MaterialBannerClosedReason int

const (
	// MaterialBannerClosedReasonUnspecified is the zero value.
	//
	// Flutter: no counterpart
	MaterialBannerClosedReasonUnspecified MaterialBannerClosedReason = iota
	// MaterialBannerClosedReasonDismiss means the banner was closed through a
	// semantics dismiss action.
	//
	// Flutter: material/banner.dart MaterialBannerClosedReason.dismiss
	MaterialBannerClosedReasonDismiss
	// MaterialBannerClosedReasonSwipe means the banner was closed by a user's
	// swipe.
	//
	// Flutter: material/banner.dart MaterialBannerClosedReason.swipe
	MaterialBannerClosedReasonSwipe
	// MaterialBannerClosedReasonHide means the banner was closed by
	// ScaffoldMessengerState.hideCurrentMaterialBanner.
	//
	// Flutter: material/banner.dart MaterialBannerClosedReason.hide
	MaterialBannerClosedReasonHide
	// MaterialBannerClosedReasonRemove means the banner was closed by
	// ScaffoldMessengerState.removeCurrentMaterialBanner.
	//
	// Flutter: material/banner.dart MaterialBannerClosedReason.remove
	MaterialBannerClosedReasonRemove
)

// Name returns the Dart enum name.
//
// Flutter: material/banner.dart MaterialBannerClosedReason.name
func (r MaterialBannerClosedReason) Name() string {
	return enumName(int(r), "dismiss", "swipe", "hide", "remove")
}

// String returns the name Dart prints, such as
// "MaterialBannerClosedReason.hide".
//
// Flutter: material/banner.dart MaterialBannerClosedReason.toString
func (r MaterialBannerClosedReason) String() string { return "MaterialBannerClosedReason." + r.Name() }

// MaterialBannerOpts are the optional parameters of NewMaterialBanner.
//
// Flutter: material/banner.dart MaterialBanner.new
type MaterialBannerOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// ContentTextStyle is the style of the text of the content; the default is the
	// body medium style of the theme.
	ContentTextStyle *painting.TextStyle
	// Elevation is the elevation of the banner; it must not be negative.
	Elevation foundation.Opt[float64]
	// Leading is the optional leading widget of the banner, typically an icon.
	Leading widgets.Widget
	// BackgroundColor is the background color of the banner.
	BackgroundColor foundation.Opt[ui.Color]
	// SurfaceTintColor is the surface tint color of the banner.
	SurfaceTintColor foundation.Opt[ui.Color]
	// ShadowColor is the color of the shadow below the banner.
	ShadowColor foundation.Opt[ui.Color]
	// DividerColor is the color of the divider below the banner when its
	// elevation is 0.
	DividerColor foundation.Opt[ui.Color]
	// Padding is the padding around the content; the default depends on the
	// number of actions.
	Padding painting.EdgeInsetsGeometry
	// Margin is the margin around the banner.
	Margin painting.EdgeInsetsGeometry
	// LeadingPadding is the padding around the leading widget; the default is 16
	// at the end.
	LeadingPadding painting.EdgeInsetsGeometry
	// ForceActionsBelow puts the actions below the content, even with only one.
	ForceActionsBelow bool
	// OverflowAlignment is how the actions are aligned when they overflow; the
	// default is end.
	OverflowAlignment rendering.OverflowBarAlignment
	// Animation is the animation driving the entrance and exit of the banner;
	// ScaffoldMessenger supplies it. Without one the banner is static.
	Animation animation.Animation[float64]
	// OnVisible is called the first time that the banner is visible.
	OnVisible func()
	// MinActionBarHeight is the minimum height of the actions bar; the default is
	// 52.
	MinActionBarHeight foundation.Opt[float64]
}

// MaterialBanner is a Material Design banner: a message with up to two actions
// that is displayed at the top of the screen and pushes the page content down.
// Show it with ScaffoldMessengerState.ShowMaterialBanner.
//
// Flutter: material/banner.dart MaterialBanner
type MaterialBanner struct {
	widgets.StatefulWidget
	content widgets.Widget
	actions []widgets.Widget
	o       MaterialBannerOpts
}

// NewMaterialBanner creates a banner. In debug builds it panics with the tag
// MaterialBannerElevationNegative when the elevation is negative.
//
// Flutter: material/banner.dart MaterialBanner.new
func NewMaterialBanner(content widgets.Widget, actions []widgets.Widget, opts ...MaterialBannerOpts) *MaterialBanner {
	o := firstOpts(opts)
	if foundation.DebugMode && o.Elevation.OrElse(0) < 0 {
		panic(errors.New("MaterialBannerElevationNegative", "the elevation of a MaterialBanner must not be negative", "elevation", o.Elevation.Value()))
	}
	if o.OverflowAlignment == rendering.OverflowBarAlignmentUnspecified {
		o.OverflowAlignment = rendering.OverflowBarAlignmentEnd
	}
	return &MaterialBanner{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), content: content, actions: actions, o: o}
}

// Content is the content of the banner.
//
// Flutter: material/banner.dart MaterialBanner.content
func (w *MaterialBanner) Content() widgets.Widget { return w.content }

// Actions are the actions of the banner.
//
// Flutter: material/banner.dart MaterialBanner.actions
func (w *MaterialBanner) Actions() []widgets.Widget { return w.actions }

// Elevation is the elevation of the banner, when it was set.
//
// Flutter: material/banner.dart MaterialBanner.elevation
func (w *MaterialBanner) Elevation() foundation.Opt[float64] { return w.o.Elevation }

// Animation is the animation of the banner, when ScaffoldMessenger gave one.
//
// Flutter: material/banner.dart MaterialBanner.animation
func (w *MaterialBanner) Animation() animation.Animation[float64] { return w.o.Animation }

// MaterialBannerCreateAnimationController creates the animation controller that
// ScaffoldMessengerState.showMaterialBanner uses.
//
// Flutter: material/banner.dart MaterialBanner.createAnimationController
func MaterialBannerCreateAnimationController(vsync scheduler.TickerProvider) *animation.AnimationController {
	return animation.NewAnimationController(vsync, animation.AnimationControllerOpts{
		Duration: foundation.Some(materialBannerTransitionDuration), DebugLabel: "MaterialBanner"})
}

// WithAnimation creates a copy of this banner with the given animation and the
// key, if it has none, of the fallback.
//
// Flutter: material/banner.dart MaterialBanner.withAnimation
func (w *MaterialBanner) WithAnimation(newAnimation animation.Animation[float64], fallbackKey foundation.Key) *MaterialBanner {
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
// Flutter: material/banner.dart MaterialBanner.createState
func (w *MaterialBanner) CreateState() widgets.State { return &materialBannerState{} }

type materialBannerState struct {
	widgets.StateOf[*MaterialBanner]
	wasVisible bool
	handle     animation.StatusListenerHandle
	height     *animation.CurvedAnimation
	slideOut   *animation.CurvedAnimation
}

func (s *materialBannerState) InitState() {
	if a := s.Widget().o.Animation; a != nil {
		s.handle = a.AddStatusListener(s.onAnimationStatusChanged)
	}
	s.setCurvedAnimations()
}

func (s *materialBannerState) DidUpdateWidget(old *MaterialBanner) {
	if s.Widget().o.Animation != old.o.Animation {
		if old.o.Animation != nil {
			old.o.Animation.RemoveStatusListener(s.handle)
		}
		if a := s.Widget().o.Animation; a != nil {
			s.handle = a.AddStatusListener(s.onAnimationStatusChanged)
		}
		s.setCurvedAnimations()
	}
}

func (s *materialBannerState) setCurvedAnimations() {
	if s.height != nil {
		s.height.Dispose()
	}
	if s.slideOut != nil {
		s.slideOut.Dispose()
	}
	if parent := s.Widget().o.Animation; parent != nil {
		s.height = animation.NewCurvedAnimation(parent, animation.Curves.FastOutSlowIn)
		s.slideOut = animation.NewCurvedAnimation(parent, animation.NewThreshold(0.0))
	} else {
		s.height, s.slideOut = nil, nil
	}
}

func (s *materialBannerState) Dispose() {
	if a := s.Widget().o.Animation; a != nil {
		a.RemoveStatusListener(s.handle)
	}
	if s.height != nil {
		s.height.Dispose()
	}
	if s.slideOut != nil {
		s.slideOut.Dispose()
	}
}

func (s *materialBannerState) onAnimationStatusChanged(status animation.AnimationStatus) {
	if status.IsCompleted() {
		if f := s.Widget().o.OnVisible; f != nil && !s.wasVisible {
			f()
		}
		s.wasVisible = true
	}
}

// Build builds the banner.
//
// Flutter: material/banner.dart _MaterialBannerState.build
func (s *materialBannerState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	o := w.o
	widgets.DebugCheckHasMediaQuery(bc)
	accessibleNavigation := widgets.MediaQueryAccessibleNavigationOf(bc)
	if foundation.DebugMode && len(w.actions) == 0 {
		panic(errors.New("MaterialBannerNoActions", "a MaterialBanner needs at least one action"))
	}
	theme := ThemeOf(bc)
	bannerTheme := MaterialBannerThemeOf(bc)
	defaults := materialBannerDefaults(theme)

	isSingleRow := len(w.actions) == 1 && !o.ForceActionsBelow
	padding := o.Padding
	if padding == nil {
		padding = bannerTheme.Padding
	}
	if padding == nil {
		if isSingleRow {
			padding = painting.NewEdgeInsetsDirectionalFromSTEB(16.0, 2.0, 0, 0)
		} else {
			padding = painting.NewEdgeInsetsDirectionalFromSTEB(16.0, 24.0, 16.0, 4.0)
		}
	}
	leadingPadding := o.LeadingPadding
	if leadingPadding == nil {
		leadingPadding = bannerTheme.LeadingPadding
	}
	if leadingPadding == nil {
		leadingPadding = painting.NewEdgeInsetsDirectionalFromSTEB(0, 0, 16.0, 0)
	}

	actionsBar := widgets.NewConstrainedBox(
		rendering.BoxConstraints{MaxWidth: math.Inf(1), MinHeight: o.MinActionBarHeight.OrElse(52.0), MaxHeight: math.Inf(1)},
		widgets.NewPadding(painting.NewEdgeInsetsSymmetric(8, 0),
			widgets.NewAlign(widgets.NewOverflowBar(w.actions, widgets.OverflowBarOpts{OverflowAlignment: o.OverflowAlignment, Spacing: 8}),
				widgets.AlignOpts{Alignment: painting.AlignmentDirectionalCenterEnd})))

	elevation := o.Elevation.OrElse(bannerTheme.Elevation.OrElse(defaults.Elevation.OrElse(0.0)))
	margin := o.Margin
	if margin == nil {
		bottom := 0.0
		if elevation > 0 {
			bottom = 10.0
		}
		margin = painting.EdgeInsets{Bottom: bottom}
	}
	backgroundColor := o.BackgroundColor.OrElse(bannerTheme.BackgroundColor.OrElse(defaults.BackgroundColor.Value()))
	surfaceTintColor := o.SurfaceTintColor
	if surfaceTintColor.IsNone() {
		surfaceTintColor = bannerTheme.SurfaceTintColor
	}
	if surfaceTintColor.IsNone() {
		surfaceTintColor = defaults.SurfaceTintColor
	}
	shadowColor := o.ShadowColor
	if shadowColor.IsNone() {
		shadowColor = bannerTheme.ShadowColor
	}
	dividerColor := o.DividerColor
	if dividerColor.IsNone() {
		dividerColor = bannerTheme.DividerColor
	}
	if dividerColor.IsNone() {
		dividerColor = defaults.DividerColor
	}
	textStyle := o.ContentTextStyle
	if textStyle == nil {
		textStyle = bannerTheme.ContentTextStyle
	}
	if textStyle == nil {
		textStyle = defaults.ContentTextStyle
	}

	var row []widgets.Widget
	if o.Leading != nil {
		row = append(row, widgets.NewPadding(leadingPadding, o.Leading))
	}
	// Set maximum text scale factor to maxContentTextScaleFactor for the content
	// to keep the visual hierarchy the same even with larger font sizes.
	row = append(row, widgets.NewMediaQueryWithClampedTextScaling(
		widgets.NewExpanded(widgets.NewDefaultTextStyle(textStyle, w.content)),
		widgets.MediaQueryClampedTextScalingOpts{MaxScaleFactor: maxContentTextScaleFactor}))
	if isSingleRow {
		row = append(row, widgets.NewMediaQueryWithClampedTextScaling(actionsBar, widgets.MediaQueryClampedTextScalingOpts{MaxScaleFactor: maxContentTextScaleFactor}))
	}
	column := []widgets.Widget{widgets.NewPadding(padding, widgets.NewRow(row))}
	if !isSingleRow {
		column = append(column, actionsBar)
	}
	if elevation == 0 {
		column = append(column, NewDivider(DividerOpts{Height: foundation.Some(0.0), Color: dividerColor}))
	}
	var banner widgets.Widget = widgets.NewPadding(margin, NewMaterial(
		widgets.NewColumn(column, widgets.ColumnOpts{MainAxisSize: rendering.MainAxisSizeMin}),
		MaterialOpts{Elevation: elevation, Color: foundation.Some(backgroundColor), SurfaceTintColor: surfaceTintColor, ShadowColor: shadowColor}))

	// This provides a static banner for backwards compatibility.
	if o.Animation == nil {
		return banner
	}
	banner = widgets.NewSafeArea(banner)
	slideOut := animation.NewTween(ui.Offset{Dx: 0, Dy: -1}, ui.Offset{}).Animate(s.slideOut)
	slid := widgets.Widget(banner)
	if !accessibleNavigation {
		slid = widgets.NewSlideTransition(slideOut, banner)
	}
	banner = widgets.NewSemantics(slid, widgets.SemanticsOpts{
		Container: true, LiveRegion: foundation.Some(true),
		OnDismiss: func() { ScaffoldMessengerOf(bc).RemoveCurrentMaterialBanner(MaterialBannerClosedReasonDismiss) },
	})
	var transition widgets.Widget
	if accessibleNavigation {
		transition = banner
	} else {
		transition = widgets.NewAnimatedBuilder(s.height, func(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
			return widgets.NewAlign(child, widgets.AlignOpts{Alignment: painting.AlignmentDirectionalBottomStart, HeightFactor: foundation.Some(s.height.Value())})
		}, banner)
	}
	return widgets.NewHero(fmt.Sprintf("<MaterialBanner Hero tag - %v>", w.content), widgets.NewClipRect(transition))
}

// materialBannerDefaults returns the defaults of the banner of the theme.
func materialBannerDefaults(theme *ThemeData) MaterialBannerThemeData {
	if theme.UseMaterial3 {
		return materialBannerDefaultsM3(theme)
	}
	return materialBannerDefaultsM2(theme)
}

// materialBannerDefaultsM2 are the Material 2 defaults of the banner.
//
// Flutter: material/banner.dart _BannerDefaultsM2
func materialBannerDefaultsM2(theme *ThemeData) MaterialBannerThemeData {
	return MaterialBannerThemeData{
		Elevation:        foundation.Some(0.0),
		BackgroundColor:  foundation.Some(theme.ColorScheme.Surface()),
		ContentTextStyle: theme.TextTheme.BodyMedium,
	}
}

// materialBannerDefaultsM3 are the Material 3 defaults of the banner (generated
// from the Material token database in Flutter).
//
// Flutter: material/banner.dart _BannerDefaultsM3
func materialBannerDefaultsM3(theme *ThemeData) MaterialBannerThemeData {
	return MaterialBannerThemeData{
		Elevation:        foundation.Some(1.0),
		BackgroundColor:  foundation.Some(theme.ColorScheme.SurfaceContainerLow()),
		SurfaceTintColor: foundation.Some(Colors.Transparent),
		DividerColor:     foundation.Some(theme.ColorScheme.OutlineVariant()),
		ContentTextStyle: theme.TextTheme.BodyMedium,
	}
}
