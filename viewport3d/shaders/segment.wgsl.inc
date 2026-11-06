// The quad of a line segment (SPEC §27.8): the vertex shaders of the lines, of the picking of lines and of the outline of a
// section expand a segment into a quad in the screen with it, so that the width is in logical pixels whatever the distance.
// It needs common.wgsl.inc.

struct SegmentQuad {
	clip: vec4<f32>,
	// view_pos is the point of the segment that the corner is at, in view space.
	view_pos: vec3<f32>,
	// across is the distance in pixels from the center of the line, along: the distance from its first end, along it.
	across: f32,
	along: f32,
	length_px: f32,
	// ta and tb are the part of the segment in front of the near plane, from 0 to 1.
	ta: f32,
	tb: f32,
	visible: bool,
}

// segment_quad expands the segment from a to b (view space) at a corner of the quad: corner.x is 0 at the first end and 1 at the
// second, corner.y is -1 and 1 for the two sides. width is in logical pixels; pull moves the depth towards the camera by a
// part of itself (so that an edge is not hidden by the face it lies on).
fn segment_quad(a: vec3<f32>, b: vec3<f32>, corner: vec2<f32>, width: f32, pull: f32) -> SegmentQuad {
	var o: SegmentQuad;
	// A segment that is wholly behind the near plane is put outside the clip volume.
	o.clip = vec4<f32>(2.0, 2.0, 2.0, 1.0);
	o.visible = false;
	let near = frame.params.x;
	let za = -a.z;
	let zb = -b.z;
	if (za < near && zb < near) {
		return o;
	}
	// The part of the segment in front of the near plane.
	var ta = 0.0;
	var tb = 1.0;
	let cut = (near - za) / (zb - za);
	if (za < near) {
		ta = cut;
	}
	if (zb < near) {
		tb = cut;
	}
	let ca = mix(a, b, ta);
	let cb = mix(a, b, tb);
	let ka = frame.proj * vec4<f32>(ca, 1.0);
	let kb = frame.proj * vec4<f32>(cb, 1.0);
	let half_vp = 0.5 * frame.viewport.xy;
	let sa = ka.xy / ka.w * half_vp;
	let sb = kb.xy / kb.w * half_vp;
	var dir = sb - sa;
	let span = length(dir);
	if (span > 0.0001) {
		dir = dir / span;
	} else {
		dir = vec2<f32>(1.0, 0.0);
	}
	let nrm = vec2<f32>(-dir.y, dir.x);
	let half_w = 0.5 * width * frame.params.y;
	let pad = half_w + 1.0;
	let at_end = corner.x;
	let sgn = at_end * 2.0 - 1.0;
	let offset = dir * (sgn * pad) + nrm * (corner.y * pad);
	let k = mix(ka, kb, at_end);
	o.clip = vec4<f32>(k.xy + offset / half_vp * k.w, k.z * (1.0 + pull), k.w);
	o.view_pos = mix(ca, cb, at_end);
	o.across = corner.y * pad;
	o.along = at_end * span + sgn * pad;
	o.length_px = span;
	o.ta = ta;
	o.tb = tb;
	o.visible = true;
	return o;
}

// segment_cover is how much of a pixel of a quad of a segment the line covers, from 0 to 1, with anti-aliased sides and ends.
fn segment_cover(width: f32, across: f32, along: f32, length_px: f32) -> f32 {
	let half_w = 0.5 * width * frame.params.y;
	var cover = clamp(half_w + 0.5 - abs(across), 0.0, 1.0);
	cover = cover * clamp(along + half_w + 0.5, 0.0, 1.0) * clamp(length_px + half_w + 0.5 - along, 0.0, 1.0);
	return cover;
}
