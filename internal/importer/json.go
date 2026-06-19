package importer

import (
	"bytes"
	"encoding/json"
)

// marshalJSON pretty-prints a value as JSON. Used to render OpenAPI example
// payloads back into request bodies.
func marshalJSON(v interface{}) (string, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return "", err
	}
	return string(bytes.TrimRight(buf.Bytes(), "\n")), nil
}
