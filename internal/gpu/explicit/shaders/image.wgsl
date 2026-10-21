// Images: one instance is one rectangle of an image, drawn through an affine transform. The
// fragment shader maps each pixel back to the image's texels and samples them with the paint's
// filter quality: nearest, bilinear, bilinear on one mip level, or bicubic. Ported from the
// GLES image shaders (image.vert.glsl, image.frag.glsl); the paint's color filters and the
// tint of a sprite (blend modes that read the color) come with step 20.5.
// Draw 6 vertices per instance.

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
// The image: premultiplied RGBA with mipmaps, and a linear sampler that picks a level.
@group(2) @binding(0) var image: texture_2d<f32>;
@group(2) @binding(1) var imageSampler: sampler;

struct Instance {
	@location(0) quad: vec4<f32>,  // left, top, right, bottom in pixels
	@location(1) inv0: vec4<f32>,  // pixels to texels: a, b, c, d
	@location(2) inv1: vec4<f32>,  // e, f, flags, unused
	@location(3) color: vec4<f32>, // premultiplied multiplier: the paint's alpha
	@location(4) clip: vec4<f32>,  // left, top, right, bottom in pixels
	@location(5) box: vec4<f32>,   // the rectangle drawn: texels, or pixels with the box flag
	@location(6) img: vec4<f32>,   // level of detail, quality, blend index (-1 none), unused
	@location(7) tint: vec4<f32>,  // premultiplied color blended with the image
}

struct VertexOutput {
	@builtin(position) position: vec4<f32>,
	@location(0) dev: vec2<f32>,
	@location(1) @interpolate(flat) inv0: vec4<f32>,
	@location(2) @interpolate(flat) inv1: vec4<f32>,
	@location(3) @interpolate(flat) color: vec4<f32>,
	@location(4) @interpolate(flat) clip: vec4<f32>,
	@location(5) @interpolate(flat) box: vec4<f32>,
	@location(6) @interpolate(flat) img: vec4<f32>,
}

const FLAG_BOX: i32 = 1;       // box is in pixels: exact coverage of an axis-aligned rectangle
const FLAG_ANTIALIAS: i32 = 2;

const QUALITY_NONE: i32 = 0;
const QUALITY_HIGH: i32 = 3;

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
	out.inv0 = instance.inv0;
	out.inv1 = instance.inv1;
	out.color = instance.color;
	out.clip = instance.clip;
	out.box = instance.box;
	out.img = instance.img;
	return out;
}

// The part of the pixel centered at c that the interval [lo, hi] covers.
fn span(c: f32, lo: f32, hi: f32) -> f32 {
	return clamp(min(c + 0.5, hi) - max(c - 0.5, lo), 0.0, 1.0);
}

// The part of the pixel centered at dev that the rectangle r covers; without antialiasing, 1
// when the center is inside.
fn boxCoverage(dev: vec2<f32>, r: vec4<f32>, antialias: bool) -> f32 {
	if antialias {
		return span(dev.x, r.x, r.z) * span(dev.y, r.y, r.w);
	}
	if dev.x >= r.x && dev.x < r.z && dev.y >= r.y && dev.y < r.w {
		return 1.0;
	}
	return 0.0;
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

// The Mitchell-Netravali cubic with B = C = 1/3.
fn mitchell(x0: f32) -> f32 {
	let b = 1.0 / 3.0;
	let c = 1.0 / 3.0;
	let x = abs(x0);
	if x < 1.0 {
		return ((12.0 - 9.0 * b - 6.0 * c) * x * x * x + (-18.0 + 12.0 * b + 6.0 * c) * x * x + (6.0 - 2.0 * b)) / 6.0;
	}
	if x < 2.0 {
		return ((-b - 6.0 * c) * x * x * x + (6.0 * b + 30.0 * c) * x * x + (-12.0 * b - 48.0 * c) * x + (8.0 * b + 24.0 * c)) / 6.0;
	}
	return 0.0;
}

// A texel of a level, with the edge repeated outside.
fn texel(p: vec2<i32>, size: vec2<i32>, level: i32) -> vec4<f32> {
	return textureLoad(image, clamp(p, vec2<i32>(0), size - vec2<i32>(1)), level);
}

fn bicubic(uv: vec2<f32>, base: vec2<i32>, level: i32) -> vec4<f32> {
	let size = vec2<i32>(textureDimensions(image, level));
	let f = uv * vec2<f32>(size) / vec2<f32>(base) - vec2<f32>(0.5);
	let f0 = floor(f);
	let t = f - f0;
	var sum = vec4<f32>(0.0);
	for (var j = -1; j <= 2; j++) {
		let wy = mitchell(f32(j) - t.y);
		for (var i = -1; i <= 2; i++) {
			let w = wy * mitchell(f32(i) - t.x);
			sum += w * texel(vec2<i32>(f0) + vec2<i32>(i, j), size, level);
		}
	}
	// The cubic can overshoot: keep a valid premultiplied color.
	let a = clamp(sum.a, 0.0, 1.0);
	return vec4<f32>(clamp(sum.rgb, vec3<f32>(0.0), vec3<f32>(a)), a);
}

@fragment
fn fs_main(in: VertexOutput) -> @location(0) vec4<f32> {
	let dev = floor(in.dev) + vec2<f32>(0.5);
	let flags = i32(in.inv1.z + 0.5);
	let aa = (flags & FLAG_ANTIALIAS) != 0;
	let uv = vec2<f32>(
		in.inv0.x * dev.x + in.inv0.z * dev.y + in.inv1.x,
		in.inv0.y * dev.x + in.inv0.w * dev.y + in.inv1.y);

	// The rectangle in texels: the distance outside it, in pixels by the screen derivatives.
	// The derivatives come before anything that returns or discards.
	let c = (in.box.xy + in.box.zw) * 0.5;
	let q = abs(uv - c) - (in.box.zw - in.box.xy) * 0.5;
	let d = max(q.x, q.y);
	let g = length(vec2<f32>(dpdx(d), dpdy(d)));

	var coverage: f32;
	if (flags & FLAG_BOX) != 0 {
		coverage = boxCoverage(dev, in.box, aa);
	} else if aa {
		var pixels = select(1.0, -1.0, d <= 0.0);
		if g > 1e-9 {
			pixels = d / g;
		}
		coverage = clamp(0.5 - pixels, 0.0, 1.0);
	} else {
		coverage = select(0.0, 1.0, d <= 0.0);
	}
	coverage = coverage * span(dev.x, in.clip.x, in.clip.z) * span(dev.y, in.clip.y, in.clip.w) * clipMaskCoverage(dev);

	let base = vec2<i32>(textureDimensions(image, 0));
	let quality = i32(in.img.y + 0.5);
	var color: vec4<f32>;
	if quality == QUALITY_NONE {
		color = texel(vec2<i32>(floor(uv)), base, 0);
	} else if quality == QUALITY_HIGH {
		color = bicubic(uv, base, i32(in.img.x + 0.5));
	} else {
		color = textureSampleLevel(image, imageSampler, uv / vec2<f32>(base), in.img.x);
	}
	color = color * in.color;
	if coverage <= 0.0 {
		discard;
	}
	return color * coverage;
}
