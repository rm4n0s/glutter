#version 300 es
// Images: one instance is one rectangle of an image, drawn through an affine
// transform. The fragment shader maps each pixel back to the image's texels.

layout(location = 0) in vec4 aQuad;   // left, top, right, bottom in pixels
layout(location = 1) in vec4 aInv0;   // pixels to texels: a, b, c, d
layout(location = 2) in vec4 aInv1;   // e, f, flags, unused
layout(location = 3) in vec4 aColor;  // premultiplied multiplier: the paint's alpha
layout(location = 4) in vec4 aClip;   // left, top, right, bottom in pixels
layout(location = 5) in vec4 aBox;    // the rectangle drawn: texels, or pixels with the box flag
layout(location = 6) in vec4 aImg;    // level of detail, quality, blend index (-1 none), unused
layout(location = 7) in vec4 aTint;   // premultiplied color blended with the image

//#include "common.glsl"

flat out vec4 vInv0;
flat out vec4 vInv1;
flat out vec4 vColor;
flat out vec4 vClip;
flat out vec4 vBox;
flat out vec4 vImg;
flat out vec4 vTint;

void main() {
	vec2 corner = vec2(float(gl_VertexID & 1), float(gl_VertexID >> 1));
	vec2 dev = mix(aQuad.xy, aQuad.zw, corner);
	gl_Position = toClip(dev);
	vInv0 = aInv0;
	vInv1 = aInv1;
	vColor = aColor;
	vClip = aClip;
	vBox = aBox;
	vImg = aImg;
	vTint = aTint;
}
