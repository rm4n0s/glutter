#version 300 es
precision highp float;
precision highp int;
precision highp sampler2D;

uniform vec2 uSurface;
uniform vec4 uPaint;     // the paint's color, premultiplied
uniform int uMode;       // blend index of paint (source) with vertex colors (destination)
uniform int uHasColors;  // 0 when the vertices have no colors: the paint as it is
uniform vec4 uClip;      // left, top, right, bottom in pixels

in vec4 vColor;

out vec4 fragColor;

//#include "sdf.glsl"
//#include "clipmask.glsl"
//#include "blendpair.glsl"

void main() {
	vec2 dev = vec2(gl_FragCoord.x, uSurface.y - gl_FragCoord.y);
	vec4 color = uPaint;
	if (uHasColors != 0) {
		color = blendPair(uMode, uPaint, vec4(vColor.rgb * vColor.a, vColor.a));
	}
	color *= span(dev.x, uClip.x, uClip.z) * span(dev.y, uClip.y, uClip.w) * clipMaskCoverage(dev);
	fragColor = color;
}
