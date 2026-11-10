//go:build darwin

package coreaudio

import (
	"sync"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/darwin/cf"
	"github.com/rm4n0s/glutter/internal/platform/darwin/dl"
)

// library is the table of the functions of CoreAudio and AudioToolbox that the backend calls. The tests of the service put their
// own functions in it.
type library struct {
	getPropertyDataSize func(obj uint32, addr *PropertyAddress, qualSize uint32, qual unsafe.Pointer, size *uint32) int32
	getPropertyData     func(obj uint32, addr *PropertyAddress, qualSize uint32, qual unsafe.Pointer, size *uint32, data unsafe.Pointer) int32
	addListener         func(obj uint32, addr *PropertyAddress, cb uintptr, ctx uintptr) int32
	removeListener      func(obj uint32, addr *PropertyAddress, cb uintptr, ctx uintptr) int32

	newOutput   func(sd *StreamDescription, cb uintptr, ctx uintptr, runLoop, mode cf.Ref, flags uint32, queue *uintptr) int32
	newInput    func(sd *StreamDescription, cb uintptr, ctx uintptr, runLoop, mode cf.Ref, flags uint32, queue *uintptr) int32
	setProperty func(queue uintptr, id uint32, data unsafe.Pointer, size uint32) int32
	allocBuffer func(queue uintptr, size uint32, buf *uintptr) int32
	enqueue     func(queue uintptr, buf uintptr, packets uint32, descs uintptr) int32
	start       func(queue uintptr, at uintptr) int32
	pause       func(queue uintptr) int32
	stop        func(queue uintptr, immediate bool) int32
	dispose     func(queue uintptr, immediate bool) int32

	// routeOnly says that the system has no AudioObject API for the devices (iOS): there is one output and one input, the ones of the
	// current route, and the list and the listeners are made up (service_darwin.go).
	routeOnly bool

	// outputCB, inputCB and listenerCB are the C function pointers of the callbacks; they find their object by the context.
	outputCB, inputCB, listenerCB uintptr
}

// loadLibrary binds Core Audio. A failure is the tag AudioUnavailable.
var loadLibrary = sync.OnceValues(func() (*library, error) {
	ca, err := dl.OpenFramework("CoreAudio")
	if err != nil {
		return nil, errors.NewErr(platform.TagAudioUnavailable, err)
	}
	at, err := dl.OpenFramework("AudioToolbox")
	if err != nil {
		return nil, errors.NewErr(platform.TagAudioUnavailable, err)
	}
	l := new(library)
	type binding struct {
		lib    *dl.Lib
		fptr   any
		symbol string
	}
	// iOS has the CoreAudio framework but not its AudioObject functions: the devices are those of the route (SPEC §30.9).
	l.routeOnly = !ca.Has("AudioObjectGetPropertyData")
	var objects []binding
	if !l.routeOnly {
		objects = []binding{
			{ca, &l.getPropertyDataSize, "AudioObjectGetPropertyDataSize"}, {ca, &l.getPropertyData, "AudioObjectGetPropertyData"},
			{ca, &l.addListener, "AudioObjectAddPropertyListener"}, {ca, &l.removeListener, "AudioObjectRemovePropertyListener"},
		}
	}
	for _, b := range append(objects, []binding{
		{at, &l.newOutput, "AudioQueueNewOutput"}, {at, &l.newInput, "AudioQueueNewInput"},
		{at, &l.setProperty, "AudioQueueSetProperty"}, {at, &l.allocBuffer, "AudioQueueAllocateBuffer"},
		{at, &l.enqueue, "AudioQueueEnqueueBuffer"}, {at, &l.start, "AudioQueueStart"}, {at, &l.pause, "AudioQueuePause"},
		{at, &l.stop, "AudioQueueStop"}, {at, &l.dispose, "AudioQueueDispose"},
	}...) {
		if err := b.lib.Bind(b.fptr, b.symbol); err != nil {
			return nil, errors.NewErr(platform.TagAudioUnavailable, err).SetMetadata("symbol", b.symbol)
		}
	}
	l.outputCB, l.inputCB, l.listenerCB = outputCallback(), inputCallback(), listenerCallback()
	return l, nil
})

// handles maps the context that Core Audio hands back to its object, because a callback made once for the process must find its
// object (SPEC §8.1 rule 2). It is process-wide state with its reason: purego callbacks are never freed. An entry is deleted when
// the device or service ends, so the map holds only what is alive.
var handles struct {
	mu   sync.RWMutex
	next uintptr
	m    map[uintptr]any
}

func register(v any) uintptr {
	handles.mu.Lock()
	defer handles.mu.Unlock()
	if handles.m == nil {
		handles.m = map[uintptr]any{}
	}
	handles.next++
	handles.m[handles.next] = v
	return handles.next
}

func lookup(h uintptr) any {
	handles.mu.RLock()
	defer handles.mu.RUnlock()
	return handles.m[h]
}

func unregister(h uintptr) {
	handles.mu.Lock()
	delete(handles.m, h)
	handles.mu.Unlock()
}

// The callbacks of Core Audio, made once, on first use (counted by package dl). They run on threads of Core Audio's, never on the UI
// thread.
var (
	outputCallback = sync.OnceValue(func() uintptr {
		return dl.Callback(func(ctx uintptr, queue uintptr, buf uintptr) {
			if d, ok := lookup(ctx).(*device); ok {
				d.onOutput(queue, buf)
			}
		})
	})
	inputCallback = sync.OnceValue(func() uintptr {
		return dl.Callback(func(ctx uintptr, queue uintptr, buf uintptr, _ uintptr, _ uint32, _ uintptr) {
			if d, ok := lookup(ctx).(*device); ok {
				d.onInput(queue, buf)
			}
		})
	})
	listenerCallback = sync.OnceValue(func() uintptr {
		return dl.Callback(func(obj uint32, n uint32, addrs uintptr, ctx uintptr) int32 {
			var sels []uint32
			for i := range n {
				a := (*PropertyAddress)(dl.Pointer(addrs + uintptr(i)*unsafe.Sizeof(PropertyAddress{})))
				sels = append(sels, a.Selector)
			}
			switch v := lookup(ctx).(type) {
			case *Service:
				v.onSystemChange(sels)
			case *device:
				v.onProperty(obj, sels)
			}
			return 0
		})
	})
)

// The names of the calls, for the errors.
const tagCoreAudioFailed = "CoreAudioFailed"

// Some OSStatus values that mean that the device cannot take the format (AudioFormat.h, AudioQueue.h, MacErrors.h).
const (
	statusFormatUnsupported     = 0x666D743F // kAudioFormatUnsupportedDataFormatError '"'"'fmt?'"'"'
	statusParamErr              = -50        // paramErr: an argument that the device does not take
	statusQueueInvalidParameter = -66682     // kAudioQueueErr_InvalidParameter
)

// statusError is the error of a call of Core Audio that returned an OSStatus other than 0. It is one table: for a call that opens a
// device, a format or an argument that the device does not take is AudioFormatUnsupported and anything else AudioDeviceOpenFailed;
// for any other call it is CoreAudioFailed. The metadata are "call" and "status". It is listed in errcheck.allow.
func statusError(call string, status int32, opening bool) error {
	if opening {
		switch status {
		case statusFormatUnsupported, statusParamErr, statusQueueInvalidParameter:
			return errors.New(platform.TagAudioFormatUnsupported, "the Core Audio device does not take the format", "call", call, "status", int(status))
		}
		return errors.New(platform.TagAudioDeviceOpenFailed, "the Core Audio device could not be opened", "call", call, "status", int(status))
	}
	return errors.New(tagCoreAudioFailed, "a call of Core Audio failed", "call", call, "status", int(status))
}
