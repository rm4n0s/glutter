// The meshes of triangles of viewport3d (SPEC §27.8): unlit and Blinn-Phong shading. The vertex entry points differ in the
// attributes they read (p position, n normal, u texture coordinates, c color); the renderer picks the one that a mesh has.
// The positions are float32 relative to the mesh's origin and the model-view matrix of each instance takes them to the view,
// whose origin is the eye (D30): the numbers are small near the camera.

struct Light {
	// dir points to the light, in view space.
	dir: vec4<f32>,
	// color is the light's color times its intensity.
	color: vec4<f32>,
}

struct Frame {
	proj: mat4x4<f32>,
	// lights are the light at the camera, then up to four directional ones; an unused one has a black color.
	lights: array<Light, 5>,
	ambient: vec4<f32>,
	// viewport is the width and the height of the target in pixels, and their inverses.
	viewport: vec4<f32>,
	// params are the near plane's distance, the device pixel ratio and two unused numbers.
	params: vec4<f32>,
}

struct Material {
	color: vec4<f32>,
	// specular is the highlight's color, and its exponent in w.
	specular: vec4<f32>,
	// flags are: x 1 when the material is lit, y 1 for vertex colors, z 1 for a texture, w 1 when the output is opaque.
	flags: vec4<f32>,
}

@group(0) @binding(0) var<uniform> frame: Frame;
@group(1) @binding(0) var<uniform> material: Material;
@group(1) @binding(1) var tex: texture_2d<f32>;
@group(1) @binding(2) var samp: sampler;

struct VOut {
	@builtin(position) clip: vec4<f32>,
	@location(0) view_pos: vec3<f32>,
	@location(1) normal: vec3<f32>,
	@location(2) uv: vec2<f32>,
	@location(3) color: vec4<f32>,
}

fn vertex_common(p: vec3<f32>, n: vec3<f32>, uv: vec2<f32>, c: vec4<f32>, m0: vec4<f32>, m1: vec4<f32>, m2: vec4<f32>, m3: vec4<f32>) -> VOut {
	let mv = mat4x4<f32>(m0, m1, m2, m3);
	let vp = mv * vec4<f32>(p, 1.0);
	var o: VOut;
	o.view_pos = vp.xyz;
	o.clip = frame.proj * vp;
	o.normal = (mv * vec4<f32>(n, 0.0)).xyz;
	o.uv = uv;
	o.color = c;
	return o;
}

@vertex
fn vs_p(@location(0) p: vec3<f32>,
	@location(4) m0: vec4<f32>, @location(5) m1: vec4<f32>, @location(6) m2: vec4<f32>, @location(7) m3: vec4<f32>) -> VOut {
	return vertex_common(p, vec3<f32>(0.0), vec2<f32>(0.0), vec4<f32>(1.0), m0, m1, m2, m3);
}

@vertex
fn vs_pn(@location(0) p: vec3<f32>, @location(1) n: vec3<f32>,
	@location(4) m0: vec4<f32>, @location(5) m1: vec4<f32>, @location(6) m2: vec4<f32>, @location(7) m3: vec4<f32>) -> VOut {
	return vertex_common(p, n, vec2<f32>(0.0), vec4<f32>(1.0), m0, m1, m2, m3);
}

@vertex
fn vs_pu(@location(0) p: vec3<f32>, @location(2) uv: vec2<f32>,
	@location(4) m0: vec4<f32>, @location(5) m1: vec4<f32>, @location(6) m2: vec4<f32>, @location(7) m3: vec4<f32>) -> VOut {
	return vertex_common(p, vec3<f32>(0.0), uv, vec4<f32>(1.0), m0, m1, m2, m3);
}

@vertex
fn vs_pc(@location(0) p: vec3<f32>, @location(3) c: vec4<f32>,
	@location(4) m0: vec4<f32>, @location(5) m1: vec4<f32>, @location(6) m2: vec4<f32>, @location(7) m3: vec4<f32>) -> VOut {
	return vertex_common(p, vec3<f32>(0.0), vec2<f32>(0.0), c, m0, m1, m2, m3);
}

@vertex
fn vs_pnu(@location(0) p: vec3<f32>, @location(1) n: vec3<f32>, @location(2) uv: vec2<f32>,
	@location(4) m0: vec4<f32>, @location(5) m1: vec4<f32>, @location(6) m2: vec4<f32>, @location(7) m3: vec4<f32>) -> VOut {
	return vertex_common(p, n, uv, vec4<f32>(1.0), m0, m1, m2, m3);
}

@vertex
fn vs_pnc(@location(0) p: vec3<f32>, @location(1) n: vec3<f32>, @location(3) c: vec4<f32>,
	@location(4) m0: vec4<f32>, @location(5) m1: vec4<f32>, @location(6) m2: vec4<f32>, @location(7) m3: vec4<f32>) -> VOut {
	return vertex_common(p, n, vec2<f32>(0.0), c, m0, m1, m2, m3);
}

@vertex
fn vs_puc(@location(0) p: vec3<f32>, @location(2) uv: vec2<f32>, @location(3) c: vec4<f32>,
	@location(4) m0: vec4<f32>, @location(5) m1: vec4<f32>, @location(6) m2: vec4<f32>, @location(7) m3: vec4<f32>) -> VOut {
	return vertex_common(p, vec3<f32>(0.0), uv, c, m0, m1, m2, m3);
}

@vertex
fn vs_pnuc(@location(0) p: vec3<f32>, @location(1) n: vec3<f32>, @location(2) uv: vec2<f32>, @location(3) c: vec4<f32>,
	@location(4) m0: vec4<f32>, @location(5) m1: vec4<f32>, @location(6) m2: vec4<f32>, @location(7) m3: vec4<f32>) -> VOut {
	return vertex_common(p, n, uv, c, m0, m1, m2, m3);
}

@fragment
fn fs_main(in: VOut, @builtin(front_facing) front: bool) -> @location(0) vec4<f32> {
	// Derivatives and sampling are taken where the control flow is uniform, then selected.
	let flat_normal = cross(dpdx(in.view_pos), dpdy(in.view_pos));
	let texel = textureSample(tex, samp, in.uv);

	var base = material.color;
	if (material.flags.y > 0.5) {
		base = base * in.color;
	}
	if (material.flags.z > 0.5) {
		base = base * texel;
	}
	var rgb = base.rgb;
	if (material.flags.x > 0.5) {
		let v = normalize(-in.view_pos);
		var n = in.normal;
		if (dot(n, n) < 0.25) {
			// No normals: the face is flat, and shaded as seen from the camera.
			n = normalize(flat_normal);
			if (dot(n, v) < 0.0) {
				n = -n;
			}
		} else {
			n = normalize(n);
			if (!front) {
				n = -n;
			}
		}
		var light = frame.ambient.rgb;
		var spec = vec3<f32>(0.0);
		for (var i = 0; i < 5; i = i + 1) {
			let l = normalize(frame.lights[i].dir.xyz);
			let nl = max(dot(n, l), 0.0);
			light = light + frame.lights[i].color.rgb * nl;
			if (nl > 0.0) {
				let h = normalize(l + v);
				spec = spec + frame.lights[i].color.rgb * pow(max(dot(n, h), 0.0), material.specular.w);
			}
		}
		rgb = rgb * light + material.specular.rgb * spec;
	}
	var a = base.a;
	if (material.flags.w > 0.5) {
		a = 1.0;
	}
	// Premultiplied alpha, for the blending of transparent items.
	return vec4<f32>(min(rgb, vec3<f32>(1.0)) * a, a);
}
