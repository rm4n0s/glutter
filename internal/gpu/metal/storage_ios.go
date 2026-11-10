//go:build ios

package metal

// managedStorageAvailable says whether a buffer can have the managed storage mode: iOS has none (the Simulator's Metal device says it has
// no unified memory, and asserts on a managed buffer), so every buffer there is shared.
const managedStorageAvailable = false
