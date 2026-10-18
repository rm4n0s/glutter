package gpu

// SurfaceHandle is an opaque handle to a native drawing surface. A platform
// backend creates it, and only a GPU backend looks inside.
//
// TODO(step 1.6): decide the concrete representation when the first GPU
// backend needs it.
//
// Flutter: no counterpart
type SurfaceHandle any
