// A vertex shader that reads instance_index and vertex_index: it can only be drawn with a first
// instance and a first vertex of 0 (SPEC §27.1).

@vertex
fn vs_main(@builtin(vertex_index) vertex: u32, @builtin(instance_index) instance: u32) -> @builtin(position) vec4<f32> {
	return vec4<f32>(f32(vertex), f32(instance), 0.0, 1.0);
}
