package ui

import (
	"runtime"
	"testing"
	"weak"
)

// TestSceneBuilderOldLayerDoesNotChainFrames checks that a layer that names the layer of the
// previous frame (OldLayer) does not keep that frame alive for good: the chain of the old layers
// of all frames would hold every picture that the app ever painted (step 29.21).
func TestSceneBuilderOldLayerDoesNotChainFrames(t *testing.T) {
	tests := []struct {
		name   string
		frames int
		push   func(b *SceneBuilder, old EngineLayer) EngineLayer
	}{
		{"opacity", 50, func(b *SceneBuilder, old EngineLayer) EngineLayer {
			o, _ := old.(*OpacityEngineLayer)
			return b.PushOpacity(1, PushOpacityOpts{OldLayer: o})
		}},
		{"offset", 50, func(b *SceneBuilder, old EngineLayer) EngineLayer {
			o, _ := old.(*OffsetEngineLayer)
			return b.PushOffset(1, 1, PushOffsetOpts{OldLayer: o})
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var first weak.Pointer[engineLayerHandle]
			var firstLayer weak.Pointer[layerBase]
			var handle EngineLayer
			var last *Scene
			for i := 0; i < tt.frames; i++ {
				b := NewSceneBuilder()
				handle = tt.push(b, handle)
				b.Pop()
				last = b.Build()
				if i == 0 {
					first = weak.Make(handle.handle())
					firstLayer = weak.Make(handle.handle().layer.layer())
				}
			}
			runtime.GC()
			runtime.GC()
			if first.Value() != nil || firstLayer.Value() != nil {
				t.Errorf("the first of %d frames is still alive: each layer holds the layer of the frame before it", tt.frames)
			}
			// The layer of the frame before is still named while the framework holds its handle.
			b := NewSceneBuilder()
			prev := handle
			handle = tt.push(b, prev)
			b.Pop()
			if got := handle.handle().layer.OldLayer(); got != prev.handle().layer {
				t.Errorf("OldLayer = %v, want the layer of the handle that was passed", got)
			}
			runtime.KeepAlive(last)
			runtime.KeepAlive(prev)
		})
	}
}
