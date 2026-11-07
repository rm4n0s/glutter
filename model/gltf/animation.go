package gltf

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/model"
)

// keyframes is the input and the output of a sampler of an animation, shared by the channels that use it.
type keyframes struct {
	times         []float32
	values        []float32
	interpolation model.Interpolation
	output        *accessor
}

// buildAnimations makes the animations.
func (l *loader) buildAnimations() error {
	for i := range l.doc.Animations {
		if err := l.checkContext(); err != nil {
			return err
		}
		aj := &l.doc.Animations[i]
		anim := &model.Animation{Name: aj.Name}

		samplers := make([]*keyframes, len(aj.Samplers))
		type target struct {
			node int
			path model.AnimationPath
		}
		seen := map[target]bool{}

		for k := range aj.Channels {
			cj := &aj.Channels[k]
			cp := ptr("animations", i, "channels", k)
			if cj.Sampler == nil {
				return errors.New(TagGLTFInvalid, "an animation channel has no sampler", "path", cp, "reason", "missing property")
			}
			if *cj.Sampler < 0 || *cj.Sampler >= len(aj.Samplers) {
				return errors.New(TagGLTFInvalid, "an animation channel has a sampler that does not exist", "path", cp+"/sampler", "reason", "index out of range", "index", *cj.Sampler)
			}

			var path model.AnimationPath
			switch cj.Target.Path {
			case "translation":
				path = model.PathTranslation
			case "rotation":
				path = model.PathRotation
			case "scale":
				path = model.PathScale
			case "weights":
				path = model.PathWeights
			case "pointer":
				l.note("animation pointer", "glTF: an animation channel that targets a pointer is not read and is left out")
				continue
			default:
				return errors.New(TagGLTFInvalid, "an animation channel has a path that is not known", "path", cp+"/target/path", "reason", "bad path", "animationPath", cj.Target.Path)
			}
			if cj.Target.Node == nil {
				l.note("animation without node", "glTF: an animation channel that has no node is not read and is left out")
				continue
			}
			node := *cj.Target.Node
			if node < 0 || node >= len(l.nodes) {
				return errors.New(TagGLTFInvalid, "an animation channel has a node that does not exist", "path", cp+"/target/node", "reason", "index out of range", "index", node)
			}
			if seen[target{node, path}] {
				return errors.New(TagGLTFInvalid, "two animation channels change the same path of the same node", "path", cp+"/target", "reason", "duplicate channel")
			}
			seen[target{node, path}] = true

			// the number of morph targets that the weights of the node's mesh have
			width := 0
			switch path {
			case model.PathTranslation, model.PathScale:
				width = 3
			case model.PathRotation:
				width = 4
			default:
				nj := &l.doc.Nodes[node]
				if nj.Mesh == nil {
					return errors.New(TagGLTFInvalid, "an animation channel changes the weights of a node that has no mesh", "path", cp+"/target", "reason", "no mesh")
				}
				width = len(l.doc.Meshes[*nj.Mesh].Primitives[0].Targets)
				if width == 0 {
					return errors.New(TagGLTFInvalid, "an animation channel changes the weights of a mesh that has no morph targets", "path", cp+"/target", "reason", "no morph targets")
				}
			}

			kf := samplers[*cj.Sampler]
			if kf == nil {
				var err error
				if kf, err = l.keyframes(&aj.Samplers[*cj.Sampler], ptr("animations", i, "samplers", *cj.Sampler)); err != nil {
					return err
				}
				samplers[*cj.Sampler] = kf
			}

			// the output has a keyframe for each time, three for a cubic spline, and each keyframe has width values
			per := 1
			if kf.interpolation == model.InterpolationCubicSpline {
				per = 3
				if len(kf.times) < 2 {
					return errors.New(TagGLTFInvalid, "a cubic spline has fewer than two keyframes", "path", cp+"/sampler", "reason", "bad count")
				}
			}
			if len(kf.values) != per*len(kf.times)*width {
				return errors.New(TagGLTFInvalid, "the output of a sampler does not have the values of its keyframes", "path", cp+"/sampler", "reason", "bad count", "values", len(kf.values), "want", per*len(kf.times)*width)
			}
			wantType := map[model.AnimationPath]string{model.PathTranslation: "VEC3", model.PathRotation: "VEC4", model.PathScale: "VEC3", model.PathWeights: "SCALAR"}[path]
			if kf.output.typ != wantType {
				return errors.New(TagGLTFInvalid, "the output of a sampler is not of the type of its path", "path", cp+"/sampler", "reason", "bad accessor type", "type", kf.output.typ, "want", wantType)
			}

			anim.Channels = append(anim.Channels, model.AnimationChannel{
				Node: l.nodes[node], Path: path, Interpolation: kf.interpolation, Times: kf.times, Values: kf.values,
			})
		}
		l.m.Animations = append(l.m.Animations, anim)
	}
	return nil
}

// keyframes reads a sampler.
func (l *loader) keyframes(sj *animationSamplerJSON, where string) (*keyframes, error) {
	if sj.Input == nil || sj.Output == nil {
		return nil, errors.New(TagGLTFInvalid, "an animation sampler has no input or no output", "path", where, "reason", "missing property")
	}
	kf := &keyframes{}
	switch sj.Interpolation {
	case "", "LINEAR":
		kf.interpolation = model.InterpolationLinear
	case "STEP":
		kf.interpolation = model.InterpolationStep
	case "CUBICSPLINE":
		kf.interpolation = model.InterpolationCubicSpline
	default:
		return nil, errors.New(TagGLTFInvalid, "an animation sampler has an interpolation that is not known", "path", where+"/interpolation", "reason", "bad interpolation", "interpolation", sj.Interpolation)
	}

	in, err := l.accessor(*sj.Input, where+"/input")
	if err != nil {
		return nil, err
	}
	if in.typ != "SCALAR" || in.ct != ctFloat {
		return nil, errors.New(TagGLTFInvalid, "the input of an animation sampler is not SCALAR floats", "path", where+"/input", "reason", "bad accessor type", "type", in.typ, "componentType", in.ct)
	}
	if kf.times, err = l.floats(in); err != nil {
		return nil, err
	}
	for k, t := range kf.times {
		if t < 0 || (k > 0 && t <= kf.times[k-1]) {
			return nil, errors.New(TagGLTFInvalid, "the times of an animation sampler are not positive and increasing", "path", where+"/input", "reason", "bad times", "item", k)
		}
	}

	out, err := l.accessor(*sj.Output, where+"/output")
	if err != nil {
		return nil, err
	}
	if out.ct != ctFloat && !out.normalized {
		return nil, errors.New(TagGLTFInvalid, "the output of an animation sampler is not floats or normalized integers", "path", where+"/output", "reason", "bad componentType", "componentType", out.ct)
	}
	if kf.values, err = l.floats(out); err != nil {
		return nil, err
	}
	kf.output = out
	return kf, nil
}
