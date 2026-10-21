// Copies the frame texture into the swapchain image, whatever its format
// (SPEC §9.3). Ported from the GLES blit shader (blit.vert.glsl, blit.frag.glsl).
//
// One triangle that covers the whole target, made from the vertex index, so it
// needs no vertex buffer: draw 3 vertices. Clip space points up and texture space
// points down, so the V coordinate is flipped and the picture stays upright.

@group(0) @binding(0) var frameTexture: texture_2d<f32>;
@group(0) @binding(1) var frameSampler: sampler;

struct VertexOutput {
	@builtin(position) position: vec4<f32>,
	@location(0) uv: vec2<f32>,
}

@vertex
fn vs_main(@builtin(vertex_index) index: u32) -> VertexOutput {
	let p = vec2<f32>(f32((index << 1u) & 2u), f32(index & 2u));
	var out: VertexOutput;
	out.position = vec4<f32>(p * 2.0 - 1.0, 0.0, 1.0);
	out.uv = vec2<f32>(p.x, 1.0 - p.y);
	return out;
}

@fragment
fn fs_main(in: VertexOutput) -> @location(0) vec4<f32> {
	return textureSample(frameTexture, frameSampler, in.uv);
}
