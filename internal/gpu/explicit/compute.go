package explicit

// ComputePipelineDesc describes a compute pipeline: the compute entry point of a generated shader.
//
// Flutter: no counterpart
type ComputePipelineDesc struct {
	// Shader is a generated shader.
	Shader *Shader
	// Entry is the name of its compute entry point.
	Entry string
}

// ComputePipeline is a GPU compute pipeline.
//
// Flutter: no counterpart
type ComputePipeline interface {
	// Desc is what the pipeline was made with.
	Desc() ComputePipelineDesc
	// Destroy releases the pipeline.
	Destroy()
}

// QuerySet is a set of timestamps that passes write and Encoder.ResolveQuerySet reads.
//
// Flutter: no counterpart
type QuerySet interface {
	// Count is the number of timestamps.
	Count() int
	// Destroy releases the query set.
	Destroy()
}

// PassTimestamps says where a pass writes its timestamps: the time at which the pass began into
// the query Begin of Set, and the time at which it ended into the query End. A negative index
// writes nothing.
//
// Flutter: no counterpart
type PassTimestamps struct {
	// Set is the query set.
	Set QuerySet
	// Begin and End are the indices of its queries.
	Begin, End int
}

// ComputePassDesc describes a compute pass.
//
// Flutter: no counterpart
type ComputePassDesc struct {
	// Timestamps, when set, is where the pass writes its timestamps; it needs Features.TimestampQuery.
	Timestamps *PassTimestamps
}

// ComputePass records the dispatches of one compute pass.
//
// Flutter: no counterpart
type ComputePass interface {
	// SetPipeline chooses the pipeline of the next dispatches.
	SetPipeline(ComputePipeline)
	// SetBindGroup binds the group number group, with one dynamic offset for each uniform buffer in
	// it, in binding order (a storage buffer has none: its offset is that of the bind group).
	SetBindGroup(group int, g BindGroup, dynamicOffsets ...uint32)
	// Dispatch runs x × y × z workgroups; each is at most Limits.MaxComputeWorkgroupsPerDimension.
	Dispatch(x, y, z int)
	// DispatchIndirect runs the workgroups whose counts are in b at offset: three 32 bit unsigned
	// integers. b has BufferIndirect; offset is a multiple of 4.
	DispatchIndirect(b Buffer, offset int)
	// End finishes the pass.
	End()
}

// BufferLayout is how the rows of a texture box lie in a buffer, for the copies between a buffer and
// a texture.
//
// Flutter: no counterpart
type BufferLayout struct {
	// Offset is where the first row starts, a multiple of 4 and of the bytes of a pixel.
	Offset int
	// BytesPerRow is the distance between two rows, a multiple of Limits.CopyRowPitchAlignment
	// (Limits.AlignedRowPitch gives one).
	BytesPerRow int
	// RowsPerImage is the number of rows of one layer or slice; the distance between two layers or
	// slices is BytesPerRow × RowsPerImage. It is at least the height of the box.
	RowsPerImage int
}
