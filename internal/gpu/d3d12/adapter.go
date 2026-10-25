package d3d12

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/rm4n0s/errors"
)

// AdapterEnvVar is the environment variable that picks the adapter: an index, a part of the name (any
// case) or "warp" (SPEC §9.8, like GLUTTER_VULKAN_DEVICE in §9.4).
//
// Flutter: no counterpart
const AdapterEnvVar = "GLUTTER_D3D12_ADAPTER"

// AdapterInfo is what the choice of an adapter needs to know about one adapter of DXGI.
//
// Flutter: no counterpart
type AdapterInfo struct {
	// Index is the position in the order that DXGI listed the adapters in.
	Index int
	// Name is the description of the adapter.
	Name string
	// VendorID, DeviceID and Revision are the fields of those names.
	VendorID, DeviceID, Revision uint32
	// Software is DXGI_ADAPTER_FLAG_SOFTWARE: the Microsoft Basic Render Driver, which is not a
	// hardware adapter and is used only as WARP (SPEC §9.8).
	Software bool
	// DedicatedVideoMemory is the memory of the GPU in bytes.
	DedicatedVideoMemory uint64
}

// candidate is an adapter to try: one of the list, or WARP.
type candidate struct {
	// Index is the index in the list of adapters; -1 for WARP.
	Index int
	// Warp says that the adapter is the one IDXGIFactory4::EnumWarpAdapter gives.
	Warp bool
}

// chooseAdapters returns the adapters to try, in order, for the selector (SPEC §9.8): the first that
// makes a device at feature level 11_0 wins.
//
//   - "" or "auto": the hardware adapters in the order DXGI listed them (the adapters with the
//     software flag are skipped), then WARP when allowWarp says so (SPEC §9.6: `auto` only).
//   - "warp" (any case): WARP alone, whether allowWarp or not (it was asked for).
//   - a number: the adapter at that index, hardware or software.
//   - anything else: every adapter whose name contains it (any case), in order; a name that matches
//     two gives both, the first that works wins.
//
// It fails with D3D12NoAdapter when the list has no candidate: no hardware adapter and no WARP, an
// index out of range, or a part of a name that no adapter has.
func chooseAdapters(list []AdapterInfo, selector string, allowWarp bool) ([]candidate, error) {
	selector = strings.TrimSpace(selector)
	switch strings.ToLower(selector) {
	case "", "auto":
		var out []candidate
		for _, a := range list {
			if !a.Software {
				out = append(out, candidate{Index: a.Index})
			}
		}
		if allowWarp {
			out = append(out, candidate{Index: -1, Warp: true})
		}
		if len(out) == 0 {
			return nil, errors.New(TagD3D12NoAdapter, "DXGI lists no hardware adapter", "adapters", adapterNames(list), "selector", selector)
		}
		return out, nil
	case "warp":
		return []candidate{{Index: -1, Warp: true}}, nil
	}
	if n, err := strconv.Atoi(selector); err == nil {
		for _, a := range list {
			if a.Index == n {
				return []candidate{{Index: a.Index}}, nil
			}
		}
		return nil, errors.New(TagD3D12NoAdapter, "no adapter has the index that "+AdapterEnvVar+" names", "adapters", adapterNames(list), "selector", selector)
	}
	var out []candidate
	for _, a := range list {
		if strings.Contains(strings.ToLower(a.Name), strings.ToLower(selector)) {
			out = append(out, candidate{Index: a.Index})
		}
	}
	if len(out) == 0 {
		return nil, errors.New(TagD3D12NoAdapter, "no adapter has the name that "+AdapterEnvVar+" names", "adapters", adapterNames(list), "selector", selector)
	}
	return out, nil
}

// adapterNames lists the adapters as "0 name (software)" for the metadata of an error.
func adapterNames(list []AdapterInfo) []string {
	names := make([]string, len(list))
	for i, a := range list {
		names[i] = fmt.Sprintf("%d %s", a.Index, a.Name)
		if a.Software {
			names[i] += " (software)"
		}
	}
	return names
}

// featureLevelName writes a feature level as the headers name it: "11_0", "12_1".
func featureLevelName(level uint32) string {
	return fmt.Sprintf("%d_%d", level>>12, (level>>8)&0xF)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	// TagD3D12NoAdapter is the tag of the error for an adapter choice that has no candidate, and for
	// a list of candidates none of which exists; the metadata are "adapters" and "selector".
	TagD3D12NoAdapter = "D3D12NoAdapter"
)
