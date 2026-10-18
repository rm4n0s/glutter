#version 300 es
precision mediump float;

// The texture to copy, on texture unit 0.
uniform sampler2D uTexture;

in vec2 vUV;
out vec4 fragColor;

void main() {
	fragColor = texture(uTexture, vUV);
}
