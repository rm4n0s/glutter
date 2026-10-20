package material

import (
	"time"

	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

const (
	// MinInteractiveDimension is the minimum dimension of any interactive
	// region according to Material guidelines.
	//
	// Flutter: material/constants.dart kMinInteractiveDimension
	MinInteractiveDimension = 48.0
	// ToolbarHeight is the height of the toolbar component of the AppBar.
	//
	// Flutter: material/constants.dart kToolbarHeight
	ToolbarHeight = 56.0
	// BottomNavigationBarHeight is the height of a BottomNavigationBar.
	//
	// Flutter: material/constants.dart kBottomNavigationBarHeight
	BottomNavigationBarHeight = 56.0
	// TextTabBarHeight is the height of a TabBar with only text labels.
	//
	// Flutter: material/constants.dart kTextTabBarHeight
	TextTabBarHeight = MinInteractiveDimension
	// ThemeChangeDuration is the duration of the animation that shows a change
	// of theme.
	//
	// Flutter: material/constants.dart kThemeChangeDuration
	ThemeChangeDuration = 200 * time.Millisecond
	// RadialReactionRadius is the radius of the radial reaction over a
	// Switch, Checkbox or Radio.
	//
	// Flutter: material/constants.dart kRadialReactionRadius
	RadialReactionRadius = 20.0
	// RadialReactionDuration is the duration of the radial reaction.
	//
	// Flutter: material/constants.dart kRadialReactionDuration
	RadialReactionDuration = 100 * time.Millisecond
	// RadialReactionAlpha is the alpha of the radial reaction.
	//
	// Flutter: material/constants.dart kRadialReactionAlpha
	RadialReactionAlpha = 0x1F
	// TabScrollDuration is the duration of the scroll that brings a tab into
	// view.
	//
	// Flutter: material/constants.dart kTabScrollDuration
	TabScrollDuration = 300 * time.Millisecond
)

var (
	// TabLabelPadding is the padding added around the tab label.
	//
	// Flutter: material/constants.dart kTabLabelPadding
	TabLabelPadding = painting.NewEdgeInsetsSymmetric(0, 16)
	// MaterialListPadding is the padding added around material list items.
	//
	// Flutter: material/constants.dart kMaterialListPadding
	MaterialListPadding = painting.NewEdgeInsetsSymmetric(8, 0)
	// DefaultIconLightColor is the default color of icons on a dark surface.
	//
	// Flutter: material/constants.dart kDefaultIconLightColor
	DefaultIconLightColor = ui.NewColor(0xFFFFFFFF)
	// DefaultIconDarkColor is the default color of icons on a light surface.
	//
	// Flutter: material/constants.dart kDefaultIconDarkColor
	DefaultIconDarkColor = ui.NewColor(0xDD000000)
)
