package tui

import (
	"strings"
	"testing"

	"senda/internal/model"
)

// testModel builds a minimal sized model with a one-request tree.
func testModel() tuiModel {
	root := &model.TreeNode{Name: "api", IsDir: true, Path: "/api"}
	req := &model.TreeNode{Name: "get-users.yaml", Path: "/api/get-users.yaml", Method: "GET"}
	root.Children = []*model.TreeNode{req}
	coll := model.Collection{Name: "api", Tree: root}
	m := newModel(coll, "/api", nil, "")
	m.w, m.h = 120, 40
	m.resize()
	return m
}

// TestRenderAllLayouts ensures every layout mode renders without panicking and
// produces non-empty output at a realistic terminal size.
func TestRenderAllLayouts(t *testing.T) {
	m := testModel()
	for mode := layoutStacked; mode < layoutModeCount; mode++ {
		m.layout = mode
		m.resize()
		out := m.render()
		if strings.TrimSpace(out) == "" {
			t.Fatalf("layout %s rendered empty", layoutNames[mode])
		}
	}
}

// TestRenderOverlays exercises the help, picker, and export overlays.
func TestRenderOverlays(t *testing.T) {
	m := testModel()
	m.loaded = true
	m.cur = model.Request{Method: "GET", URL: "https://x/y"}
	for _, set := range []func(*tuiModel){
		func(m *tuiModel) { m.showHelp = true },
		func(m *tuiModel) { m.pickerOpen = true },
		func(m *tuiModel) { m.exportOpen = true },
		func(m *tuiModel) { m.paletteOpen = true },
	} {
		mm := m
		set(&mm)
		if strings.TrimSpace(mm.render()) == "" {
			t.Fatal("overlay rendered empty")
		}
	}
}

// TestPaletteFilter checks query filtering and that selecting a request item
// moves the cursor to that row.
func TestPaletteFilter(t *testing.T) {
	m := testModel()
	all := m.paletteItems()
	if len(all) < 5 { // 1 request + 4 commands
		t.Fatalf("expected request + commands, got %d", len(all))
	}
	// Requests filter by the query; commands always remain available.
	m.paletteQuery = "users"
	got := m.paletteItems()
	reqs := 0
	for _, it := range got {
		if it.rowIdx >= 0 {
			reqs++
		}
	}
	if reqs != 1 {
		t.Fatalf("query 'users' should match the one request, got %d requests", reqs)
	}

	// A non-matching request query still leaves the commands selectable.
	m.paletteQuery = "zzz-no-match"
	cmds := m.paletteItems()
	hasEnv := false
	for _, it := range cmds {
		if it.rowIdx >= 0 {
			t.Fatalf("no request should match 'zzz-no-match', got %q", it.label)
		}
		if it.cmd == "env" {
			hasEnv = true
		}
	}
	if !hasEnv {
		t.Fatalf("env command should remain available, got %+v", cmds)
	}

	// Selecting the request row should reposition the cursor onto it.
	m.paletteOpen = true
	mm, _ := m.runPalette(paletteItem{rowIdx: 1})
	rm := mm.(tuiModel)
	if rm.paletteOpen {
		t.Fatal("palette should close after selection")
	}
	if rm.cursor != 1 || rm.focus != focusReq {
		t.Fatalf("expected cursor=1 focus=req, got cursor=%d focus=%d", rm.cursor, rm.focus)
	}
}
