package gpu

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

// Sampler says how a shader reads between the texels of a texture (WebGPU's GPUSampler).
//
// Flutter: no counterpart
type Sampler struct {
	resource
	exp explicit.Sampler
}

// NewSampler makes a sampler (WebGPU's GPUDevice.createSampler). The zero value of a filter is nearest and of an
// address mode is clamp to edge; no mip filter reads level 0 only. It fails with GPUInvalidDescriptor for a value
// that is not one of the constants, with GPUDeviceLost on a lost device, and with the error of the backend
// untouched.
//
// Flutter: no counterpart
func (d *Device) NewSampler(desc SamplerDesc) (*Sampler, error) {
	d.assertThread("NewSampler")
	if d.lostErr != nil {
		return nil, d.lostErr
	}
	for _, f := range []struct {
		field string
		v     FilterMode
	}{{"MagFilter", desc.MagFilter}, {"MinFilter", desc.MinFilter}, {"MipmapFilter", desc.MipmapFilter}} {
		if f.v < FilterModeUnspecified || f.v > FilterModeLinear {
			return nil, errors.New(TagGPUInvalidDescriptor, "a filter is one of the FilterMode values", "field", f.field, "value", int(f.v), "label", desc.Label)
		}
	}
	for _, a := range []struct {
		field string
		v     AddressMode
	}{{"AddressModeU", desc.AddressModeU}, {"AddressModeV", desc.AddressModeV}, {"AddressModeW", desc.AddressModeW}} {
		if a.v < AddressModeUnspecified || a.v > AddressModeMirrorRepeat {
			return nil, errors.New(TagGPUInvalidDescriptor, "an address mode is one of the AddressMode values", "field", a.field, "value", int(a.v), "label", desc.Label)
		}
	}
	exp, err := d.dev.NewSampler(explicit.SamplerDesc{
		MinFilter: explicit.FilterMode(desc.MinFilter), MagFilter: explicit.FilterMode(desc.MagFilter), MipmapFilter: explicit.FilterMode(desc.MipmapFilter),
		AddressU: explicit.AddressMode(desc.AddressModeU), AddressV: explicit.AddressMode(desc.AddressModeV), AddressW: explicit.AddressMode(desc.AddressModeW),
	})
	if err != nil {
		return nil, err
	}
	return &Sampler{resource: d.track(kindSampler, desc.Label, 1), exp: exp}, nil
}

// Release frees the sampler. It may be called twice.
//
// Flutter: no counterpart
func (s *Sampler) Release() {
	s.d.assertThread("Sampler.Release")
	if !s.untrack() {
		return
	}
	if s.d.lostErr == nil {
		s.exp.Destroy()
	}
}
