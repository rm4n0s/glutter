package win32

// The constants of the input methods of step 28.5 (SPEC §25.7): the messages of the IME, the flags of
// WM_IME_COMPOSITION, the styles of the composition and candidate windows and the parameters of the
// IMM32 calls. They carry no OS tag so that the layout test checks them against the headers
// (SPEC §25.2). The name is the Go spelling of the macro.

// The messages of the input method editor.
const (
	// WMIMEStartComposition is WM_IME_STARTCOMPOSITION.
	//
	// Flutter: no counterpart
	WMIMEStartComposition = 0x010D
	// WMIMEEndComposition is WM_IME_ENDCOMPOSITION.
	//
	// Flutter: no counterpart
	WMIMEEndComposition = 0x010E
	// WMIMEComposition is WM_IME_COMPOSITION.
	//
	// Flutter: no counterpart
	WMIMEComposition = 0x010F
	// WMIMESetContext is WM_IME_SETCONTEXT.
	//
	// Flutter: no counterpart
	WMIMESetContext = 0x0281
	// WMIMENotify is WM_IME_NOTIFY.
	//
	// Flutter: no counterpart
	WMIMENotify = 0x0282
	// WMIMEChar is WM_IME_CHAR.
	//
	// Flutter: no counterpart
	WMIMEChar = 0x0286
	// WMIMERequest is WM_IME_REQUEST.
	//
	// Flutter: no counterpart
	WMIMERequest = 0x0288
)

// The flags of WM_IME_COMPOSITION and the indices of ImmGetCompositionStringW.
const (
	// GCSCompReadStr is GCS_COMPREADSTR.
	//
	// Flutter: no counterpart
	GCSCompReadStr = 0x0001
	// GCSCompStr is GCS_COMPSTR.
	//
	// Flutter: no counterpart
	GCSCompStr = 0x0008
	// GCSCompAttr is GCS_COMPATTR.
	//
	// Flutter: no counterpart
	GCSCompAttr = 0x0010
	// GCSCompClause is GCS_COMPCLAUSE.
	//
	// Flutter: no counterpart
	GCSCompClause = 0x0020
	// GCSCursorPos is GCS_CURSORPOS.
	//
	// Flutter: no counterpart
	GCSCursorPos = 0x0080
	// GCSDeltaStart is GCS_DELTASTART.
	//
	// Flutter: no counterpart
	GCSDeltaStart = 0x0100
	// GCSResultStr is GCS_RESULTSTR.
	//
	// Flutter: no counterpart
	GCSResultStr = 0x0800
)

// The styles of the composition and candidate windows.
const (
	// CFSDefault is CFS_DEFAULT.
	//
	// Flutter: no counterpart
	CFSDefault = 0x0000
	// CFSRect is CFS_RECT.
	//
	// Flutter: no counterpart
	CFSRect = 0x0001
	// CFSPoint is CFS_POINT.
	//
	// Flutter: no counterpart
	CFSPoint = 0x0002
	// CFSForcePosition is CFS_FORCE_POSITION.
	//
	// Flutter: no counterpart
	CFSForcePosition = 0x0020
	// CFSCandidatePos is CFS_CANDIDATEPOS.
	//
	// Flutter: no counterpart
	CFSCandidatePos = 0x0040
	// CFSExclude is CFS_EXCLUDE.
	//
	// Flutter: no counterpart
	CFSExclude = 0x0080
)

// The parameters of ImmAssociateContextEx, ImmNotifyIME, ImmSetCompositionStringW and WM_IME_SETCONTEXT.
const (
	// IACEDefault is IACE_DEFAULT.
	//
	// Flutter: no counterpart
	IACEDefault = 0x0010
	// IACEChildren is IACE_CHILDREN.
	//
	// Flutter: no counterpart
	IACEChildren = 0x0001
	// NICompositionStr is NI_COMPOSITIONSTR.
	//
	// Flutter: no counterpart
	NICompositionStr = 0x0015
	// NICloseCandidate is NI_CLOSECANDIDATE.
	//
	// Flutter: no counterpart
	NICloseCandidate = 0x0011
	// CPSComplete is CPS_COMPLETE.
	//
	// Flutter: no counterpart
	CPSComplete = 0x0001
	// CPSCancel is CPS_CANCEL.
	//
	// Flutter: no counterpart
	CPSCancel = 0x0004
	// SCSSetStr is SCS_SETSTR.
	//
	// Flutter: no counterpart
	SCSSetStr = 0x0009
	// ISCShowUICompositionWindow is ISC_SHOWUICOMPOSITIONWINDOW.
	//
	// Flutter: no counterpart
	ISCShowUICompositionWindow = 0x80000000
	// ISCShowUIAllCandidateWindow is ISC_SHOWUIALLCANDIDATEWINDOW.
	//
	// Flutter: no counterpart
	ISCShowUIAllCandidateWindow = 0x0000000F
	// UnicodeNoChar is UNICODE_NOCHAR.
	//
	// Flutter: no counterpart
	UnicodeNoChar = 0xFFFF
)
