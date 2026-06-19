// Package load runs concurrent load tests against a folder of requests,
// collecting latency samples and streaming per-second ticks to the caller.
package load

import (
	"context"
	"sort"
	"sync"
	"time"

	"senda/internal/model"
)

// Send sends one request identified by its on-disk path and returns the
// (post-pre-script) request, its response, and any transport error.
type Send func(ctx context.Context, path string) (model.Request, model.Response, error)

// SendFactory creates an independent Send for one VU. Each VU gets its own
// session so their cookies and runtime variables don't cross-contaminate.
type SendFactory func() Send

type sample struct {
	durationMs int64
	status     int
	errored    bool
}

// Run executes the load test, blocks until it finishes, and returns a final
// summary. onTick, when non-nil, is called every second with a rolling
// snapshot of all samples collected so far.
func Run(
	ctx context.Context,
	paths []string,
	opts model.LoadOptions,
	factory SendFactory,
	onTick func(model.LoadTick),
) model.LoadSummary {
	if len(paths) == 0 || opts.VUs < 1 {
		return model.LoadSummary{}
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	if opts.Duration > 0 {
		var stop context.CancelFunc
		runCtx, stop = context.WithTimeout(runCtx, time.Duration(opts.Duration)*time.Second)
		defer stop()
	}

	// Buffer absorbs burst writes without blocking VUs.
	samples := make(chan sample, opts.VUs*16)

	var wg sync.WaitGroup
	for i := 0; i < opts.VUs; i++ {
		wg.Add(1)
		send := factory()
		vuIndex := i // capture for ramp-up delay
		go func() {
			defer wg.Done()
			// Ramp-up: stagger VU start times linearly across rampUp seconds.
			if opts.RampUp > 0 && opts.VUs > 1 {
				delay := time.Duration(float64(opts.RampUp) * float64(vuIndex) / float64(opts.VUs-1) * float64(time.Second))
				select {
				case <-time.After(delay):
				case <-runCtx.Done():
					return
				}
			}
			for iter := 0; ; iter++ {
				if opts.Iterations > 0 && iter >= opts.Iterations {
					return
				}
				for _, p := range paths {
					select {
					case <-runCtx.Done():
						return
					default:
					}
					_, resp, err := send(runCtx, p)
					s := sample{durationMs: resp.DurationMs, status: resp.Status}
					if err != nil || resp.Error != "" {
						s.errored = true
					}
					select {
					case samples <- s:
					case <-runCtx.Done():
						return
					}
				}
				// Single pass when neither Duration nor Iterations is set.
				if opts.Duration == 0 && opts.Iterations == 0 {
					return
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(samples)
	}()

	start := time.Now()
	var collected []sample

	// Incremental sort state: sortedDurs is kept sorted across ticks;
	// sortedUpTo tracks how many samples from collected are already merged in.
	// Per-tick cost is O(k log k + n) where k = new samples since last tick
	// and n = total sorted so far — avoids re-sorting all samples every second.
	var sortedDurs []int64
	sortedUpTo := 0

	// Running counters updated as samples arrive — no full scan on tick.
	errs := 0
	dist := map[int]int{}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

collect:
	for {
		select {
		case s, ok := <-samples:
			if !ok {
				break collect
			}
			collected = append(collected, s)
			if s.errored {
				errs++
			}
			if s.status > 0 {
				dist[s.status]++
			}
		case <-ticker.C:
			if onTick != nil {
				// Merge only the new samples into the sorted slice.
				if len(collected) > sortedUpTo {
					newSlice := collected[sortedUpTo:]
					newDurs := make([]int64, len(newSlice))
					for i, s := range newSlice {
						newDurs[i] = s.durationMs
					}
					sort.Slice(newDurs, func(i, j int) bool { return newDurs[i] < newDurs[j] })
					sortedDurs = mergeSorted(sortedDurs, newDurs)
					sortedUpTo = len(collected)
				}
				sec := time.Since(start).Seconds()
				rps := 0.0
				if sec > 0 {
					rps = float64(len(collected)) / sec
				}
				onTick(model.LoadTick{
					Elapsed:    sec,
					Total:      len(collected),
					Errors:     errs,
					RPS:        rps,
					P50:        percentile(sortedDurs, 0.5),
					P95:        percentile(sortedDurs, 0.95),
					P99:        percentile(sortedDurs, 0.99),
					StatusDist: copyDist(dist),
				})
			}
		}
	}

	return buildSummary(collected, time.Since(start).Seconds())
}

// mergeSorted merges two sorted int64 slices into a new sorted slice.
func mergeSorted(a, b []int64) []int64 {
	out := make([]int64, 0, len(a)+len(b))
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		if a[i] <= b[j] {
			out = append(out, a[i])
			i++
		} else {
			out = append(out, b[j])
			j++
		}
	}
	out = append(out, a[i:]...)
	out = append(out, b[j:]...)
	return out
}

func copyDist(m map[int]int) map[int]int {
	out := make(map[int]int, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func percentile(sorted []int64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	return float64(sorted[int(float64(len(sorted)-1)*p)])
}

func buildSummary(samples []sample, elapsed float64) model.LoadSummary {
	dur := make([]int64, 0, len(samples))
	errs := 0
	dist := map[int]int{}
	for _, s := range samples {
		dur = append(dur, s.durationMs)
		if s.errored {
			errs++
		}
		if s.status > 0 {
			dist[s.status]++
		}
	}
	sort.Slice(dur, func(i, j int) bool { return dur[i] < dur[j] })
	rps := 0.0
	if elapsed > 0 {
		rps = float64(len(samples)) / elapsed
	}
	return model.LoadSummary{
		Total:      len(samples),
		Errors:     errs,
		Duration:   elapsed,
		RPS:        rps,
		P50:        percentile(dur, 0.5),
		P95:        percentile(dur, 0.95),
		P99:        percentile(dur, 0.99),
		StatusDist: dist,
	}
}
