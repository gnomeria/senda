package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// paneFocus renders the minimal focus-mode view: a breadcrumb, the URL row, a
// one-line params/headers/auth summary, then the response — no tree or tabs.
func (m tuiModel) paneFocus(w, h int) string {
	if !m.loaded {
		return paneBlock([]string{base.Render(""), styleDim.Render("select a request (^K to jump)")}, w, h)
	}
	env := m.envName()
	if env == "" {
		env = "no env"
	}
	bcLeft := base.Foreground(colAccent).Render("◆ "+env) + styleDim.Render(" · "+m.breadcrumb())
	bcRight := styleDim.Render("^K to jump")
	bpad := w - lipgloss.Width(stripStyle(bcLeft)) - lipgloss.Width(stripStyle(bcRight))
	if bpad < 1 {
		bpad = 1
	}
	bc := bcLeft + base.Render(strings.Repeat(" ", bpad)) + bcRight

	lines := []string{base.Render(""), bc, base.Render(""), m.urlRow(w), m.focusSummary(), base.Render(""), m.focusStatus(w)}
	if m.resp != nil {
		body := numberedCode(prettyJSON(m.resp.Body))
		lines = append(lines, strings.Split(body, "\n")...)
	}
	return paneBlock(lines, w, h)
}

// focusSummary builds the compact "params … · +N header · Bearer" line.
func (m tuiModel) focusSummary() string {
	var b strings.Builder
	b.WriteString(styleDim.Render("params "))
	var ps []string
	for _, p := range m.cur.Params {
		if !p.Enabled {
			continue
		}
		ps = append(ps, base.Foreground(colAccent).Render(p.Key)+styleDim.Render("=")+base.Foreground(colFg).Render(p.Value))
	}
	if len(ps) == 0 {
		b.WriteString(styleDim.Render("(none)"))
	} else {
		b.WriteString(strings.Join(ps, base.Render(" ")))
	}
	if n := len(m.cur.Headers); n > 0 {
		b.WriteString(styleDim.Render(fmt.Sprintf("  ·  +%d header", n)))
	}
	if a := m.reqBadges()[3]; a != "" {
		b.WriteString(styleDim.Render("  ·  " + a))
	}
	return b.String()
}

// focusStatus renders the response status line for the focus view.
func (m tuiModel) focusStatus(w int) string {
	switch {
	case m.respErr != "":
		return base.Foreground(colBad).Render("● error: " + truncate(m.respErr, w-10))
	case m.sending:
		return base.Foreground(colWarn).Render("● ") + styleDim.Render("sending…")
	case m.resp != nil:
		sc := statusColor(m.resp.Status)
		return base.Foreground(sc).Bold(true).Render(fmt.Sprintf("● %d %s", m.resp.Status, m.resp.StatusText)) +
			styleDim.Render(fmt.Sprintf("   time %d ms   size %s", m.resp.DurationMs, humanBytes(m.resp.SizeBytes)))
	default:
		return styleDim.Render("no response yet — press s to send")
	}
}
