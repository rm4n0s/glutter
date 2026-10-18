package ui

// Offset is an immutable 2D floating-point offset: a point or a vector in
// logical or physical pixels, depending on where it is used.
//
// TODO(step 2.1): the methods and operators of Offset.
//
// Flutter: ui/geometry.dart Offset
type Offset struct {
	// Dx is the horizontal component.
	Dx float64
	// Dy is the vertical component.
	Dy float64
}

// Size holds a 2D floating-point size: a width and a height.
//
// TODO(step 2.1): the methods and operators of Size.
//
// Flutter: ui/geometry.dart Size
type Size struct {
	// Width is the horizontal extent.
	Width float64
	// Height is the vertical extent.
	Height float64
}

// Rect is an immutable, 2D, axis-aligned, floating-point rectangle whose
// coordinates are given relative to an origin.
//
// TODO(step 2.1): the methods of Rect.
//
// Flutter: ui/geometry.dart Rect
type Rect struct {
	// Left is the offset of the left edge from the x axis.
	Left float64
	// Top is the offset of the top edge from the y axis.
	Top float64
	// Right is the offset of the right edge from the x axis.
	Right float64
	// Bottom is the offset of the bottom edge from the y axis.
	Bottom float64
}
