//go:build darwin

package coreaudio

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/platform/darwin/layouttest"
)

// TestMirrorsMatchTheSDK checks the Go mirrors and the constants of this package against the headers of the SDK (skipped, with the
// install command, when the machine is not a Mac or has no Command Line Tools).
func TestMirrorsMatchTheSDK(t *testing.T) {
	layouttest.Check(t, layouttest.Spec{
		Headers: []string{"CoreAudio/CoreAudio.h", "AudioToolbox/AudioToolbox.h"},
		Structs: []layouttest.Struct{
			layouttest.StructOf("AudioStreamBasicDescription", StreamDescription{}),
			layouttest.StructOf("AudioObjectPropertyAddress", PropertyAddress{}),
			layouttest.StructOf("AudioQueueBuffer", QueueBuffer{}),
			layouttest.StructOf("AudioChannelLayout", ChannelLayout{}),
		},
		Consts: []layouttest.Const{
			{Name: "kAudioObjectSystemObject", Value: systemObject},
			{Name: "kAudioHardwarePropertyDevices", Value: int64(selDevices)},
			{Name: "kAudioHardwarePropertyDefaultOutputDevice", Value: int64(selDefaultOutput)},
			{Name: "kAudioHardwarePropertyDefaultInputDevice", Value: int64(selDefaultInput)},
			{Name: "kAudioObjectPropertyName", Value: int64(selName)},
			{Name: "kAudioDevicePropertyDeviceUID", Value: int64(selUID)},
			{Name: "kAudioDevicePropertyStreamConfiguration", Value: int64(selStreamConfig)},
			{Name: "kAudioDevicePropertyNominalSampleRate", Value: int64(selNominalRate)},
			{Name: "kAudioDevicePropertyDeviceIsAlive", Value: int64(selIsAlive)},
			{Name: "kAudioObjectPropertyScopeGlobal", Value: int64(scopeGlobal)},
			{Name: "kAudioObjectPropertyScopeInput", Value: int64(scopeInput)},
			{Name: "kAudioObjectPropertyScopeOutput", Value: int64(scopeOutput)},
			{Name: "kAudioQueueProperty_CurrentDevice", Value: int64(queueCurrentDev)},
			{Name: "kAudioQueueProperty_ChannelLayout", Value: int64(queueChannelLay)},
			{Name: "kAudioFormatLinearPCM", Value: formatLinearPCM},
			{Name: "kAudioFormatFlagIsFloat", Value: formatFlagIsFloat},
			{Name: "kAudioFormatFlagIsBigEndian", Value: formatFlagIsBigEndian},
			{Name: "kAudioFormatFlagIsPacked", Value: formatFlagIsPacked},
			{Name: "kAudioFormatFlagIsNonInterleaved", Value: formatFlagNonInterleave},
			{Name: "kAudioFormatUnsupportedDataFormatError", Value: statusFormatUnsupported},
			{Name: "kAudioQueueErr_InvalidParameter", Value: statusQueueInvalidParameter},
			{Name: "paramErr", Value: statusParamErr},
			{Name: "kAudioChannelLayoutTag_Mono", Value: int64(layoutTag(1))},
			{Name: "kAudioChannelLayoutTag_Stereo", Value: int64(layoutTag(2))},
			{Name: "kAudioChannelLayoutTag_DiscreteInOrder", Value: int64(layoutTag(5)) &^ 0xFFFF},
		},
	})
}
