package widgets

import "github.com/rm4n0s/glutter/rendering"

// rangeKeepingOffset is the offset that EditableText gives to its
// RenderEditable. When the text got shorter than the offset, it brings the
// offset to the new end in the layout that learns the new extent.
//
// Deviation: Flutter gives the scroll position as it is. Its physics correct a
// position only if it did not move since the last layout (they compare the
// metrics of the last layout), and a field that scrolled to its caret has moved;
// so the position keeps its pixels past the end, the idle activity goes
// ballistic and the field springs back over about a second. The offset of a
// user's drag or fling (its user scroll direction is not idle) is left alone.
// SPEC §12.10.
//
// Flutter: no counterpart
type rangeKeepingOffset struct {
	rendering.ViewportOffset
}

// ApplyContentDimensions corrects the offset to the new end of the content, if
// it is idle and past it, and then gives the dimensions to the offset.
//
// Flutter: no counterpart
func (o *rangeKeepingOffset) ApplyContentDimensions(minScrollExtent, maxScrollExtent float64) bool {
	if o.HasPixels() && o.UserScrollDirection() == rendering.ScrollDirectionIdle && o.Pixels() > maxScrollExtent {
		o.CorrectBy(maxScrollExtent - o.Pixels())
	}
	return o.ViewportOffset.ApplyContentDimensions(minScrollExtent, maxScrollExtent)
}
