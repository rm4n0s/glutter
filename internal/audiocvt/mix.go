package audiocvt

// This file is translated to Go and changed from src/audio/SDL_mixer.c of SDL 3 (zlib license, third_party/sdl/LICENSE): SDL
// mixes in the format of the device and clips each sum; here the sums are float32 and the clip is made once, when the mix is done.

// Mix adds src multiplied by gain to dst, for the samples that both have.
//
// Flutter: no counterpart
func Mix(dst, src []float32, gain float32) {
	n := min(len(dst), len(src))
	dst, src = dst[:n], src[:n]
	if gain == 1 {
		for i, s := range src {
			dst[i] += s
		}
		return
	}
	for i, s := range src {
		dst[i] += s * gain
	}
}

// Clip cuts the samples to the range from -1 to 1. A sample that is not a number becomes 0.
//
// Flutter: no counterpart
func Clip(samples []float32) {
	for i, s := range samples {
		switch {
		case s > 1:
			samples[i] = 1
		case s < -1:
			samples[i] = -1
		case s != s:
			samples[i] = 0
		}
	}
}

// Scale multiplies the samples by gain.
//
// Flutter: no counterpart
func Scale(samples []float32, gain float32) {
	if gain == 1 {
		return
	}
	for i := range samples {
		samples[i] *= gain
	}
}
