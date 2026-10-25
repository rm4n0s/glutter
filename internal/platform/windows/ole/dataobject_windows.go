//go:build windows

package ole

import (
	"unsafe"

	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// dataObj is the Go value behind an IDataObject.
type dataObj struct {
	env *Env
	rt  Runtime
	set *DataSet
}

func (o *dataObj) runtime() Runtime { return o.rt }

// enumObj is the Go value behind an IEnumFORMATETC.
type enumObj struct {
	env *Env
	rt  Runtime
	e   *FormatEnum
}

func (o *enumObj) runtime() Runtime { return o.rt }

// NewDataObject makes an IDataObject that offers the data set (SPEC §25.10): every format on global
// memory, which GetData copies for the caller each time. The result has one reference.
//
// Flutter: no counterpart
func (e *Env) NewDataObject(set *DataSet, rt Runtime) (win32.Object, error) {
	return e.host.NewObject(&dataObj{env: e, rt: rt, set: set}, e.data)
}

func (e *Env) dataMethods() []win32.MethodDef {
	notImpl := func(name string, args int) win32.MethodDef {
		return e.method(name, args, func(any, []uintptr) uintptr { return win32.Result(win32.ENotImpl) })
	}
	noAdvise := func(name string, args int) win32.MethodDef {
		return e.method(name, args, func(any, []uintptr) uintptr { return win32.Result(win32.OLEEAdviseNotSupported) })
	}
	return []win32.MethodDef{
		e.method("GetData", 2, func(impl any, a []uintptr) uintptr { return impl.(*dataObj).getData(a[0], a[1]) }),
		notImpl("GetDataHere", 2),
		e.method("QueryGetData", 1, func(impl any, a []uintptr) uintptr { return impl.(*dataObj).queryGetData(a[0]) }),
		e.method("GetCanonicalFormatEtc", 2, func(_ any, a []uintptr) uintptr {
			// Every format is its own canonical one.
			if a[1] != 0 {
				if in := win32.View[win32.FormatEtc](a[0]); a[0] != 0 {
					out := win32.View[win32.FormatEtc](a[1])
					*out = *in
					out.Ptd = 0
				}
			}
			return win32.Result(win32.DataSSameFormatEtc)
		}),
		notImpl("SetData", 3),
		e.method("EnumFormatEtc", 2, func(impl any, a []uintptr) uintptr { return impl.(*dataObj).enumFormatEtc(uint32(a[0]), a[1]) }),
		noAdvise("DAdvise", 4),
		noAdvise("DUnadvise", 1),
		noAdvise("EnumDAdvise", 1),
	}
}

// queryGetData is IDataObject::QueryGetData(pformatetc).
func (o *dataObj) queryGetData(pfe uintptr) uintptr {
	if pfe == 0 {
		return win32.Result(win32.EPointer)
	}
	return win32.Result(o.set.Query(*win32.View[win32.FormatEtc](pfe)))
}

// getData is IDataObject::GetData(pformatetcIn, pmedium): a copy of the data on global memory that the
// caller frees with ReleaseStgMedium.
func (o *dataObj) getData(pfe, pmedium uintptr) uintptr {
	if pfe == 0 || pmedium == 0 {
		return win32.Result(win32.EPointer)
	}
	medium := win32.View[win32.StgMedium](pmedium)
	*medium = win32.StgMedium{}
	fe := *win32.View[win32.FormatEtc](pfe)
	if hr := o.set.Query(fe); hr.Failed() {
		return win32.Result(hr)
	}
	data, _ := o.set.Get(uint32(fe.CfFormat))
	h, err := win32.NewGlobal(data)
	if err != nil {
		o.rt.log().Error("the data of a drag could not be copied to global memory", "err", err)
		return win32.Result(win32.EOutOfMemory)
	}
	*medium = win32.StgMedium{Tymed: win32.TymedHGlobal, Handle: h}
	return win32.Result(win32.SOK)
}

// enumFormatEtc is IDataObject::EnumFormatEtc(dwDirection, ppenumFormatEtc): the formats to get; the
// formats to set are none (E_NOTIMPL, which the shell reads as "ask the formats yourself").
func (o *dataObj) enumFormatEtc(direction uint32, pp uintptr) uintptr {
	if pp == 0 {
		return win32.Result(win32.EPointer)
	}
	*win32.View[uintptr](pp) = 0
	if direction != win32.DataDirGet {
		return win32.Result(win32.ENotImpl)
	}
	obj, err := o.env.host.NewObject(&enumObj{env: o.env, rt: o.rt, e: NewFormatEnum(o.set.Formats())}, o.env.enum)
	if err != nil {
		o.rt.log().Error("the enumerator of the formats of a drag could not be made", "err", err)
		return win32.Result(win32.EOutOfMemory)
	}
	*win32.View[uintptr](pp) = obj.Ptr()
	return win32.Result(win32.SOK)
}

func (e *Env) enumMethods() []win32.MethodDef {
	return []win32.MethodDef{
		e.method("Next", 3, func(impl any, a []uintptr) uintptr { return impl.(*enumObj).next(int(uint32(a[0])), a[1], a[2]) }),
		e.method("Skip", 1, func(impl any, a []uintptr) uintptr {
			if impl.(*enumObj).e.Skip(int(uint32(a[0]))) {
				return win32.Result(win32.SOK)
			}
			return win32.Result(win32.SFalse)
		}),
		e.method("Reset", 0, func(impl any, _ []uintptr) uintptr {
			impl.(*enumObj).e.Reset()
			return win32.Result(win32.SOK)
		}),
		e.method("Clone", 1, func(impl any, a []uintptr) uintptr { return impl.(*enumObj).clone(a[0]) }),
	}
}

// next is IEnumFORMATETC::Next(celt, rgelt, pceltFetched).
func (o *enumObj) next(celt int, rgelt, pFetched uintptr) uintptr {
	if rgelt == 0 || (pFetched == 0 && celt != 1) {
		return win32.Result(win32.EPointer)
	}
	formats := o.e.Next(celt)
	size := unsafe.Sizeof(win32.FormatEtc{})
	for i, f := range formats {
		*win32.View[win32.FormatEtc](rgelt + uintptr(i)*size) = FormatEtcOf(f)
	}
	if pFetched != 0 {
		*win32.View[uint32](pFetched) = uint32(len(formats))
	}
	if len(formats) == celt {
		return win32.Result(win32.SOK)
	}
	return win32.Result(win32.SFalse)
}

// clone is IEnumFORMATETC::Clone(ppenum).
func (o *enumObj) clone(pp uintptr) uintptr {
	if pp == 0 {
		return win32.Result(win32.EPointer)
	}
	*win32.View[uintptr](pp) = 0
	obj, err := o.env.host.NewObject(&enumObj{env: o.env, rt: o.rt, e: o.e.Clone()}, o.env.enum)
	if err != nil {
		o.rt.log().Error("the enumerator of the formats of a drag could not be cloned", "err", err)
		return win32.Result(win32.EOutOfMemory)
	}
	*win32.View[uintptr](pp) = obj.Ptr()
	return win32.Result(win32.SOK)
}
