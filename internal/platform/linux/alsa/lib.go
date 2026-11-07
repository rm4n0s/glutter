//go:build linux && !android

package alsa

import (
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// LibraryEnv names the environment variable that overrides the path of libasound, for setups such as NixOS (SPEC §8.1).
const LibraryEnv = "GLUTTER_ALSA_LIBRARY"

// Constants from the headers of alsa-lib (alsa/pcm.h).
const (
	streamPlayback = 0 // SND_PCM_STREAM_PLAYBACK
	streamCapture  = 1 // SND_PCM_STREAM_CAPTURE

	openNonblock = 1 // SND_PCM_NONBLOCK

	formatFloatLE = 14 // SND_PCM_FORMAT_FLOAT_LE
	accessRWInter = 3  // SND_PCM_ACCESS_RW_INTERLEAVED

	stateDisconnected = 8 // SND_PCM_STATE_DISCONNECTED
)

// The errors of the C library that the backend tells apart (errno).
const (
	errPerm    = 1  // EPERM
	errAccess  = 13 // EACCES
	errAgain   = 11 // EAGAIN
	errNoDev   = 19 // ENODEV
	errInval   = 22 // EINVAL
	errPipe    = 32 // EPIPE: an xrun
	errStrPipe = 86 // ESTRPIPE: the device was suspended
)

// library holds the bound functions of libasound. The PCM handles are uintptr; the buffers of frames are Go memory that ALSA only
// reads or writes during the call, so they go as unsafe.Pointer; strings that go in are C strings. The C functions of the hints
// return memory that is freed with free, which is the last field.
type library struct {
	open        func(pcmp uintptr, name uintptr, stream, mode int32) int32
	close       func(pcm uintptr) int32
	setParams   func(pcm uintptr, format, access int32, channels, rate uint32, softResample int32, latency uint32) int32
	getParams   func(pcm uintptr, bufferSize, periodSize uintptr) int32
	wait        func(pcm uintptr, timeout int32) int32
	availUpdate func(pcm uintptr) int64
	writei      func(pcm uintptr, buf unsafe.Pointer, frames uint64) int64
	readi       func(pcm uintptr, buf unsafe.Pointer, frames uint64) int64
	recover     func(pcm uintptr, err, silent int32) int32
	prepare     func(pcm uintptr) int32
	start       func(pcm uintptr) int32
	drop        func(pcm uintptr) int32
	state       func(pcm uintptr) int32
	strerror    func(err int32) uintptr

	nameHint     func(card int32, iface uintptr, hints uintptr) int32
	nameGetHint  func(hint uintptr, id uintptr) uintptr
	nameFreeHint func(hints uintptr) int32
	free         func(ptr uintptr)
}

// loadLibrary opens libasound.so.2 and binds the functions. It fails with AudioUnavailable when the library cannot be loaded or
// lacks a function.
func loadLibrary() (*library, error) {
	lib, err := dl.OpenOverride(LibraryEnv, "libasound.so.2", "libasound.so")
	if err != nil {
		md := map[string]any{}
		if appErr, ok := errors.FromError(err); ok {
			md = appErr.Metadata
		}
		return nil, errors.New(platform.TagAudioUnavailable, "could not load libasound", "names", md["names"], "dlerror", md["dlerror"])
	}
	libc, err := dl.Open("libc.so.6")
	if err != nil {
		md := map[string]any{}
		if appErr, ok := errors.FromError(err); ok {
			md = appErr.Metadata
		}
		return nil, errors.New(platform.TagAudioUnavailable, "could not load libc", "names", md["names"], "dlerror", md["dlerror"])
	}
	l := new(library)
	for _, b := range []struct {
		lib    *dl.Lib
		fptr   any
		symbol string
	}{
		{lib, &l.open, "snd_pcm_open"},
		{lib, &l.close, "snd_pcm_close"},
		{lib, &l.setParams, "snd_pcm_set_params"},
		{lib, &l.getParams, "snd_pcm_get_params"},
		{lib, &l.wait, "snd_pcm_wait"},
		{lib, &l.availUpdate, "snd_pcm_avail_update"},
		{lib, &l.writei, "snd_pcm_writei"},
		{lib, &l.readi, "snd_pcm_readi"},
		{lib, &l.recover, "snd_pcm_recover"},
		{lib, &l.prepare, "snd_pcm_prepare"},
		{lib, &l.start, "snd_pcm_start"},
		{lib, &l.drop, "snd_pcm_drop"},
		{lib, &l.state, "snd_pcm_state"},
		{lib, &l.strerror, "snd_strerror"},
		{lib, &l.nameHint, "snd_device_name_hint"},
		{lib, &l.nameGetHint, "snd_device_name_get_hint"},
		{lib, &l.nameFreeHint, "snd_device_name_free_hint"},
		{libc, &l.free, "free"},
	} {
		if err := b.lib.Bind(b.fptr, b.symbol); err != nil {
			md := map[string]any{}
			if appErr, ok := errors.FromError(err); ok {
				md = appErr.Metadata
			}
			return nil, errors.New(platform.TagAudioUnavailable, "libasound lacks a function", "library", md["library"], "symbol", md["symbol"])
		}
	}
	return l, nil
}
