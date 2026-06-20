package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"senda/internal/model"
)

// countPasses returns how many assert results passed.
func countPasses(rs []model.AssertResult) int {
	pass := 0
	for _, r := range rs {
		if r.Pass {
			pass++
		}
	}
	return pass
}

func max0(n int) int {
	if n < 0 {
		return 0
	}
	return n
}

// atLeast clamps n up to min, used to enforce minimum pane sizes.
func atLeast(n, min int) int {
	if n < min {
		return min
	}
	return n
}

// centerPad centers s within width w (all spaces when s is empty).
func centerPad(s string, w int) string {
	gap := w - lipgloss.Width(s)
	if gap <= 0 {
		return s
	}
	left := gap / 2
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", gap-left)
}

func mask(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 4 {
		return "••••"
	}
	return "••••" + s[len(s)-4:]
}

func assertSummary(rs []model.AssertResult) string {
	if len(rs) == 0 {
		return ""
	}
	pass := countPasses(rs)
	col := colGood
	if pass < len(rs) {
		col = colBad
	}
	return base.Foreground(col).Render(fmt.Sprintf("asserts %d/%d", pass, len(rs)))
}

func humanBytes(n int64) string {
	switch {
	case n < 1024:
		return fmt.Sprintf("%d B", n)
	case n < 1024*1024:
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	default:
		return fmt.Sprintf("%.1f MB", float64(n)/(1024*1024))
	}
}

// countCookies counts Set-Cookie headers (case-insensitive).
func countCookies(h map[string][]string) int {
	n := 0
	for k, vals := range h {
		if strings.EqualFold(k, "Set-Cookie") {
			n += len(vals)
		}
	}
	return n
}

// truncate shortens s to a printable width of w, preserving ANSI styling so
// styled spans are never cut mid-escape (which would corrupt the line).
func truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= w {
		return s
	}
	if w <= 1 {
		return "…"
	}
	return ansi.Truncate(s, w, "…")
}

func padRight(s string, w int) string {
	for lipgloss.Width(s) < w {
		s += " "
	}
	return s
}
