// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"fmt"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// ListTileStyle defines the title font used for ListTile descendants of a
// ListTileTheme.
//
// Flutter: material/list_tile.dart ListTileStyle
type ListTileStyle int

const (
	// ListTileStyleUnspecified is the zero value: the style of the theme, then list.
	//
	// Flutter: no counterpart
	ListTileStyleUnspecified ListTileStyle = iota
	// ListTileStyleList is the style for tiles in a list.
	//
	// Flutter: material/list_tile.dart ListTileStyle.list
	ListTileStyleList
	// ListTileStyleDrawer is the style for tiles in a drawer.
	//
	// Flutter: material/list_tile.dart ListTileStyle.drawer
	ListTileStyleDrawer
)

// Name returns the Dart enum name.
//
// Flutter: material/list_tile.dart ListTileStyle.name
func (s ListTileStyle) Name() string { return enumName(int(s), "list", "drawer") }

// String returns the description of the value.
//
// Flutter: material/list_tile.dart ListTileStyle.toString
func (s ListTileStyle) String() string { return "ListTileStyle." + s.Name() }

// ListTileControlAffinity defines how to position a control relative to the
// text of a CheckboxListTile, SwitchListTile or RadioListTile.
//
// Flutter: material/list_tile.dart ListTileControlAffinity
type ListTileControlAffinity int

const (
	// ListTileControlAffinityUnspecified is the zero value: the affinity of the
	// theme, then platform.
	//
	// Flutter: no counterpart
	ListTileControlAffinityUnspecified ListTileControlAffinity = iota
	// ListTileControlAffinityLeading puts the control at the start of the tile.
	//
	// Flutter: material/list_tile.dart ListTileControlAffinity.leading
	ListTileControlAffinityLeading
	// ListTileControlAffinityTrailing puts the control at the end of the tile.
	//
	// Flutter: material/list_tile.dart ListTileControlAffinity.trailing
	ListTileControlAffinityTrailing
	// ListTileControlAffinityPlatform puts the control on the side that is usual
	// for the platform.
	//
	// Flutter: material/list_tile.dart ListTileControlAffinity.platform
	ListTileControlAffinityPlatform
)

// Name returns the Dart enum name.
//
// Flutter: material/list_tile.dart ListTileControlAffinity.name
func (a ListTileControlAffinity) Name() string {
	return enumName(int(a), "leading", "trailing", "platform")
}

// String returns the description of the value.
//
// Flutter: material/list_tile.dart ListTileControlAffinity.toString
func (a ListTileControlAffinity) String() string { return "ListTileControlAffinity." + a.Name() }

// ListTileTitleAlignment defines how a ListTile's title is aligned against the
// leading and the trailing widgets.
//
// Flutter: material/list_tile.dart ListTileTitleAlignment
type ListTileTitleAlignment int

const (
	// ListTileTitleAlignmentUnspecified is the zero value: the alignment of the
	// theme, then threeLine in Material 3 and titleHeight in Material 2.
	//
	// Flutter: no counterpart
	ListTileTitleAlignmentUnspecified ListTileTitleAlignment = iota
	// ListTileTitleAlignmentThreeLine is top for a three line tile, center
	// otherwise.
	//
	// Flutter: material/list_tile.dart ListTileTitleAlignment.threeLine
	ListTileTitleAlignmentThreeLine
	// ListTileTitleAlignmentTitleHeight puts the widgets at 16 from the top of a
	// tile taller than 72, and centers them in the others.
	//
	// Flutter: material/list_tile.dart ListTileTitleAlignment.titleHeight
	ListTileTitleAlignmentTitleHeight
	// ListTileTitleAlignmentTop puts the widgets at the top of the tile.
	//
	// Flutter: material/list_tile.dart ListTileTitleAlignment.top
	ListTileTitleAlignmentTop
	// ListTileTitleAlignmentCenter centers the widgets.
	//
	// Flutter: material/list_tile.dart ListTileTitleAlignment.center
	ListTileTitleAlignmentCenter
	// ListTileTitleAlignmentBottom puts the widgets at the bottom of the tile.
	//
	// Flutter: material/list_tile.dart ListTileTitleAlignment.bottom
	ListTileTitleAlignmentBottom
)

// Name returns the Dart enum name.
//
// Flutter: material/list_tile.dart ListTileTitleAlignment.name
func (a ListTileTitleAlignment) Name() string {
	return enumName(int(a), "threeLine", "titleHeight", "top", "center", "bottom")
}

// String returns the description of the value.
//
// Flutter: material/list_tile.dart ListTileTitleAlignment.toString
func (a ListTileTitleAlignment) String() string { return "ListTileTitleAlignment." + a.Name() }

// yOffsetFor is the vertical position of a leading or trailing widget.
//
// Flutter: material/list_tile.dart ListTileTitleAlignment._yOffsetFor
func (a ListTileTitleAlignment) yOffsetFor(childHeight, tileHeight float64, listTile *renderListTile, isLeading bool) float64 {
	switch a {
	case ListTileTitleAlignmentThreeLine:
		if listTile.isThreeLine {
			return ListTileTitleAlignmentTop.yOffsetFor(childHeight, tileHeight, listTile, isLeading)
		}
		return ListTileTitleAlignmentCenter.yOffsetFor(childHeight, tileHeight, listTile, isLeading)
	case ListTileTitleAlignmentTitleHeight:
		if tileHeight > 72 {
			return 16
		}
		if isLeading {
			return min((tileHeight-childHeight)/2, 16)
		}
		return (tileHeight - childHeight) / 2
	case ListTileTitleAlignmentTop:
		return listTile.minVerticalPadding
	case ListTileTitleAlignmentBottom:
		return tileHeight - childHeight - listTile.minVerticalPadding
	}
	return (tileHeight - childHeight) / 2
}

// ListTileOpts are the optional parameters of NewListTile.
//
// Flutter: material/list_tile.dart ListTile.new
type ListTileOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Leading is a widget to display before the title.
	Leading widgets.Widget
	// Title is the primary content of the tile.
	Title widgets.Widget
	// Subtitle is additional content displayed below the title.
	Subtitle widgets.Widget
	// Trailing is a widget to display after the title.
	Trailing widgets.Widget
	// IsThreeLine is whether the tile is intended to display three lines of
	// text; it needs a subtitle.
	IsThreeLine foundation.Opt[bool]
	// Dense is whether the tile is part of a vertically dense list.
	Dense foundation.Opt[bool]
	// VisualDensity defines how compact the tile's layout is.
	VisualDensity foundation.Opt[VisualDensity]
	// Shape defines the tile's InkWell.customBorder and Ink.decoration shape.
	Shape painting.ShapeBorder
	// Style defines the font used for the title.
	Style ListTileStyle
	// SelectedColor is the color of the icons and the text of a selected tile.
	//
	// Deviation: a WidgetStateColor is not supported here (SPEC §12.10).
	SelectedColor foundation.Opt[ui.Color]
	// IconColor is the color of the icons of an unselected, enabled tile.
	IconColor foundation.Opt[ui.Color]
	// TextColor is the color of the text of an unselected, enabled tile.
	TextColor foundation.Opt[ui.Color]
	// TitleTextStyle is the text style of the title.
	TitleTextStyle *painting.TextStyle
	// SubtitleTextStyle is the text style of the subtitle.
	SubtitleTextStyle *painting.TextStyle
	// LeadingAndTrailingTextStyle is the text style of the leading and the
	// trailing widgets.
	LeadingAndTrailingTextStyle *painting.TextStyle
	// ContentPadding is the tile's internal padding.
	ContentPadding painting.EdgeInsetsGeometry
	// Disabled is true when the tile is not interactive: Flutter's enabled is
	// false.
	Disabled bool
	// OnTap is called when the user taps the tile.
	OnTap gestures.GestureTapCallback
	// OnLongPress is called when the user long-presses the tile.
	OnLongPress gestures.GestureLongPressCallback
	// OnFocusChange is called when the focus of the tile changes.
	OnFocusChange func(bool)
	// MouseCursor is the cursor of a pointer over the tile; it may be a
	// WidgetStateMouseCursor.
	MouseCursor services.MouseCursor
	// Selected is whether the tile looks selected.
	Selected bool
	// FocusColor is the color of the tile when it has the input focus.
	FocusColor foundation.Opt[ui.Color]
	// HoverColor is the color of the tile when a pointer hovers over it.
	HoverColor foundation.Opt[ui.Color]
	// SplashColor is the splash color of the tile's InkWell.
	SplashColor foundation.Opt[ui.Color]
	// FocusNode is the focus node of the tile.
	FocusNode *widgets.FocusNode
	// Autofocus is whether the tile focuses itself when it is first built.
	Autofocus bool
	// TileColor is the background color of the tile.
	TileColor foundation.Opt[ui.Color]
	// SelectedTileColor is the background color of a selected tile.
	SelectedTileColor foundation.Opt[ui.Color]
	// EnableFeedback is whether a tap gives feedback; the default is true.
	EnableFeedback foundation.Opt[bool]
	// HorizontalTitleGap is the horizontal gap between the title and the leading
	// and trailing widgets; the default is 16.
	HorizontalTitleGap foundation.Opt[float64]
	// MinVerticalPadding is the minimum padding on the top and the bottom of the
	// title and the subtitle.
	MinVerticalPadding foundation.Opt[float64]
	// MinLeadingWidth is the minimum width allocated for the leading widget.
	MinLeadingWidth foundation.Opt[float64]
	// MinTileHeight is the minimum height of the tile.
	MinTileHeight foundation.Opt[float64]
	// TitleAlignment defines how the leading and trailing widgets align against
	// the title.
	TitleAlignment ListTileTitleAlignment
	// StatesController is the controller of the states of the tile's InkWell.
	StatesController *widgets.WidgetStatesController

	// internalNoSemanticForOnTap leaves out the button semantics of the tile,
	// for the tiles that have a control with its own (Flutter's
	// internalAddSemanticForOnTap: false).
	internalNoSemanticForOnTap bool
}

// ListTile is a single fixed-height row that typically contains some text as
// well as a leading or trailing icon.
//
// Flutter: material/list_tile.dart ListTile
type ListTile struct {
	widgets.StatelessWidget
	o ListTileOpts
}

// NewListTile creates a list tile. The tag of its failure is
// ListTileThreeLineNoSubtitle (isThreeLine needs a subtitle).
//
// Flutter: material/list_tile.dart ListTile.new
func NewListTile(opts ...ListTileOpts) *ListTile {
	o := firstOpts(opts)
	if foundation.DebugMode && o.IsThreeLine.OrElse(false) && o.Subtitle == nil {
		panic(errors.New(tagListTileThreeLineNoSubtitle, "a three line list tile needs a subtitle"))
	}
	return &ListTile{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), o: o}
}

// ListTileDivideTiles returns the tiles with a divider between each of them,
// drawn as the bottom border of the tile: the color is the one that is given, or
// the divider theme's of the context.
//
// Flutter: material/list_tile.dart ListTile.divideTiles
func ListTileDivideTiles(bc widgets.BuildContext, tiles []widgets.Widget, color foundation.Opt[ui.Color]) []widgets.Widget {
	if foundation.DebugMode && color.IsNone() && bc == nil {
		panic(errors.New(tagListTileDivideTilesNoColor, "divideTiles needs a color or a context"))
	}
	if len(tiles) <= 1 {
		return tiles
	}
	wrapTile := func(tile widgets.Widget) widgets.Widget {
		return widgets.NewDecoratedBox(painting.NewBoxDecoration(painting.BoxDecorationOpts{
			Border: painting.Border{Bottom: DividerCreateBorderSide(bc, DividerBorderSideOpts{Color: color})},
		}), tile, widgets.DecoratedBoxOpts{Position: rendering.DecorationPositionForeground})
	}
	out := make([]widgets.Widget, 0, len(tiles))
	for _, t := range tiles[:len(tiles)-1] {
		out = append(out, wrapTile(t))
	}
	return append(out, tiles[len(tiles)-1])
}

func (w *ListTile) isDenseLayout(theme *ThemeData, tileTheme ListTileThemeData) bool {
	return firstOpt(w.o.Dense, tileTheme.Dense, theme.ListTileTheme.Dense).OrElse(false)
}

// resolveTileColor is Flutter's _IndividualOverrides.resolve for a color that is
// not a WidgetStateColor: the disabled color, the selected color or the enabled
// one, by the state.
//
// Flutter: material/list_tile.dart _IndividualOverrides.resolve
func resolveTileColor(disabled, selected bool, selectedColor, enabledColor, disabledColor foundation.Opt[ui.Color]) foundation.Opt[ui.Color] {
	switch {
	case disabled:
		return disabledColor
	case selected:
		return selectedColor
	}
	return enabledColor
}

// Build builds the tile.
//
// Flutter: material/list_tile.dart ListTile.build
func (w *ListTile) Build(bc widgets.BuildContext) widgets.Widget {
	o := w.o
	DebugCheckHasMaterial(bc)
	theme := ThemeOf(bc)
	iconButtonTheme := IconButtonThemeOf(bc)
	tileTheme := ListTileThemeOf(bc)
	listTileStyle := firstNonZero(o.Style, tileTheme.Style, theme.ListTileTheme.Style, ListTileStyleList)
	var defaults listTileDefaults
	if theme.UseMaterial3 {
		defaults = listTileDefaultsM3(theme)
	} else {
		defaults = listTileDefaultsM2(theme, listTileStyle)
	}
	backgroundColor := firstOpt(o.TileColor, tileTheme.TileColor, theme.ListTileTheme.TileColor, foundation.Some(defaults.tileColor)).Value()
	selectedBackgroundColor := firstOpt(o.SelectedTileColor, tileTheme.SelectedTileColor, theme.ListTileTheme.SelectedTileColor, foundation.Some(defaults.tileColor)).Value()
	effectiveTileColor := backgroundColor
	if o.Selected {
		effectiveTileColor = selectedBackgroundColor
	}
	hasOpaqueBackground := backgroundColor.A > 0 || selectedBackgroundColor.A > 0
	if foundation.DebugMode && (o.OnTap != nil || o.OnLongPress != nil || hasOpaqueBackground) {
		debugCheckListTileBackgroundIsHidden(bc)
	}

	disabled := o.Disabled
	resolve := func(selectedColor, enabledColor foundation.Opt[ui.Color], disabledColor ...foundation.Opt[ui.Color]) foundation.Opt[ui.Color] {
		return resolveTileColor(disabled, o.Selected, selectedColor, enabledColor, firstOpt(disabledColor...))
	}
	effectiveIconColor := firstOpt(
		resolve(o.SelectedColor, o.IconColor),
		resolve(tileTheme.SelectedColor, tileTheme.IconColor),
		resolve(theme.ListTileTheme.SelectedColor, theme.ListTileTheme.IconColor))
	disabledColor := foundation.Some(theme.DisabledColor)
	defaultEffectiveIconColor := resolve(defaults.selectedColor, defaults.iconColor, disabledColor)
	var styleForeground foundation.Opt[ui.Color]
	if iconButtonTheme.Style != nil && iconButtonTheme.Style.ForegroundColor != nil {
		states := widgets.WidgetStates(0)
		if disabled {
			states = states.With(widgets.WidgetStateDisabled)
		}
		if o.Selected {
			states = states.With(widgets.WidgetStateSelected)
		}
		styleForeground = iconButtonTheme.Style.ForegroundColor.Resolve(states)
	}
	effectiveIconButtonColor := firstOpt(effectiveIconColor, styleForeground, defaultEffectiveIconColor)
	effectiveIconColor = firstOpt(effectiveIconColor, defaultEffectiveIconColor)
	effectiveColor := firstOpt(
		resolve(o.SelectedColor, o.TextColor),
		resolve(tileTheme.SelectedColor, tileTheme.TextColor),
		resolve(theme.ListTileTheme.SelectedColor, theme.ListTileTheme.TextColor),
		resolve(defaults.selectedColor, defaults.textColor, disabledColor))

	iconThemeData := widgets.NewIconThemeData(widgets.IconThemeDataOpts{Color: effectiveIconColor})
	var iconButtonStyle *ButtonStyle
	foreground := widgets.NewWidgetStatePropertyAll(effectiveIconButtonColor)
	if iconButtonTheme.Style != nil {
		iconButtonStyle = iconButtonTheme.Style.CopyWith(ButtonStyleOpts{ForegroundColor: foreground})
	} else {
		iconButtonStyle = IconButtonStyleFrom(IconButtonStyleFromOpts{ForegroundColor: effectiveIconButtonColor})
	}
	iconButtonThemeData := IconButtonThemeData{Style: iconButtonStyle}

	var leadingAndTrailingStyle *painting.TextStyle
	if o.Leading != nil || o.Trailing != nil {
		leadingAndTrailingStyle = firstNonNil(o.LeadingAndTrailingTextStyle, tileTheme.LeadingAndTrailingTextStyle, defaults.leadingAndTrailingTextStyle)
		leadingAndTrailingStyle = leadingAndTrailingStyle.CopyWith(painting.TextStyleCopy{Color: effectiveColor})
	}
	var leadingIcon widgets.Widget
	if o.Leading != nil {
		leadingIcon = widgets.NewAnimatedDefaultTextStyle(leadingAndTrailingStyle, themeChangeDuration, o.Leading)
	}
	dense := w.isDenseLayout(theme, tileTheme)
	titleStyle := firstNonNil(o.TitleTextStyle, tileTheme.TitleTextStyle, defaults.titleTextStyle)
	titleCopy := painting.TextStyleCopy{Color: effectiveColor}
	if dense {
		titleCopy.FontSize = foundation.Some(13.0)
	}
	titleStyle = titleStyle.CopyWith(titleCopy)
	var title widgets.Widget = widgets.NewSizedBox(nil)
	if o.Title != nil {
		title = o.Title
	}
	titleText := widgets.NewAnimatedDefaultTextStyle(titleStyle, themeChangeDuration, title)
	var subtitleText widgets.Widget
	var subtitleStyle *painting.TextStyle
	if o.Subtitle != nil {
		subtitleStyle = firstNonNil(o.SubtitleTextStyle, tileTheme.SubtitleTextStyle, defaults.subtitleTextStyle)
		subtitleCopy := painting.TextStyleCopy{Color: effectiveColor}
		if dense {
			subtitleCopy.FontSize = foundation.Some(12.0)
		}
		subtitleStyle = subtitleStyle.CopyWith(subtitleCopy)
		subtitleText = widgets.NewAnimatedDefaultTextStyle(subtitleStyle, themeChangeDuration, o.Subtitle)
	}
	var trailingIcon widgets.Widget
	if o.Trailing != nil {
		trailingIcon = widgets.NewAnimatedDefaultTextStyle(leadingAndTrailingStyle, themeChangeDuration, o.Trailing)
	}
	textDirection := widgets.DirectionalityOf(bc)
	resolvedContentPadding := firstNonNil(o.ContentPadding, tileTheme.ContentPadding, defaults.contentPadding).Resolve(textDirection)

	mouseStates := widgets.WidgetStates(0)
	if disabled || (o.OnTap == nil && o.OnLongPress == nil) {
		mouseStates = mouseStates.With(widgets.WidgetStateDisabled)
	}
	effectiveMouseCursor := resolveMouseCursor(o.MouseCursor, mouseStates)
	if effectiveMouseCursor == nil && tileTheme.MouseCursor != nil {
		effectiveMouseCursor = tileTheme.MouseCursor.Resolve(mouseStates)
	}
	if effectiveMouseCursor == nil {
		effectiveMouseCursor = widgets.WidgetStateMouseCursorClickable.Resolve(mouseStates)
	}
	effectiveTitleAlignment := o.TitleAlignment
	if effectiveTitleAlignment == ListTileTitleAlignmentUnspecified {
		effectiveTitleAlignment = tileTheme.TitleAlignment
	}
	if effectiveTitleAlignment == ListTileTitleAlignmentUnspecified {
		if theme.UseMaterial3 {
			effectiveTitleAlignment = ListTileTitleAlignmentThreeLine
		} else {
			effectiveTitleAlignment = ListTileTitleAlignmentTitleHeight
		}
	}
	var onTap gestures.GestureTapCallback
	var onLongPress gestures.GestureLongPressCallback
	if !disabled {
		onTap, onLongPress = o.OnTap, o.OnLongPress
	}
	shape := firstNonNil(o.Shape, tileTheme.Shape)
	decorationShape := firstNonNil(shape, painting.ShapeBorder(painting.Border{}))
	visualDensity := firstOpt(o.VisualDensity, tileTheme.VisualDensity, foundation.Some(theme.VisualDensity)).Value()
	titleBaseline := titleStyle.TextBaseline
	if titleBaseline == ui.TextBaselineUnspecified {
		titleBaseline = defaults.titleTextStyle.TextBaseline
	}
	subtitleBaseline := defaults.subtitleTextStyle.TextBaseline
	if subtitleStyle != nil && subtitleStyle.TextBaseline != ui.TextBaselineUnspecified {
		subtitleBaseline = subtitleStyle.TextBaseline
	}
	return NewInkWell(widgets.NewSemantics(
		NewInk(widgets.NewSafeArea(
			widgets.NewIconThemeMerge(iconThemeData, NewIconButtonTheme(iconButtonThemeData, &listTileWidget{
				SlottedMultiChildRenderObjectWidget: widgets.NewSlottedMultiChildRenderObjectWidgetBase(),
				leading:                             leadingIcon, title: titleText, subtitle: subtitleText, trailing: trailingIcon,
				isThreeLine:          firstOpt(o.IsThreeLine, tileTheme.IsThreeLine, theme.ListTileTheme.IsThreeLine).OrElse(false),
				isDense:              dense,
				visualDensity:        visualDensity,
				textDirection:        textDirection,
				titleBaselineType:    titleBaseline,
				subtitleBaselineType: subtitleBaseline,
				horizontalTitleGap:   firstOpt(o.HorizontalTitleGap, tileTheme.HorizontalTitleGap, foundation.Some(16.0)).Value(),
				minVerticalPadding:   firstOpt(o.MinVerticalPadding, tileTheme.MinVerticalPadding, foundation.Some(defaults.minVerticalPadding)).Value(),
				minLeadingWidth:      firstOpt(o.MinLeadingWidth, tileTheme.MinLeadingWidth, foundation.Some(defaults.minLeadingWidth)).Value(),
				minTileHeight:        firstOpt(o.MinTileHeight, tileTheme.MinTileHeight),
				titleAlignment:       effectiveTitleAlignment,
			})),
			widgets.SafeAreaOpts{Top: foundation.Some(false), Bottom: foundation.Some(false), Minimum: resolvedContentPadding}),
			InkOpts{Decoration: painting.NewShapeDecoration(decorationShape, painting.ShapeDecorationOpts{Color: foundation.Some(effectiveTileColor)})}),
		widgets.SemanticsOpts{
			Button:   foundation.Some(!o.internalNoSemanticForOnTap && (o.OnTap != nil || o.OnLongPress != nil)),
			Selected: foundation.Some(o.Selected), Enabled: foundation.Some(!disabled),
		}), InkWellOpts{
		CustomBorder: shape, OnTap: onTap, OnLongPress: onLongPress, OnFocusChange: o.OnFocusChange, MouseCursor: effectiveMouseCursor,
		CanRequestFocus: foundation.Some(!disabled), FocusNode: o.FocusNode, FocusColor: o.FocusColor, HoverColor: o.HoverColor,
		SplashColor: o.SplashColor, Autofocus: o.Autofocus, EnableFeedback: firstOpt(o.EnableFeedback, tileTheme.EnableFeedback, foundation.Some(true)),
		StatesController: o.StatesController,
	})
}

// themeChangeDuration is Flutter's kThemeChangeDuration.
const themeChangeDuration = 200 * time.Millisecond

// resolveMouseCursor resolves a cursor that depends on the states; other
// cursors are as they are.
//
// Flutter: widgets/widget_state.dart WidgetStateProperty.resolveAs
func resolveMouseCursor(c services.MouseCursor, states widgets.WidgetStates) services.MouseCursor {
	if r, ok := c.(interface {
		Resolve(widgets.WidgetStates) services.MouseCursor
	}); ok {
		return r.Resolve(states)
	}
	return c
}

// debugCheckListTileBackgroundIsHidden panics with the tag
// ListTileBackgroundHidden when a widget between the tile and its Material
// paints a background that hides the tile's.
//
// Deviation: Flutter reports the error and goes on; a panic in build is caught
// by the framework and reported in the same way (SPEC §12.10).
//
// Flutter: material/list_tile.dart ListTile._debugCheckBackgroundIsHidden
func debugCheckListTileBackgroundIsHidden(bc widgets.BuildContext) {
	var intermediate widgets.Widget
	bc.(widgets.Element).VisitAncestorElements(func(ancestor widgets.Element) bool {
		w := ancestor.Widget()
		if _, ok := w.(*Material); ok {
			return false
		}
		var color foundation.Opt[ui.Color]
		switch x := w.(type) {
		case *widgets.ColoredBox:
			color = foundation.Some(x.Color())
		case *widgets.DecoratedBox:
			switch d := x.Decoration().(type) {
			case *painting.BoxDecoration:
				color = d.Color
			case *painting.ShapeDecoration:
				color = d.Color
			}
		}
		if c, ok := color.Get(); ok && c.A > 0 {
			intermediate = w
			return false
		}
		return true
	})
	if intermediate != nil {
		panic(errors.New(tagListTileBackgroundHidden, "the ListTile is wrapped in a widget that has a background color, which hides the background and the ink splashes of the tile; wrap the ListTile in its own Material, or remove the color of the widget",
			"widget", fmt.Sprintf("%T", intermediate)))
	}
}

// listTileDefaults are the defaults of a list tile: the values that its theme
// does not give.
//
// Flutter: material/list_tile.dart _LisTileDefaultsM2, _LisTileDefaultsM3
type listTileDefaults struct {
	contentPadding              painting.EdgeInsetsGeometry
	minLeadingWidth             float64
	minVerticalPadding          float64
	shape                       painting.ShapeBorder
	tileColor                   ui.Color
	titleTextStyle              *painting.TextStyle
	subtitleTextStyle           *painting.TextStyle
	leadingAndTrailingTextStyle *painting.TextStyle
	selectedColor               foundation.Opt[ui.Color]
	iconColor                   foundation.Opt[ui.Color]
	textColor                   foundation.Opt[ui.Color]
}

func listTileDefaultsM2(theme *ThemeData, style ListTileStyle) listTileDefaults {
	textTheme := theme.TextTheme
	title := textTheme.TitleMedium
	if style == ListTileStyleDrawer {
		title = textTheme.BodyLarge
	}
	d := listTileDefaults{
		contentPadding: painting.NewEdgeInsetsSymmetric(0, 16), minLeadingWidth: 40, minVerticalPadding: 4,
		shape: painting.Border{}, tileColor: Colors.Transparent, titleTextStyle: title,
		subtitleTextStyle:           textTheme.BodyMedium.CopyWith(painting.TextStyleCopy{Color: textTheme.BodySmall.Color}),
		leadingAndTrailingTextStyle: textTheme.BodyMedium, selectedColor: foundation.Some(theme.ColorScheme.Primary()),
	}
	if theme.Brightness() == ui.BrightnessLight {
		d.iconColor = foundation.Some(Colors.Black45)
	}
	return d
}

func listTileDefaultsM3(theme *ThemeData) listTileDefaults {
	colors, textTheme := theme.ColorScheme, theme.TextTheme
	return listTileDefaults{
		contentPadding: painting.NewEdgeInsetsDirectionalFromSTEB(16, 0, 24, 0), minLeadingWidth: 24, minVerticalPadding: 8,
		shape: painting.NewRoundedRectangleBorder(painting.RoundedRectangleBorderOpts{}), tileColor: Colors.Transparent,
		titleTextStyle:              textTheme.BodyLarge.CopyWith(painting.TextStyleCopy{Color: foundation.Some(colors.OnSurface())}),
		subtitleTextStyle:           textTheme.BodyMedium.CopyWith(painting.TextStyleCopy{Color: foundation.Some(colors.OnSurfaceVariant())}),
		leadingAndTrailingTextStyle: textTheme.LabelSmall.CopyWith(painting.TextStyleCopy{Color: foundation.Some(colors.OnSurfaceVariant())}),
		selectedColor:               foundation.Some(colors.Primary()), iconColor: foundation.Some(colors.OnSurfaceVariant()),
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagListTileBackgroundHidden    = "ListTileBackgroundHidden"
	tagListTileDivideTilesNoColor  = "ListTileDivideTilesNoColor"
	tagListTileThreeLineNoSubtitle = "ListTileThreeLineNoSubtitle"
)
