#version 300 es
precision mediump float;

// The color to fill with, premultiplied.
uniform vec4 uColor;

out vec4 fragColor;

void main() {
	fragColor = uColor;
}
