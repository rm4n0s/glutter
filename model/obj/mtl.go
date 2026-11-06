package obj

import (
	"bytes"
	"io/fs"
	"path"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/model"
	"github.com/rm4n0s/glutter/model/internal/parse"
)

// resolve is the path in the file system of a file that the OBJ or a material library names, or false when it cannot be one.
func (d *decoder) resolve(dir, name string) (string, bool) {
	p := path.Join(dir, strings.ReplaceAll(name, "\\", "/"))
	return p, fs.ValidPath(p)
}

// read reads a file that the OBJ or a material library names. It returns nil data, and no error, when the file cannot be opened
// (the note is in the log); an error is a read that failed or a context that is done.
func (d *decoder) read(what, dir, name string) (data []byte, p string, err error) {
	p, ok := d.resolve(dir, name)
	var f fs.File
	var openErr error
	if ok {
		f, openErr = d.opts.FS.Open(p)
	} else {
		openErr = fs.ErrInvalid
	}
	if openErr != nil {
		d.opts.logger().Warn("obj: a file could not be opened, the default is used", "what", what, "file", name, "err", openErr)
		return nil, p, nil
	}
	defer func() { _ = f.Close() }() // the file is read; a failed close of a read-only file loses nothing
	data, err = model.ReadAll(d.ctx, f)
	if err != nil {
		if appErr, isApp := errors.FromError(err); isApp {
			return nil, p, appErr.SetMetadata("file", p)
		}
		return nil, p, err
	}
	return data, p, nil
}

func (d *decoder) exists(name string) bool {
	if d.opts.FS == nil {
		return false
	}
	p, ok := d.resolve(d.opts.Dir, name)
	if !ok {
		return false
	}
	_, err := fs.Stat(d.opts.FS, p)
	return err == nil
}

// materialLibraries reads the libraries of a `mtllib` line. The names are separated by blanks, unless the whole rest of the line is
// the name of a file that is there.
func (d *decoder) materialLibraries(rest []byte) error {
	whole := string(parse.Trim(rest))
	if whole == "" {
		return nil
	}
	names := strings.Fields(whole)
	if len(names) > 1 && d.exists(whole) {
		names = []string{whole}
	}
	for _, name := range names {
		if d.libs[name] {
			continue
		}
		d.libs[name] = true
		if d.opts.FS == nil {
			if !d.missingFS {
				d.missingFS = true
				d.opts.logger().Warn("obj: material libraries are not read: the loader has no file system", "file", name)
			}
			continue
		}
		data, p, err := d.read("material library", d.opts.Dir, name)
		if err != nil {
			return err
		}
		if data == nil {
			continue
		}
		if err := d.parseMTL(data, path.Dir(p)); err != nil {
			return err
		}
	}
	return nil
}

// textureOptionArity is how many arguments each option of a texture map takes.
var textureOptionArity = map[string]int{
	"-blendu": 1, "-blendv": 1, "-bm": 1, "-boost": 1, "-cc": 1, "-clamp": 1, "-imfchan": 1, "-mm": 2, "-o": 3, "-s": 3, "-t": 3,
	"-texres": 1, "-type": 1,
}

// parseMTL reads a material library: newmtl, Kd, Ks, Ns, d and map_Kd. The rest is not used. A line that does not read is not an
// error of the OBJ file: it is left out, with a line in the log.
func (d *decoder) parseMTL(data []byte, dir string) error {
	var cur *model.Material
	var kd [3]float32
	alpha := float32(1)
	kdSet := false
	closeMaterial := func() {
		if cur == nil {
			return
		}
		if !kdSet {
			kd = [3]float32{model.DefaultColor[0], model.DefaultColor[1], model.DefaultColor[2]}
		}
		cur.BaseColor = [4]float32{kd[0], kd[1], kd[2], alpha}
		if alpha < 1 {
			cur.AlphaMode = model.AlphaBlend
		}
	}
	badLines := 0
	line := 0
	for off := 0; off < len(data); {
		line++
		if line%checkEvery == 0 {
			if err := d.ctx.Err(); err != nil {
				return errors.NewErr(model.TagModelReadFailed, err).SetMetadata("line", line, "offset", off, "library", dir)
			}
		}
		var text []byte
		if end := bytes.IndexByte(data[off:], '\n'); end < 0 {
			text, off = data[off:], len(data)
		} else {
			text, off = data[off:off+end], off+end+1
		}
		word, rest := parse.Field(text)
		if word == nil || word[0] == '#' {
			continue
		}
		key := strings.ToLower(string(word))
		switch key {
		case "newmtl":
			closeMaterial()
			cur = &model.Material{Name: string(parse.Trim(rest))}
			d.addMaterial(cur)
			kdSet, alpha = false, 1
			continue
		case "kd", "ks", "ns", "d", "map_kd":
		default:
			continue
		}
		if cur == nil {
			badLines++
			continue
		}
		var v [3]float64
		switch key {
		case "kd", "ks":
			n, _, ok := numbers(rest, v[:])
			if !ok || n < 3 {
				badLines++
				continue
			}
			c := [3]float32{float32(v[0]), float32(v[1]), float32(v[2])}
			if key == "kd" {
				kd, kdSet = c, true
			} else {
				cur.Specular = c
			}
		case "ns", "d":
			// "d -halo 0.5" has the value last.
			fields := bytes.Fields(rest)
			if len(fields) == 0 {
				badLines++
				continue
			}
			f, ok := parse.Float(fields[len(fields)-1])
			if !ok || !parse.Finite(f) {
				badLines++
				continue
			}
			if key == "ns" {
				cur.Shininess = float32(f)
			} else {
				alpha = float32(min(max(f, 0), 1))
			}
		case "map_kd":
			fields := strings.Fields(string(rest))
			i := 0
			for i < len(fields) && strings.HasPrefix(fields[i], "-") {
				i += 1 + textureOptionArity[strings.ToLower(fields[i])]
			}
			if i >= len(fields) {
				badLines++
				continue
			}
			img, err := d.image(dir, strings.Join(fields[i:], " "))
			if err != nil {
				return err
			}
			cur.BaseColorImage = img
		}
	}
	closeMaterial()
	if badLines > 0 {
		d.opts.logger().Warn("obj: lines of a material library were left out", "library", dir, "lines", badLines)
	}
	return nil
}

// image reads the image of a texture map, once for each file. It is nil when the file is not there.
func (d *decoder) image(dir, name string) (*model.Image, error) {
	p, ok := d.resolve(dir, name)
	if img, seen := d.images[p]; ok && seen {
		return img, nil
	}
	data, p, err := d.read("image", dir, name)
	if err != nil {
		return nil, err
	}
	if data == nil {
		d.images[p] = nil
		return nil, nil
	}
	img := &model.Image{Name: path.Base(p), URI: p, Type: model.DetectImageType(p, data), Data: data}
	d.images[p] = img
	d.m.Images = append(d.m.Images, img)
	return img, nil
}
