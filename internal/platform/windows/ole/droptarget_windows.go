//go:build windows

package ole

import (
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// targetObj is the Go value behind an IDropTarget.
type targetObj struct {
	rt Runtime
	t  Target
}

func (o *targetObj) runtime() Runtime { return o.rt }

// NewDropTarget makes an IDropTarget whose calls go to the handler. The result has one reference, which
// RegisterDragDrop (this package's) gives back after OLE took its own.
//
// Flutter: no counterpart
func (e *Env) NewDropTarget(h TargetHandler, rt Runtime) (win32.Object, error) {
	return e.host.NewObject(&targetObj{rt: rt, t: Target{H: h}}, e.target)
}

func (e *Env) targetMethods() []win32.MethodDef {
	return []win32.MethodDef{
		e.method("DragEnter", 4, func(impl any, a []uintptr) uintptr { return impl.(*targetObj).dragEnter(a) }),
		e.method("DragOver", 3, func(impl any, a []uintptr) uintptr { return impl.(*targetObj).dragOver(a) }),
		e.method("DragLeave", 0, func(impl any, _ []uintptr) uintptr { return impl.(*targetObj).dragLeave() }),
		e.method("Drop", 4, func(impl any, a []uintptr) uintptr { return impl.(*targetObj).drop(a) }),
	}
}

// dragEnter is IDropTarget::DragEnter(pDataObj, grfKeyState, pt, pdwEffect). The effect is set to none
// before the handler runs, so that a handler that panicked answers that.
func (o *targetObj) dragEnter(a []uintptr) uintptr {
	if a[0] == 0 || a[3] == 0 {
		return win32.Result(win32.EPointer)
	}
	effect := win32.View[uint32](a[3])
	allowed := *effect
	*effect = win32.DropEffectNone
	*effect = o.t.Enter(NewReader(win32.ObjectFromPtr(a[0])), uint32(a[1]), pointOf(a[2]), allowed)
	return win32.Result(win32.SOK)
}

// dragOver is IDropTarget::DragOver(grfKeyState, pt, pdwEffect).
func (o *targetObj) dragOver(a []uintptr) uintptr {
	if a[2] == 0 {
		return win32.Result(win32.EPointer)
	}
	effect := win32.View[uint32](a[2])
	allowed := *effect
	*effect = win32.DropEffectNone
	*effect = o.t.Over(uint32(a[0]), pointOf(a[1]), allowed)
	return win32.Result(win32.SOK)
}

// dragLeave is IDropTarget::DragLeave().
func (o *targetObj) dragLeave() uintptr {
	o.t.Leave()
	return win32.Result(win32.SOK)
}

// drop is IDropTarget::Drop(pDataObj, grfKeyState, pt, pdwEffect).
func (o *targetObj) drop(a []uintptr) uintptr {
	if a[0] == 0 || a[3] == 0 {
		return win32.Result(win32.EPointer)
	}
	effect := win32.View[uint32](a[3])
	allowed := *effect
	*effect = win32.DropEffectNone
	*effect = o.t.Drop(NewReader(win32.ObjectFromPtr(a[0])), uint32(a[1]), pointOf(a[2]), allowed)
	return win32.Result(win32.SOK)
}

// RegisterDragDrop makes a drop target of the window, and gives its own reference back: OLE keeps the
// one it took, so RevokeDragDrop frees the object.
//
// Flutter: no counterpart
func RegisterDragDrop(hwnd uintptr, target win32.Object) error {
	err := win32.RegisterDragDrop(hwnd, target)
	target.Release()
	return err
}
