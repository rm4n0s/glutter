//go:build linux && !android

package pulse

// The positions of the channels of PulseAudio (pa_channel_position_t, pulse/channelmap.h) that the maps below use.
const (
	posMono        = 0
	posFrontLeft   = 1
	posFrontRight  = 2
	posFrontCenter = 3
	posRearCenter  = 4
	posRearLeft    = 5
	posRearRight   = 6
	posLFE         = 7
	posSideLeft    = 10
	posSideRight   = 11
)

// channelMaps are the maps for 1 to 8 channels, in the order of SDL's layouts (SPEC §28.3; SDL_pulseaudio.c, translated to Go): the
// channels of the frames that the handler gets are in this order, and the server places them on the speakers by the positions.
var channelMaps = [maxAudioChans + 1][]int32{
	1: {posMono},
	2: {posFrontLeft, posFrontRight},
	3: {posFrontLeft, posFrontRight, posLFE},
	4: {posFrontLeft, posFrontRight, posRearLeft, posRearRight},
	5: {posFrontLeft, posFrontRight, posLFE, posRearLeft, posRearRight},
	6: {posFrontLeft, posFrontRight, posFrontCenter, posLFE, posRearLeft, posRearRight},
	7: {posFrontLeft, posFrontRight, posFrontCenter, posLFE, posRearCenter, posSideLeft, posSideRight},
	8: {posFrontLeft, posFrontRight, posFrontCenter, posLFE, posRearLeft, posRearRight, posSideLeft, posSideRight},
}

// fillChannelMap sets m to the map of the number of channels (1 to 8).
func fillChannelMap(m *channelMap, channels int) {
	*m = channelMap{channels: uint8(channels)}
	copy(m.pos[:], channelMaps[channels])
}
