// Package audiocvt converts and mixes audio samples (SPEC §28.3): the conversion between sample types, the mixing between channel
// layouts, the resampler and the converter that a stream of package audio uses. It is plain Go with no OS tags and takes no
// device, so a test can run any of it on any host.
//
// It is translated to Go and changed from src/audio/SDL_audiotypecvt.c, SDL_audiocvt.c, SDL_audioresample.c,
// SDL_audio_channel_converters.h and SDL_mixer.c of SDL 3 (zlib license, third_party/sdl/LICENSE). Samples are interleaved
// float32 from -1 to 1 inside the package; the layouts of channels are SDL's: mono, stereo, 2.1 (FL FR LFE), quad (FL FR BL BR),
// 4.1 (FL FR LFE BL BR), 5.1 (FL FR FC LFE BL BR), 6.1 (FL FR FC LFE BC SL SR) and 7.1 (FL FR FC LFE BL BR SL SR).
//
// Flutter: no counterpart
package audiocvt

import "math"

// Int16ToFloat32 converts signed 16-bit samples to float32 (SDL_Convert_S16_to_F32): each is divided by 32768, which is exact. dst
// must be at least as long as src.
//
// Flutter: no counterpart
func Int16ToFloat32(dst []float32, src []int16) {
	dst = dst[:len(src)]
	for i, s := range src {
		dst[i] = float32(s) * (1.0 / 32768.0)
	}
}

// Int32ToFloat32 converts signed 32-bit samples to float32 (SDL_Convert_S32_to_F32). dst must be at least as long as src.
//
// Flutter: no counterpart
func Int32ToFloat32(dst []float32, src []int32) {
	dst = dst[:len(src)]
	for i, s := range src {
		dst[i] = float32(s) * (1.0 / 2147483648.0)
	}
}

// Float32ToInt16 converts float32 samples to signed 16-bit ones (SDL_Convert_F32_to_S16): a sample is multiplied by 32768 and
// rounded to the nearest, ties to even, and what is outside the range of 16 bits is cut to it. A sample that is not a number
// is 0. dst must be at least as long as src.
//
// Flutter: no counterpart
func Float32ToInt16(dst []int16, src []float32) {
	dst = dst[:len(src)]
	for i, s := range src {
		v := float64(s) * 32768
		switch {
		case v != v:
			dst[i] = 0
		case v >= 32767:
			dst[i] = math.MaxInt16
		case v <= -32768:
			dst[i] = math.MinInt16
		default:
			dst[i] = int16(math.RoundToEven(v))
		}
	}
}

// Float32ToInt32 converts float32 samples to signed 32-bit ones (SDL_Convert_F32_to_S32): a sample is multiplied by 2147483648,
// cut to the range of 32 bits and truncated. A sample that is not a number is 0. dst must be at least as long as src.
//
// Flutter: no counterpart
func Float32ToInt32(dst []int32, src []float32) {
	dst = dst[:len(src)]
	for i, s := range src {
		v := float64(s) * 2147483648
		switch {
		case v != v:
			dst[i] = 0
		case v >= 2147483647:
			dst[i] = math.MaxInt32
		case v <= -2147483648:
			dst[i] = math.MinInt32
		default:
			dst[i] = int32(v)
		}
	}
}
