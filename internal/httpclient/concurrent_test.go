package httpclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"senda/internal/model"
	"senda/internal/vars"
)

// TestSendConcurrentNoRace sends many requests concurrently on a single Client,
// the way a flow's parallel node does. The per-send httptrace callbacks may run
// on background dial goroutines that outlive Do, so without synchronisation the
// timing capture races. Run with -race; this guards that path.
func TestSendConcurrentNoRace(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := New()
	scope := vars.Build()

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp := c.Send(context.Background(), model.Request{Method: "GET", URL: srv.URL}, scope)
			if resp.Status != http.StatusOK {
				t.Errorf("status = %d, want 200 (err %q)", resp.Status, resp.Error)
			}
		}()
	}
	wg.Wait()
}
