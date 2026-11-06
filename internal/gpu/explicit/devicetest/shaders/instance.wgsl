// Two instances of a quad drawn as a triangle strip: instance 0 covers the left half of the target in
// red, instance 1 the right half in green (the instance index, SPEC §27.1).

struct VertexOut {
	@builtin(position) pos: vec4<f32>,
	@location(0) color: vec4<f32>,
}

@vertex
fn vs_instance(@builtin(vertex_index) v: u32, @builtin(instance_index) i: u32) -> VertexOut {
	var corner = array<vec2<f32>, 4>(vec2<f32>(0.0, 0.0), vec2<f32>(1.0, 0.0), vec2<f32>(0.0, 1.0), vec2<f32>(1.0, 1.0));
	var out: VertexOut;
	out.pos = vec4<f32>(-1.0 + corner[v].x + f32(i), -1.0 + 2.0 * corner[v].y, 0.0, 1.0);
	out.color = select(vec4<f32>(0.0, 1.0, 0.0, 1.0), vec4<f32>(1.0, 0.0, 0.0, 1.0), i == 0u);
	return out;
}

@fragment
fn fs_instance(in: VertexOut) -> @location(0) vec4<f32> {
	return in.color;
}
