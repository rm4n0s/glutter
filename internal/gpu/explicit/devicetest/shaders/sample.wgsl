// Textures of every dimension and sample type, read into a color target (the texture scenes of the
// device tests). Params.v is the direction of the cube, or the layer or depth in z.

struct Params {
	v: vec4<f32>,
}

@group(0) @binding(0) var<uniform> params: Params;
@group(1) @binding(0) var cubeTex: texture_cube<f32>;
@group(1) @binding(1) var arrayTex: texture_2d_array<f32>;
@group(1) @binding(2) var volumeTex: texture_3d<f32>;
@group(1) @binding(3) var smp: sampler;
@group(1) @binding(4) var uintTex: texture_2d<u32>;
@group(1) @binding(5) var floatTex: texture_2d<f32>;

struct VertexOut {
	@builtin(position) pos: vec4<f32>,
	@location(0) uv: vec2<f32>,
}

@vertex
fn vs_uv(@builtin(vertex_index) i: u32) -> VertexOut {
	var p = array<vec2<f32>, 3>(vec2<f32>(-1.0, -1.0), vec2<f32>(3.0, -1.0), vec2<f32>(-1.0, 3.0));
	var out: VertexOut;
	out.pos = vec4<f32>(p[i], 0.0, 1.0);
	out.uv = vec2<f32>((p[i].x + 1.0) * 0.5, (1.0 - p[i].y) * 0.5);
	return out;
}

@fragment
fn fs_cube(in: VertexOut) -> @location(0) vec4<f32> {
	return textureSample(cubeTex, smp, params.v.xyz);
}

@fragment
fn fs_array(in: VertexOut) -> @location(0) vec4<f32> {
	return textureSample(arrayTex, smp, in.uv, i32(params.v.z));
}

@fragment
fn fs_volume(in: VertexOut) -> @location(0) vec4<f32> {
	return textureSample(volumeTex, smp, vec3<f32>(in.uv, params.v.z));
}

@fragment
fn fs_uint(in: VertexOut) -> @location(0) vec4<f32> {
	let v = textureLoad(uintTex, vec2<i32>(in.pos.xy), 0);
	return vec4<f32>(f32(v.x) / 255.0, f32(v.y) / 255.0, 0.0, 1.0);
}

@fragment
fn fs_float(in: VertexOut) -> @location(0) vec4<f32> {
	return textureLoad(floatTex, vec2<i32>(in.pos.xy), 0);
}
