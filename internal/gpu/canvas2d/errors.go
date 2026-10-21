package canvas2d

import (
	"strconv"

	"github.com/rm4n0s/errors"
)

// errCanvasNotFound is the error for a view whose canvas is not in the page.
func errCanvasNotFound(view uintptr) error {
	return errors.New("CanvasNotFound", "there is no canvas for the view in the page", "view", strconv.FormatUint(uint64(view), 10))
}

// errContextUnavailable is the error for a canvas that gives no 2D context: a canvas keeps
// the first kind of context it was asked for (SPEC §9.6).
func errContextUnavailable(view uintptr) error {
	return errors.New("Canvas2DUnavailable", "the canvas gave no 2D context", "view", strconv.FormatUint(uint64(view), 10))
}
