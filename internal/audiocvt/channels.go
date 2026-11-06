package audiocvt

// This file is translated to Go and changed from src/audio/SDL_audiocvt.c of SDL 3 (zlib license, third_party/sdl/LICENSE): the
// change of the number of channels. SDL has a function for each pair of layouts; the generator of channels_gen.go reads them as
// matrices, which this file applies.

//go:generate go run ../../cmd/glutter-gen audiochannels -reference ../../reference/sdl -out .

// MaxChannels is the most channels that a stream has (SDL's largest layout, 7.1).
//
// Flutter: no counterpart
const MaxChannels = 8

// ChannelMatrix returns the matrix that turns src channels into dst channels (SDL's layouts), dst rows of src columns, row after
// row; it is nil when the counts are equal or outside 1 to MaxChannels. The caller must not change it.
//
// Flutter: no counterpart
func ChannelMatrix(src, dst int) []float32 {
	if src < 1 || src > MaxChannels || dst < 1 || dst > MaxChannels {
		return nil
	}
	return channelMatrices[src-1][dst-1]
}

// ConvertChannels turns frames frames of src (srcCh channels) into dst (dstCh channels). dst and src must not overlap. With equal
// counts it copies.
//
// Flutter: no counterpart
func ConvertChannels(dst, src []float32, srcCh, dstCh, frames int) {
	if srcCh == dstCh {
		copy(dst[:frames*dstCh], src[:frames*srcCh])
		return
	}
	m := channelMatrices[srcCh-1][dstCh-1]
	switch {
	case srcCh == 1 && dstCh == 2:
		for i := 0; i < frames; i++ {
			v := src[i]
			dst[2*i], dst[2*i+1] = v, v
		}
	case srcCh == 2 && dstCh == 1:
		for i := 0; i < frames; i++ {
			dst[i] = src[2*i]*0.5 + src[2*i+1]*0.5
		}
	default:
		for i := 0; i < frames; i++ {
			in := src[i*srcCh : i*srcCh+srcCh]
			out := dst[i*dstCh : i*dstCh+dstCh]
			for k := range out {
				row := m[k*srcCh : k*srcCh+srcCh]
				var acc float32
				for j, c := range row {
					if c != 0 {
						acc += in[j] * c
					}
				}
				out[k] = acc
			}
		}
	}
}
