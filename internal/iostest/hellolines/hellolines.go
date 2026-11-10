// Package hellolines has the lines that the app of the Simulator tests prints, which the test looks for. It imports nothing, so that
// the app and the test both can.
//
// Flutter: no counterpart
package hellolines

const (
	// Launched is the line of the app that has finished launching; the value of EnvName follows it.
	//
	// Flutter: no counterpart
	Launched = "GLUTTER_IOS_HELLO launched"
	// SceneConnected is the line of the app whose scene has connected.
	//
	// Flutter: no counterpart
	SceneConnected = "GLUTTER_IOS_HELLO scene connected"
	// NotMainThread is the line of the app that called Main from a goroutine: the tag and whether the route of the error names Main,
	// which follow it ("tag=IOSNotMainThread route=true").
	//
	// Flutter: no counterpart
	NotMainThread = "GLUTTER_IOS_HELLO not main thread"
	// ModeEnv is the environment variable that makes the app call Main from a goroutine when its value is "goroutine".
	//
	// Flutter: no counterpart
	ModeEnv = "GLUTTER_IOS_HELLO_MODE"
	// EnvName is the environment variable that the test gives the app, to see that the environment of a launch arrives.
	//
	// Flutter: no counterpart
	EnvName = "GLUTTER_IOS_HELLO_ENV"
)
