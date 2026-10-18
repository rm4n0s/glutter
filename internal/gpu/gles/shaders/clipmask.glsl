// The coverage of the clip shapes that are not rectangles, drawn into a texture
// by the renderer: one 8-bit coverage per pixel of a rectangle of the surface.
// Shaders include this file with an include line.

uniform sampler2D uClipMask;  // texture unit 1
uniform vec4 uClipMaskRect;   // x, y, width, height in pixels; width 0 means no mask

// The coverage of the pixel whose center is dev (in pixels from the top left).
float clipMaskCoverage(vec2 dev) {
	if (uClipMaskRect.z <= 0.0) {
		return 1.0;
	}
	ivec2 p = ivec2(floor(dev)) - ivec2(uClipMaskRect.xy);
	if (p.x < 0 || p.y < 0 || p.x >= int(uClipMaskRect.z) || p.y >= int(uClipMaskRect.w)) {
		return 0.0;
	}
	return texelFetch(uClipMask, p, 0).r;
}
