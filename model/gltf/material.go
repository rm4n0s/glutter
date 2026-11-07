package gltf

import (
	"encoding/json"
	"slices"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/model"
)

// The values of the filters and the wraps of a sampler (OpenGL's constants).
const (
	filterNearest              = 9728
	filterLinear               = 9729
	filterNearestMipmapNearest = 9984
	filterLinearMipmapNearest  = 9985
	filterNearestMipmapLinear  = 9986
	filterLinearMipmapLinear   = 9987
	wrapClampToEdge            = 33071
	wrapMirroredRepeat         = 33648
	wrapRepeat                 = 10497
)

// buildMaterials makes the images, the textures and the materials.
func (l *loader) buildMaterials() error {
	doc := l.doc
	l.images = make([]*model.Image, len(doc.Images))
	used := make([]bool, len(doc.Images))
	for i := range doc.Textures {
		if s := doc.Textures[i].Source; s != nil && *s >= 0 && *s < len(doc.Images) {
			used[*s] = true
		}
	}
	for i := range doc.Images {
		if !used[i] {
			continue
		}
		if err := l.checkContext(); err != nil {
			return err
		}
		img, err := l.image(i)
		if err != nil {
			return err
		}
		l.images[i] = img
		l.m.Images = append(l.m.Images, img)
	}

	for i := range doc.Textures {
		tj := &doc.Textures[i]
		here := ptr("textures", i)
		t := &model.Texture{Name: tj.Name}
		if tj.Source != nil {
			if *tj.Source < 0 || *tj.Source >= len(doc.Images) {
				return errors.New(TagGLTFInvalid, "a texture has an image that does not exist", "path", here+"/source", "reason", "index out of range", "index", *tj.Source)
			}
			t.Image = l.images[*tj.Source]
		}
		if tj.Sampler != nil {
			if *tj.Sampler < 0 || *tj.Sampler >= len(doc.Samplers) {
				return errors.New(TagGLTFInvalid, "a texture has a sampler that does not exist", "path", here+"/sampler", "reason", "index out of range", "index", *tj.Sampler)
			}
			s, err := l.sampler(*tj.Sampler)
			if err != nil {
				return err
			}
			t.Sampler = s
		}
		l.textures = append(l.textures, t)
		l.m.Textures = append(l.m.Textures, t)
	}

	for i := range doc.Materials {
		if err := l.checkContext(); err != nil {
			return err
		}
		m, err := l.material(i)
		if err != nil {
			return err
		}
		l.materials = append(l.materials, m)
		l.m.Materials = append(l.m.Materials, m)
	}
	return nil
}

// image reads the bytes of an image: from a URI, or from a buffer view.
func (l *loader) image(i int) (*model.Image, error) {
	ij := &l.doc.Images[i]
	here := ptr("images", i)
	img := &model.Image{Name: ij.Name, Type: ij.MimeType}
	switch {
	case ij.URI != "" && ij.BufferView != nil:
		return nil, errors.New(TagGLTFInvalid, "an image has a uri and a bufferView", "path", here, "reason", "uri and bufferView")
	case ij.URI != "":
		data, err := l.resource(ij.URI, here+"/uri")
		if err != nil {
			return nil, err
		}
		img.Data = data
		if len(ij.URI) < 5 || ij.URI[:5] != "data:" {
			img.URI = ij.URI
		}
	case ij.BufferView != nil:
		if ij.MimeType == "" {
			return nil, errors.New(TagGLTFInvalid, "an image of a buffer view has no mimeType", "path", here, "reason", "missing property")
		}
		v, err := l.view(*ij.BufferView, here+"/bufferView")
		if err != nil {
			return nil, err
		}
		img.Data = slices.Clone(v.data)
	default:
		return nil, errors.New(TagGLTFInvalid, "an image has no uri and no bufferView", "path", here, "reason", "missing property")
	}
	if img.Type == "" {
		img.Type = model.DetectImageType(ij.URI, img.Data)
	}
	return img, nil
}

// sampler makes the sampler i.
func (l *loader) sampler(i int) (model.Sampler, error) {
	sj := &l.doc.Samplers[i]
	here := ptr("samplers", i)
	var s model.Sampler
	if sj.MagFilter != nil {
		switch *sj.MagFilter {
		case filterNearest:
			s.Mag = model.FilterNearest
		case filterLinear:
			s.Mag = model.FilterLinear
		default:
			return s, errors.New(TagGLTFInvalid, "a sampler has a magFilter that is not known", "path", here+"/magFilter", "reason", "bad filter", "filter", *sj.MagFilter)
		}
	}
	if sj.MinFilter != nil {
		switch *sj.MinFilter {
		case filterNearest:
			s.Min = model.FilterNearest
		case filterLinear:
			s.Min = model.FilterLinear
		case filterNearestMipmapNearest:
			s.Min, s.Mipmap = model.FilterNearest, model.FilterNearest
		case filterLinearMipmapNearest:
			s.Min, s.Mipmap = model.FilterLinear, model.FilterNearest
		case filterNearestMipmapLinear:
			s.Min, s.Mipmap = model.FilterNearest, model.FilterLinear
		case filterLinearMipmapLinear:
			s.Min, s.Mipmap = model.FilterLinear, model.FilterLinear
		default:
			return s, errors.New(TagGLTFInvalid, "a sampler has a minFilter that is not known", "path", here+"/minFilter", "reason", "bad filter", "filter", *sj.MinFilter)
		}
	}
	for k, w := range [2]*int{sj.WrapS, sj.WrapT} {
		if w == nil {
			continue
		}
		var wrap model.Wrap
		switch *w {
		case wrapRepeat:
			wrap = model.WrapRepeat
		case wrapMirroredRepeat:
			wrap = model.WrapMirroredRepeat
		case wrapClampToEdge:
			wrap = model.WrapClampToEdge
		default:
			return s, errors.New(TagGLTFInvalid, "a sampler has a wrap that is not known", "path", here, "reason", "bad wrap", "wrap", *w)
		}
		if k == 0 {
			s.WrapU = wrap
		} else {
			s.WrapV = wrap
		}
	}
	return s, nil
}

// textureRef makes the use of a texture of a material (the textureInfo at where).
func (l *loader) textureRef(tj *textureInfoJSON, where string) (*model.TextureRef, error) {
	if tj == nil {
		return nil, nil
	}
	if tj.Index == nil {
		return nil, errors.New(TagGLTFInvalid, "a texture info has no index", "path", where, "reason", "missing property")
	}
	if *tj.Index < 0 || *tj.Index >= len(l.textures) {
		return nil, errors.New(TagGLTFInvalid, "a texture info has a texture that does not exist", "path", where+"/index", "reason", "index out of range", "index", *tj.Index)
	}
	ref := &model.TextureRef{Texture: l.textures[*tj.Index]}
	if tj.TexCoord != nil {
		if *tj.TexCoord < 0 {
			return nil, errors.New(TagGLTFInvalid, "a texture info has a negative texCoord", "path", where+"/texCoord", "reason", "bad texCoord")
		}
		ref.TexCoord = *tj.TexCoord
	}
	if raw, ok := tj.Extensions[extensionTextureTransform]; ok && l.supported(extensionTextureTransform) {
		ep := where + "/extensions/" + extensionTextureTransform
		var ext textureTransformJSON
		if err := json.Unmarshal(raw, &ext); err != nil {
			return nil, errors.NewErr(TagGLTFInvalid, err).SetMetadata("path", ep, "reason", "a value has the wrong type")
		}
		tr := model.TextureTransform{Scale: [2]float32{1, 1}}
		if ext.Offset != nil {
			if len(ext.Offset) != 2 || !inRange(ext.Offset, -1e30, 1e30) {
				return nil, errors.New(TagGLTFInvalid, "a texture transform has an offset that is not two numbers", "path", ep+"/offset", "reason", "bad offset")
			}
			copy(tr.Offset[:], ext.Offset)
		}
		if ext.Rotation != nil {
			if !finite(*ext.Rotation) {
				return nil, errors.New(TagGLTFInvalid, "a texture transform has a rotation that is not a number", "path", ep+"/rotation", "reason", "bad rotation")
			}
			tr.Rotation = *ext.Rotation
		}
		if ext.Scale != nil {
			if len(ext.Scale) != 2 || !inRange(ext.Scale, -1e30, 1e30) {
				return nil, errors.New(TagGLTFInvalid, "a texture transform has a scale that is not two numbers", "path", ep+"/scale", "reason", "bad scale")
			}
			copy(tr.Scale[:], ext.Scale)
		}
		if ext.TexCoord != nil {
			if *ext.TexCoord < 0 {
				return nil, errors.New(TagGLTFInvalid, "a texture transform has a negative texCoord", "path", ep+"/texCoord", "reason", "bad texCoord")
			}
			ref.TexCoord = *ext.TexCoord
		}
		ref.Transform = &tr
	}
	if ref.TexCoord > 1 {
		l.note("texcoord set", "glTF: a texture reads a set of texture coordinates above 1, which meshes do not have", "texCoord", ref.TexCoord)
	}
	return ref, nil
}

// material makes the material i.
func (l *loader) material(i int) (*model.Material, error) {
	mj := &l.doc.Materials[i]
	here := ptr("materials", i)
	m := &model.Material{
		Name: mj.Name, PBR: true,
		BaseColor: [4]float32{1, 1, 1, 1}, Metallic: 1, Roughness: 1,
		NormalScale: 1, OcclusionStrength: 1,
		AlphaCutoff: 0.5, DoubleSided: mj.DoubleSided,
	}
	var err error

	if p := mj.PBR; p != nil {
		pp := here + "/pbrMetallicRoughness"
		if p.BaseColorFactor != nil {
			if len(p.BaseColorFactor) != 4 || !inRange(p.BaseColorFactor, 0, 1) {
				return nil, errors.New(TagGLTFInvalid, "a material has a baseColorFactor that is not four numbers from 0 to 1", "path", pp+"/baseColorFactor", "reason", "bad factor")
			}
			copy(m.BaseColor[:], p.BaseColorFactor)
		}
		if p.MetallicFactor != nil {
			if !inRange([]float32{*p.MetallicFactor}, 0, 1) {
				return nil, errors.New(TagGLTFInvalid, "a material has a metallicFactor that is not from 0 to 1", "path", pp+"/metallicFactor", "reason", "bad factor")
			}
			m.Metallic = *p.MetallicFactor
		}
		if p.RoughnessFactor != nil {
			if !inRange([]float32{*p.RoughnessFactor}, 0, 1) {
				return nil, errors.New(TagGLTFInvalid, "a material has a roughnessFactor that is not from 0 to 1", "path", pp+"/roughnessFactor", "reason", "bad factor")
			}
			m.Roughness = *p.RoughnessFactor
		}
		if m.BaseColorTexture, err = l.textureRef(p.BaseColorTexture, pp+"/baseColorTexture"); err != nil {
			return nil, err
		}
		if m.MetallicRoughnessTexture, err = l.textureRef(p.MetallicRoughnessTexture, pp+"/metallicRoughnessTexture"); err != nil {
			return nil, err
		}
	}
	if m.BaseColorTexture != nil {
		m.BaseColorImage = m.BaseColorTexture.Texture.Image
	}

	if m.NormalTexture, err = l.textureRef(mj.NormalTexture, here+"/normalTexture"); err != nil {
		return nil, err
	}
	if t := mj.NormalTexture; t != nil && t.Scale != nil {
		if !finite(*t.Scale) {
			return nil, errors.New(TagGLTFInvalid, "a normal texture has a scale that is not a number", "path", here+"/normalTexture/scale", "reason", "bad scale")
		}
		m.NormalScale = *t.Scale
	}
	if m.OcclusionTexture, err = l.textureRef(mj.OcclusionTexture, here+"/occlusionTexture"); err != nil {
		return nil, err
	}
	if t := mj.OcclusionTexture; t != nil && t.Strength != nil {
		if !inRange([]float32{*t.Strength}, 0, 1) {
			return nil, errors.New(TagGLTFInvalid, "an occlusion texture has a strength that is not from 0 to 1", "path", here+"/occlusionTexture/strength", "reason", "bad strength")
		}
		m.OcclusionStrength = *t.Strength
	}
	if m.EmissiveTexture, err = l.textureRef(mj.EmissiveTexture, here+"/emissiveTexture"); err != nil {
		return nil, err
	}

	if mj.EmissiveFactor != nil {
		if len(mj.EmissiveFactor) != 3 || !inRange(mj.EmissiveFactor, 0, 1) {
			return nil, errors.New(TagGLTFInvalid, "a material has an emissiveFactor that is not three numbers from 0 to 1", "path", here+"/emissiveFactor", "reason", "bad factor")
		}
		copy(m.Emissive[:], mj.EmissiveFactor)
	}
	if raw, ok := mj.Extensions[extensionEmissiveStrength]; ok && l.supported(extensionEmissiveStrength) {
		ep := here + "/extensions/" + extensionEmissiveStrength
		var ext struct {
			EmissiveStrength *float32 `json:"emissiveStrength"`
		}
		if err := json.Unmarshal(raw, &ext); err != nil {
			return nil, errors.NewErr(TagGLTFInvalid, err).SetMetadata("path", ep, "reason", "a value has the wrong type")
		}
		strength := float32(1)
		if ext.EmissiveStrength != nil {
			strength = *ext.EmissiveStrength
		}
		if strength < 0 || !finite(strength) {
			return nil, errors.New(TagGLTFInvalid, "a material has an emissiveStrength that is negative", "path", ep+"/emissiveStrength", "reason", "bad strength")
		}
		for c := range m.Emissive {
			m.Emissive[c] *= strength
		}
	}
	if _, ok := mj.Extensions[extensionUnlit]; ok && l.supported(extensionUnlit) {
		m.Unlit = true
	}

	switch mj.AlphaMode {
	case "", "OPAQUE":
		m.AlphaMode = model.AlphaOpaque
	case "MASK":
		m.AlphaMode = model.AlphaMask
	case "BLEND":
		m.AlphaMode = model.AlphaBlend
	default:
		return nil, errors.New(TagGLTFInvalid, "a material has an alphaMode that is not known", "path", here+"/alphaMode", "reason", "bad alphaMode", "alphaMode", mj.AlphaMode)
	}
	if mj.AlphaCutoff != nil {
		if *mj.AlphaCutoff < 0 || !finite(*mj.AlphaCutoff) {
			return nil, errors.New(TagGLTFInvalid, "a material has an alphaCutoff that is negative", "path", here+"/alphaCutoff", "reason", "bad alphaCutoff")
		}
		m.AlphaCutoff = *mj.AlphaCutoff
	}
	return m, nil
}
