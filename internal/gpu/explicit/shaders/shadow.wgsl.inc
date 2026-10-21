// The coverage of a rounded rectangle blurred by a Gaussian, in closed form (SPEC §9.1). Ported
// from gles/shaders/shadow.glsl. Shaders include this file with an include line.
//
// Across a row of the rectangle, the blur of an interval of columns is a difference of two
// normal distribution functions. Down the rows the Gaussian is integrated: exactly where the
// rows do not change (the straight part of the sides), and by a sum of samples over the rows of
// the rounded corners.

// The error function (Abramowitz and Stegun 7.1.26, error below 1.5e-7).
fn erfApprox(x0: f32) -> f32 {
	let s = select(1.0, -1.0, x0 < 0.0);
	let x = abs(x0);
	let t = 1.0 / (1.0 + 0.3275911 * x);
	let y = 1.0 - (((((1.061405429 * t - 1.453152027) * t) + 1.421413741) * t - 0.284496736) * t + 0.254829592) * t * exp(-x * x);
	return s * y;
}

// The normal distribution function.
fn normalCdf(x: f32) -> f32 {
	return 0.5 + 0.5 * erfApprox(x * 0.7071067811865476);
}

// How far a corner of radii rad pulls the side in at the row y, with y measured from the center
// of the rectangle: top is true for the corners at the top.
fn cornerInset(rad: vec2<f32>, y: f32, hy: f32, top: bool) -> f32 {
	if rad.y <= 0.0 || rad.x <= 0.0 {
		return 0.0;
	}
	let cy = select(hy - rad.y, -hy + rad.y, top);
	let t = select((y - cy) / rad.y, (cy - y) / rad.y, top);
	if t <= 0.0 {
		return 0.0;
	}
	return rad.x * (1.0 - sqrt(max(1.0 - t * t, 0.0)));
}

// The weight of the rows of a corner zone, [y0, y1] from the center, as seen from the pixel at p:
// the sum over the rows of the Gaussian times the columns' share.
fn zoneMass(y0: f32, y1: f32, p: vec2<f32>, h: vec2<f32>, r1: vec4<f32>, r2: vec4<f32>, sigma: f32, top: bool) -> f32 {
	let a = max(y0, p.y - 4.0 * sigma);
	let b = min(y1, p.y + 4.0 * sigma);
	if b <= a {
		return 0.0;
	}
	let dv = (b - a) / 16.0;
	var sum = 0.0;
	for (var i = 0; i < 16; i++) {
		let y = a + (f32(i) + 0.5) * dv;
		let l = -h.x + cornerInset(select(r2.zw, r1.xy, top), y, h.y, top);
		let r = h.x - cornerInset(select(r2.xy, r1.zw, top), y, h.y, top);
		let w = exp(-0.5 * (y - p.y) * (y - p.y) / (sigma * sigma));
		sum += w * max(normalCdf((r - p.x) / sigma) - normalCdf((l - p.x) / sigma), 0.0);
	}
	return sum * dv / (sigma * 2.5066282746310002);
}

// The blurred coverage at the pixel whose position from the rectangle's center is p.
fn blurredRRect(p: vec2<f32>, h: vec2<f32>, r1: vec4<f32>, r2: vec4<f32>, sigma: f32) -> f32 {
	let topH = max(r1.y, r1.w);
	let botH = max(r2.y, r2.w);
	var mass = zoneMass(-h.y, -h.y + topH, p, h, r1, r2, sigma, true);
	mass += zoneMass(h.y - botH, h.y, p, h, r1, r2, sigma, false);
	let m0 = -h.y + topH;
	let m1 = h.y - botH;
	if m1 > m0 {
		let rows = normalCdf((m1 - p.y) / sigma) - normalCdf((m0 - p.y) / sigma);
		let cols = normalCdf((h.x - p.x) / sigma) - normalCdf((-h.x - p.x) / sigma);
		mass += rows * cols;
	}
	return clamp(mass, 0.0, 1.0);
}

// The coverage of a blur style, from the blurred coverage b and the shape's own o.
fn blurStyle(style: i32, b: f32, o: f32) -> f32 {
	if style == 1 {
		return o + b * (1.0 - o); // solid
	}
	if style == 2 {
		return b * (1.0 - o); // outer
	}
	if style == 3 {
		return b * o; // inner
	}
	return b; // normal
}
