package security

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"senda/internal/model"
)

// maxBodyBytes caps how much of a response body matchers see.
const maxBodyBytes = 1 << 20 // 1 MiB

// engine executes templates against targets with a shared rate limiter.
type engine struct {
	client    *http.Client // no redirects
	redirects *http.Client // follows up to 10
	limiter   <-chan time.Time
}

func newEngine(timeout time.Duration, rate int) *engine {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	noRedirect := func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &engine{
		client:    &http.Client{Timeout: timeout, Transport: transport, CheckRedirect: noRedirect},
		redirects: &http.Client{Timeout: timeout, Transport: transport},
		limiter:   time.Tick(time.Second / time.Duration(rate)),
	}
}

// urlVars expands the nuclei placeholders for one target URL.
// BaseURL = the input as given (no trailing slash), RootURL = scheme://host[:port],
// Hostname = host[:port], Host = host.
func urlVars(target string) (map[string]string, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, err
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("target %q has no scheme or host", target)
	}
	return map[string]string{
		"{{BaseURL}}":  strings.TrimSuffix(target, "/"),
		"{{RootURL}}":  u.Scheme + "://" + u.Host,
		"{{Hostname}}": u.Host,
		"{{Host}}":     u.Hostname(),
		"{{Scheme}}":   u.Scheme,
		"{{Path}}":     u.Path,
	}, nil
}

func expand(s string, vars map[string]string) string {
	for k, v := range vars {
		s = strings.ReplaceAll(s, k, v)
	}
	return s
}

// run executes one template against one target. It returns the matched URL
// of the first request that satisfied the matchers, or "" for no match. An
// error means no probe got a response at all, so the check is inconclusive
// rather than passed.
func (e *engine) run(ctx context.Context, t *Template, target string) (string, error) {
	vars, err := urlVars(target)
	if err != nil {
		return "", err
	}
	answered := false
	var lastErr error
	for _, reqDef := range t.requests() {
		for _, p := range reqDef.Path {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-e.limiter:
			}
			reqURL := expand(p, vars)
			matched, err := e.send(ctx, &reqDef, reqURL, vars)
			if err != nil {
				lastErr = err // unreachable path: try the next
				continue
			}
			answered = true
			if matched {
				return reqURL, nil
			}
		}
	}
	if !answered && lastErr != nil {
		return "", lastErr
	}
	return "", nil
}

// send issues one request and evaluates the block's matchers against it.
func (e *engine) send(ctx context.Context, def *HTTPRequest, reqURL string, vars map[string]string) (bool, error) {
	var body io.Reader
	if def.Body != "" {
		body = strings.NewReader(expand(def.Body, vars))
	}
	req, err := http.NewRequestWithContext(ctx, def.Method, reqURL, body)
	if err != nil {
		return false, err
	}
	req.Header.Set("User-Agent", "senda-security-scan")
	for k, v := range def.Headers {
		req.Header.Set(k, expand(v, vars))
	}
	client := e.client
	if def.Redirects {
		client = e.redirects
	}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))

	r := response{
		status:  resp.StatusCode,
		body:    string(raw),
		headers: headerBlob(resp.Header),
		size:    len(raw),
	}
	return matchAll(def, r), nil
}

// response is the matcher view of an HTTP response.
type response struct {
	status  int
	body    string
	headers string
	size    int
}

func headerBlob(h http.Header) string {
	var b strings.Builder
	for k, vs := range h {
		for _, v := range vs {
			b.WriteString(k)
			b.WriteString(": ")
			b.WriteString(v)
			b.WriteString("\n")
		}
	}
	return b.String()
}

// matchAll evaluates the block's matchers under its matchers-condition.
func matchAll(def *HTTPRequest, r response) bool {
	and := def.MatchersCondition == "and"
	for i := range def.Matchers {
		ok := matchOne(&def.Matchers[i], r)
		if and && !ok {
			return false
		}
		if !and && ok {
			return true
		}
	}
	return and
}

func matchOne(m *Matcher, r response) bool {
	var ok bool
	switch m.Type {
	case "status":
		for _, s := range m.Status {
			if r.status == s {
				ok = true
				break
			}
		}
	case "size":
		for _, s := range m.Size {
			if r.size == s {
				ok = true
				break
			}
		}
	case "word":
		ok = matchValues(len(m.Words), m.Condition, func(i int) bool {
			part, word := m.part(r), m.Words[i]
			if m.CaseInsensitive {
				part, word = strings.ToLower(part), strings.ToLower(word)
			}
			return strings.Contains(part, word)
		})
	case "regex":
		ok = matchValues(len(m.compiled), m.Condition, func(i int) bool {
			return m.compiled[i].MatchString(m.part(r))
		})
	}
	if m.Negative {
		return !ok
	}
	return ok
}

// part selects the response slice a word/regex matcher inspects.
func (m *Matcher) part(r response) string {
	switch m.Part {
	case "header":
		return r.headers
	case "all":
		return r.headers + r.body
	default:
		return r.body
	}
}

// matchValues folds per-value results under the matcher's condition
// (or = any, and = all).
func matchValues(n int, condition string, f func(int) bool) bool {
	if n == 0 {
		return false
	}
	and := condition == "and"
	for i := 0; i < n; i++ {
		ok := f(i)
		if and && !ok {
			return false
		}
		if !and && ok {
			return true
		}
	}
	return and
}

// check builds the binding-friendly result for one executed template; the
// caller fills in the matched/error outcome.
func check(t *Template, target string) model.SecurityCheck {
	return model.SecurityCheck{
		TemplateID:  t.ID,
		Name:        t.Info.Name,
		Severity:    t.Info.Severity,
		Target:      target,
		Description: t.Info.Description,
		Remediation: t.Info.Remediation,
		Tags:        t.Info.Tags,
		Reference:   t.Info.Reference,
		OWASP:       t.Info.Metadata["owasp"],
		CWE:         t.Info.Classification.CWEID,
	}
}
