#version 300 es
precision highp float;
precision highp int;
precision highp sampler2D;

uniform sampler2D uLayer;  // texture unit 0: the layer's pixels, premultiplied
uniform sampler2D uLayerOwn; // texture unit 4: the layer before it was blurred, for a mask
uniform vec4 uLayerRect;   // the layer's pixels in the surface: x, y, width, height
uniform int uMaskStyle;    // -1: the layer's pixels are composited; else the layer's alpha is the
                           // coverage of a blurred shape (a blur style) that the paint's color fills
uniform int uDrawnOnly;    // 1: pixels with no alpha are left as they are

flat in vec4 vColor;
flat in vec4 vClip;

out vec4 fragColor;

//#include "common.glsl"
//#include "sdf.glsl"
//#include "clipmask.glsl"
//#include "blend.glsl"
//#include "paint.glsl"
//#include "shadow.glsl"

void main() {
	vec2 dev = devicePixel();
	float coverage = span(dev.x, vClip.x, vClip.z) * span(dev.y, vClip.y, vClip.w) * clipMaskCoverage(dev);
	if (coverage <= 0.0) {
		discard;
	}
	// The layer's framebuffer has its bottom row first.
	ivec2 p = ivec2(int(floor(dev.x - uLayerRect.x)), int(uLayerRect.w) - 1 - int(floor(dev.y - uLayerRect.y)));
	vec4 texel = texelFetch(uLayer, p, 0);
	if (uDrawnOnly != 0 && texel.a <= 0.0) {
		discard;
	}
	if (uMaskStyle >= 0) {
		float own = uMaskStyle == 0 ? 0.0 : texelFetch(uLayerOwn, p, 0).a;
		float cov = blurStyle(uMaskStyle, texel.a, own) * coverage;
		if (cov < 1.0 / 1024.0) {
			discard;
		}
		fragColor = finish(paintSource(vColor, dev), cov, dev);
		return;
	}
	vec4 color = applyFilters(texel * vColor);
	fragColor = finish(color, coverage, dev);
}
