// Command scrolling shows the scroll views of Glutter in seven tabs: a ListView
// with 10,000 rows, a GridView of colored tiles, a CustomScrollView with a pinned
// header, a list and a grid, a PageView of three pages, a ReorderableList, a
// TreeSliver (a file tree whose folders expand and collapse) and a Material
// ReorderableListView (with its drag handles).
// The tabs are plain GestureDetectors, and the views are drawn with widgets only,
// except for the last one.
// Every view has a scrollbar. It uses only the public packages.
//
// Run it with: go run ./examples/scrolling
//
// On Windows: go run ./cmd/glutter-windows run ./examples/scrolling (under Wine on Linux, headless with
// -headless; GLUTTER_BACKEND is win32 there). What Windows lacks here: a precision touchpad scrolls by its wheel messages, with no
// inertia of its own (Flutter's DirectManipulation is not done, SPEC §25.6); the wheel, touch and pen scroll as usual.
//
// Environment: GLUTTER_BACKEND=x11|wayland (Linux) or win32 (Windows), GLUTTER_SCREENSHOT=<file.png>,
// GLUTTER_SCREENSHOT_AFTER=<duration>. In debug builds: GLUTTER_DEV_TAB=<0 to 6>
// opens that tab, and GLUTTER_DEV_SCROLL=<pixels> scrolls the view of the
// CustomScrollView tab by that much.
package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/material/fonts"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// The Material 3 baseline colors that the example paints with.
var (
	primary   = ui.NewColor(0xFF6750A4)
	secondary = ui.NewColor(0xFF625B71)
	surface   = ui.NewColor(0xFFFFFBFE)
	tile      = ui.NewColor(0xFFF3EDF7)
	container = ui.NewColor(0xFFEADDFF)
	outline   = ui.NewColor(0xFF79747E)
	onSurface = ui.NewColor(0xFF1C1B1F)
	white     = ui.NewColor(0xFFFFFFFF)
)

// tabNames are the labels of the tabs, in order.
var tabNames = [numTabs]string{"ListView", "GridView", "Custom", "PageView", "ReorderList", "TreeSliver", "ReorderView"}

const numTabs = 7

const (
	tabList = iota
	tabGrid
	tabCustom
	tabPages
	tabReorder
	tabTree
	tabMaterial
)

const (
	listRows   = 10000
	headerMax  = 140.0
	headerMin  = 56.0
	reorderLen = 12
)

func textStyle(size float64, color ui.Color, weight ui.FontWeight) *painting.TextStyle {
	return painting.NewTextStyle(painting.TextStyleOpts{
		FontFamily: fonts.RobotoFamily, FontSize: foundation.Some(size), Color: foundation.Some(color), FontWeight: weight,
	})
}

func label(s string, size float64, color ui.Color, weight ui.FontWeight) widgets.Widget {
	return widgets.NewText(s, widgets.TextOpts{Style: textStyle(size, color, weight)})
}

func box(width, height float64, child widgets.Widget) widgets.Widget {
	return widgets.NewSizedBox(child, widgets.SizedBoxOpts{Width: foundation.Some(width), Height: foundation.Some(height)})
}

// tileColor is the color of the n-th tile: the hues go around the color wheel.
func tileColor(n int) ui.Color {
	return painting.NewHSVColorFromAHSV(1, float64(n*37%360), 0.45, 0.95).ToColor()
}

// visibleScrollbars is the default ScrollBehavior with scrollbars whose thumb is
// always visible, so that the scrollbars show without scrolling. Its overscroll indicator is Material's:
// the stretch on Android, as the behavior of an app that extends MaterialScrollBehavior has it.
type visibleScrollbars struct {
	widgets.ScrollBehaviorBase
}

func newVisibleScrollbars() *visibleScrollbars {
	b := &visibleScrollbars{}
	widgets.InitScrollBehaviorBase(&b.ScrollBehaviorBase, b)
	return b
}

func (b *visibleScrollbars) BuildOverscrollIndicator(bc widgets.BuildContext, child widgets.Widget, details widgets.ScrollableDetails) widgets.Widget {
	return material.NewMaterialScrollBehavior().BuildOverscrollIndicator(bc, child, details)
}

func (b *visibleScrollbars) BuildScrollbar(_ widgets.BuildContext, child widgets.Widget, details widgets.ScrollableDetails) widgets.Widget {
	return widgets.NewRawScrollbar(child, widgets.RawScrollbarOpts{
		Controller: details.Controller, ThumbVisibility: foundation.Some(true), Thickness: foundation.Some(8.0),
		Radius: foundation.Some(ui.NewRadiusCircular(4)), ThumbColor: foundation.Some(ui.NewColor(0xAA49454F)),
	})
}

// options are what the environment of a debug build can set.
type options struct {
	// tab is the tab that opens first.
	tab int
	// scroll is the initial scroll offset of the CustomScrollView tab.
	scroll float64
}

// optionsFromEnv reads GLUTTER_DEV_TAB and GLUTTER_DEV_SCROLL. Release builds
// ignore both.
func optionsFromEnv(getenv func(string) string) (options, error) {
	var o options
	if !foundation.DebugMode {
		return o, nil
	}
	if v := getenv("GLUTTER_DEV_TAB"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 || n >= len(tabNames) {
			return o, errors.New(tagScrollingTabInvalid, "GLUTTER_DEV_TAB must be a number from 0 to 6", "value", v)
		}
		o.tab = n
	}
	if v := getenv("GLUTTER_DEV_SCROLL"); v != "" {
		n, err := strconv.ParseFloat(v, 64)
		if err != nil || n < 0 {
			return o, errors.New(tagScrollingOffsetInvalid, "GLUTTER_DEV_SCROLL must be a number of pixels, zero or more", "value", v)
		}
		o.scroll = n
	}
	return o, nil
}

// scrollingApp is the whole app: the row of tabs and the view that the selected
// tab shows.
type scrollingApp struct {
	widgets.StatefulWidget
	opts options
}

func newScrollingApp(o options) *scrollingApp {
	return &scrollingApp{StatefulWidget: widgets.NewStatefulWidgetBase(), opts: o}
}

func (w *scrollingApp) CreateState() widgets.State { return &scrollingState{} }

type scrollingState struct {
	widgets.StateOf[*scrollingApp]
	tab int
	// controllers hold the position of each tab's view, so that the scrollbars
	// have a controller and the tests can read the offsets.
	controllers [numTabs]widgets.ScrollController
	pages       *widgets.PageController
	// order is the order of the items of the reorderable list.
	order []int
	// materialOrder is the order of the items of the Material reorderable list.
	materialOrder []int
	// tree is the file tree of the TreeSliver tab, and treeController expands and collapses it.
	tree           []*widgets.TreeSliverNode[string]
	treeController *widgets.TreeSliverController[string]
}

func (s *scrollingState) InitState() {
	s.tab = s.Widget().opts.tab
	for i := range s.controllers {
		offset := 0.0
		if i == tabCustom {
			offset = s.Widget().opts.scroll
		}
		s.controllers[i] = widgets.NewScrollController(widgets.ScrollControllerOpts{InitialScrollOffset: offset})
	}
	s.pages = widgets.NewPageController()
	s.order = make([]int, reorderLen)
	for i := range s.order {
		s.order[i] = i
	}
	s.materialOrder = make([]int, reorderLen)
	for i := range s.materialOrder {
		s.materialOrder[i] = i
	}
	s.tree = fileTree()
	s.treeController = widgets.NewTreeSliverController[string]()
}

func (s *scrollingState) Dispose() {
	for _, c := range s.controllers {
		c.Dispose()
	}
	s.pages.Dispose()
}

func (s *scrollingState) selectTab(i int) {
	if i != s.tab {
		s.SetState(func() { s.tab = i })
	}
}

// reorder moves an item of the reorderable list, like ReorderableListView does.
func (s *scrollingState) reorder(oldIndex, newIndex int) {
	s.SetState(func() {
		item := s.order[oldIndex]
		s.order = append(s.order[:oldIndex], s.order[oldIndex+1:]...)
		s.order = append(s.order[:newIndex], append([]int{item}, s.order[newIndex:]...)...)
	})
}

// reorderMaterial moves an item of the Material reorderable list.
func (s *scrollingState) reorderMaterial(oldIndex, newIndex int) {
	s.SetState(func() {
		item := s.materialOrder[oldIndex]
		s.materialOrder = append(s.materialOrder[:oldIndex], s.materialOrder[oldIndex+1:]...)
		s.materialOrder = append(s.materialOrder[:newIndex], append([]int{item}, s.materialOrder[newIndex:]...)...)
	})
}

func (s *scrollingState) tabBar() widgets.Widget {
	buttons := make([]widgets.Widget, len(tabNames))
	for i, name := range tabNames {
		selected := i == s.tab
		bg, fg := tile, onSurface
		if selected {
			bg, fg = primary, white
		}
		buttons[i] = widgets.NewExpanded(widgets.NewPadding(painting.NewEdgeInsetsSymmetric(4, 0), widgets.NewGestureDetector(
			widgets.NewDecoratedBox(painting.NewBoxDecoration(painting.BoxDecorationOpts{
				Color: foundation.Some(bg), BorderRadius: painting.NewBorderRadiusCircular(20),
			}), widgets.NewPadding(painting.NewEdgeInsetsSymmetric(10, 0), widgets.NewCenter(label(name, 12, fg, ui.FontWeightW500)))),
			widgets.GestureDetectorOpts{OnTap: func() { s.selectTab(i) }, Behavior: rendering.HitTestBehaviorOpaque})),
			widgets.ExpandedOpts{Flex: foundation.Some(len(name))})
	}
	return widgets.NewPadding(painting.NewEdgeInsetsAll(8), widgets.NewRow(buttons))
}

func (s *scrollingState) listTab() widgets.Widget {
	return widgets.NewListViewBuilder(func(_ widgets.BuildContext, i int) widgets.Widget {
		if i >= listRows {
			return nil
		}
		return widgets.NewSizedBox(widgets.NewPadding(painting.NewEdgeInsetsSymmetric(0, 16), widgets.NewAlign(
			label(fmt.Sprintf("Row %d", i+1), 16, onSurface, ui.FontWeightW400),
			widgets.AlignOpts{Alignment: painting.AlignmentCenterLeft})), widgets.SizedBoxOpts{Height: foundation.Some(48.0)})
	}, widgets.ListViewOpts{Controller: s.controllers[tabList], ItemExtent: foundation.Some(48.0), ItemCount: foundation.Some(listRows)})
}

func gridTile(n int) widgets.Widget {
	return widgets.NewDecoratedBox(painting.NewBoxDecoration(painting.BoxDecorationOpts{
		Color: foundation.Some(tileColor(n)), BorderRadius: painting.NewBorderRadiusCircular(12),
	}), widgets.NewCenter(label(strconv.Itoa(n+1), 20, onSurface, ui.FontWeightW500)))
}

func gridTiles(count int) []widgets.Widget {
	tiles := make([]widgets.Widget, count)
	for i := range tiles {
		tiles[i] = gridTile(i)
	}
	return tiles
}

func (s *scrollingState) gridTab() widgets.Widget {
	return widgets.NewGridViewCount(4, gridTiles(120), widgets.GridViewOpts{
		Controller: s.controllers[tabGrid], Padding: painting.NewEdgeInsetsAll(12),
		MainAxisSpacing: 8, CrossAxisSpacing: 8,
	})
}

// titleHeader is the pinned header of the CustomScrollView tab: it shrinks from
// 140 to 56 pixels while it is scrolled, and stays at the top.
type titleHeader struct {
	widgets.SliverPersistentHeaderDelegateBase
}

func (titleHeader) MinExtent() float64 { return headerMin }
func (titleHeader) MaxExtent() float64 { return headerMax }

func (titleHeader) Build(_ widgets.BuildContext, shrinkOffset float64, overlapsContent bool) widgets.Widget {
	size := 28 - 10*(shrinkOffset/(headerMax-headerMin))
	if size < 18 {
		size = 18
	}
	color := primary
	if overlapsContent {
		color = secondary
	}
	return widgets.NewDecoratedBox(painting.NewBoxDecoration(painting.BoxDecorationOpts{Color: foundation.Some(color)}),
		widgets.NewPadding(painting.NewEdgeInsetsSymmetric(0, 16), widgets.NewAlign(
			label("Pinned header", size, white, ui.FontWeightW700),
			widgets.AlignOpts{Alignment: painting.AlignmentCenterLeft})))
}

func (titleHeader) ShouldRebuild(old widgets.SliverPersistentHeaderDelegate) bool {
	_, same := old.(titleHeader)
	return !same
}

func (s *scrollingState) customTab() widgets.Widget {
	return widgets.NewCustomScrollView([]widgets.Widget{
		widgets.NewSliverPersistentHeader(titleHeader{}, widgets.SliverPersistentHeaderOpts{Pinned: true}),
		widgets.NewSliverList(widgets.NewSliverChildBuilderDelegate(func(_ widgets.BuildContext, i int) widgets.Widget {
			row := widgets.NewDecoratedBox(painting.NewBoxDecoration(painting.BoxDecorationOpts{
				Color: foundation.Some(container), BorderRadius: painting.NewBorderRadiusCircular(8),
			}), widgets.NewPadding(painting.NewEdgeInsetsAll(12), label(fmt.Sprintf("List item %d", i+1), 15, onSurface, ui.FontWeightW400)))
			return widgets.NewPadding(painting.NewEdgeInsetsSymmetric(4, 12), row)
		}, widgets.SliverChildBuilderDelegateOpts{ChildCount: foundation.Some(30)})),
		widgets.NewSliverPadding(painting.NewEdgeInsetsAll(12), widgets.NewSliverGridCount(3, gridTiles(18),
			widgets.SliverGridCountOpts{MainAxisSpacing: 8, CrossAxisSpacing: 8})),
	}, widgets.CustomScrollViewOpts{Controller: s.controllers[tabCustom]})
}

func (s *scrollingState) pagesTab() widgets.Widget {
	colors := []ui.Color{primary, secondary, ui.NewColor(0xFF7D5260)}
	pages := make([]widgets.Widget, len(colors))
	for i, c := range colors {
		pages[i] = widgets.NewPadding(painting.NewEdgeInsetsAll(24), widgets.NewDecoratedBox(painting.NewBoxDecoration(painting.BoxDecorationOpts{
			Color: foundation.Some(c), BorderRadius: painting.NewBorderRadiusCircular(24),
		}), widgets.NewCenter(label(fmt.Sprintf("Page %d of %d", i+1, len(colors)), 32, white, ui.FontWeightW700))))
	}
	return widgets.NewPageView(pages, widgets.PageViewOpts{Controller: s.pages})
}

// grip is the drag handle of an item: three short bars.
func grip() widgets.Widget {
	bar := widgets.NewPadding(painting.NewEdgeInsetsSymmetric(2, 0), widgets.NewDecoratedBox(painting.NewBoxDecoration(painting.BoxDecorationOpts{
		Color: foundation.Some(outline), BorderRadius: painting.NewBorderRadiusCircular(1),
	}), box(20, 3, nil)))
	return widgets.NewColumn([]widgets.Widget{bar, bar, bar}, widgets.ColumnOpts{MainAxisSize: rendering.MainAxisSizeMin})
}

// reorderTab is the reorderable list. The list lifts the dragged item into an
// Overlay and reads the WidgetsLocalizations for its semantics. A WidgetsApp or a
// Navigator would give both; until then the tab adds them itself.
func (s *scrollingState) reorderTab() widgets.Widget {
	list := widgets.NewReorderableList(func(_ widgets.BuildContext, i int) widgets.Widget {
		n := s.order[i]
		handle := widgets.NewReorderableDragStartListener(i, box(40, 48, widgets.NewCenter(grip())))
		row := widgets.NewDecoratedBox(painting.NewBoxDecoration(painting.BoxDecorationOpts{
			Color: foundation.Some(tileColor(n)), BorderRadius: painting.NewBorderRadiusCircular(8),
		}), widgets.NewRow([]widgets.Widget{
			handle,
			widgets.NewExpanded(label(fmt.Sprintf("Item %d", n+1), 16, onSurface, ui.FontWeightW500)),
		}))
		return widgets.NewPadding(painting.NewEdgeInsetsSymmetric(4, 12), widgets.NewSizedBox(row,
			widgets.SizedBoxOpts{Height: foundation.Some(56.0)}), widgets.PaddingOpts{Key: foundation.NewValueKey(n)})
	}, reorderLen, widgets.ReorderableListOpts{OnReorderItem: s.reorder, Controller: s.controllers[tabReorder]})
	return widgets.NewLocalizations(ui.NewLocale("en", "US"),
		[]widgets.LocalizationsDelegate{widgets.DefaultWidgetsLocalizationsDelegate}, widgets.NewOverlayWrap(list))
}

// materialTab is the Material ReorderableListView: every item has a drag handle
// at its end, and the dragged item is raised on a Material. The tab gives the list
// the theme, the localizations and the Material that a MaterialApp would.
func (s *scrollingState) materialTab() widgets.Widget {
	list := material.NewReorderableListViewBuilder(func(_ widgets.BuildContext, i int) widgets.Widget {
		n := s.materialOrder[i]
		return material.NewListTile(material.ListTileOpts{
			Key:     foundation.NewValueKey(n),
			Leading: material.NewCircleAvatar(material.CircleAvatarOpts{BackgroundColor: foundation.Some(tileColor(n)), Child: label(strconv.Itoa(n+1), 14, onSurface, ui.FontWeightW500)}),
			Title:   label(fmt.Sprintf("Item %d", n+1), 16, onSurface, ui.FontWeightW500),
		})
	}, reorderLen, material.ReorderableListViewOpts{
		OnReorderItem: s.reorderMaterial, ScrollController: s.controllers[tabMaterial],
		Header: widgets.NewPadding(painting.NewEdgeInsetsSymmetric(16, 12), label("Drag the handles to reorder", 14, secondary, ui.FontWeightW500)),
	})
	return material.NewTheme(material.NewThemeData(), widgets.NewLocalizations(ui.NewLocale("en", "US"),
		[]widgets.LocalizationsDelegate{widgets.DefaultWidgetsLocalizationsDelegate, material.DefaultMaterialLocalizationsDelegate},
		widgets.NewOverlayWrap(material.NewMaterial(list, material.MaterialOpts{Color: foundation.Some(surface)}))))
}

// fileTree is the tree of the TreeSliver tab: the folders of a Go project, the first two open.
func fileTree() []*widgets.TreeSliverNode[string] {
	node := func(name string, expanded bool, children ...*widgets.TreeSliverNode[string]) *widgets.TreeSliverNode[string] {
		return widgets.NewTreeSliverNode(name, widgets.TreeSliverNodeOpts[string]{Children: children, Expanded: expanded})
	}
	file := func(name string) *widgets.TreeSliverNode[string] { return node(name, false) }
	return []*widgets.TreeSliverNode[string]{
		node("glutter", true,
			node("cmd", false, node("glutter-gen", false, file("main.go"), file("run.go")), node("glutter-web", false, file("main.go"))),
			node("docs", false, file("SPEC.md"), file("IMPLEMENTATION.md"), file("GUIDE.md")),
			node("widgets", true, file("scroll_view.go"), file("sliver_tree.go"), file("sliver_tree_test.go"), file("text.go")),
			file("go.mod"), file("README.md")),
		node("examples", false, node("counter", false, file("main.go")), node("scrolling", false, file("main.go"))),
	}
}

// treeRow is the row of a node: an arrow that turns when a folder opens (the width of the arrow for a file, so that the
// names line up), the icon of a folder or a file, and the name. A tap anywhere on the row toggles the node.
func (s *scrollingState) treeRow(_ widgets.BuildContext, node *widgets.TreeSliverNode[string], style animation.AnimationStyle) widgets.Widget {
	isFolder := len(node.Children()) > 0
	arrow := box(24, 24, nil)
	icon := widgets.NewIconData(0xe342, widgets.IconDataOpts{FontFamily: fonts.MaterialIconsFamily}) // insert_drive_file
	if isFolder {
		turns := 0.0
		if node.IsExpanded() {
			turns = 0.25
		}
		arrow = box(24, 24, widgets.NewAnimatedRotation(turns, style.Duration.OrElse(widgets.TreeSliverDefaultAnimationDuration),
			widgets.NewIcon(widgets.NewIconData(0xe15f, widgets.IconDataOpts{FontFamily: fonts.MaterialIconsFamily}), widgets.IconOpts{Size: foundation.Some(20.0), Color: foundation.Some(outline)}),
			widgets.AnimatedRotationOpts{Curve: style.Curve}))
		icon = widgets.NewIconData(0xe2a3, widgets.IconDataOpts{FontFamily: fonts.MaterialIconsFamily}) // folder
	}
	color := outline
	if isFolder {
		color = primary
	}
	return widgets.TreeSliverWrapChildToToggleNode(node, widgets.NewDecoratedBox(
		painting.NewBoxDecoration(painting.BoxDecorationOpts{Color: foundation.Some(surface)}),
		widgets.NewPadding(painting.NewEdgeInsetsSymmetric(0, 8), widgets.NewRow([]widgets.Widget{
			arrow,
			widgets.NewIcon(icon, widgets.IconOpts{Size: foundation.Some(20.0), Color: foundation.Some(color)}),
			box(8, 0, nil),
			label(node.Content(), 15, onSurface, ui.FontWeightW400),
		}))))
}

// treeTab is the TreeSliver: the rows are 32 pixels high and each level is indented by 20.
func (s *scrollingState) treeTab() widgets.Widget {
	return widgets.NewCustomScrollView([]widgets.Widget{
		widgets.NewTreeSliver(s.tree, widgets.TreeSliverOpts[string]{
			Controller: s.treeController, TreeNodeBuilder: s.treeRow,
			TreeRowExtentBuilder: func(*widgets.TreeSliverNode[string], rendering.SliverLayoutDimensions) float64 { return 32 },
			Indentation:          foundation.Some(rendering.TreeSliverIndentationTypeCustom(20)),
		}),
	}, widgets.CustomScrollViewOpts{Controller: s.controllers[tabTree]})
}

func (s *scrollingState) body() widgets.Widget {
	switch s.tab {
	case tabList:
		return s.listTab()
	case tabGrid:
		return s.gridTab()
	case tabCustom:
		return s.customTab()
	case tabPages:
		return s.pagesTab()
	case tabTree:
		return s.treeTab()
	case tabMaterial:
		return s.materialTab()
	default:
		return s.reorderTab()
	}
}

func (s *scrollingState) Build(widgets.BuildContext) widgets.Widget {
	return widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewColoredBox(surface, widgets.NewColumn([]widgets.Widget{
		s.tabBar(),
		widgets.NewExpanded(widgets.NewScrollConfiguration(newVisibleScrollbars(),
			widgets.NewKeyedSubtree(s.body(), widgets.WidgetOpts{Key: foundation.NewValueKey(s.tab)}))),
	}, widgets.ColumnOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStretch})))
}

func main() {
	o, err := optionsFromEnv(os.Getenv)
	if err == nil {
		err = widgets.RunApp(newScrollingApp(o), widgets.RunAppOpts{
			AppID:  "dev.glutter.scrolling",
			Window: widgets.WindowConfig{Title: "Glutter – scrolling", Size: ui.Size{Width: 640, Height: 480}},
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

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagScrollingOffsetInvalid = "ScrollingOffsetInvalid"
	tagScrollingTabInvalid    = "ScrollingTabInvalid"
)
