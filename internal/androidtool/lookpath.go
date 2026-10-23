package androidtool

import "os/exec"

// lookPath is exec.LookPath; a variable so that a test can say a program is missing.
var lookPath = exec.LookPath
