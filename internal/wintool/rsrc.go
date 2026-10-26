package wintool

import (
	"cmp"
	"encoding/binary"
	"slices"

	"github.com/rm4n0s/errors"
)

// The types of the resources of an .exe (RT_ICON and the like).
const (
	// RTIcon is the type of one image of an icon.
	//
	// Flutter: no counterpart
	RTIcon = 3
	// RTGroupIcon is the type of the directory of an icon's images.
	//
	// Flutter: no counterpart
	RTGroupIcon = 14
	// RTVersion is the type of the version information.
	//
	// Flutter: no counterpart
	RTVersion = 16
	// RTManifest is the type of the application manifest.
	//
	// Flutter: no counterpart
	RTManifest = 24
	// LangEnglishUS is the language of the resources (0x0409).
	//
	// Flutter: no counterpart
	LangEnglishUS = 0x0409
)

// Resource is one resource of an .exe: a type, an ID, a language and the bytes.
//
// Flutter: no counterpart
type Resource struct {
	Type, ID, Lang uint16
	Data           []byte
}

// Resources returns the resources of an app: the manifest (ID 1, RT_MANIFEST), the icon (RT_ICON
// with the IDs 1 to 7 and the group RT_GROUP_ICON with ID 1, which is the ID the platform loads for
// the window class) and the version information (ID 1).
//
// Flutter: no counterpart
func Resources(manifest []byte, icon *Icon, info VersionInfo) []Resource {
	res := []Resource{{Type: RTManifest, ID: 1, Lang: LangEnglishUS, Data: manifest}}
	for i, img := range icon.Images {
		res = append(res, Resource{Type: RTIcon, ID: uint16(i + 1), Lang: LangEnglishUS, Data: img.Data})
	}
	res = append(res, Resource{Type: RTGroupIcon, ID: 1, Lang: LangEnglishUS, Data: icon.Group})
	return append(res, Resource{Type: RTVersion, ID: 1, Lang: LangEnglishUS, Data: info.Encode()})
}

// BuildSyso writes the .syso file for the resources: a COFF object for the architecture with a .rsrc
// section that holds the resource directory (three levels: type, ID, language), the data entries
// and the data, and one relocation for each data entry, which the linker resolves to the entry's
// address in the image (SPEC §25.1). It fails with WindowsResourceInvalid when there are no
// resources, when two have the same type, ID and language, when one is empty, and for an unknown
// architecture.
//
// Flutter: no counterpart
func BuildSyso(arch string, resources []Resource) ([]byte, error) {
	a, ok := FindArch(arch)
	if !ok {
		return nil, errors.New(tagWindowsResourceInvalid, "the architecture is not amd64 or arm64", "arch", arch)
	}
	if len(resources) == 0 {
		return nil, errors.New(tagWindowsResourceInvalid, "there are no resources to write")
	}
	res := slices.Clone(resources)
	slices.SortFunc(res, func(x, y Resource) int {
		return cmp.Or(cmp.Compare(x.Type, y.Type), cmp.Compare(x.ID, y.ID), cmp.Compare(x.Lang, y.Lang))
	})
	for i, r := range res {
		if len(r.Data) == 0 {
			return nil, errors.New(tagWindowsResourceInvalid, "a resource is empty", "type", r.Type, "id", r.ID)
		}
		if i > 0 && res[i-1].Type == r.Type && res[i-1].ID == r.ID && res[i-1].Lang == r.Lang {
			return nil, errors.New(tagWindowsResourceInvalid, "two resources have the same type, ID and language", "type", r.Type, "id", r.ID, "lang", r.Lang)
		}
	}

	// The tree: types, then the IDs of each type, then the languages of each ID. count(...) are the
	// numbers of entries of each directory, in order.
	type group struct{ key, first, n int } // res[first:first+n] share key
	groups := func(lo, hi int, key func(Resource) int) []group {
		var out []group
		for i := lo; i < hi; {
			j := i
			for j < hi && key(res[j]) == key(res[i]) {
				j++
			}
			out = append(out, group{key: key(res[i]), first: i, n: j - i})
			i = j
		}
		return out
	}
	typeKey := func(r Resource) int { return int(r.Type) }
	idKey := func(r Resource) int { return int(r.ID) }
	types := groups(0, len(res), typeKey)

	// Offsets: the root, the directory of each type, the directory of each (type, ID), the data
	// entries (16 bytes each, one per resource) and then the data (aligned to 4).
	const dirSize, entrySize, dataEntrySize = 16, 8, 16
	off := dirSize + entrySize*len(types)
	typeOff := make([]int, len(types))
	idGroups := make([][]group, len(types))
	for t, tg := range types {
		typeOff[t] = off
		idGroups[t] = groups(tg.first, tg.first+tg.n, idKey)
		off += dirSize + entrySize*len(idGroups[t])
	}
	idOff := make([][]int, len(types))
	for t := range types {
		for _, ig := range idGroups[t] {
			idOff[t] = append(idOff[t], off)
			off += dirSize + entrySize*ig.n
		}
	}
	entryOff := off // res[i]'s data entry is at entryOff + 16*i
	dataOff := make([]int, len(res))
	off += dataEntrySize * len(res)
	for i, r := range res {
		off = (off + 3) &^ 3
		dataOff[i] = off
		off += len(r.Data)
	}
	size := (off + 3) &^ 3

	le := binary.LittleEndian
	b := make([]byte, size)
	dir := func(at, named, ids int) {
		le.PutUint16(b[at+12:], uint16(named))
		le.PutUint16(b[at+14:], uint16(ids))
	}
	const subdirectory = 0x80000000
	dir(0, 0, len(types))
	for t, tg := range types {
		le.PutUint32(b[dirSize+entrySize*t:], uint32(tg.key))
		le.PutUint32(b[dirSize+entrySize*t+4:], uint32(typeOff[t])|subdirectory)
		dir(typeOff[t], 0, len(idGroups[t]))
		for k, ig := range idGroups[t] {
			at := typeOff[t] + dirSize + entrySize*k
			le.PutUint32(b[at:], uint32(ig.key))
			le.PutUint32(b[at+4:], uint32(idOff[t][k])|subdirectory)
			dir(idOff[t][k], 0, ig.n)
			for l := 0; l < ig.n; l++ {
				i := ig.first + l
				at := idOff[t][k] + dirSize + entrySize*l
				le.PutUint32(b[at:], uint32(res[i].Lang))
				le.PutUint32(b[at+4:], uint32(entryOff+dataEntrySize*i)) // a leaf: no subdirectory flag
			}
		}
	}
	var relocs []coffReloc
	for i, r := range res {
		at := entryOff + dataEntrySize*i
		le.PutUint32(b[at:], uint32(dataOff[i])) // the offset in the section; the linker adds the address
		le.PutUint32(b[at+4:], uint32(len(r.Data)))
		// the code page and the reserved field stay 0
		copy(b[dataOff[i]:], r.Data)
		relocs = append(relocs, coffReloc{Offset: uint32(at), Type: a.AddrNB})
	}
	return writeCOFF(a.Machine, b, relocs), nil
}

// The tags of the errors that this file and versioninfo.go make (rm4n0s/errors, R5).
const (
	tagWindowsResourceInvalid = TagWindowsResourceInvalid

	// TagWindowsResourceInvalid is the tag of the error for resources that cannot be written: none,
	// an empty one, a duplicate, an unknown architecture, a version that is not up to four numbers.
	TagWindowsResourceInvalid = "WindowsResourceInvalid"
)
