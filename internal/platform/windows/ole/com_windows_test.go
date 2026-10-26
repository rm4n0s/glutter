//go:build windows

package ole

import (
	"slices"
	"testing"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// These tests call the COM objects of this package through their vtables, the way OLE and the programs
// that drag do (SPEC §25.2, §25.10). They run on Windows, and under Wine.

func testEnv(t *testing.T) *Env {
	t.Helper()
	env, err := Shared()
	if err != nil {
		t.Fatal(err)
	}
	return env
}

func assertRoute(t *testing.T, err error, tag, route string) {
	t.Helper()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if appErr.Tag != tag || !appErr.HasRoute(route) {
		t.Errorf("got %s via %s, want %s via %s", appErr.Tag, appErr.Route(), tag, route)
	}
}

func newData(t *testing.T, env *Env, items ...Item) win32.Object {
	t.Helper()
	obj, err := env.NewDataObject(NewDataSet(items...), Runtime{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { obj.Release() })
	return obj
}

func TestSharedEnvIsMadeOnce(t *testing.T) {
	a, err := Shared()
	if err != nil {
		t.Fatal(err)
	}
	n := win32.CallbackCount()
	b, err := Shared()
	if err != nil || a != b || win32.CallbackCount() != n {
		t.Errorf("Shared made another Env or more callbacks: %p %p %v, %d -> %d", a, b, err, n, win32.CallbackCount())
	}
	if n > win32.CallbackLimit {
		t.Errorf("%d callbacks in the process, the limit is %d", n, win32.CallbackLimit)
	}
}

func TestDataObjectGetDataAndQueryThroughTheVTable(t *testing.T) {
	env := testEnv(t)
	obj := newData(t, env, Item{Format: win32.CFUnicodeText, Data: []byte("t\x00e\x00\x00\x00")}, Item{Format: win32.CFHDrop, Data: []byte{9, 8, 7}})

	good := FormatEtcOf(win32.CFHDrop)
	var medium win32.StgMedium
	hr := win32.FromResult(obj.Call(win32.IDataObjectGetData, uintptr(unsafe.Pointer(&good)), uintptr(unsafe.Pointer(&medium))))
	if hr != win32.SOK || medium.Tymed != win32.TymedHGlobal || medium.Handle == 0 || medium.PUnkForRelease != 0 {
		t.Fatalf("GetData = %v, medium %+v", hr, medium)
	}
	got, err := win32.GlobalBytes(medium.Handle)
	if err != nil || !slices.Equal(got[:3], []byte{9, 8, 7}) {
		t.Errorf("the data = %v, %v", got, err)
	}
	win32.ReleaseStgMedium(&medium)

	// Two calls give two copies: the caller frees its own.
	var m1, m2 win32.StgMedium
	obj.Call(win32.IDataObjectGetData, uintptr(unsafe.Pointer(&good)), uintptr(unsafe.Pointer(&m1)))
	obj.Call(win32.IDataObjectGetData, uintptr(unsafe.Pointer(&good)), uintptr(unsafe.Pointer(&m2)))
	if m1.Handle == m2.Handle {
		t.Error("GetData gave the same memory twice")
	}
	win32.ReleaseStgMedium(&m1)
	win32.ReleaseStgMedium(&m2)

	notThere := FormatEtcOf(win32.CFDIB)
	stream := good
	stream.Tymed = win32.TymedIStream
	aspect := good
	aspect.DwAspect = 2
	index := good
	index.Lindex = 0
	tests := []struct {
		name string
		fe   win32.FormatEtc
		want win32.HRESULT
	}{
		{"the format", good, win32.SOK},
		{"a format it lacks", notThere, win32.DVEFormatEtc},
		{"a medium it lacks", stream, win32.DVETymed},
		{"another aspect", aspect, win32.DVEDVAspect},
		{"an index", index, win32.DVELIndex},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if hr := win32.FromResult(obj.Call(win32.IDataObjectQueryGetData, uintptr(unsafe.Pointer(&tt.fe)))); hr != tt.want {
				t.Errorf("QueryGetData = %v, want %v", hr, tt.want)
			}
			var m win32.StgMedium
			m.Tymed = 99
			hr := win32.FromResult(obj.Call(win32.IDataObjectGetData, uintptr(unsafe.Pointer(&tt.fe)), uintptr(unsafe.Pointer(&m))))
			if hr != tt.want {
				t.Errorf("GetData = %v, want %v", hr, tt.want)
			}
			if tt.want != win32.SOK && (m.Tymed != 0 || m.Handle != 0) {
				t.Errorf("a failed GetData left the medium %+v", m)
			}
			if tt.want == win32.SOK {
				win32.ReleaseStgMedium(&m)
			}
		})
	}
}

func TestDataObjectOtherMethods(t *testing.T) {
	env := testEnv(t)
	obj := newData(t, env, Item{Format: win32.CFUnicodeText, Data: []byte{0, 0}})
	fe := FormatEtcOf(win32.CFUnicodeText)
	var out win32.FormatEtc
	out.Ptd = 5
	// Each call converts its pointers in the call itself: Call is //go:uintptrescapes, which moves fe and out to
	// the heap only then. A pointer made into a uintptr earlier (in a table) leaves them on the stack, which
	// may move while the method runs Go code, and the method then writes to the old place.
	tests := []struct {
		name string
		call func() uintptr
		want win32.HRESULT
	}{
		{"GetDataHere", func() uintptr { return obj.Call(win32.IDataObjectGetDataHere, uintptr(unsafe.Pointer(&fe)), 0) }, win32.ENotImpl},
		{"GetCanonicalFormatEtc", func() uintptr {
			return obj.Call(win32.IDataObjectGetCanonicalFormatEtc, uintptr(unsafe.Pointer(&fe)), uintptr(unsafe.Pointer(&out)))
		}, win32.DataSSameFormatEtc},
		{"SetData", func() uintptr { return obj.Call(win32.IDataObjectSetData, uintptr(unsafe.Pointer(&fe)), 0, 0) }, win32.ENotImpl},
		{"DAdvise", func() uintptr { return obj.Call(win32.IDataObjectDAdvise, uintptr(unsafe.Pointer(&fe)), 0, 0, 0) }, win32.OLEEAdviseNotSupported},
		{"DUnadvise", func() uintptr { return obj.Call(win32.IDataObjectDUnadvise, 1) }, win32.OLEEAdviseNotSupported},
		{"EnumDAdvise", func() uintptr { return obj.Call(win32.IDataObjectEnumDAdvise, 0) }, win32.OLEEAdviseNotSupported},
		{"QueryGetData with null", func() uintptr { return obj.Call(win32.IDataObjectQueryGetData, 0) }, win32.EPointer},
		{"GetData with null", func() uintptr { return obj.Call(win32.IDataObjectGetData, 0, 0) }, win32.EPointer},
		{"EnumFormatEtc with null", func() uintptr { return obj.Call(win32.IDataObjectEnumFormatEtc, win32.DataDirGet, 0) }, win32.EPointer},
	}
	for _, tt := range tests {
		if hr := win32.FromResult(tt.call()); hr != tt.want {
			t.Errorf("%s = %v, want %v", tt.name, hr, tt.want)
		}
	}
	if out.CfFormat != win32.CFUnicodeText || out.Ptd != 0 {
		t.Errorf("the canonical format = %+v", out)
	}
	var en uintptr
	if hr := win32.FromResult(obj.Call(win32.IDataObjectEnumFormatEtc, win32.DataDirSet, uintptr(unsafe.Pointer(&en)))); hr != win32.ENotImpl || en != 0 {
		t.Errorf("EnumFormatEtc(set) = %v, %#x", hr, en)
	}
}

func TestEnumFormatEtcThroughTheVTable(t *testing.T) {
	env := testEnv(t)
	obj := newData(t, env,
		Item{Format: 13, Data: []byte{0, 0}}, Item{Format: 15, Data: []byte{0}}, Item{Format: 0xC001, Data: []byte{0}}, Item{Format: 1, Data: []byte{0}})
	var pEnum uintptr
	if hr := win32.FromResult(obj.Call(win32.IDataObjectEnumFormatEtc, win32.DataDirGet, uintptr(unsafe.Pointer(&pEnum)))); hr != win32.SOK || pEnum == 0 {
		t.Fatalf("EnumFormatEtc = %v, %#x", hr, pEnum)
	}
	en := win32.ObjectFromPtr(pEnum)
	defer en.Release()

	var two [2]win32.FormatEtc
	var fetched uint32
	next := func(o win32.Object, n int) (win32.HRESULT, []uint16) {
		fetched = 0
		two = [2]win32.FormatEtc{}
		hr := win32.FromResult(o.Call(win32.IEnumFormatEtcNext, uintptr(n), uintptr(unsafe.Pointer(&two[0])), uintptr(unsafe.Pointer(&fetched))))
		var fs []uint16
		for i := 0; i < int(fetched); i++ {
			if two[i].DwAspect != win32.DVAspectContent || two[i].Lindex != -1 || two[i].Tymed != win32.TymedHGlobal || two[i].Ptd != 0 {
				t.Errorf("format %d = %+v: want the content aspect on global memory", i, two[i])
			}
			fs = append(fs, two[i].CfFormat)
		}
		return hr, fs
	}
	steps := []struct {
		name string
		do   func() (win32.HRESULT, []uint16)
		hr   win32.HRESULT
		fmts []uint16
	}{
		{"two", func() (win32.HRESULT, []uint16) { return next(en, 2) }, win32.SOK, []uint16{13, 15}},
		{"skip one", func() (win32.HRESULT, []uint16) {
			return win32.FromResult(en.Call(win32.IEnumFormatEtcSkip, 1)), nil
		}, win32.SOK, nil},
		{"the last, asking for two", func() (win32.HRESULT, []uint16) { return next(en, 2) }, win32.SFalse, []uint16{1}},
		{"at the end", func() (win32.HRESULT, []uint16) { return next(en, 1) }, win32.SFalse, nil},
		{"reset", func() (win32.HRESULT, []uint16) {
			return win32.FromResult(en.Call(win32.IEnumFormatEtcReset)), nil
		}, win32.SOK, nil},
		{"one", func() (win32.HRESULT, []uint16) { return next(en, 1) }, win32.SOK, []uint16{13}},
		{"skip past the end", func() (win32.HRESULT, []uint16) {
			return win32.FromResult(en.Call(win32.IEnumFormatEtcSkip, 9)), nil
		}, win32.SFalse, nil},
	}
	for _, s := range steps {
		hr, fmts := s.do()
		if hr != s.hr || !slices.Equal(fmts, s.fmts) {
			t.Errorf("%s: %v %v, want %v %v", s.name, hr, fmts, s.hr, s.fmts)
		}
	}
	// A clone starts where the original is, and moves on its own.
	en.Call(win32.IEnumFormatEtcReset)
	next(en, 1)
	var pClone uintptr
	if hr := win32.FromResult(en.Call(win32.IEnumFormatEtcClone, uintptr(unsafe.Pointer(&pClone)))); hr != win32.SOK || pClone == 0 {
		t.Fatalf("Clone = %v", hr)
	}
	clone := win32.ObjectFromPtr(pClone)
	defer clone.Release()
	if hr, fmts := next(clone, 2); hr != win32.SOK || !slices.Equal(fmts, []uint16{15, 0xC001}) {
		t.Errorf("clone Next = %v %v", hr, fmts)
	}
	if _, fmts := next(en, 1); !slices.Equal(fmts, []uint16{15}) {
		t.Errorf("the original moved with its clone: %v", fmts)
	}
	// A null pointer for a result, and one celt with no count are the cases of the contract.
	if hr := win32.FromResult(en.Call(win32.IEnumFormatEtcNext, 2, uintptr(unsafe.Pointer(&two[0])), 0)); hr != win32.EPointer {
		t.Errorf("Next(2) with no count = %v, want E_POINTER", hr)
	}
	if hr := win32.FromResult(en.Call(win32.IEnumFormatEtcNext, 1, uintptr(unsafe.Pointer(&two[0])), 0)); hr != win32.SOK {
		t.Errorf("Next(1) with no count = %v, want S_OK", hr)
	}
	if hr := win32.FromResult(en.Call(win32.IEnumFormatEtcClone, 0)); hr != win32.EPointer {
		t.Errorf("Clone(null) = %v", hr)
	}
}

func TestReaderListsAndReadsADataObject(t *testing.T) {
	env := testEnv(t)
	obj := newData(t, env, Item{Format: 13, Data: []byte("abc\x00")}, Item{Format: 15, Data: []byte{1, 2}}, Item{Format: 0xC001, Data: nil})
	before := env.Stats().Objects
	r := NewReader(obj)
	formats, err := r.Formats()
	if err != nil || !slices.Equal(formats, []uint32{13, 15, 0xC001}) {
		t.Errorf("Formats = %v, %v", formats, err)
	}
	if got, err := r.Read(13); err != nil || string(got) != "abc\x00" {
		t.Errorf("Read(13) = %q, %v", got, err)
	}
	// An empty format reads as no bytes... the global memory has one byte of padding that Windows
	// reports as the size.
	if got, err := r.Read(0xC001); err != nil || len(got) > 1 {
		t.Errorf("Read of an empty format = %v, %v", got, err)
	}
	_, err = r.Read(win32.CFDIB)
	assertRoute(t, err, win32.TagCOMCallFailed, "Reader.Read->Object.CallHR->HResultError.COMCallFailed")
	if appErr, _ := errors.FromError(err); appErr == nil || appErr.Metadata["hresult"] != win32.DVEFormatEtc.String() || appErr.Metadata["method"] != "IDataObject::GetData" {
		t.Errorf("metadata = %v", appErr)
	}
	r.Close()
	r.Close() // twice releases once
	if after := env.Stats().Objects; after != before {
		t.Errorf("objects: %d before the reader, %d after it closed", before, after)
	}
}

func TestReaderFailuresHaveTheirRoutes(t *testing.T) {
	env := testEnv(t)
	host := env.host
	// A data object of another program that gives its data on a stream, and one that is too large.
	var behaviour struct {
		tymed uint32
		size  uintptr
	}
	other, err := host.NewInterface(win32.InterfaceDef{Name: "IOther", IID: win32.MustParseGUID("{6C0F2E41-3B1D-4D7E-9A57-0F1E2D3C4B5B}")})
	if err != nil {
		t.Fatal(err)
	}
	fake, err := host.NewInterface(win32.InterfaceDef{Name: "IDataObject", IID: win32.IIDDataObject, Methods: []win32.MethodDef{
		{Name: "GetData", Args: 2, Fn: func(_ any, a []uintptr) uintptr {
			if behaviour.tymed != win32.TymedHGlobal {
				// A stream: ReleaseStgMedium releases the object named by pUnkForRelease.
				unk, err := host.NewObject(struct{}{}, other)
				if err != nil {
					return win32.Result(win32.EOutOfMemory)
				}
				*win32.View[win32.StgMedium](a[1]) = win32.StgMedium{Tymed: behaviour.tymed, PUnkForRelease: unk.Ptr()}
				return win32.Result(win32.SOK)
			}
			h, err := win32.GlobalAlloc(win32.GMemMoveable, behaviour.size)
			if err != nil {
				return win32.Result(win32.EOutOfMemory)
			}
			*win32.View[win32.StgMedium](a[1]) = win32.StgMedium{Tymed: behaviour.tymed, Handle: h}
			return win32.Result(win32.SOK)
		}},
		{Name: "GetDataHere", Args: 2, Fn: func(any, []uintptr) uintptr { return win32.Result(win32.ENotImpl) }},
		{Name: "QueryGetData", Args: 1, Fn: func(any, []uintptr) uintptr { return win32.Result(win32.SOK) }},
		{Name: "GetCanonicalFormatEtc", Args: 2, Fn: func(any, []uintptr) uintptr { return win32.Result(win32.ENotImpl) }},
		{Name: "SetData", Args: 3, Fn: func(any, []uintptr) uintptr { return win32.Result(win32.ENotImpl) }},
		{Name: "EnumFormatEtc", Args: 2, Fn: func(any, []uintptr) uintptr { return win32.Result(win32.EFail) }},
		{Name: "DAdvise", Args: 4, Fn: func(any, []uintptr) uintptr { return win32.Result(win32.ENotImpl) }},
		{Name: "DUnadvise", Args: 1, Fn: func(any, []uintptr) uintptr { return win32.Result(win32.ENotImpl) }},
		{Name: "EnumDAdvise", Args: 1, Fn: func(any, []uintptr) uintptr { return win32.Result(win32.ENotImpl) }},
	}})
	if err != nil {
		t.Fatal(err)
	}
	obj, err := host.NewObject(struct{}{}, fake)
	if err != nil {
		t.Fatal(err)
	}
	defer obj.Release()
	r := NewReader(obj)
	defer r.Close()

	behaviour.tymed, behaviour.size = win32.TymedIStream, 0 // a medium we do not read
	_, err = r.Read(13)
	assertRoute(t, err, TagDropMediumUnsupported, "Reader.Read.DropMediumUnsupported")
	if appErr, _ := errors.FromError(err); appErr == nil || appErr.Metadata["format"] != 13 || appErr.Metadata["tymed"] != win32.TymedIStream {
		t.Errorf("metadata = %v", appErr)
	}

	behaviour.tymed, behaviour.size = win32.TymedHGlobal, MaxDataSize+1
	_, err = r.Read(13)
	assertRoute(t, err, TagDropDataTooLarge, "Reader.Read.DropDataTooLarge")
	if appErr, _ := errors.FromError(err); appErr == nil || appErr.Metadata["size"] != MaxDataSize+1 || appErr.Metadata["max"] != MaxDataSize {
		t.Errorf("metadata = %v", appErr)
	}

	behaviour.tymed, behaviour.size = win32.TymedHGlobal, MaxDataSize
	if got, err := r.Read(13); err != nil || len(got) != MaxDataSize {
		t.Errorf("a read of exactly the maximum = %d bytes, %v", len(got), err)
	}

	_, err = r.Formats()
	assertRoute(t, err, win32.TagCOMCallFailed, "Reader.Formats->Object.CallHR->HResultError.COMCallFailed")
}

// recordingTarget records the calls of a Target and answers an effect.
type recordingTarget struct {
	pts     []win32.Point
	keys    []uint32
	allowed []uint32
	formats []uint32
	read    []byte
	left    int
	answer  uint32
	panics  bool
}

func (h *recordingTarget) DragEnter(d DataSource, key uint32, pt win32.Point, allowed uint32) uint32 {
	if h.panics {
		panic("the handler panicked")
	}
	h.pts, h.keys, h.allowed = append(h.pts, pt), append(h.keys, key), append(h.allowed, allowed)
	h.formats, _ = d.Formats()
	h.read, _ = d.Read(win32.CFUnicodeText)
	return h.answer
}
func (h *recordingTarget) DragOver(key uint32, pt win32.Point, allowed uint32) uint32 {
	h.pts, h.keys, h.allowed = append(h.pts, pt), append(h.keys, key), append(h.allowed, allowed)
	return h.answer
}
func (h *recordingTarget) DragLeave() { h.left++ }
func (h *recordingTarget) Drop(d DataSource, key uint32, pt win32.Point, allowed uint32) uint32 {
	h.pts, h.keys, h.allowed = append(h.pts, pt), append(h.keys, key), append(h.allowed, allowed)
	h.read, _ = d.Read(win32.CFUnicodeText)
	return h.answer
}

func TestDropTargetThroughTheVTable(t *testing.T) {
	env := testEnv(t)
	data := newData(t, env, Item{Format: win32.CFUnicodeText, Data: []byte("h\x00i\x00\x00\x00")})
	h := &recordingTarget{answer: win32.DropEffectMove}
	before := env.Stats().Objects
	target, err := env.NewDropTarget(h, Runtime{})
	if err != nil {
		t.Fatal(err)
	}
	// The POINTL is one register: x low, y high; negative coordinates are signed.
	pt := uintptr(uint32(0xFFFFFFF6)) | uintptr(uint32(250))<<32 // -10, 250
	effect := uint32(win32.DropEffectCopy | win32.DropEffectMove)
	hr := win32.FromResult(target.Call(win32.IDropTargetDragEnter, data.Ptr(), win32.MKShift|win32.MKLButton, pt, uintptr(unsafe.Pointer(&effect))))
	if hr != win32.SOK || effect != win32.DropEffectMove {
		t.Errorf("DragEnter = %v, effect %#x", hr, effect)
	}
	if h.pts[0] != (win32.Point{X: -10, Y: 250}) || h.keys[0] != win32.MKShift|win32.MKLButton || h.allowed[0] != win32.DropEffectCopy|win32.DropEffectMove {
		t.Errorf("DragEnter saw %v, key %#x, allowed %#x", h.pts[0], h.keys[0], h.allowed[0])
	}
	if !slices.Equal(h.formats, []uint32{win32.CFUnicodeText}) || string(h.read) != "h\x00i\x00\x00\x00" {
		t.Errorf("the handler read formats %v, text %q", h.formats, h.read)
	}
	effect = win32.DropEffectCopy
	h.answer = win32.DropEffectCopy
	if hr := win32.FromResult(target.Call(win32.IDropTargetDragOver, 0, pt, uintptr(unsafe.Pointer(&effect)))); hr != win32.SOK || effect != win32.DropEffectCopy {
		t.Errorf("DragOver = %v, effect %#x", hr, effect)
	}
	effect = win32.DropEffectCopy
	if hr := win32.FromResult(target.Call(win32.IDropTargetDrop, data.Ptr(), 0, pt, uintptr(unsafe.Pointer(&effect)))); hr != win32.SOK || effect != win32.DropEffectCopy {
		t.Errorf("Drop = %v, effect %#x", hr, effect)
	}
	// After the drop the target is idle: a DragOver answers no effect and a DragLeave does nothing.
	effect = win32.DropEffectCopy
	target.Call(win32.IDropTargetDragOver, 0, pt, uintptr(unsafe.Pointer(&effect)))
	if effect != win32.DropEffectNone {
		t.Errorf("DragOver after the drop left the effect %#x", effect)
	}
	target.Call(win32.IDropTargetDragLeave)
	if h.left != 0 {
		t.Error("DragLeave after the drop reached the handler")
	}
	// Null pointers are E_POINTER.
	if hr := win32.FromResult(target.Call(win32.IDropTargetDragEnter, 0, 0, pt, uintptr(unsafe.Pointer(&effect)))); hr != win32.EPointer {
		t.Errorf("DragEnter(null data) = %v", hr)
	}
	if hr := win32.FromResult(target.Call(win32.IDropTargetDragOver, 0, pt, 0)); hr != win32.EPointer {
		t.Errorf("DragOver(null effect) = %v", hr)
	}
	target.Release()
	if after := env.Stats().Objects; after != before {
		t.Errorf("objects: %d before, %d after the target was released", before, after)
	}
}

func TestDropTargetPanicIsGuardedAndAnswersNoEffect(t *testing.T) {
	env := testEnv(t)
	data := newData(t, env, Item{Format: win32.CFUnicodeText, Data: []byte{0, 0}})
	var recovered any
	rt := Runtime{Guard: func(fn func() uintptr) (r uintptr) {
		defer func() {
			if v := recover(); v != nil {
				recovered, r = v, 0
			}
		}()
		return fn()
	}}
	target, err := env.NewDropTarget(&recordingTarget{panics: true, answer: win32.DropEffectCopy}, rt)
	if err != nil {
		t.Fatal(err)
	}
	defer target.Release()
	effect := uint32(win32.DropEffectCopy)
	hr := win32.FromResult(target.Call(win32.IDropTargetDragEnter, data.Ptr(), 0, 0, uintptr(unsafe.Pointer(&effect))))
	if recovered == nil || hr != win32.SOK || effect != win32.DropEffectNone {
		t.Errorf("recovered %v; DragEnter = %v, effect %#x: a panic answers S_OK and no effect", recovered, hr, effect)
	}
}

func TestDropSourceThroughTheVTable(t *testing.T) {
	env := testEnv(t)
	m := &SourceMachine{Buttons: win32.MKLButton}
	src, err := env.NewDropSource(m, Runtime{})
	if err != nil {
		t.Fatal(err)
	}
	defer src.Release()
	tests := []struct {
		name   string
		escape uintptr
		key    uintptr
		want   win32.HRESULT
	}{
		{"button down", 0, win32.MKLButton, win32.SOK},
		{"button up", 0, 0, win32.DragDropSDrop},
		{"escape", 1, win32.MKLButton, win32.DragDropSCancel},
		{"escape as a BOOL with garbage in the high bits", 0xFFFFFFFF00000001, win32.MKLButton, win32.DragDropSCancel},
		{"a BOOL of zero with garbage in the high bits", 0xFFFFFFFF00000000, win32.MKLButton, win32.SOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if hr := win32.FromResult(src.Call(win32.IDropSourceQueryContinueDrag, tt.escape, tt.key)); hr != tt.want {
				t.Errorf("QueryContinueDrag = %v, want %v", hr, tt.want)
			}
		})
	}
	if hr := win32.FromResult(src.Call(win32.IDropSourceGiveFeedback, win32.DropEffectLink)); hr != win32.DragDropSUseDefaultCursors || m.LastEffect != win32.DropEffectLink {
		t.Errorf("GiveFeedback = %v, effect %#x", hr, m.LastEffect)
	}
	m.Cancelled = true
	if hr := win32.FromResult(src.Call(win32.IDropSourceQueryContinueDrag, 0, win32.MKLButton)); hr != win32.DragDropSCancel {
		t.Errorf("after the platform cancelled, QueryContinueDrag = %v", hr)
	}
}

func TestObjectsAnswerQueryInterface(t *testing.T) {
	env := testEnv(t)
	for _, tt := range []struct {
		name string
		make func() (win32.Object, error)
		iid  win32.GUID
		not  win32.GUID
	}{
		{"data object", func() (win32.Object, error) { return env.NewDataObject(NewDataSet(), Runtime{}) }, win32.IIDDataObject, win32.IIDDropTarget},
		{"drop source", func() (win32.Object, error) { return env.NewDropSource(&SourceMachine{}, Runtime{}) }, win32.IIDDropSource, win32.IIDDataObject},
		{"drop target", func() (win32.Object, error) { return env.NewDropTarget(&recordingTarget{}, Runtime{}) }, win32.IIDDropTarget, win32.IIDDropSource},
	} {
		t.Run(tt.name, func(t *testing.T) {
			obj, err := tt.make()
			if err != nil {
				t.Fatal(err)
			}
			defer obj.Release()
			same, err := obj.QueryInterface(&tt.iid)
			if err != nil || same.Ptr() != obj.Ptr() {
				t.Errorf("QueryInterface(own IID) = %v, %v", same, err)
			} else {
				same.Release()
			}
			unk, err := obj.QueryInterface(&win32.IIDUnknown)
			if err != nil {
				t.Errorf("QueryInterface(IUnknown): %v", err)
			} else {
				unk.Release()
			}
			if _, err := obj.QueryInterface(&tt.not); err == nil {
				t.Error("QueryInterface of an interface it does not have worked")
			}
		})
	}
}
