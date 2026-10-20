package material

import (
	"math"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// maxTitleTextScaleFactor is the biggest text scale factor of the title.
//
// Flutter: material/app_bar.dart _kMaxTitleTextScaleFactor
const maxTitleTextScaleFactor = 1.34

// appBarLeadingWidth is the width of the leading widget.
//
// Flutter: material/app_bar.dart _kLeadingWidth
const appBarLeadingWidth = ToolbarHeight

// AppBarOpts are the optional parameters of NewAppBar.
//
// Flutter: material/app_bar.dart AppBar.new
type AppBarOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Leading is the widget at the start of the toolbar, before the title.
	//
	// TODO(step 13.1): AutomaticallyImplyLeading (the back and drawer buttons).
	Leading widgets.Widget
	// Title is the primary widget displayed in the app bar, usually a Text.
	Title widgets.Widget
	// Actions are the widgets displayed after the title, usually icon buttons.
	Actions []widgets.Widget
	// BackgroundColor is the fill color; the default is the surface color of the
	// color scheme.
	BackgroundColor foundation.Opt[ui.Color]
	// ForegroundColor is the default color of the title and the icons; the
	// default is the on-surface color of the color scheme.
	ForegroundColor foundation.Opt[ui.Color]
	// Elevation is the z-coordinate of the app bar; the default is 0.
	Elevation foundation.Opt[float64]
	// CenterTitle says whether the title is centered; the default is false on
	// every platform but iOS and macOS, where it is true for fewer than two
	// actions.
	CenterTitle foundation.Opt[bool]
	// TitleSpacing is the spacing around the title; the default is
	// widgets.NavigationToolbarMiddleSpacing.
	TitleSpacing foundation.Opt[float64]
	// ToolbarHeight is the height of the toolbar; the default is 64 (Material 3).
	ToolbarHeight foundation.Opt[float64]
	// TitleTextStyle is the style of the title; the default is the title large
	// style of the text theme, in the foreground color.
	TitleTextStyle *painting.TextStyle
	// NoPrimary says that the app bar is not at the top of the screen, so it does
	// not pad itself with the top padding of the media query (Flutter's
	// primary: false).
	NoPrimary bool
}

// AppBar is a Material Design app bar: a toolbar with a leading widget, a
// title and actions. This is the minimum of the Material 3 app bar that the
// counter example needs.
//
// TODO(step 13.1): the theme (AppBarTheme), the bottom, the flexible space, the
// scrolled-under color, the shape, the shadow and surface tint, the system
// overlay style, the leading and actions icon button themes, the back and
// drawer buttons, the Material 2 defaults and SliverAppBar.
//
// Flutter: material/app_bar.dart AppBar
type AppBar struct {
	widgets.StatelessWidget
	o AppBarOpts
}

// NewAppBar creates an app bar.
//
// Flutter: material/app_bar.dart AppBar.new
func NewAppBar(opts ...AppBarOpts) *AppBar {
	o := firstOpts(opts)
	return &AppBar{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), o: o}
}

// Title is the primary widget displayed in the app bar.
//
// Flutter: material/app_bar.dart AppBar.title
func (w *AppBar) Title() widgets.Widget { return w.o.Title }

// Leading is the widget at the start of the toolbar.
//
// Flutter: material/app_bar.dart AppBar.leading
func (w *AppBar) Leading() widgets.Widget { return w.o.Leading }

// Actions are the widgets displayed after the title.
//
// Flutter: material/app_bar.dart AppBar.actions
func (w *AppBar) Actions() []widgets.Widget { return w.o.Actions }

// Primary is whether the app bar is at the top of the screen.
//
// Flutter: material/app_bar.dart AppBar.primary
func (w *AppBar) Primary() bool { return !w.o.NoPrimary }

// PreferredSize is the size of the app bar: infinitely wide and as high as the
// toolbar.
//
// Flutter: material/app_bar.dart AppBar.preferredSize
func (w *AppBar) PreferredSize() ui.Size {
	return ui.Size{Width: math.Inf(1), Height: w.o.ToolbarHeight.OrElse(defaultToolbarHeightM3)}
}

// defaultToolbarHeightM3 is the toolbar height of the Material 3 defaults.
//
// Flutter: material/app_bar.dart _AppBarDefaultsM3
const defaultToolbarHeightM3 = 64.0

func (w *AppBar) effectiveCenterTitle() bool {
	if c, ok := w.o.CenterTitle.Get(); ok {
		return c
	}
	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformIOS, foundation.TargetPlatformMacOS:
		return len(w.o.Actions) < 2
	}
	return false
}

// Build builds the app bar.
//
// Flutter: material/app_bar.dart _AppBarState.build
func (w *AppBar) Build(bc widgets.BuildContext) widgets.Widget {
	if !w.o.NoPrimary {
		widgets.DebugCheckHasMediaQuery(bc)
	}
	theme := ThemeOf(bc)
	colors := theme.ColorScheme

	toolbarHeight := w.o.ToolbarHeight.OrElse(defaultToolbarHeightM3)
	backgroundColor := w.o.BackgroundColor.OrElse(colors.Surface())
	foregroundColor := w.o.ForegroundColor.OrElse(colors.OnSurface())
	elevation := w.o.Elevation.OrElse(0)

	overallIconTheme := widgets.NewIconThemeData(widgets.IconThemeDataOpts{Color: foundation.Some(foregroundColor), Size: foundation.Some(24.0)})
	actionsIconTheme := widgets.NewIconThemeData(widgets.IconThemeDataOpts{
		Color: foundation.Some(w.o.ForegroundColor.OrElse(colors.OnSurfaceVariant())), Size: foundation.Some(24.0)})
	toolbarTextStyle := theme.TextTheme.BodyMedium.CopyWith(painting.TextStyleCopy{Color: foundation.Some(foregroundColor)})
	titleTextStyle := w.o.TitleTextStyle
	if titleTextStyle == nil {
		titleTextStyle = theme.TextTheme.TitleLarge.CopyWith(painting.TextStyleCopy{Color: foundation.Some(foregroundColor)})
	}

	leading := w.o.Leading
	if leading != nil {
		leading = widgets.NewConstrainedBox(
			rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{Width: foundation.Some(appBarLeadingWidth)}), leading)
	}

	title := w.o.Title
	if title != nil {
		title = widgets.NewSemantics(title, widgets.SemanticsOpts{
			NamesRoute: namesRoute(),
			Header:     foundation.Some(true),
		})
		title = widgets.NewDefaultTextStyle(titleTextStyle, title, widgets.DefaultTextStyleOpts{
			SoftWrap: foundation.Some(false),
			Overflow: painting.TextOverflowEllipsis,
		})
		title = widgets.NewMediaQueryWithClampedTextScaling(title, widgets.MediaQueryClampedTextScalingOpts{MaxScaleFactor: maxTitleTextScaleFactor})
	}

	var actions widgets.Widget
	if len(w.o.Actions) > 0 {
		actions = widgets.NewIconThemeMerge(actionsIconTheme,
			widgets.NewPadding(painting.EdgeInsetsZero, widgets.NewRow(w.o.Actions, widgets.RowOpts{
				MainAxisSize:       rendering.MainAxisSizeMin,
				CrossAxisAlignment: rendering.CrossAxisAlignmentCenter,
			})))
	}

	toolbar := widgets.NewNavigationToolbar(widgets.NavigationToolbarOpts{
		Leading:        leading,
		Middle:         title,
		Trailing:       actions,
		NoCenterMiddle: !w.effectiveCenterTitle(),
		MiddleSpacing:  foundation.Some(w.o.TitleSpacing.OrElse(widgets.NavigationToolbarMiddleSpacing)),
	})

	// If the toolbar is allocated less than toolbarHeight make it appear to
	// scroll upwards within its shrinking container.
	var appBar widgets.Widget = widgets.NewClipRect(
		widgets.NewCustomSingleChildLayout(
			&toolbarContainerLayout{SingleChildLayoutDelegateBase: rendering.NewSingleChildLayoutDelegateBase(nil), toolbarHeight: toolbarHeight},
			widgets.NewIconThemeMerge(overallIconTheme, widgets.NewDefaultTextStyle(toolbarTextStyle, toolbar)),
		),
		widgets.ClipRectOpts{ClipBehavior: ui.ClipHardEdge},
	)
	// The padding applies to the toolbar, not to the flexible space.
	if !w.o.NoPrimary {
		appBar = widgets.NewSafeArea(appBar, widgets.SafeAreaOpts{Bottom: foundation.Some(false)})
	}
	appBar = widgets.NewAlign(appBar, widgets.AlignOpts{Alignment: painting.AlignmentTopCenter})

	return widgets.NewSemantics(
		NewMaterial(
			widgets.NewSemantics(appBar, widgets.SemanticsOpts{ExplicitChildNodes: true}),
			MaterialOpts{
				Color:            foundation.Some(backgroundColor),
				Elevation:        elevation,
				ShadowColor:      foundation.Some(Colors.Transparent),
				SurfaceTintColor: foundation.Some(Colors.Transparent),
			},
		),
		widgets.SemanticsOpts{Container: true},
	)
}

// namesRoute is the namesRoute semantics of the title: true on the platforms
// other than iOS and macOS.
func namesRoute() foundation.Opt[bool] {
	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformIOS, foundation.TargetPlatformMacOS:
		return foundation.None[bool]()
	}
	return foundation.Some(true)
}

// toolbarContainerLayout lays the toolbar out at the bottom of the space of
// the app bar.
//
// Flutter: material/app_bar.dart _ToolbarContainerLayout
type toolbarContainerLayout struct {
	rendering.SingleChildLayoutDelegateBase
	toolbarHeight float64
}

func (d *toolbarContainerLayout) GetConstraintsForChild(constraints rendering.BoxConstraints) rendering.BoxConstraints {
	return constraints.Tighten(rendering.TightenOpts{Height: foundation.Some(d.toolbarHeight)})
}

func (d *toolbarContainerLayout) GetSize(constraints rendering.BoxConstraints) ui.Size {
	return ui.Size{Width: constraints.MaxWidth, Height: d.toolbarHeight}
}

func (d *toolbarContainerLayout) GetPositionForChild(size, childSize ui.Size) ui.Offset {
	return ui.Offset{Dx: 0, Dy: size.Height - childSize.Height}
}

func (d *toolbarContainerLayout) ShouldRelayout(old rendering.SingleChildLayoutDelegate) bool {
	return d.toolbarHeight != old.(*toolbarContainerLayout).toolbarHeight
}
