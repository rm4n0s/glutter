//go:build linux && !amd64 && !arm64

package wayland

// sysMemfdCreate is 0 on architectures whose number is not known here;
// memfdCreate then fails with ENOSYS. Add the number for the architecture.
const sysMemfdCreate = 0
