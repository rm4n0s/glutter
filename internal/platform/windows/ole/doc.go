// Package ole is the OLE drag and drop of the Windows backend (SPEC §25.10): the COM objects that
// Go makes for it (IDropTarget, IDropSource, IDataObject and IEnumFORMATETC, SPEC §25.2), the reader
// of a data object that another program made, and the plain-Go models they stand on, which are tested
// on any host: the effects and the keys that choose among them, the formats and their names, the set
// of data that a drag offers, the enumerator of formats, and the decisions of the drag source.
//
// The files that call Windows are named *_windows.go. The policy (what a drop means to the framework,
// what a drag offers) is package windows (dnd.go): this package knows bytes and formats, not widgets.
//
// Flutter: no counterpart (Flutter has no drag and drop between applications)
package ole
