package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/color"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"

	"senda/internal/model"
)

// respBadges returns the response tab badges (order: Body, Headers, Cookies,
// Tests, Timing, Logs). The Tests badge is always shown (even 0).
func (m tuiModel) respBadges() []string {
	if m.resp == nil {
		return nil
	}
	return []string{
		"", countBadge(len(m.resp.Headers)), countBadge(countCookies(m.resp.Headers)),
		fmt.Sprintf("%d", len(m.resp.Asserts)), "", countBadge(len(m.resp.ScriptLogs)),
	}
}

func (m tuiModel) paneResponse(w, h int) string {
	focused := m.focus == focusResp
	var status string
	switch {
	case m.respErr != "":
		status = base.Foreground(colBad).Render("● error: " + truncate(m.respErr, w-10))
	case m.sending:
		status = base.Foreground(colWarn).Render("● ") + styleDim.Render("sending…")
	case m.resp != nil:
		sc := statusColor(m.resp.Status)
		status = base.Foreground(sc).Bold(true).Render(fmt.Sprintf("● %d %s", m.resp.Status, m.resp.StatusText)) +
			styleDim.Render(fmt.Sprintf("   time %d ms   size %s", m.resp.DurationMs, humanBytes(m.resp.SizeBytes)))
		if a := assertSummary(m.resp.Asserts); a != "" {
			status += base.Render("   ") + a
		}
	default:
		status = styleDim.Render("no response yet — press s to send")
	}
	tabs := tabStrip(respTabNames, m.respBadges(), int(m.respTab))

	lines := []string{m.paneLabel("RESPONSE", focused), status, tabs}
	lines = append(lines, strings.Split(m.vp.View(), "\n")...)
	return paneBlock(lines, w, h)
}

// paneTestList renders the left pane of the tests results view: the request
// chrome with the Tests tab active, then the assertion results list.
func (m tuiModel) paneTestList(w, h int) string {
	focused := m.focus == focusReq
	tabs := tabStrip(reqTabNames, m.reqBadges(), int(tabTests))
	lines := []string{m.paneLabel("REQUEST", focused), m.urlRow(w), tabs, base.Render("")}
	lines = append(lines, strings.Split(renderTestResults(m.resp.Asserts, m.testName(), w), "\n")...)
	return paneBlock(lines, w, h)
}

// testName derives a test label from the current request's file name.
func (m tuiModel) testName() string {
	name := m.curPath
	if i := strings.LastIndexByte(name, '/'); i >= 0 {
		name = name[i+1:]
	}
	name = strings.TrimSuffix(name, ".yaml")
	if name == "" {
		return "request"
	}
	return name + " endpoint"
}

// renderTestResults renders the assertion results list (test() header, ✓/✗ rows,
// failure detail) shown in the tests view's left pane.
func renderTestResults(rs []model.AssertResult, name string, w int) string {
	var b strings.Builder
	b.WriteString(styleDim.Render("test(") + base.Foreground(colGood).Render(`"`+name+`"`) +
		styleDim.Render(") — post-response · javascript") + "\n\n")
	for _, r := range rs {
		label := r.Target
		if r.Op != "" {
			label = strings.TrimSpace(r.Target + " " + r.Op + " " + r.Value)
		}
		var mark, right string
		if r.Pass {
			mark = base.Foreground(colGood).Render("✓")
		} else {
			mark = base.Foreground(colBad).Render("✗")
			right = base.Foreground(colBad).Render("failed")
		}
		left := mark + base.Render(" ") + base.Foreground(colFg).Render(label)
		b.WriteString(padBetween(left, right, w, base) + "\n")
		if !r.Pass && r.Error != "" {
			b.WriteString(base.Foreground(colBad).Render("AssertionError: "+r.Error) + "\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// paneTestSummary renders the right pane of the tests view: a pass/fail summary
// with a proportion bar, the timing waterfall, and a protocol footer.
func (m tuiModel) paneTestSummary(w, h int) string {
	focused := m.focus == focusResp
	rs := m.resp.Asserts
	pass := countPasses(rs)
	fail := len(rs) - pass

	summary := base.Foreground(colGood).Bold(true).Render(fmt.Sprintf("%d passed", pass)) +
		base.Render("   ") + base.Foreground(colBad).Bold(true).Render(fmt.Sprintf("%d failed", fail))
	total := styleDim.Render(fmt.Sprintf("total time %d ms", m.resp.DurationMs))
	summaryRow := padBetween(summary, total, w, base)

	barW := w - 1
	if barW < 10 {
		barW = 10
	}
	gw := 0
	if len(rs) > 0 {
		gw = barW * pass / len(rs)
	}
	bar := base.Foreground(colGood).Render(strings.Repeat("━", gw)) +
		base.Foreground(colBad).Render(strings.Repeat("━", barW-gw))

	lines := []string{
		m.paneLabel("SUMMARY", focused), base.Render(""),
		summaryRow, bar, base.Render(""),
		m.paneLabel("TIMING", false) + styleDim.Render(" waterfall"), base.Render(""),
	}
	lines = append(lines, strings.Split(m.renderTiming(m.resp.Timing, w), "\n")...)

	proto := base.Foreground(colFg).Render("HTTP/2") + styleDim.Render("  TLS 1.3")
	meta := base.Foreground(statusColor(m.resp.Status)).Render(fmt.Sprintf("%d %s", m.resp.Status, m.resp.StatusText)) +
		styleDim.Render(fmt.Sprintf(" · %s · gzip", humanBytes(m.resp.SizeBytes)))
	lines = append(lines, base.Render(""), padBetween(proto, meta, w, base))
	return paneBlock(lines, w, h)
}

// renderRespTab builds the scrollable content for the active response tab.
func (m tuiModel) renderRespTab() string {
	if m.resp == nil {
		return ""
	}
	switch m.respTab {
	case rtabBody:
		return m.renderResponseBody()
	case rtabCookies:
		return renderCookies(m.resp.Headers)
	case rtabHeaders:
		return renderHeaders(m.resp.Headers)
	case rtabTests:
		return m.renderAssertResults(m.resp.Asserts, m.vp.Width())
	case rtabTiming:
		return m.renderTiming(m.resp.Timing, m.vp.Width())
	case rtabLogs:
		if len(m.resp.ScriptLogs) == 0 {
			return styleDim.Render("(no logs)")
		}
		return strings.Join(m.resp.ScriptLogs, "\n")
	}
	return ""
}

func renderHeaders(h map[string][]string) string {
	if len(h) == 0 {
		return styleDim.Render("(none)")
	}
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(styleDim.Render(k+": ") + strings.Join(h[k], ", ") + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// renderCookies extracts Set-Cookie headers (case-insensitive) and renders the
// cookie name/value plus its attributes (Path, Expires, …).
func renderCookies(h map[string][]string) string {
	var cookies []string
	for k, vals := range h {
		if strings.EqualFold(k, "Set-Cookie") {
			cookies = append(cookies, vals...)
		}
	}
	if len(cookies) == 0 {
		return styleDim.Render("(no cookies)")
	}
	sort.Strings(cookies)
	var b strings.Builder
	for _, c := range cookies {
		parts := strings.Split(c, ";")
		nv := strings.SplitN(strings.TrimSpace(parts[0]), "=", 2)
		name := nv[0]
		val := ""
		if len(nv) == 2 {
			val = nv[1]
		}
		b.WriteString(styleTitle.Render(name) + " = " + val + "\n")
		for _, attr := range parts[1:] {
			attr = strings.TrimSpace(attr)
			if attr != "" {
				b.WriteString("    " + styleDim.Render(attr) + "\n")
			}
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m tuiModel) renderAssertResults(rs []model.AssertResult, width int) string {
	if len(rs) == 0 {
		return styleDim.Render("(no asserts)")
	}
	pass := countPasses(rs)
	fail := len(rs) - pass
	header := base.Foreground(colGood).Bold(true).Render(fmt.Sprintf("%d passed", pass)) +
		base.Render("   ") + base.Foreground(colBad).Bold(true).Render(fmt.Sprintf("%d failed", fail))
	// Proportion bar: green run for passes, red for fails.
	barW := width - 2
	if barW < 10 {
		barW = 10
	}
	gw := barW * pass / len(rs)
	bar := base.Foreground(colGood).Render(strings.Repeat("━", gw)) +
		base.Foreground(colBad).Render(strings.Repeat("━", barW-gw))

	var b strings.Builder
	b.WriteString(header + "\n" + bar + "\n\n")
	for _, r := range rs {
		mark := base.Foreground(colGood).Render("✓")
		if !r.Pass {
			mark = base.Foreground(colBad).Render("✗")
		}
		line := mark + base.Render(" ") + base.Foreground(colFg).Render(strings.TrimSpace(r.Target+" "+r.Op+" "+r.Value))
		if r.Error != "" {
			line += styleDim.Render("  err: " + r.Error)
		} else if !r.Pass {
			line += styleDim.Render("  got: " + r.Actual)
		}
		b.WriteString(line + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// timingPhase is one segment of the waterfall: a label, its start offset and
// duration in ms, and a bar color.
type timingPhase struct {
	label   string
	startMs int64
	durMs   int64
	col     color.Color
}

// renderTiming draws a cumulative timing waterfall (DNS→TCP→TLS→TTFB→Download)
// with proportional bars, matching the senda tests-timing mockup. width is the
// available content width.
func (m tuiModel) renderTiming(t *model.ResponseTiming, width int) string {
	if t == nil {
		return styleDim.Render("(no timing)")
	}
	// FirstByteMs is cumulative (start→first byte); the wait/TTFB phase is the
	// remainder after the connection phases.
	conn := t.DNSMs + t.ConnectMs + t.TLSMs
	ttfb := t.FirstByteMs - conn
	if ttfb < 0 {
		ttfb = 0
	}
	var phases []timingPhase
	if !t.Reused {
		phases = append(phases,
			timingPhase{"DNS", 0, t.DNSMs, lipgloss.Color("75")},
			timingPhase{"TCP", t.DNSMs, t.ConnectMs, lipgloss.Color("141")},
			timingPhase{"TLS", t.DNSMs + t.ConnectMs, t.TLSMs, colWarn},
		)
	}
	phases = append(phases,
		timingPhase{"TTFB", conn, ttfb, colGood},
		timingPhase{"Download", t.FirstByteMs, t.DownloadMs, lipgloss.Color("78")},
	)

	total := t.FirstByteMs + t.DownloadMs
	if total <= 0 {
		total = 1
	}
	const labelW = 10
	barW := width - labelW - 11 // room for label + " 9999 ms" + gaps
	if barW < 10 {
		barW = 10
	}

	var b strings.Builder
	for _, p := range phases {
		offset := int(int64(barW) * p.startMs / total)
		fill := int(int64(barW) * p.durMs / total)
		if fill < 1 && p.durMs > 0 {
			fill = 1
		}
		trail := barW - offset - fill
		if trail < 0 {
			trail = 0
		}
		// Faint full-width track with the colored phase segment laid over it.
		// Every span carries the panel bg (base) so no cell renders black.
		track := base.Foreground(colSubtle)
		bar := track.Render(strings.Repeat("─", offset)) +
			base.Foreground(p.col).Render(strings.Repeat("━", fill)) +
			track.Render(strings.Repeat("─", trail))
		b.WriteString(
			styleDim.Render(fmt.Sprintf("%-*s", labelW, p.label)) + base.Render(" ") +
				bar + base.Render(" ") +
				styleDim.Render(fmt.Sprintf("%4d ms", p.durMs)) + "\n")
	}
	if t.Reused {
		b.WriteString(styleDim.Render("(connection reused)") + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m tuiModel) renderResponseBody() string {
	if m.resp == nil {
		return ""
	}
	body := numberedCode(prettyJSON(m.resp.Body))
	if m.resp.Truncated {
		body += "\n" + styleDim.Render("… (truncated)")
	}
	return body
}

func prettyJSON(body string) string {
	trimmed := strings.TrimSpace(body)
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		var buf bytes.Buffer
		if json.Indent(&buf, []byte(trimmed), "", "  ") == nil {
			return buf.String()
		}
	}
	return body
}

// colorJSON applies syntax coloring to one pretty-printed JSON line with a
// proper string-aware scan (so colons/braces inside string values are never
// mistaken for structure): keys in accent, string values in green, numbers in
// warn, booleans/null in cyan, punctuation dim.
func colorJSON(line string) string {
	var b strings.Builder
	i := 0
	for i < len(line) {
		c := line[i]
		switch {
		case c == '"':
			// Read the full string literal, honoring backslash escapes.
			j := i + 1
			for j < len(line) {
				if line[j] == '\\' {
					j += 2
					continue
				}
				if line[j] == '"' {
					j++
					break
				}
				j++
			}
			lit := line[i:j]
			// A key is a string immediately followed (after spaces) by a colon.
			k := j
			for k < len(line) && line[k] == ' ' {
				k++
			}
			if k < len(line) && line[k] == ':' {
				b.WriteString(base.Foreground(colAccent).Render(lit))
			} else {
				b.WriteString(base.Foreground(colGood).Render(lit))
			}
			i = j
		case c == '-' || (c >= '0' && c <= '9'):
			j := i
			for j < len(line) && (line[j] == '-' || line[j] == '+' || line[j] == '.' ||
				line[j] == 'e' || line[j] == 'E' || (line[j] >= '0' && line[j] <= '9')) {
				j++
			}
			b.WriteString(base.Foreground(colWarn).Render(line[i:j]))
			i = j
		case c >= 'a' && c <= 'z':
			j := i
			for j < len(line) && line[j] >= 'a' && line[j] <= 'z' {
				j++
			}
			word := line[i:j]
			if word == "true" || word == "false" || word == "null" {
				b.WriteString(base.Foreground(colCyan).Render(word))
			} else {
				b.WriteString(base.Render(word))
			}
			i = j
		default:
			b.WriteString(styleDim.Render(string(c)))
			i++
		}
	}
	return b.String()
}

// numberedCode prepends a dim line-number gutter and colorizes JSON-ish lines,
// matching the line-numbered body panes in the mockups.
func numberedCode(s string) string {
	lines := strings.Split(s, "\n")
	gw := len(fmt.Sprintf("%d", len(lines)))
	if gw < 2 {
		gw = 2
	}
	var b strings.Builder
	for i, ln := range lines {
		gutter := styleDim.Render(fmt.Sprintf("%*d ", gw, i+1))
		b.WriteString(gutter + colorJSON(ln) + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}
