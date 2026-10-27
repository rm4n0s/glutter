package ui

import "testing"

// mapTestScene is a scene with every kind of layer that has children, a shader mask and
// pictures, and the pictures and the shader it was made of.
func mapTestScene() (scene *Scene, a, b *Picture, shader Shader) {
	a, b = testPicture(), testPicture()
	shader = NewGradientLinear(Offset{}, Offset{Dx: 10}, []Color{NewColor(0xFF000000), NewColor(0xFFFFFFFF)})
	sb := NewSceneBuilder()
	sb.PushOffset(5, 5)
	sb.PushClipRect(NewRectFromLTWH(0, 0, 50, 50))
	sb.AddPicture(Offset{}, a)
	sb.Pop()
	sb.PushOpacity(128)
	sb.PushShaderMask(shader, NewRectFromLTWH(0, 0, 10, 10), BlendModeSrcIn)
	sb.AddPicture(Offset{}, b)
	sb.Pop()
	sb.Pop()
	sb.Pop()
	return sb.Build(), a, b, shader
}

func TestSceneMapContent(t *testing.T) {
	other := testPicture()
	otherShader := NewGradientLinear(Offset{}, Offset{Dx: 5}, []Color{NewColor(0xFF111111), NewColor(0xFF222222)})
	tests := []struct {
		name string
		// pictures and shaders map the content; nil keeps it.
		pictures func(a, b, other *Picture) func(*Picture) *Picture
		shaders  func(s Shader) func(Shader) Shader
		// wantShape is the shape of the result, wantSame whether the root is the source's.
		wantSame bool
		want     func(t *testing.T, got *Scene, other *Picture, otherShader Shader)
	}{
		{name: "no functions", wantSame: true},
		{
			name:     "functions that keep everything",
			pictures: func(a, b, other *Picture) func(*Picture) *Picture { return func(p *Picture) *Picture { return p } },
			shaders:  func(s Shader) func(Shader) Shader { return func(x Shader) Shader { return x } },
			wantSame: true,
		},
		{
			name: "a picture replaced",
			pictures: func(a, b, other *Picture) func(*Picture) *Picture {
				return func(p *Picture) *Picture {
					if p == a {
						return other
					}
					return p
				}
			},
			want: func(t *testing.T, got *Scene, other *Picture, _ Shader) {
				first := got.Root().Children()[0].Children()[0].Children()[0].(*PictureLayer)
				if first.Picture != other {
					t.Error("the picture was not replaced")
				}
			},
		},
		{
			name: "a shader replaced",
			shaders: func(s Shader) func(Shader) Shader {
				return func(x Shader) Shader { return otherShader }
			},
			want: func(t *testing.T, got *Scene, _ *Picture, otherShader Shader) {
				mask := got.Root().Children()[0].Children()[1].Children()[0].(*ShaderMaskLayer)
				if mask.Shader != otherShader {
					t.Error("the shader was not replaced")
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, a, b, shader := mapTestScene()
			var pf func(*Picture) *Picture
			var sf func(Shader) Shader
			if tt.pictures != nil {
				pf = tt.pictures(a, b, other)
			}
			if tt.shaders != nil {
				sf = tt.shaders(shader)
			}
			before := shape(src.Root())
			got := src.MapContent(pf, sf)
			if got == src {
				t.Error("the result is the source scene")
			}
			if (got.Root() == src.Root()) != tt.wantSame {
				t.Errorf("the root is shared: %v, want %v", got.Root() == src.Root(), tt.wantSame)
			}
			if shape(got.Root()) != before {
				t.Errorf("the shape is %s, want %s", shape(got.Root()), before)
			}
			if shape(src.Root()) != before {
				t.Error("the source scene changed")
			}
			if tt.want != nil {
				tt.want(t, got, other, otherShader)
			}
		})
	}
}

// TestSceneMapContentSharesWhatDidNotChange checks that the layers none of whose content changed
// are the same layers in both scenes, and the copied ones do not name an old layer.
func TestSceneMapContentSharesWhatDidNotChange(t *testing.T) {
	src, a, _, _ := mapTestScene()
	other := testPicture()
	got := src.MapContent(func(p *Picture) *Picture {
		if p == a {
			return other
		}
		return p
	}, nil)
	srcClip := src.Root().Children()[0].Children()[0]
	gotClip := got.Root().Children()[0].Children()[0]
	if srcClip == gotClip {
		t.Error("the clip layer over the replaced picture is shared")
	}
	srcOpacity := src.Root().Children()[0].Children()[1]
	if got.Root().Children()[0].Children()[1] != srcOpacity {
		t.Error("the opacity layer, which did not change, is not shared")
	}
	if gotClip.(*ClipRectLayer).Rect != srcClip.(*ClipRectLayer).Rect {
		t.Error("the clip layer lost its rectangle")
	}
	if gotClip.OldLayer() != nil {
		t.Error("a copied layer names an old layer")
	}
}

func TestNewPictureFromOps(t *testing.T) {
	tests := []struct {
		name string
		cull Rect
		ops  []DisplayOp
	}{
		{"no operations", NewRectFromLTWH(0, 0, 10, 10), nil},
		{"two operations", NewRectFromLTWH(1, 2, 30, 40), []DisplayOp{OpSave{}, OpRestore{}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPictureFromOps(tt.cull, tt.ops)
			if p.CullRect() != tt.cull {
				t.Errorf("cull rect = %v, want %v", p.CullRect(), tt.cull)
			}
			if got := p.DisplayList().Ops(); len(got) != len(tt.ops) {
				t.Errorf("%d operations, want %d", len(got), len(tt.ops))
			}
		})
	}
}
