// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"math"
	"slices"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// CollapseMode is how the background of a FlexibleSpaceBar behaves when the
// app bar collapses.
//
// Flutter: material/flexible_space_bar.dart CollapseMode
type CollapseMode int

const (
	// CollapseModeUnspecified means the default of the widget: parallax.
	CollapseModeUnspecified CollapseMode = iota
	// CollapseModeParallax scrolls the background at a different rate than the
	// content, for a parallax effect.
	//
	// Flutter: material/flexible_space_bar.dart CollapseMode.parallax
	CollapseModeParallax
	// CollapseModePin keeps the background at the same position while the app
	// bar collapses.
	//
	// Flutter: material/flexible_space_bar.dart CollapseMode.pin
	CollapseModePin
	// CollapseModeNone does not move the background.
	//
	// Flutter: material/flexible_space_bar.dart CollapseMode.none
	CollapseModeNone
)

// StretchMode is what a FlexibleSpaceBar does when the user overscrolls the
// scroll view that holds it.
//
// Flutter: material/flexible_space_bar.dart StretchMode
type StretchMode int

const (
	// StretchModeZoomBackground makes the background grow with the stretch.
	//
	// Flutter: material/flexible_space_bar.dart StretchMode.zoomBackground
	StretchModeZoomBackground StretchMode = iota + 1
	// StretchModeBlurBackground blurs the background as it stretches.
	//
	// Flutter: material/flexible_space_bar.dart StretchMode.blurBackground
	StretchModeBlurBackground
	// StretchModeFadeTitle fades the title out as it stretches.
	//
	// Flutter: material/flexible_space_bar.dart StretchMode.fadeTitle
	StretchModeFadeTitle
)

// FlexibleSpaceBarOpts are the optional parameters of NewFlexibleSpaceBar.
//
// Flutter: material/flexible_space_bar.dart FlexibleSpaceBar.new
type FlexibleSpaceBarOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Title is the primary contents of the flexible space bar when expanded.
	Title widgets.Widget
	// Background is shown behind the title, and gets the fade and the parallax
	// of the collapse.
	Background widgets.Widget
	// CenterTitle says whether the title is centered; the default is true on iOS
	// and macOS only.
	CenterTitle foundation.Opt[bool]
	// TitlePadding is the padding of the title; the default is 72 at the start
	// (when the title is not centered) and 16 at the bottom.
	TitlePadding painting.EdgeInsetsGeometry
	// CollapseMode is how the background collapses; the default is parallax.
	CollapseMode CollapseMode
	// StretchModes are the stretch effects; nil is the default,
	// {StretchModeZoomBackground}, and an empty non-nil slice is none.
	StretchModes []StretchMode
	// ExpandedTitleScale is the scale of the title when expanded; it must be at
	// least 1, and the default is 1.5.
	ExpandedTitleScale foundation.Opt[float64]
}

// FlexibleSpaceBar is the part of a Material Design AppBar that expands, collapses
// and stretches. Most commonly used in the AppBar.flexibleSpace field, it
// receives its extents from a FlexibleSpaceBarSettings above it.
//
// Flutter: material/flexible_space_bar.dart FlexibleSpaceBar
type FlexibleSpaceBar struct {
	widgets.StatefulWidget
	o FlexibleSpaceBarOpts
}

// NewFlexibleSpaceBar creates the widget. In debug builds it panics with the
// tag FlexibleSpaceBarExpandedTitleScale when the expanded title scale is below
// 1.
//
// Flutter: material/flexible_space_bar.dart FlexibleSpaceBar.new
func NewFlexibleSpaceBar(opts ...FlexibleSpaceBarOpts) *FlexibleSpaceBar {
	o := firstOpts(opts)
	if o.CollapseMode == CollapseModeUnspecified {
		o.CollapseMode = CollapseModeParallax
	}
	if o.StretchModes == nil {
		o.StretchModes = []StretchMode{StretchModeZoomBackground}
	}
	if foundation.DebugMode && o.ExpandedTitleScale.OrElse(1.5) < 1 {
		panic(errors.New(tagFlexibleSpaceBarExpandedTitleScale, "the expanded title scale must be at least 1", "scale", o.ExpandedTitleScale.Value()))
	}
	return &FlexibleSpaceBar{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), o: o}
}

// Title is the primary contents of the flexible space bar when expanded.
//
// Flutter: material/flexible_space_bar.dart FlexibleSpaceBar.title
func (w *FlexibleSpaceBar) Title() widgets.Widget { return w.o.Title }

// Background is shown behind the title.
//
// Flutter: material/flexible_space_bar.dart FlexibleSpaceBar.background
func (w *FlexibleSpaceBar) Background() widgets.Widget { return w.o.Background }

// CenterTitle says whether the title is centered, when it was set.
//
// Flutter: material/flexible_space_bar.dart FlexibleSpaceBar.centerTitle
func (w *FlexibleSpaceBar) CenterTitle() foundation.Opt[bool] { return w.o.CenterTitle }

// CollapseMode is how the background collapses.
//
// Flutter: material/flexible_space_bar.dart FlexibleSpaceBar.collapseMode
func (w *FlexibleSpaceBar) CollapseMode() CollapseMode { return w.o.CollapseMode }

// StretchModes are the stretch effects.
//
// Flutter: material/flexible_space_bar.dart FlexibleSpaceBar.stretchModes
func (w *FlexibleSpaceBar) StretchModes() []StretchMode { return w.o.StretchModes }

// TitlePadding is the padding of the title, or nil for the default.
//
// Flutter: material/flexible_space_bar.dart FlexibleSpaceBar.titlePadding
func (w *FlexibleSpaceBar) TitlePadding() painting.EdgeInsetsGeometry { return w.o.TitlePadding }

// ExpandedTitleScale is the scale of the title when expanded.
//
// Flutter: material/flexible_space_bar.dart FlexibleSpaceBar.expandedTitleScale
func (w *FlexibleSpaceBar) ExpandedTitleScale() float64 { return w.o.ExpandedTitleScale.OrElse(1.5) }

// FlexibleSpaceBarSettingsOpts are the optional parameters of
// FlexibleSpaceBarCreateSettings.
//
// Flutter: material/flexible_space_bar.dart FlexibleSpaceBar.createSettings
type FlexibleSpaceBarSettingsOpts struct {
	// ToolbarOpacity is the opacity of the toolbar; the default is 1.
	ToolbarOpacity foundation.Opt[float64]
	// MinExtent is the smallest extent; the default is the current extent.
	MinExtent foundation.Opt[float64]
	// MaxExtent is the biggest extent; the default is the current extent.
	MaxExtent foundation.Opt[float64]
	// IsScrolledUnder says whether content is scrolled under the bar.
	IsScrolledUnder foundation.Opt[bool]
	// HasLeading says whether the app bar has a leading widget.
	HasLeading foundation.Opt[bool]
}

// FlexibleSpaceBarCreateSettings creates the FlexibleSpaceBarSettings that a
// FlexibleSpaceBar below needs, with the given current extent.
//
// Flutter: material/flexible_space_bar.dart FlexibleSpaceBar.createSettings
func FlexibleSpaceBarCreateSettings(currentExtent float64, child widgets.Widget, opts ...FlexibleSpaceBarSettingsOpts) widgets.Widget {
	o := firstOpts(opts)
	return NewFlexibleSpaceBarSettings(o.ToolbarOpacity.OrElse(1.0), o.MinExtent.OrElse(currentExtent), o.MaxExtent.OrElse(currentExtent),
		currentExtent, child, FlexibleSpaceBarSettingsWidgetOpts{IsScrolledUnder: o.IsScrolledUnder, HasLeading: o.HasLeading})
}

// CreateState creates the state.
//
// Flutter: material/flexible_space_bar.dart FlexibleSpaceBar.createState
func (w *FlexibleSpaceBar) CreateState() widgets.State { return &flexibleSpaceBarState{} }

type flexibleSpaceBarState struct {
	widgets.StateOf[*FlexibleSpaceBar]
}

func (s *flexibleSpaceBarState) effectiveCenterTitle(theme *ThemeData) bool {
	if c, ok := s.Widget().o.CenterTitle.Get(); ok {
		return c
	}
	switch theme.Platform {
	case foundation.TargetPlatformIOS, foundation.TargetPlatformMacOS:
		return true
	}
	return false
}

func (s *flexibleSpaceBarState) titleAlignment(bc widgets.BuildContext, effectiveCenterTitle bool) painting.Alignment {
	if effectiveCenterTitle {
		return painting.AlignmentBottomCenter
	}
	if widgets.DirectionalityOf(bc) == ui.TextDirectionRTL {
		return painting.AlignmentBottomRight
	}
	return painting.AlignmentBottomLeft
}

func (s *flexibleSpaceBarState) collapsePadding(t float64, settings *FlexibleSpaceBarSettings) float64 {
	switch s.Widget().o.CollapseMode {
	case CollapseModePin:
		return -(settings.maxExtent - settings.currentExtent)
	case CollapseModeNone:
		return 0
	}
	// CollapseModeParallax.
	deltaExtent := settings.maxExtent - settings.minExtent
	return -ui.LerpDouble(0, deltaExtent/4.0, t)
}

// Build builds the stack of the background and the title.
//
// Flutter: material/flexible_space_bar.dart _FlexibleSpaceBarState.build
func (s *flexibleSpaceBarState) Build(bc widgets.BuildContext) widgets.Widget {
	return widgets.NewLayoutBuilder(func(bc widgets.BuildContext, constraints rendering.BoxConstraints) widgets.Widget {
		w := s.Widget()
		settings, ok := widgets.DependOn[*FlexibleSpaceBarSettings](bc)
		if !ok {
			panic(errors.New(tagFlexibleSpaceBarNoSettings, "a FlexibleSpaceBar needs a FlexibleSpaceBarSettings above it (FlexibleSpaceBarCreateSettings makes one)"))
		}
		var children []widgets.Widget
		deltaExtent := settings.maxExtent - settings.minExtent

		// 0.0 -> Expanded
		// 1.0 -> Collapsed to toolbar
		t := ui.ClampDouble(1.0-(settings.currentExtent-settings.minExtent)/deltaExtent, 0.0, 1.0)

		// background
		if w.o.Background != nil {
			fadeStart := math.Max(0.0, 1.0-ToolbarHeight/deltaExtent)
			const fadeEnd = 1.0
			// If the min and max extent are the same, the app bar cannot collapse
			// and the content should be visible, so opacity = 1.
			opacity := 1.0
			if settings.maxExtent != settings.minExtent {
				opacity = 1.0 - animation.NewInterval(fadeStart, fadeEnd).Transform(t)
			}
			height := settings.maxExtent

			// StretchMode.zoomBackground
			if slices.Contains(w.o.StretchModes, StretchModeZoomBackground) && constraints.MaxHeight > height {
				height = constraints.MaxHeight
			}
			topPadding := s.collapsePadding(t, settings)
			children = append(children, widgets.NewPositioned(
				// iOS is relying on this semantics node to correctly traverse
				// through the app bar when it is collapsed.
				newFlexibleSpaceHeaderOpacity(opacity, true, w.o.Background),
				widgets.PositionedOpts{Top: foundation.Some(topPadding), Left: foundation.Some(0.0), Right: foundation.Some(0.0), Height: foundation.Some(height)},
			))

			// StretchMode.blurBackground
			if slices.Contains(w.o.StretchModes, StretchModeBlurBackground) && constraints.MaxHeight > settings.maxExtent {
				blurAmount := (constraints.MaxHeight - settings.maxExtent) / 10
				children = append(children, widgets.NewPositionedFill(
					widgets.NewBackdropFilter(ui.NewImageFilterBlur(ui.ImageFilterBlurOpts{SigmaX: blurAmount, SigmaY: blurAmount}),
						widgets.NewColoredBox(Colors.Transparent, nil)),
				))
			}
		}

		// title
		if w.o.Title != nil {
			theme := ThemeOf(bc)

			title := w.o.Title
			switch theme.Platform {
			case foundation.TargetPlatformIOS, foundation.TargetPlatformMacOS:
			default:
				title = widgets.NewSemantics(title, widgets.SemanticsOpts{NamesRoute: foundation.Some(true)})
			}

			// StretchMode.fadeTitle
			if slices.Contains(w.o.StretchModes, StretchModeFadeTitle) && constraints.MaxHeight > settings.maxExtent {
				stretchOpacity := 1 - ui.ClampDouble((constraints.MaxHeight-settings.maxExtent)/100, 0.0, 1.0)
				title = widgets.NewOpacity(stretchOpacity, title)
			}

			opacity := settings.toolbarOpacity
			if opacity > 0.0 {
				titleStyle := theme.PrimaryTextTheme.TitleLarge
				if theme.UseMaterial3 {
					titleStyle = theme.TextTheme.TitleLarge
				}
				titleStyle = titleStyle.CopyWith(painting.TextStyleCopy{Color: foundation.Some(titleStyle.Color.Value().WithOpacity(opacity))})
				effectiveCenterTitle := s.effectiveCenterTitle(theme)
				leadingPadding := 72.0
				if !settings.hasLeading.OrElse(true) {
					leadingPadding = 0
				}
				padding := w.o.TitlePadding
				if padding == nil {
					start := leadingPadding
					if effectiveCenterTitle {
						start = 0
					}
					padding = painting.NewEdgeInsetsDirectionalFromSTEB(start, 0, 0, 16.0)
				}
				scaleValue := ui.LerpDouble(w.ExpandedTitleScale(), 1.0, t)
				scaleTransform := ui.NewMatrix4Identity()
				scaleTransform.Scale(float32(scaleValue), float32(scaleValue), 1.0)
				titleAlignment := s.titleAlignment(bc, effectiveCenterTitle)
				children = append(children, widgets.NewPadding(padding,
					widgets.NewTransform(scaleTransform,
						widgets.NewAlign(
							widgets.NewDefaultTextStyle(titleStyle,
								widgets.NewLayoutBuilder(func(_ widgets.BuildContext, constraints rendering.BoxConstraints) widgets.Widget {
									return widgets.NewSizedBox(
										widgets.NewAlign(title, widgets.AlignOpts{Alignment: titleAlignment}),
										widgets.SizedBoxOpts{Width: foundation.Some(constraints.MaxWidth / scaleValue)})
								})),
							widgets.AlignOpts{Alignment: titleAlignment}),
						widgets.TransformOpts{Alignment: titleAlignment})))
			}
		}

		return widgets.NewClipRect(widgets.NewStack(children))
	})
}

// FlexibleSpaceBarSettingsWidgetOpts are the optional parameters of
// NewFlexibleSpaceBarSettings.
//
// Flutter: material/flexible_space_bar.dart FlexibleSpaceBarSettings.new
type FlexibleSpaceBarSettingsWidgetOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// IsScrolledUnder says whether content is scrolled under the bar.
	IsScrolledUnder foundation.Opt[bool]
	// HasLeading says whether the app bar has a leading widget.
	HasLeading foundation.Opt[bool]
}

// FlexibleSpaceBarSettings provides the sizes and the opacity that a
// FlexibleSpaceBar needs. A SliverAppBar and a Scaffold's AppBar make one.
//
// Flutter: material/flexible_space_bar.dart FlexibleSpaceBarSettings
type FlexibleSpaceBarSettings struct {
	widgets.InheritedWidget
	toolbarOpacity  float64
	minExtent       float64
	maxExtent       float64
	currentExtent   float64
	isScrolledUnder foundation.Opt[bool]
	hasLeading      foundation.Opt[bool]
}

// NewFlexibleSpaceBarSettings creates the settings. In debug builds it panics
// with the tag FlexibleSpaceBarSettingsInvalid unless 0 <= minExtent <=
// currentExtent <= maxExtent and toolbarOpacity >= 0.
//
// Flutter: material/flexible_space_bar.dart FlexibleSpaceBarSettings.new
func NewFlexibleSpaceBarSettings(toolbarOpacity, minExtent, maxExtent, currentExtent float64, child widgets.Widget, opts ...FlexibleSpaceBarSettingsWidgetOpts) *FlexibleSpaceBarSettings {
	o := firstOpts(opts)
	if foundation.DebugMode && !(minExtent >= 0 && maxExtent >= 0 && currentExtent >= 0 && toolbarOpacity >= 0.0 &&
		minExtent <= maxExtent && minExtent <= currentExtent && currentExtent <= maxExtent) {
		panic(errors.New(tagFlexibleSpaceBarSettingsInvalid, "the extents of FlexibleSpaceBarSettings must satisfy 0 <= minExtent <= currentExtent <= maxExtent, and the toolbar opacity must not be negative",
			"toolbarOpacity", toolbarOpacity, "minExtent", minExtent, "maxExtent", maxExtent, "currentExtent", currentExtent))
	}
	return &FlexibleSpaceBarSettings{InheritedWidget: widgets.NewInheritedWidgetBase(child, widgets.WidgetOpts{Key: o.Key}),
		toolbarOpacity: toolbarOpacity, minExtent: minExtent, maxExtent: maxExtent, currentExtent: currentExtent,
		isScrolledUnder: o.IsScrolledUnder, hasLeading: o.HasLeading}
}

// ToolbarOpacity is the opacity of the toolbar.
//
// Flutter: material/flexible_space_bar.dart FlexibleSpaceBarSettings.toolbarOpacity
func (w *FlexibleSpaceBarSettings) ToolbarOpacity() float64 { return w.toolbarOpacity }

// MinExtent is the minimum extent of the app bar.
//
// Flutter: material/flexible_space_bar.dart FlexibleSpaceBarSettings.minExtent
func (w *FlexibleSpaceBarSettings) MinExtent() float64 { return w.minExtent }

// MaxExtent is the maximum extent of the app bar.
//
// Flutter: material/flexible_space_bar.dart FlexibleSpaceBarSettings.maxExtent
func (w *FlexibleSpaceBarSettings) MaxExtent() float64 { return w.maxExtent }

// CurrentExtent is the current extent of the app bar.
//
// Flutter: material/flexible_space_bar.dart FlexibleSpaceBarSettings.currentExtent
func (w *FlexibleSpaceBarSettings) CurrentExtent() float64 { return w.currentExtent }

// IsScrolledUnder says whether content is scrolled under the app bar, when set.
//
// Flutter: material/flexible_space_bar.dart FlexibleSpaceBarSettings.isScrolledUnder
func (w *FlexibleSpaceBarSettings) IsScrolledUnder() foundation.Opt[bool] { return w.isScrolledUnder }

// HasLeading says whether the app bar has a leading widget, when set.
//
// Flutter: material/flexible_space_bar.dart FlexibleSpaceBarSettings.hasLeading
func (w *FlexibleSpaceBarSettings) HasLeading() foundation.Opt[bool] { return w.hasLeading }

// UpdateShouldNotify is whether a field changed.
//
// Flutter: material/flexible_space_bar.dart FlexibleSpaceBarSettings.updateShouldNotify
func (w *FlexibleSpaceBarSettings) UpdateShouldNotify(old widgets.Widget) bool {
	o := old.(*FlexibleSpaceBarSettings)
	return w.toolbarOpacity != o.toolbarOpacity || w.minExtent != o.minExtent || w.maxExtent != o.maxExtent ||
		w.currentExtent != o.currentExtent || w.isScrolledUnder != o.isScrolledUnder || w.hasLeading != o.hasLeading
}

// flexibleSpaceHeaderOpacity is a SingleChildRenderObjectWidget that fades the
// background of a FlexibleSpaceBar. We need the child widget to repaint, however
// both the opacity and potentially the background can be constant, which would
// not lead to repainting (https://github.com/flutter/flutter/issues/127836).
//
// Flutter: material/flexible_space_bar.dart _FlexibleSpaceHeaderOpacity
type flexibleSpaceHeaderOpacity struct {
	widgets.SingleChildRenderObjectWidget
	opacity                float64
	alwaysIncludeSemantics bool
}

func newFlexibleSpaceHeaderOpacity(opacity float64, alwaysIncludeSemantics bool, child widgets.Widget) *flexibleSpaceHeaderOpacity {
	return &flexibleSpaceHeaderOpacity{widgets.NewSingleChildRenderObjectWidgetBase(child), opacity, alwaysIncludeSemantics}
}

func (w *flexibleSpaceHeaderOpacity) CreateRenderObject(widgets.BuildContext) rendering.RenderObject {
	return newRenderFlexibleSpaceHeaderOpacity(w.opacity, w.alwaysIncludeSemantics)
}

func (w *flexibleSpaceHeaderOpacity) UpdateRenderObject(_ widgets.BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*renderFlexibleSpaceHeaderOpacity)
	r.SetAlwaysIncludeSemantics(w.alwaysIncludeSemantics)
	r.SetOpacity(w.opacity)
}

// renderFlexibleSpaceHeaderOpacity is a RenderOpacity that is not a repaint
// boundary and always pushes its own opacity layer.
//
// Flutter: material/flexible_space_bar.dart _RenderFlexibleSpaceHeaderOpacity
type renderFlexibleSpaceHeaderOpacity struct {
	rendering.RenderOpacity
}

func newRenderFlexibleSpaceHeaderOpacity(opacity float64, alwaysIncludeSemantics bool) *renderFlexibleSpaceHeaderOpacity {
	r := &renderFlexibleSpaceHeaderOpacity{}
	rendering.InitRenderOpacity(&r.RenderOpacity, r, nil, rendering.RenderOpacityOpts{Opacity: foundation.Some(opacity), AlwaysIncludeSemantics: alwaysIncludeSemantics})
	return r
}

// IsRepaintBoundary is false.
func (r *renderFlexibleSpaceHeaderOpacity) IsRepaintBoundary() bool { return false }

// Paint pushes an opacity layer for the child, or removes the layer when the
// child is invisible.
func (r *renderFlexibleSpaceHeaderOpacity) Paint(context *rendering.PaintingContext, offset ui.Offset) {
	if r.Child() == nil {
		return
	}
	alpha := int(math.Round(r.Opacity() * 255))
	if alpha <= 0 {
		r.SetLayer(nil)
		return
	}
	old, _ := r.Layer().(*rendering.OpacityLayer)
	layer := context.PushOpacity(offset, alpha, r.RenderProxyBox.Paint, old)
	r.SetLayer(layer)
	if foundation.DebugMode {
		layer.DebugCreator = r.DebugCreator
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagFlexibleSpaceBarExpandedTitleScale = "FlexibleSpaceBarExpandedTitleScale"
	tagFlexibleSpaceBarNoSettings         = "FlexibleSpaceBarNoSettings"
	tagFlexibleSpaceBarSettingsInvalid    = "FlexibleSpaceBarSettingsInvalid"
)
