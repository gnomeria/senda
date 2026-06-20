package tui

import (
	"os"
	"path/filepath"
	"testing"

	"senda/internal/model"
)

// sendaAPITree builds the "senda-api" collection tree used in the mockups:
// Users / Auth / Orders / Realtime folders with the canonical requests.
func sendaAPITree() *model.TreeNode {
	dir := func(name, path string, kids ...*model.TreeNode) *model.TreeNode {
		return &model.TreeNode{Name: name, Path: path, IsDir: true, Children: kids}
	}
	req := func(method, name, path string) *model.TreeNode {
		return &model.TreeNode{Name: name, Path: path, Method: method}
	}
	root := dir("senda-api", "/senda-api",
		dir("Users", "/senda-api/Users",
			req("GET", "/users", "/senda-api/Users/get-users.yaml"),
			req("GET", "/users/:id", "/senda-api/Users/get-user.yaml"),
			req("POST", "/users", "/senda-api/Users/post-users.yaml"),
			req("PATCH", "/users/:id", "/senda-api/Users/patch-user.yaml"),
			req("DELETE", "/users/:id", "/senda-api/Users/delete-user.yaml"),
		),
		dir("Auth", "/senda-api/Auth",
			req("POST", "/auth/login", "/senda-api/Auth/login.yaml"),
			req("POST", "/auth/refresh", "/senda-api/Auth/refresh.yaml"),
			req("POST", "/auth/logout", "/senda-api/Auth/logout.yaml"),
		),
		dir("Orders", "/senda-api/Orders",
			req("GET", "/orders", "/senda-api/Orders/get-orders.yaml"),
			req("GET", "/orders/:id", "/senda-api/Orders/get-order.yaml"),
			req("POST", "/orders", "/senda-api/Orders/post-orders.yaml"),
			req("PATCH", "/orders/:id", "/senda-api/Orders/patch-order.yaml"),
			req("DELETE", "/orders/:id", "/senda-api/Orders/delete-order.yaml"),
			req("POST", "/orders/:id/refund", "/senda-api/Orders/refund.yaml"),
		),
		dir("Realtime", "/senda-api/Realtime",
			req("WS", "/events/stream", "/senda-api/Realtime/events.yaml"),
		),
	)
	return root
}

func snapModel(w, h int) tuiModel {
	coll := model.Collection{Name: "~/work/senda-api", Tree: sendaAPITree()}
	v := func(k, val string) model.KV { return model.KV{Key: k, Value: val, Enabled: true} }
	envs := []model.Environment{
		{Name: "local", Vars: []model.KV{
			v("base_url", "http://localhost:8080"),
		}},
		{Name: "staging", Vars: []model.KV{
			v("base_url", "staging.senda.dev/v1"),
		}},
		{Name: "prod", Vars: []model.KV{
			v("base_url", "api.senda.dev"),
			v("token", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"),
			v("api_key", "sk_live_abc123"),
			v("ws_url", "wss://rt.senda.dev"),
			v("user_id", "usr_8a2f"),
		}},
	}
	m := newModel(coll, "/senda-api", envs, "prod")
	m.w, m.h = w, h
	m.resize()
	return m
}

const sampleBody = `{
  "data": [
    {
      "id": "usr_8a2f",
      "name": "Ada Lovelace",
      "email": "ada@senda.dev",
      "role": "admin",
      "active": true,
      "created_at": "2026-01-14T09:24:00Z"
    },
    {
      "id": "usr_3c9b",
      "name": "Alan Turing",
      "email": "alan@senda.dev",
      "role": "member",
      "active": true,
      "created_at": "2026-02-02T16:05:00Z"
    }
  ],
  "meta": {
    "total": 142,
    "limit": 20,
    "page": 1
  }
}`

func writeSnap(t *testing.T, name, content string) {
	dir := os.Getenv("SENDA_SNAP_DIR")
	if dir == "" {
		dir = "tmp/snap"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".ansi"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	plain := ansiRe.ReplaceAllString(content, "")
	if err := os.WriteFile(filepath.Join(dir, name+".txt"), []byte(plain), 0o644); err != nil {
		t.Fatal(err)
	}
}

// loadUsersRequest puts the model into the GET /users state shown in mockup A.
func loadUsersRequest(m *tuiModel) {
	m.cursor = 2 // senda-api > Users > GET /users
	m.curPath = "/senda-api/Users/get-users.yaml"
	m.loaded = true
	m.cur = model.Request{
		Method: "GET",
		URL:    "https://{{base_url}}/v1/users",
		Params: []model.KV{
			{Key: "limit", Value: "20", Desc: "items per page", Enabled: true},
			{Key: "status", Value: "active", Desc: "filter by state", Enabled: true},
			{Key: "sort", Value: "-created_at", Desc: "order", Enabled: true},
		},
		Headers: []model.KV{
			{Key: "Authorization", Value: "Bearer {{token}}", Enabled: true},
			{Key: "Accept", Value: "application/json", Enabled: true},
			{Key: "X-Senda-Client", Value: "tui/1.0", Enabled: true},
		},
		Auth: model.Auth{Type: model.AuthBearer, Token: "{{token}}"},
	}
	m.openTab("/senda-api/Users/get-users.yaml", "GET")
	m.open[len(m.open)-1].name = "/users"
	m.openTab("/senda-api/Auth/login.yaml", "POST")
	m.open[len(m.open)-1].name = "/auth/login"
	m.openTab("/senda-api/Realtime/events.yaml", "WS")
	m.open[len(m.open)-1].name = "/events/stream"
	m.refreshReqView()
	m.resp = &model.Response{
		Status: 200, StatusText: "OK", DurationMs: 142, SizeBytes: 4300,
		Body: sampleBody,
		Headers: map[string][]string{
			"Content-Type": {"application/json"}, "Cache-Control": {"no-cache"},
		},
	}
	m.refreshRespView()
}

// TestSnapshots writes raw-ANSI snapshots of each screen for visual comparison
// against the assets/ mockups. Gated by SENDA_SNAP so it never runs in CI.
func TestSnapshots(t *testing.T) {
	if os.Getenv("SENDA_SNAP") == "" {
		t.Skip("set SENDA_SNAP=1 to write snapshots")
	}

	// A — three-pane
	{
		m := snapModel(120, 34)
		m.layout = layout3Pane
		m.resize()
		loadUsersRequest(&m)
		// Match the mockup's expand state: Users + Realtime open, Auth + Orders
		// collapsed (showing their request-count badges).
		m.expanded["/senda-api/Auth"] = false
		m.expanded["/senda-api/Orders"] = false
		m.rebuild()
		m.cursor = 2 // GET /users
		m.focus = focusTree
		writeSnap(t, "A-three-pane", m.render())
	}

	// B — stacked
	{
		m := snapModel(120, 34)
		m.layout = layoutStacked
		m.resize()
		loadUsersRequest(&m)
		m.envIdx = 1 // staging
		writeSnap(t, "B-stacked", m.render())
	}

	// C — focus mode
	{
		m := snapModel(120, 34)
		m.layout = layoutFocus
		m.resize()
		loadUsersRequest(&m)
		m.focus = focusResp
		writeSnap(t, "C-focus-mode", m.render())
	}

	// command palette
	{
		m := snapModel(120, 34)
		m.layout = layout3Pane
		m.resize()
		loadUsersRequest(&m)
		m.paletteOpen = true
		m.paletteQuery = "user"
		writeSnap(t, "command-palette", m.render())
	}

	// environments manager
	{
		m := snapModel(120, 34)
		m.envIdx = 2 // prod
		m.openTab("/senda-api/Users/get-users.yaml", "GET")
		m.open[len(m.open)-1].name = "/users"
		m.envMgrOpen = true
		m.envMgrIdx = 2
		writeSnap(t, "environments", m.render())
	}

	// codegen / export
	{
		m := snapModel(120, 34)
		m.layout = layout3Pane
		m.resize()
		loadUsersRequest(&m)
		m.reqTab = tabHeaders
		m.refreshReqView()
		m.exportOpen = true
		m.exportIdx = 0
		writeSnap(t, "codegen", m.render())
	}

	// tests + timing
	{
		m := snapModel(120, 34)
		m.layout = layoutStacked
		m.resize()
		loadUsersRequest(&m)
		// 6 defined asserts so the Tests tab badge reads 6, matching the mockup.
		m.cur.Asserts = make([]model.Assert, 6)
		m.resp = &model.Response{
			Status: 200, StatusText: "OK", DurationMs: 142, SizeBytes: 4300,
			Timing: &model.ResponseTiming{DNSMs: 4, ConnectMs: 11, TLSMs: 38, FirstByteMs: 124, DownloadMs: 18},
			Asserts: []model.AssertResult{
				{Target: "status code is 200", Op: "", Value: "", Pass: true},
				{Target: "response time below 300 ms", Op: "", Value: "", Pass: true},
				{Target: "content-type is application/json", Op: "", Value: "", Pass: true},
				{Target: "body.meta.limit is a number", Op: "", Value: "", Pass: true},
				{Target: "every user has a valid email", Op: "", Value: "", Pass: true},
				{Target: "role in [admin, member, guest]", Op: "", Value: "", Pass: false, Actual: "owner", Error: `expected "member" — received "owner" at data[1].role`},
			},
		}
		m.respTab = rtabTests
		m.refreshRespView()
		writeSnap(t, "tests-timing", m.render())
	}

	// websocket live
	{
		m := snapModel(120, 34)
		m.layout = layout3Pane
		m.resize()
		// Load the WS request and connect to a sample stream.
		m.cursor = 0
		m.curPath = "/senda-api/Realtime/events.yaml"
		m.loaded = true
		m.cur = model.Request{Method: "WS", URL: "wss://{{ws_url}}/events/stream"}
		m.expanded["/senda-api/Auth"] = false
		m.expanded["/senda-api/Orders"] = false
		m.rebuild()
		for i, r := range m.rows {
			if r.node.Path == "/senda-api/Realtime/events.yaml" {
				m.cursor = i
			}
		}
		m.openTab("/senda-api/Realtime/events.yaml", "WS")
		m.open[len(m.open)-1].name = "/events/stream"
		m.openTab("/senda-api/Orders/get-orders.yaml", "GET")
		m.open[len(m.open)-1].name = "/orders"
		m.ws = &wsState{
			connected: true,
			url:       "wss://rt.senda.dev/events/stream",
			msgs:      42,
			uptime:    "3m 12s",
			opcode:    "0x1 text",
			size:      "86 bytes",
			compose:   `{ "action": "subscribe", "channel": "…" }`,
			frames: []wsFrame{
				{ts: "12:04:01", out: true, text: `{ "action": "subscribe", "channel": "orders" }`},
				{ts: "12:04:01", out: false, text: `{ "event": "subscribed", "channel": "orders" }`},
				{ts: "12:04:09", out: false, text: `{ "event": "order.created", "id": "ord_5521", "total": 4200 }`},
				{ts: "12:04:14", out: false, text: `{ "event": "order.updated", "id": "ord_5521", "status": "paid" }`},
				{ts: "12:04:22", out: true, text: `{ "action": "ping" }`},
				{ts: "12:04:22", out: false, text: `{ "event": "pong", "ts": 1760440 }`},
			},
		}
		m.focus = focusReq
		writeSnap(t, "websocket", m.render())
	}
}
