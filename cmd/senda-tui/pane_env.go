package main

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"senda/internal/model"
)

func envNames(envs []model.Environment) []string {
	out := make([]string, len(envs))
	for i, e := range envs {
		out[i] = e.Name
	}
	return out
}

// secretKeywords are substrings in a variable name that mark it as holding a
// secret, so its value is masked in the UI.
var secretKeywords = []string{"token", "secret", "password", "passwd", "api_key", "apikey", "client_secret"}

// isSecret guesses whether a variable holds a secret (so its value is masked),
// since the env model has no explicit secret flag.
func isSecret(key string) bool {
	k := strings.ToLower(key)
	for _, kw := range secretKeywords {
		if strings.Contains(k, kw) {
			return true
		}
	}
	return false
}

// envMgrView is the full-screen environments manager: an environment list on
// the left and the selected env's variables table on the right, with secret
// values masked — matching the senda environments mockup.
func (m tuiModel) envMgrView() string {
	bodyH := m.h - 4 // titlebar + tab bar + status bar
	listW := m.w * 26 / 100
	if listW < 24 {
		listW = 24
	}
	tableW := m.w - listW - 4
	innerL := listW - 2
	innerR := tableW - 2

	// Left: environment list, action shortcuts, and scope precedence.
	var lb strings.Builder
	lb.WriteString(styleTitle.Render("ENVIRONMENTS") + styleDim.Render("  ^E") + "\n\n")
	for i, e := range m.envs {
		active := i == m.envIdx
		sel := i == m.envMgrIdx
		bg := bgPanel
		if sel {
			bg = bgSel
		}
		st := base.Background(bg)
		dot := st.Foreground(colSubtle).Render("●")
		if active {
			dot = st.Foreground(colGood).Render("●")
		}
		name := st.Foreground(colFg).Render(e.Name)
		left := dot + st.Render(" ") + name
		var tag string
		if active {
			tag = st.Foreground(colGood).Render("active ") + st.Foreground(colAccent).Render("◆")
		} else {
			tag = st.Foreground(colDim).Render(fmt.Sprintf("%d vars", len(e.Vars)))
		}
		pad := innerL - lipgloss.Width(stripStyle(left)) - lipgloss.Width(stripStyle(tag))
		if pad < 1 {
			pad = 1
		}
		lb.WriteString(left + st.Render(strings.Repeat(" ", pad)) + tag + "\n")
	}
	lb.WriteString("\n")
	action := func(glyph, label string) string {
		return base.Foreground(colAccent).Render(glyph) + base.Foreground(colDim).Render(" "+label)
	}
	lb.WriteString(action("⊕", "new environment") + "\n")
	lb.WriteString(action("◈", "set active") + "\n")
	lb.WriteString(action("^G", "globals & secrets") + "\n\n")
	lb.WriteString(styleDim.Render("SCOPE PRECEDENCE") + "\n")
	lb.WriteString(base.Foreground(colFg).Render("request") + styleDim.Render(" › ") +
		base.Foreground(colFg).Render("env") + styleDim.Render(" › ") +
		base.Foreground(colFg).Render("globals"))
	left := styleBorder.Width(listW).Height(bodyH).Render(lb.String())

	// Right: variables table + resolved preview for the selected environment.
	sel := m.envs[m.envMgrIdx]
	var rb strings.Builder
	rb.WriteString(styleTitle.Render("VARIABLES") + styleDim.Render(" · "+strings.ToUpper(sel.Name)) +
		styleDim.Render(fmt.Sprintf("  %d active", len(sel.Vars))) + "\n\n")
	keyW := innerR * 28 / 100
	valW := innerR * 48 / 100
	scopeW := innerR - keyW - valW
	rb.WriteString(styleDim.Render(padRight("VARIABLE", keyW)+padRight("VALUE", valW)+padRight("SCOPE", scopeW)) + "\n")
	for _, v := range sel.Vars {
		scope := base.Foreground(colDim).Render("env")
		val := base.Foreground(colFg).Render(truncate(v.Value, valW-1))
		if isSecret(v.Key) {
			val = base.Foreground(colDim).Render(strings.Repeat("•", 13)) + base.Render(" ") +
				base.Foreground(colWarn).Render("🔒")
			scope = base.Foreground(colWarn).Render("secret")
		}
		rb.WriteString(base.Foreground(colFg).Render(padRight(v.Key, keyW)) +
			padRight(val, valW) + scope + "\n")
	}
	if len(sel.Vars) == 0 {
		rb.WriteString(styleDim.Render("(no variables)") + "\n")
	}
	rb.WriteString("\n" + styleDim.Render("RESOLVED PREVIEW") + "\n")
	baseURL := envVar(sel, "base_url")
	token := envVar(sel, "token")
	rb.WriteString(base.Foreground(colGood).Bold(true).Render("GET ") +
		base.Foreground(colFg).Render("https://"+baseURL+"/v1/users") + "\n")
	rb.WriteString(base.Foreground(colDim).Render("Authorization: Bearer ") +
		base.Foreground(colFg).Render(truncate(token, 18)))
	right := styleBorderFoc.Width(tableW).Height(bodyH).Render(rb.String())

	body := lipgloss.JoinHorizontal(lipgloss.Top, left, base.Render(" "), right)

	crumb := appbg.Foreground(colDim).Render("  environments · ") +
		appbg.Foreground(colAccent).Render("◆ "+sel.Name) + appbg.Render(" ")
	hints := strings.Join([]string{
		keyHint("j/k", "move"), keyHint("e", "edit"), keyHint("s", "reveal"),
		keyHint("a", "add"), keyHint("?", "help"),
	}, appbg.Render("   ")) + appbg.Render(" ")
	bar := m.statusLine(modeChip("NORMAL", colGood)+crumb, hints)
	return lipgloss.JoinVertical(lipgloss.Left, m.titleBar(), m.reqTabsBar(), body, bar)
}

// envVar returns the value of a named variable in env, or "" if absent.
func envVar(e model.Environment, key string) string {
	for _, v := range e.Vars {
		if v.Key == key {
			return v.Value
		}
	}
	return ""
}
