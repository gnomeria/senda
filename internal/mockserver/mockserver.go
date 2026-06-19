// Package mockserver implements a local HTTP mock server that serves responses
// defined in YAML files in a collection's mocks/ directory.
//
// Two kinds of mock files are supported:
//
//   - Rule routes: method + path + one or more responses. Responses can be
//     selected by request conditions (match/when) or by the active scenario,
//     and their bodies pass through a {{...}} template engine (path params,
//     query, headers, request body, faker, uuid, randomInt, now).
//
//   - Resource routes: a `resource:` declaration auto-wires REST CRUD over an
//     in-memory store (GET list/item, POST create, PUT/PATCH update, DELETE).
//
// An optional mocks/_config.yaml sets a proxy passthrough target, toggles CORS,
// and picks the default scenario. The server hot-reloads when files change.
package mockserver

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

const maxBodyBytes = 1 << 20 // 1 MiB cap on request bodies read for matching

// Server is a running mock server.
type Server struct {
	mocksDir string

	mu               sync.RWMutex
	rules            []*route
	resources        []resourceRoute
	config           ServerConfig
	scenario         string
	scenarioOverride bool
	routeOverrides   map[string]int // routeKey -> forced response status (live per-endpoint switch)
	store            *Store

	logMu sync.Mutex
	log   []LogEntry

	onLog    func(LogEntry)
	onRoutes func()

	httpSrv *http.Server
	watcher *fsnotify.Watcher
	addr    string
}

// New creates a Server reading mock definitions from mocksDir. onLog (if
// non-nil) is called for each request; onRoutes (if non-nil) is called after a
// hot-reload so the UI can refresh.
func New(mocksDir string, onLog func(LogEntry), onRoutes func()) (*Server, error) {
	l, err := loadDir(mocksDir)
	if err != nil {
		return nil, err
	}
	s := &Server{
		mocksDir:       mocksDir,
		rules:          l.rules,
		resources:      l.resources,
		config:         l.config,
		scenario:       l.config.Scenario,
		routeOverrides: map[string]int{},
		store:          newStore(),
		onLog:          onLog,
		onRoutes:       onRoutes,
	}
	s.store.sync(l.resourceDefs())
	return s, nil
}

// Start binds on addr (e.g. ":8787"; ":0" picks a free port) and serves.
func (s *Server) Start(addr string) (string, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return "", err
	}
	s.addr = ln.Addr().String()
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handle)
	s.httpSrv = &http.Server{Handler: mux}
	go s.httpSrv.Serve(ln)
	s.startWatch()
	return s.addr, nil
}

// Stop shuts down the server and its file watcher.
func (s *Server) Stop() error {
	if s.watcher != nil {
		s.watcher.Close()
		s.watcher = nil
	}
	if s.httpSrv == nil {
		return nil
	}
	return s.httpSrv.Close()
}

// Addr returns the bound address.
func (s *Server) Addr() string { return s.addr }

// Scenario returns the active scenario.
func (s *Server) Scenario() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.scenario
}

// SetScenario switches the active scenario; it persists across hot-reloads.
func (s *Server) SetScenario(name string) {
	s.mu.Lock()
	s.scenario = name
	s.scenarioOverride = true
	s.mu.Unlock()
}

// SetRouteResponse forces the rule route identified by method+path to return the
// response variant with the given status code, live and in memory (no file
// edit). A status of 0 clears the override and restores default selection. The
// override persists across hot-reloads as long as the route still exists.
func (s *Server) SetRouteResponse(method, path string, status int) {
	key := routeKey(method, path)
	s.mu.Lock()
	if status == 0 {
		delete(s.routeOverrides, key)
	} else {
		s.routeOverrides[key] = status
	}
	s.mu.Unlock()
}

// routeKey identifies a rule route by its method and path template. An empty
// method (a match-any route) is normalised to "ANY" so it agrees with the
// method reported by Routes() to the UI.
func routeKey(method, path string) string {
	m := strings.ToUpper(method)
	if m == "" {
		m = "ANY"
	}
	return m + " " + path
}

// ResetState restores all resource records to their seeds.
func (s *Server) ResetState() { s.store.ResetState() }

// Info summarizes server config for the UI.
type Info struct {
	Addr      string   `json:"addr"`
	Scenario  string   `json:"scenario"`
	Proxy     string   `json:"proxy"`
	CORS      bool     `json:"cors"`
	Scenarios []string `json:"scenarios"`
}

// Info returns the current server configuration and the set of scenario names
// declared across all responses.
func (s *Server) Info() Info {
	s.mu.RLock()
	defer s.mu.RUnlock()
	seen := map[string]bool{}
	var scenarios []string
	for _, rt := range s.rules {
		for _, resp := range rt.def.Responses {
			if resp.Scenario != "" && !seen[resp.Scenario] {
				seen[resp.Scenario] = true
				scenarios = append(scenarios, resp.Scenario)
			}
		}
	}
	return Info{
		Addr:      s.addr,
		Scenario:  s.scenario,
		Proxy:     s.config.Proxy,
		CORS:      s.config.corsEnabled(),
		Scenarios: scenarios,
	}
}

// Log returns a copy of the request log.
func (s *Server) Log() []LogEntry {
	s.logMu.Lock()
	defer s.logMu.Unlock()
	out := make([]LogEntry, len(s.log))
	copy(out, s.log)
	return out
}

// Routes returns a flattened description of every loaded route.
func (s *Server) Routes() []RouteInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []RouteInfo
	for _, rt := range s.rules {
		status := 200
		if len(rt.def.Responses) > 0 && rt.def.Responses[0].Status != 0 {
			status = rt.def.Responses[0].Status
		}
		method := rt.method
		if method == "" {
			method = "ANY"
		}
		out = append(out, RouteInfo{
			Name: rt.def.Name, Method: method, Path: rt.def.Path,
			Status: status, Kind: "rule", Responses: len(rt.def.Responses),
			Variants: routeVariants(rt.def.Responses),
			Active:   s.routeOverrides[routeKey(rt.method, rt.def.Path)],
		})
	}
	for _, rr := range s.resources {
		out = append(out, RouteInfo{
			Name: rr.def.Name, Method: "CRUD", Path: rr.def.Path,
			Status: 200, Kind: "resource",
		})
	}
	return out
}

// routeVariants flattens a rule route's responses into the distinct status
// codes it can return, in declaration order, so the UI can offer them as live
// switchable options. The first occurrence of each status keeps its label.
func routeVariants(responses []ResponseDef) []RouteVariant {
	var out []RouteVariant
	seen := map[int]bool{}
	for _, resp := range responses {
		st := resp.Status
		if st == 0 {
			st = 200
		}
		if seen[st] {
			continue
		}
		seen[st] = true
		out = append(out, RouteVariant{Status: st, Desc: resp.Desc})
	}
	return out
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	rules := s.rules
	resources := s.resources
	config := s.config
	scenario := s.scenario
	store := s.store
	overrides := s.routeOverrides
	s.mu.RUnlock()

	if config.corsEnabled() {
		addCORS(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}

	bodyBytes, _ := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes))
	var bodyVal any
	if len(bodyBytes) > 0 {
		_ = json.Unmarshal(bodyBytes, &bodyVal)
	}
	ctx := tmplCtx{query: r.URL.Query(), headers: r.Header, body: bodyVal}

	// 1. Rule routes — most specific matching route that yields a response.
	if status, ok := s.serveRule(w, r, rules, ctx, scenario, overrides); ok {
		s.record(r, status, "mock")
		return
	}

	// 2. Resource routes — stateful CRUD.
	if status, ok := s.serveResource(w, r, resources, store, bodyVal); ok {
		s.record(r, status, "state")
		return
	}

	// 3. Proxy passthrough on no match.
	if config.Proxy != "" {
		status := proxyPass(w, r, config.Proxy, bodyBytes)
		s.record(r, status, "proxy")
		return
	}

	http.NotFound(w, r)
	s.record(r, http.StatusNotFound, "mock")
}

func (s *Server) serveRule(w http.ResponseWriter, r *http.Request, rules []*route, ctx tmplCtx, scenario string, overrides map[string]int) (int, bool) {
	var candidates []*route
	for _, rt := range rules {
		if rt.method != "" && rt.method != r.Method {
			continue
		}
		if !rt.pattern.MatchString(r.URL.Path) {
			continue
		}
		if !matchSpec(rt.def.Match, ctx) {
			continue
		}
		candidates = append(candidates, rt)
	}
	if len(candidates) == 0 {
		return 0, false
	}
	orderRoutes(candidates)
	for _, rt := range candidates {
		m := rt.pattern.FindStringSubmatch(r.URL.Path)
		params := map[string]string{}
		for i, name := range rt.params {
			if i+1 < len(m) {
				params[name] = m[i+1]
			}
		}
		ctx.params = params
		var resp *ResponseDef
		if forced := overrides[routeKey(rt.method, rt.def.Path)]; forced != 0 {
			resp = responseByStatus(rt.def.Responses, forced, ctx)
		}
		if resp == nil {
			resp = selectResponse(rt.def.Responses, ctx, scenario)
		}
		if resp == nil {
			continue
		}
		writeResponse(w, *resp, ctx)
		status := resp.Status
		if status == 0 {
			status = 200
		}
		return status, true
	}
	return 0, false
}

func writeResponse(w http.ResponseWriter, resp ResponseDef, ctx tmplCtx) {
	if resp.Delay > 0 {
		time.Sleep(time.Duration(resp.Delay) * time.Millisecond)
	}
	for k, v := range resp.Headers {
		w.Header().Set(k, renderString(v, ctx))
	}
	body := renderBody(resp, ctx)
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/json")
	}
	status := resp.Status
	if status == 0 {
		status = 200
	}
	w.WriteHeader(status)
	w.Write(body)
}

// renderBody produces the response body bytes: native objects/arrays are
// templated then JSON-encoded; strings are templated literally; an empty body
// with a Schema generates fake data.
func renderBody(resp ResponseDef, ctx tmplCtx) []byte {
	switch b := resp.Body.(type) {
	case nil:
		if resp.Schema != "" {
			return []byte(fakeFromSchema(resp.Schema))
		}
		return nil
	case string:
		return []byte(renderString(b, ctx))
	default:
		rendered := renderAny(b, ctx)
		out, err := json.Marshal(rendered)
		if err != nil {
			return nil
		}
		return out
	}
}

func (s *Server) record(r *http.Request, status int, source string) {
	entry := LogEntry{
		At:     time.Now().UTC().Format(time.RFC3339),
		Method: r.Method,
		Path:   r.URL.Path,
		Status: status,
		Source: source,
	}
	s.logMu.Lock()
	s.log = append(s.log, entry)
	s.logMu.Unlock()
	if s.onLog != nil {
		s.onLog(entry)
	}
}

// serveResource dispatches REST CRUD against the in-memory store. Returns
// (status, true) when a resource route matched the path.
func (s *Server) serveResource(w http.ResponseWriter, r *http.Request, resources []resourceRoute, store *Store, body any) (int, bool) {
	for _, rr := range resources {
		path := r.URL.Path
		if rr.collPat.MatchString(path) {
			switch r.Method {
			case http.MethodGet:
				return writeJSON(w, http.StatusOK, store.list(rr.resource)), true
			case http.MethodPost:
				bm, _ := body.(map[string]any)
				rec, ok := store.create(rr.resource, bm)
				if !ok {
					return writeJSON(w, http.StatusNotFound, errObj("unknown resource")), true
				}
				return writeJSON(w, http.StatusCreated, rec), true
			}
			continue
		}
		if m := rr.itemPat.FindStringSubmatch(path); m != nil {
			id := m[1]
			switch r.Method {
			case http.MethodGet:
				rec, ok := store.find(rr.resource, id)
				if !ok {
					return writeJSON(w, http.StatusNotFound, errObj("not found")), true
				}
				return writeJSON(w, http.StatusOK, rec), true
			case http.MethodPut, http.MethodPatch:
				bm, _ := body.(map[string]any)
				rec, ok := store.update(rr.resource, id, bm, r.Method == http.MethodPatch)
				if !ok {
					return writeJSON(w, http.StatusNotFound, errObj("not found")), true
				}
				return writeJSON(w, http.StatusOK, rec), true
			case http.MethodDelete:
				if !store.delete(rr.resource, id) {
					return writeJSON(w, http.StatusNotFound, errObj("not found")), true
				}
				w.WriteHeader(http.StatusNoContent)
				return http.StatusNoContent, true
			}
		}
	}
	return 0, false
}

func writeJSON(w http.ResponseWriter, status int, v any) int {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
	return status
}

func errObj(msg string) map[string]any { return map[string]any{"error": msg} }

func addCORS(w http.ResponseWriter) {
	h := w.Header()
	h.Set("Access-Control-Allow-Origin", "*")
	h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
	h.Set("Access-Control-Allow-Headers", "*")
}

// --- hot reload ---

func (s *Server) startWatch() {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return
	}
	if err := w.Add(s.mocksDir); err != nil {
		w.Close()
		return
	}
	s.watcher = w
	go func() {
		var timer *time.Timer
		for {
			select {
			case _, ok := <-w.Events:
				if !ok {
					return
				}
				if timer != nil {
					timer.Stop()
				}
				timer = time.AfterFunc(200*time.Millisecond, s.reload)
			case _, ok := <-w.Errors:
				if !ok {
					return
				}
			}
		}
	}()
}

func (s *Server) reload() {
	l, err := loadDir(s.mocksDir)
	if err != nil {
		return
	}
	s.mu.Lock()
	s.rules = l.rules
	s.resources = l.resources
	s.config = l.config
	if !s.scenarioOverride {
		s.scenario = l.config.Scenario
	}
	s.mu.Unlock()
	s.store.sync(l.resourceDefs())
	if s.onRoutes != nil {
		s.onRoutes()
	}
}
