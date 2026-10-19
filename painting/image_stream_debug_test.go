//go:build !glutter_release && !glutter_profile

package painting

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
)

func TestImageStreamDebugPanics(t *testing.T) {
	pt := newPaintingTester(t)
	disposed := func() *ImageStreamCompleter {
		c := NewImageStreamCompleter(pt.PaintingBinding)
		l := (&imageRecorder{}).listener()
		c.AddListener(l)
		c.RemoveListener(l)
		return c
	}
	tests := []struct {
		name string
		tag  string
		call func()
	}{
		{"add listener after dispose", "ImageStreamDisposed", func() { disposed().AddListener((&imageRecorder{}).listener()) }},
		{"set image after dispose", "ImageStreamDisposed", func() { disposed().SetImage(NewImageInfo(testImage(1, 1))) }},
		{"keep alive after dispose", "ImageStreamDisposed", func() { disposed().KeepAlive() }},
		{"completer set twice", "ImageStreamCompleterSet", func() {
			s := NewImageStream()
			s.SetCompleter(NewImageStreamCompleter(pt.PaintingBinding))
			s.SetCompleter(NewImageStreamCompleter(pt.PaintingBinding))
		}},
		{"handle disposed twice", "ImageStreamCompleterHandleDisposed", func() {
			h := NewImageStreamCompleter(pt.PaintingBinding).KeepAlive()
			h.Dispose()
			h.Dispose()
		}},
		{"negative chunk", "ImageChunkEventInvalid", func() { NewImageChunkEvent(-1, foundation.None[int]()) }},
		{"negative expected", "ImageChunkEventInvalid", func() { NewImageChunkEvent(1, foundation.Some(-1)) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { requireTaggedPanic(t, tt.tag, tt.call) })
	}
}
