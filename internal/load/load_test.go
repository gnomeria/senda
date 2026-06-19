package load_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"senda/internal/load"
	"senda/internal/model"
)

func constSend(ms int64, status int) load.Send {
	return func(ctx context.Context, _ string) (model.Request, model.Response, error) {
		select {
		case <-ctx.Done():
			return model.Request{}, model.Response{Error: ctx.Err().Error()}, ctx.Err()
		case <-time.After(time.Duration(ms) * time.Millisecond):
		}
		return model.Request{}, model.Response{Status: status, DurationMs: ms}, nil
	}
}

func constFactory(ms int64, status int) load.SendFactory {
	return func() load.Send { return constSend(ms, status) }
}

func TestIterations(t *testing.T) {
	// 3 VUs × 2 iterations × 2 paths = 12 total requests
	sum := load.Run(
		context.Background(),
		[]string{"a.yaml", "b.yaml"},
		model.LoadOptions{VUs: 3, Iterations: 2},
		constFactory(1, 200),
		nil,
	)
	if sum.Total != 12 {
		t.Fatalf("want 12 got %d", sum.Total)
	}
	if sum.Errors != 0 {
		t.Fatalf("want 0 errors got %d", sum.Errors)
	}
	if sum.StatusDist[200] != 12 {
		t.Fatalf("want 12 × 200 got %v", sum.StatusDist)
	}
}

func TestDuration(t *testing.T) {
	start := time.Now()
	sum := load.Run(
		context.Background(),
		[]string{"a.yaml"},
		model.LoadOptions{VUs: 2, Duration: 1},
		constFactory(10, 200),
		nil,
	)
	elapsed := time.Since(start)
	if elapsed < 900*time.Millisecond || elapsed > 3*time.Second {
		t.Fatalf("unexpected elapsed %v", elapsed)
	}
	if sum.Total == 0 {
		t.Fatal("want >0 requests got 0")
	}
	if sum.RPS <= 0 {
		t.Fatalf("want positive RPS got %f", sum.RPS)
	}
}

func TestErrorCounting(t *testing.T) {
	var n atomic.Int32
	errFactory := func() load.Send {
		return func(_ context.Context, _ string) (model.Request, model.Response, error) {
			// every other request errors
			if n.Add(1)%2 == 0 {
				return model.Request{}, model.Response{Status: 500, DurationMs: 1, Error: "boom"}, nil
			}
			return model.Request{}, model.Response{Status: 200, DurationMs: 1}, nil
		}
	}
	sum := load.Run(
		context.Background(),
		[]string{"a.yaml", "b.yaml", "c.yaml", "d.yaml"},
		model.LoadOptions{VUs: 1, Iterations: 1},
		errFactory,
		nil,
	)
	if sum.Total != 4 {
		t.Fatalf("want 4 got %d", sum.Total)
	}
	if sum.Errors != 2 {
		t.Fatalf("want 2 errors got %d", sum.Errors)
	}
}

func TestPercentiles(t *testing.T) {
	i := 0
	latencies := []int64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
	f := func() load.Send {
		return func(_ context.Context, _ string) (model.Request, model.Response, error) {
			ms := latencies[i%len(latencies)]
			i++
			return model.Request{}, model.Response{Status: 200, DurationMs: ms}, nil
		}
	}
	paths := make([]string, 10)
	for j := range paths {
		paths[j] = "req.yaml"
	}
	sum := load.Run(context.Background(), paths, model.LoadOptions{VUs: 1, Iterations: 1}, f, nil)
	if sum.P50 == 0 {
		t.Fatal("P50 should not be 0")
	}
	if sum.P99 < sum.P50 {
		t.Fatalf("P99 %f < P50 %f", sum.P99, sum.P50)
	}
	if sum.P99 > sum.P50*11 {
		t.Fatalf("P99 %f implausibly large vs P50 %f", sum.P99, sum.P50)
	}
}

func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()
	sum := load.Run(
		ctx,
		[]string{"a.yaml"},
		model.LoadOptions{VUs: 2, Duration: 60}, // 60s would hang without cancel
		constFactory(10, 200),
		nil,
	)
	if sum.Total == 0 {
		t.Fatal("want >0 requests before cancel")
	}
}
