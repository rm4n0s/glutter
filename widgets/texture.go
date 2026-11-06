package widgets

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// TextureOpts are the optional parameters of NewTexture.
//
// Flutter: widgets/texture.dart Texture.new
type TextureOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Freeze, when true, keeps the texture's last frame while the app resizes it; the default is false.
	Freeze bool
	// FilterQuality is the quality of the scaling; the default is FilterQualityLow.
	FilterQuality ui.FilterQuality
}

// Texture is a rectangle upon which a texture of package gpu is shown (SPEC §27.4). The texture is registered
// with Device.RegisterTexture, which gives its id. The widget takes as much room as its parent gives; wrap it in
// a SizedBox or an AspectRatio to size it.
//
// Flutter: widgets/texture.dart Texture
type Texture struct {
	LeafRenderObjectWidget
	textureID     int64
	freeze        bool
	filterQuality ui.FilterQuality
}

// NewTexture creates a widget that shows the texture with the id.
//
// Flutter: widgets/texture.dart Texture.new
func NewTexture(textureID int64, opts ...TextureOpts) *Texture {
	o := firstOpts(opts)
	if o.FilterQuality == ui.FilterQualityUnspecified {
		o.FilterQuality = ui.FilterQualityLow
	}
	return &Texture{LeafRenderObjectWidget: NewLeafRenderObjectWidgetBase(WidgetOpts{Key: o.Key}), textureID: textureID, freeze: o.Freeze, filterQuality: o.FilterQuality}
}

// TextureID is the id the registry gave the texture.
//
// Flutter: widgets/texture.dart Texture.textureId
func (w *Texture) TextureID() int64 { return w.textureID }

// Freeze says that the widget keeps showing the texture's last frame while the app resizes it.
//
// Flutter: widgets/texture.dart Texture.freeze
func (w *Texture) Freeze() bool { return w.freeze }

// FilterQuality is the quality of the scaling.
//
// Flutter: widgets/texture.dart Texture.filterQuality
func (w *Texture) FilterQuality() ui.FilterQuality { return w.filterQuality }

// CreateRenderObject creates the render box.
//
// Flutter: widgets/texture.dart Texture.createRenderObject
func (w *Texture) CreateRenderObject(BuildContext) rendering.RenderObject {
	return rendering.NewTextureBox(w.textureID, rendering.TextureBoxOpts{Freeze: w.freeze, FilterQuality: w.filterQuality})
}

// UpdateRenderObject copies the configuration to the render box.
//
// Flutter: widgets/texture.dart Texture.updateRenderObject
func (w *Texture) UpdateRenderObject(_ BuildContext, renderObject rendering.RenderObject) {
	b := renderObject.(*rendering.TextureBox)
	b.SetTextureID(w.textureID)
	b.SetFreeze(w.freeze)
	b.SetFilterQuality(w.filterQuality)
}
