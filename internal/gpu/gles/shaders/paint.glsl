// The source color of a paint: a solid color, a gradient or an image shader, then
// the paint's color filter and inverted colors, and the blend with what is
// already drawn. Fragment shaders include this file, after common.glsl and
// blend.glsl, with an include line.

const int SHADER_NONE = 0;
const int SHADER_LINEAR = 1;
const int SHADER_RADIAL = 2;
const int SHADER_CONICAL = 3;
const int SHADER_SWEEP = 4;
const int SHADER_IMAGE = 5;
const int SHADER_CONST = 6;

const int MAX_STOPS = 32;

uniform int uShader;
uniform vec4 uShaderXf0;   // the matrix from pixels to the shader's space: a, b, c, d
uniform vec2 uShaderXf1;   // e, f
uniform vec4 uShaderP0;    // linear: from x, y, dx, dy; radial: center x, y, radius; conical:
                           // focal x, y, focal radius, unused; sweep: center x, y, start, span
uniform vec4 uShaderP1;    // linear: 1 / (dx*dx + dy*dy); conical: end center x, y, end radius
uniform int uTileX;        // the tile modes in x and y: 0 clamp, 1 repeated, 2 mirror, 3 decal
uniform int uTileY;
uniform int uRampCount;
uniform vec4 uRampColor[MAX_STOPS];  // not premultiplied
uniform float uRampPos[MAX_STOPS];
uniform sampler2D uPaintImage;       // texture unit 2: the image of an image shader
uniform vec4 uPaintImageInfo;        // quality (0 none, 1 low, 2 medium, 3 high), level, unused, unused
uniform vec4 uConstColor;            // a degenerate gradient: premultiplied color
uniform int uDither;

uniform int uFilter;         // 0 none, 1 mode, 2 matrix, 3 linear to sRGB, 4 sRGB to linear
uniform vec4 uFilterColor;   // mode: premultiplied color
uniform int uFilterMode;
uniform vec4 uFilterMatrix[5]; // matrix: the columns of the 4x5 matrix in 0 to 255 units
uniform int uInvert;

uniform int uBlend;          // -1: the fixed-function blend does it; else a blend index
uniform sampler2D uDst;      // texture unit 3: a copy of the pixels being drawn over
uniform vec4 uDstRect;       // the copy's first column of pixels, and its bottom row, in the surface

const float PI = 3.14159265358979;

// The pixel being drawn, in pixels from the top left of the surface, at its center. It is the
// position that the vertex shader interpolates, not gl_FragCoord: with Mesa (llvmpipe),
// gl_FragCoord.y was the wrong way up in a draw into a layer's framebuffer when the window's
// was the one under it, and a cached layer came out empty.
in vec2 vDev;
vec2 devicePixel() {
	// The interpolation is off by a rounding error, which would decide a pixel whose center
	// is on an edge; the center is where the position is the nearest half pixel.
	return floor(vDev) + 0.5;
}

// ---- gradients

// Applies the tile mode to a position along a gradient; false where the shader
// draws nothing.
bool tileT(inout float t, int mode) {
	if (mode == 1) {
		t = t - floor(t);
		return true;
	}
	if (mode == 2) {
		float u = mod(t, 2.0);
		t = u > 1.0 ? 2.0 - u : u;
		return true;
	}
	if (mode == 3) {
		return t >= 0.0 && t <= 1.0;
	}
	t = clamp(t, 0.0, 1.0);
	return true;
}

// The premultiplied color at t, interpolated without premultiplied alpha.
vec4 rampAt(float t) {
	int n = uRampCount;
	int i = 0;
	while (i < n && uRampPos[i] < t) {
		i++;
	}
	vec4 c;
	if (i == 0) {
		c = uRampColor[0];
	} else if (i >= n) {
		c = uRampColor[n - 1];
	} else {
		float p0 = uRampPos[i - 1];
		float p1 = uRampPos[i];
		if (p1 <= p0) {
			c = uRampColor[i];
		} else {
			c = mix(uRampColor[i - 1], uRampColor[i], (t - p0) / (p1 - p0));
		}
	}
	c = clamp(c, 0.0, 1.0);
	return vec4(c.rgb * c.a, c.a);
}

vec4 gradientAt(float t) {
	if (isnan(t) || !tileT(t, uTileX)) {
		return vec4(0.0);
	}
	return rampAt(t);
}

// The two-point conical gradient from the focal circle to the end circle: the
// color at a point is that of the largest t whose circle, which moves and grows
// from one to the other, goes through the point.
vec4 conicalAt(vec2 q) {
	vec2 c0 = uShaderP0.xy;
	float r0 = uShaderP0.z;
	vec2 cd = uShaderP1.xy - c0;
	float dr = uShaderP1.z - r0;
	float a = dot(cd, cd) - dr * dr;
	vec2 p = q - c0;
	float b = dot(p, cd) + r0 * dr;
	float c = dot(p, p) - r0 * r0;
	float t;
	if (abs(a) < 1e-12) {
		if (b == 0.0) {
			return vec4(0.0);
		}
		t = c / (2.0 * b);
		if (r0 + t * dr < 0.0) {
			return vec4(0.0);
		}
	} else {
		float disc = b * b - a * c;
		if (disc < 0.0) {
			return vec4(0.0);
		}
		float sq = sqrt(disc);
		float t1 = (b + sq) / a;
		float t2 = (b - sq) / a;
		if (t1 < t2) {
			float tmp = t1;
			t1 = t2;
			t2 = tmp;
		}
		if (r0 + t1 * dr >= 0.0) {
			t = t1;
		} else if (r0 + t2 * dr >= 0.0) {
			t = t2;
		} else {
			return vec4(0.0);
		}
	}
	return gradientAt(t);
}

// ---- image shaders

// A texel of the shader's image at a level, with the tile modes applied to the
// pixel's index; transparent where decal draws nothing.
vec4 shaderTexel(ivec2 p, ivec2 size, int level) {
	int x = p.x;
	int y = p.y;
	if (uTileX == 1) {
		x = ((x % size.x) + size.x) % size.x;
	} else if (uTileX == 2) {
		int m = ((x % (2 * size.x)) + 2 * size.x) % (2 * size.x);
		x = m >= size.x ? 2 * size.x - 1 - m : m;
	} else if (uTileX == 3) {
		if (x < 0 || x >= size.x) {
			return vec4(0.0);
		}
	} else {
		x = clamp(x, 0, size.x - 1);
	}
	if (uTileY == 1) {
		y = ((y % size.y) + size.y) % size.y;
	} else if (uTileY == 2) {
		int m = ((y % (2 * size.y)) + 2 * size.y) % (2 * size.y);
		y = m >= size.y ? 2 * size.y - 1 - m : m;
	} else if (uTileY == 3) {
		if (y < 0 || y >= size.y) {
			return vec4(0.0);
		}
	} else {
		y = clamp(y, 0, size.y - 1);
	}
	return texelFetch(uPaintImage, ivec2(x, y), level);
}

float mitchellWeight(float x) {
	const float b = 1.0 / 3.0;
	const float c = 1.0 / 3.0;
	x = abs(x);
	if (x < 1.0) {
		return ((12.0 - 9.0 * b - 6.0 * c) * x * x * x + (-18.0 + 12.0 * b + 6.0 * c) * x * x + (6.0 - 2.0 * b)) / 6.0;
	}
	if (x < 2.0) {
		return ((-b - 6.0 * c) * x * x * x + (6.0 * b + 30.0 * c) * x * x + (-12.0 * b - 48.0 * c) * x + (8.0 * b + 24.0 * c)) / 6.0;
	}
	return 0.0;
}

// The shader's image at the point q of its pixels, with the filter quality.
vec4 imageShaderAt(vec2 q) {
	ivec2 base = textureSize(uPaintImage, 0);
	int quality = int(uPaintImageInfo.x + 0.5);
	int level = int(uPaintImageInfo.y + 0.5);
	ivec2 size = textureSize(uPaintImage, level);
	vec2 uv = q * vec2(size) / vec2(base);
	if (quality == 0) {
		return shaderTexel(ivec2(floor(uv)), size, level);
	}
	vec2 f = uv - 0.5;
	vec2 f0 = floor(f);
	vec2 t = f - f0;
	ivec2 i0 = ivec2(f0);
	if (quality == 3) {
		vec4 sum = vec4(0.0);
		for (int j = -1; j <= 2; j++) {
			float wy = mitchellWeight(float(j) - t.y);
			for (int i = -1; i <= 2; i++) {
				sum += wy * mitchellWeight(float(i) - t.x) * shaderTexel(i0 + ivec2(i, j), size, level);
			}
		}
		float a = clamp(sum.a, 0.0, 1.0);
		return vec4(clamp(sum.rgb, 0.0, a), a);
	}
	vec4 c00 = shaderTexel(i0, size, level);
	vec4 c10 = shaderTexel(i0 + ivec2(1, 0), size, level);
	vec4 c01 = shaderTexel(i0 + ivec2(0, 1), size, level);
	vec4 c11 = shaderTexel(i0 + ivec2(1, 1), size, level);
	return mix(mix(c00, c10, t.x), mix(c01, c11, t.x), t.y);
}

// The premultiplied color of the shader at the pixel at p (or, for vertices, at the
// texture coordinates p).
vec4 shaderColor(vec2 p) {
	vec2 q = vec2(
		uShaderXf0.x * p.x + uShaderXf0.z * p.y + uShaderXf1.x,
		uShaderXf0.y * p.x + uShaderXf0.w * p.y + uShaderXf1.y);
	if (uShader == SHADER_LINEAR) {
		return gradientAt(dot(q - uShaderP0.xy, uShaderP0.zw) * uShaderP1.x);
	}
	if (uShader == SHADER_RADIAL) {
		return gradientAt(length(q - uShaderP0.xy) / uShaderP0.z);
	}
	if (uShader == SHADER_CONICAL) {
		return conicalAt(q);
	}
	if (uShader == SHADER_SWEEP) {
		vec2 d = q - uShaderP0.xy;
		float a = atan(d.y, d.x) - uShaderP0.z;
		a -= 2.0 * PI * floor(a / (2.0 * PI));
		return gradientAt(a / uShaderP0.w);
	}
	if (uShader == SHADER_IMAGE) {
		return imageShaderAt(q);
	}
	return uConstColor;
}

// ---- color filters

vec4 unpremul(vec4 c) {
	if (c.a <= 0.0) {
		return vec4(0.0);
	}
	return vec4(clamp(c.rgb / c.a, 0.0, 1.0), c.a);
}

vec4 premul(vec4 c) { return vec4(c.rgb * c.a, c.a); }

vec3 linearToSrgb(vec3 v) {
	return mix(1.055 * pow(v, vec3(1.0 / 2.4)) - 0.055, 12.92 * v, lessThanEqual(v, vec3(0.0031308)));
}

vec3 srgbToLinear(vec3 v) {
	return mix(pow((v + 0.055) / 1.055, vec3(2.4)), v / 12.92, lessThanEqual(v, vec3(0.04045)));
}

// The paint's color filter, then its inverted colors.
vec4 applyFilters(vec4 px) {
	if (uFilter == 1) {
		px = blendPair(uFilterMode, uFilterColor, px);
	} else if (uFilter == 2) {
		vec4 c = unpremul(px) * 255.0;
		vec4 o = uFilterMatrix[0] * c.r + uFilterMatrix[1] * c.g + uFilterMatrix[2] * c.b + uFilterMatrix[3] * c.a + uFilterMatrix[4];
		px = premul(clamp(o / 255.0, 0.0, 1.0));
	} else if (uFilter == 3) {
		vec4 c = unpremul(px);
		px = premul(vec4(linearToSrgb(c.rgb), c.a));
	} else if (uFilter == 4) {
		vec4 c = unpremul(px);
		px = premul(vec4(srgbToLinear(c.rgb), c.a));
	}
	if (uInvert != 0) {
		vec4 c = unpremul(px);
		px = premul(vec4(1.0 - c.rgb, c.a));
	}
	return px;
}

// The source color of a paint at the pixel dev: base is the paint's color, premultiplied,
// or for a shader its alpha in all four components; p is the point the shader is
// evaluated at.
vec4 paintSource(vec4 base, vec2 p) {
	vec4 c = base;
	if (uShader != SHADER_NONE) {
		c = shaderColor(p) * base;
	}
	return applyFilters(c);
}

// ---- output

const float BAYER[64] = float[64](
	0.0, 32.0, 8.0, 40.0, 2.0, 34.0, 10.0, 42.0,
	48.0, 16.0, 56.0, 24.0, 50.0, 18.0, 58.0, 26.0,
	12.0, 44.0, 4.0, 36.0, 14.0, 46.0, 6.0, 38.0,
	60.0, 28.0, 52.0, 20.0, 62.0, 30.0, 54.0, 22.0,
	3.0, 35.0, 11.0, 43.0, 1.0, 33.0, 9.0, 41.0,
	51.0, 19.0, 59.0, 27.0, 49.0, 17.0, 57.0, 25.0,
	15.0, 47.0, 7.0, 39.0, 13.0, 45.0, 5.0, 37.0,
	63.0, 31.0, 55.0, 23.0, 61.0, 29.0, 53.0, 21.0);

// The ordered dither of gradients: up to half a level of 8 bits either way.
float ditherAt(vec2 dev) {
	ivec2 p = ivec2(floor(dev)) & 7;
	return ((BAYER[p.y * 8 + p.x] + 0.5) / 64.0 - 0.5) / 255.0;
}

// The pixel of the copy of what is being drawn over at the pixel dev.
vec4 destinationAt(vec2 dev) {
	ivec2 p = ivec2(floor(dev.x) - uDstRect.x, uDstRect.y - floor(dev.y));
	return texelFetch(uDst, p, 0);
}

// The color to write for the source src that covers the pixel by cov: with a
// fixed-function blend, the source scaled by the coverage; with any other mode, the
// destination mixed toward the blended color by the coverage.
vec4 finish(vec4 src, float cov, vec2 dev) {
	vec4 o;
	if (uBlend < 0) {
		o = src * cov;
	} else {
		vec4 d = destinationAt(dev);
		o = d + (blendPair(uBlend, src, d) - d) * cov;
	}
	if (uDither != 0 && src.a > 0.0) {
		o += ditherAt(dev);
	}
	return o;
}
