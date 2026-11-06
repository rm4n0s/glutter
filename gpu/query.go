package gpu

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

// maxQueries is the most timestamps a query set holds.
const maxQueries = 4096

// QuerySet is a set of timestamps that passes write (PassTimestampWrites) and CommandEncoder.ResolveQuerySet reads
// (WebGPU's GPUQuerySet). It needs Features.TimestampQuery.
//
// Flutter: no counterpart
type QuerySet struct {
	resource
	exp   explicit.QuerySet
	count int
}

// NewQuerySet makes a set of timestamps (WebGPU's GPUDevice.createQuerySet). It fails with GPUInvalidDescriptor
// when Features.TimestampQuery is false or the count is not from 1 to 4096.
//
// Flutter: no counterpart
func (d *Device) NewQuerySet(desc QuerySetDesc) (*QuerySet, error) {
	d.assertThread("NewQuerySet")
	if d.lostErr != nil {
		return nil, d.lostErr
	}
	if !d.feat.TimestampQuery {
		return nil, errors.New(TagGPUInvalidDescriptor, "a query set needs Features.TimestampQuery, which this device lacks", "field", "Features.TimestampQuery", "label", desc.Label)
	}
	if desc.Count < 1 || desc.Count > maxQueries {
		return nil, errors.New(TagGPUInvalidDescriptor, "a query set holds from 1 to 4096 timestamps", "field", "Count", "value", desc.Count, "max", maxQueries, "label", desc.Label)
	}
	exp, err := d.dev.NewQuerySet(desc.Count)
	if err != nil {
		return nil, err
	}
	return &QuerySet{resource: d.track(kindQuerySet, desc.Label, 1), exp: exp, count: desc.Count}, nil
}

// Count returns the number of timestamps.
//
// Flutter: no counterpart
func (q *QuerySet) Count() int { return q.count }

// Release frees the set. It may be called twice.
//
// Flutter: no counterpart
func (q *QuerySet) Release() {
	q.d.assertThread("QuerySet.Release")
	if !q.untrack() {
		return
	}
	if q.d.lostErr == nil {
		q.exp.Destroy()
	}
}

// timestamps checks the timestamp writes of a pass and converts them. A programmer error panics in debug builds.
func (d *Device) timestamps(call string, w *PassTimestampWrites) *explicit.PassTimestamps {
	if w == nil {
		return nil
	}
	misuse(d.feat.TimestampQuery, call, "timestamp writes need Features.TimestampQuery")
	misuse(w.QuerySet != nil && w.QuerySet.d == d, call, "the timestamps go to a query set of this device")
	w.QuerySet.usable(call)
	misuse(w.BeginIndex < w.QuerySet.count && w.EndIndex < w.QuerySet.count, call, "the timestamps of a pass are queries of the set",
		"begin", w.BeginIndex, "end", w.EndIndex, "count", w.QuerySet.count)
	misuse(w.BeginIndex < 0 || w.EndIndex < 0 || w.BeginIndex != w.EndIndex, call, "the two timestamps of a pass are two queries")
	return &explicit.PassTimestamps{Set: w.QuerySet.exp, Begin: w.BeginIndex, End: w.EndIndex}
}
