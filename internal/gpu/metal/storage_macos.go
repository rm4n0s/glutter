//go:build darwin && !ios

package metal

// managedStorageAvailable says whether a buffer can have the managed storage mode: macOS has it, for a GPU with memory of its own.
const managedStorageAvailable = true
