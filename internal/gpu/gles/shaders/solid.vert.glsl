#version 300 es
// A triangle that covers the whole viewport, made from gl_VertexID, so that it
// needs no vertex buffer: draw 3 vertices.
void main() {
	vec2 p = vec2(float((gl_VertexID << 1) & 2), float(gl_VertexID & 2));
	gl_Position = vec4(p * 2.0 - 1.0, 0.0, 1.0);
}
