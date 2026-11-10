package cupertino

import (
	"math"
	"reflect"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func menuFocusPolicy(bc widgets.BuildContext) widgets.FocusTraversalPolicy {
	if p := widgets.FocusTraversalGroupMaybeOf(bc); p != nil {
		return p
	}
	return widgets.NewReadingOrderTraversalPolicy()
}

// menuFocusAction moves the focus up or down in the menu. On the web and the other platforms than Apple's it wraps
// from the last item to the first and the other way round.
//
// Flutter: cupertino/menu_anchor.dart _FocusUpAction, _FocusDownAction
type menuFocusAction struct {
	widgets.ContextActionBase
	intentType reflect.Type
	up         bool
}

func newMenuFocusAction[T widgets.Intent](up bool) *menuFocusAction {
	a := &menuFocusAction{intentType: reflect.TypeFor[T](), up: up}
	widgets.InitContextActionBase(&a.ContextActionBase, a)
	return a
}

func (a *menuFocusAction) IntentType() reflect.Type { return a.intentType }

func (a *menuFocusAction) InvokeWithContext(intent widgets.Intent, bc widgets.BuildContext) any {
	var direction widgets.TraversalDirection
	switch i := intent.(type) {
	case menuFocusUpIntent:
		direction = i.Direction
	case menuFocusDownIntent:
		direction = i.Direction
	}
	policy := menuFocusPolicy(bc)
	primary := widgets.PrimaryFocus(bc)
	if menuIsCupertino() && !foundation.IsWeb {
		policy.InDirection(primary, direction)
		return nil
	}
	firstFocus := policy.FindFirstFocus(primary, true)
	lastFocus := policy.FindLastFocus(primary, true)
	if a.up {
		if lastFocus != nil && lastFocus.Context() != nil && (primary == &lastFocus.EnclosingScope().FocusNode || primary == firstFocus) {
			policy.RequestFocusCallback()(lastFocus, widgets.TraversalRequestFocusOpts{})
			return nil
		}
	} else if firstFocus != nil && firstFocus.Context() != nil && (primary == &firstFocus.EnclosingScope().FocusNode || primary == lastFocus) {
		policy.RequestFocusCallback()(firstFocus, widgets.TraversalRequestFocusOpts{})
		return nil
	}
	policy.InDirection(primary, direction)
	return nil
}

// menuFocusEdgeAction moves the focus to the first or the last item.
//
// Flutter: cupertino/menu_anchor.dart _FocusFirstAction, _FocusLastAction
type menuFocusEdgeAction struct {
	widgets.ContextActionBase
	intentType reflect.Type
	first      bool
}

func newMenuFocusEdgeAction[T widgets.Intent](first bool) *menuFocusEdgeAction {
	a := &menuFocusEdgeAction{intentType: reflect.TypeFor[T](), first: first}
	widgets.InitContextActionBase(&a.ContextActionBase, a)
	return a
}

func (a *menuFocusEdgeAction) IntentType() reflect.Type { return a.intentType }

func (a *menuFocusEdgeAction) InvokeWithContext(_ widgets.Intent, bc widgets.BuildContext) any {
	policy := menuFocusPolicy(bc)
	primary := widgets.PrimaryFocus(bc)
	var target *widgets.FocusNode
	if a.first {
		target = policy.FindFirstFocus(primary, true)
	} else {
		target = policy.FindLastFocus(primary, true)
	}
	if target == nil || target.Context() == nil {
		return nil
	}
	policy.RequestFocusCallback()(target, widgets.TraversalRequestFocusOpts{})
	return nil
}

// menuClampTween clamps t to its begin and end.
//
// Flutter: cupertino/menu_anchor.dart _ClampTween
type menuClampTween struct {
	animation.AnimatableBase[float64]
	begin, end float64
}

func (t *menuClampTween) Transform(v float64) float64 {
	return math.Min(math.Max(v, t.begin), t.end)
}

func newMenuClampTween(begin, end float64) *menuClampTween {
	t := &menuClampTween{begin: begin, end: end}
	animation.InitAnimatableBase[float64](&t.AnimatableBase, t)
	return t
}

// menuShadowPainter paints the soft shadow around the menu.
//
// Flutter: cupertino/menu_anchor.dart _ShadowPainter
type menuShadowPainter struct {
	rendering.CustomPainterBase
	brightness ui.Brightness
	repaint    animation.Animation[float64]
}

func newMenuShadowPainter(brightness ui.Brightness, repaint animation.Animation[float64]) *menuShadowPainter {
	return &menuShadowPainter{CustomPainterBase: rendering.NewCustomPainterBase(repaint), brightness: brightness, repaint: repaint}
}

func (p *menuShadowPainter) shadowAnimation() float64 { return ui.ClampDouble(p.repaint.Value(), 0, 1) }

func (p *menuShadowPainter) Paint(canvas *ui.Canvas, size ui.Size) {
	const (
		radius        = 13.0
		shadowOpacity = 0.12
	)
	center := ui.Offset{Dx: size.Width / 2, Dy: size.Height / 2}
	rect := ui.NewRectFromCenter(center, size.Width, size.Height)
	roundedRect := ui.NewRSuperellipseFromRectAndRadius(rect, ui.NewRadiusCircular(radius))
	animation := p.shadowAnimation()
	paint := ui.NewPaint()
	paint.SetMaskFilter(ui.NewMaskFilterBlur(ui.BlurStyleNormal, animation*50))
	paint.SetColor(ui.NewColorFromRGBO(0, 0, 10, animation*animation*shadowOpacity))
	maskPath := ui.NewPath()
	maskPath.SetFillType(ui.PathFillTypeEvenOdd)
	maskPath.AddRect(rect.Inflate(200))
	maskPath.AddRRect(ui.NewRRectFromRectAndRadius(rect, ui.NewRadiusCircular(radius)))
	canvas.Save()
	canvas.ClipPath(maskPath)
	canvas.DrawRSuperellipse(roundedRect.Inflate(50), paint)
	canvas.Restore()
}

func (p *menuShadowPainter) ShouldRepaint(old rendering.CustomPainter) bool {
	o, ok := old.(*menuShadowPainter)
	return !ok || o.brightness != p.brightness || o.repaint != p.repaint
}

func (p *menuShadowPainter) HitTest(ui.Offset) foundation.Opt[bool] { return foundation.None[bool]() }

// menuLayoutDelegate places the menu next to the anchor, flipping it to the other side when it does not fit, and
// grows it from the anchor while its height factor goes to 1.
//
// Deviation: Flutter keeps the menu out of the display features (folds and cutouts); Glutter has no display features,
// so the screen is the whole overlay (SPEC §12.10).
//
// Flutter: cupertino/menu_anchor.dart _MenuLayoutDelegate
type menuLayoutDelegate struct {
	rendering.SingleChildLayoutDelegateBase
	anchorRect      ui.Rect
	attachmentPoint ui.Offset
	heightFactor    float64
	menuAlignment   painting.Alignment
	overlayPadding  painting.EdgeInsets
}

func (d *menuLayoutDelegate) GetConstraintsForChild(constraints rendering.BoxConstraints) rendering.BoxConstraints {
	return rendering.NewBoxConstraintsLoose(constraints.Biggest()).Deflate(d.overlayPadding)
}

func (d *menuLayoutDelegate) GetPositionForChild(size, childSize ui.Size) ui.Offset {
	inverseHeightFactor := 0.0
	if d.heightFactor > 0.01 {
		inverseHeightFactor = 1 / d.heightFactor
	}
	finalHeight := math.Min(childSize.Height*inverseHeightFactor, size.Height)
	finalSize := ui.Size{Width: childSize.Width, Height: finalHeight}
	desired := d.attachmentPoint.Sub(d.menuAlignment.AlongSize(finalSize))
	screen := ui.NewRectFromLTWH(0, 0, size.Width, size.Height)
	finalPosition := d.positionChild(screen, finalSize, desired, d.anchorRect)
	growsUp := finalPosition.Dy+finalSize.Height <= d.anchorRect.Center().Dy
	if growsUp {
		return ui.Offset{Dx: finalPosition.Dx, Dy: finalPosition.Dy + (finalHeight - childSize.Height)}
	}
	initial := ui.Offset{Dx: finalPosition.Dx, Dy: d.anchorRect.Bottom}
	return ui.Offset{Dx: ui.LerpDouble(initial.Dx, finalPosition.Dx, d.heightFactor), Dy: ui.LerpDouble(initial.Dy, finalPosition.Dy, d.heightFactor)}
}

func (d *menuLayoutDelegate) positionChild(screen ui.Rect, childSize ui.Size, position ui.Offset, anchor ui.Rect) ui.Offset {
	x, y := position.Dx, position.Dy
	pad := d.overlayPadding
	overLeftEdge := func(x float64) bool { return x < screen.Left+pad.Left }
	overRightEdge := func(x float64) bool { return x > screen.Right-childSize.Width-pad.Right }
	overTopEdge := func(y float64) bool { return y < screen.Top+pad.Top }
	overBottomEdge := func(y float64) bool { return y > screen.Bottom-childSize.Height-pad.Bottom }
	hasHorizontalAnchorOverlap := childSize.Width >= screen.Width()
	if hasHorizontalAnchorOverlap {
		x = screen.Left + pad.Left
	} else if overLeftEdge(x) {
		flipX := anchor.Center().Dx*2 - position.Dx - childSize.Width
		hasHorizontalAnchorOverlap = overRightEdge(flipX)
		if hasHorizontalAnchorOverlap || overLeftEdge(flipX) {
			x = screen.Left + pad.Left
		} else {
			x = flipX
		}
	} else if overRightEdge(x) {
		flipX := anchor.Center().Dx*2 - position.Dx - childSize.Width
		hasHorizontalAnchorOverlap = overLeftEdge(flipX)
		if hasHorizontalAnchorOverlap || overRightEdge(flipX) {
			x = screen.Right - childSize.Width - pad.Right
		} else {
			x = flipX
		}
	}
	if childSize.Height >= screen.Height() {
		return ui.Offset{Dx: x, Dy: screen.Top + pad.Top}
	}
	if hasHorizontalAnchorOverlap && !anchor.IsEmpty() {
		below := anchor.Bottom - y
		above := y + childSize.Height - anchor.Top
		if below > 0 && above > 0 {
			if below > above {
				y = anchor.Top - childSize.Height
			} else {
				y = anchor.Bottom
			}
		}
	}
	if overTopEdge(y) {
		flipY := anchor.Center().Dy*2 - position.Dy - childSize.Height
		if overTopEdge(flipY) || overBottomEdge(flipY) {
			y = screen.Top + pad.Top
		} else {
			y = flipY
		}
	} else if overBottomEdge(y) {
		flipY := anchor.Center().Dy*2 - position.Dy - childSize.Height
		if overTopEdge(flipY) || overBottomEdge(flipY) {
			y = screen.Bottom - childSize.Height - pad.Bottom
		} else {
			y = flipY
		}
	}
	return ui.Offset{Dx: x, Dy: y}
}

func (d *menuLayoutDelegate) ShouldRelayout(old rendering.SingleChildLayoutDelegate) bool {
	o, ok := old.(*menuLayoutDelegate)
	return !ok || d.anchorRect != o.anchorRect || d.attachmentPoint != o.attachmentPoint || d.heightFactor != o.heightFactor ||
		d.menuAlignment != o.menuAlignment || d.overlayPadding != o.overlayPadding
}

var (
	menuImplicitDividerOverlayColor = NewCupertinoDynamicColorWithBrightness(ui.NewColorFromRGBO(140, 140, 140, 0.3), ui.NewColorFromRGBO(255, 255, 255, 0.25))
	menuImplicitDividerColor        = NewCupertinoDynamicColorWithBrightness(ui.NewColorFromRGBO(0, 0, 0, 0.25), ui.NewColorFromRGBO(255, 255, 255, 0.25))
)

// menuImplicitDivider is the hairline that the menu puts between two items.
//
// Flutter: cupertino/menu_anchor.dart _CupertinoMenuImplicitDivider
type menuImplicitDivider struct{ widgets.StatelessWidget }

func (w *menuImplicitDivider) Build(bc widgets.BuildContext) widgets.Widget {
	pixelRatio := 1.0
	if r, ok := widgets.MediaQueryMaybeDevicePixelRatioOf(bc); ok {
		pixelRatio = r
	}
	displacement := 1 / pixelRatio
	return widgets.NewCustomPaint(nil, widgets.CustomPaintOpts{Size: ui.Size{Width: math.Inf(1), Height: displacement},
		Painter: &menuDividerPainter{CustomPainterBase: rendering.NewCustomPainterBase(nil),
			color: CupertinoDynamicColorResolve(menuImplicitDividerColor, bc), overlayColor: CupertinoDynamicColorResolve(menuImplicitDividerOverlayColor, bc),
			antiAlias: pixelRatio < 1.0}})
}

// menuDividerPainter paints a hairline in the color and, on the platforms that have it, a second one with the overlay
// blend mode.
//
// Flutter: cupertino/menu_anchor.dart _CupertinoDividerPainter
type menuDividerPainter struct {
	rendering.CustomPainterBase
	color, overlayColor ui.Color
	antiAlias           bool
}

func (p *menuDividerPainter) Paint(canvas *ui.Canvas, size ui.Size) {
	p1 := ui.Offset{Dy: size.Height / 2}
	p2 := ui.Offset{Dx: size.Width, Dy: size.Height / 2}
	if !foundation.IsWeb {
		overlay := ui.NewPaint()
		overlay.SetStyle(ui.PaintingStyleStroke)
		overlay.SetColor(p.overlayColor)
		overlay.SetIsAntiAlias(p.antiAlias)
		overlay.SetBlendMode(ui.BlendModeOverlay)
		canvas.DrawLine(p1, p2, overlay)
	}
	paint := ui.NewPaint()
	paint.SetStyle(ui.PaintingStyleStroke)
	paint.SetColor(p.color)
	paint.SetIsAntiAlias(p.antiAlias)
	canvas.DrawLine(p1, p2, paint)
}

func (p *menuDividerPainter) ShouldRepaint(old rendering.CustomPainter) bool {
	o, ok := old.(*menuDividerPainter)
	return !ok || p.color != o.color || p.overlayColor != o.overlayColor || p.antiAlias != o.antiAlias
}

func (p *menuDividerPainter) HitTest(ui.Offset) foundation.Opt[bool] { return foundation.None[bool]() }

// CupertinoMenuDividerKDefaultColor is the default color of a CupertinoMenuDivider.
//
// Flutter: cupertino/menu_anchor.dart CupertinoMenuDivider.kDefaultColor
var CupertinoMenuDividerKDefaultColor = NewCupertinoDynamicColorWithBrightness(ui.NewColorFromRGBO(0, 0, 0, 0.08), ui.NewColorFromRGBO(0, 0, 0, 0.16))

// CupertinoMenuDivider is a thick divider between two groups of items of a CupertinoMenuAnchor.
//
// Flutter: cupertino/menu_anchor.dart CupertinoMenuDivider
type CupertinoMenuDivider struct {
	widgets.StatelessWidget
	color ui.ColorValue
}

// CupertinoMenuDividerOpts are the optional parameters of NewCupertinoMenuDivider.
//
// Flutter: cupertino/menu_anchor.dart CupertinoMenuDivider.new
type CupertinoMenuDividerOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Color is the color of the divider; the default is CupertinoMenuDividerKDefaultColor.
	Color ui.ColorValue
}

// NewCupertinoMenuDivider creates a divider.
//
// Flutter: cupertino/menu_anchor.dart CupertinoMenuDivider.new
func NewCupertinoMenuDivider(opts ...CupertinoMenuDividerOpts) *CupertinoMenuDivider {
	o := firstOf(opts)
	color := o.Color
	if color == nil {
		color = CupertinoMenuDividerKDefaultColor
	}
	return &CupertinoMenuDivider{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), color: color}
}

// Color is the color of the divider.
//
// Flutter: cupertino/menu_anchor.dart CupertinoMenuDivider.color
func (w *CupertinoMenuDivider) Color() ui.ColorValue { return w.color }

// IsDivider is true.
//
// Flutter: cupertino/menu_anchor.dart CupertinoMenuDivider.isDivider
func (w *CupertinoMenuDivider) IsDivider() bool { return true }

// HasLeading is false.
//
// Flutter: cupertino/menu_anchor.dart CupertinoMenuDivider.hasLeading
func (w *CupertinoMenuDivider) HasLeading(widgets.BuildContext) bool { return false }

// Build builds the divider.
//
// Flutter: cupertino/menu_anchor.dart CupertinoMenuDivider.build
func (w *CupertinoMenuDivider) Build(bc widgets.BuildContext) widgets.Widget {
	return widgets.NewColoredBox(CupertinoDynamicColorResolve(w.color, bc),
		widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(8.0), Width: foundation.Some(math.Inf(1))}))
}
