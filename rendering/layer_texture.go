package rendering

import "github.com/rm4n0s/glutter/ui"

// TextureLayer is a composited layer that shows a texture that the app registered with package gpu
// (Device.RegisterTexture), scaled to a rectangle (SPEC §27.4).
//
// Flutter: rendering/layer.dart TextureLayer
type TextureLayer struct {
	LayerBase
	rect          ui.Rect
	textureID     int64
	freeze        bool
	filterQuality ui.FilterQuality
}

// TextureLayerOpts are the optional parameters of NewTextureLayer.
//
// Flutter: rendering/layer.dart TextureLayer.new
type TextureLayerOpts struct {
	// Freeze, when true, keeps the texture's last frame while the app resizes it; the default is false.
	Freeze bool
	// FilterQuality is the quality of the scaling; the default is FilterQualityLow.
	FilterQuality ui.FilterQuality
}

// NewTextureLayer creates a layer for the texture with the id, shown in the rectangle.
//
// Flutter: rendering/layer.dart TextureLayer.new
func NewTextureLayer(rect ui.Rect, textureID int64, opts ...TextureLayerOpts) *TextureLayer {
	var o TextureLayerOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.FilterQuality == ui.FilterQualityUnspecified {
		o.FilterQuality = ui.FilterQualityLow
	}
	l := &TextureLayer{rect: rect, textureID: textureID, freeze: o.Freeze, filterQuality: o.FilterQuality}
	InitLayer(&l.LayerBase, l)
	return l
}

// Rect is the rectangle the texture is shown in, in the layer's coordinates.
//
// Flutter: rendering/layer.dart TextureLayer.rect
func (l *TextureLayer) Rect() ui.Rect { return l.rect }

// TextureID is the id the registry gave the texture.
//
// Flutter: rendering/layer.dart TextureLayer.textureId
func (l *TextureLayer) TextureID() int64 { return l.textureID }

// Freeze says that the layer keeps showing the texture's last frame while the app resizes it.
//
// Flutter: rendering/layer.dart TextureLayer.freeze
func (l *TextureLayer) Freeze() bool { return l.freeze }

// FilterQuality is the quality of the scaling.
//
// Flutter: rendering/layer.dart TextureLayer.filterQuality
func (l *TextureLayer) FilterQuality() ui.FilterQuality { return l.filterQuality }

// AddToScene adds the texture to the scene.
//
// Flutter: rendering/layer.dart TextureLayer.addToScene
func (l *TextureLayer) AddToScene(builder *ui.SceneBuilder) {
	builder.AddTexture(l.textureID, ui.AddTextureOpts{
		Offset: l.rect.TopLeft(), Width: l.rect.Width(), Height: l.rect.Height(), Freeze: l.freeze, FilterQuality: l.filterQuality,
	})
}

// FindAnnotations finds nothing: a texture has no annotations.
//
// Flutter: rendering/layer.dart TextureLayer.findAnnotations
func (l *TextureLayer) FindAnnotations(*AnnotationResult, ui.Offset, bool) bool { return false }
