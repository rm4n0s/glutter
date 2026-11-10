// Package inputlines holds the lines that the app of the input tests (internal/iostest/inputapp) prints and the tests read, so that the two
// cannot drift apart. It imports nothing, so both the app (built for iOS) and the tests (built for the host) use it.
//
// Flutter: no counterpart
package inputlines

const (
	// Tap is the line of a tap: "<x> <y> <kind>" with the place in logical pixels.
	Tap = "GLUTTER_IOS_INPUT tap"
	// Pan is the line of the end of a pan: "<dx> <dy>", the distance that the pointer went in logical pixels.
	Pan = "GLUTTER_IOS_INPUT pan"
	// Key is the line of a key event: "<down|up> <physical key> <logical key> <character>".
	Key = "GLUTTER_IOS_INPUT key"
)
