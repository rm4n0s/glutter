// The picking of triangles in viewport3d (SPEC §27.8): a pick pass draws the items into a small target that has, for each pixel,
// the number of the item and of the triangle, and the depth. The triangles are drawn from a vertex buffer of their corners that
// no index reorders, so that vertex_index / 3 is the number of the triangle. The lines and the points have vs_pick and fs_pick
// in their own shaders.

//#include "common.wgsl.inc"

@group(1) @binding(0) var<uniform> item: Item;

struct Picked {
	@builtin(position) clip: vec4<f32>,
	@location(0) view_pos: vec3<f32>,
	@location(1) @interpolate(flat) prim: u32,
}

@vertex
fn vs_tri(@location(0) p: vec3<f32>, @builtin(vertex_index) i: u32) -> Picked {
	var o: Picked;
	let v = item.mv * vec4<f32>(p, 1.0);
	o.view_pos = v.xyz;
	o.clip = frame.proj * v;
	o.prim = i / 3u;
	return o;
}

@fragment
fn fs_tri(in: Picked) -> PickOut {
	if (section_clipped(in.view_pos, -1)) {
		discard;
	}
	var o: PickOut;
	o.ids = vec2<u32>(u32(item.params.x), in.prim);
	o.depth = in.clip.z;
	return o;
}
