// The lines of viewport3d (SPEC §27.8): each segment is a quad that the vertex shader expands in the screen, so that the width
// is in logical pixels whatever the distance. The fragment shader anti-aliases the sides and the ends and cuts a dash
// pattern. A segment is an instance (its two ends in the mesh's units and the length of the polyline before it); the four
// corners of the quad come from a small vertex buffer: no built-in index is read.

struct Light {
	dir: vec4<f32>,
	color: vec4<f32>,
}

struct Frame {
	proj: mat4x4<f32>,
	lights: array<Light, 5>,
	ambient: vec4<f32>,
	viewport: vec4<f32>,
	// params are the near plane's distance, the device pixel ratio and two unused numbers.
	params: vec4<f32>,
}

struct LineMaterial {
	color: vec4<f32>,
	// params are: x the width in logical pixels, y the number of lengths of the dash pattern (0 for a solid line), z the
	// relative pull of the line towards the camera (so that an edge is not hidden by the face it lies on), w the sum of the
	// pattern.
	params: vec4<f32>,
	// pattern are the lengths in logical pixels: a dash, a gap, a dash, ...
	pattern: array<vec4<f32>, 2>,
}

struct Item {
	// mv takes the mesh's positions to the view, relative to the eye.
	mv: mat4x4<f32>,
}

@group(0) @binding(0) var<uniform> frame: Frame;
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
}

@vertex
fn vs_main(@location(0) p0: vec3<f32>, @location(1) p1: vec3<f32>, @location(2) d0: f32, @location(3) corner: vec2<f32>) -> VOut {
	var o: VOut;
	// A segment that is wholly behind the near plane is put outside the clip volume.
	o.clip = vec4<f32>(2.0, 2.0, 2.0, 1.0);
	let near = frame.params.x;
	let a = (item.mv * vec4<f32>(p0, 1.0)).xyz;
	let b = (item.mv * vec4<f32>(p1, 1.0)).xyz;
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
	let half_w = 0.5 * line.params.x * frame.params.y;
	let pad = half_w + 1.0;
	let at_end = corner.x;
	let sgn = at_end * 2.0 - 1.0;
	let offset = dir * (sgn * pad) + nrm * (corner.y * pad);
	let k = mix(ka, kb, at_end);
	o.clip = vec4<f32>(k.xy + offset / half_vp * k.w, k.z * (1.0 + line.params.z), k.w);
	o.across = corner.y * pad;
	o.along = at_end * span + sgn * pad;
	o.length_px = span;
	// The pattern runs in screen pixels: the pixels for a unit of the mesh at this segment, from its projected length.
	let seg_len = length(p1 - p0);
	let units = seg_len * (tb - ta);
	var per_unit = 0.0;
	if (units > 0.0) {
		per_unit = span / units;
	}
	o.stipple = (d0 + ta * seg_len) * per_unit + o.along;
	return o;
}

@fragment
fn fs_main(in: VOut) -> @location(0) vec4<f32> {
	let dpr = frame.params.y;
	let half_w = 0.5 * line.params.x * dpr;
	var cover = clamp(half_w + 0.5 - abs(in.across), 0.0, 1.0);
	cover = cover * clamp(in.along + half_w + 0.5, 0.0, 1.0) * clamp(in.length_px + half_w + 0.5 - in.along, 0.0, 1.0);
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
	let a = line.color.a * cover;
	return vec4<f32>(line.color.rgb * a, a);
}
