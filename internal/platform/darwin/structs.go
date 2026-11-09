package darwin

// The Go mirrors of the C structures that the macOS and iOS backends pass to the system and get back
// (SPEC §29.2). They are in a file with no OS tag, so that their layout tests run wherever the SDK's
// headers are (internal/platform/darwin/layouttest); purego passes and returns a structure by value
// on darwin (arm64 and amd64), which these are made for. Only 64-bit targets build, so CGFloat is
// float64 and NSUInteger is uint64.

// CGFloat is Core Graphics' floating point type on a 64-bit target.
//
// Flutter: no counterpart
type CGFloat = float64

// CGPoint is the point of Core Graphics (CGPoint).
//
// Flutter: no counterpart
type CGPoint struct {
	X CGFloat `c:"x"`
	Y CGFloat `c:"y"`
}

// CGSize is the size of Core Graphics (CGSize).
//
// Flutter: no counterpart
type CGSize struct {
	Width  CGFloat `c:"width"`
	Height CGFloat `c:"height"`
}

// CGRect is the rectangle of Core Graphics (CGRect); AppKit's NSRect is the same type.
//
// Flutter: no counterpart
type CGRect struct {
	Origin CGPoint `c:"origin"`
	Size   CGSize  `c:"size"`
}

// NSEdgeInsets is AppKit's NSEdgeInsets, the insets that the safe area of a window is told with.
//
// Flutter: no counterpart
type NSEdgeInsets struct {
	Top    CGFloat `c:"top"`
	Left   CGFloat `c:"left"`
	Bottom CGFloat `c:"bottom"`
	Right  CGFloat `c:"right"`
}

// NSRange is Foundation's NSRange: a location and a length in UTF-16 units, which the text input
// client of a view is told with.
//
// Flutter: no counterpart
type NSRange struct {
	Location uint64 `c:"location"`
	Length   uint64 `c:"length"`
}

// CVSMPTETime is Core Video's CVSMPTETime, the time code inside a CVTimeStamp.
//
// Flutter: no counterpart
type CVSMPTETime struct {
	Subframes       int16  `c:"subframes"`
	SubframeDivisor int16  `c:"subframeDivisor"`
	Counter         uint32 `c:"counter"`
	Type            uint32 `c:"type"`
	Flags           uint32 `c:"flags"`
	Hours           int16  `c:"hours"`
	Minutes         int16  `c:"minutes"`
	Seconds         int16  `c:"seconds"`
	Frames          int16  `c:"frames"`
}

// CVTimeStamp is Core Video's CVTimeStamp, which a display link gives the callback of the frame.
//
// Flutter: no counterpart
type CVTimeStamp struct {
	Version            uint32      `c:"version"`
	VideoTimeScale     int32       `c:"videoTimeScale"`
	VideoTime          int64       `c:"videoTime"`
	HostTime           uint64      `c:"hostTime"`
	RateScalar         float64     `c:"rateScalar"`
	VideoRefreshPeriod int64       `c:"videoRefreshPeriod"`
	SMPTETime          CVSMPTETime `c:"smpteTime"`
	Flags              uint64      `c:"flags"`
	Reserved           uint64      `c:"reserved"`
}
