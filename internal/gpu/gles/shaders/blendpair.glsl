// The Porter-Duff blend modes and the simple separable ones, for blending two
// colors inside a shader (a sprite with its tint, a paint with vertex colors).
// Both colors are premultiplied. mode is the index of the mode: 0 clear, 1 src,
// 2 dst, 3 srcOver, 4 dstOver, 5 srcIn, 6 dstIn, 7 srcOut, 8 dstOut, 9 srcATop,
// 10 dstATop, 11 xor, 12 plus, 13 modulate, 14 screen. Shaders include this file
// with an include line.

vec4 blendPair(int mode, vec4 s, vec4 d) {
	float sa = s.a;
	float da = d.a;
	switch (mode) {
	case 0: return vec4(0.0);
	case 1: return s;
	case 2: return d;
	case 3: return s + d * (1.0 - sa);
	case 4: return d + s * (1.0 - da);
	case 5: return s * da;
	case 6: return d * sa;
	case 7: return s * (1.0 - da);
	case 8: return d * (1.0 - sa);
	case 9: return s * da + d * (1.0 - sa);
	case 10: return d * sa + s * (1.0 - da);
	case 11: return s * (1.0 - da) + d * (1.0 - sa);
	case 12: return min(s + d, vec4(1.0));
	case 13: return s * d;
	case 14: return s + d - s * d;
	}
	return s;
}
