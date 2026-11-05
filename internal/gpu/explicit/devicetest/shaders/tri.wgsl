// Triangles with a position and a color from a vertex buffer (the depth, stencil, culling, topology,
// write mask and multisampling scenes of the device tests).

struct VertexOut {
	@builtin(position) pos: vec4<f32>,
	@location(0) color: vec4<f32>,
}

@vertex
fn vs_tri(@location(0) pos: vec3<f32>, @location(1) color: vec4<f32>) -> VertexOut {
	var out: VertexOut;
	out.pos = vec4<f32>(pos, 1.0);
	out.color = color;
	return out;
}

@fragment
fn fs_tri(in: VertexOut) -> @location(0) vec4<f32> {
	return in.color;
}
