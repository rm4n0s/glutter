package gltf_test

import (
	"bytes"
	"context"
	"math"
	"testing"

	"github.com/rm4n0s/glutter/model/gltf"
)

// node0 is the first node of the document of a builder.
func node0(b *builder) obj { return b.doc["nodes"].(list)[0].(obj) }

// prim0 is the first primitive of the first mesh.
func prim0(b *builder) obj {
	return b.doc["meshes"].(list)[0].(obj)["primitives"].(list)[0].(obj)
}

func TestInvalidFiles(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(b *builder)
		fn     string // the function that makes the error, at the end of the route
		path   string
		reason string
	}{
		{"no asset version", func(b *builder) { b.doc["asset"] = obj{} }, "loader.build", "/asset/version", "missing property"},
		{"glTF 1", func(b *builder) { b.doc["asset"] = obj{"version": "1.0"} }, "loader.build", "/asset/version", "bad version"},
		{"newer minVersion", func(b *builder) { b.doc["asset"] = obj{"version": "2.0", "minVersion": "2.1"} }, "loader.build", "/asset/minVersion", "bad version"},
		{"accessor without count", func(b *builder) { b.accessors[0].(obj)["count"] = 0 }, "loader.accessor", "/accessors/0/count", "bad count"},
		{"accessor of an unknown component type", func(b *builder) { b.accessors[0].(obj)["componentType"] = 5124 }, "loader.accessor", "/accessors/0/componentType", "bad componentType"},
		{"accessor of an unknown type", func(b *builder) { b.accessors[0].(obj)["type"] = "VEC5" }, "loader.accessor", "/accessors/0/type", "bad type"},
		{"normalized floats", func(b *builder) { b.accessors[0].(obj)["normalized"] = true }, "loader.accessor", "/accessors/0/normalized", "bad normalized"},
		{"negative byteOffset", func(b *builder) { b.accessors[0].(obj)["byteOffset"] = -4 }, "loader.accessor", "/accessors/0/byteOffset", "bad byteOffset"},
		{"accessor past its view", func(b *builder) { b.accessors[0].(obj)["count"] = 4 }, "loader.accessor", "/accessors/0", "accessor out of bounds"},
		{"accessor offset past its view", func(b *builder) { b.accessors[0].(obj)["byteOffset"] = 1 << 40 }, "loader.accessor", "/accessors/0", "accessor out of bounds"},
		{"view of a buffer that is not there", func(b *builder) { b.views[0].(obj)["buffer"] = 3 }, "loader.buffer", "/bufferViews/0/buffer", "index out of range"},
		{"view past its buffer", func(b *builder) { b.views[0].(obj)["byteLength"] = 1 << 20 }, "loader.view", "/bufferViews/0", "view out of bounds"},
		{"view with a stride of 5", func(b *builder) { b.views[0].(obj)["byteStride"] = 5 }, "loader.view", "/bufferViews/0/byteStride", "bad byteStride"},
		{"stride shorter than the element", func(b *builder) { b.views[0].(obj)["byteStride"] = 8 }, "loader.accessor", "/accessors/0", "bad byteStride"},
		{"view without length", func(b *builder) { delete(b.views[0].(obj), "byteLength") }, "loader.view", "/bufferViews/0", "missing property"},
		{"accessor of a view that is not there", func(b *builder) { b.accessors[0].(obj)["bufferView"] = 9 }, "loader.view", "/accessors/0/bufferView", "index out of range"},
		{"accessor that is not there", func(b *builder) { prim0(b)["attributes"].(obj)["POSITION"] = 7 }, "loader.accessor", "/meshes/0/primitives/0/attributes/POSITION", "index out of range"},
		{"no position", func(b *builder) { delete(prim0(b)["attributes"].(obj), "POSITION") }, "loader.primitive", "/meshes/0/primitives/0/attributes", "no POSITION"},
		{"position of the wrong type", func(b *builder) { b.accessors[0].(obj)["type"] = "VEC2" }, "loader.checkAttribute", "/meshes/0/primitives/0/attributes/POSITION", "bad accessor type"},
		{"unknown mode", func(b *builder) { prim0(b)["mode"] = 7 }, "loader.primitive", "/meshes/0/primitives/0/mode", "bad mode"},
		{"material that is not there", func(b *builder) { prim0(b)["material"] = 0 }, "loader.primitive", "/meshes/0/primitives/0/material", "index out of range"},
		{"mesh without primitives", func(b *builder) { b.doc["meshes"].(list)[0].(obj)["primitives"] = list{} }, "loader.buildMeshes", "/meshes/0/primitives", "no primitives"},
		{"mesh that is not there", func(b *builder) { node0(b)["mesh"] = 4 }, "loader.linkNodes", "/nodes/0/mesh", "index out of range"},
		{"child that is not there", func(b *builder) { node0(b)["children"] = list{5} }, "loader.linkNodes", "/nodes/0/children/0", "index out of range"},
		{"node with two parents", func(b *builder) {
			b.doc["nodes"] = list{obj{"mesh": 0, "children": list{1, 1}}, obj{}}
		}, "loader.linkNodes", "/nodes/0/children/1", "two parents"},
		{"matrix and translation", func(b *builder) {
			node0(b)["matrix"] = list{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
			node0(b)["translation"] = list{0, 0, 0}
		}, "loader.buildNodes", "/nodes/0", "matrix and TRS"},
		{"matrix of 15", func(b *builder) { node0(b)["matrix"] = list{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0} }, "loader.buildNodes", "/nodes/0/matrix", "bad matrix"},
		{"rotation of length 0", func(b *builder) { node0(b)["rotation"] = list{0, 0, 0, 0} }, "loader.buildNodes", "/nodes/0/rotation", "bad rotation"},
		{"translation of two", func(b *builder) { node0(b)["translation"] = list{1, 2} }, "loader.buildNodes", "/nodes/0/translation", "bad translation"},
		{"scale of four", func(b *builder) { node0(b)["scale"] = list{1, 2, 3, 4} }, "loader.buildNodes", "/nodes/0/scale", "bad scale"},
		{"camera that is not there", func(b *builder) { node0(b)["camera"] = 1 }, "loader.buildNodes", "/nodes/0/camera", "index out of range"},
		{"skin that is not there", func(b *builder) { node0(b)["skin"] = 1 }, "loader.linkNodes", "/nodes/0/skin", "index out of range"},
		{"scene that is not there", func(b *builder) { b.doc["scene"] = 3 }, "loader.buildScenes", "/scene", "index out of range"},
		{"scene node that is not there", func(b *builder) { b.doc["scenes"] = list{obj{"nodes": list{8}}} }, "loader.buildScenes", "/scenes/0/nodes/0", "index out of range"},
		{"unknown alpha mode", func(b *builder) {
			prim0(b)["material"] = 0
			b.doc["materials"] = list{obj{"alphaMode": "GLOW"}}
		}, "loader.material", "/materials/0/alphaMode", "bad alphaMode"},
		{"emissive factor out of range", func(b *builder) {
			prim0(b)["material"] = 0
			b.doc["materials"] = list{obj{"emissiveFactor": list{2, 0, 0}}}
		}, "loader.material", "/materials/0/emissiveFactor", "bad factor"},
		{"roughness out of range", func(b *builder) {
			b.doc["materials"] = list{obj{"pbrMetallicRoughness": obj{"roughnessFactor": 2}}}
		}, "loader.material", "/materials/0/pbrMetallicRoughness/roughnessFactor", "bad factor"},
		{"texture that is not there", func(b *builder) {
			b.doc["materials"] = list{obj{"normalTexture": obj{"index": 1}}}
		}, "loader.textureRef", "/materials/0/normalTexture/index", "index out of range"},
		{"texture info without index", func(b *builder) {
			b.doc["materials"] = list{obj{"emissiveTexture": obj{}}}
		}, "loader.textureRef", "/materials/0/emissiveTexture", "missing property"},
		{"texture of an image that is not there", func(b *builder) {
			b.doc["textures"] = list{obj{"source": 2}}
		}, "loader.buildMaterials", "/textures/0/source", "index out of range"},
		{"texture of a sampler that is not there", func(b *builder) {
			b.doc["textures"] = list{obj{"sampler": 2}}
		}, "loader.buildMaterials", "/textures/0/sampler", "index out of range"},
		{"image without data", func(b *builder) {
			b.doc["images"] = list{obj{}}
			b.doc["textures"] = list{obj{"source": 0}}
		}, "loader.image", "/images/0", "missing property"},
		{"image that is not base64", func(b *builder) {
			b.doc["images"] = list{obj{"uri": "data:image/png,abc"}}
			b.doc["textures"] = list{obj{"source": 0}}
		}, "loader.resource", "/images/0/uri", "bad data URI"},
		{"image of bad base64", func(b *builder) {
			b.doc["images"] = list{obj{"uri": "data:image/png;base64,@@@"}}
			b.doc["textures"] = list{obj{"source": 0}}
		}, "loader.resource", "/images/0/uri", "bad data URI"},
		{"image of a view without a type", func(b *builder) {
			b.doc["images"] = list{obj{"bufferView": 0}}
			b.doc["textures"] = list{obj{"source": 0}}
		}, "loader.image", "/images/0", "missing property"},
		{"sampler of an unknown filter", func(b *builder) {
			b.doc["samplers"] = list{obj{"magFilter": 1}}
			b.doc["textures"] = list{obj{"sampler": 0}}
		}, "loader.sampler", "/samplers/0/magFilter", "bad filter"},
		{"sampler of an unknown wrap", func(b *builder) {
			b.doc["samplers"] = list{obj{"wrapS": 1}}
			b.doc["textures"] = list{obj{"sampler": 0}}
		}, "loader.sampler", "/samplers/0", "bad wrap"},
		{"camera of an unknown type", func(b *builder) { b.doc["cameras"] = list{obj{"type": "fisheye"}} }, "loader.buildCameras", "/cameras/0/type", "bad type"},
		{"perspective camera without yfov", func(b *builder) {
			b.doc["cameras"] = list{obj{"type": "perspective", "perspective": obj{"znear": 1}}}
		}, "loader.buildCameras", "/cameras/0/perspective", "missing property"},
		{"orthographic camera with the far plane before the near", func(b *builder) {
			b.doc["cameras"] = list{obj{"type": "orthographic", "orthographic": obj{"xmag": 1, "ymag": 1, "znear": 2, "zfar": 1}}}
		}, "loader.buildCameras", "/cameras/0/orthographic", "bad camera"},
		{"light of an unknown type", func(b *builder) {
			b.doc["extensionsUsed"] = list{"KHR_lights_punctual"}
			b.doc["extensions"] = obj{"KHR_lights_punctual": obj{"lights": list{obj{"type": "area"}}}}
		}, "loader.buildLights", "/extensions/KHR_lights_punctual/lights/0/type", "bad type"},
		{"spot light with the inner cone wider than the outer", func(b *builder) {
			b.doc["extensionsUsed"] = list{"KHR_lights_punctual"}
			b.doc["extensions"] = obj{"KHR_lights_punctual": obj{"lights": list{obj{"type": "spot", "spot": obj{"innerConeAngle": 1, "outerConeAngle": 0.5}}}}}
		}, "loader.buildLights", "/extensions/KHR_lights_punctual/lights/0/spot", "bad cone angle"},
		{"node of a light that is not there", func(b *builder) {
			b.doc["extensionsUsed"] = list{"KHR_lights_punctual"}
			b.doc["extensions"] = obj{"KHR_lights_punctual": obj{"lights": list{}}}
			node0(b)["extensions"] = obj{"KHR_lights_punctual": obj{"light": 0}}
		}, "loader.buildNodes", "/nodes/0/extensions/KHR_lights_punctual/light", "index out of range"},
		{"skin without joints", func(b *builder) {
			b.doc["skins"] = list{obj{"joints": list{}}}
		}, "loader.buildSkins", "/skins/0/joints", "no joints"},
		{"skin of a joint that is not there", func(b *builder) {
			b.doc["skins"] = list{obj{"joints": list{9}}}
		}, "loader.buildSkins", "/skins/0/joints/0", "index out of range"},
		{"skin of a joint twice", func(b *builder) {
			b.doc["skins"] = list{obj{"joints": list{0, 0}}}
		}, "loader.buildSkins", "/skins/0/joints/1", "duplicate joint"},
		{"skin of a skeleton that is not there", func(b *builder) {
			b.doc["skins"] = list{obj{"joints": list{0}, "skeleton": 4}}
		}, "loader.buildSkins", "/skins/0/skeleton", "index out of range"},
		{"inverse bind matrices of the wrong type", func(b *builder) {
			b.doc["skins"] = list{obj{"joints": list{0}, "inverseBindMatrices": 0}}
		}, "loader.buildSkins", "/skins/0/inverseBindMatrices", "bad accessor type"},
		{"texture transform with an offset of one number", func(b *builder) {
			b.doc["extensionsUsed"] = list{"KHR_texture_transform"}
			b.doc["images"] = list{obj{"uri": "data:image/png;base64,AAAA"}}
			b.doc["textures"] = list{obj{"source": 0}}
			b.doc["materials"] = list{obj{"emissiveTexture": obj{"index": 0, "extensions": obj{"KHR_texture_transform": obj{"offset": list{1}}}}}}
		}, "loader.textureRef", "/materials/0/emissiveTexture/extensions/KHR_texture_transform/offset", "bad offset"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := oneTriangle()
			tt.mutate(b)
			_, err := decodeJSON(t, b.json())
			failure(t, err, tt.fn+".GLTFInvalid", map[string]any{"path": tt.path, "reason": tt.reason})
		})
	}
}

func TestInvalidCycle(t *testing.T) {
	// a node that is the child of two nodes, two nodes that are each other's children, and a node that is its own child
	b := oneTriangle()
	b.doc["nodes"] = list{obj{"mesh": 0, "children": list{1}}, obj{"children": list{2}}, obj{"children": list{1}}}
	_, err := decodeJSON(t, b.json())
	failure(t, err, "loader.linkNodes.GLTFInvalid", map[string]any{"reason": "two parents"})

	b = oneTriangle()
	b.doc["nodes"] = list{obj{"mesh": 0, "children": list{1}}, obj{"children": list{0}}}
	_, err = decodeJSON(t, b.json())
	failure(t, err, "loader.linkNodes.GLTFInvalid", map[string]any{"reason": "cycle"})

	b = oneTriangle()
	b.doc["nodes"] = list{obj{"mesh": 0, "children": list{0}}}
	_, err = decodeJSON(t, b.json())
	failure(t, err, "loader.linkNodes.GLTFInvalid", map[string]any{"reason": "cycle"})
}

func TestInvalidData(t *testing.T) {
	tests := []struct {
		name   string
		build  func() *builder
		fn     string
		path   string
		reason string
	}{
		{"index past the vertices", func() *builder {
			b := newBuilder()
			p := b.triangle()
			i := b.data(u16s(0, 1, 3), ctUshort, "SCALAR", 3, nil)
			b.scene(obj{"attributes": obj{"POSITION": p}, "indices": i})
			return b
		}, "loader.primitive", "/meshes/0/primitives/0/indices", "index out of range"},
		{"indices that are not scalar", func() *builder {
			b := newBuilder()
			p := b.triangle()
			i := b.data(u16s(0, 1, 2, 0), ctUshort, "VEC2", 2, nil)
			b.scene(obj{"attributes": obj{"POSITION": p}, "indices": i})
			return b
		}, "loader.primitive", "/meshes/0/primitives/0/indices", "bad accessor type"},
		{"indices of floats", func() *builder {
			b := newBuilder()
			p := b.triangle()
			i := b.data(f32s(0, 1, 2), ctFloat, "SCALAR", 3, nil)
			b.scene(obj{"attributes": obj{"POSITION": p}, "indices": i})
			return b
		}, "loader.uints", "/accessors/1", "bad componentType"},
		{"two indices for a triangle", func() *builder {
			b := newBuilder()
			p := b.triangle()
			i := b.data(u16s(0, 1, 2, 0), ctUshort, "SCALAR", 4, nil)
			b.scene(obj{"attributes": obj{"POSITION": p}, "indices": i})
			return b
		}, "loader.primitive", "/meshes/0/primitives/0", "bad count"},
		{"three indices for lines", func() *builder {
			b := newBuilder()
			p := b.triangle()
			i := b.data(u16s(0, 1, 2), ctUshort, "SCALAR", 3, nil)
			b.scene(obj{"attributes": obj{"POSITION": p}, "indices": i, "mode": 1})
			return b
		}, "loader.primitive", "/meshes/0/primitives/0", "bad count"},
		{"attributes of different counts", func() *builder {
			b := newBuilder()
			p := b.triangle()
			n := b.data(f32s(0, 0, 1, 0, 0, 1), ctFloat, "VEC3", 2, nil)
			b.scene(obj{"attributes": obj{"POSITION": p, "NORMAL": n}})
			return b
		}, "loader.primitive", "/meshes/0/primitives/0/attributes/POSITION", "bad count"},
		{"a position that is not a number", func() *builder {
			b := newBuilder()
			p := b.data(f32s(0, 0, 0, 1, float32(math.NaN()), 0, 0, 1, 0), ctFloat, "VEC3", 3, nil)
			b.scene(obj{"attributes": obj{"POSITION": p}})
			return b
		}, "loader.floats", "/accessors/0", "not finite"},
		{"joints without weights", func() *builder {
			b := newBuilder()
			p := b.triangle()
			j := b.data([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, ctUbyte, "VEC4", 3, nil)
			b.scene(obj{"attributes": obj{"POSITION": p, "JOINTS_0": j}})
			return b
		}, "loader.primitive", "/meshes/0/primitives/0/attributes", "joints and weights"},
		{"joints as floats", func() *builder {
			b := newBuilder()
			p := b.triangle()
			j := b.data(f32s(0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0), ctFloat, "VEC4", 3, nil)
			b.scene(obj{"attributes": obj{"POSITION": p, "JOINTS_0": j}})
			return b
		}, "loader.checkAttribute", "/meshes/0/primitives/0/attributes/JOINTS_0", "bad componentType"},
		{"a joint that the skin does not have", func() *builder {
			b := newBuilder()
			p := b.triangle()
			j := b.data([]byte{0, 0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0}, ctUbyte, "VEC4", 3, nil)
			w := b.data(f32s(1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0), ctFloat, "VEC4", 3, nil)
			b.scene(obj{"attributes": obj{"POSITION": p, "JOINTS_0": j, "WEIGHTS_0": w}})
			node0(b)["skin"] = 0
			b.doc["skins"] = list{obj{"joints": list{0}}}
			return b
		}, "loader.checkSkinnedMeshes", "/nodes/0/skin", "joint out of range"},
		{"morph targets that differ between primitives", func() *builder {
			b := newBuilder()
			p := b.triangle()
			d := b.data(f32s(0, 0, 1, 0, 0, 1, 0, 0, 1), ctFloat, "VEC3", 3, nil)
			b.scene(obj{"attributes": obj{"POSITION": p}, "targets": list{obj{"POSITION": d}}}, obj{"attributes": obj{"POSITION": p}})
			return b
		}, "loader.buildMeshes", "/meshes/0/primitives/1/targets", "bad targets"},
		{"weights that are not as many as the targets", func() *builder {
			b := newBuilder()
			p := b.triangle()
			d := b.data(f32s(0, 0, 1, 0, 0, 1, 0, 0, 1), ctFloat, "VEC3", 3, nil)
			b.scene(obj{"attributes": obj{"POSITION": p}, "targets": list{obj{"POSITION": d}}})
			b.doc["meshes"].(list)[0].(obj)["weights"] = list{0, 1}
			return b
		}, "loader.buildMeshes", "/meshes/0/weights", "bad weights"},
		{"a morph target of another count", func() *builder {
			b := newBuilder()
			p := b.triangle()
			d := b.data(f32s(0, 0, 1, 0, 0, 1), ctFloat, "VEC3", 2, nil)
			b.scene(obj{"attributes": obj{"POSITION": p}, "targets": list{obj{"POSITION": d}}})
			return b
		}, "loader.morphTarget", "/meshes/0/primitives/0/targets/0/POSITION", "bad count"},
		{"an accessor of zeros that is too large", func() *builder {
			b := newBuilder()
			b.triangle()
			pos := b.accessor(-1, ctFloat, "VEC3", 1<<30, nil)
			b.scene(obj{"attributes": obj{"POSITION": pos}})
			return b
		}, "loader.accessor", "/accessors/1/count", "bad count"},
		{"sparse indices that do not increase", func() *builder {
			b := newBuilder()
			base := b.view(f32s(0, 0, 0, 1, 0, 0, 0, 1, 0), 0)
			sparse := obj{"count": 2, "indices": obj{"bufferView": b.view([]byte{2, 1}, 0), "componentType": ctUbyte}, "values": obj{"bufferView": b.view(f32s(1, 1, 1, 2, 2, 2), 0)}}
			p := b.accessor(base, ctFloat, "VEC3", 3, obj{"sparse": sparse})
			b.scene(obj{"attributes": obj{"POSITION": p}})
			return b
		}, "loader.sparse", "/accessors/0/sparse/indices", "bad sparse index"},
		{"a sparse index past the accessor", func() *builder {
			b := newBuilder()
			base := b.view(f32s(0, 0, 0, 1, 0, 0, 0, 1, 0), 0)
			sparse := obj{"count": 1, "indices": obj{"bufferView": b.view([]byte{3}, 0), "componentType": ctUbyte}, "values": obj{"bufferView": b.view(f32s(1, 1, 1), 0)}}
			p := b.accessor(base, ctFloat, "VEC3", 3, obj{"sparse": sparse})
			b.scene(obj{"attributes": obj{"POSITION": p}})
			return b
		}, "loader.sparse", "/accessors/0/sparse/indices", "bad sparse index"},
		{"a sparse count past the accessor", func() *builder {
			b := newBuilder()
			base := b.view(f32s(0, 0, 0, 1, 0, 0, 0, 1, 0), 0)
			sparse := obj{"count": 4, "indices": obj{"bufferView": b.view([]byte{0, 1, 2, 3}, 0), "componentType": ctUbyte}, "values": obj{"bufferView": b.view(f32s(1, 1, 1), 0)}}
			p := b.accessor(base, ctFloat, "VEC3", 3, obj{"sparse": sparse})
			b.scene(obj{"attributes": obj{"POSITION": p}})
			return b
		}, "loader.sparse", "/accessors/0/sparse/count", "bad count"},
		{"sparse values past their view", func() *builder {
			b := newBuilder()
			base := b.view(f32s(0, 0, 0, 1, 0, 0, 0, 1, 0), 0)
			sparse := obj{"count": 2, "indices": obj{"bufferView": b.view([]byte{0, 1}, 0), "componentType": ctUbyte}, "values": obj{"bufferView": b.view(f32s(1, 1, 1), 0)}}
			p := b.accessor(base, ctFloat, "VEC3", 3, obj{"sparse": sparse})
			b.scene(obj{"attributes": obj{"POSITION": p}})
			return b
		}, "loader.sparse", "/accessors/0/sparse/values", "accessor out of bounds"},
		{"sparse indices of floats", func() *builder {
			b := newBuilder()
			base := b.view(f32s(0, 0, 0, 1, 0, 0, 0, 1, 0), 0)
			sparse := obj{"count": 1, "indices": obj{"bufferView": b.view(f32s(1), 0), "componentType": ctFloat}, "values": obj{"bufferView": b.view(f32s(1, 1, 1), 0)}}
			p := b.accessor(base, ctFloat, "VEC3", 3, obj{"sparse": sparse})
			b.scene(obj{"attributes": obj{"POSITION": p}})
			return b
		}, "loader.sparse", "/accessors/0/sparse/indices/componentType", "bad componentType"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := decodeJSON(t, tt.build().json())
			failure(t, err, tt.fn+".GLTFInvalid", map[string]any{"path": tt.path, "reason": tt.reason})
		})
	}
}

func TestInvalidAnimations(t *testing.T) {
	build := func(mutate func(b *builder, anim obj)) *builder {
		b := oneTriangle()
		times := b.data(f32s(0, 1), ctFloat, "SCALAR", 2, nil)
		trans := b.data(f32s(0, 0, 0, 1, 1, 1), ctFloat, "VEC3", 2, nil)
		rot := b.data(f32s(0, 0, 0, 1, 0, 0, 0, 1), ctFloat, "VEC4", 2, nil)
		weights := b.data(f32s(0, 1), ctFloat, "SCALAR", 2, nil)
		anim := obj{
			"samplers": list{obj{"input": times, "output": trans}, obj{"input": times, "output": rot}, obj{"input": times, "output": weights}},
			"channels": list{obj{"sampler": 0, "target": obj{"node": 0, "path": "translation"}}},
		}
		mutate(b, anim)
		b.doc["animations"] = list{anim}
		return b
	}
	chan0 := func(anim obj) obj { return anim["channels"].(list)[0].(obj) }
	smp := func(anim obj, i int) obj { return anim["samplers"].(list)[i].(obj) }
	tests := []struct {
		name   string
		mutate func(b *builder, anim obj)
		fn     string
		path   string
		reason string
	}{
		{"channel without sampler", func(b *builder, a obj) { delete(chan0(a), "sampler") }, "loader.buildAnimations", "/animations/0/channels/0", "missing property"},
		{"sampler that is not there", func(b *builder, a obj) { chan0(a)["sampler"] = 5 }, "loader.buildAnimations", "/animations/0/channels/0/sampler", "index out of range"},
		{"unknown path", func(b *builder, a obj) { chan0(a)["target"] = obj{"node": 0, "path": "color"} }, "loader.buildAnimations", "/animations/0/channels/0/target/path", "bad path"},
		{"node that is not there", func(b *builder, a obj) { chan0(a)["target"] = obj{"node": 3, "path": "translation"} }, "loader.buildAnimations", "/animations/0/channels/0/target/node", "index out of range"},
		{"the same path twice", func(b *builder, a obj) {
			a["channels"] = append(a["channels"].(list), chan0(a))
		}, "loader.buildAnimations", "/animations/0/channels/1/target", "duplicate channel"},
		{"weights of a mesh without targets", func(b *builder, a obj) {
			chan0(a)["sampler"] = 2
			chan0(a)["target"] = obj{"node": 0, "path": "weights"}
		}, "loader.buildAnimations", "/animations/0/channels/0/target", "no morph targets"},
		{"rotation with the output of a translation", func(b *builder, a obj) {
			chan0(a)["target"] = obj{"node": 0, "path": "rotation"}
		}, "loader.buildAnimations", "/animations/0/channels/0/sampler", "bad count"},
		{"unknown interpolation", func(b *builder, a obj) { smp(a, 0)["interpolation"] = "SMOOTH" }, "loader.keyframes", "/animations/0/samplers/0/interpolation", "bad interpolation"},
		{"sampler without output", func(b *builder, a obj) { delete(smp(a, 0), "output") }, "loader.keyframes", "/animations/0/samplers/0", "missing property"},
		{"cubic spline with two keyframes of values for one", func(b *builder, a obj) { smp(a, 0)["interpolation"] = "CUBICSPLINE" }, "loader.buildAnimations", "/animations/0/channels/0/sampler", "bad count"},
		{"times that go back", func(b *builder, a obj) {
			smp(a, 0)["input"] = b.data(f32s(1, 0), ctFloat, "SCALAR", 2, nil)
		}, "loader.keyframes", "/animations/0/samplers/0/input", "bad times"},
		{"negative times", func(b *builder, a obj) {
			smp(a, 0)["input"] = b.data(f32s(-1, 0), ctFloat, "SCALAR", 2, nil)
		}, "loader.keyframes", "/animations/0/samplers/0/input", "bad times"},
		{"times that are not floats", func(b *builder, a obj) {
			smp(a, 0)["input"] = b.data(u16s(0, 1), ctUshort, "SCALAR", 2, nil)
		}, "loader.keyframes", "/animations/0/samplers/0/input", "bad accessor type"},
		{"output of integers that are not normalized", func(b *builder, a obj) {
			smp(a, 0)["output"] = b.data(u16s(0, 0, 0, 1, 1, 1), ctUshort, "VEC3", 2, nil)
		}, "loader.keyframes", "/animations/0/samplers/0/output", "bad componentType"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := decodeJSON(t, build(tt.mutate).json())
			failure(t, err, tt.fn+".GLTFInvalid", map[string]any{"path": tt.path, "reason": tt.reason})
		})
	}
}

func TestInvalidContainers(t *testing.T) {
	good := oneTriangle().glb()
	set := func(data []byte, at int, v uint32) []byte {
		out := bytes.Clone(data)
		copy(out[at:], u32s(v))
		return out
	}
	tests := []struct {
		name   string
		data   []byte
		route  string
		path   string
		reason string
		offset int
	}{
		{"JSON that is cut", []byte(`{"asset":`), "decode.GLTFInvalid", "", "the JSON is malformed", 9},
		{"not JSON", []byte(`gltf`), "decode.GLTFInvalid", "", "the JSON is malformed", 1},
		{"empty", nil, "decode.GLTFInvalid", "", "the JSON is malformed", 0},
		{"a version that is a number", []byte(`{"asset":{"version":2}}`), "decode.GLTFInvalid", "asset.version", "a value has the wrong type", 21},
		{"binary file that is short", good[:16], "decode.GLTFInvalid", "", "the binary file is shorter than its header and its first chunk header", 16},
		{"binary container of another version", set(good, 4, 1), "decode.GLTFInvalid", "", "the binary container is not version 2", 4},
		{"binary file of a wrong length", set(good, 8, uint32(len(good)+4)), "decode.GLTFInvalid", "", "the length of the binary file is not the length of its data", 8},
		{"chunk longer than the file", set(good, 12, 1<<20), "decode.GLTFInvalid", "", "a chunk is longer than the file", 12},
		{"first chunk that is not JSON", set(good, 16, 0x004E4942), "decode.GLTFInvalid", "", "the first chunk is not the JSON chunk", 12},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := gltf.Decode(context.Background(), bytes.NewReader(tt.data), gltf.Options{Logger: discard()})
			meta := map[string]any{"path": tt.path, "reason": tt.reason}
			if tt.offset > 0 {
				meta["offset"] = tt.offset
			}
			failure(t, err, tt.route+"", meta)
		})
	}
}

func TestInvalidBuffers(t *testing.T) {
	// the buffer of a binary file must be its first, and every buffer must have the bytes of its length
	t.Run("a buffer of a binary file that is not the first", func(t *testing.T) {
		b := oneTriangle()
		js := b.json()
		js = bytes.Replace(js, []byte(`"buffers":[`), []byte(`"buffers":[{"byteLength":4},`), 1)
		_, err := decodeJSON(t, js)
		failure(t, err, "loader.buffer.GLTFInvalid", map[string]any{"path": "/buffers/0", "reason": "no data"})
	})
	t.Run("a buffer shorter than its length", func(t *testing.T) {
		b := oneTriangle()
		js := bytes.Replace(b.json(), []byte(`"byteLength":36,"uri"`), []byte(`"byteLength":400,"uri"`), 1)
		_, err := decodeJSON(t, js)
		failure(t, err, "loader.buffer.GLTFInvalid", map[string]any{"path": "/buffers/0", "reason": "short buffer"})
	})
	t.Run("a buffer without a length", func(t *testing.T) {
		b := oneTriangle()
		js := bytes.Replace(b.json(), []byte(`"byteLength":36,"uri"`), []byte(`"uri"`), 1)
		_, err := decodeJSON(t, js)
		failure(t, err, "loader.buffer.GLTFInvalid", map[string]any{"path": "/buffers/0", "reason": "no byteLength"})
	})
	t.Run("a data URI with no comma", func(t *testing.T) {
		b := oneTriangle()
		js := bytes.Replace(b.json(), []byte(`data:application/octet-stream;base64,`), []byte(`data:`), 1)
		_, err := decodeJSON(t, js)
		failure(t, err, "loader.resource.GLTFInvalid", map[string]any{"path": "/buffers/0/uri", "reason": "bad data URI"})
	})
}
