package viewport3d

import (
	"log/slog"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/viewport3d/shaders"
)

// Environment is the light that comes from all around a scene, for the image-based lighting of PBRMaterial (SPEC §28.5): a
// cube map of the sky. The renderer prefilters it with compute passes, once, into a cube of the specular reflections (one mip
// level for each roughness) and a cube of the diffuse irradiance. A scene with no Environment has the neutral studio that the
// renderer makes itself (a gradient, a large soft box and a fill light).
//
// Flutter: no counterpart
type Environment struct {
	cube *gpu.Texture
}

// NewEnvironment makes an environment of a cube map: a texture of the kind gpu.TextureKindCube with the usage
// gpu.TextureUsageTextureBinding, in the format RGBA16Float (linear radiance, which may be above 1) or RGBA8Unorm (sRGB colors).
// Mip levels help the prefiltering (a cube with one level is filtered less well). The environment does not own the texture and
// the texture must live as long as a renderer may build it. It panics with TagEnvironmentInvalid for a texture that is none of
// these.
//
// Flutter: no counterpart
func NewEnvironment(cube *gpu.Texture) *Environment {
	if cube == nil || cube.Kind() != gpu.TextureKindCube || cube.Usage()&gpu.TextureUsageTextureBinding == 0 ||
		(cube.Format() != gpu.TextureFormatRGBA16Float && cube.Format() != gpu.TextureFormatRGBA8Unorm) {
		panic(errors.New(TagEnvironmentInvalid, "an environment is a cube texture with TextureBinding, in RGBA16Float or RGBA8Unorm", "cube", cube != nil))
	}
	return &Environment{cube: cube}
}

// The sizes of the environment that the renderer makes.
const (
	skySize        = 128
	skyLevels      = 6
	specularSize   = 64
	specularLevels = 5
	irradianceSize = 16
	// specularSamples is the number of samples for each texel of a prefiltered level (the level of a mirror has one).
	specularSamples = 128
	irradianceSteps = 16
	// envParamsStride is the size of the record of the parameters of one dispatch in the uniform buffer.
	envParamsStride = uniformAlign
)

// envState is the environment that a renderer made for the scene: the cubes that the PBR shader samples.
type envState struct {
	// src is the environment they were made for; nil for the default one.
	src  *Environment
	done bool
	// sky is the cube of the default environment, which the prefiltering reads (the user's cube is read in place).
	sky  *gpu.Texture
	spec *gpu.Texture
	irr  *gpu.Texture
	samp *gpu.Sampler
}

// release lets go of the cubes, when the frame that encoded their use is done.
func (e *envState) release(r *Renderer) {
	for _, t := range []*gpu.Texture{e.sky, e.spec, e.irr} {
		if t != nil {
			r.later(t)
		}
	}
	if e.samp != nil {
		r.later(e.samp)
	}
	*e = envState{}
}

// envDispatch is one dispatch of the environment passes: the entry point, the size of the level it writes and the parameters.
type envDispatch struct {
	entry string
	size  int
	// rough, count and srcSize are the parameters of cs_specular and cs_irradiance, and sky the strength of cs_sky.
	rough, count, srcSize float32
	srgb, sky             float32
}

// ensureEnvironment makes the cubes of the environment (the first time, and when the scene's environment changes), with the
// compute passes in the encoder. It returns false when they could not be made.
func (r *Renderer) ensureEnvironment(enc *gpu.CommandEncoder, env *Environment) bool {
	e := &r.pbr.env
	if e.done && e.src == env {
		return true
	}
	if e.done || e.spec != nil {
		e.release(r)
	}
	if err := r.buildEnvironment(enc, env); err != nil {
		slog.Error("viewport3d: the environment could not be made", "err", err)
		e.release(r)
		return false
	}
	e.src, e.done = env, true
	return true
}

// buildEnvironment encodes the passes that make the sky (for the default environment), the prefiltered specular levels and the
// irradiance. Each level is written as an array texture of six layers, which is then copied into the level of the cube.
func (r *Renderer) buildEnvironment(enc *gpu.CommandEncoder, env *Environment) error {
	e := &r.pbr.env
	if r.mods.env == nil {
		var err error
		if r.mods.env, err = r.dev.NewShaderModule(shaders.Env); err != nil {
			return err
		}
	}
	var err error
	if e.samp, err = r.dev.NewSampler(gpu.SamplerDesc{Label: "viewport3d environment", MagFilter: gpu.FilterModeLinear, MinFilter: gpu.FilterModeLinear,
		MipmapFilter: gpu.FilterModeLinear}); err != nil {
		return err
	}
	// The passes read the source with one sample from the nearest texel of two mip levels: the level comes from the solid angle of
	// the sample, so the filter inside a level adds little, and Android's emulator (SwiftShader) gives wrong values for the linear
	// filter of a cube in a compute shader (it read texels that were never written).
	csamp, err := r.dev.NewSampler(gpu.SamplerDesc{Label: "viewport3d environment passes", MagFilter: gpu.FilterModeNearest, MinFilter: gpu.FilterModeNearest,
		MipmapFilter: gpu.FilterModeLinear})
	if err != nil {
		return err
	}
	r.later(csamp)
	cube := func(label string, size, levels int) (*gpu.Texture, error) {
		return r.dev.NewTexture(gpu.TextureDesc{Label: label, Kind: gpu.TextureKindCube, Size: gpu.Extent3D{Width: size, Height: size, DepthOrArrayLayers: 6},
			MipLevelCount: levels, Format: gpu.TextureFormatRGBA16Float, Usage: gpu.TextureUsageTextureBinding | gpu.TextureUsageCopySrc})
	}
	if e.spec, err = cube("viewport3d specular environment", specularSize, specularLevels); err != nil {
		return err
	}
	if e.irr, err = cube("viewport3d irradiance", irradianceSize, 1); err != nil {
		return err
	}
	source := (*gpu.Texture)(nil)
	srcSize, srgb := float32(skySize), float32(0)
	var jobs []envDispatch
	var skyJobs []envDispatch
	if env != nil {
		source = env.cube
		srcSize = float32(source.Size().Width)
		if source.Format() == gpu.TextureFormatRGBA8Unorm {
			srgb = 1
		}
	} else {
		if e.sky, err = cube("viewport3d sky", skySize, skyLevels); err != nil {
			return err
		}
		source = e.sky
		for l := 0; l < skyLevels; l++ {
			skyJobs = append(skyJobs, envDispatch{entry: "cs_sky", size: skySize >> l, sky: 1})
		}
	}
	for l := 0; l < specularLevels; l++ {
		count := float32(specularSamples)
		jobs = append(jobs, envDispatch{entry: "cs_specular", size: specularSize >> l, rough: float32(l) / float32(specularLevels-1), count: count, srcSize: srcSize, srgb: srgb})
	}
	jobs = append(jobs, envDispatch{entry: "cs_irradiance", size: irradianceSize, count: irradianceSteps, srcSize: srcSize, srgb: srgb})

	// One uniform buffer holds the parameters of every dispatch, each in its own aligned record.
	all := append(append([]envDispatch(nil), skyJobs...), jobs...)
	params := make([]byte, len(all)*envParamsStride)
	for i, j := range all {
		putVec4(params, i*envParamsStride, float32(j.size), j.rough, j.count, j.srcSize)
		putVec4(params, i*envParamsStride+16, j.srgb, j.sky, 0, 0)
	}
	pbuf, err := r.newBuffer("viewport3d environment parameters", params, gpu.BufferUsageUniform)
	if err != nil {
		return err
	}
	r.later(pbuf)

	pipes := map[string]*gpu.ComputePipeline{}
	pipeOf := func(entry string) (*gpu.ComputePipeline, error) {
		if p := pipes[entry]; p != nil {
			return p, nil
		}
		p, err := r.dev.NewComputePipeline(gpu.ComputePipelineDesc{Label: "viewport3d " + entry, Module: r.mods.env, EntryPoint: entry})
		if err != nil {
			return nil, err
		}
		pipes[entry] = p
		r.later(p)
		return p, nil
	}
	// The textures that the passes write: one array of six layers for each dispatch, copied into the cubes afterwards.
	temps := make([]*gpu.Texture, len(all))
	groups := make([]*gpu.BindGroup, len(all))
	for i, j := range all {
		if temps[i], err = r.dev.NewTexture(gpu.TextureDesc{Label: "viewport3d environment level", Kind: gpu.TextureKind2DArray,
			Size: gpu.Extent3D{Width: j.size, Height: j.size, DepthOrArrayLayers: 6}, Format: gpu.TextureFormatRGBA16Float,
			Usage: gpu.TextureUsageStorageBinding | gpu.TextureUsageCopySrc}); err != nil {
			return err
		}
		r.later(temps[i])
	}

	run := func(from, to int, label string) error {
		pass := enc.BeginComputePass(gpu.ComputePassDesc{Label: label})
		for i := from; i < to; i++ {
			j := all[i]
			pipe, err := pipeOf(j.entry)
			if err != nil {
				pass.End()
				return err
			}
			entries := []gpu.BindGroupEntry{{Binding: 0, Buffer: pbuf, Size: 32}, {Binding: 3, TextureView: temps[i].View()}}
			if j.entry != "cs_sky" {
				entries = append(entries, gpu.BindGroupEntry{Binding: 1, TextureView: source.View()}, gpu.BindGroupEntry{Binding: 2, Sampler: csamp})
			}
			if groups[i], err = r.dev.NewBindGroup(gpu.BindGroupDesc{Label: "viewport3d environment", Layout: pipe.BindGroupLayout(0), Entries: entries}); err != nil {
				pass.End()
				return err
			}
			r.later(groups[i])
			pass.SetPipeline(pipe)
			pass.SetBindGroup(0, groups[i], uint32(i*envParamsStride))
			n := (j.size + 7) / 8
			pass.DispatchWorkgroups(n, n, 6)
		}
		pass.End()
		return nil
	}
	if len(skyJobs) > 0 {
		if err := run(0, len(skyJobs), "viewport3d sky"); err != nil {
			return err
		}
		for l := range skyJobs {
			enc.CopyTextureToTexture(gpu.TextureCopy{Texture: temps[l]}, gpu.TextureCopy{Texture: e.sky, MipLevel: l},
				gpu.Extent3D{Width: skySize >> l, Height: skySize >> l, DepthOrArrayLayers: 6})
		}
	}
	if err := run(len(skyJobs), len(all), "viewport3d prefilter"); err != nil {
		return err
	}
	for l := 0; l < specularLevels; l++ {
		i := len(skyJobs) + l
		enc.CopyTextureToTexture(gpu.TextureCopy{Texture: temps[i]}, gpu.TextureCopy{Texture: e.spec, MipLevel: l},
			gpu.Extent3D{Width: specularSize >> l, Height: specularSize >> l, DepthOrArrayLayers: 6})
	}
	enc.CopyTextureToTexture(gpu.TextureCopy{Texture: temps[len(all)-1]}, gpu.TextureCopy{Texture: e.irr},
		gpu.Extent3D{Width: irradianceSize, Height: irradianceSize, DepthOrArrayLayers: 6})
	return nil
}

// TagEnvironmentInvalid is the tag of the panic of NewEnvironment for a texture that is not a cube of a format it reads.
//
// Flutter: no counterpart
const TagEnvironmentInvalid = "EnvironmentInvalid"
