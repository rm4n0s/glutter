package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// TestTexture ports test/widgets/texture_test.dart: the four tests of the widget, the render box and the layer
// it paints.
func TestTexture(t *testing.T) {
	tests := []struct {
		name        string
		opts        TextureOpts
		wantFreeze  bool
		wantQuality ui.FilterQuality
	}{
		{"Texture with freeze set to true", TextureOpts{Freeze: true}, true, ui.FilterQualityLow},
		{"Texture with default FilterQuality", TextureOpts{}, false, ui.FilterQualityLow},
		{"Texture with FilterQuality.none", TextureOpts{FilterQuality: ui.FilterQualityNone}, false, ui.FilterQualityNone},
		{"Texture with FilterQuality.low", TextureOpts{FilterQuality: ui.FilterQualityLow}, false, ui.FilterQualityLow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			ts.PumpWidget(NewCenter(NewTexture(1, tt.opts)))

			w, ok := ts.Widget(gluttertest.ByType[*Texture]()).(*Texture)
			if !ok {
				t.Fatal("no Texture widget")
			}
			if w.TextureID() != 1 || w.Freeze() != tt.wantFreeze || w.FilterQuality() != tt.wantQuality {
				t.Errorf("widget = id %d, freeze %v, quality %v", w.TextureID(), w.Freeze(), w.FilterQuality())
			}
			box, ok := ts.RenderObject(gluttertest.ByType[*Texture]()).(*rendering.TextureBox)
			if !ok {
				t.Fatal("the render object is not a TextureBox")
			}
			if box.TextureID() != 1 || box.Freeze() != tt.wantFreeze || box.FilterQuality() != tt.wantQuality {
				t.Errorf("render box = id %d, freeze %v, quality %v", box.TextureID(), box.Freeze(), box.FilterQuality())
			}

			container := rendering.NewContainerLayer()
			defer container.Dispose()
			box.Paint(rendering.NewPaintingContext(container, ui.Rect{}), ui.OffsetZero)
			layer, ok := container.LastChild().(*rendering.TextureLayer)
			if !ok {
				t.Fatalf("the last child is %T, not a TextureLayer", container.LastChild())
			}
			if layer.TextureID() != 1 || layer.Freeze() != tt.wantFreeze || layer.FilterQuality() != tt.wantQuality {
				t.Errorf("layer = id %d, freeze %v, quality %v", layer.TextureID(), layer.Freeze(), layer.FilterQuality())
			}
		})
	}
}

// TestTextureFillsItsParent: the box takes as much room as it is given, and a Center around it gives the whole view.
func TestTextureFillsItsParent(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(NewCenter(NewSizedBox(NewTexture(7), SizedBoxOpts{Width: foundation.Some(40.0), Height: foundation.Some(30.0)})))
	if got := ts.Size(gluttertest.ByType[*Texture]()); got != (ui.Size{Width: 40, Height: 30}) {
		t.Errorf("size = %v, want 40 x 30", got)
	}
	ts.PumpWidget(NewCenter(NewTexture(7)))
	if got := ts.Size(gluttertest.ByType[*Texture]()); got != (ui.Size{Width: 800, Height: 600}) {
		t.Errorf("size = %v, want the whole view", got)
	}
}

// TestTextureChangesRepaint: the properties of the box repaint it, and a new id changes the layer.
func TestTextureChangesRepaint(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(NewCenter(NewTexture(1)))
	box := ts.RenderObject(gluttertest.ByType[*Texture]()).(*rendering.TextureBox)
	ts.PumpWidget(NewCenter(NewTexture(2, TextureOpts{Freeze: true, FilterQuality: ui.FilterQualityHigh})))
	if box.TextureID() != 2 || !box.Freeze() || box.FilterQuality() != ui.FilterQualityHigh {
		t.Errorf("render box = id %d, freeze %v, quality %v", box.TextureID(), box.Freeze(), box.FilterQuality())
	}
}
