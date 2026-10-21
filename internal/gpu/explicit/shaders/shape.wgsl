// Analytic shapes (SPEC §9.1, §9.3): one instance is one shape. The vertex shader only
// places a quad that covers the shape's bounds in pixels; the fragment shader works out
// the coverage of each pixel from a signed distance field. Ported from the GLES shape
// shaders (shape.vert.glsl, shape.frag.glsl) with sdf.glsl and common.glsl's toClip.
//
// Draw 6 vertices per instance: two triangles made from the vertex index. The instance
// is twelve vec4 attributes (gpu.ShapeInstance), the same for the corners, so each is
// passed on as a flat varying.
//
// The color is the paint's solid color, premultiplied; the blend unit scales it by the
// coverage (gradients and blend modes that read the destination come in step 20.5). The
// clip masks of shape clips multiply the coverage (clipmask.glsl).

struct Frame {
	// The size of the texture being drawn into, in pixels.
	surface: vec2<f32>,
	// The pixel of the surface that the texture's top left corner is: a layer draws
	// into a texture that covers a rectangle of the surface.
	origin: vec2<f32>,
}

// What a draw call sets: the clip mask's rectangle in the surface (x, y, width, height; a
// width of 0 means no mask).
struct Draw {
	clipMask: vec4<f32>,
}

// The uniform buffers are bound with dynamic offsets.
@group(0) @binding(0) var<uniform> frame: Frame;
@group(0) @binding(1) var<uniform> draw: Draw;

// The coverage of the clip shapes that are not rectangles: one 8-bit coverage per pixel of a
// rectangle of the surface; a 1x1 texture when there is no mask.
@group(1) @binding(0) var clipMask: texture_2d<f32>;

struct Instance {
	@location(0) quad: vec4<f32>,   // left, top, right, bottom in pixels
	@location(1) inv0: vec4<f32>,   // the inverse transform: a, b, c, d
	@location(2) inv1: vec4<f32>,   // e, f, kind, flags
	@location(3) color: vec4<f32>,  // premultiplied
	@location(4) clip: vec4<f32>,   // left, top, right, bottom in pixels
	@location(5) a: vec4<f32>,      // shape A: center x, y, half width, half height
	@location(6) arad1: vec4<f32>,  // top left x, y; top right x, y
	@location(7) arad2: vec4<f32>,  // bottom right x, y; bottom left x, y
	@location(8) b: vec4<f32>,      // shape B, which is cut out of A
	@location(9) brad1: vec4<f32>,
	@location(10) brad2: vec4<f32>,
	@location(11) param: vec4<f32>, // half stroke width, cap, exponent, pixels per local unit
}

struct VertexOutput {
	@builtin(position) position: vec4<f32>,
	@location(0) dev: vec2<f32>,
	@location(1) @interpolate(flat) inv0: vec4<f32>,
	@location(2) @interpolate(flat) inv1: vec4<f32>,
	@location(3) @interpolate(flat) color: vec4<f32>,
	@location(4) @interpolate(flat) clip: vec4<f32>,
	@location(5) @interpolate(flat) a: vec4<f32>,
	@location(6) @interpolate(flat) arad1: vec4<f32>,
	@location(7) @interpolate(flat) arad2: vec4<f32>,
	@location(8) @interpolate(flat) b: vec4<f32>,
	@location(9) @interpolate(flat) brad1: vec4<f32>,
	@location(10) @interpolate(flat) brad2: vec4<f32>,
	@location(11) @interpolate(flat) param: vec4<f32>,
}

// The kinds of shape (gpu.ShapeKind*).
const KIND_RRECT: i32 = 0;   // a rounded rectangle, minus shape B when the flags say so
const KIND_SUPER: i32 = 1;   // a rounded superellipse
const KIND_SEGMENT: i32 = 2; // a line segment with caps, or a dot
const KIND_FILL: i32 = 3;    // every pixel
const KIND_BOX: i32 = 4;     // an axis-aligned rectangle in pixels, minus shape B, by exact area
const KIND_OUTLINE: i32 = 5; // every point within a half width of a rounded rectangle's outline

// The flags (gpu.ShapeFlag*).
const FLAG_HAS_B: i32 = 1;
const FLAG_ANTIALIAS: i32 = 2;
const FLAG_SIMILARITY: i32 = 4;

// The clip space position of the point dev, in pixels from the top left of the surface.
fn toClip(dev: vec2<f32>) -> vec4<f32> {
	let p = dev - frame.origin;
	return vec4<f32>(p.x / frame.surface.x * 2.0 - 1.0, 1.0 - p.y / frame.surface.y * 2.0, 0.0, 1.0);
}

@vertex
fn vs_main(@builtin(vertex_index) index: u32, instance: Instance) -> VertexOutput {
	// The corners of the two triangles: (0,0) (1,0) (0,1) and (1,0) (1,1) (0,1).
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
	out.a = instance.a;
	out.arad1 = instance.arad1;
	out.arad2 = instance.arad2;
	out.b = instance.b;
	out.brad1 = instance.brad1;
	out.brad2 = instance.brad2;
	out.param = instance.param;
	return out;
}

// The coverage of the pixel whose center is dev (in pixels from the top left).
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

// ---- signed distance fields (sdf.glsl)

// Distance from the corner q of an ellipse with radii r, with q measured from the
// ellipse's center: a first-order estimate that is exact for a circle.
fn ellipseDistance(q: vec2<f32>, r: vec2<f32>) -> f32 {
	let k0 = length(q / r);
	let k1 = length(q / (r * r));
	if k1 > 0.0 {
		return k0 * (k0 - 1.0) / k1;
	}
	return -min(r.x, r.y);
}

// The corner radii of the quadrant that p is in (y points down).
fn cornerRadii(p: vec2<f32>, r1: vec4<f32>, r2: vec4<f32>) -> vec2<f32> {
	if p.x < 0.0 {
		if p.y < 0.0 {
			return r1.xy;
		}
		return r2.zw;
	}
	if p.y < 0.0 {
		return r1.zw;
	}
	return r2.xy;
}

// A rounded rectangle centered on the origin with half extents h.
fn rrectDistance(p: vec2<f32>, h: vec2<f32>, r1: vec4<f32>, r2: vec4<f32>) -> f32 {
	let r = cornerRadii(p, r1, r2);
	let q = abs(p) - (h - r);
	if q.x > 0.0 && q.y > 0.0 {
		if r.x > 0.0 && r.y > 0.0 {
			return ellipseDistance(q, r);
		}
		return length(q);
	}
	return max(q.x - r.x, q.y - r.y);
}

// The part of the pixel centered at c that the interval [lo, hi] covers.
fn span(c: f32, lo: f32, hi: f32) -> f32 {
	return clamp(min(c + 0.5, hi) - max(c - 0.5, lo), 0.0, 1.0);
}

// The part of the pixel centered at dev that the rectangle r (left, top, right, bottom)
// covers; without antialiasing, 1 when the center is inside.
fn boxCoverage(dev: vec2<f32>, r: vec4<f32>, antialias: bool) -> f32 {
	if antialias {
		return span(dev.x, r.x, r.z) * span(dev.y, r.y, r.w);
	}
	if dev.x >= r.x && dev.x < r.z && dev.y >= r.y && dev.y < r.w {
		return 1.0;
	}
	return 0.0;
}

// A rounded superellipse: the corner boxes in r1 and r2 are filled by the curve
// |x|^n + |y|^n = 1.
fn superDistance(p: vec2<f32>, h: vec2<f32>, r1: vec4<f32>, r2: vec4<f32>, n: f32) -> f32 {
	let r = cornerRadii(p, r1, r2);
	let q = abs(p) - (h - r);
	if q.x > 0.0 && q.y > 0.0 {
		if r.x <= 0.0 || r.y <= 0.0 {
			return length(q);
		}
		let u = q / r;
		let f = pow(u.x, n) + pow(u.y, n) - 1.0;
		let g = n * length(pow(u, vec2<f32>(n - 1.0)) / r);
		if g > 0.0 {
			return f / g;
		}
		return 0.0;
	}
	return max(q.x - r.x, q.y - r.y);
}

// A segment from a to b of half width hw. cap is 0 for butt, 1 for round, 2 for square.
// A segment of no length is a dot: round or square, nothing for butt.
fn segmentDistance(p: vec2<f32>, a: vec2<f32>, b: vec2<f32>, hw: f32, cap: f32) -> f32 {
	let ba = b - a;
	let len = length(ba);
	if len < 1e-5 {
		let pa = p - a;
		if cap == 1.0 {
			return length(pa) - hw;
		}
		if cap == 2.0 {
			return max(abs(pa.x), abs(pa.y)) - hw;
		}
		return 1e9;
	}
	let dir = ba / len;
	let pa = p - (a + b) * 0.5;
	let along = abs(dot(pa, dir));
	let across = abs(dot(pa, vec2<f32>(-dir.y, dir.x)));
	if cap == 1.0 {
		return length(vec2<f32>(max(along - len * 0.5, 0.0), across)) - hw;
	}
	if cap == 2.0 {
		return max(along - len * 0.5 - hw, across - hw);
	}
	return max(along - len * 0.5, across - hw);
}

@fragment
fn fs_main(in: VertexOutput) -> @location(0) vec4<f32> {
	// The interpolation is off by a rounding error, which would decide a pixel whose
	// center is on an edge; the center is where the position is the nearest half pixel.
	let dev = floor(in.dev) + vec2<f32>(0.5);

	// The clip rectangle: exact coverage, so anti-aliased clip edges are smooth.
	let clipCoverage = span(dev.x, in.clip.x, in.clip.z) * span(dev.y, in.clip.y, in.clip.w) * clipMaskCoverage(dev);

	let kind = i32(in.inv1.z + 0.5);
	let flags = i32(in.inv1.w + 0.5);
	let antialias = (flags & FLAG_ANTIALIAS) != 0;
	let hasB = (flags & FLAG_HAS_B) != 0;
	let p = vec2<f32>(
		in.inv0.x * dev.x + in.inv0.z * dev.y + in.inv1.x,
		in.inv0.y * dev.x + in.inv0.w * dev.y + in.inv1.y);

	// The distance d of the pixel from the shape, negative inside, in the shape's units.
	// A box has its coverage from the exact area instead.
	var d = -1.0;
	var boxCov = 0.0;
	if kind == KIND_BOX {
		boxCov = boxCoverage(dev, in.a, antialias);
		if hasB {
			boxCov -= boxCoverage(dev, in.b, antialias);
		}
	} else if kind == KIND_RRECT {
		d = rrectDistance(p - in.a.xy, in.a.zw, in.arad1, in.arad2);
		if hasB {
			d = max(d, -rrectDistance(p - in.b.xy, in.b.zw, in.brad1, in.brad2));
		}
	} else if kind == KIND_OUTLINE {
		d = abs(rrectDistance(p - in.a.xy, in.a.zw, in.arad1, in.arad2)) - in.param.x;
	} else if kind == KIND_SUPER {
		d = superDistance(p - in.a.xy, in.a.zw, in.arad1, in.arad2, in.param.z);
	} else if kind == KIND_SEGMENT {
		d = segmentDistance(p, in.a.xy, in.a.zw, in.param.x, in.param.y);
	}

	// The screen derivatives need every pixel of the quad: they come before anything
	// that returns or discards.
	let g = length(vec2<f32>(dpdx(d), dpdy(d)));

	var coverage: f32;
	if kind == KIND_BOX {
		coverage = boxCov;
	} else if !antialias {
		coverage = select(0.0, 1.0, d <= 0.0);
	} else if kind == KIND_FILL {
		coverage = 1.0;
	} else {
		// The distance in pixels: the same everywhere for a transform that only rotates
		// and scales evenly, found from the screen derivatives otherwise.
		var pixels: f32;
		if (flags & FLAG_SIMILARITY) != 0 {
			pixels = d * in.param.w;
		} else if g > 1e-9 {
			pixels = d / g;
		} else {
			pixels = select(1.0, -1.0, d <= 0.0);
		}
		coverage = clamp(0.5 - pixels, 0.0, 1.0);
	}
	let total = coverage * clipCoverage;
	if total <= 0.0 {
		discard;
	}
	return in.color * total;
}
