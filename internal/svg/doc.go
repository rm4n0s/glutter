// Package svg parses SVG documents into a tree of resolved drawing commands.
//
// It is the first half of Glutter's SVG support (SPEC §26, D27); the painter that records the tree into a ui.Picture
// comes in later steps, and the package svg at the root of the module (flutter_svg's widgets) comes last. The
// package is plain Go and uses only the standard library (encoding/xml).
//
// # What Parse does
//
//   - It reads the XML into a light element tree (xml.go): elements and attributes of the SVG and xlink
//     namespaces, CDATA and entities. Elements of other namespaces, and unknown elements, are skipped with their
//     children.
//   - It runs the CSS cascade (css.go, style.go): presentation attributes, then the rules of the <style> sheets by
//     specificity and order, then the style attribute; !important beats all of them. The properties that inherit are
//     inherited down the tree.
//   - It resolves the values (values.go, affine.go, pathdata.go): lengths with every unit, colors, paints, transforms,
//     viewBox, preserveAspectRatio and path data (as ui.Path).
//   - It builds the tree of Node (build.go, resources.go, textnode.go): use is expanded with cycles cut and a limit,
//     symbol and nested svg become viewports, switch keeps its first child whose conditions hold, and the references by
//     id (gradients, patterns, clip paths, masks, filters, markers) become pointers.
//
// # Where units resolve
//
// A length is a user-space number in the tree wherever the parser can know it: px, pt, pc, mm, cm and in at parse
// time, em, ex and rem with the element's computed font size (the root's is Theme.FontSize), and a percentage against
// the nearest viewport in its dimension. What stays a Length with its unit is what depends on the object a paint
// server or a resource is applied to: the coordinates of gradients, patterns, masks and filters (a fraction of the
// bounding box under objectBoundingBox, a length of the viewport under userSpaceOnUse), which the painter resolves
// with Length.Resolve. A document whose root has neither a viewBox nor a width and a height has no viewport: its
// percentages are invalid and ignored.
//
// currentColor is resolved at parse time with the color property (Theme.CurrentColor at the root), so a Document is
// made for one Theme, and the cache of package svg is keyed by source and theme.
//
// # Source of the behavior
//
// The code follows the SVG 1.1 and SVG 2 specifications and CSS 2.1; comments name the section they implement, in
// place of "Flutter:".
package svg
