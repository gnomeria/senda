package schemaval

import (
	"testing"
)

const schema = `{
	"type": "object",
	"required": ["id", "name"],
	"properties": {
		"id":   {"type": "integer"},
		"name": {"type": "string"}
	}
}`

func TestValidBody(t *testing.T) {
	results := Validate(schema, `{"id":1,"name":"Alice"}`)
	if len(results) != 1 || !results[0].Pass {
		t.Errorf("expected one passing result, got %+v", results)
	}
}

func TestMissingField(t *testing.T) {
	results := Validate(schema, `{"id":1}`)
	if len(results) == 0 {
		t.Fatal("expected validation errors")
	}
	for _, r := range results {
		if r.Pass {
			t.Errorf("expected fail, got pass: %+v", r)
		}
	}
}

func TestWrongType(t *testing.T) {
	results := Validate(schema, `{"id":"not-an-int","name":"Bob"}`)
	if len(results) == 0 || results[0].Pass {
		t.Errorf("expected type mismatch failure, got %+v", results)
	}
}

func TestNonJSONBody(t *testing.T) {
	results := Validate(schema, `<html>not json</html>`)
	if len(results) == 0 || results[0].Pass {
		t.Errorf("expected error for non-JSON body, got %+v", results)
	}
}

func TestEmptySchema(t *testing.T) {
	results := Validate("", `{"anything":"goes"}`)
	if len(results) != 0 {
		t.Errorf("empty schema should return nil, got %+v", results)
	}
}
