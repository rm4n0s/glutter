package ole

// FormatEnum is the state of an IEnumFORMATETC over the formats of a data set: the position, with the
// four operations of the interface. The COM object (dataobject_windows.go) only moves the formats to
// and from the memory that the caller gave.
//
// Flutter: no counterpart
type FormatEnum struct {
	formats []uint32
	pos     int
}

// NewFormatEnum enumerates a copy of the formats, from the first.
//
// Flutter: no counterpart
func NewFormatEnum(formats []uint32) *FormatEnum {
	return &FormatEnum{formats: append([]uint32(nil), formats...)}
}

// Next returns up to n formats from the position and moves past them. Fewer than n means the end was
// reached (the method returns S_FALSE).
//
// Flutter: no counterpart
func (e *FormatEnum) Next(n int) []uint32 {
	if n <= 0 || e.pos >= len(e.formats) {
		return nil
	}
	end := min(e.pos+n, len(e.formats))
	out := append([]uint32(nil), e.formats[e.pos:end]...)
	e.pos = end
	return out
}

// Skip moves past n formats and reports whether there were that many (S_OK); the position is the end
// otherwise (S_FALSE).
//
// Flutter: no counterpart
func (e *FormatEnum) Skip(n int) bool {
	if n < 0 {
		return false
	}
	if e.pos+n > len(e.formats) {
		e.pos = len(e.formats)
		return false
	}
	e.pos += n
	return true
}

// Reset goes back to the first format.
//
// Flutter: no counterpart
func (e *FormatEnum) Reset() { e.pos = 0 }

// Clone is an enumerator at the same position, which moves on its own.
//
// Flutter: no counterpart
func (e *FormatEnum) Clone() *FormatEnum {
	return &FormatEnum{formats: e.formats, pos: e.pos}
}
