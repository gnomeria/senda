package mockserver

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// route is a compiled rule route ready for matching.
type route struct {
	def         MockDef
	method      string // upper-case; "" = any
	pattern     *regexp.Regexp
	params      []string
	priority    int
	specificity int
	order       int
}

// resourceRoute is a compiled stateful CRUD resource (collection + item paths).
type resourceRoute struct {
	def      MockDef
	resource string
	idName   string
	collPat  *regexp.Regexp // ^/users$
	itemPat  *regexp.Regexp // ^/users/([^/]+)$
}

// loaded is the full result of reading a mocks/ directory.
type loaded struct {
	rules     []*route
	resources []resourceRoute
	config    ServerConfig
}

// resourceDefs returns the MockDefs backing the resource routes (for the store).
func (l loaded) resourceDefs() []MockDef {
	out := make([]MockDef, len(l.resources))
	for i, r := range l.resources {
		out[i] = r.def
	}
	return out
}

// loadDir reads every *.yaml in dir into rule routes, resource routes and the
// optional _config.yaml. A missing directory yields an empty (non-error) set.
func loadDir(dir string) (loaded, error) {
	var out loaded
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return out, err
	}
	order := 0
	for _, e := range entries {
		if e.IsDir() || !isYAML(e.Name()) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		if e.Name() == "_config.yaml" || e.Name() == "_config.yml" {
			_ = yaml.Unmarshal(data, &out.config)
			continue
		}
		var def MockDef
		if err := yaml.Unmarshal(data, &def); err != nil {
			continue
		}
		if !enabled(def) {
			continue
		}
		if def.Resource != "" {
			out.resources = append(out.resources, compileResource(def))
			continue
		}
		if def.Path == "" {
			continue
		}
		pat, params := pathToRegexp(def.Path)
		out.rules = append(out.rules, &route{
			def:         normalizeRule(def),
			method:      strings.ToUpper(def.Method),
			pattern:     pat,
			params:      params,
			priority:    def.Priority,
			specificity: specificity(def.Path),
			order:       order,
		})
		order++
	}
	return out, nil
}

// normalizeRule folds v1 top-level response fields into Responses so the
// handler only ever deals with the Responses slice.
func normalizeRule(def MockDef) MockDef {
	if len(def.Responses) == 0 {
		def.Responses = []ResponseDef{{
			Status:  def.Status,
			Headers: def.Headers,
			Body:    def.Body,
			Delay:   def.Delay,
			Schema:  def.Schema,
		}}
	}
	return def
}

func compileResource(def MockDef) resourceRoute {
	id := def.Key
	if id == "" {
		id = "id"
	}
	base := strings.TrimRight(def.Path, "/")
	return resourceRoute{
		def:      def,
		resource: def.Resource,
		idName:   id,
		collPat:  regexp.MustCompile("^" + regexp.QuoteMeta(base) + "/?$"),
		itemPat:  regexp.MustCompile("^" + regexp.QuoteMeta(base) + "/([^/]+)$"),
	}
}

func isYAML(name string) bool {
	return strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml")
}

// specificity counts literal (non-parameter) path segments; higher = more
// specific, so /users/me beats /users/:id.
func specificity(path string) int {
	n := 0
	for _, seg := range strings.Split(strings.Trim(path, "/"), "/") {
		if seg != "" && !strings.HasPrefix(seg, ":") {
			n++
		}
	}
	return n
}

// pathToRegexp converts a path template like /users/:id to a regexp and returns
// the parameter names in order.
func pathToRegexp(path string) (*regexp.Regexp, []string) {
	var params []string
	re := regexp.MustCompile(`:([a-zA-Z_][a-zA-Z0-9_]*)`)
	escaped := regexp.QuoteMeta(path)
	escaped = strings.ReplaceAll(escaped, `\:`, ":") // QuoteMeta escapes the colon
	result := re.ReplaceAllStringFunc(escaped, func(m string) string {
		params = append(params, m[1:])
		return `([^/]+)`
	})
	return regexp.MustCompile("^" + result + "$"), params
}
