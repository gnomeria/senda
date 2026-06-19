// Package script runs a request's pre/post JavaScript via goja.
//
// Scripts see plain JSON-shaped objects (`req`, `res`) plus a `senda` global
// with setVar/getVar for runtime variables, a `console` global for logging,
// and a `pm` global providing Postman-compatible script helpers.
package script

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dop251/goja"

	"senda/internal/model"
)

// Timeout aborts runaway scripts (infinite loops etc.).
const Timeout = 5 * time.Second

// GetVar resolves a variable name for senda.getVar (runtime vars layered over
// the send scope). Missing names return "".
type GetVar func(name string) string

// SetVar stores a runtime variable visible to later interpolation and sends.
type SetVar func(name, value string)

// preludePre defines request helpers. Functions are skipped by
// JSON.stringify, so they never leak back into the model.
const preludePre = `
req.params = req.params || [];
req.headers = req.headers || [];
req.getHeader = function (k) {
  k = String(k).toLowerCase();
  for (var i = 0; i < req.headers.length; i++)
    if (req.headers[i].key.toLowerCase() === k) return req.headers[i].value;
  return null;
};
req.setHeader = function (k, v) {
  var lk = String(k).toLowerCase();
  for (var i = 0; i < req.headers.length; i++)
    if (req.headers[i].key.toLowerCase() === lk) {
      req.headers[i].value = String(v);
      req.headers[i].enabled = true;
      return;
    }
  req.headers.push({ key: String(k), value: String(v), enabled: true });
};
req.setParam = function (k, v) {
  for (var i = 0; i < req.params.length; i++)
    if (req.params[i].key === k) {
      req.params[i].value = String(v);
      req.params[i].enabled = true;
      return;
    }
  req.params.push({ key: String(k), value: String(v), enabled: true });
};
`

const preludePost = `
try { res.json = JSON.parse(res.body); } catch (e) { res.json = null; }
`

// preludePM provides the Postman-compatible pm.* global.
// pm.test results are accumulated in __pmTests and collected after execution.
const preludePM = `
var __pmTests = [];
var pm = (function() {
  var _vars = {
    get: function(k) { return senda.getVar(k) || ""; },
    set: function(k, v) { senda.setVar(k, String(v)); },
    unset: function(k) { senda.setVar(k, ""); }
  };

  function makeChain(actual, neg) {
    function check(pass, msgPos, msgNeg) {
      if ((neg ? pass : !pass)) throw new Error(neg ? msgNeg : msgPos);
    }
    var c = {};
    Object.defineProperty(c, 'not', { get: function() { return makeChain(actual, !neg); }, enumerable: true });
    c.equal = function(exp) {
      check(
        actual === exp || JSON.stringify(actual) === JSON.stringify(exp),
        "Expected " + JSON.stringify(exp) + " but got " + JSON.stringify(actual),
        "Expected not equal to " + JSON.stringify(exp)
      );
      return c;
    };
    c.eql = c.equal;
    c.include = function(sub) {
      check(
        actual != null && String(actual).indexOf(String(sub)) >= 0,
        "Expected to include " + JSON.stringify(sub),
        "Expected not to include " + JSON.stringify(sub)
      );
      return c;
    };
    c.contain = c.include;
    c.match = function(re) {
      check(re.test(String(actual)), "Expected to match " + re, "Expected not to match " + re);
      return c;
    };
    c.a = function(t) {
      var got = Array.isArray(actual) ? 'array' : typeof actual;
      check(got === t, "Expected type " + t + " but got " + got, "Expected not type " + t);
      return c;
    };
    c.an = c.a;
    c.above = function(n) {
      check(actual > n, "Expected " + actual + " > " + n, "Expected " + actual + " not > " + n);
      return c;
    };
    c.below = function(n) {
      check(actual < n, "Expected " + actual + " < " + n, "Expected " + actual + " not < " + n);
      return c;
    };
    c.least = function(n) {
      check(actual >= n, "Expected " + actual + " >= " + n, "Expected " + actual + " not >= " + n);
      return c;
    };
    c.most = function(n) {
      check(actual <= n, "Expected " + actual + " <= " + n, "Expected " + actual + " not <= " + n);
      return c;
    };
    c.property = function(k) {
      check(actual != null && k in Object(actual),
        "Expected to have property " + k, "Expected not to have property " + k);
      return c;
    };
    c.lengthOf = function(n) {
      var len = actual ? actual.length : 0;
      check(len === n, "Expected length " + n + " but got " + len, "Expected length not " + n);
      return c;
    };
    c.ok = function() {
      check(!!actual, "Expected truthy value", "Expected falsy value");
      return c;
    };
    c.true = function() {
      check(actual === true, "Expected true", "Expected not true");
      return c;
    };
    c.false = function() {
      check(actual === false, "Expected false", "Expected not false");
      return c;
    };
    c.null = function() {
      check(actual === null, "Expected null", "Expected not null");
      return c;
    };
    c.undefined = function() {
      check(actual === undefined, "Expected undefined", "Expected not undefined");
      return c;
    };
    c.status = function(code) {
      check(actual === code, "Expected status " + code + " but got " + actual, "Expected status not " + code);
      return c;
    };
    // chainable no-ops for fluency
    c.to = c; c.be = c; c.have = c; c.that = c; c.and = c;
    c.which = c; c.is = c; c.does = c; c.deep = c; c.been = c;
    return c;
  }

  var o = {
    environment: _vars,
    collectionVariables: _vars,
    variables: _vars,
    globals: _vars,
    test: function(name, fn) {
      var pass = true, err = "";
      try { fn(); } catch(e) { pass = false; err = (e && e.message) ? e.message : String(e); }
      __pmTests.push({ target: name, op: "pm.test", pass: pass, error: err });
    },
    expect: function(actual) { return makeChain(actual, false); },
    sendRequest: function() {},
    response: null
  };
  return o;
})();
`

// preludePostPM wires pm.response after res is available.
const preludePostPM = `
if (typeof res !== "undefined" && pm) {
  pm.response = {
    json: function() { return res.json; },
    text: function() { return res.body; },
    code: res.status,
    status: res.statusText,
    responseTime: res.durationMs,
    to: {
      have: {
        status: function(code) {
          if (res.status !== code)
            throw new Error("Expected status " + code + " but got " + res.status);
        }
      }
    }
  };
}
`

// newVM builds a runtime with the senda global and an interrupt timer. The
// returned stop func must be called when the script is done.
func newVM(getVar GetVar, setVar SetVar) (*goja.Runtime, func()) {
	vm := goja.New()
	senda := vm.NewObject()
	_ = senda.Set("getVar", func(name string) string { return getVar(name) })
	_ = senda.Set("setVar", func(name, value string) { setVar(name, value) })
	_ = vm.Set("senda", senda)
	timer := time.AfterFunc(Timeout, func() { vm.Interrupt("script timeout") })
	return vm, func() { timer.Stop() }
}

// setupConsole installs a console global and returns a pointer to the log slice.
func setupConsole(vm *goja.Runtime) *[]string {
	var logs []string
	console := vm.NewObject()
	logFn := func(call goja.FunctionCall) goja.Value {
		parts := make([]string, len(call.Arguments))
		for i, a := range call.Arguments {
			if a == nil || goja.IsUndefined(a) {
				parts[i] = "undefined"
			} else if goja.IsNull(a) {
				parts[i] = "null"
			} else {
				parts[i] = a.String()
			}
		}
		logs = append(logs, strings.Join(parts, " "))
		return goja.Undefined()
	}
	_ = console.Set("log", logFn)
	_ = console.Set("error", logFn)
	_ = console.Set("warn", logFn)
	_ = console.Set("info", logFn)
	_ = console.Set("debug", logFn)
	_ = vm.Set("console", console)
	return &logs
}

// collectPMTests reads the __pmTests array from the VM after script execution.
func collectPMTests(vm *goja.Runtime) []model.AssertResult {
	val, err := vm.RunString("JSON.stringify(__pmTests)")
	if err != nil {
		return nil
	}
	var tests []model.AssertResult
	if err := json.Unmarshal([]byte(val.String()), &tests); err != nil {
		return nil
	}
	return tests
}

// inject parses v to JSON and defines it as a global named name.
func inject(vm *goja.Runtime, name string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = vm.RunString("var " + name + " = " + string(data) + ";")
	return err
}

// RunPre executes a pre-request script. Returns the (possibly mutated) request,
// console log lines, and any error. On error the original request is returned unchanged.
func RunPre(src string, req model.Request, getVar GetVar, setVar SetVar) (model.Request, []string, error) {
	vm, stop := newVM(getVar, setVar)
	defer stop()
	logs := setupConsole(vm)
	if _, err := vm.RunString(preludePM); err != nil {
		return req, *logs, fmt.Errorf("pre-script: %w", err)
	}
	if err := inject(vm, "req", req); err != nil {
		return req, *logs, fmt.Errorf("pre-script: %w", err)
	}
	if _, err := vm.RunString(preludePre); err != nil {
		return req, *logs, fmt.Errorf("pre-script: %w", err)
	}
	if _, err := vm.RunString(src); err != nil {
		return req, *logs, fmt.Errorf("pre-script: %w", err)
	}
	out, err := vm.RunString("JSON.stringify(req)")
	if err != nil {
		return req, *logs, fmt.Errorf("pre-script: %w", err)
	}
	var mutated model.Request
	if err := json.Unmarshal([]byte(out.String()), &mutated); err != nil {
		return req, *logs, fmt.Errorf("pre-script: request no longer valid: %w", err)
	}
	return mutated, *logs, nil
}

// RunPost executes a post-response script against the (read-only) request and
// response. Returns pm.test results, console log lines, and any error.
func RunPost(src string, req model.Request, resp model.Response, getVar GetVar, setVar SetVar) ([]model.AssertResult, []string, error) {
	vm, stop := newVM(getVar, setVar)
	defer stop()
	logs := setupConsole(vm)
	if _, err := vm.RunString(preludePM); err != nil {
		return nil, *logs, fmt.Errorf("post-script: %w", err)
	}
	if err := inject(vm, "req", req); err != nil {
		return nil, *logs, fmt.Errorf("post-script: %w", err)
	}
	if err := inject(vm, "res", resp); err != nil {
		return nil, *logs, fmt.Errorf("post-script: %w", err)
	}
	if _, err := vm.RunString(preludePost); err != nil {
		return nil, *logs, fmt.Errorf("post-script: %w", err)
	}
	if _, err := vm.RunString(preludePostPM); err != nil {
		return nil, *logs, fmt.Errorf("post-script: %w", err)
	}
	if _, err := vm.RunString(src); err != nil {
		return nil, *logs, fmt.Errorf("post-script: %w", err)
	}
	return collectPMTests(vm), *logs, nil
}
