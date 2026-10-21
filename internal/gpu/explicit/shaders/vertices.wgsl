// Triangles from vertices (Canvas.drawVertices): positions are already in pixels. Ported from
// the GLES vertices shaders (vertices.vert.glsl, vertices.frag.glsl). Draw the vertices as they
// are: three for each triangle.

//#include "paint.wgsl.inc"

struct Frame {
	// The size of the texture being drawn into, in pixels.
	surface: vec2<f32>,
	// The pixel of the surface that the texture's top left corner is.
	origin: vec2<f32>,
}

// What a draw call sets.
struct Draw {
	// The clip mask's rectangle in the surface (x, y, width, height; a width of 0 means none).
	clipMask: vec4<f32>,
	// The paint's color, premultiplied.
	paint: vec4<f32>,
	// The clip rectangle: left, top, right, bottom in pixels.
	clip: vec4<f32>,
	// The blend index of the paint (source) with the vertex colors (destination), whether the
	// vertices have colors (0 or 1), whether the shader is evaluated at the texture coordinates
	// (0 or 1), and one unused.
	mode: vec4<f32>,
}

@group(0) @binding(0) var<uniform> frame: Frame;
@group(0) @binding(1) var<uniform> draw: Draw;
@group(1) @binding(0) var clipMask: texture_2d<f32>;

struct VertexInput {
	@location(0) pos: vec2<f32>,
	@location(1) color: vec4<f32>, // not premultiplied
	@location(2) tex: vec2<f32>,   // texture coordinates, for a shader
}

struct VertexOutput {
	@builtin(position) position: vec4<f32>,
	@location(0) dev: vec2<f32>,
	@location(1) color: vec4<f32>,
	@location(2) tex: vec2<f32>,
}

fn toClip(dev: vec2<f32>) -> vec4<f32> {
	let p = dev - frame.origin;
	return vec4<f32>(p.x / frame.surface.x * 2.0 - 1.0, 1.0 - p.y / frame.surface.y * 2.0, 0.0, 1.0);
}

@vertex
fn vs_main(v: VertexInput) -> VertexOutput {
	var out: VertexOutput;
	out.position = toClip(v.pos);
	out.dev = v.pos;
	out.color = v.color;
	out.tex = v.tex;
	return out;
}

// The part of the pixel centered at c that the interval [lo, hi] covers.
fn span(c: f32, lo: f32, hi: f32) -> f32 {
	return clamp(min(c + 0.5, hi) - max(c - 0.5, lo), 0.0, 1.0);
}

fn clipMaskCoverage(dev: vec2<f32>) -> f32 {
	if draw.clipMask.z <= 0.0 {
		return 1.0;
	}
	let p = vec2<i32>(floor(dev)) - vec2<i32>(draw.clipMask.xy);
	if p.x < 0 || p.y < 0 || p.x >= i32(draw.clipMask.z) || p.y >= i32(draw.clipMask.w) {
		return 0.0;
	}
	return textureLoad(clipMask, p, 0).r;
}

@fragment
fn fs_main(in: VertexOutput) -> @location(0) vec4<f32> {
	let dev = floor(in.dev) + vec2<f32>(0.5);
	var color = draw.paint;
	if i32(paint.shader.x + 0.5) != SHADER_NONE {
		var at = dev;
		if draw.mode.z != 0.0 {
			at = in.tex;
		}
		color = shaderColor(at) * draw.paint;
	}
	if draw.mode.y != 0.0 {
		color = blendPair(i32(draw.mode.x + 0.5), color, vec4<f32>(in.color.rgb * in.color.a, in.color.a));
	}
	color = applyFilters(color);
	let coverage = span(dev.x, draw.clip.x, draw.clip.z) * span(dev.y, draw.clip.y, draw.clip.w) * clipMaskCoverage(dev);
	return finish(color, coverage, dev);
}
