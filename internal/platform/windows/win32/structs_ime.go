package win32

// The Go mirrors of the structures of IMM32 (step 28.5). Like those of structs.go they carry no OS
// tag, so that the layout test checks them against the headers (SPEC §25.2); the `c` tag of a field
// is its name in the C header.

// CompositionForm is the Go mirror of COMPOSITIONFORM (ImmSetCompositionWindow): where the input
// method draws the composition string.
//
// Flutter: no counterpart
type CompositionForm struct {
	Style      uint32 `c:"dwStyle"`
	CurrentPos Point  `c:"ptCurrentPos"`
	Area       Rect   `c:"rcArea"`
}

// CandidateForm is the Go mirror of CANDIDATEFORM (ImmSetCandidateWindow): where the input method
// opens the window of its candidates.
//
// Flutter: no counterpart
type CandidateForm struct {
	Index      uint32 `c:"dwIndex"`
	Style      uint32 `c:"dwStyle"`
	CurrentPos Point  `c:"ptCurrentPos"`
	Area       Rect   `c:"rcArea"`
}
