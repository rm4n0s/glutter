// A layer composited into its parent: one instance is the rectangle of the layer's pixels that
// the clip leaves. Ported from the GLES layer shaders (layer.vert.glsl, layer.frag.glsl). Draw 6
// vertices per instance.

//#include "paint.wgsl.inc"
//#include "shadow.wgsl.inc"

struct Frame {
	// The size of the texture being drawn into, in pixels.
	surface: vec2<f32>,
	// The pixel of the surface that the texture's top left corner is.
	origin: vec2<f32>,
}

// What a draw call sets: the clip mask's rectangle in the surface (x, y, width, height; a width
// of 0 means none), the layer's pixels in the surface (x, y, width, height), and how the pixels
// are read: the mask style (-1: the layer's pixels are composited; else the layer's alpha is the
// coverage of a blurred shape, a blur style, that the paint's color fills) and drawnOnly (1:
// pixels with no alpha are left as they are).
struct Draw {
	clipMask: vec4<f32>,
	layerRect: vec4<f32>,
	read: vec4<f32>,
}

@group(0) @binding(0) var<uniform> frame: Frame;
@group(0) @binding(1) var<uniform> draw: Draw;
@group(1) @binding(0) var clipMask: texture_2d<f32>;
// The layer's pixels, premultiplied.
@group(2) @binding(0) var layer: texture_2d<f32>;
// The layer before it was blurred, for a mask style other than normal.
@group(2) @binding(1) var layerOwn: texture_2d<f32>;

struct Instance {
	@location(0) quad: vec4<f32>,  // left, top, right, bottom in pixels
	@location(1) color: vec4<f32>, // the layer's alpha in all four components
	@location(2) clip: vec4<f32>,  // left, top, right, bottom in pixels
}

struct VertexOutput {
	@builtin(position) position: vec4<f32>,
	@location(0) dev: vec2<f32>,
	@location(1) @interpolate(flat) color: vec4<f32>,
	@location(2) @interpolate(flat) clip: vec4<f32>,
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
	let coverage = span(dev.x, in.clip.x, in.clip.z) * span(dev.y, in.clip.y, in.clip.w) * clipMaskCoverage(dev);
	if coverage <= 0.0 {
		discard;
	}
	let p = vec2<i32>(floor(dev - draw.layerRect.xy));
	let texel = textureLoad(layer, p, 0);
	if draw.read.y != 0.0 && texel.a <= 0.0 {
		discard;
	}
	let style = i32(draw.read.x);
	if style >= 0 {
		var own = 0.0;
		if style != 0 {
			own = textureLoad(layerOwn, p, 0).a;
		}
		let cov = blurStyle(style, texel.a, own) * coverage;
		if cov < 1.0 / 1024.0 {
			discard;
		}
		return finish(paintSource(in.color, dev), cov, dev);
	}
	let color = applyFilters(texel * in.color);
	return finish(color, coverage, dev);
}
