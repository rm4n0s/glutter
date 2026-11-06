// Compute shaders of the device tests (SPEC §27.12, step 30.2): squares a million numbers, paints a
// storage texture, and writes the arguments of indirect draws and dispatches.

struct Params {
	count: u32,
	a: u32,
	b: u32,
	c: u32,
}

@group(0) @binding(0) var<uniform> params: Params;
@group(0) @binding(1) var<storage, read> input: array<u32>;
@group(0) @binding(2) var<storage, read_write> output: array<u32>;
@group(1) @binding(0) var img: texture_storage_2d<rgba8unorm, write>;
@group(1) @binding(1) var<storage, read_write> args: array<u32>;

@compute @workgroup_size(64)
fn cs_square(@builtin(global_invocation_id) id: vec3<u32>) {
	if (id.x < params.count) {
		output[id.x] = input[id.x] * input[id.x];
	}
}

// Paints texel (x, y) of a texture of 4 by 4 with (x*60, y*60, 128, 255).
@compute @workgroup_size(2, 2)
fn cs_paint(@builtin(global_invocation_id) id: vec3<u32>) {
	textureStore(img, vec2<i32>(id.xy), vec4<f32>(f32(id.x) * 60.0 / 255.0, f32(id.y) * 60.0 / 255.0, 128.0 / 255.0, 1.0));
}

// The arguments of DrawIndirect: 3 vertices, 1 instance, the first vertex 0 and the first instance 0.
@compute @workgroup_size(1)
fn cs_draw_args() {
	args[0] = 3u;
	args[1] = 1u;
	args[2] = 0u;
	args[3] = 0u;
}

// The arguments of DrawIndexedIndirect: 3 indices, 1 instance, the first index 0, the base vertex 0 and
// the first instance 0.
@compute @workgroup_size(1)
fn cs_indexed_args() {
	args[0] = 3u;
	args[1] = 1u;
	args[2] = 0u;
	args[3] = 0u;
	args[4] = 0u;
}

// The arguments of DispatchIndirect: params.a workgroups in x.
@compute @workgroup_size(1)
fn cs_dispatch_args() {
	args[0] = params.a;
	args[1] = 1u;
	args[2] = 1u;
}
