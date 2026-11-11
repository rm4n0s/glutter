//go:build darwin

package coreaudio

// The property selectors, scopes and queue properties that the backend uses (AudioHardware.h, AudioQueue.h).
var (
	selDevices       = fourcc("dev#")
	selDefaultOutput = fourcc("dOut")
	selDefaultInput  = fourcc("dIn ")
	selName          = fourcc("lnam")
	selUID           = fourcc("uid ")
	selStreamConfig  = fourcc("slay")
	selNominalRate   = fourcc("nsrt")
	selIsAlive       = fourcc("livn")
	scopeGlobal      = fourcc("glob")
	scopeInput       = fourcc("inpt")
	scopeOutput      = fourcc("outp")
	queueCurrentDev  = fourcc("aqcd")
	queueChannelLay  = fourcc("aqcl")
)

// systemObject is the AudioObjectID of the system object (kAudioObjectSystemObject).
const systemObject = 1
