package explicit

import (
	"image"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/ui"
)

// logTextureOnce logs, at Info level, that a texture layer draws nothing, once for each texture id and reason.
func (b *frameBuilder) logTextureOnce(id int64, reason string) {
	if b.log == nil {
		return
	}
	key := id
	if b.missing[key] {
		return
	}
	if b.missing == nil || len(b.missing) > maxMissingTextures {
		b.missing = map[int64]bool{}
	}
	b.missing[key] = true
	b.log.Info("a texture layer draws nothing", "texture", id, "reason", reason)
}

// maxMissingTextures is how many ids the log-once table holds before it starts over.
const maxMissingTextures = 1024

// texture draws a texture layer (SPEC §27.4): the registered texture, scaled to the layer's rectangle, premultiplied,
// with the layer's filter quality, under the command's transform and clip. It is an image draw: the image shader reads
// the texture directly, so nothing is copied; an RGBA16Float texture holds linear values, which the shader encodes.
// A texture that is not registered, or that the shader cannot read, draws nothing and is logged once.
func (s *shapeContext) texture(l *ui.TextureLayer) (bool, error) {
	b := s.b
	if b.textures == nil {
		b.logTextureOnce(l.TextureID, "the app has no GPU textures")
		return true, nil
	}
	tex, ok := b.textures.Lookup(b.dev, l.TextureID)
	if !ok {
		b.logTextureOnce(l.TextureID, "the texture is not registered")
		return true, nil
	}
	d := tex.Desc()
	if d.Kind != TextureKind2D || d.SampleCount != 1 || d.Usage&TextureSampled == 0 {
		b.logTextureOnce(l.TextureID, "the texture is not a 2D texture of one sample that shaders sample")
		return true, nil
	}
	linear := false
	switch d.Format {
	case TextureFormatRGBA8Unorm, TextureFormatBGRA8Unorm:
	case TextureFormatRGBA16Float:
		linear = true
	default:
		b.logTextureOnce(l.TextureID, "the format of the texture is not RGBA8Unorm, BGRA8Unorm or RGBA16Float")
		return true, nil
	}
	if l.Width <= 0 || l.Height <= 0 {
		return true, nil
	}
	var info paintInfo
	var sp gpu.PaintSpec
	b.setBlend(&info, &sp, ui.BlendModeSrcOver)
	info.color = [4]float32{1, 1, 1, 1}
	info.aa = true
	info.spec = b.intern(&sp)
	entry := &imageEntry{tex: tex, w: d.Width, h: d.Height, levels: d.MipLevels}
	paint := ui.NewPaint()
	paint.SetFilterQuality(l.FilterQuality)
	enc := s.imageEncoder(info, entry, paint)
	if linear {
		emit := enc.Emit
		enc.Emit = func(in *gpu.ImageInstance, covers image.Rectangle) {
			in[11] = 1
			emit(in, covers)
		}
	}
	enc.Rect(d.Width, d.Height, ui.NewRectFromLTWH(0, 0, float64(d.Width), float64(d.Height)),
		ui.NewRectFromLTWH(l.Offset.Dx, l.Offset.Dy, l.Width, l.Height))
	return true, nil
}
