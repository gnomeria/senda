package vars

import (
	"reflect"
	"testing"

	"senda/internal/model"
)

func kv(k, v string) model.KV { return model.KV{Key: k, Value: v, Enabled: true} }

func TestPrecedence(t *testing.T) {
	coll := []model.KV{kv("baseUrl", "http://coll"), kv("token", "coll-tok")}
	env := []model.KV{kv("baseUrl", "http://env")}
	// Lowest precedence first: collection, then environment.
	sc := Build(coll, env)

	if got := sc.Apply("{{baseUrl}}/x"); got != "http://env/x" {
		t.Errorf("env should override collection: got %q", got)
	}
	if got := sc.Apply("{{token}}"); got != "coll-tok" {
		t.Errorf("fall back to collection: got %q", got)
	}
}

func TestUnresolved(t *testing.T) {
	sc := Build([]model.KV{kv("a", "1")})
	got := sc.Apply("{{a}}-{{missing}}-{{missing}}")
	if got != "1-{{missing}}-{{missing}}" {
		t.Errorf("unknown left verbatim: got %q", got)
	}
	if !reflect.DeepEqual(sc.Unresolved, []string{"missing"}) {
		t.Errorf("unresolved dedup: got %v", sc.Unresolved)
	}
}

func TestWhitespaceAndDisabled(t *testing.T) {
	sc := Build([]model.KV{
		kv("x", "ok"),
		{Key: "y", Value: "no", Enabled: false},
	})
	if got := sc.Apply("{{ x }}"); got != "ok" {
		t.Errorf("trim inside braces: got %q", got)
	}
	if got := sc.Apply("{{y}}"); got != "{{y}}" {
		t.Errorf("disabled var ignored: got %q", got)
	}
}

func TestApplyKVsDropsDisabled(t *testing.T) {
	sc := Build([]model.KV{kv("v", "X")})
	in := []model.KV{kv("k", "{{v}}"), {Key: "off", Value: "y", Enabled: false}}
	out := sc.ApplyKVs(in)
	if len(out) != 1 || out[0].Value != "X" {
		t.Errorf("interpolate + drop disabled: got %+v", out)
	}
}
