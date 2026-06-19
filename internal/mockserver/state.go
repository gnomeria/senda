package mockserver

import (
	"fmt"
	"strconv"
	"sync"
)

// Store holds the in-memory state for resource routes. Records seed from YAML
// on load and mutate in RAM; ResetState restores the seeds. Nothing is written
// to disk.
type Store struct {
	mu        sync.Mutex
	resources map[string]*resourceState
}

type resourceState struct {
	key     string
	seed    []map[string]any // immutable copy for reset
	records []map[string]any
	nextID  int
}

func newStore() *Store { return &Store{resources: map[string]*resourceState{}} }

// sync reconciles the store with the given resource defs: adds new resources
// (seeded), drops removed ones, and leaves existing resources' live records
// untouched (so a hot-reload doesn't wipe mutations).
func (s *Store) sync(defs []MockDef) {
	s.mu.Lock()
	defer s.mu.Unlock()
	seen := map[string]bool{}
	for _, d := range defs {
		seen[d.Resource] = true
		if _, ok := s.resources[d.Resource]; ok {
			continue
		}
		s.resources[d.Resource] = newResourceState(d)
	}
	for name := range s.resources {
		if !seen[name] {
			delete(s.resources, name)
		}
	}
}

func newResourceState(d MockDef) *resourceState {
	key := d.Key
	if key == "" {
		key = "id"
	}
	rs := &resourceState{key: key}
	for _, rec := range d.Seed {
		rs.seed = append(rs.seed, cloneRecord(rec))
	}
	rs.reset()
	return rs
}

func (rs *resourceState) reset() {
	rs.records = nil
	rs.nextID = 1
	for _, rec := range rs.seed {
		c := cloneRecord(rec)
		rs.records = append(rs.records, c)
		if n := numericID(c[rs.key]); n >= rs.nextID {
			rs.nextID = n + 1
		}
	}
}

// ResetState restores every resource to its seed.
func (s *Store) ResetState() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, rs := range s.resources {
		rs.reset()
	}
}

func (s *Store) get(resource string) *resourceState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.resources[resource]
}

// --- CRUD operations (each locks the store) ---

func (s *Store) list(resource string) []map[string]any {
	rs := s.get(resource)
	if rs == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]map[string]any, len(rs.records))
	copy(out, rs.records)
	return out
}

func (s *Store) find(resource, id string) (map[string]any, bool) {
	rs := s.get(resource)
	if rs == nil {
		return nil, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, rec := range rs.records {
		if fmt.Sprintf("%v", rec[rs.key]) == id {
			return rec, true
		}
	}
	return nil, false
}

func (s *Store) create(resource string, body map[string]any) (map[string]any, bool) {
	rs := s.get(resource)
	if rs == nil {
		return nil, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rec := cloneRecord(body)
	if _, ok := rec[rs.key]; !ok || fmt.Sprintf("%v", rec[rs.key]) == "" {
		rec[rs.key] = rs.nextID
		rs.nextID++
	} else if n := numericID(rec[rs.key]); n >= rs.nextID {
		rs.nextID = n + 1
	}
	rs.records = append(rs.records, rec)
	return rec, true
}

func (s *Store) update(resource, id string, body map[string]any, merge bool) (map[string]any, bool) {
	rs := s.get(resource)
	if rs == nil {
		return nil, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, rec := range rs.records {
		if fmt.Sprintf("%v", rec[rs.key]) != id {
			continue
		}
		if merge {
			for k, v := range body {
				rec[k] = v
			}
		} else {
			rec = cloneRecord(body)
			rec[rs.key] = rs.records[i][rs.key] // preserve id
			rs.records[i] = rec
		}
		return rec, true
	}
	return nil, false
}

func (s *Store) delete(resource, id string) bool {
	rs := s.get(resource)
	if rs == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, rec := range rs.records {
		if fmt.Sprintf("%v", rec[rs.key]) == id {
			rs.records = append(rs.records[:i], rs.records[i+1:]...)
			return true
		}
	}
	return false
}

func cloneRecord(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func numericID(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	case string:
		if i, err := strconv.Atoi(n); err == nil {
			return i
		}
	}
	return 0
}
