package importer

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"senda/internal/mockserver"
)

const petstoreMockSample = `
openapi: 3.0.0
info: { title: Petstore, version: "1.0" }
servers:
  - url: https://api.test/v1
paths:
  /pets:
    get:
      operationId: listPets
      tags: [pets]
      responses:
        "200":
          description: A list of pets
          content:
            application/json:
              example:
                - id: 1
                  name: Rex
  /pets/{petId}:
    get:
      operationId: getPet
      tags: [pets]
      responses:
        "200":
          description: Found pet
          content:
            application/json:
              schema:
                type: object
                properties:
                  id: { type: integer }
                  name: { type: string }
                  email: { type: string, format: email }
        "404":
          description: Pet not found
          content:
            application/json:
              schema:
                type: object
                properties:
                  error: { type: string }
`

func defByName(t *testing.T, defs []mockserver.MockDef, name string) mockserver.MockDef {
	t.Helper()
	for _, d := range defs {
		if d.Name == name {
			return d
		}
	}
	t.Fatalf("mock def %q missing (have %d)", name, len(defs))
	return mockserver.MockDef{}
}

func TestOpenAPIMocksStructure(t *testing.T) {
	defs, err := OpenAPIMocks([]byte(petstoreMockSample))
	if err != nil {
		t.Fatal(err)
	}
	if len(defs) != 2 {
		t.Fatalf("generated %d defs, want 2", len(defs))
	}

	list := defByName(t, defs, "listPets")
	if list.Method != "GET" || list.Path != "/pets" {
		t.Errorf("listPets = %s %s", list.Method, list.Path)
	}
	if len(list.Responses) != 1 || list.Responses[0].Body == nil {
		t.Fatalf("listPets responses = %+v", list.Responses)
	}

	get := defByName(t, defs, "getPet")
	if get.Path != "/pets/:petId" { // braces converted to colon form
		t.Errorf("path = %q, want /pets/:petId", get.Path)
	}
	if len(get.Responses) != 2 {
		t.Fatalf("getPet responses = %d, want 2", len(get.Responses))
	}
	// 2xx sorts first and becomes the default; it has no inline example so the
	// schema is embedded for request-time faking.
	if get.Responses[0].Status != 200 || get.Responses[0].Schema == "" {
		t.Errorf("first response = %+v, want 200 w/ schema", get.Responses[0])
	}
	if !strings.Contains(get.Responses[0].Schema, "email") {
		t.Errorf("schema missing properties: %q", get.Responses[0].Schema)
	}
	if get.Responses[1].Status != 404 || get.Responses[1].Desc != "Pet not found" {
		t.Errorf("second response = %+v, want 404 'Pet not found'", get.Responses[1])
	}
}

// TestOpenAPIMocksServe is an end-to-end check: generate defs, write them as
// YAML into a mocks dir, start a mock server over it, and confirm it serves the
// example body and can live-switch an endpoint to its 404 variant.
func TestOpenAPIMocksServe(t *testing.T) {
	defs, err := OpenAPIMocks([]byte(petstoreMockSample))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for _, d := range defs {
		out, err := yaml.Marshal(d)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, d.Name+".yaml"), out, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	srv, err := mockserver.New(dir, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	addr, err := srv.Start("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Stop()
	base := "http://" + addr

	// Example list body served verbatim.
	resp, body := getBody(t, base+"/pets")
	if resp.StatusCode != 200 {
		t.Fatalf("GET /pets = %d", resp.StatusCode)
	}
	if !strings.Contains(body, "Rex") {
		t.Errorf("list body = %q, want example with Rex", body)
	}

	// Schema-faked detail body: default 200 returns an object with the schema's
	// properties faked.
	resp, body = getBody(t, base+"/pets/7")
	if resp.StatusCode != 200 {
		t.Fatalf("GET /pets/7 = %d", resp.StatusCode)
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(body), &obj); err != nil {
		t.Fatalf("detail body not json: %q", body)
	}
	if _, ok := obj["name"]; !ok {
		t.Errorf("faked detail missing name: %q", body)
	}

	// Live-switch the endpoint to its 404 variant — no file edit.
	srv.SetRouteResponse("GET", "/pets/:petId", 404)
	resp, _ = getBody(t, base+"/pets/7")
	if resp.StatusCode != 404 {
		t.Fatalf("after override GET /pets/7 = %d, want 404", resp.StatusCode)
	}

	// Clearing the override restores the default 200.
	srv.SetRouteResponse("GET", "/pets/:petId", 0)
	resp, _ = getBody(t, base+"/pets/7")
	if resp.StatusCode != 200 {
		t.Fatalf("after clear GET /pets/7 = %d, want 200", resp.StatusCode)
	}
}

// TestOpenAPIMocksTrainTravel exercises the generator against the real Train
// Travel spec (OpenAPI 3.1, heavy $ref schemas) and confirms every generated
// def serves without error, with schemas inlined (no dangling $ref).
func TestOpenAPIMocksTrainTravel(t *testing.T) {
	data, err := os.ReadFile("testdata/train-travel-3.1.yaml")
	if err != nil {
		t.Skipf("testdata missing: %v", err)
	}
	defs, err := OpenAPIMocks(data)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if len(defs) == 0 {
		t.Fatal("no mock defs generated")
	}
	for _, d := range defs {
		for _, r := range d.Responses {
			if strings.Contains(r.Schema, "$ref") {
				t.Errorf("%s response %d schema has unresolved $ref", d.Name, r.Status)
			}
		}
	}
}

func getBody(t *testing.T, url string) (*http.Response, string) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return resp, string(b)
}
