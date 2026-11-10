package cupertino

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// The measures of the toolbar of macOS (the constants of cupertino/desktop_text_selection_toolbar.dart).
const (
	desktopToolbarScreenPadding   = 8.0
	desktopToolbarSaturationBoost = 3.0
	desktopToolbarBlurSigma       = 20.0
	desktopToolbarWidth           = 222.0
)

var (
	desktopToolbarBorderRadius = ui.NewRadiusCircular(8.0)
	desktopToolbarPadding      = painting.NewEdgeInsetsAll(6.0)
	desktopToolbarBorderColor  = NewCupertinoDynamicColorWithBrightness(ui.NewColor(0xFFB8B8B8), ui.NewColor(0xFF5B5B5B))
	desktopToolbarBackground   = NewCupertinoDynamicColorWithBrightness(ui.NewColor(0xB2FFFFFF), ui.NewColor(0xB2303030))
)

// CupertinoDesktopTextSelectionToolbar is the context menu of the text of macOS: a column of buttons on a blurred,
// translucent card with a border.
//
// Flutter: cupertino/desktop_text_selection_toolbar.dart CupertinoDesktopTextSelectionToolbar
type CupertinoDesktopTextSelectionToolbar struct {
	widgets.StatelessWidget
	anchor   ui.Offset
	children []widgets.Widget
}

// NewCupertinoDesktopTextSelectionToolbar creates a toolbar of the buttons at the anchor. In debug builds it panics
// with the tag CupertinoDesktopTextSelectionToolbarNoChildren when there are none.
//
// Flutter: cupertino/desktop_text_selection_toolbar.dart CupertinoDesktopTextSelectionToolbar.new
func NewCupertinoDesktopTextSelectionToolbar(anchor ui.Offset, children []widgets.Widget, opts ...widgets.WidgetOpts) *CupertinoDesktopTextSelectionToolbar {
	if foundation.DebugMode && len(children) == 0 {
		panic(errors.New(tagCupertinoDesktopTextSelectionToolbarNoChildren, "a CupertinoDesktopTextSelectionToolbar needs at least one child"))
	}
	return &CupertinoDesktopTextSelectionToolbar{StatelessWidget: widgets.NewStatelessWidgetBase(firstOf(opts)), anchor: anchor, children: children}
}

// Anchor is the point that the toolbar is placed at.
//
// Flutter: cupertino/desktop_text_selection_toolbar.dart CupertinoDesktopTextSelectionToolbar.anchor
func (w *CupertinoDesktopTextSelectionToolbar) Anchor() ui.Offset { return w.anchor }

// Children are the buttons.
//
// Flutter: cupertino/desktop_text_selection_toolbar.dart CupertinoDesktopTextSelectionToolbar.children
func (w *CupertinoDesktopTextSelectionToolbar) Children() []widgets.Widget { return w.children }

// desktopToolbarMatrixWithSaturation is the color matrix that changes the saturation.
//
// Flutter: cupertino/desktop_text_selection_toolbar.dart CupertinoDesktopTextSelectionToolbar._matrixWithSaturation
func desktopToolbarMatrixWithSaturation(saturation float64) [20]float64 {
	r := 0.213 * (1 - saturation)
	g := 0.715 * (1 - saturation)
	b := 0.072 * (1 - saturation)
	return [20]float64{
		r + saturation, g, b, 0, 0,
		r, g + saturation, b, 0, 0,
		r, g, b + saturation, 0, 0,
		0, 0, 0, 1, 0,
	}
}

// desktopToolbarDefaultBuilder puts the child on the card of the toolbar.
//
// Flutter: cupertino/desktop_text_selection_toolbar.dart CupertinoDesktopTextSelectionToolbar._defaultToolbarBuilder
func desktopToolbarDefaultBuilder(bc widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return widgets.NewContainer(widgets.NewBackdropFilter(
		ui.NewImageFilterCompose(ui.NewColorFilterMatrix(desktopToolbarMatrixWithSaturation(desktopToolbarSaturationBoost)),
			ui.NewImageFilterBlur(ui.ImageFilterBlurOpts{SigmaX: desktopToolbarBlurSigma, SigmaY: desktopToolbarBlurSigma})),
		widgets.NewDecoratedBox(painting.NewShapeDecoration(painting.NewRoundedSuperellipseBorder(painting.RoundedSuperellipseBorderOpts{
			Side:         painting.NewBorderSide(painting.BorderSideOpts{Color: foundation.Some(CupertinoDynamicColorResolve(desktopToolbarBorderColor, bc))}),
			BorderRadius: painting.NewBorderRadiusAll(desktopToolbarBorderRadius)}),
			painting.ShapeDecorationOpts{Color: foundation.Some(CupertinoDynamicColorResolve(desktopToolbarBackground, bc))}),
			widgets.NewPadding(desktopToolbarPadding, child))),
		widgets.ContainerOpts{
			Width:        foundation.Some(desktopToolbarWidth),
			ClipBehavior: ui.ClipHardEdge,
			Decoration: painting.NewShapeDecoration(painting.NewRoundedSuperellipseBorder(painting.RoundedSuperellipseBorderOpts{
				BorderRadius: painting.NewBorderRadiusAll(desktopToolbarBorderRadius)}),
				painting.ShapeDecorationOpts{Shadows: []painting.BoxShadow{{
					Shadow:       ui.NewShadow(ui.ShadowOpts{Color: ptrColor(ui.NewColorFromARGB(60, 0, 0, 0)), Offset: ui.Offset{Dy: 4.0}, BlurRadius: 10.0}),
					SpreadRadius: 0.5}}}),
		})
}

// Build puts the buttons in the card and the card at the anchor.
//
// Flutter: cupertino/desktop_text_selection_toolbar.dart CupertinoDesktopTextSelectionToolbar.build
func (w *CupertinoDesktopTextSelectionToolbar) Build(bc widgets.BuildContext) widgets.Widget {
	widgets.DebugCheckHasMediaQuery(bc)
	paddingAbove := widgets.MediaQueryPaddingOf(bc).Top + desktopToolbarScreenPadding
	localAdjustment := ui.Offset{Dx: desktopToolbarScreenPadding, Dy: paddingAbove}
	return widgets.NewPadding(
		painting.NewEdgeInsetsFromLTRB(desktopToolbarScreenPadding, paddingAbove, desktopToolbarScreenPadding, desktopToolbarScreenPadding),
		widgets.NewCustomSingleChildLayout(
			widgets.NewDesktopTextSelectionToolbarLayoutDelegate(w.anchor.Sub(localAdjustment)),
			desktopToolbarDefaultBuilder(bc, widgets.NewColumn(w.children, widgets.ColumnOpts{MainAxisSize: rendering.MainAxisSizeMin}))))
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const tagCupertinoDesktopTextSelectionToolbarNoChildren = "CupertinoDesktopTextSelectionToolbarNoChildren"
