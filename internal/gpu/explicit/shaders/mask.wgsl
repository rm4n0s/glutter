// Coverage masks (SPEC §9.1 "General paths"): one instance is one tile of the coverage atlas,
// drawn over the pixels it covers. Ported from the GLES mask shaders (mask.vert.glsl,
// mask.frag.glsl). Draw 6 vertices per instance.

//#include "paint.wgsl.inc"

struct Frame {
	// The size of the texture being drawn into, in pixels.
	surface: vec2<f32>,
	// The pixel of the surface that the texture's top left corner is.
	origin: vec2<f32>,
}

// The clip mask's rectangle in the surface (x, y, width, height; a width of 0 means none).
struct Draw {
	clipMask: vec4<f32>,
}

@group(0) @binding(0) var<uniform> frame: Frame;
@group(0) @binding(1) var<uniform> draw: Draw;
@group(1) @binding(0) var clipMask: texture_2d<f32>;
// An atlas page: 8-bit coverage.
@group(2) @binding(0) var atlas: texture_2d<f32>;

struct Instance {
	@location(0) quad: vec4<f32>,  // left, top, right, bottom in pixels
	@location(1) tile: vec4<f32>,  // the tile's top left texel in the atlas page
	@location(2) color: vec4<f32>, // premultiplied
	@location(3) clip: vec4<f32>,  // left, top, right, bottom in pixels
}

struct VertexOutput {
	@builtin(position) position: vec4<f32>,
	@location(0) dev: vec2<f32>,
	@location(1) @interpolate(flat) quad: vec4<f32>,
	@location(2) @interpolate(flat) tile: vec4<f32>,
	@location(3) @interpolate(flat) color: vec4<f32>,
	@location(4) @interpolate(flat) clip: vec4<f32>,
}

fn toClip(dev: vec2<f32>) -> vec4<f32> {
	let p = dev - frame.origin;
	return vec4<f32>(p.x / frame.surface.x * 2.0 - 1.0, 1.0 - p.y / frame.surface.y * 2.0, 0.0, 1.0);
}

@vertex
fn vs_main(@builtin(vertex_index) index: u32, instance: Instance) -> VertexOutput {
	var corners = array<vec2<f32>, 6>(
		vec2<f32>(0.0, 0.0), vec2<f32>(1.0, 0.0), vec2<f32>(0.0, 1.0),
		vec2<f32>(1.0, 0.0), vec2<f32>(1.0, 1.0), vec2<f32>(0.0, 1.0),
	);
	let dev = mix(instance.quad.xy, instance.quad.zw, corners[index]);
	var out: VertexOutput;
	out.position = toClip(dev);
	out.dev = dev;
	out.quad = instance.quad;
	out.tile = instance.tile;
	out.color = instance.color;
	out.clip = instance.clip;
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
	// The quad covers whole pixels, one texel each: the texel of this pixel is the tile's
	// corner plus its distance from the quad's corner.
	let texel = vec2<i32>(in.tile.xy) + vec2<i32>(floor(dev)) - vec2<i32>(in.quad.xy);
	var coverage = textureLoad(atlas, texel, 0).r;
	coverage = coverage * span(dev.x, in.clip.x, in.clip.z) * span(dev.y, in.clip.y, in.clip.w) * clipMaskCoverage(dev);
	if coverage <= 0.0 {
		discard;
	}
	return finish(paintSource(in.color, dev), coverage, dev);
}
