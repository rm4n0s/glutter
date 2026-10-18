#version 300 es
// A triangle that covers the whole viewport, with texture coordinates that map
// the viewport to the whole texture: draw 3 vertices.
out vec2 vUV;

void main() {
	vec2 p = vec2(float((gl_VertexID << 1) & 2), float(gl_VertexID & 2));
	vUV = p;
	gl_Position = vec4(p * 2.0 - 1.0, 0.0, 1.0);
}
