package uia

// The identifiers of UI Automation: the control types, the properties, the patterns and the events. They
// carry no OS tag so that the layout test checks each value against the mingw-w64 headers (SPEC §25.2).

// ControlType is a UIA_*ControlTypeId.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
type ControlType int32

// PropertyID is a UIA_*PropertyId.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
type PropertyID int32

// PatternID is a UIA_*PatternId.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
type PatternID int32

// EventID is a UIA_*EventId.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
type EventID int32

// The control types (UIA_*ControlTypeId).
const (
	// ControlTypeButton is UIA_ButtonControlTypeId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ControlTypeButton ControlType = 50000
	// ControlTypeCheckBox is UIA_CheckBoxControlTypeId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ControlTypeCheckBox ControlType = 50002
	// ControlTypeComboBox is UIA_ComboBoxControlTypeId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ControlTypeComboBox ControlType = 50003
	// ControlTypeEdit is UIA_EditControlTypeId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ControlTypeEdit ControlType = 50004
	// ControlTypeHyperlink is UIA_HyperlinkControlTypeId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ControlTypeHyperlink ControlType = 50005
	// ControlTypeImage is UIA_ImageControlTypeId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ControlTypeImage ControlType = 50006
	// ControlTypeListItem is UIA_ListItemControlTypeId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ControlTypeListItem ControlType = 50007
	// ControlTypeList is UIA_ListControlTypeId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ControlTypeList ControlType = 50008
	// ControlTypeMenu is UIA_MenuControlTypeId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ControlTypeMenu ControlType = 50009
	// ControlTypeMenuBar is UIA_MenuBarControlTypeId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ControlTypeMenuBar ControlType = 50010
	// ControlTypeMenuItem is UIA_MenuItemControlTypeId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ControlTypeMenuItem ControlType = 50011
	// ControlTypeProgressBar is UIA_ProgressBarControlTypeId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ControlTypeProgressBar ControlType = 50012
	// ControlTypeRadioButton is UIA_RadioButtonControlTypeId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ControlTypeRadioButton ControlType = 50013
	// ControlTypeScrollBar is UIA_ScrollBarControlTypeId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ControlTypeScrollBar ControlType = 50014
	// ControlTypeSlider is UIA_SliderControlTypeId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ControlTypeSlider ControlType = 50015
	// ControlTypeSpinner is UIA_SpinnerControlTypeId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ControlTypeSpinner ControlType = 50016
	// ControlTypeStatusBar is UIA_StatusBarControlTypeId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ControlTypeStatusBar ControlType = 50017
	// ControlTypeTab is UIA_TabControlTypeId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ControlTypeTab ControlType = 50018
	// ControlTypeTabItem is UIA_TabItemControlTypeId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ControlTypeTabItem ControlType = 50019
	// ControlTypeText is UIA_TextControlTypeId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ControlTypeText ControlType = 50020
	// ControlTypeToolBar is UIA_ToolBarControlTypeId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ControlTypeToolBar ControlType = 50021
	// ControlTypeToolTip is UIA_ToolTipControlTypeId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ControlTypeToolTip ControlType = 50022
	// ControlTypeTree is UIA_TreeControlTypeId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ControlTypeTree ControlType = 50023
	// ControlTypeTreeItem is UIA_TreeItemControlTypeId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ControlTypeTreeItem ControlType = 50024
	// ControlTypeCustom is UIA_CustomControlTypeId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ControlTypeCustom ControlType = 50025
	// ControlTypeGroup is UIA_GroupControlTypeId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ControlTypeGroup ControlType = 50026
	// ControlTypeThumb is UIA_ThumbControlTypeId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ControlTypeThumb ControlType = 50027
	// ControlTypeDataGrid is UIA_DataGridControlTypeId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ControlTypeDataGrid ControlType = 50028
	// ControlTypeDataItem is UIA_DataItemControlTypeId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ControlTypeDataItem ControlType = 50029
	// ControlTypeDocument is UIA_DocumentControlTypeId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ControlTypeDocument ControlType = 50030
	// ControlTypeSplitButton is UIA_SplitButtonControlTypeId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ControlTypeSplitButton ControlType = 50031
	// ControlTypeWindow is UIA_WindowControlTypeId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ControlTypeWindow ControlType = 50032
	// ControlTypePane is UIA_PaneControlTypeId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ControlTypePane ControlType = 50033
	// ControlTypeHeader is UIA_HeaderControlTypeId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ControlTypeHeader ControlType = 50034
	// ControlTypeHeaderItem is UIA_HeaderItemControlTypeId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ControlTypeHeaderItem ControlType = 50035
	// ControlTypeTable is UIA_TableControlTypeId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ControlTypeTable ControlType = 50036
	// ControlTypeTitleBar is UIA_TitleBarControlTypeId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ControlTypeTitleBar ControlType = 50037
	// ControlTypeSeparator is UIA_SeparatorControlTypeId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	ControlTypeSeparator ControlType = 50038
)

// The properties (UIA_*PropertyId).
const (
	// PropRuntimeId is UIA_RuntimeIdPropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropRuntimeId PropertyID = 30000
	// PropBoundingRectangle is UIA_BoundingRectanglePropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropBoundingRectangle PropertyID = 30001
	// PropProcessId is UIA_ProcessIdPropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropProcessId PropertyID = 30002
	// PropControlType is UIA_ControlTypePropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropControlType PropertyID = 30003
	// PropLocalizedControlType is UIA_LocalizedControlTypePropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropLocalizedControlType PropertyID = 30004
	// PropName is UIA_NamePropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropName PropertyID = 30005
	// PropAcceleratorKey is UIA_AcceleratorKeyPropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropAcceleratorKey PropertyID = 30006
	// PropAccessKey is UIA_AccessKeyPropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropAccessKey PropertyID = 30007
	// PropHasKeyboardFocus is UIA_HasKeyboardFocusPropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropHasKeyboardFocus PropertyID = 30008
	// PropIsKeyboardFocusable is UIA_IsKeyboardFocusablePropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropIsKeyboardFocusable PropertyID = 30009
	// PropIsEnabled is UIA_IsEnabledPropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropIsEnabled PropertyID = 30010
	// PropAutomationId is UIA_AutomationIdPropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropAutomationId PropertyID = 30011
	// PropClassName is UIA_ClassNamePropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropClassName PropertyID = 30012
	// PropHelpText is UIA_HelpTextPropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropHelpText PropertyID = 30013
	// PropClickablePoint is UIA_ClickablePointPropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropClickablePoint PropertyID = 30014
	// PropCulture is UIA_CulturePropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropCulture PropertyID = 30015
	// PropIsControlElement is UIA_IsControlElementPropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropIsControlElement PropertyID = 30016
	// PropIsContentElement is UIA_IsContentElementPropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropIsContentElement PropertyID = 30017
	// PropLabeledBy is UIA_LabeledByPropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropLabeledBy PropertyID = 30018
	// PropIsPassword is UIA_IsPasswordPropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropIsPassword PropertyID = 30019
	// PropNativeWindowHandle is UIA_NativeWindowHandlePropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropNativeWindowHandle PropertyID = 30020
	// PropItemType is UIA_ItemTypePropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropItemType PropertyID = 30021
	// PropIsOffscreen is UIA_IsOffscreenPropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropIsOffscreen PropertyID = 30022
	// PropOrientation is UIA_OrientationPropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropOrientation PropertyID = 30023
	// PropFrameworkId is UIA_FrameworkIdPropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropFrameworkId PropertyID = 30024
	// PropIsRequiredForForm is UIA_IsRequiredForFormPropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropIsRequiredForForm PropertyID = 30025
	// PropItemStatus is UIA_ItemStatusPropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropItemStatus PropertyID = 30026
	// PropIsExpandCollapsePatternAvailable is UIA_IsExpandCollapsePatternAvailablePropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropIsExpandCollapsePatternAvailable PropertyID = 30028
	// PropIsInvokePatternAvailable is UIA_IsInvokePatternAvailablePropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropIsInvokePatternAvailable PropertyID = 30031
	// PropIsRangeValuePatternAvailable is UIA_IsRangeValuePatternAvailablePropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropIsRangeValuePatternAvailable PropertyID = 30033
	// PropIsScrollPatternAvailable is UIA_IsScrollPatternAvailablePropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropIsScrollPatternAvailable PropertyID = 30034
	// PropIsSelectionItemPatternAvailable is UIA_IsSelectionItemPatternAvailablePropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropIsSelectionItemPatternAvailable PropertyID = 30036
	// PropIsSelectionPatternAvailable is UIA_IsSelectionPatternAvailablePropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropIsSelectionPatternAvailable PropertyID = 30037
	// PropIsTextPatternAvailable is UIA_IsTextPatternAvailablePropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropIsTextPatternAvailable PropertyID = 30040
	// PropIsTogglePatternAvailable is UIA_IsTogglePatternAvailablePropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropIsTogglePatternAvailable PropertyID = 30041
	// PropIsValuePatternAvailable is UIA_IsValuePatternAvailablePropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropIsValuePatternAvailable PropertyID = 30043
	// PropValueValue is UIA_ValueValuePropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropValueValue PropertyID = 30045
	// PropValueIsReadOnly is UIA_ValueIsReadOnlyPropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropValueIsReadOnly PropertyID = 30046
	// PropRangeValueValue is UIA_RangeValueValuePropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropRangeValueValue PropertyID = 30047
	// PropRangeValueIsReadOnly is UIA_RangeValueIsReadOnlyPropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropRangeValueIsReadOnly PropertyID = 30048
	// PropRangeValueMinimum is UIA_RangeValueMinimumPropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropRangeValueMinimum PropertyID = 30049
	// PropRangeValueMaximum is UIA_RangeValueMaximumPropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropRangeValueMaximum PropertyID = 30050
	// PropRangeValueLargeChange is UIA_RangeValueLargeChangePropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropRangeValueLargeChange PropertyID = 30051
	// PropRangeValueSmallChange is UIA_RangeValueSmallChangePropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropRangeValueSmallChange PropertyID = 30052
	// PropScrollHorizontalScrollPercent is UIA_ScrollHorizontalScrollPercentPropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropScrollHorizontalScrollPercent PropertyID = 30053
	// PropScrollHorizontalViewSize is UIA_ScrollHorizontalViewSizePropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropScrollHorizontalViewSize PropertyID = 30054
	// PropScrollVerticalScrollPercent is UIA_ScrollVerticalScrollPercentPropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropScrollVerticalScrollPercent PropertyID = 30055
	// PropScrollVerticalViewSize is UIA_ScrollVerticalViewSizePropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropScrollVerticalViewSize PropertyID = 30056
	// PropScrollHorizontallyScrollable is UIA_ScrollHorizontallyScrollablePropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropScrollHorizontallyScrollable PropertyID = 30057
	// PropScrollVerticallyScrollable is UIA_ScrollVerticallyScrollablePropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropScrollVerticallyScrollable PropertyID = 30058
	// PropSelectionSelection is UIA_SelectionSelectionPropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropSelectionSelection PropertyID = 30059
	// PropSelectionCanSelectMultiple is UIA_SelectionCanSelectMultiplePropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropSelectionCanSelectMultiple PropertyID = 30060
	// PropSelectionIsSelectionRequired is UIA_SelectionIsSelectionRequiredPropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropSelectionIsSelectionRequired PropertyID = 30061
	// PropExpandCollapseExpandCollapseState is UIA_ExpandCollapseExpandCollapseStatePropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropExpandCollapseExpandCollapseState PropertyID = 30070
	// PropSelectionItemIsSelected is UIA_SelectionItemIsSelectedPropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropSelectionItemIsSelected PropertyID = 30079
	// PropToggleToggleState is UIA_ToggleToggleStatePropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropToggleToggleState PropertyID = 30086
	// PropLiveSetting is UIA_LiveSettingPropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropLiveSetting PropertyID = 30135
	// PropFullDescription is UIA_FullDescriptionPropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropFullDescription PropertyID = 30159
	// PropHeadingLevel is UIA_HeadingLevelPropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropHeadingLevel PropertyID = 30173
	// PropIsDialog is UIA_IsDialogPropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropIsDialog PropertyID = 30174
	// PropPositionInSet is UIA_PositionInSetPropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropPositionInSet PropertyID = 30152
	// PropSizeOfSet is UIA_SizeOfSetPropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropSizeOfSet PropertyID = 30153
	// PropLevel is UIA_LevelPropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropLevel PropertyID = 30154
	// PropLandmarkType is UIA_LandmarkTypePropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropLandmarkType PropertyID = 30157
	// PropLocalizedLandmarkType is UIA_LocalizedLandmarkTypePropertyId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PropLocalizedLandmarkType PropertyID = 30158
)

// The patterns (UIA_*PatternId).
const (
	// PatternInvoke is UIA_InvokePatternId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PatternInvoke PatternID = 10000
	// PatternSelection is UIA_SelectionPatternId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PatternSelection PatternID = 10001
	// PatternValue is UIA_ValuePatternId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PatternValue PatternID = 10002
	// PatternRangeValue is UIA_RangeValuePatternId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PatternRangeValue PatternID = 10003
	// PatternScroll is UIA_ScrollPatternId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PatternScroll PatternID = 10004
	// PatternExpandCollapse is UIA_ExpandCollapsePatternId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PatternExpandCollapse PatternID = 10005
	// PatternSelectionItem is UIA_SelectionItemPatternId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PatternSelectionItem PatternID = 10010
	// PatternText is UIA_TextPatternId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PatternText PatternID = 10014
	// PatternToggle is UIA_TogglePatternId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PatternToggle PatternID = 10015
	// PatternScrollItem is UIA_ScrollItemPatternId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PatternScrollItem PatternID = 10017
	// PatternWindow is UIA_WindowPatternId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	PatternWindow PatternID = 10009
)

// The events (UIA_*EventId).
const (
	// EventStructureChanged is UIA_StructureChangedEventId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	EventStructureChanged EventID = 20002
	// EventAutomationPropertyChanged is UIA_AutomationPropertyChangedEventId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	EventAutomationPropertyChanged EventID = 20004
	// EventAutomationFocusChanged is UIA_AutomationFocusChangedEventId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	EventAutomationFocusChanged EventID = 20005
	// EventInvokeInvoked is UIA_Invoke_InvokedEventId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	EventInvokeInvoked EventID = 20009
	// EventSelectionItemElementSelected is UIA_SelectionItem_ElementSelectedEventId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	EventSelectionItemElementSelected EventID = 20012
	// EventSelectionInvalidated is UIA_Selection_InvalidatedEventId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	EventSelectionInvalidated EventID = 20013
	// EventTextTextSelectionChanged is UIA_Text_TextSelectionChangedEventId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	EventTextTextSelectionChanged EventID = 20014
	// EventTextTextChanged is UIA_Text_TextChangedEventId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	EventTextTextChanged EventID = 20015
	// EventWindowWindowOpened is UIA_Window_WindowOpenedEventId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	EventWindowWindowOpened EventID = 20016
	// EventWindowWindowClosed is UIA_Window_WindowClosedEventId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	EventWindowWindowClosed EventID = 20017
	// EventLiveRegionChanged is UIA_LiveRegionChangedEventId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	EventLiveRegionChanged EventID = 20024
	// EventSystemAlert is UIA_SystemAlertEventId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	EventSystemAlert EventID = 20023
	// EventNotification is UIA_NotificationEventId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	EventNotification EventID = 20035
	// EventLayoutInvalidated is UIA_LayoutInvalidatedEventId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	EventLayoutInvalidated EventID = 20008
	// EventMenuOpened is UIA_MenuOpenedEventId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	EventMenuOpened EventID = 20003
	// EventMenuClosed is UIA_MenuClosedEventId.
	//
	// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc
	EventMenuClosed EventID = 20007
)
