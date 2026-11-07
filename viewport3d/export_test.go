package viewport3d

import "github.com/rm4n0s/glutter/gpu"

// EnvironmentCubes gives the cubes that the renderer made for its environment (the sky, the prefiltered levels and the irradiance).
func (r *Renderer) EnvironmentCubes() (sky, spec, irr *gpu.Texture) {
	return r.pbr.env.sky, r.pbr.env.spec, r.pbr.env.irr
}
