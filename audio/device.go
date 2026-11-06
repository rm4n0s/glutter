package audio

import (
	"math"
	"sync"
	"sync/atomic"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/audiocvt"
	"github.com/rm4n0s/glutter/internal/platform"
)

// Device is an open playback or capture device. Streams made with NewStream are mixed into a playback device (or fed from a capture
// device). It is a platform.AudioHandler: the backend calls it on the audio thread.
//
// Flutter: no counterpart
type Device struct {
	sys    *System
	kind   DeviceKind
	format Format
	plat   platform.AudioDevice

	gain    atomic.Uint32
	state   atomic.Uint32
	streams atomic.Pointer[[]*Stream]
	// smu serializes the changes of the list of streams.
	smu sync.Mutex

	// scratch is used only by the audio thread: the room for the frames of one stream.
	scratch []float32

	// notifier calls the listeners of the state on the UI thread.
	notifier foundation.ChangeNotifier
	closeMu  sync.Mutex
}

// math32Bits1 is the bits of the float32 1, the gain of a device that was not changed.
const math32Bits1 = 0x3f800000

var _ platform.AudioHandler = (*Device)(nil)

// Kind says whether the device plays or records.
//
// Flutter: no counterpart
func (d *Device) Kind() DeviceKind { return d.kind }

// Format returns the format of the device: the one the streams are converted to (or from). It may differ from the one that was
// asked for in DeviceOpts, and does not change while the device is open.
//
// Flutter: no counterpart
func (d *Device) Format() Format { return d.format }

// State returns the state of the device. It may be called from any goroutine.
//
// Flutter: no counterpart
func (d *Device) State() DeviceState { return DeviceState(d.state.Load()) }

// AddListener adds a function that is called, on the UI thread, when the state of the device changes (it is paused or resumed,
// suspended or running again, lost or closed). It returns the function that removes it.
//
// Flutter: no counterpart
func (d *Device) AddListener(fn func()) (remove func()) {
	h := d.notifier.AddListener(fn)
	return func() { d.notifier.RemoveListener(h) }
}

// SetGain sets the volume of the whole device: 1 is as it is, 0 is silence; a negative gain is 0. It is applied to the mix of the
// streams, which are clipped after it.
//
// Flutter: no counterpart
func (d *Device) SetGain(g float64) { d.gain.Store(math.Float32bits(float32(max(g, 0)))) }

// Gain returns the volume of the device.
//
// Flutter: no counterpart
func (d *Device) Gain() float64 { return float64(math.Float32frombits(d.gain.Load())) }

// NewStream makes a stream of the given format for the device. The streams of a playback device are played mixed together;
// the format is the one that Put takes. For a capture device the format is the one that Get gives. It panics with
// AudioFormatUnsupported when the format is outside the limits (a rate from MinRate to MaxRate and 1 to MaxChannels channels).
//
// Flutter: no counterpart
func (d *Device) NewStream(format Format) *Stream {
	if !format.Valid() {
		panic(errors.New(TagFormatUnsupported, "the format of a stream has a rate from 1000 to 768000 and 1 to 8 channels", "rate", format.Rate, "channels", format.Channels))
	}
	return d.addStream(format, nil)
}

// NewCallbackStream makes a stream of a playback device whose samples come from fill: the device calls it, on the audio thread,
// when it needs more, with a buffer of interleaved float32 frames in the format of the stream, to be filled completely (what it
// leaves is silence). fill must not touch the framework, wait for the UI thread or allocate. A callback stream has no Put. It panics
// with AudioFormatUnsupported when the format is outside the limits or the device records.
//
// Flutter: no counterpart
func (d *Device) NewCallbackStream(format Format, fill func(out []float32)) *Stream {
	if !format.Valid() || d.kind != Playback || fill == nil {
		panic(errors.New(TagFormatUnsupported, "a callback stream has a valid format, a function and a playback device", "rate", format.Rate, "channels", format.Channels, "kind", d.kind.String()))
	}
	return d.addStream(format, fill)
}

func (d *Device) addStream(format Format, fill func(out []float32)) *Stream {
	st := &Stream{dev: d, format: format, devFormat: d.format, fill: fill, gain: 1}
	if d.kind == Playback {
		st.conv = audiocvt.NewConverter(format.Rate, format.Channels, d.format.Rate, d.format.Channels)
	} else {
		st.conv = audiocvt.NewConverter(d.format.Rate, d.format.Channels, format.Rate, format.Channels)
	}
	d.smu.Lock()
	var next []*Stream
	if cur := d.streams.Load(); cur != nil {
		next = append(next, *cur...)
	}
	next = append(next, st)
	d.streams.Store(&next)
	d.smu.Unlock()
	return st
}

func (d *Device) removeStream(st *Stream) {
	d.smu.Lock()
	defer d.smu.Unlock()
	cur := d.streams.Load()
	if cur == nil {
		return
	}
	next := make([]*Stream, 0, len(*cur))
	for _, s := range *cur {
		if s != st {
			next = append(next, s)
		}
	}
	d.streams.Store(&next)
}

// Pause stops the device from playing or recording; Resume starts it again. The streams keep what they have queued.
//
// Flutter: no counterpart
func (d *Device) Pause() error {
	if d.State() == DeviceStateClosed {
		return nil
	}
	if err := d.plat.Pause(); err != nil {
		return err
	}
	d.setState(DeviceStatePaused)
	return nil
}

// Resume starts a paused device again.
//
// Flutter: no counterpart
func (d *Device) Resume() error {
	if d.State() == DeviceStateClosed {
		return nil
	}
	if err := d.plat.Resume(); err != nil {
		return err
	}
	if d.State() == DeviceStatePaused {
		d.setState(DeviceStateRunning)
	}
	return nil
}

// Close ends the device and its streams. When it returns no more of the app's callbacks run. It may be called more than once.
//
// Flutter: no counterpart
func (d *Device) Close() error {
	d.closeMu.Lock()
	defer d.closeMu.Unlock()
	if d.State() == DeviceStateClosed {
		return nil
	}
	err := d.plat.Close()
	d.setState(DeviceStateClosed)
	d.smu.Lock()
	cur := d.streams.Swap(nil)
	d.smu.Unlock()
	if cur != nil {
		for _, st := range *cur {
			st.mu.Lock()
			st.closed = true
			st.conv.Clear()
			st.mu.Unlock()
		}
	}
	d.sys.forget(d)
	return err
}

// setState records a state and tells the listeners, on the UI thread.
func (d *Device) setState(s DeviceState) {
	if DeviceState(d.state.Swap(uint32(s))) == s {
		return
	}
	d.sys.post(func() { d.notifier.NotifyListeners() })
}

// Fill implements platform.AudioHandler: it is the audio thread, and mixes the streams into out.
func (d *Device) Fill(out []float32) {
	clear(out)
	cur := d.streams.Load()
	if cur == nil || len(*cur) == 0 {
		return
	}
	ch := d.format.Channels
	frames := len(out) / ch
	if cap(d.scratch) < len(out) {
		d.scratch = make([]float32, len(out), len(out)*2)
	}
	scratch := d.scratch[:len(out)]
	gain := math.Float32frombits(d.gain.Load())
	for _, st := range *cur {
		st.mixInto(out, frames, gain, scratch)
	}
	audiocvt.Clip(out)
}

// Capture implements platform.AudioHandler: it is the audio thread, and gives the recording to the streams.
func (d *Device) Capture(in []float32) {
	cur := d.streams.Load()
	if cur == nil {
		return
	}
	for _, st := range *cur {
		st.capture(in)
	}
}

// StateChanged implements platform.AudioHandler.
func (d *Device) StateChanged(s platform.AudioState) {
	if d.State() == DeviceStateClosed {
		return
	}
	d.setState(DeviceState(s))
}
