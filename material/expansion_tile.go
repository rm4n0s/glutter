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
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// ExpansionTileOpts are the optional parameters of NewExpansionTile.
//
// Flutter: material/expansion_tile.dart ExpansionTile.new
type ExpansionTileOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Leading is a widget to display before the title.
	Leading widgets.Widget
	// Subtitle is additional content displayed below the title.
	Subtitle widgets.Widget
	// OnExpansionChanged is called when the tile expands or collapses.
	OnExpansionChanged func(bool)
	// Children are the widgets that are shown when the tile is expanded.
	Children []widgets.Widget
	// Trailing is a widget to display after the title; the default is a rotating
	// arrow.
	Trailing widgets.Widget
	// NoShowTrailingIcon leaves out the trailing widget: Flutter's
	// showTrailingIcon is false.
	NoShowTrailingIcon bool
	// InitiallyExpanded is whether the tile starts expanded.
	InitiallyExpanded bool
	// MaintainState keeps the children in the tree while the tile is collapsed.
	MaintainState bool
	// TilePadding is the padding of the tile's header.
	TilePadding painting.EdgeInsetsGeometry
	// ExpandedCrossAxisAlignment is the horizontal alignment of the children;
	// the default is center.
	ExpandedCrossAxisAlignment rendering.CrossAxisAlignment
	// ExpandedAlignment is the alignment of the children in the tile.
	ExpandedAlignment painting.AlignmentGeometry
	// ChildrenPadding is the padding around the children.
	ChildrenPadding painting.EdgeInsetsGeometry
	// BackgroundColor is the color of the tile when it is expanded.
	BackgroundColor foundation.Opt[ui.Color]
	// CollapsedBackgroundColor is the color of the tile when it is collapsed.
	CollapsedBackgroundColor foundation.Opt[ui.Color]
	// TextColor is the color of the title when the tile is expanded.
	TextColor foundation.Opt[ui.Color]
	// CollapsedTextColor is the color of the title when the tile is collapsed.
	CollapsedTextColor foundation.Opt[ui.Color]
	// IconColor is the color of the icons when the tile is expanded.
	IconColor foundation.Opt[ui.Color]
	// CollapsedIconColor is the color of the icons when the tile is collapsed.
	CollapsedIconColor foundation.Opt[ui.Color]
	// Shape is the shape of the tile when it is expanded.
	Shape painting.ShapeBorder
	// CollapsedShape is the shape of the tile when it is collapsed.
	CollapsedShape painting.ShapeBorder
	// ClipBehavior is how the tile is clipped; the default is antiAlias.
	ClipBehavior ui.Clip
	// ControlAffinity is where to put the arrow: leading, or trailing (the
	// default; platform is trailing).
	ControlAffinity ListTileControlAffinity
	// Controller expands and collapses the tile from outside.
	Controller *widgets.ExpansibleController
	// Dense is whether the tile is part of a vertically dense list.
	Dense foundation.Opt[bool]
	// SplashColor is the splash color of the header.
	SplashColor foundation.Opt[ui.Color]
	// VisualDensity defines how compact the header's layout is.
	VisualDensity foundation.Opt[VisualDensity]
	// MinTileHeight is the minimum height of the header.
	MinTileHeight foundation.Opt[float64]
	// EnableFeedback is whether a tap gives feedback; the default is true.
	EnableFeedback foundation.Opt[bool]
	// Disabled makes the header not interactive: Flutter's enabled is false.
	Disabled bool
	// ExpansionAnimationStyle overrides the duration and the curves of the
	// expansion.
	ExpansionAnimationStyle *animation.AnimationStyle
	// InternalAddSemanticForOnTap gives the header the semantics of a button.
	InternalAddSemanticForOnTap bool
	// StatesController is the controller of the states of the header.
	StatesController *widgets.WidgetStatesController
}

// ExpansionTile is a single-line ListTile with an expansion arrow icon that
// expands or collapses the tile to reveal or hide the children.
//
// Flutter: material/expansion_tile.dart ExpansionTile
type ExpansionTile struct {
	widgets.StatefulWidget
	title widgets.Widget
	o     ExpansionTileOpts
}

// NewExpansionTile creates an expansion tile. The tag of its failure is
// ExpansionTileCrossAxisAlignmentBaseline.
//
// Flutter: material/expansion_tile.dart ExpansionTile.new
func NewExpansionTile(title widgets.Widget, opts ...ExpansionTileOpts) *ExpansionTile {
	o := firstOpts(opts)
	if foundation.DebugMode && o.ExpandedCrossAxisAlignment == rendering.CrossAxisAlignmentBaseline {
		panic(errors.New("ExpansionTileCrossAxisAlignmentBaseline", "CrossAxisAlignment.baseline is not supported since the expanded children are aligned in a column, not a row"))
	}
	return &ExpansionTile{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), title: title, o: o}
}

// CreateState creates the state.
//
// Flutter: material/expansion_tile.dart ExpansionTile.createState
func (w *ExpansionTile) CreateState() widgets.State { return &expansionTileState{} }

const expansionTileDuration = 200 * time.Millisecond

type expansionTileState struct {
	widgets.StateOf[*ExpansionTile]
	borderTween          *animation.Tween[painting.ShapeBorder]
	headerColorTween     *animation.Tween[foundation.Opt[ui.Color]]
	iconColorTween       *animation.Tween[foundation.Opt[ui.Color]]
	backgroundColorTween *animation.Tween[foundation.Opt[ui.Color]]
	easeIn, easeOut      animation.Animatable[float64]
	halfTween            *animation.Tween[float64]
	expansionTileTheme   ExpansionTileThemeData
	tileController       *widgets.ExpansibleController
	listener             foundation.ListenerHandle
	curve                animation.Curve
	reverseCurve         animation.Curve
	duration             time.Duration
}

func (s *expansionTileState) InitState() {
	w := s.Widget()
	s.borderTween = animation.NewTweenFunc[painting.ShapeBorder](nil, nil, func(a, b painting.ShapeBorder, t float64) painting.ShapeBorder {
		return painting.ShapeBorderLerp(a, b, t)
	})
	s.headerColorTween = animation.NewColorTween(foundation.None[ui.Color](), foundation.None[ui.Color]())
	s.iconColorTween = animation.NewColorTween(foundation.None[ui.Color](), foundation.None[ui.Color]())
	s.backgroundColorTween = animation.NewColorTween(foundation.None[ui.Color](), foundation.None[ui.Color]())
	s.easeIn = animation.NewCurveTween(animation.Curves.EaseIn)
	s.easeOut = animation.NewCurveTween(animation.Curves.EaseOut)
	s.halfTween = animation.NewTween(0.0, 0.5)
	s.curve = animation.Curves.EaseIn
	s.duration = expansionTileDuration
	s.tileController = w.o.Controller
	if s.tileController == nil {
		s.tileController = widgets.NewExpansibleController()
	}
	if w.o.InitiallyExpanded {
		s.tileController.Expand()
	}
	s.listener = s.tileController.AddListener(s.onExpansionChanged)
}

func (s *expansionTileState) Dispose() {
	s.tileController.RemoveListener(s.listener)
	if s.Widget().o.Controller == nil {
		s.tileController.Dispose()
	}
}

// onExpansionChanged tells the app.
//
// Deviation: Flutter also announces the new state with SemanticsService on the
// platforms other than Android; the announcement is not ported (SPEC §12.10).
//
// Flutter: material/expansion_tile.dart _ExpansionTileState._onExpansionChanged
func (s *expansionTileState) onExpansionChanged() {
	if f := s.Widget().o.OnExpansionChanged; f != nil {
		f(s.tileController.IsExpanded())
	}
}

func (s *expansionTileState) effectiveAffinity() ListTileControlAffinity {
	listTileTheme := ListTileThemeOf(s.Context())
	if firstNonZero(s.Widget().o.ControlAffinity, listTileTheme.ControlAffinity, ListTileControlAffinityTrailing) == ListTileControlAffinityLeading {
		return ListTileControlAffinityLeading
	}
	return ListTileControlAffinityTrailing
}

func (s *expansionTileState) buildIcon(a animation.Animation[float64]) widgets.Widget {
	turns := animation.Drive[float64](animation.Drive[float64](a, s.easeIn), s.halfTween)
	return widgets.NewRotationTransition(turns, widgets.NewIcon(Icons.ExpandMore()))
}

func (s *expansionTileState) buildLeadingIcon(_ widgets.BuildContext, a animation.Animation[float64]) widgets.Widget {
	if s.effectiveAffinity() != ListTileControlAffinityLeading {
		return nil
	}
	return s.buildIcon(a)
}

func (s *expansionTileState) buildTrailingIcon(_ widgets.BuildContext, a animation.Animation[float64]) widgets.Widget {
	if s.effectiveAffinity() != ListTileControlAffinityTrailing {
		return nil
	}
	return s.buildIcon(a)
}

// buildHeader builds the header.
//
// Deviation: the semantics' onTapHint is not ported: the semantics widget has no
// tap hint (SPEC §12.10).
//
// Flutter: material/expansion_tile.dart _ExpansionTileState._buildHeader
func (s *expansionTileState) buildHeader(bc widgets.BuildContext, a animation.Animation[float64]) widgets.Widget {
	o := s.Widget().o
	eased := animation.Drive[float64](a, s.easeIn)
	iconColor := animation.Drive[foundation.Opt[ui.Color]](eased, s.iconColorTween)
	headerColor := animation.Drive[foundation.Opt[ui.Color]](eased, s.headerColorTween)
	localizations := MaterialLocalizationsOf(bc)
	expanded := s.tileController.IsExpanded()
	semanticsHint := localizations.ExpandedHint()
	if expanded {
		semanticsHint = localizations.CollapsedHint()
	}
	if p := ThemeOf(bc).Platform; p == foundation.TargetPlatformIOS || p == foundation.TargetPlatformMacOS {
		if expanded {
			semanticsHint = localizations.CollapsedHint() + "\n " + localizations.ExpansionTileExpandedHint()
		} else {
			semanticsHint = localizations.ExpandedHint() + "\n " + localizations.ExpansionTileCollapsedHint()
		}
	}
	leading := o.Leading
	if leading == nil {
		leading = s.buildLeadingIcon(bc, a)
	}
	var trailing widgets.Widget
	if !o.NoShowTrailingIcon {
		trailing = o.Trailing
		if trailing == nil {
			trailing = s.buildTrailingIcon(bc, a)
		}
	}
	onTap := s.tileController.Expand
	if expanded {
		onTap = s.tileController.Collapse
	}
	child := ListTileThemeMerge(ListTileThemeData{
		IconColor: firstOpt(iconColor.Value(), s.expansionTileTheme.IconColor), TextColor: headerColor.Value(),
	}, NewListTile(ListTileOpts{
		Disabled: o.Disabled, OnTap: onTap, Dense: o.Dense, SplashColor: o.SplashColor, VisualDensity: o.VisualDensity,
		EnableFeedback: firstOpt(o.EnableFeedback, foundation.Some(true)), ContentPadding: firstNonNil(o.TilePadding, s.expansionTileTheme.TilePadding),
		Leading: leading, Title: s.Widget().title, Subtitle: o.Subtitle, Trailing: trailing, MinTileHeight: o.MinTileHeight,
		internalNoSemanticForOnTap: !o.InternalAddSemanticForOnTap, StatesController: o.StatesController,
	}))
	if ThemeOf(bc).Platform == foundation.TargetPlatformAndroid {
		return widgets.NewSemantics(widgets.NewSemantics(child, widgets.SemanticsOpts{Hint: foundation.Some(semanticsHint)}),
			widgets.SemanticsOpts{Label: foundation.Some(semanticsHint), LiveRegion: foundation.Some(true)})
	}
	return widgets.NewSemantics(child, widgets.SemanticsOpts{Hint: foundation.Some(semanticsHint)})
}

func (s *expansionTileState) buildBody(widgets.BuildContext, animation.Animation[float64]) widgets.Widget {
	o := s.Widget().o
	return widgets.NewAlign(widgets.NewPadding(
		firstNonNil(o.ChildrenPadding, s.expansionTileTheme.ChildrenPadding, painting.EdgeInsetsGeometry(painting.EdgeInsetsZero)),
		widgets.NewColumn(o.Children, widgets.ColumnOpts{CrossAxisAlignment: o.ExpandedCrossAxisAlignment})),
		widgets.AlignOpts{Alignment: firstNonNil(o.ExpandedAlignment, s.expansionTileTheme.ExpandedAlignment, painting.AlignmentGeometry(painting.AlignmentCenter))})
}

func transparentTopBottomBorder() painting.ShapeBorder {
	side := painting.NewBorderSide(painting.BorderSideOpts{Color: foundation.Some(Colors.Transparent)})
	return painting.Border{Top: side, Bottom: side}
}

func (s *expansionTileState) buildExpansible(bc widgets.BuildContext, header, body widgets.Widget, a animation.Animation[float64]) widgets.Widget {
	o := s.Widget().o
	eased := animation.Drive[float64](a, s.easeOut)
	backgroundColor := firstOpt(animation.Drive[foundation.Opt[ui.Color]](eased, s.backgroundColorTween).Value(), s.expansionTileTheme.BackgroundColor, foundation.Some(Colors.Transparent)).Value()
	border := animation.Drive[painting.ShapeBorder](eased, s.borderTween).Value()
	if border == nil {
		border = transparentTopBottomBorder()
	}
	clip := firstNonZero(o.ClipBehavior, s.expansionTileTheme.ClipBehavior, ui.ClipAntiAlias)
	decoration := painting.NewShapeDecoration(border, painting.ShapeDecorationOpts{Color: foundation.Some(backgroundColor)})
	var tile widgets.Widget = widgets.NewPadding(decoration.Padding(), widgets.NewColumn([]widgets.Widget{header, body},
		widgets.ColumnOpts{MainAxisSize: rendering.MainAxisSizeMin}))
	isShapeProvided := o.Shape != nil || s.expansionTileTheme.Shape != nil || o.CollapsedShape != nil || s.expansionTileTheme.CollapsedShape != nil
	if isShapeProvided {
		return NewMaterial(tile, MaterialOpts{ClipBehavior: clip, Color: foundation.Some(backgroundColor), Shape: border})
	}
	if backgroundColor.A > 0 {
		tile = NewMaterial(tile, MaterialOpts{Type: MaterialTypeTransparency})
	}
	return widgets.NewDecoratedBox(decoration, tile)
}

func (s *expansionTileState) expansionTileDefaults(theme *ThemeData) ExpansionTileThemeData {
	if theme.UseMaterial3 {
		colors := theme.ColorScheme
		return ExpansionTileThemeData{
			TextColor: foundation.Some(colors.OnSurface()), IconColor: foundation.Some(colors.Primary()),
			CollapsedTextColor: foundation.Some(colors.OnSurface()), CollapsedIconColor: foundation.Some(colors.OnSurfaceVariant()),
		}
	}
	return ExpansionTileThemeData{
		TextColor: foundation.Some(theme.ColorScheme.Primary()), IconColor: foundation.Some(theme.ColorScheme.Primary()),
		CollapsedTextColor: theme.TextTheme.TitleMedium.Color, CollapsedIconColor: foundation.Some(theme.UnselectedWidgetColor),
	}
}

func (s *expansionTileState) updateAnimationDuration() {
	o := s.Widget().o
	s.duration = expansionTileDuration
	if o.ExpansionAnimationStyle != nil && o.ExpansionAnimationStyle.Duration.IsSome() {
		s.duration = o.ExpansionAnimationStyle.Duration.Value()
	} else if st := s.expansionTileTheme.ExpansionAnimationStyle; st.Duration.IsSome() {
		s.duration = st.Duration.Value()
	}
}

func (s *expansionTileState) updateShapeBorder(theme *ThemeData) {
	o := s.Widget().o
	s.borderTween.Begin = firstNonNil(o.CollapsedShape, s.expansionTileTheme.CollapsedShape, transparentTopBottomBorder())
	side := painting.NewBorderSide(painting.BorderSideOpts{Color: foundation.Some(theme.DividerColor)})
	s.borderTween.End = firstNonNil(o.Shape, s.expansionTileTheme.Shape, painting.ShapeBorder(painting.Border{Top: side, Bottom: side}))
}

func (s *expansionTileState) updateHeaderColor(defaults ExpansionTileThemeData) {
	o := s.Widget().o
	s.headerColorTween.Begin = firstOpt(o.CollapsedTextColor, s.expansionTileTheme.CollapsedTextColor, defaults.CollapsedTextColor)
	s.headerColorTween.End = firstOpt(o.TextColor, s.expansionTileTheme.TextColor, defaults.TextColor)
}

func (s *expansionTileState) updateIconColor(defaults ExpansionTileThemeData) {
	o := s.Widget().o
	s.iconColorTween.Begin = firstOpt(o.CollapsedIconColor, s.expansionTileTheme.CollapsedIconColor, defaults.CollapsedIconColor)
	s.iconColorTween.End = firstOpt(o.IconColor, s.expansionTileTheme.IconColor, defaults.IconColor)
}

func (s *expansionTileState) updateBackgroundColor() {
	o := s.Widget().o
	s.backgroundColorTween.Begin = firstOpt(o.CollapsedBackgroundColor, s.expansionTileTheme.CollapsedBackgroundColor)
	s.backgroundColorTween.End = firstOpt(o.BackgroundColor, s.expansionTileTheme.BackgroundColor)
}

func (s *expansionTileState) updateHeightFactorCurve() {
	o := s.Widget().o
	s.curve = animation.Curves.EaseIn
	s.reverseCurve = nil
	if st := o.ExpansionAnimationStyle; st != nil && st.Curve != nil {
		s.curve = st.Curve
	} else if st := s.expansionTileTheme.ExpansionAnimationStyle; st.Curve != nil {
		s.curve = st.Curve
	}
	if st := o.ExpansionAnimationStyle; st != nil && st.ReverseCurve != nil {
		s.reverseCurve = st.ReverseCurve
	} else if st := s.expansionTileTheme.ExpansionAnimationStyle; st.ReverseCurve != nil {
		s.reverseCurve = st.ReverseCurve
	}
}

func (s *expansionTileState) refreshDependent() {
	theme := ThemeOf(s.Context())
	s.expansionTileTheme = ExpansionTileThemeOf(s.Context())
	defaults := s.expansionTileDefaults(theme)
	s.updateAnimationDuration()
	s.updateShapeBorder(theme)
	s.updateHeaderColor(defaults)
	s.updateIconColor(defaults)
	s.updateBackgroundColor()
	s.updateHeightFactorCurve()
}

func (s *expansionTileState) DidChangeDependencies() { s.refreshDependent() }

func (s *expansionTileState) DidUpdateWidget(old *ExpansionTile) {
	w := s.Widget()
	theme := ThemeOf(s.Context())
	s.expansionTileTheme = ExpansionTileThemeOf(s.Context())
	defaults := s.expansionTileDefaults(theme)
	if w.o.CollapsedShape != old.o.CollapsedShape || w.o.Shape != old.o.Shape {
		s.updateShapeBorder(theme)
	}
	if w.o.CollapsedTextColor != old.o.CollapsedTextColor || w.o.TextColor != old.o.TextColor {
		s.updateHeaderColor(defaults)
	}
	if w.o.CollapsedIconColor != old.o.CollapsedIconColor || w.o.IconColor != old.o.IconColor {
		s.updateIconColor(defaults)
	}
	if w.o.BackgroundColor != old.o.BackgroundColor || w.o.CollapsedBackgroundColor != old.o.CollapsedBackgroundColor {
		s.updateBackgroundColor()
	}
	if (w.o.ExpansionAnimationStyle == nil) != (old.o.ExpansionAnimationStyle == nil) ||
		w.o.ExpansionAnimationStyle != nil && !foundation.ValuesEqual(*w.o.ExpansionAnimationStyle, *old.o.ExpansionAnimationStyle) {
		s.updateAnimationDuration()
		s.updateHeightFactorCurve()
	}
	if w.o.Controller != old.o.Controller {
		s.tileController.RemoveListener(s.listener)
		if old.o.Controller == nil {
			s.tileController.Dispose()
		}
		s.tileController = w.o.Controller
		if s.tileController == nil {
			s.tileController = widgets.NewExpansibleController()
		}
		s.listener = s.tileController.AddListener(s.onExpansionChanged)
	}
}

func (s *expansionTileState) Build(widgets.BuildContext) widgets.Widget {
	return widgets.NewExpansible(s.tileController, s.buildHeader, s.buildBody, widgets.ExpansibleOpts{
		Curve: s.curve, Duration: foundation.Some(s.duration), ReverseCurve: s.reverseCurve,
		NoMaintainState: !s.Widget().o.MaintainState, ExpansibleBuilder: s.buildExpansible,
	})
}
