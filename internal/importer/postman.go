package importer

import (
	"encoding/json"
	"fmt"
	"strings"

	"senda/internal/model"
)

// Imported is one request placed at a folder path (Dir) within the target
// collection. Dir is empty for top-level requests.
type Imported struct {
	Dir     []string
	Request model.Request
}

// Postman parses a Postman Collection v2.1 JSON document into a flat list of
// imported requests, preserving its folder structure in Dir.
func Postman(data []byte) ([]Imported, error) {
	var doc struct {
		Info struct {
			Name string `json:"name"`
		} `json:"info"`
		Item []pmItem `json:"item"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("postman: %w", err)
	}
	var out []Imported
	walkPostman(doc.Item, nil, &out)
	if len(out) == 0 {
		return nil, fmt.Errorf("postman: no requests found")
	}
	return out, nil
}

type pmItem struct {
	Name    string          `json:"name"`
	Item    []pmItem        `json:"item"` // present => folder
	Request *pmRequest      `json:"request"`
	Raw     json.RawMessage `json:"-"`
}

type pmRequest struct {
	Method string          `json:"method"`
	Header pmHeaders       `json:"header"`
	Body   *pmBody         `json:"body"`
	URL    json.RawMessage `json:"url"`
	Auth   *pmAuth         `json:"auth"`
}

// UnmarshalJSON accepts the v2 string shorthand — `"request": "https://…"`
// means a GET to that URL — alongside the full object form.
func (r *pmRequest) UnmarshalJSON(data []byte) error {
	var s string
	if json.Unmarshal(data, &s) == nil {
		*r = pmRequest{Method: "GET", URL: json.RawMessage(data)}
		return nil
	}
	type alias pmRequest // drop methods to avoid recursion
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*r = pmRequest(a)
	return nil
}

type pmHeader struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Disabled bool   `json:"disabled"`
}

// pmHeaders accepts the array form and the v2 raw-HTTP-block string form
// ("Key: value\r\nKey2: value2").
type pmHeaders []pmHeader

func (h *pmHeaders) UnmarshalJSON(data []byte) error {
	var arr []pmHeader
	if json.Unmarshal(data, &arr) == nil {
		*h = arr
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("header: expected array or string")
	}
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimRight(line, "\r")
		k, v, ok := strings.Cut(line, ":")
		if !ok || strings.TrimSpace(k) == "" {
			continue
		}
		*h = append(*h, pmHeader{Key: strings.TrimSpace(k), Value: strings.TrimSpace(v)})
	}
	return nil
}

type pmBody struct {
	Mode       string      `json:"mode"`
	Raw        string      `json:"raw"`
	URLEncoded []pmHeader  `json:"urlencoded"`
	FormData   []pmFormRow `json:"formdata"`
	GraphQL    *struct {
		Query     string `json:"query"`
		Variables string `json:"variables"`
	} `json:"graphql"`
}

// pmFormRow is one formdata row; file rows carry src (string or array).
type pmFormRow struct {
	Key      string          `json:"key"`
	Value    string          `json:"value"`
	Type     string          `json:"type"` // "text" | "file"
	Src      json.RawMessage `json:"src"`
	Disabled bool            `json:"disabled"`
}

type pmAuth struct {
	Type   string    `json:"type"`
	Bearer pmAuthKVs `json:"bearer"`
	Basic  pmAuthKVs `json:"basic"`
	APIKey pmAuthKVs `json:"apikey"`
}

type pmAuthKV struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// pmAuthKVs accepts the v2.1 array form ([{key,value}]) and the v2.0 object
// form ({"username": "...", "password": "..."}).
type pmAuthKVs []pmAuthKV

func (p *pmAuthKVs) UnmarshalJSON(data []byte) error {
	var arr []pmAuthKV
	if json.Unmarshal(data, &arr) == nil {
		*p = arr
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("auth params: expected array or object")
	}
	for k, v := range m {
		s, _ := v.(string)
		if s == "" && v != nil {
			s = fmt.Sprint(v)
		}
		*p = append(*p, pmAuthKV{Key: k, Value: s})
	}
	return nil
}

func walkPostman(items []pmItem, dir []string, out *[]Imported) {
	for _, it := range items {
		if len(it.Item) > 0 {
			walkPostman(it.Item, append(append([]string{}, dir...), sanitize(it.Name)), out)
			continue
		}
		if it.Request == nil {
			continue
		}
		req := convertPostman(it.Name, it.Request)
		*out = append(*out, Imported{Dir: dir, Request: req})
	}
}

func convertPostman(name string, pr *pmRequest) model.Request {
	req := model.Request{
		Name:   sanitize(name),
		Method: strings.ToUpper(pr.Method),
		Auth:   model.Auth{Type: model.AuthInherit},
		Body:   model.Body{Type: model.BodyNone},
	}
	if req.Method == "" {
		req.Method = "GET"
	}

	rawURL, query := parsePostmanURL(pr.URL)
	req.URL = rawURL
	req.Params = query

	for _, h := range pr.Header {
		req.Headers = append(req.Headers, model.KV{
			Key: h.Key, Value: h.Value, Enabled: !h.Disabled,
		})
	}

	if pr.Body != nil {
		switch pr.Body.Mode {
		case "raw":
			if looksJSON(pr.Body.Raw) {
				req.Body = model.Body{Type: model.BodyJSON, Raw: pr.Body.Raw}
			} else {
				req.Body = model.Body{Type: model.BodyRaw, Raw: pr.Body.Raw}
			}
		case "urlencoded":
			req.Body = model.Body{Type: model.BodyForm, Form: pmKVs(pr.Body.URLEncoded)}
		case "formdata":
			req.Body = model.Body{Type: model.BodyMultipart, Form: pmFormKVs(pr.Body.FormData)}
		case "graphql":
			if pr.Body.GraphQL != nil {
				req.Body = model.Body{
					Type:      model.BodyGraphQL,
					Raw:       pr.Body.GraphQL.Query,
					Variables: pr.Body.GraphQL.Variables,
				}
			}
		}
	}

	if pr.Auth != nil {
		req.Auth = convertPostmanAuth(pr.Auth)
	}
	return req
}

func convertPostmanAuth(a *pmAuth) model.Auth {
	get := func(kvs []pmAuthKV, key string) string {
		for _, kv := range kvs {
			if kv.Key == key {
				return kv.Value
			}
		}
		return ""
	}
	switch a.Type {
	case "bearer":
		return model.Auth{Type: model.AuthBearer, Token: get(a.Bearer, "token")}
	case "basic":
		return model.Auth{Type: model.AuthBasic, Username: get(a.Basic, "username"), Password: get(a.Basic, "password")}
	case "apikey":
		placement := model.APIKeyHeader
		if get(a.APIKey, "in") == "query" {
			placement = model.APIKeyQuery
		}
		return model.Auth{Type: model.AuthAPIKey, Key: get(a.APIKey, "key"), KeyValue: get(a.APIKey, "value"), Placement: placement}
	case "noauth":
		return model.Auth{Type: model.AuthNone}
	}
	return model.Auth{Type: model.AuthInherit}
}

func pmKVs(in []pmHeader) []model.KV {
	out := make([]model.KV, 0, len(in))
	for _, h := range in {
		out = append(out, model.KV{Key: h.Key, Value: h.Value, Enabled: !h.Disabled})
	}
	return out
}

// pmFormKVs converts formdata rows; file rows take their path from src
// (string, or first element of an array).
func pmFormKVs(in []pmFormRow) []model.KV {
	out := make([]model.KV, 0, len(in))
	for _, r := range in {
		kv := model.KV{Key: r.Key, Value: r.Value, Enabled: !r.Disabled}
		if r.Type == "file" {
			kv.File = true
			var s string
			var parts []string
			if json.Unmarshal(r.Src, &s) == nil {
				kv.Value = s
			} else if json.Unmarshal(r.Src, &parts) == nil && len(parts) > 0 {
				kv.Value = parts[0]
			}
		}
		out = append(out, kv)
	}
	return out
}

// parsePostmanURL handles both the string and object URL forms. The object
// form carries an explicit query array; the string form is returned verbatim
// (its query string remains in the URL). Object URLs without a "raw" field
// (seen in v2.0 exports) are rebuilt from protocol/host/port/path, where host
// and path may each be a string or an array of segments.
func parsePostmanURL(raw json.RawMessage) (string, []model.KV) {
	if len(raw) == 0 {
		return "", nil
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s, nil
	}
	var obj struct {
		Raw      string          `json:"raw"`
		Protocol string          `json:"protocol"`
		Host     json.RawMessage `json:"host"`
		Port     string          `json:"port"`
		Path     json.RawMessage `json:"path"`
		Query    []struct {
			Key      string `json:"key"`
			Value    string `json:"value"`
			Disabled bool   `json:"disabled"`
		} `json:"query"`
	}
	if json.Unmarshal(raw, &obj) != nil {
		return "", nil
	}
	url := obj.Raw
	if url == "" {
		host := joinSegments(obj.Host, ".")
		if host != "" {
			proto := obj.Protocol
			if proto == "" {
				proto = "https"
			}
			url = proto + "://" + host
			if obj.Port != "" {
				url += ":" + obj.Port
			}
			if path := joinSegments(obj.Path, "/"); path != "" {
				url += "/" + strings.TrimPrefix(path, "/")
			}
		}
	}
	if i := strings.Index(url, "?"); i >= 0 {
		url = url[:i] // params captured separately
	}
	var params []model.KV
	for _, q := range obj.Query {
		params = append(params, model.KV{Key: q.Key, Value: q.Value, Enabled: !q.Disabled})
	}
	return url, params
}

// joinSegments decodes a Postman string-or-array field ("a.b.c" vs
// ["a","b","c"]) into one string joined by sep.
func joinSegments(raw json.RawMessage, sep string) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var parts []string
	if json.Unmarshal(raw, &parts) == nil {
		return strings.Join(parts, sep)
	}
	return ""
}

// sanitize trims a name into something usable as a file/folder component.
func sanitize(name string) string {
	name = strings.TrimSpace(name)
	name = strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			return '-'
		}
		return r
	}, name)
	if name == "" {
		return "imported"
	}
	return name
}
