// A vertex shader with inputs of several types, directly and in a struct.

struct Instance {
	@location(4) offset: vec3<f32>,
	@location(5) color: vec4<u32>,
	@location(6) flags: u32,
	@location(7) shift: vec2<i32>,
}

struct Out {
	@builtin(position) position: vec4<f32>,
	@location(0) color: vec4<f32>,
}

@vertex
fn vs_main(
	@location(0) position: vec2<f32>,
	@location(1) uv: vec3<f32>,
	@location(2) weight: f32,
	@location(3) normal: vec4<f32>,
	inst: Instance,
) -> Out {
	var out: Out;
	out.position = vec4<f32>(position + inst.offset.xy + vec2<f32>(inst.shift), uv.z * weight, 1.0);
	out.color = vec4<f32>(inst.color) * normal * f32(inst.flags);
	return out;
}

@fragment
fn fs_main(in: Out) -> @location(0) vec4<f32> {
	return in.color;
}
