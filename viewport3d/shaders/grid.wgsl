// The grid of viewport3d (SPEC §27.8, "Overlays"): an infinite grid on a plane, with thin minor lines and thicker major ones that
// fade away with the distance. One triangle covers the view; for each pixel the fragment shader meets the line of sight with the
// plane, the way fs_cap of section.wgsl does, works out where that point is in the plane's own coordinates, and covers the pixel
// by its distance to the nearest line of each kind, in pixels (the derivatives of the coordinates), so that the lines have the
// same width everywhere and fade out where they come closer than a few pixels, instead of turning to noise.
//
// The plane's coordinates are the point's position in the plane's two axes, in float64 on the CPU for the eye's part: the
// shader adds the eye's coordinates (reduced to the period of the major lines) to the position relative to the eye (D30), so a
// grid 50 km from the origin has lines as sharp as one at the origin.
//
// The record is not an Item: it is seven vec4 (the size of an Item).

//#include "common.wgsl.inc"

struct GridItem {
	// plane is the plane in view space: xyz its unit normal, w its distance (a point p is on it when dot(xyz, p) + w = 0).
	plane: vec4<f32>,
	// u and v are the plane's axes in view space; their w is the eye's coordinate along the axis, reduced to the major period.
	u: vec4<f32>,
	v: vec4<f32>,
	// params are the spacing of the minor lines, how many of them make a major line, and the distances where the lines start to
	// fade and where they are gone.
	params: vec4<f32>,
	minor: vec4<f32>,
	major: vec4<f32>,
	// extra has the width of the lines in pixels in x.
	extra: vec4<f32>,
}

@group(1) @binding(0) var<uniform> item: GridItem;

struct Full {
	@builtin(position) clip: vec4<f32>,
	@location(0) ndc: vec2<f32>,
}

@vertex
fn vs_full(@builtin(vertex_index) i: u32) -> Full {
	// One triangle that covers the view: (-1, -1), (3, -1) and (-1, 3).
	let x = f32((i << 1u) & 2u);
	let y = f32(i & 2u);
	var o: Full;
	o.ndc = vec2<f32>(x * 2.0 - 1.0, y * 2.0 - 1.0);
	o.clip = vec4<f32>(o.ndc, 0.0, 1.0);
	return o;
}

struct Gridded {
	@location(0) color: vec4<f32>,
	@builtin(frag_depth) depth: f32,
}

// cover is how much of the pixel a line covers: the distance in pixels of the coordinate from the nearest whole number, and the
// width of the line. The line is gone where the cells are smaller than a few pixels.
fn cover(coord: vec2<f32>, width: f32) -> f32 {
	let fw = max(fwidth(coord), vec2<f32>(1.0e-6));
	let dist = abs(fract(coord - 0.5) - 0.5) / fw;
	let line = clamp(0.5 * width + 0.5 - dist, vec2<f32>(0.0), vec2<f32>(1.0));
	// The size of a cell in pixels, along each axis.
	let cell = 1.0 / fw;
	let visible = clamp((cell - 3.0) / 5.0, vec2<f32>(0.0), vec2<f32>(1.0));
	return max(line.x * visible.x, line.y * visible.y);
}

@fragment
fn fs_grid(in: Full) -> Gridded {
	var o: Gridded;
	o.color = vec4<f32>(0.0);
	o.depth = 0.0;
	// The line of sight of the pixel: from the eye, or, in an orthographic view, parallel to the axis of the view.
	var origin = vec3<f32>(0.0);
	var dir = vec3<f32>(in.ndc.x / frame.proj[0][0], in.ndc.y / frame.proj[1][1], -1.0);
	let ortho = frame.proj[3][3] > 0.5;
	if (ortho) {
		origin = vec3<f32>(dir.xy, 0.0);
		dir = vec3<f32>(0.0, 0.0, -1.0);
	}
	let facing = dot(item.plane.xyz, dir);
	// Everything below is worked out for every pixel, and the pixels that the plane does not reach are dropped at the end, so
	// that the derivatives are taken in uniform control flow.
	let ok = abs(facing) > 1.0e-9;
	let t = select(0.0, -(item.plane.w + dot(item.plane.xyz, origin)) / facing, ok);
	let p = origin + dir * t;
	let hit = ok && t > 0.0;
	let spacing = item.params.x;
	let major_every = item.params.y;
	let u = dot(p, item.u.xyz) + item.u.w;
	let v = dot(p, item.v.xyz) + item.v.w;
	let width = item.extra.x;
	let minor_cover = cover(vec2<f32>(u, v) / spacing, width);
	let major_cover = cover(vec2<f32>(u, v) / (spacing * major_every), 2.0 * width);
	// Fade with the distance, and where the plane is seen edge on, where the lines would only be a haze.
	var fade = 1.0;
	if (!ortho) {
		fade = 1.0 - smoothstep(item.params.z, item.params.w, length(p));
	}
	let edge_on = smoothstep(0.0, 0.2, abs(facing) / length(dir));
	let a_minor = minor_cover * item.minor.a * fade * edge_on;
	let a_major = major_cover * item.major.a * fade * edge_on;
	// The major lines are over the minor ones.
	let rgb = item.major.rgb * a_major + item.minor.rgb * a_minor * (1.0 - a_major);
	let a = a_major + a_minor * (1.0 - a_major);
	if (!hit || a < 0.003 || section_clipped(p, -1)) {
		discard;
	}
	let k = frame.proj * vec4<f32>(p, 1.0);
	o.depth = k.z / k.w;
	o.color = vec4<f32>(rgb, a);
	return o;
}
