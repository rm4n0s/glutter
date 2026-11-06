package audiocvt

import (
	"fmt"
	"math"
	"testing"
)

func TestSampleTypeConversions(t *testing.T) {
	t.Run("int16 to float32", func(t *testing.T) {
		tests := []struct {
			in   int16
			want float32
		}{{0, 0}, {-32768, -1}, {32767, 32767.0 / 32768}, {16384, 0.5}, {-16384, -0.5}, {1, 1.0 / 32768}}
		for _, tt := range tests {
			dst := make([]float32, 1)
			Int16ToFloat32(dst, []int16{tt.in})
			if dst[0] != tt.want {
				t.Errorf("%d = %v, want %v", tt.in, dst[0], tt.want)
			}
		}
	})
	t.Run("int32 to float32", func(t *testing.T) {
		tests := []struct {
			in   int32
			want float32
		}{{0, 0}, {math.MinInt32, -1}, {1 << 30, 0.5}, {-(1 << 30), -0.5}, {math.MaxInt32, 1}}
		for _, tt := range tests {
			dst := make([]float32, 1)
			Int32ToFloat32(dst, []int32{tt.in})
			if dst[0] != tt.want {
				t.Errorf("%d = %v, want %v", tt.in, dst[0], tt.want)
			}
		}
	})
	nan := float32(math.NaN())
	t.Run("float32 to int16", func(t *testing.T) {
		tests := []struct {
			in   float32
			want int16
		}{
			{0, 0}, {1, 32767}, {-1, -32768}, {2, 32767}, {-2, -32768}, {0.5, 16384}, {-0.5, -16384}, {nan, 0},
			{0.5 / 32768, 0}, {1.5 / 32768, 2}, {2.5 / 32768, 2}, {-0.5 / 32768, 0}, {-1.5 / 32768, -2}, // ties go to even
			{float32(math.Inf(1)), 32767}, {float32(math.Inf(-1)), -32768},
		}
		for _, tt := range tests {
			dst := make([]int16, 1)
			Float32ToInt16(dst, []float32{tt.in})
			if dst[0] != tt.want {
				t.Errorf("%v = %d, want %d", tt.in, dst[0], tt.want)
			}
		}
	})
	t.Run("float32 to int32", func(t *testing.T) {
		tests := []struct {
			in   float32
			want int32
		}{{0, 0}, {1, math.MaxInt32}, {-1, math.MinInt32}, {3, math.MaxInt32}, {-3, math.MinInt32}, {0.5, 1 << 30}, {-0.5, -(1 << 30)}, {nan, 0}}
		for _, tt := range tests {
			dst := make([]int32, 1)
			Float32ToInt32(dst, []float32{tt.in})
			if dst[0] != tt.want {
				t.Errorf("%v = %d, want %d", tt.in, dst[0], tt.want)
			}
		}
	})
	t.Run("round trip", func(t *testing.T) {
		var in16 []int16
		for v := -32768; v <= 32767; v += 7 {
			in16 = append(in16, int16(v))
		}
		f := make([]float32, len(in16))
		Int16ToFloat32(f, in16)
		out := make([]int16, len(in16))
		Float32ToInt16(out, f)
		for i := range in16 {
			if out[i] != in16[i] {
				t.Fatalf("%d came back as %d", in16[i], out[i])
			}
		}
		in32 := []int32{math.MinInt32, -123456789, -1 << 16, -1, 0, 1, 1 << 16, 123456789, math.MaxInt32 - 127}
		f = make([]float32, len(in32))
		Int32ToFloat32(f, in32)
		out32 := make([]int32, len(in32))
		Float32ToInt32(out32, f)
		for i := range in32 {
			// float32 has 24 bits: the numbers come back within 128.
			if d := int64(out32[i]) - int64(in32[i]); d < -128 || d > 128 {
				t.Errorf("%d came back as %d", in32[i], out32[i])
			}
		}
	})
}

func TestChannelMatrixDimensionsAndEnergy(t *testing.T) {
	for src := 1; src <= MaxChannels; src++ {
		for dst := 1; dst <= MaxChannels; dst++ {
			m := ChannelMatrix(src, dst)
			if src == dst {
				if m != nil {
					t.Errorf("%d to %d has a matrix", src, dst)
				}
				continue
			}
			if len(m) != src*dst {
				t.Errorf("%d to %d: %d entries, want %d", src, dst, len(m), src*dst)
				continue
			}
			for k := 0; k < dst; k++ {
				var sum float32
				for j := 0; j < src; j++ {
					sum += m[k*src+j]
				}
				// No channel is louder than the loudest signal that goes into it.
				if sum > 1.01 { // SDL's 7.1 to mono gains add up to 1.0017
					t.Errorf("%d to %d, channel %d: the gains add up to %v", src, dst, k, sum)
				}
			}
		}
	}
	for _, bad := range [][2]int{{0, 1}, {1, 0}, {9, 1}, {1, 9}} {
		if ChannelMatrix(bad[0], bad[1]) != nil {
			t.Errorf("%v has a matrix", bad)
		}
	}
}

// TestConvertChannelsAgainstSDLsTables checks conversions with the numbers of SDL's own tables (SDL_audio_channel_converters.h).
func TestConvertChannelsAgainstSDLsTables(t *testing.T) {
	tests := []struct {
		name         string
		srcCh, dstCh int
		in, want     []float32
	}{
		{"mono to stereo", 1, 2, []float32{0.5, -0.25}, []float32{0.5, 0.5, -0.25, -0.25}},
		{"stereo to mono", 2, 1, []float32{1, 0.5, -1, 1}, []float32{0.75, 0}},
		{"mono to 5.1: the front centre is not used", 1, 6, []float32{1}, []float32{1, 1, 0, 0, 0, 0}},
		{"stereo to 5.1", 2, 6, []float32{1, -1}, []float32{1, -1, 0, 0, 0, 0}},
		{"5.1 to stereo, the front left", 6, 2, []float32{1, 0, 0, 0, 0, 0}, []float32{0.294545442, 0}},
		{"5.1 to stereo, the centre", 6, 2, []float32{0, 0, 1, 0, 0, 0}, []float32{0.208181813, 0.208181813}},
		{"5.1 to stereo, the back left", 6, 2, []float32{0, 0, 0, 0, 1, 0}, []float32{0.251818180, 0.154545456}},
		{"quad to stereo, the back right", 4, 2, []float32{0, 0, 0, 1}, []float32{0.219999999, 0.358999997}},
		{"same count copies", 3, 3, []float32{1, 2, 3}, []float32{1, 2, 3}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frames := len(tt.in) / tt.srcCh
			got := make([]float32, frames*tt.dstCh)
			ConvertChannels(got, tt.in, tt.srcCh, tt.dstCh, frames)
			for i := range tt.want {
				if math.Abs(float64(got[i]-tt.want[i])) > 1e-6 {
					t.Fatalf("got %v, want %v", got, tt.want)
				}
			}
		})
	}
	// Every pair of layouts makes silence of silence and keeps a sound that is in the front.
	for src := 1; src <= MaxChannels; src++ {
		for dst := 1; dst <= MaxChannels; dst++ {
			in := make([]float32, src*3)
			out := make([]float32, dst*3)
			ConvertChannels(out, in, src, dst, 3)
			for _, v := range out {
				if v != 0 {
					t.Fatalf("%d to %d: silence became %v", src, dst, out)
				}
			}
			if src == 1 || dst == 1 {
				continue
			}
			in[0] = 1 // the front left of the first frame
			ConvertChannels(out, in, src, dst, 3)
			if out[0] <= 0 || out[0] > 1.001 {
				t.Errorf("%d to %d: the front left became %v", src, dst, out[0])
			}
		}
	}
}

func TestResamplerPositionsAreSDLs(t *testing.T) {
	// SDL's own comment: a ratio of 2:3 is rounded up, so that three output frames use two input frames.
	if got := ResampleRate(2, 3); got != 0xAAAAAAAB {
		t.Errorf("ResampleRate(2, 3) = %#x, want 0xAAAAAAAB", got)
	}
	if got := ResampleRate(48000, 48000); got != 1<<32 {
		t.Errorf("ResampleRate(48000, 48000) = %#x", got)
	}
	if got := ResampleRate(8000, 192000); got != 178956971 {
		t.Errorf("ResampleRate(8000, 192000) = %d", got)
	}
	rate := ResampleRate(44100, 48000)
	for _, out := range []int64{1, 2, 480, 4800, 48000} {
		in := ResamplerInputFrames(out, rate, 0)
		offset := int64(0)
		if back := ResamplerOutputFrames(in+ResamplerPaddingFrames, rate, &offset); back < out {
			t.Errorf("%d output frames use %d input frames, which give only %d back", out, in, back)
		}
	}
	// The offset after the frames is what is left over of the last input frame.
	offset := int64(0)
	out := ResamplerOutputFrames(44100, rate, &offset)
	if out != 48000 || offset < 0 || offset >= rate {
		t.Errorf("44100 frames give %d, offset %d", out, offset)
	}
	if ResamplerInputFrames(0, rate, 0) != 0 {
		t.Error("no output frames use input")
	}
}

// tone makes frames frames of a sine of the given frequency and amplitude, ch channels with the same signal.
func tone(freq float64, rate, ch, frames int, amp float64) []float32 {
	out := make([]float32, frames*ch)
	for i := 0; i < frames; i++ {
		v := float32(amp * math.Sin(2*math.Pi*freq*float64(i)/float64(rate)))
		for c := 0; c < ch; c++ {
			out[i*ch+c] = v
		}
	}
	return out
}

// snr is the ratio, in dB, of the energy of an ideal sine to the energy of the error of got, for the frames from skip to len-skip.
func snr(got []float32, ch int, freq float64, rate int, amp float64, skip int) float64 {
	frames := len(got) / ch
	var sig, noise float64
	for i := skip; i < frames-skip; i++ {
		want := amp * math.Sin(2*math.Pi*freq*float64(i)/float64(rate))
		for c := 0; c < ch; c++ {
			e := float64(got[i*ch+c]) - want
			sig += want * want
			noise += e * e
		}
	}
	return 10 * math.Log10(sig/noise)
}

// convertAll puts all of in into a converter, flushes it, and reads everything out.
func convertAll(c *Converter, in []float32) []float32 {
	c.Put(in)
	c.Flush()
	var out []float32
	buf := make([]float32, 1000*c.dstCh)
	for {
		n := c.Read(buf, 1000)
		if n == 0 {
			return out
		}
		out = append(out, buf[:n*c.dstCh]...)
	}
}

func TestSineThroughEveryRatio(t *testing.T) {
	pairs := [][2]int{
		{44100, 48000}, {48000, 44100}, {8000, 192000}, {192000, 8000}, {22050, 44100}, {44100, 22050}, {8000, 48000}, {48000, 8000},
		{96000, 44100}, {11025, 48000}, {48000, 48000}, {16000, 44100}, {32000, 48000}, {48000, 32000}, {8000, 11025}, {11025, 8000},
	}
	for _, p := range pairs {
		src, dst := p[0], p[1]
		for _, ch := range []int{1, 2, 6} {
			t.Run(fmt.Sprintf("%d to %d, %d channels", src, dst, ch), func(t *testing.T) {
				freq := 0.1 * float64(min(src, dst)) // a fifth of the lower Nyquist frequency
				frames := src / 2
				c := NewConverter(src, ch, dst, ch)
				out := convertAll(c, tone(freq, src, ch, frames, 0.5))
				wantFrames := int(math.Round(float64(frames) * float64(dst) / float64(src)))
				if got := len(out) / ch; got < wantFrames-1 || got > wantFrames+1 {
					t.Errorf("%d frames came out, want about %d", got, wantFrames)
				}
				got := snr(out, ch, freq, dst, 0.5, 64)
				if src != dst && got < 70 {
					t.Errorf("signal to noise %.1f dB, want 70 or more", got)
				}
				if src == dst && got < 120 {
					t.Errorf("a sound that keeps its rate changed: %.1f dB", got)
				}
			})
		}
	}
}

func TestSpeedChangesLengthAndPitch(t *testing.T) {
	const rate = 48000
	in := tone(480, rate, 1, rate, 0.5)
	for _, speed := range []float64{0.5, 1, 2, 4} {
		c := NewConverter(rate, 1, rate, 1)
		c.SetSpeed(speed)
		out := convertAll(c, in)
		want := int(float64(rate) / speed)
		if got := len(out); got < want-2 || got > want+2 {
			t.Errorf("speed %v: %d frames, want about %d", speed, got, want)
		}
		if got := snr(out, 1, 480*speed, rate, 0.5, 64); got < 55 {
			t.Errorf("speed %v: signal to noise %.1f dB, the pitch is wrong", speed, got)
		}
	}
	c := NewConverter(rate, 1, rate, 1)
	c.SetSpeed(1000)
	if c.Speed() != 100 {
		t.Errorf("speed 1000 is %v", c.Speed())
	}
	c.SetSpeed(0)
	if c.Speed() != 0.01 {
		t.Errorf("speed 0 is %v", c.Speed())
	}
}

// TestChunksDoNotChangeTheOutput: the converter makes the same samples whatever the sizes of the pieces that are put in and read.
func TestChunksDoNotChangeTheOutput(t *testing.T) {
	const src, dst = 44100, 48000
	in := tone(1000, src, 2, 20000, 0.5)
	whole := convertAll(NewConverter(src, 2, dst, 2), in)
	for _, chunk := range []int{1, 7, 100, 4410, 19999} {
		c := NewConverter(src, 2, dst, 2)
		var out []float32
		buf := make([]float32, 513*2)
		for i := 0; i < len(in); i += chunk * 2 {
			end := min(i+chunk*2, len(in))
			c.Put(in[i:end])
			for {
				n := c.Read(buf, 513)
				if n == 0 {
					break
				}
				out = append(out, buf[:n*2]...)
			}
		}
		c.Flush()
		for {
			n := c.Read(buf, 513)
			if n == 0 {
				break
			}
			out = append(out, buf[:n*2]...)
		}
		if len(out) != len(whole) {
			t.Fatalf("chunk %d: %d samples, want %d", chunk, len(out), len(whole))
		}
		for i := range out {
			if out[i] != whole[i] {
				t.Fatalf("chunk %d: sample %d is %v, want %v", chunk, i, out[i], whole[i])
			}
		}
	}
}

func TestConverterChangesChannelsAndRate(t *testing.T) {
	const src, dst = 44100, 48000
	// Stereo to mono, with a different signal in each channel: the mono has their mean.
	frames := 20000
	in := make([]float32, frames*2)
	left, right := tone(1000, src, 1, frames, 0.4), tone(1000, src, 1, frames, 0.2)
	for i := 0; i < frames; i++ {
		in[2*i], in[2*i+1] = left[i], right[i]
	}
	out := convertAll(NewConverter(src, 2, dst, 1), in)
	if got := snr(out, 1, 1000, dst, 0.3, 64); got < 60 {
		t.Errorf("stereo to mono: %.1f dB", got)
	}
	// Mono to stereo makes the same signal twice.
	out = convertAll(NewConverter(src, 1, dst, 2), tone(1000, src, 1, frames, 0.5))
	for i := 64; i < len(out)/2-64; i++ {
		if out[2*i] != out[2*i+1] {
			t.Fatalf("frame %d: %v and %v", i, out[2*i], out[2*i+1])
		}
	}
	if got := snr(out, 2, 1000, dst, 0.5, 64); got < 60 {
		t.Errorf("mono to stereo: %.1f dB", got)
	}
	// 5.1 to stereo: the front left alone reaches the left with the gain of SDL's table.
	in = make([]float32, frames*6)
	for i, v := range tone(1000, src, 1, frames, 0.5) {
		in[6*i] = v
	}
	out = convertAll(NewConverter(src, 6, dst, 2), in)
	left32 := make([]float32, len(out)/2)
	for i := range left32 {
		left32[i] = out[2*i]
	}
	if got := snr(left32, 1, 1000, dst, 0.5*0.294545442, 64); got < 60 {
		t.Errorf("5.1 to stereo: %.1f dB", got)
	}
}

func TestConverterQueueAndFlush(t *testing.T) {
	// Without a change of rate, speed or position, what is put in is what is read, and nothing waits for the filter.
	c := NewConverter(48000, 1, 48000, 1)
	c.Put([]float32{1, 2, 3})
	if c.QueuedFrames() != 3 || c.Available() != 3 {
		t.Fatalf("queued %d, available %d", c.QueuedFrames(), c.Available())
	}
	buf := make([]float32, 8)
	if n := c.Read(buf, 8); n != 3 || buf[0] != 1 || buf[2] != 3 {
		t.Fatalf("read %d: %v", n, buf)
	}
	if n := c.Read(buf, 8); n != 0 {
		t.Errorf("an empty queue gave %d frames", n)
	}
	if n := c.Read(buf, 0); n != 0 {
		t.Errorf("a read of nothing gave %d frames", n)
	}

	// With a change of rate the last frames wait for the frames the filter needs after them, until the stream is flushed.
	c = NewConverter(44100, 1, 48000, 1)
	c.Put(tone(1000, 44100, 1, 100, 0.5))
	before := c.Available()
	if before >= 100*48000/44100 {
		t.Errorf("%d frames are ready from 100, the filter needs more after them", before)
	}
	c.Flush()
	after := c.Available()
	if after <= before || after < 108 {
		t.Errorf("flushed: %d frames are ready, before %d", after, before)
	}
	// Put after Flush is a new sound.
	c.Put(tone(1000, 44100, 1, 100, 0.5))
	if c.Available() > before*3 {
		t.Errorf("the sound after a flush did not start waiting again: %d", c.Available())
	}
	c.Clear()
	if c.QueuedFrames() != 0 || c.Available() != 0 {
		t.Errorf("a cleared converter has %d queued, %d available", c.QueuedFrames(), c.Available())
	}
	// Put of nothing does nothing.
	c.Put(nil)
	if c.QueuedFrames() != 0 {
		t.Error("an empty put queued frames")
	}
}

func TestConverterStreamsForALongTime(t *testing.T) {
	// A second after another second after another: the queue is used up and compacted many times, and the output stays continuous.
	const src, dst = 44100, 48000
	c := NewConverter(src, 2, dst, 2)
	buf := make([]float32, 480*2)
	pos := 0
	var out []float32
	for i := 0; i < 400; i++ {
		c.Put(tone(997, src, 2, 441, 0.5)[:0]) // an empty put in the middle
		piece := make([]float32, 441*2)
		for k := 0; k < 441; k++ {
			v := float32(0.5 * math.Sin(2*math.Pi*997*float64(pos+k)/src))
			piece[2*k], piece[2*k+1] = v, v
		}
		pos += 441
		c.Put(piece)
		for {
			n := c.Read(buf, 480)
			if n == 0 {
				break
			}
			out = append(out, buf[:n*2]...)
		}
	}
	if got := snr(out, 2, 997, dst, 0.5, 100); got < 60 {
		t.Errorf("signal to noise %.1f dB after %d frames", got, len(out)/2)
	}
	if cap(c.buf) > 64*1024 {
		t.Errorf("the buffer grew to %d samples for a queue that never held more than a few thousand", cap(c.buf))
	}
}

func TestReadAndPutDoNotAllocateOnceWarm(t *testing.T) {
	for _, tt := range []struct {
		name                         string
		srcRate, srcCh, dstRate, dch int
	}{
		{"same format", 48000, 2, 48000, 2},
		{"resampling", 44100, 2, 48000, 2},
		{"resampling and mixing down", 44100, 6, 48000, 2},
		{"resampling and mixing up", 22050, 1, 48000, 6},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := NewConverter(tt.srcRate, tt.srcCh, tt.dstRate, tt.dch)
			piece := tone(440, tt.srcRate, tt.srcCh, 441, 0.5)
			buf := make([]float32, 512*tt.dch)
			step := func() {
				c.Put(piece)
				for c.Read(buf, 512) > 0 {
				}
			}
			for i := 0; i < 50; i++ {
				step()
			}
			if got := testing.AllocsPerRun(100, step); got != 0 {
				t.Errorf("%v allocations for a Put and its Reads", got)
			}
		})
	}
}

func TestMixAndClip(t *testing.T) {
	dst := []float32{0.5, -0.5, 0.25, 1}
	Mix(dst, []float32{0.25, 0.25, 0.25, 0.5, 9}, 1)
	if want := []float32{0.75, -0.25, 0.5, 1.5}; fmt.Sprint(dst) != fmt.Sprint(want) {
		t.Errorf("mix = %v, want %v", dst, want)
	}
	Mix(dst, []float32{1, 1}, 0.5)
	if want := []float32{1.25, 0.25, 0.5, 1.5}; fmt.Sprint(dst) != fmt.Sprint(want) {
		t.Errorf("mix with gain = %v, want %v", dst, want)
	}
	Clip(dst)
	if want := []float32{1, 0.25, 0.5, 1}; fmt.Sprint(dst) != fmt.Sprint(want) {
		t.Errorf("clip = %v, want %v", dst, want)
	}
	s := []float32{-3, float32(math.NaN()), 0.5}
	Clip(s)
	if s[0] != -1 || s[1] != 0 || s[2] != 0.5 {
		t.Errorf("clip = %v", s)
	}
	Scale(s, 2)
	if s[2] != 1 {
		t.Errorf("scale = %v", s)
	}
	Scale(s, 1)
	if s[2] != 1 {
		t.Errorf("scale by 1 = %v", s)
	}
	allocs := testing.AllocsPerRun(100, func() {
		Mix(dst, dst, 0.5)
		Clip(dst)
		Scale(dst, 0.5)
	})
	if allocs != 0 {
		t.Errorf("%v allocations", allocs)
	}
}
