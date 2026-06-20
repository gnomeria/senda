package tui

import (
	"fmt"
	"image/color"
	"regexp"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
)

// Dark "Tokyo Night"-ish palette matching the senda mockups.
var (
	bgApp     = lipgloss.Color("#15161a") // app background (outer chrome)
	bgPanel   = lipgloss.Color("#191b20") // pane background (subtle card lift)
	bgInput   = lipgloss.Color("#101116") // URL field / input chips (dark inset)
	bgSel     = lipgloss.Color("#20232b") // selected row / active tab strip
	colFg     = lipgloss.Color("#a9b1d6")
	colSubtle = lipgloss.Color("#3b4261") // separators / faint
	colDim    = lipgloss.Color("#565f89") // labels / secondary text
	colAccent = lipgloss.Color("#7aa2f7") // blue accent
	colGood   = lipgloss.Color("#9ece6a") // 2xx / GET
	colWarn   = lipgloss.Color("#e0af68") // 3xx / POST
	colBad    = lipgloss.Color("#f7768e") // 4xx / DELETE
	colCyan   = lipgloss.Color("#7dcfff")
	colSendBg = lipgloss.Color("#2e8b57")
	colBorder = lipgloss.Color("#24272e") // subtle rounded-card border stroke

	// base carries the panel background so every styled span paints uniformly.
	base       = lipgloss.NewStyle().Background(bgPanel).Foreground(colFg)
	styleDim   = base.Foreground(colDim)
	styleTitle = base.Foreground(colAccent).Bold(true)
	styleSel   = base.Background(bgSel).Foreground(colFg)
	styleRule  = lipgloss.NewStyle().Background(bgPanel).Foreground(colSubtle)

	// appbg paints the outer app-chrome background (status bar, gaps, margins).
	appbg = lipgloss.NewStyle().Background(bgApp)

	// Overlay popups keep a rounded border on the panel background.
	styleBorder    = lipgloss.NewStyle().Background(bgPanel).Foreground(colFg).Border(lipgloss.RoundedBorder()).BorderForeground(colSubtle).BorderBackground(bgPanel)
	styleBorderFoc = lipgloss.NewStyle().Background(bgPanel).Foreground(colFg).Border(lipgloss.RoundedBorder()).BorderForeground(colAccent).BorderBackground(bgPanel)
)

// ansiRe strips SGR sequences so a styled string can be re-measured/re-styled.
var ansiRe = regexp.MustCompile("\x1b\\[[0-9;?]*[a-zA-Z]")

func stripStyle(s string) string { return ansiRe.ReplaceAllString(s, "") }

// sgrRe matches a single SGR sequence (…m) so its color params can be rewritten.
var sgrRe = regexp.MustCompile("\x1b\\[([0-9;]*)m")

// dimANSI blends every truecolor fg/bg in s toward the app background so the
// live screen recedes behind a modal overlay, matching the mockups' dimmed
// backdrop. factor 0 = unchanged, 1 = fully background.
func dimANSI(s string, factor float64) string {
	br, bg, bb := 13.0, 15.0, 17.0 // dim backdrop #0d0f11 (darker than app bg)
	blend := func(v float64, b float64) string {
		return fmt.Sprintf("%d", int(v+(b-v)*factor))
	}
	return sgrRe.ReplaceAllStringFunc(s, func(m string) string {
		parts := strings.Split(m[2:len(m)-1], ";")
		out := make([]string, 0, len(parts))
		for i := 0; i < len(parts); i++ {
			if (parts[i] == "38" || parts[i] == "48") && i+4 < len(parts) && parts[i+1] == "2" {
				r, _ := strconv.Atoi(parts[i+2])
				g, _ := strconv.Atoi(parts[i+3])
				bl, _ := strconv.Atoi(parts[i+4])
				out = append(out, parts[i], "2",
					blend(float64(r), br), blend(float64(g), bg), blend(float64(bl), bb))
				i += 4
				continue
			}
			out = append(out, parts[i])
		}
		return "\x1b[" + strings.Join(out, ";") + "m"
	})
}

func methodColor(method string) color.Color {
	switch strings.ToUpper(method) {
	case "GET":
		return colGood
	case "POST", "PUT":
		return colWarn
	case "PATCH":
		return colAccent
	case "DELETE":
		return colBad
	case "WS":
		return colCyan
	default:
		return colDim
	}
}

func statusColor(code int) color.Color {
	switch {
	case code >= 200 && code < 300:
		return colGood
	case code >= 300 && code < 400:
		return colWarn
	case code >= 400:
		return colBad
	default:
		return colDim
	}
}

// card wraps a flush pane block in a rounded border on the app background so the
// panels read as separated cards, matching the mockups. The border brightens
// slightly when the pane is focused.
func card(body string, focused bool) string {
	bc := colBorder
	if focused {
		bc = colSubtle
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(bc).BorderBackground(bgApp).
		Render(body)
}

// vstrip returns an h-row block of w app-background spaces (a gap/margin column).
func vstrip(w, h int) string {
	row := appbg.Render(strings.Repeat(" ", max0(w)))
	rows := make([]string, h)
	for i := range rows {
		rows[i] = row
	}
	return strings.Join(rows, "\n")
}

// cardsRow joins panel cards horizontally with app-bg gaps between them and an
// outer margin on each side, padded to the full screen width. h is the card-area
// (outer card) height so the gap/margin columns line up exactly.
func (m tuiModel) cardsRow(h int, cards ...string) string {
	var parts []string
	if outerMargin > 0 {
		parts = append(parts, vstrip(outerMargin, h))
	}
	for i, c := range cards {
		if i > 0 && paneGap > 0 {
			parts = append(parts, vstrip(paneGap, h))
		}
		parts = append(parts, c)
	}
	if outerMargin > 0 {
		parts = append(parts, vstrip(outerMargin, h))
	}
	row := lipgloss.JoinHorizontal(lipgloss.Top, parts...)
	return appbg.Width(m.w).Render(row)
}

// fillRight truncates line to width w then pads the remainder with panel-bg
// spaces emitted as their OWN styled span. This survives inner SGR resets
// (e.g. from syntax-highlighted JSON) that would otherwise leave the padding
// painted in the terminal's default background (black). Wrapping the whole line
// in a bg style does not work: a reset mid-line clears the ambient background.
func fillRight(line string, w int) string {
	line = truncate(line, w)
	if pad := w - lipgloss.Width(line); pad > 0 {
		line += base.Render(strings.Repeat(" ", pad))
	}
	return line
}

// blankLine returns a full-width panel-bg row.
func blankLine(w int) string { return base.Render(strings.Repeat(" ", max0(w))) }

// padBetween lays left flush-left and right flush-right across width w, filling
// the gap (at least one cell) with fill-styled spaces. lipgloss.Width already
// ignores SGR sequences, so styled spans measure by their visible width.
func padBetween(left, right string, w int, fill lipgloss.Style) string {
	pad := w - lipgloss.Width(left) - lipgloss.Width(right)
	if pad < 1 {
		pad = 1
	}
	return left + fill.Render(strings.Repeat(" ", pad)) + right
}

// paneBlock pads each line to width w on the panel bg and forces exactly h
// rows, producing a flush pane fully painted in the panel background.
func paneBlock(lines []string, w, h int) string {
	out := make([]string, 0, h)
	for i := 0; i < h; i++ {
		if i < len(lines) {
			out = append(out, fillRight(lines[i], w))
		} else {
			out = append(out, blankLine(w))
		}
	}
	return strings.Join(out, "\n")
}

// padContent fills viewport content so every cell carries the panel background:
// each line is padded to width w and the block to at least h rows. Without this
// the viewport's internal width/height padding uses an empty style and renders
// black. Content longer than h is left intact (the viewport scrolls it).
func padContent(s string, w, h int) string {
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = fillRight(lines[i], w)
	}
	for len(lines) < h {
		lines = append(lines, blankLine(w))
	}
	return strings.Join(lines, "\n")
}
