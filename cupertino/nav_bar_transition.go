package cupertino

import (
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// The curves of the parts of the bottom and the top bar that slide in and out during a flight.
var (
	topNavBarHeaderTransitionCurve    animation.Curve = animation.NewCubic(0.0, 0.45, 0.45, 0.98)
	bottomNavBarHeaderTransitionCurve animation.Curve = animation.NewCubic(0.05, 0.90, 0.90, 0.95)
)

// transitionableNavigationBarOpts are the inputs of newTransitionableNavigationBar.
type transitionableNavigationBarOpts struct {
	componentsKeys                           *navBarComponentsKeys
	backgroundColor                          foundation.Opt[ui.Color]
	backButtonTextStyle, titleTextStyle      *painting.TextStyle
	largeTitleTextStyle                      *painting.TextStyle
	border                                   foundation.Opt[painting.Border]
	hasUserMiddle, largeExpanded, searchable bool
	automaticBackgroundVisibility            bool
	child                                    widgets.Widget
}

// transitionableNavigationBar marks a navigation bar as an end of a flight: it is the child of a Hero, and it
// carries what the flight needs to know about the bar. Its key is the one of the box of the bar.
//
// Flutter: cupertino/nav_bar.dart _TransitionableNavigationBar
type transitionableNavigationBar struct {
	widgets.StatelessWidget
	o transitionableNavigationBarOpts
}

func newTransitionableNavigationBar(o transitionableNavigationBarOpts) *transitionableNavigationBar {
	if foundation.DebugMode && o.largeExpanded && o.largeTitleTextStyle == nil {
		panic(errors.New(tagNavBarLargeTitleStyleMissing, "a bar with its large title expanded needs the style of the large title"))
	}
	return &transitionableNavigationBar{widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.componentsKeys.navBarBoxKey}), o}
}

// renderBox is the render box of the bar, that is laid out and painted when a flight shuttle is built.
//
// Flutter: cupertino/nav_bar.dart _TransitionableNavigationBar.renderBox
func (w *transitionableNavigationBar) renderBox() rendering.Box {
	return w.o.componentsKeys.navBarBoxKey.CurrentContext().FindRenderObject().(rendering.Box)
}

// userGestureInProgress is whether the navigator of the bar is in a gesture (the swipe back).
//
// Flutter: cupertino/nav_bar.dart _TransitionableNavigationBar.userGestureInProgress
func (w *transitionableNavigationBar) userGestureInProgress() bool {
	return widgets.NavigatorOf(w.o.componentsKeys.navBarBoxKey.CurrentContext()).UserGestureInProgress()
}

// Build is the child: the bar. In debug builds it reports the tag NavigationBarTransitionableOutsideHero when
// the bar is not the child of a Hero, and NavigationBarTransitionableInFlight when it is inside a flight.
//
// Flutter: cupertino/nav_bar.dart _TransitionableNavigationBar.build
func (w *transitionableNavigationBar) Build(bc widgets.BuildContext) widgets.Widget {
	if foundation.DebugMode {
		inHero := false
		bc.VisitAncestorElements(func(ancestor widgets.Element) bool {
			if _, ok := ancestor.Widget().(navBarFlight); ok {
				panic(errors.New(tagNavBarTransitionableInFlight, "a transitionable bar should never re-appear inside a bar transition; keyed bars only serve as anchor points in routes rather than appearing inside hero flights themselves"))
			}
			if _, ok := ancestor.Widget().(*widgets.Hero); ok {
				inHero = true
			}
			return true
		})
		if !inHero {
			panic(errors.New(tagNavBarTransitionableOutsideHero, "a transitionable bar should only be added as the immediate child of Hero widgets"))
		}
	}
	return w.o.child
}

// navBarFlight marks the widgets that are a flight between bars: a transitionable bar is never inside one.
type navBarFlight interface{ isNavBarFlight() }

func (*navBarTransition) isNavBarFlight() {}

// navBarTransition is the widget that flies between two navigation bars: the backgrounds and the parts of
// the bars, each moving and fading as Cupertino does.
//
// Flutter: cupertino/nav_bar.dart _NavigationBarTransition
type navBarTransition struct {
	widgets.StatelessWidget
	animation    animation.Animation[float64]
	topNavBar    *transitionableNavigationBar
	bottomNavBar *transitionableNavigationBar
	heightBegin  float64
	heightEnd    float64
}

func newNavBarTransition(a animation.Animation[float64], topNavBar, bottomNavBar *transitionableNavigationBar) *navBarTransition {
	return &navBarTransition{widgets.NewStatelessWidgetBase(widgets.WidgetOpts{}), a, topNavBar, bottomNavBar,
		bottomNavBar.renderBox().Size().Height, topNavBar.renderBox().Size().Height}
}

func (w *navBarTransition) Build(bc widgets.BuildContext) widgets.Widget {
	c := newNavBarComponentsTransition(w.animation, w.bottomNavBar, w.topNavBar, widgets.DirectionalityOf(bc))
	var children []widgets.Widget
	for _, child := range []widgets.Widget{
		c.bottomNavBarBackground(), c.bottomBackChevron(), c.bottomBackLabel(), c.bottomLeading(), c.bottomMiddle(),
		c.bottomLargeTitle(), c.bottomTrailing(), c.bottomNavBarBottom(),
		c.topNavBarBackground(), c.topLeading(), c.topBackChevron(), c.topBackLabel(), c.topMiddle(),
		c.topLargeTitle(), c.topTrailing(), c.topNavBarBottom(),
	} {
		if child != nil {
			children = append(children, child)
		}
	}
	return widgets.NewMediaQueryWithNoTextScaling(widgets.NewSizedBox(widgets.NewStack(children), widgets.SizedBoxOpts{
		Height: foundation.Some(math.Max(w.heightBegin, w.heightEnd) + widgets.MediaQueryPaddingOf(bc).Top), Width: foundation.Some(infinity)}))
}

// navBarComponentsTransition builds the parts of a flight between two bars.
//
// Flutter: cupertino/nav_bar.dart _NavigationBarComponentsTransition
type navBarComponentsTransition struct {
	animation                                         animation.Animation[float64]
	bottomComponents, topComponents                   *navBarComponentsKeys
	bottomNavBarBox, topNavBarBox                     rendering.Box
	bottomBackButtonTextStyle, topBackButtonTextStyle *painting.TextStyle
	bottomTitleTextStyle, topTitleTextStyle           *painting.TextStyle
	bottomLargeTitleTextStyle, topLargeTitleTextStyle *painting.TextStyle
	bottomHasUserMiddle, topHasUserMiddle             bool
	bottomLargeExpanded, topLargeExpanded             bool
	userGestureInProgress                             bool
	searchable                                        bool
	bottomAutomaticBackgroundVisibility               bool
	bottomBackgroundColor, topBackgroundColor         foundation.Opt[ui.Color]
	bottomBorder, topBorder                           foundation.Opt[painting.Border]
	transitionBox                                     ui.Rect
	forwardDirection                                  float64
}

func newNavBarComponentsTransition(a animation.Animation[float64], bottomNavBar, topNavBar *transitionableNavigationBar, directionality ui.TextDirection) *navBarComponentsTransition {
	bottomBox, topBox := bottomNavBar.renderBox(), topNavBar.renderBox()
	t := &navBarComponentsTransition{
		animation: a, bottomComponents: bottomNavBar.o.componentsKeys, topComponents: topNavBar.o.componentsKeys,
		bottomNavBarBox: bottomBox, topNavBarBox: topBox,
		bottomBackButtonTextStyle: bottomNavBar.o.backButtonTextStyle, topBackButtonTextStyle: topNavBar.o.backButtonTextStyle,
		bottomTitleTextStyle: bottomNavBar.o.titleTextStyle, topTitleTextStyle: topNavBar.o.titleTextStyle,
		bottomLargeTitleTextStyle: bottomNavBar.o.largeTitleTextStyle, topLargeTitleTextStyle: topNavBar.o.largeTitleTextStyle,
		bottomHasUserMiddle: bottomNavBar.o.hasUserMiddle, topHasUserMiddle: topNavBar.o.hasUserMiddle,
		bottomLargeExpanded: bottomNavBar.o.largeExpanded, topLargeExpanded: topNavBar.o.largeExpanded,
		bottomBackgroundColor: bottomNavBar.o.backgroundColor, topBackgroundColor: topNavBar.o.backgroundColor,
		bottomBorder: bottomNavBar.o.border, topBorder: topNavBar.o.border,
		bottomAutomaticBackgroundVisibility: bottomNavBar.o.automaticBackgroundVisibility,
		userGestureInProgress:               topNavBar.userGestureInProgress() || bottomNavBar.userGestureInProgress(),
		searchable:                          topNavBar.o.searchable && bottomNavBar.o.searchable,
		transitionBox:                       bottomBox.PaintBounds().ExpandToInclude(topBox.PaintBounds()),
		forwardDirection:                    1.0,
	}
	if directionality != ui.TextDirectionLTR {
		t.forwardDirection = -1.0
	}
	return t
}

var (
	fadeOutTween = animation.NewTween(1.0, 0.0)
	fadeInTween  = animation.NewTween(0.0, 1.0)
)

// positionInTransitionBox is the rect of the part under the key, in the transition box, with respect to the
// bar.
//
// Flutter: cupertino/nav_bar.dart _NavigationBarComponentsTransition.positionInTransitionBox
func (c *navBarComponentsTransition) positionInTransitionBox(key *widgets.GlobalKey[widgets.State], from rendering.Box) rendering.RelativeRect {
	componentBox := key.CurrentContext().FindRenderObject().(rendering.Box)
	origin := componentBox.LocalToGlobal(ui.OffsetZero, from)
	return rendering.NewRelativeRectFromRect(ui.NewRectFromLTWH(origin.Dx, origin.Dy, componentBox.Size().Width, componentBox.Size().Height), c.transitionBox)
}

func renderBoxOf(key *widgets.GlobalKey[widgets.State]) rendering.Box {
	return key.CurrentContext().FindRenderObject().(rendering.Box)
}

// slideFromLeadingEdge moves a widget of a fixed size, from the leading edge of a part of the bar to the leading
// edge of a part of the other.
//
// Flutter: cupertino/nav_bar.dart _NavigationBarComponentsTransition.slideFromLeadingEdge
func (c *navBarComponentsTransition) slideFromLeadingEdge(fromKey *widgets.GlobalKey[widgets.State], fromNavBarBox rendering.Box,
	toKey *widgets.GlobalKey[widgets.State], toNavBarBox rendering.Box, curve animation.Curve, child widgets.Widget) widgets.Widget {
	fromBox, toBox := renderBoxOf(fromKey), renderBoxOf(toKey)
	isLTR := c.forwardDirection > 0

	fromAnchorLocal := ui.Offset{Dy: fromBox.Size().Height / 2}
	toAnchorLocal := ui.Offset{Dy: toBox.Size().Height / 2}
	if !isLTR {
		fromAnchorLocal.Dx, toAnchorLocal.Dx = fromBox.Size().Width, toBox.Size().Width
	}
	fromAnchorInFromBox := fromBox.LocalToGlobal(fromAnchorLocal, fromNavBarBox)
	toAnchorInToBox := toBox.LocalToGlobal(toAnchorLocal, toNavBarBox)

	// We can't get the anchor positions in the transition box by comparing the positions of the boxes in the
	// local coordinate systems, because the boxes are not in the same coordinate system (the bars are not
	// placed in the same place), but a translation between the anchors in the bars does work.
	var translation ui.Offset
	if isLTR {
		translation = toAnchorInToBox.Sub(fromAnchorInFromBox)
	} else {
		translation = ui.Offset{Dx: toNavBarBox.Size().Width - toAnchorInToBox.Dx, Dy: toAnchorInToBox.Dy}.
			Sub(ui.Offset{Dx: fromNavBarBox.Size().Width - fromAnchorInFromBox.Dx, Dy: fromAnchorInFromBox.Dy})
	}

	fromBoxMargin := c.positionInTransitionBox(fromKey, fromNavBarBox)
	fromOriginInTransitionBox := ui.Offset{Dx: fromBoxMargin.Right, Dy: fromBoxMargin.Top}
	if isLTR {
		fromOriginInTransitionBox.Dx = fromBoxMargin.Left
	}
	anchorMovement := animation.NewTween(fromOriginInTransitionBox, fromOriginInTransitionBox.Add(translation)).SetName("Offset")
	return newFixedSizeSlidingTransition(isLTR,
		animation.Drive[ui.Offset](animation.Drive[float64](c.animation, animation.NewCurveTween(curve)), anchorMovement),
		fromNavBarBox.Size().Width, fromBox.Size().Height, child)
}

func (c *navBarComponentsTransition) fadeInFrom(t float64, curve ...animation.Curve) animation.Animation[float64] {
	var cv animation.Curve = animation.Curves.EaseIn
	if len(curve) > 0 {
		cv = curve[0]
	}
	return animation.Drive[float64](c.animation, fadeInTween.Chain(animation.NewCurveTween(animation.NewInterval(t, 1.0, cv))))
}

func (c *navBarComponentsTransition) fadeOutBy(t float64, curve ...animation.Curve) animation.Animation[float64] {
	var cv animation.Curve = animation.Curves.EaseOut
	if len(curve) > 0 {
		cv = curve[0]
	}
	return animation.Drive[float64](c.animation, fadeOutTween.Chain(animation.NewCurveTween(animation.NewInterval(0.0, t, cv))))
}

// routeAnimation is the animation of the route that the flight follows, before the curve of the hero.
//
// Flutter: cupertino/nav_bar.dart _NavigationBarComponentsTransition.routeAnimation
func (c *navBarComponentsTransition) routeAnimation() animation.Animation[float64] {
	return c.animation.(*animation.CurvedAnimation).Parent
}

func drive(a animation.Animation[float64], curve animation.Curve) animation.Animation[float64] {
	return animation.Drive[float64](a, animation.NewCurveTween(curve))
}

func (c *navBarComponentsTransition) isForward() bool {
	return c.animation.Status() == animation.AnimationStatusForward
}

func relativeRectTween(begin, end rendering.RelativeRect) *animation.Tween[rendering.RelativeRect] {
	return widgets.NewRelativeRectTween(begin, end)
}

func keyedChild(key *widgets.GlobalKey[widgets.State]) *widgets.KeyedSubtree {
	if k, ok := key.CurrentWidget().(*widgets.KeyedSubtree); ok {
		return k
	}
	return nil
}

func (c *navBarComponentsTransition) bottomNavBarBackground() widgets.Widget {
	if c.bottomBackgroundColor.IsNone() || (c.bottomLargeExpanded && c.bottomAutomaticBackgroundVisibility) {
		return nil
	}
	var animationCurve animation.Curve = animation.Curves.FastEaseInToSlowEaseOut
	if !c.isForward() {
		animationCurve = animation.Curves.FastEaseInToSlowEaseOut.Flipped()
	}
	curve := animationCurve
	if c.userGestureInProgress {
		curve = animation.Curves.Linear
	}
	pageTransitionAnimation := drive(c.routeAnimation(), curve)
	from := c.positionInTransitionBox(c.bottomComponents.navBarBoxKey, c.bottomNavBarBox)
	positionTween := relativeRectTween(from, from.Shift(ui.Offset{Dx: c.forwardDirection * -c.bottomNavBarBox.Size().Width}))
	return widgets.NewPositionedTransition(animation.Drive[rendering.RelativeRect](pageTransitionAnimation, positionTween),
		wrapWithBackground(c.topBorder, c.bottomBackgroundColor.Value(), ui.BrightnessUnspecified,
			widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(c.bottomNavBarBox.Size().Height), Width: foundation.Some(infinity)}), false, true))
}

func (c *navBarComponentsTransition) bottomLeading() widgets.Widget {
	leading := keyedChild(c.bottomComponents.leadingKey)
	if leading == nil {
		return nil
	}
	return widgets.NewPositionedFromRelativeRect(c.positionInTransitionBox(c.bottomComponents.leadingKey, c.bottomNavBarBox),
		widgets.NewFadeTransition(c.fadeOutBy(0.4), leading.Child()))
}

func (c *navBarComponentsTransition) bottomBackChevron() widgets.Widget {
	chevron := keyedChild(c.bottomComponents.backChevronKey)
	if chevron == nil {
		return nil
	}
	return widgets.NewPositionedFromRelativeRect(c.positionInTransitionBox(c.bottomComponents.backChevronKey, c.bottomNavBarBox),
		widgets.NewFadeTransition(c.fadeOutBy(0.6), widgets.NewDefaultTextStyle(c.bottomBackButtonTextStyle, chevron.Child())))
}

func (c *navBarComponentsTransition) bottomBackLabel() widgets.Widget {
	label := keyedChild(c.bottomComponents.backLabelKey)
	if label == nil {
		return nil
	}
	from := c.positionInTransitionBox(c.bottomComponents.backLabelKey, c.bottomNavBarBox)
	positionTween := relativeRectTween(from, from.Shift(ui.Offset{Dx: c.forwardDirection * (-c.bottomNavBarBox.Size().Width / 2.0)}))
	return widgets.NewPositionedTransition(animation.Drive[rendering.RelativeRect](c.animation, positionTween),
		widgets.NewFadeTransition(c.fadeOutBy(0.2), widgets.NewDefaultTextStyle(c.bottomBackButtonTextStyle, label.Child())))
}

func (c *navBarComponentsTransition) bottomMiddle() widgets.Widget {
	middle := keyedChild(c.bottomComponents.middleKey)
	topBackLabel := keyedChild(c.topComponents.backLabelKey)
	topLeading := keyedChild(c.topComponents.leadingKey)

	// The middle component is non-null when the nav bar is a large title nav bar but would switch to the
	// normal title in the first half of the animation when going forward.
	if !c.bottomHasUserMiddle && c.bottomLargeExpanded {
		return nil
	}
	fade := 0.7
	if c.bottomHasUserMiddle {
		fade = 0.4
	}
	if middle != nil && topBackLabel != nil {
		return c.slideFromLeadingEdge(c.bottomComponents.middleKey, c.bottomNavBarBox, c.topComponents.backLabelKey, c.topNavBarBox,
			animation.NewInterval(0.0, 1.0),
			widgets.NewFadeTransition(c.fadeOutBy(fade), widgets.NewAlign(
				widgets.NewDefaultTextStyleTransition(animation.Drive[*painting.TextStyle](c.animation,
					widgets.NewTextStyleTween(c.bottomTitleTextStyle, c.topBackButtonTextStyle)), middle.Child()),
				widgets.AlignOpts{Alignment: painting.AlignmentDirectionalCenterStart})))
	}
	// When the top page has a leading widget override (one of the nav bars' leading widgets are overridden by
	// the user), the middle fades out in place.
	if middle != nil && topLeading != nil {
		return widgets.NewPositionedFromRelativeRect(c.positionInTransitionBox(c.bottomComponents.middleKey, c.bottomNavBarBox),
			widgets.NewFadeTransition(c.fadeOutBy(fade), widgets.NewDefaultTextStyle(c.bottomTitleTextStyle, middle.Child())))
	}
	return nil
}

func (c *navBarComponentsTransition) bottomLargeTitle() widgets.Widget {
	largeTitle := keyedChild(c.bottomComponents.largeTitleKey)
	topBackLabel := keyedChild(c.topComponents.backLabelKey)
	if largeTitle == nil || !c.bottomLargeExpanded {
		return nil
	}
	if topBackLabel != nil {
		end := 1.0
		if c.isForward() {
			end = 0.7
		}
		return c.slideFromLeadingEdge(c.bottomComponents.largeTitleKey, c.bottomNavBarBox, c.topComponents.backLabelKey, c.topNavBarBox,
			animation.NewInterval(0.0, end),
			widgets.NewFadeTransition(c.fadeOutBy(0.6), widgets.NewAlign(
				widgets.NewDefaultTextStyleTransition(animation.Drive[*painting.TextStyle](c.animation,
					widgets.NewTextStyleTween(c.bottomLargeTitleTextStyle, c.topBackButtonTextStyle)),
					largeTitle.Child(), widgets.DefaultTextStyleTransitionOpts{MaxLines: foundation.Some(1), Overflow: painting.TextOverflowEllipsis}),
				widgets.AlignOpts{Alignment: painting.AlignmentDirectionalCenterStart})))
	}
	// Large title fades out and moves out of the screen if no back button label to morph into.
	from := c.positionInTransitionBox(c.bottomComponents.largeTitleKey, c.bottomNavBarBox)
	positionTween := relativeRectTween(from, from.Shift(ui.Offset{Dx: c.forwardDirection * c.bottomNavBarBox.Size().Width / 4.0}))
	return widgets.NewPositionedTransition(animation.Drive[rendering.RelativeRect](c.animation, positionTween),
		widgets.NewFadeTransition(c.fadeOutBy(0.4), widgets.NewDefaultTextStyle(c.bottomLargeTitleTextStyle, largeTitle.Child())))
}

func (c *navBarComponentsTransition) bottomTrailing() widgets.Widget {
	trailing := keyedChild(c.bottomComponents.trailingKey)
	if trailing == nil {
		return nil
	}
	return widgets.NewPositionedFromRelativeRect(c.positionInTransitionBox(c.bottomComponents.trailingKey, c.bottomNavBarBox),
		widgets.NewFadeTransition(c.fadeOutBy(0.6), trailing.Child()))
}

func (c *navBarComponentsTransition) bottomNavBarBottom() widgets.Widget {
	bottom := keyedChild(c.bottomComponents.navBarBottomKey)
	if bottom == nil {
		return nil
	}
	from := c.positionInTransitionBox(c.bottomComponents.navBarBottomKey, c.bottomNavBarBox)
	positionTween := relativeRectTween(from, from.Shift(ui.Offset{Dx: c.forwardDirection * -c.bottomNavBarBox.Size().Width}))
	child := bottom.Child()
	animationCurve := bottomNavBarHeaderTransitionCurve
	if !c.isForward() {
		animationCurve = bottomNavBarHeaderTransitionCurve.Flipped()
	}
	if !c.searchable {
		child = widgets.NewFadeTransition(c.fadeOutBy(0.8, animationCurve), child)
	}
	return widgets.NewPositionedTransition(c.positionAnimation(positionTween, animationCurve), widgets.NewClipRect(child))
}

// positionAnimation is the animation of a position: it follows the route animation linearly during a user
// gesture, and the flight animation with the curve otherwise.
func (c *navBarComponentsTransition) positionAnimation(tween *animation.Tween[rendering.RelativeRect], curve animation.Curve) animation.Animation[rendering.RelativeRect] {
	if c.userGestureInProgress {
		return animation.Drive[rendering.RelativeRect](drive(c.routeAnimation(), animation.Curves.Linear), tween)
	}
	return animation.Drive[rendering.RelativeRect](drive(c.animation, curve), tween)
}

func (c *navBarComponentsTransition) topNavBarBackground() widgets.Widget {
	if c.topBackgroundColor.IsNone() {
		return nil
	}
	var animationCurve animation.Curve = animation.Curves.FastEaseInToSlowEaseOut
	if !c.isForward() {
		animationCurve = animation.Curves.FastEaseInToSlowEaseOut.Flipped()
	}
	curve := animationCurve
	if c.userGestureInProgress {
		curve = animation.Curves.Linear
	}
	pageTransitionAnimation := drive(c.routeAnimation(), curve)
	to := c.positionInTransitionBox(c.topComponents.navBarBoxKey, c.topNavBarBox)
	positionTween := relativeRectTween(to.Shift(ui.Offset{Dx: c.forwardDirection * c.topNavBarBox.Size().Width}), to)
	return widgets.NewPositionedTransition(animation.Drive[rendering.RelativeRect](pageTransitionAnimation, positionTween),
		wrapWithBackground(c.topBorder, c.topBackgroundColor.Value(), ui.BrightnessUnspecified,
			widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(c.topNavBarBox.Size().Height), Width: foundation.Some(infinity)}), false, true))
}

func (c *navBarComponentsTransition) topLeading() widgets.Widget {
	leading := keyedChild(c.topComponents.leadingKey)
	if leading == nil {
		return nil
	}
	return widgets.NewPositionedFromRelativeRect(c.positionInTransitionBox(c.topComponents.leadingKey, c.topNavBarBox),
		widgets.NewFadeTransition(c.fadeInFrom(0.6), leading.Child()))
}

func (c *navBarComponentsTransition) topBackChevron() widgets.Widget {
	topChevron := keyedChild(c.topComponents.backChevronKey)
	bottomChevron := keyedChild(c.bottomComponents.backChevronKey)
	if topChevron == nil {
		return nil
	}
	to := c.positionInTransitionBox(c.topComponents.backChevronKey, c.topNavBarBox)
	from := to
	child := topChevron.Child()

	// If it's the first page with a back chevron, move it in from the right (or left in RTL) and scale it.
	var effectiveScaleCurve, effectivePositionCurve animation.Curve
	if c.isForward() {
		effectiveScaleCurve, effectivePositionCurve = animation.NewInterval(0.0, 0.2), animation.NewInterval(0.0, 0.5)
	} else {
		effectiveScaleCurve, effectivePositionCurve = animation.NewInterval(0.8, 1.0), animation.NewInterval(0.5, 1.0)
	}
	if bottomChevron == nil {
		topChevronBox := renderBoxOf(c.topComponents.backChevronKey)
		from = to.Shift(ui.Offset{Dx: c.forwardDirection * topChevronBox.Size().Width * 2.0})
		child = widgets.NewScaleTransition(drive(c.routeAnimation(), effectiveScaleCurve), child)
	}
	positionTween := relativeRectTween(from, to)
	begin := 0.4
	if bottomChevron == nil && !c.isForward() {
		begin = 0.9
	}
	return widgets.NewPositionedTransition(animation.Drive[rendering.RelativeRect](drive(c.routeAnimation(), effectivePositionCurve), positionTween),
		widgets.NewFadeTransition(drive(c.routeAnimation(), animation.NewInterval(begin, 1.0)), widgets.NewDefaultTextStyle(c.topBackButtonTextStyle, child)))
}

func (c *navBarComponentsTransition) topBackLabel() widgets.Widget {
	bottomMiddle := keyedChild(c.bottomComponents.middleKey)
	bottomLargeTitle := keyedChild(c.bottomComponents.largeTitleKey)
	topBackLabel := keyedChild(c.topComponents.backLabelKey)
	if topBackLabel == nil {
		return nil
	}
	// The label may be in the middle of fading (when the user taps back during the transition), take its
	// opacity into account.
	var midClickOpacity animation.Animation[float64]
	if opacity, ok := widgets.FindAncestorRenderObjectOfType[*rendering.RenderAnimatedOpacity](c.topComponents.backLabelKey.CurrentContext()); ok && opacity.Opacity().Value() < 1.0 {
		midClickOpacity = animation.Drive[float64](c.animation, animation.NewTween(0.0, opacity.Opacity().Value()))
	}
	pick := func(fallback animation.Animation[float64]) animation.Animation[float64] {
		if midClickOpacity != nil {
			return midClickOpacity
		}
		return fallback
	}
	if bottomLargeTitle != nil && c.bottomLargeExpanded {
		end := 1.0
		if c.isForward() {
			end = 0.7
		}
		return c.slideFromLeadingEdge(c.bottomComponents.largeTitleKey, c.bottomNavBarBox, c.topComponents.backLabelKey, c.topNavBarBox,
			animation.NewInterval(0.0, end),
			widgets.NewFadeTransition(pick(c.fadeInFrom(0.4)), widgets.NewDefaultTextStyleTransition(
				animation.Drive[*painting.TextStyle](c.animation, widgets.NewTextStyleTween(c.bottomLargeTitleTextStyle, c.topBackButtonTextStyle)),
				topBackLabel.Child(), widgets.DefaultTextStyleTransitionOpts{MaxLines: foundation.Some(1), Overflow: painting.TextOverflowEllipsis})))
	}
	if bottomMiddle != nil {
		return c.slideFromLeadingEdge(c.bottomComponents.middleKey, c.bottomNavBarBox, c.topComponents.backLabelKey, c.topNavBarBox,
			animation.NewInterval(0.0, 1.0),
			widgets.NewFadeTransition(pick(c.fadeInFrom(0.3)), widgets.NewDefaultTextStyleTransition(
				animation.Drive[*painting.TextStyle](c.animation, widgets.NewTextStyleTween(c.bottomTitleTextStyle, c.topBackButtonTextStyle)),
				topBackLabel.Child())))
	}
	return nil
}

func (c *navBarComponentsTransition) topMiddle() widgets.Widget {
	middle := keyedChild(c.topComponents.middleKey)
	if middle == nil {
		return nil
	}
	// The middle component is non-null when the nav bar is a large title nav bar but would switch to the
	// normal title in the first half of the animation when going backwards.
	if !c.topHasUserMiddle && c.topLargeExpanded {
		return nil
	}
	to := c.positionInTransitionBox(c.topComponents.middleKey, c.topNavBarBox)
	toBox := renderBoxOf(c.topComponents.middleKey)
	isLTR := c.forwardDirection > 0

	toAnchorInTransitionBox := ui.Offset{Dx: to.Right, Dy: to.Top}
	if isLTR {
		toAnchorInTransitionBox.Dx = to.Left
	}
	// Anchor the top middle at the middle of the right edge of the bar at the start, to move to its place.
	anchorMovement := animation.NewTween(ui.Offset{Dx: c.topNavBarBox.Size().Width - toBox.Size().Width/2, Dy: to.Top}, toAnchorInTransitionBox).SetName("Offset")
	return newFixedSizeSlidingTransition(isLTR, animation.Drive[ui.Offset](c.animation, anchorMovement), toBox.Size().Width, toBox.Size().Height,
		widgets.NewFadeTransition(c.fadeInFrom(0.25), widgets.NewDefaultTextStyle(c.topTitleTextStyle, middle.Child())))
}

func (c *navBarComponentsTransition) topTrailing() widgets.Widget {
	trailing := keyedChild(c.topComponents.trailingKey)
	if trailing == nil {
		return nil
	}
	return widgets.NewPositionedFromRelativeRect(c.positionInTransitionBox(c.topComponents.trailingKey, c.topNavBarBox),
		widgets.NewFadeTransition(c.fadeInFrom(0.4), trailing.Child()))
}

func (c *navBarComponentsTransition) topLargeTitle() widgets.Widget {
	largeTitle := keyedChild(c.topComponents.largeTitleKey)
	if largeTitle == nil || !c.topLargeExpanded {
		return nil
	}
	to := c.positionInTransitionBox(c.topComponents.largeTitleKey, c.topNavBarBox)
	positionTween := relativeRectTween(to.Shift(ui.Offset{Dx: c.forwardDirection * c.topNavBarBox.Size().Width}), to)
	animationCurve := topNavBarHeaderTransitionCurve
	if !c.isForward() {
		animationCurve = topNavBarHeaderTransitionCurve.Flipped()
	}
	return widgets.NewPositionedTransition(c.positionAnimation(positionTween, animationCurve),
		widgets.NewFadeTransition(c.fadeInFrom(0.0, animationCurve), widgets.NewDefaultTextStyle(c.topLargeTitleTextStyle, largeTitle.Child(),
			widgets.DefaultTextStyleOpts{MaxLines: foundation.Some(1), Overflow: painting.TextOverflowEllipsis})))
}

func (c *navBarComponentsTransition) topNavBarBottom() widgets.Widget {
	bottom := keyedChild(c.topComponents.navBarBottomKey)
	if bottom == nil {
		return nil
	}
	to := c.positionInTransitionBox(c.topComponents.navBarBottomKey, c.topNavBarBox)
	positionTween := relativeRectTween(to.Shift(ui.Offset{Dx: c.forwardDirection * c.topNavBarBox.Size().Width}), to)
	child := bottom.Child()
	animationCurve := topNavBarHeaderTransitionCurve
	if !c.isForward() {
		animationCurve = topNavBarHeaderTransitionCurve.Flipped()
	}
	if !c.searchable {
		child = widgets.NewFadeTransition(c.fadeInFrom(0.0, animationCurve), child)
	}
	return widgets.NewPositionedTransition(c.positionAnimation(positionTween, animationCurve), widgets.NewClipRect(child))
}

// linearTranslateWithLargestRectSizeTween is the rect tween of the flights of the bars: the rect moves from the
// top left of the first to the top left of the second, and it has the size of the larger one throughout, so
// the bars are laid out in a box that is as big as either.
//
// Flutter: cupertino/nav_bar.dart _linearTranslateWithLargestRectSizeTween
func linearTranslateWithLargestRectSizeTween(begin, end foundation.Opt[ui.Rect]) *animation.Tween[foundation.Opt[ui.Rect]] {
	b, e := begin.Value(), end.Value()
	largest := ui.Size{Width: math.Max(b.Size().Width, e.Size().Width), Height: math.Max(b.Size().Height, e.Size().Height)}
	rect := func(r ui.Rect) foundation.Opt[ui.Rect] {
		return foundation.Some(ui.NewRectFromLTWH(r.Left, r.Top, largest.Width, largest.Height))
	}
	return animation.NewRectTween(rect(b), rect(e))
}

// navBarHeroLaunchPadBuilder builds what stays in the place of a bar during a flight: the bar, invisible, with
// its size and its state.
//
// Flutter: cupertino/nav_bar.dart _navBarHeroLaunchPadBuilder
func navBarHeroLaunchPadBuilder(bc widgets.BuildContext, heroSize ui.Size, child widgets.Widget) widgets.Widget {
	return widgets.NewVisibility(child, widgets.VisibilityOpts{Visible: foundation.Some(false), MaintainSize: true, MaintainAnimation: true, MaintainState: true})
}

// navBarHeroFlightShuttleBuilder builds the widget that flies between two bars: a transition with the bar of
// the route that is on top and the one that is below it.
//
// Flutter: cupertino/nav_bar.dart _navBarHeroFlightShuttleBuilder
func navBarHeroFlightShuttleBuilder(flightContext widgets.BuildContext, a animation.Animation[float64], flightDirection widgets.HeroFlightDirection,
	fromHeroContext, toHeroContext widgets.BuildContext) widgets.Widget {
	fromNavBar := fromHeroContext.Widget().(*widgets.Hero).Child().(*transitionableNavigationBar)
	toNavBar := toHeroContext.Widget().(*widgets.Hero).Child().(*transitionableNavigationBar)
	if flightDirection == widgets.HeroFlightDirectionPush {
		return newNavBarTransition(a, toNavBar, fromNavBar)
	}
	return newNavBarTransition(a, fromNavBar, toNavBar)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagNavBarLargeTitleStyleMissing    = "NavigationBarLargeTitleStyleMissing"
	tagNavBarTransitionableInFlight    = "NavigationBarTransitionableInFlight"
	tagNavBarTransitionableOutsideHero = "NavigationBarTransitionableOutsideHero"
)
