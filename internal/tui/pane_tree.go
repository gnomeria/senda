package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"senda/internal/model"
)

// countRequests returns the number of request files at or below a tree node.
func countRequests(n *model.TreeNode) int {
	if n == nil {
		return 0
	}
	if !n.IsDir {
		return 1
	}
	total := 0
	for _, c := range n.Children {
		total += countRequests(c)
	}
	return total
}

func (m tuiModel) paneTree(w, h int) string {
	header := m.paneLabel("COLLECTIONS", m.focus == focusTree) +
		styleDim.Render(fmt.Sprintf("  %d", countRequests(m.coll.Tree)))
	lines := []string{header, base.Render("")}
	// Vertical scroll window: show rows [treeOff, treeOff+capacity).
	capacity := h - 2
	if capacity < 1 {
		capacity = 1
	}
	start := m.treeOff
	if start < 0 {
		start = 0
	}
	end := start + capacity
	if end > len(m.rows) {
		end = len(m.rows)
	}
	for i := start; i < end; i++ {
		r := m.rows[i]
		sel := i == m.cursor
		bg := bgPanel
		if sel {
			bg = bgSel
		}
		st := base.Background(bg)
		indent := strings.Repeat("  ", r.depth)
		var label string
		if r.node.IsDir {
			arrow := "▸"
			if m.expanded[r.node.Path] {
				arrow = "▾"
			}
			nameStyle := st.Foreground(colFg)
			if r.depth == 0 {
				nameStyle = nameStyle.Bold(true)
			}
			label = st.Foreground(colDim).Render(indent+arrow+" ") + nameStyle.Render(r.node.Name)
			// Right-aligned request-count badge for folders (not the root).
			if r.depth > 0 {
				badge := st.Foreground(colDim).Render(fmt.Sprintf("%d", countRequests(r.node)))
				pad := w - lipgloss.Width(stripStyle(label)) - lipgloss.Width(stripStyle(badge))
				if pad > 0 {
					label += st.Render(strings.Repeat(" ", pad)) + badge
				}
			}
		} else {
			method := strings.ToUpper(r.node.Method)
			if method == "" {
				method = "—"
			}
			name := strings.TrimSuffix(r.node.Name, ".yaml")
			label = st.Render(indent) +
				st.Foreground(methodColor(method)).Bold(true).Render(fmt.Sprintf("%-6s ", method)) +
				st.Foreground(colFg).Render(name)
		}
		if sel {
			plainW := lipgloss.Width(stripStyle(label))
			if plainW < w {
				label += st.Render(strings.Repeat(" ", w-plainW))
			}
		}
		lines = append(lines, label)
	}
	return paneBlock(lines, w, h)
}
