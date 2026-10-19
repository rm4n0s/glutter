package proof

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

func init() {
	register(Scene{
		Name: "decorations",
		Description: "painting decorations in four rows of five: boxes with a uniform border, with rounded and non-uniform borders, a circle and " +
			"directional borders; box shadows and shape decorations (stadium, circle, star, beveled, continuous, superellipse); linear, radial " +
			"and sweep gradients; linear and oval borders; and the Flutter logo as a mark, next to its name and stacked over it",
		Width:  720,
		Height: 560,
		Build:  func() *ui.Scene { return DecorationsScene() },
	})
}

// DecorationsScene draws the decorations proof: the scene of the decorations proof.
//
// Flutter: no counterpart
func DecorationsScene() *ui.Scene {
	return singlePicture(720, 560, func(c *ui.Canvas) {
		c.DrawColor(ui.NewColor(0xFFF5F5F5), ui.BlendModeSrcOver)
		const cols, cellW, cellH, gap = 5, 120.0, 100.0, 20.0
		cell := func(i int) ui.Rect {
			col, row := i%cols, i/cols
			return ui.NewRectFromLTWH(gap+float64(col)*(cellW+gap), gap+float64(row)*(cellH+gap), cellW, cellH)
		}
		engine := BundledTextEngine()
		decorations := decorationCells(engine)
		for i, d := range decorations {
			paintDecorationIn(c, d, cell(i))
		}
	})
}

func paintDecorationIn(c *ui.Canvas, d painting.Decoration, rect ui.Rect) {
	// The cell is painted at the origin of a translated canvas, like Flutter's tests do: LinearBorder
	// ignores the left of its rect, as Flutter's does.
	c.Save()
	c.Translate(rect.Left, rect.Top)
	defer c.Restore()
	painter := d.CreateBoxPainter(func() {})
	painter.Paint(c, ui.Offset{}, painting.ImageConfiguration{
		Size:          foundation.Some(ui.Size{Width: rect.Width(), Height: rect.Height()}),
		TextDirection: ui.TextDirectionLTR,
	})
	painter.Dispose()
}

func decorationCells(engine ui.ParagraphFactory) []painting.Decoration {
	argb := ui.NewColor
	some := foundation.Some[ui.Color]
	w := func(v float64) foundation.Opt[float64] { return foundation.Some(v) }
	purple, teal, orange, red, blue := argb(0xFF6750A4), argb(0xFF00897B), argb(0xFFFB8C00), argb(0xFFE53935), argb(0xFF1E88E5)
	pale := argb(0xFFEADDFF)
	side := func(color ui.Color, width float64) painting.BorderSide {
		return painting.NewBorderSide(painting.BorderSideOpts{Color: some(color), Width: w(width)})
	}
	shadow := func(dy, blur, spread float64, color ui.Color) painting.BoxShadow {
		return painting.NewBoxShadow(painting.BoxShadowOpts{Color: some(color), Offset: ui.Offset{Dy: dy}, BlurRadius: blur, SpreadRadius: spread})
	}
	return []painting.Decoration{
		// row 1: BoxDecoration with borders
		painting.NewBoxDecoration(painting.BoxDecorationOpts{
			Color: some(pale), Border: painting.NewBorderAll(painting.BorderSideOpts{Color: some(purple), Width: w(4)})}),
		painting.NewBoxDecoration(painting.BoxDecorationOpts{
			Color: some(pale), Border: painting.NewBorderAll(painting.BorderSideOpts{Color: some(purple), Width: w(4)}),
			BorderRadius: painting.NewBorderRadiusCircular(24)}),
		painting.NewBoxDecoration(painting.BoxDecorationOpts{
			Color: some(argb(0xFFB2DFDB)), Border: painting.NewBorderAll(painting.BorderSideOpts{Color: some(teal), Width: w(6)}), Shape: painting.BoxShapeCircle}),
		painting.NewBoxDecoration(painting.BoxDecorationOpts{
			Color:        some(argb(0xFFFFE0B2)),
			Border:       painting.Border{Top: side(orange, 2), Right: side(orange, 8), Bottom: side(orange, 2), Left: side(orange, 8)},
			BorderRadius: painting.NewBorderRadiusCircular(16)}),
		painting.NewBoxDecoration(painting.BoxDecorationOpts{
			Color:  some(argb(0xFFFFFFFF)),
			Border: painting.BorderDirectional{Start: side(red, 8), Top: side(blue, 3), End: side(teal, 8), Bottom: side(purple, 3)}}),
		// row 2: shadows and shape decorations
		painting.NewBoxDecoration(painting.BoxDecorationOpts{
			Color: some(argb(0xFFFFFFFF)), BorderRadius: painting.NewBorderRadiusCircular(12),
			BoxShadow: []painting.BoxShadow{shadow(4, 12, 0, argb(0x66000000)), shadow(1, 3, 0, argb(0x33000000))}}),
		painting.NewShapeDecoration(painting.NewStadiumBorder(painting.StadiumBorderOpts{Side: side(purple, 3)}),
			painting.ShapeDecorationOpts{Color: some(pale)}),
		painting.NewShapeDecoration(painting.NewCircleBorder(painting.CircleBorderOpts{Side: side(blue, 4)}),
			painting.ShapeDecorationOpts{Gradient: painting.NewLinearGradient([]ui.Color{argb(0xFF42A5F5), argb(0xFF1565C0)},
				painting.LinearGradientOpts{Begin: painting.AlignmentTopLeft, End: painting.AlignmentBottomRight})}),
		painting.NewShapeDecoration(painting.NewStarBorder(painting.StarBorderOpts{Side: side(orange, 3), Points: w5(), InnerRadiusRatio: w(0.45), PointRounding: 0.2}),
			painting.ShapeDecorationOpts{Color: some(argb(0xFFFFCC80))}),
		painting.NewShapeDecoration(painting.NewBeveledRectangleBorder(painting.BeveledRectangleBorderOpts{
			Side: side(teal, 3), BorderRadius: painting.NewBorderRadiusCircular(24)}), painting.ShapeDecorationOpts{Color: some(argb(0xFFB2DFDB))}),
		// row 3: gradients and more shapes
		painting.NewBoxDecoration(painting.BoxDecorationOpts{
			BorderRadius: painting.NewBorderRadiusCircular(12),
			Gradient: painting.NewLinearGradient([]ui.Color{red, orange, argb(0xFFFDD835), teal, blue},
				painting.LinearGradientOpts{Begin: painting.AlignmentCenterLeft, End: painting.AlignmentCenterRight})}),
		painting.NewBoxDecoration(painting.BoxDecorationOpts{
			BorderRadius: painting.NewBorderRadiusCircular(12),
			Gradient: painting.NewRadialGradient([]ui.Color{argb(0xFFFFF176), argb(0xFFFB8C00), argb(0xFF6D4C41)},
				painting.RadialGradientOpts{Radius: foundation.Some(0.7), Stops: []float64{0, 0.5, 1}})}),
		painting.NewBoxDecoration(painting.BoxDecorationOpts{
			Shape: painting.BoxShapeCircle,
			Gradient: painting.NewSweepGradient([]ui.Color{red, orange, argb(0xFFFDD835), teal, blue, purple, red},
				painting.SweepGradientOpts{})}),
		painting.NewShapeDecoration(painting.NewContinuousRectangleBorder(painting.ContinuousRectangleBorderOpts{
			Side: side(purple, 3), BorderRadius: painting.NewBorderRadiusCircular(40)}), painting.ShapeDecorationOpts{Color: some(pale)}),
		painting.NewShapeDecoration(painting.NewRoundedSuperellipseBorder(painting.RoundedSuperellipseBorderOpts{
			Side: side(blue, 3), BorderRadius: painting.NewBorderRadiusCircular(40)}), painting.ShapeDecorationOpts{Color: some(argb(0xFFBBDEFB))}),
		// row 4: linear and oval borders and the logo
		painting.NewShapeDecoration(painting.NewLinearBorder(painting.LinearBorderOpts{
			Side:   side(purple, 4),
			Top:    foundation.Some(painting.NewLinearBorderEdge(painting.LinearBorderEdgeOpts{Size: w(0.6)})),
			Bottom: foundation.Some(painting.NewLinearBorderEdge())}), painting.ShapeDecorationOpts{Color: some(argb(0xFFFFFFFF))}),
		painting.NewShapeDecoration(painting.NewOvalBorder(painting.OvalBorderOpts{Side: side(red, 4)}),
			painting.ShapeDecorationOpts{Color: some(argb(0xFFFFCDD2))}),
		painting.NewFlutterLogoDecoration(painting.FlutterLogoDecorationOpts{TextEngine: engine}),
		painting.NewFlutterLogoDecoration(painting.FlutterLogoDecorationOpts{
			Style: painting.FlutterLogoStyleHorizontal, Margin: painting.NewEdgeInsetsAll(4), TextEngine: engine}),
		painting.NewFlutterLogoDecoration(painting.FlutterLogoDecorationOpts{
			Style: painting.FlutterLogoStyleStacked, Margin: painting.NewEdgeInsetsAll(2), TextEngine: engine}),
	}
}

func w5() foundation.Opt[float64] { return foundation.Some(5.0) }
