package assert

import (
	"testing"

	"senda/internal/model"
)

var resp = model.Response{
	Status:     200,
	DurationMs: 42,
	SizeBytes:  1234,
	Headers: map[string][]string{
		"Content-Type": {"application/json; charset=utf-8"},
	},
	Body: `{"user":{"id":7,"name":"Ada","tags":["x","y"],"active":true,"score":1.0,"nick":null},"items":[{"id":1},{"id":2}]}`,
}

func one(target, op, value string) model.AssertResult {
	out := Eval([]model.Assert{{Target: target, Op: op, Value: value, Enabled: true}}, resp)
	if len(out) != 1 {
		panic("expected one result")
	}
	return out[0]
}

func TestDisabledSkipped(t *testing.T) {
	out := Eval([]model.Assert{{Target: "status", Op: "eq", Value: "200", Enabled: false}}, resp)
	if len(out) != 0 {
		t.Fatalf("disabled assert produced result: %+v", out)
	}
}

func TestPassing(t *testing.T) {
	cases := []struct{ target, op, value string }{
		{"status", "eq", "200"},
		{"status", "neq", "404"},
		{"status", "gte", "200"},
		{"status", "lt", "300"},
		{"duration", "lte", "42"},
		{"size", "gt", "1000"},
		{"body", "contains", `"name":"Ada"`},
		{"body", "notcontains", "error"},
		{"header.content-type", "contains", "application/json"},
		{"header.Content-Type", "matches", `^application/json`},
		{"header.Content-Type", "exists", ""},
		{"header.X-Missing", "notexists", ""},
		{"json.user.id", "eq", "7"},
		{"json.user.name", "eq", "Ada"},
		{"json.user.active", "eq", "true"},
		{"json.user.nick", "eq", "null"},
		{"json.user.score", "eq", "1"}, // numeric eq: 1.0 == 1
		{"json.user.tags[1]", "eq", "y"},
		{"json.items[1].id", "eq", "2"},
		{"json.user.tags", "eq", `["x","y"]`},
		{"json.user.id", "exists", ""},
		{"json.user.missing", "notexists", ""},
		{"json.items[9]", "notexists", ""},
	}
	for _, c := range cases {
		if r := one(c.target, c.op, c.value); !r.Pass || r.Error != "" {
			t.Errorf("%s %s %q: pass=%v err=%q actual=%q", c.target, c.op, c.value, r.Pass, r.Error, r.Actual)
		}
	}
}

func TestFailing(t *testing.T) {
	cases := []struct{ target, op, value string }{
		{"status", "eq", "404"},
		{"body", "contains", "nope"},
		{"json.user.id", "gt", "10"},
		{"json.user.missing", "exists", ""},
		{"header.X-Missing", "exists", ""},
	}
	for _, c := range cases {
		if r := one(c.target, c.op, c.value); r.Pass {
			t.Errorf("%s %s %q: expected fail, got pass (actual %q)", c.target, c.op, c.value, r.Actual)
		}
	}
}

func TestErrors(t *testing.T) {
	cases := []struct{ target, op, value string }{
		{"", "eq", "1"},                  // empty target
		{"bogus", "eq", "1"},             // unknown target
		{"status", "frob", "1"},          // unknown op
		{"status", "", "1"},              // empty op
		{"body", "gt", "1"},              // non-numeric gt
		{"status", "matches", "("},       // bad regex
		{"json.user.missing", "eq", "x"}, // missing path under value op
		{"json.user[", "eq", "x"},        // malformed path
	}
	for _, c := range cases {
		if r := one(c.target, c.op, c.value); r.Pass || r.Error == "" {
			t.Errorf("%s %s %q: expected error, got pass=%v err=%q", c.target, c.op, c.value, r.Pass, r.Error)
		}
	}
}

func TestNonJSONBodyJSONTarget(t *testing.T) {
	r := Eval([]model.Assert{{Target: "json.a", Op: "eq", Value: "1", Enabled: true}},
		model.Response{Body: "<html>"})
	if len(r) != 1 || r[0].Pass || r[0].Error == "" {
		t.Fatalf("expected JSON decode error, got %+v", r)
	}
}

func TestActualReported(t *testing.T) {
	r := one("json.user.name", "eq", "Bob")
	if r.Pass || r.Actual != "Ada" {
		t.Fatalf("want fail with actual Ada, got %+v", r)
	}
}
