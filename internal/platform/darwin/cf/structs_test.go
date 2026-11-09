package cf

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/platform/darwin/layouttest"
)

// TestStructMirrorsMatchTheSDK checks the Go mirrors of this package against the headers of the SDK
// (skipped, with the install command, when the machine is not a Mac).
func TestStructMirrorsMatchTheSDK(t *testing.T) {
	layouttest.Check(t, layouttest.Spec{
		Headers: []string{"CoreFoundation/CoreFoundation.h"},
		Structs: []layouttest.Struct{
			layouttest.StructOf("CFRange", Range{}),
			layouttest.StructOf("CFRunLoopSourceContext", SourceContext{}),
			layouttest.StructOf("CFRunLoopTimerContext", TimerContext{}),
		},
		Consts: []layouttest.Const{
			{Name: "kCFStringEncodingUTF8", Value: kCFStringEncodingUTF8Value},
			{Name: "kCFRunLoopRunFinished", Value: 1},
			{Name: "kCFRunLoopRunStopped", Value: 2},
			{Name: "kCFRunLoopRunTimedOut", Value: 3},
			{Name: "kCFRunLoopRunHandledSource", Value: 4},
		},
	})
}

// kCFStringEncodingUTF8Value is the value that the darwin code uses; it is repeated here because that code
// does not build on the hosts that run this test without a Mac.
const kCFStringEncodingUTF8Value = 0x08000100
