package codegen

import (
	"strings"
	"testing"

	"senda/internal/model"
)

func sampleReq() model.Request {
	return model.Request{
		Name:   "create-user",
		Method: "post",
		URL:    "https://api.test/users",
		Params: []model.KV{{Key: "verbose", Value: "1", Enabled: true}},
		Headers: []model.KV{
			{Key: "Accept", Value: "application/json", Enabled: true},
			{Key: "X-Off", Value: "no", Enabled: false},
		},
		Body: model.Body{Type: model.BodyJSON, Raw: `{"name":"bob"}`},
		Auth: model.Auth{Type: model.AuthBearer, Token: "{{token}}"},
	}
}

func TestGenerateAllTargets(t *testing.T) {
	req := sampleReq()
	for _, target := range Targets {
		out, err := Generate(req, target)
		if err != nil {
			t.Fatalf("%s: %v", target, err)
		}
		if out == "" {
			t.Errorf("%s: empty output", target)
		}
		if !strings.Contains(out, "api.test/users") {
			t.Errorf("%s: url missing:\n%s", target, out)
		}
		if strings.Contains(out, "X-Off") {
			t.Errorf("%s: disabled header leaked:\n%s", target, out)
		}
		if !strings.Contains(out, "verbose=1") {
			t.Errorf("%s: enabled param missing:\n%s", target, out)
		}
		if !strings.Contains(out, "{{token}}") {
			t.Errorf("%s: placeholder not preserved:\n%s", target, out)
		}
	}
}

func TestGenerateCurlMethodAndBody(t *testing.T) {
	out, err := Generate(sampleReq(), "curl")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "-X POST") {
		t.Errorf("missing method:\n%s", out)
	}
	if !strings.Contains(out, `Authorization: Bearer {{token}}`) {
		t.Errorf("missing bearer header:\n%s", out)
	}
}

func TestGenerateUnknownTarget(t *testing.T) {
	if _, err := Generate(sampleReq(), "cobol"); err == nil {
		t.Error("want error for unknown target")
	}
}

func TestGenerateBasicAuthCurl(t *testing.T) {
	req := sampleReq()
	req.Auth = model.Auth{Type: model.AuthBasic, Username: "u", Password: "p"}
	out, _ := Generate(req, "curl")
	if !strings.Contains(out, "-u 'u:p'") {
		t.Errorf("basic auth -u missing:\n%s", out)
	}
}

func TestGenerateAPIKeyQuery(t *testing.T) {
	req := sampleReq()
	req.Auth = model.Auth{Type: model.AuthAPIKey, Key: "api_key", KeyValue: "xyz", Placement: model.APIKeyQuery}
	out, _ := Generate(req, "curl")
	if !strings.Contains(out, "api_key=xyz") {
		t.Errorf("api key query missing:\n%s", out)
	}
}
