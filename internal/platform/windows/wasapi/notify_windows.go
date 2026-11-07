//go:build windows

package wasapi

import (
	"log/slog"
	"runtime/debug"
	"sync"

	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// notifyEnv is the Host that makes the notification client and its interface type, made once for the process: the callbacks of a
// process are never freed (SPEC §6.3 rule 11), so the five methods of IMMNotificationClient and the three of IUnknown are made one
// time, and every service finds its object by the handle (SPEC §25.2).
type notifyEnv struct {
	host *win32.Host
	typ  *win32.InterfaceType
}

var sharedNotifyEnv = sync.OnceValues(newNotifyEnv)

// newNotifyEnv makes the Host and the interface type of IMMNotificationClient.
func newNotifyEnv() (*notifyEnv, error) {
	host, err := win32.NewHost(nil)
	if err != nil {
		return nil, err
	}
	typ, err := host.NewInterface(win32.InterfaceDef{Name: "IMMNotificationClient", IID: win32.IIDMMNotificationClient, Methods: []win32.MethodDef{
		{Name: "OnDeviceStateChanged", Args: 2, Fn: func(impl any, a []uintptr) uintptr {
			return impl.(*notifier).run("OnDeviceStateChanged", func(n *notifier) { n.svc.endpointChanged(win32.WideString(a[0])) })
		}},
		{Name: "OnDeviceAdded", Args: 1, Fn: func(impl any, a []uintptr) uintptr {
			return impl.(*notifier).run("OnDeviceAdded", func(n *notifier) { n.svc.changed() })
		}},
		{Name: "OnDeviceRemoved", Args: 1, Fn: func(impl any, a []uintptr) uintptr {
			return impl.(*notifier).run("OnDeviceRemoved", func(n *notifier) { n.svc.endpointChanged(win32.WideString(a[0])) })
		}},
		{Name: "OnDefaultDeviceChanged", Args: 3, Fn: func(impl any, a []uintptr) uintptr {
			// Windows sends one for each role; the console role is the one that programs play on, and the same device change is
			// told to the other roles too.
			return impl.(*notifier).run("OnDefaultDeviceChanged", func(n *notifier) {
				if uint32(a[1]) == win32.EConsole {
					n.svc.defaultChanged(uint32(a[0]), win32.WideString(a[2]))
				}
			})
		}},
		// The properties of an endpoint change with every volume or setting; nothing here depends on them.
		{Name: "OnPropertyValueChanged", Args: 2, Fn: func(impl any, a []uintptr) uintptr { return win32.Result(win32.SOK) }},
	}})
	if err != nil {
		return nil, err
	}
	return &notifyEnv{host: host, typ: typ}, nil
}

// notifier is the Go value behind the notification client of a Service: its methods run on a thread of Windows.
type notifier struct{ svc *Service }

// run calls fn under a guard that keeps a panic from unwinding through the frames of Windows, and answers S_OK.
func (n *notifier) run(method string, fn func(*notifier)) (result uintptr) {
	defer func() {
		if r := recover(); r != nil {
			n.svc.log.Error("audio: a notification of the devices panicked", slog.String("method", method), "panic", r, "stack", string(debug.Stack()))
		}
		result = win32.Result(win32.SOK)
	}()
	fn(n)
	return win32.Result(win32.SOK)
}
