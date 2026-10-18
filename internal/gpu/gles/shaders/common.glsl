// What every shader shares: the surface it draws on. Shaders include this file
// with an include line.

uniform vec2 uSurface; // the size of the framebuffer being drawn into, in pixels
uniform vec2 uOrigin;  // the pixel of the surface that the framebuffer's top left corner is:
                       // a layer draws into a framebuffer that covers a rectangle of the surface

// The clip space position of the point dev, in pixels from the top left of the surface.
vec4 toClip(vec2 dev) {
	vec2 p = dev - uOrigin;
	return vec4(p.x / uSurface.x * 2.0 - 1.0, 1.0 - p.y / uSurface.y * 2.0, 0.0, 1.0);
}
