package win32

// The values of OLE drag and drop (SPEC §25.10): the effects, the clipboard formats and media of
// the data objects, and the results of the COM methods that Glutter implements or calls. They carry
// no OS tag, so that the layout test checks them against the headers.

// The effects of a drag and drop operation (DROPEFFECT_*), a bit set.
const (
	// DropEffectNone is DROPEFFECT_NONE: the drop is not accepted.
	//
	// Flutter: no counterpart
	DropEffectNone = 0
	// DropEffectCopy is DROPEFFECT_COPY.
	//
	// Flutter: no counterpart
	DropEffectCopy = 1
	// DropEffectMove is DROPEFFECT_MOVE.
	//
	// Flutter: no counterpart
	DropEffectMove = 2
	// DropEffectLink is DROPEFFECT_LINK.
	//
	// Flutter: no counterpart
	DropEffectLink = 4
	// DropEffectScroll is DROPEFFECT_SCROLL: the target scrolls while the pointer is over it.
	//
	// Flutter: no counterpart
	DropEffectScroll = 0x80000000
)

// MKAlt is MK_ALT: the Alt key in the key state of a drop target (the value of MK_XBUTTON1 in the mouse
// messages).
//
// Flutter: no counterpart
const MKAlt = 0x20

// The clipboard formats of the data of a drag besides CF_UNICODETEXT and CF_HDROP.
const (
	// CFText is CF_TEXT: ANSI text.
	//
	// Flutter: no counterpart
	CFText = 1
	// CFOEMText is CF_OEMTEXT: text in the OEM code page.
	//
	// Flutter: no counterpart
	CFOEMText = 7
)

// The registered clipboard formats of a URL (CFSTR_INETURLW and CFSTR_INETURLA): an Internet
// shortcut dragged from a browser. The data is the URL, UTF-16 for the first and ANSI for the second,
// ended by a NUL.
const (
	// FormatNameURLW is the name of the registered format that holds a URL as UTF-16.
	//
	// Flutter: no counterpart
	FormatNameURLW = "UniformResourceLocatorW"
	// FormatNameURL is the name of the registered format that holds a URL as ANSI text.
	//
	// Flutter: no counterpart
	FormatNameURL = "UniformResourceLocator"
)

// The media and aspects of FORMATETC and STGMEDIUM.
const (
	// TymedHGlobal is TYMED_HGLOBAL: the data is in global memory.
	//
	// Flutter: no counterpart
	TymedHGlobal = 1
	// TymedIStream is TYMED_ISTREAM: the data is a stream.
	//
	// Flutter: no counterpart
	TymedIStream = 4
	// DVAspectContent is DVASPECT_CONTENT: the data itself.
	//
	// Flutter: no counterpart
	DVAspectContent = 1
	// DataDirGet is DATADIR_GET: the formats that GetData can give.
	//
	// Flutter: no counterpart
	DataDirGet = 1
	// DataDirSet is DATADIR_SET: the formats that SetData can take.
	//
	// Flutter: no counterpart
	DataDirSet = 2
)

// The results of the drag source's methods and of DoDragDrop. They are success codes, so they are
// not errors (HResultError returns nil for them).
const (
	// DragDropSDrop is DRAGDROP_S_DROP: the drop happens (QueryContinueDrag), or it happened
	// (DoDragDrop).
	//
	// Flutter: no counterpart
	DragDropSDrop HRESULT = 0x00040100
	// DragDropSCancel is DRAGDROP_S_CANCEL: the drag ends without a drop.
	//
	// Flutter: no counterpart
	DragDropSCancel HRESULT = 0x00040101
	// DragDropSUseDefaultCursors is DRAGDROP_S_USEDEFAULTCURSORS: GiveFeedback leaves the cursors of
	// Windows.
	//
	// Flutter: no counterpart
	DragDropSUseDefaultCursors HRESULT = 0x00040102
	// DataSSameFormatEtc is DATA_S_SAMEFORMATETC: GetCanonicalFormatEtc says the format is its own
	// canonical one.
	//
	// Flutter: no counterpart
	DataSSameFormatEtc HRESULT = 0x00040130
	// SFalse is S_FALSE.
	//
	// Flutter: no counterpart
	SFalse HRESULT = 1
)

// The failures of the data object's methods and of the registration of a drop target.
const (
	// EAbort is E_ABORT.
	//
	// Flutter: no counterpart
	EAbort HRESULT = -0x7FFFBFFC // 0x80004004
	// DVEFormatEtc is DV_E_FORMATETC: the data object does not have the format.
	//
	// Flutter: no counterpart
	DVEFormatEtc HRESULT = -0x7FFBFF9C // 0x80040064
	// DVELIndex is DV_E_LINDEX: the index of the FORMATETC is not -1.
	//
	// Flutter: no counterpart
	DVELIndex HRESULT = -0x7FFBFF98 // 0x80040068
	// DVETymed is DV_E_TYMED: the data object does not have the format on that medium.
	//
	// Flutter: no counterpart
	DVETymed HRESULT = -0x7FFBFF97 // 0x80040069
	// DVEDVAspect is DV_E_DVASPECT: the aspect is not DVASPECT_CONTENT.
	//
	// Flutter: no counterpart
	DVEDVAspect HRESULT = -0x7FFBFF95 // 0x8004006B
	// OLEEAdviseNotSupported is OLE_E_ADVISENOTSUPPORTED: the data object has no change notifications.
	//
	// Flutter: no counterpart
	OLEEAdviseNotSupported HRESULT = -0x7FFBFFFD // 0x80040003
	// DragDropENotRegistered is DRAGDROP_E_NOTREGISTERED.
	//
	// Flutter: no counterpart
	DragDropENotRegistered HRESULT = -0x7FFBFF00 // 0x80040100
	// DragDropEAlreadyRegistered is DRAGDROP_E_ALREADYREGISTERED.
	//
	// Flutter: no counterpart
	DragDropEAlreadyRegistered HRESULT = -0x7FFBFEFF // 0x80040101
)

// The interface IDs of the COM interfaces of drag and drop (the ones Go implements, and IEnumFORMATETC).
var (
	// IIDDropTarget is the interface ID of IDropTarget.
	//
	// Flutter: no counterpart
	IIDDropTarget = MustParseGUID("{00000122-0000-0000-C000-000000000046}")
	// IIDDropSource is the interface ID of IDropSource.
	//
	// Flutter: no counterpart
	IIDDropSource = MustParseGUID("{00000121-0000-0000-C000-000000000046}")
	// IIDDataObject is the interface ID of IDataObject.
	//
	// Flutter: no counterpart
	IIDDataObject = MustParseGUID("{0000010E-0000-0000-C000-000000000046}")
	// IIDEnumFormatEtc is the interface ID of IEnumFORMATETC.
	//
	// Flutter: no counterpart
	IIDEnumFormatEtc = MustParseGUID("{00000103-0000-0000-C000-000000000046}")
)

// The positions of the methods in the vtables of drag and drop. The layout test checks each against
// the headers.
const (
	// IDropTargetDragEnter is the index of IDropTarget::DragEnter.
	//
	// Flutter: no counterpart
	IDropTargetDragEnter = 3
	// IDropTargetDragOver is the index of IDropTarget::DragOver.
	//
	// Flutter: no counterpart
	IDropTargetDragOver = 4
	// IDropTargetDragLeave is the index of IDropTarget::DragLeave.
	//
	// Flutter: no counterpart
	IDropTargetDragLeave = 5
	// IDropTargetDrop is the index of IDropTarget::Drop.
	//
	// Flutter: no counterpart
	IDropTargetDrop = 6

	// IDropSourceQueryContinueDrag is the index of IDropSource::QueryContinueDrag.
	//
	// Flutter: no counterpart
	IDropSourceQueryContinueDrag = 3
	// IDropSourceGiveFeedback is the index of IDropSource::GiveFeedback.
	//
	// Flutter: no counterpart
	IDropSourceGiveFeedback = 4

	// IDataObjectGetData is the index of IDataObject::GetData.
	//
	// Flutter: no counterpart
	IDataObjectGetData = 3
	// IDataObjectGetDataHere is the index of IDataObject::GetDataHere.
	//
	// Flutter: no counterpart
	IDataObjectGetDataHere = 4
	// IDataObjectQueryGetData is the index of IDataObject::QueryGetData.
	//
	// Flutter: no counterpart
	IDataObjectQueryGetData = 5
	// IDataObjectGetCanonicalFormatEtc is the index of IDataObject::GetCanonicalFormatEtc.
	//
	// Flutter: no counterpart
	IDataObjectGetCanonicalFormatEtc = 6
	// IDataObjectSetData is the index of IDataObject::SetData.
	//
	// Flutter: no counterpart
	IDataObjectSetData = 7
	// IDataObjectEnumFormatEtc is the index of IDataObject::EnumFormatEtc.
	//
	// Flutter: no counterpart
	IDataObjectEnumFormatEtc = 8
	// IDataObjectDAdvise is the index of IDataObject::DAdvise.
	//
	// Flutter: no counterpart
	IDataObjectDAdvise = 9
	// IDataObjectDUnadvise is the index of IDataObject::DUnadvise.
	//
	// Flutter: no counterpart
	IDataObjectDUnadvise = 10
	// IDataObjectEnumDAdvise is the index of IDataObject::EnumDAdvise.
	//
	// Flutter: no counterpart
	IDataObjectEnumDAdvise = 11

	// IEnumFormatEtcNext is the index of IEnumFORMATETC::Next.
	//
	// Flutter: no counterpart
	IEnumFormatEtcNext = 3
	// IEnumFormatEtcSkip is the index of IEnumFORMATETC::Skip.
	//
	// Flutter: no counterpart
	IEnumFormatEtcSkip = 4
	// IEnumFormatEtcReset is the index of IEnumFORMATETC::Reset.
	//
	// Flutter: no counterpart
	IEnumFormatEtcReset = 5
	// IEnumFormatEtcClone is the index of IEnumFORMATETC::Clone.
	//
	// Flutter: no counterpart
	IEnumFormatEtcClone = 6
)
