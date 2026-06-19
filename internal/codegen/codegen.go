// Package codegen renders a model.Request as a runnable snippet in a target
// language/tool (curl, fetch, httpie, python-requests, go). {{var}}
// placeholders are emitted verbatim — generation is offline and does not
// resolve the environment.
package codegen

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"senda/internal/model"
)

// Targets lists the supported snippet targets in display order.
var Targets = []string{"curl", "fetch", "httpie", "python", "go"}

// Generate renders req as a snippet for the named target.
func Generate(req model.Request, target string) (string, error) {
	p := prepare(req)
	switch target {
	case "curl":
		return genCurl(p), nil
	case "fetch":
		return genFetch(p), nil
	case "httpie":
		return genHTTPie(p), nil
	case "python":
		return genPython(p), nil
	case "go":
		return genGo(p), nil
	default:
		return "", fmt.Errorf("unknown target %q", target)
	}
}

// prepared is a request flattened for code generation: a fully-built URL,
// concrete header list, and a body string with its content type. multipart
// keeps its rows (curl renders -F flags; other targets show a comment).
type prepared struct {
	method      string
	url         string
	headers     []model.KV
	body        string
	contentType string
	basicUser   string
	basicPass   string
	multipart   []model.KV
}

func prepare(req model.Request) prepared {
	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = "GET"
	}
	p := prepared{method: method, url: buildURL(req)}

	for _, h := range req.Headers {
		if h.Enabled && h.Key != "" {
			p.headers = append(p.headers, model.KV{Key: h.Key, Value: h.Value})
		}
	}

	switch req.Body.Type {
	case model.BodyJSON:
		p.body = req.Body.Raw
		p.contentType = "application/json"
	case model.BodyRaw:
		p.body = req.Body.Raw
	case model.BodyForm:
		form := url.Values{}
		for _, kv := range req.Body.Form {
			if kv.Enabled {
				form.Add(kv.Key, kv.Value)
			}
		}
		p.body = form.Encode()
		p.contentType = "application/x-www-form-urlencoded"
	case model.BodyMultipart:
		for _, kv := range req.Body.Form {
			if kv.Enabled && kv.Key != "" {
				p.multipart = append(p.multipart, kv)
			}
		}
	case model.BodyGraphQL:
		payload := map[string]any{"query": req.Body.Raw}
		if v := strings.TrimSpace(req.Body.Variables); v != "" {
			payload["variables"] = json.RawMessage(v)
		}
		if data, err := json.Marshal(payload); err == nil {
			p.body = string(data)
		}
		p.contentType = "application/json"
	}
	if p.contentType != "" && !hasHeader(p.headers, "Content-Type") {
		p.headers = append(p.headers, model.KV{Key: "Content-Type", Value: p.contentType})
	}

	applyAuth(&p, req.Auth)
	// Preserve request header order (auth/content-type appended last) so the
	// generated snippet mirrors the request as authored.
	return p
}

func applyAuth(p *prepared, a model.Auth) {
	switch a.Type {
	case model.AuthBearer:
		// An explicit Authorization header takes precedence over the auth block,
		// so don't emit a duplicate.
		if !hasHeader(p.headers, "Authorization") {
			p.headers = append(p.headers, model.KV{Key: "Authorization", Value: "Bearer " + a.Token})
		}
	case model.AuthBasic:
		p.basicUser, p.basicPass = a.Username, a.Password
		if !hasHeader(p.headers, "Authorization") {
			p.headers = append(p.headers, model.KV{Key: "Authorization", Value: "Basic <base64(user:pass)>"})
		}
	case model.AuthAPIKey:
		if a.Key == "" {
			return
		}
		if a.Placement == model.APIKeyQuery {
			p.url = appendQuery(p.url, a.Key, a.KeyValue)
		} else {
			p.headers = append(p.headers, model.KV{Key: a.Key, Value: a.KeyValue})
		}
	}
}

func buildURL(req model.Request) string {
	base := req.URL
	for _, kv := range req.Params {
		if kv.Enabled && kv.Key != "" {
			base = appendQuery(base, kv.Key, kv.Value)
		}
	}
	return base
}

func appendQuery(rawURL, key, val string) string {
	q := url.QueryEscape(key) + "=" + url.QueryEscape(val)
	if strings.Contains(rawURL, "?") {
		return rawURL + "&" + q
	}
	return rawURL + "?" + q
}

func hasHeader(hs []model.KV, key string) bool {
	for _, h := range hs {
		if strings.EqualFold(h.Key, key) {
			return true
		}
	}
	return false
}

// --- per-target renderers -------------------------------------------------

func genCurl(p prepared) string {
	var b strings.Builder
	b.WriteString("curl")
	if p.method != "GET" {
		b.WriteString(" -X " + p.method)
	}
	b.WriteString(" " + shq(p.url))
	if p.basicUser != "" {
		b.WriteString(" \\\n  -u " + shq(p.basicUser+":"+p.basicPass))
	}
	for _, h := range p.headers {
		if h.Key == "Authorization" && p.basicUser != "" {
			continue // covered by -u
		}
		b.WriteString(" \\\n  -H " + shq(h.Key+": "+h.Value))
	}
	for _, kv := range p.multipart {
		if kv.File {
			b.WriteString(" \\\n  -F " + shq(kv.Key+"=@"+kv.Value))
		} else {
			b.WriteString(" \\\n  -F " + shq(kv.Key+"="+kv.Value))
		}
	}
	if p.body != "" {
		b.WriteString(" \\\n  -d " + shq(p.body))
	}
	return b.String()
}

func genFetch(p prepared) string {
	var b strings.Builder
	fmt.Fprintf(&b, "await fetch(%s, {\n", jsStr(p.url))
	fmt.Fprintf(&b, "  method: %s,\n", jsStr(p.method))
	if len(p.headers) > 0 {
		b.WriteString("  headers: {\n")
		for _, h := range p.headers {
			fmt.Fprintf(&b, "    %s: %s,\n", jsStr(h.Key), jsStr(h.Value))
		}
		b.WriteString("  },\n")
	}
	if p.body != "" {
		fmt.Fprintf(&b, "  body: %s,\n", jsStr(p.body))
	}
	if len(p.multipart) > 0 {
		b.WriteString("  // multipart: build a FormData with these fields\n")
		for _, kv := range p.multipart {
			if kv.File {
				fmt.Fprintf(&b, "  //   %s: <file %s>\n", kv.Key, kv.Value)
			} else {
				fmt.Fprintf(&b, "  //   %s: %s\n", kv.Key, kv.Value)
			}
		}
	}
	b.WriteString("});")
	return b.String()
}

func genHTTPie(p prepared) string {
	var b strings.Builder
	b.WriteString("http " + p.method + " " + shq(p.url))
	if p.basicUser != "" {
		b.WriteString(" -a " + shq(p.basicUser+":"+p.basicPass))
	}
	for _, h := range p.headers {
		if h.Key == "Authorization" && p.basicUser != "" {
			continue
		}
		b.WriteString(" " + shq(h.Key+":"+h.Value))
	}
	if len(p.multipart) > 0 {
		b.WriteString(" --multipart")
		for _, kv := range p.multipart {
			if kv.File {
				b.WriteString(" " + shq(kv.Key+"@"+kv.Value))
			} else {
				b.WriteString(" " + shq(kv.Key+"="+kv.Value))
			}
		}
	}
	if p.body != "" {
		b.WriteString(" \\\n  <<< " + shq(p.body))
	}
	return b.String()
}

func genPython(p prepared) string {
	var b strings.Builder
	b.WriteString("import requests\n\n")
	if len(p.headers) > 0 {
		b.WriteString("headers = {\n")
		for _, h := range p.headers {
			fmt.Fprintf(&b, "    %s: %s,\n", pyStr(h.Key), pyStr(h.Value))
		}
		b.WriteString("}\n")
	}
	if p.body != "" {
		fmt.Fprintf(&b, "data = %s\n", pyStr(p.body))
	}
	if len(p.multipart) > 0 {
		b.WriteString("files = {\n")
		for _, kv := range p.multipart {
			if kv.File {
				fmt.Fprintf(&b, "    %s: open(%s, \"rb\"),\n", pyStr(kv.Key), pyStr(kv.Value))
			}
		}
		b.WriteString("}\n")
		b.WriteString("form = {\n")
		for _, kv := range p.multipart {
			if !kv.File {
				fmt.Fprintf(&b, "    %s: %s,\n", pyStr(kv.Key), pyStr(kv.Value))
			}
		}
		b.WriteString("}\n")
	}
	b.WriteString("\nresp = requests.request(\n")
	fmt.Fprintf(&b, "    %s, %s,\n", pyStr(p.method), pyStr(p.url))
	if len(p.headers) > 0 {
		b.WriteString("    headers=headers,\n")
	}
	if p.body != "" {
		b.WriteString("    data=data,\n")
	}
	if len(p.multipart) > 0 {
		b.WriteString("    files=files,\n    data=form,\n")
	}
	if p.basicUser != "" {
		fmt.Fprintf(&b, "    auth=(%s, %s),\n", pyStr(p.basicUser), pyStr(p.basicPass))
	}
	b.WriteString(")\nprint(resp.status_code, resp.text)")
	return b.String()
}

func genGo(p prepared) string {
	var b strings.Builder
	b.WriteString("package main\n\nimport (\n\t\"fmt\"\n\t\"io\"\n\t\"net/http\"\n\t\"strings\"\n)\n\n")
	b.WriteString("func main() {\n")
	if len(p.multipart) > 0 {
		b.WriteString("\t// multipart body: build with mime/multipart; fields:\n")
		for _, kv := range p.multipart {
			if kv.File {
				fmt.Fprintf(&b, "\t//   %s: file %s\n", kv.Key, kv.Value)
			} else {
				fmt.Fprintf(&b, "\t//   %s: %s\n", kv.Key, kv.Value)
			}
		}
	}
	if p.body != "" {
		fmt.Fprintf(&b, "\tbody := strings.NewReader(%s)\n", goStr(p.body))
		fmt.Fprintf(&b, "\treq, _ := http.NewRequest(%s, %s, body)\n", goStr(p.method), goStr(p.url))
	} else {
		fmt.Fprintf(&b, "\treq, _ := http.NewRequest(%s, %s, nil)\n", goStr(p.method), goStr(p.url))
	}
	for _, h := range p.headers {
		fmt.Fprintf(&b, "\treq.Header.Set(%s, %s)\n", goStr(h.Key), goStr(h.Value))
	}
	if p.basicUser != "" {
		fmt.Fprintf(&b, "\treq.SetBasicAuth(%s, %s)\n", goStr(p.basicUser), goStr(p.basicPass))
	}
	b.WriteString("\tresp, err := http.DefaultClient.Do(req)\n")
	b.WriteString("\tif err != nil {\n\t\tpanic(err)\n\t}\n")
	b.WriteString("\tdefer resp.Body.Close()\n")
	b.WriteString("\tout, _ := io.ReadAll(resp.Body)\n")
	b.WriteString("\tfmt.Println(resp.Status, string(out))\n")
	b.WriteString("}")
	return b.String()
}

// --- quoting helpers ------------------------------------------------------

// shq single-quotes a string for POSIX shells, escaping embedded quotes.
func shq(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func jsStr(s string) string { return strconv.Quote(s) }
func goStr(s string) string { return strconv.Quote(s) }
func pyStr(s string) string { return strconv.Quote(s) }
