//go:build windows

package ole

import (
	"slices"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// MaxDataSize is the most bytes of one format that Read copies: a hostile source cannot ask the app for
// gigabytes. A list of files of several hundred thousand paths is smaller.
//
// Flutter: no counterpart
const MaxDataSize = 16 << 20

// Reader reads the data of an IDataObject that another program (or this app) made: it is a
// DataSource. It holds a reference to the object, which Close gives back.
//
// Flutter: no counterpart
type Reader struct {
	obj    win32.Object
	closed bool
}

var _ DataSource = (*Reader)(nil)

// NewReader takes a reference to the data object.
//
// Flutter: no counterpart
func NewReader(obj win32.Object) *Reader {
	obj.AddRef()
	return &Reader{obj: obj}
}

// Close releases the data object; closing twice does nothing.
//
// Flutter: no counterpart
func (r *Reader) Close() {
	if r.closed {
		return
	}
	r.closed = true
	r.obj.Release()
}

// Formats lists the formats that the object can give on global memory, once each, in its order
// (IDataObject::EnumFormatEtc). It fails with COMCallFailed when the object has no enumerator.
//
// Flutter: no counterpart
func (r *Reader) Formats() ([]uint32, error) {
	var pEnum uintptr
	if err := r.obj.CallHR("IDataObject::EnumFormatEtc", win32.IDataObjectEnumFormatEtc, win32.DataDirGet, uintptr(unsafe.Pointer(&pEnum))); err != nil {
		return nil, err
	}
	en := win32.ObjectFromPtr(pEnum)
	defer en.Release()
	var formats []uint32
	for range 4096 { // a source that never ends cannot hold the app
		var fe win32.FormatEtc
		var fetched uint32
		hr := win32.FromResult(en.Call(win32.IEnumFormatEtcNext, 1, uintptr(unsafe.Pointer(&fe)), uintptr(unsafe.Pointer(&fetched))))
		if hr.Failed() {
			return formats, win32.HResultError("IEnumFORMATETC::Next", hr)
		}
		if fetched != 1 {
			break
		}
		if fe.Tymed&win32.TymedHGlobal != 0 && !slices.Contains(formats, uint32(fe.CfFormat)) {
			formats = append(formats, uint32(fe.CfFormat))
		}
	}
	return formats, nil
}

// Read copies the data of the format (IDataObject::GetData with TYMED_HGLOBAL, then ReleaseStgMedium). It
// fails with COMCallFailed when the object does not have the format, DropMediumUnsupported when it gave
// another medium than global memory, and DropDataTooLarge above MaxDataSize.
//
// Flutter: no counterpart
func (r *Reader) Read(format uint32) ([]byte, error) {
	fe := FormatEtcOf(format)
	var medium win32.StgMedium
	if err := r.obj.CallHR("IDataObject::GetData", win32.IDataObjectGetData, uintptr(unsafe.Pointer(&fe)), uintptr(unsafe.Pointer(&medium))); err != nil {
		return nil, err
	}
	defer win32.ReleaseStgMedium(&medium)
	if medium.Tymed != win32.TymedHGlobal {
		return nil, errors.New(TagDropMediumUnsupported, "the data of a drag is not on global memory", "format", int(format), "tymed", int(medium.Tymed))
	}
	if size := win32.GlobalSize(medium.Handle); size > MaxDataSize {
		return nil, errors.New(TagDropDataTooLarge, "the data of a drag is too large", "format", int(format), "size", int(size), "max", MaxDataSize)
	}
	return win32.GlobalBytes(medium.Handle)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	// TagDropMediumUnsupported is the tag of the error for the data of a drag that is not on global
	// memory (a stream or a file, which the app does not read); the metadata are "format" and "tymed".
	TagDropMediumUnsupported = "DropMediumUnsupported"
	// TagDropDataTooLarge is the tag of the error for the data of a drag above MaxDataSize; the
	// metadata are "format", "size" and "max".
	TagDropDataTooLarge = "DropDataTooLarge"
)
