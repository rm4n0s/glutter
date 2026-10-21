// The source color of a paint: a solid color, a gradient or an image shader, then the paint's
// color filter and inverted colors, and the blend with what is already drawn. Ported from
// gles/shaders/paint.glsl. Fragment shaders include this file, after blend.wgsl.inc.
//
//#include "blend.wgsl.inc"

const SHADER_NONE: i32 = 0;
const SHADER_LINEAR: i32 = 1;
const SHADER_RADIAL: i32 = 2;
const SHADER_CONICAL: i32 = 3;
const SHADER_SWEEP: i32 = 4;
const SHADER_IMAGE: i32 = 5;
const SHADER_CONST: i32 = 6;

const MAX_STOPS: i32 = 32;
const PI: f32 = 3.14159265358979;

// What a draw call's paint sets, the layout of gpu.PaintSpec as float4s (see paintUniform).
struct Paint {
	// The shader kind, the tile modes in x and y (0 clamp, 1 repeated, 2 mirror, 3 decal) and
	// the number of color stops.
	shader: vec4<f32>,
	// The matrix from pixels to the shader's space: a, b, c, d, then e, f.
	xf0: vec4<f32>,
	xf1: vec4<f32>,
	// linear: from x, y, dx, dy; radial: center x, y, radius; conical: focal x, y, focal radius;
	// sweep: center x, y, start, span.
	p0: vec4<f32>,
	// linear: 1 / (dx*dx + dy*dy); conical: end center x, y, end radius.
	p1: vec4<f32>,
	// The image shader's quality (0 none, 1 low, 2 medium, 3 high) and level.
	imageInfo: vec4<f32>,
	// A degenerate gradient: premultiplied color.
	constColor: vec4<f32>,
	// Dither (0 or 1), the color filter (0 none, 1 mode, 2 matrix, 3 linear to sRGB, 4 sRGB to
	// linear), the blend index of a mode filter, and inverted colors (0 or 1).
	flags: vec4<f32>,
	// The blend index of a blend that reads the destination (-1: the blend unit does it), and the
	// pixel of the surface that the copy of the destination starts at.
	blend: vec4<f32>,
	// A mode filter's premultiplied color.
	filterColor: vec4<f32>,
	// A matrix filter: the columns of the 4x5 matrix in 0 to 255 units.
	filterMatrix: array<vec4<f32>, 5>,
	// The gradient's colors (not premultiplied) and positions, four to a float4.
	rampColor: array<vec4<f32>, 32>,
	rampPos: array<vec4<f32>, 8>,
}

@group(0) @binding(2) var<uniform> paint: Paint;
// The image of an image shader, and a copy of the pixels being drawn over.
@group(3) @binding(0) var paintImage: texture_2d<f32>;
@group(3) @binding(1) var dst: texture_2d<f32>;

// ---- gradients

// Applies the tile mode to a position along a gradient; the second component is 0 where the
// shader draws nothing.
fn tileT(t: f32, mode: i32) -> vec2<f32> {
	if mode == 1 {
		return vec2<f32>(t - floor(t), 1.0);
	}
	if mode == 2 {
		let u = t - 2.0 * floor(t / 2.0);
		if u > 1.0 {
			return vec2<f32>(2.0 - u, 1.0);
		}
		return vec2<f32>(u, 1.0);
	}
	if mode == 3 {
		if t >= 0.0 && t <= 1.0 {
			return vec2<f32>(t, 1.0);
		}
		return vec2<f32>(t, 0.0);
	}
	return vec2<f32>(clamp(t, 0.0, 1.0), 1.0);
}

fn rampPosAt(i: i32) -> f32 {
	return paint.rampPos[i >> 2u][i & 3];
}

// The premultiplied color at t, interpolated without premultiplied alpha.
fn rampAt(t: f32) -> vec4<f32> {
	let n = i32(paint.shader.w + 0.5);
	var i = 0;
	while i < n && rampPosAt(i) < t {
		i++;
	}
	var c: vec4<f32>;
	if i == 0 {
		c = paint.rampColor[0];
	} else if i >= n {
		c = paint.rampColor[n - 1];
	} else {
		let p0 = rampPosAt(i - 1);
		let p1 = rampPosAt(i);
		if p1 <= p0 {
			c = paint.rampColor[i];
		} else {
			c = mix(paint.rampColor[i - 1], paint.rampColor[i], (t - p0) / (p1 - p0));
		}
	}
	c = clamp(c, vec4<f32>(0.0), vec4<f32>(1.0));
	return vec4<f32>(c.rgb * c.a, c.a);
}

fn gradientAt(t0: f32) -> vec4<f32> {
	if t0 != t0 {
		return vec4<f32>(0.0);
	}
	let tt = tileT(t0, i32(paint.shader.y + 0.5));
	if tt.y == 0.0 {
		return vec4<f32>(0.0);
	}
	return rampAt(tt.x);
}

// The two-point conical gradient from the focal circle to the end circle: the color at a point
// is that of the largest t whose circle, which moves and grows from one to the other, goes
// through the point.
fn conicalAt(q: vec2<f32>) -> vec4<f32> {
	let c0 = paint.p0.xy;
	let r0 = paint.p0.z;
	let cd = paint.p1.xy - c0;
	let dr = paint.p1.z - r0;
	let a = dot(cd, cd) - dr * dr;
	let p = q - c0;
	let b = dot(p, cd) + r0 * dr;
	let c = dot(p, p) - r0 * r0;
	var t: f32;
	if abs(a) < 1e-12 {
		if b == 0.0 {
			return vec4<f32>(0.0);
		}
		t = c / (2.0 * b);
		if r0 + t * dr < 0.0 {
			return vec4<f32>(0.0);
		}
	} else {
		let disc = b * b - a * c;
		if disc < 0.0 {
			return vec4<f32>(0.0);
		}
		let sq = sqrt(disc);
		var t1 = (b + sq) / a;
		var t2 = (b - sq) / a;
		if t1 < t2 {
			let tmp = t1;
			t1 = t2;
			t2 = tmp;
		}
		if r0 + t1 * dr >= 0.0 {
			t = t1;
		} else if r0 + t2 * dr >= 0.0 {
			t = t2;
		} else {
			return vec4<f32>(0.0);
		}
	}
	return gradientAt(t);
}

// ---- image shaders

// A texel of the shader's image at a level, with the tile modes applied to the pixel's index;
// transparent where decal draws nothing.
fn shaderTexel(p: vec2<i32>, size: vec2<i32>, level: i32) -> vec4<f32> {
	var x = p.x;
	var y = p.y;
	let tileX = i32(paint.shader.y + 0.5);
	let tileY = i32(paint.shader.z + 0.5);
	if tileX == 1 {
		x = ((x % size.x) + size.x) % size.x;
	} else if tileX == 2 {
		let m = ((x % (2 * size.x)) + 2 * size.x) % (2 * size.x);
		if m >= size.x {
			x = 2 * size.x - 1 - m;
		} else {
			x = m;
		}
	} else if tileX == 3 {
		if x < 0 || x >= size.x {
			return vec4<f32>(0.0);
		}
	} else {
		x = clamp(x, 0, size.x - 1);
	}
	if tileY == 1 {
		y = ((y % size.y) + size.y) % size.y;
	} else if tileY == 2 {
		let m = ((y % (2 * size.y)) + 2 * size.y) % (2 * size.y);
		if m >= size.y {
			y = 2 * size.y - 1 - m;
		} else {
			y = m;
		}
	} else if tileY == 3 {
		if y < 0 || y >= size.y {
			return vec4<f32>(0.0);
		}
	} else {
		y = clamp(y, 0, size.y - 1);
	}
	return textureLoad(paintImage, vec2<i32>(x, y), level);
}

fn mitchellWeight(x0: f32) -> f32 {
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

// The shader's image at the point q of its pixels, with the filter quality.
fn imageShaderAt(q: vec2<f32>) -> vec4<f32> {
	let base = vec2<i32>(textureDimensions(paintImage, 0));
	let quality = i32(paint.imageInfo.x + 0.5);
	let level = i32(paint.imageInfo.y + 0.5);
	let size = vec2<i32>(textureDimensions(paintImage, level));
	let uv = q * vec2<f32>(size) / vec2<f32>(base);
	if quality == 0 {
		return shaderTexel(vec2<i32>(floor(uv)), size, level);
	}
	let f = uv - vec2<f32>(0.5);
	let f0 = floor(f);
	let t = f - f0;
	let i0 = vec2<i32>(f0);
	if quality == 3 {
		var sum = vec4<f32>(0.0);
		for (var j = -1; j <= 2; j++) {
			let wy = mitchellWeight(f32(j) - t.y);
			for (var i = -1; i <= 2; i++) {
				sum += wy * mitchellWeight(f32(i) - t.x) * shaderTexel(i0 + vec2<i32>(i, j), size, level);
			}
		}
		let a = clamp(sum.a, 0.0, 1.0);
		return vec4<f32>(clamp(sum.rgb, vec3<f32>(0.0), vec3<f32>(a)), a);
	}
	let c00 = shaderTexel(i0, size, level);
	let c10 = shaderTexel(i0 + vec2<i32>(1, 0), size, level);
	let c01 = shaderTexel(i0 + vec2<i32>(0, 1), size, level);
	let c11 = shaderTexel(i0 + vec2<i32>(1, 1), size, level);
	return mix(mix(c00, c10, t.x), mix(c01, c11, t.x), t.y);
}

// The premultiplied color of the shader at the pixel at p (or, for vertices, at the texture
// coordinates p).
fn shaderColor(p: vec2<f32>) -> vec4<f32> {
	let q = vec2<f32>(
		paint.xf0.x * p.x + paint.xf0.z * p.y + paint.xf1.x,
		paint.xf0.y * p.x + paint.xf0.w * p.y + paint.xf1.y);
	let kind = i32(paint.shader.x + 0.5);
	if kind == SHADER_LINEAR {
		return gradientAt(dot(q - paint.p0.xy, paint.p0.zw) * paint.p1.x);
	}
	if kind == SHADER_RADIAL {
		return gradientAt(length(q - paint.p0.xy) / paint.p0.z);
	}
	if kind == SHADER_CONICAL {
		return conicalAt(q);
	}
	if kind == SHADER_SWEEP {
		let d = q - paint.p0.xy;
		var a = atan2(d.y, d.x) - paint.p0.z;
		a = a - 2.0 * PI * floor(a / (2.0 * PI));
		return gradientAt(a / paint.p0.w);
	}
	if kind == SHADER_IMAGE {
		return imageShaderAt(q);
	}
	return paint.constColor;
}

// ---- color filters

fn unpremul(c: vec4<f32>) -> vec4<f32> {
	if c.a <= 0.0 {
		return vec4<f32>(0.0);
	}
	return vec4<f32>(clamp(c.rgb / c.a, vec3<f32>(0.0), vec3<f32>(1.0)), c.a);
}

fn premul(c: vec4<f32>) -> vec4<f32> {
	return vec4<f32>(c.rgb * c.a, c.a);
}

fn linearToSrgb(v: vec3<f32>) -> vec3<f32> {
	return select(1.055 * pow(v, vec3<f32>(1.0 / 2.4)) - vec3<f32>(0.055), 12.92 * v, v <= vec3<f32>(0.0031308));
}

fn srgbToLinear(v: vec3<f32>) -> vec3<f32> {
	return select(pow((v + vec3<f32>(0.055)) / 1.055, vec3<f32>(2.4)), v / 12.92, v <= vec3<f32>(0.04045));
}

// The paint's color filter, then its inverted colors.
fn applyFilters(px0: vec4<f32>) -> vec4<f32> {
	var px = px0;
	let kind = i32(paint.flags.y + 0.5);
	if kind == 1 {
		px = blendPair(i32(paint.flags.z + 0.5), paint.filterColor, px);
	} else if kind == 2 {
		let c = unpremul(px) * 255.0;
		let o = paint.filterMatrix[0] * c.r + paint.filterMatrix[1] * c.g + paint.filterMatrix[2] * c.b + paint.filterMatrix[3] * c.a + paint.filterMatrix[4];
		px = premul(clamp(o / 255.0, vec4<f32>(0.0), vec4<f32>(1.0)));
	} else if kind == 3 {
		let c = unpremul(px);
		px = premul(vec4<f32>(linearToSrgb(c.rgb), c.a));
	} else if kind == 4 {
		let c = unpremul(px);
		px = premul(vec4<f32>(srgbToLinear(c.rgb), c.a));
	}
	if paint.flags.w != 0.0 {
		let c = unpremul(px);
		px = premul(vec4<f32>(vec3<f32>(1.0) - c.rgb, c.a));
	}
	return px;
}

// The source color of a paint at the pixel dev: base is the paint's color, premultiplied, or for
// a shader its alpha in all four components; p is the point the shader is evaluated at.
fn paintSource(base: vec4<f32>, p: vec2<f32>) -> vec4<f32> {
	var c = base;
	if i32(paint.shader.x + 0.5) != SHADER_NONE {
		c = shaderColor(p) * base;
	}
	return applyFilters(c);
}

// ---- output

var<private> bayer: array<f32, 64> = array<f32, 64>(
	0.0, 32.0, 8.0, 40.0, 2.0, 34.0, 10.0, 42.0,
	48.0, 16.0, 56.0, 24.0, 50.0, 18.0, 58.0, 26.0,
	12.0, 44.0, 4.0, 36.0, 14.0, 46.0, 6.0, 38.0,
	60.0, 28.0, 52.0, 20.0, 62.0, 30.0, 54.0, 22.0,
	3.0, 35.0, 11.0, 43.0, 1.0, 33.0, 9.0, 41.0,
	51.0, 19.0, 59.0, 27.0, 49.0, 17.0, 57.0, 25.0,
	15.0, 47.0, 7.0, 39.0, 13.0, 45.0, 5.0, 37.0,
	63.0, 31.0, 55.0, 23.0, 61.0, 29.0, 53.0, 21.0);

// The ordered dither of gradients: up to half a level of 8 bits either way.
fn ditherAt(dev: vec2<f32>) -> f32 {
	let p = vec2<i32>(floor(dev)) & vec2<i32>(7);
	return ((bayer[p.y * 8 + p.x] + 0.5) / 64.0 - 0.5) / 255.0;
}

// The pixel of the copy of what is being drawn over at the pixel dev.
fn destinationAt(dev: vec2<f32>) -> vec4<f32> {
	return textureLoad(dst, vec2<i32>(floor(dev)) - vec2<i32>(paint.blend.yz), 0);
}

// The color to write for the source src that covers the pixel by cov: with a fixed-function
// blend, the source scaled by the coverage; with any other mode, the destination mixed toward
// the blended color by the coverage.
fn finish(src: vec4<f32>, cov: f32, dev: vec2<f32>) -> vec4<f32> {
	var o: vec4<f32>;
	if paint.blend.x < 0.0 {
		o = src * cov;
	} else {
		let d = destinationAt(dev);
		o = d + (blendPair(i32(paint.blend.x + 0.5), src, d) - d) * cov;
	}
	if paint.flags.x != 0.0 && src.a > 0.0 {
		o = o + vec4<f32>(ditherAt(dev));
	}
	return o;
}
