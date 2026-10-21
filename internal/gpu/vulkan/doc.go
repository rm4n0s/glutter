// Package vulkan is the Vulkan backend of the explicit renderer (SPEC §9.4): it
// loads libvulkan.so.1 through purego and implements explicit.Device on it. It
// has no cgo, and no Vulkan type leaves the package.
//
// Flutter: Impeller's Vulkan backend
// (engine/src/flutter/impeller/renderer/backend/vulkan).
package vulkan
