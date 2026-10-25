//go:build windows

package ole

import "github.com/rm4n0s/glutter/internal/platform/windows/win32"

// sourceObj is the Go value behind an IDropSource.
type sourceObj struct {
	rt Runtime
	m  *SourceMachine
}

func (o *sourceObj) runtime() Runtime { return o.rt }

// NewDropSource makes an IDropSource that follows the machine. The result has one reference.
//
// Flutter: no counterpart
func (e *Env) NewDropSource(m *SourceMachine, rt Runtime) (win32.Object, error) {
	return e.host.NewObject(&sourceObj{rt: rt, m: m}, e.source)
}

func (e *Env) sourceMethods() []win32.MethodDef {
	return []win32.MethodDef{
		// QueryContinueDrag(fEscapePressed BOOL, grfKeyState DWORD): both are 32 bits of a register.
		e.method("QueryContinueDrag", 2, func(impl any, a []uintptr) uintptr {
			d := impl.(*sourceObj).m.Query(uint32(a[0]) != 0, uint32(a[1]))
			return win32.Result(d.HRESULT())
		}),
		// GiveFeedback(dwEffect DWORD).
		e.method("GiveFeedback", 1, func(impl any, a []uintptr) uintptr {
			return win32.Result(impl.(*sourceObj).m.Feedback(uint32(a[0])))
		}),
	}
}
