package proof

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/renderhost"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

func init() {
	register(Scene{
		Name: "rendering-tour",
		Description: "a render tree laid out and painted through the pipeline: a title paragraph, a row of three decorated boxes with a flexible " +
			"one in the middle, a stack with a positioned round badge, a wrap of text chips that breaks into two runs, a paragraph cut at two " +
			"lines with an ellipsis, and a generated image sized with its aspect ratio",
		Width:  640,
		Height: 540,
		Build:  func() *ui.Scene { return RenderingTourScene() },
	})
}

// RenderingTourScene builds the render tree of the rendering tour proof, lays
// it out and paints it with the rendering binding of a fake engine, and returns
// the scene that the view rendered.
//
// Flutter: no counterpart
func RenderingTourScene() *ui.Scene {
	stack := renderhost.New(renderhost.Opts{Size: ui.Size{Width: 640, Height: 540}, TextEngine: BundledTextEngine()})
	stack.SetRoot(renderingTourTree())
	stack.Pump()
	return stack.LastScene()
}

func renderingTourTree() rendering.Box {
	argb := ui.NewColor
	purple, teal, orange, red := argb(0xFF6750A4), argb(0xFF00897B), argb(0xFFFB8C00), argb(0xFFE53935)
	engine := BundledTextEngine()
	style := func(size float64, weight ui.FontWeight, color ui.Color) *painting.TextStyle {
		return painting.NewTextStyle(painting.TextStyleOpts{
			FontFamily: "Roboto", FontSize: foundation.Some(size), FontWeight: weight, Color: foundation.Some(color),
		})
	}
	paragraph := func(text string, size float64, weight ui.FontWeight, color ui.Color, opts ...rendering.RenderParagraphOpts) *rendering.RenderParagraph {
		var o rendering.RenderParagraphOpts
		if len(opts) > 0 {
			o = opts[0]
		}
		o.TextEngine = engine
		span := painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some(text), Style: style(size, weight, color)})
		return rendering.NewRenderParagraph(span, ui.TextDirectionLTR, o)
	}
	box := func(color ui.Color, w, h float64, radius float64) rendering.Box {
		dec := painting.NewBoxDecoration(painting.BoxDecorationOpts{Color: foundation.Some(color), BorderRadius: painting.NewBorderRadiusCircular(radius)})
		return rendering.NewRenderDecoratedBox(dec, rendering.NewRenderConstrainedBox(
			rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{Width: foundation.Some(w), Height: foundation.Some(h)}), nil))
	}
	gap := func(h float64) rendering.Box {
		return rendering.NewRenderConstrainedBox(rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{Height: foundation.Some(h)}), nil)
	}

	// A row: two boxes of a fixed size around a box that takes the rest.
	flexible := box(teal, 0, 60, 12)
	row := rendering.NewRenderFlex([]rendering.Box{box(purple, 80, 60, 12), flexible, box(orange, 80, 60, 12)}, rendering.RenderFlexOpts{
		Direction: painting.AxisHorizontal, TextDirection: ui.TextDirectionLTR, CrossAxisAlignment: rendering.CrossAxisAlignmentCenter,
	})
	flexible.ParentData().(*rendering.FlexParentData).Flex = foundation.Some(1)

	// A stack: a card with a badge in its corner and a caption in the other.
	badge := rendering.NewRenderDecoratedBox(painting.NewBoxDecoration(painting.BoxDecorationOpts{
		Color: foundation.Some(red), Shape: painting.BoxShapeCircle}),
		rendering.NewRenderPadding(painting.NewEdgeInsetsAll(10), paragraph("3", 20, ui.FontWeightW700, argb(0xFFFFFFFF))))
	caption := paragraph("Stack", 18, ui.FontWeightW500, argb(0xFFFFFFFF))
	card := rendering.NewRenderStack([]rendering.Box{box(argb(0xFF3F51B5), 600, 100, 16), badge, caption},
		rendering.RenderStackOpts{TextDirection: ui.TextDirectionLTR})
	badge.ParentData().(*rendering.StackParentData).Top = foundation.Some(10.0)
	badge.ParentData().(*rendering.StackParentData).Right = foundation.Some(12.0)
	caption.ParentData().(*rendering.StackParentData).Left = foundation.Some(16.0)
	caption.ParentData().(*rendering.StackParentData).Bottom = foundation.Some(12.0)

	// A wrap of chips.
	var chips []rendering.Box
	for i, label := range []string{"Layout", "Paint", "Hit testing", "Intrinsics", "Baselines", "Clipping", "Text", "Images", "Selection"} {
		colors := []ui.Color{argb(0xFFEADDFF), argb(0xFFC8E6C9), argb(0xFFFFE0B2)}
		dec := painting.NewBoxDecoration(painting.BoxDecorationOpts{
			Color: foundation.Some(colors[i%len(colors)]), BorderRadius: painting.NewBorderRadiusCircular(14)})
		chips = append(chips, rendering.NewRenderDecoratedBox(dec, rendering.NewRenderPadding(
			painting.NewEdgeInsetsSymmetric(5, 12), paragraph(label, 14, ui.FontWeightW400, argb(0xFF1D1B20)))))
	}
	wrap := rendering.NewRenderWrap(chips, rendering.RenderWrapOpts{Spacing: 8, RunSpacing: 8, TextDirection: ui.TextDirectionLTR})

	// A paragraph cut at two lines.
	long := paragraph("A render object lays out its children with constraints that go down, and they answer with sizes that go up. "+
		"Then it positions them, paints them and lets pointer events find them. This paragraph has more text than two lines hold.",
		16, ui.FontWeightW400, argb(0xFF49454F), rendering.RenderParagraphOpts{
			MaxLines: foundation.Some(2), Overflow: painting.TextOverflowEllipsis})

	// A generated image, laid out at its aspect ratio.
	image := rendering.NewRenderImage(rendering.RenderImageOpts{Image: tourImage(96, 48), Fit: painting.BoxFitCover, FilterQuality: ui.FilterQualityMedium})
	imageRow := rendering.NewRenderFlex([]rendering.Box{
		rendering.NewRenderConstrainedBox(rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{Width: foundation.Some(192.0)}), image),
		rendering.NewRenderPadding(painting.NewEdgeInsetsAll(12), paragraph("RenderImage keeps the aspect ratio of the picture.", 14, ui.FontWeightW400, argb(0xFF49454F))),
	}, rendering.RenderFlexOpts{Direction: painting.AxisHorizontal, TextDirection: ui.TextDirectionLTR, CrossAxisAlignment: rendering.CrossAxisAlignmentCenter})

	column := rendering.NewRenderFlex([]rendering.Box{
		paragraph("Rendering tour", 32, ui.FontWeightW700, purple), gap(16),
		row, gap(16),
		card, gap(16),
		wrap, gap(16),
		long, gap(16),
		imageRow,
	}, rendering.RenderFlexOpts{
		Direction: painting.AxisVertical, MainAxisSize: rendering.MainAxisSizeMin, CrossAxisAlignment: rendering.CrossAxisAlignmentStretch,
	})
	page := rendering.NewRenderDecoratedBox(
		painting.NewBoxDecoration(painting.BoxDecorationOpts{Color: foundation.Some(argb(0xFFFEF7FF))}),
		rendering.NewRenderPadding(painting.NewEdgeInsetsAll(20), column))
	return page
}

// tourImage is a width x height picture with a diagonal gradient and a bright dot.
func tourImage(width, height int) *ui.Image {
	pix := make([]byte, 4*width*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			i := 4 * (y*width + x)
			pix[i] = byte(255 * x / width)
			pix[i+1] = byte(255 * y / height)
			pix[i+2] = 200
			pix[i+3] = 255
			dx, dy := x-width*3/4, y-height/2
			if dx*dx+dy*dy < 100 {
				pix[i], pix[i+1], pix[i+2] = 255, 255, 255
			}
		}
	}
	return ui.NewImageFromRGBA(pix, width, height)
}
