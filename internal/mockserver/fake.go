package mockserver

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// Hand-rolled faker. Small word lists keep the binary small (no faker
// dependency). Used both by template functions ({{faker.name}}) and by
// schema-driven fake bodies.

var (
	fakeFirst = []string{"Ada", "Alan", "Grace", "Linus", "Ken", "Dennis", "Margaret", "Edsger", "Barbara", "Donald", "Katherine", "Tim"}
	fakeLast  = []string{"Lovelace", "Turing", "Hopper", "Torvalds", "Thompson", "Ritchie", "Hamilton", "Dijkstra", "Liskov", "Knuth", "Johnson", "Berners-Lee"}
	fakeWords = []string{"lorem", "ipsum", "dolor", "sit", "amet", "consectetur", "adipiscing", "elit", "sed", "tempor", "labore", "magna"}
	fakeCity  = []string{"London", "Helsinki", "Tokyo", "Berlin", "Austin", "Toronto", "Oslo", "Lisbon"}
	fakeCtry  = []string{"Finland", "Japan", "Germany", "Canada", "Portugal", "Norway", "Brazil", "Kenya"}
	fakeCorp  = []string{"Acme", "Globex", "Initech", "Umbrella", "Hooli", "Vandelay", "Stark", "Wayne"}
)

func pick(s []string) string { return s[rand.Intn(len(s))] }

func fakeName() string { return pick(fakeFirst) + " " + pick(fakeLast) }
func fakeEmail() string {
	return strings.ToLower(pick(fakeFirst) + "." + pick(fakeLast) + "@example.com")
}
func fakeUsername() string { return strings.ToLower(pick(fakeFirst)) + fmt.Sprint(rand.Intn(1000)) }
func fakeWord() string     { return pick(fakeWords) }
func fakeSentence() string {
	n := 4 + rand.Intn(6)
	w := make([]string, n)
	for i := range w {
		w[i] = pick(fakeWords)
	}
	s := strings.Join(w, " ")
	return strings.ToUpper(s[:1]) + s[1:] + "."
}
func fakeUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
func fakePhone() string {
	return fmt.Sprintf("+1-%03d-%03d-%04d", rand.Intn(1000), rand.Intn(1000), rand.Intn(10000))
}

// fakerFunc resolves {{faker.NAME}} to a generated value.
func fakerFunc(name string) (string, bool) {
	switch strings.ToLower(name) {
	case "name":
		return fakeName(), true
	case "firstname":
		return pick(fakeFirst), true
	case "lastname":
		return pick(fakeLast), true
	case "email":
		return fakeEmail(), true
	case "username":
		return fakeUsername(), true
	case "uuid":
		return fakeUUID(), true
	case "int":
		return fmt.Sprint(rand.Intn(1000)), true
	case "float":
		return fmt.Sprintf("%.2f", rand.Float64()*1000), true
	case "bool":
		return fmt.Sprint(rand.Intn(2) == 1), true
	case "word":
		return fakeWord(), true
	case "sentence":
		return fakeSentence(), true
	case "city":
		return pick(fakeCity), true
	case "country":
		return pick(fakeCtry), true
	case "company":
		return pick(fakeCorp), true
	case "phone":
		return fakePhone(), true
	case "date":
		return time.Now().UTC().Format("2006-01-02"), true
	case "datetime":
		return time.Now().UTC().Format(time.RFC3339), true
	}
	return "", false
}

// fakeFromSchema generates a minimal fake JSON object from a JSON Schema.
// Supports type:object with string/integer/number/boolean/array properties.
func fakeFromSchema(schema string) string {
	var s map[string]any
	if err := json.Unmarshal([]byte(schema), &s); err != nil {
		return "{}"
	}
	out, err := json.MarshalIndent(buildFake(s), "", "  ")
	if err != nil {
		return "{}"
	}
	return string(out)
}

func buildFake(schema map[string]any) any {
	// Honour a concrete value the schema carries before synthesising one.
	if ex, ok := schema["example"]; ok {
		return ex
	}
	if ex, ok := schema["examples"].([]any); ok && len(ex) > 0 {
		return ex[0]
	}
	if d, ok := schema["default"]; ok {
		return d
	}
	switch t := schemaType(schema); t {
	case "object":
		props, _ := schema["properties"].(map[string]any)
		obj := map[string]any{}
		for k, v := range props {
			if vs, ok := v.(map[string]any); ok {
				obj[k] = buildFakeProp(k, vs)
			}
		}
		return obj
	case "array":
		if items, ok := schema["items"].(map[string]any); ok {
			return []any{buildFake(items)}
		}
		return []any{}
	case "integer", "number":
		if ex, ok := schema["examples"].([]any); ok && len(ex) > 0 {
			return ex[0]
		}
		return rand.Intn(1000)
	case "boolean":
		return rand.Intn(2) == 1
	default: // string
		return fakeStringFor("", schema)
	}
}

// schemaType returns the JSON Schema type, tolerating OpenAPI 3.1's union form
// where "type" is a list (e.g. ["string","null"]) — the first non-null entry
// wins. An absent type yields "" (treated as string downstream).
func schemaType(schema map[string]any) string {
	switch t := schema["type"].(type) {
	case string:
		return t
	case []any:
		for _, v := range t {
			if s, ok := v.(string); ok && s != "null" {
				return s
			}
		}
	}
	return ""
}

// buildFakeProp generates a value for a named property, using the property name
// as a hint (e.g. "email", "name") when the schema gives no format/examples.
func buildFakeProp(name string, schema map[string]any) any {
	if _, ok := schema["example"]; ok {
		return buildFake(schema)
	}
	switch schemaType(schema) {
	case "string", "":
		return fakeStringFor(name, schema)
	default:
		return buildFake(schema)
	}
}

func fakeStringFor(name string, schema map[string]any) string {
	if ex, ok := schema["examples"].([]any); ok && len(ex) > 0 {
		return fmt.Sprintf("%v", ex[0])
	}
	if enum, ok := schema["enum"].([]any); ok && len(enum) > 0 {
		return fmt.Sprintf("%v", enum[0])
	}
	if f, _ := schema["format"].(string); f != "" {
		switch f {
		case "email":
			return fakeEmail()
		case "uuid":
			return fakeUUID()
		case "date":
			return time.Now().UTC().Format("2006-01-02")
		case "date-time":
			return time.Now().UTC().Format(time.RFC3339)
		case "uri", "url":
			return "https://example.com"
		}
	}
	// Property-name heuristics.
	switch n := strings.ToLower(name); {
	case strings.Contains(n, "email"):
		return fakeEmail()
	case strings.Contains(n, "name"):
		return fakeName()
	case strings.Contains(n, "phone"):
		return fakePhone()
	case strings.Contains(n, "city"):
		return pick(fakeCity)
	case strings.Contains(n, "country"):
		return pick(fakeCtry)
	case strings.Contains(n, "id"):
		return fakeUUID()
	case n != "":
		return fakeWord()
	}
	return "example"
}
