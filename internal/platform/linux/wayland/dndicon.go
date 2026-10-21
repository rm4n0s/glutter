//go:build linux

package wayland

import (
	"math"

	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/wl"
)

// The icon that follows the pointer during a drag that the app started. The data
// of the drag has no picture, so the icon is a small card.
const (
	dragIconWidth  = 56
	dragIconHeight = 40
)

// dragIcon is the surface and the buffer of the drag icon.
type dragIcon struct {
	surface *wl.WlSurface
	buffer  *shmBuffer
}

// newDragIcon makes the icon surface: a translucent rounded card with a border.
func (p *Platform) newDragIcon() (*dragIcon, error) {
	buf, err := newShmBufferFormat(p.sess.shm, dragIconWidth, dragIconHeight, wl.WlShmFormatArgb8888, paintDragIcon, nil)
	if err != nil {
		return nil, err
	}
	surface := p.sess.compositor.CreateSurface()
	surface.Attach(buf.buffer, 0, 0)
	surface.Damage(0, 0, dragIconWidth, dragIconHeight)
	surface.Commit()
	return &dragIcon{surface: surface, buffer: buf}, nil
}

func (i *dragIcon) destroy() {
	i.surface.Destroy()
	i.buffer.free()
}

// paintDragIcon draws the card into premultiplied ARGB pixels.
func paintDragIcon(data []byte) {
	const radius = 8.0
	w, h := float64(dragIconWidth), float64(dragIconHeight)
	for y := range dragIconHeight {
		for x := range dragIconWidth {
			// The distance outside the rounded rectangle, negative inside.
			px, py := float64(x)+0.5, float64(y)+0.5
			dx := math.Max(math.Abs(px-w/2)-(w/2-radius), 0)
			dy := math.Max(math.Abs(py-h/2)-(h/2-radius), 0)
			dist := math.Hypot(dx, dy) - radius
			coverage := math.Min(math.Max(0.5-dist, 0), 1)
			// A light fill with a darker border one pixel wide.
			fill, alpha := 0xE8, 0.85
			if dist > -1.5 {
				fill, alpha = 0x70, 0.95
			}
			a := coverage * alpha
			c := byte(math.Round(float64(fill) * a))
			off := (y*dragIconWidth + x) * 4
			data[off], data[off+1], data[off+2], data[off+3] = c, c, c, byte(math.Round(a*255))
		}
	}
}
