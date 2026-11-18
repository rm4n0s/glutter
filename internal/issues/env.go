package issues

// Env is the environment variable that names the issue an app of a device test reproduces
// (examples/dev/android-parity, internal/iostest/issueapp): the app and the test on the host agree
// on it here.
const Env = "GLUTTER_ISSUE"

// The lines that an app which prints its report into its output (internal/iostest/issueapp, in the
// Simulator) ends the report with, so that the test on the host knows when the report is whole. The
// lines of the report are those of the issue's Lines.
const (
	// DoneLine follows the last line of the report: the run of the issue ended without an error.
	DoneLine = "GLUTTER_ISSUE done"
	// FailedLine is the line of a run that failed, followed by the error.
	FailedLine = "GLUTTER_ISSUE failed"
)
