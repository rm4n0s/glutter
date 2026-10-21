// Solid rectangles, one instance each (SPEC §9.3). Ported from the GLES solid
// shader (solid.vert.glsl, solid.frag.glsl) and common.glsl's toClip, with the
// rectangle and the color per instance, as the GLES renderer's instanced quads.
//
// Draw 6 vertices per instance: two triangles made from the vertex index.

struct Frame {
	// The size of the texture being drawn into, in pixels.
	surface: vec2<f32>,
	// The pixel of the surface that the texture's top left corner is: a layer draws
	// into a texture that covers a rectangle of the surface.
	origin: vec2<f32>,
}

// The uniform buffer is bound with a dynamic offset.
@group(0) @binding(0) var<uniform> frame: Frame;

struct Instance {
	// x, y, width and height in pixels from the top left of the surface.
	@location(0) rect: vec4<f32>,
	// The color to fill with, premultiplied.
	@location(1) color: vec4<f32>,
}

struct VertexOutput {
	@builtin(position) position: vec4<f32>,
	@location(0) color: vec4<f32>,
}

// The clip space position of the point dev, in pixels from the top left of the surface.
fn toClip(dev: vec2<f32>) -> vec4<f32> {
	let p = dev - frame.origin;
	return vec4<f32>(p.x / frame.surface.x * 2.0 - 1.0, 1.0 - p.y / frame.surface.y * 2.0, 0.0, 1.0);
}

@vertex
fn vs_main(@builtin(vertex_index) index: u32, instance: Instance) -> VertexOutput {
	// The corners of the two triangles: (0,0) (1,0) (0,1) and (1,0) (1,1) (0,1).
	var corners = array<vec2<f32>, 6>(
		vec2<f32>(0.0, 0.0), vec2<f32>(1.0, 0.0), vec2<f32>(0.0, 1.0),
		vec2<f32>(1.0, 0.0), vec2<f32>(1.0, 1.0), vec2<f32>(0.0, 1.0),
	);
	let corner = corners[index];
	var out: VertexOutput;
	out.position = toClip(instance.rect.xy + corner * instance.rect.zw);
	out.color = instance.color;
	return out;
}

@fragment
fn fs_main(in: VertexOutput) -> @location(0) vec4<f32> {
	return in.color;
}
