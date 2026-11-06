package audiocvt

import (
	"math"
	"sync"
)

// This file is translated to Go and changed from src/audio/SDL_audioresample.c of SDL 3 (zlib license,
// third_party/sdl/LICENSE): the resampler, which is a bandlimited interpolation (https://ccrma.stanford.edu/~jos/resample/) with a
// Kaiser window of 80 dB, a table of cubic polynomials of the filter, and a position in the input kept as a fixed point number of
// 32 bits and 32 bits of fraction. SDL's generic version is taken, with the 6 zero crossings of its version for SIMD.

const (
	// resamplerZeroCrossings is the number of zero crossings of the filter on each side.
	resamplerZeroCrossings = 6
	// resamplerSamplesPerFrame is the number of input frames that make an output frame.
	resamplerSamplesPerFrame = resamplerZeroCrossings * 2
	// resamplerMaxPaddingFrames is the frames needed before and after the input: the history, and what the filter reaches.
	resamplerMaxPaddingFrames = resamplerZeroCrossings + 1

	resamplerBitsPerZeroCrossing    = 3
	resamplerSamplesPerZeroCrossing = 1 << resamplerBitsPerZeroCrossing
	resamplerFilterInterpBits       = 32 - resamplerBitsPerZeroCrossing
	resamplerFilterInterpRange      = 1 << resamplerFilterInterpBits
)

// cubic is a polynomial of the third degree: v[0] + v[1] x + v[2] x² + v[3] x³.
type cubic [4]float32

// filterTable is the table of the filter, made when it is first used. It never changes afterwards.
var filterTable = sync.OnceValue(generateResamplerFilter)

// cubicLeastSquares fits a cubic to four points at 0, 1/3, 2/3 and 1 (https://en.wikipedia.org/wiki/Polynomial_regression).
func cubicLeastSquares(y0, y1, y2, y3 float32) cubic {
	return cubic{
		y0,
		-5.5*y0 + 9.0*y1 - 4.5*y2 + y3,
		9.0*y0 - 22.5*y1 + 18.0*y2 - 4.5*y3,
		-4.5*y0 + 13.5*y1 - 13.5*y2 + 4.5*y3,
	}
}

// besselI0 is the zeroth order modified Bessel function of the first kind.
func besselI0(x float64) float64 {
	const epsilon = 1.1920929e-07 // FLT_EPSILON
	sum, i, t := 0.0, 1.0, 1.0
	x *= x * 0.25
	for t >= sum*epsilon {
		sum += t
		t *= x / (i * i)
		i++
	}
	return sum
}

// generateResamplerFilter makes the table of the filter: for each of the positions between two input frames, the cubics of the
// frame's samples.
func generateResamplerFilter() *[resamplerSamplesPerZeroCrossing][resamplerSamplesPerFrame]cubic {
	// Samples are made at 3 times the resolution of the table, so that each position has the points at [0, 1/3, 2/3, 1].
	const (
		tableSamplesPerZeroCrossing = resamplerSamplesPerZeroCrossing * 3
		tableSize                   = resamplerZeroCrossings * tableSamplesPerZeroCrossing
	)
	// dB > 50: beta = 0.1102 * (dB - 8.7), according to Matlab.
	const dB = 80.0
	beta := 0.1102 * (dB - 8.7)
	besselBeta := besselI0(beta)
	lensqr := float64(tableSize * tableSize)

	// sin(pi x) / pi for 180 degrees.
	var sinc [tableSamplesPerZeroCrossing]float64
	for i := range sinc {
		sinc[i] = math.Sin(float64(i)*(math.Pi/tableSamplesPerZeroCrossing)) / math.Pi
	}
	sincAt := func(x, y int) float64 {
		s := sinc[x%y]
		if (x/y)&1 != 0 {
			s = -s
		}
		return s * float64(y) / float64(x)
	}

	// One wing of the filter (https://en.wikipedia.org/wiki/Kaiser_window).
	var filter [tableSize + 1]float32
	filter[0] = 1
	for i := 1; i <= tableSize; i++ {
		b := besselI0(beta*math.Sqrt((lensqr-float64(i*i))/lensqr)) / besselBeta
		filter[i] = float32(b * sincAt(i, tableSamplesPerZeroCrossing))
	}

	// The left wing is the filter reversed, the right wing the filter offset by 1: interpolating backwards between the same
	// points. interp(p[n], p[n+1], t) = interp(p[n+1], p[n], 1 - t).
	var table [resamplerSamplesPerZeroCrossing][resamplerSamplesPerFrame]cubic
	for i := 0; i < resamplerSamplesPerZeroCrossing; i++ {
		for j := 0; j < resamplerZeroCrossings; j++ {
			ys := filter[(j*resamplerSamplesPerZeroCrossing+i)*3:]
			table[i][resamplerZeroCrossings-j-1] = cubicLeastSquares(ys[0], ys[1], ys[2], ys[3])
			table[resamplerSamplesPerZeroCrossing-i-1][resamplerZeroCrossings+j] = cubicLeastSquares(ys[3], ys[2], ys[1], ys[0])
		}
	}
	return &table
}

// ResampleRate returns the rate of the resampler for two sample rates: how far in the input one output frame goes, as a fixed point
// number with 32 bits of fraction, rounded up so that the input is used up (SDL_GetResampleRate). Both rates must be positive.
//
// Flutter: no counterpart
func ResampleRate(srcRate, dstRate int) int64 {
	numerator := int64(srcRate) << 32
	denominator := int64(dstRate)
	return (numerator-1)/denominator + 1
}

// ResamplerHistoryFrames is how many frames before the first one that is not used up yet are kept for the filter
// (SDL_GetResamplerHistoryFrames).
//
// Flutter: no counterpart
const ResamplerHistoryFrames = resamplerMaxPaddingFrames

// ResamplerPaddingFrames is how many frames the filter needs before the input and after it (SDL_GetResamplerPaddingFrames).
//
// Flutter: no counterpart
const ResamplerPaddingFrames = resamplerMaxPaddingFrames

// ResamplerInputFrames returns how many input frames outFrames output frames use, given the offset the resampler is at
// (SDL_GetResamplerInputFrames).
//
// Flutter: no counterpart
func ResamplerInputFrames(outFrames, rate, offset int64) int64 {
	// The index of the last input frame, plus 1: ((((outFrames - 1) * rate) + offset) >> 32) + 1.
	if outFrames <= 0 {
		return 0
	}
	off := (outFrames-1)*rate + offset + 1<<32
	n := int64(int32(off >> 32))
	return max(n, 0)
}

// ResamplerOutputFrames returns how many output frames inFrames input frames give from the offset *offset, and sets *offset to what
// it is afterwards (SDL_GetResamplerOutputFrames).
//
// Flutter: no counterpart
func ResamplerOutputFrames(inFrames, rate int64, offset *int64) int64 {
	inputOffset := inFrames<<32 - *offset
	var out int64
	if inputOffset > 0 {
		out = (inputOffset-1)/rate + 1
	}
	*offset = out*rate - inputOffset
	return out
}

// Resample makes outFrames frames of chans channels from inFrames input frames (SDL_ResampleAudio). src is the input with its
// padding: frame 0 of the input is at src[base*chans], and ResamplerPaddingFrames frames lie before it and after the last one. The
// offset is where in the input the first output frame is (a fixed point number, 32 bits of fraction) and is updated to the next
// call's. The rate comes from ResampleRate.
//
// Flutter: no counterpart
func Resample(chans int, src []float32, base int, inFrames int, dst []float32, outFrames int, rate int64, offset *int64) {
	table := filterTable()
	srcpos := *offset
	start := (base - (resamplerZeroCrossings - 1)) * chans
	var scales [resamplerSamplesPerFrame]float32
	for i := 0; i < outFrames; i++ {
		srcindex := int(int32(srcpos >> 32))
		srcfraction := uint32(srcpos & 0xFFFFFFFF)
		srcpos += rate

		filter := &table[srcfraction>>resamplerFilterInterpBits]
		frac := float32(srcfraction&(resamplerFilterInterpRange-1)) * (1.0 / resamplerFilterInterpRange)
		frac2 := frac * frac
		frac3 := frac * frac2
		for k := range scales {
			f := &filter[k]
			scales[k] = f[0] + f[1]*frac + f[2]*frac2 + f[3]*frac3
		}
		frame := src[start+srcindex*chans:]
		out := dst[i*chans : i*chans+chans]
		switch chans {
		case 1:
			var acc float32
			for k, s := range scales {
				acc += frame[k] * s
			}
			out[0] = acc
		case 2:
			var a0, a1 float32
			for k, s := range scales {
				a0 += frame[2*k] * s
				a1 += frame[2*k+1] * s
			}
			out[0], out[1] = a0, a1
		default:
			for c := range out {
				var acc float32
				for k, s := range scales {
					acc += frame[k*chans+c] * s
				}
				out[c] = acc
			}
		}
	}
	*offset = srcpos - int64(inFrames)<<32
}
