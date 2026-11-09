package mactool

import (
	"bytes"
	"debug/macho"
	"encoding/binary"

	"github.com/rm4n0s/errors"
)

// FatSlice is a thin Mach-O file that goes into a universal one.
//
// Flutter: no counterpart
type FatSlice struct {
	// Name says which file it is, for the errors ("arm64").
	Name string
	// Data is the thin Mach-O file.
	Data []byte
}

const (
	fatMagic = 0xCAFEBABE
	// fatArchSize is the size of a fat_arch record.
	fatArchSize = 20
)

// Fat writes a universal Mach-O file with the thin files as its slices, as `lipo -create` does: the header of big-endian numbers
// (the magic, the number of slices, and for each slice its CPU type and subtype, offset, size and alignment), then the slices at
// the offsets that their alignment gives (2^14 for arm64 and 2^12 for the rest, which are what lipo uses). The CPU type of a slice is
// read from its own header, so a file that is not a thin Mach-O, or two slices of one architecture, is MacOSMachOInvalid.
//
// Flutter: no counterpart
func Fat(slices []FatSlice) ([]byte, error) {
	if len(slices) == 0 {
		return nil, errors.New(tagMacOSMachOInvalid, "a universal file needs at least one slice")
	}
	type placed struct {
		cpu, sub uint32
		align    uint32
		off      uint64
	}
	infos := make([]placed, len(slices))
	seen := map[macho.Cpu]string{}
	for i, s := range slices {
		f, err := macho.NewFile(bytes.NewReader(s.Data))
		if err != nil {
			return nil, errors.NewErr(tagMacOSMachOInvalid, err).SetMetadata("slice", s.Name)
		}
		cpu := f.Cpu
		if other, dup := seen[cpu]; dup {
			return nil, errors.New(tagMacOSMachOInvalid, "two slices are for the same architecture", "slice", s.Name, "other", other)
		}
		seen[cpu] = s.Name
		infos[i] = placed{cpu: uint32(cpu), sub: f.SubCpu, align: 12}
		if cpu == macho.CpuArm64 {
			infos[i].align = 14
		}
	}
	// The fat header is a Mach-O header of big-endian numbers, so a slice that is more than 4 GB cannot be in it.
	off := uint64(8 + fatArchSize*len(slices))
	for i, s := range slices {
		a := uint64(1) << infos[i].align
		off = (off + a - 1) / a * a
		infos[i].off = off
		off += uint64(len(s.Data))
		if off > 1<<32-1 {
			return nil, errors.New(tagMacOSMachOInvalid, "the universal file would be larger than 4 GB", "slice", s.Name)
		}
	}
	out := make([]byte, off)
	binary.BigEndian.PutUint32(out[0:], fatMagic)
	binary.BigEndian.PutUint32(out[4:], uint32(len(slices)))
	for i, s := range slices {
		h := out[8+fatArchSize*i:]
		binary.BigEndian.PutUint32(h[0:], infos[i].cpu)
		binary.BigEndian.PutUint32(h[4:], infos[i].sub)
		binary.BigEndian.PutUint32(h[8:], uint32(infos[i].off))
		binary.BigEndian.PutUint32(h[12:], uint32(len(s.Data)))
		binary.BigEndian.PutUint32(h[16:], infos[i].align)
		copy(out[infos[i].off:], s.Data)
	}
	return out, nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagMacOSMachOInvalid = TagMacOSMachOInvalid

	// TagMacOSMachOInvalid is the tag of the error for a file that is not a thin Mach-O, or slices that cannot make a universal
	// file.
	//
	// Flutter: no counterpart
	TagMacOSMachOInvalid = "MacOSMachOInvalid"
)
