package uia

// The enumerations of the UI Automation provider interfaces, and the HRESULT codes that providers
// answer. They carry no OS tag so that the layout test checks them against the mingw-w64 headers.

// The directions of IRawElementProviderFragment::Navigate (NavigateDirection).
const (
	// NavigateParent is NavigateDirection_Parent.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	NavigateParent = 0
	// NavigateNextSibling is NavigateDirection_NextSibling.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	NavigateNextSibling = 1
	// NavigatePreviousSibling is NavigateDirection_PreviousSibling.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	NavigatePreviousSibling = 2
	// NavigateFirstChild is NavigateDirection_FirstChild.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	NavigateFirstChild = 3
	// NavigateLastChild is NavigateDirection_LastChild.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	NavigateLastChild = 4
)

// The options of a provider (ProviderOptions), a bit set.
const (
	// ProviderServerSide is ProviderOptions_ServerSideProvider.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ProviderServerSide = 0x2
	// ProviderUseCOMThreading is ProviderOptions_UseComThreading.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ProviderUseCOMThreading = 0x20
)

// The kinds of change of the tree (StructureChangeType).
const (
	// StructureChildAdded is StructureChangeType_ChildAdded.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	StructureChildAdded = 0
	// StructureChildRemoved is StructureChangeType_ChildRemoved.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	StructureChildRemoved = 1
	// StructureChildrenInvalidated is StructureChangeType_ChildrenInvalidated.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	StructureChildrenInvalidated = 2
	// StructureChildrenReordered is StructureChangeType_ChildrenReordered.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	StructureChildrenReordered = 5
)

// The states of the Toggle pattern (ToggleState).
const (
	// ToggleOff is ToggleState_Off.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ToggleOff = 0
	// ToggleOn is ToggleState_On.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ToggleOn = 1
	// ToggleIndeterminate is ToggleState_Indeterminate.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ToggleIndeterminate = 2
)

// The states of the ExpandCollapse pattern (ExpandCollapseState).
const (
	// ExpandCollapseCollapsed is ExpandCollapseState_Collapsed.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ExpandCollapseCollapsed = 0
	// ExpandCollapseExpanded is ExpandCollapseState_Expanded.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ExpandCollapseExpanded = 1
	// ExpandCollapseLeafNode is ExpandCollapseState_LeafNode.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ExpandCollapseLeafNode = 3
)

// The amounts of the Scroll pattern (ScrollAmount).
const (
	// ScrollLargeDecrement is ScrollAmount_LargeDecrement.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ScrollLargeDecrement = 0
	// ScrollSmallDecrement is ScrollAmount_SmallDecrement.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ScrollSmallDecrement = 1
	// ScrollNoAmount is ScrollAmount_NoAmount.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ScrollNoAmount = 2
	// ScrollLargeIncrement is ScrollAmount_LargeIncrement.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ScrollLargeIncrement = 3
	// ScrollSmallIncrement is ScrollAmount_SmallIncrement.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ScrollSmallIncrement = 4
)

// The units of text (TextUnit).
const (
	// TextUnitCharacter is TextUnit_Character.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_textrangeprovider_win.cc
	TextUnitCharacter = 0
	// TextUnitFormat is TextUnit_Format.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_textrangeprovider_win.cc
	TextUnitFormat = 1
	// TextUnitWord is TextUnit_Word.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_textrangeprovider_win.cc
	TextUnitWord = 2
	// TextUnitLine is TextUnit_Line.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_textrangeprovider_win.cc
	TextUnitLine = 3
	// TextUnitParagraph is TextUnit_Paragraph.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_textrangeprovider_win.cc
	TextUnitParagraph = 4
	// TextUnitPage is TextUnit_Page.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_textrangeprovider_win.cc
	TextUnitPage = 5
	// TextUnitDocument is TextUnit_Document.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_textrangeprovider_win.cc
	TextUnitDocument = 6
)

// The ends of a text range (TextPatternRangeEndpoint).
const (
	// EndpointStart is TextPatternRangeEndpoint_Start.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_textrangeprovider_win.cc
	EndpointStart = 0
	// EndpointEnd is TextPatternRangeEndpoint_End.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_textrangeprovider_win.cc
	EndpointEnd = 1
)

// The live settings of an element (LiveSetting).
const (
	// LiveOff is LiveSetting Off.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	LiveOff = 0
	// LivePolite is LiveSetting Polite.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	LivePolite = 1
)

// SupportedTextSelectionSingle is SupportedTextSelection_Single: a text field has one selection.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_textprovider_win.cc
const SupportedTextSelectionSingle = 1

// The results that a provider answers besides S_OK, E_NOTIMPL and E_FAIL, as the unsigned numbers
// that the headers write.
const (
	// ErrElementNotEnabled is UIA_E_ELEMENTNOTENABLED: the element is disabled.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ErrElementNotEnabled uint32 = 0x80040200
	// ErrElementNotAvailable is UIA_E_ELEMENTNOTAVAILABLE: the element is gone.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ErrElementNotAvailable uint32 = 0x80040201
	// ErrNotSupported is UIA_E_NOTSUPPORTED: the element does not support the operation.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ErrNotSupported uint32 = 0x80040204
	// ErrInvalidOperation is UIA_E_INVALIDOPERATION.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ErrInvalidOperation uint32 = 0x80131509
)

// AppendRuntimeID is UiaAppendRuntimeId: the first number of a runtime ID that UI Automation completes
// with the identity of the window.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
const AppendRuntimeID = 3

// RootObjectID is UiaRootObjectId, the object ID of the WM_GETOBJECT with which a client asks for the
// UI Automation provider of a window.
//
// Flutter: shell/platform/windows/flutter_window.cc
const RootObjectID int32 = -25
