//go:build windows

package windows

import (
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/wasapi"
)

// audioService makes the audio service (WASAPI, SPEC §28.3, step 31.6) on the first call, which may come from any goroutine. Nothing
// of the audio API of Windows is touched until the app lists or opens a device.
func (p *Platform) audioService() platform.Audio {
	p.audioOnce.Do(func() { p.audio = wasapi.New(p.log) })
	return p.audio
}
