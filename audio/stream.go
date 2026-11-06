package audio

import (
	"fmt"
	"sync"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/audiocvt"
)

// maxCaptureQueue is how much a capture stream keeps that nobody reads, in seconds: past it the oldest sound is dropped.
const maxCaptureQueue = 10

// Stream is a flow of samples into a playback device, which mixes it with the other streams of the device, or out of a capture
// device. A stream has the format of the app's samples; the conversion to the device's format (channels, rate and speed) is done
// when the device needs the samples. Put and Get may be called from any goroutine.
//
// Flutter: no counterpart
type Stream struct {
	dev    *Device
	format Format
	// devFormat is the format of the device's side: of the mix for a playback stream, of the recording for a capture stream.
	devFormat Format
	// fill is the function of a callback stream, nil for a stream that is put into.
	fill func(out []float32)

	mu     sync.Mutex
	conv   *audiocvt.Converter
	gain   float32
	closed bool
	// tmp converts the samples of Put to float32; cb is where a callback stream's function writes. Both are kept between calls.
	tmp, cb []float32
}

// Format returns the format of the samples of the stream: the ones that Put takes, that the function of a callback stream makes,
// or that Get gives.
//
// Flutter: no counterpart
func (st *Stream) Format() Format { return st.format }

// Put queues samples to be played: a []float32 (from -1 to 1), []int16 or []int32, interleaved, in the format of the stream, a whole
// number of frames. It fails with AudioFormatUnsupported for any other type or for samples that are not a whole number of frames,
// AudioStreamWrongKind on the stream of a capture device or a callback stream, AudioStreamClosed on a stream that was closed, and
// AudioDeviceLost when the device is gone. The samples are copied.
//
// Flutter: no counterpart
func (st *Stream) Put(samples any) error {
	if st.dev.kind != Playback || st.fill != nil {
		return errors.New(TagWrongKind, "Put is for the streams of a playback device that are not callback streams", "kind", st.dev.kind.String())
	}
	var n int
	switch s := samples.(type) {
	case []float32:
		n = len(s)
	case []int16:
		n = len(s)
	case []int32:
		n = len(s)
	default:
		return errors.New(TagFormatUnsupported, "a stream takes []float32, []int16 or []int32", "type", typeName(samples))
	}
	if n%st.format.Channels != 0 {
		return errors.New(TagFormatUnsupported, "the samples are not a whole number of frames", "samples", n, "channels", st.format.Channels)
	}
	if st.dev.State() == DeviceStateLost {
		return errors.New(TagDeviceLost, "the device of the stream is gone")
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.closed {
		return errors.New(TagStreamClosed, "the stream was closed")
	}
	switch s := samples.(type) {
	case []float32:
		st.conv.Put(s)
	case []int16:
		st.tmp = growSamples(st.tmp, len(s))
		audiocvt.Int16ToFloat32(st.tmp, s)
		st.conv.Put(st.tmp)
	case []int32:
		st.tmp = growSamples(st.tmp, len(s))
		audiocvt.Int32ToFloat32(st.tmp, s)
		st.conv.Put(st.tmp)
	}
	return nil
}

// Flush says that no more samples come for now, so that the last ones are played out too: a stream whose rate or speed differs
// from the device's holds back the last few frames until the next ones come, which a sound that ends has none of. Put after
// Flush starts a new sound.
//
// Flutter: no counterpart
func (st *Stream) Flush() {
	st.mu.Lock()
	st.conv.Flush()
	st.mu.Unlock()
}

// Get reads the sound that a capture device recorded into out, converted to the format of the stream, and returns the number of
// frames that it wrote (out holds len(out)/Channels frames at most). It does not wait: it returns 0 when nothing is queued. The
// gain of the stream applies. A stream of a playback device returns 0.
//
// Flutter: no counterpart
func (st *Stream) Get(out []float32) int {
	if st.dev.kind != Capture {
		return 0
	}
	frames := len(out) / st.format.Channels
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.closed {
		return 0
	}
	n := st.conv.Read(out, frames)
	audiocvt.Scale(out[:n*st.format.Channels], st.gain)
	return n
}

// Queued returns how much sound is queued: for a playback stream what has been put and not played, for a capture stream what has
// been recorded and not read.
//
// Flutter: no counterpart
func (st *Stream) Queued() time.Duration {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.queuedFormat().Duration(st.conv.QueuedFrames())
}

// queuedFormat is the format of the frames in the queue. The lock is held.
func (st *Stream) queuedFormat() Format {
	if st.dev.kind == Capture {
		return st.devFormat
	}
	return st.format
}

// SetGain sets the volume of the stream: 1 is as it is, 0 is silence; a negative gain is 0. It takes effect at the next frames the
// device asks for.
//
// Flutter: no counterpart
func (st *Stream) SetGain(g float64) {
	st.mu.Lock()
	st.gain = float32(max(g, 0))
	st.mu.Unlock()
}

// Gain returns the volume of the stream.
//
// Flutter: no counterpart
func (st *Stream) Gain() float64 {
	st.mu.Lock()
	defer st.mu.Unlock()
	return float64(st.gain)
}

// SetSpeed sets how fast the stream plays: 2 plays it twice as fast and an octave higher, 0.5 half as fast and an octave lower.
// The ratio is cut to the range from 0.01 to 100.
//
// Flutter: no counterpart
func (st *Stream) SetSpeed(ratio float64) {
	st.mu.Lock()
	st.conv.SetSpeed(ratio)
	st.mu.Unlock()
}

// Speed returns the speed of the stream.
//
// Flutter: no counterpart
func (st *Stream) Speed() float64 {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.conv.Speed()
}

// Clear drops what is queued, so that it is not played (or not read).
//
// Flutter: no counterpart
func (st *Stream) Clear() {
	st.mu.Lock()
	st.conv.Clear()
	st.mu.Unlock()
}

// Close takes the stream out of its device; what is queued is dropped. It may be called more than once.
//
// Flutter: no counterpart
func (st *Stream) Close() {
	st.mu.Lock()
	already := st.closed
	st.closed = true
	st.conv.Clear()
	st.mu.Unlock()
	if !already {
		st.dev.removeStream(st)
	}
}

// mixInto is called on the audio thread by the playback device: it adds the next frames of the stream, with the gain, to the mix.
// scratch has room for frames frames of the device's format. It does not allocate once the buffers have grown.
func (st *Stream) mixInto(mix []float32, frames int, devGain float32, scratch []float32) {
	if st.fill != nil {
		st.refill(frames)
	}
	st.mu.Lock()
	if st.closed {
		st.mu.Unlock()
		return
	}
	n := st.conv.Read(scratch, frames)
	gain := st.gain * devGain
	st.mu.Unlock()
	if n > 0 {
		audiocvt.Mix(mix[:n*st.devFormat.Channels], scratch[:n*st.devFormat.Channels], gain)
	}
}

// refill calls the function of a callback stream for the frames that frames output frames need.
func (st *Stream) refill(frames int) {
	st.mu.Lock()
	need := st.conv.Needed(frames)
	closed := st.closed
	if need > 0 && !closed {
		st.cb = growSamples(st.cb, need*st.format.Channels)
	}
	cb := st.cb
	st.mu.Unlock()
	if need <= 0 || closed {
		return
	}
	buf := cb[:need*st.format.Channels]
	clear(buf)
	st.fill(buf)
	st.mu.Lock()
	if !st.closed {
		st.conv.Put(buf)
	}
	st.mu.Unlock()
}

// capture is called on the audio thread by a capture device with the frames it recorded.
func (st *Stream) capture(in []float32) {
	st.mu.Lock()
	if !st.closed {
		if st.conv.QueuedFrames()+len(in)/st.devFormat.Channels > maxCaptureQueue*st.devFormat.Rate {
			st.conv.Clear()
		}
		st.conv.Put(in)
	}
	st.mu.Unlock()
}

// growSamples returns a slice of n samples, made again only when the old one is too small.
func growSamples(buf []float32, n int) []float32 {
	if cap(buf) < n {
		return make([]float32, n, n*2)
	}
	return buf[:n]
}

func typeName(v any) string { return fmt.Sprintf("%T", v) }
