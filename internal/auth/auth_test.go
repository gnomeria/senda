package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"senda/internal/model"
	"senda/internal/vars"
)

func newReq(t *testing.T) *http.Request {
	t.Helper()
	r, err := http.NewRequest(http.MethodGet, "https://example.test/path?a=1", nil)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestBearer(t *testing.T) {
	r := newReq(t)
	scope := vars.Build([]model.KV{{Key: "tok", Value: "secret", Enabled: true}})
	if err := Apply(context.Background(), http.DefaultClient, r, model.Auth{Type: model.AuthBearer, Token: "{{tok}}"}, scope); err != nil {
		t.Fatal(err)
	}
	if got := r.Header.Get("Authorization"); got != "Bearer secret" {
		t.Errorf("got %q", got)
	}
}

func TestBasic(t *testing.T) {
	r := newReq(t)
	if err := Apply(context.Background(), http.DefaultClient, r, model.Auth{Type: model.AuthBasic, Username: "u", Password: "p"}, vars.Build()); err != nil {
		t.Fatal(err)
	}
	u, p, ok := r.BasicAuth()
	if !ok || u != "u" || p != "p" {
		t.Errorf("basic auth: %q %q ok=%v", u, p, ok)
	}
}

func TestAPIKeyHeader(t *testing.T) {
	r := newReq(t)
	a := model.Auth{Type: model.AuthAPIKey, Key: "X-API-Key", KeyValue: "abc", Placement: model.APIKeyHeader}
	if err := Apply(context.Background(), http.DefaultClient, r, a, vars.Build()); err != nil {
		t.Fatal(err)
	}
	if got := r.Header.Get("X-API-Key"); got != "abc" {
		t.Errorf("got %q", got)
	}
}

func TestAPIKeyQuery(t *testing.T) {
	r := newReq(t)
	a := model.Auth{Type: model.AuthAPIKey, Key: "api_key", KeyValue: "abc", Placement: model.APIKeyQuery}
	if err := Apply(context.Background(), http.DefaultClient, r, a, vars.Build()); err != nil {
		t.Fatal(err)
	}
	q := r.URL.Query()
	if q.Get("api_key") != "abc" {
		t.Errorf("query api_key: got %q", q.Get("api_key"))
	}
	if q.Get("a") != "1" {
		t.Errorf("existing query param clobbered: got %q", q.Get("a"))
	}
}

func TestNoneAndInheritAreNoOps(t *testing.T) {
	for _, ty := range []model.AuthType{model.AuthNone, model.AuthInherit, ""} {
		r := newReq(t)
		if err := Apply(context.Background(), http.DefaultClient, r, model.Auth{Type: ty}, vars.Build()); err != nil {
			t.Fatal(err)
		}
		if r.Header.Get("Authorization") != "" {
			t.Errorf("type %q should not set Authorization", ty)
		}
	}
}

func TestOAuth2ClientCredentials(t *testing.T) {
	var gotGrant, gotID, gotSecret, gotScope string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		_ = req.ParseForm()
		gotGrant = req.Form.Get("grant_type")
		gotID = req.Form.Get("client_id")
		gotSecret = req.Form.Get("client_secret")
		gotScope = req.Form.Get("scope")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"tok-123","token_type":"Bearer"}`))
	}))
	defer srv.Close()

	r := newReq(t)
	a := model.Auth{
		Type: model.AuthOAuth2, Grant: model.OAuth2ClientCredentials,
		TokenURL: srv.URL, ClientID: "id", ClientSecret: "sec", Scope: "read write",
	}
	if err := Apply(context.Background(), srv.Client(), r, a, vars.Build()); err != nil {
		t.Fatal(err)
	}
	if r.Header.Get("Authorization") != "Bearer tok-123" {
		t.Errorf("authorization: got %q", r.Header.Get("Authorization"))
	}
	if gotGrant != "client_credentials" || gotID != "id" || gotSecret != "sec" || gotScope != "read write" {
		t.Errorf("token request form wrong: grant=%q id=%q secret=%q scope=%q", gotGrant, gotID, gotSecret, gotScope)
	}
}

func TestOAuth2ErrorPropagates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"error":"invalid_client"}`))
	}))
	defer srv.Close()

	r := newReq(t)
	a := model.Auth{Type: model.AuthOAuth2, TokenURL: srv.URL, ClientID: "id"}
	err := Apply(context.Background(), srv.Client(), r, a, vars.Build())
	if err == nil {
		t.Fatal("expected error from failing token endpoint")
	}
}
