package threemf

import (
	"archive/zip"
	"encoding/xml"
	"io"
	"strings"

	"github.com/rm4n0s/errors"
)

// relationships is the XML of _rels/.rels: the relationships of the package.
type relationships struct {
	Relationship []struct {
		Type   string `xml:"Type,attr"`
		Target string `xml:"Target,attr"`
	} `xml:"Relationship"`
}

// findModelPart finds the model part of the archive: the target of the relationship of the 3D model type in _rels/.rels, or
// 3D/3dmodel.model when the package has no such relationship. Names are compared in lower case, as the packaging convention says.
func findModelPart(files map[string]*zip.File) (*zip.File, error) {
	target := defaultModelPart
	if rels, ok := files["_rels/.rels"]; ok {
		rc, err := rels.Open()
		if err != nil {
			return nil, errors.NewErr(TagThreeMFInvalid, err).SetMetadata("reason", "bad part", "part", rels.Name)
		}
		defer func() { _ = rc.Close() }() // the part is read; a failed close of a read-only part loses nothing
		var r relationships
		if err := xml.NewDecoder(io.LimitReader(rc, maxRelsBytes)).Decode(&r); err != nil {
			return nil, errors.NewErr(TagThreeMFInvalid, err).SetMetadata("reason", "bad relationships", "part", rels.Name)
		}
		for _, rel := range r.Relationship {
			if rel.Type == modelRelationship {
				target = rel.Target
				break
			}
		}
	}
	part, ok := files[strings.ToLower(strings.TrimPrefix(target, "/"))]
	if !ok {
		return nil, errors.New(TagThreeMFInvalid, "the archive has no model part", "reason", "no model part", "part", target)
	}
	return part, nil
}
