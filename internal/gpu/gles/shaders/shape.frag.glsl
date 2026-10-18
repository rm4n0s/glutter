#version 300 es
precision highp float;

// The coverage of a pixel by a shape comes from the shape's signed distance
// field d: negative inside, positive outside, in the shape's own units. The pixel
// is then 0.5 - d / (the size of a pixel in those units), clamped to 0..1. That is
// exact along a straight edge and close at corners.

uniform vec2 uSurface;

flat in vec4 vInv0;
flat in vec4 vInv1;
flat in vec4 vColor;
flat in vec4 vClip;
flat in vec4 vA;
flat in vec4 vARad1;
flat in vec4 vARad2;
flat in vec4 vB;
flat in vec4 vBRad1;
flat in vec4 vBRad2;
flat in vec4 vParam;

out vec4 fragColor;

const int KIND_RRECT = 0;      // a rounded rectangle, minus shape B when the flags say so
const int KIND_SUPER = 1;      // a rounded superellipse
const int KIND_SEGMENT = 2;    // a line segment with caps, or a dot
const int KIND_FILL = 3;       // every pixel
const int KIND_BOX = 4;        // an axis-aligned rectangle in pixels, minus shape B, by exact area
const int KIND_OUTLINE = 5;    // every point within a half width of a rounded rectangle's outline

const int FLAG_HAS_B = 1;
const int FLAG_ANTIALIAS = 2;
const int FLAG_SIMILARITY = 4; // the transform only rotates, flips and scales evenly

//#include "sdf.glsl"
//#include "clipmask.glsl"

// A rounded superellipse: the corner boxes in r1 and r2 are filled by the curve
// |x|^n + |y|^n = 1.
float superDistance(vec2 p, vec2 h, vec4 r1, vec4 r2, float n) {
	vec2 r = cornerRadii(p, r1, r2);
	vec2 q = abs(p) - (h - r);
	if (q.x > 0.0 && q.y > 0.0) {
		if (r.x <= 0.0 || r.y <= 0.0) {
			return length(q);
		}
		vec2 u = q / r;
		float f = pow(u.x, n) + pow(u.y, n) - 1.0;
		float g = n * length(pow(u, vec2(n - 1.0)) / r);
		return g > 0.0 ? f / g : 0.0;
	}
	return max(q.x - r.x, q.y - r.y);
}

// A segment from a to b of half width hw. cap is 0 for butt, 1 for round, 2 for
// square. A segment of no length is a dot: round or square, nothing for butt.
float segmentDistance(vec2 p, vec2 a, vec2 b, float hw, float cap) {
	vec2 ba = b - a;
	float len = length(ba);
	if (len < 1e-5) {
		vec2 pa = p - a;
		if (cap == 1.0) {
			return length(pa) - hw;
		}
		if (cap == 2.0) {
			return max(abs(pa.x), abs(pa.y)) - hw;
		}
		return 1e9;
	}
	vec2 dir = ba / len;
	vec2 pa = p - (a + b) * 0.5;
	float along = abs(dot(pa, dir));
	float across = abs(dot(pa, vec2(-dir.y, dir.x)));
	if (cap == 1.0) {
		return length(vec2(max(along - len * 0.5, 0.0), across)) - hw;
	}
	if (cap == 2.0) {
		return max(along - len * 0.5 - hw, across - hw);
	}
	return max(along - len * 0.5, across - hw);
}

void main() {
	vec2 dev = vec2(gl_FragCoord.x, uSurface.y - gl_FragCoord.y);

	// The clip rectangle: exact coverage, so anti-aliased clip edges are smooth.
	float clipCoverage = span(dev.x, vClip.x, vClip.z) * span(dev.y, vClip.y, vClip.w) * clipMaskCoverage(dev);

	int kind = int(vInv1.z + 0.5);
	int flags = int(vInv1.w + 0.5);
	vec2 p = vec2(
		vInv0.x * dev.x + vInv0.z * dev.y + vInv1.x,
		vInv0.y * dev.x + vInv0.w * dev.y + vInv1.y);

	if (kind == KIND_BOX) {
		bool aa = (flags & FLAG_ANTIALIAS) != 0;
		float cov = boxCoverage(dev, vA, aa);
		if ((flags & FLAG_HAS_B) != 0) {
			cov -= boxCoverage(dev, vB, aa);
		}
		cov *= clipCoverage;
		if (cov <= 0.0) {
			discard;
		}
		fragColor = vColor * cov;
		return;
	}

	float d;
	if (kind == KIND_RRECT) {
		d = rrectDistance(p - vA.xy, vA.zw, vARad1, vARad2);
		if ((flags & FLAG_HAS_B) != 0) {
			d = max(d, -rrectDistance(p - vB.xy, vB.zw, vBRad1, vBRad2));
		}
	} else if (kind == KIND_OUTLINE) {
		d = abs(rrectDistance(p - vA.xy, vA.zw, vARad1, vARad2)) - vParam.x;
	} else if (kind == KIND_SUPER) {
		d = superDistance(p - vA.xy, vA.zw, vARad1, vARad2, vParam.z);
	} else if (kind == KIND_SEGMENT) {
		d = segmentDistance(p, vA.xy, vA.zw, vParam.x, vParam.y);
	} else {
		d = -1.0;
	}

	float coverage;
	if ((flags & FLAG_ANTIALIAS) == 0) {
		coverage = d <= 0.0 ? 1.0 : 0.0;
	} else if (kind == KIND_FILL) {
		coverage = 1.0;
	} else {
		// The distance in pixels: the same everywhere for a transform that only
		// rotates and scales evenly, found from the screen derivatives otherwise.
		float pixels;
		if ((flags & FLAG_SIMILARITY) != 0) {
			pixels = d * vParam.w;
		} else {
			float g = length(vec2(dFdx(d), dFdy(d)));
			pixels = g > 1e-9 ? d / g : (d <= 0.0 ? -1.0 : 1.0);
		}
		coverage = clamp(0.5 - pixels, 0.0, 1.0);
	}
	// Discard last: the screen derivatives above need every pixel of the quad.
	if (coverage * clipCoverage <= 0.0) {
		discard;
	}
	fragColor = vColor * (coverage * clipCoverage);
}
