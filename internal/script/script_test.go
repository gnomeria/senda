package script

import (
	"strings"
	"testing"

	"senda/internal/model"
)

func varStore() (map[string]string, GetVar, SetVar) {
	store := map[string]string{}
	return store,
		func(name string) string { return store[name] },
		func(name, value string) { store[name] = value }
}

func TestPreMutatesRequest(t *testing.T) {
	_, get, set := varStore()
	req := model.Request{
		Method: "GET",
		URL:    "https://api.test/users",
		Headers: []model.KV{
			{Key: "Accept", Value: "text/plain", Enabled: true},
		},
	}
	src := `
		req.method = "POST";
		req.url = req.url + "/active";
		req.setHeader("accept", "application/json");
		req.setHeader("X-Trace", "abc");
		req.setParam("page", "2");
		req.body = { type: "json", raw: JSON.stringify({ ok: true }) };
	`
	out, _, err := RunPre(src, req, get, set)
	if err != nil {
		t.Fatal(err)
	}
	if out.Method != "POST" || out.URL != "https://api.test/users/active" {
		t.Errorf("method/url = %q %q", out.Method, out.URL)
	}
	if len(out.Headers) != 2 || out.Headers[0].Value != "application/json" || out.Headers[1].Key != "X-Trace" {
		t.Errorf("headers = %+v", out.Headers)
	}
	if len(out.Params) != 1 || out.Params[0].Key != "page" || out.Params[0].Value != "2" {
		t.Errorf("params = %+v", out.Params)
	}
	if out.Body.Type != model.BodyJSON || out.Body.Raw != `{"ok":true}` {
		t.Errorf("body = %+v", out.Body)
	}
}

func TestPreSetAndGetVars(t *testing.T) {
	store, get, set := varStore()
	store["base"] = "https://api.test"
	req := model.Request{URL: "ignored"}
	src := `
		req.url = senda.getVar("base") + "/things";
		senda.setVar("stamp", "123");
	`
	out, _, err := RunPre(src, req, get, set)
	if err != nil {
		t.Fatal(err)
	}
	if out.URL != "https://api.test/things" {
		t.Errorf("url = %q", out.URL)
	}
	if store["stamp"] != "123" {
		t.Errorf("stamp = %q", store["stamp"])
	}
}

func TestPreErrorLeavesRequestUntouched(t *testing.T) {
	_, get, set := varStore()
	req := model.Request{URL: "https://x.test"}
	out, _, err := RunPre(`throw new Error("boom")`, req, get, set)
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("err = %v", err)
	}
	if out.URL != "https://x.test" {
		t.Errorf("request mutated on error: %+v", out)
	}
}

func TestPostExtractsJSON(t *testing.T) {
	store, get, set := varStore()
	resp := model.Response{
		Status: 200,
		Body:   `{"access_token":"tok-1","user":{"id":7}}`,
		Headers: map[string][]string{
			"X-Request-Id": {"rid-9"},
		},
	}
	src := `
		senda.setVar("token", res.json.access_token);
		senda.setVar("uid", String(res.json.user.id));
		senda.setVar("status", String(res.status));
		senda.setVar("rid", res.headers["X-Request-Id"][0]);
	`
	_, _, err := RunPost(src, model.Request{}, resp, get, set)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"token": "tok-1", "uid": "7", "status": "200", "rid": "rid-9"}
	for k, v := range want {
		if store[k] != v {
			t.Errorf("%s = %q, want %q", k, store[k], v)
		}
	}
}

func TestPostNonJSONBody(t *testing.T) {
	store, get, set := varStore()
	resp := model.Response{Status: 200, Body: "<html>"}
	src := `senda.setVar("isNull", String(res.json === null));`
	_, _, err := RunPost(src, model.Request{}, resp, get, set)
	if err != nil {
		t.Fatal(err)
	}
	if store["isNull"] != "true" {
		t.Errorf("res.json should be null for non-JSON body, store=%v", store)
	}
}

func TestPostScriptError(t *testing.T) {
	_, get, set := varStore()
	_, _, err := RunPost(`nope.nope()`, model.Request{}, model.Response{}, get, set)
	if err == nil || !strings.Contains(err.Error(), "post-script") {
		t.Fatalf("err = %v", err)
	}
}

func TestInfiniteLoopInterrupted(t *testing.T) {
	if testing.Short() {
		t.Skip("waits out the script timeout")
	}
	_, get, set := varStore()
	_, _, err := RunPre(`while (true) {}`, model.Request{}, get, set)
	if err == nil || !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("err = %v", err)
	}
}

func TestPMTest(t *testing.T) {
	_, get, set := varStore()
	resp := model.Response{Status: 200, Body: `{"id":1}`}
	src := `
		pm.test("status is 200", function() {
			pm.expect(res.status).to.equal(200);
		});
		pm.test("body has id", function() {
			pm.expect(res.json.id).to.equal(1);
		});
		pm.test("will fail", function() {
			pm.expect(res.status).to.equal(404);
		});
	`
	results, _, err := RunPost(src, model.Request{}, resp, get, set)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 pm.test results, got %d", len(results))
	}
	if !results[0].Pass || results[0].Target != "status is 200" {
		t.Errorf("test[0] = %+v", results[0])
	}
	if !results[1].Pass || results[1].Target != "body has id" {
		t.Errorf("test[1] = %+v", results[1])
	}
	if results[2].Pass || results[2].Target != "will fail" {
		t.Errorf("test[2] should fail: %+v", results[2])
	}
}

func TestPMEnvGetSet(t *testing.T) {
	store, get, set := varStore()
	store["apiKey"] = "secret-123"
	resp := model.Response{Status: 200, Body: `{"token":"abc"}`}
	src := `
		var key = pm.environment.get("apiKey");
		pm.environment.set("lastToken", res.json.token);
		senda.setVar("captured", key);
	`
	_, _, err := RunPost(src, model.Request{}, resp, get, set)
	if err != nil {
		t.Fatal(err)
	}
	if store["captured"] != "secret-123" {
		t.Errorf("captured = %q", store["captured"])
	}
	if store["lastToken"] != "abc" {
		t.Errorf("lastToken = %q", store["lastToken"])
	}
}

func TestPMExpectNot(t *testing.T) {
	_, get, set := varStore()
	resp := model.Response{Status: 200, Body: `{}`}
	src := `
		pm.test("not 404", function() {
			pm.expect(res.status).to.not.equal(404);
		});
		pm.test("not include xyz", function() {
			pm.expect(res.body).to.not.include("xyz");
		});
	`
	results, _, err := RunPost(src, model.Request{}, resp, get, set)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 || !results[0].Pass || !results[1].Pass {
		t.Errorf("negation tests failed: %+v", results)
	}
}

func TestConsoleLog(t *testing.T) {
	_, get, set := varStore()
	resp := model.Response{Status: 200, Body: "ok"}
	src := `
		console.log("hello", "world");
		console.error("oops");
	`
	_, logs, err := RunPost(src, model.Request{}, resp, get, set)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 2 {
		t.Fatalf("expected 2 log lines, got %d: %v", len(logs), logs)
	}
	if logs[0] != "hello world" {
		t.Errorf("log[0] = %q", logs[0])
	}
	if logs[1] != "oops" {
		t.Errorf("log[1] = %q", logs[1])
	}
}
