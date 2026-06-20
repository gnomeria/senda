package tui

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"senda/internal/model"
)

func TestDumpFeatures(t *testing.T) {
	if os.Getenv("SENDA_DUMP") == "" {
		t.Skip()
	}
	ansi := regexp.MustCompile("\x1b\\[[0-9;?]*[a-zA-Z]")
	strip := func(s string) string { return ansi.ReplaceAllString(s, "") }
	m := testModel()
	m.loaded = true
	m.cur = model.Request{Method: "GET", URL: "https://{{base_url}}/v1/users",
		Params:  []model.KV{{Key: "limit", Value: "20", Enabled: true}, {Key: "status", Value: "active", Enabled: true}},
		Headers: []model.KV{{Key: "Accept", Value: "application/json", Enabled: true}}}
	m.w, m.h = 120, 34
	m.resize()
	m.curPath = "/api/get-users.yaml"
	m.cursor = 1
	m.openTab("/api/get-users.yaml", "GET")
	m.openTab("/api/auth-login.yaml", "POST")
	m.refreshReqView()
	m.resp = &model.Response{Status: 200, StatusText: "OK", DurationMs: 142, SizeBytes: 4300,
		Body: `{"data":[{"id":"usr_8a2f","name":"Ada Lovelace"}],"meta":{"total":142}}`}
	m.refreshRespView()
	fmt.Printf("\n===== FULL 3-PANE =====\n%s\n", strip(m.render()))

	m.paletteOpen = true
	m.paletteQuery = ""
	fmt.Printf("\n===== PALETTE =====\n%s\n", strip(m.render()))
	m.paletteOpen = false

	m.resp = &model.Response{Status: 200, StatusText: "OK", DurationMs: 142, SizeBytes: 4300,
		Timing: &model.ResponseTiming{DNSMs: 4, ConnectMs: 11, TLSMs: 38, FirstByteMs: 71, DownloadMs: 18},
		Asserts: []model.AssertResult{
			{Target: "status", Op: "==", Value: "200", Pass: true},
			{Target: "time", Op: "<", Value: "300", Pass: true},
			{Target: "role", Op: "in", Value: "[admin]", Pass: false, Actual: "owner"},
		}}
	m.respTab = rtabTiming
	m.refreshRespView()
	fmt.Printf("\n===== TIMING WATERFALL =====\n%s\n", strip(m.vp.View()))
	m.respTab = rtabTests
	m.refreshRespView()
	fmt.Printf("\n===== TESTS SUMMARY =====\n%s\n", strip(m.vp.View()))

	em := testModel()
	em.w, em.h = 110, 24
	em.envs = []model.Environment{
		{Name: "local", Vars: []model.KV{{Key: "base_url", Value: "http://localhost"}}},
		{Name: "prod", Vars: []model.KV{
			{Key: "base_url", Value: "api.senda.dev/v1"},
			{Key: "token", Value: "eyJhbGc"},
			{Key: "api_key", Value: "sk_live_123"},
			{Key: "user_id", Value: "usr_8a2f"},
		}},
	}
	em.envIdx = 1
	em.envMgrOpen = true
	em.envMgrIdx = 1
	fmt.Printf("\n===== ENV MANAGER =====\n%s\n", strip(em.envMgrView()))
}
