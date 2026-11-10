// Package serviceslines holds the lines that the app of the services tests (internal/iostest/serviceapp) prints and the test reads, and
// the names of its steps, so that the two cannot drift apart. It imports nothing but the standard library, so both the app (built for
// iOS) and the test (built for the host) use it.
//
// Flutter: no counterpart
package serviceslines

// StepsEnv is the environment variable with the steps of the app, separated by ";". Each step runs 400 ms after the one before, and
// prints its line.
//
//	style:dark|light            SetSystemUIOverlayStyle with that status bar brightness
//	mode:edge|hidden|top|bottom SetEnabledSystemUIMode: edge to edge, the bars hidden, or manual with the top or the bottom overlay
//	orient:portrait|landscape|all  SetPreferredOrientations
//	report                      logs what UIKit has of the system UI (the platform's "ios chrome" line)
//	copy:text                   puts the text on the clipboard
//	paste                       reads the text of the clipboard
//	spell:locale:text           the misspelled words of the text
//	haptic, sound               every kind of haptic feedback and of system sound
//	settings                    prints the settings
//	open:uri                    opens the URI
//	notify                      shows a notification (the permission prompt is not answered: the call ends with its context)
const StepsEnv = "GLUTTER_IOS_SERVICES"

// The lines of the app; each is followed by its values.
const (
	// Step is the line of a step that ran: "<step> <result>". The result of copy is "err=<tag or nil>", of paste
	// "ok=<bool> text=<quoted> err=<tag or nil>", of spell "spans=<start,end,first suggestion;...> err=<tag or nil>", of open and of
	// notify "err=<tag or nil>", and of the others "done".
	Step = "GLUTTER_IOS_SERVICES step"
	// Ready is the line that says the app is waiting for its steps.
	Ready = "GLUTTER_IOS_SERVICES ready"
	// Settings is the line of the settings: at the start, for the step "settings" and at each change of them. Its values are
	// "brightness=<light|dark> scale=<text scale> font17=<the scaled size of a 17 point font> bold=<bool> motion=<bool>
	// contrast=<bool> voiceover=<bool> h24=<bool> locale=<the first locale>".
	Settings = "GLUTTER_IOS_SERVICES settings"
	// Done is the line after the last step.
	Done = "GLUTTER_IOS_SERVICES done"
)
