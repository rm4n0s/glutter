#version 300 es
precision highp float;
precision highp int;
precision highp sampler2D;

// One pass of an image filter over a layer: a copy, one axis of a Gaussian blur, one
// axis of a dilation or erosion, a transform of the pixels, a color filter, or the
// two resamplings that let a large blur work on a smaller picture. All of them read
// the texture on unit 0 with texelFetch and write premultiplied pixels. Rows are
// counted from the top in this file, as in the surface; the textures hold their
// bottom row first.

uniform sampler2D uSrc;

const int PASS_COPY = 0;
const int PASS_BLUR = 1;
const int PASS_MORPH = 2;
const int PASS_MATRIX = 3;
const int PASS_COLOR = 4;
const int PASS_DOWN = 5;
const int PASS_UP = 6;

uniform int uPass;
uniform vec2 uSrcSize;    // the size of the source texture in pixels
uniform vec2 uOutSize;    // the size of the framebuffer drawn into
uniform int uAxis;        // 0 along x, 1 along y
uniform float uSigma;     // blur: the deviation in pixels
uniform int uRadius;      // blur and morphology: pixels on each side of the center
uniform float uNorm;      // blur: 1 over the sum of the weights
uniform int uTile;        // what lies beyond the edges: 0 clamp, 1 repeated, 2 mirror, 3 decal
uniform int uDilate;      // morphology: 1 for the largest value, 0 for the smallest
uniform vec2 uFactor;     // resampling: source pixels per output pixel in x and y
uniform vec4 uMat0;       // matrix: from output to source pixels of the surface, a, b, c, d
uniform vec2 uMat1;       // e, f
uniform vec2 uRectOrigin; // matrix: the surface pixel of the layer's top left corner
uniform int uQuality;     // matrix: 0 bilinear, 1 nearest, 2 draws nothing

//#include "common.glsl"
//#include "blend.glsl"
//#include "paint.glsl"

out vec4 fragColor;

// i modulo m in 0..m-1, also for a negative i, for which % is undefined.
int floorMod(int i, int m) {
	return i - m * int(floor(float(i) / float(m)));
}

// The index in 0..n-1 that i stands for under the tile mode; false for transparent.
bool wrapIndex(inout int i, int n, int mode) {
	if (mode == 1) {
		i = floorMod(i, n);
		return true;
	}
	if (mode == 2) {
		int m = floorMod(i, 2 * n);
		i = m >= n ? 2 * n - 1 - m : m;
		return true;
	}
	if (mode == 3) {
		return i >= 0 && i < n;
	}
	i = clamp(i, 0, n - 1);
	return true;
}

// The pixel at column x and row y from the top of a texture of the given size.
vec4 fetchTop(ivec2 p, ivec2 size, int mode) {
	if (!wrapIndex(p.x, size.x, mode) || !wrapIndex(p.y, size.y, mode)) {
		return vec4(0.0);
	}
	return texelFetch(uSrc, ivec2(p.x, size.y - 1 - p.y), 0);
}

// The same without tiling: transparent outside.
vec4 fetchOrClear(ivec2 p, ivec2 size) {
	return fetchTop(p, size, 3);
}

void main() {
	ivec2 outSize = ivec2(uOutSize + 0.5);
	ivec2 srcSize = ivec2(uSrcSize + 0.5);
	ivec2 q = ivec2(gl_FragCoord.x, uOutSize.y - gl_FragCoord.y); // column and row from the top
	ivec2 axis = uAxis == 0 ? ivec2(1, 0) : ivec2(0, 1);
	vec4 o = vec4(0.0);

	if (uPass == PASS_COPY) {
		o = fetchOrClear(q, srcSize);
	} else if (uPass == PASS_BLUR) {
		float k = -0.5 / (uSigma * uSigma);
		for (int i = -uRadius; i <= uRadius; i++) {
			o += fetchTop(q + axis * i, srcSize, uTile) * exp(float(i * i) * k);
		}
		o *= uNorm;
	} else if (uPass == PASS_MORPH) {
		o = uDilate != 0 ? vec4(0.0) : vec4(1.0);
		for (int i = -uRadius; i <= uRadius; i++) {
			vec4 c = fetchOrClear(q + axis * i, srcSize);
			o = uDilate != 0 ? max(o, c) : min(o, c);
		}
	} else if (uPass == PASS_MATRIX) {
		if (uQuality != 2) {
			vec2 dev = uRectOrigin + vec2(q) + 0.5;
			vec2 s = vec2(uMat0.x * dev.x + uMat0.z * dev.y + uMat1.x, uMat0.y * dev.x + uMat0.w * dev.y + uMat1.y) - uRectOrigin;
			if (uQuality == 1) {
				o = fetchOrClear(ivec2(floor(s)), srcSize);
			} else {
				vec2 f = s - 0.5;
				ivec2 i0 = ivec2(floor(f));
				vec2 t = f - vec2(i0);
				vec4 top = mix(fetchOrClear(i0, srcSize), fetchOrClear(i0 + ivec2(1, 0), srcSize), t.x);
				vec4 bot = mix(fetchOrClear(i0 + ivec2(0, 1), srcSize), fetchOrClear(i0 + ivec2(1, 1), srcSize), t.x);
				o = mix(top, bot, t.y);
			}
		}
	} else if (uPass == PASS_COLOR) {
		o = clamp(applyFilters(fetchOrClear(q, srcSize)), 0.0, 1.0);
	} else if (uPass == PASS_DOWN) {
		ivec2 f = ivec2(uFactor + 0.5);
		for (int j = 0; j < f.y; j++) {
			for (int i = 0; i < f.x; i++) {
				o += fetchTop(q * f + ivec2(i, j), srcSize, uTile);
			}
		}
		o /= float(f.x * f.y);
	} else {
		// Bilinear, from a picture with texels uFactor times as large.
		vec2 f = (vec2(q) + 0.5) / uFactor - 0.5;
		ivec2 i0 = ivec2(floor(f));
		vec2 t = f - vec2(i0);
		vec4 top = mix(fetchTop(i0, srcSize, 0), fetchTop(i0 + ivec2(1, 0), srcSize, 0), t.x);
		vec4 bot = mix(fetchTop(i0 + ivec2(0, 1), srcSize, 0), fetchTop(i0 + ivec2(1, 1), srcSize, 0), t.x);
		o = mix(top, bot, t.y);
	}
	fragColor = o;
}
