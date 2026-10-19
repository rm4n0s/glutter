package x11

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// The sizes and offsets below were printed by a C program that includes the
// headers of libX11, libXi and libXcursor on a 64-bit Linux system.
func TestStructLayouts(t *testing.T) {
	if xEventSize != 192 {
		t.Errorf("xEventSize = %d, want sizeof(XEvent) = 192", xEventSize)
	}
	dl.AssertLayout(t, "XAnyEvent", 40, map[string]uintptr{
		"typ": 0, "serial": 8, "sendEvent": 16, "display": 24, "window": 32,
	}, xAnyEvent{})
	dl.AssertLayout(t, "XConfigureEvent", 88, map[string]uintptr{
		"typ": 0, "serial": 8, "sendEvent": 16, "display": 24, "event": 32, "window": 40,
		"x": 48, "y": 52, "width": 56, "height": 60, "borderWidth": 64, "above": 72, "overrideRedirect": 80,
	}, xConfigureEvent{})
	dl.AssertLayout(t, "XClientMessageEvent", 96, map[string]uintptr{
		"typ": 0, "serial": 8, "sendEvent": 16, "display": 24, "window": 32, "messageType": 40, "format": 48, "data": 56,
	}, xClientMessageEvent{})
	dl.AssertLayout(t, "XFocusChangeEvent", 48, map[string]uintptr{
		"typ": 0, "serial": 8, "sendEvent": 16, "display": 24, "window": 32, "mode": 40, "detail": 44,
	}, xFocusChangeEvent{})
	dl.AssertLayout(t, "XExposeEvent", 64, map[string]uintptr{
		"typ": 0, "serial": 8, "sendEvent": 16, "display": 24, "window": 32, "x": 40, "y": 44, "width": 48, "height": 52, "count": 56,
	}, xExposeEvent{})
	dl.AssertLayout(t, "XMapEvent", 56, map[string]uintptr{
		"typ": 0, "serial": 8, "sendEvent": 16, "display": 24, "event": 32, "window": 40, "overrideRedirect": 48,
	}, xMapEvent{})
	dl.AssertLayout(t, "XDestroyWindowEvent", 48, map[string]uintptr{
		"typ": 0, "serial": 8, "sendEvent": 16, "display": 24, "event": 32, "window": 40,
	}, xDestroyWindowEvent{})
	dl.AssertLayout(t, "XSelectionRequestEvent", 80, map[string]uintptr{
		"typ": 0, "serial": 8, "sendEvent": 16, "display": 24, "owner": 32, "requestor": 40, "selection": 48, "target": 56,
		"property": 64, "time": 72,
	}, xSelectionRequestEvent{})
	dl.AssertLayout(t, "XSelectionEvent", 72, map[string]uintptr{
		"typ": 0, "serial": 8, "sendEvent": 16, "display": 24, "requestor": 32, "selection": 40, "target": 48,
		"property": 56, "time": 64,
	}, xSelectionEvent{})
	dl.AssertLayout(t, "XSelectionClearEvent", 56, map[string]uintptr{
		"typ": 0, "serial": 8, "sendEvent": 16, "display": 24, "window": 32, "selection": 40, "time": 48,
	}, xSelectionClearEvent{})
	dl.AssertLayout(t, "XPropertyEvent", 64, map[string]uintptr{
		"typ": 0, "serial": 8, "sendEvent": 16, "display": 24, "window": 32, "atom": 40, "time": 48, "state": 56,
	}, xPropertyEvent{})
	dl.AssertLayout(t, "XGenericEventCookie", 56, map[string]uintptr{
		"typ": 0, "serial": 8, "sendEvent": 16, "display": 24, "extension": 32, "evtype": 36, "cookie": 40, "data": 48,
	}, xGenericEventCookie{})
	dl.AssertLayout(t, "XErrorEvent", 40, map[string]uintptr{
		"typ": 0, "display": 8, "resourceID": 16, "serial": 24, "errorCode": 32, "requestCode": 33, "minorCode": 34,
	}, xErrorEvent{})
	dl.AssertLayout(t, "XSetWindowAttributes", 112, map[string]uintptr{
		"backgroundPixmap": 0, "backgroundPixel": 8, "borderPixmap": 16, "borderPixel": 24, "bitGravity": 32, "winGravity": 36,
		"backingStore": 40, "backingPlanes": 48, "backingPixel": 56, "saveUnder": 64, "eventMask": 72,
		"doNotPropagateMask": 80, "overrideRedirect": 88, "colormap": 96, "cursor": 104,
	}, xSetWindowAttributes{})
	dl.AssertLayout(t, "XVisualInfo", 64, map[string]uintptr{
		"visual": 0, "visualID": 8, "screen": 16, "depth": 20, "class": 24, "redMask": 32, "greenMask": 40, "blueMask": 48,
		"colormapSize": 56, "bitsPerRGB": 60,
	}, xVisualInfo{})
	dl.AssertLayout(t, "XSizeHints", 80, map[string]uintptr{
		"flags": 0, "x": 8, "y": 12, "width": 16, "height": 20, "minWidth": 24, "minHeight": 28, "maxWidth": 32,
		"maxHeight": 36, "widthInc": 40, "heightInc": 44, "minAspect": 48, "maxAspect": 56, "baseWidth": 64,
		"baseHeight": 68, "winGravity": 72,
	}, xSizeHints{})
	dl.AssertLayout(t, "XClassHint", 16, map[string]uintptr{"resName": 0, "resClass": 8}, xClassHint{})
	dl.AssertLayout(t, "XIEventMask", 16, map[string]uintptr{"deviceID": 0, "maskLen": 4, "mask": 8}, xiEventMask{})
	dl.AssertLayout(t, "XIButtonState", 16, map[string]uintptr{"maskLen": 0, "mask": 8}, xiButtonState{})
	dl.AssertLayout(t, "XIValuatorState", 24, map[string]uintptr{"maskLen": 0, "mask": 8, "values": 16}, xiValuatorState{})
	dl.AssertLayout(t, "XIModifierState", 16, map[string]uintptr{"base": 0, "latched": 4, "locked": 8, "effective": 12}, xiModifierState{})
	dl.AssertLayout(t, "XIDeviceEvent", 200, map[string]uintptr{
		"typ": 0, "serial": 8, "sendEvent": 16, "display": 24, "extension": 32, "evtype": 36, "time": 40, "deviceID": 48,
		"sourceID": 52, "detail": 56, "root": 64, "event": 72, "child": 80, "rootX": 88, "rootY": 96, "eventX": 104,
		"eventY": 112, "flags": 120, "buttons": 128, "valuators": 144, "mods": 168, "group": 184,
	}, xiDeviceEvent{})
	dl.AssertLayout(t, "XIEnterEvent", 184, map[string]uintptr{
		"typ": 0, "serial": 8, "sendEvent": 16, "display": 24, "extension": 32, "evtype": 36, "time": 40, "deviceID": 48,
		"sourceID": 52, "detail": 56, "root": 64, "event": 72, "child": 80, "rootX": 88, "rootY": 96, "eventX": 104,
		"eventY": 112, "mode": 120, "focus": 124, "sameScreen": 128, "buttons": 136, "mods": 152, "group": 168,
	}, xiEnterEvent{})
	dl.AssertLayout(t, "XIDeviceInfo", 40, map[string]uintptr{
		"deviceID": 0, "name": 8, "use": 16, "attachment": 20, "enabled": 24, "numClasses": 28, "classes": 32,
	}, xiDeviceInfo{})
	dl.AssertLayout(t, "XIAnyClassInfo", 8, map[string]uintptr{"typ": 0, "sourceID": 4}, xiAnyClassInfo{})
	dl.AssertLayout(t, "XIScrollClassInfo", 32, map[string]uintptr{
		"typ": 0, "sourceID": 4, "number": 8, "scrollType": 12, "increment": 16, "flags": 24,
	}, xiScrollClassInfo{})
	dl.AssertLayout(t, "XIValuatorClassInfo", 56, map[string]uintptr{
		"typ": 0, "sourceID": 4, "number": 8, "label": 16, "min": 24, "max": 32, "value": 40, "resolution": 48, "mode": 52,
	}, xiValuatorClassInfo{})
	dl.AssertLayout(t, "XColor", 16, map[string]uintptr{
		"pixel": 0, "red": 8, "green": 10, "blue": 12, "flags": 14,
	}, xColor{})
}
