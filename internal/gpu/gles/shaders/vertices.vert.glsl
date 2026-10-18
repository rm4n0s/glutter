#version 300 es
// Triangles from vertices (Canvas.drawVertices): positions are already in pixels.

layout(location = 0) in vec2 aPos;
layout(location = 1) in vec4 aColor; // not premultiplied

uniform vec2 uSurface; // the size of the surface in pixels

out vec4 vColor;

void main() {
	gl_Position = vec4(aPos.x / uSurface.x * 2.0 - 1.0, 1.0 - aPos.y / uSurface.y * 2.0, 0.0, 1.0);
	vColor = aColor;
}
