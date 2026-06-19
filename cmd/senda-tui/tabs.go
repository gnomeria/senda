package main

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// tabStripHit maps a click x-offset (relative to the tab strip's start) to a tab
// index, mirroring tabStrip's segment widths (" name " + optional " badge " pill
// + trailing space). Returns -1 if the click is past the last tab.
func tabStripHit(names, badges []string, localX int) int {
	x := 0
	for i, n := range names {
		segW := lipgloss.Width(n) + 2
		if i < len(badges) && badges[i] != "" {
			segW += lipgloss.Width(badges[i]) + 2 + 1
		}
		if localX >= x && localX < x+segW {
			return i
		}
		x += segW
	}
	return -1
}

// tabStrip renders mockup-style tabs: the active tab in accent, others dim, each
// optionally carrying a small rounded badge (a count or a label like "Bearer").
func tabStrip(names []string, badges []string, active int) string {
	var parts []string
	for i, n := range names {
		on := i == active
		nameStyle := styleDim
		if on {
			nameStyle = base.Foreground(colAccent).Bold(true).Underline(true)
		}
		seg := nameStyle.Render(" " + n + " ")
		if i < len(badges) && badges[i] != "" {
			seg += tabBadge(badges[i], on) + base.Render(" ")
		}
		parts = append(parts, seg)
	}
	row := strings.Join(parts, "")
	// Underline accent beneath the active tab label, matching the mockups.
	return row
}

// tabBadge renders a tab's count/label as a small pill on the input background.
func tabBadge(text string, active bool) string {
	fg := colDim
	if active {
		fg = colAccent
	}
	return base.Background(bgInput).Foreground(fg).Render(" " + text + " ")
}

func countBadge(n int) string {
	if n <= 0 {
		return ""
	}
	return fmt.Sprintf("%d", n)
}
