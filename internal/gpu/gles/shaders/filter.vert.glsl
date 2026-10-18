#version 300 es
// A pass over a layer's pixels (image filters, SPEC §9.1): one rectangle that covers
// the whole framebuffer it draws into. Every output pixel is worked out in the fragment
// shader from the pixels of the source texture.

void main() {
	vec2 corner = vec2(float(gl_VertexID & 1), float(gl_VertexID >> 1));
	gl_Position = vec4(corner * 2.0 - 1.0, 0.0, 1.0);
}
