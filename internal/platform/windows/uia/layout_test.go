//go:build !windows

package uia

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/platform/windows/win32/layouttest"
)

// TestLayoutAgainstTheHeaders checks the vtables of the provider interfaces that Go implements (the
// names and the positions of the methods), the identifiers and the enumerations against the mingw-w64
// headers (SPEC §25.2). It runs on the Linux host and is skipped, with the install command, when the
// compiler is missing.
func TestLayoutAgainstTheHeaders(t *testing.T) {
	var vtables []layouttest.VTable
	for _, iface := range Interfaces {
		v := layouttest.VTable{CName: iface.Name + "Vtbl"}
		for i, m := range iface.Methods {
			v.Methods = append(v.Methods, layouttest.Method{Name: m, Index: 3 + i})
		}
		vtables = append(vtables, v)
	}
	var consts []layouttest.Const
	for _, c := range idTable() {
		consts = append(consts, layouttest.Const{Name: c.name, Value: uint32(c.value)})
	}
	u := func(v int) uint32 { return uint32(int32(v)) }
	consts = append(consts,
		layouttest.Const{Name: "NavigateDirection_Parent", Value: u(NavigateParent)},
		layouttest.Const{Name: "NavigateDirection_NextSibling", Value: u(NavigateNextSibling)},
		layouttest.Const{Name: "NavigateDirection_PreviousSibling", Value: u(NavigatePreviousSibling)},
		layouttest.Const{Name: "NavigateDirection_FirstChild", Value: u(NavigateFirstChild)},
		layouttest.Const{Name: "NavigateDirection_LastChild", Value: u(NavigateLastChild)},
		layouttest.Const{Name: "ProviderOptions_ServerSideProvider", Value: u(ProviderServerSide)},
		layouttest.Const{Name: "ProviderOptions_UseComThreading", Value: u(ProviderUseCOMThreading)},
		layouttest.Const{Name: "StructureChangeType_ChildAdded", Value: u(StructureChildAdded)},
		layouttest.Const{Name: "StructureChangeType_ChildRemoved", Value: u(StructureChildRemoved)},
		layouttest.Const{Name: "StructureChangeType_ChildrenInvalidated", Value: u(StructureChildrenInvalidated)},
		layouttest.Const{Name: "StructureChangeType_ChildrenReordered", Value: u(StructureChildrenReordered)},
		layouttest.Const{Name: "ToggleState_Off", Value: u(ToggleOff)},
		layouttest.Const{Name: "ToggleState_On", Value: u(ToggleOn)},
		layouttest.Const{Name: "ToggleState_Indeterminate", Value: u(ToggleIndeterminate)},
		layouttest.Const{Name: "ExpandCollapseState_Collapsed", Value: u(ExpandCollapseCollapsed)},
		layouttest.Const{Name: "ExpandCollapseState_Expanded", Value: u(ExpandCollapseExpanded)},
		layouttest.Const{Name: "ExpandCollapseState_LeafNode", Value: u(ExpandCollapseLeafNode)},
		layouttest.Const{Name: "ScrollAmount_LargeDecrement", Value: u(ScrollLargeDecrement)},
		layouttest.Const{Name: "ScrollAmount_SmallDecrement", Value: u(ScrollSmallDecrement)},
		layouttest.Const{Name: "ScrollAmount_NoAmount", Value: u(ScrollNoAmount)},
		layouttest.Const{Name: "ScrollAmount_LargeIncrement", Value: u(ScrollLargeIncrement)},
		layouttest.Const{Name: "ScrollAmount_SmallIncrement", Value: u(ScrollSmallIncrement)},
		layouttest.Const{Name: "TextUnit_Character", Value: u(TextUnitCharacter)},
		layouttest.Const{Name: "TextUnit_Format", Value: u(TextUnitFormat)},
		layouttest.Const{Name: "TextUnit_Word", Value: u(TextUnitWord)},
		layouttest.Const{Name: "TextUnit_Line", Value: u(TextUnitLine)},
		layouttest.Const{Name: "TextUnit_Paragraph", Value: u(TextUnitParagraph)},
		layouttest.Const{Name: "TextUnit_Page", Value: u(TextUnitPage)},
		layouttest.Const{Name: "TextUnit_Document", Value: u(TextUnitDocument)},
		layouttest.Const{Name: "TextPatternRangeEndpoint_Start", Value: u(EndpointStart)},
		layouttest.Const{Name: "TextPatternRangeEndpoint_End", Value: u(EndpointEnd)},
		layouttest.Const{Name: "SupportedTextSelection_Single", Value: u(SupportedTextSelectionSingle)},
		layouttest.Const{Name: "UIA_E_ELEMENTNOTENABLED", Value: ErrElementNotEnabled},
		layouttest.Const{Name: "UIA_E_ELEMENTNOTAVAILABLE", Value: ErrElementNotAvailable},
		layouttest.Const{Name: "UIA_E_NOTSUPPORTED", Value: ErrNotSupported},
		layouttest.Const{Name: "UIA_E_INVALIDOPERATION", Value: ErrInvalidOperation},
		layouttest.Const{Name: "UiaAppendRuntimeId", Value: u(AppendRuntimeID)},
		layouttest.Const{Name: "UiaRootObjectId", Value: u(int(RootObjectID))},
	)
	layouttest.Check(t, layouttest.Spec{
		Headers: []string{"windows.h", "unknwn.h", "oleauto.h", "uiautomationcore.h", "uiautomationclient.h", "uiautomationcoreapi.h"},
		VTables: vtables,
		Consts:  consts,
	})
}
