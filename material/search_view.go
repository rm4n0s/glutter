// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"math"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// searchViewContent is the content of the search view: the text field, the divider
// and the suggestions, in the rectangle that the transition gives.
//
// Flutter: material/search_anchor.dart _ViewContent
type searchViewContent struct {
	widgets.StatefulWidget
	route        *searchViewRoute
	animation    animation.Animation[float64]
	topPadding   float64
	viewMaxWidth float64
	viewRect     ui.Rect
}

func (w *searchViewContent) CreateState() widgets.State { return &searchViewContentState{} }

type searchViewContentState struct {
	widgets.StateOf[*searchViewContent]
	screenSize                foundation.Opt[ui.Size]
	viewRect                  ui.Rect
	viewIconsFadeCurve        *animation.CurvedAnimation
	viewDividerFadeCurve      *animation.CurvedAnimation
	viewListFadeOnIntervalCrv *animation.CurvedAnimation
	controller                *SearchController
	controllerHandle          foundation.ListenerHandle
	result                    []widgets.Widget
	searchValue               foundation.Opt[string]
	pending                   int
}

func (s *searchViewContentState) InitState() {
	w := s.Widget()
	s.viewRect = w.viewRect
	s.controller = w.route.searchController
	s.controllerHandle = s.controller.AddListener(s.updateSuggestions)
	s.setupAnimations()
}

func (s *searchViewContentState) DidUpdateWidget(old *searchViewContent) {
	if s.Widget().viewRect != old.viewRect {
		s.SetState(func() { s.viewRect = s.Widget().viewRect })
	}
}

// DidChangeDependencies reads the size of the screen, and asks for the suggestions
// after the frame when the text is not the one that they were made for.
//
// Deviation: Flutter asks from a zero-length Timer; the request waits for the end of
// the frame instead (SPEC §12.10).
//
// Flutter: material/search_anchor.dart _ViewContentState.didChangeDependencies
func (s *searchViewContentState) DidChangeDependencies() {
	updated := widgets.MediaQuerySizeOf(s.Context())
	if old, ok := s.screenSize.Get(); !ok || old != updated {
		s.screenSize = foundation.Some(updated)
		if s.Widget().route.showFullScreenView {
			s.viewRect = ui.Offset{}.And(updated)
		}
	}
	if s.searchValue != foundation.Some(s.controller.Text()) {
		s.pending++
		ticket := s.pending
		widgets.WidgetsBindingOf(s.Context()).Scheduler().AddPostFrameCallback(func(time.Duration) {
			if ticket != s.pending || !s.Mounted() {
				return
			}
			s.searchValue = foundation.Some(s.controller.Text())
			s.requestSuggestions(ticket)
		})
	}
}

func (s *searchViewContentState) requestSuggestions(ticket int) {
	s.Widget().route.suggestionsBuilder(s.Context(), s.controller, func(suggestions []widgets.Widget) {
		if ticket != s.pending || !s.Mounted() {
			return
		}
		s.SetState(func() { s.result = suggestions })
	})
}

func (s *searchViewContentState) Dispose() {
	s.controller.RemoveListener(s.controllerHandle)
	s.viewIconsFadeCurve.Dispose()
	s.viewDividerFadeCurve.Dispose()
	s.viewListFadeOnIntervalCrv.Dispose()
	s.pending++
}

func (s *searchViewContentState) setupAnimations() {
	a := s.Widget().animation
	s.viewIconsFadeCurve = animation.NewCurvedAnimation(a, searchViewIconsFadeOnInterval, searchViewIconsFadeOnInterval.Flipped())
	s.viewDividerFadeCurve = animation.NewCurvedAnimation(a, searchViewDividerFadeOnInterval, searchViewFadeOnInterval.Flipped())
	s.viewListFadeOnIntervalCrv = animation.NewCurvedAnimation(a, searchViewListFadeOnInterval, searchViewListFadeOnInterval.Flipped())
}

// updateSuggestions asks for the suggestions when the text changed.
//
// Flutter: material/search_anchor.dart _ViewContentState.updateSuggestions
func (s *searchViewContentState) updateSuggestions() {
	if s.searchValue != foundation.Some(s.controller.Text()) {
		s.searchValue = foundation.Some(s.controller.Text())
		s.pending++
		s.requestSuggestions(s.pending)
	}
}

// Build builds the view.
//
// Flutter: material/search_anchor.dart _ViewContentState.build
func (s *searchViewContentState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	o := w.route.o
	showFull := w.route.showFullScreenView
	defaultLeading := NewBackButton(ActionButtonOpts{
		Style:     NewButtonStyle(ButtonStyleOpts{TapTargetSize: MaterialTapTargetSizeShrinkWrap}),
		OnPressed: func() { widgets.NavigatorOf(bc).Pop() },
	})
	var defaultTrailing []widgets.Widget
	if s.controller.Text() != "" {
		defaultTrailing = []widgets.Widget{NewIconButton(func() { s.controller.Clear() }, widgets.NewIcon(Icons.Close()),
			IconButtonOpts{Tooltip: MaterialLocalizationsOf(bc).ClearButtonTooltip()})}
	}

	viewDefaults := searchViewDefaultsM3(bc, showFull)
	viewTheme := SearchViewThemeOf(bc)

	effectiveBackgroundColor := firstOptColor(o.ViewBackgroundColor, viewTheme.BackgroundColor, viewDefaults.BackgroundColor).Value()
	effectiveSurfaceTint := firstOptColor(o.ViewSurfaceTintColor, viewTheme.SurfaceTintColor, viewDefaults.SurfaceTintColor).Value()
	effectiveElevation := firstOptDouble(o.ViewElevation, viewTheme.Elevation, viewDefaults.Elevation).Value()
	effectiveSide := firstSide(o.ViewSide, viewTheme.Side, viewDefaults.Side)
	var effectiveShape painting.OutlinedBorder
	for _, shape := range []painting.OutlinedBorder{o.ViewShape, viewTheme.Shape, viewDefaults.Shape} {
		if !isNilValue(shape) {
			effectiveShape = shape
			break
		}
	}
	effectiveShape = outlinedWithSide(effectiveShape, effectiveSide)
	// TODO(step 13.9): the color of the DividerTheme goes between the color of the
	// view theme and the default.
	effectiveDividerColor := firstOptColor(o.DividerColor, viewTheme.DividerColor, viewDefaults.DividerColor).Value()
	effectiveHeaderHeight := firstOptDouble(o.HeaderHeight, viewTheme.HeaderHeight)
	var headerConstraints foundation.Opt[rendering.BoxConstraints]
	if h, ok := effectiveHeaderHeight.Get(); ok {
		headerConstraints = foundation.Some(rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{Height: foundation.Some(h)}))
	}
	effectiveTextStyle := firstTextStyle(o.HeaderTextStyle, viewTheme.HeaderTextStyle, viewDefaults.HeaderTextStyle)
	effectiveHintStyle := firstTextStyle(o.HeaderHintStyle, viewTheme.HeaderHintStyle, o.HeaderTextStyle, viewTheme.HeaderTextStyle, viewDefaults.HeaderHintStyle)
	effectivePadding := firstInsets(o.ViewPadding, viewTheme.Padding, viewDefaults.Padding)
	effectiveBarPadding := firstInsets(o.ViewBarPadding, viewTheme.BarPadding, viewDefaults.BarPadding)

	effectiveConstraints := firstOpt(o.ViewConstraints, viewTheme.Constraints, viewDefaults.Constraints).Value()
	minWidth := math.Min(effectiveConstraints.MinWidth, s.viewRect.Width())
	minHeight := math.Min(effectiveConstraints.MinHeight, s.viewRect.Height())
	effectiveShrinkWrap := firstOpt(o.ShrinkWrap, viewTheme.ShrinkWrap, viewDefaults.ShrinkWrap).Value()

	viewDivider := tempDivider(bc, tempDividerOpts{height: 1, color: foundation.Some(effectiveDividerColor)})

	leading := o.ViewLeading
	if leading == nil {
		leading = defaultLeading
	}
	trailing := o.ViewTrailing
	if trailing == nil {
		trailing = defaultTrailing
	}
	barConstraints := headerConstraints
	if barConstraints.IsNone() && showFull {
		barConstraints = foundation.Some(rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{MinHeight: foundation.Some(searchViewFullScreenBarHeight)}))
	}
	all := func(c ui.Color) widgets.WidgetStateProperty[foundation.Opt[ui.Color]] {
		return widgets.NewWidgetStatePropertyAll(foundation.Some(c))
	}
	searchBar := NewSearchBar(SearchBarOpts{
		AutoFocus: true, Constraints: barConstraints,
		Padding: widgets.NewWidgetStatePropertyAll(effectiveBarPadding), Leading: leading, Trailing: trailing, HintText: o.ViewHintText,
		BackgroundColor: all(Colors.Transparent), OverlayColor: all(Colors.Transparent),
		Elevation: widgets.NewWidgetStatePropertyAll(foundation.Some(0.0)),
		TextStyle: widgets.NewWidgetStatePropertyAll(effectiveTextStyle), HintStyle: widgets.NewWidgetStatePropertyAll(effectiveHintStyle),
		Controller: s.controller.TextEditingController,
		OnChanged: func(value string) {
			if o.ViewOnChanged != nil {
				o.ViewOnChanged(value)
			}
			s.updateSuggestions()
		},
		OnSubmitted: o.ViewOnSubmitted, TextCapitalization: o.TextCapitalization, TextInputAction: o.TextInputAction,
		KeyboardType: o.KeyboardType, SmartDashesType: o.SmartDashesType, SmartQuotesType: o.SmartQuotesType,
	})

	children := []widgets.Widget{
		widgets.NewPadding(painting.NewEdgeInsetsFromLTRB(0, w.topPadding, 0, 0),
			widgets.NewSafeArea(searchBar, widgets.SafeAreaOpts{Top: foundation.Some(false), Bottom: foundation.Some(false)})),
	}
	if !effectiveShrinkWrap || minHeight > 0 || showFull || len(s.result) > 0 {
		fit := rendering.FlexFitTight
		if effectiveShrinkWrap && !showFull {
			fit = rendering.FlexFitLoose
		}
		var list widgets.Widget
		if o.ViewBuilder == nil {
			list = widgets.NewMediaQueryRemovePadding(bc, widgets.RemoveSides{Top: true},
				widgets.NewListView(append([]widgets.Widget(nil), s.result...), widgets.ListViewOpts{
					Padding:    painting.NewEdgeInsetsFromLTRB(0, 0, 0, widgets.MediaQueryViewInsetsOf(bc).Bottom),
					ShrinkWrap: effectiveShrinkWrap}))
		} else {
			list = o.ViewBuilder(s.result)
		}
		children = append(children,
			widgets.NewFadeTransition(s.viewDividerFadeCurve, viewDivider),
			widgets.NewFlexible(widgets.NewFadeTransition(s.viewListFadeOnIntervalCrv, list), widgets.FlexibleOpts{Fit: fit}))
	}

	var padding painting.EdgeInsetsGeometry = painting.EdgeInsetsZero
	if !showFull && effectivePadding != nil {
		padding = effectivePadding
	}
	var shape painting.ShapeBorder
	if !isNilValue(effectiveShape) {
		shape = effectiveShape
	}
	return widgets.NewAlign(widgets.NewTransformTranslate(s.viewRect.TopLeft(), widgets.NewConstrainedBox(
		rendering.BoxConstraints{MinWidth: minWidth, MaxWidth: s.viewRect.Width(), MinHeight: minHeight, MaxHeight: s.viewRect.Height()},
		widgets.NewPadding(padding, NewMaterial(
			widgets.NewOverflowBox(widgets.NewFadeTransition(s.viewIconsFadeCurve, widgets.NewColumn(children, widgets.ColumnOpts{
				MainAxisSize: rendering.MainAxisSizeMin, CrossAxisAlignment: rendering.CrossAxisAlignmentStretch})),
				widgets.OverflowBoxOpts{Alignment: painting.AlignmentTopLeft, MaxWidth: foundation.Some(math.Min(w.viewMaxWidth, s.screenSize.Value().Width)),
					MinWidth: foundation.Some(0.0), Fit: rendering.OverflowBoxFitDeferToChild}),
			MaterialOpts{ClipBehavior: ui.ClipAntiAlias, Shape: shape, Color: foundation.Some(effectiveBackgroundColor),
				SurfaceTintColor: foundation.Some(effectiveSurfaceTint), Elevation: effectiveElevation})))),
		widgets.AlignOpts{Alignment: painting.AlignmentTopLeft})
}
