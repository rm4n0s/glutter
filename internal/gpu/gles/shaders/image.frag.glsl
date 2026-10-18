#version 300 es
precision highp float;
precision highp int;
precision highp sampler2D;

uniform vec2 uSurface;
uniform sampler2D uImage; // texture unit 0: premultiplied RGBA with mipmaps

flat in vec4 vInv0;
flat in vec4 vInv1;
flat in vec4 vColor;
flat in vec4 vClip;
flat in vec4 vBox;
flat in vec4 vImg;
flat in vec4 vTint;

out vec4 fragColor;

const int FLAG_BOX = 1;       // vBox is in pixels: exact coverage of an axis-aligned rectangle
const int FLAG_ANTIALIAS = 2;

const int QUALITY_NONE = 0;
const int QUALITY_LOW = 1;
const int QUALITY_MEDIUM = 2;
const int QUALITY_HIGH = 3;

//#include "sdf.glsl"
//#include "clipmask.glsl"
//#include "blendpair.glsl"

// The Mitchell-Netravali cubic with B = C = 1/3.
float mitchell(float x) {
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

// A texel of a level, with the edge repeated outside.
vec4 texel(ivec2 p, ivec2 size, int level) {
	return texelFetch(uImage, clamp(p, ivec2(0), size - 1), level);
}

vec4 bicubic(vec2 uv, ivec2 base, int level) {
	ivec2 size = textureSize(uImage, level);
	vec2 f = uv * vec2(size) / vec2(base) - 0.5;
	vec2 f0 = floor(f);
	vec2 t = f - f0;
	vec4 sum = vec4(0.0);
	for (int j = -1; j <= 2; j++) {
		float wy = mitchell(float(j) - t.y);
		for (int i = -1; i <= 2; i++) {
			float w = wy * mitchell(float(i) - t.x);
			sum += w * texel(ivec2(f0) + ivec2(i, j), size, level);
		}
	}
	// The cubic can overshoot: keep a valid premultiplied color.
	float a = clamp(sum.a, 0.0, 1.0);
	return vec4(clamp(sum.rgb, 0.0, a), a);
}

void main() {
	vec2 dev = vec2(gl_FragCoord.x, uSurface.y - gl_FragCoord.y);
	int flags = int(vInv1.z + 0.5);
	bool aa = (flags & FLAG_ANTIALIAS) != 0;
	vec2 uv = vec2(
		vInv0.x * dev.x + vInv0.z * dev.y + vInv1.x,
		vInv0.y * dev.x + vInv0.w * dev.y + vInv1.y);

	float coverage;
	if ((flags & FLAG_BOX) != 0) {
		coverage = boxCoverage(dev, vBox, aa);
	} else {
		// The rectangle in texels: the distance outside it, in pixels by the screen
		// derivatives.
		vec2 c = (vBox.xy + vBox.zw) * 0.5;
		vec2 q = abs(uv - c) - (vBox.zw - vBox.xy) * 0.5;
		float d = max(q.x, q.y);
		if (aa) {
			float g = length(vec2(dFdx(d), dFdy(d)));
			coverage = clamp(0.5 - (g > 1e-9 ? d / g : (d <= 0.0 ? -1.0 : 1.0)), 0.0, 1.0);
		} else {
			coverage = d <= 0.0 ? 1.0 : 0.0;
		}
	}
	coverage *= span(dev.x, vClip.x, vClip.z) * span(dev.y, vClip.y, vClip.w) * clipMaskCoverage(dev);

	ivec2 base = textureSize(uImage, 0);
	int quality = int(vImg.y + 0.5);
	vec4 color;
	if (quality == QUALITY_NONE) {
		color = texel(ivec2(floor(uv)), base, 0);
	} else if (quality == QUALITY_HIGH) {
		color = bicubic(uv, base, int(vImg.x + 0.5));
	} else {
		color = textureLod(uImage, uv / vec2(base), vImg.x);
	}
	color *= vColor;
	int mode = int(vImg.z + 0.5);
	if (vImg.z >= 0.0) {
		color = blendPair(mode, color, vTint);
	}
	color *= coverage;
	if (color.a <= 0.0 && color.r <= 0.0 && color.g <= 0.0 && color.b <= 0.0) {
		discard;
	}
	fragColor = color;
}
