package win32

// The Go mirrors of the structures of the audio APIs (SPEC §28.3). The layout test of this package checks them against the
// mingw-w64 headers.

// WaveFormatExtensible is the Go mirror of WAVEFORMATEXTENSIBLE, written flat: its first eighteen bytes are the WAVEFORMATEX that
// the header packs to one byte, and the structure is packed too, so the fields after CbSize sit at 18, 20 and 24 (their natural
// places, which is why a flat struct with Go's alignment has the layout of the packed one; the layout test checks it). A format
// with WaveFormatTagPCM or WaveFormatTagIEEEFloat as its tag and 0 as CbSize is a plain WAVEFORMATEX and ends after CbSize.
//
// Flutter: no counterpart
type WaveFormatExtensible struct {
	// FormatTag is wFormatTag: WaveFormatTagExtensible here.
	FormatTag uint16 `c:"Format.wFormatTag"`
	// Channels is nChannels.
	Channels uint16 `c:"Format.nChannels"`
	// SamplesPerSec is nSamplesPerSec: the frames a second.
	SamplesPerSec uint32 `c:"Format.nSamplesPerSec"`
	// AvgBytesPerSec is nAvgBytesPerSec.
	AvgBytesPerSec uint32 `c:"Format.nAvgBytesPerSec"`
	// BlockAlign is nBlockAlign: the bytes of a frame.
	BlockAlign uint16 `c:"Format.nBlockAlign"`
	// BitsPerSample is wBitsPerSample.
	BitsPerSample uint16 `c:"Format.wBitsPerSample"`
	// CbSize is cbSize: 22 for the extensible format.
	CbSize uint16 `c:"Format.cbSize"`
	// ValidBitsPerSample is Samples.wValidBitsPerSample.
	ValidBitsPerSample uint16 `c:"Samples.wValidBitsPerSample"`
	// ChannelMask is dwChannelMask: the speakers of the channels, in the order of the bits.
	ChannelMask uint32 `c:"dwChannelMask"`
	// SubFormat is SubFormat: the kind of the samples (KSDataFormatSubTypeIEEEFloat).
	SubFormat GUID `c:"SubFormat"`
}

// PropertyKey is the Go mirror of PROPERTYKEY (a property of a property store: the GUID of its set and its number).
//
// Flutter: no counterpart
type PropertyKey struct {
	// FmtID is fmtid.
	FmtID GUID `c:"fmtid"`
	// PID is pid.
	PID uint32 `c:"pid"`
}

// PropVariant is the Go mirror of PROPVARIANT: a type and 16 bytes of union, of which the first pointer-sized word holds what the
// types that the audio code reads put there (the pointer of a VT_LPWSTR string).
//
// Flutter: no counterpart
type PropVariant struct {
	// VT is vt: the type (VTLPWStr).
	VT uint16 `c:"vt"`
	_  [3]uint16
	// Val is the first word of the union (pwszVal).
	Val uintptr `c:"pwszVal"`
	_   uintptr
}
