package audiocvt

// This file is translated to Go and changed from src/audio/SDL_audiocvt.c and SDL_audioqueue.c of SDL 3 (zlib license,
// third_party/sdl/LICENSE): the audio stream, which queues what is put in and converts it when it is taken out, so that the work is
// done by the thread that needs the samples and a change of speed takes effect at once. SDL's queue of buffers is one slice here,
// and the stream converts only float32; the type conversion is done when samples are put in.

// Converter queues interleaved float32 frames of one format and gives them out in another: another number of channels, another
// rate and, with SetSpeed, another speed. It is not safe for use from several goroutines; package audio holds a lock around it.
// After the first calls (which make its buffers) Read and Put do not allocate unless the queue grows.
//
// Flutter: no counterpart
type Converter struct {
	srcRate, dstRate int
	srcCh, dstCh     int
	speed            float64

	// buf holds the frames: the history (frames that were used up, which the filter still needs), then the queue, buf[r:w].
	buf     []float32
	r, w    int
	flushed bool
	// offset is where in the queue the next output frame is, a fixed point number with 32 bits of fraction.
	offset int64

	// work holds the input of the resampler with its padding, out the resampled frames before their channels change.
	work, out []float32
}

// NewConverter makes a converter from srcCh channels at srcRate to dstCh channels at dstRate. The rates are positive and the
// channels from 1 to MaxChannels.
//
// Flutter: no counterpart
func NewConverter(srcRate, srcCh, dstRate, dstCh int) *Converter {
	c := &Converter{srcRate: srcRate, dstRate: dstRate, srcCh: srcCh, dstCh: dstCh, speed: 1}
	c.reset()
	return c
}

// reset empties the queue; the history is silence.
func (c *Converter) reset() {
	hist := ResamplerHistoryFrames * c.srcCh
	if cap(c.buf) < hist {
		c.buf = make([]float32, hist, hist*8)
	}
	c.buf = c.buf[:hist]
	clear(c.buf)
	c.r, c.w = hist, hist
	c.flushed = false
	c.offset = 0
}

// SetSpeed sets how fast the input is played: 2 takes twice as many frames of input for the same output, and the pitch goes up
// with it. The speed is cut to the range 0.01 to 100.
//
// Flutter: no counterpart
func (c *Converter) SetSpeed(ratio float64) {
	c.speed = min(max(ratio, 0.01), 100)
}

// Speed returns the speed that was set.
//
// Flutter: no counterpart
func (c *Converter) Speed() float64 { return c.speed }

// Put queues frames; samples holds a whole number of frames of the source format. Put after Flush starts a new sound.
//
// Flutter: no counterpart
func (c *Converter) Put(samples []float32) {
	if len(samples) == 0 {
		return
	}
	c.flushed = false
	c.compact()
	c.buf = append(c.buf[:c.w], samples...)
	c.w = len(c.buf)
}

// compact moves the queue and its history to the start of the buffer when a lot of the buffer has been used up.
func (c *Converter) compact() {
	hist := ResamplerHistoryFrames * c.srcCh
	dead := c.r - hist
	// Moving costs a copy of the queue: it is done when the unused part is large and is at least half of what is held.
	if dead < 4096 || dead*2 < c.w {
		return
	}
	n := copy(c.buf, c.buf[dead:c.w])
	c.buf = c.buf[:n]
	c.r, c.w = hist, n
}

// Flush says that no more frames come for now: the last frames are given out too, with silence after them for the filter.
//
// Flutter: no counterpart
func (c *Converter) Flush() { c.flushed = true }

// Clear drops what is queued, the history and the position in the input.
//
// Flutter: no counterpart
func (c *Converter) Clear() { c.reset() }

// QueuedFrames is the number of frames of the source format that are queued.
//
// Flutter: no counterpart
func (c *Converter) QueuedFrames() int { return (c.w - c.r) / c.srcCh }

// resampleRate is the rate of the resampler now, 0 when the frames only change their channels.
func (c *Converter) resampleRate() int64 {
	rate := ResampleRate(int(float32(c.srcRate)*float32(c.speed)), c.dstRate)
	if rate == 1<<32 && c.offset == 0 {
		return 0
	}
	return rate
}

// Available returns how many output frames Read can make from what is queued.
//
// Flutter: no counterpart
func (c *Converter) Available() int {
	queued := int64(c.QueuedFrames())
	rate := c.resampleRate()
	if rate == 0 {
		return int(queued)
	}
	if !c.flushed {
		queued -= ResamplerPaddingFrames
	}
	if queued <= 0 {
		return 0
	}
	offset := c.offset
	return int(min(ResamplerOutputFrames(queued, rate, &offset), 1<<30))
}

// Needed returns how many more frames of the source format have to be queued for Read to make frames output frames, counting the
// frames that the filter needs after them. A callback stream uses it to ask for just what it needs.
//
// Flutter: no counterpart
func (c *Converter) Needed(frames int) int {
	if frames <= 0 {
		return 0
	}
	queued := c.QueuedFrames()
	rate := c.resampleRate()
	if rate == 0 {
		return max(0, frames-queued)
	}
	in := int(ResamplerInputFrames(int64(frames), rate, c.offset)) + ResamplerPaddingFrames
	return max(0, in-queued)
}

// Read converts up to frames output frames into dst (dstCh channels each) and returns how many it made, which is fewer when the
// queue does not have enough. dst must have room for frames frames.
//
// Flutter: no counterpart
func (c *Converter) Read(dst []float32, frames int) int {
	if frames <= 0 {
		return 0
	}
	rate := c.resampleRate()
	if rate == 0 {
		n := min(frames, c.QueuedFrames())
		if n == 0 {
			return 0
		}
		ConvertChannels(dst, c.buf[c.r:], c.srcCh, c.dstCh, n)
		c.r += n * c.srcCh
		return n
	}
	n := min(frames, c.Available())
	if n == 0 {
		return 0
	}
	in := int(ResamplerInputFrames(int64(n), rate, c.offset))
	pad := ResamplerPaddingFrames
	ch := min(c.srcCh, c.dstCh)

	// The input of the resampler, with the history before it and what follows it after: in the channels that the resampler works
	// in (when there are fewer channels afterwards they are made first, so that the interpolation is made of the mixed signal).
	need := (pad + in + pad) * ch
	if cap(c.work) < need {
		c.work = make([]float32, need, need*2)
	}
	work := c.work[:need]
	histStart := c.r - pad*c.srcCh
	avail := c.QueuedFrames()
	realAfter := min(avail-in, pad)                                         // the frames after the input that are queued
	ConvertChannels(work, c.buf[histStart:], c.srcCh, ch, pad+in+realAfter) // a copy when the channels are the same
	clear(work[(pad+in+realAfter)*ch:])

	resampled := dst
	if c.dstCh != ch {
		if cap(c.out) < n*ch {
			c.out = make([]float32, n*ch, n*ch*2)
		}
		resampled = c.out[:n*ch]
	}
	Resample(ch, work, pad, in, resampled, n, rate, &c.offset)
	if c.dstCh != ch {
		ConvertChannels(dst, resampled, ch, c.dstCh, n)
	}
	c.r += in * c.srcCh
	return n
}
