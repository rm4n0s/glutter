// The coverage of a rounded rectangle blurred by a Gaussian, in closed form
// (SPEC §9.1). Shaders include this file with an include line.
//
// Across a row of the rectangle, the blur of an interval of columns is a difference of
// two normal distribution functions. Down the rows the Gaussian is integrated: exactly
// where the rows do not change (the straight part of the sides), and by a sum of
// samples over the rows of the rounded corners.

// The error function (Abramowitz and Stegun 7.1.26, error below 1.5e-7).
float erfApprox(float x) {
	float s = x < 0.0 ? -1.0 : 1.0;
	x = abs(x);
	float t = 1.0 / (1.0 + 0.3275911 * x);
	float y = 1.0 - (((((1.061405429 * t - 1.453152027) * t) + 1.421413741) * t - 0.284496736) * t + 0.254829592) * t * exp(-x * x);
	return s * y;
}

// The normal distribution function.
float normalCdf(float x) {
	return 0.5 + 0.5 * erfApprox(x * 0.7071067811865476);
}

// How far a corner of radii rad pulls the side in at the row y, with y measured from
// the center of the rectangle: top is true for the corners at the top.
float cornerInset(vec2 rad, float y, float hy, bool top) {
	if (rad.y <= 0.0 || rad.x <= 0.0) {
		return 0.0;
	}
	float cy = top ? -hy + rad.y : hy - rad.y;
	float t = top ? (cy - y) / rad.y : (y - cy) / rad.y;
	if (t <= 0.0) {
		return 0.0;
	}
	return rad.x * (1.0 - sqrt(max(1.0 - t * t, 0.0)));
}

// The weight of the rows of a corner zone, [y0, y1] from the center, as seen from the
// pixel at p: the sum over the rows of the Gaussian times the columns' share.
float zoneMass(float y0, float y1, vec2 p, vec2 h, vec4 r1, vec4 r2, float sigma, bool top) {
	float a = max(y0, p.y - 4.0 * sigma);
	float b = min(y1, p.y + 4.0 * sigma);
	if (b <= a) {
		return 0.0;
	}
	const int N = 16;
	float dv = (b - a) / float(N);
	float sum = 0.0;
	for (int i = 0; i < N; i++) {
		float y = a + (float(i) + 0.5) * dv;
		float l = -h.x + cornerInset(top ? r1.xy : r2.zw, y, h.y, top);
		float r = h.x - cornerInset(top ? r1.zw : r2.xy, y, h.y, top);
		float w = exp(-0.5 * (y - p.y) * (y - p.y) / (sigma * sigma));
		sum += w * max(normalCdf((r - p.x) / sigma) - normalCdf((l - p.x) / sigma), 0.0);
	}
	return sum * dv / (sigma * 2.5066282746310002);
}

// The blurred coverage at the pixel whose position from the rectangle's center is p.
float blurredRRect(vec2 p, vec2 h, vec4 r1, vec4 r2, float sigma) {
	float topH = max(r1.y, r1.w);
	float botH = max(r2.y, r2.w);
	float mass = zoneMass(-h.y, -h.y + topH, p, h, r1, r2, sigma, true);
	mass += zoneMass(h.y - botH, h.y, p, h, r1, r2, sigma, false);
	float m0 = -h.y + topH;
	float m1 = h.y - botH;
	if (m1 > m0) {
		float rows = normalCdf((m1 - p.y) / sigma) - normalCdf((m0 - p.y) / sigma);
		float cols = normalCdf((h.x - p.x) / sigma) - normalCdf((-h.x - p.x) / sigma);
		mass += rows * cols;
	}
	return clamp(mass, 0.0, 1.0);
}

// The coverage of a blur style, from the blurred coverage b and the shape's own o.
float blurStyle(int style, float b, float o) {
	if (style == 1) {
		return o + b * (1.0 - o);  // solid
	}
	if (style == 2) {
		return b * (1.0 - o);      // outer
	}
	if (style == 3) {
		return b * o;              // inner
	}
	return b;                      // normal
}
