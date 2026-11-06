// Every kind of binding that an app's shader may have, and a compute entry point.

struct Frame {
	view: mat4x4<f32>,
	eye: vec3<f32>,
	time: f32,
	tint: vec4<f32>,
}

struct Light {
	direction: vec3<f32>,
	intensity: f32,
}

struct Particle {
	position: vec2<f32>,
	velocity: vec2<f32>,
}

struct Particles {
	count: u32,
	items: array<Particle>,
}

@group(0) @binding(0) var<uniform> frame: Frame;
@group(0) @binding(1) var<uniform> lights: array<Light, 4>;
@group(0) @binding(2) var color: texture_2d<f32>;
@group(0) @binding(3) var layers: texture_2d_array<f32>;
@group(0) @binding(4) var sky: texture_cube<f32>;
@group(0) @binding(5) var volume: texture_3d<f32>;
@group(0) @binding(6) var ids: texture_2d<u32>;
@group(0) @binding(7) var smp: sampler;

@group(1) @binding(0) var<storage, read> input: Particles;
@group(1) @binding(1) var<storage, read_write> output: Particles;
@group(1) @binding(2) var paint: texture_storage_2d<rgba8unorm, write>;
@group(1) @binding(3) var paint_layers: texture_storage_2d_array<rgba16float, write>;
@group(1) @binding(4) var field: texture_storage_3d<r32float, write>;

@fragment
fn fs_main(@location(0) uv: vec2<f32>) -> @location(0) vec4<f32> {
	let a = textureSample(color, smp, uv);
	let b = textureSample(layers, smp, uv, 1);
	let c = textureSample(sky, smp, vec3<f32>(uv, 1.0));
	let d = textureSample(volume, smp, vec3<f32>(uv, 0.5));
	let e = f32(textureLoad(ids, vec2<i32>(0, 0), 0).x);
	return (a + b + c + d) * frame.tint * lights[0].intensity + vec4<f32>(e * frame.time);
}

@compute @workgroup_size(8, 4)
fn cs_main(@builtin(global_invocation_id) id: vec3<u32>) {
	if (id.x >= input.count) {
		return;
	}
	var p = input.items[id.x];
	p.position = p.position + p.velocity * frame.time;
	output.items[id.x] = p;
	textureStore(paint, vec2<i32>(id.xy), frame.tint);
	textureStore(paint_layers, vec2<i32>(id.xy), 0, frame.tint);
	textureStore(field, vec3<i32>(id), vec4<f32>(frame.time));
}
