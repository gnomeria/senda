package mockserver

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newTestServer writes the given files into a temp mocks dir and returns a
// Server plus an httptest server driving its handler (no port binding/watcher).
func newTestServer(t *testing.T, files map[string]string) (*Server, *httptest.Server) {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	s, err := New(dir, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(http.HandlerFunc(s.handle))
	t.Cleanup(ts.Close)
	return s, ts
}

func do(t *testing.T, ts *httptest.Server, method, path, body string) (*http.Response, string) {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, ts.URL+path, r)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return resp, string(b)
}

func TestV1BackwardCompat(t *testing.T) {
	_, ts := newTestServer(t, map[string]string{
		"old.yaml": "method: GET\npath: /old\nstatus: 201\nbody: '{\"ok\": true}'\n",
	})
	resp, body := do(t, ts, "GET", "/old", "")
	if resp.StatusCode != 201 {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}
	if !strings.Contains(body, `"ok"`) {
		t.Fatalf("body = %q", body)
	}
}

func TestTemplatingParamsAndQuery(t *testing.T) {
	_, ts := newTestServer(t, map[string]string{
		"u.yaml": `
method: GET
path: /users/:id
responses:
  - status: 200
    body:
      id: "{{params.id}}"
      q: "{{query.x}}"
`,
	})
	resp, body := do(t, ts, "GET", "/users/42?x=hi", "")
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("bad json %q: %v", body, err)
	}
	if got["id"] != "42" || got["q"] != "hi" {
		t.Fatalf("got %v", got)
	}
}

func TestUUIDAndRandomInt(t *testing.T) {
	_, ts := newTestServer(t, map[string]string{
		"r.yaml": "method: GET\npath: /r\nresponses:\n  - body:\n      id: \"{{uuid}}\"\n      n: \"{{randomInt 5 5}}\"\n",
	})
	_, body := do(t, ts, "GET", "/r", "")
	var got map[string]any
	json.Unmarshal([]byte(body), &got)
	if id, _ := got["id"].(string); !strings.Contains(id, "-") || len(id) != 36 {
		t.Fatalf("uuid = %q", got["id"])
	}
	if got["n"] != "5" {
		t.Fatalf("randomInt 5 5 = %v, want 5", got["n"])
	}
}

func TestResponseMatchWhen(t *testing.T) {
	_, ts := newTestServer(t, map[string]string{
		"m.yaml": `
method: GET
path: /m
responses:
  - when: { query: { v: "2" } }
    status: 222
    body: "v2"
  - status: 200
    body: "default"
`,
	})
	if r, _ := do(t, ts, "GET", "/m?v=2", ""); r.StatusCode != 222 {
		t.Fatalf("v=2 status %d, want 222", r.StatusCode)
	}
	if r, _ := do(t, ts, "GET", "/m", ""); r.StatusCode != 200 {
		t.Fatalf("default status %d, want 200", r.StatusCode)
	}
}

func TestScenarioSelection(t *testing.T) {
	s, ts := newTestServer(t, map[string]string{
		"s.yaml": `
method: GET
path: /s
responses:
  - scenario: error
    status: 500
    body: "boom"
  - status: 200
    body: "ok"
`,
	})
	if r, _ := do(t, ts, "GET", "/s", ""); r.StatusCode != 200 {
		t.Fatalf("default status %d, want 200", r.StatusCode)
	}
	s.SetScenario("error")
	if r, _ := do(t, ts, "GET", "/s", ""); r.StatusCode != 500 {
		t.Fatalf("scenario=error status %d, want 500", r.StatusCode)
	}
}

func TestRouteMatchAndSpecificity(t *testing.T) {
	_, ts := newTestServer(t, map[string]string{
		"id.yaml": "method: GET\npath: /users/:id\nresponses:\n  - body: \"by-id\"\n",
		"me.yaml": "method: GET\npath: /users/me\nresponses:\n  - body: \"me\"\n",
	})
	if _, b := do(t, ts, "GET", "/users/me", ""); b != "me" {
		t.Fatalf("/users/me = %q, want literal route", b)
	}
	if _, b := do(t, ts, "GET", "/users/7", ""); b != "by-id" {
		t.Fatalf("/users/7 = %q", b)
	}
}

func TestBodyMatch(t *testing.T) {
	_, ts := newTestServer(t, map[string]string{
		"b.yaml": `
method: POST
path: /b
match:
  body: { role: admin }
responses:
  - status: 201
    body: "admin"
`,
	})
	if r, _ := do(t, ts, "POST", "/b", `{"role":"admin"}`); r.StatusCode != 201 {
		t.Fatalf("admin body status %d, want 201", r.StatusCode)
	}
	if r, _ := do(t, ts, "POST", "/b", `{"role":"user"}`); r.StatusCode != 404 {
		t.Fatalf("non-admin status %d, want 404", r.StatusCode)
	}
}

func TestResourceCRUD(t *testing.T) {
	_, ts := newTestServer(t, map[string]string{
		"users.yaml": "resource: users\npath: /users\nseed:\n  - { id: 1, name: Alice }\n",
	})
	// list
	if _, b := do(t, ts, "GET", "/users", ""); !strings.Contains(b, "Alice") {
		t.Fatalf("list = %q", b)
	}
	// create
	r, b := do(t, ts, "POST", "/users", `{"name":"Bob"}`)
	if r.StatusCode != 201 {
		t.Fatalf("create status %d", r.StatusCode)
	}
	var created map[string]any
	json.Unmarshal([]byte(b), &created)
	if created["id"] == nil || created["name"] != "Bob" {
		t.Fatalf("created = %v", created)
	}
	// get by assigned id (2)
	if r, _ := do(t, ts, "GET", "/users/2", ""); r.StatusCode != 200 {
		t.Fatalf("get /users/2 status %d", r.StatusCode)
	}
	// patch
	if _, b := do(t, ts, "PATCH", "/users/2", `{"name":"Bobby"}`); !strings.Contains(b, "Bobby") {
		t.Fatalf("patch = %q", b)
	}
	// delete
	if r, _ := do(t, ts, "DELETE", "/users/2", ""); r.StatusCode != 204 {
		t.Fatalf("delete status %d, want 204", r.StatusCode)
	}
	if r, _ := do(t, ts, "GET", "/users/2", ""); r.StatusCode != 404 {
		t.Fatalf("get deleted status %d, want 404", r.StatusCode)
	}
}

func TestResourceResetState(t *testing.T) {
	s, ts := newTestServer(t, map[string]string{
		"users.yaml": "resource: users\npath: /users\nseed:\n  - { id: 1, name: Alice }\n",
	})
	do(t, ts, "POST", "/users", `{"name":"Bob"}`)
	s.ResetState()
	_, b := do(t, ts, "GET", "/users", "")
	var list []map[string]any
	json.Unmarshal([]byte(b), &list)
	if len(list) != 1 {
		t.Fatalf("after reset len = %d, want 1", len(list))
	}
}

func TestCORS(t *testing.T) {
	_, ts := newTestServer(t, map[string]string{
		"c.yaml": "method: GET\npath: /c\nresponses:\n  - body: \"ok\"\n",
	})
	resp, _ := do(t, ts, "OPTIONS", "/c", "")
	if resp.StatusCode != 204 {
		t.Fatalf("preflight status %d, want 204", resp.StatusCode)
	}
	if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Fatal("missing CORS header on preflight")
	}
	resp2, _ := do(t, ts, "GET", "/c", "")
	if resp2.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Fatal("missing CORS header on GET")
	}
}

func TestSchemaFakeData(t *testing.T) {
	_, ts := newTestServer(t, map[string]string{
		"f.yaml": `
method: GET
path: /f
responses:
  - schema: '{"type":"object","properties":{"email":{"type":"string","format":"email"}}}'
`,
	})
	_, b := do(t, ts, "GET", "/f", "")
	var got map[string]any
	if err := json.Unmarshal([]byte(b), &got); err != nil {
		t.Fatalf("bad json %q", b)
	}
	if e, _ := got["email"].(string); !strings.Contains(e, "@") {
		t.Fatalf("email = %q", got["email"])
	}
}

func TestNoMatch404(t *testing.T) {
	_, ts := newTestServer(t, map[string]string{
		"c.yaml": "method: GET\npath: /c\nresponses:\n  - body: \"ok\"\n",
	})
	if r, _ := do(t, ts, "GET", "/nope", ""); r.StatusCode != 404 {
		t.Fatalf("status %d, want 404", r.StatusCode)
	}
}
