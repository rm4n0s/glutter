// The cube of examples/gpu-triangle: one directional light and Blinn-Phong shading. The matrices come from the
// program (a column-major uniform), the vertices are a position and a normal.

struct Scene {
	mvp: mat4x4<f32>,
	model: mat4x4<f32>,
	// light is the direction towards the light, in world space; eye is the camera's position.
	light: vec4<f32>,
	eye: vec4<f32>,
	color: vec4<f32>,
}

@group(0) @binding(0) var<uniform> scene: Scene;

struct VertexIn {
	@location(0) position: vec3<f32>,
	@location(1) normal: vec3<f32>,
}

struct VertexOut {
	@builtin(position) clip: vec4<f32>,
	@location(0) world: vec3<f32>,
	@location(1) normal: vec3<f32>,
	@location(2) tint: vec3<f32>,
}

@vertex
fn vs_main(v: VertexIn) -> VertexOut {
	var out: VertexOut;
	out.clip = scene.mvp * vec4<f32>(v.position, 1.0);
	out.world = (scene.model * vec4<f32>(v.position, 1.0)).xyz;
	out.normal = (scene.model * vec4<f32>(v.normal, 0.0)).xyz;
	// Each face leans towards its own axis color, so that the faces are told apart.
	out.tint = abs(v.normal);
	return out;
}

// to_srgb encodes a linear color: an RGBA8Unorm target holds sRGB-encoded values.
fn to_srgb(c: vec3<f32>) -> vec3<f32> {
	let low = c * 12.92;
	let high = 1.055 * pow(c, vec3<f32>(1.0 / 2.4)) - vec3<f32>(0.055);
	return select(high, low, c <= vec3<f32>(0.0031308));
}

@fragment
fn fs_main(f: VertexOut) -> @location(0) vec4<f32> {
	let n = normalize(f.normal);
	let l = normalize(scene.light.xyz);
	let v = normalize(scene.eye.xyz - f.world);
	let h = normalize(l + v);
	let lit = max(dot(n, l), 0.0);
	let specular = pow(max(dot(n, h), 0.0), 64.0) * step(0.0001, lit);
	let base = mix(scene.color.rgb, f.tint, 0.5);
	let rgb = base * (0.10 + 0.90 * lit) + vec3<f32>(0.5 * specular);
	return vec4<f32>(to_srgb(rgb), 1.0);
}
