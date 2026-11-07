package win32

// The constants of the audio APIs of Windows (WASAPI and the device enumerator of MMDevice, SPEC §28.3): the interface IDs, the
// positions of the methods in the vtables, the flags and the results. The layout test checks the indexes and the numbers against
// the mingw-w64 headers; the IDs are copied from them (the identity of an interface is not a constant that the compiler can
// assert, so the live tests under Wine are what proves them: a wrong ID makes QueryInterface fail).

// The class and interface IDs of the device enumerator and of the audio client.
var (
	// CLSIDMMDeviceEnumerator is the class ID of the device enumerator.
	//
	// Flutter: no counterpart
	CLSIDMMDeviceEnumerator = MustParseGUID("{BCDE0395-E52F-467C-8E3D-C4579291692E}")
	// IIDMMDeviceEnumerator is the interface ID of IMMDeviceEnumerator.
	//
	// Flutter: no counterpart
	IIDMMDeviceEnumerator = MustParseGUID("{A95664D2-9614-4F35-A746-DE8DB63617E6}")
	// IIDMMNotificationClient is the interface ID of IMMNotificationClient, which Go implements.
	//
	// Flutter: no counterpart
	IIDMMNotificationClient = MustParseGUID("{7991EEC9-7E89-4D85-8390-6C703CEC60C0}")
	// IIDAudioClient is the interface ID of IAudioClient.
	//
	// Flutter: no counterpart
	IIDAudioClient = MustParseGUID("{1CB9AD4C-DBFA-4C32-B178-C2F568A703B2}")
	// IIDAudioClient3 is the interface ID of IAudioClient3 (Windows 10).
	//
	// Flutter: no counterpart
	IIDAudioClient3 = MustParseGUID("{7ED4EE07-8E67-4CD4-8C1A-2B7A5987AD42}")
	// IIDAudioRenderClient is the interface ID of IAudioRenderClient.
	//
	// Flutter: no counterpart
	IIDAudioRenderClient = MustParseGUID("{F294ACFC-3146-4483-A7BF-ADDCA7C260E2}")
	// IIDAudioCaptureClient is the interface ID of IAudioCaptureClient.
	//
	// Flutter: no counterpart
	IIDAudioCaptureClient = MustParseGUID("{C8ADBD64-E71E-48A0-A4DE-185C395CD317}")
	// KSDataFormatSubTypeIEEEFloat is KSDATAFORMAT_SUBTYPE_IEEE_FLOAT, the sub-format of the extensible wave format that says
	// 32-bit float samples.
	//
	// Flutter: no counterpart
	KSDataFormatSubTypeIEEEFloat = MustParseGUID("{00000003-0000-0010-8000-00AA00389B71}")
	// PKeyDeviceFriendlyName is PKEY_Device_FriendlyName, the name of a device that a user sees ("Speakers (Realtek Audio)").
	//
	// Flutter: no counterpart
	PKeyDeviceFriendlyName = PropertyKey{FmtID: MustParseGUID("{A45C254E-DF1C-4EFD-8020-67D146A850E0}"), PID: 14}
)

// The positions of the methods in the vtables of the audio interfaces. The layout test checks each against the headers.
const (
	// IMMDeviceEnumeratorEnumAudioEndpoints is the index of IMMDeviceEnumerator::EnumAudioEndpoints.
	//
	// Flutter: no counterpart
	IMMDeviceEnumeratorEnumAudioEndpoints = 3
	// IMMDeviceEnumeratorGetDefaultAudioEndpoint is the index of IMMDeviceEnumerator::GetDefaultAudioEndpoint.
	//
	// Flutter: no counterpart
	IMMDeviceEnumeratorGetDefaultAudioEndpoint = 4
	// IMMDeviceEnumeratorGetDevice is the index of IMMDeviceEnumerator::GetDevice.
	//
	// Flutter: no counterpart
	IMMDeviceEnumeratorGetDevice = 5
	// IMMDeviceEnumeratorRegisterEndpointNotificationCallback is the index of
	// IMMDeviceEnumerator::RegisterEndpointNotificationCallback.
	//
	// Flutter: no counterpart
	IMMDeviceEnumeratorRegisterEndpointNotificationCallback = 6
	// IMMDeviceEnumeratorUnregisterEndpointNotificationCallback is the index of
	// IMMDeviceEnumerator::UnregisterEndpointNotificationCallback.
	//
	// Flutter: no counterpart
	IMMDeviceEnumeratorUnregisterEndpointNotificationCallback = 7

	// IMMDeviceCollectionGetCount is the index of IMMDeviceCollection::GetCount.
	//
	// Flutter: no counterpart
	IMMDeviceCollectionGetCount = 3
	// IMMDeviceCollectionItem is the index of IMMDeviceCollection::Item.
	//
	// Flutter: no counterpart
	IMMDeviceCollectionItem = 4

	// IMMDeviceActivate is the index of IMMDevice::Activate.
	//
	// Flutter: no counterpart
	IMMDeviceActivate = 3
	// IMMDeviceOpenPropertyStore is the index of IMMDevice::OpenPropertyStore.
	//
	// Flutter: no counterpart
	IMMDeviceOpenPropertyStore = 4
	// IMMDeviceGetID is the index of IMMDevice::GetId.
	//
	// Flutter: no counterpart
	IMMDeviceGetID = 5
	// IMMDeviceGetState is the index of IMMDevice::GetState.
	//
	// Flutter: no counterpart
	IMMDeviceGetState = 6

	// IPropertyStoreGetValue is the index of IPropertyStore::GetValue.
	//
	// Flutter: no counterpart
	IPropertyStoreGetValue = 5

	// IAudioClientInitialize is the index of IAudioClient::Initialize.
	//
	// Flutter: no counterpart
	IAudioClientInitialize = 3
	// IAudioClientGetBufferSize is the index of IAudioClient::GetBufferSize.
	//
	// Flutter: no counterpart
	IAudioClientGetBufferSize = 4
	// IAudioClientGetStreamLatency is the index of IAudioClient::GetStreamLatency.
	//
	// Flutter: no counterpart
	IAudioClientGetStreamLatency = 5
	// IAudioClientGetCurrentPadding is the index of IAudioClient::GetCurrentPadding.
	//
	// Flutter: no counterpart
	IAudioClientGetCurrentPadding = 6
	// IAudioClientIsFormatSupported is the index of IAudioClient::IsFormatSupported.
	//
	// Flutter: no counterpart
	IAudioClientIsFormatSupported = 7
	// IAudioClientGetMixFormat is the index of IAudioClient::GetMixFormat.
	//
	// Flutter: no counterpart
	IAudioClientGetMixFormat = 8
	// IAudioClientGetDevicePeriod is the index of IAudioClient::GetDevicePeriod.
	//
	// Flutter: no counterpart
	IAudioClientGetDevicePeriod = 9
	// IAudioClientStart is the index of IAudioClient::Start.
	//
	// Flutter: no counterpart
	IAudioClientStart = 10
	// IAudioClientStop is the index of IAudioClient::Stop.
	//
	// Flutter: no counterpart
	IAudioClientStop = 11
	// IAudioClientReset is the index of IAudioClient::Reset.
	//
	// Flutter: no counterpart
	IAudioClientReset = 12
	// IAudioClientSetEventHandle is the index of IAudioClient::SetEventHandle.
	//
	// Flutter: no counterpart
	IAudioClientSetEventHandle = 13
	// IAudioClientGetService is the index of IAudioClient::GetService.
	//
	// Flutter: no counterpart
	IAudioClientGetService = 14

	// IAudioClient3GetSharedModeEnginePeriod is the index of IAudioClient3::GetSharedModeEnginePeriod.
	//
	// Flutter: no counterpart
	IAudioClient3GetSharedModeEnginePeriod = 18
	// IAudioClient3InitializeSharedAudioStream is the index of IAudioClient3::InitializeSharedAudioStream.
	//
	// Flutter: no counterpart
	IAudioClient3InitializeSharedAudioStream = 20

	// IAudioRenderClientGetBuffer is the index of IAudioRenderClient::GetBuffer.
	//
	// Flutter: no counterpart
	IAudioRenderClientGetBuffer = 3
	// IAudioRenderClientReleaseBuffer is the index of IAudioRenderClient::ReleaseBuffer.
	//
	// Flutter: no counterpart
	IAudioRenderClientReleaseBuffer = 4

	// IAudioCaptureClientGetBuffer is the index of IAudioCaptureClient::GetBuffer.
	//
	// Flutter: no counterpart
	IAudioCaptureClientGetBuffer = 3
	// IAudioCaptureClientReleaseBuffer is the index of IAudioCaptureClient::ReleaseBuffer.
	//
	// Flutter: no counterpart
	IAudioCaptureClientReleaseBuffer = 4
	// IAudioCaptureClientGetNextPacketSize is the index of IAudioCaptureClient::GetNextPacketSize.
	//
	// Flutter: no counterpart
	IAudioCaptureClientGetNextPacketSize = 5

	// IMMNotificationClientOnDeviceStateChanged is the index of IMMNotificationClient::OnDeviceStateChanged.
	//
	// Flutter: no counterpart
	IMMNotificationClientOnDeviceStateChanged = 3
	// IMMNotificationClientOnDeviceAdded is the index of IMMNotificationClient::OnDeviceAdded.
	//
	// Flutter: no counterpart
	IMMNotificationClientOnDeviceAdded = 4
	// IMMNotificationClientOnDeviceRemoved is the index of IMMNotificationClient::OnDeviceRemoved.
	//
	// Flutter: no counterpart
	IMMNotificationClientOnDeviceRemoved = 5
	// IMMNotificationClientOnDefaultDeviceChanged is the index of IMMNotificationClient::OnDefaultDeviceChanged.
	//
	// Flutter: no counterpart
	IMMNotificationClientOnDefaultDeviceChanged = 6
	// IMMNotificationClientOnPropertyValueChanged is the index of IMMNotificationClient::OnPropertyValueChanged.
	//
	// Flutter: no counterpart
	IMMNotificationClientOnPropertyValueChanged = 7
)

// The flags, enumerators and formats of the audio APIs.
const (
	// AudclntShareModeShared is AUDCLNT_SHAREMODE_SHARED: the stream goes through the engine that mixes the streams of all programs.
	//
	// Flutter: no counterpart
	AudclntShareModeShared = 0
	// AudclntStreamFlagsEventCallback is AUDCLNT_STREAMFLAGS_EVENTCALLBACK: the engine sets an event when it wants or has data.
	//
	// Flutter: no counterpart
	AudclntStreamFlagsEventCallback = 0x00040000
	// AudclntStreamFlagsAutoConvertPCM is AUDCLNT_STREAMFLAGS_AUTOCONVERTPCM: the engine converts the format of the stream to its own.
	//
	// Flutter: no counterpart
	AudclntStreamFlagsAutoConvertPCM = 0x80000000
	// AudclntStreamFlagsSrcDefaultQuality is AUDCLNT_STREAMFLAGS_SRC_DEFAULT_QUALITY: the sample rate converter of the default quality.
	//
	// Flutter: no counterpart
	AudclntStreamFlagsSrcDefaultQuality = 0x08000000
	// AudclntBufferFlagsDataDiscontinuity is AUDCLNT_BUFFERFLAGS_DATA_DISCONTINUITY: a packet of a recording follows a gap.
	//
	// Flutter: no counterpart
	AudclntBufferFlagsDataDiscontinuity = 0x1
	// AudclntBufferFlagsSilent is AUDCLNT_BUFFERFLAGS_SILENT: the packet of a recording is silence whatever its bytes say.
	//
	// Flutter: no counterpart
	AudclntBufferFlagsSilent = 0x2
	// DeviceStateActive is DEVICE_STATE_ACTIVE: the endpoint is present and works.
	//
	// Flutter: no counterpart
	DeviceStateActive = 0x1
	// ERender is eRender (EDataFlow): an endpoint that plays.
	//
	// Flutter: no counterpart
	ERender = 0
	// ECapture is eCapture (EDataFlow): an endpoint that records.
	//
	// Flutter: no counterpart
	ECapture = 1
	// EConsole is eConsole (ERole): the default device of games, system sounds and most programs.
	//
	// Flutter: no counterpart
	EConsole = 0
	// WaveFormatTagPCM is WAVE_FORMAT_PCM.
	//
	// Flutter: no counterpart
	WaveFormatTagPCM = 1
	// WaveFormatTagIEEEFloat is WAVE_FORMAT_IEEE_FLOAT.
	//
	// Flutter: no counterpart
	WaveFormatTagIEEEFloat = 3
	// WaveFormatTagExtensible is WAVE_FORMAT_EXTENSIBLE: the format is a WAVEFORMATEXTENSIBLE.
	//
	// Flutter: no counterpart
	WaveFormatTagExtensible = 0xFFFE
	// STGMRead is STGM_READ.
	//
	// Flutter: no counterpart
	STGMRead = 0x0
	// VTLPWStr is VT_LPWSTR: a PROPVARIANT that holds a wide string.
	//
	// Flutter: no counterpart
	VTLPWStr = 31
	// COInitMultithreaded is COINIT_MULTITHREADED: the thread joins the multithreaded apartment of the process.
	//
	// Flutter: no counterpart
	COInitMultithreaded = 0x0
	// ClsctxAll is CLSCTX_ALL.
	//
	// Flutter: no counterpart
	ClsctxAll = 0x17
)

// The speaker positions of a channel mask (SPEAKER_*): the channels of a frame come in the order of the bits that are set.
const (
	// SpeakerFrontLeft is SPEAKER_FRONT_LEFT.
	//
	// Flutter: no counterpart
	SpeakerFrontLeft = 0x1
	// SpeakerFrontRight is SPEAKER_FRONT_RIGHT.
	//
	// Flutter: no counterpart
	SpeakerFrontRight = 0x2
	// SpeakerFrontCenter is SPEAKER_FRONT_CENTER.
	//
	// Flutter: no counterpart
	SpeakerFrontCenter = 0x4
	// SpeakerLowFrequency is SPEAKER_LOW_FREQUENCY.
	//
	// Flutter: no counterpart
	SpeakerLowFrequency = 0x8
	// SpeakerBackLeft is SPEAKER_BACK_LEFT.
	//
	// Flutter: no counterpart
	SpeakerBackLeft = 0x10
	// SpeakerBackRight is SPEAKER_BACK_RIGHT.
	//
	// Flutter: no counterpart
	SpeakerBackRight = 0x20
	// SpeakerBackCenter is SPEAKER_BACK_CENTER.
	//
	// Flutter: no counterpart
	SpeakerBackCenter = 0x100
	// SpeakerSideLeft is SPEAKER_SIDE_LEFT.
	//
	// Flutter: no counterpart
	SpeakerSideLeft = 0x200
	// SpeakerSideRight is SPEAKER_SIDE_RIGHT.
	//
	// Flutter: no counterpart
	SpeakerSideRight = 0x400
)

// The results of the audio client that the audio code tells apart.
const (
	// AudclntEDeviceInvalidated is AUDCLNT_E_DEVICE_INVALIDATED: the endpoint was unplugged, disabled or reconfigured, and the
	// client must be made again.
	//
	// Flutter: no counterpart
	AudclntEDeviceInvalidated HRESULT = -0x7776FFFC // 0x88890004
	// AudclntEUnsupportedFormat is AUDCLNT_E_UNSUPPORTED_FORMAT.
	//
	// Flutter: no counterpart
	AudclntEUnsupportedFormat HRESULT = -0x7776FFF8 // 0x88890008
	// AudclntEDeviceInUse is AUDCLNT_E_DEVICE_IN_USE: another program has the device in exclusive mode.
	//
	// Flutter: no counterpart
	AudclntEDeviceInUse HRESULT = -0x7776FFF6 // 0x8889000A
	// AudclntEResourcesInvalidated is AUDCLNT_E_RESOURCES_INVALIDATED: the resources of the client were taken away.
	//
	// Flutter: no counterpart
	AudclntEResourcesInvalidated HRESULT = -0x7776FFDA // 0x88890026
	// AudclntEServiceNotRunning is AUDCLNT_E_SERVICE_NOT_RUNNING: the Windows Audio service is stopped.
	//
	// Flutter: no counterpart
	AudclntEServiceNotRunning HRESULT = -0x7776FFF0 // 0x88890010
	// EAccessDenied is E_ACCESSDENIED: the user's settings do not let the program use the microphone.
	//
	// Flutter: no counterpart
	EAccessDenied HRESULT = -0x7FF8FFFB // 0x80070005
	// ENotFound is HRESULT_FROM_WIN32(ERROR_NOT_FOUND): an endpoint that is not there.
	//
	// Flutter: no counterpart
	ENotFound HRESULT = -0x7FF8FB70 // 0x80070490
)
