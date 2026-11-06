package gpu

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

const textureUsageKnown = TextureUsageTextureBinding | TextureUsageRenderAttachment | TextureUsageCopySrc | TextureUsageStorageBinding

func (f TextureFormat) isDepthStencil() bool { return explicit.TextureFormat(f).IsDepthStencil() }
func (f TextureFormat) allowedUsage() TextureUsage {
	return TextureUsage(explicit.TextureFormat(f).AllowedUsage())
}
func (f TextureFormat) valid() bool {
	return f > TextureFormatUnspecified && f <= TextureFormatDepth24PlusStencil8
}

// Texture is an image, an array of images, a cube or a volume in GPU memory (WebGPU's GPUTexture), with mip levels
// and, for a render attachment, samples. A shader samples it or a pass draws into it through a TextureView. Release
// it when it is not needed; the memory is freed when the work that uses it is done.
//
// Flutter: no counterpart
type Texture struct {
	resource
	exp explicit.Texture
	// desc is the description with the defaults filled in.
	desc  explicit.TextureDesc
	views map[viewKey]*TextureView
}

type viewKey struct {
	level, layer int
}

// whole is the key of the view of the whole texture.
var whole = viewKey{-1, -1}

// validateTexture checks a description. It fails with GPUInvalidDescriptor for a texture that cannot be made (no
// format, a size of 0, a use that the format does not allow, more mip levels than the size has, a cube that is not
// square, a volume that is drawn into, a multisampled texture that is more than a render attachment), and with
// GPULimitExceeded for a size over the limits.
func (d *Device) validateTexture(desc TextureDesc, n explicit.TextureDesc) error {
	if !desc.Format.valid() {
		return errors.New(TagGPUInvalidDescriptor, "a texture has a format", "field", "Format", "value", int(desc.Format), "label", desc.Label)
	}
	if desc.Kind < TextureKindUnspecified || desc.Kind > TextureKind3D {
		return errors.New(TagGPUInvalidDescriptor, "the kind of a texture is one of the TextureKind values", "field", "Kind", "value", int(desc.Kind), "label", desc.Label)
	}
	if desc.Size.Width < 1 || desc.Size.Height < 1 || desc.Size.DepthOrArrayLayers < 0 {
		return errors.New(TagGPUInvalidDescriptor, "a texture is at least 1 by 1", "field", "Size", "value", desc.Size, "label", desc.Label)
	}
	maxSide := d.limits.MaxTextureSize
	if n.Kind == explicit.TextureKind3D {
		maxSide = d.limits.MaxTextureSize3D
	}
	if desc.Size.Width > maxSide || desc.Size.Height > maxSide {
		return errors.New(TagGPULimitExceeded, "the texture is larger than the limit for its kind", "field", "Size", "value", desc.Size, "max", maxSide, "label", desc.Label)
	}
	switch n.Kind {
	case explicit.TextureKind2DArray:
		if n.DepthOrLayers > d.limits.MaxTextureArrayLayers {
			return errors.New(TagGPULimitExceeded, "the texture has more layers than Limits.MaxTextureArrayLayers", "field", "Size.DepthOrArrayLayers", "value", n.DepthOrLayers, "max", d.limits.MaxTextureArrayLayers, "label", desc.Label)
		}
	case explicit.TextureKind3D:
		if n.DepthOrLayers > maxSide {
			return errors.New(TagGPULimitExceeded, "the volume is deeper than Limits.MaxTextureDimension3D", "field", "Size.DepthOrArrayLayers", "value", n.DepthOrLayers, "max", maxSide, "label", desc.Label)
		}
	}
	if desc.Usage == 0 || desc.Usage&^textureUsageKnown != 0 {
		return errors.New(TagGPUInvalidDescriptor, "a texture has at least one use, and only the uses that TextureUsage has", "field", "Usage", "value", int(desc.Usage), "label", desc.Label)
	}
	if allowed := desc.Format.allowedUsage(); desc.Usage&^allowed != 0 {
		return errors.New(TagGPUInvalidDescriptor, "the format does not allow one of the uses", "field", "Usage", "value", int(desc.Usage), "allowed", int(allowed), "format", int(desc.Format), "label", desc.Label)
	}
	if levels := explicit.MipLevelsOf(n.Width, n.Height); n.MipLevels > levels {
		return errors.New(TagGPUInvalidDescriptor, "the texture has more mip levels than its size has", "field", "MipLevelCount", "value", n.MipLevels, "max", levels, "label", desc.Label)
	}
	switch n.Kind {
	case explicit.TextureKind2D:
		if n.DepthOrLayers != 1 {
			return errors.New(TagGPUInvalidDescriptor, "a 2D texture has one layer: use TextureKind2DArray", "field", "Size.DepthOrArrayLayers", "value", n.DepthOrLayers, "label", desc.Label)
		}
	case explicit.TextureKindCube:
		if n.DepthOrLayers != 6 || n.Width != n.Height {
			return errors.New(TagGPUInvalidDescriptor, "a cube has six square faces", "field", "Size", "value", desc.Size, "label", desc.Label)
		}
	case explicit.TextureKind3D:
		if desc.Usage&TextureUsageRenderAttachment != 0 || desc.Format.isDepthStencil() {
			return errors.New(TagGPUInvalidDescriptor, "a volume is not drawn into and has no depth format", "field", "Usage", "value", int(desc.Usage), "label", desc.Label)
		}
	}
	if n.SampleCount != 1 && n.SampleCount != 4 {
		return errors.New(TagGPUInvalidDescriptor, "a texture has 1 or 4 samples", "field", "SampleCount", "value", n.SampleCount, "label", desc.Label)
	}
	if n.SampleCount > 1 && (n.Kind != explicit.TextureKind2D || n.MipLevels != 1 || desc.Usage != TextureUsageRenderAttachment) {
		return errors.New(TagGPUInvalidDescriptor, "a multisampled texture is one level of a 2D texture that is only a render attachment", "field", "SampleCount", "value", n.SampleCount, "label", desc.Label)
	}
	return nil
}

// NewTexture makes a texture (WebGPU's GPUDevice.createTexture). It fails with GPUInvalidDescriptor and
// GPULimitExceeded for a description that cannot be made (the metadata "field" and "value" say which part), with
// GPUDeviceLost on a lost device, and with the error of the backend (GPUOutOfMemory) untouched.
//
// Flutter: no counterpart
func (d *Device) NewTexture(desc TextureDesc) (*Texture, error) {
	d.assertThread("NewTexture")
	if d.lostErr != nil {
		return nil, d.lostErr
	}
	ed := explicit.TextureDesc{
		Width: desc.Size.Width, Height: desc.Size.Height, Kind: explicit.TextureKind(desc.Kind), DepthOrLayers: desc.Size.DepthOrArrayLayers,
		Format: explicit.TextureFormat(desc.Format), MipLevels: desc.MipLevelCount, SampleCount: desc.SampleCount, Usage: explicit.TextureUsage(desc.Usage),
	}
	n := ed.Normalized()
	if err := d.validateTexture(desc, n); err != nil {
		return nil, err
	}
	exp, err := d.dev.NewTexture(ed)
	if err != nil {
		return nil, err
	}
	return &Texture{resource: d.track(kindTexture, desc.Label, 1), exp: exp, desc: n, views: map[viewKey]*TextureView{}}, nil
}

// Size returns the width and height, and the layers of an array, 6 for a cube, or the depth of a volume (1 for a
// 2D texture).
//
// Flutter: no counterpart
func (t *Texture) Size() Extent3D {
	return Extent3D{Width: t.desc.Width, Height: t.desc.Height, DepthOrArrayLayers: t.desc.DepthOrLayers}
}

// Format returns the format of the texels.
//
// Flutter: no counterpart
func (t *Texture) Format() TextureFormat { return TextureFormat(t.desc.Format) }

// Kind returns the shape of the texture.
//
// Flutter: no counterpart
func (t *Texture) Kind() TextureKind { return TextureKind(t.desc.Kind) }

// MipLevelCount returns the number of mip levels.
//
// Flutter: no counterpart
func (t *Texture) MipLevelCount() int { return t.desc.MipLevels }

// SampleCount returns the number of samples of each texel, 1 or 4.
//
// Flutter: no counterpart
func (t *Texture) SampleCount() int { return t.desc.SampleCount }

// Usage returns the uses the texture was made for.
//
// Flutter: no counterpart
func (t *Texture) Usage() TextureUsage { return TextureUsage(t.desc.Usage) }

// levelSize returns the size of a mip level: the width, the height and the layers or the depth.
func (t *Texture) levelSize(level int) (w, h, depth int) {
	w, h = max(t.desc.Width>>level, 1), max(t.desc.Height>>level, 1)
	depth = t.desc.Layers()
	if t.desc.Kind == explicit.TextureKind3D {
		depth = max(t.desc.DepthOrLayers>>level, 1)
	}
	return w, h, depth
}

// boxInside says whether the box is inside its mip level.
func (t *Texture) boxInside(r explicit.TextureRegion) bool {
	if r.Level < 0 || r.Level >= t.desc.MipLevels || r.Size[0] < 1 || r.Size[1] < 1 || r.Size[2] < 1 {
		return false
	}
	w, h, d := t.levelSize(r.Level)
	o := r.Origin
	return o[0] >= 0 && o[1] >= 0 && o[2] >= 0 && o[0]+r.Size[0] <= w && o[1]+r.Size[1] <= h && o[2]+r.Size[2] <= d
}

// Release frees the texture, and the views of it, when the work that uses them is done. It may be called twice.
// After the device is lost it only forgets the texture.
//
// Flutter: no counterpart
func (t *Texture) Release() {
	t.d.assertThread("Texture.Release")
	if !t.untrack() {
		return
	}
	t.d.unregisterAll(t)
	if t.d.lostErr == nil {
		t.exp.Destroy()
	}
}

// TextureView is a view of a texture (WebGPU's GPUTextureView): the whole texture, one mip level of it, or one
// layer of a level. A view is not released: it goes with its texture. A pass draws into a view and a bind group
// gives a shader one.
//
// Flutter: no counterpart
type TextureView struct {
	tex *Texture
	exp explicit.Texture
	// level and layer are -1 for the whole texture.
	level, layer int
	desc         explicit.TextureDesc
}

// View returns the view of the whole texture, with all its mip levels and layers.
//
// Flutter: no counterpart
func (t *Texture) View() *TextureView {
	t.d.assertThread("Texture.View")
	t.usable("Texture.View")
	return t.view(whole, func() explicit.Texture { return t.exp })
}

// LevelView returns the view of one mip level, with all its layers. A sampled view of it reads that level alone; a
// storage binding takes level 0 of a texture.
//
// Flutter: no counterpart
func (t *Texture) LevelView(level int) *TextureView {
	t.d.assertThread("Texture.LevelView")
	t.usable("Texture.LevelView")
	misuse(level >= 0 && level < t.desc.MipLevels, "Texture.LevelView", "the texture has the mip level", "level", level, "levels", t.desc.MipLevels)
	return t.view(viewKey{level, -1}, func() explicit.Texture { return t.exp.Level(level) })
}

// LayerView returns the 2D view of one layer of one mip level of an array or a cube. A pass can draw into it and a
// copy can read it; on Direct3D 12 a shader cannot sample or write it (SPEC §12.10): sample the texture, or a
// LevelView.
//
// Flutter: no counterpart
func (t *Texture) LayerView(level, layer int) *TextureView {
	t.d.assertThread("Texture.LayerView")
	t.usable("Texture.LayerView")
	misuse(t.desc.Kind == explicit.TextureKind2DArray || t.desc.Kind == explicit.TextureKindCube, "Texture.LayerView", "the texture is an array or a cube")
	misuse(level >= 0 && level < t.desc.MipLevels && layer >= 0 && layer < t.desc.Layers(), "Texture.LayerView", "the texture has the level and the layer",
		"level", level, "layer", layer)
	return t.view(viewKey{level, layer}, func() explicit.Texture { return t.exp.Layer(level, layer) })
}

// view returns the view for the key, made by mk the first time.
func (t *Texture) view(k viewKey, mk func() explicit.Texture) *TextureView {
	if v, ok := t.views[k]; ok {
		return v
	}
	exp := mk()
	v := &TextureView{tex: t, exp: exp, level: k.level, layer: k.layer, desc: exp.Desc().Normalized()}
	t.views[k] = v
	return v
}

// Texture returns the texture that the view is of.
//
// Flutter: no counterpart
func (v *TextureView) Texture() *Texture { return v.tex }

// Size returns the size of the view: of its mip level, with one layer for a LayerView.
//
// Flutter: no counterpart
func (v *TextureView) Size() Extent3D {
	return Extent3D{Width: v.desc.Width, Height: v.desc.Height, DepthOrArrayLayers: v.desc.DepthOrLayers}
}

// Format returns the format of the texels.
//
// Flutter: no counterpart
func (v *TextureView) Format() TextureFormat { return TextureFormat(v.desc.Format) }
