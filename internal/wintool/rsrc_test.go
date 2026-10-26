package wintool

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"slices"
	"testing"

	"github.com/rm4n0s/errors"
)

// readResources walks the resource directory in the data of a .rsrc section: three levels (type, ID,
// language) down to the data entries. base is subtracted from the offsets of the data (the RVA of
// the section for a linked image, 0 for an object file, whose offsets are relative to the section).
// It fails the test on a structure that is wrong.
func readResources(t testing.TB, data []byte, base uint32) []Resource {
	t.Helper()
	le := binary.LittleEndian
	type dirEntry struct {
		id     uint32
		offset uint32
		sub    bool
	}
	readDir := func(at uint32) []dirEntry {
		if int(at)+16 > len(data) {
			t.Fatalf("directory at %d is outside the section of %d bytes", at, len(data))
		}
		named, ids := int(le.Uint16(data[at+12:])), int(le.Uint16(data[at+14:]))
		if named != 0 {
			t.Fatalf("named entries are not written")
		}
		var out []dirEntry
		for i := 0; i < ids; i++ {
			p := int(at) + 16 + 8*i
			if p+8 > len(data) {
				t.Fatalf("entry %d of the directory at %d is outside the section", i, at)
			}
			id, off := le.Uint32(data[p:]), le.Uint32(data[p+4:])
			out = append(out, dirEntry{id: id, offset: off &^ 0x80000000, sub: off&0x80000000 != 0})
		}
		for i := 1; i < len(out); i++ {
			if out[i-1].id >= out[i].id {
				t.Fatalf("the entries of the directory at %d are not sorted by ID", at)
			}
		}
		return out
	}
	var res []Resource
	types := readDir(0)
	for _, typ := range types {
		if !typ.sub {
			t.Fatalf("type %d is not a subdirectory", typ.id)
		}
		ids := readDir(typ.offset)
		for _, id := range ids {
			if !id.sub {
				t.Fatalf("ID %d is not a subdirectory", id.id)
			}
			langs := readDir(id.offset)
			for _, l := range langs {
				if l.sub || int(l.offset)+16 > len(data) {
					t.Fatalf("language %d has no data entry", l.id)
				}
				rva, size := le.Uint32(data[l.offset:]), le.Uint32(data[l.offset+4:])
				if rva < base || int(rva-base)+int(size) > len(data) {
					t.Fatalf("the data of %d/%d/%d (rva %#x, size %d) is outside the section", typ.id, id.id, l.id, rva, size)
				}
				res = append(res, Resource{Type: uint16(typ.id), ID: uint16(id.id), Lang: uint16(l.id), Data: data[rva-base : rva-base+size]})
			}
		}
	}
	return res
}

func sampleResources() []Resource {
	return []Resource{
		{Type: RTVersion, ID: 1, Lang: LangEnglishUS, Data: []byte("version data")},
		{Type: RTIcon, ID: 2, Lang: LangEnglishUS, Data: []byte("icon two, a little longer")},
		{Type: RTManifest, ID: 1, Lang: LangEnglishUS, Data: []byte("<assembly/>")},
		{Type: RTIcon, ID: 1, Lang: LangEnglishUS, Data: []byte("icon one")},
		{Type: RTGroupIcon, ID: 1, Lang: LangEnglishUS, Data: []byte("group")},
		{Type: RTIcon, ID: 2, Lang: 0, Data: []byte("icon two, neutral")},
	}
}

func TestBuildSysoReadsBackWithDebugPE(t *testing.T) {
	for _, arch := range Archs {
		t.Run(arch.Name, func(t *testing.T) {
			in := sampleResources()
			syso, err := BuildSyso(arch.Name, in)
			if err != nil {
				t.Fatal(err)
			}
			f, err := pe.NewFile(bytes.NewReader(syso))
			if err != nil {
				t.Fatalf("debug/pe does not read the object: %v", err)
			}
			if f.Machine != arch.Machine {
				t.Errorf("machine = %#x, want %#x", f.Machine, arch.Machine)
			}
			if len(f.Sections) != 1 || f.Sections[0].Name != ".rsrc" {
				t.Fatalf("sections = %v, want just .rsrc", f.Sections)
			}
			sec := f.Sections[0]
			if sec.Characteristics != 0x40000040 {
				t.Errorf("characteristics = %#x, want initialized data, readable", sec.Characteristics)
			}
			data, err := sec.Data()
			if err != nil {
				t.Fatal(err)
			}
			if len(data)%4 != 0 {
				t.Errorf("the section is %d bytes: not a multiple of 4", len(data))
			}

			// One relocation per data entry, of the type that stores an offset from the image base,
			// against the section's symbol (index 0), at the first field of the data entry.
			if len(sec.Relocs) != len(in) {
				t.Fatalf("%d relocations, want %d", len(sec.Relocs), len(in))
			}
			got := readResources(t, data, 0)
			for i, r := range sec.Relocs {
				if r.Type != arch.AddrNB || r.SymbolTableIndex != 0 {
					t.Errorf("relocation %d = type %d symbol %d, want type %d symbol 0", i, r.Type, r.SymbolTableIndex, arch.AddrNB)
				}
				// The field holds the offset of the data; the relocation makes it an address.
				field := binary.LittleEndian.Uint32(data[r.VirtualAddress:])
				if int(field) >= len(data) || !bytes.Contains(data[field:], got[i].Data) {
					t.Errorf("relocation %d at %#x: the field %#x does not point at the data of resource %d", i, r.VirtualAddress, field, i)
				}
			}

			// Symbols: the section symbol of the Go linker's loader (static, type 0, named .rsrc) and
			// its auxiliary record; nothing else.
			if len(f.COFFSymbols) != 2 {
				t.Fatalf("%d symbol records, want 2", len(f.COFFSymbols))
			}
			sym := f.COFFSymbols[0]
			name, err := sym.FullName(f.StringTable)
			if err != nil || name != ".rsrc" || sym.StorageClass != 3 || sym.Type != 0 || sym.SectionNumber != 1 || sym.Value != 0 || sym.NumberOfAuxSymbols != 1 {
				t.Errorf("symbol = %q %+v (%v), want the static section symbol .rsrc with one aux record", name, sym, err)
			}
			aux, err := f.COFFSymbolReadSectionDefAux(0)
			if err != nil || aux.Size != uint32(len(data)) || aux.NumRelocs != uint16(len(in)) {
				t.Errorf("aux = %+v (%v), want size %d and %d relocations", aux, err, len(data), len(in))
			}

			// The directory: sorted by type, then ID, then language, with the sizes of the data.
			want := slices.Clone(in)
			slices.SortFunc(want, func(a, b Resource) int {
				switch {
				case a.Type != b.Type:
					return int(a.Type) - int(b.Type)
				case a.ID != b.ID:
					return int(a.ID) - int(b.ID)
				}
				return int(a.Lang) - int(b.Lang)
			})
			if len(got) != len(want) {
				t.Fatalf("read %d resources, want %d", len(got), len(want))
			}
			for i := range want {
				if got[i].Type != want[i].Type || got[i].ID != want[i].ID || got[i].Lang != want[i].Lang || !bytes.Equal(got[i].Data, want[i].Data) {
					t.Errorf("resource %d = %d/%d/%d %q, want %d/%d/%d %q", i, got[i].Type, got[i].ID, got[i].Lang, got[i].Data, want[i].Type, want[i].ID, want[i].Lang, want[i].Data)
				}
			}
		})
	}
}

func TestBuildSysoIsReproducible(t *testing.T) {
	a, err := BuildSyso("amd64", sampleResources())
	if err != nil {
		t.Fatal(err)
	}
	reversed := slices.Clone(sampleResources())
	slices.Reverse(reversed)
	b, err := BuildSyso("amd64", reversed)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Error("the same resources in another order give another object")
	}
}

func TestBuildSysoErrors(t *testing.T) {
	tests := []struct {
		name  string
		arch  string
		res   []Resource
		route string
		meta  string
	}{
		{"an unknown architecture", "386", sampleResources(), "BuildSyso.WindowsResourceInvalid", "arch"},
		{"no resources", "amd64", nil, "BuildSyso.WindowsResourceInvalid", ""},
		{"an empty resource", "amd64", []Resource{{Type: RTIcon, ID: 1, Lang: 0x409}}, "BuildSyso.WindowsResourceInvalid", "id"},
		{"a duplicate", "amd64", []Resource{{Type: RTIcon, ID: 1, Lang: 0x409, Data: []byte("a")}, {Type: RTIcon, ID: 1, Lang: 0x409, Data: []byte("b")}}, "BuildSyso.WindowsResourceInvalid", "lang"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := BuildSyso(tt.arch, tt.res)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if !appErr.HasRoute(tt.route) {
				t.Errorf("route %s, want %s", appErr.Route(), tt.route)
			}
			if _, has := appErr.Metadata[tt.meta]; tt.meta != "" && !has {
				t.Errorf("metadata %v has no %q", appErr.Metadata, tt.meta)
			}
		})
	}
}

func TestResourcesOfAnApp(t *testing.T) {
	png, err := DefaultIcon()
	if err != nil {
		t.Fatal(err)
	}
	icon, err := NewIcon(png)
	if err != nil {
		t.Fatal(err)
	}
	res := Resources([]byte("<m/>"), icon, VersionInfo{Version: Version{1, 2, 3, 4}, Name: "N", AppID: "a.b", FileName: "n.exe"})
	var icons, groups, versions, manifests int
	for _, r := range res {
		switch r.Type {
		case RTIcon:
			icons++
		case RTGroupIcon:
			groups++
			if r.ID != 1 {
				t.Errorf("the group icon has ID %d, want 1: the window class loads resource 1", r.ID)
			}
		case RTVersion:
			versions++
		case RTManifest:
			manifests++
			if r.ID != 1 {
				t.Errorf("the manifest has ID %d, want 1 (CREATEPROCESS_MANIFEST_RESOURCE_ID)", r.ID)
			}
		}
	}
	if icons != len(IconSizes) || groups != 1 || versions != 1 || manifests != 1 {
		t.Errorf("icons %d groups %d versions %d manifests %d", icons, groups, versions, manifests)
	}
}
