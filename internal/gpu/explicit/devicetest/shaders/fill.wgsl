// Full-screen scenes of the device tests (SPEC §27.12): a triangle that covers the target, and
// fragment shaders that write constants of every output type.

@vertex
fn vs_full(@builtin(vertex_index) i: u32) -> @builtin(position) vec4<f32> {
	var p = array<vec2<f32>, 3>(vec2<f32>(-1.0, -1.0), vec2<f32>(3.0, -1.0), vec2<f32>(-1.0, 3.0));
	return vec4<f32>(p[i], 0.0, 1.0);
}

@fragment
fn fs_float() -> @location(0) vec4<f32> {
	return vec4<f32>(0.25, 0.5, 0.75, 1.0);
}

@fragment
fn fs_uint() -> @location(0) vec4<u32> {
	return vec4<u32>(7u, 9u, 11u, 13u);
}

struct Targets {
	@location(0) a: vec4<f32>,
	@location(1) b: vec4<f32>,
	@location(2) c: vec4<u32>,
	@location(3) d: vec4<f32>,
}

@fragment
fn fs_four() -> Targets {
	var out: Targets;
	out.a = vec4<f32>(1.0, 0.0, 0.0, 0.5);
	out.b = vec4<f32>(1.0, 0.0, 0.0, 0.5);
	out.c = vec4<u32>(42u, 0u, 0u, 0u);
	out.d = vec4<f32>(0.5, 1.5, 2.5, 3.5);
	return out;
}
