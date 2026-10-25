package d3d12

import (
	"reflect"
	"testing"

	"github.com/rm4n0s/errors"
)

// Fake lists of adapters, as DXGI would list them.
var (
	hardwareOnly = []AdapterInfo{
		{Index: 0, Name: "NVIDIA GeForce RTX 4070"},
		{Index: 1, Name: "Intel(R) UHD Graphics 770"},
	}
	withSoftware = []AdapterInfo{
		{Index: 0, Name: "Microsoft Basic Render Driver", Software: true},
		{Index: 1, Name: "AMD Radeon RX 7600"},
		{Index: 2, Name: "AMD Radeon(TM) Graphics"},
	}
	onlySoftware = []AdapterInfo{{Index: 0, Name: "Microsoft Basic Render Driver", Software: true}}
)

func TestChooseAdapters(t *testing.T) {
	warp := candidate{Index: -1, Warp: true}
	tests := []struct {
		name      string
		list      []AdapterInfo
		selector  string
		allowWarp bool
		want      []candidate
	}{
		{"auto keeps the order DXGI gave", hardwareOnly, "", false, []candidate{{Index: 0}, {Index: 1}}},
		{"auto written out", hardwareOnly, "auto", false, []candidate{{Index: 0}, {Index: 1}}},
		{"auto skips the software adapter", withSoftware, "auto", false, []candidate{{Index: 1}, {Index: 2}}},
		{"auto ends with WARP when allowed", withSoftware, "", true, []candidate{{Index: 1}, {Index: 2}, warp}},
		{"auto with only a software adapter is WARP", onlySoftware, "auto", true, []candidate{warp}},
		{"auto with an empty list is WARP", nil, "", true, []candidate{warp}},
		{"warp by name", hardwareOnly, "warp", false, []candidate{warp}},
		{"warp by name, any case, around spaces", hardwareOnly, " WARP ", false, []candidate{warp}},
		{"an index", hardwareOnly, "1", false, []candidate{{Index: 1}}},
		{"an index can name the software adapter", withSoftware, "0", false, []candidate{{Index: 0}}},
		{"a part of a name, any case", hardwareOnly, "intel", false, []candidate{{Index: 1}}},
		{"a name that matches two gives both in order", withSoftware, "AMD", false, []candidate{{Index: 1}, {Index: 2}}},
		{"a name can pick the software adapter", withSoftware, "basic render", false, []candidate{{Index: 0}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := chooseAdapters(tt.list, tt.selector, tt.allowWarp)
			if err != nil {
				t.Fatalf("chooseAdapters: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("candidates = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChooseAdaptersFailures(t *testing.T) {
	tests := []struct {
		name      string
		list      []AdapterInfo
		selector  string
		allowWarp bool
	}{
		{"auto with no hardware adapter and no WARP", onlySoftware, "auto", false},
		{"auto with an empty list", nil, "", false},
		{"an index out of range", hardwareOnly, "2", false},
		{"a negative index", hardwareOnly, "-1", false},
		{"a name that no adapter has", hardwareOnly, "radeon", false},
		{"a name on an empty list", nil, "nvidia", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := chooseAdapters(tt.list, tt.selector, tt.allowWarp)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if appErr.Tag != TagD3D12NoAdapter || !appErr.HasRoute("chooseAdapters.D3D12NoAdapter") {
				t.Errorf("got %s via %s, want chooseAdapters.D3D12NoAdapter", appErr.Tag, appErr.Route())
			}
			if appErr.Metadata["selector"] != tt.selector && tt.selector != "" {
				t.Errorf("metadata selector = %v, want %q", appErr.Metadata["selector"], tt.selector)
			}
			if _, ok := appErr.Metadata["adapters"]; !ok {
				t.Errorf("the metadata do not list the adapters: %v", appErr.Metadata)
			}
		})
	}
}

func TestAdapterNamesMarkSoftware(t *testing.T) {
	got := adapterNames(withSoftware)
	want := []string{"0 Microsoft Basic Render Driver (software)", "1 AMD Radeon RX 7600", "2 AMD Radeon(TM) Graphics"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("adapterNames = %q, want %q", got, want)
	}
}

func TestFeatureLevelName(t *testing.T) {
	tests := []struct {
		level uint32
		want  string
	}{
		{featureLevel11_0, "11_0"}, {featureLevel11_1, "11_1"}, {featureLevel12_0, "12_0"}, {featureLevel12_1, "12_1"}, {featureLevel12_2, "12_2"},
	}
	for _, tt := range tests {
		if got := featureLevelName(tt.level); got != tt.want {
			t.Errorf("featureLevelName(%#x) = %q, want %q", tt.level, got, tt.want)
		}
	}
}
