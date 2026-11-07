// The shadow map of viewport3d (SPEC §28.5): the items seen from the light, as depth. The matrix of an instance takes the item's
// positions straight to the clip space of the light (an orthographic projection fitted to the scene), so the shader needs no
// frame data. The depth is written as a color of a float target (the depth formats cannot be sampled, SPEC §27.2) and read with
// textureLoad by pbr.wgsl. The skin and the morph targets of an item are applied as in the PBR vertex shader.

@group(0) @binding(0) var<storage, read> joint_matrices: array<mat4x4<f32>>;
@group(0) @binding(1) var<storage, read> morph_weights: array<f32>;
@group(1) @binding(0) var<storage, read> morph_deltas: array<vec4<f32>>;

struct VOut {
	@builtin(position) clip: vec4<f32>,
	@location(0) depth: f32,
}

const SKINNED: u32 = 32u;

// vs_shadow_simple is for the meshes of the other materials: positions only.
@vertex
fn vs_shadow_simple(@location(0) p: vec3<f32>,
	@location(8) m0: vec4<f32>, @location(9) m1: vec4<f32>, @location(10) m2: vec4<f32>, @location(11) m3: vec4<f32>) -> VOut {
	var o: VOut;
	o.clip = mat4x4<f32>(m0, m1, m2, m3) * vec4<f32>(p, 1.0);
	o.depth = o.clip.z;
	return o;
}

// vs_shadow_full is for the meshes of the PBR material, with the vertex layout of pbr.wgsl.
@vertex
fn vs_shadow_full(@builtin(vertex_index) vertex: u32, @location(0) p: vec3<f32>, @location(6) joints: vec4<u32>, @location(7) weights: vec4<f32>,
	@location(8) m0: vec4<f32>, @location(9) m1: vec4<f32>, @location(10) m2: vec4<f32>, @location(11) m3: vec4<f32>,
	@location(13) extra: vec4<f32>) -> VOut {
	var pos = p;
	let targets = u32(extra.z);
	if (targets > 0u) {
		let wbase = u32(extra.y);
		for (var k = 0u; k < targets; k = k + 1u) {
			let w = morph_weights[wbase + k];
			if (w != 0.0) {
				pos = pos + w * morph_deltas[(vertex * targets + k) * 2u].xyz;
			}
		}
	}
	if ((u32(extra.w) & SKINNED) != 0u) {
		let jb = u32(extra.x);
		let skin = joint_matrices[jb + joints.x] * weights.x + joint_matrices[jb + joints.y] * weights.y +
			joint_matrices[jb + joints.z] * weights.z + joint_matrices[jb + joints.w] * weights.w;
		pos = (skin * vec4<f32>(pos, 1.0)).xyz;
	}
	var o: VOut;
	o.clip = mat4x4<f32>(m0, m1, m2, m3) * vec4<f32>(pos, 1.0);
	o.depth = o.clip.z;
	return o;
}

@fragment
fn fs_shadow(in: VOut) -> @location(0) f32 {
	return in.depth;
}
