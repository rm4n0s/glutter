//go:build linux && !android

package input

// PointerIDs hands out the pointer identifiers of a platform: the number that tells the gesture
// arena which contact an event belongs to. Every contact (a mouse press, a finger on the screen, a
// pen on the tablet, a touchpad gesture) gets a new one at its start, and the numbers are never
// given twice in a run, whatever the device, as in Flutter's engine, where the converter has one
// counter for all its pointers. The identifier 0 is what an event has before any contact of its
// pointer: the batchers of one platform share one PointerIDs.
//
// Deviation: Flutter's embedder hands the engine raw pointer data and the engine's
// PointerDataPacketConverter gives the identifiers; here the platform makes the final data, so the
// batchers give them (SPEC §8.5, §12.10).
//
// A PointerIDs is used on the UI thread only.
//
// Flutter: lib/ui/window/pointer_data_packet_converter.h PointerDataPacketConverter::pointer_
type PointerIDs struct {
	last int
}

// NewPointerIDs makes a counter whose first identifier is 1.
//
// Flutter: no counterpart
func NewPointerIDs() *PointerIDs { return &PointerIDs{} }

// next returns a new identifier. A nil counter is never used: the constructors of the batchers
// make one when they get nil.
//
// Flutter: lib/ui/window/pointer_data_packet_converter.cc PointerDataPacketConverter::UpdatePointerIdentifier
func (p *PointerIDs) next() int {
	p.last++
	return p.last
}
