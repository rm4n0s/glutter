package uia

import "github.com/rm4n0s/glutter/internal/platform/windows/win32"

// The interfaces of the providers that Go implements (SPEC §25.2, §25.11): the IID of each and the
// names of its methods in vtable order, after the three of IUnknown. The glue (provider_windows.go)
// builds each vtable from these lists and fails at startup if a method has no function; the layout test
// checks every name and position against the mingw-w64 headers, so a wrong order is a compile error.

// Interface describes a COM interface of a provider.
//
// Flutter: no counterpart
type Interface struct {
	// Name is the name in the header, such as "IRawElementProviderSimple" (its vtable is Name + "Vtbl").
	Name string
	// IID is the interface ID.
	IID win32.GUID
	// Methods are the names of the methods in the vtable, after IUnknown's.
	Methods []string
}

// The interfaces.
var (
	// IfaceSimple is IRawElementProviderSimple.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc AXPlatformNodeWin
	IfaceSimple = Interface{"IRawElementProviderSimple", win32.MustParseGUID("{d6dd68d1-86fd-4332-8666-9abedea2d24c}"),
		[]string{"get_ProviderOptions", "GetPatternProvider", "GetPropertyValue", "get_HostRawElementProvider"}}
	// IfaceFragment is IRawElementProviderFragment.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc AXPlatformNodeWin
	IfaceFragment = Interface{"IRawElementProviderFragment", win32.MustParseGUID("{f7063da8-8359-439c-9297-bbc5299a7d87}"),
		[]string{"Navigate", "GetRuntimeId", "get_BoundingRectangle", "GetEmbeddedFragmentRoots", "SetFocus", "get_FragmentRoot"}}
	// IfaceFragmentRoot is IRawElementProviderFragmentRoot.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_fragment_root_win.cc AXFragmentRootWin
	IfaceFragmentRoot = Interface{"IRawElementProviderFragmentRoot", win32.MustParseGUID("{620ce2a5-ab8f-40a9-86cb-de3c75599b58}"),
		[]string{"ElementProviderFromPoint", "GetFocus"}}
	// IfaceInvoke is IInvokeProvider.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc AXPlatformNodeWin
	IfaceInvoke = Interface{"IInvokeProvider", win32.MustParseGUID("{54fcb24b-e18e-47a2-b4d3-eccbe77599a2}"),
		[]string{"Invoke"}}
	// IfaceToggle is IToggleProvider.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc AXPlatformNodeWin
	IfaceToggle = Interface{"IToggleProvider", win32.MustParseGUID("{56d00bd0-c4f4-433c-a836-1a52a57e0892}"),
		[]string{"Toggle", "get_ToggleState"}}
	// IfaceValue is IValueProvider.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc AXPlatformNodeWin
	IfaceValue = Interface{"IValueProvider", win32.MustParseGUID("{c7935180-6fb3-4201-b174-7df73adbf64a}"),
		[]string{"SetValue", "get_Value", "get_IsReadOnly"}}
	// IfaceRangeValue is IRangeValueProvider.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc AXPlatformNodeWin
	IfaceRangeValue = Interface{"IRangeValueProvider", win32.MustParseGUID("{36dc7aef-33e6-4691-afe1-2be7274b3d33}"),
		[]string{"SetValue", "get_Value", "get_IsReadOnly", "get_Maximum", "get_Minimum", "get_LargeChange", "get_SmallChange"}}
	// IfaceScroll is IScrollProvider.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc AXPlatformNodeWin
	IfaceScroll = Interface{"IScrollProvider", win32.MustParseGUID("{b38b8077-1fc3-42a5-8cae-d40c2215055a}"),
		[]string{"Scroll", "SetScrollPercent", "get_HorizontalScrollPercent", "get_VerticalScrollPercent", "get_HorizontalViewSize",
			"get_VerticalViewSize", "get_HorizontallyScrollable", "get_VerticallyScrollable"}}
	// IfaceExpandCollapse is IExpandCollapseProvider.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc AXPlatformNodeWin
	IfaceExpandCollapse = Interface{"IExpandCollapseProvider", win32.MustParseGUID("{d847d3a5-cab0-4a98-8c32-ecb45c59ad24}"),
		[]string{"Expand", "Collapse", "get_ExpandCollapseState"}}
	// IfaceSelection is ISelectionProvider.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc AXPlatformNodeWin
	IfaceSelection = Interface{"ISelectionProvider", win32.MustParseGUID("{fb8b03af-3bdf-48d4-bd36-1a65793be168}"),
		[]string{"GetSelection", "get_CanSelectMultiple", "get_IsSelectionRequired"}}
	// IfaceSelectionItem is ISelectionItemProvider.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc AXPlatformNodeWin
	IfaceSelectionItem = Interface{"ISelectionItemProvider", win32.MustParseGUID("{2acad808-b2d4-452d-a407-91ff1ad167b2}"),
		[]string{"Select", "AddToSelection", "RemoveFromSelection", "get_IsSelected", "get_SelectionContainer"}}
	// IfaceText is ITextProvider.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_textprovider_win.cc AXPlatformNodeTextProviderWin
	IfaceText = Interface{"ITextProvider", win32.MustParseGUID("{3589c92c-63f3-4367-99bb-ada653b77cf2}"),
		[]string{"GetSelection", "GetVisibleRanges", "RangeFromChild", "RangeFromPoint", "get_DocumentRange", "get_SupportedTextSelection"}}
	// IfaceTextRange is ITextRangeProvider.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_textrangeprovider_win.cc AXPlatformNodeTextRangeProviderWin
	IfaceTextRange = Interface{"ITextRangeProvider", win32.MustParseGUID("{5347ad7b-c355-46f8-aff5-909033582f63}"),
		[]string{"Clone", "Compare", "CompareEndpoints", "ExpandToEnclosingUnit", "FindAttribute", "FindText", "GetAttributeValue",
			"GetBoundingRectangles", "GetEnclosingElement", "GetText", "Move", "MoveEndpointByUnit", "MoveEndpointByRange", "Select",
			"AddToSelection", "RemoveFromSelection", "ScrollIntoView", "GetChildren"}}
)

// Interfaces lists every interface the providers implement.
//
// Flutter: no counterpart
var Interfaces = []Interface{
	IfaceSimple, IfaceFragment, IfaceFragmentRoot, IfaceInvoke, IfaceToggle, IfaceValue, IfaceRangeValue,
	IfaceScroll, IfaceExpandCollapse, IfaceSelection, IfaceSelectionItem, IfaceText, IfaceTextRange,
}

// Index is the position of the method in the vtable, counted in pointers from its start (IUnknown's
// three methods come first), or -1.
//
// Flutter: no counterpart
func (i Interface) Index(method string) int {
	for k, m := range i.Methods {
		if m == method {
			return 3 + k
		}
	}
	return -1
}
