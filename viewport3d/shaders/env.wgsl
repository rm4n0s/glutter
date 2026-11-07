// The environment of viewport3d's image-based lighting (SPEC §28.5), made by compute passes: the neutral default sky, the
// prefiltered specular levels (GGX, importance sampled; one level for each roughness) and the diffuse irradiance. Each entry
// point writes one level of a cube, as a texture array of six layers (a storage binding cannot take a cube), which the renderer
// copies into the cube that the material samples.
//
// A texel of a face is a direction by the table of faces of OpenGL, Vulkan, Direct3D and WebGPU, which the hardware uses to
// sample a cube: +x, -x, +y, -y, +z, -z are the layers 0 to 5.

struct Params {
	// size: x the size of the level that is written, y the roughness, z the number of samples, w the size of the source's first level.
	size: vec4<f32>,
	// options: x 1 when the source is 8 bit sRGB, y the strength of the sky.
	options: vec4<f32>,
}

@group(0) @binding(0) var<uniform> params: Params;
@group(0) @binding(1) var src: texture_cube<f32>;
@group(0) @binding(2) var samp: sampler;
@group(0) @binding(3) var dst: texture_storage_2d_array<rgba16float, write>;

const PI: f32 = 3.14159265358979;

// face_direction is the direction of the texel (x, y) of a face of a cube of the given size. (The texel comes as i32: Wine's
// d3dcompiler_47 turns f32 of a component of the thread's id, a u32, into a wrong value, and f32 of an i32 into the right one.)
fn face_direction(face: u32, x: i32, y: i32, size: f32) -> vec3<f32> {
	let u = 2.0 * (f32(x) + 0.5) / size - 1.0;
	let v = 2.0 * (f32(y) + 0.5) / size - 1.0;
	var d = vec3<f32>(0.0);
	switch face {
		case 0u: { d = vec3<f32>(1.0, -v, -u); }
		case 1u: { d = vec3<f32>(-1.0, -v, u); }
		case 2u: { d = vec3<f32>(u, 1.0, v); }
		case 3u: { d = vec3<f32>(u, -1.0, -v); }
		case 4u: { d = vec3<f32>(u, -v, 1.0); }
		default: { d = vec3<f32>(-u, -v, -1.0); }
	}
	return normalize(d);
}

fn srgb_to_linear(c: vec3<f32>) -> vec3<f32> {
	let lo = c / 12.92;
	let hi = pow((c + vec3<f32>(0.055)) / 1.055, vec3<f32>(2.4));
	return select(hi, lo, c <= vec3<f32>(0.04045));
}

fn source(d: vec3<f32>, lod: f32) -> vec3<f32> {
	var c = textureSampleLevel(src, samp, d, lod).rgb;
	if (params.options.x > 0.5) {
		c = srgb_to_linear(c);
	}
	return c;
}

// radical_inverse is the Van der Corput sequence of i, and hammersley the point i of n of the Hammersley set.
fn hammersley(i: u32, n: u32) -> vec2<f32> {
	var bits = i;
	bits = (bits << 16u) | (bits >> 16u);
	bits = ((bits & 0x55555555u) << 1u) | ((bits & 0xAAAAAAAAu) >> 1u);
	bits = ((bits & 0x33333333u) << 2u) | ((bits & 0xCCCCCCCCu) >> 2u);
	bits = ((bits & 0x0F0F0F0Fu) << 4u) | ((bits & 0xF0F0F0F0u) >> 4u);
	bits = ((bits & 0x00FF00FFu) << 8u) | ((bits & 0xFF00FF00u) >> 8u);
	return vec2<f32>(f32(i) / f32(n), f32(bits) * 2.3283064365386963e-10);
}

// The sky is a neutral studio: a gradient from a dark floor through a bright horizon to a pale zenith, a large soft box that
// is the key light, and a weaker fill from the other side. It is in linear units (the box is far above 1: the specular
// highlights of a smooth surface show it).
fn sky(d: vec3<f32>) -> vec3<f32> {
	let up = d.y;
	let floor_color = vec3<f32>(0.10, 0.10, 0.11);
	let horizon = vec3<f32>(0.62, 0.64, 0.67);
	let zenith = vec3<f32>(0.75, 0.80, 0.90);
	var c = mix(horizon, zenith, smoothstep(0.0, 1.0, up));
	// (smoothstep needs its first edge below its second one: the result is not defined otherwise, and Direct3D gives NaNs.)
	c = mix(c, floor_color, 1.0 - smoothstep(-0.45, 0.0, up));
	let key_dir = normalize(vec3<f32>(0.55, 0.75, 0.45));
	let key = pow(max(dot(d, key_dir), 0.0), 24.0);
	let fill_dir = normalize(vec3<f32>(-0.7, 0.35, -0.5));
	let fill = pow(max(dot(d, fill_dir), 0.0), 10.0);
	return (c + vec3<f32>(1.0, 0.97, 0.92) * key * 6.0 + vec3<f32>(0.75, 0.85, 1.0) * fill * 0.8) * params.options.y;
}

// cs_sky writes the default sky.
@compute @workgroup_size(8, 8, 1)
fn cs_sky(@builtin(global_invocation_id) id: vec3<u32>) {
	let size = u32(params.size.x);
	if (id.x >= size || id.y >= size) {
		return;
	}
	let d = face_direction(id.z, i32(id.x), i32(id.y), params.size.x);
	textureStore(dst, vec2<i32>(i32(id.x), i32(id.y)), i32(id.z), vec4<f32>(sky(d), 1.0));
}

// cs_specular prefilters the source for the roughness of the params: GGX importance sampling with the view and the normal
// both along the direction of the texel, and the level of the source taken from the solid angle of a sample (Krivanek and
// Colbert, "Real-time Shading with Filtered Importance Sampling").
@compute @workgroup_size(8, 8, 1)
fn cs_specular(@builtin(global_invocation_id) id: vec3<u32>) {
	let size = u32(params.size.x);
	if (id.x >= size || id.y >= size) {
		return;
	}
	let n = face_direction(id.z, i32(id.x), i32(id.y), params.size.x);
	let roughness = params.size.y;
	let count = u32(params.size.z);
	let a = roughness * roughness;
	let a2 = a * a;
	var up = vec3<f32>(0.0, 1.0, 0.0);
	if (abs(n.y) > 0.999) {
		up = vec3<f32>(1.0, 0.0, 0.0);
	}
	let tx = normalize(cross(up, n));
	let ty = cross(n, tx);
	let texel_solid_angle = 4.0 * PI / (6.0 * params.size.w * params.size.w);
	if (a2 < 1.0e-6) {
		// A mirror: the source as it is.
		textureStore(dst, vec2<i32>(i32(id.x), i32(id.y)), i32(id.z), vec4<f32>(source(n, 0.0), 1.0));
		return;
	}
	var color = vec3<f32>(0.0);
	var weight = 0.0;
	for (var i = 0u; i < count; i = i + 1u) {
		let xi = hammersley(i, count);
		let phi = 2.0 * PI * xi.x;
		let cos_t = sqrt((1.0 - xi.y) / (1.0 + (a2 - 1.0) * xi.y));
		let sin_t = sqrt(max(1.0 - cos_t * cos_t, 0.0));
		let h = tx * (sin_t * cos(phi)) + ty * (sin_t * sin(phi)) + n * cos_t;
		let l = 2.0 * dot(n, h) * h - n;
		let nl = dot(n, l);
		if (nl > 0.0) {
			let nh = max(dot(n, h), 0.0);
			let d = nh * nh * (a2 - 1.0) + 1.0;
			let pdf = a2 / (PI * d * d) / 4.0;
			let sample_solid_angle = 1.0 / (f32(count) * pdf + 1.0e-4);
			let lod = max(0.5 * log2(sample_solid_angle / texel_solid_angle) + 1.0, 0.0);
			color = color + source(l, lod) * nl;
			weight = weight + nl;
		}
	}
	textureStore(dst, vec2<i32>(i32(id.x), i32(id.y)), i32(id.z), vec4<f32>(color / max(weight, 1.0e-4), 1.0));
}

// cs_irradiance convolves the source with the cosine over the hemisphere of each texel: the irradiance divided by pi.
@compute @workgroup_size(8, 8, 1)
fn cs_irradiance(@builtin(global_invocation_id) id: vec3<u32>) {
	let size = u32(params.size.x);
	if (id.x >= size || id.y >= size) {
		return;
	}
	let n = face_direction(id.z, i32(id.x), i32(id.y), params.size.x);
	var up = vec3<f32>(0.0, 1.0, 0.0);
	if (abs(n.y) > 0.999) {
		up = vec3<f32>(1.0, 0.0, 0.0);
	}
	let tx = normalize(cross(up, n));
	let ty = cross(n, tx);
	let steps = u32(params.size.z);
	let dphi = 2.0 * PI / f32(steps);
	let dtheta = 0.5 * PI / f32(steps / 2u);
	var sum = vec3<f32>(0.0);
	var total = 0.0;
	for (var i = 0u; i < steps; i = i + 1u) {
		let phi = (f32(i) + 0.5) * dphi;
		for (var j = 0u; j < steps / 2u; j = j + 1u) {
			let theta = (f32(j) + 0.5) * dtheta;
			let d = tx * (sin(theta) * cos(phi)) + ty * (sin(theta) * sin(phi)) + n * cos(theta);
			let w = cos(theta) * sin(theta);
			// The irradiance is smooth: the source is read from a coarse level.
			sum = sum + source(d, 3.0) * w;
			total = total + w;
		}
	}
	textureStore(dst, vec2<i32>(i32(id.x), i32(id.y)), i32(id.z), vec4<f32>(sum / total, 1.0));
}
