package history

import (
	"testing"

	"senda/internal/model"
)

func TestAppendAndList(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 3; i++ {
		err := Append(dir, model.HistoryEntry{Method: "GET", URL: "https://x.test", Status: 200 + i})
		if err != nil {
			t.Fatal(err)
		}
	}
	got, err := List(dir, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	// Newest first.
	if got[0].Status != 202 {
		t.Errorf("newest status = %d, want 202", got[0].Status)
	}
	if got[0].At == "" {
		t.Error("timestamp not set")
	}
}

func TestListMissingIsEmpty(t *testing.T) {
	got, err := List(t.TempDir(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("want empty, got %d", len(got))
	}
}

func TestListLimit(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 5; i++ {
		_ = Append(dir, model.HistoryEntry{Method: "GET", URL: "u", Status: i})
	}
	got, _ := List(dir, 2)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Status != 4 {
		t.Errorf("status = %d, want 4", got[0].Status)
	}
}

func TestClear(t *testing.T) {
	dir := t.TempDir()
	_ = Append(dir, model.HistoryEntry{Method: "GET", URL: "u"})
	if err := Clear(dir); err != nil {
		t.Fatal(err)
	}
	got, _ := List(dir, 10)
	if len(got) != 0 {
		t.Errorf("want empty after clear, got %d", len(got))
	}
	// Clearing again is a no-op.
	if err := Clear(dir); err != nil {
		t.Errorf("second clear errored: %v", err)
	}
}
