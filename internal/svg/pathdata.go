package svg

import (
	"math"

	"github.com/rm4n0s/glutter/ui"
)

// ParsePathData parses the d attribute of a path into a ui.Path with the commands M, L, H, V, C, S, Q, T, A and Z,
// absolute and relative. Numbers may be written as the specification allows ("1.5.5" is 1.5 and .5, "-1-2" is -1
// and -2, and the flags of an arc may follow each other without separators). Parsing stops at the first error, as
// browsers do, and the path keeps what came before it; the second result is false in that case. Data that does not
// start with a moveto gives an empty path.
//
// SVG 1.1: paths.html#PathDataBNF; SVG 2: paths.html#PathDataErrorHandling
func ParsePathData(d string) (*ui.Path, bool) {
	p := ui.NewPath()
	ok := appendPathData(p, d, nil)
	return p, ok
}

// appendPathData adds the path data to p; it reports whether the data was well formed to its end. When vc is not nil,
// it gets each command, for the vertices of markers.
func appendPathData(p *ui.Path, d string, vc *vertexCollector) bool {
	var (
		cx, cy   float64 // the current point
		sx, sy   float64 // the start of the current subpath
		lcx, lcy float64 // the last control point, for S and T
		prev     byte    // the previous command, upper case
		cmd      byte
		haveMove bool
		i        = skipSpace(d, 0)
		first    = true
		num      [7]float64
	)
	for i < len(d) {
		c := d[i]
		isCmd := c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z'
		if isCmd {
			if !isPathCommand(c) {
				return false
			}
			cmd = c
			i = skipSpace(d, i+1)
		} else if first || cmd == 0 || cmd == 'Z' || cmd == 'z' {
			// Numbers with no command to continue, or after Z.
			return false
		} else {
			// Implicit repetition: moveto becomes lineto.
			switch cmd {
			case 'M':
				cmd = 'L'
			case 'm':
				cmd = 'l'
			}
		}
		if first {
			if cmd != 'M' && cmd != 'm' {
				return false
			}
			first = false
		}
		up := cmd &^ 0x20
		rel := cmd >= 'a'
		if up == 'Z' {
			if haveMove {
				p.Close()
				cx, cy = sx, sy
				if vc != nil {
					vc.closePath()
				}
			}
			prev = 'Z'
			lcx, lcy = cx, cy
			i = skipCommaSpace(d, i)
			continue
		}
		n := pathArgCount(up)
		badTail := false
		for k := 0; k < n; k++ {
			if up == 'A' && (k == 3 || k == 4) {
				// The flags are single characters 0 or 1.
				if i >= len(d) || (d[i] != '0' && d[i] != '1') {
					return false
				}
				num[k] = float64(d[i] - '0')
				i = skipCommaSpace(d, i+1)
				continue
			}
			v, j, ok := scanNumber(d, i)
			if !ok {
				return false
			}
			num[k] = v
			i = skipCommaSpace(d, j)
			if k == n-1 && j < len(d) && i > skipSpace(d, j) && (i >= len(d) || isCommandByte(d[i])) {
				// A comma must be followed by a number: the segment is complete, and the data ends here.
				badTail = true
			}
		}
		ox, oy := 0.0, 0.0
		if rel {
			ox, oy = cx, cy
		}
		switch up {
		case 'M':
			cx, cy = num[0]+ox, num[1]+oy
			sx, sy = cx, cy
			p.MoveTo(cx, cy)
			if vc != nil {
				vc.move(cx, cy)
			}
			haveMove = true
			lcx, lcy = cx, cy
		case 'L':
			cx, cy = num[0]+ox, num[1]+oy
			p.LineTo(cx, cy)
			if vc != nil {
				vc.line(cx, cy)
			}
			lcx, lcy = cx, cy
		case 'H':
			cx = num[0] + ox
			p.LineTo(cx, cy)
			if vc != nil {
				vc.line(cx, cy)
			}
			lcx, lcy = cx, cy
		case 'V':
			cy = num[0] + oy
			p.LineTo(cx, cy)
			if vc != nil {
				vc.line(cx, cy)
			}
			lcx, lcy = cx, cy
		case 'C':
			x1, y1 := num[0]+ox, num[1]+oy
			x2, y2 := num[2]+ox, num[3]+oy
			if vc != nil {
				vc.cubic(x1, y1, x2, y2, num[4]+ox, num[5]+oy)
			}
			cx, cy = num[4]+ox, num[5]+oy
			p.CubicTo(x1, y1, x2, y2, cx, cy)
			lcx, lcy = x2, y2
		case 'S':
			x1, y1 := cx, cy
			if prev == 'C' || prev == 'S' {
				x1, y1 = 2*cx-lcx, 2*cy-lcy
			}
			x2, y2 := num[0]+ox, num[1]+oy
			if vc != nil {
				vc.cubic(x1, y1, x2, y2, num[2]+ox, num[3]+oy)
			}
			cx, cy = num[2]+ox, num[3]+oy
			p.CubicTo(x1, y1, x2, y2, cx, cy)
			lcx, lcy = x2, y2
		case 'Q':
			x1, y1 := num[0]+ox, num[1]+oy
			if vc != nil {
				vc.quad(x1, y1, num[2]+ox, num[3]+oy)
			}
			cx, cy = num[2]+ox, num[3]+oy
			p.QuadraticBezierTo(x1, y1, cx, cy)
			lcx, lcy = x1, y1
		case 'T':
			x1, y1 := cx, cy
			if prev == 'Q' || prev == 'T' {
				x1, y1 = 2*cx-lcx, 2*cy-lcy
			}
			if vc != nil {
				vc.quad(x1, y1, num[0]+ox, num[1]+oy)
			}
			cx, cy = num[0]+ox, num[1]+oy
			p.QuadraticBezierTo(x1, y1, cx, cy)
			lcx, lcy = x1, y1
		case 'A':
			rx, ry := math.Abs(num[0]), math.Abs(num[1])
			ex, ey := num[5]+ox, num[6]+oy
			sweep := num[4] == 1
			arcOpts := ui.ArcToPointOpts{
				Radius:    ui.Radius{X: rx, Y: ry},
				Rotation:  num[2] * math.Pi / 180,
				LargeArc:  num[3] == 1,
				Clockwise: &sweep,
			}
			if vc != nil {
				vc.arc(ex, ey, arcOpts)
			}
			cx, cy = ex, ey
			p.ArcToPoint(ui.Offset{Dx: cx, Dy: cy}, arcOpts)
			lcx, lcy = cx, cy
		}
		prev = up
		if badTail {
			return false
		}
	}
	return true
}

// isCommandByte reports whether c is a letter, which starts a command (or is an error).
func isCommandByte(c byte) bool { return c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' }

func isPathCommand(c byte) bool {
	switch c &^ 0x20 {
	case 'M', 'L', 'H', 'V', 'C', 'S', 'Q', 'T', 'A', 'Z':
		return true
	}
	return false
}

func pathArgCount(up byte) int {
	switch up {
	case 'M', 'L', 'T':
		return 2
	case 'H', 'V':
		return 1
	case 'C':
		return 6
	case 'S', 'Q':
		return 4
	case 'A':
		return 7
	}
	return 0
}
