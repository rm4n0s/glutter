#version 300 es
precision highp float;
precision highp int;
precision highp sampler2D;

uniform vec4 uPaint;     // the paint's color, premultiplied; for a shader, its alpha in all four
uniform int uMode;       // blend index of paint (source) with vertex colors (destination)
uniform int uHasColors;  // 0 when the vertices have no colors: the paint as it is
uniform int uAtTexture;  // 1 when the shader is evaluated at the texture coordinates
uniform vec4 uClip;      // left, top, right, bottom in pixels

in vec4 vColor;
in vec2 vTex;

out vec4 fragColor;

//#include "common.glsl"
//#include "sdf.glsl"
//#include "clipmask.glsl"
//#include "blend.glsl"
//#include "paint.glsl"

void main() {
	vec2 dev = devicePixel();
	vec4 color = uPaint;
	if (uShader != SHADER_NONE) {
		color = shaderColor(uAtTexture != 0 ? vTex : dev) * uPaint;
	}
	if (uHasColors != 0) {
		color = blendPair(uMode, color, vec4(vColor.rgb * vColor.a, vColor.a));
	}
	color = applyFilters(color);
	float coverage = span(dev.x, uClip.x, uClip.z) * span(dev.y, uClip.y, uClip.w) * clipMaskCoverage(dev);
	fragColor = finish(color, coverage, dev);
}
