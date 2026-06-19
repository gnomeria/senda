package importer

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"

	"senda/internal/mockserver"
)

// OpenAPIMocks turns an OpenAPI 3 document into mock-server definitions — one
// rule route per path+operation, with one response variant per documented
// status code. A response body is taken from the spec's example when present,
// otherwise the response schema is embedded so the mock server fakes a body at
// request time. The result can be marshalled to YAML and dropped into a
// collection's mocks/ directory to serve the API immediately.
func OpenAPIMocks(data []byte) ([]mockserver.MockDef, error) {
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

	items := v3doc.Model.Paths.PathItems
	paths := make([]string, 0, items.Len())
	for p := range items.KeysFromOldest() {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	var out []mockserver.MockDef
	for _, p := range paths {
		item := items.GetOrZero(p)
		if item == nil {
			continue
		}
		for method, op := range item.GetOperations().FromOldest() {
			if op == nil {
				continue
			}
			out = append(out, mockDefForOp(p, method, op))
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("openapi: no operations found")
	}
	return out, nil
}

func mockDefForOp(path, method string, op *v3.Operation) mockserver.MockDef {
	name := op.OperationId
	if name == "" {
		name = strings.ToLower(method) + "-" + strings.Trim(path, "/")
	}
	return mockserver.MockDef{
		Name:      sanitize(name),
		Method:    strings.ToUpper(method),
		Path:      openAPIPathToMock(path),
		Responses: responseDefs(op),
	}
}

// responseDefs builds one ResponseDef per documented status code, sorted so a
// 2xx comes first (it becomes the default the server returns), then the rest in
// ascending order. A bare "default" response maps to 200 only when no explicit
// 2xx exists, so it never shadows a real success response.
func responseDefs(op *v3.Operation) []mockserver.ResponseDef {
	if op.Responses == nil {
		return nil
	}
	type rd struct {
		status int
		def    mockserver.ResponseDef
	}
	var coded []rd
	hasSuccess := false
	if op.Responses.Codes != nil {
		for code, resp := range op.Responses.Codes.FromOldest() {
			st, ok := statusCode(code)
			if !ok {
				continue
			}
			if st >= 200 && st < 300 {
				hasSuccess = true
			}
			coded = append(coded, rd{status: st, def: responseDef(st, resp)})
		}
	}
	if d := op.Responses.Default; d != nil && !hasSuccess {
		coded = append(coded, rd{status: 200, def: responseDef(200, d)})
	}

	sort.SliceStable(coded, func(i, j int) bool {
		si, sj := coded[i].status, coded[j].status
		ok2 := func(s int) bool { return s >= 200 && s < 300 }
		if ok2(si) != ok2(sj) {
			return ok2(si) // 2xx first
		}
		return si < sj
	})

	out := make([]mockserver.ResponseDef, len(coded))
	for i, c := range coded {
		out[i] = c.def
	}
	return out
}

func responseDef(status int, resp *v3.Response) mockserver.ResponseDef {
	def := mockserver.ResponseDef{Status: status, Desc: strings.TrimSpace(resp.Description)}
	mt := jsonMediaType(resp.Content)
	if mt == nil {
		return def
	}
	if body, ok := mediaTypeExample(mt); ok {
		def.Body = body
		return def
	}
	if schema, ok := schemaJSON(mt.Schema); ok {
		def.Schema = schema
	}
	return def
}

// jsonMediaType returns the application/json media type, falling back to the
// first *+json content type (e.g. application/hal+json) when present.
func jsonMediaType(content *orderedmap.Map[string, *v3.MediaType]) *v3.MediaType {
	if content == nil {
		return nil
	}
	if mt := content.GetOrZero("application/json"); mt != nil {
		return mt
	}
	for ct, mt := range content.FromOldest() {
		if strings.HasPrefix(ct, "application/") && strings.HasSuffix(ct, "+json") {
			return mt
		}
	}
	return nil
}

// mediaTypeExample pulls a concrete example payload from a media type: a direct
// example, the first of the named examples, or a schema-level example.
func mediaTypeExample(mt *v3.MediaType) (any, bool) {
	var val any
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
	if val == nil && mt.Schema != nil {
		if sc := mt.Schema.Schema(); sc != nil && sc.Example != nil {
			_ = sc.Example.Decode(&val)
		}
	}
	if val == nil {
		return nil, false
	}
	return val, true
}

// schemaJSON renders a resolved schema to a self-contained JSON string (refs
// inlined) for the mock server's schema: field, which fakes a body from it.
func schemaJSON(sp *base.SchemaProxy) (string, bool) {
	if sp == nil {
		return "", false
	}
	sc := sp.Schema()
	if sc == nil {
		return "", false
	}
	b, err := sc.MarshalJSONInline()
	if err != nil || len(b) == 0 {
		return "", false
	}
	return string(b), true
}

// openAPIPathToMock converts OpenAPI path templating ({petId}) to the mock
// server's colon form (:petId).
var braceParam = regexp.MustCompile(`\{([^}]+)\}`)

func openAPIPathToMock(path string) string {
	return braceParam.ReplaceAllString(path, ":$1")
}

func statusCode(code string) (int, bool) {
	n, err := strconv.Atoi(strings.TrimSpace(code))
	if err != nil || n < 100 || n > 599 {
		return 0, false
	}
	return n, true
}
