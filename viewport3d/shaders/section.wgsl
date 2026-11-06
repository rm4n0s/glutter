// The sections of viewport3d (SPEC §27.8): the cap that closes a solid mesh where a section plane cuts it, and the outline of the
// cut. A cap is drawn in three steps with the stencil, for each plane and each closed item: vs_count and fs_count draw the part
// of the item that the plane leaves, back faces adding one to the stencil and front faces taking one away, so that a pixel
// whose line of sight ends inside the solid, beyond the plane, is not zero; vs_full and fs_cap then draw the plane itself
// over the whole view, where the stencil is not zero, with the depth that the plane has at each pixel; and vs_full with
// fs_clear sets the stencil to zero again. The outline is a quad for each triangle that the plane crosses, made by
// vs_outline from the triangle's three corners.
//
// The record of the item has: tint the color of the cap, extra the color of the outline, and params the index of the plane in
// x, the width of the outline in logical pixels in y and the pull of the outline towards the camera in z.

//#include "common.wgsl.inc"
//#include "segment.wgsl.inc"

@group(1) @binding(0) var<uniform> item: Item;

struct Counted {
	@builtin(position) clip: vec4<f32>,
	@location(0) view_pos: vec3<f32>,
}

@vertex
fn vs_count(@location(0) p: vec3<f32>) -> Counted {
	var o: Counted;
	let v = item.mv * vec4<f32>(p, 1.0);
	o.view_pos = v.xyz;
	o.clip = frame.proj * v;
	return o;
}

@fragment
fn fs_count(in: Counted) -> @location(0) vec4<f32> {
	// Only what the plane leaves counts.
	let plane = frame.planes[i32(item.params.x)];
	if (dot(plane.xyz, in.view_pos) + plane.w > 0.0) {
		discard;
	}
	return vec4<f32>(0.0);
}

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

struct Capped {
	@location(0) color: vec4<f32>,
	@builtin(frag_depth) depth: f32,
}

@fragment
fn fs_cap(in: Full) -> Capped {
	var o: Capped;
	o.color = vec4<f32>(0.0);
	o.depth = 0.0;
	let plane = frame.planes[i32(item.params.x)];
	// The line of sight of the pixel: from the eye, or, in an orthographic view, parallel to the axis of the view.
	var origin = vec3<f32>(0.0);
	var dir = vec3<f32>(in.ndc.x / frame.proj[0][0], in.ndc.y / frame.proj[1][1], -1.0);
	if (frame.proj[3][3] > 0.5) {
		origin = vec3<f32>(dir.xy, 0.0);
		dir = vec3<f32>(0.0, 0.0, -1.0);
	}
	let facing = dot(plane.xyz, dir);
	if (abs(facing) < 1.0e-9) {
		discard;
	}
	let t = -(plane.w + dot(plane.xyz, origin)) / facing;
	let p = origin + dir * t;
	if (t <= 0.0 || section_clipped(p, i32(item.params.x))) {
		discard;
	}
	let k = frame.proj * vec4<f32>(p, 1.0);
	o.depth = k.z / k.w;
	o.color = vec4<f32>(item.tint.rgb, 1.0);
	return o;
}

@fragment
fn fs_clear(in: Full) -> @location(0) vec4<f32> {
	return vec4<f32>(0.0);
}

struct Outlined {
	@builtin(position) clip: vec4<f32>,
	@location(0) across: f32,
	@location(1) along: f32,
	@location(2) length_px: f32,
	@location(3) view_pos: vec3<f32>,
}

@vertex
fn vs_outline(@location(0) a: vec3<f32>, @location(1) b: vec3<f32>, @location(2) c: vec3<f32>, @location(3) corner: vec2<f32>) -> Outlined {
	var o: Outlined;
	o.clip = vec4<f32>(2.0, 2.0, 2.0, 1.0);
	let plane = frame.planes[i32(item.params.x)];
	let va = (item.mv * vec4<f32>(a, 1.0)).xyz;
	let vb = (item.mv * vec4<f32>(b, 1.0)).xyz;
	let vc = (item.mv * vec4<f32>(c, 1.0)).xyz;
	let da = dot(plane.xyz, va) + plane.w;
	let db = dot(plane.xyz, vb) + plane.w;
	let dc = dot(plane.xyz, vc) + plane.w;
	let sa = da > 0.0;
	let sb = db > 0.0;
	let sc = dc > 0.0;
	// The points where the edges that cross the plane cross it.
	var p0 = vec3<f32>(0.0);
	var p1 = vec3<f32>(0.0);
	var n = 0;
	if (sa != sb) {
		p0 = mix(va, vb, da / (da - db));
		n = 1;
	}
	if (sb != sc) {
		let p = mix(vb, vc, db / (db - dc));
		if (n == 0) {
			p0 = p;
		} else {
			p1 = p;
		}
		n = n + 1;
	}
	if (sc != sa) {
		let p = mix(vc, va, dc / (dc - da));
		if (n == 0) {
			p0 = p;
		} else {
			p1 = p;
		}
		n = n + 1;
	}
	if (n != 2) {
		return o;
	}
	let q = segment_quad(p0, p1, corner, item.params.y, item.params.z);
	o.clip = q.clip;
	o.across = q.across;
	o.along = q.along;
	o.length_px = q.length_px;
	o.view_pos = q.view_pos;
	return o;
}

@fragment
fn fs_outline(in: Outlined) -> @location(0) vec4<f32> {
	if (section_clipped(in.view_pos, i32(item.params.x))) {
		discard;
	}
	let cover = segment_cover(item.params.y, in.across, in.along, in.length_px);
	let a = item.extra.a * cover;
	return vec4<f32>(item.extra.rgb * a, a);
}
