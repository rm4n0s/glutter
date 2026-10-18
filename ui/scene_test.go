package ui

import (
	"testing"

	"github.com/rm4n0s/errors"
)

func testPicture() *Picture {
	rec := NewPictureRecorder()
	c := NewCanvas(rec)
	c.DrawRect(NewRectFromLTWH(0, 0, 10, 10), &Paint{})
	return rec.EndRecording()
}

// shape describes a layer tree as nested names, for comparing.
func shape(l Layer) string {
	name := ""
	switch v := l.(type) {
	case *ContainerLayer:
		name = "container"
	case *TransformLayer:
		name = "transform"
	case *ClipRectLayer:
		name = "clipRect"
	case *ClipRRectLayer:
		name = "clipRRect"
	case *ClipRSuperellipseLayer:
		name = "clipRSuperellipse"
	case *ClipPathLayer:
		name = "clipPath"
	case *OpacityLayer:
		name = "opacity"
	case *ColorFilterLayer:
		name = "colorFilter"
	case *ImageFilterLayer:
		name = "imageFilter"
	case *BackdropFilterLayer:
		name = "backdrop"
	case *ShaderMaskLayer:
		name = "shaderMask"
	case *PictureLayer:
		name = "picture"
	default:
		name = "?"
		_ = v
	}
	kids := l.Children()
	if len(kids) == 0 {
		return name
	}
	s := name + "("
	for i, k := range kids {
		if i > 0 {
			s += ","
		}
		s += shape(k)
	}
	return s + ")"
}

func TestSceneBuilderTreeShape(t *testing.T) {
	pic := testPicture()
	tests := []struct {
		name  string
		build func(b *SceneBuilder)
		want  string
	}{
		{"empty", func(b *SceneBuilder) {}, "container"},
		{"picture", func(b *SceneBuilder) { b.AddPicture(Offset{}, pic) }, "container(picture)"},
		{"nested pushes", func(b *SceneBuilder) {
			b.PushOffset(1, 2)
			b.PushClipRect(NewRectFromLTWH(0, 0, 5, 5))
			b.AddPicture(Offset{}, pic)
			b.Pop()
			b.PushOpacity(128)
			b.AddPicture(Offset{Dx: 3}, pic)
			b.Pop()
			b.Pop()
			b.AddPicture(Offset{}, pic)
		}, "container(transform(clipRect(picture),opacity(picture)),picture)"},
		{"every push", func(b *SceneBuilder) {
			b.PushTransform(NewMatrix4Identity())
			b.PushClipRRect(RRect{})
			b.PushClipRSuperellipse(RSuperellipse{})
			b.PushClipPath(NewPath())
			b.PushColorFilter(NewColorFilterMode(Color{}, BlendModeSrcOver))
			b.PushImageFilter(NewImageFilterBlur())
			b.PushBackdropFilter(NewImageFilterBlur())
			b.PushShaderMask(nil, Rect{}, BlendModeSrcIn)
		}, "container(transform(clipRRect(clipRSuperellipse(clipPath(colorFilter(imageFilter(backdrop(shaderMask))))))))"},
		{"pop at root is ignored", func(b *SceneBuilder) {
			b.Pop()
			b.AddPicture(Offset{}, pic)
		}, "container(picture)"},
		{"nil picture is ignored", func(b *SceneBuilder) { b.AddPicture(Offset{}, nil) }, "container"},
		{"open layers are closed by Build", func(b *SceneBuilder) {
			b.PushOpacity(10)
			b.AddPicture(Offset{}, pic)
		}, "container(opacity(picture))"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewSceneBuilder()
			tt.build(b)
			if got := shape(b.Build().Root()); got != tt.want {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestSceneBuilderLayerProperties(t *testing.T) {
	pic := testPicture()
	b := NewSceneBuilder()
	m := NewMatrix4Identity()
	m.Translate(4, 5, 0)
	tl := b.PushTransform(m)
	m.Translate(100, 0, 0) // the layer holds a copy
	b.Pop()
	off := b.PushOffset(7, 8)
	b.Pop()
	cr := b.PushClipRect(NewRectFromLTWH(1, 2, 3, 4))
	b.Pop()
	cp := b.PushClipPath(NewPath(), PushClipPathOpts{ClipBehavior: ClipHardEdge})
	b.Pop()
	op := b.PushOpacity(77, PushOpacityOpts{Offset: Offset{Dx: 1, Dy: 2}})
	b.Pop()
	id := 9
	bd := b.PushBackdropFilter(NewImageFilterBlur(), PushBackdropFilterOpts{BackdropID: &id})
	b.Pop()
	sm := b.PushShaderMask(nil, NewRectFromLTWH(0, 0, 1, 1), BlendModeDstIn)
	b.Pop()
	b.AddPicture(Offset{Dx: 3}, pic, AddPictureOpts{IsComplexHint: true})
	scene := b.Build()

	if got := tl.Layer().(*TransformLayer).Matrix.Storage(); got[12] != 4 || got[13] != 5 {
		t.Errorf("transform layer matrix translation = %v, %v", got[12], got[13])
	}
	if got := off.Layer().(*TransformLayer).Matrix.Storage(); got[12] != 7 || got[13] != 8 {
		t.Errorf("offset layer translation = %v, %v", got[12], got[13])
	}
	if l := cr.Layer().(*ClipRectLayer); l.ClipBehavior != ClipAntiAlias || l.Rect != NewRectFromLTWH(1, 2, 3, 4) {
		t.Errorf("clip rect layer = %+v", l)
	}
	if l := cp.Layer().(*ClipPathLayer); l.ClipBehavior != ClipHardEdge {
		t.Errorf("clip path behavior = %v", l.ClipBehavior)
	}
	if l := op.Layer().(*OpacityLayer); l.Alpha != 77 || l.Offset != (Offset{Dx: 1, Dy: 2}) {
		t.Errorf("opacity layer = %+v", l)
	}
	if l := bd.Layer().(*BackdropFilterLayer); l.BlendMode != BlendModeSrcOver || !l.HasBackdropID || l.BackdropID != 9 {
		t.Errorf("backdrop layer = %+v", l)
	}
	if l := sm.Layer().(*ShaderMaskLayer); l.FilterQuality != FilterQualityLow || l.BlendMode != BlendModeDstIn {
		t.Errorf("shader mask layer = %+v", l)
	}
	kids := scene.Root().Children()
	pl := kids[len(kids)-1].(*PictureLayer)
	if pl.Offset.Dx != 3 || !pl.IsComplexHint || pl.WillChangeHint || pl.Picture != pic {
		t.Errorf("picture layer = %+v", pl)
	}
}

func TestSceneBuilderRetainedLayerIsReused(t *testing.T) {
	pic := testPicture()
	b1 := NewSceneBuilder()
	h := b1.PushOpacity(100)
	b1.AddPicture(Offset{}, pic)
	b1.Pop()
	s1 := b1.Build()

	b2 := NewSceneBuilder()
	b2.AddRetained(h)
	b2.PushOffset(1, 1)
	b2.AddRetained(h)
	b2.Pop()
	s2 := b2.Build()

	first := s1.Root().Children()[0]
	if got := s2.Root().Children()[0]; got != first {
		t.Errorf("retained layer is not the same layer")
	}
	nested := s2.Root().Children()[1].Children()[0]
	if nested != first {
		t.Errorf("retained layer under a push is not the same layer")
	}
	if got := shape(s2.Root()); got != "container(opacity(picture),transform(opacity(picture)))" {
		t.Errorf("shape = %s", got)
	}
}

func TestSceneBuilderOldLayer(t *testing.T) {
	b1 := NewSceneBuilder()
	h := b1.PushOpacity(1)
	b1.Pop()
	b1.Build()

	b2 := NewSceneBuilder()
	h2 := b2.PushOpacity(2, PushOpacityOpts{OldLayer: h})
	b2.Pop()
	n := b2.PushOffset(0, 0)
	b2.Pop()
	if h2.Layer().OldLayer() != h.Layer() {
		t.Errorf("OldLayer not recorded")
	}
	if n.Layer().OldLayer() != nil {
		t.Errorf("layer without oldLayer has OldLayer %v", n.Layer().OldLayer())
	}
}

func TestSceneBuilderPanics(t *testing.T) {
	tests := []struct {
		name string
		fn   func()
		tag  string
	}{
		{"disposed retained", func() {
			b := NewSceneBuilder()
			h := b.PushOpacity(1)
			h.Dispose()
			b.AddRetained(h)
		}, "EngineLayerDisposed"},
		{"push after build", func() {
			b := NewSceneBuilder()
			b.Build()
			b.PushOpacity(1)
		}, "SceneBuilderBuilt"},
		{"pop after build", func() {
			b := NewSceneBuilder()
			b.Build()
			b.Pop()
		}, "SceneBuilderBuilt"},
		{"build twice", func() {
			b := NewSceneBuilder()
			b.Build()
			b.Build()
		}, "SceneBuilderBuilt"},
		{"two options", func() {
			NewSceneBuilder().PushOpacity(1, PushOpacityOpts{}, PushOpacityOpts{})
		}, "MultipleOptions"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				appErr, ok := errors.FromError(recoverError(t, recover()))
				if !ok || !appErr.HasRoute(tt.tag) {
					t.Errorf("expected %s, got %v", tt.tag, appErr)
				}
			}()
			tt.fn()
		})
	}
}

func TestSceneToImage(t *testing.T) {
	s := NewSceneBuilder().Build()
	tests := []struct {
		w, h int
		tag  string
	}{
		{0, 10, "ImageSizeInvalid"},
		{10, -1, "ImageSizeInvalid"},
		{10, 10, "NotImplemented"},
	}
	for _, tt := range tests {
		_, err := s.ToImage(tt.w, tt.h)
		appErr, ok := errors.FromError(err)
		if !ok || !appErr.HasRoute(tt.tag) {
			t.Errorf("ToImage(%d,%d): expected %s, got %v", tt.w, tt.h, tt.tag, err)
		}
	}
	if _, err := s.ToImageSync(10, 10); err == nil {
		t.Error("ToImageSync should fail until step 2.10")
	}
	s.Dispose()
	if s.String() != "Scene" || NewSceneBuilder().String() != "SceneBuilder" {
		t.Error("unexpected String()")
	}
}
