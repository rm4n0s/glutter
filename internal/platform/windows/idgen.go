package windows

// sequentialIDGenerator makes a series of sequential ids so that a new id is always the lowest
// one that is free. The touch and pen contacts of Windows have pointer ids that grow without
// end; the framework gets small ids that are reused (SPEC §25.6).
//
// Deviation: the generator is also used for the pen, which Flutter's embedder does too (one
// generator for every pointer message); only the device id built from the generated id differs
// (pointer.go).
//
// Flutter: shell/platform/windows/sequential_id_generator.cc SequentialIdGenerator
type sequentialIDGenerator struct {
	numberToID map[uint32]uint32
	idToNumber map[uint32]uint32
	minID      uint32
	maxID      uint32
	// minAvailable is the lowest id that may be free.
	minAvailable uint32
}

// newSequentialIDGenerator makes a generator whose ids are in [minID, maxID).
//
// Flutter: shell/platform/windows/sequential_id_generator.cc SequentialIdGenerator::SequentialIdGenerator
func newSequentialIDGenerator(minID, maxID uint32) *sequentialIDGenerator {
	return &sequentialIDGenerator{
		numberToID: map[uint32]uint32{}, idToNumber: map[uint32]uint32{},
		minID: minID, maxID: maxID, minAvailable: minID,
	}
}

// generatedID is the id that stands for the number: the one it had, or the lowest free one.
//
// Flutter: shell/platform/windows/sequential_id_generator.cc SequentialIdGenerator::GetGeneratedId
func (g *sequentialIDGenerator) generatedID(number uint32) uint32 {
	if id, ok := g.numberToID[number]; ok {
		return id
	}
	id := g.nextAvailable()
	g.numberToID[number] = id
	g.idToNumber[id] = number
	return id
}

// hasGeneratedIDFor reports whether the number has an id.
//
// Flutter: shell/platform/windows/sequential_id_generator.cc SequentialIdGenerator::HasGeneratedIdFor
func (g *sequentialIDGenerator) hasGeneratedIDFor(number uint32) bool {
	_, ok := g.numberToID[number]
	return ok
}

// releaseNumber frees the id of the number; a number without one does nothing.
//
// Flutter: shell/platform/windows/sequential_id_generator.cc SequentialIdGenerator::ReleaseNumber
func (g *sequentialIDGenerator) releaseNumber(number uint32) {
	id, ok := g.numberToID[number]
	if !ok {
		return
	}
	g.updateAfterRelease(id)
	delete(g.numberToID, number)
	delete(g.idToNumber, id)
}

// releaseID frees an id; an id that is not in use does nothing.
//
// Flutter: shell/platform/windows/sequential_id_generator.cc SequentialIdGenerator::ReleaseId
func (g *sequentialIDGenerator) releaseID(id uint32) {
	number, ok := g.idToNumber[id]
	if !ok {
		return
	}
	g.updateAfterRelease(id)
	delete(g.numberToID, number)
	delete(g.idToNumber, id)
}

// nextAvailable is the lowest free id; when all are taken it starts at the lowest again, as the
// C++ does.
//
// Flutter: shell/platform/windows/sequential_id_generator.cc SequentialIdGenerator::GetNextAvailableId
func (g *sequentialIDGenerator) nextAvailable() uint32 {
	for {
		if _, taken := g.idToNumber[g.minAvailable]; !taken || g.minAvailable >= g.maxID {
			break
		}
		g.minAvailable++
	}
	if g.minAvailable >= g.maxID {
		g.minAvailable = g.minID
	}
	return g.minAvailable
}

// updateAfterRelease lowers the lowest free id when the released id is lower.
//
// Flutter: shell/platform/windows/sequential_id_generator.cc SequentialIdGenerator::UpdateNextAvailableIdAfterRelease
func (g *sequentialIDGenerator) updateAfterRelease(id uint32) {
	if id < g.minAvailable {
		g.minAvailable = id
	}
}
