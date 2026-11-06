package gpu

import (
	"image"
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func TestFlattenMakesATextureCommand(t *testing.T) {
	b := ui.NewSceneBuilder()
	b.PushOffset(10, 20)
	b.PushClipRect(ui.NewRectFromLTWH(0, 0, 30, 30))
	b.AddTexture(7, ui.AddTextureOpts{Offset: ui.Offset{Dx: 1, Dy: 2}, Width: 50, Height: 60, FilterQuality: ui.FilterQualityHigh})
	b.Pop()
	b.Pop()
	list, err := Flatten(b.Build())
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Commands) != 1 {
		t.Fatalf("%d commands, want 1", len(list.Commands))
	}
	c := list.Commands[0]
	l, ok := c.Layer.(*ui.TextureLayer)
	if c.Kind != CommandTexture || !ok || l.TextureID != 7 || l.Width != 50 || l.Height != 60 || l.FilterQuality != ui.FilterQualityHigh {
		t.Fatalf("command = %+v", c)
	}
	// The transform carries the offset of the layers above the texture, the layer its own offset.
	if got := c.Transform.TransformPoint(l.Offset); got != (ui.Offset{Dx: 11, Dy: 22}) {
		t.Errorf("the layer's corner is at %v, want (11, 22)", got)
	}
	if c.Clip.Bounds != (ui.Rect{Left: 10, Top: 20, Right: 40, Bottom: 50}) {
		t.Errorf("clip = %v", c.Clip.Bounds)
	}
}

func TestDamageOfATextureLayer(t *testing.T) {
	versions := map[int64]uint64{1: 1, 2: 1}
	tracker := NewDamageTracker()
	tracker.TextureVersion = func(id int64) uint64 { return versions[id] }
	// The picture is the same object in every frame: reuse it.
	pic := rectPicture(ui.NewRectFromLTWH(0, 0, 400, 300), 0xFF202020)
	build := func(freeze bool) *ui.Scene {
		b := ui.NewSceneBuilder()
		b.AddPicture(ui.Offset{}, pic)
		b.AddTexture(1, ui.AddTextureOpts{Offset: ui.Offset{Dx: 70, Dy: 70}, Width: 60, Height: 60, Freeze: freeze})
		b.AddTexture(2, ui.AddTextureOpts{Offset: ui.Offset{Dx: 270, Dy: 170}, Width: 60, Height: 60})
		return b.Build()
	}
	if d := tracker.Update(build(false), 400, 300); !d.Full {
		t.Fatalf("the first frame is not full: %+v", d)
	}
	if d := tracker.Update(build(false), 400, 300); d.Full || len(d.Rects) != 0 {
		t.Errorf("a frame with nothing new damaged %+v", d)
	}
	versions[1]++
	d := tracker.Update(build(false), 400, 300)
	if d.Full || !covers(d, pt(100, 100)) || covers(d, pt(300, 200)) || covers(d, pt(10, 10)) {
		t.Errorf("a new frame of texture 1 damaged %+v, want only its rectangle", d)
	}
	versions[2]++
	d = tracker.Update(build(false), 400, 300)
	if d.Full || !covers(d, pt(300, 200)) || covers(d, pt(100, 100)) {
		t.Errorf("a new frame of texture 2 damaged %+v, want only its rectangle", d)
	}
	// A frozen layer does not take new frames: marking one damages nothing; freezing it is a change of the layer.
	d = tracker.Update(build(true), 400, 300)
	if !covers(d, pt(100, 100)) {
		t.Errorf("freezing the layer damaged %+v, want its rectangle", d)
	}
	versions[1]++
	if d = tracker.Update(build(true), 400, 300); d.Full || len(d.Rects) != 0 {
		t.Errorf("a new frame of a frozen texture damaged %+v", d)
	}
	// Unregistering (version 0) is a change.
	versions[1] = 0
	if d = tracker.Update(build(false), 400, 300); !covers(d, pt(100, 100)) {
		t.Errorf("unregistering the texture damaged %+v", d)
	}
}

func pt(x, y int) image.Point { return image.Pt(x, y) }
