package explicit

import "math"

// Float16Bits converts a float32 to the bits of an IEEE 754 half float, rounding to nearest even: the
// bytes that an RGBA16Float texture holds for a value. Values too large become infinity, and the
// smallest ones subnormal or zero.
//
// Flutter: no counterpart
func Float16Bits(f float32) uint16 {
	b := math.Float32bits(f)
	sign := uint16(b>>16) & 0x8000
	exp := int32(b>>23)&0xff - 127 + 15
	mant := b & 0x7fffff
	switch {
	case int32(b>>23)&0xff == 0xff: // infinity and NaN
		if mant != 0 {
			return sign | 0x7e00
		}
		return sign | 0x7c00
	case exp >= 0x1f:
		return sign | 0x7c00
	case exp <= 0:
		if exp < -10 {
			return sign
		}
		mant |= 0x800000
		shift := uint(14 - exp)
		half := uint16(mant >> shift)
		if rem := mant & (1<<shift - 1); rem > 1<<(shift-1) || (rem == 1<<(shift-1) && half&1 == 1) {
			half++
		}
		return sign | half
	}
	half := sign | uint16(exp)<<10 | uint16(mant>>13)
	if rem := mant & 0x1fff; rem > 0x1000 || (rem == 0x1000 && half&1 == 1) {
		half++
	}
	return half
}
