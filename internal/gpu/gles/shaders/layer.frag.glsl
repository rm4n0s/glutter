#version 300 es
precision highp float;
precision highp int;
precision highp sampler2D;

uniform sampler2D uLayer;  // texture unit 0: the layer's pixels, premultiplied
uniform vec4 uLayerRect;   // the layer's pixels in the surface: x, y, width, height

flat in vec4 vColor;
flat in vec4 vClip;

out vec4 fragColor;

//#include "common.glsl"
//#include "sdf.glsl"
//#include "clipmask.glsl"
//#include "blend.glsl"
//#include "paint.glsl"

void main() {
	vec2 dev = devicePixel();
	float coverage = span(dev.x, vClip.x, vClip.z) * span(dev.y, vClip.y, vClip.w) * clipMaskCoverage(dev);
	if (coverage <= 0.0) {
		discard;
	}
	// The layer's framebuffer has its bottom row first.
	ivec2 p = ivec2(int(floor(dev.x - uLayerRect.x)), int(uLayerRect.w) - 1 - int(floor(dev.y - uLayerRect.y)));
	vec4 color = applyFilters(texelFetch(uLayer, p, 0) * vColor);
	fragColor = finish(color, coverage, dev);
}
