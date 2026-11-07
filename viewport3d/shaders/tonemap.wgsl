// The tone mapping of viewport3d (SPEC §28.5): the frame was drawn as linear radiance into a float target (the PBR materials
// make values above 1), and this pass turns it into the colors of the display: the exposure, a curve (the Khronos PBR Neutral
// tone mapper, which leaves the colors below 0.8 as they are, or the ACES fit of Stephen Hill), and the sRGB transfer function.
// The input is premultiplied by the alpha; the output is too.

struct Tone {
	// params: x the exposure, y the curve (0 none, 1 PBR Neutral, 2 ACES).
	params: vec4<f32>,
}

@group(0) @binding(0) var<uniform> tone: Tone;
@group(0) @binding(1) var hdr: texture_2d<f32>;

@vertex
fn vs_full(@builtin(vertex_index) i: u32) -> @builtin(position) vec4<f32> {
	let x = f32((i << 1u) & 2u);
	let y = f32(i & 2u);
	return vec4<f32>(x * 2.0 - 1.0, 1.0 - y * 2.0, 0.0, 1.0);
}

// neutral is the Khronos PBR Neutral tone mapper (github.com/KhronosGroup/ToneMapping): the colors under about 0.76 keep
// their value, and the brighter ones are compressed towards white and desaturated.
fn neutral(color: vec3<f32>) -> vec3<f32> {
	let start_compression = 0.8 - 0.04;
	let desaturation = 0.15;
	let x = min(color.r, min(color.g, color.b));
	var offset = 0.04;
	if (x < 0.08) {
		offset = x - 6.25 * x * x;
	}
	var c = color - vec3<f32>(offset);
	let peak = max(c.r, max(c.g, c.b));
	if (peak < start_compression) {
		return c;
	}
	let d = 1.0 - start_compression;
	let new_peak = 1.0 - d * d / (peak + d - start_compression);
	c = c * (new_peak / peak);
	let g = 1.0 - 1.0 / (desaturation * (peak - new_peak) + 1.0);
	return mix(c, vec3<f32>(new_peak), g);
}

fn rrt_and_odt_fit(v: vec3<f32>) -> vec3<f32> {
	let a = v * (v + vec3<f32>(0.0245786)) - vec3<f32>(0.000090537);
	let b = v * (0.983729 * v + vec3<f32>(0.4329510)) + vec3<f32>(0.238081);
	return a / b;
}

// aces is the fit of the ACES reference rendering transform and output transform by Stephen Hill (BakingLab).
fn aces(color: vec3<f32>) -> vec3<f32> {
	let input_mat = mat3x3<f32>(vec3<f32>(0.59719, 0.07600, 0.02840), vec3<f32>(0.35458, 0.90834, 0.13383), vec3<f32>(0.04823, 0.01566, 0.83777));
	let output_mat = mat3x3<f32>(vec3<f32>(1.60475, -0.10208, -0.00327), vec3<f32>(-0.53108, 1.10813, -0.07276), vec3<f32>(-0.07367, -0.00605, 1.07602));
	return clamp(output_mat * rrt_and_odt_fit(input_mat * color), vec3<f32>(0.0), vec3<f32>(1.0));
}

fn linear_to_srgb(c: vec3<f32>) -> vec3<f32> {
	let lo = c * 12.92;
	let hi = 1.055 * pow(c, vec3<f32>(1.0 / 2.4)) - vec3<f32>(0.055);
	return select(hi, lo, c <= vec3<f32>(0.0031308));
}

@fragment
fn fs_tonemap(@builtin(position) pos: vec4<f32>) -> @location(0) vec4<f32> {
	let px = textureLoad(hdr, vec2<i32>(pos.xy), 0);
	var rgb = vec3<f32>(0.0);
	if (px.a > 0.0) {
		rgb = max(px.rgb / px.a, vec3<f32>(0.0));
	}
	rgb = rgb * tone.params.x;
	let mode = u32(tone.params.y);
	if (mode == 1u) {
		rgb = neutral(rgb);
	} else if (mode == 2u) {
		// The fit is made for an exposure that is 1/0.6 of the one of the other curves.
		rgb = aces(rgb / 0.6);
	}
	rgb = linear_to_srgb(clamp(rgb, vec3<f32>(0.0), vec3<f32>(1.0)));
	return vec4<f32>(rgb * px.a, px.a);
}
