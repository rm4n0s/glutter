package rendering

import "github.com/rm4n0s/glutter/ui"

// TextureBox is a rectangle upon which a texture of package gpu is shown (SPEC §27.4). It takes as much room as
// its parent gives, is a repaint boundary, and always needs compositing, since it paints a TextureLayer.
//
// Flutter: rendering/texture.dart TextureBox
type TextureBox struct {
	RenderBox
	textureID     int64
	freeze        bool
	filterQuality ui.FilterQuality
}

// TextureBoxOpts are the optional parameters of NewTextureBox.
//
// Flutter: rendering/texture.dart TextureBox.new
type TextureBoxOpts struct {
	// Freeze, when true, keeps the texture's last frame while the app resizes it; the default is false.
	Freeze bool
	// FilterQuality is the quality of the scaling; the default is FilterQualityLow.
	FilterQuality ui.FilterQuality
}

// NewTextureBox creates a box that shows the texture with the id.
//
// Flutter: rendering/texture.dart TextureBox.new
func NewTextureBox(textureID int64, opts ...TextureBoxOpts) *TextureBox {
	var o TextureBoxOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.FilterQuality == ui.FilterQualityUnspecified {
		o.FilterQuality = ui.FilterQualityLow
	}
	b := &TextureBox{textureID: textureID, freeze: o.Freeze, filterQuality: o.FilterQuality}
	InitRenderBox(&b.RenderBox, b)
	return b
}

// TextureID is the id the registry gave the texture.
//
// Flutter: rendering/texture.dart TextureBox.textureId
func (b *TextureBox) TextureID() int64 { return b.textureID }

// SetTextureID changes the texture and repaints when it differs.
//
// Flutter: rendering/texture.dart TextureBox.textureId
func (b *TextureBox) SetTextureID(v int64) {
	if v != b.textureID {
		b.textureID = v
		b.MarkNeedsPaint()
	}
}

// Freeze says that the box keeps showing the texture's last frame while the app resizes it.
//
// Flutter: rendering/texture.dart TextureBox.freeze
func (b *TextureBox) Freeze() bool { return b.freeze }

// SetFreeze changes freeze and repaints when it differs.
//
// Flutter: rendering/texture.dart TextureBox.freeze
func (b *TextureBox) SetFreeze(v bool) {
	if v != b.freeze {
		b.freeze = v
		b.MarkNeedsPaint()
	}
}

// FilterQuality is the quality of the scaling.
//
// Flutter: rendering/texture.dart TextureBox.filterQuality
func (b *TextureBox) FilterQuality() ui.FilterQuality { return b.filterQuality }

// SetFilterQuality changes the quality and repaints when it differs.
//
// Flutter: rendering/texture.dart TextureBox.filterQuality
func (b *TextureBox) SetFilterQuality(v ui.FilterQuality) {
	if v == ui.FilterQualityUnspecified {
		v = ui.FilterQualityLow
	}
	if v != b.filterQuality {
		b.filterQuality = v
		b.MarkNeedsPaint()
	}
}

// SizedByParent is true: the box takes as much room as it is given.
//
// Flutter: rendering/texture.dart TextureBox.sizedByParent
func (b *TextureBox) SizedByParent() bool { return true }

// AlwaysNeedsCompositing is true.
//
// Flutter: rendering/texture.dart TextureBox.alwaysNeedsCompositing
func (b *TextureBox) AlwaysNeedsCompositing() bool { return true }

// IsRepaintBoundary is true.
//
// Flutter: rendering/texture.dart TextureBox.isRepaintBoundary
func (b *TextureBox) IsRepaintBoundary() bool { return true }

// ComputeDryLayout is the biggest size the constraints allow.
//
// Flutter: rendering/texture.dart TextureBox.computeDryLayout
func (b *TextureBox) ComputeDryLayout(c BoxConstraints) ui.Size { return c.Biggest() }

// HitTestSelf is true.
//
// Flutter: rendering/texture.dart TextureBox.hitTestSelf
func (b *TextureBox) HitTestSelf(ui.Offset) bool { return true }

// Paint adds a TextureLayer for the box's rectangle.
//
// Flutter: rendering/texture.dart TextureBox.paint
func (b *TextureBox) Paint(context *PaintingContext, offset ui.Offset) {
	size := b.Size()
	context.AddLayer(NewTextureLayer(ui.NewRectFromLTWH(offset.Dx, offset.Dy, size.Width, size.Height), b.textureID,
		TextureLayerOpts{Freeze: b.freeze, FilterQuality: b.filterQuality}))
}
