#version 300 es
// A pass over a layer's pixels (image filters, SPEC §9.1): one rectangle that covers
// the whole framebuffer it draws into. Every output pixel is worked out in the fragment
// shader from the pixels of the source texture.

uniform vec2 uOutSize; // the size of the framebuffer being drawn into, in pixels

// The output pixel, in pixels from the top left, at its center. gl_FragCoord is not used
// for it (see devicePixel in paint.glsl).
out vec2 vDev;

void main() {
	vec2 corner = vec2(float(gl_VertexID & 1), float(gl_VertexID >> 1));
	gl_Position = vec4(corner * 2.0 - 1.0, 0.0, 1.0);
	vDev = vec2(corner.x, 1.0 - corner.y) * uOutSize;
}
