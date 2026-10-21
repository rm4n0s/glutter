// The 29 blend modes of Flutter's BlendMode, on premultiplied colors, as the software renderer
// computes them (internal/gpu/soft/blend.go). mode is the index of the mode in BlendMode's
// order: 0 clear, 1 src, 2 dst, 3 srcOver, 4 dstOver, 5 srcIn, 6 dstIn, 7 srcOut, 8 dstOut,
// 9 srcATop, 10 dstATop, 11 xor, 12 plus, 13 modulate, 14 screen, 15 overlay, 16 darken,
// 17 lighten, 18 colorDodge, 19 colorBurn, 20 hardLight, 21 softLight, 22 difference,
// 23 exclusion, 24 multiply, 25 hue, 26 saturation, 27 color, 28 luminosity. Ported from
// gles/shaders/blend.glsl. Shaders include this file with an include line.

fn hardLightChannel(s: f32, d: f32, sa: f32, da: f32) -> f32 {
	if 2.0 * s <= sa {
		return s * (1.0 - da) + d * (1.0 - sa) + 2.0 * s * d;
	}
	return s * (1.0 - da) + d * (1.0 - sa) + sa * da - 2.0 * (da - d) * (sa - s);
}

fn softLightChannel(s: f32, d: f32, sa: f32, da: f32) -> f32 {
	var m = 0.0;
	if da > 0.0 {
		m = d / da;
	}
	let s2 = 2.0 * s;
	let m4 = 4.0 * m;
	let darkSrc = d * (sa + (s2 - sa) * (1.0 - m));
	let darkDst = (m4 * m4 + m4) * (m - 1.0) + 7.0 * m;
	let liteDst = sqrt(m) - m;
	var tail = liteDst;
	if 4.0 * d <= da {
		tail = darkDst;
	}
	let liteSrc = d * sa + da * (s2 - sa) * tail;
	var body = liteSrc;
	if s2 <= sa {
		body = darkSrc;
	}
	return s * (1.0 - da) + d * (1.0 - sa) + body;
}

fn blendChannel(mode: i32, s: f32, d: f32, sa: f32, da: f32) -> f32 {
	switch mode {
	case 15: {
		return hardLightChannel(d, s, da, sa); // overlay
	}
	case 16: {
		return s + d - max(s * da, d * sa); // darken
	}
	case 17: {
		return s + d - min(s * da, d * sa); // lighten
	}
	case 18: { // colorDodge
		if d == 0.0 {
			return s * (1.0 - da);
		}
		if s == sa {
			return s + d * (1.0 - sa);
		}
		return sa * min(da, d * sa / (sa - s)) + s * (1.0 - da) + d * (1.0 - sa);
	}
	case 19: { // colorBurn
		if d == da {
			return d + s * (1.0 - da);
		}
		if s == 0.0 {
			return d * (1.0 - sa);
		}
		return sa * (da - min(da, (da - d) * sa / s)) + s * (1.0 - da) + d * (1.0 - sa);
	}
	case 20: {
		return hardLightChannel(s, d, sa, da); // hardLight
	}
	case 21: {
		return softLightChannel(s, d, sa, da); // softLight
	}
	case 22: {
		return s + d - 2.0 * min(s * da, d * sa); // difference
	}
	case 23: {
		return s + d - 2.0 * s * d; // exclusion
	}
	case 24: {
		return s * (1.0 - da) + d * (1.0 - sa) + s * d; // multiply
	}
	default: {
		return s + d * (1.0 - sa);
	}
	}
}

fn lum(c: vec3<f32>) -> f32 {
	return 0.30 * c.r + 0.59 * c.g + 0.11 * c.b;
}

fn sat(c: vec3<f32>) -> f32 {
	return max(c.r, max(c.g, c.b)) - min(c.r, min(c.g, c.b));
}

fn setSat(c: vec3<f32>, s: f32) -> vec3<f32> {
	let mn = min(c.r, min(c.g, c.b));
	let mx = max(c.r, max(c.g, c.b));
	let range = mx - mn;
	if range == 0.0 {
		return vec3<f32>(0.0);
	}
	return (c - vec3<f32>(mn)) * s / range;
}

fn setLum(c: vec3<f32>, l: f32) -> vec3<f32> {
	return c + vec3<f32>(l - lum(c));
}

fn clipColor(c0: vec3<f32>, a: f32) -> vec3<f32> {
	var c = c0;
	let mn = min(c.r, min(c.g, c.b));
	let mx = max(c.r, max(c.g, c.b));
	let l = lum(c);
	if mn < 0.0 {
		c = vec3<f32>(l) + (c - vec3<f32>(l)) * l / (l - mn);
	}
	if mx > a {
		c = vec3<f32>(l) + (c - vec3<f32>(l)) * (a - l) / (mx - l);
	}
	return max(c, vec3<f32>(0.0));
}

fn blendNonSeparable(mode: i32, s: vec4<f32>, d: vec4<f32>) -> vec4<f32> {
	let sa = s.a;
	let da = d.a;
	var c: vec3<f32>;
	if mode == 25 { // hue
		c = s.rgb * sa;
		c = setSat(c, sat(d.rgb) * sa);
		c = setLum(c, lum(d.rgb) * sa);
	} else if mode == 26 { // saturation
		c = d.rgb * sa;
		c = setSat(c, sat(s.rgb) * da);
		c = setLum(c, lum(d.rgb) * sa);
	} else if mode == 27 { // color
		c = s.rgb * da;
		c = setLum(c, lum(d.rgb) * sa);
	} else { // luminosity
		c = d.rgb * sa;
		c = setLum(c, lum(s.rgb) * da);
	}
	c = clipColor(c, sa * da);
	return vec4<f32>(s.rgb * (1.0 - da) + d.rgb * (1.0 - sa) + c, sa + da - sa * da);
}

// blendPair blends the source s over the destination d with the mode.
fn blendPair(mode: i32, s: vec4<f32>, d: vec4<f32>) -> vec4<f32> {
	let sa = s.a;
	let da = d.a;
	switch mode {
	case 0: {
		return vec4<f32>(0.0);
	}
	case 1: {
		return s;
	}
	case 2: {
		return d;
	}
	case 3: {
		return s + d * (1.0 - sa);
	}
	case 4: {
		return d + s * (1.0 - da);
	}
	case 5: {
		return s * da;
	}
	case 6: {
		return d * sa;
	}
	case 7: {
		return s * (1.0 - da);
	}
	case 8: {
		return d * (1.0 - sa);
	}
	case 9: {
		return s * da + d * (1.0 - sa);
	}
	case 10: {
		return d * sa + s * (1.0 - da);
	}
	case 11: {
		return s * (1.0 - da) + d * (1.0 - sa);
	}
	case 12: {
		return min(s + d, vec4<f32>(1.0));
	}
	case 13: {
		return s * d;
	}
	case 14: {
		return s + d - s * d;
	}
	case 25, 26, 27, 28: {
		return blendNonSeparable(mode, s, d);
	}
	default: {
		// The separable modes defined per channel, whose alpha is that of source-over.
		return vec4<f32>(
			blendChannel(mode, s.r, d.r, sa, da),
			blendChannel(mode, s.g, d.g, sa, da),
			blendChannel(mode, s.b, d.b, sa, da),
			sa + da - sa * da);
	}
	}
}
