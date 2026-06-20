package tui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"senda/internal/codegen"
)

// codegenTab is one language tab in the codegen pane. sub is a small secondary
// label (e.g. "fetch" under JavaScript); target maps to codegen.Generate, empty
// for languages without a generator yet.
type codegenTab struct {
	label  string
	sub    string
	target string
}

var codegenTabs = []codegenTab{
	{"cURL", "", "curl"},
	{"JavaScript", "fetch", "fetch"},
	{"Python", "httpx", "python"},
	{"Go", "", "go"},
	{"Rust", "", ""},
	{"HTTPie", "", "httpie"},
}

// paneCodegen renders the GENERATE CODE pane (shown in place of the response
// pane while exporting): language tabs, the numbered snippet with non-secret
// variables resolved, and a copy/save footer — matching the codegen mockup.
func (m tuiModel) paneCodegen(w, h int) string {
	focused := m.focus == focusResp
	tab := codegenTabs[m.exportIdx]

	var tabsB, subB strings.Builder
	for i, t := range codegenTabs {
		seg := " " + t.label + " "
		segW := lipgloss.Width(seg)
		if i == m.exportIdx {
			tabsB.WriteString(base.Background(bgInput).Foreground(colAccent).Bold(true).Render(seg))
		} else {
			tabsB.WriteString(styleDim.Render(seg))
		}
		subB.WriteString(styleDim.Render(centerPad(t.sub, segW)))
	}

	var code string
	switch {
	case tab.target == "":
		code = styleDim.Render("(" + tab.label + " export not available)")
	default:
		raw, err := codegen.Generate(m.cur, tab.target)
		if err != nil {
			code = base.Foreground(colBad).Render(err.Error())
		} else {
			code = numberedShell(m.resolveNonSecret(raw))
		}
	}

	lines := []string{
		m.paneLabel("GENERATE CODE", focused) + styleDim.Render("  ⇧S"),
		tabsB.String(),
		subB.String(),
		base.Render(""),
	}
	lines = append(lines, strings.Split(code, "\n")...)

	env := m.envName()
	if env == "" {
		env = "none"
	}
	footL := keyHintBg("yy", "copy") + base.Render("   ") + keyHintBg("^S", "save snippet") +
		base.Render("   ") + keyHintBg("v", "resolve variables")
	footR := styleDim.Render("variables resolved from ") + base.Foreground(colAccent).Render("◆ "+env)
	pad := w - lipgloss.Width(stripStyle(footL)) - lipgloss.Width(stripStyle(footR))
	if pad < 1 {
		// Too narrow for both halves on one line: drop the resolved-from note
		// rather than letting the two collide.
		pad = w - lipgloss.Width(stripStyle(footL))
		footR = ""
		if pad < 1 {
			pad = 1
		}
	}
	lines = append(lines, base.Render(""), footL+base.Render(strings.Repeat(" ", pad))+footR)
	return paneBlock(lines, w, h)
}
