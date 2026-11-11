package coreaudio

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
)

// The limits of a format (SPEC §28.3).
const (
	minRate     = 1000
	maxRate     = 768000
	maxChannels = 8
)

// The values of AudioStreamBasicDescription that this backend uses (CoreAudioBaseTypes.h).
const (
	formatLinearPCM         = 0x6C70636D // 'lpcm'
	formatFlagIsFloat       = 1 << 0
	formatFlagIsBigEndian   = 1 << 1
	formatFlagIsPacked      = 1 << 3
	formatFlagNonInterleave = 1 << 5
)

// StreamDescription is the Go mirror of AudioStreamBasicDescription.
//
// Flutter: no counterpart
type StreamDescription struct {
	SampleRate       float64 `c:"mSampleRate"`
	FormatID         uint32  `c:"mFormatID"`
	FormatFlags      uint32  `c:"mFormatFlags"`
	BytesPerPacket   uint32  `c:"mBytesPerPacket"`
	FramesPerPacket  uint32  `c:"mFramesPerPacket"`
	BytesPerFrame    uint32  `c:"mBytesPerFrame"`
	ChannelsPerFrame uint32  `c:"mChannelsPerFrame"`
	BitsPerChannel   uint32  `c:"mBitsPerChannel"`
	Reserved         uint32  `c:"mReserved"`
}

// NewStreamDescription makes the description of the frames that the toolkit passes: interleaved float32 in the byte order of the
// machine, one frame to a packet.
//
// Flutter: no counterpart
func NewStreamDescription(f platform.AudioFormat) StreamDescription {
	return StreamDescription{
		SampleRate:       float64(f.Rate),
		FormatID:         formatLinearPCM,
		FormatFlags:      formatFlagIsFloat | formatFlagIsPacked,
		BytesPerPacket:   uint32(4 * f.Channels),
		FramesPerPacket:  1,
		BytesPerFrame:    uint32(4 * f.Channels),
		ChannelsPerFrame: uint32(f.Channels),
		BitsPerChannel:   32,
	}
}

// Format returns the format of a description, and false when it is not the one that NewStreamDescription makes: linear PCM, float,
// 32 bits, packed, interleaved, in the byte order of the machine (little endian on every Mac that Go runs on).
//
// Flutter: no counterpart
func (s StreamDescription) Format() (platform.AudioFormat, bool) {
	const want = formatFlagIsFloat | formatFlagIsPacked
	if s.FormatID != formatLinearPCM || s.FormatFlags&(want|formatFlagIsBigEndian|formatFlagNonInterleave) != want || s.BitsPerChannel != 32 ||
		s.ChannelsPerFrame == 0 || s.ChannelsPerFrame > maxChannels || s.BytesPerFrame != 4*s.ChannelsPerFrame || s.SampleRate < 1 {
		return platform.AudioFormat{}, false
	}
	return platform.AudioFormat{Rate: int(s.SampleRate + 0.5), Channels: int(s.ChannelsPerFrame)}, true
}

// CheckFormat says whether the format that an app asks for (a zero field leaves the choice to the backend) is one that the backend
// can give: 1 to 8 channels, and a rate of 1000 to 768000.
//
// Flutter: no counterpart
func CheckFormat(f platform.AudioFormat) error {
	if r, c := f.Rate, f.Channels; r < 0 || c < 0 || c > maxChannels || (r != 0 && (r < minRate || r > maxRate)) {
		return errors.New(platform.TagAudioFormatUnsupported, "Core Audio devices take 1 to 8 channels and a rate of 1000 to 768000",
			"rate", r, "channels", c)
	}
	return nil
}

// bufferFrames is the size of the buffers of a queue in frames: what the app asked for, or about ten milliseconds, so that the
// three buffers of a queue are about 30 ms, the target of SPEC §28.6.
func bufferFrames(asked, rate int) int {
	if asked > 0 {
		return asked
	}
	return max(rate/100, 64)
}

// fourcc is the number of a four-character code, as Core Audio writes its selectors.
func fourcc(s string) uint32 {
	return uint32(s[0])<<24 | uint32(s[1])<<16 | uint32(s[2])<<8 | uint32(s[3])
}

// PropertyAddress is the Go mirror of AudioObjectPropertyAddress.
//
// Flutter: no counterpart
type PropertyAddress struct {
	Selector uint32 `c:"mSelector"`
	Scope    uint32 `c:"mScope"`
	Element  uint32 `c:"mElement"`
}

// QueueBuffer is the Go mirror of AudioQueueBuffer, which Core Audio hands to the callbacks of a queue.
//
// Flutter: no counterpart
type QueueBuffer struct {
	Capacity    uint32  `c:"mAudioDataBytesCapacity"`
	_           [4]byte `c:"-"`
	Data        uintptr `c:"mAudioData"`
	ByteSize    uint32  `c:"mAudioDataByteSize"`
	_           [4]byte `c:"-"`
	UserData    uintptr `c:"mUserData"`
	PacketCap   uint32  `c:"mPacketDescriptionCapacity"`
	_           [4]byte `c:"-"`
	PacketDescs uintptr `c:"mPacketDescriptions"`
	PacketCount uint32  `c:"mPacketDescriptionCount"`
	_           [4]byte `c:"-"`
}

// ChannelLayout is AudioChannelLayout with room for the one description that C's declaration has, none of which is used: a layout
// that is a tag.
//
// Flutter: no counterpart
type ChannelLayout struct {
	Tag          uint32 `c:"mChannelLayoutTag"`
	Bitmap       uint32 `c:"mChannelBitmap"`
	Descriptions uint32 `c:"mNumberChannelDescriptions"`
	// Description is the one AudioChannelDescription of C's declaration (a label, flags and three coordinates).
	Description [20]byte `c:"mChannelDescriptions"`
}

// layoutTag is the channel layout tag of a count of channels: mono, stereo, and for more channels "discrete in order"
// (kAudioChannelLayoutTag_DiscreteInOrder), so that channel n goes out of output n.
//
// Deviation: SDL names the speakers of its channel orders (5.1, 7.1); here the channels of a stream with more than two are
// the outputs of the device in order, which is what the order of SDL's layouts is for the first six or eight outputs.
func layoutTag(channels int) uint32 {
	switch channels {
	case 1:
		return 100<<16 | 1 // kAudioChannelLayoutTag_Mono
	case 2:
		return 101<<16 | 2 // kAudioChannelLayoutTag_Stereo
	}
	return 147<<16 | uint32(channels) // kAudioChannelLayoutTag_DiscreteInOrder
}
