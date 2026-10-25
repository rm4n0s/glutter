package ole

import "github.com/rm4n0s/glutter/internal/platform/windows/win32"

// DataSource is the data of a drag as a drop target sees it: the formats on offer and the bytes of
// each. Reader implements it over the IDataObject of the source; the tests use a fake. Close gives the
// reference to the data object back; the target closes a source once.
//
// Flutter: no counterpart
type DataSource interface {
	// Formats lists the formats that can be read as global memory, in the source's order.
	Formats() ([]uint32, error)
	// Read returns a copy of the data of the format.
	Read(format uint32) ([]byte, error)
	// Close releases the data object.
	Close()
}

// TargetHandler is what a window does with the drag over it (SPEC §25.10). The effect arguments and
// results are DROPEFFECT_* sets: the argument is what the source allows, the result is the effect the
// target chooses (DROPEFFECT_NONE for none). The point is in screen coordinates. All of it runs on the
// UI thread.
//
// Flutter: no counterpart
type TargetHandler interface {
	// DragEnter is a drag that came over the window. data is valid until DragLeave or Drop returns.
	DragEnter(data DataSource, keyState uint32, pt win32.Point, effect uint32) uint32
	// DragOver is the drag moving over the window, or the keys changing.
	DragOver(keyState uint32, pt win32.Point, effect uint32) uint32
	// DragLeave is the drag leaving the window without a drop.
	DragLeave()
	// Drop is the drop on the window; the target reads the data now.
	Drop(data DataSource, keyState uint32, pt win32.Point, effect uint32) uint32
}

// Target is the state machine of an IDropTarget: it keeps the data object between DragEnter and
// DragLeave or Drop, and makes sure it is released once, in the order OLE calls the methods. A DragOver
// or DragLeave with no DragEnter before it does nothing and answers no effect.
//
// Flutter: no counterpart
type Target struct {
	// H handles the drag.
	H    TargetHandler
	data DataSource
}

func (t *Target) release() {
	if t.data != nil {
		t.data.Close()
		t.data = nil
	}
}

// Enter is IDropTarget::DragEnter: data is a new reference that the target keeps until the drag ends.
//
// Flutter: no counterpart
func (t *Target) Enter(data DataSource, keyState uint32, pt win32.Point, effect uint32) uint32 {
	t.release() // a DragEnter after a drag that never ended
	t.data = data
	return t.H.DragEnter(data, keyState, pt, effect)
}

// Over is IDropTarget::DragOver.
//
// Flutter: no counterpart
func (t *Target) Over(keyState uint32, pt win32.Point, effect uint32) uint32 {
	if t.data == nil {
		return win32.DropEffectNone
	}
	return t.H.DragOver(keyState, pt, effect)
}

// Leave is IDropTarget::DragLeave.
//
// Flutter: no counterpart
func (t *Target) Leave() {
	if t.data == nil {
		return
	}
	t.H.DragLeave()
	t.release()
}

// Drop is IDropTarget::Drop: data is a new reference that is released when the call returns, as is the
// one that DragEnter kept.
//
// Flutter: no counterpart
func (t *Target) Drop(data DataSource, keyState uint32, pt win32.Point, effect uint32) uint32 {
	defer data.Close()
	defer t.release()
	return t.H.Drop(data, keyState, pt, effect)
}
