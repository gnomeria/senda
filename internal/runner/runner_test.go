package runner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"senda/internal/model"
)

func TestRunFolderOrderAndStatus(t *testing.T) {
	paths := []string{"a.yaml", "b.yaml", "c.yaml"}
	send := func(ctx context.Context, p string) (model.Request, model.Response, error) {
		switch p {
		case "a.yaml":
			return model.Request{Name: "a", Method: "GET", URL: "u/a"}, model.Response{Status: 200, DurationMs: 5}, nil
		case "b.yaml":
			return model.Request{Name: "b"}, model.Response{Status: 500}, nil
		default:
			return model.Request{Name: "c"}, model.Response{Error: "dial tcp: refused"}, nil
		}
	}
	var streamed []model.RunResult
	res := RunFolder(context.Background(), paths, send, func(r model.RunResult) {
		streamed = append(streamed, r)
	})
	if len(res) != 3 {
		t.Fatalf("len = %d", len(res))
	}
	if len(streamed) != 3 || streamed[0].Name != "a" {
		t.Errorf("streamed results mismatch: %+v", streamed)
	}
	if !res[0].OK || res[0].Name != "a" {
		t.Errorf("a = %+v", res[0])
	}
	if res[1].OK {
		t.Errorf("b should fail (500): %+v", res[1])
	}
	if res[2].Error == "" || res[2].OK {
		t.Errorf("c should carry transport error: %+v", res[2])
	}
}

func TestRunFolderAsserts(t *testing.T) {
	send := func(ctx context.Context, p string) (model.Request, model.Response, error) {
		switch p {
		case "pass.yaml":
			return model.Request{Name: "p"}, model.Response{Status: 200, Asserts: []model.AssertResult{
				{Target: "status", Op: "eq", Value: "200", Pass: true},
			}}, nil
		default:
			return model.Request{Name: "f"}, model.Response{Status: 200, Asserts: []model.AssertResult{
				{Target: "status", Op: "eq", Value: "200", Pass: true},
				{Target: "json.id", Op: "eq", Value: "1", Pass: false},
			}}, nil
		}
	}
	res := RunFolder(context.Background(), []string{"pass.yaml", "fail.yaml"}, send, nil)
	if !res[0].OK || res[0].AssertPass != 1 || res[0].AssertFail != 0 {
		t.Errorf("pass = %+v", res[0])
	}
	if res[1].OK || res[1].AssertPass != 1 || res[1].AssertFail != 1 {
		t.Errorf("2xx with failed assert must not be OK: %+v", res[1])
	}
}

func TestRunFolderReadError(t *testing.T) {
	send := func(ctx context.Context, p string) (model.Request, model.Response, error) {
		return model.Request{}, model.Response{}, errors.New("parse failed")
	}
	res := RunFolder(context.Background(), []string{"x.yaml"}, send, nil)
	if res[0].Error != "parse failed" || res[0].OK {
		t.Errorf("want read error result, got %+v", res[0])
	}
}

func TestRunFolderCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	res := RunFolder(ctx, []string{"a.yaml", "b.yaml"}, func(ctx context.Context, p string) (model.Request, model.Response, error) {
		t.Fatal("send should not be called after cancel")
		return model.Request{}, model.Response{}, nil
	}, nil)
	if len(res) != 1 || res[0].Error == "" {
		t.Errorf("want one cancelled result, got %+v", res)
	}
}

func TestOnFailStop(t *testing.T) {
	paths := []string{"a.yaml", "b.yaml", "c.yaml"}
	send := func(ctx context.Context, p string) (model.Request, model.Response, error) {
		if p == "b.yaml" {
			return model.Request{Name: "b", OnFail: "stop"}, model.Response{Status: 500}, nil
		}
		return model.Request{Name: p}, model.Response{Status: 200}, nil
	}
	res := RunFolder(context.Background(), paths, send, nil)
	// should stop after b — c never runs
	if len(res) != 2 {
		t.Fatalf("expected 2 results (stop after b), got %d", len(res))
	}
}

func TestOnFailContinue(t *testing.T) {
	paths := []string{"a.yaml", "b.yaml", "c.yaml"}
	send := func(ctx context.Context, p string) (model.Request, model.Response, error) {
		if p == "b.yaml" {
			return model.Request{Name: "b", OnFail: "continue"}, model.Response{Status: 500}, nil
		}
		return model.Request{Name: p}, model.Response{Status: 200}, nil
	}
	res := RunFolder(context.Background(), paths, send, nil)
	if len(res) != 3 {
		t.Fatalf("expected 3 results (continue after b), got %d", len(res))
	}
}

func TestLoadDataFileCSV(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "data.csv")
	if err := os.WriteFile(f, []byte("user,role\nalice,admin\nbob,viewer\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rows, err := LoadDataFile(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0]["user"] != "alice" || rows[1]["role"] != "viewer" {
		t.Errorf("rows = %v", rows)
	}
}

func TestLoadDataFileJSON(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "data.json")
	if err := os.WriteFile(f, []byte(`[{"id":"1","name":"Alice"},{"id":"2","name":"Bob"}]`), 0o644); err != nil {
		t.Fatal(err)
	}
	rows, err := LoadDataFile(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0]["name"] != "Alice" {
		t.Errorf("rows = %v", rows)
	}
}
