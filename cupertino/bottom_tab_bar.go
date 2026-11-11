package cupertino

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// cupertinoTabBarHeight is the standard iOS 10 tab bar height.
const cupertinoTabBarHeight = 50.0

// CupertinoTabBarOpts are the optional parameters of NewCupertinoTabBar.
//
// Flutter: cupertino/bottom_tab_bar.dart CupertinoTabBar.new
type CupertinoTabBarOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// OnTap is called with the index of the item that is tapped. The widget that makes the bar keeps the
	// current index and rebuilds the bar with it.
	OnTap func(index int)
	// CurrentIndex is the index of the active item; it is from 0 to the number of items minus 1.
	CurrentIndex int
	// BackgroundColor is the background of the bar; the default is the bar background color of the theme.
	// When it is not opaque the bar blurs the content behind it.
	BackgroundColor ui.ColorValue
	// ActiveColor is the color of the icon and the label of the active item; the default is the primary
	// color of the theme.
	ActiveColor ui.ColorValue
	// InactiveColor is the color of the icon and the label of the other items; the default is
	// CupertinoColors.InactiveGray.
	InactiveColor ui.ColorValue
	// IconSize is the size of the icons; the default is 30.
	IconSize foundation.Opt[float64]
	// Height is the height of the bar without the bottom padding; the default is 50.
	Height foundation.Opt[float64]
	// Border is the border of the bar; the default is a top border of one physical pixel with a dynamic gray.
	// Deviation: the colors of a border that is given are used as they are (a painting.BorderSide has a
	// plain color, not a dynamic one); the default border resolves its color from the context.
	Border foundation.Opt[painting.Border]
	// NoBorder says that the bar has no border (Flutter's border: null).
	NoBorder bool
}

// CupertinoTabBar is an iOS styled bottom navigation tab bar: items with an icon and a label, one active. It
// does not keep the active tab: the widget that makes it listens to OnTap and rebuilds it with the new
// CurrentIndex (CupertinoTabScaffold does this). A background that is not opaque blurs what is behind the bar.
//
// Flutter: cupertino/bottom_tab_bar.dart CupertinoTabBar
type CupertinoTabBar struct {
	widgets.StatelessWidget
	items []widgets.BottomNavigationBarItem
	o     CupertinoTabBarOpts
}

// NewCupertinoTabBar creates a tab bar for the items. In debug builds it panics with the tag
// CupertinoTabBarItemsTooFew with fewer than 2 items, CupertinoTabBarCurrentIndexInvalid when the current
// index is not an item, and CupertinoTabBarHeightInvalid when the height is negative.
//
// Flutter: cupertino/bottom_tab_bar.dart CupertinoTabBar.new
func NewCupertinoTabBar(items []widgets.BottomNavigationBarItem, opts ...CupertinoTabBarOpts) *CupertinoTabBar {
	o := firstOf(opts)
	if foundation.DebugMode {
		if len(items) < 2 {
			panic(errors.New(tagCupertinoTabBarItemsTooFew, "tabs need at least 2 items to conform to Apple's HIG", "items", len(items)))
		}
		if o.CurrentIndex < 0 || o.CurrentIndex >= len(items) {
			panic(errors.New(tagCupertinoTabBarCurrentIndexInvalid, "the current index is not an item", "currentIndex", o.CurrentIndex, "items", len(items)))
		}
		if o.Height.IsSome() && o.Height.Value() < 0 {
			panic(errors.New(tagCupertinoTabBarHeightInvalid, "the height must not be negative", "height", o.Height.Value()))
		}
	}
	if o.InactiveColor == nil {
		o.InactiveColor = CupertinoColors.InactiveGray
	}
	return &CupertinoTabBar{widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), items, o}
}

// Items are the items of the bar.
//
// Flutter: cupertino/bottom_tab_bar.dart CupertinoTabBar.items
func (w *CupertinoTabBar) Items() []widgets.BottomNavigationBarItem { return w.items }

// OnTap is the callback for a tap on an item, or nil.
//
// Flutter: cupertino/bottom_tab_bar.dart CupertinoTabBar.onTap
func (w *CupertinoTabBar) OnTap() func(int) { return w.o.OnTap }

// CurrentIndex is the index of the active item.
//
// Flutter: cupertino/bottom_tab_bar.dart CupertinoTabBar.currentIndex
func (w *CupertinoTabBar) CurrentIndex() int { return w.o.CurrentIndex }

// BackgroundColor is the background color, or nil for the theme's.
//
// Flutter: cupertino/bottom_tab_bar.dart CupertinoTabBar.backgroundColor
func (w *CupertinoTabBar) BackgroundColor() ui.ColorValue { return w.o.BackgroundColor }

// ActiveColor is the color of the active item, or nil for the primary color.
//
// Flutter: cupertino/bottom_tab_bar.dart CupertinoTabBar.activeColor
func (w *CupertinoTabBar) ActiveColor() ui.ColorValue { return w.o.ActiveColor }

// InactiveColor is the color of the items that are not active.
//
// Flutter: cupertino/bottom_tab_bar.dart CupertinoTabBar.inactiveColor
func (w *CupertinoTabBar) InactiveColor() ui.ColorValue { return w.o.InactiveColor }

// IconSize is the size of the icons.
//
// Flutter: cupertino/bottom_tab_bar.dart CupertinoTabBar.iconSize
func (w *CupertinoTabBar) IconSize() float64 { return w.o.IconSize.OrElse(30.0) }

// Height is the height of the bar, without the bottom padding.
//
// Flutter: cupertino/bottom_tab_bar.dart CupertinoTabBar.height
func (w *CupertinoTabBar) Height() float64 { return w.o.Height.OrElse(cupertinoTabBarHeight) }

// PreferredSize is the size with the height of the bar and no width.
//
// Flutter: cupertino/bottom_tab_bar.dart CupertinoTabBar.preferredSize
func (w *CupertinoTabBar) PreferredSize() ui.Size {
	return ui.Size{Width: infinity, Height: w.Height()}
}

// Opaque is whether the bar is fully opaque, or lets what is behind it show.
//
// Flutter: cupertino/bottom_tab_bar.dart CupertinoTabBar.opaque
func (w *CupertinoTabBar) Opaque(bc widgets.BuildContext) bool {
	return reduceTransparency(bc, CupertinoDynamicColorResolve(w.backgroundOrTheme(bc), bc)).Alpha() == 0xFF
}

func (w *CupertinoTabBar) backgroundOrTheme(bc widgets.BuildContext) ui.ColorValue {
	if w.o.BackgroundColor != nil {
		return w.o.BackgroundColor
	}
	return CupertinoThemeOf(bc).BarBackgroundColor()
}

// tabBarDefaultBorderColor is the color of the default border of the bar.
var tabBarDefaultBorderColor = NewCupertinoDynamicColorWithBrightness(ui.NewColor(0x4D000000), ui.NewColor(0x29000000))

// Build builds the bar.
//
// Flutter: cupertino/bottom_tab_bar.dart CupertinoTabBar.build
func (w *CupertinoTabBar) Build(bc widgets.BuildContext) widgets.Widget {
	widgets.DebugCheckHasMediaQuery(bc)
	bottomPadding := widgets.MediaQueryViewPaddingOf(bc).Bottom
	backgroundColor := reduceTransparency(bc, CupertinoDynamicColorResolve(w.backgroundOrTheme(bc), bc))

	var border painting.BoxBorder
	switch {
	case w.o.NoBorder:
	case w.o.Border.IsSome():
		border = w.o.Border.Value()
	default:
		border = painting.Border{Top: painting.NewBorderSide(painting.BorderSideOpts{
			Color: foundation.Some(CupertinoDynamicColorResolve(tabBarDefaultBorderColor, bc)),
			Width: foundation.Some(0.0), // 0.0 means one physical pixel
		}), Right: painting.BorderSideNone, Bottom: painting.BorderSideNone, Left: painting.BorderSideNone}
	}

	inactive := CupertinoDynamicColorResolve(w.o.InactiveColor, bc)
	var result widgets.Widget = widgets.NewDecoratedBox(
		painting.NewBoxDecoration(painting.BoxDecorationOpts{Border: border, Color: foundation.Some(backgroundColor)}),
		widgets.NewSizedBox(
			widgets.NewIconThemeMerge(
				// Default with the inactive state.
				widgets.NewIconThemeData(widgets.IconThemeDataOpts{Color: foundation.Some(inactive), Size: foundation.Some(w.IconSize())}),
				widgets.NewDefaultTextStyle(
					// Default with the inactive state.
					CupertinoThemeOf(bc).TextTheme().TabLabelTextStyle().CopyWith(painting.TextStyleCopy{Color: foundation.Some(inactive)}),
					widgets.NewPadding(painting.EdgeInsets{Bottom: bottomPadding},
						widgets.NewSemantics(
							widgets.NewRow(w.buildTabItems(bc), widgets.RowOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentEnd}),
							widgets.SemanticsOpts{ExplicitChildNodes: true}))),
			), widgets.SizedBoxOpts{Height: foundation.Some(w.Height() + bottomPadding)}))

	if !w.Opaque(bc) {
		// For non-opaque backgrounds, apply a blur effect.
		result = widgets.NewClipRect(widgets.NewBackdropFilter(
			ui.NewImageFilterBlur(ui.ImageFilterBlurOpts{SigmaX: 10, SigmaY: 10}), result))
	}
	return result
}

func (w *CupertinoTabBar) buildTabItems(bc widgets.BuildContext) []widgets.Widget {
	var result []widgets.Widget
	localizations := CupertinoLocalizationsOf(bc)
	for index := range w.items {
		index := index
		active := index == w.o.CurrentIndex
		var onTap func()
		if w.o.OnTap != nil {
			onTap = func() { w.o.OnTap(index) }
		}
		cursor := services.MouseCursorDefer
		if foundation.IsWeb {
			cursor = services.SystemMouseCursors.Click
		}
		result = append(result, w.wrapActiveItem(bc,
			widgets.NewExpanded(
				// Make tab items part of the EditableText tap region so that switching tabs doesn't unfocus
				// text fields.
				widgets.NewTextFieldTapRegion(
					widgets.NewSemantics(
						widgets.NewMouseRegion(
							widgets.NewGestureDetector(
								widgets.NewPadding(painting.EdgeInsets{Bottom: 4.0},
									widgets.NewColumn(w.buildSingleTabItem(w.items[index], active),
										widgets.ColumnOpts{MainAxisAlignment: rendering.MainAxisAlignmentEnd})),
								widgets.GestureDetectorOpts{Behavior: rendering.HitTestBehaviorOpaque, OnTap: onTap}),
							widgets.MouseRegionOpts{Cursor: cursor}),
						widgets.SemanticsOpts{Selected: foundation.Some(active),
							Hint: foundation.Some(localizations.TabSemanticsLabel(index+1, len(w.items)))}))),
			active))
	}
	return result
}

func (w *CupertinoTabBar) buildSingleTabItem(item widgets.BottomNavigationBarItem, active bool) []widgets.Widget {
	icon := item.Icon()
	if active {
		icon = item.ActiveIcon()
	}
	children := []widgets.Widget{widgets.NewExpanded(widgets.NewCenter(icon))}
	if item.Label() != "" {
		children = append(children, widgets.NewText(item.Label(), widgets.TextOpts{SemanticsLabel: item.SemanticsLabel()}))
	}
	return children
}

// wrapActiveItem changes the icon and the label of the active item to the active color.
func (w *CupertinoTabBar) wrapActiveItem(bc widgets.BuildContext, item widgets.Widget, active bool) widgets.Widget {
	if !active {
		return item
	}
	activeColor := w.o.ActiveColor
	if activeColor == nil {
		activeColor = CupertinoThemeOf(bc).PrimaryColor()
	}
	color := CupertinoDynamicColorResolve(activeColor, bc)
	return widgets.NewIconThemeMerge(
		widgets.NewIconThemeData(widgets.IconThemeDataOpts{Color: foundation.Some(color)}),
		widgets.NewDefaultTextStyleMerge(item, widgets.DefaultTextStyleMergeOpts{
			Style: painting.NewTextStyle(painting.TextStyleOpts{Color: foundation.Some(color)})}))
}

// CupertinoTabBarCopyOpts are the parameters of CupertinoTabBar.CopyWith: the ones that are set replace those
// of the bar.
//
// Flutter: cupertino/bottom_tab_bar.dart CupertinoTabBar.copyWith
type CupertinoTabBarCopyOpts struct {
	Key             foundation.Key
	Items           []widgets.BottomNavigationBarItem
	BackgroundColor ui.ColorValue
	ActiveColor     ui.ColorValue
	InactiveColor   ui.ColorValue
	IconSize        foundation.Opt[float64]
	Height          foundation.Opt[float64]
	Border          foundation.Opt[painting.Border]
	NoBorder        bool
	CurrentIndex    foundation.Opt[int]
	OnTap           func(int)
}

// CopyWith creates a bar like this one with the parameters that are set replaced.
//
// Flutter: cupertino/bottom_tab_bar.dart CupertinoTabBar.copyWith
func (w *CupertinoTabBar) CopyWith(c CupertinoTabBarCopyOpts) *CupertinoTabBar {
	o := w.o
	if c.Key != nil {
		o.Key = c.Key
	}
	items := w.items
	if c.Items != nil {
		items = c.Items
	}
	if c.BackgroundColor != nil {
		o.BackgroundColor = c.BackgroundColor
	}
	if c.ActiveColor != nil {
		o.ActiveColor = c.ActiveColor
	}
	if c.InactiveColor != nil {
		o.InactiveColor = c.InactiveColor
	}
	if c.IconSize.IsSome() {
		o.IconSize = c.IconSize
	}
	if c.Height.IsSome() {
		o.Height = c.Height
	}
	if c.Border.IsSome() {
		o.Border, o.NoBorder = c.Border, false
	}
	if c.NoBorder {
		o.NoBorder = true
	}
	if c.CurrentIndex.IsSome() {
		o.CurrentIndex = c.CurrentIndex.Value()
	}
	if c.OnTap != nil {
		o.OnTap = c.OnTap
	}
	return NewCupertinoTabBar(items, o)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagCupertinoTabBarItemsTooFew         = "CupertinoTabBarItemsTooFew"
	tagCupertinoTabBarCurrentIndexInvalid = "CupertinoTabBarCurrentIndexInvalid"
	tagCupertinoTabBarHeightInvalid       = "CupertinoTabBarHeightInvalid"
)
