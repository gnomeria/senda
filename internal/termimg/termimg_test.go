package termimg

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"testing"
)

// parser tests construct a bare Renderer (no fonts) since parse only reads opt.
func testParser() *Renderer {
	return &Renderer{opt: Options{
		CellW: 8, CellH: 16,
		DefaultFg: color.RGBA{200, 200, 200, 255},
		DefaultBg: color.RGBA{16, 16, 16, 255},
	}}
}

func TestParseTruecolorSGR(t *testing.T) {
	r := testParser()
	// bold + truecolor red fg on blue bg for "AB", then reset, then "C".
	g := r.parse("\x1b[1;38;2;255;0;0;48;2;0;0;255mAB\x1b[0mC")

	a := g.at(0, 0)
	if a.r != 'A' || !a.bold {
		t.Fatalf("cell A: got r=%q bold=%v", a.r, a.bold)
	}
	if a.fg != (color.RGBA{255, 0, 0, 255}) {
		t.Errorf("cell A fg = %v, want red", a.fg)
	}
	if a.bg != (color.RGBA{0, 0, 255, 255}) {
		t.Errorf("cell A bg = %v, want blue", a.bg)
	}
	if b := g.at(1, 0); b.r != 'B' || b.fg != (color.RGBA{255, 0, 0, 255}) {
		t.Errorf("cell B not carried: %+v", b)
	}
	c := g.at(2, 0)
	if c.r != 'C' || c.bold {
		t.Errorf("cell C: got r=%q bold=%v, want reset", c.r, c.bold)
	}
	if c.fg != r.opt.DefaultFg || c.bg != r.opt.DefaultBg {
		t.Errorf("cell C colours not reset to defaults: fg=%v bg=%v", c.fg, c.bg)
	}
}

func TestParseIgnoresNonSGR(t *testing.T) {
	r := testParser()
	// Cursor move + clear sequences should be skipped, not rendered as text.
	g := r.parse("\x1b[2J\x1b[10;5HX\x1b[K")
	if g.at(0, 0).r != 'X' {
		t.Fatalf("expected X at 0,0, got %q (cols=%d)", g.at(0, 0).r, g.cols)
	}
	if g.cols != 1 {
		t.Errorf("non-SGR sequences leaked into the grid: cols=%d", g.cols)
	}
}

func TestParseNewlineAdvancesRow(t *testing.T) {
	r := testParser()
	g := r.parse("a\nbb")
	if g.at(0, 0).r != 'a' {
		t.Errorf("row 0 col 0 = %q", g.at(0, 0).r)
	}
	if g.at(0, 1).r != 'b' || g.at(1, 1).r != 'b' {
		t.Errorf("row 1 not populated: %q %q", g.at(0, 1).r, g.at(1, 1).r)
	}
}

func TestParseWideRune(t *testing.T) {
	r := testParser()
	g := r.parse("🔒X") // lock is width 2; X must land at column 2
	if got := g.at(0, 0).r; got != '🔒' {
		t.Errorf("col 0 = %q, want lock", got)
	}
	if got := g.at(2, 0).r; got != 'X' {
		t.Errorf("col 2 = %q, want X (wide rune should occupy 2 columns)", got)
	}
}

func TestBuildPaletteBounded(t *testing.T) {
	// 4096 distinct colours must collapse to ≤ 256 palette entries, index 0
	// reserved transparent.
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	i := 0
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			img.Pix[i] = uint8(x * 4)
			img.Pix[i+1] = uint8(y * 4)
			img.Pix[i+2] = uint8((x + y) * 2)
			img.Pix[i+3] = 255
			i += 4
		}
	}
	pal := buildPalette([]*image.RGBA{img}, 255)
	if len(pal) > 256 {
		t.Fatalf("palette too large: %d", len(pal))
	}
	if _, _, _, a := pal[0].RGBA(); a != 0 {
		t.Errorf("palette[0] must be transparent, got alpha %d", a)
	}
}

// newOrSkip builds a real Renderer or skips when the dev fonts aren't installed
// (e.g. minimal CI images without fonts-dejavu-core).
func newOrSkip(t *testing.T) *Renderer {
	t.Helper()
	r, err := New(Defaults())
	if err != nil {
		t.Skipf("fonts unavailable, skipping render test: %v", err)
	}
	return r
}

func TestRenderImageSmoke(t *testing.T) {
	r := newOrSkip(t)
	img := r.Image("\x1b[38;2;255;0;0mhello\x1b[0m")
	if img.Bounds().Dx() == 0 || img.Bounds().Dy() == 0 {
		t.Fatal("empty image")
	}
}

func TestGIFSmoke(t *testing.T) {
	r := newOrSkip(t)
	var buf bytes.Buffer
	frames := []Frame{
		{ANSI: "\x1b[38;2;255;0;0mone\x1b[0m", Delay: 30},
		{ANSI: "\x1b[38;2;0;255;0mtwo\x1b[0m", Delay: 30},
	}
	if err := r.GIF(&buf, frames); err != nil {
		t.Fatal(err)
	}
	g, err := gif.DecodeAll(&buf)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(g.Image) != 2 {
		t.Errorf("frames = %d, want 2", len(g.Image))
	}
}
