package cupertino

import (
	"math"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// The measures of the navigation bars (the constants of cupertino/nav_bar.dart).
const (
	// navBarPersistentHeight is the height of the navigation bar without the large title.
	navBarPersistentHeight = MinInteractiveDimensionCupertino
	// navBarLargeTitleHeightExtension is the height that the large title adds to the bar.
	navBarLargeTitleHeightExtension = 52.0
	// navBarShowLargeTitleThreshold is the scroll distance before the end of the shrink at which the middle
	// shows in place of the large title.
	navBarShowLargeTitleThreshold = 10.0
	// navBarScrollUnderAnimationExtent is the scroll distance in which the background of the bar fades in.
	navBarScrollUnderAnimationExtent = 10.0
	// navBarEdgePadding is the padding at the edges of the bar.
	navBarEdgePadding = 16.0
	// navBarBottomPadding is the padding under the large title and the search field.
	navBarBottomPadding = 8.0
	// navBarBackButtonTapWidth is the least width of the back button.
	navBarBackButtonTapWidth = 50.0
	// navBarMinScaleFactor and navBarMaxScaleFactor limit the scale of the text of the bar.
	navBarMinScaleFactor = 0.9
	navBarMaxScaleFactor = 1.235
	// navBarLargeTitleScaleDampingRatio damps the scale of the large title.
	navBarLargeTitleScaleDampingRatio = 3.0
	// navBarSearchFieldCancelButtonWidth is the width of the cancel button of the search bar.
	navBarSearchFieldCancelButtonWidth = 67.0
	// navBarSearchFieldHeight is the height of the search field.
	navBarSearchFieldHeight = 36.0
	// navBarSearchDuration and navBarTitleFadeDuration are the durations of the search animation and of the
	// fade of the titles.
	navBarSearchDuration    = 300 * time.Millisecond
	navBarTitleFadeDuration = 150 * time.Millisecond
)

// navBarSearchCurve is the curve of the search animation.
var navBarSearchCurve animation.Curve = animation.Curves.EaseInOut

// navBarDefaultBorderColor is the color of the border of the bar.
var navBarDefaultBorderColor = ui.NewColor(0x4D000000)

// navBarDefaultBorder is the border of the bar: its bottom side is one physical pixel.
func navBarDefaultBorder() painting.Border {
	return painting.Border{Top: painting.BorderSideNone, Right: painting.BorderSideNone, Left: painting.BorderSideNone,
		Bottom: painting.NewBorderSide(painting.BorderSideOpts{Color: foundation.Some(navBarDefaultBorderColor), Width: foundation.Some(0.0)})}
}

// navBarTransparentBorder is the border of a bar whose background is not shown yet.
func navBarTransparentBorder() painting.Border {
	return painting.Border{Top: painting.BorderSideNone, Right: painting.BorderSideNone, Left: painting.BorderSideNone,
		Bottom: painting.NewBorderSide(painting.BorderSideOpts{Color: foundation.Some(ui.NewColor(0x00000000)), Width: foundation.Some(0.0)})}
}

// navBarHeroTag is the tag of the hero of the navigation bars of a navigator.
//
// Flutter: cupertino/nav_bar.dart _HeroTag
type navBarHeroTag struct{ navigator *widgets.NavigatorState }

// String describes the tag.
func (t navBarHeroTag) String() string {
	return "Default Hero tag for Cupertino navigation bars with navigator " + t.navigatorName()
}

func (t navBarHeroTag) navigatorName() string {
	if t.navigator == nil {
		return "null"
	}
	return "NavigatorState"
}

// effectiveBorder is the border that the bar shows: the one that is given or the default; none when the
// options say so.
func effectiveNavBarBorder(given foundation.Opt[painting.Border], none bool) foundation.Opt[painting.Border] {
	switch {
	case none:
		return foundation.None[painting.Border]()
	case given.IsSome():
		return given
	}
	return foundation.Some(navBarDefaultBorder())
}

// dampScaleFactor is the factor that scales a font size by a part of the text scale.
//
// Flutter: cupertino/nav_bar.dart _dampScaleFactor
func dampScaleFactor(scaledFontSize, unscaledFontSize, dampingRatio float64) float64 {
	scaleFactor := scaledFontSize / unscaledFontSize
	if scaleFactor < 1.0 {
		return math.Max(navBarMinScaleFactor, scaleFactor)
	}
	return 1.0 + (scaleFactor-1.0)/dampingRatio
}

// isTransitionable is whether the bar of the context takes part in the transitions between routes: it is
// in a page route that is not a full screen dialog, nor under a sheet.
//
// Flutter: cupertino/nav_bar.dart _isTransitionable
func isTransitionable(bc widgets.BuildContext) bool {
	route, ok := widgets.ModalRouteOf(bc).(widgets.PageRoute)
	return ok && route != nil && !route.FullscreenDialog() && !CupertinoSheetRouteHasParentSheet(bc)
}

// wrapWithBackground paints the background of a bar under the child: the color with the border, blurring what
// is behind it when it is not opaque, and with the style of the status bar that suits its brightness.
//
// Flutter: cupertino/nav_bar.dart _wrapWithBackground
func wrapWithBackground(border foundation.Opt[painting.Border], backgroundColor ui.Color, brightness ui.Brightness, child widgets.Widget,
	updateSystemUiOverlay, enableBackgroundFilterBlur bool) widgets.Widget {
	result := child
	if updateSystemUiOverlay {
		isDark := backgroundColor.ComputeLuminance() < 0.179
		newBrightness := brightness
		if newBrightness == ui.BrightnessUnspecified {
			newBrightness = ui.BrightnessLight
			if isDark {
				newBrightness = ui.BrightnessDark
			}
		}
		overlayStyle := services.SystemUiOverlayStyleDark
		if newBrightness == ui.BrightnessDark {
			overlayStyle = services.SystemUiOverlayStyleLight
		}
		result = widgets.NewAnnotatedRegion(services.SystemUiOverlayStyle{
			StatusBarColor:                  overlayStyle.StatusBarColor,
			StatusBarBrightness:             overlayStyle.StatusBarBrightness,
			StatusBarIconBrightness:         overlayStyle.StatusBarIconBrightness,
			SystemStatusBarContrastEnforced: overlayStyle.SystemStatusBarContrastEnforced,
		}, result)
	}
	var borderValue painting.BoxBorder
	if b, ok := border.Get(); ok {
		borderValue = b
	}
	childWithBackground := widgets.NewDecoratedBox(
		painting.NewBoxDecoration(painting.BoxDecorationOpts{Border: borderValue, Color: foundation.Some(backgroundColor)}), result)
	return widgets.NewClipRect(widgets.NewBackdropFilter(ui.NewImageFilterBlur(ui.ImageFilterBlurOpts{SigmaX: 10, SigmaY: 10}), childWithBackground,
		widgets.BackdropFilterOpts{Enabled: foundation.Some(backgroundColor.Alpha() != 0xFF && enableBackgroundFilterBlur)}))
}

// fixedSizeSlidingTransition positions a child of a fixed size at the offset of an animation, from the leading
// edge: what a bar-to-bar flight uses to slide a title.
//
// Flutter: cupertino/nav_bar.dart _FixedSizeSlidingTransition
type fixedSizeSlidingTransition struct {
	widgets.AnimatedWidget
	isLTR           bool
	width, height   float64
	offsetAnimation animation.Animation[ui.Offset]
	child           widgets.Widget
}

func newFixedSizeSlidingTransition(isLTR bool, offsetAnimation animation.Animation[ui.Offset], width, height float64, child widgets.Widget) *fixedSizeSlidingTransition {
	return &fixedSizeSlidingTransition{widgets.NewAnimatedWidgetBase(offsetAnimation), isLTR, width, height, offsetAnimation, child}
}

func (w *fixedSizeSlidingTransition) Build(widgets.BuildContext) widgets.Widget {
	offset := w.offsetAnimation.Value()
	opts := widgets.PositionedOpts{Top: foundation.Some(offset.Dy), Width: foundation.Some(w.width), Height: foundation.Some(w.height)}
	if w.isLTR {
		opts.Left = foundation.Some(offset.Dx)
	} else {
		opts.Right = foundation.Some(offset.Dx)
	}
	return widgets.NewPositioned(w.child, opts)
}

// navBarComponentsKeys are the keys of the parts of a bar, for a flight between two bars to find them.
//
// Flutter: cupertino/nav_bar.dart _NavigationBarStaticComponentsKeys
type navBarComponentsKeys struct {
	navBarBoxKey, leadingKey, backChevronKey, backLabelKey, middleKey, trailingKey, largeTitleKey, navBarBottomKey *widgets.GlobalKey[widgets.State]
}

func newNavBarComponentsKeys() *navBarComponentsKeys {
	key := func(label string) *widgets.GlobalKey[widgets.State] {
		return widgets.NewGlobalKey[widgets.State](widgets.GlobalKeyOpts{DebugLabel: label})
	}
	return &navBarComponentsKeys{key("Navigation bar render box"), key("Leading"), key("Back chevron"), key("Back label"), key("Middle"),
		key("Trailing"), key("Large title"), key("Navigation bar bottom")}
}

// navBarComponents are the parts of a bar, each under a keyed subtree: what the user gave, or what the route
// implies.
//
// Flutter: cupertino/nav_bar.dart _NavigationBarStaticComponents
type navBarComponents struct {
	leading, backChevron, backLabel, middle, trailing, largeTitle, navBarBottom *widgets.KeyedSubtree
}

// navBarComponentsOpts are the inputs of newNavBarComponents.
type navBarComponentsOpts struct {
	keys                                                              *navBarComponentsKeys
	route                                                             widgets.ModalRoute
	userLeading, userMiddle, userTrailing, userLargeTitle, userBottom widgets.Widget
	automaticallyImplyLeading, automaticallyImplyTitle                bool
	previousPageTitle                                                 foundation.Opt[string]
	padding                                                           *painting.EdgeInsetsDirectional
	large, staticBar                                                  bool
}

func newNavBarComponents(bc widgets.BuildContext, o navBarComponentsOpts) *navBarComponents {
	return &navBarComponents{
		leading:     createNavBarLeading(bc, o),
		backChevron: createNavBarBackChevron(bc, o),
		backLabel:   createNavBarBackLabel(bc, o),
		middle:      createNavBarMiddle(bc, o),
		trailing:    createNavBarTrailing(bc, o),
		largeTitle:  createNavBarLargeTitle(bc, o),
		navBarBottom: widgets.NewKeyedSubtree(widgets.NewMediaQuery(
			widgets.MediaQueryOf(bc).CopyWith(widgets.MediaQueryDataCopyOpts{TextScaler: foundation.Some(widgets.MediaQueryTextScalerOf(bc))}),
			orShrink(o.userBottom)), widgets.WidgetOpts{Key: o.keys.navBarBottomKey}),
	}
}

func orShrink(w widgets.Widget) widgets.Widget {
	if w == nil {
		return widgets.NewSizedBoxShrink(nil)
	}
	return w
}

// derivedNavBarTitle is the title that the route gives, as a Text, when the bar implies it.
//
// Flutter: cupertino/nav_bar.dart _NavigationBarStaticComponents._derivedTitle
func derivedNavBarTitle(automaticallyImplyTitle bool, route widgets.ModalRoute) widgets.Widget {
	if cr, ok := route.(CupertinoRoute); ok && automaticallyImplyTitle {
		if title, ok := cr.Title().Get(); ok {
			return widgets.NewText(title)
		}
	}
	return nil
}

func navBarClampedTextScaler(bc widgets.BuildContext) painting.TextScaler {
	return widgets.MediaQueryTextScalerOf(bc).Clamp(painting.TextScalerClampOpts{MinScaleFactor: 1.0, MaxScaleFactor: foundation.Some(navBarMaxScaleFactor)})
}

func navBarTextScaled(bc widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return widgets.NewMediaQuery(widgets.MediaQueryOf(bc).CopyWith(widgets.MediaQueryDataCopyOpts{TextScaler: foundation.Some(navBarClampedTextScaler(bc))}), child)
}

// routeCanPopAsBack is whether the route has a back button to imply: it can be popped and it is not a full
// screen dialog.
func routeImpliesBack(route widgets.ModalRoute) bool {
	if route == nil || !route.CanPop() {
		return false
	}
	if pr, ok := route.(widgets.PageRoute); ok && pr.FullscreenDialog() {
		return false
	}
	return true
}

// Flutter: cupertino/nav_bar.dart _NavigationBarStaticComponents.createLeading
func createNavBarLeading(bc widgets.BuildContext, o navBarComponentsOpts) *widgets.KeyedSubtree {
	var leadingContent widgets.Widget
	if o.userLeading != nil {
		leadingContent = o.userLeading
	} else if pr, ok := o.route.(widgets.PageRoute); ok && o.automaticallyImplyLeading && pr.CanPop() && pr.FullscreenDialog() {
		leadingContent = NewCupertinoButton(func() { pr.Navigator().MaybePop() },
			widgets.NewText(CupertinoLocalizationsOf(bc).CancelButtonLabel()), CupertinoButtonOpts{Padding: painting.EdgeInsets{}})
	}
	if leadingContent == nil {
		return nil
	}
	start := navBarEdgePadding
	if o.padding != nil {
		start = o.padding.Start
	}
	return widgets.NewKeyedSubtree(widgets.NewPadding(painting.EdgeInsetsDirectional{Start: start},
		navBarTextScaled(bc, widgets.NewIconThemeMerge(widgets.NewIconThemeData(widgets.IconThemeDataOpts{Size: foundation.Some(32.0)}), leadingContent))),
		widgets.WidgetOpts{Key: o.keys.leadingKey})
}

// Flutter: cupertino/nav_bar.dart _NavigationBarStaticComponents.createBackChevron
func createNavBarBackChevron(bc widgets.BuildContext, o navBarComponentsOpts) *widgets.KeyedSubtree {
	if o.userLeading != nil || !o.automaticallyImplyLeading || !routeImpliesBack(o.route) {
		return nil
	}
	return widgets.NewKeyedSubtree(navBarTextScaled(bc, newBackChevron()), widgets.WidgetOpts{Key: o.keys.backChevronKey})
}

// Flutter: cupertino/nav_bar.dart _NavigationBarStaticComponents.createBackLabel
func createNavBarBackLabel(bc widgets.BuildContext, o navBarComponentsOpts) *widgets.KeyedSubtree {
	if o.userLeading != nil || !o.automaticallyImplyLeading || !routeImpliesBack(o.route) {
		return nil
	}
	return widgets.NewKeyedSubtree(navBarTextScaled(bc, newBackLabel(o.previousPageTitle, o.route)), widgets.WidgetOpts{Key: o.keys.backLabelKey})
}

// Flutter: cupertino/nav_bar.dart _NavigationBarStaticComponents.createMiddle
func createNavBarMiddle(bc widgets.BuildContext, o navBarComponentsOpts) *widgets.KeyedSubtree {
	middleContent := o.userMiddle
	if o.large && o.staticBar {
		return nil
	}
	if o.large && middleContent == nil {
		middleContent = o.userLargeTitle
	}
	if middleContent == nil {
		middleContent = derivedNavBarTitle(o.automaticallyImplyTitle, o.route)
	}
	if middleContent == nil {
		return nil
	}
	return widgets.NewKeyedSubtree(navBarTextScaled(bc, middleContent), widgets.WidgetOpts{Key: o.keys.middleKey})
}

// Flutter: cupertino/nav_bar.dart _NavigationBarStaticComponents.createTrailing
func createNavBarTrailing(bc widgets.BuildContext, o navBarComponentsOpts) *widgets.KeyedSubtree {
	if o.userTrailing == nil {
		return nil
	}
	end := navBarEdgePadding
	if o.padding != nil {
		end = o.padding.End
	}
	return widgets.NewKeyedSubtree(widgets.NewPadding(painting.EdgeInsetsDirectional{End: end},
		navBarTextScaled(bc, widgets.NewIconThemeMerge(widgets.NewIconThemeData(widgets.IconThemeDataOpts{Size: foundation.Some(32.0)}), o.userTrailing))),
		widgets.WidgetOpts{Key: o.keys.trailingKey})
}

// Flutter: cupertino/nav_bar.dart _NavigationBarStaticComponents.createLargeTitle
func createNavBarLargeTitle(bc widgets.BuildContext, o navBarComponentsOpts) *widgets.KeyedSubtree {
	if !o.large {
		return nil
	}
	content := o.userLargeTitle
	if content == nil {
		content = derivedNavBarTitle(o.automaticallyImplyTitle, o.route)
	}
	if content == nil {
		panic(errors.New(tagNavBarLargeTitleMissing, "largeTitle was not provided and there was no title from the route"))
	}
	scale := dampScaleFactor(widgets.MediaQueryTextScalerOf(bc).Scale(navBarLargeTitleHeightExtension), navBarLargeTitleHeightExtension, navBarLargeTitleScaleDampingRatio)
	return widgets.NewKeyedSubtree(widgets.NewMediaQuery(
		widgets.MediaQueryOf(bc).CopyWith(widgets.MediaQueryDataCopyOpts{TextScaler: foundation.Some(painting.NewTextScalerLinear(scale))}), content),
		widgets.WidgetOpts{Key: o.keys.largeTitleKey})
}

// persistentNavigationBar is the row of the bar that stays when the large title scrolls away: the leading
// widget or the back button, the middle and the trailing widget.
//
// Flutter: cupertino/nav_bar.dart _PersistentNavigationBar
type persistentNavigationBar struct {
	widgets.StatelessWidget
	components *navBarComponents
	padding    *painting.EdgeInsetsDirectional
	// middleVisible is whether the middle shows; none (nil) keeps it always visible.
	middleVisible *bool
}

func newPersistentNavigationBar(components *navBarComponents, padding *painting.EdgeInsetsDirectional, middleVisible *bool) *persistentNavigationBar {
	return &persistentNavigationBar{widgets.NewStatelessWidgetBase(widgets.WidgetOpts{}), components, padding, middleVisible}
}

func (w *persistentNavigationBar) Build(bc widgets.BuildContext) widgets.Widget {
	var middle widgets.Widget
	if w.components.middle != nil {
		middle = widgets.NewDefaultTextStyle(CupertinoThemeOf(bc).TextTheme().NavTitleTextStyle(),
			widgets.NewSemantics(w.components.middle, widgets.SemanticsOpts{Header: foundation.Some(true)}))
		if w.middleVisible != nil {
			opacity := 0.0
			if *w.middleVisible {
				opacity = 1.0
			}
			middle = widgets.NewAnimatedOpacity(opacity, navBarTitleFadeDuration, middle)
		}
	}

	var leading widgets.Widget
	if w.components.leading != nil {
		leading = w.components.leading
	}
	if leading == nil && w.components.backChevron != nil && w.components.backLabel != nil && !CupertinoSheetRouteHasParentSheet(bc) {
		leading = newCupertinoNavigationBarBackButtonAssemble(w.components.backChevron, w.components.backLabel)
	} else {
		leading = widgets.NewAlign(leading, widgets.AlignOpts{WidthFactor: foundation.Some(1.0)})
	}

	var trailing widgets.Widget
	if w.components.trailing != nil {
		trailing = w.components.trailing
	}
	var paddedToolbar widgets.Widget = widgets.NewNavigationToolbar(widgets.NavigationToolbarOpts{
		Leading: leading, Middle: middle, Trailing: trailing, MiddleSpacing: foundation.Some(6.0)})

	if w.padding != nil {
		paddedToolbar = widgets.NewPadding(painting.EdgeInsets{Top: w.padding.Top, Bottom: w.padding.Bottom}, paddedToolbar)
	}
	hasParentSheet := CupertinoSheetRouteHasParentSheet(bc)
	return widgets.NewSizedBox(widgets.NewSafeArea(paddedToolbar, widgets.SafeAreaOpts{Top: foundation.Some(!hasParentSheet), Bottom: foundation.Some(false)}),
		widgets.SizedBoxOpts{Height: foundation.Some(navBarPersistentHeight + widgets.MediaQueryPaddingOf(bc).Top)})
}

// CupertinoNavigationBarBackButtonOpts are the optional parameters of NewCupertinoNavigationBarBackButton.
//
// Flutter: cupertino/nav_bar.dart CupertinoNavigationBarBackButton.new
type CupertinoNavigationBarBackButtonOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Color is the color of the chevron and the label; the default is the action color of the theme.
	Color ui.ColorValue
	// PreviousPageTitle is the label of the button; the default is the title of the previous route when it
	// has one, and none otherwise.
	PreviousPageTitle foundation.Opt[string]
	// OnPressed is called when the button is pressed; the default pops the route.
	OnPressed func()
}

// CupertinoNavigationBarBackButton is the back button of a navigation bar: a chevron and the title of the
// previous page. A bar that is in a route that can be popped makes one by itself.
//
// Flutter: cupertino/nav_bar.dart CupertinoNavigationBarBackButton
type CupertinoNavigationBarBackButton struct {
	widgets.StatelessWidget
	o                      CupertinoNavigationBarBackButtonOpts
	backChevron, backLabel widgets.Widget
}

// NewCupertinoNavigationBarBackButton creates a back button.
//
// Flutter: cupertino/nav_bar.dart CupertinoNavigationBarBackButton.new
func NewCupertinoNavigationBarBackButton(opts ...CupertinoNavigationBarBackButtonOpts) *CupertinoNavigationBarBackButton {
	o := firstOf(opts)
	return &CupertinoNavigationBarBackButton{widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), o, nil, nil}
}

func newCupertinoNavigationBarBackButtonAssemble(backChevron, backLabel widgets.Widget) *CupertinoNavigationBarBackButton {
	return &CupertinoNavigationBarBackButton{widgets.NewStatelessWidgetBase(widgets.WidgetOpts{}), CupertinoNavigationBarBackButtonOpts{}, backChevron, backLabel}
}

// Color is the color of the button, or nil.
//
// Flutter: cupertino/nav_bar.dart CupertinoNavigationBarBackButton.color
func (w *CupertinoNavigationBarBackButton) Color() ui.ColorValue { return w.o.Color }

// PreviousPageTitle is the label that the button was given.
//
// Flutter: cupertino/nav_bar.dart CupertinoNavigationBarBackButton.previousPageTitle
func (w *CupertinoNavigationBarBackButton) PreviousPageTitle() foundation.Opt[string] {
	return w.o.PreviousPageTitle
}

// Build builds the button. In debug builds it reports the tag NavigationBarBackButtonNotPoppable when the
// button has no OnPressed and the route cannot be popped.
//
// Flutter: cupertino/nav_bar.dart CupertinoNavigationBarBackButton.build
func (w *CupertinoNavigationBarBackButton) Build(bc widgets.BuildContext) widgets.Widget {
	currentRoute := widgets.ModalRouteOf(bc)
	if w.o.OnPressed == nil && foundation.DebugMode && (currentRoute == nil || !currentRoute.CanPop()) {
		panic(errors.New(tagNavBarBackButtonNotPoppable, "CupertinoNavigationBarBackButton should only be used in routes that can be popped"))
	}
	actionTextStyle := CupertinoThemeOf(bc).TextTheme().NavActionTextStyle()
	if w.o.Color != nil {
		if c := CupertinoDynamicColorMaybeResolve(w.o.Color, bc); c.IsSome() {
			actionTextStyle = actionTextStyle.CopyWith(painting.TextStyleCopy{Color: c})
		}
	}
	localizations := CupertinoLocalizationsOf(bc)
	var chevron, label widgets.Widget = w.backChevron, w.backLabel
	if chevron == nil {
		chevron = newBackChevron()
	}
	if label == nil {
		label = newBackLabel(w.o.PreviousPageTitle, currentRoute)
	}
	return NewCupertinoButton(func() {
		if w.o.OnPressed != nil {
			w.o.OnPressed()
		} else {
			widgets.NavigatorMaybePop(bc)
		}
	}, widgets.NewSemantics(
		widgets.NewDefaultTextStyle(actionTextStyle,
			widgets.NewConstrainedBox(rendering.BoxConstraints{MinWidth: navBarBackButtonTapWidth, MaxWidth: infinity, MaxHeight: infinity},
				widgets.NewRow([]widgets.Widget{
					widgets.NewPadding(painting.EdgeInsetsDirectional{Start: 8.0}, nil),
					chevron,
					widgets.NewPadding(painting.EdgeInsetsDirectional{Start: 6.0}, nil),
					widgets.NewFlexible(label),
				}, widgets.RowOpts{MainAxisSize: rendering.MainAxisSizeMin}))),
		widgets.SemanticsOpts{Container: true, ExcludeSemantics: true, Label: foundation.Some(localizations.BackButtonLabel()), Button: foundation.Some(true)}),
		CupertinoButtonOpts{Padding: painting.EdgeInsets{}})
}

// backChevron is the chevron of the back button: the glyph of CupertinoIcons.back, mirrored in right to left
// text.
//
// Flutter: cupertino/nav_bar.dart _BackChevron
type backChevron struct{ widgets.StatelessWidget }

func newBackChevron() *backChevron {
	return &backChevron{widgets.NewStatelessWidgetBase(widgets.WidgetOpts{})}
}

func (w *backChevron) Build(bc widgets.BuildContext) widgets.Widget {
	textDirection := widgets.DirectionalityOf(bc)
	textStyle := widgets.DefaultTextStyleOf(bc).Style()
	back := CupertinoIcons.Back()
	widgets.IconFontEnsure(bc, back)

	var iconWidget widgets.Widget = widgets.NewPadding(painting.EdgeInsetsDirectional{Start: 6, End: 2},
		widgets.NewTextRich(painting.NewTextSpan(painting.TextSpanOpts{
			Text: foundation.Some(string(rune(back.CodePoint))),
			Style: painting.NewTextStyle(painting.TextStyleOpts{Inherit: foundation.Some(false), Color: textStyle.Color, FontSize: foundation.Some(30.0),
				FontFamily: back.FontFamily, Package: back.FontPackage}),
		})))
	if textDirection == ui.TextDirectionRTL {
		iconWidget = widgets.NewTransform(ui.NewMatrix4Diagonal3Values(-1, 1, 1), iconWidget,
			widgets.TransformOpts{Alignment: painting.AlignmentCenter, TransformHitTests: foundation.Some(false)})
	}
	return widgets.NewKeyedSubtree(iconWidget, widgets.WidgetOpts{Key: widgets.StandardComponentTypeBackButton.Key()})
}

// backLabel is the label of the back button: the title of the previous page, or "Back" when it is long.
//
// Flutter: cupertino/nav_bar.dart _BackLabel
type backLabel struct {
	widgets.StatelessWidget
	specifiedPreviousTitle foundation.Opt[string]
	route                  widgets.ModalRoute
}

func newBackLabel(specifiedPreviousTitle foundation.Opt[string], route widgets.ModalRoute) *backLabel {
	return &backLabel{widgets.NewStatelessWidgetBase(widgets.WidgetOpts{}), specifiedPreviousTitle, route}
}

func buildPreviousTitleWidget(bc widgets.BuildContext, previousTitle foundation.Opt[string]) widgets.Widget {
	title, ok := previousTitle.Get()
	if !ok {
		return widgets.NewSizedBoxShrink(nil)
	}
	textWidget := widgets.NewText(title, widgets.TextOpts{MaxLines: foundation.Some(1), Overflow: painting.TextOverflowEllipsis})
	if len([]rune(title)) > 12 {
		textWidget = widgets.NewText(CupertinoLocalizationsOf(bc).BackButtonLabel())
	}
	return widgets.NewAlign(textWidget, widgets.AlignOpts{Alignment: painting.AlignmentDirectionalCenterStart, WidthFactor: foundation.Some(1.0)})
}

func (w *backLabel) Build(bc widgets.BuildContext) widgets.Widget {
	if w.specifiedPreviousTitle.IsSome() {
		return buildPreviousTitleWidget(bc, w.specifiedPreviousTitle)
	}
	if cr, ok := w.route.(CupertinoRoute); ok && !w.route.IsFirst() {
		return widgets.NewValueListenableBuilder(cr.PreviousTitle(), func(bc widgets.BuildContext, previousTitle foundation.Opt[string], _ widgets.Widget) widgets.Widget {
			return buildPreviousTitleWidget(bc, previousTitle)
		}, nil)
	}
	return widgets.NewSizedBoxShrink(nil)
}

// cancelButton is the button that ends a search: "Cancel", at its natural width.
//
// Flutter: cupertino/nav_bar.dart _CancelButton
type cancelButton struct {
	widgets.StatelessWidget
	opacity   float64
	onPressed func()
}

func newCancelButton(opacity float64, onPressed func()) *cancelButton {
	return &cancelButton{widgets.NewStatelessWidgetBase(widgets.WidgetOpts{}), opacity, onPressed}
}

func (w *cancelButton) Build(bc widgets.BuildContext) widgets.Widget {
	localizations := CupertinoLocalizationsOf(bc)
	return widgets.NewMediaQueryWithNoTextScaling(widgets.NewAlign(
		widgets.NewOpacity(w.opacity, NewCupertinoButton(w.onPressed,
			widgets.NewText(localizations.CancelButtonLabel(), widgets.TextOpts{MaxLines: foundation.Some(1), Overflow: painting.TextOverflowClip}),
			CupertinoButtonOpts{Padding: painting.EdgeInsets{}})),
		widgets.AlignOpts{Alignment: painting.AlignmentCenterLeft}))
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagNavBarLargeTitleMissing     = "NavigationBarLargeTitleMissing"
	tagNavBarBackButtonNotPoppable = "NavigationBarBackButtonNotPoppable"
)
