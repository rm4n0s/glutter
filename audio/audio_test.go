package audio_test

import (
	"context"
	"fmt"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/audio"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
)

// setup makes a tester and its audio, with a playback device open in the default format (48 kHz, stereo).
func setup(t *testing.T, opts ...audio.DeviceOpts) (*gluttertest.Tester, *audio.System, *audio.Device, *gluttertest.TestAudioDevice) {
	t.Helper()
	tester := gluttertest.New(t)
	sys := audio.ForBinding(tester.Binding)
	dev, err := sys.OpenPlayback(context.Background(), opts...)
	if err != nil {
		t.Fatal(err)
	}
	return tester, sys, dev, tester.Audio().Playback()
}

func ramp(n int) []float32 {
	out := make([]float32, n)
	for i := range out {
		out[i] = float32(i+1) / 1024
	}
	return out
}

func TestSystemListsDevicesAndTellsOfChanges(t *testing.T) {
	tester := gluttertest.New(t)
	sys := audio.ForBinding(tester.Binding)
	if !sys.Available() {
		t.Fatal("the test platform has audio")
	}
	play := sys.Devices(audio.Playback)
	if len(play) != 1 || !play[0].Default || play[0].Kind != audio.Playback || play[0].Format != (audio.Format{Rate: 48000, Channels: 2}) || play[0].Name == "" {
		t.Errorf("playback devices = %+v", play)
	}
	if rec := sys.Devices(audio.Capture); len(rec) != 1 || rec[0].Format.Channels != 1 {
		t.Errorf("capture devices = %+v", rec)
	}

	changes := 0
	remove := sys.AddListener(func() { changes++ })
	tester.Audio().AddDevice(audio.DeviceInfo{ID: "usb", Name: "USB headset", Kind: audio.Playback, Default: true, Format: audio.Format{Rate: 44100, Channels: 2}})
	if changes != 1 {
		t.Errorf("a device added told %d times", changes)
	}
	play = sys.Devices(audio.Playback)
	if len(play) != 2 || play[0].Default || !play[1].Default || play[1].ID != "usb" {
		t.Errorf("playback devices = %+v", play)
	}
	tester.Audio().SetDefault("fake-speakers")
	if changes != 2 || !sys.Devices(audio.Playback)[0].Default {
		t.Errorf("the default changed: told %d times, %+v", changes, sys.Devices(audio.Playback))
	}
	tester.Audio().RemoveDevice("usb")
	if changes != 3 || len(sys.Devices(audio.Playback)) != 1 {
		t.Errorf("a device removed: told %d times", changes)
	}
	remove()
	tester.Audio().AddDevice(audio.DeviceInfo{ID: "other", Name: "Other", Kind: audio.Capture})
	if changes != 3 {
		t.Errorf("a listener that was removed was told")
	}
	// The same System is returned for the same binding.
	if audio.ForBinding(tester.Binding) != sys {
		t.Error("another System for the same binding")
	}
}

func TestOpenErrorsHaveTheirRoutes(t *testing.T) {
	tester := gluttertest.New(t)
	sys := audio.ForBinding(tester.Binding)
	tests := []struct {
		name  string
		open  func() error
		tag   string
		route string
	}{
		{"no audio server", func() error {
			tester.Audio().SetUnavailable(true)
			_, err := sys.OpenPlayback(context.Background())
			return err
		}, audio.TagUnavailable, ""},
		{"a format that is too fast", func() error {
			tester.Audio().SetUnavailable(false)
			_, err := sys.OpenPlayback(context.Background(), audio.DeviceOpts{Format: audio.Format{Rate: 10_000_000}})
			return err
		}, audio.TagFormatUnsupported, "System.OpenPlayback->System.open.AudioFormatUnsupported"},
		{"too many channels", func() error {
			_, err := sys.OpenCapture(context.Background(), audio.DeviceOpts{Format: audio.Format{Channels: 9}})
			return err
		}, audio.TagFormatUnsupported, "System.OpenCapture->System.open.AudioFormatUnsupported"},
		{"a negative buffer", func() error {
			_, err := sys.OpenPlayback(context.Background(), audio.DeviceOpts{BufferFrames: -1})
			return err
		}, audio.TagFormatUnsupported, "System.OpenPlayback->System.open.AudioFormatUnsupported"},
		{"a device that is not there", func() error {
			_, err := sys.OpenPlayback(context.Background(), audio.DeviceOpts{ID: "nope"})
			return err
		}, audio.TagDeviceOpenFailed, ""},
		{"a canceled context", func() error {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			_, err := sys.OpenPlayback(ctx)
			return err
		}, audio.TagDeviceOpenFailed, ""},
		{"the user said no", func() error {
			tester.Audio().FailOpen(errors.New(audio.TagCapturePermissionDenied, "the microphone was refused"))
			defer tester.Audio().FailOpen(nil)
			_, err := sys.OpenCapture(context.Background())
			return err
		}, audio.TagCapturePermissionDenied, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.open()
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if appErr.Tag != tt.tag || (tt.route != "" && !appErr.HasRoute(tt.route)) {
				t.Errorf("tag %s, route %s", appErr.Tag, appErr.Route())
			}
		})
	}
	if sys.Devices(audio.Playback) != nil && tester.Audio().Devices() != nil {
		t.Errorf("a failed open left devices: %v", tester.Audio().Devices())
	}
}

func TestOpenWithoutAnAudioServiceOrWithTwoOptions(t *testing.T) {
	tester := gluttertest.New(t)
	tester.Audio().SetUnavailable(true)
	sys := audio.ForBinding(tester.Binding)
	if got := sys.Devices(audio.Playback); len(got) != 0 {
		t.Errorf("devices of a system with no server: %v", got)
	}
	func() {
		defer func() {
			r := recover()
			appErr, ok := r.(*errors.Error)
			if !ok || appErr.Tag != "MultipleOptions" || !appErr.HasRoute("System.OpenPlayback->System.open.MultipleOptions") {
				t.Errorf("recovered %v", r)
			}
		}()
		_, _ = sys.OpenPlayback(context.Background(), audio.DeviceOpts{}, audio.DeviceOpts{})
	}()
}

func TestPlaybackOfOneStream(t *testing.T) {
	_, _, dev, td := setup(t)
	if dev.Format() != (audio.Format{Rate: 48000, Channels: 2}) || dev.Kind() != audio.Playback || dev.State() != audio.DeviceStateRunning {
		t.Fatalf("device = %+v %v %v", dev.Format(), dev.Kind(), dev.State())
	}
	st := dev.NewStream(audio.Format{Rate: 48000, Channels: 2})
	if st.Format() != (audio.Format{Rate: 48000, Channels: 2}) {
		t.Errorf("stream format = %+v", st.Format())
	}
	samples := ramp(200)
	if err := st.Put(samples); err != nil {
		t.Fatal(err)
	}
	if got := st.Queued(); got != (audio.Format{Rate: 48000}).Duration(100) {
		t.Errorf("queued = %v", got)
	}
	got := td.Pull(150, 64)
	for i := 0; i < 200; i++ {
		if got[i] != samples[i] {
			t.Fatalf("sample %d = %v, want %v", i, got[i], samples[i])
		}
	}
	for i := 200; i < 300; i++ {
		if got[i] != 0 {
			t.Fatalf("sample %d after the end = %v", i, got[i])
		}
	}
	if st.Queued() != 0 {
		t.Errorf("queued after playing = %v", st.Queued())
	}
	// Sound that is put later plays after the silence.
	if err := st.Put(samples[:20]); err != nil {
		t.Fatal(err)
	}
	if got := td.Pull(10, 0); got[0] != samples[0] {
		t.Errorf("the next sound began with %v", got[0])
	}
}

func TestStreamTakesEveryTypeOfSamples(t *testing.T) {
	_, _, dev, td := setup(t)
	st := dev.NewStream(audio.Format{Rate: 48000, Channels: 2})
	if err := st.Put([]int16{16384, -16384, -32768, 8192}); err != nil {
		t.Fatal(err)
	}
	if err := st.Put([]int32{1 << 30, -(1 << 30)}); err != nil {
		t.Fatal(err)
	}
	if err := st.Put([]float32{0.125, -0.125}); err != nil {
		t.Fatal(err)
	}
	want := []float32{0.5, -0.5, -1, 0.25, 0.5, -0.5, 0.125, -0.125}
	got := td.Pull(4, 0)
	for i, w := range want {
		if got[i] != w {
			t.Fatalf("sample %d = %v, want %v (%v)", i, got[i], w, got)
		}
	}
}

func TestPutErrorsHaveTheirRoutes(t *testing.T) {
	tester, _, dev, td := setup(t)
	st := dev.NewStream(audio.Format{Rate: 48000, Channels: 2})
	closed := dev.NewStream(audio.Format{Rate: 48000, Channels: 2})
	closed.Close()
	cb := dev.NewCallbackStream(audio.Format{Rate: 48000, Channels: 2}, func([]float32) {})
	rec, err := audio.ForBinding(tester.Binding).OpenCapture(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name  string
		call  func() error
		tag   string
		route string
	}{
		{"bytes", func() error { return st.Put([]byte{1, 2}) }, audio.TagFormatUnsupported, "Stream.Put.AudioFormatUnsupported"},
		{"float64", func() error { return st.Put([]float64{1, 2}) }, audio.TagFormatUnsupported, "Stream.Put.AudioFormatUnsupported"},
		{"a partial frame", func() error { return st.Put([]float32{1, 2, 3}) }, audio.TagFormatUnsupported, "Stream.Put.AudioFormatUnsupported"},
		{"a closed stream", func() error { return closed.Put([]float32{1, 2}) }, audio.TagStreamClosed, "Stream.Put.AudioStreamClosed"},
		{"a callback stream", func() error { return cb.Put([]float32{1, 2}) }, audio.TagWrongKind, "Stream.Put.AudioStreamWrongKind"},
		{"a capture stream", func() error { return rec.NewStream(audio.Format{Rate: 48000, Channels: 1}).Put([]float32{1}) }, audio.TagWrongKind, "Stream.Put.AudioStreamWrongKind"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if appErr.Tag != tt.tag || !appErr.HasRoute(tt.route) {
				t.Errorf("unexpected failure: tag %s, route %s", appErr.Tag, appErr.Route())
			}
		})
	}
	// A device that is gone: the sound cannot be played any more.
	td.Lose()
	err = st.Put([]float32{1, 2})
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != audio.TagDeviceLost || !appErr.HasRoute("Stream.Put.AudioDeviceLost") {
		t.Errorf("a lost device: %v", err)
	}
}

func TestMixingTwoStreamsAddsAndClips(t *testing.T) {
	_, _, dev, td := setup(t)
	a := dev.NewStream(audio.Format{Rate: 48000, Channels: 2})
	b := dev.NewStream(audio.Format{Rate: 48000, Channels: 2})
	if err := a.Put([]float32{0.25, -0.25, 0.75, 0.5}); err != nil {
		t.Fatal(err)
	}
	if err := b.Put([]float32{0.25, 0.25, 0.75, -1.5}); err != nil {
		t.Fatal(err)
	}
	got := td.Pull(2, 0)
	want := []float32{0.5, 0, 1, -1} // 0.75+0.75 is cut to 1, 0.5-1.5 to -1
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("mix = %v, want %v", got, want)
		}
	}
	// The mix is cut once, after the sum: two loud streams are not cut before they are added to a third that cancels them.
	c := dev.NewStream(audio.Format{Rate: 48000, Channels: 2})
	for _, st := range []*audio.Stream{a, b, c} {
		st.Clear()
	}
	_ = a.Put([]float32{0.9, 0.9})
	_ = b.Put([]float32{0.9, 0.9})
	_ = c.Put([]float32{-1.8, -1.8})
	if got := td.Pull(1, 0); math.Abs(float64(got[0])) > 1e-6 {
		t.Errorf("0.9 + 0.9 - 1.8 = %v", got[0])
	}
}

func TestGainsSpeedAndClear(t *testing.T) {
	_, _, dev, td := setup(t)
	st := dev.NewStream(audio.Format{Rate: 48000, Channels: 2})
	if st.Gain() != 1 || st.Speed() != 1 || dev.Gain() != 1 {
		t.Fatalf("gain %v, speed %v, device gain %v", st.Gain(), st.Speed(), dev.Gain())
	}
	st.SetGain(0.5)
	dev.SetGain(0.5)
	_ = st.Put([]float32{0.8, -0.8})
	if got := td.Pull(1, 0); math.Abs(float64(got[0]-0.2)) > 1e-6 || math.Abs(float64(got[1]+0.2)) > 1e-6 {
		t.Errorf("gain 0.5 of 0.5 of 0.8 = %v", got)
	}
	st.SetGain(-3)
	dev.SetGain(-3)
	if st.Gain() != 0 || dev.Gain() != 0 {
		t.Errorf("a negative gain is %v and %v", st.Gain(), dev.Gain())
	}
	st.SetGain(1)
	dev.SetGain(1)
	// Twice the speed takes half the time.
	st.SetSpeed(2)
	if st.Speed() != 2 {
		t.Errorf("speed = %v", st.Speed())
	}
	st.SetSpeed(1000)
	if st.Speed() != 100 {
		t.Errorf("speed 1000 is %v", st.Speed())
	}
	st.SetSpeed(2)
	frames := 4800
	tone := make([]float32, frames*2)
	for i := 0; i < frames; i++ {
		v := float32(0.5 * math.Sin(2*math.Pi*480*float64(i)/48000))
		tone[2*i], tone[2*i+1] = v, v
	}
	_ = st.Put(tone)
	st.Flush()
	out := td.Pull(frames, 0)
	end := frames
	for end > 0 && out[2*(end-1)] == 0 {
		end--
	}
	if end < frames/2-4 || end > frames/2+4 {
		t.Errorf("the sound at twice the speed lasted %d frames, want about %d", end, frames/2)
	}
	// Clear drops what is queued.
	st.SetSpeed(1)
	_ = st.Put(ramp(400))
	st.Clear()
	if st.Queued() != 0 {
		t.Errorf("queued after Clear = %v", st.Queued())
	}
	for _, v := range td.Pull(10, 0) {
		if v != 0 {
			t.Fatalf("a cleared stream played %v", v)
		}
	}
}

func TestStreamOfAnotherFormatIsConverted(t *testing.T) {
	_, _, dev, td := setup(t)
	// Mono at 24 kHz into a stereo device at 48 kHz: the sound is twice as long, in both channels.
	st := dev.NewStream(audio.Format{Rate: 24000, Channels: 1})
	const frames = 4800
	in := make([]float32, frames)
	for i := range in {
		in[i] = float32(0.5 * math.Sin(2*math.Pi*1000*float64(i)/24000))
	}
	if err := st.Put(in); err != nil {
		t.Fatal(err)
	}
	st.Flush()
	out := td.Pull(2*frames, 480)
	var sig, noise float64
	for i := 100; i < 2*frames-100; i++ {
		want := 0.5 * math.Sin(2*math.Pi*1000*float64(i)/48000)
		if out[2*i] != out[2*i+1] {
			t.Fatalf("the channels differ at %d: %v %v", i, out[2*i], out[2*i+1])
		}
		sig += want * want
		noise += (float64(out[2*i]) - want) * (float64(out[2*i]) - want)
	}
	if snr := 10 * math.Log10(sig/noise); snr < 70 {
		t.Errorf("signal to noise %.1f dB", snr)
	}
	// The stream's queue is in the stream's own rate.
	_ = st.Put(make([]float32, 2400))
	if got := st.Queued(); got != 100*time.Millisecond {
		t.Errorf("queued = %v, want 100ms", got)
	}
}

func TestPauseInBackgroundIsOnForAndroidByDefault(t *testing.T) {
	tests := []struct {
		name     string
		platform foundation.Opt[foundation.TargetPlatform]
		opts     audio.DeviceOpts
		want     bool
	}{
		{"the default of the machine", foundation.None[foundation.TargetPlatform](), audio.DeviceOpts{}, false},
		{"Android", foundation.Some(foundation.TargetPlatformAndroid), audio.DeviceOpts{}, true},
		{"Android, said no", foundation.Some(foundation.TargetPlatformAndroid), audio.DeviceOpts{PauseInBackground: foundation.Some(false)}, false},
		{"iOS", foundation.Some(foundation.TargetPlatformIOS), audio.DeviceOpts{}, false},
		{"Linux, said yes", foundation.Some(foundation.TargetPlatformLinux), audio.DeviceOpts{PauseInBackground: foundation.Some(true)}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			foundation.SetDebugDefaultTargetPlatformOverride(tt.platform)
			t.Cleanup(func() { foundation.SetDebugDefaultTargetPlatformOverride(foundation.None[foundation.TargetPlatform]()) })
			_, _, _, td := setup(t, tt.opts)
			if td.PauseInBackground() != tt.want {
				t.Errorf("the backend got PauseInBackground %v, want %v", td.PauseInBackground(), tt.want)
			}
		})
	}
}

func TestDeviceFormatFollowsTheOptions(t *testing.T) {
	_, _, dev, td := setup(t, audio.DeviceOpts{Format: audio.Format{Rate: 44100, Channels: 1}, BufferFrames: 512, PauseInBackground: foundation.Some(true)})
	if dev.Format() != (audio.Format{Rate: 44100, Channels: 1}) || td.Format() != dev.Format() {
		t.Errorf("format = %+v, backend %+v", dev.Format(), td.Format())
	}
	if td.BufferFrames() != 512 || !td.PauseInBackground() {
		t.Errorf("the backend got buffer %d, pause %v", td.BufferFrames(), td.PauseInBackground())
	}
	st := dev.NewStream(audio.Format{Rate: 44100, Channels: 1})
	_ = st.Put([]float32{0.5, 0.25})
	if got := td.Pull(2, 0); len(got) != 2 || got[0] != 0.5 || got[1] != 0.25 {
		t.Errorf("mono device played %v", got)
	}
}

func TestCallbackStreams(t *testing.T) {
	_, _, dev, td := setup(t)
	var calls, asked int
	phase := 0
	st := dev.NewCallbackStream(audio.Format{Rate: 48000, Channels: 1}, func(out []float32) {
		calls++
		asked += len(out)
		for i := range out {
			out[i] = float32(phase+i) / 1000
		}
		phase += len(out)
	})
	got := td.Pull(300, 100)
	if calls == 0 || asked != 300 {
		t.Errorf("the function was called %d times for %d frames, want 300", calls, asked)
	}
	for i := 0; i < 300; i++ {
		want := float32(i) / 1000
		if got[2*i] != want || got[2*i+1] != want {
			t.Fatalf("frame %d = %v %v, want %v", i, got[2*i], got[2*i+1], want)
		}
	}
	// A function that leaves part of the buffer alone makes silence there.
	st.Close()
	dev.NewCallbackStream(audio.Format{Rate: 48000, Channels: 1}, func(out []float32) { out[0] = 0.5 })
	got = td.Pull(4, 0)
	if got[0] != 0.5 || got[2] != 0 || got[4] != 0 {
		t.Errorf("a short fill played %v", got)
	}
	// A callback stream in another format is asked for the frames the conversion needs.
	asked = 0
	dev.NewCallbackStream(audio.Format{Rate: 24000, Channels: 2}, func(out []float32) { asked += len(out) / 2 })
	td.Pull(4800, 480)
	if asked < 2400 || asked > 2400+64 {
		t.Errorf("a stream at half the rate was asked for %d frames, want about 2400", asked)
	}
}

func TestDevicePauseResumeAndState(t *testing.T) {
	tester, _, dev, td := setup(t)
	changes := 0
	remove := dev.AddListener(func() { changes++ })
	st := dev.NewStream(audio.Format{Rate: 48000, Channels: 2})
	_ = st.Put(ramp(40))
	if err := dev.Pause(); err != nil {
		t.Fatal(err)
	}
	tester.Pump(0)
	if dev.State() != audio.DeviceStatePaused || changes != 1 {
		t.Errorf("paused: state %v, %d changes", dev.State(), changes)
	}
	for _, v := range td.Pull(10, 0) {
		if v != 0 {
			t.Fatal("a paused device played sound")
		}
	}
	if err := dev.Resume(); err != nil {
		t.Fatal(err)
	}
	tester.Pump(0)
	if dev.State() != audio.DeviceStateRunning || changes != 2 {
		t.Errorf("resumed: state %v, %d changes", dev.State(), changes)
	}
	if got := td.Pull(1, 0); got[0] != ramp(2)[0] {
		t.Errorf("the sound that was queued is %v", got)
	}
	// A device that the platform suspends, as a browser's, and lets run.
	td.Suspend()
	if dev.State() != audio.DeviceStateSuspended || changes != 3 {
		t.Errorf("suspended: state %v, %d changes", dev.State(), changes)
	}
	td.Unsuspend()
	if dev.State() != audio.DeviceStateRunning || changes != 4 {
		t.Errorf("unsuspended: state %v, %d changes", dev.State(), changes)
	}
	remove()
	td.Lose()
	if dev.State() != audio.DeviceStateLost || changes != 4 {
		t.Errorf("lost: state %v, %d changes", dev.State(), changes)
	}
	// Closing a lost device works, and so does a second Close.
	if err := dev.Close(); err != nil {
		t.Fatal(err)
	}
	if err := dev.Close(); err != nil {
		t.Errorf("the second Close failed: %v", err)
	}
	if dev.State() != audio.DeviceStateClosed {
		t.Errorf("closed: state %v", dev.State())
	}
	if err := dev.Pause(); err != nil || dev.Resume() != nil {
		t.Errorf("Pause or Resume of a closed device failed")
	}
	if err := st.Put([]float32{1, 1}); err == nil {
		t.Error("a stream of a closed device took sound")
	}
}

func TestSystemCloseClosesTheDevicesOfTheApp(t *testing.T) {
	tester := gluttertest.New(t)
	sys := audio.ForBinding(tester.Binding)
	a, _ := sys.OpenPlayback(context.Background())
	b, _ := sys.OpenCapture(context.Background())
	if len(tester.Audio().Devices()) != 2 {
		t.Fatalf("%d devices are open", len(tester.Audio().Devices()))
	}
	// The host calls the release hooks when the app ends.
	tester.RunReleaseHooks()
	if a.State() != audio.DeviceStateClosed || b.State() != audio.DeviceStateClosed || len(tester.Audio().Devices()) != 0 {
		t.Errorf("states %v %v, %d devices left", a.State(), b.State(), len(tester.Audio().Devices()))
	}
	if _, err := sys.OpenPlayback(context.Background()); err == nil {
		t.Error("a closed System opened a device")
	}
	sys.Close() // twice is fine
}

func TestCaptureStreams(t *testing.T) {
	tester := gluttertest.New(t)
	sys := audio.ForBinding(tester.Binding)
	dev, err := sys.OpenCapture(context.Background(), audio.DeviceOpts{Format: audio.Format{Rate: 48000, Channels: 1}})
	if err != nil {
		t.Fatal(err)
	}
	td := tester.Audio().Capture()
	same := dev.NewStream(audio.Format{Rate: 48000, Channels: 1})
	stereo := dev.NewStream(audio.Format{Rate: 48000, Channels: 2})
	half := dev.NewStream(audio.Format{Rate: 24000, Channels: 1})
	if got := same.Get(make([]float32, 8)); got != 0 {
		t.Errorf("an empty stream gave %d frames", got)
	}
	in := make([]float32, 4800)
	for i := range in {
		in[i] = float32(0.5 * math.Sin(2*math.Pi*500*float64(i)/48000))
	}
	td.Push(in)
	if got := same.Queued(); got != 100*time.Millisecond {
		t.Errorf("queued = %v", got)
	}
	out := make([]float32, 4800)
	if n := same.Get(out); n != 4800 || out[100] != in[100] {
		t.Errorf("same format: %d frames, %v", n, out[100])
	}
	out2 := make([]float32, 4800*2)
	if n := stereo.Get(out2); n != 4800 || out2[2*100] != in[100] || out2[2*100+1] != in[100] {
		t.Errorf("stereo: %d frames", n)
	}
	// At half the rate there are half the frames, once the filter has what it needs after them.
	n := half.Get(out)
	if n < 2400-8 || n > 2400 {
		t.Errorf("half the rate: %d frames, want about 2400", n)
	}
	for i := 50; i < n-50; i++ {
		want := 0.5 * math.Sin(2*math.Pi*500*float64(i)/24000)
		if math.Abs(float64(out[i])-want) > 1e-3 {
			t.Fatalf("frame %d = %v, want %v", i, out[i], want)
		}
	}
	// The gain of the stream applies to what it gives.
	same.SetGain(0.5)
	td.Push(in[:100])
	if n := same.Get(out); n != 100 || math.Abs(float64(out[50]-in[50]*0.5)) > 1e-7 {
		t.Errorf("gain: %d frames, %v", n, out[50])
	}
	// Get on a playback stream gives nothing.
	play, _ := sys.OpenPlayback(context.Background())
	if play.NewStream(audio.Format{Rate: 48000, Channels: 1}).Get(out) != 0 {
		t.Error("Get on a playback stream gave frames")
	}
	// A stream that nobody reads does not hold on to more than ten seconds.
	slow := dev.NewStream(audio.Format{Rate: 48000, Channels: 1})
	chunk := make([]float32, 48000)
	for i := 0; i < 15; i++ {
		td.Push(chunk)
	}
	if got := slow.Queued(); got > 10*time.Second {
		t.Errorf("an unread stream holds %v", got)
	}
	// A paused capture device drops what it records; a closed stream gives nothing.
	_ = dev.Pause()
	slow.Clear()
	td.Push(chunk)
	if slow.Queued() != 0 {
		t.Error("a paused device recorded")
	}
	slow.Close()
	if slow.Get(out) != 0 {
		t.Error("a closed stream gave frames")
	}
}

func TestStreamsFromSeveralGoroutines(t *testing.T) {
	_, _, dev, td := setup(t)
	st := dev.NewStream(audio.Format{Rate: 44100, Channels: 2})
	var wg sync.WaitGroup
	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			piece := make([]float32, 882)
			for i := 0; i < 200; i++ {
				_ = st.Put(piece)
				_ = st.Queued()
				st.SetGain(0.9)
			}
		}()
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]float32, 480*2)
		for i := 0; i < 400; i++ {
			td.PullInto(buf)
		}
	}()
	wg.Wait()
	<-done
	other := dev.NewStream(audio.Format{Rate: 48000, Channels: 2})
	other.Close()
	st.Close()
}

func TestInvalidFormatsPanic(t *testing.T) {
	_, _, dev, _ := setup(t)
	tests := []struct {
		name string
		call func()
	}{
		{"zero rate", func() { dev.NewStream(audio.Format{Channels: 2}) }},
		{"too many channels", func() { dev.NewStream(audio.Format{Rate: 48000, Channels: 9}) }},
		{"a callback stream with no function", func() { dev.NewCallbackStream(audio.Format{Rate: 48000, Channels: 2}, nil) }},
		{"a callback stream with a bad format", func() { dev.NewCallbackStream(audio.Format{Rate: 5, Channels: 2}, func([]float32) {}) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				appErr, ok := recover().(*errors.Error)
				if !ok || appErr.Tag != audio.TagFormatUnsupported {
					t.Errorf("recovered %v, want AudioFormatUnsupported", appErr)
				}
			}()
			tt.call()
		})
	}
}

func TestFormatHelpers(t *testing.T) {
	tests := []struct {
		f     audio.Format
		valid bool
	}{
		{audio.Format{Rate: 48000, Channels: 2}, true}, {audio.Format{Rate: 1000, Channels: 1}, true}, {audio.Format{Rate: 768000, Channels: 8}, true},
		{audio.Format{Rate: 999, Channels: 1}, false}, {audio.Format{Rate: 768001, Channels: 1}, false}, {audio.Format{Rate: 48000, Channels: 0}, false},
		{audio.Format{Rate: 48000, Channels: 9}, false}, {audio.Format{}, false},
	}
	for _, tt := range tests {
		if tt.f.Valid() != tt.valid {
			t.Errorf("%+v.Valid() = %v", tt.f, !tt.valid)
		}
	}
	f := audio.Format{Rate: 48000, Channels: 2}
	if f.Frames(time.Second) != 48000 || f.Duration(24000) != 500*time.Millisecond || (audio.Format{}).Duration(5) != 0 {
		t.Error("Frames or Duration is wrong")
	}
	if audio.Playback.String() != "playback" || audio.Capture.String() != "capture" {
		t.Error("DeviceKind.String is wrong")
	}
	_ = fmt.Sprint(f)
}

// TestMixDoesNotAllocate: once the buffers have grown, the callback of a device allocates nothing, whatever its streams are. (The
// Put of the app is not measured: it boxes the slice in an interface, which is the caller's allocation.)
func TestMixDoesNotAllocate(t *testing.T) {
	tests := []struct {
		name string
		make func(dev *audio.Device)
	}{
		{"one stream of the device's format", func(dev *audio.Device) {
			_ = dev.NewStream(audio.Format{Rate: 48000, Channels: 2}).Put(make([]float32, 2*48000*10))
		}},
		{"streams of other formats", func(dev *audio.Device) {
			a := dev.NewStream(audio.Format{Rate: 44100, Channels: 1})
			b := dev.NewStream(audio.Format{Rate: 22050, Channels: 6})
			b.SetSpeed(1.25)
			_ = a.Put(make([]float32, 44100*10))
			_ = b.Put(make([]float32, 22050*6*10))
		}},
		{"callback streams", func(dev *audio.Device) {
			dev.NewCallbackStream(audio.Format{Rate: 32000, Channels: 2}, func(out []float32) {
				for i := range out {
					out[i] = 0.1
				}
			})
			dev.NewCallbackStream(audio.Format{Rate: 48000, Channels: 2}, func([]float32) {})
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, dev, td := setup(t)
			tt.make(dev)
			buf := make([]float32, 480*2)
			step := func() { td.PullInto(buf) }
			for i := 0; i < 100; i++ {
				step()
			}
			if got := testing.AllocsPerRun(200, step); got != 0 {
				t.Errorf("%v allocations for a callback", got)
			}
		})
	}
}
