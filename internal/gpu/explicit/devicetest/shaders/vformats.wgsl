// Vertex attributes of several formats (the vertex format scene of the device tests): a position
// from snorm16x2, a color from unorm8x4, an id from uint32, a half float and bytes of uint8x4.

struct VertexOut {
	@builtin(position) pos: vec4<f32>,
	@location(0) color: vec4<f32>,
}

@vertex
fn vs_formats(
	@location(0) p: vec2<f32>,
	@location(1) c: vec4<f32>,
	@location(2) id: u32,
	@location(3) h: vec2<f32>,
	@location(4) b: vec4<u32>,
) -> VertexOut {
	var out: VertexOut;
	out.pos = vec4<f32>(p, 0.0, 1.0);
	out.color = vec4<f32>(c.r, f32(id) / 255.0, h.x, f32(b.x) / 255.0);
	return out;
}

@fragment
fn fs_formats(in: VertexOut) -> @location(0) vec4<f32> {
	return in.color;
}
