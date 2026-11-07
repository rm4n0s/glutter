// The PBR metallic-roughness material of viewport3d (SPEC §28.5): the BRDF of the glTF 2.0 specification (appendix B: GGX
// distribution, the height-correlated Smith visibility, Schlick's Fresnel and a Lambert diffuse), the scene's directional
// lights, image-based lighting from the prefiltered environment (split sum, with the analytic fit of the BRDF's table by
// Karis), one directional shadow map, normal, occlusion and emissive maps, skinning and morph targets in the vertex shader.
//
// A mesh of this material has one vertex buffer of a fixed layout (position, normal, tangent, two texture coordinates, color,
// joints and weights; what the mesh lacks is zero, and the item's flags say what it has), so there is one vertex function. The
// shader writes linear radiance, premultiplied by the alpha: the renderer draws the frame into a float target and tonemap.wgsl
// makes the colors of the display (SPEC §28.5, "Decided in step 31.9").
//
// The positions are float32 relative to the mesh's origin; the instance's matrix takes them to the view space, whose origin is
// the eye (D30). Skin matrices are in the mesh's space (the renderer makes them in float64 relative to the item).

//#include "common.wgsl.inc"

struct TexSlot {
	// a is the first row of the texture coordinate transform (m00, m01, tx) and the set of coordinates (0 or 1) in w; b is the
	// second row (m10, m11, ty) and in w 1 when the material has the texture.
	a: vec4<f32>,
	b: vec4<f32>,
}

struct Material {
	// color is the base color factor, linear, with the alpha.
	color: vec4<f32>,
	// factors are the metallic factor, the roughness factor, the scale of the normal and the strength of the occlusion.
	factors: vec4<f32>,
	// emissive is the emitted radiance, linear.
	emissive: vec4<f32>,
	// flags: x the alpha mode (0 opaque, 1 mask, 2 blend), y the cutoff, z the bits 1 unlit and 2 vertex colors.
	flags: vec4<f32>,
	// slots are the textures: base color, metallic-roughness, normal, occlusion, emissive.
	slots: array<TexSlot, 5>,
}

@group(1) @binding(0) var<uniform> material: Material;
@group(1) @binding(1) var tex_base: texture_2d<f32>;
@group(1) @binding(2) var tex_mr: texture_2d<f32>;
@group(1) @binding(3) var tex_normal: texture_2d<f32>;
@group(1) @binding(4) var tex_occlusion: texture_2d<f32>;
@group(1) @binding(5) var tex_emissive: texture_2d<f32>;
@group(1) @binding(6) var samp_base: sampler;
@group(1) @binding(7) var samp_mr: sampler;
@group(1) @binding(8) var samp_normal: sampler;
@group(1) @binding(9) var samp_occlusion: sampler;
@group(1) @binding(10) var samp_emissive: sampler;

struct PBRFrame {
	// shadow_vp takes a position of the view space (the eye is the origin) to the clip space of the shadow map.
	shadow_vp: mat4x4<f32>,
	// view_to_world turns a direction of the view space into the world's.
	view_to_world: mat4x4<f32>,
	// params: x the strength of the environment's light, y the index in the frame's lights of the light that casts the shadow (-1:
	// none), z the depth
	// bias, w the offset along the normal, in the units of the view space.
	params: vec4<f32>,
	// shadow: x the strength of the shadow, y the size of the map in texels, z unused, w unused.
	shadow: vec4<f32>,
	// env: x the last mip level of the specular cube.
	env: vec4<f32>,
}

@group(2) @binding(0) var<uniform> pf: PBRFrame;
@group(2) @binding(1) var env_spec: texture_cube<f32>;
@group(2) @binding(2) var env_irr: texture_cube<f32>;
@group(2) @binding(3) var env_samp: sampler;
@group(2) @binding(4) var shadow_map: texture_2d<f32>;
@group(2) @binding(5) var<storage, read> joint_matrices: array<mat4x4<f32>>;
@group(2) @binding(6) var<storage, read> morph_weights: array<f32>;
// morph_deltas are, for each vertex and each target, the change of the position and of the normal (xyz of two vec4).
@group(3) @binding(0) var<storage, read> morph_deltas: array<vec4<f32>>;

// The bits of what a mesh has, in the instance's last number.
const HAS_NORMAL: u32 = 1u;
const HAS_TANGENT: u32 = 2u;
const HAS_UV0: u32 = 4u;
const HAS_UV1: u32 = 8u;
const SKINNED: u32 = 32u;

const PI: f32 = 3.14159265358979;

struct VOut {
	@builtin(position) clip: vec4<f32>,
	@location(0) view_pos: vec3<f32>,
	@location(1) normal: vec3<f32>,
	@location(2) tangent: vec4<f32>,
	@location(3) uv0: vec2<f32>,
	@location(4) uv1: vec2<f32>,
	@location(5) color: vec4<f32>,
	// tint is the color of a selected item, and in a the part of it that replaces the material's.
	@location(6) tint: vec4<f32>,
	@location(7) @interpolate(flat) flags: f32,
}

@vertex
fn vs_pbr(@builtin(vertex_index) vertex: u32,
	@location(0) p: vec3<f32>, @location(1) n: vec3<f32>, @location(2) t: vec4<f32>,
	@location(3) uv0: vec2<f32>, @location(4) uv1: vec2<f32>, @location(5) c: vec4<f32>,
	@location(6) joints: vec4<u32>, @location(7) weights: vec4<f32>,
	@location(8) m0: vec4<f32>, @location(9) m1: vec4<f32>, @location(10) m2: vec4<f32>, @location(11) m3: vec4<f32>,
	@location(12) tint: vec4<f32>, @location(13) extra: vec4<f32>) -> VOut {
	let flags = u32(extra.w);
	var pos = p;
	var nrm = n;
	var tan = t.xyz;

	// Morph targets: the weights of the item, and the changes of this vertex.
	let targets = u32(extra.z);
	if (targets > 0u) {
		let wbase = u32(extra.y);
		for (var k = 0u; k < targets; k = k + 1u) {
			let w = morph_weights[wbase + k];
			if (w != 0.0) {
				let i = (vertex * targets + k) * 2u;
				pos = pos + w * morph_deltas[i].xyz;
				nrm = nrm + w * morph_deltas[i + 1u].xyz;
			}
		}
	}

	// Skinning: the four joints' matrices, blended by the weights.
	if ((flags & SKINNED) != 0u) {
		let jb = u32(extra.x);
		let skin = joint_matrices[jb + joints.x] * weights.x + joint_matrices[jb + joints.y] * weights.y +
			joint_matrices[jb + joints.z] * weights.z + joint_matrices[jb + joints.w] * weights.w;
		pos = (skin * vec4<f32>(pos, 1.0)).xyz;
		nrm = (skin * vec4<f32>(nrm, 0.0)).xyz;
		tan = (skin * vec4<f32>(tan, 0.0)).xyz;
	}

	let mv = mat4x4<f32>(m0, m1, m2, m3);
	let vp = mv * vec4<f32>(pos, 1.0);
	var o: VOut;
	o.view_pos = vp.xyz;
	o.clip = frame.proj * vp;
	o.normal = (mv * vec4<f32>(nrm, 0.0)).xyz;
	o.tangent = vec4<f32>((mv * vec4<f32>(tan, 0.0)).xyz, t.w);
	o.uv0 = uv0;
	o.uv1 = uv1;
	o.color = c;
	o.tint = tint;
	o.flags = extra.w;
	return o;
}

// srgb_to_linear decodes the sRGB transfer function.
fn srgb_to_linear(c: vec3<f32>) -> vec3<f32> {
	let lo = c / 12.92;
	let hi = pow((c + vec3<f32>(0.055)) / 1.055, vec3<f32>(2.4));
	return select(hi, lo, c <= vec3<f32>(0.04045));
}

// slot_uv is the texture coordinates of a slot of the material: the set it reads, then its transform.
fn slot_uv(i: i32, uv0: vec2<f32>, uv1: vec2<f32>) -> vec2<f32> {
	let s = material.slots[i];
	var uv = uv0;
	if (s.a.w > 0.5) {
		uv = uv1;
	}
	return vec2<f32>(s.a.x * uv.x + s.a.y * uv.y + s.a.z, s.b.x * uv.x + s.b.y * uv.y + s.b.z);
}

// slot_has is 1 when the material has the texture of the slot.
fn slot_has(i: i32) -> bool {
	return material.slots[i].b.w > 0.5;
}

fn distribution_ggx(nh: f32, alpha: f32) -> f32 {
	let a2 = alpha * alpha;
	let d = nh * nh * (a2 - 1.0) + 1.0;
	return a2 / (PI * d * d);
}

fn visibility_smith_ggx(nl: f32, nv: f32, alpha: f32) -> f32 {
	let a2 = alpha * alpha;
	let gv = nl * sqrt(nv * nv * (1.0 - a2) + a2);
	let gl = nv * sqrt(nl * nl * (1.0 - a2) + a2);
	let s = gv + gl;
	if (s > 0.0) {
		return 0.5 / s;
	}
	return 0.0;
}

// shadow_visibility is the part of the light that reaches a position of the view space: 1 where the shadow map has no shadow.
fn shadow_visibility(view_pos: vec3<f32>, n: vec3<f32>) -> f32 {
	let p = view_pos + n * pf.params.w;
	let c = pf.shadow_vp * vec4<f32>(p, 1.0);
	let size = pf.shadow.y;
	let uv = vec2<f32>(c.x * 0.5 + 0.5, 0.5 - c.y * 0.5);
	let coord = uv * size;
	let depth = c.z - pf.params.z;
	if (uv.x < 0.0 || uv.y < 0.0 || uv.x >= 1.0 || uv.y >= 1.0 || c.z > 1.0) {
		return 1.0;
	}
	// A 3 by 3 filter of comparisons, so that the edge of a shadow is not made of texels.
	let base = vec2<i32>(floor(coord));
	let top = i32(size) - 1;
	var lit = 0.0;
	for (var dy = -1; dy <= 1; dy = dy + 1) {
		for (var dx = -1; dx <= 1; dx = dx + 1) {
			let q = clamp(base + vec2<i32>(dx, dy), vec2<i32>(0), vec2<i32>(top));
			let stored = textureLoad(shadow_map, q, 0).x;
			if (depth <= stored) {
				lit = lit + 1.0;
			}
		}
	}
	return mix(1.0, lit / 9.0, pf.shadow.x);
}

@fragment
fn fs_pbr(in: VOut, @builtin(front_facing) front: bool) -> @location(0) vec4<f32> {
	let flags = u32(in.flags);

	// Derivatives and sampling are taken where the control flow is uniform, then selected.
	let dp_dx = dpdx(in.view_pos);
	let dp_dy = dpdy(in.view_pos);
	let uv_base = slot_uv(0, in.uv0, in.uv1);
	let uv_mr = slot_uv(1, in.uv0, in.uv1);
	let uv_normal = slot_uv(2, in.uv0, in.uv1);
	let uv_occlusion = slot_uv(3, in.uv0, in.uv1);
	let uv_emissive = slot_uv(4, in.uv0, in.uv1);
	let duv_dx = dpdx(uv_normal);
	let duv_dy = dpdy(uv_normal);
	let t_base = textureSample(tex_base, samp_base, uv_base);
	let t_mr = textureSample(tex_mr, samp_mr, uv_mr);
	let t_normal = textureSample(tex_normal, samp_normal, uv_normal);
	let t_occlusion = textureSample(tex_occlusion, samp_occlusion, uv_occlusion);
	let t_emissive = textureSample(tex_emissive, samp_emissive, uv_emissive);
	if (section_clipped(in.view_pos, -1)) {
		discard;
	}

	var base = material.color;
	let bits = u32(material.flags.z);
	if ((bits & 2u) != 0u) {
		base = base * vec4<f32>(srgb_to_linear(in.color.rgb), in.color.a);
	}
	if (slot_has(0)) {
		base = base * vec4<f32>(srgb_to_linear(t_base.rgb), t_base.a);
	}
	let mode = u32(material.flags.x);
	var alpha = base.a;
	if (mode == 0u) {
		alpha = 1.0;
	} else if (mode == 1u && alpha < material.flags.y) {
		discard;
	}
	let tint = vec4<f32>(srgb_to_linear(in.tint.rgb), in.tint.a);

	// An unlit material shows its color.
	if ((bits & 1u) != 0u) {
		let c = mix(base.rgb, tint.rgb, tint.a);
		return vec4<f32>(c * alpha, alpha);
	}

	var metallic = material.factors.x;
	var roughness = material.factors.y;
	if (slot_has(1)) {
		roughness = roughness * t_mr.g;
		metallic = metallic * t_mr.b;
	}
	metallic = clamp(metallic, 0.0, 1.0);
	roughness = clamp(roughness, 0.045, 1.0);
	var occlusion = 1.0;
	if (slot_has(3)) {
		occlusion = 1.0 + material.factors.w * (t_occlusion.r - 1.0);
	}
	var emissive = material.emissive.rgb;
	if (slot_has(4)) {
		emissive = emissive * srgb_to_linear(t_emissive.rgb);
	}

	// The shading frame. A mesh with no normals is flat, shaded as seen from the camera; a back face of a double sided material is
	// lit as seen from behind.
	let v = normalize(-in.view_pos);
	var n = in.normal;
	// sign is -1 for the back face of a double sided material, which is shaded as seen from behind: the frame turns with it.
	var sign = 1.0;
	var bitangent_geom = vec3<f32>(0.0);
	if ((flags & HAS_NORMAL) == 0u) {
		n = normalize(cross(dp_dx, dp_dy));
		if (dot(n, v) < 0.0) {
			n = -n;
		}
	} else {
		n = normalize(n);
		bitangent_geom = cross(n, normalize(in.tangent.xyz)) * in.tangent.w;
		if (!front) {
			n = -n;
			sign = -1.0;
		}
	}
	if (slot_has(2)) {
		var tangent: vec3<f32>;
		var bitangent: vec3<f32>;
		if ((flags & HAS_TANGENT) != 0u) {
			tangent = normalize(in.tangent.xyz) * sign;
			bitangent = bitangent_geom * sign;
		} else {
			// The frame from the derivatives of the position and of the texture coordinates: the tangent points the way that u grows
			// on the surface (solving dp = T du + B dv for T), made orthogonal to the normal; the bitangent is the cross product of the
			// normal and the tangent, as the reference viewer of glTF has it (the green channel of a normal map points up in the
			// image, towards smaller v).
			let det = duv_dx.x * duv_dy.y - duv_dy.x * duv_dx.y;
			var t = (dp_dx * duv_dy.y - dp_dy * duv_dx.y) * select(1.0, -1.0, det < 0.0);
			t = t - n * dot(n, t);
			tangent = t * inverseSqrt(max(dot(t, t), 1.0e-20));
			bitangent = cross(n, tangent);
		}
		var tn = t_normal.xyz * 2.0 - vec3<f32>(1.0);
		tn = vec3<f32>(tn.xy * material.factors.z, tn.z);
		n = normalize(tangent * tn.x + bitangent * tn.y + n * tn.z);
	}

	let nv = clamp(abs(dot(n, v)), 1.0e-4, 1.0);
	let f0 = mix(vec3<f32>(0.04), base.rgb, metallic);
	let diffuse_color = base.rgb * (1.0 - metallic);
	let alpha_r = roughness * roughness;

	// The lights: the one at the camera, then the scene's directional lights. A light of intensity 1 gives a white surface that
	// faces it its color, as in the Blinn-Phong material, so the BRDF is scaled by pi.
	var color = vec3<f32>(0.0);
	for (var i = 0; i < 5; i = i + 1) {
		let l = normalize(frame.lights[i].dir.xyz);
		let nl = dot(n, l);
		if (nl > 0.0) {
			let h = normalize(l + v);
			let nh = clamp(dot(n, h), 0.0, 1.0);
			let vh = clamp(dot(v, h), 0.0, 1.0);
			let fresnel = f0 + (vec3<f32>(1.0) - f0) * pow(1.0 - vh, 5.0);
			let spec = fresnel * distribution_ggx(nh, alpha_r) * visibility_smith_ggx(nl, nv, alpha_r);
			let diff = (vec3<f32>(1.0) - fresnel) * diffuse_color / PI;
			var visible = 1.0;
			if (i == i32(pf.params.y)) {
				visible = shadow_visibility(in.view_pos, n);
			}
			color = color + (diff + spec) * frame.lights[i].color.rgb * nl * PI * visible;
		}
	}

	// Image-based lighting: the irradiance for the diffuse part and the prefiltered environment for the specular part, with the
	// fit of the split-sum table by Karis (Mobile Graphics, SIGGRAPH 2014).
	let nw = (pf.view_to_world * vec4<f32>(n, 0.0)).xyz;
	let rw = (pf.view_to_world * vec4<f32>(reflect(-v, n), 0.0)).xyz;
	let irradiance = textureSampleLevel(env_irr, env_samp, nw, 0.0).rgb;
	let prefiltered = textureSampleLevel(env_spec, env_samp, rw, roughness * pf.env.x).rgb;
	let c0 = vec4<f32>(-1.0, -0.0275, -0.572, 0.022);
	let c1 = vec4<f32>(1.0, 0.0425, 1.04, -0.04);
	let r = roughness * c0 + c1;
	let a004 = min(r.x * r.x, exp2(-9.28 * nv)) * r.x + r.y;
	let ab = vec2<f32>(-1.04, 1.04) * a004 + r.zw;
	let ibl = irradiance * diffuse_color + prefiltered * (f0 * ab.x + vec3<f32>(ab.y));
	color = color + ibl * occlusion * pf.params.x + emissive;

	color = mix(color, tint.rgb, tint.a);
	return vec4<f32>(color * alpha, alpha);
}
