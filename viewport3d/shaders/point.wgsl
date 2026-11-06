// The points of viewport3d (SPEC §27.8): each point is a quad of a size in logical pixels that the vertex shader makes in the
// screen, and the fragment shader cuts it to a disc or a square with an anti-aliased edge. A point is an instance (its position
// in the mesh's units); the corners of the quad come from a small vertex buffer.

//#include "common.wgsl.inc"

struct PointMaterial {
	color: vec4<f32>,
	// params are: x the size in logical pixels, y 0 for round points and 1 for square ones.
	params: vec4<f32>,
}

@group(1) @binding(0) var<uniform> point: PointMaterial;
@group(2) @binding(0) var<uniform> item: Item;

struct VOut {
	@builtin(position) clip: vec4<f32>,
	// local is the position in pixels from the center of the point.
	@location(0) local: vec2<f32>,
	@location(1) view_pos: vec3<f32>,
	// index is the number of the point in its mesh, which the picking reports.
	@location(2) @interpolate(flat) index: f32,
}

@vertex
fn vs_main(@location(0) p: vec3<f32>, @location(1) corner: vec2<f32>) -> VOut {
	return vertex_point(p, corner, 0.0);
}

// vs_pick is vs_main for the picking: the point also has its number.
@vertex
fn vs_pick(@location(0) p: vec3<f32>, @location(1) corner: vec2<f32>, @location(2) index: f32) -> VOut {
	return vertex_point(p, corner, index);
}

fn vertex_point(p: vec3<f32>, corner: vec2<f32>, index: f32) -> VOut {
	var o: VOut;
	o.index = index;
	o.clip = vec4<f32>(2.0, 2.0, 2.0, 1.0);
	let v = item.mv * vec4<f32>(p, 1.0);
	if (-v.z < frame.params.x) {
		return o;
	}
	let k = frame.proj * v;
	let half_size = 0.5 * point.params.x * frame.params.y + 1.0;
	let offset = corner * half_size / (0.5 * frame.viewport.xy);
	o.clip = vec4<f32>(k.xy + offset * k.w, k.z, k.w);
	o.local = corner * half_size;
	o.view_pos = v.xyz;
	return o;
}

@fragment
fn fs_main(in: VOut) -> @location(0) vec4<f32> {
	if (section_clipped(in.view_pos, -1)) {
		discard;
	}
	let half_size = 0.5 * point.params.x * frame.params.y;
	var cover = 0.0;
	if (point.params.y > 0.5) {
		cover = clamp(half_size + 0.5 - abs(in.local.x), 0.0, 1.0) * clamp(half_size + 0.5 - abs(in.local.y), 0.0, 1.0);
	} else {
		cover = clamp(half_size + 0.5 - length(in.local), 0.0, 1.0);
	}
	let rgb = mix(point.color.rgb, item.tint.rgb, item.tint.a);
	let a = point.color.a * cover;
	return vec4<f32>(rgb * a, a);
}

@fragment
fn fs_pick(in: VOut) -> PickOut {
	if (section_clipped(in.view_pos, -1)) {
		discard;
	}
	let half_size = 0.5 * point.params.x * frame.params.y;
	var cover = 0.0;
	if (point.params.y > 0.5) {
		cover = clamp(half_size + 0.5 - abs(in.local.x), 0.0, 1.0) * clamp(half_size + 0.5 - abs(in.local.y), 0.0, 1.0);
	} else {
		cover = clamp(half_size + 0.5 - length(in.local), 0.0, 1.0);
	}
	if (cover < 0.5) {
		discard;
	}
	var o: PickOut;
	o.ids = vec2<u32>(u32(item.params.x), u32(in.index + 0.5));
	o.depth = in.clip.z;
	return o;
}
