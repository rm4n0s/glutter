package uiatest

import (
	"github.com/rm4n0s/glutter/internal/platform/windows/uia"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// The interfaces of the UI Automation client that the test client calls or implements: the IID of each and the
// names of its methods in vtable order after IUnknown's (see package uia, Interface). The layout test checks them
// against the mingw-w64 headers.

// CLSIDAutomation is CLSID_CUIAutomation, the class of the client.
//
// Flutter: no counterpart
var CLSIDAutomation = win32.MustParseGUID("{ff48dba4-60ef-4201-aa87-54103eef594e}")

// IfaceAutomation is IUIAutomation.
//
// Flutter: no counterpart
var IfaceAutomation = uia.Interface{Name: "IUIAutomation", IID: win32.MustParseGUID("{30cbe57d-d9d0-452a-ab13-7ac5ac4825ee}"), Methods: []string{"CompareElements", "CompareRuntimeIds", "GetRootElement", "ElementFromHandle", "ElementFromPoint", "GetFocusedElement", "GetRootElementBuildCache", "ElementFromHandleBuildCache", "ElementFromPointBuildCache", "GetFocusedElementBuildCache", "CreateTreeWalker", "get_ControlViewWalker", "get_ContentViewWalker", "get_RawViewWalker", "get_RawViewCondition", "get_ControlViewCondition", "get_ContentViewCondition", "CreateCacheRequest", "CreateTrueCondition", "CreateFalseCondition", "CreatePropertyCondition", "CreatePropertyConditionEx", "CreateAndCondition", "CreateAndConditionFromArray", "CreateAndConditionFromNativeArray", "CreateOrCondition", "CreateOrConditionFromArray", "CreateOrConditionFromNativeArray", "CreateNotCondition", "AddAutomationEventHandler", "RemoveAutomationEventHandler", "AddPropertyChangedEventHandlerNativeArray", "AddPropertyChangedEventHandler", "RemovePropertyChangedEventHandler", "AddStructureChangedEventHandler", "RemoveStructureChangedEventHandler", "AddFocusChangedEventHandler", "RemoveFocusChangedEventHandler", "RemoveAllEventHandlers", "IntNativeArrayToSafeArray", "IntSafeArrayToNativeArray", "RectToVariant", "VariantToRect", "SafeArrayToRectNativeArray", "CreateProxyFactoryEntry", "get_ProxyFactoryMapping", "GetPropertyProgrammaticName", "GetPatternProgrammaticName", "PollForPotentialSupportedPatterns", "PollForPotentialSupportedProperties", "CheckNotSupported", "get_ReservedNotSupportedValue", "get_ReservedMixedAttributeValue", "ElementFromIAccessible", "ElementFromIAccessibleBuildCache"}}

// IfaceElement is IUIAutomationElement.
//
// Flutter: no counterpart
var IfaceElement = uia.Interface{Name: "IUIAutomationElement", IID: win32.MustParseGUID("{d22108aa-8ac5-49a5-837b-37bbb3d7591e}"), Methods: []string{"SetFocus", "GetRuntimeId", "FindFirst", "FindAll", "FindFirstBuildCache", "FindAllBuildCache", "BuildUpdatedCache", "GetCurrentPropertyValue", "GetCurrentPropertyValueEx", "GetCachedPropertyValue", "GetCachedPropertyValueEx", "GetCurrentPatternAs", "GetCachedPatternAs", "GetCurrentPattern", "GetCachedPattern", "GetCachedParent", "GetCachedChildren", "get_CurrentProcessId", "get_CurrentControlType", "get_CurrentLocalizedControlType", "get_CurrentName", "get_CurrentAcceleratorKey", "get_CurrentAccessKey", "get_CurrentHasKeyboardFocus", "get_CurrentIsKeyboardFocusable", "get_CurrentIsEnabled", "get_CurrentAutomationId", "get_CurrentClassName", "get_CurrentHelpText", "get_CurrentCulture", "get_CurrentIsControlElement", "get_CurrentIsContentElement", "get_CurrentIsPassword", "get_CurrentNativeWindowHandle", "get_CurrentItemType", "get_CurrentIsOffscreen", "get_CurrentOrientation", "get_CurrentFrameworkId", "get_CurrentIsRequiredForForm", "get_CurrentItemStatus", "get_CurrentBoundingRectangle", "get_CurrentLabeledBy", "get_CurrentAriaRole", "get_CurrentAriaProperties", "get_CurrentIsDataValidForForm", "get_CurrentControllerFor", "get_CurrentDescribedBy", "get_CurrentFlowsTo", "get_CurrentProviderDescription", "get_CachedProcessId", "get_CachedControlType", "get_CachedLocalizedControlType", "get_CachedName", "get_CachedAcceleratorKey", "get_CachedAccessKey", "get_CachedHasKeyboardFocus", "get_CachedIsKeyboardFocusable", "get_CachedIsEnabled", "get_CachedAutomationId", "get_CachedClassName", "get_CachedHelpText", "get_CachedCulture", "get_CachedIsControlElement", "get_CachedIsContentElement", "get_CachedIsPassword", "get_CachedNativeWindowHandle", "get_CachedItemType", "get_CachedIsOffscreen", "get_CachedOrientation", "get_CachedFrameworkId", "get_CachedIsRequiredForForm", "get_CachedItemStatus", "get_CachedBoundingRectangle", "get_CachedLabeledBy", "get_CachedAriaRole", "get_CachedAriaProperties", "get_CachedIsDataValidForForm", "get_CachedControllerFor", "get_CachedDescribedBy", "get_CachedFlowsTo", "get_CachedProviderDescription", "GetClickablePoint"}}

// IfaceTreeWalker is IUIAutomationTreeWalker.
//
// Flutter: no counterpart
var IfaceTreeWalker = uia.Interface{Name: "IUIAutomationTreeWalker", IID: win32.MustParseGUID("{4042c624-389c-4afc-a630-9df854a541fc}"), Methods: []string{"GetParentElement", "GetFirstChildElement", "GetLastChildElement", "GetNextSiblingElement", "GetPreviousSiblingElement", "NormalizeElement", "GetParentElementBuildCache", "GetFirstChildElementBuildCache", "GetLastChildElementBuildCache", "GetNextSiblingElementBuildCache", "GetPreviousSiblingElementBuildCache", "NormalizeElementBuildCache", "get_Condition"}}

// IfaceElementArray is IUIAutomationElementArray.
//
// Flutter: no counterpart
var IfaceElementArray = uia.Interface{Name: "IUIAutomationElementArray", IID: win32.MustParseGUID("{14314595-b4bc-4055-95f2-58f2e42c9855}"), Methods: []string{"get_Length", "GetElement"}}

// IfaceInvoke is IUIAutomationInvokePattern.
//
// Flutter: no counterpart
var IfaceInvoke = uia.Interface{Name: "IUIAutomationInvokePattern", IID: win32.MustParseGUID("{fb377fbe-8ea6-46d5-9c73-6499642d3059}"), Methods: []string{"Invoke"}}

// IfaceToggle is IUIAutomationTogglePattern.
//
// Flutter: no counterpart
var IfaceToggle = uia.Interface{Name: "IUIAutomationTogglePattern", IID: win32.MustParseGUID("{94cf8058-9b8d-4ab9-8bfd-4cd0a33c8c70}"), Methods: []string{"Toggle", "get_CurrentToggleState", "get_CachedToggleState"}}

// IfaceValue is IUIAutomationValuePattern.
//
// Flutter: no counterpart
var IfaceValue = uia.Interface{Name: "IUIAutomationValuePattern", IID: win32.MustParseGUID("{a94cd8b1-0844-4cd6-9d2d-640537ab39e9}"), Methods: []string{"SetValue", "get_CurrentValue", "get_CurrentIsReadOnly", "get_CachedValue", "get_CachedIsReadOnly"}}

// IfaceRangeValue is IUIAutomationRangeValuePattern.
//
// Flutter: no counterpart
var IfaceRangeValue = uia.Interface{Name: "IUIAutomationRangeValuePattern", IID: win32.MustParseGUID("{59213f4f-7346-49e5-b120-80555987a148}"), Methods: []string{"SetValue", "get_CurrentValue", "get_CurrentIsReadOnly", "get_CurrentMaximum", "get_CurrentMinimum", "get_CurrentLargeChange", "get_CurrentSmallChange", "get_CachedValue", "get_CachedIsReadOnly", "get_CachedMaximum", "get_CachedMinimum", "get_CachedLargeChange", "get_CachedSmallChange"}}

// IfaceScroll is IUIAutomationScrollPattern.
//
// Flutter: no counterpart
var IfaceScroll = uia.Interface{Name: "IUIAutomationScrollPattern", IID: win32.MustParseGUID("{88f4d42a-e881-459d-a77c-73bbbb7e02dc}"), Methods: []string{"Scroll", "SetScrollPercent", "get_CurrentHorizontalScrollPercent", "get_CurrentVerticalScrollPercent", "get_CurrentHorizontalViewSize", "get_CurrentVerticalViewSize", "get_CurrentHorizontallyScrollable", "get_CurrentVerticallyScrollable", "get_CachedHorizontalScrollPercent", "get_CachedVerticalScrollPercent", "get_CachedHorizontalViewSize", "get_CachedVerticalViewSize", "get_CachedHorizontallyScrollable", "get_CachedVerticallyScrollable"}}

// IfaceExpandCollapse is IUIAutomationExpandCollapsePattern.
//
// Flutter: no counterpart
var IfaceExpandCollapse = uia.Interface{Name: "IUIAutomationExpandCollapsePattern", IID: win32.MustParseGUID("{619be086-1f4e-4ee4-bafa-210128738730}"), Methods: []string{"Expand", "Collapse", "get_CurrentExpandCollapseState", "get_CachedExpandCollapseState"}}

// IfaceSelectionItem is IUIAutomationSelectionItemPattern.
//
// Flutter: no counterpart
var IfaceSelectionItem = uia.Interface{Name: "IUIAutomationSelectionItemPattern", IID: win32.MustParseGUID("{a8efa66a-0fda-421a-9194-38021f3578ea}"), Methods: []string{"Select", "AddToSelection", "RemoveFromSelection", "get_CurrentIsSelected", "get_CurrentSelectionContainer", "get_CachedIsSelected", "get_CachedSelectionContainer"}}

// IfaceSelection is IUIAutomationSelectionPattern.
//
// Flutter: no counterpart
var IfaceSelection = uia.Interface{Name: "IUIAutomationSelectionPattern", IID: win32.MustParseGUID("{5ed5202e-b2ac-47a6-b638-4b0bf140d78e}"), Methods: []string{"GetCurrentSelection", "get_CurrentCanSelectMultiple", "get_CurrentIsSelectionRequired", "GetCachedSelection", "get_CachedCanSelectMultiple", "get_CachedIsSelectionRequired"}}

// IfaceText is IUIAutomationTextPattern.
//
// Flutter: no counterpart
var IfaceText = uia.Interface{Name: "IUIAutomationTextPattern", IID: win32.MustParseGUID("{32eba289-3583-42c9-9c59-3b6d9a1e9b6a}"), Methods: []string{"RangeFromPoint", "RangeFromChild", "GetSelection", "GetVisibleRanges", "get_DocumentRange", "get_SupportedTextSelection"}}

// IfaceTextRange is IUIAutomationTextRange.
//
// Flutter: no counterpart
var IfaceTextRange = uia.Interface{Name: "IUIAutomationTextRange", IID: win32.MustParseGUID("{a543cc6a-f4ae-494b-8239-c814481187a8}"), Methods: []string{"Clone", "Compare", "CompareEndpoints", "ExpandToEnclosingUnit", "FindAttribute", "FindText", "GetAttributeValue", "GetBoundingRectangles", "GetEnclosingElement", "GetText", "Move", "MoveEndpointByUnit", "MoveEndpointByRange", "Select", "AddToSelection", "RemoveFromSelection", "ScrollIntoView", "GetChildren"}}

// IfaceTextRangeArray is IUIAutomationTextRangeArray.
//
// Flutter: no counterpart
var IfaceTextRangeArray = uia.Interface{Name: "IUIAutomationTextRangeArray", IID: win32.MustParseGUID("{ce4ae76a-e717-4c98-81ea-47371d028eb6}"), Methods: []string{"get_Length", "GetElement"}}

// IfaceFocusHandler is IUIAutomationFocusChangedEventHandler.
//
// Flutter: no counterpart
var IfaceFocusHandler = uia.Interface{Name: "IUIAutomationFocusChangedEventHandler", IID: win32.MustParseGUID("{c270f6b5-5c69-4290-9745-7a7f97169468}"), Methods: []string{"HandleFocusChangedEvent"}}

// IfacePropertyHandler is IUIAutomationPropertyChangedEventHandler.
//
// Flutter: no counterpart
var IfacePropertyHandler = uia.Interface{Name: "IUIAutomationPropertyChangedEventHandler", IID: win32.MustParseGUID("{40cd37d4-c756-4b0c-8c6f-bddfeeb13b50}"), Methods: []string{"HandlePropertyChangedEvent"}}

// IfaceEventHandler is IUIAutomationEventHandler.
//
// Flutter: no counterpart
var IfaceEventHandler = uia.Interface{Name: "IUIAutomationEventHandler", IID: win32.MustParseGUID("{146c3c17-f12e-4e22-8c27-f894b9b79c69}"), Methods: []string{"HandleAutomationEvent"}}

// IfaceStructureHandler is IUIAutomationStructureChangedEventHandler.
//
// Flutter: no counterpart
var IfaceStructureHandler = uia.Interface{Name: "IUIAutomationStructureChangedEventHandler", IID: win32.MustParseGUID("{e81d1b4e-11c5-42f8-9754-e7036c79f054}"), Methods: []string{"HandleStructureChangedEvent"}}

// Interfaces lists every interface of the client.
//
// Flutter: no counterpart
var Interfaces = []uia.Interface{IfaceAutomation, IfaceElement, IfaceTreeWalker, IfaceElementArray, IfaceInvoke, IfaceToggle, IfaceValue, IfaceRangeValue, IfaceScroll, IfaceExpandCollapse, IfaceSelectionItem, IfaceSelection, IfaceText, IfaceTextRange, IfaceTextRangeArray, IfaceFocusHandler, IfacePropertyHandler, IfaceEventHandler, IfaceStructureHandler}
