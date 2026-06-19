package main

// paneDims is the computed geometry for the three panes. Widths/heights are
// border-content sizes (passed to lipgloss .Width/.Height); the rendered pane
// is two cells larger in each axis for its border. treeW==0 hides the tree.
type paneDims struct {
	treeW, treeH int
	reqW, reqH   int
	respW, respH int
	bodyH        int // rows available below titlebar, above status bar
}

// Borderless panes: each pane is a flush content block. The request pane
// reserves a label row, the URL row, and the tab row above its viewport; the
// response pane reserves a label row, the status row, and the tab row. Panes
// are separated by single-column vertical rules.
const (
	reqChrome  = 3 // label + url row + tab row
	respChrome = 3 // label + status row + tab row

	outerMargin = 0 // app-bg margin between the panel area and the screen edges
	paneGap     = 0 // app-bg gap between adjacent panel cards (borders abut)
)

// dims computes pane geometry for the active layout mode. Widths/heights are
// the inner content sizes of each rounded-border card (the rendered card is two
// cells larger in each axis for its border). bodyH is the full card-area height
// (== a full-height card's outer height), used to size the gap/margin columns.
func (m tuiModel) dims() paneDims {
	d := paneDims{}
	// titlebar + tabs + status bar = 3 chrome rows (panels sit flush, no gap rows).
	regionH := atLeast(m.h-3, 8)
	d.bodyH = regionH
	fullH := regionH - 2 // inner height inside a full-height card border
	d.treeH = fullH

	// Codegen and the tests results view hide the tree and split the body into
	// two equal cards (request | codegen, or test list | summary+timing).
	if m.exportOpen || m.testsView() {
		d.treeW = 0
		inner := m.w - 2*outerMargin - 4 - paneGap // 2 cards × 2 border + 1 gap
		d.reqW = inner / 2
		d.respW = inner - d.reqW
		d.reqH, d.respH = fullH, fullH
		return d
	}

	// WebSocket view: tree | connection log | frame inspector (three cards).
	if m.wsView() {
		inner := m.w - 2*outerMargin - 6 - 2*paneGap
		d.treeW = atLeast(inner*22/100, 20)
		frameW := atLeast(inner*24/100, 18)
		d.respW = frameW
		d.reqW = inner - d.treeW - frameW
		d.reqH, d.respH = fullH, fullH
		return d
	}

	switch m.layout {
	case layout3Pane:
		inner := m.w - 2*outerMargin - 6 - 2*paneGap // 3 cards × 2 border + 2 gaps
		d.treeW = atLeast(inner*24/100, 20)
		rest := inner - d.treeW
		d.reqW = rest / 2
		d.respW = rest - d.reqW
		d.reqH, d.respH = fullH, fullH

	case layoutFocus:
		d.treeW = 0
		colW := (m.w - 2*outerMargin - 2) * 72 / 100
		d.reqW, d.respW = colW, colW
		d.reqH, d.respH = fullH, fullH

	default: // layoutStacked: tree card | (request card over response card)
		inner := m.w - 2*outerMargin - 4 - paneGap // tree card + right card
		d.treeW = atLeast(inner*28/100, 22)
		rightW := inner - d.treeW
		d.reqW, d.respW = rightW, rightW
		// Right column stacks two cards inside the region:
		// (reqH+2) + paneGap + (respH+2) == regionH.
		stackInner := regionH - 4 - paneGap
		d.reqH = atLeast(stackInner*45/100, reqChrome+2)
		d.respH = atLeast(stackInner-d.reqH, respChrome+1)
	}
	return d
}

// resize recomputes viewport sizes from the current terminal dimensions.
func (m *tuiModel) resize() {
	if m.w == 0 || m.h == 0 {
		return
	}
	d := m.dims()
	m.reqVp.SetWidth(d.reqW)
	m.reqVp.SetHeight(d.reqH - reqChrome)
	m.vp.SetWidth(d.respW)
	m.vp.SetHeight(d.respH - respChrome)
	m.clampTreeScroll()
}
