package mockserver

// MockDef is one mock definition file under mocks/. It is either a rule route
// (method/path + responses) or a resource route (resource + seed). The v1
// single-response fields (Status/Headers/Body/Delay/Schema) are still honoured:
// when Responses is empty they are synthesized into one response, so old files
// keep working unchanged.
type MockDef struct {
	Name      string        `yaml:"name,omitempty" json:"name,omitempty"`
	Method    string        `yaml:"method,omitempty" json:"method,omitempty"`
	Path      string        `yaml:"path,omitempty" json:"path,omitempty"`
	Enabled   *bool         `yaml:"enabled,omitempty" json:"enabled,omitempty"` // nil = true
	Priority  int           `yaml:"priority,omitempty" json:"priority,omitempty"`
	Match     *MatchSpec    `yaml:"match,omitempty" json:"match,omitempty"`
	Responses []ResponseDef `yaml:"responses,omitempty" json:"responses,omitempty"`

	// v1 single-response fields — used only when Responses is empty.
	Status  int               `yaml:"status,omitempty" json:"status,omitempty"`
	Headers map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
	Body    any               `yaml:"body,omitempty" json:"body,omitempty"`
	Delay   int               `yaml:"delay,omitempty" json:"delay,omitempty"`
	Schema  string            `yaml:"schema,omitempty" json:"schema,omitempty"`

	// Resource route fields. When Resource is set this file declares a stateful
	// CRUD collection instead of a rule route.
	Resource string           `yaml:"resource,omitempty" json:"resource,omitempty"`
	Key      string           `yaml:"key,omitempty" json:"key,omitempty"` // id field, default "id"
	Seed     []map[string]any `yaml:"seed,omitempty" json:"seed,omitempty"`
}

// ResponseDef is one possible response for a rule route. When several are
// listed, the first whose When matches the request and whose Scenario matches
// the active scenario wins. Body may be a native object/array (rendered to
// JSON) or a string (served literally); both pass through templating.
type ResponseDef struct {
	Status   int               `yaml:"status,omitempty" json:"status,omitempty"`
	Desc     string            `yaml:"desc,omitempty" json:"desc,omitempty"` // human label, e.g. an OpenAPI response description
	Headers  map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
	Body     any               `yaml:"body,omitempty" json:"body,omitempty"`
	Delay    int               `yaml:"delay,omitempty" json:"delay,omitempty"`
	Schema   string            `yaml:"schema,omitempty" json:"schema,omitempty"`
	When     *MatchSpec        `yaml:"when,omitempty" json:"when,omitempty"`
	Scenario string            `yaml:"scenario,omitempty" json:"scenario,omitempty"`
}

// MatchSpec is a set of AND-ed conditions on a request. Query and Headers are
// exact key/value checks; Body is a deep-contains check against the decoded
// JSON request body.
type MatchSpec struct {
	Query   map[string]string `yaml:"query,omitempty" json:"query,omitempty"`
	Headers map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
	Body    map[string]any    `yaml:"body,omitempty" json:"body,omitempty"`
}

// ServerConfig is the optional mocks/_config.yaml controlling server-wide
// behaviour.
type ServerConfig struct {
	Proxy    string `yaml:"proxy,omitempty" json:"proxy,omitempty"`       // passthrough target on no match
	CORS     *bool  `yaml:"cors,omitempty" json:"cors,omitempty"`         // nil = true
	Scenario string `yaml:"scenario,omitempty" json:"scenario,omitempty"` // default active scenario
}

func (c ServerConfig) corsEnabled() bool { return c.CORS == nil || *c.CORS }

// LogEntry records one request handled by the mock server.
type LogEntry struct {
	At     string `json:"at"`
	Method string `json:"method"`
	Path   string `json:"path"`
	Status int    `json:"status"`
	Source string `json:"source,omitempty"` // "mock" | "state" | "proxy"
}

// RouteInfo is a flattened description of a loaded route for the UI/CLI.
type RouteInfo struct {
	Name      string         `json:"name"`
	Method    string         `json:"method"`
	Path      string         `json:"path"`
	Status    int            `json:"status"`
	Kind      string         `json:"kind"`               // "rule" | "resource"
	Responses int            `json:"responses"`          // number of response variants (rule routes)
	Variants  []RouteVariant `json:"variants,omitempty"` // selectable response variants (rule routes)
	Active    int            `json:"active,omitempty"`   // forced response status, 0 = default selection
}

// RouteVariant is one selectable response of a rule route — its status code and
// optional human label — used by the UI to live-switch what an endpoint returns.
type RouteVariant struct {
	Status int    `json:"status"`
	Desc   string `json:"desc,omitempty"`
}

func enabled(d MockDef) bool { return d.Enabled == nil || *d.Enabled }
