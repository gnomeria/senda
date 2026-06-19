package mockserver

import (
	"fmt"
	"reflect"
	"sort"
)

// matchSpec reports whether the request context satisfies all conditions in
// spec (query + headers exact, body deep-contains). A nil spec always matches.
func matchSpec(spec *MatchSpec, ctx tmplCtx) bool {
	if spec == nil {
		return true
	}
	for k, v := range spec.Query {
		if ctx.query == nil || ctx.query.Get(k) != v {
			return false
		}
	}
	for k, v := range spec.Headers {
		if ctx.headers == nil || ctx.headers.Get(k) != v {
			return false
		}
	}
	for k, want := range spec.Body {
		bm, ok := ctx.body.(map[string]any)
		if !ok {
			return false
		}
		got, ok := bm[k]
		if !ok || !valuesEqual(want, got) {
			return false
		}
	}
	return true
}

// valuesEqual compares a YAML-decoded expected value with a JSON-decoded actual
// value. Numbers and bools are compared by their string form so 1 (int) equals
// 1 (float64) across the YAML/JSON boundary; maps recurse (deep-contains).
func valuesEqual(want, got any) bool {
	if wm, ok := want.(map[string]any); ok {
		gm, ok := got.(map[string]any)
		if !ok {
			return false
		}
		for k, wv := range wm {
			gv, ok := gm[k]
			if !ok || !valuesEqual(wv, gv) {
				return false
			}
		}
		return true
	}
	if reflect.DeepEqual(want, got) {
		return true
	}
	return fmt.Sprintf("%v", want) == fmt.Sprintf("%v", got)
}

// selectResponse picks a response by precedence:
//  1. a response tagged with the active scenario whose When matches,
//  2. an untagged (default) response whose When matches,
//  3. any response whose When matches.
//
// Returns nil when none qualify.
func selectResponse(responses []ResponseDef, ctx tmplCtx, scenario string) *ResponseDef {
	if scenario != "" {
		for i := range responses {
			if responses[i].Scenario == scenario && matchSpec(responses[i].When, ctx) {
				return &responses[i]
			}
		}
	}
	for i := range responses {
		if responses[i].Scenario == "" && matchSpec(responses[i].When, ctx) {
			return &responses[i]
		}
	}
	for i := range responses {
		if matchSpec(responses[i].When, ctx) {
			return &responses[i]
		}
	}
	return nil
}

// responseByStatus returns the first response with the given status code whose
// When condition matches the request — used by the live per-endpoint override.
// Returns nil when no response carries that status.
func responseByStatus(responses []ResponseDef, status int, ctx tmplCtx) *ResponseDef {
	for i := range responses {
		rs := responses[i].Status
		if rs == 0 {
			rs = 200
		}
		if rs == status && matchSpec(responses[i].When, ctx) {
			return &responses[i]
		}
	}
	return nil
}

// orderRoutes sorts candidate routes by priority desc, then specificity desc
// (more literal path segments first), then original file order — so the most
// specific intentional route wins.
func orderRoutes(routes []*route) {
	sort.SliceStable(routes, func(i, j int) bool {
		a, b := routes[i], routes[j]
		if a.priority != b.priority {
			return a.priority > b.priority
		}
		if a.specificity != b.specificity {
			return a.specificity > b.specificity
		}
		return a.order < b.order
	})
}
