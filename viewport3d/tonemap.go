package viewport3d

import (
	"log/slog"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/viewport3d/shaders"
)

// ToneMapping is the curve that turns the radiance of the PBR materials, which may be above 1, into the colors of the display
// (SPEC §28.5). A scene with a PBR item is drawn into an RGBA16Float target and tone mapped over the view.
//
// Flutter: no counterpart
type ToneMapping int

const (
	// ToneMappingUnspecified is the zero value: ToneMappingNeutral.
	ToneMappingUnspecified ToneMapping = iota
	// ToneMappingNeutral is the Khronos PBR Neutral tone mapper: the colors below about 0.76 keep their value, the brighter ones
	// are compressed towards white. It keeps the colors of a material as they were authored.
	ToneMappingNeutral
	// ToneMappingACES is the fit of the ACES reference rendering transform by Stephen Hill: a filmic curve with more contrast.
	ToneMappingACES
	// ToneMappingNone clips the radiance to 0 to 1.
	ToneMappingNone
)

// String names the curve.
//
// Flutter: no counterpart
func (t ToneMapping) String() string {
	switch t {
	case ToneMappingACES:
		return "ACES"
	case ToneMappingNone:
		return "none"
	}
	return "PBR Neutral"
}

// resolvedTone is the tone mapping with its defaults: the curve as the shader numbers it (0 none, 1 neutral, 2 ACES) and the
// exposure.
type resolvedTone struct {
	curve    float32
	exposure float32
}

func resolveTone(t ToneMapping, exposure foundation.Opt[float64]) resolvedTone {
	r := resolvedTone{curve: 1, exposure: float32(exposure.OrElse(1))}
	switch t {
	case ToneMappingACES:
		r.curve = 2
	case ToneMappingNone:
		r.curve = 0
	}
	return r
}

// hdrState is the float target that the PBR items are drawn into, and what the tone mapping reads it through.
type hdrState struct {
	// color is the target, multisampled like the view's, and resolve its single sample copy (nil when the view has one sample).
	color, resolve *gpu.Texture
	w, h, samples  int
	toneBuf        *gpu.Buffer
	toneGroup      *gpu.BindGroup
	// pipe owns the layout of toneGroup.
	pipe *gpu.RenderPipeline
}

func (h *hdrState) release(r *Renderer) {
	if h.toneGroup != nil {
		r.later(h.toneGroup)
	}
	for _, t := range []*gpu.Texture{h.color, h.resolve} {
		if t != nil {
			r.later(t)
		}
	}
	if h.toneBuf != nil {
		r.later(h.toneBuf)
	}
	*h = hdrState{}
}

// ensureHDR makes the float target of the size and the group that reads it for the tone mapping.
func (r *Renderer) ensureHDR(w, h, samples int, tonePipe *gpu.RenderPipeline) bool {
	s := &r.pbr.hdr
	if s.color != nil && s.w == w && s.h == h && s.samples == samples && s.pipe == tonePipe {
		return true
	}
	s.release(r)
	desc := gpu.TextureDesc{Label: "viewport3d hdr", Size: gpu.Extent3D{Width: w, Height: h, DepthOrArrayLayers: 1}, Format: gpu.TextureFormatRGBA16Float,
		SampleCount: samples, Usage: gpu.TextureUsageRenderAttachment}
	if samples == 1 {
		desc.Usage |= gpu.TextureUsageTextureBinding
	}
	var err error
	if s.color, err = r.dev.NewTexture(desc); err != nil {
		slog.Error("viewport3d: the float target could not be made", "err", err)
		s.release(r)
		return false
	}
	sampled := s.color
	if samples > 1 {
		desc.SampleCount, desc.Usage = 1, gpu.TextureUsageRenderAttachment|gpu.TextureUsageTextureBinding
		if s.resolve, err = r.dev.NewTexture(desc); err != nil {
			slog.Error("viewport3d: the float target could not be made", "err", err)
			s.release(r)
			return false
		}
		sampled = s.resolve
	}
	if s.toneBuf, err = r.dev.NewBuffer(gpu.BufferDesc{Label: "viewport3d tone mapping", Size: 16, Usage: gpu.BufferUsageUniform | gpu.BufferUsageCopyDst}); err != nil {
		slog.Error("viewport3d: the float target could not be made", "err", err)
		s.release(r)
		return false
	}
	s.toneGroup, err = r.dev.NewBindGroup(gpu.BindGroupDesc{Label: "viewport3d tone mapping", Layout: tonePipe.BindGroupLayout(0), Entries: []gpu.BindGroupEntry{
		{Binding: 0, Buffer: s.toneBuf, Size: 16}, {Binding: 1, TextureView: sampled.View()}}})
	if err != nil {
		slog.Error("viewport3d: the float target could not be made", "err", err)
		s.release(r)
		return false
	}
	s.w, s.h, s.samples, s.pipe = w, h, samples, tonePipe
	return true
}

// ensureToneModule makes the module of the tone mapping.
func (r *Renderer) ensureToneModule() error {
	if r.mods.tone != nil {
		return nil
	}
	var err error
	r.mods.tone, err = r.dev.NewShaderModule(shaders.Tonemap)
	return err
}
