#version 300 es
// Coverage masks (SPEC §9.1, "General paths"): one instance is one tile of the
// coverage atlas, drawn over the pixels it covers.

layout(location = 0) in vec4 aQuad;   // left, top, right, bottom in pixels
layout(location = 1) in vec4 aAtlas;  // the tile's top left texel in the atlas page
layout(location = 2) in vec4 aColor;  // premultiplied
layout(location = 3) in vec4 aClip;   // left, top, right, bottom in pixels

uniform vec2 uSurface; // the size of the surface in pixels

flat out vec4 vQuad;
flat out vec4 vAtlas;
flat out vec4 vColor;
flat out vec4 vClip;

void main() {
	vec2 corner = vec2(float(gl_VertexID & 1), float(gl_VertexID >> 1));
	vec2 dev = mix(aQuad.xy, aQuad.zw, corner);
	gl_Position = vec4(dev.x / uSurface.x * 2.0 - 1.0, 1.0 - dev.y / uSurface.y * 2.0, 0.0, 1.0);
	vQuad = aQuad;
	vAtlas = aAtlas;
	vColor = aColor;
	vClip = aClip;
}
