#version 300 es
// A layer composited into its parent: one instance is the rectangle of the
// layer's pixels that the clip leaves.

layout(location = 0) in vec4 aQuad;   // left, top, right, bottom in pixels
layout(location = 1) in vec4 aColor;  // the layer's alpha in all four components
layout(location = 2) in vec4 aClip;   // left, top, right, bottom in pixels

//#include "common.glsl"

flat out vec4 vColor;
flat out vec4 vClip;

void main() {
	vec2 corner = vec2(float(gl_VertexID & 1), float(gl_VertexID >> 1));
	gl_Position = toClip(mix(aQuad.xy, aQuad.zw, corner));
	vColor = aColor;
	vClip = aClip;
}
