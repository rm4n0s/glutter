//go:build !glutter_release && !glutter_profile

package painting

import "testing"

func TestImageCacheDebugPanics(t *testing.T) {
	pt := newPaintingTester(t)
	tests := []struct {
		name string
		call func()
	}{
		{"negative size", func() { pt.ImageCache().SetMaximumSize(-1) }},
		{"negative bytes", func() { pt.ImageCache().SetMaximumSizeBytes(-1) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { requireTaggedPanic(t, "ImageCacheSizeInvalid", tt.call) })
	}
}
