// Command navigation shows the navigation of Glutter on a WidgetsApp, without
// Material. The home page has two tabs. The first one navigates with the
// imperative API: a thumbnail (a Hero) opens a detail page through a custom
// PageRouteBuilder that slides in, and the "Pick a color" link opens a page that
// returns a value, which the home page reads from the channel of NavigatorPush.
// The second tab keeps a stack of pages in a nested Navigator, the Pages API: the
// "Push page" link adds a page to the list, and the Back link of a page removes
// it again. Every link takes the keyboard focus: Tab and Shift+Tab move between
// the links, the focused link has a thick border, Enter or Space follows it, and
// Escape goes back (every page maps the DismissIntent to a pop). It uses only
// the public packages.
//
// Run it with: go run ./examples/navigation
//
// Environment: GLUTTER_BACKEND=x11|wayland, GLUTTER_SCREENSHOT=<file.png>,
// GLUTTER_SCREENSHOT_AFTER=<duration>. In debug builds: GLUTTER_DEV_ROUTE=<"/",
// "/detail" or "/pick"> opens that page first, and GLUTTER_DEV_TAB=<0 or 1>
// opens that tab of the home page.
package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"reflect"
	"strconv"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/material/fonts"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// The Material 3 baseline colors that the example paints with.
var (
	primary   = ui.NewColor(0xFF6750A4)
	secondary = ui.NewColor(0xFF625B71)
	tertiary  = ui.NewColor(0xFF7D5260)
	surface   = ui.NewColor(0xFFFFFBFE)
	tile      = ui.NewColor(0xFFF3EDF7)
	hovered   = ui.NewColor(0xFFE8DEF8)
	onSurface = ui.NewColor(0xFF1C1B1F)
	white     = ui.NewColor(0xFFFFFFFF)
	// focusRing is the color of the border of the focused link.
	focusRing = ui.NewColor(0xFF006874)
	sky       = ui.NewColor(0xFF8FD0E0)
	sun       = ui.NewColor(0xFFFFD166)
)

// The names of the routes.
const (
	routeHome   = "/"
	routeDetail = "/detail"
	routePick   = "/pick"
)

// The tabs of the home page.
const (
	tabImperative = iota
	tabPages
	numTabs
)

var tabNames = [numTabs]string{"Imperative", "Pages API"}

// colorNames are the choices of the picker page.
var colorNames = []string{"Teal", "Amber", "Rose"}

// photoTag is the tag of the hero that flies between the home and the detail
// page.
const photoTag = "photo"

func textStyle(size float64, color ui.Color, weight ui.FontWeight) *painting.TextStyle {
	return painting.NewTextStyle(painting.TextStyleOpts{
		FontFamily: fonts.RobotoFamily, FontSize: foundation.Some(size), Color: foundation.Some(color), FontWeight: weight,
	})
}

func label(s string, size float64, color ui.Color, weight ui.FontWeight) widgets.Widget {
	return widgets.NewText(s, widgets.TextOpts{Style: textStyle(size, color, weight)})
}

func gap(height float64) widgets.Widget {
	return widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(height)})
}

// options are what the environment of a debug build can set.
type options struct {
	// route is the name of the route that opens first.
	route string
	// tab is the tab of the home page that opens first.
	tab int
}

// optionsFromEnv reads GLUTTER_DEV_ROUTE and GLUTTER_DEV_TAB. Release builds
// ignore both.
func optionsFromEnv(getenv func(string) string) (options, error) {
	o := options{route: routeHome}
	if !foundation.DebugMode {
		return o, nil
	}
	if v := getenv("GLUTTER_DEV_ROUTE"); v != "" {
		switch v {
		case routeHome, routeDetail, routePick:
			o.route = v
		default:
			return o, errors.New("NavigationRouteInvalid", `GLUTTER_DEV_ROUTE must be "/", "/detail" or "/pick"`, "value", v)
		}
	}
	if v := getenv("GLUTTER_DEV_TAB"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 || n >= numTabs {
			return o, errors.New("NavigationTabInvalid", "GLUTTER_DEV_TAB must be 0 or 1", "value", v)
		}
		o.tab = n
	}
	return o, nil
}

// popAction is the action of the DismissIntent (Escape): it pops the route of the
// nearest Navigator of the focused widget, when that navigator has a route to go
// back to. A page has to provide it itself with escapeToPop: the Actions of a
// ModalRoute map the DismissIntent too, to an action that is off unless the
// barrier of the route dismisses it, and the closest mapping wins.
type popAction struct {
	widgets.ContextActionBase
}

func newPopAction() *popAction {
	a := &popAction{}
	widgets.InitContextActionBase(&a.ContextActionBase, a)
	return a
}

func (a *popAction) IntentType() reflect.Type { return reflect.TypeFor[widgets.DismissIntent]() }

func (a *popAction) IsEnabledWithContext(_ widgets.Intent, bc widgets.BuildContext) bool {
	return bc != nil && widgets.NavigatorCanPop(bc)
}

func (a *popAction) InvokeWithContext(_ widgets.Intent, bc widgets.BuildContext) any {
	if bc == nil {
		return nil
	}
	if navigator := widgets.NavigatorMaybeOf(bc); navigator != nil {
		return navigator.MaybePop()
	}
	return nil
}

// escapeToPop makes Escape pop the route of the page.
func escapeToPop(page widgets.Widget) widgets.Widget {
	return widgets.NewActions(widgets.ActionMap{widgets.IntentTypeOf[widgets.DismissIntent](): newPopAction()}, page)
}

// newApp is the whole app: a WidgetsApp whose routes are made by generateRoute.
func newApp(o options) widgets.Widget {
	heroes := widgets.NewHeroController()
	return widgets.NewWidgetsApp(surface, widgets.WidgetsAppOpts{
		Title: "Glutter navigation",
		// The window is a proof image: it has no debug ribbon.
		DebugShowCheckedModeBanner: foundation.Some(false),
		TextStyle:                  textStyle(15, onSurface, ui.FontWeightW400),
		InitialRoute:               o.route,
		OnGenerateRoute:            func(s widgets.RouteSettings) widgets.Route { return generateRoute(s, o) },
		// MaterialApp gives its navigator a HeroController; here the app does.
		Builder: func(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
			return widgets.NewHeroControllerScope(heroes, child)
		},
	})
}

// generateRoute makes the route of a name: the home page appears without a
// transition, the detail page slides in from the right and the picker page rises
// from the bottom.
func generateRoute(settings widgets.RouteSettings, o options) widgets.Route {
	switch settings.Name() {
	case routeHome:
		return pageRoute(settings, func(widgets.BuildContext) widgets.Widget { return newHomePage(o.tab) }, nil)
	case routeDetail:
		return detailRoute()
	case routePick:
		return pickRoute()
	}
	return nil
}

// detailRoute is the route of the detail page, which slides in from the right.
func detailRoute() widgets.Route {
	return pageRoute(widgets.NewRouteSettings(widgets.RouteSettingsOpts{Name: routeDetail}),
		func(widgets.BuildContext) widgets.Widget { return newDetailPage() }, slideFromRight)
}

// pickRoute is the route of the picker page, which rises from the bottom.
func pickRoute() widgets.Route {
	return pageRoute(widgets.NewRouteSettings(widgets.RouteSettingsOpts{Name: routePick}),
		func(widgets.BuildContext) widgets.Widget { return newPickPage() }, riseFromBottom)
}

// pageRoute is a PageRouteBuilder with the transition, or none.
func pageRoute(settings widgets.RouteSettings, page widgets.WidgetBuilder, transition widgets.RouteTransitionsBuilder) widgets.Route {
	return widgets.NewPageRouteBuilder(
		func(bc widgets.BuildContext, _, _ animation.Animation[float64]) widgets.Widget {
			return widgets.NewColoredBox(surface, escapeToPop(page(bc)))
		},
		widgets.PageRouteBuilderOpts{
			PageRouteOpts:      widgets.PageRouteOpts{RouteOpts: widgets.RouteOpts{Settings: settings}},
			TransitionsBuilder: transition,
			TransitionDuration: foundation.Some(350 * time.Millisecond),
		})
}

func offsetAnimation(a animation.Animation[float64], begin ui.Offset, curve animation.Curve) animation.Animation[ui.Offset] {
	return animation.Drive[ui.Offset](a, animation.NewTween(begin, ui.Offset{}).Chain(animation.NewCurveTween(curve)))
}

// slideFromRight is the transition of the detail page: the page slides in from
// the right edge.
func slideFromRight(_ widgets.BuildContext, a, _ animation.Animation[float64], child widgets.Widget) widgets.Widget {
	return widgets.NewSlideTransition(offsetAnimation(a, ui.Offset{Dx: 1}, animation.Curves.EaseOutCubic), child)
}

// riseFromBottom is the transition of the picker page: the page slides up while
// it fades in.
func riseFromBottom(_ widgets.BuildContext, a, _ animation.Animation[float64], child widgets.Widget) widgets.Widget {
	return widgets.NewFadeTransition(a, widgets.NewSlideTransition(offsetAnimation(a, ui.Offset{Dy: 0.25}, animation.Curves.EaseOutCubic), child))
}

// fade is the transition of the pages of the Pages API tab.
func fade(_ widgets.BuildContext, a, _ animation.Animation[float64], child widgets.Widget) widgets.Widget {
	return widgets.NewFadeTransition(a, child)
}

// linkOpts are the optional parameters of newLink.
type linkOpts struct {
	// selected paints the link as the current tab.
	selected bool
	// autofocus gives the link the focus when it is built.
	autofocus bool
}

// link is a button made of the focus widgets: a FocusableActionDetector takes the
// focus, maps the ActivateIntent (Enter and Space, by the shortcuts of the app)
// to the tap, and tells when to draw the focus highlight; a GestureDetector
// handles the pointer.
type link struct {
	widgets.StatefulWidget
	text  string
	onTap func()
	opts  linkOpts
}

func newLink(text string, onTap func(), opts ...linkOpts) *link {
	l := &link{StatefulWidget: widgets.NewStatefulWidgetBase(), text: text, onTap: onTap}
	if len(opts) > 0 {
		l.opts = opts[0]
	}
	return l
}

func (w *link) CreateState() widgets.State { return &linkState{} }

type linkState struct {
	widgets.StateOf[*link]
	focused, hovered bool
}

func (s *linkState) setFocused(v bool) { s.SetState(func() { s.focused = v }) }
func (s *linkState) setHovered(v bool) { s.SetState(func() { s.hovered = v }) }

func (s *linkState) Build(widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	bg, fg := tile, primary
	switch {
	case w.opts.selected:
		bg, fg = primary, white
	case s.hovered:
		bg = hovered
	}
	ring := ui.Color{}
	if s.focused {
		ring = focusRing
	}
	body := widgets.NewDecoratedBox(painting.NewBoxDecoration(painting.BoxDecorationOpts{
		Color:        foundation.Some(bg),
		BorderRadius: painting.NewBorderRadiusCircular(22),
		Border:       painting.NewBorderAll(painting.BorderSideOpts{Color: foundation.Some(ring), Width: foundation.Some(3.0)}),
	}), widgets.NewPadding(painting.NewEdgeInsetsSymmetric(16, 9), label(w.text, 14, fg, ui.FontWeightW500)))
	return widgets.NewFocusableActionDetector(
		widgets.NewGestureDetector(body, widgets.GestureDetectorOpts{OnTap: w.onTap, Behavior: rendering.HitTestBehaviorOpaque}),
		widgets.FocusableActionDetectorOpts{
			Autofocus: w.opts.autofocus,
			Actions: widgets.ActionMap{
				widgets.IntentTypeOf[widgets.ActivateIntent](): widgets.NewCallbackAction(func(widgets.ActivateIntent) any {
					w.onTap()
					return nil
				}),
			},
			OnShowFocusHighlight: s.setFocused,
			OnShowHoverHighlight: s.setHovered,
			MouseCursor:          services.SystemMouseCursors.Click,
		})
}

// photo is the picture of the hero: a sky with a sun. The home page shows it as
// a thumbnail and the detail page as a large picture.
func photo(width, height float64) widgets.Widget {
	picture := widgets.NewDecoratedBox(painting.NewBoxDecoration(painting.BoxDecorationOpts{
		BorderRadius: painting.NewBorderRadiusCircular(16),
		Gradient: painting.NewLinearGradient([]ui.Color{sky, primary, tertiary}, painting.LinearGradientOpts{
			Begin: painting.AlignmentTopLeft, End: painting.AlignmentBottomRight,
		}),
	}), widgets.NewAlign(
		widgets.NewFractionallySizedBox(widgets.NewAspectRatio(1, widgets.NewDecoratedBox(painting.NewBoxDecoration(painting.BoxDecorationOpts{
			Color: foundation.Some(sun), Shape: painting.BoxShapeCircle,
		}), nil)), widgets.FractionallySizedBoxOpts{HeightFactor: foundation.Some(0.3)}),
		widgets.AlignOpts{Alignment: painting.Alignment{X: 0.5, Y: -0.4}}))
	sized := widgets.NewSizedBox(picture, widgets.SizedBoxOpts{Width: foundation.Some(width), Height: foundation.Some(height)})
	return widgets.NewHero(photoTag, sized)
}

// homePage is the first page: a title, the tabs and the content of the tab.
type homePage struct {
	widgets.StatefulWidget
	tab int
}

func newHomePage(tab int) *homePage {
	return &homePage{StatefulWidget: widgets.NewStatefulWidgetBase(), tab: tab}
}

func (w *homePage) CreateState() widgets.State { return &homeState{} }

type homeState struct {
	widgets.StateOf[*homePage]
	tab int
	// picked is the color that the picker page returned, or "".
	picked string
	// stack holds the numbers of the pages of the Pages API tab, bottom first.
	stack []int
}

func (s *homeState) InitState() {
	s.tab = s.Widget().tab
	s.stack = []int{1}
}

func (s *homeState) selectTab(i int) { s.SetState(func() { s.tab = i }) }

func (s *homeState) openDetail() {
	widgets.NavigatorPush(s.Context(), detailRoute())
}

// pickColor opens the picker page and waits for its result in a goroutine: the
// channel that NavigatorPush returns gets the value that the page popped with
// (nil when it was dismissed).
func (s *homeState) pickColor() {
	result := widgets.NavigatorPush(s.Context(), pickRoute())
	s.Go(func(ctx context.Context) {
		select {
		case v := <-result:
			if name, ok := v.(string); ok {
				s.SetState(func() { s.picked = name })
			}
		case <-ctx.Done():
		}
	})
}

func (s *homeState) pushPage() {
	s.SetState(func() { s.stack = append(s.stack, s.stack[len(s.stack)-1]+1) })
}

// removePage is called by the nested navigator when a page was popped: the list
// of pages must follow.
func (s *homeState) removePage(p widgets.Page) {
	step := p.(*stepPage).step
	s.SetState(func() {
		for i, n := range s.stack {
			if n == step {
				s.stack = append(s.stack[:i:i], s.stack[i+1:]...)
				return
			}
		}
	})
}

func (s *homeState) tabBar() widgets.Widget {
	links := make([]widgets.Widget, 0, 2*numTabs)
	for i, name := range tabNames {
		links = append(links, newLink(name, func() { s.selectTab(i) }, linkOpts{selected: i == s.tab}), widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(8.0)}))
	}
	return widgets.NewRow(links)
}

func (s *homeState) imperativeTab() widgets.Widget {
	picked := s.picked
	if picked == "" {
		picked = "none yet"
	}
	thumb := widgets.NewGestureDetector(photo(150, 100), widgets.GestureDetectorOpts{OnTap: s.openDetail})
	actions := widgets.NewColumn([]widgets.Widget{
		newLink("Open detail page", s.openDetail),
		gap(8),
		newLink("Pick a color", s.pickColor),
	}, widgets.ColumnOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStart})
	return widgets.NewColumn([]widgets.Widget{
		widgets.NewRow([]widgets.Widget{thumb, widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(24.0)}), actions}),
		gap(20),
		label("Picked color: "+picked, 18, onSurface, ui.FontWeightW500),
		gap(12),
		label("Tab and Shift+Tab move the focus, Enter follows a link, Escape goes back.", 14, secondary, ui.FontWeightW400),
	}, widgets.ColumnOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStart})
}

func (s *homeState) pagesTab() widgets.Widget {
	pages := make([]widgets.Page, len(s.stack))
	names := ""
	for i, n := range s.stack {
		pages[i] = newStepPage(n)
		if i > 0 {
			names += " > "
		}
		names += "Step " + strconv.Itoa(n)
	}
	nested := widgets.NewDecoratedBox(painting.NewBoxDecoration(painting.BoxDecorationOpts{
		BorderRadius: painting.NewBorderRadiusCircular(16),
		Border:       painting.NewBorderAll(painting.BorderSideOpts{Color: foundation.Some(secondary), Width: foundation.Some(1.0)}),
	}), widgets.NewClipRRect(widgets.NewNavigator(widgets.NavigatorOpts{Pages: pages, OnDidRemovePage: s.removePage}),
		widgets.ClipRRectOpts{BorderRadius: painting.NewBorderRadiusCircular(15)}))
	return widgets.NewColumn([]widgets.Widget{
		widgets.NewRow([]widgets.Widget{
			newLink("Push page", s.pushPage),
			widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(16.0)}),
			label("Stack: "+names, 16, onSurface, ui.FontWeightW500),
		}),
		gap(12),
		widgets.NewExpanded(nested),
	}, widgets.ColumnOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStretch})
}

func (s *homeState) Build(widgets.BuildContext) widgets.Widget {
	var body widgets.Widget
	if s.tab == tabPages {
		body = s.pagesTab()
	} else {
		body = s.imperativeTab()
	}
	return widgets.NewPadding(painting.NewEdgeInsetsAll(20), widgets.NewColumn([]widgets.Widget{
		label("Navigation", 28, onSurface, ui.FontWeightW700),
		gap(16),
		s.tabBar(),
		gap(20),
		widgets.NewExpanded(widgets.NewKeyedSubtree(body, widgets.WidgetOpts{Key: foundation.NewValueKey(s.tab)})),
	}, widgets.ColumnOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStretch}))
}

// stepPage is a page of the Pages API: the Navigator makes its route, and keeps
// it while a page with the same key is in the list.
type stepPage struct {
	widgets.PageBase
	step int
}

func newStepPage(step int) *stepPage {
	p := &stepPage{step: step}
	widgets.InitPageBase(&p.PageBase, p, widgets.PageOpts{Key: foundation.NewValueKey(step), Name: "/step/" + strconv.Itoa(step)})
	return p
}

func (p *stepPage) CreateRoute(widgets.BuildContext) widgets.Route {
	return widgets.NewPageRouteBuilder(
		func(widgets.BuildContext, animation.Animation[float64], animation.Animation[float64]) widgets.Widget {
			return widgets.NewColoredBox(tileColor(p.step), escapeToPop(p.content()))
		},
		widgets.PageRouteBuilderOpts{
			PageRouteOpts:      widgets.PageRouteOpts{RouteOpts: widgets.RouteOpts{Settings: p}},
			TransitionsBuilder: fade,
			TransitionDuration: foundation.Some(250 * time.Millisecond),
		})
}

// tileColor is the color of the page of a step: the hues go around the wheel.
func tileColor(step int) ui.Color {
	return painting.NewHSVColorFromAHSV(1, float64(step*47%360), 0.25, 0.97).ToColor()
}

func (p *stepPage) content() widgets.Widget {
	items := []widgets.Widget{label("Step "+strconv.Itoa(p.step), 30, onSurface, ui.FontWeightW700)}
	if p.step > 1 {
		items = append(items, gap(16), widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
			return newLink("Back", func() { widgets.NavigatorPop(bc) }, linkOpts{autofocus: true})
		}))
	}
	return widgets.NewCenter(widgets.NewColumn(items, widgets.ColumnOpts{MainAxisSize: rendering.MainAxisSizeMin}))
}

// detailPage shows the picture of the hero in a large size, with a description
// and two links.
type detailPage struct {
	widgets.StatefulWidget
}

func newDetailPage() *detailPage { return &detailPage{widgets.NewStatefulWidgetBase()} }

func (w *detailPage) CreateState() widgets.State { return &detailState{} }

type detailState struct {
	widgets.StateOf[*detailPage]
	favorite bool
}

func (s *detailState) Build(bc widgets.BuildContext) widgets.Widget {
	favorite := "Favorite"
	if s.favorite {
		favorite = "Favorited"
	}
	return widgets.NewPadding(painting.NewEdgeInsetsAll(20), widgets.NewColumn([]widgets.Widget{
		photo(math.Inf(1), 220),
		gap(20),
		label("Sunrise over the lake", 26, onSurface, ui.FontWeightW700),
		gap(8),
		label("The hero above flew here from the thumbnail of the home page, and flies back when the page is popped.", 15, secondary, ui.FontWeightW400),
		gap(24),
		widgets.NewRow([]widgets.Widget{
			newLink("Back", func() { widgets.NavigatorPop(bc) }, linkOpts{autofocus: true}),
			widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(12.0)}),
			newLink(favorite, func() { s.SetState(func() { s.favorite = !s.favorite }) }),
		}),
	}, widgets.ColumnOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStretch}))
}

// pickPage lists the colors; following a link pops the page with the name of the
// color, and Cancel (or Escape) pops it with no result.
type pickPage struct {
	widgets.StatelessWidget
}

func newPickPage() *pickPage { return &pickPage{widgets.NewStatelessWidgetBase()} }

func (w *pickPage) Build(bc widgets.BuildContext) widgets.Widget {
	items := []widgets.Widget{label("Pick a color", 28, onSurface, ui.FontWeightW700), gap(20)}
	for i, name := range colorNames {
		items = append(items, newLink(name, func() { widgets.NavigatorPop(bc, name) }, linkOpts{autofocus: i == 0}), gap(10))
	}
	items = append(items, gap(10), newLink("Cancel", func() { widgets.NavigatorPop(bc) }))
	return widgets.NewPadding(painting.NewEdgeInsetsAll(20), widgets.NewColumn(items,
		widgets.ColumnOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStart}))
}

func main() {
	o, err := optionsFromEnv(os.Getenv)
	if err == nil {
		err = widgets.RunApp(newApp(o), widgets.RunAppOpts{
			AppID:  "dev.glutter.navigation",
			Window: widgets.WindowConfig{Title: "Glutter – navigation", Size: ui.Size{Width: 640, Height: 480}},
		})
	}
	if err != nil {
		if appErr, ok := errors.FromError(err); ok {
			fmt.Fprintf(os.Stderr, "%s: %s %v\n", appErr.Tag, appErr.Message, appErr.Metadata)
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}
