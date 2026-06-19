package security

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Template is the subset of the nuclei template format this engine executes:
// HTTP requests with status/word/regex/size matchers. Templates using other
// protocols or features (payloads, raw requests, dsl matchers, extractors)
// are skipped at load time so a community template dir can be pointed at
// wholesale. The builtin checks (templates/*.yaml) use only this subset and
// remain runnable by real nuclei.
type Template struct {
	ID   string `yaml:"id"`
	Info struct {
		Name        string  `yaml:"name"`
		Severity    string  `yaml:"severity"`
		Description string  `yaml:"description"`
		Remediation string  `yaml:"remediation"`
		Tags        TagList `yaml:"tags"`
		Reference   RefList `yaml:"reference"`
		// Classification carries the standard nuclei CWE field; the OWASP
		// API Top 10 id lives in metadata (nuclei has no native owasp key).
		Classification struct {
			CWEID TagList `yaml:"cwe-id"`
		} `yaml:"classification"`
		Metadata map[string]string `yaml:"metadata"`
	} `yaml:"info"`
	HTTP []HTTPRequest `yaml:"http"`
	// Requests is the pre-v3 alias for HTTP still used by many templates.
	Requests []HTTPRequest `yaml:"requests"`
}

// requests returns the request blocks regardless of which key declared them.
func (t *Template) requests() []HTTPRequest {
	if len(t.HTTP) > 0 {
		return t.HTTP
	}
	return t.Requests
}

// HTTPRequest is one http block: paths to try with the given method, plus
// matchers deciding whether a response is a finding.
type HTTPRequest struct {
	Method            string            `yaml:"method"`
	Path              []string          `yaml:"path"`
	Headers           map[string]string `yaml:"headers"`
	Body              string            `yaml:"body"`
	Redirects         bool              `yaml:"redirects"`
	MaxRedirects      int               `yaml:"max-redirects"`
	StopAtFirstMatch  bool              `yaml:"stop-at-first-match"`
	MatchersCondition string            `yaml:"matchers-condition"` // and|or, default or
	Matchers          []Matcher         `yaml:"matchers"`
	Raw               []string          `yaml:"raw"`      // unsupported -> skip template
	Payloads          map[string]any    `yaml:"payloads"` // unsupported -> skip template
}

// Matcher is one response check. Type is status, word, regex or size.
type Matcher struct {
	Type            string   `yaml:"type"`
	Part            string   `yaml:"part"`      // body|header|all (default body)
	Condition       string   `yaml:"condition"` // and|or for multi-value, default or
	Negative        bool     `yaml:"negative"`
	CaseInsensitive bool     `yaml:"case-insensitive"`
	Status          []int    `yaml:"status"`
	Words           []string `yaml:"words"`
	Regex           []string `yaml:"regex"`
	Size            []int    `yaml:"size"`

	compiled []*regexp.Regexp
}

// TagList accepts both nuclei tag styles: a comma string or a YAML list.
type TagList []string

func (t *TagList) UnmarshalYAML(node *yaml.Node) error {
	var s string
	if err := node.Decode(&s); err == nil {
		for _, p := range strings.Split(s, ",") {
			if p = strings.TrimSpace(p); p != "" {
				*t = append(*t, p)
			}
		}
		return nil
	}
	var list []string
	if err := node.Decode(&list); err != nil {
		return err
	}
	*t = list
	return nil
}

// RefList accepts a single string or a YAML list.
type RefList []string

func (r *RefList) UnmarshalYAML(node *yaml.Node) error {
	var s string
	if err := node.Decode(&s); err == nil {
		if s != "" {
			*r = []string{s}
		}
		return nil
	}
	var list []string
	if err := node.Decode(&list); err != nil {
		return err
	}
	*r = list
	return nil
}

// ParseTemplate decodes one template and validates it against the supported
// subset. Returns an error describing why a template is unsupported; callers
// loading a directory treat that as "skip", not failure.
func ParseTemplate(data []byte) (*Template, error) {
	var t Template
	if err := yaml.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	if t.ID == "" {
		return nil, fmt.Errorf("template has no id")
	}
	reqs := t.requests()
	if len(reqs) == 0 {
		return nil, fmt.Errorf("%s: no http requests (unsupported protocol)", t.ID)
	}
	for i := range reqs {
		r := &reqs[i]
		if len(r.Raw) > 0 {
			return nil, fmt.Errorf("%s: raw requests unsupported", t.ID)
		}
		if len(r.Payloads) > 0 {
			return nil, fmt.Errorf("%s: payloads unsupported", t.ID)
		}
		if len(r.Path) == 0 {
			return nil, fmt.Errorf("%s: request has no path", t.ID)
		}
		if r.Method == "" {
			r.Method = "GET"
		}
		if len(r.Matchers) == 0 {
			return nil, fmt.Errorf("%s: request has no matchers", t.ID)
		}
		for j := range r.Matchers {
			m := &r.Matchers[j]
			switch m.Type {
			case "status", "word", "regex", "size":
			default:
				return nil, fmt.Errorf("%s: matcher type %q unsupported", t.ID, m.Type)
			}
			for _, expr := range m.Regex {
				re, err := regexp.Compile(expr)
				if err != nil {
					return nil, fmt.Errorf("%s: bad regex: %w", t.ID, err)
				}
				m.compiled = append(m.compiled, re)
			}
		}
	}
	t.Info.Severity = strings.ToLower(strings.TrimSpace(t.Info.Severity))
	if t.Info.Severity == "" {
		t.Info.Severity = "info"
	}
	return &t, nil
}

// LoadDir parses every .yaml/.yml under dir (recursively) from fsys,
// silently skipping unsupported templates. Returns templates keyed in
// encounter order.
func LoadDir(fsys fs.FS, dir string) []*Template {
	var out []*Template
	_ = fs.WalkDir(fsys, dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return nil
		}
		if t, err := ParseTemplate(data); err == nil {
			out = append(out, t)
		}
		return nil
	})
	return out
}

// Filter returns the templates matching a severity list ("high,critical";
// empty = all) and tag list (template must carry at least one; empty = all).
// Duplicate IDs keep the last occurrence so user templates override builtins.
func Filter(templates []*Template, severity string, tags []string) []*Template {
	sevSet := map[string]bool{}
	for _, s := range strings.Split(severity, ",") {
		if s = strings.ToLower(strings.TrimSpace(s)); s != "" {
			sevSet[s] = true
		}
	}
	byID := map[string]int{}
	var out []*Template
	for _, t := range templates {
		if len(sevSet) > 0 && !sevSet[t.Info.Severity] {
			continue
		}
		if len(tags) > 0 && !hasAnyTag(t.Info.Tags, tags) {
			continue
		}
		if i, dup := byID[t.ID]; dup {
			out[i] = t
			continue
		}
		byID[t.ID] = len(out)
		out = append(out, t)
	}
	return out
}

func hasAnyTag(have []string, want []string) bool {
	set := map[string]bool{}
	for _, h := range have {
		set[strings.ToLower(h)] = true
	}
	for _, w := range want {
		if set[strings.ToLower(strings.TrimSpace(w))] {
			return true
		}
	}
	return false
}
