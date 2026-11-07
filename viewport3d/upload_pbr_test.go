package viewport3d_test

import (
	"context"
	"image/color"
	"math"
	"os"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/model"
	"github.com/rm4n0s/glutter/model/gltf"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/viewport3d"
)

// TestUploadMakesPBRMaterialsOfGLTFMaterials checks what Upload makes of a metallic-roughness material: the values (the base color
// factor, which glTF has linear, in the sRGB that the material takes), the textures with their samplers, mip levels and transforms.
func TestUploadMakesPBRMaterialsOfGLTFMaterials(t *testing.T) {
	rig := newDrawRig(t, viewport3d.NewScene(), frontCamera())
	dev := rig.frame.Device
	img := &model.Image{Name: "px.png", Type: "image/png", Data: pngBytes(t, 8, 4, color.NRGBA{R: 200, G: 100, B: 50, A: 255})}
	clamp := model.Sampler{Mag: model.FilterNearest, WrapU: model.WrapClampToEdge, WrapV: model.WrapMirroredRepeat}
	base := &model.TextureRef{Texture: &model.Texture{Image: img, Sampler: clamp}}
	normal := &model.TextureRef{Texture: &model.Texture{Image: img, Sampler: clamp}, TexCoord: 1,
		Transform: &model.TextureTransform{Offset: [2]float32{0.5, 0}, Rotation: 0.5, Scale: [2]float32{2, 2}}}
	missing := &model.TextureRef{Texture: &model.Texture{}} // an image that the loader could not read
	mat := &model.Material{PBR: true, BaseColor: [4]float32{0.2, 0.5, 1, 0.5}, Metallic: 0.3, Roughness: 0.7, Emissive: [3]float32{2, 1, 0}, NormalScale: 0.8, OcclusionStrength: 0.6,
		AlphaMode: model.AlphaBlend, AlphaCutoff: 0.4, DoubleSided: true, BaseColorTexture: base, NormalTexture: normal, EmissiveTexture: missing}
	mesh := triangleMesh("pbr")
	mesh.Material = mat
	mesh.Tangents = []float32{1, 0, 0, 1, 1, 0, 0, 1, 1, 0, 0, 1}
	mesh.TexCoords1 = []float32{0, 0, 1, 0, 0, 1}
	node := model.NewNode("n")
	node.Meshes = []*model.Mesh{mesh}
	u, err := viewport3d.Upload(context.Background(), dev, &model.Model{Nodes: []*model.Node{node}})
	if err != nil {
		t.Fatal(err)
	}
	defer u.Release()
	pm, ok := u.Parts[0].Item.Material().(*viewport3d.PBRMaterial)
	if !ok {
		t.Fatalf("the material is %T", u.Parts[0].Item.Material())
	}
	c := pm.Color()
	for i, p := range []struct{ got, linear float32 }{{c.R, 0.2}, {c.G, 0.5}, {c.B, 1}} {
		// The sRGB value, decoded, is the factor of the file.
		lin := float32(math.Pow((float64(p.got)+0.055)/1.055, 2.4))
		if math.Abs(float64(lin-p.linear)) > 1e-4 {
			t.Errorf("channel %d: the color decodes to %v, the file has %v", i, lin, p.linear)
		}
	}
	if c.A != 0.5 || pm.Metallic() != float64(float32(0.3)) || pm.Roughness() != float64(float32(0.7)) || pm.AlphaMode() != viewport3d.AlphaModeBlend || !pm.DoubleSided() || !pm.Transparent() {
		t.Errorf("alpha %v, metallic %v, roughness %v, mode %v, double sided %v", c.A, pm.Metallic(), pm.Roughness(), pm.AlphaMode(), pm.DoubleSided())
	}
	if e := pm.Emissive(); e != [3]float64{2, 1, 0} {
		t.Errorf("emissive %v", e)
	}
	bt, nt := pm.Texture(viewport3d.PBRSlotBaseColor), pm.Texture(viewport3d.PBRSlotNormal)
	if bt.Texture == nil || nt.Texture == nil || bt.Texture != nt.Texture || bt.Sampler == nil || bt.Sampler != nt.Sampler {
		t.Fatalf("the textures %+v %+v: an image and a sampler are made once", bt, nt)
	}
	if bt.Texture.MipLevelCount() != 4 {
		t.Errorf("an 8 by 4 image has %d mip levels, want 4", bt.Texture.MipLevelCount())
	}
	if nt.TexCoord != 1 || nt.Transform == nil || nt.Transform.Rotation != 0.5 || nt.Transform.Scale != [2]float32{2, 2} || nt.Transform.Offset != [2]float32{0.5, 0} {
		t.Errorf("the normal texture: %+v %+v", nt, nt.Transform)
	}
	if pm.Texture(viewport3d.PBRSlotEmissive).Texture != nil || pm.Texture(viewport3d.PBRSlotOcclusion).Texture != nil {
		t.Error("a texture that the model does not have, or whose image was not read, is there")
	}
	d := u.Parts[0].Item.Mesh().Data()
	if len(d.Tangents) != 12 || len(d.TexCoords1) != 6 {
		t.Errorf("the mesh lost its tangents or second texture coordinates: %d %d", len(d.Tangents), len(d.TexCoords1))
	}
}

// TestUploadGivesASkinnedMeshWithNoMaterialTheDefaultLookInPBR checks that a mesh that only the PBR material can move is made one.
func TestUploadGivesASkinnedMeshWithNoMaterialTheDefaultLookInPBR(t *testing.T) {
	rig := newDrawRig(t, viewport3d.NewScene(), frontCamera())
	skinned, morphing, plain := triangleMesh("skinned"), triangleMesh("morphing"), triangleMesh("plain")
	skinned.Joints, skinned.Weights = []uint16{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, []float32{1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0}
	morphing.MorphTargets = []model.MorphTarget{{Positions: make([]float32, 9)}}
	node := model.NewNode("n")
	node.Meshes = []*model.Mesh{skinned, morphing, plain}
	u, err := viewport3d.Upload(context.Background(), rig.frame.Device, &model.Model{Nodes: []*model.Node{node}})
	if err != nil {
		t.Fatal(err)
	}
	defer u.Release()
	kinds := []bool{}
	for _, p := range u.Parts {
		_, pbr := p.Item.Material().(*viewport3d.PBRMaterial)
		kinds = append(kinds, pbr)
	}
	if len(kinds) != 3 || !kinds[0] || !kinds[1] || kinds[2] {
		t.Errorf("the materials are PBR: %v, want [true true false]", kinds)
	}
}

// poseModel is a model of a skinned triangle on two nodes of a chain, a mesh with a morph target on a node of its own, and an animation
// of the joint's translation and the weights.
func poseModel() (*model.Model, *model.Node, *model.Node, *model.Node) {
	skinned := triangleMesh("skinned")
	skinned.Joints, skinned.Weights = []uint16{0, 1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0}, []float32{0.5, 0.5, 0, 0, 0.5, 0.5, 0, 0, 0.5, 0.5, 0, 0}
	morphing := triangleMesh("morphing")
	morphing.MorphTargets = []model.MorphTarget{{Positions: []float32{0, 0, 1, 0, 0, 1, 0, 0, 1}}}
	morphing.MorphWeights = []float32{0.1}
	root, joint, ball := model.NewNode("root"), model.NewNode("joint"), model.NewNode("ball")
	root.TRS = &model.TRS{Translation: vecmath.Vec3{X: 10}, Rotation: vecmath.QuatIdentity(), Scale: vecmath.Vec3{X: 1, Y: 1, Z: 1}}
	root.Transform = root.TRS.Matrix()
	joint.TRS = &model.TRS{Translation: vecmath.Vec3{X: 1}, Rotation: vecmath.QuatIdentity(), Scale: vecmath.Vec3{X: 1, Y: 1, Z: 1}}
	joint.Transform = joint.TRS.Matrix()
	root.Children = []*model.Node{joint}
	root.Meshes = []*model.Mesh{skinned}
	root.Skin = &model.Skin{Joints: []*model.Node{root, joint}, InverseBindMatrices: []vecmath.Mat4{vecmath.NewMat4Translation(vecmath.Vec3{X: -10}), vecmath.NewMat4Translation(vecmath.Vec3{X: -11})}}
	ball.Meshes = []*model.Mesh{morphing}
	return &model.Model{Nodes: []*model.Node{root, ball}, Animations: []*model.Animation{{Name: "move", Channels: []model.AnimationChannel{
		{Node: joint, Path: model.PathTranslation, Times: []float32{0, 1}, Values: []float32{1, 0, 0, 1, 2, 0}},
		{Node: ball, Path: model.PathWeights, Times: []float32{0, 1}, Values: []float32{0, 1}},
	}}}}, root, joint, ball
}

// TestSetPoseMovesTheItemsTheSkinsAndTheWeights checks Uploaded.SetPose: the nodes' world transforms follow the pose, the items follow
// their nodes, the joints of a skin get their nodes' world transforms, and the weights come from the pose, the node or the mesh.
func TestSetPoseMovesTheItemsTheSkinsAndTheWeights(t *testing.T) {
	rig := newDrawRig(t, viewport3d.NewScene(), frontCamera())
	m, root, _, _ := poseModel()
	u, err := viewport3d.Upload(context.Background(), rig.frame.Device, m, viewport3d.UploadOpts{Transform: foundation.Some(vecmath.NewMat4Translation(vecmath.Vec3{Z: 100}))})
	if err != nil {
		t.Fatal(err)
	}
	defer u.Release()
	skinnedItem, ballItem := u.Parts[0].Item, u.Parts[1].Item
	skin := skinnedItem.Skin()
	if skin == nil || skin.JointCount() != 2 || ballItem.Skin() != nil {
		t.Fatalf("skin %v", skin)
	}
	at := func(x, y, z float64) vecmath.Mat4 { return vecmath.NewMat4Translation(vecmath.Vec3{X: x, Y: y, Z: z}) }
	// At rest (Upload leaves the model so): the root at x = 10, the joint at x = 11, everything 100 behind.
	if skinnedItem.Transform() != at(10, 0, 100) || skin.JointWorld(1) != at(11, 0, 100) || skin.JointWorld(0) != at(10, 0, 100) {
		t.Errorf("at rest the item is at %v, the joints at %v and %v", skinnedItem.Transform().Translation(), skin.JointWorld(0).Translation(), skin.JointWorld(1).Translation())
	}
	if w := ballItem.MorphWeights(); len(w) != 1 || w[0] != 0.1 {
		t.Errorf("at rest the weights are %v, the mesh's 0.1", w)
	}
	pose := viewport3d.NewPose()
	viewport3d.NewAnimation(m.Animations[0]).Sample(1, pose)
	u.SetPose(pose)
	if got := skin.JointWorld(1); got != at(11, 2, 100) {
		t.Errorf("the joint is at %v, want (11, 2, 100)", got.Translation())
	}
	if got := skin.JointWorld(0); got != at(10, 0, 100) {
		t.Errorf("the root joint moved to %v", got.Translation())
	}
	if w := ballItem.MorphWeights(); len(w) != 1 || w[0] != 1 {
		t.Errorf("the weights are %v, want [1]", w)
	}
	// A node's own weights replace the mesh's, and a pose's replace the node's.
	pose.Clear()
	u.SetPose(pose)
	if w := ballItem.MorphWeights(); len(w) != 1 || w[0] != 0.1 {
		t.Errorf("after the pose was cleared the weights are %v", w)
	}
	m.Nodes[1].MorphWeights = []float32{0.7}
	u.SetPose(nil)
	if w := ballItem.MorphWeights(); len(w) != 1 || w[0] != 0.7 {
		t.Errorf("the node's weights are %v, want [0.7]", w)
	}
	// The pose changes the root: the child and the joint follow.
	root.TRS = nil
	u.SetPose(nil)
	if got := skin.JointWorld(1); got != at(11, 0, 100) {
		t.Errorf("after SetPose(nil) the joint is at %v, want (11, 0, 100)", got.Translation())
	}
}

// TestSampleModelsPlayTheirAnimations loads CC0 models of the Khronos group, uploads them and plays their animations at a few times:
// the skin of SimpleSkin bends, the weights of AnimatedMorphCube change, and every pose is made without a message of the log.
func TestSampleModelsPlayTheirAnimations(t *testing.T) {
	rig := newDrawRig(t, viewport3d.NewScene(), frontCamera())
	fsys := os.DirFS("../model/gltf/testdata/sample-assets")
	load := func(name string) (*model.Model, *viewport3d.Uploaded) {
		m, err := gltf.Load(context.Background(), fsys, name)
		if err != nil {
			t.Fatal(err)
		}
		u, err := viewport3d.Upload(context.Background(), rig.frame.Device, m)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(u.Release)
		return m, u
	}
	m, u := load("SimpleSkin/glTF/SimpleSkin.gltf")
	skin := u.Parts[0].Item.Skin()
	if skin == nil {
		t.Fatal("SimpleSkin has no skin")
	}
	anim := viewport3d.NewAnimation(m.Animations[0])
	pose := viewport3d.NewPose()
	rest := skin.JointWorld(1)
	anim.Sample(0.75*anim.Duration(), pose)
	u.SetPose(pose)
	if skin.JointWorld(1) == rest {
		t.Error("the joint of SimpleSkin did not move")
	}
	anim.Sample(0, pose)
	u.SetPose(pose)
	if skin.JointWorld(1).NearlyEqual(rest, 1e-5) == false {
		t.Errorf("at time 0 the joint is %v, at rest %v", skin.JointWorld(1), rest)
	}

	m, u = load("AnimatedMorphCube/glTF-Binary/AnimatedMorphCube.glb")
	anim = viewport3d.NewAnimation(m.Animations[0])
	sum := func() float32 {
		var s float32
		for _, p := range u.Parts {
			for _, w := range p.Item.MorphWeights() {
				s += w
			}
		}
		return s
	}
	anim.Sample(0, pose)
	u.SetPose(pose)
	a := sum()
	anim.Sample(0.5*anim.Duration(), pose)
	u.SetPose(pose)
	if b := sum(); b == a {
		t.Errorf("the weights of AnimatedMorphCube are %v at both times", a)
	}
}

// TestPoseAndSetPoseAllocateNothingOnceMade is the allocation test of the animation: sampling into a pose that has been sampled once
// and showing it allocate nothing.
func TestPoseAndSetPoseAllocateNothingOnceMade(t *testing.T) {
	if raceEnabled {
		t.Skip("the race detector changes what a call allocates")
	}
	rig := newDrawRig(t, viewport3d.NewScene(), frontCamera())
	rig.rec.Device().Quiet = true
	m, _, _, _ := poseModel()
	u, err := viewport3d.Upload(context.Background(), rig.frame.Device, m)
	if err != nil {
		t.Fatal(err)
	}
	defer u.Release()
	anim := viewport3d.NewAnimation(m.Animations[0])
	pose := viewport3d.NewPose()
	tm := float32(0)
	step := func() {
		tm += 0.01
		anim.Sample(tm, pose)
		u.SetPose(pose)
	}
	step()
	if got := testing.AllocsPerRun(100, step); got != 0 {
		t.Errorf("sampling and showing a pose allocates %v times, want 0", got)
	}
}
