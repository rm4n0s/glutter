// Signed distance fields of rounded rectangles and exact box coverage, shared by
// the shaders that draw shapes and images. Shaders include this file with
// an include line.

// Distance from the corner q of an ellipse with radii r, with q measured from the
// ellipse's center: a first-order estimate that is exact for a circle.
float ellipseDistance(vec2 q, vec2 r) {
	float k0 = length(q / r);
	float k1 = length(q / (r * r));
	return k1 > 0.0 ? k0 * (k0 - 1.0) / k1 : -min(r.x, r.y);
}

// The corner radii of the quadrant that p is in (y points down).
vec2 cornerRadii(vec2 p, vec4 r1, vec4 r2) {
	if (p.x < 0.0) {
		return p.y < 0.0 ? r1.xy : r2.zw;
	}
	return p.y < 0.0 ? r1.zw : r2.xy;
}

// A rounded rectangle centered on the origin with half extents h.
float rrectDistance(vec2 p, vec2 h, vec4 r1, vec4 r2) {
	vec2 r = cornerRadii(p, r1, r2);
	vec2 q = abs(p) - (h - r);
	if (q.x > 0.0 && q.y > 0.0) {
		return (r.x > 0.0 && r.y > 0.0) ? ellipseDistance(q, r) : length(q);
	}
	return max(q.x - r.x, q.y - r.y);
}

// The part of the pixel centered at c that the interval [lo, hi] covers.
float span(float c, float lo, float hi) {
	return clamp(min(c + 0.5, hi) - max(c - 0.5, lo), 0.0, 1.0);
}

// The part of the pixel centered at dev that the rectangle r (left, top, right,
// bottom) covers; with antialias false, 1 when the center is inside.
float boxCoverage(vec2 dev, vec4 r, bool antialias) {
	if (antialias) {
		return span(dev.x, r.x, r.z) * span(dev.y, r.y, r.w);
	}
	return (dev.x >= r.x && dev.x < r.z && dev.y >= r.y && dev.y < r.w) ? 1.0 : 0.0;
}
