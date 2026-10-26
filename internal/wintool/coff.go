package wintool

import (
	"encoding/binary"
)

// Arch is a processor architecture an .exe is built for.
//
// Flutter: no counterpart
type Arch struct {
	// Name is the name the command line and the manifest use: "amd64" or "arm64".
	Name string
	// GOARCH is the value of GOARCH for the go command.
	GOARCH string
	// Machine is the IMAGE_FILE_MACHINE value of the COFF object.
	Machine uint16
	// AddrNB is the type of the relocation that stores an address as an offset from the image base
	// (IMAGE_REL_AMD64_ADDR32NB, IMAGE_REL_ARM64_ADDR32NB).
	AddrNB uint16
}

// Archs are the architectures of Windows that Glutter builds for (SPEC §25.1), amd64 first.
//
// Flutter: no counterpart
var Archs = []Arch{
	{Name: "amd64", GOARCH: "amd64", Machine: 0x8664, AddrNB: 3},
	{Name: "arm64", GOARCH: "arm64", Machine: 0xAA64, AddrNB: 2},
}

// FindArch returns the architecture with the name.
//
// Flutter: no counterpart
func FindArch(name string) (Arch, bool) {
	for _, a := range Archs {
		if a.Name == name {
			return a, true
		}
	}
	return Arch{}, false
}

// coffReloc is a relocation of the section against the section's own symbol: the 32-bit field at
// Offset holds an offset into the section, and the linker adds the section's address (minus the
// image base, for the ADDR32NB types).
type coffReloc struct {
	Offset uint32
	Type   uint16
}

// writeCOFF writes a COFF object file (what a .syso is) with one section, named .rsrc, that holds
// data and the relocations, and the symbols that the Go linker's PE loader needs: the section's
// symbol, with its auxiliary record. The sizes and offsets follow the PE/COFF specification:
//
//	file header (20) | section header (40) | raw data | relocations (10 each) | symbols (2 × 18) | string table (4)
//
// The time stamp is 0, so the same input gives the same bytes.
func writeCOFF(machine uint16, data []byte, relocs []coffReloc) []byte {
	const (
		headerSize  = 20
		sectionSize = 40
		relocSize   = 10
		symbolSize  = 18
	)
	rawOff := headerSize + sectionSize
	relocOff := rawOff + len(data)
	symOff := relocOff + relocSize*len(relocs)
	le := binary.LittleEndian

	b := make([]byte, 0, symOff+2*symbolSize+4)
	// IMAGE_FILE_HEADER
	b = le.AppendUint16(b, machine)
	b = le.AppendUint16(b, 1) // NumberOfSections
	b = le.AppendUint32(b, 0) // TimeDateStamp
	b = le.AppendUint32(b, uint32(symOff))
	b = le.AppendUint32(b, 2) // NumberOfSymbols: the section symbol and its auxiliary record
	b = le.AppendUint16(b, 0) // SizeOfOptionalHeader
	b = le.AppendUint16(b, 0) // Characteristics
	// IMAGE_SECTION_HEADER
	b = append(b, ".rsrc\x00\x00\x00"...)
	b = le.AppendUint32(b, 0) // VirtualSize
	b = le.AppendUint32(b, 0) // VirtualAddress
	b = le.AppendUint32(b, uint32(len(data)))
	b = le.AppendUint32(b, uint32(rawOff))
	b = le.AppendUint32(b, uint32(relocOff))
	b = le.AppendUint32(b, 0) // PointerToLinenumbers
	b = le.AppendUint16(b, uint16(len(relocs)))
	b = le.AppendUint16(b, 0)          // NumberOfLinenumbers
	b = le.AppendUint32(b, 0x40000040) // IMAGE_SCN_CNT_INITIALIZED_DATA | IMAGE_SCN_MEM_READ
	// the raw data
	b = append(b, data...)
	// the relocations, all against symbol 0
	for _, r := range relocs {
		b = le.AppendUint32(b, r.Offset)
		b = le.AppendUint32(b, 0)
		b = le.AppendUint16(b, r.Type)
	}
	// the symbol of the section: ".rsrc", value 0, section 1, type 0, class STATIC, one auxiliary record
	b = append(b, ".rsrc\x00\x00\x00"...)
	b = le.AppendUint32(b, 0)
	b = le.AppendUint16(b, 1)
	b = le.AppendUint16(b, 0)
	b = append(b, 3, 1)
	// its auxiliary record: length, relocations, line numbers, checksum, number, selection
	b = le.AppendUint32(b, uint32(len(data)))
	b = le.AppendUint16(b, uint16(len(relocs)))
	b = le.AppendUint16(b, 0)
	b = le.AppendUint32(b, 0)
	b = le.AppendUint16(b, 0)
	b = append(b, 0, 0, 0, 0) // selection and padding
	// the string table: just its own size
	return le.AppendUint32(b, 4)
}
