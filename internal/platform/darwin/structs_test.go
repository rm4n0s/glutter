package darwin

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/platform/darwin/layouttest"
)

// TestStructMirrorsMatchTheSDK checks every Go mirror of this package against the headers of the SDK
// (skipped, with the install command, when the machine is not a Mac or has no Command Line Tools).
func TestStructMirrorsMatchTheSDK(t *testing.T) {
	layouttest.Check(t, layouttest.Spec{
		Headers: []string{"CoreGraphics/CoreGraphics.h", "Foundation/Foundation.h", "AppKit/AppKit.h", "CoreVideo/CoreVideo.h"},
		Structs: []layouttest.Struct{
			layouttest.StructOf("CGPoint", CGPoint{}),
			layouttest.StructOf("CGSize", CGSize{}),
			layouttest.StructOf("CGRect", CGRect{}),
			layouttest.StructOf("NSEdgeInsets", NSEdgeInsets{}),
			layouttest.StructOf("NSRange", NSRange{}),
			layouttest.StructOf("CVSMPTETime", CVSMPTETime{}),
			layouttest.StructOf("CVTimeStamp", CVTimeStamp{}),
		},
		Consts: []layouttest.Const{
			{Name: "NSUTF8StringEncoding", Value: 4},
		},
	})
}

// TestNSRectIsCGRect checks that the one Go type stands for both names: AppKit's NSRect is a CGRect.
func TestNSRectIsCGRect(t *testing.T) {
	layouttest.Check(t, layouttest.Spec{
		Headers: []string{"AppKit/AppKit.h"},
		Structs: []layouttest.Struct{layouttest.StructOf("NSRect", CGRect{})},
	})
}
