// Package behaviorlines holds the lines that the app of the framework-behavior tests (internal/iostest/behaviorapp) prints and the tests
// read. It imports nothing, so both the app (built for iOS) and the tests (built for the host) use it.
//
// Flutter: no counterpart
package behaviorlines

const (
	// ModeEnv is the environment variable that picks the page of the app: "scroll", "back" or "select".
	ModeEnv = "GLUTTER_IOS_BEHAVIOR"
	// Offset is the line of a change of the scroll offset of the list: "<pixels>" (a negative number is the bounce above the first item).
	Offset = "GLUTTER_IOS_BEHAVIOR offset"
	// Popped is the line of the end of the detail page of the "back" mode: the page was popped (the back swipe finished).
	Popped = "GLUTTER_IOS_BEHAVIOR popped"
	// Pushed is the line of the first build of the detail page of the "back" mode.
	Pushed = "GLUTTER_IOS_BEHAVIOR pushed"
	// Selection is the line of a change of the selection of the field of the "select" mode: "<base> <extent>".
	Selection = "GLUTTER_IOS_BEHAVIOR selection"
)
