package gestures

import "github.com/rm4n0s/glutter/foundation"

func someInt(v int) foundation.Opt[int]           { return foundation.Some(v) }
func someFloat(v float64) foundation.Opt[float64] { return foundation.Some(v) }
