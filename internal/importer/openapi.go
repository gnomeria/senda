package importer

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"

	"senda/internal/model"
)

// OpenAPI parses an OpenAPI 3 document (JSON or YAML — JSON is valid YAML) into
// one request per path+operation. The first server URL is used as the base.
// $ref references (parameters, request bodies, schemas) are resolved by
// libopenapi before walking the model.
func OpenAPI(data []byte) ([]Imported, error) {
	// SkipExternalRefResolution leaves external file/remote refs (e.g. a docs
	// markdown file referenced from an extension) as-is instead of failing when
	// the sibling files aren't present. Internal #/components refs still resolve.
	doc, err := libopenapi.NewDocumentWithConfiguration(data, &datamodel.DocumentConfiguration{
		SkipExternalRefResolution: true,
	})
	if err != nil {
		return nil, fmt.Errorf("openapi: %w", err)
	}
	v3doc, err := doc.BuildV3Model()
	if err != nil {
		return nil, fmt.Errorf("openapi: %w", err)
	}
	if v3doc == nil || v3doc.Model.Paths == nil || v3doc.Model.Paths.PathItems == nil {
		return nil, fmt.Errorf("openapi: no paths found")
	}

	base := "{{baseUrl}}"
	for _, s := range v3doc.Model.Servers {
		if s != nil && s.URL != "" {
			base = strings.TrimRight(s.URL, "/")
			break
		}
	}

	items := v3doc.Model.Paths.PathItems

	// Walk paths in sorted order for deterministic output.
	paths := make([]string, 0, items.Len())
	for p := range items.KeysFromOldest() {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	var out []Imported
	for _, p := range paths {
		item := items.GetOrZero(p)
		if item == nil {
			continue
		}
		// Path-level parameters apply to every operation under the path.
		for method, op := range item.GetOperations().FromOldest() {
			if op == nil {
				continue
			}
			out = append(out, Imported{
				Dir:     dirFromTags(op.Tags),
				Request: convertOpenAPI(base, p, method, op, item.Parameters),
			})
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("openapi: no operations found")
	}
	return out, nil
}

func convertOpenAPI(base, path, method string, op *v3.Operation, pathParams []*v3.Parameter) model.Request {
	name := op.OperationId
	if name == "" {
		name = strings.ToLower(method) + "-" + strings.Trim(path, "/")
	}
	req := model.Request{
		Name:   sanitize(name),
		Method: strings.ToUpper(method),
		URL:    base + path,
		Auth:   model.Auth{Type: model.AuthInherit},
		Body:   model.Body{Type: model.BodyNone},
	}
	// Operation parameters override path-level ones with the same name+location.
	seen := map[string]bool{}
	addParam := func(pr *v3.Parameter) {
		if pr == nil || pr.Name == "" {
			return
		}
		key := pr.In + "\x00" + pr.Name
		if seen[key] {
			return
		}
		seen[key] = true
		switch pr.In {
		case "query":
			req.Params = append(req.Params, model.KV{Key: pr.Name, Value: "", Enabled: false})
		case "header":
			req.Headers = append(req.Headers, model.KV{Key: pr.Name, Value: "", Enabled: false})
		}
	}
	for _, pr := range op.Parameters {
		addParam(pr)
	}
	for _, pr := range pathParams {
		addParam(pr)
	}

	if raw, ok := jsonExample(op.RequestBody); ok {
		req.Body = model.Body{Type: model.BodyJSON, Raw: raw}
		req.Headers = append(req.Headers, model.KV{Key: "Content-Type", Value: "application/json", Enabled: true})
	}
	return req
}

// jsonExample pulls an example payload for application/json if the spec carries
// one, marshalling it back to a pretty JSON string.
func jsonExample(rb *v3.RequestBody) (string, bool) {
	if rb == nil || rb.Content == nil {
		return "", false
	}
	mt := rb.Content.GetOrZero("application/json")
	if mt == nil {
		return "", false
	}
	var val interface{}
	switch {
	case mt.Example != nil:
		_ = mt.Example.Decode(&val)
	case mt.Examples != nil:
		for ex := range mt.Examples.ValuesFromOldest() {
			if ex != nil && ex.Value != nil {
				_ = ex.Value.Decode(&val)
				break
			}
		}
	}
	if val == nil {
		return "", false
	}
	b, err := marshalJSON(val)
	if err != nil {
		return "", false
	}
	return b, true
}

func dirFromTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	return []string{sanitize(tags[0])}
}
