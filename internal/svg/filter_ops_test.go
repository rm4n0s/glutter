package svg

import (
	"math"
	"math/rand"
	"testing"
)

// pixel makes an image of one row with the given RGBA pixels (premultiplied).
func pixelRow(px ...[4]uint8) *pixbuf {
	b := newPixbuf(len(px), 1)
	for i, p := range px {
		copy(b.pix[4*i:], p[:])
	}
	return b
}

func at(b *pixbuf, x, y int) [4]uint8 {
	var p [4]uint8
	copy(p[:], b.pix[4*(y*b.w+x):])
	return p
}

func nearPx(a, b [4]uint8, tol int) bool {
	for i := range a {
		d := int(a[i]) - int(b[i])
		if d < -tol || d > tol {
			return false
		}
	}
	return true
}

func TestColorSpaceTables(t *testing.T) {
	tests := []struct {
		name     string
		in       uint8
		lut      *[256]uint8
		want     uint8
		tolerate int
	}{
		{"sRGB 0 is linear 0", 0, &srgbToLinearLUT, 0, 0},
		{"sRGB 255 is linear 255", 255, &srgbToLinearLUT, 255, 0},
		{"sRGB 128 is linear 55", 128, &srgbToLinearLUT, 55, 1},
		{"sRGB 10 is linear 1", 10, &srgbToLinearLUT, 1, 1},
		{"linear 55 is sRGB 128", 55, &linearToSRGBLUT, 128, 1},
		{"linear 1 is sRGB 13", 1, &linearToSRGBLUT, 13, 1},
		{"linear 255 is sRGB 255", 255, &linearToSRGBLUT, 255, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.lut[tt.in]
			if d := int(got) - int(tt.want); d < -tt.tolerate || d > tt.tolerate {
				t.Errorf("lut[%d] = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
	for i := 1; i < 256; i++ {
		if srgbToLinearLUT[i] < srgbToLinearLUT[i-1] || linearToSRGBLUT[i] < linearToSRGBLUT[i-1] {
			t.Fatalf("a table is not monotonic at %d", i)
		}
	}
}

func TestPremultiply(t *testing.T) {
	tests := []struct {
		c, a, straight uint8
	}{
		{0, 0, 0}, {255, 255, 255}, {100, 255, 100}, {64, 128, 128}, {10, 20, 128},
	}
	for _, tt := range tests {
		if got := unpremul(tt.c, tt.a); got != tt.straight {
			t.Errorf("unpremul(%d, %d) = %d, want %d", tt.c, tt.a, got, tt.straight)
		}
	}
	for c := 0; c < 256; c += 5 {
		for a := 1; a < 256; a += 7 {
			p := premul(uint8(c), uint8(a))
			back := int(unpremul(p, uint8(a)))
			if d := back - c; d > 255/a+1 || d < -(255/a+1) {
				t.Fatalf("premul/unpremul of %d at alpha %d gave %d", c, a, back)
			}
		}
	}
}

func TestInSpaceConvertsOnlyWhenNeeded(t *testing.T) {
	b := pixelRow([4]uint8{128, 64, 0, 255}, [4]uint8{64, 32, 0, 128})
	if got := b.inSpace(false); got != b {
		t.Error("an image in sRGB asked for sRGB is copied")
	}
	lin := b.inSpace(true)
	if lin == b || !lin.linear {
		t.Fatal("no conversion")
	}
	if got := at(lin, 0, 0); !nearPx(got, [4]uint8{55, 13, 0, 255}, 1) {
		t.Errorf("opaque pixel in linearRGB = %v", got)
	}
	if got := lin.inSpace(false); !nearPx(at(got, 0, 0), at(b, 0, 0), 2) {
		t.Errorf("round trip = %v, want %v", at(got, 0, 0), at(b, 0, 0))
	}
	// A pixel with alpha converts its straight color: 128 of alpha 255 is as 64 of 128.
	if got := at(lin, 1, 0); got[3] != 128 || got[0] > 64 {
		t.Errorf("translucent pixel in linearRGB = %v", got)
	}
}

func TestBoxesFor(t *testing.T) {
	tests := []struct {
		d    int
		want []box
	}{
		{3, []box{{3, 1, 1}, {3, 1, 1}, {3, 1, 1}}},
		{5, []box{{5, 2, 2}, {5, 2, 2}, {5, 2, 2}}},
		{4, []box{{4, 2, 1}, {4, 1, 2}, {5, 2, 2}}},
		{6, []box{{6, 3, 2}, {6, 2, 3}, {7, 3, 3}}},
		{0, []box{{1, 0, 0}, {1, 0, 0}, {1, 0, 0}}},
	}
	for _, tt := range tests {
		got := boxesFor(tt.d)
		for i := range tt.want {
			if got[i] != tt.want[i] {
				t.Errorf("boxesFor(%d)[%d] = %+v, want %+v", tt.d, i, got[i], tt.want[i])
			}
		}
		// Every box covers its size.
		for _, b := range got {
			if b.left+b.right+1 != b.size {
				t.Errorf("boxesFor(%d): box %+v does not cover its size", tt.d, b)
			}
		}
	}
}

func TestBlur(t *testing.T) {
	delta := func(w, h int) *pixbuf {
		b := newPixbuf(w, h)
		i := 4 * ((h/2)*w + w/2)
		b.pix[i], b.pix[i+1], b.pix[i+2], b.pix[i+3] = 255, 255, 255, 255
		return b
	}
	sum := func(b *pixbuf) (s int) {
		for i := 3; i < len(b.pix); i += 4 {
			s += int(b.pix[i])
		}
		return
	}
	tests := []struct {
		name   string
		sx, sy float64
	}{
		{"gaussian, small deviation", 1, 1}, {"boxes, the smallest deviation", 2, 2}, {"boxes, odd size", 3, 3},
		{"boxes, even size", 4, 4}, {"horizontal only", 3, 0}, {"vertical only", 0, 3}, {"different deviations", 2, 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := delta(61, 61)
			out := in.blurred(tt.sx, tt.sy)
			// Blurring spreads the alpha without changing its total (up to the rounding of 8 bits).
			if got, want := sum(out), sum(in); math.Abs(float64(got-want)) > float64(want)*0.12 {
				t.Errorf("total alpha = %d, want about %d", got, want)
			}
			// It is symmetric about the center.
			for d := 1; d < 10; d++ {
				l, r := at(out, 30-d, 30)[3], at(out, 30+d, 30)[3]
				u, v := at(out, 30, 30-d)[3], at(out, 30, 30+d)[3]
				if (tt.sx > 0 && (int(l)-int(r) > 2 || int(r)-int(l) > 2)) || (tt.sy > 0 && (int(u)-int(v) > 2 || int(v)-int(u) > 2)) {
					t.Fatalf("not symmetric at distance %d: %d %d %d %d", d, l, r, u, v)
				}
			}
			// The peak is lower than the input's and the spread is wider than one pixel in a blurred direction.
			if tt.sx > 0 && at(out, 30+int(tt.sx), 30)[3] == 0 {
				t.Errorf("no spread to the right")
			}
			if tt.sx == 0 && at(out, 31, 30)[3] != 0 {
				t.Errorf("spread in a direction with a deviation of 0")
			}
			if tt.sy == 0 && at(out, 30, 31)[3] != 0 {
				t.Errorf("spread in a direction with a deviation of 0")
			}
			if in.pix[4*(30*61+30)+3] != 255 {
				t.Error("the input was changed")
			}
		})
	}
	t.Run("a deviation of 0 gives the image", func(t *testing.T) {
		in := delta(9, 9)
		out := in.blurred(0, 0)
		for i := range in.pix {
			if in.pix[i] != out.pix[i] {
				t.Fatal("the image changed")
			}
		}
	})
	t.Run("the edge of the image is transparent beyond", func(t *testing.T) {
		in := newPixbuf(20, 1)
		for i := 0; i < 20; i++ {
			in.pix[4*i], in.pix[4*i+1], in.pix[4*i+2], in.pix[4*i+3] = 255, 255, 255, 255
		}
		out := in.blurred(3, 0)
		if a := at(out, 0, 0)[3]; a > 200 || a < 50 {
			t.Errorf("alpha at the edge = %d, want a fade in", a)
		}
		if a := at(out, 10, 0)[3]; a < 250 {
			t.Errorf("alpha in the middle = %d, want about 255", a)
		}
	})
	t.Run("a huge deviation ends", func(t *testing.T) {
		in := delta(40, 40)
		out := in.blurred(1e12, 1e12)
		if out.w != 40 {
			t.Fatal("size")
		}
	})
}

func TestSlideExtremeMatchesTheDefinition(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for _, dilate := range []bool{true, false} {
		for _, n := range []int{1, 2, 5, 17, 64} {
			for r := 0; r <= 6; r++ {
				line := make([]uint8, n)
				for i := range line {
					line[i] = uint8(rng.Intn(256))
				}
				got := make([]uint8, n)
				slideExtreme(line, got, make([]int32, n), r, dilate)
				for i := 0; i < n; i++ {
					var want uint8
					if !dilate {
						want = 255
					}
					for j := i - r; j <= i+r; j++ {
						v := uint8(0)
						if j >= 0 && j < n {
							v = line[j]
						}
						if dilate && v > want || !dilate && v < want {
							want = v
						}
					}
					if got[i] != want {
						t.Fatalf("dilate=%v n=%d r=%d: res[%d] = %d, want %d (line %v)", dilate, n, r, i, got[i], want, line)
					}
				}
			}
		}
	}
}

func TestMorphology(t *testing.T) {
	dot := newPixbuf(21, 21)
	i := 4 * (10*21 + 10)
	dot.pix[i], dot.pix[i+3] = 255, 255
	d := dot.morphology(3, 2, true)
	count := 0
	for y := 0; y < 21; y++ {
		for x := 0; x < 21; x++ {
			if at(d, x, y)[3] == 255 {
				count++
				if x < 7 || x > 13 || y < 8 || y > 12 {
					t.Fatalf("dilation reached %d,%d", x, y)
				}
			}
		}
	}
	if count != 7*5 {
		t.Errorf("dilated pixels = %d, want %d", count, 7*5)
	}
	block := newPixbuf(21, 21)
	for y := 5; y < 16; y++ {
		for x := 5; x < 16; x++ {
			i := 4 * (y*21 + x)
			block.pix[i], block.pix[i+3] = 255, 255
		}
	}
	e := block.morphology(2, 2, false)
	count = 0
	for y := 0; y < 21; y++ {
		for x := 0; x < 21; x++ {
			if at(e, x, y)[3] == 255 {
				count++
			}
		}
	}
	if count != 7*7 {
		t.Errorf("eroded pixels = %d, want %d", count, 7*7)
	}
	if got := block.morphology(0, 0, true); got == block || at(got, 7, 7) != at(block, 7, 7) {
		t.Error("a radius of 0 changes the image")
	}
}

func TestOffset(t *testing.T) {
	b := newPixbuf(5, 3)
	copy(b.pix[4*(1*5+2):], []byte{255, 0, 0, 255})
	tests := []struct {
		dx, dy int
		x, y   int
		gone   bool
	}{
		{0, 0, 2, 1, false}, {1, 0, 3, 1, false}, {-2, 0, 0, 1, false}, {0, 1, 2, 2, false}, {2, -1, 4, 0, false},
		{3, 0, 0, 0, true}, {-3, 0, 0, 0, true}, {0, 5, 0, 0, true}, {100, 100, 0, 0, true},
	}
	for _, tt := range tests {
		o := b.offset(tt.dx, tt.dy)
		if tt.gone {
			for _, v := range o.pix {
				if v != 0 {
					t.Errorf("offset %d,%d: the pixel is still there", tt.dx, tt.dy)
					break
				}
			}
			continue
		}
		if got := at(o, tt.x, tt.y); got != [4]uint8{255, 0, 0, 255} {
			t.Errorf("offset %d,%d: pixel at %d,%d = %v", tt.dx, tt.dy, tt.x, tt.y, got)
		}
	}
}

func TestColorMatrixTypes(t *testing.T) {
	src := pixelRow([4]uint8{255, 0, 0, 255}, [4]uint8{0, 128, 255, 255}, [4]uint8{50, 100, 150, 255})
	run := func(typ, values string) *pixbuf {
		return src.colorMatrix(colorMatrixOf(&FilterPrimitive{Attrs: map[string]string{"type": typ, "values": values}}))
	}
	tests := []struct {
		name        string
		typ, values string
		pixel       int
		want        [4]uint8
		tol         int
	}{
		{"identity matrix", "matrix", "1 0 0 0 0  0 1 0 0 0  0 0 1 0 0  0 0 0 1 0", 1, [4]uint8{0, 128, 255, 255}, 0},
		{"missing values are the identity", "matrix", "", 1, [4]uint8{0, 128, 255, 255}, 0},
		{"a wrong count of values is the identity", "matrix", "1 2 3", 1, [4]uint8{0, 128, 255, 255}, 0},
		{"swap red and blue", "matrix", "0 0 1 0 0  0 1 0 0 0  1 0 0 0 0  0 0 0 1 0", 0, [4]uint8{0, 0, 255, 255}, 0},
		{"an offset", "matrix", "1 0 0 0 0.5  0 1 0 0 0  0 0 1 0 0  0 0 0 1 0", 1, [4]uint8{128, 128, 255, 255}, 1},
		{"saturate 1 is the identity", "saturate", "1", 2, [4]uint8{50, 100, 150, 255}, 1},
		{"saturate 0 is gray", "saturate", "0", 0, [4]uint8{54, 54, 54, 255}, 1},
		{"saturate with no value is the identity", "saturate", "", 2, [4]uint8{50, 100, 150, 255}, 1},
		{"hueRotate 0 is the identity", "hueRotate", "0", 2, [4]uint8{50, 100, 150, 255}, 1},
		{"hueRotate 360 is the identity", "hueRotate", "360", 2, [4]uint8{50, 100, 150, 255}, 2},
		{"hueRotate 120 turns red toward green, as the matrix of the specification says", "hueRotate", "120", 0, [4]uint8{0, 113, 0, 255}, 1},
		{"luminanceToAlpha of white", "luminanceToAlpha", "", 0, [4]uint8{0, 0, 0, 54}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := at(run(tt.typ, tt.values), tt.pixel, 0)
			if !nearPx(got, tt.want, tt.tol) {
				t.Errorf("pixel = %v, want %v", got, tt.want)
			}
		})
	}
	// A pixel with no alpha stays transparent through a matrix with no offset, and the straight color is what
	// the matrix sees.
	half := pixelRow([4]uint8{128, 0, 0, 128})
	got := at(half.colorMatrix(colorMatrixOf(&FilterPrimitive{Attrs: map[string]string{"type": "matrix", "values": "0 0 1 0 0  0 1 0 0 0  1 0 0 0 0  0 0 0 1 0"}})), 0, 0)
	if !nearPx(got, [4]uint8{0, 0, 128, 128}, 1) {
		t.Errorf("translucent red with red and blue swapped = %v", got)
	}
}

func TestComponentTransferFunctions(t *testing.T) {
	fn := func(typ string, attrs map[string]string) *[256]uint8 {
		a := map[string]string{"type": typ}
		for k, v := range attrs {
			a[k] = v
		}
		return transferFuncs(&FilterPrimitive{Children: []*FilterPrimitive{{Name: "feFuncR", Attrs: a}}})[0]
	}
	tests := []struct {
		name  string
		typ   string
		attrs map[string]string
		x     int
		want  int
	}{
		{"table of two is the identity", "table", map[string]string{"tableValues": "0 1"}, 128, 128},
		{"table inverts", "table", map[string]string{"tableValues": "1 0"}, 0, 255},
		{"table with three values at the middle", "table", map[string]string{"tableValues": "0 1 0"}, 128, 254},
		{"table with one value", "table", map[string]string{"tableValues": "0.5"}, 200, 128},
		{"discrete of three, first third", "discrete", map[string]string{"tableValues": "0 0.5 1"}, 10, 0},
		{"discrete of three, middle third", "discrete", map[string]string{"tableValues": "0 0.5 1"}, 128, 128},
		{"discrete of three, last", "discrete", map[string]string{"tableValues": "0 0.5 1"}, 255, 255},
		{"linear", "linear", map[string]string{"slope": "0.5", "intercept": "0.25"}, 255, 191},
		{"linear clamps", "linear", map[string]string{"slope": "3", "intercept": "0"}, 200, 255},
		{"gamma", "gamma", map[string]string{"amplitude": "1", "exponent": "2", "offset": "0"}, 128, 64},
		{"gamma with offset", "gamma", map[string]string{"amplitude": "0.5", "exponent": "1", "offset": "0.5"}, 255, 255},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := fn(tt.typ, tt.attrs)
			if f == nil {
				t.Fatal("no function")
			}
			if d := int(f[tt.x]) - tt.want; d < -1 || d > 1 {
				t.Errorf("f(%d) = %d, want %d", tt.x, f[tt.x], tt.want)
			}
		})
	}
	for _, typ := range []string{"identity", "bogus", ""} {
		if f := fn(typ, nil); f != nil {
			t.Errorf("type %q makes a function", typ)
		}
	}
	if f := fn("table", nil); f != nil {
		t.Error("a table with no values makes a function")
	}
	if f := fn("discrete", nil); f != nil {
		t.Error("a discrete table with no values makes a function")
	}
	// Alpha works on alpha, and the colors follow the straight values.
	b := pixelRow([4]uint8{100, 100, 100, 255})
	res := b.transfer(transferFuncs(&FilterPrimitive{Children: []*FilterPrimitive{{Name: "feFuncA", Attrs: map[string]string{"type": "linear", "slope": "0.5", "intercept": "0"}}}}))
	if got := at(res, 0, 0); !nearPx(got, [4]uint8{50, 50, 50, 128}, 1) {
		t.Errorf("half alpha = %v", got)
	}
}

func TestCompositeOperators(t *testing.T) {
	a := pixelRow([4]uint8{255, 0, 0, 255}, [4]uint8{128, 0, 0, 128}, [4]uint8{0, 0, 0, 0})
	b := pixelRow([4]uint8{0, 0, 255, 255}, [4]uint8{0, 0, 128, 128}, [4]uint8{0, 255, 0, 255})
	tests := []struct {
		op   string
		k    [4]float64
		want [3][4]uint8
	}{
		{"over", [4]float64{}, [3][4]uint8{{255, 0, 0, 255}, {128, 0, 64, 192}, {0, 255, 0, 255}}},
		{"in", [4]float64{}, [3][4]uint8{{255, 0, 0, 255}, {64, 0, 0, 64}, {0, 0, 0, 0}}},
		{"out", [4]float64{}, [3][4]uint8{{0, 0, 0, 0}, {64, 0, 0, 64}, {0, 0, 0, 0}}},
		{"atop", [4]float64{}, [3][4]uint8{{255, 0, 0, 255}, {64, 0, 64, 128}, {0, 255, 0, 255}}},
		{"xor", [4]float64{}, [3][4]uint8{{0, 0, 0, 0}, {64, 0, 64, 128}, {0, 255, 0, 255}}},
		{"arithmetic", [4]float64{0, 1, 1, 0}, [3][4]uint8{{255, 0, 255, 255}, {128, 0, 128, 255}, {0, 255, 0, 255}}},
		{"arithmetic", [4]float64{1, 0, 0, 0}, [3][4]uint8{{0, 0, 0, 255}, {0, 0, 0, 64}, {0, 0, 0, 0}}},
		{"arithmetic", [4]float64{0, 0, 0, 0.5}, [3][4]uint8{{128, 128, 128, 128}, {128, 128, 128, 128}, {128, 128, 128, 128}}},
	}
	for _, tt := range tests {
		t.Run(tt.op, func(t *testing.T) {
			out := compositeImages(a, b, tt.op, tt.k)
			for i, want := range tt.want {
				tol := 2
				if tt.op == "arithmetic" && tt.k[3] > 0 {
					// Premultiplied colors cannot be above the alpha: the constant 0.5 makes a gray with alpha 0.5.
					want = [4]uint8{128, 128, 128, 128}
				}
				if got := at(out, i, 0); !nearPx(got, want, tol) {
					t.Errorf("pixel %d = %v, want %v", i, got, want)
				}
			}
		})
	}
}

func TestBlendModes(t *testing.T) {
	top := pixelRow([4]uint8{200, 100, 50, 255})
	back := pixelRow([4]uint8{100, 200, 150, 255})
	tests := []struct {
		mode string
		want [4]uint8
		tol  int
	}{
		{"normal", [4]uint8{200, 100, 50, 255}, 1},
		{"unknown names are normal", [4]uint8{200, 100, 50, 255}, 1},
		{"multiply", [4]uint8{78, 78, 29, 255}, 1},
		{"screen", [4]uint8{222, 222, 171, 255}, 1},
		{"darken", [4]uint8{100, 100, 50, 255}, 1},
		{"lighten", [4]uint8{200, 200, 150, 255}, 1},
		{"difference", [4]uint8{100, 100, 100, 255}, 1},
		{"exclusion", [4]uint8{144, 144, 142, 255}, 2},
		{"overlay", [4]uint8{157, 188, 86, 255}, 2},
		{"hard-light", [4]uint8{188, 157, 59, 255}, 2},
		{"color-dodge", [4]uint8{255, 255, 187, 255}, 2},
	}
	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			got := at(blendImages(top, back, tt.mode), 0, 0)
			if !nearPx(got, tt.want, tt.tol) {
				t.Errorf("pixel = %v, want %v", got, tt.want)
			}
		})
	}
	// With a transparent top the backdrop stays, and over a transparent backdrop the top stays, for every mode.
	clear := pixelRow([4]uint8{})
	for _, mode := range []string{"multiply", "screen", "hue", "saturation", "color", "luminosity", "soft-light", "color-burn"} {
		if got := at(blendImages(clear, back, mode), 0, 0); !nearPx(got, at(back, 0, 0), 1) {
			t.Errorf("%s with a transparent top = %v, want the backdrop %v", mode, got, at(back, 0, 0))
		}
		if got := at(blendImages(top, clear, mode), 0, 0); !nearPx(got, at(top, 0, 0), 1) {
			t.Errorf("%s over a transparent backdrop = %v, want the top %v", mode, got, at(top, 0, 0))
		}
	}
	// The non-separable modes keep their defining channel: color takes the hue of the top with the luminosity of the
	// backdrop.
	got := at(blendImages(pixelRow([4]uint8{255, 0, 0, 255}), pixelRow([4]uint8{128, 128, 128, 255}), "color"), 0, 0)
	if got[0] <= got[1] || got[1] != got[2] {
		t.Errorf("color blend of red on gray = %v, want a red tint", got)
	}
}

func TestMergeFloodAndClip(t *testing.T) {
	a := pixelRow([4]uint8{255, 0, 0, 255}, [4]uint8{128, 0, 0, 128})
	b := pixelRow([4]uint8{0, 0, 255, 255}, [4]uint8{0, 0, 255, 255})
	m := mergeImages(2, 1, []*pixbuf{b, a})
	if got := at(m, 0, 0); got != [4]uint8{255, 0, 0, 255} {
		t.Errorf("opaque over = %v", got)
	}
	if got := at(m, 1, 0); !nearPx(got, [4]uint8{128, 0, 127, 255}, 1) {
		t.Errorf("half over = %v", got)
	}
	if got := mergeImages(2, 1, nil); at(got, 0, 0) != [4]uint8{} {
		t.Errorf("empty merge = %v", at(got, 0, 0))
	}
	f := flooded(4, 3, 255, 0, 0, 0.5, 1, 1, 3, 2)
	if got := at(f, 1, 1); !nearPx(got, [4]uint8{128, 0, 0, 128}, 1) {
		t.Errorf("flood = %v", got)
	}
	if got := at(f, 0, 0); got != [4]uint8{} {
		t.Errorf("outside the flood = %v", got)
	}
	if got := at(f, 3, 1); got != [4]uint8{} {
		t.Errorf("outside the flood on the right = %v", got)
	}
	f.clipTo(2, 0, 100, 100)
	if got := at(f, 1, 1); got != [4]uint8{} {
		t.Errorf("clipped away = %v", got)
	}
	if got := at(f, 2, 1); got[3] == 0 {
		t.Errorf("clip kept nothing")
	}
	// A clip with the corners the wrong way round, or outside the image, empties it and does not panic.
	f.clipTo(10, 10, 0, 0)
	f.clipTo(-5, -5, -1, -1)
	if got := at(f, 2, 1); got != [4]uint8{} {
		t.Errorf("clip outside left %v", got)
	}
}

func TestAlphaOnlyAndColored(t *testing.T) {
	b := pixelRow([4]uint8{100, 50, 25, 200}, [4]uint8{})
	a := b.alphaOnly()
	if got := at(a, 0, 0); got != [4]uint8{0, 0, 0, 200} {
		t.Errorf("alphaOnly = %v", got)
	}
	c := b.colored(255, 0, 0)
	if got := at(c, 0, 0); !nearPx(got, [4]uint8{200, 0, 0, 200}, 1) {
		t.Errorf("colored = %v", got)
	}
	if got := at(c, 1, 0); got != [4]uint8{} {
		t.Errorf("colored transparent = %v", got)
	}
}
