// The outline of the selection in viewport3d (SPEC §27.8): the selected items that have faces are drawn white into a mask the size
// of the view, and fs_outline, over a rectangle around them, colors each pixel that is not in the mask and has a pixel of it within
// the radius, with an anti-aliased edge: a line of the radius's width around the silhouette.
//
// The record has the color of the outline in tint and the radius in pixels in params.x.

//#include "common.wgsl.inc"

@group(1) @binding(0) var<uniform> item: Item;
@group(1) @binding(1) var mask: texture_2d<f32>;

struct Full {
	@builtin(position) clip: vec4<f32>,
}

@vertex
fn vs_full(@builtin(vertex_index) i: u32) -> Full {
	// One triangle that covers the view: (-1, -1), (3, -1) and (-1, 3).
	let x = f32((i << 1u) & 2u);
	let y = f32(i & 2u);
	var o: Full;
	o.clip = vec4<f32>(x * 2.0 - 1.0, y * 2.0 - 1.0, 0.0, 1.0);
	return o;
}

@fragment
fn fs_outline(@builtin(position) pos: vec4<f32>) -> @location(0) vec4<f32> {
	let size = vec2<i32>(frame.viewport.xy);
	let here = vec2<i32>(pos.xy);
	if (textureLoad(mask, here, 0).r > 0.5) {
		discard;
	}
	let radius = i32(item.params.x);
	var nearest = 1.0e9;
	for (var dy = -radius; dy <= radius; dy = dy + 1) {
		for (var dx = -radius; dx <= radius; dx = dx + 1) {
			let q = here + vec2<i32>(dx, dy);
			if (q.x < 0 || q.y < 0 || q.x >= size.x || q.y >= size.y) {
				continue;
			}
			if (textureLoad(mask, q, 0).r > 0.5) {
				nearest = min(nearest, f32(dx * dx + dy * dy));
			}
		}
	}
	if (nearest > 1.0e8) {
		discard;
	}
	let cover = clamp(f32(radius) + 0.5 - sqrt(nearest), 0.0, 1.0);
	let a = item.tint.a * cover;
	return vec4<f32>(item.tint.rgb * a, a);
}
