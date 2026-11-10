package ios

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/ui"
)

// This file has what the iOS platform does not have in step 35.2 and answers with an error, in plain Go (no OS tag), so that a test of
// each route runs on every host. The steps that follow replace them: the pasteboard came with step 35.6 (pasteboard.go).

// unsupportedWindow is embedded by the window: the calls of platform.Window that iOS has no counterpart for.
type unsupportedWindow struct{}

// StartDrag fails with IOSDragUnsupported: dragging out of the app is not in 1.15 (SPEC §30.9).
func (unsupportedWindow) StartDrag(ui.DragRequest) error {
	return errors.New(tagIOSDragUnsupported, "dragging out of an iOS app is not supported")
}

const (
	tagIOSDragUnsupported = "IOSDragUnsupported"
)
