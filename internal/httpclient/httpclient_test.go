package httpclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"senda/internal/model"
	"senda/internal/vars"
)

func emptyScope() *vars.Scope { return vars.Build() }

func TestSendBuildsRequest(t *testing.T) {
	var gotMethod, gotAuth, gotBody, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotAuth = r.Header.Get("Authorization")
		gotQuery = r.URL.Query().Get("page")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()

	req := model.Request{
		Method:  "POST",
		URL:     srv.URL,
		Params:  []model.KV{{Key: "page", Value: "2", Enabled: true}},
		Headers: []model.KV{{Key: "Authorization", Value: "Bearer t", Enabled: true}},
		Body:    model.Body{Type: model.BodyJSON, Raw: `{"a":1}`},
	}
	resp := New().Send(context.Background(), req, emptyScope())

	if resp.Error != "" {
		t.Fatalf("unexpected error: %s", resp.Error)
	}
	if gotMethod != "POST" {
		t.Errorf("method: got %q", gotMethod)
	}
	if gotAuth != "Bearer t" {
		t.Errorf("auth header: got %q", gotAuth)
	}
	if gotQuery != "2" {
		t.Errorf("query param: got %q", gotQuery)
	}
	if gotBody != `{"a":1}` {
		t.Errorf("body: got %q", gotBody)
	}
	if resp.Status != 200 {
		t.Errorf("status: got %d", resp.Status)
	}
	// JSON pretty-printed.
	if !strings.Contains(resp.Body, "\n") {
		t.Errorf("expected indented JSON, got %q", resp.Body)
	}
}

func TestMultipartBody(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/avatar.png"
	if err := os.WriteFile(path, []byte("PNGDATA"), 0o644); err != nil {
		t.Fatal(err)
	}

	var gotName, gotFile, gotField string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("not multipart: %v", err)
			return
		}
		gotField = r.FormValue("name")
		f, hdr, err := r.FormFile("avatar")
		if err != nil {
			t.Errorf("file field: %v", err)
			return
		}
		defer f.Close()
		data, _ := io.ReadAll(f)
		gotName, gotFile = hdr.Filename, string(data)
	}))
	defer srv.Close()

	req := model.Request{
		Method: "POST",
		URL:    srv.URL,
		Body: model.Body{Type: model.BodyMultipart, Form: []model.KV{
			{Key: "name", Value: "bob", Enabled: true},
			{Key: "avatar", Value: path, Enabled: true, File: true},
			{Key: "skipped", Value: "x", Enabled: false},
		}},
	}
	resp := New().Send(context.Background(), req, emptyScope())
	if resp.Error != "" {
		t.Fatalf("error: %s", resp.Error)
	}
	if gotField != "bob" || gotName != "avatar.png" || gotFile != "PNGDATA" {
		t.Errorf("got field=%q file=%q data=%q", gotField, gotName, gotFile)
	}
}

func TestMultipartMissingFile(t *testing.T) {
	req := model.Request{
		Method: "POST",
		URL:    "http://unused.invalid",
		Body: model.Body{Type: model.BodyMultipart, Form: []model.KV{
			{Key: "f", Value: "/no/such/file", Enabled: true, File: true},
		}},
	}
	resp := New().Send(context.Background(), req, emptyScope())
	if resp.Error == "" || !strings.Contains(resp.Error, "multipart field") {
		t.Fatalf("want multipart error, got %+v", resp)
	}
}

func TestGraphQLBody(t *testing.T) {
	var gotBody, gotCT string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody, gotCT = string(b), r.Header.Get("Content-Type")
	}))
	defer srv.Close()

	req := model.Request{
		Method: "POST",
		URL:    srv.URL,
		Body: model.Body{
			Type:      model.BodyGraphQL,
			Raw:       `query($id: ID!) { user(id: $id) { name } }`,
			Variables: `{"id": "7"}`,
		},
	}
	resp := New().Send(context.Background(), req, emptyScope())
	if resp.Error != "" {
		t.Fatalf("error: %s", resp.Error)
	}
	if gotCT != "application/json" {
		t.Errorf("content-type = %q", gotCT)
	}
	if !strings.Contains(gotBody, `"query":"query($id: ID!)`) || !strings.Contains(gotBody, `"variables":{"id":"7"}`) {
		t.Errorf("body = %s", gotBody)
	}
}

func TestGraphQLBadVariables(t *testing.T) {
	req := model.Request{
		Method: "POST",
		URL:    "http://unused.invalid",
		Body:   model.Body{Type: model.BodyGraphQL, Raw: "{ ping }", Variables: "{nope"},
	}
	resp := New().Send(context.Background(), req, emptyScope())
	if resp.Error == "" || !strings.Contains(resp.Error, "variables") {
		t.Fatalf("want variables error, got %+v", resp)
	}
}

func TestCookieJarPersistsAcrossSends(t *testing.T) {
	var gotCookie string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "s-1", Path: "/"})
		case "/me":
			if c, err := r.Cookie("session"); err == nil {
				gotCookie = c.Value
			}
		}
	}))
	defer srv.Close()

	c := New()
	ctx := context.Background()
	c.Send(ctx, model.Request{Method: "GET", URL: srv.URL + "/login"}, emptyScope())
	c.Send(ctx, model.Request{Method: "GET", URL: srv.URL + "/me"}, emptyScope())
	if gotCookie != "s-1" {
		t.Fatalf("cookie not persisted, got %q", gotCookie)
	}

	cookies, err := c.Cookies(srv.URL)
	if err != nil || len(cookies) != 1 || cookies[0].Name != "session" {
		t.Errorf("Cookies() = %+v, %v", cookies, err)
	}

	c.ClearCookies()
	gotCookie = ""
	c.Send(ctx, model.Request{Method: "GET", URL: srv.URL + "/me"}, emptyScope())
	if gotCookie != "" {
		t.Errorf("cookie survived ClearCookies: %q", gotCookie)
	}
}

func TestTruncation(t *testing.T) {
	big := strings.Repeat("x", MaxInlineBytes+1000)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, big)
	}))
	defer srv.Close()

	resp := New().Send(context.Background(), model.Request{Method: "GET", URL: srv.URL}, emptyScope())
	if !resp.Truncated {
		t.Error("expected truncated=true for oversized body")
	}
	if int(resp.SizeBytes) != len(big) {
		t.Errorf("size should reflect full body: got %d want %d", resp.SizeBytes, len(big))
	}
	if len(resp.Body) > MaxInlineBytes {
		t.Errorf("inline body should be capped: got %d", len(resp.Body))
	}
}

func TestVarInterpolationInURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "path=%s", r.URL.Path)
	}))
	defer srv.Close()

	scope := vars.Build([]model.KV{{Key: "base", Value: srv.URL, Enabled: true}})
	resp := New().Send(context.Background(), model.Request{Method: "GET", URL: "{{base}}/users"}, scope)
	if resp.Body != "path=/users" {
		t.Errorf("interpolated URL path: got %q", resp.Body)
	}
}

func TestTransportErrorCaptured(t *testing.T) {
	resp := New().Send(context.Background(), model.Request{Method: "GET", URL: "http://127.0.0.1:1"}, emptyScope())
	if resp.Error == "" {
		t.Error("expected transport error to be captured in Response.Error")
	}
}
