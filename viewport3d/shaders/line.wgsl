// The lines of viewport3d (SPEC §27.8): each segment is a quad that the vertex shader expands in the screen, so that the width
// is in logical pixels whatever the distance. The fragment shader anti-aliases the sides and the ends and cuts a dash
// pattern. A segment is an instance (its two ends in the mesh's units and the length of the polyline before it); the four
// corners of the quad come from a small vertex buffer: no built-in index is read.

//#include "common.wgsl.inc"
//#include "segment.wgsl.inc"

struct LineMaterial {
	color: vec4<f32>,
	// params are: x the width in logical pixels, y the number of lengths of the dash pattern (0 for a solid line), z the
	// relative pull of the line towards the camera (so that an edge is not hidden by the face it lies on), w the sum of the
	// pattern.
	params: vec4<f32>,
	// pattern are the lengths in logical pixels: a dash, a gap, a dash, ...
	pattern: array<vec4<f32>, 2>,
}

@group(1) @binding(0) var<uniform> line: LineMaterial;
@group(2) @binding(0) var<uniform> item: Item;

struct VOut {
	@builtin(position) clip: vec4<f32>,
	// across is the distance in pixels from the center of the line, along: the distance from its first end, along it.
	@location(0) across: f32,
	@location(1) along: f32,
	@location(2) length_px: f32,
	// stipple is the position along the dash pattern, in pixels.
	@location(3) stipple: f32,
	@location(4) view_pos: vec3<f32>,
	// seg is the number of the segment in its mesh, which the picking reports.
	@location(5) @interpolate(flat) seg: f32,
}

@vertex
fn vs_main(@location(0) p0: vec3<f32>, @location(1) p1: vec3<f32>, @location(2) d0: f32, @location(3) corner: vec2<f32>) -> VOut {
	return vertex_line(p0, p1, d0, corner, 0.0);
}

// vs_pick is vs_main for the picking: the segment also has its number.
@vertex
fn vs_pick(@location(0) p0: vec3<f32>, @location(1) p1: vec3<f32>, @location(2) d0: f32, @location(3) corner: vec2<f32>, @location(4) seg: f32) -> VOut {
	return vertex_line(p0, p1, d0, corner, seg);
}

fn vertex_line(p0: vec3<f32>, p1: vec3<f32>, d0: f32, corner: vec2<f32>, seg: f32) -> VOut {
	var o: VOut;
	o.seg = seg;
	let a = (item.mv * vec4<f32>(p0, 1.0)).xyz;
	let b = (item.mv * vec4<f32>(p1, 1.0)).xyz;
	let q = segment_quad(a, b, corner, line.params.x, line.params.z);
	o.clip = q.clip;
	if (!q.visible) {
		return o;
	}
	o.across = q.across;
	o.along = q.along;
	o.length_px = q.length_px;
	o.view_pos = q.view_pos;
	// The pattern runs in screen pixels: the pixels for a unit of the mesh at this segment, from its projected length.
	let seg_len = length(p1 - p0);
	let units = seg_len * (q.tb - q.ta);
	var per_unit = 0.0;
	if (units > 0.0) {
		per_unit = q.length_px / units;
	}
	o.stipple = (d0 + q.ta * seg_len) * per_unit + o.along;
	return o;
}

@fragment
fn fs_main(in: VOut) -> @location(0) vec4<f32> {
	if (section_clipped(in.view_pos, -1)) {
		discard;
	}
	let dpr = frame.params.y;
	var cover = segment_cover(line.params.x, in.across, in.along, in.length_px);
	let count = i32(line.params.y);
	if (count > 0) {
		let total = line.params.w * dpr;
		let m = in.stipple - floor(in.stipple / total) * total;
		var acc = 0.0;
		var dash = 1.0;
		for (var i = 0; i < 8; i = i + 1) {
			if (i >= count) {
				break;
			}
			let piece = line.pattern[i / 4][i % 4] * dpr;
			if (m < acc + piece) {
				let inside = min(m - acc, acc + piece - m);
				if (i % 2 == 0) {
					dash = clamp(0.5 + inside, 0.0, 1.0);
				} else {
					dash = clamp(0.5 - inside, 0.0, 1.0);
				}
				break;
			}
			acc = acc + piece;
		}
		cover = cover * dash;
	}
	let rgb = mix(line.color.rgb, item.tint.rgb, item.tint.a);
	let a = line.color.a * cover;
	return vec4<f32>(rgb * a, a);
}

@fragment
fn fs_pick(in: VOut) -> PickOut {
	if (section_clipped(in.view_pos, -1)) {
		discard;
	}
	if (segment_cover(line.params.x, in.across, in.along, in.length_px) < 0.5) {
		discard;
	}
	var o: PickOut;
	o.ids = vec2<u32>(u32(item.params.x), u32(in.seg + 0.5));
	o.depth = in.clip.z;
	return o;
}
