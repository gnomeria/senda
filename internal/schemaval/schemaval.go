// Package schemaval validates a JSON response body against an inline JSON
// Schema. Results are returned as AssertResult rows so they appear alongside
// regular assertions in the response panel.
package schemaval

import (
	"fmt"
	"strings"

	"github.com/xeipuuv/gojsonschema"

	"senda/internal/model"
)

// Validate checks body against the provided inline JSON Schema string and
// returns one AssertResult per schema violation (or one pass result when
// the body is valid). Returns nil when schema is empty.
func Validate(schema, body string) []model.AssertResult {
	schema = strings.TrimSpace(schema)
	if schema == "" {
		return nil
	}

	sl := gojsonschema.NewStringLoader(schema)
	bl := gojsonschema.NewStringLoader(body)
	result, err := gojsonschema.Validate(sl, bl)
	if err != nil {
		return []model.AssertResult{{
			Target: "schema",
			Op:     "valid",
			Pass:   false,
			Error:  fmt.Sprintf("schema validation error: %v", err),
		}}
	}

	if result.Valid() {
		return []model.AssertResult{{
			Target: "schema",
			Op:     "valid",
			Pass:   true,
		}}
	}

	out := make([]model.AssertResult, 0, len(result.Errors()))
	for _, e := range result.Errors() {
		field := e.Field()
		if field == "(root)" {
			field = "schema"
		} else {
			field = "schema." + field
		}
		out = append(out, model.AssertResult{
			Target: field,
			Op:     "valid",
			Pass:   false,
			Error:  e.Description(),
		})
	}
	return out
}
