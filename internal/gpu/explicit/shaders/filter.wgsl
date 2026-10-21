// One pass of an image filter over a layer: a copy, one axis of a Gaussian blur, one axis of a
// dilation or erosion, a transform of the pixels, a color filter, or the two resamplings that let
// a large blur work on a smaller picture (gpu.FilterPass). Ported from
// gles/shaders/filter.frag.glsl. All of them read the source with textureLoad and write
// premultiplied pixels, rows counted from the top as in the surface. Draw 3 vertices: one triangle
// covers the whole target.

//#include "paint.wgsl.inc"

const PASS_COPY: i32 = 0;
const PASS_BLUR: i32 = 1;
const PASS_MORPH: i32 = 2;
const PASS_MATRIX: i32 = 3;
const PASS_COLOR: i32 = 4;
const PASS_DOWN: i32 = 5;
const PASS_UP: i32 = 6;

// What a pass sets.
struct Pass {
	// The pass kind, the axis (0 along x, 1 along y), the tile mode beyond the edges (0 clamp,
	// 1 repeated, 2 mirror, 3 decal), and 1 for a dilation or 0 for an erosion.
	kind: vec4<f32>,
	// The size of the source in pixels, and of the target.
	sizes: vec4<f32>,
	// The Gaussian's deviation, the taps on each side of the center, and 1 over the sum of the
	// weights.
	blur: vec4<f32>,
	// Resampling: the source pixels per output pixel in x and y; and the quality of a matrix pass
	// (0 bilinear, 1 nearest, 2 draws nothing).
	factor: vec4<f32>,
	// A matrix pass: from output to source pixels of the surface, a, b, c, d, then e, f and the
	// surface pixel of the layer's top left corner.
	mat0: vec4<f32>,
	mat1: vec4<f32>,
}

@group(0) @binding(1) var<uniform> params: Pass;
@group(1) @binding(0) var src: texture_2d<f32>;

struct VertexOutput {
	@builtin(position) position: vec4<f32>,
}

@vertex
fn vs_main(@builtin(vertex_index) index: u32) -> VertexOutput {
	let p = vec2<f32>(f32((index << 1u) & 2u), f32(index & 2u));
	var out: VertexOutput;
	out.position = vec4<f32>(p * 2.0 - 1.0, 0.0, 1.0);
	return out;
}

// i modulo m in 0..m-1, also for a negative i.
fn floorMod(i: i32, m: i32) -> i32 {
	return i - m * i32(floor(f32(i) / f32(m)));
}

// The index in 0..n-1 that i stands for under the tile mode; the second component is 0 for
// transparent.
fn wrapIndex(i: i32, n: i32, mode: i32) -> vec2<i32> {
	if mode == 1 {
		return vec2<i32>(floorMod(i, n), 1);
	}
	if mode == 2 {
		let m = floorMod(i, 2 * n);
		return vec2<i32>(select(m, 2 * n - 1 - m, m >= n), 1);
	}
	if mode == 3 {
		return vec2<i32>(i, select(0, 1, i >= 0 && i < n));
	}
	return vec2<i32>(clamp(i, 0, n - 1), 1);
}

// The pixel at column x and row y from the top of a texture of the given size.
fn fetchTop(p: vec2<i32>, size: vec2<i32>, mode: i32) -> vec4<f32> {
	let x = wrapIndex(p.x, size.x, mode);
	let y = wrapIndex(p.y, size.y, mode);
	if x.y == 0 || y.y == 0 {
		return vec4<f32>(0.0);
	}
	return textureLoad(src, vec2<i32>(x.x, y.x), 0);
}

// The same without tiling: transparent outside.
fn fetchOrClear(p: vec2<i32>, size: vec2<i32>) -> vec4<f32> {
	return fetchTop(p, size, 3);
}

@fragment
fn fs_main(in: VertexOutput) -> @location(0) vec4<f32> {
	let srcSize = vec2<i32>(params.sizes.xy + vec2<f32>(0.5));
	let q = vec2<i32>(floor(in.position.xy));
	let kind = i32(params.kind.x + 0.5);
	let axis = select(vec2<i32>(0, 1), vec2<i32>(1, 0), params.kind.y < 0.5);
	let tile = i32(params.kind.z + 0.5);
	let radius = i32(params.blur.y + 0.5);
	var o = vec4<f32>(0.0);

	if kind == PASS_COPY {
		o = fetchOrClear(q, srcSize);
	} else if kind == PASS_BLUR {
		let k = -0.5 / (params.blur.x * params.blur.x);
		for (var i = -radius; i <= radius; i++) {
			o += fetchTop(q + axis * i, srcSize, tile) * exp(f32(i * i) * k);
		}
		o *= params.blur.z;
	} else if kind == PASS_MORPH {
		let dilate = params.kind.w > 0.5;
		o = select(vec4<f32>(1.0), vec4<f32>(0.0), dilate);
		for (var i = -radius; i <= radius; i++) {
			let c = fetchOrClear(q + axis * i, srcSize);
			o = select(min(o, c), max(o, c), dilate);
		}
	} else if kind == PASS_MATRIX {
		if params.factor.z < 1.5 {
			let origin = params.mat1.zw;
			let dev = origin + vec2<f32>(q) + vec2<f32>(0.5);
			let s = vec2<f32>(
				params.mat0.x * dev.x + params.mat0.z * dev.y + params.mat1.x,
				params.mat0.y * dev.x + params.mat0.w * dev.y + params.mat1.y) - origin;
			if params.factor.z > 0.5 {
				o = fetchOrClear(vec2<i32>(floor(s)), srcSize);
			} else {
				let f = s - vec2<f32>(0.5);
				let i0 = vec2<i32>(floor(f));
				let t = f - vec2<f32>(i0);
				let top = mix(fetchOrClear(i0, srcSize), fetchOrClear(i0 + vec2<i32>(1, 0), srcSize), t.x);
				let bot = mix(fetchOrClear(i0 + vec2<i32>(0, 1), srcSize), fetchOrClear(i0 + vec2<i32>(1, 1), srcSize), t.x);
				o = mix(top, bot, t.y);
			}
		}
	} else if kind == PASS_COLOR {
		o = clamp(applyFilters(fetchOrClear(q, srcSize)), vec4<f32>(0.0), vec4<f32>(1.0));
	} else if kind == PASS_DOWN {
		let f = vec2<i32>(params.factor.xy + vec2<f32>(0.5));
		for (var j = 0; j < f.y; j++) {
			for (var i = 0; i < f.x; i++) {
				o += fetchTop(q * f + vec2<i32>(i, j), srcSize, tile);
			}
		}
		o /= f32(f.x * f.y);
	} else {
		// Bilinear, from a picture with texels factor times as large.
		let f = (vec2<f32>(q) + vec2<f32>(0.5)) / params.factor.xy - vec2<f32>(0.5);
		let i0 = vec2<i32>(floor(f));
		let t = f - vec2<f32>(i0);
		let top = mix(fetchTop(i0, srcSize, 0), fetchTop(i0 + vec2<i32>(1, 0), srcSize, 0), t.x);
		let bot = mix(fetchTop(i0 + vec2<i32>(0, 1), srcSize, 0), fetchTop(i0 + vec2<i32>(1, 1), srcSize, 0), t.x);
		o = mix(top, bot, t.y);
	}
	return o;
}
