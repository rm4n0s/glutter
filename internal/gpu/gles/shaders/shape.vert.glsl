#version 300 es
// Analytic shapes (SPEC §9.1): one instance is one shape. The vertex shader only
// places a quad that covers the shape's bounds in pixels; the fragment shader
// works out the coverage of each pixel from a signed distance field.
//
// Every attribute is per instance (divisor 1) and the same for the four
// corners, so each is passed on as a flat varying.

layout(location = 0) in vec4 aQuad;   // left, top, right, bottom in pixels
layout(location = 1) in vec4 aInv0;   // the inverse transform: a, b, c, d
layout(location = 2) in vec4 aInv1;   // e, f, kind, flags
layout(location = 3) in vec4 aColor;  // premultiplied
layout(location = 4) in vec4 aClip;   // left, top, right, bottom in pixels
layout(location = 5) in vec4 aA;      // shape A: center x, y, half width, half height
layout(location = 6) in vec4 aARad1;  // top left x, y; top right x, y
layout(location = 7) in vec4 aARad2;  // bottom right x, y; bottom left x, y
layout(location = 8) in vec4 aB;      // shape B, which is cut out of A
layout(location = 9) in vec4 aBRad1;
layout(location = 10) in vec4 aBRad2;
layout(location = 11) in vec4 aParam; // half stroke width, cap, exponent, pixels per local unit

//#include "common.glsl"

flat out vec4 vInv0;
flat out vec4 vInv1;
flat out vec4 vColor;
flat out vec4 vClip;
flat out vec4 vA;
flat out vec4 vARad1;
flat out vec4 vARad2;
flat out vec4 vB;
flat out vec4 vBRad1;
flat out vec4 vBRad2;
flat out vec4 vParam;

void main() {
	vec2 corner = vec2(float(gl_VertexID & 1), float(gl_VertexID >> 1));
	vec2 dev = mix(aQuad.xy, aQuad.zw, corner);
	gl_Position = toClip(dev);
	vInv0 = aInv0;
	vInv1 = aInv1;
	vColor = aColor;
	vClip = aClip;
	vA = aA;
	vARad1 = aARad1;
	vARad2 = aARad2;
	vB = aB;
	vBRad1 = aBRad1;
	vBRad2 = aBRad2;
	vParam = aParam;
}
