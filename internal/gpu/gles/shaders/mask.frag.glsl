#version 300 es
precision highp float;
precision highp int;
precision highp sampler2D;

uniform sampler2D uAtlas; // texture unit 0: 8-bit coverage

flat in vec4 vQuad;
flat in vec4 vAtlas;
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
	// The quad covers whole pixels, one texel each: the texel of this pixel is the
	// tile's corner plus its distance from the quad's corner.
	ivec2 texel = ivec2(vAtlas.xy) + ivec2(floor(dev)) - ivec2(vQuad.xy);
	float coverage = texelFetch(uAtlas, texel, 0).r;
	coverage *= span(dev.x, vClip.x, vClip.z) * span(dev.y, vClip.y, vClip.w) * clipMaskCoverage(dev);
	if (coverage <= 0.0) {
		discard;
	}
	fragColor = finish(paintSource(vColor, dev), coverage, dev);
}
