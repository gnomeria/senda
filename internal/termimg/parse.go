package termimg

import (
	"image/color"
	"strconv"
	"strings"
	"unicode/utf8"
)

// cell is one terminal grid cell after SGR resolution.
type cell struct {
	r                               rune
	fg, bg                          color.RGBA
	bold, underline, faint, reverse bool
}

// grid is a rectangular cell buffer grown on demand.
type grid struct {
	rows, cols int
	cells      []cell
	def        cell
}

func (g *grid) ensure(row, col int) {
	if col >= g.cols || row >= g.rows {
		nc, nr := g.cols, g.rows
		if col >= nc {
			nc = col + 1
		}
		if row >= nr {
			nr = row + 1
		}
		next := make([]cell, nr*nc)
		for i := range next {
			next[i] = g.def
		}
		for y := 0; y < g.rows; y++ {
			copy(next[y*nc:y*nc+g.cols], g.cells[y*g.cols:y*g.cols+g.cols])
		}
		g.cells, g.rows, g.cols = next, nr, nc
	}
}

func (g *grid) set(row, col int, c cell) {
	g.ensure(row, col)
	g.cells[row*g.cols+col] = c
}

func (g *grid) at(col, row int) cell {
	if row < 0 || row >= g.rows || col < 0 || col >= g.cols {
		return g.def
	}
	return g.cells[row*g.cols+col]
}

// parse walks an ANSI string and builds the cell grid, applying SGR colour and
// attribute sequences and ignoring other control sequences (cursor moves,
// erases, OSC, …) that lipgloss/Bubble Tea may interleave.
func (r *Renderer) parse(s string) *grid {
	def := cell{r: ' ', fg: r.opt.DefaultFg, bg: r.opt.DefaultBg}
	g := &grid{def: def}
	cur := def
	row, col := 0, 0

	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c == 0x1b: // ESC
			if i+1 < len(s) && s[i+1] == '[' { // CSI
				j := i + 2
				for j < len(s) && !(s[j] >= 0x40 && s[j] <= 0x7e) {
					j++
				}
				if j >= len(s) {
					i = len(s)
					continue
				}
				if s[j] == 'm' {
					applySGR(&cur, def, s[i+2:j])
				}
				i = j + 1
				continue
			}
			if i+1 < len(s) && s[i+1] == ']' { // OSC: skip to BEL or ST
				j := i + 2
				for j < len(s) && s[j] != 0x07 {
					if s[j] == 0x1b && j+1 < len(s) && s[j+1] == '\\' {
						j++
						break
					}
					j++
				}
				i = j + 1
				continue
			}
			i++ // lone ESC / other two-byte escapes
			continue
		case c == '\n':
			row++
			col = 0
			i++
			continue
		case c == '\r':
			col = 0
			i++
			continue
		case c == '\t':
			col += 4 - col%4
			i++
			continue
		}
		ch, size := utf8.DecodeRuneInString(s[i:])
		i += size
		if ch == utf8.RuneError && size == 1 {
			continue
		}
		w := runewidthOr1(ch)
		nc := cur
		nc.r = ch
		g.set(row, col, nc)
		// Blank the spanned-over cell of a wide rune so nothing else lands there.
		if w == 2 {
			spacer := cur
			spacer.r = 0
			g.set(row, col+1, spacer)
		}
		col += w
	}
	return g
}

// applySGR mutates cur per the parameters of an SGR (…m) sequence.
func applySGR(cur *cell, def cell, params string) {
	if params == "" {
		params = "0"
	}
	toks := strings.Split(params, ";")
	for i := 0; i < len(toks); i++ {
		n, _ := strconv.Atoi(toks[i])
		switch {
		case n == 0:
			*cur = cell{r: cur.r, fg: def.fg, bg: def.bg}
		case n == 1:
			cur.bold = true
		case n == 2:
			cur.faint = true
		case n == 4:
			cur.underline = true
		case n == 7:
			cur.reverse = true
		case n == 22:
			cur.bold, cur.faint = false, false
		case n == 24:
			cur.underline = false
		case n == 27:
			cur.reverse = false
		case n >= 30 && n <= 37:
			cur.fg = ansi16[n-30]
		case n == 38:
			if c, adv, ok := readExtended(toks, i); ok {
				cur.fg = c
				i += adv
			}
		case n == 39:
			cur.fg = def.fg
		case n >= 40 && n <= 47:
			cur.bg = ansi16[n-40]
		case n == 48:
			if c, adv, ok := readExtended(toks, i); ok {
				cur.bg = c
				i += adv
			}
		case n == 49:
			cur.bg = def.bg
		case n >= 90 && n <= 97:
			cur.fg = ansi16[8+n-90]
		case n >= 100 && n <= 107:
			cur.bg = ansi16[8+n-100]
		}
	}
}

// readExtended handles the "5;n" (256-colour) and "2;r;g;b" (truecolor) forms
// following a 38/48 token at index i, returning the colour and how many extra
// tokens it consumed.
func readExtended(toks []string, i int) (color.RGBA, int, bool) {
	if i+1 >= len(toks) {
		return color.RGBA{}, 0, false
	}
	mode := toks[i+1]
	switch mode {
	case "2":
		if i+4 < len(toks) {
			r, _ := strconv.Atoi(toks[i+2])
			g, _ := strconv.Atoi(toks[i+3])
			b, _ := strconv.Atoi(toks[i+4])
			return color.RGBA{uint8(r), uint8(g), uint8(b), 0xff}, 4, true
		}
	case "5":
		if i+2 < len(toks) {
			idx, _ := strconv.Atoi(toks[i+2])
			return ansi256(idx), 2, true
		}
	}
	return color.RGBA{}, 0, false
}

// ansi16 is a standard 16-colour terminal palette (used only if the TUI ever
// emits basic colours; it overwhelmingly uses truecolor).
var ansi16 = [16]color.RGBA{
	{0x1d, 0x1f, 0x21, 0xff}, {0xcc, 0x66, 0x66, 0xff}, {0xb5, 0xbd, 0x68, 0xff}, {0xf0, 0xc6, 0x74, 0xff},
	{0x81, 0xa2, 0xbe, 0xff}, {0xb2, 0x94, 0xbb, 0xff}, {0x8a, 0xbe, 0xb7, 0xff}, {0xc5, 0xc8, 0xc6, 0xff},
	{0x66, 0x66, 0x66, 0xff}, {0xd5, 0x4e, 0x53, 0xff}, {0xb9, 0xca, 0x4a, 0xff}, {0xe7, 0xc5, 0x47, 0xff},
	{0x7a, 0xa6, 0xda, 0xff}, {0xc3, 0x97, 0xd8, 0xff}, {0x70, 0xc0, 0xb1, 0xff}, {0xee, 0xee, 0xee, 0xff},
}

// ansi256 maps an xterm-256 index to RGB.
func ansi256(i int) color.RGBA {
	switch {
	case i < 16:
		return ansi16[i]
	case i < 232:
		i -= 16
		levels := []uint8{0, 0x5f, 0x87, 0xaf, 0xd7, 0xff}
		return color.RGBA{levels[(i/36)%6], levels[(i/6)%6], levels[i%6], 0xff}
	default:
		v := uint8(8 + (i-232)*10)
		return color.RGBA{v, v, v, 0xff}
	}
}
