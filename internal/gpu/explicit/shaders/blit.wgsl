// Copies the frame texture into the swapchain image, whatever its format
// (SPEC §9.3). Ported from the GLES blit shader (blit.vert.glsl, blit.frag.glsl).
//
// One triangle that covers the whole target, made from the vertex index, so it
// needs no vertex buffer: draw 3 vertices. Clip space points up and texture space
// points down, so the V coordinate is flipped and the picture stays upright.
//
// The instance index is the pre-rotation of the swapchain (explicit.Rotation, SPEC §24.4): the
// image of the swapchain is the frame rotated clockwise by that many quarter turns, so that the
// presentation engine need not rotate it. The renderer draws one instance, numbered with the
// rotation (firstInstance), which is 0 on every window system but Android's.

@group(0) @binding(0) var frameTexture: texture_2d<f32>;
@group(0) @binding(1) var frameSampler: sampler;

struct VertexOutput {
	@builtin(position) position: vec4<f32>,
	@location(0) uv: vec2<f32>,
}

@vertex
fn vs_main(@builtin(vertex_index) index: u32, @builtin(instance_index) rotation: u32) -> VertexOutput {
	let p = vec2<f32>(f32((index << 1u) & 2u), f32(index & 2u));
	// The position in the image, from its top left, as a fraction.
	let a = p.x;
	let b = 1.0 - p.y;
	// The point of the frame that lands there, when the frame is rotated clockwise by rotation
	// quarter turns: a point (u, v) of the frame goes to (1 - v, u) for one turn.
	var uv = vec2<f32>(a, b);
	if (rotation == 1u) {
		uv = vec2<f32>(b, 1.0 - a);
	} else if (rotation == 2u) {
		uv = vec2<f32>(1.0 - a, 1.0 - b);
	} else if (rotation == 3u) {
		uv = vec2<f32>(1.0 - b, a);
	}
	var out: VertexOutput;
	out.position = vec4<f32>(p * 2.0 - 1.0, 0.0, 1.0);
	out.uv = uv;
	return out;
}

@fragment
fn fs_main(in: VertexOutput) -> @location(0) vec4<f32> {
	return textureSample(frameTexture, frameSampler, in.uv);
}
