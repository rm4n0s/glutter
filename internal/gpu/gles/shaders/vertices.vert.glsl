#version 300 es
// Triangles from vertices (Canvas.drawVertices): positions are already in pixels.

layout(location = 0) in vec2 aPos;
layout(location = 1) in vec4 aColor; // not premultiplied
layout(location = 2) in vec2 aTex;   // texture coordinates, for a shader

//#include "common.glsl"

out vec4 vColor;
out vec2 vTex;

out vec2 vDev;

void main() {
	gl_Position = toClip(aPos);
	vDev = aPos;
	vColor = aColor;
	vTex = aTex;
}
