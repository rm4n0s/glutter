//go:build linux

package vulkan

import (
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

func TestDescriptorTypeOfEachBindingKind(t *testing.T) {
	tests := []struct {
		kind explicit.BindingKind
		want uint32
	}{
		{explicit.BindingUniformBuffer, descriptorUniformDynamic},
		{explicit.BindingSampledTexture, descriptorSampledImage},
		{explicit.BindingSampler, descriptorSampler},
		{explicit.BindingStorageBufferReadOnly, descriptorStorageBuffer},
		{explicit.BindingStorageBuffer, descriptorStorageBuffer},
		{explicit.BindingStorageTexture, descriptorStorageImage},
	}
	for _, tt := range tests {
		if got := descriptorType(tt.kind); got != tt.want {
			t.Errorf("%v: descriptor type %d, want %d", tt.kind, got, tt.want)
		}
	}
}

func TestQuerySetsNeedTimestampQueries(t *testing.T) {
	d := &Device{}
	_, err := d.NewQuerySet(2)
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T", err)
	}
	if appErr.Tag != explicit.TagGPUQueryUnsupported || !appErr.HasRoute("Device.NewQuerySet.GPUQueryUnsupported") {
		t.Errorf("unexpected failure path: %s", appErr.Route())
	}
}

func TestTimestampQueriesOfTheQueue(t *testing.T) {
	tests := []struct {
		name   string
		info   physicalInfo
		family uint32
		want   bool
		period float64
	}{
		{"a queue that writes timestamps", physicalInfo{TimestampValidBits: []uint32{64}, TimestampComputeAndGraphics: true, TimestampPeriod: 1}, 0, true, 1},
		{"a period that is not 1", physicalInfo{TimestampValidBits: []uint32{36, 64}, TimestampComputeAndGraphics: true, TimestampPeriod: 52.5}, 1, true, 52.5},
		{"a queue without valid bits", physicalInfo{TimestampValidBits: []uint32{0}, TimestampComputeAndGraphics: true, TimestampPeriod: 1}, 0, false, 0},
		{"timestamps only on some queues", physicalInfo{TimestampValidBits: []uint32{64}, TimestampPeriod: 1}, 0, false, 0},
		{"a period of 0", physicalInfo{TimestampValidBits: []uint32{64}, TimestampComputeAndGraphics: true}, 0, false, 0},
		{"a family that does not exist", physicalInfo{TimestampValidBits: []uint32{64}, TimestampComputeAndGraphics: true, TimestampPeriod: 1}, 3, false, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Device{info: tt.info, family: tt.family}
			d.queryTimestamps()
			f := d.Features()
			if f.TimestampQuery != tt.want || f.TimestampPeriod != tt.period {
				t.Errorf("Features %+v, want TimestampQuery %v and period %g", f, tt.want, tt.period)
			}
		})
	}
}

func TestBufferTextureCopyRegions(t *testing.T) {
	tests := []struct {
		name   string
		tex    *texture
		region explicit.TextureRegion
		layout explicit.BufferLayout
		want   bufferImageCopy
	}{
		{
			"a 2D texture, rows with padding",
			&texture{desc: explicit.TextureDesc{Width: 9, Height: 6, Format: explicit.TextureFormatRGBA8Unorm, MipLevels: 1, SampleCount: 1, DepthOrLayers: 1}},
			explicit.TextureRegion{Origin: [3]int{2, 1, 0}, Size: [3]int{5, 3, 1}},
			explicit.BufferLayout{Offset: 16, BytesPerRow: 256, RowsPerImage: 4},
			bufferImageCopy{bufferOffset: 16, bufferRowLength: 64, bufferImageHeight: 4,
				imageSubresource: imageSubresourceLayers{aspectMask: imageAspectColor, layerCount: 1},
				imageOffset:      offset3D{2, 1, 0}, imageExtent: extent3D{5, 3, 1}},
		},
		{
			"a level of a one byte texture",
			&texture{desc: explicit.TextureDesc{Width: 9, Height: 6, Format: explicit.TextureFormatR8Unorm, MipLevels: 3, SampleCount: 1, DepthOrLayers: 1}},
			explicit.TextureRegion{Level: 2, Size: [3]int{2, 1, 1}},
			explicit.BufferLayout{BytesPerRow: 256, RowsPerImage: 1},
			bufferImageCopy{bufferRowLength: 256, bufferImageHeight: 1,
				imageSubresource: imageSubresourceLayers{aspectMask: imageAspectColor, mipLevel: 2, layerCount: 1},
				imageExtent:      extent3D{2, 1, 1}},
		},
		{
			"layers of an array",
			&texture{desc: explicit.TextureDesc{Kind: explicit.TextureKind2DArray, Width: 6, Height: 4, DepthOrLayers: 3, Format: explicit.TextureFormatRGBA8Unorm, MipLevels: 1, SampleCount: 1}},
			explicit.TextureRegion{Origin: [3]int{0, 0, 1}, Size: [3]int{6, 4, 2}},
			explicit.BufferLayout{BytesPerRow: 256, RowsPerImage: 5},
			bufferImageCopy{bufferRowLength: 64, bufferImageHeight: 5,
				imageSubresource: imageSubresourceLayers{aspectMask: imageAspectColor, baseArrayLayer: 1, layerCount: 2},
				imageExtent:      extent3D{6, 4, 1}},
		},
		{
			"slices of a volume",
			&texture{desc: explicit.TextureDesc{Kind: explicit.TextureKind3D, Width: 6, Height: 4, DepthOrLayers: 3, Format: explicit.TextureFormatRGBA16Float, MipLevels: 1, SampleCount: 1}},
			explicit.TextureRegion{Origin: [3]int{1, 0, 1}, Size: [3]int{4, 4, 2}},
			explicit.BufferLayout{Offset: 8, BytesPerRow: 256, RowsPerImage: 4},
			bufferImageCopy{bufferOffset: 8, bufferRowLength: 32, bufferImageHeight: 4,
				imageSubresource: imageSubresourceLayers{aspectMask: imageAspectColor, layerCount: 1},
				imageOffset:      offset3D{1, 0, 1}, imageExtent: extent3D{4, 4, 2}},
		},
		{
			"a view of one layer of an array",
			&texture{desc: explicit.TextureDesc{Width: 6, Height: 4, Format: explicit.TextureFormatRGBA8Unorm, MipLevels: 1, SampleCount: 1, DepthOrLayers: 1}, base: 1, baseLayer: 2},
			explicit.TextureRegion{Size: [3]int{3, 2, 1}},
			explicit.BufferLayout{BytesPerRow: 256, RowsPerImage: 2},
			bufferImageCopy{bufferRowLength: 64, bufferImageHeight: 2,
				imageSubresource: imageSubresourceLayers{aspectMask: imageAspectColor, mipLevel: 1, baseArrayLayer: 2, layerCount: 1},
				imageExtent:      extent3D{3, 2, 1}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := bufferTextureCopy(tt.tex, tt.region, tt.layout); got != tt.want {
				t.Errorf("region\n%+v\nwant\n%+v", got, tt.want)
			}
		})
	}
}
