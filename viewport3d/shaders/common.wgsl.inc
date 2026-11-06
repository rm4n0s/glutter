// What the shaders of viewport3d share (SPEC §27.8): the frame's uniform data, the record of an item, and the section planes.

struct Light {
	// dir points to the light, in view space.
	dir: vec4<f32>,
	// color is the light's color times its intensity.
	color: vec4<f32>,
}

struct Frame {
	proj: mat4x4<f32>,
	// lights are the light at the camera, then up to four directional ones; an unused one has a black color.
	lights: array<Light, 5>,
	ambient: vec4<f32>,
	// viewport is the width and the height of the target in pixels, and their inverses.
	viewport: vec4<f32>,
	// params are the near plane's distance, the device pixel ratio and two unused numbers.
	params: vec4<f32>,
	// planes are the section planes in view space, relative to the eye (D30): a point p with dot(plane.xyz, p) + plane.w > 0 is
	// cut away.
	planes: array<vec4<f32>, 6>,
	// section is the number of planes in x.
	section: vec4<f32>,
}

@group(0) @binding(0) var<uniform> frame: Frame;

// Item is the record of a line, a point or a picked item: the matrix that takes the mesh's positions to the view, relative to
// the eye, a tint (the highlight of a selected item: its color, and the part of it that replaces the material's in a), and
// numbers for the shader that reads it.
struct Item {
	mv: mat4x4<f32>,
	tint: vec4<f32>,
	params: vec4<f32>,
	extra: vec4<f32>,
}

// section_clipped says whether a point of the view space is cut away by a section plane. skip is the plane to leave out (the
// plane that a cap lies on), or -1.
fn section_clipped(p: vec3<f32>, skip: i32) -> bool {
	let count = i32(frame.section.x);
	for (var i = 0; i < 6; i = i + 1) {
		if (i >= count) {
			break;
		}
		if (i == skip) {
			continue;
		}
		let plane = frame.planes[i];
		if (dot(plane.xyz, p) + plane.w > 0.0) {
			return true;
		}
	}
	return false;
}

// PickOut is what a pick pass writes for a pixel: the number of the item (from 1, 0 for nothing) and of the primitive in it, and
// the depth.
struct PickOut {
	@location(0) ids: vec2<u32>,
	@location(1) depth: f32,
}
