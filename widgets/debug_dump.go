package widgets

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/semantics"
)

// DebugDescribeApp describes the element tree of the app: the text that DebugDumpApp
// prints.
//
// Flutter: widgets/binding.dart debugDumpApp
func (b *WidgetsBinding) DebugDescribeApp() string {
	if b.rootElement == nil {
		return "<no tree currently mounted>"
	}
	return foundation.ToStringDeep(b.rootElement)
}

// DebugDumpApp prints the element tree of the app of the context, through the
// binding's debug Print (standard error unless the app set one). Debug builds only.
//
// Deviation: Flutter's debugDumpApp is a global function; this one finds the
// binding through the context (SPEC §12.10).
//
// Flutter: widgets/binding.dart debugDumpApp
func DebugDumpApp(bc BuildContext) {
	if foundation.DebugMode {
		if b := MaybeWidgetsBindingOf(bc); b != nil {
			b.debug.DebugPrint(b.DebugDescribeApp())
		}
	}
}

// DebugDumpRenderTree prints the render tree of every view of the app of the context.
// Debug builds only.
//
// Flutter: rendering/binding.dart debugDumpRenderTree
func DebugDumpRenderTree(bc BuildContext) {
	if foundation.DebugMode {
		if b := MaybeWidgetsBindingOf(bc); b != nil {
			b.debug.DebugPrint(b.renderer.DebugDumpRenderTree())
		}
	}
}

// DebugDumpLayerTree prints the layer tree of every view of the app of the context.
// Debug builds only.
//
// Flutter: rendering/binding.dart debugDumpLayerTree
func DebugDumpLayerTree(bc BuildContext) {
	if foundation.DebugMode {
		if b := MaybeWidgetsBindingOf(bc); b != nil {
			b.debug.DebugPrint(b.renderer.DebugDumpLayerTree())
		}
	}
}

// DebugDumpSemanticsTree prints the semantics tree of every view of the app of the
// context, in the order given (the zero value is traversal order, Flutter's default).
// Debug builds only.
//
// Flutter: rendering/binding.dart debugDumpSemanticsTree
func DebugDumpSemanticsTree(bc BuildContext, order ...semantics.DebugSemanticsDumpOrder) {
	if foundation.DebugMode {
		if b := MaybeWidgetsBindingOf(bc); b != nil {
			o := semantics.DebugSemanticsDumpOrderTraversal
			if len(order) > 0 {
				o = order[0]
			}
			b.debug.DebugPrint(b.renderer.DebugDumpSemanticsTree(o))
		}
	}
}

// DebugDumpPipelineOwnerTree prints the tree of pipeline owners of the app of the
// context. Debug builds only.
//
// Flutter: rendering/binding.dart debugDumpPipelineOwnerTree
func DebugDumpPipelineOwnerTree(bc BuildContext) {
	if foundation.DebugMode {
		if b := MaybeWidgetsBindingOf(bc); b != nil {
			b.debug.DebugPrint(b.renderer.DebugDumpPipelineOwnerTree())
		}
	}
}
