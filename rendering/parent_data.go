package rendering

import (
	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// ParentData is the base class for data that a parent render object stores on
// its children: where to position them, who their siblings are, and so on.
// A type that holds parent data embeds ParentDataBase.
//
// Flutter: rendering/object.dart ParentData
type ParentData interface {
	// Detach is called when the child is removed from its parent.
	//
	// Flutter: rendering/object.dart ParentData.detach
	Detach()
	// String describes the data.
	String() string
}

// ParentDataBase is the default implementation of ParentData to embed.
//
// Flutter: rendering/object.dart ParentData
type ParentDataBase struct{}

// Detach does nothing.
//
// Flutter: rendering/object.dart ParentData.detach
func (*ParentDataBase) Detach() {}

// String is "<none>".
//
// Flutter: rendering/object.dart ParentData.toString
func (*ParentDataBase) String() string { return "<none>" }

// BoxParentData is the parent data for render boxes: the offset of the child
// in the parent's coordinates.
//
// Flutter: rendering/box.dart BoxParentData
type BoxParentData struct {
	ParentDataBase
	// Offset is the position of the child, in the coordinates of the parent.
	//
	// Flutter: rendering/box.dart BoxParentData.offset
	Offset ui.Offset
}

// String is "offset=" and the offset.
//
// Flutter: rendering/box.dart BoxParentData.toString
func (d *BoxParentData) String() string { return "offset=" + d.Offset.String() }

func (d *BoxParentData) boxParentData() *BoxParentData { return d }

// boxParentDataHolder is a ParentData that is or embeds a BoxParentData.
type boxParentDataHolder interface {
	ParentData
	boxParentData() *BoxParentData
}

// ContainerParentData is the parent data that links a child to its siblings,
// for a parent with a list of children of type C. Embed it (or
// ContainerBoxParentData) in the parent data type.
//
// Flutter: rendering/object.dart ContainerParentDataMixin
type ContainerParentData[C RenderObject] struct {
	// PreviousSibling is the child before this one, or nil.
	//
	// Flutter: rendering/object.dart ContainerParentDataMixin.previousSibling
	PreviousSibling C
	// NextSibling is the child after this one, or nil.
	//
	// Flutter: rendering/object.dart ContainerParentDataMixin.nextSibling
	NextSibling C
}

func (d *ContainerParentData[C]) links() *ContainerParentData[C] { return d }

// containerParentDataLinks is a ParentData that embeds a ContainerParentData.
type containerParentDataLinks[C RenderObject] interface {
	links() *ContainerParentData[C]
}

// checkDetach panics in debug builds if the sibling links are still set.
func (d *ContainerParentData[C]) checkDetach() {
	if foundation.DebugMode && (!isNil(d.PreviousSibling) || !isNil(d.NextSibling)) {
		panic(errors.New(tagParentDataSiblingsSet, "pointers to siblings must be nulled before detaching ParentData"))
	}
}

// ContainerBoxParentData is the parent data for the children of a render box
// with a list of children: an offset and the links to the siblings.
//
// Flutter: rendering/box.dart ContainerBoxParentData
type ContainerBoxParentData[C RenderObject] struct {
	BoxParentData
	ContainerParentData[C]
}

// Detach panics in debug builds if the sibling links are still set.
//
// Flutter: rendering/object.dart ContainerParentDataMixin.detach
func (d *ContainerBoxParentData[C]) Detach() { d.checkDetach() }

func isNil[T any](v T) bool { return any(v) == nil }

// BoxParentDataOf is the BoxParentData of a render box that its parent laid out,
// or the one embedded in the parent data of the child.
//
// Flutter: no counterpart (the cast `child.parentData! as BoxParentData`)
func BoxParentDataOf(child RenderObject) *BoxParentData {
	return child.ParentData().(boxParentDataHolder).boxParentData()
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagParentDataSiblingsSet = "ParentDataSiblingsSet"
)
