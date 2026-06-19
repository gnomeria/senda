package termimg

import (
	"image"
	"image/color"
	"image/draw"
)

// drawBox strokes the line/corner runes the TUI uses for panel borders
// (lipgloss RoundedBorder) as filled rectangles, so adjacent cells join with no
// sub-pixel gaps the way font glyphs would. It returns false for runes it does
// not handle, so the caller falls back to the font.
func (r *Renderer) drawBox(img *image.RGBA, x0, y0 int, fg color.RGBA, ch rune) bool {
	cw, chh := r.opt.CellW, r.opt.CellH
	t := 1 // light stroke thickness
	if ch == '━' || ch == '┃' {
		t = 2 // heavy
	}
	vx := x0 + (cw-t)/2  // left edge of a vertical stroke (centered)
	hy := y0 + (chh-t)/2 // top edge of a horizontal stroke (centered)
	xL, xR := x0, x0+cw
	yT, yB := y0, y0+chh

	fill := func(a, b, c, d int) {
		draw.Draw(img, image.Rect(a, b, c, d), &image.Uniform{fg}, image.Point{}, draw.Src)
	}

	switch ch {
	case '─', '━': // horizontal
		fill(xL, hy, xR, hy+t)
	case '│', '┃': // vertical
		fill(vx, yT, vx+t, yB)
	case '╭': // ┌ rounded: connects right + down
		fill(vx, hy, xR, hy+t)
		fill(vx, hy, vx+t, yB)
	case '╮': // ┐ : connects left + down
		fill(xL, hy, vx+t, hy+t)
		fill(vx, hy, vx+t, yB)
	case '╰': // └ : connects right + up
		fill(vx, hy, xR, hy+t)
		fill(vx, yT, vx+t, hy+t)
	case '╯': // ┘ : connects left + up
		fill(xL, hy, vx+t, hy+t)
		fill(vx, yT, vx+t, hy+t)
	case '┌':
		fill(vx, hy, xR, hy+t)
		fill(vx, hy, vx+t, yB)
	case '┐':
		fill(xL, hy, vx+t, hy+t)
		fill(vx, hy, vx+t, yB)
	case '└':
		fill(vx, hy, xR, hy+t)
		fill(vx, yT, vx+t, hy+t)
	case '┘':
		fill(xL, hy, vx+t, hy+t)
		fill(vx, yT, vx+t, hy+t)
	case '├':
		fill(vx, yT, vx+t, yB)
		fill(vx, hy, xR, hy+t)
	case '┤':
		fill(vx, yT, vx+t, yB)
		fill(xL, hy, vx+t, hy+t)
	case '┬':
		fill(xL, hy, xR, hy+t)
		fill(vx, hy, vx+t, yB)
	case '┴':
		fill(xL, hy, xR, hy+t)
		fill(vx, yT, vx+t, hy+t)
	case '┼':
		fill(xL, hy, xR, hy+t)
		fill(vx, yT, vx+t, yB)
	case '▏': // left one-eighth block (thin accent / cursor bar)
		w := cw / 8
		if w < 1 {
			w = 1
		}
		fill(xL, yT, xL+w, yB)
	default:
		return false
	}
	return true
}
