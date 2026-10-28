package offscreen

import (
	"image"
	"math"
	"sort"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/internal/gpu/textdraw"
	"github.com/rm4n0s/glutter/ui"
)

// The limits of a glyph sheet, in pixels: the side of one glyph, the side of the sheet and its area.
const (
	maxGlyphSide  = 1 << 13
	maxSheetSide  = 1 << 14
	maxSheetArea  = 1 << 25
	glyphPadding  = 1
	sheetWidthMul = 1.15
)

// Glyph is one grapheme cluster of a GlyphSheet: where its pixels are in the image and how to place
// them. All the lengths are in the pixels of the image.
//
// To draw the cluster with its pen (its origin on the baseline) at (x, y) of a canvas, draw the part
// Rect of the image with its top left corner at (x+Offset.Dx, y+Offset.Dy), and then move the pen right
// by Advance. DrawAtlas does that with an RSTransform whose translation is that point and whose scale is 1.
//
// Flutter: no counterpart
type Glyph struct {
	// Cluster is the text of the cluster, as it was asked.
	Cluster string
	// Rect is the part of the image that has the pixels of the cluster; it is empty for a cluster that
	// draws nothing (a space). The pixels around it, up to the next rectangle, are transparent.
	Rect ui.Rect
	// Offset is the vector from the pen position, on the baseline, to the top left corner of Rect.
	Offset ui.Offset
	// Advance is how far the pen moves right after the cluster: the width of a paragraph of the cluster,
	// times the device pixel ratio.
	Advance float64
	// Colored says that the cluster is drawn with colors of its own (a color emoji), which the pixels
	// keep, so that tinting multiplies them. Other clusters are white, with the coverage as alpha.
	Colored bool
}

// GlyphSheet is an image of many grapheme clusters drawn in one text style, with the place of each
// (NewGlyphSheet). It is what a grid of text draws from, with Canvas.DrawAtlas: one draw tints
// thousands of cells.
//
// Flutter: no counterpart
type GlyphSheet struct {
	// Image has the pixels of every cluster: premultiplied RGBA, white where an outline glyph covers
	// the pixel and the coverage as the alpha, transparent elsewhere. The sheet gives it to the
	// caller, who disposes it.
	Image *ui.Image
	// DevicePixelRatio is the ratio the clusters were drawn at: the font size in pixels is the size
	// of the style times it.
	DevicePixelRatio float64
	// Glyphs has one entry for each cluster that NewGlyphSheet was given, in its order. A cluster that
	// was given twice shares its Rect.
	Glyphs []Glyph
}

// NewGlyphSheet shapes each cluster with the text engine, with the same fonts, fallback fonts and glyph
// outlines that a paragraph in the style uses, and draws them all into one image with the software
// renderer. A cluster is drawn the way a paragraph draws it with its origin on a whole pixel: the same
// paths, quarter-pixel positions and coverage, in white (the style's color, shadows, decorations and
// background are left out); a color glyph keeps its colors. No paragraph is drawn or recorded for each
// glyph, and nothing is kept after the call: the engine's own caches are the only state. devicePixelRatio
// is how many pixels a logical pixel is, so the glyphs are drawn at size*ratio. A nil style is the
// default style, and a cluster that is empty or draws nothing gets an empty Rect and its advance. It
// is for the UI thread, which owns the text engine. It fails with GlyphEngineMissing when engine is nil,
// GlyphEngineUnsupported when the engine's paragraphs cannot paint themselves, GlyphClustersMissing for
// an empty list, GlyphPixelRatioInvalid for a ratio that is not a positive finite number, and
// GlyphSheetTooLarge when a glyph or the sheet would not fit in memory.
//
// Flutter: no counterpart
func NewGlyphSheet(engine ui.ParagraphFactory, style *ui.TextStyle, devicePixelRatio float64, clusters []string) (*GlyphSheet, error) {
	if engine == nil {
		return nil, errors.New(tagGlyphEngineMissing, "NewGlyphSheet needs the text engine")
	}
	if len(clusters) == 0 {
		return nil, errors.New(tagGlyphClustersMissing, "NewGlyphSheet needs at least one cluster")
	}
	if !(devicePixelRatio > 0) || math.IsInf(devicePixelRatio, 0) {
		return nil, errors.New(tagGlyphPixelRatioInvalid, "the device pixel ratio of a glyph sheet is a positive finite number",
			"ratio", devicePixelRatio)
	}
	resolved := glyphStyle(style)

	// Shape each different cluster, and find the box of its pixels.
	index := make(map[string]int, len(clusters))
	var shapes []*clusterShape
	of := make([]int, len(clusters))
	for i, text := range clusters {
		k, ok := index[text]
		if !ok {
			k = len(shapes)
			index[text] = k
			s := &clusterShape{text: text}
			if text != "" {
				para := shapeCluster(engine, text, &resolved)
				s.advance = para.MaxIntrinsicWidth() * devicePixelRatio
				s.penY = int(math.Round(devicePixelRatio * para.AlphabeticBaseline()))
				if err := s.measure(para, devicePixelRatio); err != nil {
					return nil, err
				}
			}
			shapes = append(shapes, s)
		}
		of[i] = k
	}

	width, height, err := pack(shapes)
	if err != nil {
		return nil, err
	}

	// Draw them all in one scene: the operations that each paragraph painted, moved to their places.
	var ops []ui.DisplayOp
	for _, s := range shapes {
		if s.ink.Empty() {
			continue
		}
		dx, dy := float32(s.at.X-s.ink.Min.X), float32(s.at.Y-s.ink.Min.Y)
		for _, d := range s.drawn {
			m := ui.NewMatrix4TranslationValues(dx, dy, 0)
			m.MultiplyBy(&d.m)
			ops = append(ops, ui.OpSave{}, ui.OpTransform{Matrix: *m}, d.op, ui.OpRestore{})
		}
	}
	sb := ui.NewSceneBuilder()
	sb.AddPicture(ui.Offset{}, ui.NewPictureFromOps(ui.NewRectFromLTWH(0, 0, float64(width), float64(height)), ops))
	pixels, err := soft.Render(sb.Build(), width, height)
	if err != nil {
		return nil, err
	}

	sheet := &GlyphSheet{Image: ui.NewImageFromRGBA(pixels.Pix, width, height), DevicePixelRatio: devicePixelRatio, Glyphs: make([]Glyph, len(clusters))}
	for i, text := range clusters {
		s := shapes[of[i]]
		g := Glyph{Cluster: text, Advance: s.advance, Colored: s.colored}
		if !s.ink.Empty() {
			g.Rect = ui.Rect{Left: float64(s.at.X), Top: float64(s.at.Y), Right: float64(s.at.X + s.ink.Dx()), Bottom: float64(s.at.Y + s.ink.Dy())}
			g.Offset = ui.Offset{Dx: float64(s.ink.Min.X), Dy: float64(s.ink.Min.Y - s.penY)}
		}
		sheet.Glyphs[i] = g
	}
	return sheet, nil
}

// glyphStyle resolves the style against the defaults of a paragraph and takes out what a sheet does
// not draw: the sheet is white coverage, so the color is opaque white and the paints, shadows and
// decorations are gone.
func glyphStyle(style *ui.TextStyle) ui.ResolvedTextStyle {
	resolved := ui.NewParagraphStyle().BaseTextStyle()
	if style != nil {
		resolved = style.Resolve(resolved)
	}
	resolved.Color = ui.NewColor(0xFFFFFFFF)
	resolved.Foreground, resolved.Background = nil, nil
	resolved.Shadows = nil
	resolved.Decoration = ui.TextDecorationNone
	return resolved
}

// shapeCluster makes the laid-out paragraph of a cluster, of one line as wide as it needs.
func shapeCluster(engine ui.ParagraphFactory, text string, style *ui.ResolvedTextStyle) *ui.Paragraph {
	p := ui.NewParagraph(engine.BuildParagraph(&ui.ParagraphContent{
		Style: ui.NewParagraphStyle(),
		Text:  text,
		Runs:  []ui.TextRun{{Start: 0, End: len(text), Style: style}},
	}))
	p.Layout(ui.NewParagraphConstraints(math.Inf(1)))
	return p
}

// clusterShape is a cluster that the sheet draws.
type clusterShape struct {
	text    string
	advance float64
	// penY is the baseline, in pixels below the top of the paragraph when its origin is on a whole pixel.
	penY int
	// drawn is what the paragraph of the cluster paints, with its transforms, when its origin is the
	// origin of the sheet's pixels; empty for an empty cluster.
	drawn []drawnOp
	// ink is the box of the pixels, in pixels from the paragraph's origin.
	ink     image.Rectangle
	colored bool
	// at is the top left corner of ink's pixels in the sheet.
	at image.Point
}

// drawnOp is an operation of a paragraph (a glyph's path or picture) and the transform it is drawn under.
type drawnOp struct {
	op ui.DisplayOp
	m  ui.Matrix4
}

// measure paints the paragraph under the scale of the ratio, keeps what it paints, and finds the box
// of its pixels from the paths under the transforms the drawing has. A color glyph is a picture,
// whose box is found by drawing it.
func (s *clusterShape) measure(para *ui.Paragraph, ratio float64) error {
	ctm := ui.NewMatrix4Diagonal3Values(float32(ratio), float32(ratio), 1)
	var failure error
	var ink image.Rectangle
	op := ui.OpDrawParagraph{Paragraph: para}
	if !textdraw.Expand(op, ctm, func(d ui.DisplayOp, m *ui.Matrix4) {
		if failure != nil {
			return
		}
		switch d := d.(type) {
		case ui.OpDrawPath:
			s.drawn = append(s.drawn, drawnOp{d, *m})
			ink = ink.Union(outerBox(m.TransformRect(d.Path.GetBounds())))
		case ui.OpDrawPicture:
			s.drawn = append(s.drawn, drawnOp{d, *m})
			box, err := pictureInk(d.Picture, m)
			if err != nil {
				failure = err
				return
			}
			if !box.Empty() {
				s.colored = true
			}
			ink = ink.Union(box)
		}
	}) {
		return errors.New(tagGlyphEngineUnsupported, "the paragraphs of the text engine cannot paint themselves")
	}
	if failure != nil {
		return failure
	}
	if ink.Dx() > maxGlyphSide || ink.Dy() > maxGlyphSide {
		return errors.New(tagGlyphSheetTooLarge, "a glyph of the sheet is too large", "cluster", s.text, "width", ink.Dx(), "height", ink.Dy())
	}
	s.ink = ink
	if ink.Empty() {
		s.drawn = nil
	}
	return nil
}

// outerBox is the smallest whole-pixel box that holds a rectangle. A rectangle with no area, or with
// a coordinate that is not finite, gives the empty box.
func outerBox(r ui.Rect) image.Rectangle {
	if !r.IsFinite() || r.IsEmpty() {
		return image.Rectangle{}
	}
	return image.Rect(int(math.Floor(r.Left)), int(math.Floor(r.Top)), int(math.Ceil(r.Right)), int(math.Ceil(r.Bottom)))
}

// pictureInk draws the picture of a color glyph under the transform and returns the box of the pixels
// that it covers, in the coordinates of the transform's target.
func pictureInk(pic *ui.Picture, m *ui.Matrix4) (image.Rectangle, error) {
	region := outerBox(m.TransformRect(pic.CullRect()))
	if region.Empty() {
		return image.Rectangle{}, nil
	}
	if region.Dx() > maxGlyphSide || region.Dy() > maxGlyphSide {
		return image.Rectangle{}, errors.New(tagGlyphSheetTooLarge, "a color glyph of the sheet is too large", "width", region.Dx(), "height", region.Dy())
	}
	rec := ui.NewPictureRecorder()
	c := ui.NewCanvas(rec)
	c.Translate(float64(-region.Min.X), float64(-region.Min.Y))
	c.Transform(m)
	c.DrawPicture(pic)
	sb := ui.NewSceneBuilder()
	sb.AddPicture(ui.Offset{}, rec.EndRecording())
	img, err := soft.Render(sb.Build(), region.Dx(), region.Dy())
	if err != nil {
		return image.Rectangle{}, err
	}
	var box image.Rectangle
	for y := 0; y < region.Dy(); y++ {
		row := img.Pix[y*img.Stride : y*img.Stride+4*region.Dx()]
		for x := 0; x < region.Dx(); x++ {
			if row[4*x+3] != 0 {
				box = box.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	return box.Add(region.Min), nil
}

// pack places the pixels of the shapes in rows of a sheet, the tallest first, with a transparent pixel
// around each, and returns the size of the sheet. A sheet with nothing to draw is one pixel.
func pack(shapes []*clusterShape) (width, height int, err error) {
	var drawn []*clusterShape
	area, widest := 0, 0
	for _, s := range shapes {
		if s.ink.Empty() {
			continue
		}
		drawn = append(drawn, s)
		w, h := s.ink.Dx()+2*glyphPadding, s.ink.Dy()+2*glyphPadding
		area += w * h
		widest = max(widest, w)
	}
	if len(drawn) == 0 {
		return 1, 1, nil
	}
	sort.SliceStable(drawn, func(i, j int) bool { return drawn[i].ink.Dy() > drawn[j].ink.Dy() })
	width = max(widest, (int(math.Ceil(math.Sqrt(float64(area)*sheetWidthMul)))+15)/16*16)
	x, y, rowHeight := 0, 0, 0
	for _, s := range drawn {
		w, h := s.ink.Dx()+2*glyphPadding, s.ink.Dy()+2*glyphPadding
		if x+w > width {
			x, y, rowHeight = 0, y+rowHeight, 0
		}
		s.at = image.Pt(x+glyphPadding, y+glyphPadding)
		x += w
		rowHeight = max(rowHeight, h)
	}
	height = y + rowHeight
	if width > maxSheetSide || height > maxSheetSide || width*height > maxSheetArea {
		return 0, 0, errors.New(tagGlyphSheetTooLarge, "the glyph sheet would be too large", "width", width, "height", height, "glyphs", len(drawn))
	}
	return width, height, nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagGlyphEngineMissing     = "GlyphEngineMissing"
	tagGlyphEngineUnsupported = "GlyphEngineUnsupported"
	tagGlyphClustersMissing   = "GlyphClustersMissing"
	tagGlyphPixelRatioInvalid = "GlyphPixelRatioInvalid"
	tagGlyphSheetTooLarge     = "GlyphSheetTooLarge"
)
