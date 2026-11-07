//go:build linux && !android

package pulse

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// LibraryEnv names the environment variable that overrides the path of libpulse, for setups such as NixOS (SPEC §8.1).
const LibraryEnv = "GLUTTER_PULSE_LIBRARY"

// Constants from the headers of libpulse (pulse/def.h, sample.h, context.h, stream.h, subscribe.h).
const (
	contextNoAutoSpawn = 0x0002 // PA_CONTEXT_NOAUTOSPAWN

	contextReady      = 4 // PA_CONTEXT_READY
	contextFailed     = 5 // PA_CONTEXT_FAILED
	contextTerminated = 6 // PA_CONTEXT_TERMINATED

	streamReady      = 2 // PA_STREAM_READY
	streamFailed     = 3 // PA_STREAM_FAILED
	streamTerminated = 4 // PA_STREAM_TERMINATED

	operationRunning = 0 // PA_OPERATION_RUNNING

	sampleFloat32LE = 5 // PA_SAMPLE_FLOAT32LE

	streamAdjustLatency = 0x2000 // PA_STREAM_ADJUST_LATENCY

	seekRelative = 0 // PA_SEEK_RELATIVE

	subscriptionMaskSink   = 0x0001 // PA_SUBSCRIPTION_MASK_SINK
	subscriptionMaskSource = 0x0002 // PA_SUBSCRIPTION_MASK_SOURCE
	subscriptionMaskServer = 0x0080 // PA_SUBSCRIPTION_MASK_SERVER

	paNoValue = ^uint32(0) // (uint32_t) -1: PulseAudio's "not set" of the buffer attributes
)

// library holds the bound functions of libpulse. Pointers (the main loop, the context, streams, operations) are uintptr, and so are
// the callbacks and the user data; the strings that go in are C strings. Every one of them except the main loop's own lock calls
// must run with the lock of the threaded main loop held.
type library struct {
	mainloopNew       func() uintptr
	mainloopFree      func(m uintptr)
	mainloopStart     func(m uintptr) int32
	mainloopStop      func(m uintptr)
	mainloopLock      func(m uintptr)
	mainloopUnlock    func(m uintptr)
	mainloopWait      func(m uintptr)
	mainloopSignal    func(m uintptr, waitForAccept int32)
	mainloopGetAPI    func(m uintptr) uintptr
	contextNew        func(api, name uintptr) uintptr
	contextUnref      func(c uintptr)
	contextConnect    func(c, server uintptr, flags int32, spawn uintptr) int32
	contextDisconnect func(c uintptr)
	contextGetState   func(c uintptr) int32
	contextErrno      func(c uintptr) int32
	strerror          func(err int32) uintptr

	contextSetStateCallback     func(c, cb, userdata uintptr)
	contextSetSubscribeCallback func(c, cb, userdata uintptr)
	contextSubscribe            func(c uintptr, mask uint32, cb, userdata uintptr) uintptr
	contextGetServerInfo        func(c, cb, userdata uintptr) uintptr
	contextGetSinkInfoList      func(c, cb, userdata uintptr) uintptr
	contextGetSourceInfoList    func(c, cb, userdata uintptr) uintptr
	operationUnref              func(op uintptr)
	operationGetState           func(op uintptr) int32

	streamNew                  func(c, name, spec, cmap uintptr) uintptr
	streamUnref                func(s uintptr)
	streamGetState             func(s uintptr) int32
	streamSetStateCallback     func(s, cb, userdata uintptr)
	streamSetWriteCallback     func(s, cb, userdata uintptr)
	streamSetReadCallback      func(s, cb, userdata uintptr)
	streamSetUnderflowCallback func(s, cb, userdata uintptr)
	streamConnectPlayback      func(s, dev, attr uintptr, flags int32, volume, syncStream uintptr) int32
	streamConnectRecord        func(s, dev, attr uintptr, flags int32) int32
	streamDisconnect           func(s uintptr) int32
	streamGetBufferAttr        func(s uintptr) uintptr
	streamWritableSize         func(s uintptr) uintptr
	streamBeginWrite           func(s, data, nbytes uintptr) int32
	streamWrite                func(s, data, nbytes, freeCb uintptr, offset int64, seek int32) int32
	streamCancelWrite          func(s uintptr) int32
	streamPeek                 func(s, data, nbytes uintptr) int32
	streamDrop                 func(s uintptr) int32
	streamCork                 func(s uintptr, b int32, cb, userdata uintptr) uintptr
}

// loadLibrary opens libpulse.so.0 and binds the functions. It fails with AudioUnavailable when the library cannot be loaded or lacks a
// function.
func loadLibrary() (*library, error) {
	lib, err := dl.OpenOverride(LibraryEnv, "libpulse.so.0", "libpulse.so")
	if err != nil {
		md := map[string]any{}
		if appErr, ok := errors.FromError(err); ok {
			md = appErr.Metadata
		}
		return nil, errors.New(platform.TagAudioUnavailable, "could not load libpulse", "names", md["names"], "dlerror", md["dlerror"])
	}
	l := new(library)
	for _, b := range []struct {
		fptr   any
		symbol string
	}{
		{&l.mainloopNew, "pa_threaded_mainloop_new"},
		{&l.mainloopFree, "pa_threaded_mainloop_free"},
		{&l.mainloopStart, "pa_threaded_mainloop_start"},
		{&l.mainloopStop, "pa_threaded_mainloop_stop"},
		{&l.mainloopLock, "pa_threaded_mainloop_lock"},
		{&l.mainloopUnlock, "pa_threaded_mainloop_unlock"},
		{&l.mainloopWait, "pa_threaded_mainloop_wait"},
		{&l.mainloopSignal, "pa_threaded_mainloop_signal"},
		{&l.mainloopGetAPI, "pa_threaded_mainloop_get_api"},
		{&l.contextNew, "pa_context_new"},
		{&l.contextUnref, "pa_context_unref"},
		{&l.contextConnect, "pa_context_connect"},
		{&l.contextDisconnect, "pa_context_disconnect"},
		{&l.contextGetState, "pa_context_get_state"},
		{&l.contextErrno, "pa_context_errno"},
		{&l.strerror, "pa_strerror"},
		{&l.contextSetStateCallback, "pa_context_set_state_callback"},
		{&l.contextSetSubscribeCallback, "pa_context_set_subscribe_callback"},
		{&l.contextSubscribe, "pa_context_subscribe"},
		{&l.contextGetServerInfo, "pa_context_get_server_info"},
		{&l.contextGetSinkInfoList, "pa_context_get_sink_info_list"},
		{&l.contextGetSourceInfoList, "pa_context_get_source_info_list"},
		{&l.operationUnref, "pa_operation_unref"},
		{&l.operationGetState, "pa_operation_get_state"},
		{&l.streamNew, "pa_stream_new"},
		{&l.streamUnref, "pa_stream_unref"},
		{&l.streamGetState, "pa_stream_get_state"},
		{&l.streamSetStateCallback, "pa_stream_set_state_callback"},
		{&l.streamSetWriteCallback, "pa_stream_set_write_callback"},
		{&l.streamSetReadCallback, "pa_stream_set_read_callback"},
		{&l.streamSetUnderflowCallback, "pa_stream_set_underflow_callback"},
		{&l.streamConnectPlayback, "pa_stream_connect_playback"},
		{&l.streamConnectRecord, "pa_stream_connect_record"},
		{&l.streamDisconnect, "pa_stream_disconnect"},
		{&l.streamGetBufferAttr, "pa_stream_get_buffer_attr"},
		{&l.streamWritableSize, "pa_stream_writable_size"},
		{&l.streamBeginWrite, "pa_stream_begin_write"},
		{&l.streamWrite, "pa_stream_write"},
		{&l.streamCancelWrite, "pa_stream_cancel_write"},
		{&l.streamPeek, "pa_stream_peek"},
		{&l.streamDrop, "pa_stream_drop"},
		{&l.streamCork, "pa_stream_cork"},
	} {
		if err := lib.Bind(b.fptr, b.symbol); err != nil {
			md := map[string]any{}
			if appErr, ok := errors.FromError(err); ok {
				md = appErr.Metadata
			}
			return nil, errors.New(platform.TagAudioUnavailable, "libpulse lacks a function", "library", md["library"], "symbol", md["symbol"])
		}
	}
	return l, nil
}

// sampleSpec mirrors pa_sample_spec (pulse/sample.h).
type sampleSpec struct {
	format   int32
	rate     uint32
	channels uint8
	_        [3]byte
}

// maxChannels is PA_CHANNELS_MAX; a channel map has room for that many positions.
const maxChannels = 32

// channelMap mirrors pa_channel_map (pulse/channelmap.h).
type channelMap struct {
	channels uint8
	_        [3]byte
	pos      [maxChannels]int32
}

// bufferAttr mirrors pa_buffer_attr (pulse/def.h).
type bufferAttr struct {
	maxlength, tlength, prebuf, minreq, fragsize uint32
}

// The offsets of the fields that are read from the info structures that the server's callbacks pass (pulse/introspect.h): the
// name, the index, the description and the sample specification come first in a sink and in a source, and the default names are
// after the user, host, version and name strings, the sample specification, in a server.
const (
	infoNameOffset        = 0
	infoDescriptionOffset = 16
	infoSampleSpecOffset  = 24

	serverDefaultSinkOffset   = 48
	serverDefaultSourceOffset = 56
)
